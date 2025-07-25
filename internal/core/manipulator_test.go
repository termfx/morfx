package core

import (
	"regexp"
	"testing"

	"github.com/garaekz/fileman/internal/model"
)

func TestManipulator_ApplyWithContext(t *testing.T) {
	cfg := model.ModificationConfig{
		Pattern:     "world",
		Replacement: "Go",
		Operation:   model.OpReplace,
		Context: &model.Context{
			Before: "^line before$",
		},
	}
	manip := NewManipulator(cfg)
	original := "line before\nhello world"
	modified, changes, err := manip.Apply(original)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if modified != "line before\nhello Go" {
		t.Errorf("Expected 'line before\\nhello Go', got '%s'", modified)
	}
	if len(changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changes))
	}
}

func TestManipulator_GetCached(t *testing.T) {
	// This function is not directly testable as it's an internal helper
	// and relies on a global cache. We'll test it indirectly through
	// functions that use it, or if it becomes public.
}

func TestManipulator_cacheKey(t *testing.T) {
	// This function is not directly testable as it's an internal helper
	// and relies on a global cache. We'll test it indirectly through
	// functions that use it, or if it becomes public.
}

func TestManipulator_boolToStr(t *testing.T) {
	if boolToStr(true) != "1" {
		t.Errorf("Expected '1' for true, got '%s'", boolToStr(true))
	}
	if boolToStr(false) != "0" {
		t.Errorf("Expected '0' for false, got '%s'", boolToStr(false))
	}
}

func TestManipulator_findMatchesBytes(t *testing.T) {
	cfg := model.ModificationConfig{
		Pattern: "test",
	}
	manip := NewManipulator(cfg)
	content := []byte("this is a test string with test")
	matches, err := manip.findMatchesBytes(content)
	if err != nil {
		t.Fatalf("findMatchesBytes failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}
	if matches[0][0] != 10 || matches[0][1] != 14 {
		t.Errorf("Expected first match at 10-14, got %d-%d", matches[0][0], matches[0][1])
	}
	if matches[1][0] != 27 || matches[1][1] != 31 {
		t.Errorf("Expected second match at 27-31, got %d-%d", matches[1][0], matches[1][1])
	}
}

func TestManipulator_ApplyAST(t *testing.T) {
	cfg := model.ModificationConfig{
		Pattern:     "(function_declaration name: (identifier) @name (#eq? @name \"main\"))",
		Replacement: "newMain",
		Operation:   model.OpReplace,
		UseAST:      true,
		Lang:        "go",
	}
	manip := NewManipulator(cfg)
	original := `package main

func main() {
	println("hello")
}`
	expected := `package main

func newMain() {
	println("hello")
}`
	modified, changes, err := manip.Apply(original)
	if err != nil {
		t.Fatalf("ApplyAST failed: %v", err)
	}
	if modified != expected {
		t.Errorf("Expected '%s', got '%s'", expected, modified)
	}
	if len(changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changes))
	}

	// Test with no matches
	cfgNoMatch := model.ModificationConfig{
		Pattern: "(function_declaration name: (identifier) @name (#eq? @name \"nonexistent\"))",
		UseAST:  true,
		Lang:    "go",
	}
	manipNoMatch := NewManipulator(cfgNoMatch)
	originalNoMatch := `package main

func main() {
	println("hello")
}`
	modifiedNoMatch, changesNoMatch, errNoMatch := manipNoMatch.Apply(originalNoMatch)
	if errNoMatch != nil {
		t.Fatalf("ApplyAST (no match) failed: %v", errNoMatch)
	}
	if modifiedNoMatch != originalNoMatch {
		t.Errorf("Expected no change, got '%s'", modifiedNoMatch)
	}
	if len(changesNoMatch) != 0 {
		t.Errorf("Expected 0 changes, got %d", len(changesNoMatch))
	}

	// Test with invalid pattern
	cfgInvalidPattern := model.ModificationConfig{
		Pattern: "(", // Invalid Tree-sitter query
		UseAST:  true,
		Lang:    "go",
	}
	manipInvalidPattern := NewManipulator(cfgInvalidPattern)
	_, _, errInvalidPattern := manipInvalidPattern.Apply(original)
	if errInvalidPattern == nil {
		t.Error("Expected error for invalid AST pattern, got nil")
	}
}

