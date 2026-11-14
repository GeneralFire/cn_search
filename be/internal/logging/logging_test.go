package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func lines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if l == "" {
			continue
		}
		m := map[string]any{}
		require.NoError(t, json.Unmarshal([]byte(l), &m))
		out = append(out, m)
	}
	return out
}

func TestNewRequestID(t *testing.T) {
	require.Equal(t, "test-123", NewRequestID("test-123"))
	require.Equal(t, "a_B-9", NewRequestID("a_B-9"))
	for _, bad := range []string{"", "a b", "a\nb", "a;b", "é", strings.Repeat("x", 65)} {
		got := NewRequestID(bad)
		_, err := uuid.Parse(got)
		require.NoError(t, err, "для %q ждали UUID, получили %q", bad, got)
	}
	require.Len(t, NewRequestID(strings.Repeat("x", 64)), 64)
}

func call(ctx context.Context, i grpc.UnaryServerInterceptor, h grpc.UnaryHandler) error {
	_, err := i(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/carsearch.CarSearchService/Search"}, h)
	return err
}

func TestInterceptor_UsesIncomingID_AndLogsFinalLine(t *testing.T) {
	var buf bytes.Buffer
	i := Interceptor(newSlog(&buf), Options{
		UserID: func(context.Context) string { return "u1" },
		IP:     func(context.Context) string { return "1.2.3.4" },
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-request-id", "test-123"))
	err := call(ctx, i, func(ctx context.Context, _ any) (any, error) {
		From(ctx).Info("внутри")
		require.Equal(t, "test-123", RequestID(ctx))
		return nil, nil
	})
	require.NoError(t, err)

	ls := lines(t, &buf)
	require.Len(t, ls, 2)
	for _, l := range ls {
		require.Equal(t, "test-123", l["request_id"])
		require.Equal(t, "/carsearch.CarSearchService/Search", l["method"])
		require.Equal(t, "u1", l["user_id"])
	}
	last := ls[1]
	require.Equal(t, "OK", last["code"])
	require.Equal(t, "1.2.3.4", last["ip"])
	require.Contains(t, last, "duration_ms")
}

func TestInterceptor_GeneratesID_AndCodeLevel(t *testing.T) {
	var buf bytes.Buffer
	i := Interceptor(newSlog(&buf), Options{})
	err := call(context.Background(), i, func(context.Context, any) (any, error) {
		return nil, status.Error(codes.Internal, "boom")
	})
	require.Equal(t, codes.Internal, status.Code(err))
	ls := lines(t, &buf)
	require.Len(t, ls, 1)
	_, perr := uuid.Parse(ls[0]["request_id"].(string))
	require.NoError(t, perr)
	require.Equal(t, "Internal", ls[0]["code"])
	require.Equal(t, "ERROR", ls[0]["level"])
	require.NotContains(t, ls[0], "user_id")
}

func TestHTTP_HeaderRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	i := Interceptor(newSlog(&buf), Options{})
	h := HTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// как clay: заголовки запроса → metadata
		ctx := metadata.NewIncomingContext(r.Context(), metadata.Pairs("x-request-id", r.Header.Get(Header)))
		_ = call(ctx, i, func(context.Context, any) (any, error) { return nil, nil })
	}))

	for _, tc := range []struct{ in, want string }{{"test-123", "test-123"}, {"", ""}, {"bad id!", ""}} {
		buf.Reset()
		req := httptest.NewRequest(http.MethodPost, "/api/x", nil)
		if tc.in != "" {
			req.Header.Set(Header, tc.in)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		got := rec.Header().Values(Header)
		require.Len(t, got, 1, "заголовок ответа должен быть один")
		if tc.want != "" {
			require.Equal(t, tc.want, got[0])
		} else {
			_, err := uuid.Parse(got[0])
			require.NoError(t, err)
		}
		require.Equal(t, got[0], lines(t, &buf)[0]["request_id"])
	}
}

func TestInterceptor_GRPCResponseHeader(t *testing.T) {
	var buf bytes.Buffer
	i := Interceptor(newSlog(&buf), Options{})
	stream := &fakeStream{}
	ctx := grpc.NewContextWithServerTransportStream(
		metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-request-id", "test-123")), stream)
	require.NoError(t, call(ctx, i, func(context.Context, any) (any, error) { return nil, nil }))
	require.Equal(t, []string{"test-123"}, stream.md.Get("x-request-id"))
}

func TestChain_Order(t *testing.T) {
	var order []string
	mk := func(n string) grpc.UnaryServerInterceptor {
		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
			order = append(order, n)
			return h(ctx, req)
		}
	}
	require.NoError(t, call(context.Background(), Chain(mk("a"), mk("b")), func(context.Context, any) (any, error) {
		order = append(order, "h")
		return nil, nil
	}))
	require.Equal(t, []string{"a", "b", "h"}, order)
}

// Секреты из заголовков не попадают в логи: интерсептор их не читает, а в поля идут только id/метод/код/ip.
func TestNoSecretsInLogs(t *testing.T) {
	secrets := []string{"cst_SECRETTOKEN.sig", "cookie-secret-value", "admin-secret-value", "bearer-secret-value"}
	var buf bytes.Buffer
	i := Interceptor(newSlog(&buf), Options{
		UserID: func(context.Context) string { return "u1" },
		IP:     func(context.Context) string { return "1.2.3.4" },
	})
	md := metadata.Pairs(
		"cookie", "cs_token=cookie-secret-value; other=1",
		"authorization", "Bearer bearer-secret-value",
		"admin-token", "admin-secret-value",
		"x-request-id", "test-123",
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_ = call(ctx, i, func(ctx context.Context, _ any) (any, error) {
		From(ctx).Info("work")
		return nil, status.Error(codes.Unauthenticated, "неверный токен")
	})
	for _, s := range secrets[1:] {
		require.NotContains(t, buf.String(), s)
	}
	require.NotContains(t, buf.String(), "cs_token")
	require.NotContains(t, buf.String(), "cst_")
	require.NotContains(t, strings.ToLower(buf.String()), "authorization")
}
