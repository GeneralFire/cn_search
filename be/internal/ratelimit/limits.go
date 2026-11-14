package ratelimit

import (
	"time"

	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// Лимиты пишущих ручек на одного клиента (пользователь или IP).
const (
	ClarifyPerMinute             = 10
	ComplainPerMinute            = 10
	SubmitCameraRequestPerMinute = 5
	VoteCameraRequestPerMinute   = 30
	RegisterPerHour              = 5
	// IngestPerMinute — частота загрузок фото; суточная квота тарифа (Free 5/день) считается
	// отдельно в usecase Ingest, это ограничение сверху на всплески (в том числе для Pro).
	IngestPerMinute = 10
)

// Limits — лимиты по полному имени метода; методов, которых здесь нет, ограничение не касается.
var Limits = map[string]Limit{
	pb.CarSearchService_Clarify_FullMethodName:             {ClarifyPerMinute, time.Minute},
	pb.CarSearchService_Complain_FullMethodName:            {ComplainPerMinute, time.Minute},
	pb.CarSearchService_SubmitCameraRequest_FullMethodName: {SubmitCameraRequestPerMinute, time.Minute},
	pb.CarSearchService_VoteCameraRequest_FullMethodName:   {VoteCameraRequestPerMinute, time.Minute},
	pb.CarSearchService_Register_FullMethodName:            {RegisterPerHour, time.Hour},
	pb.CarSearchService_Ingest_FullMethodName:              {IngestPerMinute, time.Minute},
}