func TestManipulator_preserveIndentation(t *testing.T) {
	original1 := `  line1
    line2`
	replacement1 := `new_line1
new_line2`
	expected1 := `  new_line1
  new_line2`
	result1 := preserveIndentation(original1, 2, replacement1)
	if result1 != expected1 {
		t.Errorf("Expected '%s', got '%s'", expected1, result1)
	}

	original2 := `line1
    line2`
	replacement2 := `new_line1
new_line2`
	expected2 := `new_line1
new_line2`
	result2 := preserveIndentation(original2, 0, replacement2)
	if result2 != expected2 {
		t.Errorf("Expected '%s', got '%s'", expected2, result2)
	}
}

func TestManipulator_buildMatcher(t *testing.T) {
	// Test NewRegex path
	cfgRegex := model.ModificationConfig{Pattern: "abc", UseAST: false}
	mtRegex, err := buildMatcher(cfgRegex)
	if err != nil {
		t.Fatalf("buildMatcher (regex) failed: %v", err)
	}
	if mtRegex == nil {
		t.Error("Expected a regex matcher, got nil")
	}

	// Test NewAST path
	cfgAST := model.ModificationConfig{Pattern: "(function_declaration)", UseAST: true, Lang: "go"}
	mtAST, err := buildMatcher(cfgAST)
	if err != nil {
		t.Fatalf("buildMatcher (AST) failed: %v", err)
	}
	if mtAST == nil {
		t.Error("Expected an AST matcher, got nil")
	}

	// Test NewAST path with empty lang
	cfgASTEmptyLang := model.ModificationConfig{Pattern: "(function_declaration)", UseAST: true, Lang: ""}
	mtASTEmptyLang, err := buildMatcher(cfgASTEmptyLang)
	if err != nil {
		t.Fatalf("buildMatcher (AST empty lang) failed: %v", err)
	}
	if mtASTEmptyLang == nil {
		t.Error("Expected an AST matcher with empty lang, got nil")
	}
}

