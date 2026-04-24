package core

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// ParseDSL parses the compact Morfx selector syntax into AgentQuery.
//
// Supported syntax:
//   - func:Handle*
//   - !func:Test*
//   - struct:* > field:Secret string
//   - field:Secret type=string visibility=private
//   - (func:* | method:*) > call:os.Getenv
//   - func:* & import:fmt
func ParseDSL(dsl string) (AgentQuery, error) {
	parser, err := newDSLParser(dsl)
	if err != nil {
		return AgentQuery{}, err
	}
	query, err := parser.parse()
	if err != nil {
		return AgentQuery{}, err
	}
	return query, nil
}

// ParseAgentQueryPayload parses either a JSON AgentQuery payload or a DSL selector.
// A non-empty DSL selector takes precedence and lets callers support both public
// surfaces without duplicating parsing rules.
func ParseAgentQueryPayload(raw json.RawMessage, dsl string) (AgentQuery, error) {
	if strings.TrimSpace(dsl) != "" {
		return ParseDSL(dsl)
	}
	if len(raw) == 0 {
		return AgentQuery{}, fmt.Errorf("query payload is required")
	}

	var query AgentQuery
	if err := json.Unmarshal(raw, &query); err != nil {
		return AgentQuery{}, err
	}
	return query, nil
}

// ParseOptionalAgentQueryPayload parses an optional JSON or DSL query. The bool
// return value reports whether any query was provided.
func ParseOptionalAgentQueryPayload(raw json.RawMessage, dsl string) (AgentQuery, bool, error) {
	if strings.TrimSpace(dsl) != "" {
		query, err := ParseDSL(dsl)
		return query, true, err
	}
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return AgentQuery{}, false, nil
	}

	var query AgentQuery
	if err := json.Unmarshal(raw, &query); err != nil {
		return AgentQuery{}, true, err
	}
	return query, true, nil
}

type dslTokenKind int

const (
	dslTokenSelector dslTokenKind = iota
	dslTokenOperator
	dslTokenLParen
	dslTokenRParen
	dslTokenAttribute
)

type dslToken struct {
	kind  dslTokenKind
	value string
}

type dslParser struct {
	tokens []dslToken
	pos    int
}

func newDSLParser(input string) (*dslParser, error) {
	tokens, err := tokenizeDSL(input)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("dsl query is required")
	}
	return &dslParser{tokens: tokens}, nil
}

func tokenizeDSL(input string) ([]dslToken, error) {
	var tokens []dslToken
	trimmed := strings.TrimSpace(input)
	for i := 0; i < len(trimmed); {
		r := rune(trimmed[i])
		if unicode.IsSpace(r) {
			i++
			continue
		}

		switch trimmed[i] {
		case '(':
			tokens = append(tokens, dslToken{kind: dslTokenLParen, value: "("})
			i++
			continue
		case ')':
			tokens = append(tokens, dslToken{kind: dslTokenRParen, value: ")"})
			i++
			continue
		case '!', '&', '|', '>':
			tokens = append(tokens, dslToken{kind: dslTokenOperator, value: string(trimmed[i])})
			i++
			continue
		}

		start := i
		for i < len(trimmed) {
			ch := trimmed[i]
			if unicode.IsSpace(rune(ch)) || ch == '(' || ch == ')' || ch == '!' || ch == '&' || ch == '|' || ch == '>' {
				break
			}
			i++
		}
		value := strings.TrimSpace(trimmed[start:i])
		if value == "" {
			continue
		}
		kind := dslTokenAttribute
		if strings.Contains(value, ":") {
			kind = dslTokenSelector
		}
		tokens = append(tokens, dslToken{kind: kind, value: value})
	}
	return tokens, nil
}

func (p *dslParser) parse() (AgentQuery, error) {
	query, err := p.parseOr()
	if err != nil {
		return AgentQuery{}, err
	}
	if p.hasNext() {
		return AgentQuery{}, fmt.Errorf("unexpected token %q", p.peek().value)
	}
	return query, nil
}

func (p *dslParser) parseOr() (AgentQuery, error) {
	left, err := p.parseAnd()
	if err != nil {
		return AgentQuery{}, err
	}
	for p.matchOperator("|") {
		right, err := p.parseAnd()
		if err != nil {
			return AgentQuery{}, fmt.Errorf("invalid OR operand: %w", err)
		}
		left = mergeBinaryQuery("OR", left, right)
	}
	return left, nil
}

func (p *dslParser) parseAnd() (AgentQuery, error) {
	left, err := p.parseContains()
	if err != nil {
		return AgentQuery{}, err
	}
	for p.matchOperator("&") {
		right, err := p.parseContains()
		if err != nil {
			return AgentQuery{}, fmt.Errorf("invalid AND operand: %w", err)
		}
		left = mergeBinaryQuery("AND", left, right)
	}
	return left, nil
}

