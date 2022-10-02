package crawler

import (
	"io"
	"net/http"
	"strings"
)

var (
	DecoderWindows1252 = func(runes []byte) (str string) {
		for _, r := range runes {
			str += string(r)
		}
		return
	}
	DecoderUtf8 = func(runes []byte) string {
		return string(runes)
	}
)

type Decoder func([]byte) string

func fetchHtmlFromUrl(url string, decoder Decoder) (string, error) {
	resp, err := http.Get(url)
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

func normalizeString(str string) string {
	str = strings.Replace(str, " ", "", -1)
	str = strings.Replace(str, "&nbsp;", "", -1)
	str = strings.Replace(str, "\u00A0", "", -1)

	return str
}