func TestManipulator_applyMatchesOnOriginal(t *testing.T) {
	cfg := model.ModificationConfig{
		Pattern:             "world",
		Replacement:         "Go",
		Operation:           model.OpReplace,
		NormalizeWhitespace: true,
	}
	manip := NewManipulator(cfg)
	original := "hello world"
	re := regexp.MustCompile("world")
	matches := re.FindAllStringSubmatchIndex(original, -1)
	occ := model.OccurrenceSpec{Max: -1}

	modified, changes, err := manip.applyMatchesOnOriginal(original, re, matches, occ)
	if err != nil {
		t.Fatalf("applyMatchesOnOriginal failed: %v", err)
	}
	if modified != "hello Go" {
		t.Errorf("Expected 'hello Go', got '%s'", modified)
	}
	if len(changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changes))
	}

	// Test with occ.Max limiting matches
	original2 := "one two three four"
	re2 := regexp.MustCompile("\\w+")
	matches2 := re2.FindAllStringSubmatchIndex(original2, -1)
	occ2 := model.OccurrenceSpec{Max: 2} // Limit to 2 matches

	modified2, changes2, err2 := manip.applyMatchesOnOriginal(original2, re2, matches2, occ2)
	if err2 != nil {
		t.Fatalf("applyMatchesOnOriginal (limited) failed: %v", err2)
	}
	// The replacement is "Go", so "one" and "two" should be replaced
	if modified2 != "Go Go three four" {
		t.Errorf("Expected 'Go Go three four', got '%s'", modified2)
	}
	if len(changes2) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(changes2))
	}

	// Test with no matches
	original3 := "no match here"
	re3 := regexp.MustCompile("xyz")
	matches3 := re3.FindAllStringSubmatchIndex(original3, -1)
	occ3 := model.OccurrenceSpec{Max: -1}

	modified3, changes3, err3 := manip.applyMatchesOnOriginal(original3, re3, matches3, occ3)
	if err3 != nil {
		t.Fatalf("applyMatchesOnOriginal (no matches) failed: %v", err3)
	}
	if modified3 != original3 {
		t.Errorf("Expected no change, got '%s'", modified3)
	}
	if len(changes3) != 0 {
		t.Errorf("Expected 0 changes, got %d", len(changes3))
	}

	// Test OpInsertBefore
	cfgInsertBefore := model.ModificationConfig{
		Pattern:             "world",
		Replacement:         "hello ",
		Operation:           model.OpInsertBefore,
		NormalizeWhitespace: true,
	}
	manipInsertBefore := NewManipulator(cfgInsertBefore)
	originalInsertBefore := "world"
	reInsertBefore := regexp.MustCompile("world")
	matchesInsertBefore := reInsertBefore.FindAllStringSubmatchIndex(originalInsertBefore, -1)
	occInsertBefore := model.OccurrenceSpec{Max: -1}

	modifiedInsertBefore, changesInsertBefore, errInsertBefore := manipInsertBefore.applyMatchesOnOriginal(originalInsertBefore, reInsertBefore, matchesInsertBefore, occInsertBefore)
	if errInsertBefore != nil {
		t.Fatalf("applyMatchesOnOriginal (insert before) failed: %v", errInsertBefore)
	}
	if modifiedInsertBefore != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", modifiedInsertBefore)
	}
	if len(changesInsertBefore) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changesInsertBefore))
	}

	// Test OpInsertAfter
	cfgInsertAfter := model.ModificationConfig{
		Pattern:             "hello",
		Replacement:         " world",
		Operation:           model.OpInsertAfter,
		NormalizeWhitespace: true,
	}
	manipInsertAfter := NewManipulator(cfgInsertAfter)
	originalInsertAfter := "hello"
	reInsertAfter := regexp.MustCompile("hello")
	matchesInsertAfter := reInsertAfter.FindAllStringSubmatchIndex(originalInsertAfter, -1)
	occInsertAfter := model.OccurrenceSpec{Max: -1}

	modifiedInsertAfter, changesInsertAfter, errInsertAfter := manipInsertAfter.applyMatchesOnOriginal(originalInsertAfter, reInsertAfter, matchesInsertAfter, occInsertAfter)
	if errInsertAfter != nil {
		t.Fatalf("applyMatchesOnOriginal (insert after) failed: %v", errInsertAfter)
	}
	if modifiedInsertAfter != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", modifiedInsertAfter)
	}
	if len(changesInsertAfter) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changesInsertAfter))
	}

	// Test OpDelete
	cfgDelete := model.ModificationConfig{
		Pattern:             "world",
		Operation:           model.OpDelete,
		NormalizeWhitespace: true,
	}
	manipDelete := NewManipulator(cfgDelete)
	originalDelete := "hello world"
	reDelete := regexp.MustCompile("world")
	matchesDelete := reDelete.FindAllStringSubmatchIndex(originalDelete, -1)
	occDelete := model.OccurrenceSpec{Max: -1}

	modifiedDelete, changesDelete, errDelete := manipDelete.applyMatchesOnOriginal(originalDelete, reDelete, matchesDelete, occDelete)
	if errDelete != nil {
		t.Fatalf("applyMatchesOnOriginal (delete) failed: %v", errDelete)
	}
	if modifiedDelete != "hello " {
		t.Errorf("Expected 'hello ', got '%s'", modifiedDelete)
	}
	if len(changesDelete) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changesDelete))
	}
}

