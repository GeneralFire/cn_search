package models

import (
	"encoding/json"
	"time"
)

// Действия журнала админов (admin_audit.action).
const (
	AuditHideSighting          = "hide_sighting"
	AuditShowSighting          = "show_sighting"
	AuditRestorePhoto          = "restore_photo"
	AuditSetClarificationState = "set_clarification_status"
	AuditResolveComplaint      = "resolve_complaint"
	AuditSetCameraStatus       = "set_camera_status"
	AuditCreateApiToken        = "create_api_token"
	AuditRevokeApiToken        = "revoke_api_token"
)

// Типы объектов журнала (admin_audit.target_type).
const (
	AuditTargetSighting      = "sighting"
	AuditTargetClarification = "clarification"
	AuditTargetComplaint     = "complaint"
	AuditTargetCameraRequest = "camera_request"
	AuditTargetApiToken      = "api_token"
)

// AuditActorAdminToken — актор, пока админ один (по заголовку admin-token).
const AuditActorAdminToken = "admin-token"

// AuditEntry — запись журнала действий админа (таблица admin_audit).
type AuditEntry struct {
	ID         string          `db:"id"`
	Action     string          `db:"action"`
	Actor      string          `db:"actor"`
	TargetType string          `db:"target_type"`
	TargetID   string          `db:"target_id"`
	Details    json.RawMessage `db:"details"` // {"from": …, "to": …}
	RequestID  string          `db:"request_id"`
	CreatedAt  time.Time       `db:"created_at"`
	TargetName string          `db:"target_name"` // имя объекта (токен); только в ListAudit
}

// ListAuditOpts — фильтры и страница журнала; пустые Action/TargetID — без фильтра.
type ListAuditOpts struct {
	Action   string
	TargetID string
	Offset   int
	Limit    int
}

// AuditPage — страница журнала.
type AuditPage struct {
	Items      []AuditEntry
	Total      int
	NextOffset *int
}
