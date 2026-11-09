package implementation

import (
	"testing"
	"time"

	"github.com/GeneralFire/cn_search/be/internal/implementation/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

var sighting = models.PlateSighting{
	ID: "00000000-0000-0000-0000-000000000001", PhotoKey: "car-1.jpg", ThumbKey: "car-1_thumb.jpg", Address: "Москва, Пресненская наб., 10", Plate: "А123ВС 777",
	CreatedAt: time.Date(2026, 11, 2, 9, 15, 0, 0, time.UTC),
}

func newSvc(t *testing.T) (*CarSearch, *mocks.PlateSightingsMock) {
	uc := mocks.NewPlateSightingsMock(t)
	return NewCarSearch(uc, "/s3/cars/"), uc
}
