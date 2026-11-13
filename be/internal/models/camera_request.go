package models

import "time"

// CameraRequestStatus — статус заявки на камеру (enum camera_request_status в БД).
type CameraRequestStatus string

const (
	CameraRequestNew       CameraRequestStatus = "new"
	CameraRequestApproved  CameraRequestStatus = "approved"
	CameraRequestRejected  CameraRequestStatus = "rejected"
	CameraRequestInstalled CameraRequestStatus = "installed"
)

// CameraRequest — заявка на расположение камеры (таблица camera_requests).
type CameraRequest struct {
	ID        string              `db:"id"`
	Lat       float64             `db:"lat"`
	Lon       float64             `db:"lon"`
	Address   string              `db:"address"`
	Reason    string              `db:"reason"`
	Contact   string              `db:"contact"` // только для администратора
	Status    CameraRequestStatus `db:"status"`
	Votes     int                 `db:"votes"`
	CreatedAt time.Time           `db:"created_at"`
}

// SubmitCameraRequestOpts — новая заявка.
type SubmitCameraRequestOpts struct {
	Lat, Lon float64
	Address  string
	Reason   string
	Contact  string
}

// GeoBox — рамка карты (градусы).
type GeoBox struct {
	MinLat, MinLon, MaxLat, MaxLon float64
}

// ListCameraRequestsOpts — страница заявок.
type ListCameraRequestsOpts struct {
	BBox            *GeoBox             // nil — без рамки
	Status          CameraRequestStatus // пусто — все
	Offset          int
	Limit           int
	IncludeTestData bool
}

// CameraRequestsPage — страница заявок.
type CameraRequestsPage struct {
	Items      []CameraRequest
	Total      int  // всего по фильтру
	NextOffset *int // nil — дальше ничего нет
}

// VoteCameraRequestOpts — голос за заявку.
type VoteCameraRequestOpts struct {
	ID              string
	IncludeTestData bool // можно голосовать за тестовую заявку (заголовок include-test-data)
}
