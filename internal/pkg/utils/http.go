// #nosec G404 // not used in security context, no strong randomness needed
//
//nolint:revive,nolintlint // I like this package name, leave me alone
package utils

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

var (
	// Unused but keep for now. Banks have ancient systems, SEB used this encoding previously, which was painful to fix.
	DecoderWindows1252 Decoder = func(runes []byte) (str string) { //nolint: gochecknoglobals
		for _, r := range runes {
			str += string(r)
		}
		return
	}
	DecoderUtf8 Decoder = func(runes []byte) string { //nolint: gochecknoglobals
		return string(runes)
	}
)

type Decoder func([]byte) string

// BrowserInfo contains User-Agent and Sec-Ch-Ua headers that must have matching versions.
type BrowserInfo struct {
	UserAgent string
	SecChUa   string
}

func FetchRawContentFromURL(url string, decoder Decoder, headers map[string]string) (string, error) {
	client := http.Client{Timeout: 30 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

	for key, value := range headers { // overwrites default headers if same key
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return decoder(body), nil
}

// randomBrowserInfo generates matching User-Agent and Sec-Ch-Ua headers.
// The Chrome/Chromium/Brave versions are kept in sync between both headers.
func randomBrowserInfo() BrowserInfo {
	// Chrome/Chromium version range (recent versions)
	majorVersion := rand.Intn(25) + 120 // Version 120-144

	// Platform for User-Agent
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
				macMajor := rand.Intn(3) + 13 // macOS 13-15
				macMinor := rand.Intn(10)
				macPatch := rand.Intn(10)
				return fmt.Sprintf("Macintosh; Intel Mac OS X %d_%d_%d", macMajor, macMinor, macPatch)
			},
		},
	}

	platformInfo := platforms[rand.Intn(len(platforms))]
	platform := platformInfo.generator()

	// Build User-Agent (Chrome-based)
	minorVersion := rand.Intn(10)
	patchVersion := rand.Intn(1000)
	userAgent := fmt.Sprintf(
		"Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.%d.%d Safari/537.36",
		platform, majorVersion, minorVersion, patchVersion,
	)

	// Build matching Sec-Ch-Ua header
	// Format: "Chromium";v="VERSION", "Brave";v="VERSION", "Not_A Brand";v="99"
	secChUa := fmt.Sprintf(
		`"Chromium";v="%d", "Brave";v="%d", "Not_A Brand";v="99"`,
		majorVersion, majorVersion,
	)

	return BrowserInfo{
		UserAgent: userAgent,
		SecChUa:   secChUa,
	}
}
