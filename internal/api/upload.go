package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/auth"
)

// ChunkSize is the resumable-upload chunk size. Google requires a multiple of
// 256 KiB for every chunk except the last.
const ChunkSize = 32 << 20

// resumableThreshold is the size above which uploads switch from a single
// request to the resumable protocol.
const resumableThreshold = 16 << 20

// uploadURL builds an upload endpoint URL. kind is "upload" or
// "resumable/upload"; pathAndQuery is a short API path, optionally with a query
// string, to which uploadType is added.
func uploadURL(kind, pathAndQuery, uploadType string) string {
	path, query, _ := strings.Cut(pathAndQuery, "?")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		values = url.Values{}
	}
	values.Set("uploadType", uploadType)
	return BaseURL + "/" + kind + Prefix + path + "?" + values.Encode()
}

// UploadFile uploads a file to an Android Publisher media endpoint and returns
// the decoded response. Small files go up in one request; larger ones use the
// resumable protocol with progress on stderr.
func (c *Client) UploadFile(ctx context.Context, path, filePath, contentType string) (Doc, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if c.DryRun {
		fmt.Fprintf(os.Stderr, "DRY-RUN UPLOAD %s <- %s (%s, %s)\n",
			uploadURL("upload", path, "media"), filePath, humanBytes(info.Size()), contentType)
		return Doc{}, nil
	}
	if info.Size() <= resumableThreshold {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return c.UploadBytes(ctx, path, data, contentType)
	}
	return c.uploadResumable(ctx, path, filePath, contentType, info.Size())
}

// UploadBytes performs a single-request media upload.
func (c *Client) UploadBytes(ctx context.Context, path string, data []byte, contentType string) (Doc, error) {
	absURL := uploadURL("upload", path, "media")
	if c.DryRun {
		fmt.Fprintf(os.Stderr, "DRY-RUN UPLOAD %s (%s, %s)\n", absURL, humanBytes(int64(len(data))), contentType)
		return Doc{}, nil
	}
	token, err := c.Token(ctx, auth.ScopeAndroidPublisher)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, absURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(len(data))
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, apiError(resp.StatusCode, body)
	}
	return decodeDoc(body)
}

// UploadMultipart sends a file together with a JSON metadata part, which is
// what endpoints whose request body carries required fields alongside the media
// expect (uploadType=multipart).
func (c *Client) UploadMultipart(ctx context.Context, path, filePath, contentType string, metadata any) (Doc, error) {
	absURL := uploadURL("upload", path, "multipart")
	meta, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	if c.DryRun {
		fmt.Fprintf(os.Stderr, "DRY-RUN UPLOAD %s <- %s (%s)\n%s\n", absURL, filePath, contentType, meta)
		return Doc{}, nil
	}
	media, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	const boundary = "gplay-multipart-boundary"
	var body bytes.Buffer
	fmt.Fprintf(&body, "--%s\r\nContent-Type: application/json; charset=UTF-8\r\n\r\n%s\r\n", boundary, meta)
	fmt.Fprintf(&body, "--%s\r\nContent-Type: %s\r\n\r\n", boundary, contentType)
	body.Write(media)
	fmt.Fprintf(&body, "\r\n--%s--\r\n", boundary)

	token, err := c.Token(ctx, auth.ScopeAndroidPublisher)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, absURL, bytes.NewReader(body.Bytes()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "multipart/related; boundary="+boundary)
	req.ContentLength = int64(body.Len())
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
		return nil, apiError(resp.StatusCode, data)
	}
	return decodeDoc(data)
}

// uploadResumable implements the Google resumable upload protocol: start a
// session, then PUT chunks until the server returns the final response.
func (c *Client) uploadResumable(ctx context.Context, path, filePath, contentType string, size int64) (Doc, error) {
	session, err := c.startResumable(ctx, path, contentType, size)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := make([]byte, ChunkSize)
	var offset int64
	started := time.Now()
	for offset < size {
		n, err := f.ReadAt(buf, offset)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			return nil, fmt.Errorf("unexpected end of %s at offset %d", filePath, offset)
		}
		doc, next, err := c.putChunk(ctx, session, buf[:n], offset, size)
		if err != nil {
			return nil, err
		}
		if doc != nil {
			fmt.Fprintf(os.Stderr, "\rUploaded %s in %s.%s\n",
				humanBytes(size), time.Since(started).Truncate(time.Second), strings.Repeat(" ", 20))
			return doc, nil
		}
		offset = next
		fmt.Fprintf(os.Stderr, "\rUploading %s: %s / %s (%.1f%%)", filePath,
			humanBytes(offset), humanBytes(size), float64(offset)/float64(size)*100)
	}
	fmt.Fprintln(os.Stderr)
	return nil, fmt.Errorf("upload of %s finished without a final response from the server", filePath)
}

func (c *Client) startResumable(ctx context.Context, path, contentType string, size int64) (string, error) {
	token, err := c.Token(ctx, auth.ScopeAndroidPublisher)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL("resumable/upload", path, "resumable"), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Upload-Content-Type", contentType)
	req.Header.Set("X-Upload-Content-Length", strconv.FormatInt(size, 10))
	req.ContentLength = 0
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", apiError(resp.StatusCode, body)
	}
	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("resumable upload session did not return a Location header (HTTP %d)", resp.StatusCode)
	}
	return location, nil
}

// putChunk uploads one chunk. It returns the final decoded response once the
// server accepts the whole file, or the next offset to continue from.
func (c *Client) putChunk(ctx context.Context, session string, chunk []byte, offset, size int64) (Doc, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, session, bytes.NewReader(chunk))
	if err != nil {
		return nil, 0, err
	}
	end := offset + int64(len(chunk)) - 1
	req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, end, size))
	req.ContentLength = int64(len(chunk))
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode == 308: // Resume Incomplete
		next := end + 1
		if rangeHeader := resp.Header.Get("Range"); rangeHeader != "" {
			// Range: bytes=0-524287
			if _, last, ok := strings.Cut(rangeHeader, "-"); ok {
				if n, err := strconv.ParseInt(last, 10, 64); err == nil {
					next = n + 1
				}
			}
		}
		return nil, next, nil
	case resp.StatusCode < 300:
		doc, err := decodeDoc(body)
		if err != nil {
			return nil, 0, err
		}
		if doc == nil {
			doc = Doc{}
		}
		return doc, end + 1, nil
	default:
		return nil, 0, apiError(resp.StatusCode, body)
	}
}

func decodeDoc(body []byte) (Doc, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return Doc{}, nil
	}
	var doc Doc
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse upload response: %w (body: %.200s)", err, body)
	}
	return doc, nil
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
