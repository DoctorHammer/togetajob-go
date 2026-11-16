// Package togetajob is the official Go client for togetajob.com, a free,
// independent guide to New York City government jobs and civil service exams.
//
// The public API needs no account and no key. Public reads are rate-limited
// to 100 requests a minute per caller; a free API key (created with
// Client.CreateKey, no sign-in) raises that to 600 a minute. Pass the key to
// NewClient with WithAPIKey, or leave it off and stay anonymous.
//
// The data feeds are open JSON: the exam calendar, the job guides, and the
// open-postings index. The API is documented at https://togetajob.com/developers
// and the machine-readable contract at https://togetajob.com/data/openapi.json.
package togetajob

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the public togetajob site.
const DefaultBaseURL = "https://togetajob.com"

// DefaultTimeout bounds every request the client makes.
const DefaultTimeout = 30 * time.Second

// Client talks to the togetajob public API and data feeds.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL points the client at a different base URL (tests, mirrors).
func WithBaseURL(u string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimSuffix(u, "/")
	}
}

// WithAPIKey sends the key as X-Api-Key on every request. A key is optional;
// without one the client reads as an anonymous caller (100 requests a minute).
// With a key the limit is 600 a minute. Create one with Client.CreateKey.
func WithAPIKey(key string) Option {
	return func(c *Client) {
		c.apiKey = key
	}
}

// WithHTTPClient replaces the default HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// NewClient builds a client for the public togetajob API.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: DefaultTimeout},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// APIError is a non-2xx response from the API. The body is JSON
// ({"error": "...", "code": "..."}) and both fields are carried here.
type APIError struct {
	Status int
	Code   string
	// Message is the error string from the response body.
	Message string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("togetajob: %s (%d, %s)", e.Message, e.Status, e.Code)
	}
	return fmt.Sprintf("togetajob: %s (%d)", e.Message, e.Status)
}

func (c *Client) get(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.decode(resp, v)
}

func (c *Client) post(ctx context.Context, path string, body io.Reader, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.decode(resp, v)
}

func (c *Client) delete(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.decode(resp, v)
}

func (c *Client) decode(resp *http.Response, v any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e struct {
			Error      string `json:"error"`
			Code       string `json:"code"`
			Resolution string `json:"resolution"`
		}
		_ = json.Unmarshal(body, &e)
		msg := e.Error
		if msg == "" {
			msg = strings.TrimSpace(string(body))
		}
		if msg == "" {
			msg = resp.Status
		}
		if e.Resolution != "" {
			msg = msg + ": " + e.Resolution
		}
		return &APIError{Status: resp.StatusCode, Code: e.Code, Message: msg}
	}
	if v == nil {
		return nil
	}
	return json.Unmarshal(body, v)
}

// Health reports whether the API is up and the date of the last data refresh.
func (c *Client) Health(ctx context.Context) (*Health, error) {
	var h Health
	if err := c.get(ctx, "/api/health", &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// Exams returns the civil service exam calendar: every exam with its filing
// window, fee, and waivers.
func (c *Client) Exams(ctx context.Context) (*ExamsFeed, error) {
	var f ExamsFeed
	if err := c.get(ctx, "/data/exams.json", &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Guides returns the step-by-step job guides: what each entry-level city
// title pays, whether it needs an exam, and the exact steps to get hired.
func (c *Client) Guides(ctx context.Context) (*GuidesFeed, error) {
	var f GuidesFeed
	if err := c.get(ctx, "/data/guides.json", &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Jobs returns the open-postings index: every open city job posting, sorted
// easiest to get first.
func (c *Client) Jobs(ctx context.Context) (*JobsIndex, error) {
	var f JobsIndex
	if err := c.get(ctx, "/jobs/index.json", &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Posting looks up one open posting by its city job ID (the number in the
// official posting URL, e.g. "808867").
func (c *Client) Posting(ctx context.Context, jobID string) (*Posting, error) {
	var p Posting
	q := url.Values{"id": []string{jobID}}
	if err := c.get(ctx, "/api/posting?"+q.Encode(), &p); err != nil {
		return nil, err
	}
	if !p.Found {
		return nil, &APIError{Status: http.StatusNotFound, Code: "not_found", Message: "no open posting with id " + jobID}
	}
	return &p, nil
}

// CreateKey mints a free, anonymous API key (no sign-in). The key is returned
// once, in full; only its hash is stored server-side, so keep it somewhere
// safe. It raises the rate limit from 100 to 600 requests a minute and
// expires a year after creation. At most 5 keys an hour can be created from
// one caller.
func (c *Client) CreateKey(ctx context.Context) (*APIKey, error) {
	var k APIKey
	if err := c.post(ctx, "/api/keys", nil, &k); err != nil {
		return nil, err
	}
	return &k, nil
}

// RevokeKey deletes the key (the one the client was built with, or the key
// passed here). After revocation the caller falls back to the anonymous
// 100-requests-a-minute limit. Revoking an unknown key returns a 404.
func (c *Client) RevokeKey(ctx context.Context, key string) error {
	// Revoke with the key itself, not the client's stored key, so a caller
	// can revoke a key it no longer wants to keep configured.
	old := c.apiKey
	c.apiKey = key
	defer func() { c.apiKey = old }()
	var out struct {
		Revoked bool `json:"revoked"`
	}
	return c.delete(ctx, "/api/keys", &out)
}
