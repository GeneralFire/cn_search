package usecase

//go:generate minimock -i transactor -o ./mocks/transactor_mock.go -n TransactorMock -p mocks
//go:generate minimock -i repository -o ./mocks/repository_mock.go -n RepositoryMock -p mocks

import (
	"context"

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
	ListSamePlate(ctx context.Context, tx db.Executor, id string) ([]models.PlateSighting, error)
}
