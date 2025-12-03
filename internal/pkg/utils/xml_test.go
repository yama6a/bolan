//nolint:revive,nolintlint // package name matches the package being tested
package utils

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestFindTokenizedTableByTextBeforeTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		html         string
		searchText   string
		wantErr      bool
		wantFirstRow []string // first data row to verify correct table found
	}{
		{
			name: "finds table after text",
			html: `<html>
				<body>
					<h1>Rates</h1>
					<p>Current mortgage rates</p>
					<table>
						<tr><th>Term</th><th>Rate</th></tr>
						<tr><td>3 mån</td><td>3.45%</td></tr>
					</table>
				</body>
			</html>`,
			searchText:   "Current mortgage rates",
			wantErr:      false,
			wantFirstRow: []string{"Term", "Rate"},
		},
		{
			name: "finds table with partial text match",
			html: `<html>
				<body>
					<p>Snitträntor för bolån</p>
					<table>
						<tr><th>Bindningstid</th><th>Ränta</th></tr>
						<tr><td>1 år</td><td>2.50%</td></tr>
					</table>
				</body>
			</html>`,
			searchText:   "Snitträntor",
			wantErr:      false,
			wantFirstRow: []string{"Bindningstid", "Ränta"},
		},
		{
			name: "text not found",
			html: `<html>
				<body>
					<p>Some other content</p>
					<table>
						<tr><td>Data</td></tr>
					</table>
				</body>
			</html>`,
			searchText: "Nonexistent text",
			wantErr:    true,
		},
		{
			name: "table not found after text",
			html: `<html>
				<body>
					<table>
						<tr><td>Before</td></tr>
					</table>
					<p>Target text but no table after</p>
				</body>
			</html>`,
			searchText: "Target text",
			wantErr:    true,
		},
		{
			name: "finds first table when multiple exist after text",
			html: `<html>
				<body>
					<p>Section header</p>
					<table>
						<tr><th>First</th></tr>
					</table>
					<table>
						<tr><th>Second</th></tr>
					</table>
				</body>
			</html>`,
			searchText:   "Section header",
			wantErr:      false,
			wantFirstRow: []string{"First"},
		},
		{
			name:       "empty html",
			html:       "",
			searchText: "anything",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokenizer, err := FindTokenizedTableByTextBeforeTable(tt.html, tt.searchText)

			if tt.wantErr {
				if err == nil {
					t.Error("FindTokenizedTableByTextBeforeTable() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("FindTokenizedTableByTextBeforeTable() unexpected error: %v", err)
			}

			// Verify we found the correct table by parsing it
			table, err := ParseTable(tokenizer)
			if err != nil {
				t.Fatalf("ParseTable() error: %v", err)
			}

			if len(table.Header) != len(tt.wantFirstRow) {
				t.Errorf("Header length = %d, want %d", len(table.Header), len(tt.wantFirstRow))
				return
			}

			for i, want := range tt.wantFirstRow {
				if table.Header[i] != want {
					t.Errorf("Header[%d] = %q, want %q", i, table.Header[i], want)
				}
			}
		})
	}
}

