package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"

	"github.com/GeneralFire/cn_search/be/internal/implementation"
	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/repository"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// httpStub — заглушка репозитория: здесь проверяются ручки clay, JSON и ошибки, а не БД.
// Search: пустая строка — 3 фиксации постранично, "Х000ХХ 000" — ничего. ListSamePlate: id 1 — две фиксации.
type httpStub struct{}

var stubItems = repository.Fixed[:3]

func (httpStub) Search(_ context.Context, q repository.Query) (repository.Page, error) {
	p := repository.Page{Items: []models.PlateSighting{}, Offset: q.Offset, Limit: q.Limit}
	if q.Plate != "" {
		return p, nil
	}
	p.Total = len(stubItems)
	p.Items = stubItems[min(q.Offset, p.Total):min(q.Offset+q.Limit, p.Total)]
	if next := q.Offset + len(p.Items); next < p.Total {
		p.NextOffset = &next
	}
	return p, nil
}

func (httpStub) ListSamePlate(_ context.Context, id int) ([]models.PlateSighting, error) {
	if id != 1 {
		return nil, repository.ErrNotFound
	}
	return []models.PlateSighting{repository.Fixed[3], repository.Fixed[0]}, nil
}

// Проверяем сгенерированные clay ручки вместе с нашей настройкой JSON и ошибок.
func newHTTP(t *testing.T) *httptest.Server {
	t.Helper()
	configureHTTPRuntime()
	desc := pb.NewCarSearchServiceServiceDesc(implementation.NewCarSearch(httpStub{}, "/s3/cars"))
	r := chi.NewRouter()
	desc.RegisterHTTP(r)
	root := chi.NewRouter()
	root.Mount(apiPrefix, r)
	srv := httptest.NewServer(logRequests(slog.New(slog.NewTextHandler(io.Discard, nil)), root))
	t.Cleanup(srv.Close)
	return srv
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

func TestHTTPEmptyResultHasTotal(t *testing.T) {
	srv := newHTTP(t)
	code, out := call(t, srv, "Search", `{"plate":"Х000ХХ 000"}`)
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if out["total"] != float64(0) {
		t.Errorf("total = %v, want 0 (поле должно быть всегда)", out["total"])
	}
	if items, ok := out["items"].([]any); !ok || len(items) != 0 {
		t.Errorf("items = %v, want []", out["items"])
	}
	if _, has := out["nextOffset"]; has {
		t.Errorf("nextOffset должен отсутствовать: %v", out)
	}
}

func TestHTTPSearchAndPaging(t *testing.T) {
	srv := newHTTP(t)
	_, out := call(t, srv, "Search", `{"limit":2}`)
	if out["total"] != float64(3) || len(out["items"].([]any)) != 2 || out["nextOffset"] != float64(2) {
		t.Fatalf("page 1: %v", out)
	}
	first := out["items"].([]any)[0].(map[string]any)
	if first["photoUrl"] == nil || first["plate"] == nil || first["createdAt"] == nil {
		t.Errorf("camelCase поля: %v", first)
	}
}

func TestHTTPGetReturnsAllSightings(t *testing.T) {
	srv := newHTTP(t)
	code, out := call(t, srv, "Get", `{"id":1}`)
	items, _ := out["items"].([]any)
	if code != http.StatusOK || len(items) != 2 {
		t.Fatalf("Get: %d %v", code, out)
	}
	if first := items[0].(map[string]any); first["id"] != float64(4) || first["plate"] != "А123ВС 777" {
		t.Errorf("первая фиксация: %v", first)
	}
}

func TestHTTPErrors(t *testing.T) {
	srv := newHTTP(t)
	tests := []struct {
		method, body string
		status       int
		code         string
	}{
		{"Search", `{"limit":101}`, http.StatusBadRequest, "InvalidArgument"},
		{"Search", `{не json`, http.StatusBadRequest, "InvalidArgument"},
		{"Get", `{"id":100500}`, http.StatusNotFound, "NotFound"},
		{"Get", `{"id":0}`, http.StatusBadRequest, "InvalidArgument"},
	}
	for _, tt := range tests {
		status, out := call(t, srv, tt.method, tt.body)
		if status != tt.status || out["code"] != tt.code {
			t.Errorf("%s %s: %d %v, want %d %s", tt.method, tt.body, status, out, tt.status, tt.code)
		}
		if msg, _ := out["message"].(string); msg == "" || strings.Contains(msg, "rpc error") {
			t.Errorf("%s %s: message = %q", tt.method, tt.body, msg)
		}
	}
}
