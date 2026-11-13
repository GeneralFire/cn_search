package models

import (
	"errors"
	"time"
)

// Ошибки проверки API-токена в rpc Get / TokenInfo.
var (
	ErrTokenInvalid   = errors.New("api token: invalid")   // подпись не сошлась или токена нет в БД
	ErrTokenRevoked   = errors.New("api token: revoked")   // токен отозван в админке
	ErrTokenExhausted = errors.New("api token: exhausted") // запросы токена кончились
)

// ApiToken — API-токен (таблица api_tokens); самого токена и подписи здесь нет.
type ApiToken struct {
	ID            string     `db:"id"`
	Name          string     `db:"name"`
	RequestsTotal int        `db:"requests_total"`
	RequestsLeft  int        `db:"requests_left"`
	Revoked       bool       `db:"revoked"`
	CreatedAt     time.Time  `db:"created_at"`
	LastUsedAt    *time.Time `db:"last_used_at"` // nil — ещё не использовался
	ViewsCount    int        `db:"views_count"`  // открытий страниц с токеном (только ListApiTokens)
}

// CreateApiTokenOpts — новый токен; ID и SignatureHash считает usecase (apitoken.Signer).
type CreateApiTokenOpts struct {
	ID            string
	Name          string
	Requests      int
	SignatureHash string
}

// ListApiTokensOpts — страница токенов для админки.
type ListApiTokensOpts struct {
	Offset          int
	Limit           int
	IncludeTestData bool
}

// ApiTokensPage — страница токенов.
type ApiTokensPage struct {
	Items      []ApiToken
	Total      int
	NextOffset *int
}

// PlateView — открытие страницы номера (таблица plate_views) вместе с фиксацией и именем токена.
type PlateView struct {
	ID        string    `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	TokenID   *string   `db:"token_id"`
	TokenName *string   `db:"token_name"`
	Consumed  bool      `db:"consumed"`
	IsAdmin   bool      `db:"is_admin"`
	UserAgent string    `db:"user_agent"`

	Sighting PlateSighting `db:"sighting"`
}

// NewPlateView — запись журнала.
type NewPlateView struct {
	SightingID string
	PlateID    string
	TokenID    *string
	Consumed   bool
	IsAdmin    bool
	UserID     *string // пользователь (токен "cs_…"); nil — не вошёл
	UserAgent  string  // обрезается до MaxUserAgent символов
}

// MaxUserAgent — сколько символов user-agent хранить (CHECK в plate_views).
const MaxUserAgent = 300

// ListPlateViewsOpts — страница журнала; пустые фильтры — без фильтра.
type ListPlateViewsOpts struct {
	TokenID string
	PlateID string
	Offset  int
	Limit   int
}

// PlateViewsPage — страница журнала.
type PlateViewsPage struct {
	Items      []PlateView
	Total      int
	NextOffset *int
}

// OpenPlateOpts — открытие страницы номера (rpc Get): все фиксации номера + запись в журнал.
type OpenPlateOpts struct {
	SamePlateOpts
	Token     string  // API-токен "cst_…" из cookie / authorization; пусто — без токена
	IsAdmin   bool    // запрос с верным admin-token: квота не списывается
	UserID    *string // пользователь (токен "cs_…") — в журнал; nil — не вошёл
	UserAgent string
}
