package usecase

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	// ruSeriesRe — серия номера РФ (без региона) после замены латинских двойников: "Р466УК".
	ruSeriesRe = regexp.MustCompile(`^[АВЕКМНОРСТУХ]\d{3}[АВЕКМНОРСТУХ]{2}$`)
	// ruRegionRe — регион номера РФ: 2–3 цифры.
	ruRegionRe = regexp.MustCompile(`^\d{2,3}$`)

	// Форматы, по которым узнаём номер ЕС/США в строке без разделителей (как читает OCR).
	// Это эвристика: форматы стран пересекаются, поэтому порядок проверки важен.
	usCARe   = regexp.MustCompile(`^\d[A-Z]{3}\d{3}$`)        // Калифорния: 7ABC123
	euFRITRe = regexp.MustCompile(`^[A-Z]{2}\d{3}[A-Z]{2}$`)  // Франция/Италия: AB123CD
	euESRe   = regexp.MustCompile(`^\d{4}[A-Z]{3}$`)          // Испания: 1234ABC
	euDERe   = regexp.MustCompile(`^[A-Z]{2,5}\d{1,4}[EH]?$`) // Германия: округ + буквы + цифры (MKD4419)
)

// classifyPlate приводит номер, прочитанный OCR (латиница и цифры), к формату страны:
// plate — номер для БД, plateType — plates.meta.type ("ru", "eu", "us"; "" — неизвестен),
// region — plates.meta.region (страна ЕС), если понятна по формату.
// РФ проверяется первой: латинские двойники → кириллица, "Р466УК 79".
func classifyPlate(raw string) (plate, plateType, region string) {
	raw = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToUpper(r)
		}
		return -1
	}, raw)
	if p, ok := ruPlate(raw); ok {
		return p, "ru", ""
	}
	switch {
	case usCARe.MatchString(raw):
		return raw, "us", ""
	case euFRITRe.MatchString(raw):
		return raw[:2] + " " + raw[2:5] + " " + raw[5:], "eu", ""
	case euESRe.MatchString(raw):
		return raw[:4] + " " + raw[4:], "eu", "E"
	case euDERe.MatchString(raw):
		return raw, "eu", "D"
	}
	return raw, "", ""
}

// ruPlate — номер РФ из строки OCR: серия из 6 символов и регион из 2–3 цифр (регион OCR иногда
// теряет — тогда только серия). Латинские двойники → кириллица; в буквенных позициях «0» → «О»,
// в цифровых «О» → «0». Не похоже на номер РФ — ok = false.
func ruPlate(raw string) (string, bool) {
	r := []rune(ruLatinTwins.Replace(raw))
	if n := len(r); n != 6 && n != 8 && n != 9 {
		return "", false
	}
	for i := range r {
		letterPos := i == 0 || i == 4 || i == 5
		switch {
		case letterPos && r[i] == '0':
			r[i] = 'О'
		case !letterPos && r[i] == 'О':
			r[i] = '0'
		}
	}
	series, region := string(r[:6]), string(r[6:])
	if !ruSeriesRe.MatchString(series) {
		return "", false
	}
	if region == "" {
		return series, true
	}
	if !ruRegionRe.MatchString(region) {
		return "", false
	}
	return series + " " + region, true
}

// addressLocale — язык адреса по тексту: кириллица — "ru" (с украинскими буквами — "uk"),
// иероглифы — "zh", латиница с умлаутами/ß — "de", прочая латиница — "en"; пусто — "ru".
func addressLocale(address string) string {
	var cyr, latin, han, ukr, de bool
	for _, r := range address {
		switch {
		case unicode.Is(unicode.Han, r):
			han = true
		case unicode.Is(unicode.Cyrillic, r):
			cyr = true
			ukr = ukr || strings.ContainsRune("ІіЇїЄєҐґ", r)
		case unicode.Is(unicode.Latin, r):
			latin = true
			de = de || strings.ContainsRune("ÄäÖöÜüß", r)
		}
	}
	switch {
	case han:
		return "zh"
	case cyr && ukr:
		return "uk"
	case cyr:
		return "ru"
	case de:
		return "de"
	case latin:
		return "en"
	}
	return "ru"
}
