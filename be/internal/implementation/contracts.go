package implementation

//go:generate minimock -i plateSightings -o ./mocks/plate_sightings_mock.go -n PlateSightingsMock -p mocks

import (
	"context"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// plateSightings — сценарии, которые вызывают RPC (реализация — usecase.PlateSightings).
type plateSightings interface {
	Search(ctx context.Context, q models.SearchOpts) (models.SearchPage, error)
	ListSamePlate(ctx context.Context, o models.SamePlateOpts) ([]models.PlateSighting, error)
	SeeAlso(ctx context.Context, o models.SeeAlsoOpts) ([]models.PlateSighting, error)
	Clarify(ctx context.Context, o models.ClarifyOpts) (models.Clarification, error)
}
