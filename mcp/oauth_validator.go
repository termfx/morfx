package mcp

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/patrickmn/go-cache"
)

// AuthMode represents the authentication mode
type AuthMode string

const (
	AuthModeNone   AuthMode = "none"   // --no-auth
	AuthModeAPIKey AuthMode = "apikey" // --api-key (default)
	AuthModeOAuth  AuthMode = "oauth"  // --oauth-provider
)

// OAuthProvider represents different OAuth providers
type OAuthProvider string

const (
	OAuthProviderOpenAI OAuthProvider = "openai"
	OAuthProviderGitHub OAuthProvider = "github"
	OAuthProviderGoogle OAuthProvider = "google"
	OAuthProviderAuth0  OAuthProvider = "auth0"
	OAuthProviderCustom OAuthProvider = "custom"
)

// OAuthConfig holds OAuth configuration
type OAuthConfig struct {
	Provider     OAuthProvider
	ClientID     string
	ClientSecret string
	IssuerURL    string
	JWKSURL      string
	Audience     string
	Scopes       []string

	// Provider-specific
	Domain string // For Auth0

	// Validation settings
	SkipIssuerCheck   bool
	SkipAudienceCheck bool
	AllowExpired      bool // DANGEROUS: only for testing
}

// OAuthValidator validates OAuth tokens
type OAuthValidator struct {
	config     *OAuthConfig
	verifier   *oidc.IDTokenVerifier
	provider   *oidc.Provider
	httpClient *http.Client

	// Caches
	tokenCache *cache.Cache
	keyCache   *cache.Cache

	debugLog func(format string, args ...any)
}

// NewOAuthValidator creates a new OAuth validator
func NewOAuthValidator(config *OAuthConfig, debug bool) (*OAuthValidator, error) {
	v := &OAuthValidator{
		config:     config,
		tokenCache: cache.New(5*time.Minute, 10*time.Minute),
		keyCache:   cache.New(1*time.Hour, 2*time.Hour),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	if debug {
		v.debugLog = func(format string, args ...any) {
			fmt.Printf("[OAuth] "+format+"\n", args...)
		}
	} else {
		v.debugLog = func(format string, args ...any) {}
	}

	// Configure based on provider
	if err := v.configureProvider(); err != nil {
		return nil, fmt.Errorf("failed to configure provider: %w", err)
	}

	// Initialize OIDC if possible
	if config.IssuerURL != "" {
		ctx := context.Background()
		provider, err := oidc.NewProvider(ctx, config.IssuerURL)
		if err != nil {
			v.debugLog("Failed to create OIDC provider: %v", err)
			// Don't fail, we can still validate JWTs manually
		} else {
			v.provider = provider

			// Configure verifier
			verifierConfig := &oidc.Config{
				ClientID:          config.ClientID,
				SkipClientIDCheck: config.ClientID == "",
				SkipIssuerCheck:   config.SkipIssuerCheck,
			}

			if !config.SkipAudienceCheck && config.Audience != "" {
				verifierConfig.ClientID = config.Audience
			}

			v.verifier = provider.Verifier(verifierConfig)
			v.debugLog("OIDC provider configured for %s", config.IssuerURL)
		}
	}

	return v, nil
}

// configureProvider sets up provider-specific configuration
func (v *OAuthValidator) configureProvider() error {
	switch v.config.Provider {
	case OAuthProviderOpenAI:
		if v.config.IssuerURL == "" {
			v.config.IssuerURL = "https://auth0.openai.com/"
		}
		if v.config.JWKSURL == "" {
			v.config.JWKSURL = "https://auth0.openai.com/.well-known/jwks.json"
		}
		if v.config.Audience == "" {
			v.config.Audience = "https://api.openai.com/v1"
		}

	case OAuthProviderGitHub:
		// GitHub Actions OIDC
		if v.config.IssuerURL == "" {
			v.config.IssuerURL = "https://token.actions.githubusercontent.com"
		}
		if v.config.JWKSURL == "" {
			v.config.JWKSURL = "https://token.actions.githubusercontent.com/.well-known/jwks"
		}

	case OAuthProviderGoogle:
		if v.config.IssuerURL == "" {
			v.config.IssuerURL = "https://accounts.google.com"
		}
		if v.config.JWKSURL == "" {
			v.config.JWKSURL = "https://www.googleapis.com/oauth2/v3/certs"
		}

	case OAuthProviderAuth0:
		if v.config.Domain == "" {
			return fmt.Errorf("Auth0 domain is required")
		}
		if v.config.IssuerURL == "" {
			v.config.IssuerURL = fmt.Sprintf("https://%s/", v.config.Domain)
		}
		if v.config.JWKSURL == "" {
			v.config.JWKSURL = fmt.Sprintf("https://%s/.well-known/jwks.json", v.config.Domain)
		}

	case OAuthProviderCustom:
		if v.config.IssuerURL == "" && v.config.JWKSURL == "" {
			return fmt.Errorf("custom provider requires issuer URL or JWKS URL")
		}
	}

	v.debugLog("Provider %s configured: issuer=%s", v.config.Provider, v.config.IssuerURL)
	return nil
}

// isJWT checks if a token is in JWT format (has exactly 2 dots)
func (v *OAuthValidator) isJWT(token string) bool {
	return strings.Count(token, ".") == 2
}

// ValidateToken validates an OAuth token
func (v *OAuthValidator) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	// Mask token for logging
	maskedToken := tokenString
	if len(tokenString) > 20 {
		maskedToken = tokenString[:10] + "..." + tokenString[len(tokenString)-10:]
	}
	v.debugLog("Validating token: %s", maskedToken)

	// Check cache first
	if cached, found := v.tokenCache.Get(tokenString); found {
		v.debugLog("Token found in cache")
		return cached.(jwt.MapClaims), nil
	}

	// Check if token is JWT format
	if !v.isJWT(tokenString) {
		v.debugLog("Token is not JWT format, checking for opaque token validation")

		// Handle opaque tokens (like Google access_token)
		if v.config.Provider == OAuthProviderGoogle {
			claims, err := v.validateGoogleOpaqueToken(tokenString)
			if err != nil {
				return nil, err
			}
			v.tokenCache.Set(tokenString, claims, cache.DefaultExpiration)
			return claims, nil
		}

		return nil, fmt.Errorf("non-JWT token not supported for provider: %s", v.config.Provider)
	}

	// Try OIDC validation for JWT tokens
	if v.verifier != nil {
		claims, err := v.validateWithOIDC(tokenString)
		if err == nil {
			v.tokenCache.Set(tokenString, claims, cache.DefaultExpiration)
			return claims, nil
		}
		v.debugLog("OIDC validation failed: %v, trying manual JWT", err)
	}

	// Fallback to manual JWT validation
	claims, err := v.validateJWT(tokenString)
	if err != nil {
		return nil, err
	}

	// Cache valid token
	v.tokenCache.Set(tokenString, claims, cache.DefaultExpiration)
	return claims, nil
}

