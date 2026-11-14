package implementation

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// includeTestDataHeader — заголовок (gRPC metadata), с которым ответы включают тестовые данные
// (is_test_data = true). Clay кладёт HTTP-заголовки в incoming metadata в нижнем регистре.
const includeTestDataHeader = "include-test-data"

// includeTestData: заголовок передан с любым значением, кроме "false" и "0".
func includeTestData(ctx context.Context) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}
	for _, v := range md.Get(includeTestDataHeader) {
		if v != "false" && v != "0" {
			return true
		}
	}
	return false
}

// adminTokenHeader — заголовок (gRPC metadata) с паролем админ-панели; сравнивается с ADMIN_TOKEN бэка.
const adminTokenHeader = "admin-token"

// requireAdmin — доступ к админским методам: заголовок admin-token равен want. Нет заголовка, он неверный
// или want пуст (ADMIN_TOKEN не задан — админка выключена) — PermissionDenied.
// Сравнение — постоянного времени по sha256 (не выдаёт ни совпавший префикс, ни длину токена).
func requireAdmin(ctx context.Context, want string) error {
	denied := status.Error(codes.PermissionDenied, "нужен верный admin-token")
	if want == "" {
		return denied
	}
	md, _ := metadata.FromIncomingContext(ctx)
	got := md.Get(adminTokenHeader)
	if len(got) != 1 {
		return denied
	}
	g, w := sha256.Sum256([]byte(got[0])), sha256.Sum256([]byte(want))
	if subtle.ConstantTimeCompare(g[:], w[:]) != 1 {
		return denied
	}
	return nil
}

// userTokenCookie — cookie с токеном пользователя (ставит фронт); clay кладёт заголовок cookie в metadata.
const userTokenCookie = "cs_token"

// userToken — токен пользователя из metadata: сначала cookie cs_token, затем authorization: Bearer <токен>.
// "" — токена нет.
func userToken(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	for _, line := range md.Get("cookie") {
		cookies, err := http.ParseCookie(line)
		if err != nil {
			continue
		}
		for _, c := range cookies {
			if c.Name == userTokenCookie && c.Value != "" {
				return c.Value
			}
		}
	}
	for _, v := range md.Get("authorization") {
		if scheme, tok, ok := strings.Cut(strings.TrimSpace(v), " "); ok && strings.EqualFold(scheme, "Bearer") {
			if tok = strings.TrimSpace(tok); tok != "" {
				return tok
			}
		}
	}
	return ""
}

// currentUser — пользователь по токену из metadata (userToken). Нет токена или он неверный —
// Unauthenticated; сбой БД — Internal. Тестовые пользователи — только с заголовком include-test-data.
func (s *CarSearch) currentUser(ctx context.Context) (models.User, error) {
	if s.accounts == nil {
		return models.User{}, status.Error(codes.Unimplemented, "аккаунты не включены")
	}
	tok := userToken(ctx)
	if tok == "" {
		return models.User{}, status.Error(codes.Unauthenticated, "нужен токен: cookie cs_token или Authorization: Bearer")
	}
	u, err := s.accounts.UserByToken(ctx, tok, includeTestData(ctx))
	switch {
	case errors.Is(err, models.ErrNotFound):
		return models.User{}, status.Error(codes.Unauthenticated, "неверный токен")
	case err != nil:
		return models.User{}, status.Error(codes.Internal, "не удалось проверить токен")
	}
	return u, nil
}

// RateLimitKey — ключ клиента для ограничения частоты (ratelimit): пользователь по токену
// (cookie cs_token или Bearer, в том числе cst_), иначе IP. Токен хешируется — в памяти лимитера
// его нет. IP — последний адрес X-Forwarded-For (его дописывает наш прокси, левые значения
// подделывает клиент), иначе адрес peer (gRPC; для HTTP его кладёт WithPeerFromRequest).
func RateLimitKey(ctx context.Context) string {
	if tok := userToken(ctx); tok != "" {
		h := sha256.Sum256([]byte(tok))
		return "u:" + hex.EncodeToString(h[:16])
	}
	return "ip:" + clientIP(ctx)
}

// clientIP — IP клиента: последний непустой адрес X-Forwarded-For, иначе peer; "" — неизвестен.
func clientIP(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
		parts := strings.Split(xff[len(xff)-1], ",")
		for i := len(parts) - 1; i >= 0; i-- {
			if ip := strings.TrimSpace(parts[i]); ip != "" {
				return ip
			}
		}
	}
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		if host, _, err := net.SplitHostPort(p.Addr.String()); err == nil {
			return host
		}
		return p.Addr.String()
	}
	return ""
}

// WithPeerFromRequest — HTTP-middleware: кладёт r.RemoteAddr в контекст как gRPC peer, чтобы
// ручки clay (они вызывают тот же интерсептор) видели адрес клиента так же, как по gRPC.
func WithPeerFromRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if addr, err := net.ResolveTCPAddr("tcp", r.RemoteAddr); err == nil {
			r = r.WithContext(peer.NewContext(r.Context(), &peer.Peer{Addr: addr}))
		}
		next.ServeHTTP(w, r)
	})
}

// IsAdmin — запрос с верным admin-token (ограничение частоты на него не действует).
func (s *CarSearch) IsAdmin(ctx context.Context) bool {
	return requireAdmin(ctx, s.adminToken) == nil
}
