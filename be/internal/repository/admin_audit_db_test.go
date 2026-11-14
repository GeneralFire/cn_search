package repository

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// WriteAudit → ListAudit: фильтры по действию и объекту, порядок от новых к старым, имя токена,
// снимки объектов до/после действия.
func TestAdminAuditSQL(t *testing.T) {
	const sightingID = "00000000-0000-0000-0000-000000000001" // тестовая фиксация
	const tokenID = "20000000-0000-0000-0000-0000000000a1"
	r := New()
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		if _, err := tx.Exec(ctx, `INSERT INTO api_tokens (id, name, requests_total, requests_left, signature_hash)
			VALUES ($1, 'партнёр', 5, 5, 'h-audit')`, tokenID); err != nil {
			t.Fatal(err)
		}
		before, err := r.AuditSnapshot(ctx, tx, models.AuditTargetSighting, sightingID)
		if err != nil || string(before) != `{"hidden": false, "photo_hidden": false}` {
			t.Fatalf("snapshot до: %s %v", before, err)
		}
		if _, err := r.HideSighting(ctx, tx, sightingID, true); err != nil {
			t.Fatal(err)
		}
		after, _ := r.AuditSnapshot(ctx, tx, models.AuditTargetSighting, sightingID)
		details, _ := json.Marshal(map[string]json.RawMessage{"from": before, "to": after})
		for _, e := range []models.AuditEntry{
			{Action: models.AuditHideSighting, Actor: models.AuditActorAdminToken, TargetType: models.AuditTargetSighting, TargetID: sightingID, Details: details, RequestID: "req-a"},
			{Action: models.AuditRevokeApiToken, Actor: models.AuditActorAdminToken, TargetType: models.AuditTargetApiToken, TargetID: tokenID, RequestID: "req-b"},
		} {
			if err := r.WriteAudit(ctx, tx, e); err != nil {
				t.Fatal(err)
			}
		}
		if s, err := r.AuditSnapshot(ctx, tx, models.AuditTargetSighting, "00000000-0000-0000-0000-0000000fffff"); err != nil || s != nil {
			t.Errorf("нет объекта: %s %v", s, err)
		}
		if _, err := r.AuditSnapshot(ctx, tx, "bogus", "x"); err == nil {
			t.Error("неизвестный тип объекта должен быть ошибкой")
		}

		page, err := r.ListAudit(ctx, tx, models.ListAuditOpts{Limit: 10})
		if err != nil || page.Total != 2 || len(page.Items) != 2 {
			t.Fatalf("ListAudit: %+v %v", page, err)
		}
		if page.Items[0].RequestID != "req-b" || page.Items[0].TargetName != "партнёр" || string(page.Items[0].Details) != `{}` {
			t.Errorf("первая (новая) запись: %+v", page.Items[0])
		}
		h := page.Items[1]
		if h.Action != models.AuditHideSighting || h.RequestID != "req-a" || h.Actor != "admin-token" {
			t.Errorf("запись hide: %+v", h)
		}
		var d struct{ From, To map[string]bool }
		if err := json.Unmarshal(h.Details, &d); err != nil || d.From["hidden"] || !d.To["hidden"] {
			t.Errorf("details = %s (%v)", h.Details, err)
		}

		byAction, err := r.ListAudit(ctx, tx, models.ListAuditOpts{Action: models.AuditHideSighting, Limit: 10})
		if err != nil || byAction.Total != 1 || byAction.Items[0].TargetID != sightingID {
			t.Errorf("фильтр action: %+v %v", byAction, err)
		}
		byTarget, err := r.ListAudit(ctx, tx, models.ListAuditOpts{TargetID: tokenID, Limit: 10})
		if err != nil || byTarget.Total != 1 || byTarget.Items[0].Action != models.AuditRevokeApiToken {
			t.Errorf("фильтр target_id: %+v %v", byTarget, err)
		}
		p1, err := r.ListAudit(ctx, tx, models.ListAuditOpts{Limit: 1})
		if err != nil || p1.NextOffset == nil || *p1.NextOffset != 1 {
			t.Errorf("страница: %+v %v", p1, err)
		}
		if empty, err := r.ListAudit(ctx, tx, models.ListAuditOpts{Offset: 5, Limit: 10}); err != nil || len(empty.Items) != 0 || empty.Total != 2 {
			t.Errorf("за пределами: %+v %v", empty, err)
		}
	})
}
