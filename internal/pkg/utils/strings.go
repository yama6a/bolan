//nolint:revive,nolintlint // I like this package name, leave me alone
package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/yama6a/bolan-compare/internal/pkg/model"
)

var ErrTermHeader = errors.New("row is a term header row")

func NormalizeSpaces(str string) string {
	str = strings.ReplaceAll(str, "&nbsp;", " ") // html non-breaking space
	str = strings.ReplaceAll(str, "\u00A0", " ") // no-break space
	str = strings.ReplaceAll(str, "\u0085", " ") // next line
	str = strings.ReplaceAll(str, "\u2009", " ") // thin space
	str = strings.ReplaceAll(str, "\u200A", " ") // hair space
	str = strings.ReplaceAll(str, "\u200B", " ") // zero-width space
	str = strings.ReplaceAll(str, "\u200C", " ") // zero-width non-joiner
	str = strings.ReplaceAll(str, "\u200D", " ") // zero-width joiner
	str = strings.ReplaceAll(str, "\uFEFF", " ") // zero-width non-breaking space
	str = strings.ReplaceAll(str, "\u202F", " ") // narrow no-break space
	str = strings.ReplaceAll(str, "\t", " ")     // tab
	str = strings.ReplaceAll(str, "\n", " ")     // newline
	str = strings.ReplaceAll(str, "\r", " ")     // carriage return
	str = strings.ReplaceAll(str, "\v", " ")     // vertical tab
	str = strings.ReplaceAll(str, "\f", " ")     // form feed
	str = strings.Join(strings.Fields(str), " ") // replace consecutive spaces with single space
	str = strings.TrimSpace(str)                 // remove leading and trailing spaces

	return str
}

// nolint: cyclop // it's just one big switch statement, still readable
func ParseTerm(data string) (model.Term, error) {
	str := NormalizeSpaces(data)
	str = strings.ToLower(str)
	str = strings.ReplaceAll(str, " ", "")

	switch {
	case strings.Contains(str, "Genomsnittlig"), strings.Contains(str, "Bindningstid"), str == "tot":
		return "", ErrTermHeader
	case strings.Contains(str, "3mån"), strings.Contains(str, "3mo"):
		return model.Term3months, nil
	case strings.Contains(str, "1år"), strings.Contains(str, "1yr"):
		return model.Term1year, nil
	case strings.Contains(str, "2år"), strings.Contains(str, "2yr"):
		return model.Term2years, nil
	case strings.Contains(str, "3år"), strings.Contains(str, "3yr"):
		return model.Term3years, nil
	case strings.Contains(str, "4år"), strings.Contains(str, "4yr"):
		return model.Term4years, nil
	case strings.Contains(str, "5år"), strings.Contains(str, "5yr"):
		return model.Term5years, nil
	case strings.Contains(str, "6år"), strings.Contains(str, "6yr"):
		return model.Term6years, nil
	case strings.Contains(str, "7år"), strings.Contains(str, "7yr"):
		return model.Term7years, nil
	case strings.Contains(str, "8år"), strings.Contains(str, "8yr"):
		return model.Term8years, nil
	case strings.Contains(str, "9år"), strings.Contains(str, "9yr"):
		return model.Term9years, nil
	case strings.Contains(str, "10år"), strings.Contains(str, "10yr"):
		return model.Term10years, nil
	}

	return "", fmt.Errorf("could not parse term: %s", data)
}
