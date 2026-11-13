package implementation

import (
	"context"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/apitoken"
	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// apiTokenFromMD — API-токен запроса "cst_…" из cookie cs_token или authorization: Bearer (хелпер
// userToken из metadata.go — там же пользовательские токены "cs_…", они сюда не попадают). Нет — "".
func apiTokenFromMD(ctx context.Context) string {
	if tok := userToken(ctx); strings.HasPrefix(tok, apitoken.Prefix) {
		return tok
	}
	return ""
}

// viewUserID — пользователь (токен "cs_…") для журнала просмотров; не вошёл или токен неверный — nil.
func (s *CarSearch) viewUserID(ctx context.Context) *string {
	tok := userToken(ctx)
	if s.accounts == nil || tok == "" || strings.HasPrefix(tok, apitoken.Prefix) {
		return nil
	}
	u, err := s.accounts.UserByToken(ctx, tok, includeTestData(ctx))
	if err != nil {
		return nil
	}
	return &u.ID
}

// userAgent — заголовок user-agent запроса (в журнал просмотров).
func userAgent(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	if v := md.Get("user-agent"); len(v) > 0 {
		return v[0]
	}
	return ""
}

// apiTokenErr — ошибка проверки API-токена в gRPC-статус; nil — не ошибка токена.
func apiTokenErr(err error) error {
	switch {
	case errors.Is(err, models.ErrTokenInvalid):
		return status.Error(codes.Unauthenticated, "API-токен недействителен")
	case errors.Is(err, models.ErrTokenRevoked):
		return status.Error(codes.PermissionDenied, "API-токен отозван")
	case errors.Is(err, models.ErrTokenExhausted):
		return status.Error(codes.ResourceExhausted, "лимит запросов токена исчерпан")
	}
	return nil
}

func tsOrNil(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func apiTokenPB(t models.ApiToken) *pb.ApiToken {
	return &pb.ApiToken{
		Id: t.ID, Name: t.Name,
		RequestsTotal: int32(t.RequestsTotal), RequestsLeft: int32(t.RequestsLeft),
		Revoked: t.Revoked, CreatedAt: timestamppb.New(t.CreatedAt), LastUsedAt: tsOrNil(t.LastUsedAt),
		ViewsCount: int32(t.ViewsCount),
	}
}

// requireTokensAdmin — админский метод токенов: admin-token верен и токены включены.
func (s *CarSearch) requireTokensAdmin(ctx context.Context) error {
	if err := requireAdmin(ctx, s.adminToken); err != nil {
		return err
	}
	if s.tokens == nil {
		return status.Error(codes.Unimplemented, "API-токены не включены")
	}
	return nil
}

// pageArgs проверяет offset/limit админских списков; limit 0 — defaultComplaintsLimit.
func pageArgs(offset, limit int32) (int, int, error) {
	switch {
	case offset < 0:
		return 0, 0, status.Error(codes.InvalidArgument, "offset: не может быть отрицательным")
	case limit < 0 || limit > MaxLimit:
		return 0, 0, status.Errorf(codes.InvalidArgument, "limit: от 1 до %d (0 — по умолчанию %d)", MaxLimit, defaultComplaintsLimit)
	case limit == 0:
		limit = defaultComplaintsLimit
	}
	return int(offset), int(limit), nil
}

func nextOffsetPB(n *int) *int32 {
	if n == nil {
		return nil
	}
	v := int32(*n)
	return &v
}
