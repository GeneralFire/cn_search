// Package implementation реализует gRPC-контракт CarSearchService (be/api/carsearch/v1).
// HTTP-ручки поверх него генерирует clay. Один RPC — один файл: search.go, get.go.
package implementation

import (
	"context"
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

	complaints complaints // жалобы на фото (WithComplaints); nil — методы жалоб недоступны
	adminToken string     // ADMIN_TOKEN: пусто — админские методы выключены
	admin      admin      // админские сценарии (WithAdmin); nil — HideSighting и разбор уточнений недоступны

	accounts accounts // пользователи и оплата (WithAccounts); nil — методы аккаунта недоступны
}

// Option — необязательная часть сервиса.
type Option func(*CarSearch)

// WithComplaints включает жалобы на фото (Complain) и их разбор (ListComplaints, ResolveComplaint)
// по заголовку admin-token, равному adminToken; пустой adminToken — админские методы выключены.
func WithComplaints(c complaints, adminToken string) Option {
	return func(s *CarSearch) { s.complaints, s.adminToken = c, adminToken }
}

// WithAdmin включает админские методы (HideSighting, SetClarificationStatus, ListClarifications) по заголовку
// admin-token, равному adminToken (тот же ADMIN_TOKEN, что у WithComplaints).
func WithAdmin(a admin, adminToken string) Option {
	return func(s *CarSearch) { s.admin, s.adminToken = a, adminToken }
}

// isAdmin — запрос с верным admin-token: ему отдаются и скрытые фиксации.
func (s *CarSearch) isAdmin(ctx context.Context) bool {
	return requireAdmin(ctx, s.adminToken) == nil
}

// WithAccounts включает пользователей (вход по токену), тарифы и мок-оплату.
func WithAccounts(a accounts) Option {
	return func(s *CarSearch) { s.accounts = a }
}

func NewCarSearch(uc plateSightings, photoBaseURL string, opts ...Option) *CarSearch {
	s := &CarSearch{uc: uc, photoBase: strings.TrimRight(photoBaseURL, "/")}
	for _, o := range opts {
		o(s)
	}
	return s
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

		Hidden: c.Hidden,
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
