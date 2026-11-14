package implementation

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

const maxEmailLen = 254

// Register — rpc Register: новый пользователь Free, токен — в ответе один раз.
func (s *CarSearch) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if s.accounts == nil {
		return nil, status.Error(codes.Unimplemented, "аккаунты не включены")
	}
	email := strings.TrimSpace(req.GetEmail())
	if email != "" {
		a, err := mail.ParseAddress(email)
		if err != nil || a.Address != email || len(email) > maxEmailLen {
			return nil, status.Error(codes.InvalidArgument, "email: неверный адрес")
		}
	}
	token, u, err := s.accounts.Register(ctx, email)
	switch {
	case errors.Is(err, models.ErrEmailTaken):
		return nil, status.Error(codes.AlreadyExists, "email: уже зарегистрирован")
	case err != nil:
		return nil, internalErr(ctx, err, "не удалось зарегистрироваться")
	}
	return &pb.RegisterResponse{Token: token, User: userPB(u)}, nil
}

func userPB(u models.User) *pb.User {
	out := &pb.User{Id: u.ID, Email: u.Email, Plan: planPB[u.Plan], CreatedAt: timestamppb.New(u.CreatedAt)}
	if u.ProUntil != nil {
		out.ProUntil = timestamppb.New(*u.ProUntil)
	}
	return out
}

var planPB = map[models.Plan]pb.UserPlan{
	models.PlanFree: pb.UserPlan_USER_PLAN_FREE,
	models.PlanPro:  pb.UserPlan_USER_PLAN_PRO,
}