func (p *dslParser) parseContains() (AgentQuery, error) {
	left, err := p.parseUnary()
	if err != nil {
		return AgentQuery{}, err
	}
	for p.matchOperator(">") {
		right, err := p.parseUnary()
		if err != nil {
			return AgentQuery{}, fmt.Errorf("invalid child selector: %w", err)
		}
		left = attachContains(left, right)
	}
	return left, nil
}

func (p *dslParser) parseUnary() (AgentQuery, error) {
	if p.matchOperator("!") {
		operand, err := p.parseUnary()
		if err != nil {
			return AgentQuery{}, fmt.Errorf("invalid NOT operand: %w", err)
		}
		return AgentQuery{Operator: "NOT", Operands: []AgentQuery{operand}}, nil
	}
	return p.parsePrimary()
}

func (p *dslParser) parsePrimary() (AgentQuery, error) {
	if !p.hasNext() {
		return AgentQuery{}, fmt.Errorf("selector is required")
	}
	if p.matchKind(dslTokenLParen) {
		query, err := p.parseOr()
		if err != nil {
			return AgentQuery{}, err
		}
		if !p.matchKind(dslTokenRParen) {
			return AgentQuery{}, fmt.Errorf("missing closing parenthesis")
		}
		return query, nil
	}
	return p.parseSimple()
}

func (p *dslParser) parseSimple() (AgentQuery, error) {
	if !p.hasNext() || p.peek().kind != dslTokenSelector {
		return AgentQuery{}, fmt.Errorf("selector must use kind:pattern syntax")
	}

	token := p.next()
	kindAndName := strings.SplitN(token.value, ":", 2)
	if len(kindAndName) != 2 || strings.TrimSpace(kindAndName[0]) == "" {
		return AgentQuery{}, fmt.Errorf("selector must use kind:pattern syntax")
	}

	query := AgentQuery{
		Type: strings.TrimSpace(kindAndName[0]),
		Name: strings.TrimSpace(kindAndName[1]),
	}
	if query.Name == "" {
		query.Name = "*"
	}

	attributes, err := p.parseAttributes()
	if err != nil {
		return AgentQuery{}, err
	}
	query.Attributes = attributes
	return query, nil
}

func (p *dslParser) parseAttributes() (map[string]string, error) {
	var shorthand []string
	attributes := make(map[string]string)

	for p.hasNext() && p.peek().kind == dslTokenAttribute {
		token := p.next()
		if strings.Contains(token.value, "=") {
			key, value, ok := strings.Cut(token.value, "=")
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if !ok || key == "" || value == "" {
				return nil, fmt.Errorf("invalid attribute %q", token.value)
			}
			if _, exists := attributes[key]; exists {
				return nil, fmt.Errorf("duplicate attribute %q", key)
			}
			attributes[key] = value
			continue
		}
		shorthand = append(shorthand, token.value)
	}

	if len(shorthand) > 0 {
		if _, exists := attributes["type"]; exists {
			return nil, fmt.Errorf("type attribute cannot be combined with shorthand type")
		}
		attributes["type"] = strings.Join(shorthand, " ")
	}
	if len(attributes) == 0 {
		return nil, nil
	}
	return attributes, nil
}

func mergeBinaryQuery(operator string, left, right AgentQuery) AgentQuery {
	operands := []AgentQuery{left, right}
	if left.Operator == operator {
		operands = append([]AgentQuery{}, left.Operands...)
		operands = append(operands, right)
	}
	return AgentQuery{Operator: operator, Operands: operands}
}

func attachContains(parent, child AgentQuery) AgentQuery {
	switch parent.Operator {
	case "AND", "OR":
		operands := make([]AgentQuery, len(parent.Operands))
		for i, operand := range parent.Operands {
			operands[i] = attachContains(operand, child)
		}
		parent.Operands = operands
		return parent
	}

	if parent.Contains == nil {
		parent.Contains = &child
		return parent
	}
	nested := attachContains(*parent.Contains, child)
	parent.Contains = &nested
	return parent
}

func (p *dslParser) matchOperator(operator string) bool {
	if !p.hasNext() {
		return false
	}
	token := p.peek()
	if token.kind != dslTokenOperator || token.value != operator {
		return false
	}
	p.pos++
	return true
}

func (p *dslParser) matchKind(kind dslTokenKind) bool {
	if !p.hasNext() || p.peek().kind != kind {
		return false
	}
	p.pos++
	return true
}

func (p *dslParser) hasNext() bool {
	return p.pos < len(p.tokens)
}

func (p *dslParser) peek() dslToken {
	return p.tokens[p.pos]
}

func (p *dslParser) next() dslToken {
	token := p.tokens[p.pos]
	p.pos++
	return token
}
