package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/config"
)

// ScopeAndroidPublisher is the scope for the Google Play Developer Publishing
// API (Android Publisher v3).
const ScopeAndroidPublisher = "https://www.googleapis.com/auth/androidpublisher"

// ScopePlayDeveloperReporting is the scope for the Play Developer Reporting
// API, which "gplay apps list" uses because Android Publisher has no endpoint
// that enumerates apps.
const ScopePlayDeveloperReporting = "https://www.googleapis.com/auth/playdeveloperreporting"

// MaxTTL is the maximum assertion lifetime Google accepts.
const MaxTTL = time.Hour

// DefaultTTL leaves headroom for clock skew.
const DefaultTTL = 55 * time.Minute

const grantTypeJWTBearer = "urn:ietf:params:oauth:grant-type:jwt-bearer"

// Token is an OAuth 2.0 access token with its expiry.
type Token struct {
	AccessToken string
	TokenType   string
	Expiry      time.Time
}

// Valid reports whether the token is usable for at least another minute.
func (t *Token) Valid() bool {
	return t != nil && t.AccessToken != "" && time.Now().Add(time.Minute).Before(t.Expiry)
}

// ParsePrivateKey parses the PEM-encoded RSA private key found in the
// private_key field of a service account JSON key.
func ParsePrivateKey(pemData string) (*rsa.PrivateKey, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(pemData))
	if err != nil {
		return nil, fmt.Errorf("parse service account private_key: %w", err)
	}
	return key, nil
}

// Assertion builds the signed JWT (RS256) that is exchanged for an access token.
func Assertion(creds *config.Credentials, scope string, ttl time.Duration) (string, error) {
	if creds == nil || creds.Key == nil {
		return "", errors.New("no credentials resolved")
	}
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if ttl > MaxTTL {
		return "", fmt.Errorf("assertion TTL %s exceeds the Google maximum of %s", ttl, MaxTTL)
	}
	key, err := ParsePrivateKey(creds.Key.PrivateKey)
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   creds.Key.ClientEmail,
		"scope": scope,
		"aud":   creds.Key.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if creds.Key.PrivateKeyID != "" {
		token.Header["kid"] = creds.Key.PrivateKeyID
	}
	return token.SignedString(key)
}

// Exchange trades a signed assertion for an access token at the key's token_uri.
func Exchange(ctx context.Context, httpClient *http.Client, creds *config.Credentials, scope string) (*Token, error) {
	assertion, err := Assertion(creds, scope, DefaultTTL)
	if err != nil {
		return nil, err
	}
	form := url.Values{
		"grant_type": {grantTypeJWTBearer},
		"assertion":  {assertion},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, creds.Key.TokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, tokenError(resp.StatusCode, body, creds)
	}
	var payload struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if payload.AccessToken == "" {
		return nil, errors.New("token endpoint returned no access_token")
	}
	if payload.ExpiresIn == 0 {
		payload.ExpiresIn = 3600
	}
	return &Token{
		AccessToken: payload.AccessToken,
		TokenType:   payload.TokenType,
		Expiry:      time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second),
	}, nil
}

func tokenError(status int, body []byte, creds *config.Credentials) error {
	var payload struct {
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	msg := strings.TrimSpace(string(body))
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error != "" {
		msg = payload.Error
		if payload.Description != "" {
			msg += ": " + payload.Description
		}
	}
	out := fmt.Sprintf("token exchange failed (HTTP %d): %s", status, msg)
	if strings.Contains(msg, "invalid_grant") {
		out += "\nHint: invalid_grant usually means the system clock is skewed, or the service account key was deleted."
	}
	if creds != nil && creds.Key != nil {
		out += fmt.Sprintf("\nService account: %s", creds.Key.ClientEmail)
	}
	return errors.New(out)
}
