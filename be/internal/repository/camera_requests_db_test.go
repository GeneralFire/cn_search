package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

func TestCameraRequestsSQL(t *testing.T) {
	const testID = "00000000-0000-0000-0000-0000000ca001" // тестовая заявка (Москва, Тверская, new, 12 голосов)
	r := New()
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		if _, err := r.CreateCameraRequest(ctx, tx, models.SubmitCameraRequestOpts{Lat: 1, Lon: 1, Reason: strings.Repeat("ж", 1001)}); err == nil {
			t.Error("CHECK на 1000 символов не сработал")
		}
	})
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		if _, err := r.CreateCameraRequest(ctx, tx, models.SubmitCameraRequestOpts{Lat: 91, Lon: 1, Reason: "x"}); err == nil {
			t.Error("CHECK на широту не сработал")
		}
	})
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		c, err := r.CreateCameraRequest(ctx, tx, models.SubmitCameraRequestOpts{Lat: 52.52, Lon: 13.40, Address: "Alexanderplatz", Reason: "Kreuzung", Contact: "@me"})
		if err != nil || c.ID == "" || c.Status != models.CameraRequestNew || c.Votes != 0 || c.Contact != "@me" || c.CreatedAt.IsZero() {
			t.Fatalf("CreateCameraRequest: %+v %v", c, err)
		}

		// Без тестовых — только новая.
		prod, err := r.ListCameraRequests(ctx, tx, models.ListCameraRequestsOpts{Limit: 10})
		if err != nil || prod.Total != 1 || prod.Items[0].ID != c.ID {
			t.Fatalf("без тестовых: %+v %v", prod, err)
		}
		// Все: 5 тестовых + новая; новая первая (самая свежая), страница по 4.
		all, err := r.ListCameraRequests(ctx, tx, models.ListCameraRequestsOpts{Limit: 4, IncludeTestData: true})
		if err != nil || all.Total != 6 || len(all.Items) != 4 || all.Items[0].ID != c.ID || all.NextOffset == nil || *all.NextOffset != 4 {
			t.Fatalf("все: %+v %v", all, err)
		}
		// Рамка Москвы: 2 тестовые.
		moscow, err := r.ListCameraRequests(ctx, tx, models.ListCameraRequestsOpts{
			BBox: &models.GeoBox{MinLat: 55, MinLon: 37, MaxLat: 56.5, MaxLon: 38.5}, Limit: 10, IncludeTestData: true,
		})
		if err != nil || moscow.Total != 2 {
			t.Errorf("рамка Москвы: %+v %v", moscow, err)
		}
		// Статус + рамка Киева: одна новая.
		kyiv, err := r.ListCameraRequests(ctx, tx, models.ListCameraRequestsOpts{
			BBox: &models.GeoBox{MinLat: 50, MinLon: 30, MaxLat: 51, MaxLon: 31}, Status: models.CameraRequestNew, Limit: 10, IncludeTestData: true,
		})
		if err != nil || kyiv.Total != 1 || kyiv.Items[0].Address == "" {
			t.Errorf("Киев, new: %+v %v", kyiv, err)
		}

		// Голос: тестовая без флага — как будто её нет.
		if _, err := r.VoteCameraRequest(ctx, tx, models.VoteCameraRequestOpts{ID: testID}); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("тестовая без флага: %v", err)
		}
		v, err := r.VoteCameraRequest(ctx, tx, models.VoteCameraRequestOpts{ID: testID, IncludeTestData: true})
		if err != nil || v.Votes != 13 {
			t.Errorf("голос: %+v %v", v, err)
		}

		s, err := r.SetCameraRequestStatus(ctx, tx, testID, models.CameraRequestInstalled)
		if err != nil || s.Status != models.CameraRequestInstalled || s.Votes != 13 {
			t.Errorf("статус: %+v %v", s, err)
		}
		if _, err := r.SetCameraRequestStatus(ctx, tx, "00000000-0000-0000-0000-0000000fffff", models.CameraRequestRejected); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("нет заявки: %v", err)
		}
	})
}
