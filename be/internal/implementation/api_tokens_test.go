package implementation

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/implementation/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

func newTokensSvc(t *testing.T) (*CarSearch, *mocks.ApiTokensMock) {
	tok := mocks.NewApiTokensMock(t)
	return NewCarSearch(mocks.NewPlateSightingsMock(t), "/s3/cars", WithComplaints(nil, "adm"), WithApiTokens(tok)), tok
}

func mdCtx(kv ...string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(kv...))
}

func TestApiTokenFromMD(t *testing.T) {
	for name, c := range map[string]struct {
		ctx  context.Context
		want string
	}{
		"нет":                {context.Background(), ""},
		"bearer":             {mdCtx("authorization", "Bearer cst_a.b"), "cst_a.b"},
		"cookie":             {mdCtx("cookie", "theme=dark; cs_token=cst_x.y"), "cst_x.y"},
		"токен пользователя": {mdCtx("cookie", "cs_token=cs_user"), ""},
		"bearer не cst":      {mdCtx("authorization", "Bearer cs_user"), ""},
	} {
		if got := apiTokenFromMD(c.ctx); got != c.want {
			t.Errorf("%s: %q, want %q", name, got, c.want)
		}
	}
}

func TestGetWithTokens(t *testing.T) {
	const id = "00000000-0000-0000-0000-000000000001"
	svc, tok := newTokensSvc(t)
	tok.OpenPlateMock.Expect(minimock.AnyContext, models.OpenPlateOpts{
		SamePlateOpts: models.SamePlateOpts{ID: id}, Token: "cst_a.b", UserAgent: "curl",
	}).Return(models.SamePlate{Items: []models.PlateSighting{sighting}}, nil)
	ctx := mdCtx("cookie", "cs_token=cst_a.b", "user-agent", "curl")
	if resp, err := svc.Get(ctx, &pb.GetRequest{Id: id}); err != nil || len(resp.GetItems()) != 1 {
		t.Fatalf("Get: %v %v", resp, err)
	}
}

func TestGetAdminFlag(t *testing.T) {
	const id = "00000000-0000-0000-0000-000000000001"
	svc, tok := newTokensSvc(t)
	tok.OpenPlateMock.Expect(minimock.AnyContext, models.OpenPlateOpts{
		SamePlateOpts: models.SamePlateOpts{ID: id, IncludeHidden: true}, IsAdmin: true,
	}).Return(models.SamePlate{Items: []models.PlateSighting{sighting}}, nil)
	if _, err := svc.Get(mdCtx("admin-token", "adm"), &pb.GetRequest{Id: id}); err != nil {
		t.Fatal(err)
	}
}

func TestGetTokenErrors(t *testing.T) {
	for err, want := range map[error]codes.Code{
		models.ErrTokenInvalid:   codes.Unauthenticated,
		models.ErrTokenRevoked:   codes.PermissionDenied,
		models.ErrTokenExhausted: codes.ResourceExhausted,
	} {
		svc, tok := newTokensSvc(t)
		tok.OpenPlateMock.Return(models.SamePlate{}, err)
		_, got := svc.Get(context.Background(), &pb.GetRequest{Id: "00000000-0000-0000-0000-000000000001"})
		if status.Code(got) != want {
			t.Errorf("%v: %v, want %v", err, got, want)
		}
	}
}

func TestCreateApiToken(t *testing.T) {
	svc, tok := newTokensSvc(t)
	if _, err := svc.CreateApiToken(context.Background(), &pb.CreateApiTokenRequest{Name: "x", Requests: 3}); status.Code(err) != codes.PermissionDenied {
		t.Errorf("без admin-token: %v", err)
	}
	adm := mdCtx("admin-token", "adm")
	for _, bad := range []*pb.CreateApiTokenRequest{{Name: " ", Requests: 3}, {Name: "x", Requests: 0}, {Name: "x", Requests: maxApiTokenRequests + 1}} {
		if _, err := svc.CreateApiToken(adm, bad); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%v: %v", bad, err)
		}
	}
	tok.CreateApiTokenMock.Expect(minimock.AnyContext, "партнёр", 3).Return("cst_t.s", models.ApiToken{ID: "id1", RequestsTotal: 3, RequestsLeft: 3}, nil)
	resp, err := svc.CreateApiToken(adm, &pb.CreateApiTokenRequest{Name: " партнёр ", Requests: 3})
	if err != nil || resp.GetToken() != "cst_t.s" || resp.GetInfo().GetRequestsLeft() != 3 || resp.GetInfo().LastUsedAt != nil {
		t.Fatalf("%v %v", resp, err)
	}
}

func TestTokenInfo(t *testing.T) {
	svc, tok := newTokensSvc(t)
	if _, err := svc.TokenInfo(context.Background(), &pb.TokenInfoRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Errorf("без токена: %v", err)
	}
	tok.TokenInfoMock.Expect(minimock.AnyContext, "cst_a.b").Return(models.ApiToken{RequestsLeft: 2, RequestsTotal: 3}, nil)
	info, err := svc.TokenInfo(mdCtx("authorization", "Bearer cst_a.b"), &pb.TokenInfoRequest{})
	if err != nil || info.GetRequestsLeft() != 2 {
		t.Fatalf("%v %v", info, err)
	}
}

func TestListPlateViewsValidation(t *testing.T) {
	svc, _ := newTokensSvc(t)
	adm := mdCtx("admin-token", "adm")
	if _, err := svc.ListPlateViews(adm, &pb.ListPlateViewsRequest{TokenId: "nope"}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("token_id: %v", err)
	}
	if _, err := svc.RevokeApiToken(adm, &pb.RevokeApiTokenRequest{Id: "nope"}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("revoke id: %v", err)
	}
}
