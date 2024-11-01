package utils

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

var (
	// Unused but keep for now. Banks have ancient systems, SEB used this encoding previously, which was painful to fix.
	DecoderWindows1252 Decoder = func(runes []byte) (str string) {
		for _, r := range runes {
			str += string(r)
		}
		return
	}
	DecoderUtf8 Decoder = func(runes []byte) string {
		return string(runes)
	}
)

type Decoder func([]byte) string

func FetchRawContentFromUrl(url string, decoder Decoder, headers map[string]string) (string, error) {
	client := http.Client{Timeout: 30 * time.Second}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	defaultHeaders := map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9,application/json",
		"Accept-Language": "sv-SE,sv;q=0.5",
		"Connection":      "keep-alive",
		"User-Agent":      randomUserAgent(),
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
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return decoder(body), nil
}

// randomUserAgent generates a random User-Agent string for different browsers and platforms.
func randomUserAgent() string {
	rand.Seed(time.Now().UnixNano())

	browsers := []string{"Chrome", "Firefox", "Safari", "Edge"}
	platforms := []string{
		"Windows NT 10.0; Win64; x64",
		"Macintosh; Intel Mac OS X %d_%d_%d",
		"iPhone; CPU iPhone OS %d_%d like Mac OS X",
		"Linux; Android %d; Pixel 3",
	}

	// Randomly select browser and platform
	browser := browsers[rand.Intn(len(browsers))]
	platformTemplate := platforms[rand.Intn(len(platforms))]

	// Generate random version numbers
	majorVersion := rand.Intn(20) + 80 // Major version for browser (e.g., Chrome 80-99)
	minorVersion := rand.Intn(10)      // Minor version
	patchVersion := rand.Intn(1000)    // Patch version

	// Generate random OS versions
	macMajor := rand.Intn(6) + 10      // macOS version 10-15
	macMinor := rand.Intn(10)          // Minor version for macOS
	macPatch := rand.Intn(10)          // Patch version for macOS
	iosMajor := rand.Intn(3) + 13      // iOS version 13-15
	iosMinor := rand.Intn(5)           // Minor version for iOS
	androidVersion := rand.Intn(6) + 7 // Android version 7-12

	// Fill in the platform template with random OS versions
	var platform string
	switch platformTemplate {
	case "Macintosh; Intel Mac OS X %d_%d_%d":
		platform = fmt.Sprintf(platformTemplate, macMajor, macMinor, macPatch)
	case "iPhone; CPU iPhone OS %d_%d like Mac OS X":
		platform = fmt.Sprintf(platformTemplate, iosMajor, iosMinor)
	case "Linux; Android %d; Pixel 3":
		platform = fmt.Sprintf(platformTemplate, androidVersion)
	default:
		platform = platformTemplate // Windows platform doesn't require formatting
	}

	// Construct User-Agent based on browser type
	var userAgent string
	switch browser {
	case "Chrome":
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.%d.%d Safari/537.36", platform, majorVersion, minorVersion, patchVersion)
	case "Firefox":
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s; rv:%d.0) Gecko/20100101 Firefox/%d.0", platform, majorVersion, majorVersion)
	case "Safari":
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%d.0 Safari/605.1.15", platform, majorVersion)
	case "Edge":
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.%d.%d Safari/537.36 Edg/%d.0.%d.%d", platform, majorVersion, minorVersion, patchVersion, majorVersion, minorVersion, patchVersion)
	}

	return userAgent
}
