package implementation

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc/metadata"

	"github.com/GeneralFire/cn_search/be/internal/implementation/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// TestCoordsVisibility: аноним не видит lat/lon (coords_hidden=true), вошедшие — видят.
func TestCoordsVisibility(t *testing.T) {
	lat, lon := 55.7579, 37.6115
	withCoords := sighting
	withCoords.Lat, withCoords.Lon = &lat, &lon
	page := models.SearchPage{Items: []models.PlateSighting{withCoords}, Suggestions: []models.PlateSighting{withCoords}}

	cases := map[string]struct {
		md      []string
		user    bool // токен пользователя валиден
		apiTok  bool // API-токен валиден
		visible bool
	}{
		"аноним":              {md: nil},
		"неверный токен":      {md: []string{"cookie", "cs_token=cs_bad"}},
		"неверный api-токен":  {md: []string{"cookie", "cs_token=cst_bad"}},
		"пользователь":        {md: []string{"cookie", "cs_token=cs_ok"}, user: true, visible: true},
		"пользователь Bearer": {md: []string{"authorization", "Bearer cs_ok"}, user: true, visible: true},
		"api-токен":           {md: []string{"cookie", "cs_token=cst_ok"}, apiTok: true, visible: true},
		"админ":               {md: []string{"admin-token", "adm"}, visible: true},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			uc := mocks.NewPlateSightingsMock(t)
			acc := mocks.NewAccountsMock(t)
			tok := mocks.NewApiTokensMock(t)
			svc := NewCarSearch(uc, "/s3/", WithAccounts(acc), WithApiTokens(tok), WithComplaints(nil, "adm"))
			acc.UserByTokenMock.Optional().Set(func(_ context.Context, token string, _ bool) (models.User, error) {
				if c.user && token == "cs_ok" {
					return models.User{ID: "u1"}, nil
				}
				return models.User{}, models.ErrNotFound
			})
			tok.TokenInfoMock.Optional().Set(func(_ context.Context, token string) (models.ApiToken, error) {
				if c.apiTok && token == "cst_ok" {
					return models.ApiToken{}, nil
				}
				return models.ApiToken{}, models.ErrTokenInvalid
			})
			tok.OpenPlateMock.Optional().Return(models.SamePlate{Items: []models.PlateSighting{withCoords}}, nil)
			uc.SearchMock.Optional().Return(page, nil)
			uc.SeeAlsoMock.Optional().Return([]models.PlateSighting{withCoords}, nil)

			ctx := context.Background()
			if len(c.md) > 0 {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(c.md...))
			}
			var got []*pb.PlateSighting
			g, err := svc.Get(ctx, &pb.GetRequest{Id: "00000000-0000-0000-0000-000000000001"})
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, g.GetItems()...)
			s, err := svc.Search(ctx, &pb.SearchRequest{Plate: "А123"})
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, s.GetItems()...)
			got = append(got, s.GetSuggestions()...)
			sa, err := svc.SeeAlso(ctx, &pb.SeeAlsoRequest{})
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, sa.GetItems()...)
			if len(got) != 4 {
				t.Fatalf("ожидалось 4 фиксации, получено %d", len(got))
			}
			for _, it := range got {
				hasCoords := it.Lat != nil && it.Lon != nil && it.GetLat() == lat && it.GetLon() == lon
				if c.visible && (!hasCoords || it.GetCoordsHidden()) {
					t.Errorf("координаты должны быть видны: %v", it)
				}
				if !c.visible && (it.Lat != nil || it.Lon != nil || !it.GetCoordsHidden()) {
					t.Errorf("координаты должны быть скрыты: %v", it)
				}
			}
		})
	}
}

var _ = minimock.AnyContext
