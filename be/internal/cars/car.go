// Package cars — предметная область: автомобили, поиск по номеру и ключу.
package cars

import (
	"strings"
	"unicode"
)

// Car — запись об автомобиле.
type Car struct {
	ID       int
	PhotoKey string // ключ объекта в S3-бакете с картинками: "car-3.png"
	Address  string
	Plate    string // госномер кириллицей: "А123ВС 777"
	Key      string // ключ: "KEY-0451"
	Date     string // YYYY-MM-DD
}

// В российских номерах используются только 12 букв, совпадающих по
// написанию с латинскими; пользователь часто набирает их латиницей.
var latToCyr = map[rune]rune{
	'A': 'А', 'B': 'В', 'E': 'Е', 'K': 'К', 'M': 'М', 'H': 'Н',
	'O': 'О', 'P': 'Р', 'C': 'С', 'T': 'Т', 'Y': 'У', 'X': 'Х',
}

// squash — верхний регистр, без пробельных символов.
func squash(s string, mapRune func(rune) rune) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToUpper(s) {
		if unicode.IsSpace(r) {
			continue
		}
		if mapRune != nil {
			r = mapRune(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// NormalizePlate приводит госномер к виду для сравнения:
// "a123bc 777" → "А123ВС777".
func NormalizePlate(s string) string {
	return squash(s, func(r rune) rune {
		if c, ok := latToCyr[r]; ok {
			return c
		}
		return r
	})
}

// NormalizeKey приводит ключ к виду для сравнения: "key-0451" → "KEY-0451".
func NormalizeKey(s string) string {
	return squash(s, nil)
}