// GoogleUserInfo represents Google userinfo response
type GoogleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
}

// validateGoogleOpaqueToken validates Google access_token using userinfo endpoint
func (v *OAuthValidator) validateGoogleOpaqueToken(accessToken string) (jwt.MapClaims, error) {
	v.debugLog("Validating Google opaque token via userinfo")

	// Call Google userinfo endpoint
	req, err := http.NewRequest("GET", "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("userinfo failed with status: %d", resp.StatusCode)
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo: %w", err)
	}

	// Convert to jwt.MapClaims format
	claims := jwt.MapClaims{
		"sub":            userInfo.Sub,
		"email":          userInfo.Email,
		"email_verified": userInfo.EmailVerified,
		"name":           userInfo.Name,
		"picture":        userInfo.Picture,
		"given_name":     userInfo.GivenName,
		"family_name":    userInfo.FamilyName,
		"iss":            "https://accounts.google.com",
		"token_type":     "access_token",
		"validated_at":   time.Now().Unix(),
	}

	v.debugLog("Google opaque token validated for user: %s", userInfo.Email)
	return claims, nil
}

// validateWithOIDC uses OIDC library for validation
func (v *OAuthValidator) validateWithOIDC(tokenString string) (jwt.MapClaims, error) {
	ctx := context.Background()
	idToken, err := v.verifier.Verify(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("OIDC verification failed: %w", err)
	}

	// Extract claims
	claims := jwt.MapClaims{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	v.debugLog("OIDC validation successful for subject: %v", claims["sub"])
	return claims, nil
}

// validateJWT manually validates a JWT token
func (v *OAuthValidator) validateJWT(tokenString string) (jwt.MapClaims, error) {
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// Validate algorithm
		switch token.Method.(type) {
		case *jwt.SigningMethodRSA:
			// RSA - most common for OAuth
		case *jwt.SigningMethodHMAC:
			// HMAC - for client secret validation
			if v.config.ClientSecret == "" {
				return nil, fmt.Errorf("HMAC validation requires client secret")
			}
			return []byte(v.config.ClientSecret), nil
		default:
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// For RSA, fetch public key from JWKS
		return v.getPublicKey(token)
	})
	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %w", err)
	}

	if !token.Valid && !v.config.AllowExpired {
		return nil, fmt.Errorf("token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to extract claims")
	}

	// Validate standard claims
	if !v.config.SkipIssuerCheck && v.config.IssuerURL != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != v.config.IssuerURL {
			return nil, fmt.Errorf("invalid issuer: got %s, want %s", iss, v.config.IssuerURL)
		}
	}

	if !v.config.SkipAudienceCheck && v.config.Audience != "" {
		if aud, ok := claims["aud"].(string); !ok || aud != v.config.Audience {
			// Check if audience is in array format
			if audArray, ok := claims["aud"].([]any); ok {
				found := false
				for _, a := range audArray {
					if a == v.config.Audience {
						found = true
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("invalid audience")
				}
			} else {
				return nil, fmt.Errorf("invalid audience: got %s, want %s", aud, v.config.Audience)
			}
		}
	}

	// Check expiration
	if !v.config.AllowExpired {
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return nil, fmt.Errorf("token expired")
			}
		}
	}

	v.debugLog("JWT validation successful")
	return claims, nil
}

