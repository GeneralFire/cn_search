// Package service реализует gRPC-контракт CarSearchService (be/api/carsearch/v1).
// HTTP-ручки поверх него генерирует clay.
package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/cars"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch/v1"
)

const (
	DefaultLimit   = 35 // кратно 5 и 7 — по числу колонок сетки на фронте
	MaxLimit       = 100
	maxQueryLength = 32
)

// CarSearch — реализация CarSearchServiceServer.
type CarSearch struct {
	pb.UnimplementedCarSearchServiceServer

	repo      cars.Repository
	photoBase string // публичный URL бакета с картинками, напр. "/s3/cars"
}

func NewCarSearch(repo cars.Repository, photoBaseURL string) *CarSearch {
	return &CarSearch{repo: repo, photoBase: strings.TrimRight(photoBaseURL, "/")}
}

func (s *CarSearch) SearchCars(ctx context.Context, req *pb.SearchCarsRequest) (*pb.SearchCarsResponse, error) {
	if err := validateSearch(req); err != nil {
		return nil, err
	}
	limit := int(req.GetLimit())
	if limit == 0 {
		limit = DefaultLimit
	}

	page, err := s.repo.Search(ctx, cars.Query{
		Plate:  req.GetPlate(),
		Key:    req.GetKey(),
		Offset: int(req.GetOffset()),
		Limit:  limit,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "не удалось выполнить поиск")
	}

	resp := &pb.SearchCarsResponse{
		Items: make([]*pb.Car, len(page.Items)),
		Total: int32(page.Total),
	}
	for i, c := range page.Items {
		resp.Items[i] = s.toPB(c)
	}
	if page.NextOffset != nil {
		next := int32(*page.NextOffset)
		resp.NextOffset = &next
	}
	return resp, nil
}

func (s *CarSearch) GetCar(ctx context.Context, req *pb.GetCarRequest) (*pb.GetCarResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id должен быть положительным")
	}
	c, err := s.repo.Get(ctx, int(req.GetId()))
	switch {
	case errors.Is(err, cars.ErrNotFound):
		return nil, status.Error(codes.NotFound, "машина не найдена")
	case err != nil:
		return nil, status.Error(codes.Internal, "не удалось получить машину")
	}
	return &pb.GetCarResponse{Car: s.toPB(c)}, nil
}

func validateSearch(req *pb.SearchCarsRequest) error {
	switch {
	case utf8.RuneCountInString(req.GetPlate()) > maxQueryLength:
		return status.Errorf(codes.InvalidArgument, "plate: не длиннее %d символов", maxQueryLength)
	case utf8.RuneCountInString(req.GetKey()) > maxQueryLength:
		return status.Errorf(codes.InvalidArgument, "key: не длиннее %d символов", maxQueryLength)
	case req.GetOffset() < 0:
		return status.Error(codes.InvalidArgument, "offset: не может быть отрицательным")
	case req.GetLimit() < 0 || req.GetLimit() > MaxLimit:
		return status.Errorf(codes.InvalidArgument, "limit: от 1 до %d (0 — по умолчанию %d)", MaxLimit, DefaultLimit)
	}
	return nil
}

func (s *CarSearch) toPB(c cars.Car) *pb.Car {
	out := &pb.Car{
		Id:      int32(c.ID),
		Address: c.Address,
		Plate:   c.Plate,
		Key:     c.Key,
		Date:    c.Date,
	}
	if c.PhotoKey != "" {
		out.PhotoUrl = s.photoBase + "/" + c.PhotoKey
	}
	return out
}
