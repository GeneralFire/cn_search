package logging

import (
	"reflect"
	"strings"
	"testing"

	"io"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)


func newSlog(w io.Writer) *slog.Logger { return slog.New(slog.NewJSONHandler(w, nil)) }

// fakeStream — grpc.ServerTransportStream, запоминающий заголовки ответа.
type fakeStream struct{ md metadata.MD }

var _ grpc.ServerTransportStream = (*fakeStream)(nil)

func (f *fakeStream) Method() string { return "/x/y" }
func (f *fakeStream) SetHeader(md metadata.MD) error {
	f.md = metadata.Join(f.md, md)
	return nil
}
func (f *fakeStream) SendHeader(md metadata.MD) error { return f.SetHeader(md) }
func (f *fakeStream) SetTrailer(md metadata.MD) error { return nil }

// require — минимальные проверки без внешних зависимостей (testify в go.mod только косвенный).
var require = req{}

type req struct{}

func (req) NoError(t *testing.T, err error, _ ...any) {
	t.Helper()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
}

func (req) Equal(t *testing.T, want, got any, _ ...any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("ждали %#v, получили %#v", want, got)
	}
}

func (req) Len(t *testing.T, v any, n int, _ ...any) {
	t.Helper()
	if l := reflect.ValueOf(v).Len(); l != n {
		t.Fatalf("длина %d, ждали %d: %v", l, n, v)
	}
}

func has(v any, key string) bool {
	switch x := v.(type) {
	case string:
		return strings.Contains(x, key)
	case map[string]any:
		_, ok := x[key]
		return ok
	}
	return false
}

func (req) Contains(t *testing.T, v any, key string) {
	t.Helper()
	if !has(v, key) {
		t.Fatalf("%v не содержит %q", v, key)
	}
}

func (req) NotContains(t *testing.T, v any, key string) {
	t.Helper()
	if has(v, key) {
		t.Fatalf("%v содержит %q", v, key)
	}
}
