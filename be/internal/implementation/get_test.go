package implementation

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch/v1"
)

func TestGet(t *testing.T) {
	s := newSvc()
	ctx := context.Background()

	resp, err := s.Get(ctx, &pb.GetRequest{Id: 2})
	if err != nil || resp.GetCar().GetPlate() != "М456ОР 178" {
		t.Fatalf("Get(2) = %v, %v", resp, err)
	}
	if _, err := s.Get(ctx, &pb.GetRequest{Id: 100500}); status.Code(err) != codes.NotFound {
		t.Errorf("несуществующий: %v", status.Code(err))
	}
	if _, err := s.Get(ctx, &pb.GetRequest{Id: 0}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("id=0: %v", status.Code(err))
	}
}
