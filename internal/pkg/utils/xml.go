//nolint:revive,nolintlint // I like this package name, leave me alone
package utils

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

const (
	TagTable = "table"
	TagTr    = "tr"
	TagTh    = "th"
	TagTd    = "td"
)

type Table struct {
	Header []string
	Rows   [][]string
}

// FindTokenizedTableByTextBeforeTable returns the tokenizer for a table which occurs after a given text in an xml document.
func FindTokenizedTableByTextBeforeTable(rawHTML, stringToFind string) (*html.Tokenizer, error) {
	return FindTokenizedNthTableByTextBeforeTable(rawHTML, stringToFind, 0)
}

// FindTokenizedTableByTextInCaption returns the tokenizer for a table whose <caption> contains the given text.
func FindTokenizedTableByTextInCaption(rawHTML, stringToFind string) (*html.Tokenizer, error) {
	tokenizer := html.NewTokenizer(strings.NewReader(rawHTML))
	for node := tokenizer.Next(); node != html.ErrorToken; node = tokenizer.Next() {
		if node != html.StartTagToken {
			continue
		}

		tn, _ := tokenizer.TagName()
		if string(tn) != TagTable {
			continue
		}

		// Found a table, now look for caption
		if tableHasCaptionWithText(tokenizer, stringToFind) {
			return tokenizer, nil
		}
	}
	return nil, fmt.Errorf("failed to find table with caption containing %q: %w", stringToFind, tokenizer.Err())
}

// tableHasCaptionWithText checks if the table (starting after <table> tag) has a caption containing the text.
func tableHasCaptionWithText(tokenizer *html.Tokenizer, textToFind string) bool {
	// Look for caption tag within the next few tokens
	for range 10 {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			return false
		}
		if tt == html.StartTagToken {
			tn, _ := tokenizer.TagName()
			tagName := string(tn)
			if tagName == "caption" {
				// Extract text from caption
				captionText, _ := extractTextFromCaption(tokenizer)
				return strings.Contains(captionText, textToFind)
			}
			// If we hit thead or tbody before caption, there's no caption
			if tagName == "thead" || tagName == "tbody" || tagName == TagTr {
				return false
			}
		}
	}
	return false
}

// extractTextFromCaption extracts text content from a caption element.
func extractTextFromCaption(tokenizer *html.Tokenizer) (string, error) {
	var sb strings.Builder
	for {
		tt := tokenizer.Next()
		switch tt { //nolint: exhaustive // we only care about text and end tags
		case html.ErrorToken:
			return sb.String(), tokenizer.Err()
		case html.TextToken:
			sb.Write(tokenizer.Text())
		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			if string(tn) == "caption" {
				return NormalizeSpaces(sb.String()), nil
			}
		}
	}
}

// FindTokenizedNthTableByTextBeforeTable returns the tokenizer for the Nth table (0-indexed) which occurs after a given text.
// Use skip=0 for the first table, skip=1 to skip the first and return the second, etc.
func FindTokenizedNthTableByTextBeforeTable(rawHTML, stringToFind string, skip int) (*html.Tokenizer, error) {
	tokenizer := html.NewTokenizer(strings.NewReader(rawHTML))
	for node := tokenizer.Next(); node != html.ErrorToken; node = tokenizer.Next() {
		if node != html.TextToken {
			continue
		}

		tkn := tokenizer.Token()
		if !strings.Contains(tkn.Data, stringToFind) {
			continue
		}

		tablesFound := 0
		for {
			node = tokenizer.Next()
			switch node { //nolint: exhaustive // we only care about start tags and errors
			case html.ErrorToken:
				return nil, fmt.Errorf("failed to find table after text %q: %w", stringToFind, tokenizer.Err())
			case html.StartTagToken:
				tkn := tokenizer.Token()
				if tkn.Data == TagTable {
					if tablesFound == skip {
						return tokenizer, nil
					}
					tablesFound++
				}
			default:
				continue
			}
		}
	}
	return nil, fmt.Errorf("failed to find text %q before table: %w", stringToFind, tokenizer.Err())
}

// ParseTable parses the Table starting from the current position of the tokenizer.
func ParseTable(tokenizer *html.Tokenizer) (Table, error) {
	var t Table

	for {
		tt := tokenizer.Next()
		switch tt { //nolint: exhaustive // we only care about start tags, end tags, and errors
		case html.ErrorToken:
			return t, handleErrToken(tokenizer.Err())

		case html.StartTagToken:
			var err error
			row, err := extractTableRow(tokenizer)
			if len(row) > 0 { // even with an error, there might be a valid row
				addRowToTable(&t, row)
			}
			if errors.Is(err, io.EOF) {
				return t, nil // reached end of table or document
			}

		case html.EndTagToken:
			if isClosingTableTag(tokenizer) {
				return t, nil
			}

		default:
			continue
		}
	}
}

func addRowToTable(t *Table, row []string) {
	if t.Header == nil {
		t.Header = row
	} else {
		t.Rows = append(t.Rows, row)
	}
}

func isClosingTableTag(tokenizer *html.Tokenizer) bool {
	tn, _ := tokenizer.TagName()
	tagName := string(tn)
	return tagName == TagTable
}

func extractTableRow(tokenizer *html.Tokenizer) ([]string, error) {
	tn, _ := tokenizer.TagName()
	tagName := string(tn)
	if tagName != TagTr {
		return nil, nil
	}

	return parseTableRow(tokenizer)
}

func handleErrToken(err error) error {
	if errors.Is(err, io.EOF) {
		return nil // done, no error
	}
	return err
}

// Helper function to parse a Table row.
func parseTableRow(tokenizer *html.Tokenizer) ([]string, error) {
	var row []string

	for {
		tt := tokenizer.Next()
		switch tt { //nolint: exhaustive // we only care about start tags, end tags, and errors
		case html.ErrorToken:
			return row, handleErrToken(tokenizer.Err())

		case html.StartTagToken:
			if !isTableCell(tokenizer) {
				continue
			}

			cellContent, err := extractTextFromTd(tokenizer)
			row = append(row, cellContent)
			if err != nil {
				return row, err
			}

		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			tagName := string(tn)
			if tagName == TagTr {
				return row, nil
			}
			if tagName == TagTable {
				return row, io.EOF
			}
		default:
			continue
		}
	}
}

func isTableCell(tokenizer *html.Tokenizer) bool {
	tn, _ := tokenizer.TagName()
	tagName := string(tn)
	return tagName == TagTh || tagName == TagTd
}

// Function to extract text content from within a tag.
func extractTextFromTd(tokenizer *html.Tokenizer) (string, error) {
	var sb strings.Builder
	var err error
loop:
	for {
		tt := tokenizer.Next()
		switch tt { //nolint: exhaustive // we only care about text tokens, end tags, and errors
		case html.ErrorToken:
			err = tokenizer.Err()
			break loop

		case html.TextToken:
			sb.Write(tokenizer.Text())

		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			tag := string(tn)
			if tag == TagTd || tag == TagTh || tag == TagTr {
				break loop
			}
			if tag == TagTable {
				err = io.EOF
				break loop
			}
		default:
			continue
		}
	}

	return NormalizeSpaces(sb.String()), err
}
