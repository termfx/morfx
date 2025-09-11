package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// OAuth flow handlers for complete authentication

// handleOAuthAuth initiates OAuth flow
func (s *HTTPServer) handleOAuthAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Only GET method allowed")
		return
	}

	// CRITICAL: ChatGPT passes its own state (UUID) - we MUST reuse it
	chatgptState := r.URL.Query().Get("state")
	chatgptChallenge := r.URL.Query().Get("code_challenge")
	challengeMethod := r.URL.Query().Get("code_challenge_method")

	s.debugLog("=== OAUTH AUTH DEBUG ===")
	s.debugLog("Raw URL: %s", r.URL.String())
	s.debugLog("All query params: %v", r.URL.Query())
	s.debugLog("Extracted state: '%s'", chatgptState)
	s.debugLog("ChatGPT code_challenge: '%s'", chatgptChallenge)
	s.debugLog("Challenge method: '%s'", challengeMethod)

	if chatgptState == "" {
		s.debugLog("ERROR: ChatGPT state is empty - this breaks the flow!")
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing state parameter from ChatGPT")
		return
	}

	if chatgptChallenge == "" {
		s.debugLog("ERROR: ChatGPT code_challenge is empty - this breaks PKCE!")
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing code_challenge from ChatGPT")
		return
	}

	// Parse query parameters
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = string(s.oauthConfig.Provider) // Default to configured provider
	}

	// Parse scopes
	scopesParam := r.URL.Query().Get("scopes")
	scopes := s.oauthConfig.Scopes // Default scopes
	if scopesParam != "" {
		scopes = strings.Split(scopesParam, ",")
	}

	// For Google, ensure minimum OIDC scopes are present
	if provider == string(OAuthProviderGoogle) {
		minScopes := []string{"openid", "email", "profile"}
		for _, minScope := range minScopes {
			found := slices.Contains(scopes, minScope)
			if !found {
				scopes = append(scopes, minScope)
			}
		}
	}

	// Create session using ChatGPT's PKCE parameters (critical for MCP flow)
	session, err := s.sessionStore.CreateSessionWithChatGPTPKCE(
		chatgptState,
		provider,
		scopes,
		chatgptChallenge,
		challengeMethod,
	)
	if err != nil {
		s.debugLog("Failed to create OAuth session: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Build authorization URL
	authURL, err := s.buildAuthURL(session)
	if err != nil {
		s.debugLog("Failed to build auth URL: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to build authorization URL")
		return
	}

	s.debugLog("OAuth flow initiated for provider %s, ChatGPT state: %s", provider, chatgptState[:8])
	s.debugLog("Using ChatGPT PKCE - Challenge: %s..., Method: %s", chatgptChallenge[:16], challengeMethod)

	// CRITICAL: MCP requires 302 redirect, not JSON response
	s.debugLog("Redirecting to auth URL: %s", authURL)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleOAuthCallback handles OAuth provider redirect
func (s *HTTPServer) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Only GET method allowed")
		return
	}

	// Extract parameters
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	errorParam := r.URL.Query().Get("error")

	// Check for OAuth errors
	if errorParam != "" {
		errorDesc := r.URL.Query().Get("error_description")
		s.debugLog("OAuth error: %s - %s", errorParam, errorDesc)
		s.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("OAuth error: %s", errorDesc))
		return
	}

	if state == "" || code == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing state or code parameter")
		return
	}

	// Validate and consume session
	session, exists := s.sessionStore.ConsumeSession(state)
	if !exists {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid or expired state")
		return
	}

	s.debugLog("OAuth callback received for state: %s", state[:8])

	// Exchange code for token
	token, err := s.exchangeCodeForToken(session, code)
	if err != nil {
		s.debugLog("Token exchange failed: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Token exchange failed")
		return
	}

	// Validate token with our validator
	var claims jwt.MapClaims
	var validationErr error

	// For Google OAuth, try id_token first (OIDC), fallback to access_token
	if token.Extra("id_token") != nil {
		idToken := token.Extra("id_token").(string)
		s.debugLog("Validating Google id_token (OIDC)")
		claims, validationErr = s.oauthValidator.ValidateToken(idToken)
	} else {
		s.debugLog("No id_token found, validating access_token")
		claims, validationErr = s.oauthValidator.ValidateToken(token.AccessToken)
	}

	if validationErr != nil {
		s.debugLog("Token validation failed: %v", validationErr)
		s.writeErrorResponse(w, http.StatusUnauthorized, "Invalid token received")
		return
	}

	subject := s.oauthValidator.GetSubject(claims)
	scopes := s.oauthValidator.ExtractScopes(claims)

	s.debugLog("OAuth flow completed for user: %s", subject)

	// Return successful response
	response := map[string]any{
		"success":      true,
		"access_token": token.AccessToken,
		"token_type":   token.TokenType,
		"expires_in":   int(time.Until(token.Expiry).Seconds()),
		"subject":      subject,
		"scopes":       scopes,
	}

	if token.RefreshToken != "" {
		response["refresh_token"] = token.RefreshToken
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleOAuthToken handles token exchange/refresh
func (s *HTTPServer) handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Only POST method allowed")
		return
	}

	s.debugLog("=== TOKEN ENDPOINT DEBUG ===")
	s.debugLog("Raw URL: %s", r.URL.String())
	s.debugLog("Headers: %v", r.Header)
	s.debugLog("Content-Type: %s", r.Header.Get("Content-Type"))

	// Parse form data
	if err := r.ParseForm(); err != nil {
		s.debugLog("ERROR: Failed to parse form data: %v", err)
		s.writeErrorResponse(w, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	s.debugLog("Form data: %v", r.Form)
	s.debugLog("POST body params:")
	for key, values := range r.Form {
		s.debugLog("  %s: %v", key, values)
	}

	grantType := r.Form.Get("grant_type")
	s.debugLog("Grant type: '%s'", grantType)

	switch grantType {
	case "authorization_code":
		s.debugLog("Processing authorization_code grant")
		s.handleAuthorizationCodeGrant(w, r)
	case "refresh_token":
		s.debugLog("Processing refresh_token grant")
		s.handleRefreshTokenGrant(w, r)
	default:
		s.debugLog("ERROR: Unsupported grant type: '%s'", grantType)
		s.writeErrorResponse(w, http.StatusBadRequest, "Unsupported grant type")
	}
}

// handleAuthorizationCodeGrant processes authorization code grant
func (s *HTTPServer) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.Form.Get("code")
	state := r.Form.Get("state")
	codeVerifier := r.Form.Get("code_verifier")

	s.debugLog("=== AUTHORIZATION CODE GRANT DEBUG ===")
	s.debugLog("Received code: '%s' (length: %d)", code[:min(len(code), 20)]+"...", len(code))
	s.debugLog("Received state: '%s'", state)
	s.debugLog("Received code_verifier: '%s'", codeVerifier)

	if code == "" {
		s.debugLog("ERROR: Missing required code parameter")
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing code parameter")
		return
	}

	// MCP FLOW: ChatGPT sends code_verifier directly, no session lookup needed
	if codeVerifier != "" {
		s.debugLog("MCP flow detected - using direct code_verifier from ChatGPT")
		token, err := s.exchangeCodeForTokenDirect(code, codeVerifier)
		if err != nil {
			s.debugLog("ERROR: Direct token exchange failed: %v", err)
			s.writeErrorResponse(w, http.StatusInternalServerError, "Token exchange failed")
			return
		}

		s.debugLog("Direct token exchange successful")
		s.writeTokenResponse(w, token)
		return
	}

	// FALLBACK: Traditional OAuth flow with session lookup
	if state == "" {
		s.debugLog("ERROR: Missing both state and code_verifier")
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing state or code_verifier")
		return
	}

	// Get session (don't consume yet in case of retries)
	session, exists := s.sessionStore.GetSession(state)
	if !exists {
		s.debugLog("ERROR: Session not found for state: '%s'", state)
		// Log all active sessions for debugging
		stats := s.sessionStore.Stats()
		s.debugLog("Session store stats: %v", stats)
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid or expired state")
		return
	}

	s.debugLog("Session found - Provider: %s, Scopes: %v", session.Provider, session.Scopes)
	s.debugLog("PKCE verifier: %s...", session.CodeVerifier[:min(len(session.CodeVerifier), 10)])

	// Exchange code for token
	token, err := s.exchangeCodeForToken(session, code)
	if err != nil {
		s.debugLog("ERROR: Token exchange failed: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Token exchange failed")
		return
	}

	s.debugLog("Token exchange successful - Token type: %s", token.TokenType)

	// Now consume the session
	s.sessionStore.ConsumeSession(state)

	// Return token response
	s.writeTokenResponse(w, token)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleRefreshTokenGrant processes refresh token grant
func (s *HTTPServer) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.Form.Get("refresh_token")
	if refreshToken == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing refresh token")
		return
	}

	// This would require implementing refresh logic with the OAuth provider
	// For now, return not implemented
	s.writeErrorResponse(w, http.StatusNotImplemented, "Refresh token grant not implemented")
}