func TestManipulator_applyWithContextOnOriginal(t *testing.T) {
	cfg := model.ModificationConfig{
		Pattern:             "world",
		Replacement:         "Go",
		Operation:           model.OpReplace,
		NormalizeWhitespace: true,
		Context: &model.Context{
			Before: "^hello$",
		},
	}
	manip := NewManipulator(cfg)
	original := "hello\nworld"
	re := regexp.MustCompile("world")
	matches := re.FindAllStringSubmatchIndex(original, -1)
	occ := model.OccurrenceSpec{Max: -1}

	modified, changes, err := manip.applyWithContextOnOriginal(original, re, matches, occ)
	if err != nil {
		t.Fatalf("applyWithContextOnOriginal failed: %v", err)
	}
	if modified != "hello\nGo" {
		t.Errorf("Expected 'hello\\nGo', got '%s'", modified)
	}
	if len(changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changes))
	}

	// Test with context not matching
	cfg2 := model.ModificationConfig{
		Pattern:             "world",
		Replacement:         "Go",
		Operation:           model.OpReplace,
		NormalizeWhitespace: true,
		Context: &model.Context{
			Before: "^nomatch$",
		},
	}
	manip2 := NewManipulator(cfg2)
	original2 := "hello\nworld"
	re2 := regexp.MustCompile("world")
	matches2 := re2.FindAllStringSubmatchIndex(original2, -1)
	occ2 := model.OccurrenceSpec{Max: -1}

	modified2, changes2, err2 := manip2.applyWithContextOnOriginal(original2, re2, matches2, occ2)
	if err2 != nil {
		t.Fatalf("applyWithContextOnOriginal (no match) failed: %v", err2)
	}
	if modified2 != original2 {
		t.Errorf("Expected no change, got '%s'", modified2)
	}
	if len(changes2) != 0 {
		t.Errorf("Expected 0 changes, got %d", len(changes2))
	}

	// Test with no matches
	original3 := "no match here"
	re3 := regexp.MustCompile("xyz")
	matches3 := re3.FindAllStringSubmatchIndex(original3, -1)
	occ3 := model.OccurrenceSpec{Max: -1}

	modified3, changes3, err3 := manip.applyWithContextOnOriginal(original3, re3, matches3, occ3)
	if err3 != nil {
		t.Fatalf("applyWithContextOnOriginal (no matches) failed: %v", err3)
	}
	if modified3 != original3 {
		t.Errorf("Expected no change, got '%s'", modified3)
	}
	if len(changes3) != 0 {
		t.Errorf("Expected 0 changes, got %d", len(changes3))
	}

	// Test OpInsertBefore
	cfgInsertBefore := model.ModificationConfig{
		Pattern:             "world",
		Replacement:         "hello ",
		Operation:           model.OpInsertBefore,
		NormalizeWhitespace: true,
		Context: &model.Context{
			Before: "^hello$",
		},
	}
	manipInsertBefore := NewManipulator(cfgInsertBefore)
	originalInsertBefore := "hello\nworld"
	reInsertBefore := regexp.MustCompile("world")
	matchesInsertBefore := reInsertBefore.FindAllStringSubmatchIndex(originalInsertBefore, -1)
	occInsertBefore := model.OccurrenceSpec{Max: -1}

	modifiedInsertBefore, changesInsertBefore, errInsertBefore := manipInsertBefore.applyWithContextOnOriginal(originalInsertBefore, reInsertBefore, matchesInsertBefore, occInsertBefore)
	if errInsertBefore != nil {
		t.Fatalf("applyWithContextOnOriginal (insert before) failed: %v", errInsertBefore)
	}
	if modifiedInsertBefore != "hello\nhello world" {
		t.Errorf("Expected 'hello\\nhello world', got '%s'", modifiedInsertBefore)
	}
	if len(changesInsertBefore) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changesInsertBefore))
	}

	// Test OpInsertAfter
	cfgInsertAfter := model.ModificationConfig{
		Pattern:             "hello",
		Replacement:         " world",
		Operation:           model.OpInsertAfter,
		NormalizeWhitespace: true,
		Context: &model.Context{
			After: "",
		},
	}
	manipInsertAfter := NewManipulator(cfgInsertAfter)
	originalInsertAfter := "hello"
	reInsertAfter := regexp.MustCompile("hello")
	matchesInsertAfter := reInsertAfter.FindAllStringSubmatchIndex(originalInsertAfter, -1)
	occInsertAfter := model.OccurrenceSpec{Max: -1}

	modifiedInsertAfter, changesInsertAfter, errInsertAfter := manipInsertAfter.applyWithContextOnOriginal(originalInsertAfter, reInsertAfter, matchesInsertAfter, occInsertAfter)
	if errInsertAfter != nil {
		t.Fatalf("applyWithContextOnOriginal (insert after) failed: %v", errInsertAfter)
	}
	if modifiedInsertAfter != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", modifiedInsertAfter)
	}
	if len(changesInsertAfter) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changesInsertAfter))
	}

	// Test OpDelete
	cfgDelete := model.ModificationConfig{
		Pattern:             "world",
		Operation:           model.OpDelete,
		NormalizeWhitespace: true,
		Context: &model.Context{
			Before: "^hello$",
		},
	}
	manipDelete := NewManipulator(cfgDelete)
	originalDelete := "hello\nworld"
	reDelete := regexp.MustCompile("world")
	matchesDelete := reDelete.FindAllStringSubmatchIndex(originalDelete, -1)
	occDelete := model.OccurrenceSpec{Max: -1}

	modifiedDelete, changesDelete, errDelete := manipDelete.applyWithContextOnOriginal(originalDelete, reDelete, matchesDelete, occDelete)
	if errDelete != nil {
		t.Fatalf("applyWithContextOnOriginal (delete) failed: %v", errDelete)
	}
	if modifiedDelete != "hello\n" {
		t.Errorf("Expected 'hello\\n', got '%s'", modifiedDelete)
	}
	if len(changesDelete) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changesDelete))
	}
}

