package main

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/gojuno/minimock/v3"

	"github.com/GeneralFire/cn_search/be/internal/implementation"
	"github.com/GeneralFire/cn_search/be/internal/implementation/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// Проверяем сгенерированные clay ручки с его стандартными маршалером и форматом ошибок.
func newHTTP(t *testing.T) (*httptest.Server, *mocks.PlateSightingsMock) {
	t.Helper()
	uc := mocks.NewPlateSightingsMock(t)
	desc := pb.NewCarSearchServiceServiceDesc(implementation.NewCarSearch(uc, "/s3/cars"))
	r := chi.NewRouter()
	desc.RegisterHTTP(r)
	root := chi.NewRouter()
	root.Mount(apiPrefix, r)
	srv := httptest.NewServer(logRequests(slog.New(slog.NewTextHandler(io.Discard, nil)), root))
	t.Cleanup(srv.Close)
	return srv, uc
}

func call(t *testing.T, srv *httptest.Server, method, body string) (int, map[string]any) {
	t.Helper()
	res, err := http.Post(srv.URL+apiPrefix+"/carsearch.CarSearchService/"+method, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("%s: невалидный JSON: %v", method, err)
	}
	return res.StatusCode, out
}

var sighting = models.PlateSighting{
	ID: "00000000-0000-0000-0000-000000000001", PhotoKey: "car-3.png", Address: "Москва, Пресненская наб., 10", Plate: "А123ВС 777",
	CreatedAt: time.Date(2026, 11, 2, 9, 15, 0, 0, time.UTC),
}

func TestHTTPSearch(t *testing.T) {
	srv, uc := newHTTP(t)
	next := 2
	uc.SearchMock.Expect(minimock.AnyContext, models.SearchOpts{Limit: 2}).
		Return(models.SearchPage{Items: []models.PlateSighting{sighting}, Total: 3, NextOffset: &next}, nil)

	code, out := call(t, srv, "Search", `{"limit":2}`)
	if code != http.StatusOK || out["total"] != float64(3) || out["nextOffset"] != float64(2) {
		t.Fatalf("Search: %d %v", code, out)
	}
	first := out["items"].([]any)[0].(map[string]any)
	if first["photoUrl"] != "/s3/cars/car-3.png" || first["createdAt"] != "2026-11-02T09:15:00Z" {
		t.Errorf("camelCase поля и время RFC 3339: %v", first)
	}
}

func TestHTTPEmptySearchOmitsZeroFields(t *testing.T) {
	srv, uc := newHTTP(t)
	uc.SearchMock.Return(models.SearchPage{Items: []models.PlateSighting{}}, nil)

	code, out := call(t, srv, "Search", `{"plate":"Х000ХХ 000"}`)
	if code != http.StatusOK || len(out) != 0 {
		t.Errorf("пустая выдача: %d %v, want 200 {} (нулевые поля clay не отдаёт)", code, out)
	}
}

// Заголовок include-test-data доходит через clay (HTTP-заголовки → gRPC metadata) до опций usecase.
func TestHTTPIncludeTestDataHeader(t *testing.T) {
	srv, uc := newHTTP(t)
	uc.SearchMock.Expect(minimock.AnyContext, models.SearchOpts{Limit: 35, IncludeTestData: true}).
		Return(models.SearchPage{}, nil)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+apiPrefix+"/carsearch.CarSearchService/Search", strings.NewReader(`{}`))
	req.Header.Set("Include-Test-Data", "true")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestHTTPGet(t *testing.T) {
	srv, uc := newHTTP(t)
	second := sighting
	second.ID = "00000000-0000-0000-0000-000000000004"
	uc.ListSamePlateMock.Expect(minimock.AnyContext, models.SamePlateOpts{ID: "00000000-0000-0000-0000-000000000001"}).Return([]models.PlateSighting{second, sighting}, nil)

	code, out := call(t, srv, "Get", `{"id":"00000000-0000-0000-0000-000000000001"}`)
	items, _ := out["items"].([]any)
	if code != http.StatusOK || len(items) != 2 || items[0].(map[string]any)["id"] != "00000000-0000-0000-0000-000000000004" {
		t.Fatalf("Get: %d %v", code, out)
	}
}

func TestHTTPErrors(t *testing.T) {
	srv, uc := newHTTP(t)
	uc.ListSamePlateMock.When(minimock.AnyContext, models.SamePlateOpts{ID: "00000000-0000-0000-0000-0000000a0500"}).Then(nil, models.ErrNotFound)
	uc.ListSamePlateMock.When(minimock.AnyContext, models.SamePlateOpts{ID: "00000000-0000-0000-0000-000000000004"}).Then(nil, errors.New("db down"))

	tests := []struct {
		method, body string
		status       int
		contains     string
	}{
		{"Search", `{"limit":101}`, http.StatusBadRequest, "code = InvalidArgument"},
		{"Search", `{не json`, http.StatusInternalServerError, "couldn't parse request"},
		{"Search", `{"unknown":1}`, http.StatusInternalServerError, "unknown field"},
		{"Get", `{"id":"1"}`, http.StatusBadRequest, "code = InvalidArgument"},
		{"Get", `{"id":"00000000-0000-0000-0000-0000000a0500"}`, http.StatusNotFound, "code = NotFound"},
		{"Get", `{"id":"00000000-0000-0000-0000-000000000004"}`, http.StatusInternalServerError, "code = Internal"},
	}
	for _, tt := range tests {
		status, out := call(t, srv, tt.method, tt.body)
		if msg, _ := out["error"].(string); status != tt.status || !strings.Contains(msg, tt.contains) {
			t.Errorf("%s %s: %d %v, want %d и error с %q", tt.method, tt.body, status, out, tt.status, tt.contains)
		}
	}
}
