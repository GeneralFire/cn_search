package implementation

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func TestIncludeTestData(t *testing.T) {
	tests := map[string]struct {
		md   metadata.MD
		want bool
	}{
		"нет metadata":  {nil, false},
		"нет заголовка": {metadata.Pairs("x-other", "1"), false},
		"true":          {metadata.Pairs("include-test-data", "true"), true},
		"пустое":        {metadata.Pairs("include-test-data", ""), true},
		"false":         {metadata.Pairs("include-test-data", "false"), false},
		"0":             {metadata.Pairs("include-test-data", "0"), false},
	}
	for name, tt := range tests {
		ctx := context.Background()
		if tt.md != nil {
			ctx = metadata.NewIncomingContext(ctx, tt.md)
		}
		if got := includeTestData(ctx); got != tt.want {
			t.Errorf("%s: %v, want %v", name, got, tt.want)
		}
	}
}

func TestRequireAdmin(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("admin-token", "tok"))
	if err := requireAdmin(ctx, "tok"); err != nil {
		t.Errorf("верный токен: %v", err)
	}
	if err := requireAdmin(ctx, "tok2"); err == nil {
		t.Error("неверный токен принят")
	}
	if err := requireAdmin(ctx, ""); err == nil {
		t.Error("ADMIN_TOKEN не задан — должно быть выключено")
	}
}

func TestRateLimitKey(t *testing.T) {
	tcp := &net.TCPAddr{IP: net.ParseIP("10.0.0.9"), Port: 5555}
	withPeer := peer.NewContext(context.Background(), &peer.Peer{Addr: tcp})
	tests := map[string]struct {
		ctx  context.Context
		want string
	}{
		"peer":            {withPeer, "ip:10.0.0.9"},
		"xff — последний": {metadata.NewIncomingContext(withPeer, metadata.Pairs("x-forwarded-for", "1.1.1.1, 2.2.2.2")), "ip:2.2.2.2"},
		"ничего":          {context.Background(), "ip:"},
	}
	for name, tt := range tests {
		if got := RateLimitKey(tt.ctx); got != tt.want {
			t.Errorf("%s: %q, want %q", name, got, tt.want)
		}
	}

	// Пользователь — по токену (cookie или Bearer дают один ключ), а не по IP.
	cookie := RateLimitKey(metadata.NewIncomingContext(withPeer, metadata.Pairs("cookie", "cs_token=abc")))
	bearer := RateLimitKey(metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer abc")))
	other := RateLimitKey(metadata.NewIncomingContext(withPeer, metadata.Pairs("cookie", "cs_token=xyz")))
	if cookie != bearer || cookie == other || !strings.HasPrefix(cookie, "u:") || strings.Contains(cookie, "abc") {
		t.Errorf("ключи пользователя: %q %q %q", cookie, bearer, other)
	}
}

func TestWithPeerFromRequest(t *testing.T) {
	var got string
	h := WithPeerFromRequest(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = RateLimitKey(r.Context())
	}))
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = "192.0.2.7:4321"
	h.ServeHTTP(httptest.NewRecorder(), r)
	if got != "ip:192.0.2.7" {
		t.Errorf("ключ %q", got)
	}
}

func TestIsAdmin(t *testing.T) {
	s := &CarSearch{adminToken: "secret"}
	if s.IsAdmin(context.Background()) {
		t.Error("без заголовка — не админ")
	}
	if !s.IsAdmin(metadata.NewIncomingContext(context.Background(), metadata.Pairs("admin-token", "secret"))) {
		t.Error("верный admin-token — админ")
	}
}
