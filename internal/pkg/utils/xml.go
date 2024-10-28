package utils

import (
	"errors"
	"fmt"
	"io"
	"strings"

	errors2 "github.com/ymakhloufi/bolan-compare/internal/pkg/errors"
	"golang.org/x/net/html"
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
							return nil, fmt.Errorf("failed finding section on Danske List Rates website, EOF: %w", errors2.ErrNoInterestSetFound)
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
		return nil, fmt.Errorf("failed finding section on Danske Bank website, EOF: %w", errors2.ErrNoInterestSetFound)
	}
	return nil, tokenizer.Err()
}

func PrintTable(table Table) {
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
	firstRow := true

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			err := tokenizer.Err()
			if err == io.EOF {
				return t, nil
			}
			return t, err

		case html.StartTagToken:
			tn, _ := tokenizer.TagName()
			tagName := string(tn)
			if tagName != "tr" {
				continue
			}

			row, err := parseTableRow(tokenizer)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return t, nil
				}
				return t, err
			}
			if firstRow {
				t.Header = row
				firstRow = false
			} else {
				t.Rows = append(t.Rows, row)
			}

		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			tagName := string(tn)
			if tagName == "table" {
				return t, nil
			}
		default:
			continue
		}
	}
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
