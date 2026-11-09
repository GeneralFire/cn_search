package cars

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// GenerateOptions — параметры генератора тестовых данных.
type GenerateOptions struct {
	Total    int                 // сколько машин
	Seed     uint64              // одинаковый seed → одинаковый набор
	Base     time.Time           // самая поздняя возможная дата
	DaysBack int                 // насколько далеко в прошлое разбрасывать даты
	PhotoKey func(id int) string // ключ картинки в S3 для машины; nil — без фото
}

// DefaultGenerateOptions — набор, который поднимается по умолчанию.
func DefaultGenerateOptions(photoKey func(id int) string) GenerateOptions {
	return GenerateOptions{
		Total:    600,
		Seed:     20261109,
		Base:     time.Date(2026, time.November, 9, 0, 0, 0, 0, time.UTC),
		DaysBack: 180,
		PhotoKey: photoKey,
	}
}

// Буквы, допустимые в российских госномерах.
var plateLetters = []rune("АВЕКМНОРСТУХ")

type city struct {
	name    string
	regions []string
	streets []string
}

var cities = []city{
	{"Москва", []string{"77", "97", "99", "177", "197", "199", "777", "797", "799", "977"},
		[]string{"Пресненская наб.", "ул. Тверская", "Ленинградский пр.", "ул. Арбат", "Кутузовский пр.", "ул. Покровка", "Профсоюзная ул.", "ул. Большая Ордынка"}},
	{"Санкт-Петербург", []string{"78", "98", "178", "198"},
		[]string{"Невский пр.", "Лиговский пр.", "ул. Рубинштейна", "Московский пр.", "наб. реки Фонтанки", "Большой пр. П.С."}},
	{"Казань", []string{"16", "116", "716"}, []string{"ул. Баумана", "ул. Пушкина", "пр. Победы", "ул. Петербургская"}},
	{"Екатеринбург", []string{"66", "96", "196"}, []string{"ул. Ленина", "ул. Малышева", "ул. Вайнера", "пр. Космонавтов"}},
	{"Новосибирск", []string{"54", "154"}, []string{"Красный пр.", "ул. Ленина", "ул. Кирова", "Вокзальная магистраль"}},
	{"Нижний Новгород", []string{"52", "152"}, []string{"ул. Большая Покровская", "ул. Рождественская", "пр. Гагарина"}},
	{"Краснодар", []string{"23", "93", "123", "193"}, []string{"ул. Красная", "ул. Северная", "ул. Ставропольская"}},
	{"Самара", []string{"63", "163"}, []string{"ул. Куйбышева", "Московское ш.", "ул. Ленинградская"}},
}

// Fixed — три фиксированные записи, на них удобно проверять поиск руками.
var Fixed = []Car{
	{ID: 1, Address: "Москва, Пресненская наб., 10", Plate: "А123ВС 777", Key: "KEY-0451", Date: "2026-11-02"},
	{ID: 2, Address: "Санкт-Петербург, Невский пр., 28", Plate: "М456ОР 178", Key: "KEY-1187", Date: "2026-10-27"},
	{ID: 3, Address: "Казань, ул. Баумана, 44", Plate: "Т789УХ 116", Key: "KEY-2093", Date: "2026-11-08"},
}

// Generate детерминированно генерирует тестовые машины: уникальные номера
// и ключи, корректные буквы и коды регионов. Первые записи — Fixed.
func Generate(opt GenerateOptions) []Car {
	r := rand.New(rand.NewPCG(opt.Seed, opt.Seed^0x9e3779b97f4a7c15))
	pick := func(xs []string) string { return xs[r.IntN(len(xs))] }
	letter := func() string { return string(plateLetters[r.IntN(len(plateLetters))]) }

	out := make([]Car, 0, max(opt.Total, len(Fixed)))
	out = append(out, Fixed...)
	plates := make(map[string]struct{}, opt.Total)
	keys := make(map[string]struct{}, opt.Total)
	for _, c := range Fixed {
		plates[c.Plate] = struct{}{}
		keys[c.Key] = struct{}{}
	}

	for len(out) < opt.Total {
		c := cities[r.IntN(len(cities))]
		plate := fmt.Sprintf("%s%03d%s%s %s", letter(), r.IntN(999)+1, letter(), letter(), pick(c.regions))
		key := fmt.Sprintf("KEY-%04d", r.IntN(10000))
		if _, dup := plates[plate]; dup {
			continue
		}
		if _, dup := keys[key]; dup {
			continue
		}
		plates[plate] = struct{}{}
		keys[key] = struct{}{}

		out = append(out, Car{
			ID:      len(out) + 1,
			Address: fmt.Sprintf("%s, %s, %d", c.name, pick(c.streets), r.IntN(150)+1),
			Plate:   plate,
			Key:     key,
			Date:    opt.Base.AddDate(0, 0, -r.IntN(opt.DaysBack)).Format(time.DateOnly),
		})
	}

	if opt.PhotoKey != nil {
		for i := range out {
			out[i].PhotoKey = opt.PhotoKey(out[i].ID)
		}
	}
	return out
}
