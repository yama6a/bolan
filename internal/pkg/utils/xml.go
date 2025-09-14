package utils

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

var (
	errIsNoTrTag  = errors.New("not a tr tag")
	ErrIsFirstRow = errors.New("first row")
)

// Define the Table struct
type Table struct {
	Header []string
	Rows   [][]string
}

// FindTokenizedTableByTextBeforeTable returns the tokenizer for a table which occurs after a given text in an xml document
func FindTokenizedTableByTextBeforeTable(rawHtml, stringToFind string) (*html.Tokenizer, error) {
	tokenizer := html.NewTokenizer(strings.NewReader(rawHtml))
	for node := tokenizer.Next(); node != html.ErrorToken; node = tokenizer.Next() {
		if node == html.TextToken {
			tkn := tokenizer.Token()
			if strings.Contains(tkn.Data, stringToFind) {
				for {
					node = tokenizer.Next()
					switch node {
					case html.ErrorToken:
						if tokenizer.Err() == io.EOF {
							return nil, fmt.Errorf("failed finding section on Danske List Rates website, EOF: %w", ErrNoInterestSetFound)
						}
						return nil, tokenizer.Err()
					case html.StartTagToken:
						tkn := tokenizer.Token()
						if tkn.Data == "table" {
							return tokenizer, nil
						}
					default:
						continue
					}
				}
			}
		}
	}
	if tokenizer.Err() == io.EOF {
		return nil, fmt.Errorf("failed finding section on Danske Bank website, EOF: %w", ErrNoInterestSetFound)
	}
	return nil, tokenizer.Err()
}

// PrintTable prints the Table to standard output for debugging purposes
func PrintTable(table Table) { //nolint: unused // useful for debugging
	for _, string := range table.Header {
		fmt.Print(string, "|")
	}
	fmt.Println("\n--------------------")
	for _, row := range table.Rows {
		for _, string := range row {
			fmt.Print(string, "|")
		}
		fmt.Println()
	}
}

// ParseTable parses the Table starting from the current position of the tokenizer
func ParseTable(tokenizer *html.Tokenizer) (Table, error) {
	var t Table

	for {
		tt := tokenizer.Next()
		switch tt {
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
	return tagName == "table"
}

func extractTableRow(tokenizer *html.Tokenizer) ([]string, error) {
	tn, _ := tokenizer.TagName()
	tagName := string(tn)
	if tagName != "tr" {
		return nil, errIsNoTrTag
	}

	return parseTableRow(tokenizer)
}

func handleErrToken(err error) error {
	if err == io.EOF {
		return nil // done, no error
	}
	return err
}

// Helper function to parse a Table row
func parseTableRow(tokenizer *html.Tokenizer) ([]string, error) {
	var row []string

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return row, tokenizer.Err()

		case html.StartTagToken:
			tn, _ := tokenizer.TagName()
			tagName := string(tn)
			if tagName != "th" && tagName != "td" {
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
			if tagName == "tr" {
				return row, nil
			}
			if tagName == "table" {
				return row, io.EOF
			}
		default:
			continue
		}
	}
}

// Function to extract text content from within a tag
func extractTextFromTd(tokenizer *html.Tokenizer) (string, error) {
	var sb strings.Builder
	var err error
loop:
	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			err = tokenizer.Err()
			break loop

		case html.TextToken:
			sb.WriteString(string(tokenizer.Text()))

		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			tag := string(tn)
			if tag == "th" || tag == "td" {
				break loop
			}
			if tag == "table" {
				err = io.EOF
				break loop
			}
		default:
			continue
		}
	}

	return NormalizeSpaces(sb.String()), err
}
