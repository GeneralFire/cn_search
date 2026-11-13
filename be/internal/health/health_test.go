package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/GeneralFire/cn_search/be/internal/health"
	"github.com/GeneralFire/cn_search/be/internal/health/mocks"
)

func ready(t *testing.T, c *health.Checker) (int, health.Response) {
	t.Helper()
	rec := httptest.NewRecorder()
	c.Ready(rec, httptest.NewRequest(http.MethodGet, "/api/ready", nil))
	var resp health.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("тело ответа: %v", err)
	}
	return rec.Code, resp
}

func TestReadyOK(t *testing.T) {
	mc := minimock.NewController(t)
	db := mocks.NewPingerMock(mc).PingMock.Return(nil)
	s3 := mocks.NewPingerMock(mc).PingMock.Return(nil)

	code, resp := ready(t, health.New(0).With("db", db).With("s3", s3))
	if code != http.StatusOK || resp.Status != "ok" || resp.Checks["db"] != "ok" || resp.Checks["s3"] != "ok" {
		t.Fatalf("got %d %+v", code, resp)
	}
}

func TestReadyFail(t *testing.T) {
	mc := minimock.NewController(t)
	db := mocks.NewPingerMock(mc).PingMock.Return(errors.New("connection refused"))
	s3 := mocks.NewPingerMock(mc).PingMock.Return(nil)

	code, resp := ready(t, health.New(0).With("db", db).With("s3", s3))
	if code != http.StatusServiceUnavailable || resp.Status != "fail" ||
		resp.Checks["db"] != "connection refused" || resp.Checks["s3"] != "ok" {
		t.Fatalf("got %d %+v", code, resp)
	}
}

// Зависшая зависимость не держит ответ дольше таймаута.
func TestReadyTimeout(t *testing.T) {
	mc := minimock.NewController(t)
	db := mocks.NewPingerMock(mc).PingMock.Set(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	start := time.Now()
	code, resp := ready(t, health.New(20*time.Millisecond).With("db", db))
	if code != http.StatusServiceUnavailable || resp.Checks["db"] != context.DeadlineExceeded.Error() {
		t.Fatalf("got %d %+v", code, resp)
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("ответ занял %v", d)
	}
}

func TestLive(t *testing.T) {
	rec := httptest.NewRecorder()
	health.Live(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}
