// Package implementation реализует gRPC-контракт CarSearchService (be/api/carsearch/v1).
// HTTP-ручки поверх него генерирует clay. Один RPC — один файл: search.go, get.go.
package implementation

import (
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// CarSearch — реализация CarSearchServiceServer.
type CarSearch struct {
	pb.UnimplementedCarSearchServiceServer

	uc        plateSightings
	photoBase string // публичный URL бакета с картинками, напр. "/s3/cars"
}

func NewCarSearch(uc plateSightings, photoBaseURL string) *CarSearch {
	return &CarSearch{uc: uc, photoBase: strings.TrimRight(photoBaseURL, "/")}
}

func (s *CarSearch) toPB(c models.PlateSighting) *pb.PlateSighting {
	out := &pb.PlateSighting{
		Id:        c.ID,
		Address:   c.Address,
		Plate:     c.Plate,
		CreatedAt: timestamppb.New(c.CreatedAt),

		PlateType:     plateTypePB[c.PlateType],
		PlateRegion:   c.PlateRegion,
		AddressLocale: c.AddressLocale,

		Lat: c.Lat,
		Lon: c.Lon,
	}
	if c.PhotoKey != "" {
		out.PhotoUrl = s.photoBase + "/" + c.PhotoKey
	}
	if c.ThumbKey != "" {
		out.ThumbUrl = s.photoBase + "/" + c.ThumbKey
	}
	return out
}

var plateTypePB = map[string]pb.PlateType{
	"ru": pb.PlateType_PLATE_TYPE_RU,
	"eu": pb.PlateType_PLATE_TYPE_EU,
	"us": pb.PlateType_PLATE_TYPE_US,
	"ua": pb.PlateType_PLATE_TYPE_UA,
	"cn": pb.PlateType_PLATE_TYPE_CN,
}
