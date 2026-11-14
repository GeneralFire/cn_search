package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/logging"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// audited выполняет админское действие do и пишет запись журнала в той же транзакции:
// details = {"from": состояние объекта до, "to": после} (null — объекта не было).
// Ошибка действия или записи откатывает всю транзакцию.
func audited(ctx context.Context, repo auditRepository, tx db.Executor, action, targetType, targetID string, do func() error) error {
	before, err := repo.AuditSnapshot(ctx, tx, targetType, targetID)
	if err != nil {
		return err
	}
	if err := do(); err != nil {
		return err
	}
	after, err := repo.AuditSnapshot(ctx, tx, targetType, targetID)
	if err != nil {
		return err
	}
	details, err := json.Marshal(map[string]json.RawMessage{"from": orNull(before), "to": orNull(after)})
	if err != nil {
		return fmt.Errorf("audit details: %w", err)
	}
	return repo.WriteAudit(ctx, tx, models.AuditEntry{
		Action: action, Actor: models.AuditActorAdminToken, TargetType: targetType, TargetID: targetID,
		Details: details, RequestID: logging.RequestID(ctx),
	})
}

func orNull(b json.RawMessage) json.RawMessage {
	if len(b) == 0 {
		return json.RawMessage("null")
	}
	return b
}

// ListAudit — страница журнала админов; count и страница читаются в одном снимке.
func (u *Admin) ListAudit(ctx context.Context, o models.ListAuditOpts) (page models.AuditPage, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		page, err = u.repo.ListAudit(ctx, tx, o)
		return err
	})
	return page, err
}
