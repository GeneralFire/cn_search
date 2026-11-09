package service

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/cars"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch/v1"
)

func newSvc() *CarSearch {
	all := cars.Generate(cars.DefaultGenerateOptions(func(id int) string { return "car-1.png" }))
	return NewCarSearch(cars.NewMemoryStore(all), "/s3/cars/")
}

func TestSearchCars(t *testing.T) {
	s := newSvc()
	ctx := context.Background()

	resp, err := s.SearchCars(ctx, &pb.SearchCarsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetTotal() != 600 || len(resp.GetItems()) != DefaultLimit || resp.NextOffset == nil || resp.GetNextOffset() != DefaultLimit {
		t.Fatalf("по умолчанию: total=%d items=%d next=%v", resp.GetTotal(), len(resp.GetItems()), resp.NextOffset)
	}
	if got := resp.GetItems()[0].GetPhotoUrl(); got != "/s3/cars/car-1.png" {
		t.Errorf("photo_url = %q", got)
	}

	resp, err = s.SearchCars(ctx, &pb.SearchCarsRequest{Plate: "a123bc", Key: "0451"})
	if err != nil || resp.GetTotal() != 1 || resp.GetItems()[0].GetId() != 1 || resp.NextOffset != nil {
		t.Fatalf("номер И ключ: %v, %v", resp, err)
	}

	resp, err = s.SearchCars(ctx, &pb.SearchCarsRequest{Offset: 595, Limit: 10})
	if err != nil || len(resp.GetItems()) != 5 || resp.NextOffset != nil {
		t.Fatalf("последняя страница: items=%d next=%v err=%v", len(resp.GetItems()), resp.NextOffset, err)
	}
}

func TestSearchCarsValidation(t *testing.T) {
	s := newSvc()
	for name, req := range map[string]*pb.SearchCarsRequest{
		"limit > max":   {Limit: MaxLimit + 1},
		"limit < 0":     {Limit: -1},
		"offset < 0":    {Offset: -1},
		"длинный номер": {Plate: "АААААААААААААААААААААААААААААААААААААА"},
		"длинный ключ":  {Key: "KKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKK"},
	} {
		if _, err := s.SearchCars(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: code = %v, want InvalidArgument", name, status.Code(err))
		}
	}
}

func TestGetCar(t *testing.T) {
	s := newSvc()
	ctx := context.Background()

	resp, err := s.GetCar(ctx, &pb.GetCarRequest{Id: 2})
	if err != nil || resp.GetCar().GetPlate() != "М456ОР 178" {
		t.Fatalf("GetCar(2) = %v, %v", resp, err)
	}
	if _, err := s.GetCar(ctx, &pb.GetCarRequest{Id: 100500}); status.Code(err) != codes.NotFound {
		t.Errorf("несуществующий: %v", status.Code(err))
	}
	if _, err := s.GetCar(ctx, &pb.GetCarRequest{Id: 0}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("id=0: %v", status.Code(err))
	}
}
