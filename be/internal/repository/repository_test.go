package repository

import (
	"regexp"
	"slices"
	"testing"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

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
	perPlate, ids := map[string]int{}, map[int]bool{}
	for _, c := range got {
		if !plateRe.MatchString(c.Plate) {
			t.Errorf("некорректный номер %q", c.Plate)
		}
		if ids[c.ID] {
			t.Errorf("дубликат id: %+v", c)
		}
		ids[c.ID] = true
		perPlate[c.Plate]++
		if c.CreatedAt.After(opt.Base) || c.CreatedAt.Before(opt.Base.AddDate(0, 0, -opt.DaysBack)) {
			t.Errorf("время вне диапазона: %+v", c)
		}
		if c.PhotoKey != "/p" {
			t.Errorf("нет фото у %d", c.ID)
		}
	}

	if perPlate["А123ВС 777"] != 3 || perPlate["М456ОР 178"] != 1 {
		t.Errorf("фиксированные номера встретились и в случайной части: %d, %d", perPlate["А123ВС 777"], perPlate["М456ОР 178"])
	}
	repeated := 0
	for _, n := range perPlate {
		if n > 1 {
			repeated++
		}
	}
	if repeated < len(perPlate)/2 {
		t.Errorf("номера почти не повторяются: %d из %d", repeated, len(perPlate))
	}

	if !slices.Equal(got, Generate(opt)) {
		t.Error("генерация недетерминирована")
	}
}

func withPhoto(cs []models.PlateSighting, p string) []models.PlateSighting {
	out := slices.Clone(cs)
	for i := range out {
		out[i].PhotoKey = p
	}
	return out
}