func TestManipulator_applySimple(t *testing.T) {
	cfg := model.ModificationConfig{
		Pattern:     "world",
		Replacement: "Go",
		Operation:   model.OpReplace,
	}
	manip := NewManipulator(cfg)
	original := "hello world"
	re := regexp.MustCompile("world")
	occ := model.OccurrenceSpec{Max: -1}

	modified, changes, err := manip.applySimple(original, re, occ)
	if err != nil {
		t.Fatalf("applySimple failed: %v", err)
	}
	if modified != "hello Go" {
		t.Errorf("Expected 'hello Go', got '%s'", modified)
	}
	if len(changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changes))
	}

	// Test with occ.Max limiting matches
	original2 := "one two three four"
	re2 := regexp.MustCompile("\\w+")
	occ2 := model.OccurrenceSpec{Max: 2} // Limit to 2 matches

	modified2, changes2, err2 := manip.applySimple(original2, re2, occ2)
	if err2 != nil {
		t.Fatalf("applySimple (limited) failed: %v", err2)
	}
	// The replacement is "Go", so "one" and "two" should be replaced
	if modified2 != "Go Go three four" {
		t.Errorf("Expected 'Go Go three four', got '%s'", modified2)
	}
	if len(changes2) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(changes2))
	}

	// Test with no matches
	original3 := "no match here"
	re3 := regexp.MustCompile("xyz")
	occ3 := model.OccurrenceSpec{Max: -1}

	modified3, changes3, err3 := manip.applySimple(original3, re3, occ3)
	if err3 != nil {
		t.Fatalf("applySimple (no matches) failed: %v", err3)
	}
	if modified3 != original3 {
		t.Errorf("Expected no change, got '%s'", modified3)
	}
	if len(changes3) != 0 {
		t.Errorf("Expected 0 changes, got %d", len(changes3))
	}

	// Test OpInsertBefore
	cfgInsertBefore := model.ModificationConfig{
		Pattern:     "world",
		Replacement: "hello ",
		Operation:   model.OpInsertBefore,
	}
	manipInsertBefore := NewManipulator(cfgInsertBefore)
	originalInsertBefore := "world"
	reInsertBefore := regexp.MustCompile("world")
	occInsertBefore := model.OccurrenceSpec{Max: -1}

	modifiedInsertBefore, changesInsertBefore, errInsertBefore := manipInsertBefore.applySimple(originalInsertBefore, reInsertBefore, occInsertBefore)
	if errInsertBefore != nil {
		t.Fatalf("applySimple (insert before) failed: %v", errInsertBefore)
	}
	if modifiedInsertBefore != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", modifiedInsertBefore)
	}
	if len(changesInsertBefore) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changesInsertBefore))
	}

	// Test OpInsertAfter
	cfgInsertAfter := model.ModificationConfig{
		Pattern:     "hello",
		Replacement: " world",
		Operation:   model.OpInsertAfter,
	}
	manipInsertAfter := NewManipulator(cfgInsertAfter)
	originalInsertAfter := "hello"
	reInsertAfter := regexp.MustCompile("hello")
	occInsertAfter := model.OccurrenceSpec{Max: -1}

	modifiedInsertAfter, changesInsertAfter, errInsertAfter := manipInsertAfter.applySimple(originalInsertAfter, reInsertAfter, occInsertAfter)
	if errInsertAfter != nil {
		t.Fatalf("applySimple (insert after) failed: %v", errInsertAfter)
	}
	if modifiedInsertAfter != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", modifiedInsertAfter)
	}
	if len(changesInsertAfter) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changesInsertAfter))
	}

	// Test OpDelete
	cfgDelete := model.ModificationConfig{
		Pattern:   "world",
		Operation: model.OpDelete,
	}
	manipDelete := NewManipulator(cfgDelete)
	originalDelete := "hello world"
	reDelete := regexp.MustCompile("world")
	occDelete := model.OccurrenceSpec{Max: -1}

	modifiedDelete, changesDelete, errDelete := manipDelete.applySimple(originalDelete, reDelete, occDelete)
	if errDelete != nil {
		t.Fatalf("applySimple (delete) failed: %v", errDelete)
	}
	if modifiedDelete != "hello " {
		t.Errorf("Expected 'hello ', got '%s'", modifiedDelete)
	}
	if len(changesDelete) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changesDelete))
	}
}

