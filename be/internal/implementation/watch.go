package implementation

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

var errWatchesOff = status.Error(codes.Unimplemented, "подписки не включены")

// watchUser — пользователь запроса для методов подписок: включены ли подписки, затем токен.
func (s *CarSearch) watchUser(ctx context.Context) (models.User, error) {
	if s.watches == nil {
		return models.User{}, errWatchesOff
	}
	return s.currentUser(ctx)
}

// maxWatchPlate — длина номера подписки, символов (как поле поиска на фронте).
const maxWatchPlate = 32

// watchPlate — номер из запроса: без пробелов по краям, не длиннее maxWatchPlate, хотя бы с одной буквой или цифрой.
func watchPlate(field, raw string) (string, error) {
	plate := strings.TrimSpace(raw)
	if utf8.RuneCountInString(plate) > maxWatchPlate || !strings.ContainsFunc(plate, func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	}) {
		return "", status.Errorf(codes.InvalidArgument, "%s: нужен номер (буквы/цифры, до %d символов)", field, maxWatchPlate)
	}
	return plate, nil
}

// Watch — rpc Watch: подписка на номер (он может ещё не встречаться в базе).
func (s *CarSearch) Watch(ctx context.Context, req *pb.WatchRequest) (*pb.WatchResponse, error) {
	u, err := s.watchUser(ctx)
	if err != nil {
		return nil, err
	}
	plate, err := watchPlate("plate", req.GetPlate())
	if err != nil {
		return nil, err
	}
	if err := s.watches.Watch(ctx, u.ID, plate); err != nil {
		return nil, status.Error(codes.Internal, "не удалось подписаться")
	}
	return &pb.WatchResponse{}, nil
}