// handleOAuthLogout clears OAuth session
func (s *HTTPServer) handleOAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Only POST method allowed")
		return
	}

	// Extract token from Authorization header
	auth := r.Header.Get("Authorization")
	if auth != "" && strings.HasPrefix(auth, "Bearer ") {
		// In a real implementation, you'd invalidate the token
		// For now, just return success
		s.debugLog("Logout requested")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Logged out successfully",
	})
}

// buildAuthURL constructs the OAuth authorization URL
func (s *HTTPServer) buildAuthURL(session *OAuthSession) (string, error) {
	config := s.getOAuth2Config()

	// Check if this is MCP flow with ChatGPT's PKCE
	if session.Metadata["mcp_flow"] == "true" {
		s.debugLog("Building MCP auth URL with ChatGPT's PKCE")
		authURL := config.AuthCodeURL(session.State,
			oauth2.SetAuthURLParam("code_challenge", session.CodeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", session.Metadata["challenge_method"]),
		)
		return authURL, nil
	}

	// Traditional OAuth flow with our own PKCE
	authURL := config.AuthCodeURL(session.State,
		oauth2.SetAuthURLParam("code_challenge", session.CodeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	return authURL, nil
}

// exchangeCodeForToken exchanges authorization code for access token (session-based)
func (s *HTTPServer) exchangeCodeForToken(session *OAuthSession, code string) (*oauth2.Token, error) {
	config := s.getOAuth2Config()
	ctx := context.Background()

	// Exchange with PKCE verifier
	token, err := config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", session.CodeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %w", err)
	}

	return token, nil
}

// exchangeCodeForTokenDirect exchanges code for token using direct code_verifier (MCP flow)
func (s *HTTPServer) exchangeCodeForTokenDirect(code, codeVerifier string) (*oauth2.Token, error) {
	config := s.getOAuth2Config()
	ctx := context.Background()

	s.debugLog("Direct token exchange - Code: %s..., Verifier: %s...",
		code[:min(len(code), 10)], codeVerifier[:min(len(codeVerifier), 10)])

	// Exchange with PKCE verifier from ChatGPT
	token, err := config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("direct code exchange failed: %w", err)
	}

	return token, nil
}

