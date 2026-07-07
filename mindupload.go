package mindupload

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is the Mind Upload partner API endpoint.
const DefaultBaseURL = "https://partner.mindupload.app"

const authHeader = "X-Partner-Key"

// Only server backpressure is retried. Operations are non-idempotent POSTs (Rag
// spends credits, Create* mutate), so 5xx / network / timeout failures are
// surfaced immediately rather than risking a duplicate side effect.
var retryStatuses = map[int]bool{429: true, 503: true}

// Client is a Mind Upload partner API client. Create one with New; it is safe
// for concurrent use.
type Client struct {
	partnerKey        string
	baseURL           string
	preferredLanguage string
	httpClient        *http.Client
	maxRetries        int
	timeout           time.Duration
	userAgent         string
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API endpoint.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// WithPreferredLanguage sets a default locale sent with every call; a per-call
// PreferredLanguage overrides it.
func WithPreferredLanguage(lang string) Option {
	return func(c *Client) { c.preferredLanguage = lang }
}

// WithHTTPClient supplies a custom *http.Client (for proxies, transports, etc.).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

// WithTimeout sets a per-call timeout, applied via context so it composes with
// any custom HTTP client (and never mutates a caller-owned one). Zero disables it.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithMaxRetries sets how many times a 429/5xx/network failure is retried.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		if n >= 0 {
			c.maxRetries = n
		}
	}
}

// WithUserAgent overrides the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		if ua != "" {
			c.userAgent = ua
		}
	}
}

// New creates a client. partnerKey is a server-side secret — never embed it in
// client-side code.
func New(partnerKey string, opts ...Option) *Client {
	c := &Client{
		partnerKey: partnerKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{},
		maxRetries: 2,
		timeout:    30 * time.Second,
		userAgent:  "mindupload-go/" + Version,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Response is a decoded API response. Success is always set; every returned
// field is available via Get/String or Decode, keyed by its documented name.
type Response struct {
	Success      bool
	ErrorMessage string
	Data         map[string]any
}

// Get returns a raw field from the response (nil if absent).
func (r *Response) Get(key string) any { return r.Data[key] }

// String returns a string field (empty if absent or not a string).
func (r *Response) String(key string) string {
	if s, ok := r.Data[key].(string); ok {
		return s
	}
	return ""
}

// Bool returns a boolean field (false if absent or not a bool).
func (r *Response) Bool(key string) bool {
	if b, ok := r.Data[key].(bool); ok {
		return b
	}
	return false
}

// Decode unmarshals the full response into v (a struct or map).
func (r *Response) Decode(v any) error {
	b, err := json.Marshal(r.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// Ptr returns a pointer to v — for optional scalar params, e.g. Ptr(true).
func Ptr[T any](v T) *T { return &v }

func structToMap(v any) map[string]any {
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil || m == nil {
		return map[string]any{}
	}
	return m
}

func (c *Client) do(ctx context.Context, operation string, params map[string]any) (*Response, error) {
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	if _, ok := params["preferred_language"]; !ok && c.preferredLanguage != "" {
		params["preferred_language"] = c.preferredLanguage
	}
	payload, err := json.Marshal(params)
	if err != nil {
		return nil, &Error{Operation: operation, Message: "failed to encode request: " + err.Error()}
	}
	url := c.baseURL + "/v1/" + operation

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return nil, &Error{Operation: operation, Message: err.Error(), err: ErrConnection}
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set(authHeader, c.partnerKey)
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				// Wrap the real context error so errors.Is(err, context.DeadlineExceeded) works.
				return nil, &Error{Operation: operation, Message: ctx.Err().Error(), err: ctx.Err()}
			}
			// Not retried: the request may already have reached the backend.
			return nil, &Error{Operation: operation, Message: err.Error(), err: ErrConnection}
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		data := map[string]any{}
		if len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &data)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if retryStatuses[resp.StatusCode] && attempt < c.maxRetries {
				if werr := sleepCtx(ctx, backoff(attempt+1, resp.Header.Get("Retry-After"))); werr != nil {
					return nil, werr
				}
				continue
			}
			return nil, errorFor(resp.StatusCode, data, operation, resp.Header.Get("Retry-After"))
		}

		success, _ := data["success"].(bool)
		msg, _ := data["error_message"].(string)
		if !success {
			if msg == "" {
				msg = operation + " failed"
			}
			return nil, &Error{Operation: operation, Message: msg, Response: data}
		}
		return &Response{Success: true, ErrorMessage: msg, Data: data}, nil
	}
}

func errorFor(status int, data map[string]any, operation, retryAfter string) *Error {
	msg, _ := data["error_message"].(string)
	if msg == "" {
		msg = "HTTP " + strconv.Itoa(status)
	}
	e := &Error{Operation: operation, Status: status, Message: msg, Response: data}
	switch status {
	case 401:
		e.err = ErrAuthentication
	case 429:
		e.err = ErrRateLimit
		if ra, perr := strconv.ParseFloat(retryAfter, 64); perr == nil {
			e.RetryAfter = ra
		}
	}
	return e
}

func backoff(attempt int, retryAfter string) time.Duration {
	if retryAfter != "" {
		if s, err := strconv.ParseFloat(retryAfter, 64); err == nil {
			d := time.Duration(s * float64(time.Second))
			if d > 60*time.Second {
				d = 60 * time.Second
			}
			return d
		}
	}
	ms := 500 * (1 << uint(attempt-1))
	if ms > 8000 {
		ms = 8000
	}
	return time.Duration(ms) * time.Millisecond
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return &Error{Message: ctx.Err().Error(), err: ctx.Err()}
	case <-t.C:
		return nil
	}
}