func TestManipulator_dedupeInsert(t *testing.T) {
	buf := []byte("abcdefg")
	insert := []byte("xyz")

	// Test insert before, no dedupe
	if !dedupeInsert(buf, 3, insert, true) {
		t.Errorf("Expected true for no dedupe insert before")
	}

	// Test insert before, with dedupe
	buf = []byte("abxyzcdefg")
	if dedupeInsert(buf, 5, insert, true) {
		t.Errorf("Expected false for dedupe insert before")
	}

	// Test insert after, no dedupe
	buf = []byte("abcdefg")
	if !dedupeInsert(buf, 3, insert, false) {
		t.Errorf("Expected true for no dedupe insert after")
	}

	// Test insert after, with dedupe
	buf = []byte("abcxyzdefg")
	if dedupeInsert(buf, 3, insert, false) {
		t.Errorf("Expected false for dedupe insert after")
	}
}

func TestManipulator_Apply_NoNormalize(t *testing.T) {
	cfg := model.ModificationConfig{
		Pattern:             "world",
		Replacement:         "Go",
		Operation:           model.OpReplace,
		NormalizeWhitespace: false,
	}
	manip := NewManipulator(cfg)
	original := "hello world"
	modified, changes, err := manip.Apply(original)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if modified != "hello Go" {
		t.Errorf("Expected 'hello Go', got '%s'", modified)
	}
	if len(changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changes))
	}
}

func TestManipulator_Apply_LiteralPattern(t *testing.T) {
	cfg := model.ModificationConfig{
		Pattern:        "hello.world",
		Replacement:    "hello_world",
		Operation:      model.OpReplace,
		LiteralPattern: true,
	}
	manip := NewManipulator(cfg)
	original := "hello.world"
	modified, changes, err := manip.Apply(original)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if modified != "hello_world" {
		t.Errorf("Expected 'hello_world', got '%s'", modified)
	}
	if len(changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changes))
	}
}

func TestManipulator_Apply_Multiline(t *testing.T) {
	cfg := model.ModificationConfig{
		Pattern:     "^line2$",
		Replacement: "newLine2",
		Operation:   model.OpReplace,
		Multiline:   true,
	}
	manip := NewManipulator(cfg)
	original := "line1\nline2\nline3"
	expected := "line1\nnewLine2\nline3"
	modified, changes, err := manip.Apply(original)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if modified != expected {
		t.Errorf("Expected '%s', got '%s'", expected, modified)
	}
	if len(changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changes))
	}
}

func TestManipulator_Apply_DotAll(t *testing.T) {
	cfg := model.ModificationConfig{
		Pattern:     "line1.*line3",
		Replacement: "replaced",
		Operation:   model.OpReplace,
		DotAll:      true,
	}
	manip := NewManipulator(cfg)
	original := "line1\nline2\nline3"
	expected := "replaced"
	modified, changes, err := manip.Apply(original)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if modified != expected {
		t.Errorf("Expected '%s', got '%s'", expected, modified)
	}
	if len(changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changes))
	}
}

func TestManipulator_Apply_NormalizeWhitespaceLiteralPattern(t *testing.T) {
	cfg := model.ModificationConfig{
		Pattern:             "hello   world",
		Replacement:         "hello_world",
		Operation:           model.OpReplace,
		NormalizeWhitespace: true,
		LiteralPattern:      true,
	}
	manip := NewManipulator(cfg)
	original := "hello   world"
	modified, changes, err := manip.Apply(original)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if modified != "hello_world" {
		t.Errorf("Expected 'hello_world', got '%s'", modified)
	}
	if len(changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changes))
	}
}