func TestFindTokenizedNthTableByTextBeforeTable(t *testing.T) {
	t.Parallel()

	htmlWithMultipleTables := `<html>
		<body>
			<p>Tables section</p>
			<table>
				<tr><th>Table0</th></tr>
				<tr><td>Data0</td></tr>
			</table>
			<table>
				<tr><th>Table1</th></tr>
				<tr><td>Data1</td></tr>
			</table>
			<table>
				<tr><th>Table2</th></tr>
				<tr><td>Data2</td></tr>
			</table>
		</body>
	</html>`

	tests := []struct {
		name         string
		html         string
		searchText   string
		skip         int
		wantErr      bool
		wantFirstRow []string
	}{
		{
			name:         "skip=0 finds first table",
			html:         htmlWithMultipleTables,
			searchText:   "Tables section",
			skip:         0,
			wantErr:      false,
			wantFirstRow: []string{"Table0"},
		},
		{
			name:         "skip=1 finds second table",
			html:         htmlWithMultipleTables,
			searchText:   "Tables section",
			skip:         1,
			wantErr:      false,
			wantFirstRow: []string{"Table1"},
		},
		{
			name:         "skip=2 finds third table",
			html:         htmlWithMultipleTables,
			searchText:   "Tables section",
			skip:         2,
			wantErr:      false,
			wantFirstRow: []string{"Table2"},
		},
		{
			name:       "skip too high returns error",
			html:       htmlWithMultipleTables,
			searchText: "Tables section",
			skip:       10,
			wantErr:    true,
		},
		{
			name: "skip works with single table",
			html: `<html>
				<body>
					<p>Single</p>
					<table>
						<tr><th>Only</th></tr>
					</table>
				</body>
			</html>`,
			searchText:   "Single",
			skip:         0,
			wantErr:      false,
			wantFirstRow: []string{"Only"},
		},
		{
			name: "skip=1 with single table returns error",
			html: `<html>
				<body>
					<p>Single</p>
					<table>
						<tr><th>Only</th></tr>
					</table>
				</body>
			</html>`,
			searchText: "Single",
			skip:       1,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokenizer, err := FindTokenizedNthTableByTextBeforeTable(tt.html, tt.searchText, tt.skip)

			if tt.wantErr {
				if err == nil {
					t.Error("FindTokenizedNthTableByTextBeforeTable() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("FindTokenizedNthTableByTextBeforeTable() unexpected error: %v", err)
			}

			table, err := ParseTable(tokenizer)
			if err != nil {
				t.Fatalf("ParseTable() error: %v", err)
			}

			if len(table.Header) != len(tt.wantFirstRow) {
				t.Errorf("Header length = %d, want %d", len(table.Header), len(tt.wantFirstRow))
				return
			}

			for i, want := range tt.wantFirstRow {
				if table.Header[i] != want {
					t.Errorf("Header[%d] = %q, want %q", i, table.Header[i], want)
				}
			}
		})
	}
}

// createTableTokenizer creates a tokenizer positioned after the opening table tag.
func createTableTokenizer(t *testing.T, rawHTML string) *html.Tokenizer {
	t.Helper()

	tokenizer := html.NewTokenizer(strings.NewReader(rawHTML))
	for {
		tok := tokenizer.Next()
		if tok == html.ErrorToken {
			t.Fatal("Could not find table tag in test HTML")
		}
		if tok == html.StartTagToken {
			tn, _ := tokenizer.TagName()
			if string(tn) == TagTable {
				return tokenizer
			}
		}
	}
}

// assertTableHeader checks that the table header matches the expected values.
func assertTableHeader(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("Header length = %d, want %d", len(got), len(want))
		return
	}

	for i, w := range want {
		if got[i] != w {
			t.Errorf("Header[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// assertTableRows checks that the table rows match the expected values.
func assertTableRows(t *testing.T, got, want [][]string) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("Rows length = %d, want %d", len(got), len(want))
		return
	}

	for i, wantRow := range want {
		if len(got[i]) != len(wantRow) {
			t.Errorf("Row[%d] length = %d, want %d", i, len(got[i]), len(wantRow))
			continue
		}
		for j, w := range wantRow {
			if got[i][j] != w {
				t.Errorf("Rows[%d][%d] = %q, want %q", i, j, got[i][j], w)
			}
		}
	}
}

func TestParseTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		html       string
		wantHeader []string
		wantRows   [][]string
	}{
		{
			name: "simple table with th header",
			html: `<table>
				<tr><th>Col1</th><th>Col2</th></tr>
				<tr><td>A</td><td>B</td></tr>
				<tr><td>C</td><td>D</td></tr>
			</table>`,
			wantHeader: []string{"Col1", "Col2"},
			wantRows:   [][]string{{"A", "B"}, {"C", "D"}},
		},
		{
			name: "table with td header (first row becomes header)",
			html: `<table>
				<tr><td>Header1</td><td>Header2</td></tr>
				<tr><td>Value1</td><td>Value2</td></tr>
			</table>`,
			wantHeader: []string{"Header1", "Header2"},
			wantRows:   [][]string{{"Value1", "Value2"}},
		},
		{
			name: "table with thead and tbody",
			html: `<table>
				<thead>
					<tr><th>Name</th><th>Rate</th></tr>
				</thead>
				<tbody>
					<tr><td>Nordea</td><td>3.5%</td></tr>
					<tr><td>SEB</td><td>3.4%</td></tr>
				</tbody>
			</table>`,
			wantHeader: []string{"Name", "Rate"},
			wantRows:   [][]string{{"Nordea", "3.5%"}, {"SEB", "3.4%"}},
		},
		{
			name:       "empty table",
			html:       `<table></table>`,
			wantHeader: nil,
			wantRows:   nil,
		},
		{
			name:       "table with empty rows",
			html:       `<table><tr></tr><tr></tr></table>`,
			wantHeader: nil,
			wantRows:   nil,
		},
		{
			name: "table with nested elements in cells",
			html: `<table>
				<tr><th><strong>Bold Header</strong></th></tr>
				<tr><td><a href="#">Link Text</a></td></tr>
			</table>`,
			wantHeader: []string{"Bold Header"},
			wantRows:   [][]string{{"Link Text"}},
		},
		{
			name: "table with whitespace normalization",
			html: `<table>
				<tr><th>  Spaced   Header  </th></tr>
				<tr><td>	Tabbed	Value	</td></tr>
			</table>`,
			wantHeader: []string{"Spaced Header"},
			wantRows:   [][]string{{"Tabbed Value"}},
		},
		{
			name: "table with Swedish characters",
			html: `<table>
				<tr><th>Bindningstid</th><th>Ränta</th></tr>
				<tr><td>3 mån</td><td>3,45 %</td></tr>
				<tr><td>1 år</td><td>2,50 %</td></tr>
			</table>`,
			wantHeader: []string{"Bindningstid", "Ränta"},
			wantRows:   [][]string{{"3 mån", "3,45 %"}, {"1 år", "2,50 %"}},
		},
		{
			name: "table with non-breaking spaces",
			html: `<table>
				<tr><th>Term</th></tr>
				<tr><td>3&nbsp;mån</td></tr>
			</table>`,
			wantHeader: []string{"Term"},
			wantRows:   [][]string{{"3 mån"}},
		},
		{
			name: "table with mixed th and td in same row",
			html: `<table>
				<tr><th>Header</th><td>Also Header</td></tr>
				<tr><td>Data1</td><td>Data2</td></tr>
			</table>`,
			wantHeader: []string{"Header", "Also Header"},
			wantRows:   [][]string{{"Data1", "Data2"}},
		},
		{
			name: "single row table (header only)",
			html: `<table>
				<tr><th>Only Header</th></tr>
			</table>`,
			wantHeader: []string{"Only Header"},
			wantRows:   nil,
		},
		{
			name: "table with multiple text nodes in cell",
			html: `<table>
				<tr><td>Part1<br/>Part2</td></tr>
			</table>`,
			wantHeader: []string{"Part1Part2"},
			wantRows:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokenizer := createTableTokenizer(t, tt.html)
			table, err := ParseTable(tokenizer)
			if err != nil {
				t.Fatalf("ParseTable() unexpected error: %v", err)
			}

			assertTableHeader(t, table.Header, tt.wantHeader)
			assertTableRows(t, table.Rows, tt.wantRows)
		})
	}
}

