package health

//go:generate minimock -i pinger -o ./mocks/pinger_mock.go -n PingerMock -p mocks

import "context"

// pinger — проверяемая зависимость (БД: *pgxpool.Pool; позже — S3 через HeadBucket).
type pinger interface {
	Ping(ctx context.Context) error
}
