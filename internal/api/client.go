package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/auth"
	"github.com/ideamans/google-play-developer-publishing-cli/internal/config"
)

// BaseURL is the API origin. It is a variable so tests can point the client at
// a local mock server.
var BaseURL = "https://androidpublisher.googleapis.com"

// ReportingBaseURL is the Play Developer Reporting API origin, used only by
// commands that enumerate apps (Android Publisher has no such endpoint).
var ReportingBaseURL = "https://playdeveloperreporting.googleapis.com"

// Prefix is the path prefix every Android Publisher v3 endpoint shares.
const Prefix = "/androidpublisher/v3"

type Client struct {
	creds *config.Credentials
	http  *http.Client

	// DryRun, when true, makes mutating helpers (Post/Patch/Put/Delete/Upload)
	// print the intended request to stderr and skip it. Reads still execute.
	DryRun bool

	mu     sync.Mutex
	tokens map[string]*auth.Token // scope -> cached token for this process
}

func New(creds *config.Credentials) *Client {
	return &Client{
		creds:  creds,
		http:   &http.Client{Timeout: 30 * time.Minute},
		tokens: map[string]*auth.Token{},
	}
}

// Credentials returns the credentials the client authenticates with.
func (c *Client) Credentials() *config.Credentials { return c.creds }

// Token returns a cached-or-fresh access token for the given scope. An empty
// scope means the Android Publisher scope.
func (c *Client) Token(ctx context.Context, scope string) (string, error) {
	if scope == "" {
		scope = auth.ScopeAndroidPublisher
	}
	c.mu.Lock()
	cached := c.tokens[scope]
	c.mu.Unlock()
	if cached.Valid() {
		return cached.AccessToken, nil
	}
	token, err := auth.Exchange(ctx, c.http, c.creds, scope)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.tokens[scope] = token
	c.mu.Unlock()
	return token.AccessToken, nil
}

// URL expands an API path into an absolute URL. pathOrURL may be an absolute
// URL (returned as-is), a full path starting with the v3 prefix, or a short
// path like "/applications/{package}/edits".
func URL(pathOrURL string) string {
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		return pathOrURL
	}
	p := pathOrURL
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if !strings.HasPrefix(p, Prefix+"/") && p != Prefix {
		p = Prefix + p
	}
	return BaseURL + p
}

// Do sends an authenticated request against the Android Publisher API.
func (c *Client) Do(ctx context.Context, method, pathOrURL string, body io.Reader) ([]byte, error) {
	return c.DoScoped(ctx, method, URL(pathOrURL), auth.ScopeAndroidPublisher, body)
}

// DoScoped sends an authenticated request to an absolute URL with an explicit
// OAuth scope (used for the Play Developer Reporting API).
func (c *Client) DoScoped(ctx context.Context, method, absURL, scope string, body io.Reader) ([]byte, error) {
	token, err := c.Token(ctx, scope)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, absURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return data, apiError(resp.StatusCode, data)
	}
	return data, nil
}

