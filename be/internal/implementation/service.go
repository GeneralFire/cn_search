// Package implementation реализует gRPC-контракт CarSearchService (be/api/carsearch/v1).
// HTTP-ручки поверх него генерирует clay. Один RPC — один файл: search.go, get.go.
package implementation

import (
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/repository"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// CarSearch — реализация CarSearchServiceServer.
type CarSearch struct {
	pb.UnimplementedCarSearchServiceServer

	repo      repository.Repository
	photoBase string // публичный URL бакета с картинками, напр. "/s3/cars"
}

func NewCarSearch(repo repository.Repository, photoBaseURL string) *CarSearch {
	return &CarSearch{repo: repo, photoBase: strings.TrimRight(photoBaseURL, "/")}
}

func (s *CarSearch) toPB(c models.PlateSighting) *pb.PlateSighting {
	out := &pb.PlateSighting{
		Id:        int32(c.ID),
		Address:   c.Address,
		Plate:     c.Plate,
		CreatedAt: timestamppb.New(c.CreatedAt),
	}
	if c.PhotoKey != "" {
		out.PhotoUrl = s.photoBase + "/" + c.PhotoKey
	}
	return out
}
