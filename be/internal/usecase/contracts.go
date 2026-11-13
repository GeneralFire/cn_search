package usecase

//go:generate minimock -i transactor -o ./mocks/transactor_mock.go -n TransactorMock -p mocks
//go:generate minimock -i repository -o ./mocks/repository_mock.go -n RepositoryMock -p mocks
//go:generate minimock -i accountsRepository -o ./mocks/accounts_repository_mock.go -n AccountsRepositoryMock -p mocks
//go:generate minimock -i complaintsRepository -o ./mocks/complaints_repository_mock.go -n ComplaintsRepositoryMock -p mocks
//go:generate minimock -i adminRepository -o ./mocks/admin_repository_mock.go -n AdminRepositoryMock -p mocks
//go:generate minimock -i cameraRequestsRepository -o ./mocks/camera_requests_repository_mock.go -n CameraRequestsRepositoryMock -p mocks
//go:generate minimock -i apiTokensRepository -o ./mocks/api_tokens_repository_mock.go -n ApiTokensRepositoryMock -p mocks
//go:generate minimock -i ingestRepository -o ./mocks/ingest_repository_mock.go -n IngestRepositoryMock -p mocks
//go:generate minimock -i recognizer -o ./mocks/recognizer_mock.go -n RecognizerMock -p mocks
//go:generate minimock -i photoStorage -o ./mocks/photo_storage_mock.go -n PhotoStorageMock -p mocks

import (
	"context"
	"time"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// transactor открывает транзакции (реализация — db.Cluster).
type transactor interface {
	InReadTx(ctx context.Context, fn db.TxFunc) error
	InWriteTx(ctx context.Context, fn db.TxFunc) error
}

// repository — запросы к БД внутри переданной транзакции (реализация — repository.New()).
type repository interface {
	Search(ctx context.Context, tx db.Executor, q models.SearchOpts) (models.SearchPage, error)
	Suggest(ctx context.Context, tx db.Executor, q models.SearchOpts) ([]models.PlateSighting, error)
	ListSamePlate(ctx context.Context, tx db.Executor, o models.SamePlateOpts) ([]models.PlateSighting, error)
	SeeAlso(ctx context.Context, tx db.Executor, o models.SeeAlsoOpts) ([]models.PlateSighting, error)
	ListPlateClarifications(ctx context.Context, tx db.Executor, o models.SamePlateOpts) ([]models.PlateClarification, error)
	CreateClarification(ctx context.Context, tx db.Executor, o models.ClarifyOpts) (models.Clarification, error)
	MatchesSightingPlate(ctx context.Context, tx db.Executor, sightingID, plate string) (bool, error)
	CountMatchingClarifications(ctx context.Context, tx db.Executor, sightingID, plate string) (int, error)
	AcceptMatchingClarifications(ctx context.Context, tx db.Executor, sightingID, plate string) error
}

// complaintsRepository — запросы к жалобам на фото (реализация — repository.New()).
type complaintsRepository interface {
	CreateComplaint(ctx context.Context, tx db.Executor, o models.ComplainOpts) (models.Complaint, error)
	ListComplaints(ctx context.Context, tx db.Executor, o models.ListComplaintsOpts) (models.ComplaintsPage, error)
	ResolveComplaint(ctx context.Context, tx db.Executor, id string) (models.Complaint, error)
}

// adminRepository — админские запросы: скрытие фиксаций, разбор уточнений (реализация — repository.New()).
type adminRepository interface {
	HideSighting(ctx context.Context, tx db.Executor, id string, hidden bool) (models.PlateSighting, error)
	SetClarificationStatus(ctx context.Context, tx db.Executor, id string, st models.ClarificationStatus) (models.Clarification, error)
	ListClarifications(ctx context.Context, tx db.Executor, o models.ListClarificationsOpts) (models.ClarificationsPage, error)
}

// accountsRepository — пользователи и платежи (реализация — repository.New()).
type accountsRepository interface {
	CreateUser(ctx context.Context, tx db.Executor, tokenHash, email string) (models.User, error)
	UserByTokenHash(ctx context.Context, tx db.Executor, tokenHash string, includeTestData bool) (models.User, error)
	CreatePayment(ctx context.Context, tx db.Executor, o models.NewPaymentOpts) (models.Payment, error)
	PaymentForUpdate(ctx context.Context, tx db.Executor, userID, paymentID string) (models.Payment, error)
	SetPaymentStatus(ctx context.Context, tx db.Executor, paymentID string, st models.PaymentStatus) (models.Payment, error)
	ExtendPro(ctx context.Context, tx db.Executor, userID string, period time.Duration) (models.User, error)
	ListPayments(ctx context.Context, tx db.Executor, userID string, limit int) ([]models.Payment, error)
}

// cameraRequestsRepository — запросы к заявкам на камеры (реализация — repository.New()).
type cameraRequestsRepository interface {
	CreateCameraRequest(ctx context.Context, tx db.Executor, o models.SubmitCameraRequestOpts) (models.CameraRequest, error)
	ListCameraRequests(ctx context.Context, tx db.Executor, o models.ListCameraRequestsOpts) (models.CameraRequestsPage, error)
	VoteCameraRequest(ctx context.Context, tx db.Executor, o models.VoteCameraRequestOpts) (models.CameraRequest, error)
	SetCameraRequestStatus(ctx context.Context, tx db.Executor, id string, st models.CameraRequestStatus) (models.CameraRequest, error)
}

// apiTokensRepository — API-токены и журнал открытий страниц номеров (реализация — repository.New()).
type apiTokensRepository interface {
	CreateApiToken(ctx context.Context, tx db.Executor, o models.CreateApiTokenOpts) (models.ApiToken, error)
	ListApiTokens(ctx context.Context, tx db.Executor, o models.ListApiTokensOpts) (models.ApiTokensPage, error)
	RevokeApiToken(ctx context.Context, tx db.Executor, id string) (models.ApiToken, error)
	FindApiToken(ctx context.Context, tx db.Executor, id, sigHash string) (models.ApiToken, error)
	ConsumeApiToken(ctx context.Context, tx db.Executor, id string) (models.ApiToken, bool, error)
	CreatePlateView(ctx context.Context, tx db.Executor, v models.NewPlateView) error
	ListPlateViews(ctx context.Context, tx db.Executor, o models.ListPlateViewsOpts) (models.PlateViewsPage, error)
	ListSamePlate(ctx context.Context, tx db.Executor, o models.SamePlateOpts) ([]models.PlateSighting, error)
	ListPlateClarifications(ctx context.Context, tx db.Executor, o models.SamePlateOpts) ([]models.PlateClarification, error)
}

// ingestRepository — запись номеров и фиксаций для rpc Ingest (реализация — repository.New()).
type ingestRepository interface {
	UpsertPlate(ctx context.Context, tx db.Executor, o models.UpsertPlateOpts) (string, error)
	CreateSighting(ctx context.Context, tx db.Executor, o models.CreateSightingOpts) (models.PlateSighting, error)
}

// recognizer — распознавание номеров на фото (реализация — ocr.Client, HTTP к сервису ocr/).
// Номера — как прочитала модель (латиница, цифры); ошибка связи — models.ErrRecognizerUnavailable.
type recognizer interface {
	Recognize(ctx context.Context, photo []byte) ([]models.PlateCandidate, error)
}

// photoStorage — хранилище фото (реализация — storage.S3).
type photoStorage interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
}
