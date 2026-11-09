package implementation

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
