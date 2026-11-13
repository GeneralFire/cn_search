package repository

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

func hasID(items []models.PlateSighting, id string) (models.PlateSighting, bool) {
	i := slices.IndexFunc(items, func(s models.PlateSighting) bool { return s.ID == id })
	if i < 0 {
		return models.PlateSighting{}, false
	}
	return items[i], true
}

// Скрытая фиксация пропадает из Search/Get/SeeAlso без IncludeHidden и остаётся (hidden = true) с ним.
func TestHideSightingSQL(t *testing.T) {
	const sightingID = "00000000-0000-0000-0000-000000000001" // тестовая фиксация А123ВС 777
	r := New()
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		if _, err := r.HideSighting(ctx, tx, "00000000-0000-0000-0000-0000000fffff", true); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("нет фиксации: %v", err)
		}
		s, err := r.HideSighting(ctx, tx, sightingID, true)
		if err != nil || !s.Hidden || s.Plate != "А123ВС 777" {
			t.Fatalf("HideSighting: %+v %v", s, err)
		}

		for _, admin := range []bool{false, true} {
			page, err := r.Search(ctx, tx, models.SearchOpts{Plate: "А123ВС", Limit: 100, IncludeTestData: true, IncludeHidden: admin})
			if err != nil {
				t.Fatal(err)
			}
			// В поиске по номеру — последняя фиксация; у скрытой она может быть другой, поэтому
			// проверяем только, что без IncludeHidden скрытая не отдаётся.
			if got, ok := hasID(page.Items, sightingID); !admin && ok || ok && !got.Hidden {
				t.Errorf("Search admin=%v: %+v", admin, page.Items)
			}

			same, err := r.ListSamePlate(ctx, tx, models.SamePlateOpts{ID: sightingID, IncludeTestData: true, IncludeHidden: admin})
			if err != nil && !(errors.Is(err, models.ErrNotFound) && !admin) {
				t.Fatal(err)
			}
			if got, ok := hasID(same, sightingID); ok != admin || ok && !got.Hidden {
				t.Errorf("ListSamePlate admin=%v: %+v", admin, same)
			}

			see, err := r.SeeAlso(ctx, tx, models.SeeAlsoOpts{Limit: 1000, IncludeTestData: true, IncludeHidden: admin})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := hasID(see, sightingID); !admin && ok {
				t.Errorf("SeeAlso: скрытая фиксация в выдаче")
			}
		}

		if s, err := r.HideSighting(ctx, tx, sightingID, false); err != nil || s.Hidden {
			t.Fatalf("показать: %+v %v", s, err)
		}
		same, err := r.ListSamePlate(ctx, tx, models.SamePlateOpts{ID: sightingID, IncludeTestData: true})
		if _, ok := hasID(same, sightingID); err != nil || !ok {
			t.Errorf("после показа: %+v %v", same, err)
		}
	})
}

func TestClarificationsAdminSQL(t *testing.T) {
	r := New()
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		// Без include-test-data тестовые уточнения не видны.
		page, err := r.ListClarifications(ctx, tx, models.ListClarificationsOpts{Limit: 30})
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range page.Items {
			if it.Sighting.ID == "" {
				t.Errorf("без фиксации: %+v", it)
			}
		}
		page, err = r.ListClarifications(ctx, tx, models.ListClarificationsOpts{Status: models.ClarificationNotAccepted, Limit: 1, IncludeTestData: true})
		if err != nil || page.Total == 0 || len(page.Items) != 1 {
			t.Fatalf("ListClarifications: %+v %v", page, err)
		}
		it := page.Items[0]
		if it.Status != models.ClarificationNotAccepted || it.Sighting.ID != it.SightingID || it.Sighting.Plate == "" {
			t.Errorf("элемент: %+v", it)
		}
		if page.Total > 1 && (page.NextOffset == nil || *page.NextOffset != 1) {
			t.Errorf("next_offset: %v", page.NextOffset)
		}

		bySighting, err := r.ListClarifications(ctx, tx, models.ListClarificationsOpts{SightingID: it.SightingID, Limit: 100, IncludeTestData: true})
		if err != nil || bySighting.Total == 0 {
			t.Fatalf("фильтр по фиксации: %+v %v", bySighting, err)
		}
		for _, x := range bySighting.Items {
			if x.SightingID != it.SightingID {
				t.Errorf("чужая фиксация: %+v", x)
			}
		}

		c, err := r.SetClarificationStatus(ctx, tx, it.ID, models.ClarificationAccepted)
		if err != nil || c.Status != models.ClarificationAccepted || c.ID != it.ID {
			t.Fatalf("SetClarificationStatus: %+v %v", c, err)
		}
		after, err := r.ListClarifications(ctx, tx, models.ListClarificationsOpts{Status: models.ClarificationNotAccepted, Limit: 1, IncludeTestData: true})
		if err != nil || after.Total != page.Total-1 {
			t.Errorf("после принятия: %d, было %d (%v)", after.Total, page.Total, err)
		}
		if _, err := r.SetClarificationStatus(ctx, tx, "00000000-0000-0000-0000-0000000fffff", models.ClarificationAccepted); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("нет уточнения: %v", err)
		}
	})
}
