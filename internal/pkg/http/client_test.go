//nolint:revive,nolintlint // var-naming: package name matches the package being tested
package http

import (
	gohttp "net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	httpClient := &gohttp.Client{Timeout: 10 * time.Second}
	timeout := 5 * time.Second

	c := NewClient(httpClient, timeout)

	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestClient_Fetch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseBody   string
		responseStatus int
		headers        map[string]string
		wantBody       string
		wantErr        bool
	}{
		{
			name:           "successful request with UTF-8",
			responseBody:   "<html><body>Hello World</body></html>",
			responseStatus: gohttp.StatusOK,
			headers:        nil,
			wantBody:       "<html><body>Hello World</body></html>",
			wantErr:        false,
		},
		{
			name:           "successful request with custom headers",
			responseBody:   `{"rate": 3.45}`,
			responseStatus: gohttp.StatusOK,
			headers:        map[string]string{"X-API-Key": "test-key"},
			wantBody:       `{"rate": 3.45}`,
			wantErr:        false,
		},
		{
			name:           "handles non-200 status (still returns body)",
			responseBody:   "Not Found",
			responseStatus: gohttp.StatusNotFound,
			headers:        nil,
			wantBody:       "Not Found",
			wantErr:        false,
		},
		{
			name:           "Swedish characters with UTF-8",
			responseBody:   "Snitträntor för bolån",
			responseStatus: gohttp.StatusOK,
			headers:        nil,
			wantBody:       "Snitträntor för bolån",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, _ *gohttp.Request) {
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			httpClient := &gohttp.Client{Timeout: 5 * time.Second}
			c := NewClient(httpClient, 5*time.Second)

			got, err := c.Fetch(server.URL, tt.headers)

			if (err != nil) != tt.wantErr {
				t.Errorf("Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantBody {
				t.Errorf("Fetch() = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestClient_Fetch_CustomHeadersSent(t *testing.T) {
	t.Parallel()

	var receivedHeaders gohttp.Header
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(gohttp.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	httpClient := &gohttp.Client{Timeout: 5 * time.Second}
	c := NewClient(httpClient, 5*time.Second)

	customHeaders := map[string]string{
		"X-API-Key": "secret-key",
		"X-Custom":  "custom-value",
	}

	_, err := c.Fetch(server.URL, customHeaders)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Verify custom headers were sent.
	if got := receivedHeaders.Get("X-Api-Key"); got != "secret-key" {
		t.Errorf("X-Api-Key header = %q, want %q", got, "secret-key")
	}
	if got := receivedHeaders.Get("X-Custom"); got != "custom-value" {
		t.Errorf("X-Custom header = %q, want %q", got, "custom-value")
	}

	// Verify default headers are also present.
	if got := receivedHeaders.Get("Accept"); got == "" {
		t.Error("Accept header should be set by default")
	}
	if got := receivedHeaders.Get("User-Agent"); got == "" {
		t.Error("User-Agent header should be set by default")
	}
}

func TestClient_Fetch_InvalidURL(t *testing.T) {
	t.Parallel()

	httpClient := &gohttp.Client{Timeout: 5 * time.Second}
	c := NewClient(httpClient, 5*time.Second)

	_, err := c.Fetch("://invalid-url", nil)
	if err == nil {
		t.Error("Fetch() expected error for invalid URL")
	}
}

func TestClient_Fetch_ConnectionError(t *testing.T) {
	t.Parallel()

	httpClient := &gohttp.Client{Timeout: 1 * time.Second}
	c := NewClient(httpClient, 1*time.Second)

	// Use a URL that will fail to connect.
	_, err := c.Fetch("http://localhost:1", nil)
	if err == nil {
		t.Error("Fetch() expected error for connection failure")
	}
}
