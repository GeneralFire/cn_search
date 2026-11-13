package implementation

//go:generate minimock -i plateSightings -o ./mocks/plate_sightings_mock.go -n PlateSightingsMock -p mocks
//go:generate minimock -i accounts -o ./mocks/accounts_mock.go -n AccountsMock -p mocks
//go:generate minimock -i complaints -o ./mocks/complaints_mock.go -n ComplaintsMock -p mocks
//go:generate minimock -i admin -o ./mocks/admin_mock.go -n AdminMock -p mocks
//go:generate minimock -i cameraRequests -o ./mocks/camera_requests_mock.go -n CameraRequestsMock -p mocks

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
