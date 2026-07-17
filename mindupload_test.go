package mindupload

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient points a client at a stub server and disables retry backoff waits
// by keeping maxRetries small; the stub decides status per call.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func TestRequestShape(t *testing.T) {
	var gotPath, gotKey, gotUA string
	var body map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-Partner-Key")
		gotUA = r.Header.Get("User-Agent")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"jwt":"tok"}`))
	})
	defer srv.Close()

	mu := New("pk_test", WithBaseURL(srv.URL), WithPreferredLanguage("en"))
	resp, err := mu.RequestUploadURL(context.Background(), RequestUploadURLParams{
		Username:      "ada",
		FileSizeBytes: Ptr[int64](10),
		HasThumbnail:  Ptr(false),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/v1/request_upload_url" {
		t.Errorf("path = %q", gotPath)
	}
	if gotKey != "pk_test" {
		t.Errorf("partner key header = %q", gotKey)
	}
	if gotUA == "" {
		t.Errorf("missing user agent")
	}
	if body["username"] != "ada" || body["preferred_language"] != "en" {
		t.Errorf("body = %v", body)
	}
	// A meaningful false must survive (pointer-based optional).
	if v, ok := body["has_thumbnail"].(bool); !ok || v != false {
		t.Errorf("has_thumbnail not sent as false: %v", body["has_thumbnail"])
	}
	if resp.String("jwt") != "tok" {
		t.Errorf("jwt = %q", resp.String("jwt"))
	}
}

func TestLogicalFailure(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"error_message":"no such user"}`))
	})
	defer srv.Close()
	mu := New("pk", WithBaseURL(srv.URL))
	_, err := mu.GetUser(context.Background(), GetUserParams{Username: "nobody"})
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *Error, got %T", err)
	}
	if apiErr.Status != 0 || apiErr.Message != "no such user" || apiErr.Operation != "get_user" {
		t.Errorf("error = %+v", apiErr)
	}
}

func TestAuthenticationError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"success":false,"error_message":"bad key"}`))
	})
	defer srv.Close()
	mu := New("pk", WithBaseURL(srv.URL), WithMaxRetries(0))
	_, err := mu.Login(context.Background(), LoginParams{Username: "a"})
	if !errors.Is(err, ErrAuthentication) {
		t.Fatalf("want ErrAuthentication, got %v", err)
	}
	var apiErr *Error
	if errors.As(err, &apiErr); apiErr.Status != 401 {
		t.Errorf("status = %d", apiErr.Status)
	}
}

func TestRateLimitRetries(t *testing.T) {
	var calls int
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"success":false,"error_message":"slow"}`))
	})
	defer srv.Close()
	mu := New("pk", WithBaseURL(srv.URL), WithMaxRetries(1))
	_, err := mu.Rag(context.Background(), RagParams{Username: "a"})
	if !errors.Is(err, ErrRateLimit) {
		t.Fatalf("want ErrRateLimit, got %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestServerErrorNotRetried(t *testing.T) {
	var calls int
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"success":false,"error_message":"boom"}`))
	})
	defer srv.Close()
	mu := New("pk", WithBaseURL(srv.URL), WithMaxRetries(2))
	_, err := mu.Rag(context.Background(), RagParams{Username: "a"})
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Status != 503 {
		t.Fatalf("want *Error status 503, got %v", err)
	}
	if calls != 1 {
		t.Errorf("5xx must not be retried (non-idempotent POST); calls = %d", calls)
	}
}

func TestWaitForExternalAuthorizationReturnsExchange(t *testing.T) {
	var gotDeviceCode string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotDeviceCode, _ = body["device_code"].(string)
		_, _ = w.Write([]byte(
			`{"success":true,"status":"exchanged","access_token":"access","clone_ids":["clone"]}`,
		))
	})
	defer srv.Close()
	mu := New("pk", WithBaseURL(srv.URL))
	result, err := mu.WaitForExternalAuthorization(
		context.Background(), "mindupload_external_device_test",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String("access_token") != "access" {
		t.Errorf("access_token = %q", result.String("access_token"))
	}
	if cloneIDs := result.Strings("clone_ids"); len(cloneIDs) != 1 || cloneIDs[0] != "clone" {
		t.Errorf("clone_ids = %v", cloneIDs)
	}
	if gotDeviceCode != "mindupload_external_device_test" {
		t.Errorf("device_code = %q", gotDeviceCode)
	}
}

func TestContextCancelUnwrapsContextError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true}`))
	})
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before the call
	mu := New("pk", WithBaseURL(srv.URL), WithMaxRetries(0))
	_, err := mu.Rag(ctx, RagParams{Username: "a"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled (unwrapped), got %v", err)
	}
}
