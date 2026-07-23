package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/config"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"test","expires_in":3600}`))
			return
		}
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	prev := BaseURL
	BaseURL = srv.URL
	t.Cleanup(func() { BaseURL = prev })

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	creds := &config.Credentials{Key: &config.ServiceAccountKey{
		Type:        "service_account",
		PrivateKey:  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		ClientEmail: "sa@proj.iam.gserviceaccount.com",
		TokenURI:    srv.URL + "/token",
	}}
	return New(creds), srv
}

func TestURLAddsTheVersionPrefix(t *testing.T) {
	prev := BaseURL
	BaseURL = "https://example.test"
	defer func() { BaseURL = prev }()

	cases := map[string]string{
		"/applications/com.x/edits":                     "https://example.test/androidpublisher/v3/applications/com.x/edits",
		"applications/com.x/edits":                      "https://example.test/androidpublisher/v3/applications/com.x/edits",
		"/androidpublisher/v3/applications/com.x/edits": "https://example.test/androidpublisher/v3/applications/com.x/edits",
		"https://other.test/thing":                      "https://other.test/thing",
	}
	for in, want := range cases {
		if got := URL(in); got != want {
			t.Errorf("URL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTokenIsCachedForTheProcess(t *testing.T) {
	var tokenCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		_, _ = w.Write([]byte(`{"access_token":"test","expires_in":3600}`))
	}))
	defer srv.Close()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(key)
	c := New(&config.Credentials{Key: &config.ServiceAccountKey{
		Type:        "service_account",
		PrivateKey:  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		ClientEmail: "sa@proj.iam.gserviceaccount.com",
		TokenURI:    srv.URL,
	}})
	for range 3 {
		if _, err := c.Token(context.Background(), ""); err != nil {
			t.Fatal(err)
		}
	}
	if tokenCalls != 1 {
		t.Errorf("token endpoint hit %d times, want 1", tokenCalls)
	}
}

func TestListAllFollowsPageToken(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("pageToken") {
		case "":
			_, _ = w.Write([]byte(`{"subscriptions":[{"productId":"a"}],"nextPageToken":"p2"}`))
		case "p2":
			_, _ = w.Write([]byte(`{"subscriptions":[{"productId":"b"}]}`))
		default:
			t.Errorf("unexpected pageToken %q", r.URL.Query().Get("pageToken"))
		}
	})
	items, err := c.ListAll(context.Background(), "/applications/com.x/subscriptions?pageSize=100", "subscriptions")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Str("productId") != "a" || items[1].Str("productId") != "b" {
		t.Fatalf("items = %v", items)
	}
}

func TestListAllFollowsTokenPagination(t *testing.T) {
	var seen []string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		seen = append(seen, token)
		if token == "" {
			_, _ = w.Write([]byte(`{"reviews":[{"reviewId":"r1"}],"tokenPagination":{"nextPageToken":"t2"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"reviews":[{"reviewId":"r2"}]}`))
	})
	items, err := c.ListAll(context.Background(), "/applications/com.x/reviews?maxResults=100", "reviews")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if len(seen) != 2 || seen[1] != "t2" {
		t.Errorf("tokens seen = %v, want the second page to use token=t2", seen)
	}
}

func TestDryRunSkipsMutations(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("dry run sent %s %s", r.Method, r.URL.Path)
	})
	c.DryRun = true
	if _, err := c.Post(context.Background(), "/applications/com.x/edits", map[string]any{"a": 1}); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(context.Background(), "/applications/com.x/edits/1"); err != nil {
		t.Fatal(err)
	}
}

func TestErrorsCarryStatusAndHint(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprint(w, `{"error":{"code":403,"status":"PERMISSION_DENIED","message":"The caller does not have permission","errors":[{"reason":"forbidden"}]}}`)
	})
	_, err := c.GetMap(context.Background(), "/applications/com.x/edits/1")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "PERMISSION_DENIED") {
		t.Errorf("error = %v, want the API status", err)
	}
	if !strings.Contains(err.Error(), "Play Console") {
		t.Errorf("error = %v, want the permissions hint", err)
	}
	var apiErr *Error
	if !isAPIError(err, &apiErr) || apiErr.Status != 403 {
		t.Errorf("error does not carry the status: %v", err)
	}
}

func TestIsNotFound(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":404,"status":"NOT_FOUND","message":"nope"}}`))
	})
	doc, err := c.GetOptional(context.Background(), "/applications/com.x/edits/1/details")
	if err != nil {
		t.Fatalf("GetOptional returned %v, want a nil document", err)
	}
	if doc != nil {
		t.Errorf("doc = %v, want nil on 404", doc)
	}
}

func TestUploadBytesUsesTheUploadEndpoint(t *testing.T) {
	var gotPath, gotQuery, gotType string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery, gotType = r.URL.Path, r.URL.RawQuery, r.Header.Get("Content-Type")
		_, _ = w.Write([]byte(`{"image":{"id":"i1"}}`))
	})
	doc, err := c.UploadBytes(context.Background(),
		"/applications/com.x/edits/E/listings/ja/icon", []byte("png"), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if want := "/upload" + Prefix + "/applications/com.x/edits/E/listings/ja/icon"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if gotQuery != "uploadType=media" {
		t.Errorf("query = %q, want uploadType=media", gotQuery)
	}
	if gotType != "image/png" {
		t.Errorf("Content-Type = %q", gotType)
	}
	if doc.Doc("image").Str("id") != "i1" {
		t.Errorf("response not decoded: %v", doc)
	}
}

func isAPIError(err error, target **Error) bool {
	e, ok := err.(*Error)
	if ok {
		*target = e
	}
	return ok
}
