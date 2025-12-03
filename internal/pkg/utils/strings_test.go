//nolint:revive,nolintlint // package name matches the package being tested
package utils

import (
	"errors"
	"testing"

	"github.com/yama6a/bolan-compare/internal/pkg/model"
)

func TestNormalizeSpaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text unchanged",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "trims leading and trailing spaces",
			input: "  hello world  ",
			want:  "hello world",
		},
		{
			name:  "collapses multiple spaces",
			input: "hello    world",
			want:  "hello world",
		},
		{
			name:  "replaces HTML non-breaking space",
			input: "hello&nbsp;world",
			want:  "hello world",
		},
		{
			name:  "replaces unicode no-break space",
			input: "hello\u00A0world",
			want:  "hello world",
		},
		{
			name:  "replaces thin space",
			input: "hello\u2009world",
			want:  "hello world",
		},
		{
			name:  "replaces hair space",
			input: "hello\u200Aworld",
			want:  "hello world",
		},
		{
			name:  "replaces zero-width space",
			input: "hello\u200Bworld",
			want:  "hello world",
		},
		{
			name:  "replaces narrow no-break space",
			input: "hello\u202Fworld",
			want:  "hello world",
		},
		{
			name:  "replaces tab",
			input: "hello\tworld",
			want:  "hello world",
		},
		{
			name:  "replaces newline",
			input: "hello\nworld",
			want:  "hello world",
		},
		{
			name:  "replaces carriage return",
			input: "hello\rworld",
			want:  "hello world",
		},
		{
			name:  "replaces vertical tab",
			input: "hello\vworld",
			want:  "hello world",
		},
		{
			name:  "replaces form feed",
			input: "hello\fworld",
			want:  "hello world",
		},
		{
			name:  "handles multiple different whitespace types",
			input: "  hello\t\n\u00A0world  ",
			want:  "hello world",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   \t\n  ",
			want:  "",
		},
		{
			name:  "Swedish characters preserved",
			input: "Snitträntor för bolån",
			want:  "Snitträntor för bolån",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeSpaces(tt.input); got != tt.want {
				t.Errorf("NormalizeSpaces() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseTerm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    model.Term
		wantErr error
	}{
		// 3 months
		{name: "3 mån", input: "3 mån", want: model.Term3months},
		{name: "3mån no space", input: "3mån", want: model.Term3months},
		{name: "3 mo", input: "3 mo", want: model.Term3months},
		{name: "3mo no space", input: "3mo", want: model.Term3months},
		{name: "3 MÅN uppercase", input: "3 MÅN", want: model.Term3months},

		// 1 year
		{name: "1 år", input: "1 år", want: model.Term1year},
		{name: "1år no space", input: "1år", want: model.Term1year},
		{name: "1 yr", input: "1 yr", want: model.Term1year},
		{name: "1yr no space", input: "1yr", want: model.Term1year},
		{name: "1 ÅR uppercase", input: "1 ÅR", want: model.Term1year},

		// 2 years
		{name: "2 år", input: "2 år", want: model.Term2years},
		{name: "2 yr", input: "2 yr", want: model.Term2years},

		// 3 years
		{name: "3 år", input: "3 år", want: model.Term3years},
		{name: "3 yr", input: "3 yr", want: model.Term3years},

		// 4 years
		{name: "4 år", input: "4 år", want: model.Term4years},
		{name: "4 yr", input: "4 yr", want: model.Term4years},

		// 5 years
		{name: "5 år", input: "5 år", want: model.Term5years},
		{name: "5 yr", input: "5 yr", want: model.Term5years},

		// 6 years
		{name: "6 år", input: "6 år", want: model.Term6years},
		{name: "6 yr", input: "6 yr", want: model.Term6years},

		// 7 years
		{name: "7 år", input: "7 år", want: model.Term7years},
		{name: "7 yr", input: "7 yr", want: model.Term7years},

		// 8 years
		{name: "8 år", input: "8 år", want: model.Term8years},
		{name: "8 yr", input: "8 yr", want: model.Term8years},

		// 9 years
		{name: "9 år", input: "9 år", want: model.Term9years},
		{name: "9 yr", input: "9 yr", want: model.Term9years},

		// 10 years
		{name: "10 år", input: "10 år", want: model.Term10years},
		{name: "10 yr", input: "10 yr", want: model.Term10years},

		// With extra whitespace
		{name: "with leading/trailing spaces", input: "  3 mån  ", want: model.Term3months},
		{name: "with non-breaking space", input: "3\u00A0mån", want: model.Term3months},

		// Header rows (should return ErrTermHeader)
		{name: "header Bindningstid", input: "Bindningstid", want: "", wantErr: ErrTermHeader},
		{name: "header bindningstid lowercase", input: "bindningstid", want: "", wantErr: ErrTermHeader},
		{name: "header Genomsnittlig", input: "Genomsnittlig", want: "", wantErr: ErrTermHeader},
		{name: "header genomsnittlig lowercase", input: "genomsnittlig ränta", want: "", wantErr: ErrTermHeader},
		{name: "header Månad", input: "Månad", want: "", wantErr: ErrTermHeader},
		{name: "header månad lowercase", input: "månad", want: "", wantErr: ErrTermHeader},
		{name: "header tot", input: "tot", want: "", wantErr: ErrTermHeader},

		// Invalid terms
		{name: "invalid empty", input: "", want: "", wantErr: nil},
		{name: "invalid random text", input: "random", want: "", wantErr: nil},
		{name: "invalid number only", input: "5", want: "", wantErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseTerm(tt.input)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseTerm() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if tt.want == "" && err == nil {
				t.Errorf("ParseTerm() expected error for invalid input %q", tt.input)
				return
			}

			if tt.want != "" {
				if err != nil {
					t.Errorf("ParseTerm() unexpected error = %v", err)
					return
				}
				if got != tt.want {
					t.Errorf("ParseTerm() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
