package implementation

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
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
