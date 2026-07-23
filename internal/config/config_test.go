package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testKeyJSON(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(map[string]string{
		"type":         "service_account",
		"project_id":   "proj",
		"private_key":  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		"client_email": "sa@proj.iam.gserviceaccount.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestParseServiceAccountRejectsOAuthClient(t *testing.T) {
	_, err := ParseServiceAccount([]byte(`{"installed":{"client_id":"x"}}`))
	if err == nil || !strings.Contains(err.Error(), "not a service account key") {
		t.Fatalf("error = %v, want a complaint about the key type", err)
	}
}

func TestParseServiceAccountDefaultsTokenURI(t *testing.T) {
	key, err := ParseServiceAccount(testKeyJSON(t))
	if err != nil {
		t.Fatal(err)
	}
	if key.TokenURI != DefaultTokenURI {
		t.Errorf("TokenURI = %q, want the default %q", key.TokenURI, DefaultTokenURI)
	}
}

func TestResolvePrefersEnvironmentOverProfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("GPLAY_PROFILE", "")
	t.Setenv("GPLAY_SERVICE_ACCOUNT_BASE64", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")

	keyPath := filepath.Join(dir, "sa.json")
	if err := os.WriteFile(keyPath, testKeyJSON(t), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GPLAY_SERVICE_ACCOUNT_JSON", keyPath)
	t.Setenv("GPLAY_PACKAGE", "com.example.env")

	creds, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if creds.Source != "env" {
		t.Errorf("Source = %q, want env", creds.Source)
	}
	if creds.Package != "com.example.env" {
		t.Errorf("Package = %q, want com.example.env", creds.Package)
	}
}

func TestResolveBase64Credentials(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GPLAY_SERVICE_ACCOUNT_JSON", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	t.Setenv("GPLAY_SERVICE_ACCOUNT_BASE64", base64.StdEncoding.EncodeToString(testKeyJSON(t)))

	creds, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if creds.Key.ClientEmail != "sa@proj.iam.gserviceaccount.com" {
		t.Errorf("ClientEmail = %q", creds.Key.ClientEmail)
	}
}

func TestResolveProfileAndDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	for _, name := range []string{"GPLAY_SERVICE_ACCOUNT_JSON", "GPLAY_SERVICE_ACCOUNT_BASE64",
		"GOOGLE_APPLICATION_CREDENTIALS", "GPLAY_PACKAGE", "GPLAY_DEVELOPER_ID", "GPLAY_PROFILE"} {
		t.Setenv(name, "")
	}
	keysDir, err := KeysDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(keysDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keysDir, "client-a.json"), testKeyJSON(t), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		DefaultProfile: "client-a",
		Profiles: map[string]Profile{
			"client-a": {ServiceAccount: "keys/client-a.json", Package: "com.client.a", DeveloperID: "42"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	creds, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if creds.Source != "profile:client-a" {
		t.Errorf("Source = %q", creds.Source)
	}
	if creds.Package != "com.client.a" || creds.DeveloperID != "42" {
		t.Errorf("profile defaults not applied: %+v", creds)
	}

	if _, err := Resolve("nope"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("Resolve(unknown profile) = %v, want a not-found error", err)
	}
}

func TestResolveWithoutAnythingConfigured(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, name := range []string{"GPLAY_SERVICE_ACCOUNT_JSON", "GPLAY_SERVICE_ACCOUNT_BASE64",
		"GOOGLE_APPLICATION_CREDENTIALS", "GPLAY_PROFILE"} {
		t.Setenv(name, "")
	}
	_, err := Resolve("")
	if err == nil || !strings.Contains(err.Error(), "gplay configure") {
		t.Fatalf("error = %v, want a pointer to \"gplay configure\"", err)
	}
}
