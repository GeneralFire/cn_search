package implementation

import (
	"context"

	"google.golang.org/grpc/metadata"
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