// ExtractScopes extracts scopes from claims
func (v *OAuthValidator) ExtractScopes(claims jwt.MapClaims) []string {
	// Different providers use different claim names
	scopeClaimNames := []string{"scope", "scopes", "permissions", "scp"}

	for _, claimName := range scopeClaimNames {
		if scopeVal, ok := claims[claimName]; ok {
			switch s := scopeVal.(type) {
			case string:
				return strings.Split(s, " ")
			case []any:
				scopes := make([]string, 0, len(s))
				for _, scope := range s {
					if str, ok := scope.(string); ok {
						scopes = append(scopes, str)
					}
				}
				return scopes
			}
		}
	}

	return []string{}
}

// HasScope checks if the token has a specific scope
func (v *OAuthValidator) HasScope(claims jwt.MapClaims, requiredScope string) bool {
	scopes := v.ExtractScopes(claims)
	return slices.Contains(scopes, requiredScope)
}

// GetSubject extracts the subject (user ID) from claims
func (v *OAuthValidator) GetSubject(claims jwt.MapClaims) string {
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	// Fallback to email or username
	if email, ok := claims["email"].(string); ok {
		return email
	}
	if username, ok := claims["username"].(string); ok {
		return username
	}
	return ""
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string   `json:"kty"` // Key Type
	Kid string   `json:"kid"` // Key ID
	Use string   `json:"use"` // Public Key Use
	N   string   `json:"n"`   // RSA Modulus
	E   string   `json:"e"`   // RSA Exponent
	X5c []string `json:"x5c"` // X.509 Certificate Chain
}

// JWKSet represents a JSON Web Key Set
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// getPublicKey fetches RSA public key from JWKS
func (v *OAuthValidator) getPublicKey(token *jwt.Token) (any, error) {
	// Extract key ID from token header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("token missing kid header")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("jwk:%s", kid)
	if cached, found := v.keyCache.Get(cacheKey); found {
		v.debugLog("RSA key found in cache for kid: %s", kid)
		return cached, nil
	}

	// Determine JWKS URL
	jwksURL := v.config.JWKSURL
	if jwksURL == "" && v.config.IssuerURL != "" {
		jwksURL = strings.TrimSuffix(v.config.IssuerURL, "/") + "/.well-known/jwks.json"
	}
	if jwksURL == "" {
		return nil, fmt.Errorf("no JWKS URL configured")
	}

	// Fetch JWKS
	resp, err := v.httpClient.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("JWKS fetch failed with status: %d", resp.StatusCode)
	}

	// Parse JWKS
	var jwks JWKSet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	// Find matching key
	var jwk *JWK
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			jwk = &key
			break
		}
	}
	if jwk == nil {
		return nil, fmt.Errorf("key not found in JWKS: %s", kid)
	}

	// Convert to RSA public key
	publicKey, err := v.jwkToRSAPublicKey(jwk)
	if err != nil {
		return nil, fmt.Errorf("failed to convert JWK to RSA: %w", err)
	}

	// Cache the key
	v.keyCache.Set(cacheKey, publicKey, cache.DefaultExpiration)
	v.debugLog("RSA key cached for kid: %s", kid)

	return publicKey, nil
}

// jwkToRSAPublicKey converts JWK to RSA public key
func (v *OAuthValidator) jwkToRSAPublicKey(jwk *JWK) (*rsa.PublicKey, error) {
	// Decode base64url encoded modulus and exponent
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert to big integers
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	// Create RSA public key
	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
