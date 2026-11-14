package usecase

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/logging"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// auditRec подменяет WriteAudit и AuditSnapshot репозитория: снимки отдаются по очереди
// (до действия, после), записи журнала копятся.
type auditRec struct {
	snaps   []string
	next    int
	entries []models.AuditEntry
}

func (a *auditRec) snapshot(_ context.Context, _ db.Executor, _, _ string) (json.RawMessage, error) {
	if a.next >= len(a.snaps) {
		return nil, nil
	}
	a.next++
	return json.RawMessage(a.snaps[a.next-1]), nil
}

func (a *auditRec) write(_ context.Context, _ db.Executor, e models.AuditEntry) error {
	a.entries = append(a.entries, e)
	return nil
}

// ctxReq — контекст запроса с request_id.
func ctxReq() context.Context { return logging.WithRequestID(context.Background(), "req-1") }

// check — одна запись журнала с нужным действием, объектом, request_id и деталями from/to.
func (a *auditRec) check(t *testing.T, action, targetType, id, from, to string) {
	t.Helper()
	if len(a.entries) != 1 {
		t.Fatalf("записей журнала: %d, нужна 1", len(a.entries))
	}
	e := a.entries[0]
	if e.Action != action || e.TargetType != targetType || e.TargetID != id || e.Actor != models.AuditActorAdminToken || e.RequestID != "req-1" {
		t.Errorf("запись: %+v", e)
	}
	want := `{"from":` + from + `,"to":` + to + `}`
	if string(e.Details) != want {
		t.Errorf("details = %s, want %s", e.Details, want)
	}
}

func TestAuditedFailedActionWritesNothing(t *testing.T) {
	rec := &auditRec{}
	repo := &stubAudit{rec: rec}
	err := audited(ctxReq(), repo, nil, "x", "sighting", "s1", func() error { return models.ErrNotFound })
	if err == nil || len(rec.entries) != 0 {
		t.Fatalf("err = %v, записей %d", err, len(rec.entries))
	}
}

// stubAudit — минимальный auditRepository поверх auditRec.
type stubAudit struct{ rec *auditRec }

func (s *stubAudit) WriteAudit(ctx context.Context, tx db.Executor, e models.AuditEntry) error {
	return s.rec.write(ctx, tx, e)
}

func (s *stubAudit) AuditSnapshot(ctx context.Context, tx db.Executor, tt, id string) (json.RawMessage, error) {
	return s.rec.snapshot(ctx, tx, tt, id)
}
