package cars

import (
	"context"
	"regexp"
	"slices"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct{ in, plate, key string }{
		{"а123вс 777", "А123ВС777", "А123ВС777"},
		{"A123BC", "А123ВС", "A123BC"}, // латиница → кириллица только для номера
		{" key-0451 ", "КЕУ-0451", "KEY-0451"},
		{"", "", ""},
	}
	for _, tt := range tests {
		if got := NormalizePlate(tt.in); got != tt.plate {
			t.Errorf("NormalizePlate(%q) = %q, want %q", tt.in, got, tt.plate)
		}
		if got := NormalizeKey(tt.in); got != tt.key {
			t.Errorf("NormalizeKey(%q) = %q, want %q", tt.in, got, tt.key)
		}
	}
}

func TestGenerate(t *testing.T) {
	opt := DefaultGenerateOptions(func(id int) string { return "/p" })
	got := Generate(opt)

	if len(got) != opt.Total {
		t.Fatalf("len = %d, want %d", len(got), opt.Total)
	}
	if !slices.Equal(got[:len(Fixed)], withPhoto(Fixed, "/p")) {
		t.Errorf("первые записи должны быть Fixed")
	}

	plateRe := regexp.MustCompile(`^[АВЕКМНОРСТУХ]\d{3}[АВЕКМНОРСТУХ]{2} \d{2,3}$`)
	plates, keys, ids := map[string]bool{}, map[string]bool{}, map[int]bool{}
	for _, c := range got {
		if !plateRe.MatchString(c.Plate) {
			t.Errorf("некорректный номер %q", c.Plate)
		}
		if plates[c.Plate] || keys[c.Key] || ids[c.ID] {
			t.Errorf("дубликат: %+v", c)
		}
		plates[c.Plate], keys[c.Key], ids[c.ID] = true, true, true
		if c.PhotoKey != "/p" {
			t.Errorf("нет фото у %d", c.ID)
		}
	}

	if !slices.Equal(got, Generate(opt)) {
		t.Error("генерация недетерминирована")
	}
}

func withPhoto(cs []Car, p string) []Car {
	out := slices.Clone(cs)
	for i := range out {
		out[i].PhotoKey = p
	}
	return out
}

func ids(cs []Car) []int {
	out := make([]int, len(cs))
	for i, c := range cs {
		out[i] = c.ID
	}
	return out
}

func TestSearch(t *testing.T) {
	s := NewMemoryStore(Fixed) // даты: 3 — 08.11, 1 — 02.11, 2 — 27.10
	ctx := context.Background()

	tests := []struct {
		name string
		q    Query
		want []int
	}{
		{"пусто — всё, новые первыми", Query{Limit: 10}, []int{3, 1, 2}},
		{"номер кириллицей с пробелом", Query{Plate: "а123вс 777", Limit: 10}, []int{1}},
		{"номер латиницей", Query{Plate: "A123BC", Limit: 10}, []int{1}},
		{"только ключ", Query{Key: "key-1187", Limit: 10}, []int{2}},
		{"номер И ключ совпадают", Query{Plate: "А123", Key: "0451", Limit: 10}, []int{1}},
		{"номер И ключ от разных машин", Query{Plate: "А123", Key: "1187", Limit: 10}, []int{}},
		{"ничего", Query{Plate: "Х000ХХ", Limit: 10}, []int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := s.Search(ctx, tt.q)
			if err != nil {
				t.Fatal(err)
			}
			if got := ids(p.Items); !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if p.Total != len(tt.want) {
				t.Errorf("total = %d, want %d", p.Total, len(tt.want))
			}
		})
	}
}

func TestSearchPagination(t *testing.T) {
	s := NewMemoryStore(Generate(DefaultGenerateOptions(nil)))
	ctx := context.Background()

	var all []int
	offset, pages := 0, 0
	for {
		p, err := s.Search(ctx, Query{Offset: offset, Limit: 35})
		if err != nil {
			t.Fatal(err)
		}
		if p.Total != 600 {
			t.Fatalf("total = %d", p.Total)
		}
		all = append(all, ids(p.Items)...)
		pages++
		if p.NextOffset == nil {
			break
		}
		if *p.NextOffset != offset+35 {
			t.Fatalf("next_offset = %d, want %d", *p.NextOffset, offset+35)
		}
		offset = *p.NextOffset
	}
	if len(all) != 600 || pages != 18 { // 17 полных по 35 + 5
		t.Fatalf("собрано %d записей за %d страниц", len(all), pages)
	}
	seen := map[int]bool{}
	for _, id := range all {
		if seen[id] {
			t.Fatalf("id %d встретился дважды", id)
		}
		seen[id] = true
	}

	// За пределами — пустая страница без next_offset, но с правильным total.
	p, _ := s.Search(ctx, Query{Offset: 1000, Limit: 35})
	if len(p.Items) != 0 || p.NextOffset != nil || p.Total != 600 {
		t.Errorf("за пределами: %+v", p)
	}
}

func TestGet(t *testing.T) {
	s := NewMemoryStore(Fixed)
	c, err := s.Get(context.Background(), 2)
	if err != nil || c.Plate != "М456ОР 178" {
		t.Errorf("Get(2) = %+v, %v", c, err)
	}
	if _, err := s.Get(context.Background(), 999); err != ErrNotFound {
		t.Errorf("Get(999) err = %v, want ErrNotFound", err)
	}
}