func TestParseTable_EndOfDocument(t *testing.T) {
	t.Parallel()

	// Test case where table is not properly closed (document ends)
	rawHTML := `<table><tr><td>Unclosed`

	tokenizer := createTableTokenizer(t, rawHTML)
	table, err := ParseTable(tokenizer)
	if err != nil {
		t.Fatalf("ParseTable() error = %v, want nil (graceful handling)", err)
	}

	// Should still capture partial data
	if len(table.Header) != 1 || table.Header[0] != "Unclosed" {
		t.Errorf("Header = %v, want [Unclosed]", table.Header)
	}
}

func TestTable_StructFields(t *testing.T) {
	t.Parallel()

	// Test that Table struct has expected fields
	table := Table{
		Header: []string{"A", "B"},
		Rows:   [][]string{{"1", "2"}, {"3", "4"}},
	}

	if len(table.Header) != 2 {
		t.Errorf("Header length = %d, want 2", len(table.Header))
	}

	if len(table.Rows) != 2 {
		t.Errorf("Rows length = %d, want 2", len(table.Rows))
	}

	if table.Header[0] != "A" || table.Header[1] != "B" {
		t.Errorf("Header = %v, want [A B]", table.Header)
	}

	if table.Rows[0][0] != "1" || table.Rows[1][1] != "4" {
		t.Errorf("Rows not as expected: %v", table.Rows)
	}
}

func TestTagConstants(t *testing.T) {
	t.Parallel()

	// Verify tag constants are correct
	if TagTable != "table" {
		t.Errorf("TagTable = %q, want %q", TagTable, "table")
	}
	if TagTr != "tr" {
		t.Errorf("TagTr = %q, want %q", TagTr, "tr")
	}
	if TagTh != "th" {
		t.Errorf("TagTh = %q, want %q", TagTh, "th")
	}
	if TagTd != "td" {
		t.Errorf("TagTd = %q, want %q", TagTd, "td")
	}
}