// getOAuth2Config builds OAuth2 config from OAuthConfig
func (s *HTTPServer) getOAuth2Config() *oauth2.Config {
	var endpoint oauth2.Endpoint
	var scopes []string

	// Configure endpoint and scopes based on provider
	switch s.oauthConfig.Provider {
	case OAuthProviderGoogle:
		// Use OIDC discovery instead of hardcoded endpoints
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth", // Updated v2 endpoint
			TokenURL: "https://oauth2.googleapis.com/token",
		}
		// CRITICAL: Google OIDC requires these minimum scopes
		scopes = []string{"openid", "email", "profile"}
		if len(s.oauthConfig.Scopes) > 0 {
			// Add configured scopes after mandatory ones
			scopes = append(scopes, s.oauthConfig.Scopes...)
		}
	case OAuthProviderGitHub:
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		}
		scopes = s.oauthConfig.Scopes
	case OAuthProviderOpenAI:
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://auth0.openai.com/authorize",
			TokenURL: "https://auth0.openai.com/oauth/token",
		}
		scopes = s.oauthConfig.Scopes
	default:
		// Custom provider - try to extract from issuer URL
		if s.oauthConfig.IssuerURL != "" {
			base := strings.TrimSuffix(s.oauthConfig.IssuerURL, "/")
			endpoint = oauth2.Endpoint{
				AuthURL:  base + "/authorize",
				TokenURL: base + "/oauth/token",
			}
		}
		scopes = s.oauthConfig.Scopes
	}

	return &oauth2.Config{
		ClientID:     s.oauthConfig.ClientID,
		ClientSecret: s.oauthConfig.ClientSecret,
		Endpoint:     endpoint,
		Scopes:       scopes,
		RedirectURL:  "https://chatgpt.com/connector_platform_oauth_redirect", // CRITICAL: MCP redirect
	}
}

// writeTokenResponse writes standardized token response
func (s *HTTPServer) writeTokenResponse(w http.ResponseWriter, token *oauth2.Token) {
	response := map[string]any{
		"access_token": token.AccessToken,
		"token_type":   token.TokenType,
		"expires_in":   int(time.Until(token.Expiry).Seconds()),
	}

	if token.RefreshToken != "" {
		response["refresh_token"] = token.RefreshToken
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getBaseURL extracts base URL from request
func (s *HTTPServer) getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	host := r.Host
	if host == "" {
		host = r.Header.Get("X-Forwarded-Host")
	}
	if host == "" {
		host = "localhost:8080" // fallback
	}

	return fmt.Sprintf("%s://%s", scheme, host)
}
