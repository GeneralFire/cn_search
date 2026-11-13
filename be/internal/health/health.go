// Package health — ручки liveness (/api/health) и readiness (/api/ready).
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"time"
)

// DefaultTimeout — сколько ждём ответа каждой зависимости при проверке готовности.
const DefaultTimeout = time.Second

// Response — тело ответа /api/health и /api/ready.
type Response struct {
	Status string            `json:"status"`           // ok | fail
	Checks map[string]string `json:"checks,omitempty"` // имя проверки → ok | текст ошибки
}

// Checker проверяет зависимости сервиса (каждую — с таймаутом).
type Checker struct {
	timeout time.Duration
	checks  map[string]pinger
}

// New — проверка готовности без зависимостей; timeout ≤ 0 — DefaultTimeout.
func New(timeout time.Duration) *Checker {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Checker{timeout: timeout, checks: map[string]pinger{}}
}

// With добавляет проверку по имени ("db", "s3" …): зависимость должна ответить на Ping.
func (c *Checker) With(name string, p pinger) *Checker {
	c.checks[name] = p
	return c
}

// Check опрашивает все зависимости параллельно; ok — все ответили без ошибки.
func (c *Checker) Check(ctx context.Context) (Response, bool) {
	names := make([]string, 0, len(c.checks))
	for name := range c.checks {
		names = append(names, name)
	}
	sort.Strings(names)

	errs := make([]error, len(names))
	done := make(chan struct{}, len(names))
	for i, name := range names {
		go func() {
			defer func() { done <- struct{}{} }()
			cctx, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()
			errs[i] = c.checks[name].Ping(cctx)
		}()
	}
	for range names {
		<-done
	}

	resp, ok := Response{Status: "ok", Checks: make(map[string]string, len(names))}, true
	for i, name := range names {
		if errs[i] != nil {
			resp.Checks[name], ok = errs[i].Error(), false
			continue
		}
		resp.Checks[name] = "ok"
	}
	if !ok {
		resp.Status = "fail"
	}
	return resp, ok
}

// Live — liveness: процесс жив и отвечает, зависимости не трогаем.
func Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, Response{Status: "ok"})
}

// Ready — readiness: 200, если все зависимости доступны, иначе 503 с причиной.
func (c *Checker) Ready(w http.ResponseWriter, r *http.Request) {
	resp, ok := c.Check(r.Context())
	code := http.StatusOK
	if !ok {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, resp)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