func TestManipulator_GetCached_CacheHit(t *testing.T) {
	// Prime the cache
	cfg1 := model.ModificationConfig{Pattern: "test1", UseAST: false}
	_, _ = GetCached(cfg1)

	// Should hit cache
	cfg2 := model.ModificationConfig{Pattern: "test1", UseAST: false}
	mt, err := GetCached(cfg2)
	if err != nil {
		t.Fatalf("GetCached failed: %v", err)
	}
	if mt == nil {
		t.Error("Expected a matcher, got nil")
	}
}

func TestManipulator_GetCached_CacheMiss(t *testing.T) {
	// Ensure cache is clean for this test (not strictly necessary due to map behavior, but good practice)
	mu.Lock()
	data = make(map[string]*entry)
	mu.Unlock()

	cfg := model.ModificationConfig{Pattern: "test2", UseAST: false}
	mt, err := GetCached(cfg)
	if err != nil {
		t.Fatalf("GetCached failed: %v", err)
	}
	if mt == nil {
		t.Error("Expected a matcher, got nil")
	}
}

func TestManipulator_buildMatcher_NewRegex(t *testing.T) {
	cfg := model.ModificationConfig{Pattern: "abc", UseAST: false}
	mt, err := buildMatcher(cfg)
	if err != nil {
		t.Fatalf("buildMatcher (regex) failed: %v", err)
	}
	if mt == nil {
		t.Error("Expected a regex matcher, got nil")
	}
}

func TestManipulator_buildMatcher_NewAST(t *testing.T) {
	// Test NewAST path
	cfgAST := model.ModificationConfig{Pattern: "(function_declaration)", UseAST: true, Lang: "go"}
	mtAST, err := buildMatcher(cfgAST)
	if err != nil {
		t.Fatalf("buildMatcher (AST) failed: %v", err)
	}
	if mtAST == nil {
		t.Error("Expected an AST matcher, got nil")
	}

	// Test NewAST path with empty lang
	cfgASTEmptyLang := model.ModificationConfig{Pattern: "(function_declaration)", UseAST: true, Lang: ""}
	mtASTEmptyLang, err := buildMatcher(cfgASTEmptyLang)
	if err != nil {
		t.Fatalf("buildMatcher (AST empty lang) failed: %v", err)
	}
	if mtASTEmptyLang == nil {
		t.Error("Expected an AST matcher with empty lang, got nil")
	}
}

func TestManipulator_parseOccurrences(t *testing.T) {
	// Test "all"
	occ, err := parseOccurrences("all")
	if err != nil {
		t.Fatalf("parseOccurrences failed: %v", err)
	}
	if occ.Max != -1 {
		t.Errorf("Expected Max -1, got %d", occ.Max)
	}

	// Test "first"
	occ, err = parseOccurrences("first")
	if err != nil {
		t.Fatalf("parseOccurrences failed: %v", err)
	}
	if occ.Max != 1 {
		t.Errorf("Expected Max 1, got %d", occ.Max)
	}

	// Test a number
	occ, err = parseOccurrences("5")
	if err != nil {
		t.Fatalf("parseOccurrences failed: %v", err)
	}
	if occ.Max != 5 {
		t.Errorf("Expected Max 5, got %d", occ.Max)
	}

	// Test invalid number
	_, err = parseOccurrences("abc")
	if err == nil {
		t.Error("Expected error for invalid occurrences, got nil")
	}

	// Test zero
	_, err = parseOccurrences("0")
	if err == nil {
		t.Error("Expected error for zero occurrences, got nil")
	}
}

func TestManipulator_applyMatches_InvalidOperation(t *testing.T) {
	cfg := model.ModificationConfig{
		Pattern:     "world",
		Replacement: "Go",
		Operation:   model.Operation("invalid"), // Invalid operation
	}
	manip := NewManipulator(cfg)
	original := "hello world"
	re := regexp.MustCompile("world")
	matches := re.FindAllStringSubmatchIndex(original, -1)

	_, _, err := manip.applyMatches(original, re, matches)
	if err == nil {
		t.Error("Expected error for invalid operation, got nil")
	}
}