// GetJSON performs a GET and unmarshals the response into v.
func (c *Client) GetJSON(ctx context.Context, path string, v any) error {
	data, err := c.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	if v == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

// GetMap performs a GET and returns the response as a generic document.
func (c *Client) GetMap(ctx context.Context, path string) (Doc, error) {
	var doc Doc
	if err := c.GetJSON(ctx, path, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// GetOptional is like GetMap but returns (nil, nil) on 404.
func (c *Client) GetOptional(ctx context.Context, path string) (Doc, error) {
	doc, err := c.GetMap(ctx, path)
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return doc, nil
}

// Post sends a POST with a JSON body. payload may be nil for endpoints whose
// request body is empty (":commit", ":validate", ":consume", ...).
func (c *Client) Post(ctx context.Context, path string, payload any) (Doc, error) {
	return c.mutate(ctx, http.MethodPost, path, payload)
}

// PostReadOnly sends a POST that changes no state — validating an edit,
// converting prices, reading a batch — and therefore runs even under --dry-run,
// where suppressing it would leave the caller with nothing to show.
func (c *Client) PostReadOnly(ctx context.Context, path string, payload any) (Doc, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}
	data, err := c.Do(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return Doc{}, nil
	}
	var doc Doc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// Patch sends a PATCH with a JSON body.
func (c *Client) Patch(ctx context.Context, path string, payload any) (Doc, error) {
	return c.mutate(ctx, http.MethodPatch, path, payload)
}

// Put sends a PUT with a JSON body.
func (c *Client) Put(ctx context.Context, path string, payload any) (Doc, error) {
	return c.mutate(ctx, http.MethodPut, path, payload)
}

// Delete removes a resource.
func (c *Client) Delete(ctx context.Context, path string) error {
	if c.DryRun {
		fmt.Fprintf(os.Stderr, "DRY-RUN DELETE %s\n", URL(path))
		return nil
	}
	_, err := c.Do(ctx, http.MethodDelete, path, nil)
	return err
}

func (c *Client) mutate(ctx context.Context, method, path string, payload any) (Doc, error) {
	var body io.Reader
	var encoded []byte
	if payload != nil {
		var err error
		encoded, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}
	if c.DryRun {
		fmt.Fprintf(os.Stderr, "DRY-RUN %s %s\n", method, URL(path))
		if len(encoded) > 0 {
			var pretty bytes.Buffer
			if json.Indent(&pretty, encoded, "", "  ") == nil {
				fmt.Fprintln(os.Stderr, pretty.String())
			}
		}
		return Doc{}, nil
	}
	data, err := c.Do(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return Doc{}, nil
	}
	var doc Doc
	if err := json.Unmarshal(data, &doc); err != nil {
		// Some endpoints (orders:refund, purchases:consume) return an empty body
		// or a bare value; treat anything unparseable as "no content".
		return Doc{}, nil
	}
	return doc, nil
}

// Download performs a GET and returns the raw response body, for endpoints that
// return binary content (generated APK / system APK downloads, image previews).
// pathOrURL may be an API path or an absolute URL; the bearer token is attached
// only for Google hosts.
func (c *Client) Download(ctx context.Context, pathOrURL string) ([]byte, error) {
	absURL := URL(pathOrURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, absURL, nil)
	if err != nil {
		return nil, err
	}
	if u, err := url.Parse(absURL); err == nil && strings.HasSuffix(u.Host, "googleapis.com") {
		token, err := c.Token(ctx, auth.ScopeAndroidPublisher)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return data, apiError(resp.StatusCode, data)
	}
	return data, nil
}

// --- Pagination ---------------------------------------------------------------

// ListAll fetches every page of a collection and returns the concatenated
// items under itemsKey. It handles both pagination styles in this API:
// nextPageToken/pageToken (newer endpoints) and tokenPagination.nextPageToken
// with a "token" query parameter (reviews, voidedpurchases, inappproducts).
func (c *Client) ListAll(ctx context.Context, path, itemsKey string) ([]Doc, error) {
	var out []Doc
	next := path
	seen := map[string]bool{}
	for next != "" {
		doc, err := c.GetMap(ctx, next)
		if err != nil {
			return nil, err
		}
		out = append(out, doc.Docs(itemsKey)...)

		token := doc.Str("nextPageToken")
		param := "pageToken"
		if token == "" {
			token = doc.Doc("tokenPagination").Str("nextPageToken")
			param = "token"
		}
		if token == "" || seen[token] {
			break
		}
		seen[token] = true
		next = withQuery(path, param, token)
	}
	return out, nil
}

func withQuery(path, key, value string) string {
	sep := "?"
	if strings.Contains(path, "?") {
		// Drop a previous value for the same key so tokens do not accumulate.
		base, query, _ := strings.Cut(path, "?")
		values, err := url.ParseQuery(query)
		if err == nil {
			values.Set(key, value)
			return base + "?" + values.Encode()
		}
		sep = "&"
	}
	return path + sep + key + "=" + url.QueryEscape(value)
}

// --- Errors -------------------------------------------------------------------

// Error is an API error that carries the HTTP status code.
type Error struct {
	Status  int
	Reason  string
	Message string
}

func (e *Error) Error() string { return e.Message }

// IsNotFound reports whether err is an API error with HTTP 404.
func IsNotFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Status == http.StatusNotFound
}

// IsConflict reports whether err is an API error with HTTP 409.
func IsConflict(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Status == http.StatusConflict
}

func apiError(status int, body []byte) error {
	var payload struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
			Errors  []struct {
				Reason  string `json:"reason"`
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"error"`
	}
	msg := strings.TrimSpace(string(body))
	reason := ""
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error.Message != "" {
		msg = payload.Error.Message
		if payload.Error.Status != "" {
			msg = payload.Error.Status + ": " + msg
		}
		if len(payload.Error.Errors) > 0 {
			reason = payload.Error.Errors[0].Reason
			msg += " (" + reason + ")"
		}
	}
	if len(msg) > 800 {
		msg = msg[:800] + "..."
	}
	full := fmt.Sprintf("HTTP %d: %s", status, msg)
	if hint := hintFor(status, reason, msg); hint != "" {
		full += "\nHint: " + hint
	}
	return &Error{Status: status, Reason: reason, Message: full}
}

func hintFor(status int, reason, msg string) string {
	switch {
	case status == http.StatusUnauthorized:
		return "check the service account key; the token was rejected."
	case status == http.StatusForbidden && strings.Contains(msg, "not been used"):
		return "enable the Google Play Android Developer API for the key's Google Cloud project, then retry."
	case status == http.StatusForbidden:
		return "the service account must be invited in Play Console > Users and permissions and granted access to this app. " +
			"Permission changes can take a few minutes to propagate."
	case status == http.StatusNotFound && strings.Contains(msg, "package"):
		return "the package name must match an app that has at least one APK/AAB published (even to internal testing) in this account."
	case status == http.StatusConflict:
		return "another edit may already be open, or the resource already exists. " +
			`List and delete stale edits, or pass --edit <id> to reuse one.`
	case reason == "editAlreadyCommitted" || strings.Contains(msg, "edit is not valid"):
		return "the edit expired or was already committed; edits are short-lived, so create a new one."
	}
	return ""
}
