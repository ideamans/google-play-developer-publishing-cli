package cmd

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

// testServer wires a mock Android Publisher API: it records every request,
// serves the OAuth token endpoint, and returns canned JSON per "METHOD path".
// Handlers registered with handle() win over responses.
type testServer struct {
	t         *testing.T
	mu        sync.Mutex
	requests  []recordedRequest
	responses map[string]string
	handlers  map[string]http.HandlerFunc
	srv       *httptest.Server
}

type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Body   []byte
}

// newTestServer starts the mock API, points api.BaseURL at it, and installs a
// throwaway service account key whose token_uri is the mock token endpoint.
func newTestServer(t *testing.T) *testServer {
	t.Helper()
	ts := &testServer{
		t:         t,
		responses: map[string]string{},
		handlers:  map[string]http.HandlerFunc{},
	}
	ts.srv = httptest.NewServer(http.HandlerFunc(ts.serve))
	t.Cleanup(ts.srv.Close)

	prevBase := api.BaseURL
	api.BaseURL = ts.srv.URL
	t.Cleanup(func() { api.BaseURL = prevBase })

	setTestCredentials(t, ts.srv.URL+"/token")
	return ts
}

func (ts *testServer) url() string { return ts.srv.URL }

// respond registers a canned response body for "METHOD /path" (query ignored).
func (ts *testServer) respond(methodAndPath, body string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.responses[methodAndPath] = body
}

// handle registers a custom handler for "METHOD /path".
func (ts *testServer) handle(methodAndPath string, h http.HandlerFunc) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.handlers[methodAndPath] = h
}

func (ts *testServer) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))

	if r.URL.Path == "/token" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
		return
	}
	// The v3 prefix is noise in test expectations; record the short path.
	path := strings.TrimPrefix(r.URL.Path, api.Prefix)
	path = strings.TrimPrefix(path, "/upload"+api.Prefix)

	key := r.Method + " " + path
	ts.mu.Lock()
	ts.requests = append(ts.requests, recordedRequest{
		Method: r.Method, Path: path, Query: r.URL.RawQuery, Body: body,
	})
	h := ts.handlers[key]
	resp, ok := ts.responses[key]
	ts.mu.Unlock()

	if h != nil {
		h(w, r)
		return
	}
	if !ok {
		ts.t.Errorf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":404,"status":"NOT_FOUND","message":"not stubbed"}}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(resp))
}

// calls returns the recorded requests matching "METHOD /path".
func (ts *testServer) calls(methodAndPath string) []recordedRequest {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	var out []recordedRequest
	for _, req := range ts.requests {
		if req.Method+" "+req.Path == methodAndPath {
			out = append(out, req)
		}
	}
	return out
}

// lastBody unmarshals the last request body for "METHOD /path" into a map.
func (ts *testServer) lastBody(methodAndPath string) map[string]any {
	ts.t.Helper()
	reqs := ts.calls(methodAndPath)
	if len(reqs) == 0 {
		ts.t.Fatalf("no request recorded for %s", methodAndPath)
	}
	var doc map[string]any
	if err := json.Unmarshal(reqs[len(reqs)-1].Body, &doc); err != nil {
		ts.t.Fatalf("unmarshal body of %s: %v", methodAndPath, err)
	}
	return doc
}

// setTestCredentials generates a throwaway RSA key and writes a service account
// JSON file pointing at the mock token endpoint.
func setTestCredentials(t *testing.T, tokenURI string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	sa := map[string]string{
		"type":           "service_account",
		"project_id":     "test-project",
		"private_key_id": "TESTKEYID",
		"private_key":    string(pemData),
		"client_email":   "gplay-test@test-project.iam.gserviceaccount.com",
		"token_uri":      tokenURI,
	}
	encoded, err := json.Marshal(sa)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GPLAY_SERVICE_ACCOUNT_JSON", path)
	t.Setenv("GPLAY_SERVICE_ACCOUNT_BASE64", "")
	t.Setenv("GPLAY_PACKAGE", "com.example.app")
	t.Setenv("GPLAY_DEVELOPER_ID", "1234567890")
}

// runCommand executes the CLI in-process (gplay <args...>) and returns the
// error. Flag state persists on the package-level cobra commands between
// Execute calls (slice flags even append), so everything is reset first.
func runCommand(t *testing.T, args ...string) error {
	t.Helper()
	resetFlags(rootCmd)
	rootCmd.SetArgs(args)
	defer rootCmd.SetArgs(nil)
	return rootCmd.Execute()
}

func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed {
			return
		}
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			_ = sv.Replace(nil)
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	})
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

// stubEdit registers the edits.insert / delete / commit endpoints an
// edit-scoped command needs.
func stubEdit(ts *testServer, editID string) {
	ts.respond("POST /applications/com.example.app/edits",
		fmt.Sprintf(`{"id":%q,"expiryTimeSeconds":"1800000000"}`, editID))
	ts.respond("POST /applications/com.example.app/edits/"+editID+":commit",
		fmt.Sprintf(`{"id":%q}`, editID))
	ts.handle("DELETE /applications/com.example.app/edits/"+editID, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}

// dig walks a decoded JSON document by keys/indexes.
func dig(t *testing.T, doc any, path ...any) any {
	t.Helper()
	cur := doc
	for _, step := range path {
		switch key := step.(type) {
		case string:
			m, ok := cur.(map[string]any)
			if !ok {
				t.Fatalf("dig %v: not an object at %v", path, step)
			}
			cur = m[key]
		case int:
			arr, ok := cur.([]any)
			if !ok || key >= len(arr) {
				t.Fatalf("dig %v: not an array (or too short) at %v", path, step)
			}
			cur = arr[key]
		}
	}
	return cur
}
