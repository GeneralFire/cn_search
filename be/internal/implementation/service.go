// Package implementation реализует gRPC-контракт CarSearchService (be/api/carsearch/v1).
// HTTP-ручки поверх него генерирует clay. Один RPC — один файл: search.go, get.go.
package implementation

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/apitoken"
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

	cameraRequests cameraRequests // заявки на камеры (WithCameraRequests); nil — методы заявок недоступны
	tokens         apiTokens      // API-токены и журнал просмотров (WithApiTokens); nil — Get без журнала
	watches        watches        // подписки на номера (WithWatches); nil — методы подписок недоступны
	ingest         ingest         // новая фиксация по фото (WithIngest); nil — Ingest недоступен
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

// WithCameraRequests включает заявки на камеры. Админские методы — по тому же admin-token, что и жалобы
// (adminToken из WithComplaints).
func WithCameraRequests(c cameraRequests) Option {
	return func(s *CarSearch) { s.cameraRequests = c }
}

// WithApiTokens включает API-токены: списание запросов и журнал открытий в Get, админские методы
// токенов и журнала (заголовок admin-token, см. WithComplaints) и TokenInfo.
func WithApiTokens(t apiTokens) Option {
	return func(s *CarSearch) { s.tokens = t }
}

// WithWatches включает подписки на номера (Watch, Unwatch, ListWatches, MarkWatchSeen).
func WithWatches(w watches) Option {
	return func(s *CarSearch) { s.watches = w }
}

// WithIngest включает rpc Ingest (фото → OCR → фиксация).
func WithIngest(i ingest) Option {
	return func(s *CarSearch) { s.ingest = i }
}

func NewCarSearch(uc plateSightings, photoBaseURL string, opts ...Option) *CarSearch {
	s := &CarSearch{uc: uc, photoBase: strings.TrimRight(photoBaseURL, "/")}
	for _, o := range opts {
		o(s)
	}
	return s
}

// canSeeCoords — запрос с входом: admin-token, действующий токен пользователя (cs_…) или API-токен (cst_…).
// Аноним координат не видит (см. toPB). Неверный токен — тоже аноним.
func (s *CarSearch) canSeeCoords(ctx context.Context) bool {
	if s.isAdmin(ctx) {
		return true
	}
	tok := userToken(ctx)
	if tok == "" {
		return false
	}
	if strings.HasPrefix(tok, apitoken.Prefix) {
		if s.tokens == nil {
			return false
		}
		_, err := s.tokens.TokenInfo(ctx, tok)
		// Исчерпанный токен — всё равно действующий вход; неверный и отозванный — аноним.
		return err == nil || errors.Is(err, models.ErrTokenExhausted)
	}
	if s.accounts == nil {
		return false
	}
	_, err := s.accounts.UserByToken(ctx, tok, includeTestData(ctx))
	return err == nil
}

// toPB — модель фиксации в ответ. coords=false (аноним, см. canSeeCoords): lat/lon не заполняются,
// coords_hidden=true — единственное место, где скрываются координаты.
func (s *CarSearch) toPB(c models.PlateSighting, coords bool) *pb.PlateSighting {
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

		Hidden:      c.Hidden,
		PhotoHidden: c.PhotoHidden,
	}
	if !coords {
		out.Lat, out.Lon, out.CoordsHidden = nil, nil, true
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
