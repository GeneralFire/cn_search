package repository

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// GenerateOptions — параметры генератора тестовых данных.
type GenerateOptions struct {
	Total    int                 // сколько записей
	Seed     uint64              // одинаковый seed → одинаковый набор
	Base     time.Time           // самое позднее возможное время появления
	DaysBack int                 // насколько далеко в прошлое разбрасывать время
	PhotoKey func(id int) string // ключ картинки в S3 для машины; nil — без фото
}

// DefaultGenerateOptions — набор, который поднимается по умолчанию.
func DefaultGenerateOptions(photoKey func(id int) string) GenerateOptions {
	return GenerateOptions{
		Total:    600,
		Seed:     20261109,
		Base:     time.Date(2026, time.November, 9, 21, 0, 0, 0, time.UTC),
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

// Fixed — фиксированные записи для ручной проверки: у А123ВС 777 три фиксации,
// у М456ОР 178 и Т789УХ 116 — по одной. Их номера в случайной части не встречаются.
var Fixed = []models.PlateSighting{
	{ID: 1, Address: "Москва, Пресненская наб., 10", Plate: "А123ВС 777", CreatedAt: time.Date(2026, 11, 2, 9, 15, 0, 0, time.UTC)},
	{ID: 2, Address: "Санкт-Петербург, Невский пр., 28", Plate: "М456ОР 178", CreatedAt: time.Date(2026, 10, 27, 18, 40, 0, 0, time.UTC)},
	{ID: 3, Address: "Казань, ул. Баумана, 44", Plate: "Т789УХ 116", CreatedAt: time.Date(2026, 11, 8, 12, 5, 0, 0, time.UTC)},
	{ID: 4, Address: "Москва, ул. Тверская, 7", Plate: "А123ВС 777", CreatedAt: time.Date(2026, 11, 5, 18, 30, 0, 0, time.UTC)},
	{ID: 5, Address: "Москва, Кутузовский пр., 30", Plate: "А123ВС 777", CreatedAt: time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)},
}

type plateOfCity struct {
	plate string
	city  city
}

// Generate детерминированно генерирует тестовые фиксации номеров. Номера берутся из пула
// (примерно треть от числа записей), поэтому у номера обычно несколько фиксаций —
// в разных местах его города. Буквы и коды регионов корректные. Первые записи — Fixed.
func Generate(opt GenerateOptions) []models.PlateSighting {
	r := rand.New(rand.NewPCG(opt.Seed, opt.Seed^0x9e3779b97f4a7c15))
	pick := func(xs []string) string { return xs[r.IntN(len(xs))] }
	letter := func() string { return string(plateLetters[r.IntN(len(plateLetters))]) }

	out := make([]models.PlateSighting, 0, max(opt.Total, len(Fixed)))
	out = append(out, Fixed...)

	seen := make(map[string]bool, opt.Total)
	for _, c := range Fixed {
		seen[c.Plate] = true
	}
	pool := make([]plateOfCity, 0, max(1, (opt.Total-len(Fixed))/3))
	for len(pool) < cap(pool) {
		c := cities[r.IntN(len(cities))]
		plate := fmt.Sprintf("%s%03d%s%s %s", letter(), r.IntN(999)+1, letter(), letter(), pick(c.regions))
		if seen[plate] {
			continue
		}
		seen[plate] = true
		pool = append(pool, plateOfCity{plate, c})
	}

	for len(out) < opt.Total {
		p := pool[r.IntN(len(pool))]
		out = append(out, models.PlateSighting{
			ID:        len(out) + 1,
			Address:   fmt.Sprintf("%s, %s, %d", p.city.name, pick(p.city.streets), r.IntN(150)+1),
			Plate:     p.plate,
			CreatedAt: opt.Base.Add(-time.Duration(r.Int64N(int64(opt.DaysBack) * int64(24*time.Hour)))).Truncate(time.Second),
		})
	}

	if opt.PhotoKey != nil {
		for i := range out {
			out[i].PhotoKey = opt.PhotoKey(out[i].ID)
		}
	}
	return out
}
