package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

func TestApiTokensSQL(t *testing.T) {
	const (
		sightingID = "00000000-0000-0000-0000-000000000001" // тестовая фиксация А123ВС 777
		tokenID    = "aaaaaaaa-0000-0000-0000-000000000001"
	)
	r := New()
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		tok, err := r.CreateApiToken(ctx, tx, models.CreateApiTokenOpts{ID: tokenID, Name: "тест", Requests: 2, SignatureHash: "h1"})
		if err != nil || tok.RequestsLeft != 2 || tok.RequestsTotal != 2 || tok.LastUsedAt != nil {
			t.Fatalf("CreateApiToken: %+v %v", tok, err)
		}
		if _, err := r.FindApiToken(ctx, tx, tokenID, "чужой"); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("чужой hash: %v", err)
		}
		if f, err := r.FindApiToken(ctx, tx, tokenID, "h1"); err != nil || f.ID != tokenID {
			t.Fatalf("FindApiToken: %+v %v", f, err)
		}

		// Два списания проходят, третье — нет; остаток не уходит в минус.
		for i, want := range []bool{true, true, false} {
			c, ok, err := r.ConsumeApiToken(ctx, tx, tokenID)
			if err != nil || ok != want || (ok && c.LastUsedAt == nil) {
				t.Fatalf("списание %d: %+v %v %v", i+1, c, ok, err)
			}
		}

		var plateID string
		if err := tx.Get(ctx, &plateID, `SELECT plate_id FROM plate_sightings WHERE id = $1`, sightingID); err != nil {
			t.Fatal(err)
		}
		id := tokenID
		for _, v := range []models.NewPlateView{
			{SightingID: sightingID, PlateID: plateID, TokenID: &id, Consumed: true, UserAgent: strings.Repeat("ю", 400)},
			{SightingID: sightingID, PlateID: plateID, IsAdmin: true},
		} {
			if err := r.CreatePlateView(ctx, tx, v); err != nil {
				t.Fatalf("CreatePlateView: %v", err)
			}
		}

		page, err := r.ListPlateViews(ctx, tx, models.ListPlateViewsOpts{TokenID: tokenID, Limit: 10})
		if err != nil || page.Total != 1 || len(page.Items) != 1 {
			t.Fatalf("ListPlateViews(token): %+v %v", page, err)
		}
		v := page.Items[0]
		if !v.Consumed || v.TokenName == nil || *v.TokenName != "тест" || v.Sighting.Plate != "А123ВС 777" || len([]rune(v.UserAgent)) != models.MaxUserAgent {
			t.Errorf("запись журнала: %+v", v)
		}
		if page, err := r.ListPlateViews(ctx, tx, models.ListPlateViewsOpts{PlateID: plateID, Limit: 1}); err != nil || page.Total < 2 || page.NextOffset == nil {
			t.Errorf("ListPlateViews(plate): %+v %v", page, err)
		}

		list, err := r.ListApiTokens(ctx, tx, models.ListApiTokensOpts{Limit: 100})
		if err != nil || list.Total < 1 {
			t.Fatalf("ListApiTokens: %+v %v", list, err)
		}
		for _, it := range list.Items {
			if it.ID == tokenID && (it.ViewsCount != 1 || it.RequestsLeft != 0) {
				t.Errorf("токен в списке: %+v", it)
			}
		}

		if rv, err := r.RevokeApiToken(ctx, tx, tokenID); err != nil || !rv.Revoked {
			t.Errorf("RevokeApiToken: %+v %v", rv, err)
		}
		if _, err := r.RevokeApiToken(ctx, tx, "aaaaaaaa-0000-0000-0000-0000000fffff"); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("нет токена: %v", err)
		}
	})
}
