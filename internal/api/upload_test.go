package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/config"
)

// TestUploadFileResumable drives the full resumable protocol with a chunk size
// small enough to need several PUTs: session start, 308 continuations, final
// response.
func TestUploadFileResumable(t *testing.T) {
	const total = 3*ChunkSize + 1024

	var (
		mu       sync.Mutex
		received []byte
		starts   int
		puts     int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			_, _ = w.Write([]byte(`{"access_token":"test","expires_in":3600}`))

		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/resumable/upload"):
			mu.Lock()
			starts++
			mu.Unlock()
			if got := r.Header.Get("X-Upload-Content-Length"); got != fmt.Sprint(total) {
				t.Errorf("X-Upload-Content-Length = %q, want %d", got, total)
			}
			if r.URL.Query().Get("uploadType") != "resumable" {
				t.Errorf("uploadType = %q, want resumable", r.URL.Query().Get("uploadType"))
			}
			w.Header().Set("Location", "http://"+r.Host+"/session/1")
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodPut && r.URL.Path == "/session/1":
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			puts++
			received = append(received, body...)
			done := len(received) >= total
			end := len(received) - 1
			mu.Unlock()
			if !strings.HasPrefix(r.Header.Get("Content-Range"), "bytes ") {
				t.Errorf("missing Content-Range: %q", r.Header.Get("Content-Range"))
			}
			if !done {
				w.Header().Set("Range", fmt.Sprintf("bytes=0-%d", end))
				w.WriteHeader(308)
				return
			}
			_, _ = w.Write([]byte(`{"versionCode":42}`))

		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	prev := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = prev }()

	path := filepath.Join(t.TempDir(), "app.aab")
	payload := make([]byte, total)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	// Progress output goes to stderr; keep the test output clean.
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devnull.Close()
	realStderr := os.Stderr
	os.Stderr = devnull
	defer func() { os.Stderr = realStderr }()

	doc, err := uploadTestClient(t, srv.URL).UploadFile(context.Background(),
		"/applications/com.x/edits/E/bundles", path, "application/octet-stream")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Int("versionCode") != 42 {
		t.Errorf("versionCode = %d, want 42", doc.Int("versionCode"))
	}
	if starts != 1 {
		t.Errorf("resumable sessions started = %d, want 1", starts)
	}
	if puts != 4 {
		t.Errorf("chunks uploaded = %d, want 4", puts)
	}
	if len(received) != total {
		t.Fatalf("received %d bytes, want %d", len(received), total)
	}
	for i := range payload {
		if received[i] != payload[i] {
			t.Fatalf("payload differs at byte %d", i)
		}
	}
}

func TestUploadURLKeepsExistingQuery(t *testing.T) {
	prev := BaseURL
	BaseURL = "https://example.test"
	defer func() { BaseURL = prev }()

	got := uploadURL("upload", "/applications/com.x/edits/E/bundles?ackBundleInstallationWarning=true", "media")
	if !strings.Contains(got, "ackBundleInstallationWarning=true") || !strings.Contains(got, "uploadType=media") {
		t.Errorf("uploadURL = %q, want both query parameters", got)
	}
	if !strings.HasPrefix(got, "https://example.test/upload"+Prefix+"/applications/") {
		t.Errorf("uploadURL = %q, want the /upload prefix", got)
	}
}

func uploadTestClient(t *testing.T, tokenHost string) *Client {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return New(&config.Credentials{Key: &config.ServiceAccountKey{
		Type:        "service_account",
		PrivateKey:  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		ClientEmail: "sa@proj.iam.gserviceaccount.com",
		TokenURI:    tokenHost + "/token",
	}})
}
