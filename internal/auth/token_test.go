package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/config"
)

func testCredentials(t *testing.T, tokenURI string) *config.Credentials {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return &config.Credentials{
		Key: &config.ServiceAccountKey{
			Type:         "service_account",
			PrivateKeyID: "KEYID",
			PrivateKey:   string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
			ClientEmail:  "sa@proj.iam.gserviceaccount.com",
			TokenURI:     tokenURI,
		},
	}
}

func TestAssertionClaims(t *testing.T) {
	creds := testCredentials(t, config.DefaultTokenURI)
	signed, err := Assertion(creds, ScopeAndroidPublisher, DefaultTTL)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _, err := jwt.NewParser().ParseUnverified(signed, jwt.MapClaims{})
	if err != nil {
		t.Fatal(err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["iss"] != creds.Key.ClientEmail {
		t.Errorf("iss = %v, want the client email", claims["iss"])
	}
	if claims["scope"] != ScopeAndroidPublisher {
		t.Errorf("scope = %v", claims["scope"])
	}
	if claims["aud"] != config.DefaultTokenURI {
		t.Errorf("aud = %v, want the token endpoint", claims["aud"])
	}
	if parsed.Header["kid"] != "KEYID" {
		t.Errorf("kid = %v", parsed.Header["kid"])
	}
	if parsed.Method.Alg() != "RS256" {
		t.Errorf("alg = %v, want RS256", parsed.Method.Alg())
	}
}

func TestAssertionRejectsTooLongTTL(t *testing.T) {
	creds := testCredentials(t, config.DefaultTokenURI)
	if _, err := Assertion(creds, ScopeAndroidPublisher, 2*time.Hour); err == nil {
		t.Fatal("expected an error for a TTL beyond one hour")
	}
}

func TestExchange(t *testing.T) {
	var gotGrant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotGrant = r.Form.Get("grant_type")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ya29.test","token_type":"Bearer","expires_in":3599}`))
	}))
	defer srv.Close()

	token, err := Exchange(context.Background(), srv.Client(), testCredentials(t, srv.URL), ScopeAndroidPublisher)
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "ya29.test" {
		t.Errorf("AccessToken = %q", token.AccessToken)
	}
	if !token.Valid() {
		t.Error("a token expiring in an hour should be valid")
	}
	if gotGrant != grantTypeJWTBearer {
		t.Errorf("grant_type = %q, want %q", gotGrant, grantTypeJWTBearer)
	}
}

func TestExchangeSurfacesOAuthErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Invalid JWT Signature."}`))
	}))
	defer srv.Close()

	_, err := Exchange(context.Background(), srv.Client(), testCredentials(t, srv.URL), ScopeAndroidPublisher)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "invalid_grant") || !strings.Contains(err.Error(), "clock") {
		t.Errorf("error = %v, want the invalid_grant hint", err)
	}
}

func TestTokenValidity(t *testing.T) {
	var nilToken *Token
	if nilToken.Valid() {
		t.Error("a nil token must not be valid")
	}
	expiring := &Token{AccessToken: "x", Expiry: time.Now().Add(30 * time.Second)}
	if expiring.Valid() {
		t.Error("a token expiring within the minute must be refreshed")
	}
}
