package usecase

import (
	"regexp"
	"strings"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// ClarificationConsensus — столько совпавших (по plate_search_key) уточнений одной фиксации
// достаточно, чтобы все они стали «приняты».
const ClarificationConsensus = 3

// Ошибки формата номера — показываются пользователю как есть.
const (
	errRUPlateFormat    = "номер РФ — в формате А123ВС 77 или А123ВС 777 (буквы А В Е К М Н О Р С Т У Х)"
	errLatinPlateFormat = "номер — от 2 до 12 символов: латинские буквы, цифры, пробел, дефис"
	errSameAsCurrent    = "уточнение совпадает с текущим номером"
)

var (
	// ruPlateRe — номер РФ после нормализации: "А123ВС 77", "А123ВС 777".
	ruPlateRe = regexp.MustCompile(`^[АВЕКМНОРСТУХ]\d{3}[АВЕКМНОРСТУХ]{2} \d{2,3}$`)
	// latinPlateRe — номера ЕС/США: проверка мягкая, форматы стран разные.
	latinPlateRe = regexp.MustCompile(`^[A-Z0-9 -]{2,12}$`)

	// ruLatinTwins — латинские буквы, которые на номере РФ не отличить от кириллических.
	ruLatinTwins = strings.NewReplacer(
		"A", "А", "B", "В", "E", "Е", "K", "К", "M", "М", "H", "Н",
		"O", "О", "P", "Р", "C", "С", "T", "Т", "Y", "У", "X", "Х",
	)
)

// normalizeClarifyPlate приводит номер уточнения к каноническому виду и проверяет формат по типу:
// пробелы по краям убраны, внутри схлопнуты до одного, верхний регистр. Для РФ ещё латинские двойники
// → кириллица, «0» в буквенных позициях → «О», пробел перед регионом ("А123ВС77" → "А123ВС 77").
// Тип не задан — формат не проверяется. Ошибка — *models.ValidationError.
func normalizeClarifyPlate(plate, plateType string) (string, error) {
	plate = strings.Join(strings.Fields(strings.ToUpper(plate)), " ")
	switch plateType {
	case "ru":
		r := []rune(ruLatinTwins.Replace(strings.ReplaceAll(plate, " ", "")))
		if len(r) != 8 && len(r) != 9 { // серия из 6 символов + регион из 2–3 цифр
			return "", &models.ValidationError{Message: errRUPlateFormat}
		}
		for _, i := range []int{0, 4, 5} { // буквенные позиции серии
			if r[i] == '0' {
				r[i] = 'О'
			}
		}
		plate = string(r[:6]) + " " + string(r[6:])
		if !ruPlateRe.MatchString(plate) {
			return "", &models.ValidationError{Message: errRUPlateFormat}
		}
	case "eu", "us":
		if !latinPlateRe.MatchString(plate) {
			return "", &models.ValidationError{Message: errLatinPlateFormat}
		}
	}
	return plate, nil
}
