package implementation

import (
	"context"
	"unicode/utf8"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

const (
	DefaultLimit   = 35 // кратно 5 и 7 — по числу колонок сетки на фронте
	MaxLimit       = 100
	maxQueryLength = 32
)

// Search — rpc Search: появления номеров по частям номера, постранично.
func (s *CarSearch) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	if err := validateSearch(req); err != nil {
		return nil, err
	}
	limit := int(req.GetLimit())
	if limit == 0 {
		limit = DefaultLimit
	}

	page, err := s.uc.Search(ctx, models.SearchOpts{
		Plate:           req.GetPlate(),
		Offset:          int(req.GetOffset()),
		Limit:           limit,
		IncludeTestData: includeTestData(ctx),
		Market:          models.Market(req.GetMarket()),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "не удалось выполнить поиск")
	}

	resp := &pb.SearchResponse{
		Items: make([]*pb.PlateSighting, len(page.Items)),
		Total: int32(page.Total),
	}
	for i, c := range page.Items {
		resp.Items[i] = s.toPB(c)
	}
	for _, c := range page.Suggestions {
		resp.Suggestions = append(resp.Suggestions, s.toPB(c))
	}
	if page.NextOffset != nil {
		next := int32(*page.NextOffset)
		resp.NextOffset = &next
	}
	return resp, nil
}

func validateSearch(req *pb.SearchRequest) error {
	switch {
	case utf8.RuneCountInString(req.GetPlate()) > maxQueryLength:
		return status.Errorf(codes.InvalidArgument, "plate: не длиннее %d символов", maxQueryLength)
	case req.GetOffset() < 0:
		return status.Error(codes.InvalidArgument, "offset: не может быть отрицательным")
	case !models.Market(req.GetMarket()).Valid():
		return status.Error(codes.InvalidArgument, "market: ru, ua, de, us, cn, other или пусто")
	case req.GetLimit() < 0 || req.GetLimit() > MaxLimit:
		return status.Errorf(codes.InvalidArgument, "limit: от 1 до %d (0 — по умолчанию %d)", MaxLimit, DefaultLimit)
	}
	return nil
}
