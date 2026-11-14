package models

import (
	"strings"
	"unicode"
)

// MaskMinKnown — сколько известных символов (не «*») должно быть в маске поиска.
const MaskMinKnown = 3

// CleanMask оставляет в маске только буквы, цифры, пробелы и «*» (остальное отбрасывается) и
// возвращает число известных символов — без «*» и пробелов.
func CleanMask(s string) (clean string, known int) {
	clean = strings.Map(func(r rune) rune {
		if r == '*' || unicode.IsSpace(r) || unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
	for _, r := range clean {
		if r != '*' && !unicode.IsSpace(r) {
			known++
		}
	}
	return clean, known
}
