package implementation

//go:generate minimock -i plateSightings -o ./mocks/plate_sightings_mock.go -n PlateSightingsMock -p mocks
//go:generate minimock -i accounts -o ./mocks/accounts_mock.go -n AccountsMock -p mocks
//go:generate minimock -i complaints -o ./mocks/complaints_mock.go -n ComplaintsMock -p mocks
//go:generate minimock -i admin -o ./mocks/admin_mock.go -n AdminMock -p mocks
//go:generate minimock -i cameraRequests -o ./mocks/camera_requests_mock.go -n CameraRequestsMock -p mocks
//go:generate minimock -i apiTokens -o ./mocks/api_tokens_mock.go -n ApiTokensMock -p mocks
//go:generate minimock -i ingest -o ./mocks/ingest_mock.go -n IngestMock -p mocks

import (
	"context"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// plateSightings — сценарии, которые вызывают RPC (реализация — usecase.PlateSightings).
type plateSightings interface {
	Search(ctx context.Context, q models.SearchOpts) (models.SearchPage, error)
	ListSamePlate(ctx context.Context, o models.SamePlateOpts) (models.SamePlate, error)
	SeeAlso(ctx context.Context, o models.SeeAlsoOpts) ([]models.PlateSighting, error)
	Clarify(ctx context.Context, o models.ClarifyOpts) (models.Clarification, error)
}

// complaints — сценарии жалоб на фото (реализация — usecase.Complaints).
type complaints interface {
	Complain(ctx context.Context, o models.ComplainOpts) (models.Complaint, error)
	ListComplaints(ctx context.Context, o models.ListComplaintsOpts) (models.ComplaintsPage, error)
	ResolveComplaint(ctx context.Context, id string) (models.Complaint, error)
}

// admin — админские сценарии: скрытие фиксаций и разбор уточнений (реализация — usecase.Admin).
type admin interface {
	HideSighting(ctx context.Context, id string, hidden bool) (models.PlateSighting, error)
	SetClarificationStatus(ctx context.Context, id string, st models.ClarificationStatus) (models.Clarification, error)
	ListClarifications(ctx context.Context, o models.ListClarificationsOpts) (models.ClarificationsPage, error)
}

// accounts — пользователи, тарифы и мок-оплата (реализация — usecase.Accounts).
type accounts interface {
	Register(ctx context.Context, email string) (string, models.User, error)
	UserByToken(ctx context.Context, token string, includeTestData bool) (models.User, error)
	Plans() []models.PlanInfo
	CreatePayment(ctx context.Context, userID string, plan models.Plan) (models.Payment, error)
	ConfirmMockPayment(ctx context.Context, userID, paymentID string, success bool) (models.Payment, error)
	ListPayments(ctx context.Context, userID string) ([]models.Payment, error)
}

// cameraRequests — сценарии заявок на камеры (реализация — usecase.CameraRequests).
type cameraRequests interface {
	Submit(ctx context.Context, o models.SubmitCameraRequestOpts) (models.CameraRequest, error)
	List(ctx context.Context, o models.ListCameraRequestsOpts) (models.CameraRequestsPage, error)
	Vote(ctx context.Context, o models.VoteCameraRequestOpts) (models.CameraRequest, error)
	SetStatus(ctx context.Context, id string, st models.CameraRequestStatus) (models.CameraRequest, error)
}

// apiTokens — API-токены и журнал открытий страниц номеров (реализация — usecase.ApiTokens).
type apiTokens interface {
	CreateApiToken(ctx context.Context, name string, requests int) (string, models.ApiToken, error)
	ListApiTokens(ctx context.Context, o models.ListApiTokensOpts) (models.ApiTokensPage, error)
	RevokeApiToken(ctx context.Context, id string) (models.ApiToken, error)
	ListPlateViews(ctx context.Context, o models.ListPlateViewsOpts) (models.PlateViewsPage, error)
	TokenInfo(ctx context.Context, token string) (models.ApiToken, error)
	OpenPlate(ctx context.Context, o models.OpenPlateOpts) (models.SamePlate, error)
}

// ingest — новая фиксация по фото (реализация — usecase.Ingest).
type ingest interface {
	Ingest(ctx context.Context, o models.IngestOpts) (models.IngestResult, error)
	UploadsLeft(ctx context.Context, user models.User) (*int, error)
}
