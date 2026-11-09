package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"google.golang.org/grpc/codes"

	"github.com/GeneralFire/cn_search/be/internal/cars"
	"github.com/GeneralFire/cn_search/be/internal/service"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch/v1"
)

// Проверяем сгенерированные clay ручки вместе с нашей настройкой JSON и ошибок.
func newHTTP(t *testing.T) *httptest.Server {
	t.Helper()
	configureHTTPRuntime()
	desc := pb.NewCarSearchServiceServiceDesc(service.NewCarSearch(cars.NewMemoryStore(cars.Fixed), "/s3/cars"))
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
	res, err := http.Post(srv.URL+apiPrefix+"/carsearch.v1.CarSearchService/"+method, "application/json", strings.NewReader(body))
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
	code, out := call(t, srv, "SearchCars", `{"plate":"Х000ХХ"}`)
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
	_, out := call(t, srv, "SearchCars", `{"limit":2}`)
	if out["total"] != float64(3) || len(out["items"].([]any)) != 2 || out["nextOffset"] != float64(2) {
		t.Fatalf("page 1: %v", out)
	}
	first := out["items"].([]any)[0].(map[string]any)
	if first["photoUrl"] == nil || first["plate"] == nil {
		t.Errorf("camelCase поля: %v", first)
	}
}

func TestHTTPErrors(t *testing.T) {
	srv := newHTTP(t)
	tests := []struct {
		method, body string
		status       int
		code         string
	}{
		{"SearchCars", `{"limit":101}`, http.StatusBadRequest, "invalid_argument"},
		{"SearchCars", `{не json`, http.StatusBadRequest, "invalid_argument"},
		{"GetCar", `{"id":100500}`, http.StatusNotFound, "not_found"},
		{"GetCar", `{"id":0}`, http.StatusBadRequest, "invalid_argument"},
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

func TestSnakeCode(t *testing.T) {
	for c, want := range map[codes.Code]string{
		codes.InvalidArgument: "invalid_argument",
		codes.NotFound:        "not_found",
		codes.Internal:        "internal",
	} {
		if got := snakeCode(c); got != want {
			t.Errorf("snakeCode(%v) = %q, want %q", c, got, want)
		}
	}
}
