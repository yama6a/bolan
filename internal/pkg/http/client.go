// Package http provides HTTP fetching capabilities for crawlers.
//
//go:generate go run -mod=mod github.com/matryer/moq -out httpmock/client_mock.go -pkg httpmock . Client
package http

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	gohttp "net/http"
	"time"
)

// Compile-time interface compliance check.
var _ Client = &client{}

// Client defines the interface for HTTP content fetching.
type Client interface {
	// Fetch retrieves content from a URL with optional custom headers.
	// Response body is decoded as UTF-8.
	Fetch(url string, headers map[string]string) (string, error)
}

// client implements the Client interface using standard net/http.
type client struct {
	httpClient *gohttp.Client
	timeout    time.Duration
}

// NewClient creates a new Client wrapping the provided http.Client.
func NewClient(httpClient *gohttp.Client, timeout time.Duration) Client {
	return &client{
		httpClient: httpClient,
		timeout:    timeout,
	}
}

// Fetch retrieves content from a URL with optional custom headers.
func (c *client) Fetch(url string, headers map[string]string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := gohttp.NewRequestWithContext(ctx, gohttp.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	browserInfo := randomBrowserInfo()
	defaultHeaders := map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9,application/json",
		"Accept-Language": "en-US,en;q=0.9,de-DE;q=0.8,de;q=0.7",
		"Connection":      "keep-alive",
		"User-Agent":      browserInfo.UserAgent,
		"Sec-Ch-Ua":       browserInfo.SecChUa,
		"Cache-Control":   "no-cache",
	}
	for key, value := range defaultHeaders {
		req.Header.Set(key, value)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// browserInfo contains User-Agent and Sec-Ch-Ua headers that must have matching versions.
type browserInfo struct {
	UserAgent string
	SecChUa   string
}

// randomBrowserInfo generates matching User-Agent and Sec-Ch-Ua headers.
// #nosec G404 // not used in security context, no strong randomness needed
func randomBrowserInfo() browserInfo {
	majorVersion := rand.Intn(25) + 120 // Version 120-144

	platforms := []struct {
		uaPlatform string
		generator  func() string
	}{
		{
			uaPlatform: "Windows NT 10.0; Win64; x64",
			generator:  func() string { return "Windows NT 10.0; Win64; x64" },
		},
		{
			uaPlatform: "Macintosh",
			generator: func() string {
				macMajor := rand.Intn(3) + 13
				macMinor := rand.Intn(10)
				macPatch := rand.Intn(10)
				return fmt.Sprintf("Macintosh; Intel Mac OS X %d_%d_%d", macMajor, macMinor, macPatch)
			},
		},
	}

	platformInfo := platforms[rand.Intn(len(platforms))]
	platform := platformInfo.generator()

	minorVersion := rand.Intn(10)
	patchVersion := rand.Intn(1000)
	userAgent := fmt.Sprintf(
		"Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.%d.%d Safari/537.36",
		platform, majorVersion, minorVersion, patchVersion,
	)

	secChUa := fmt.Sprintf(
		`"Chromium";v="%d", "Brave";v="%d", "Not_A Brand";v="99"`,
		majorVersion, majorVersion,
	)

	return browserInfo{
		UserAgent: userAgent,
		SecChUa:   secChUa,
	}
}
