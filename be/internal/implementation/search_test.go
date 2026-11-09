package implementation

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch/v1"
)

func TestSearch(t *testing.T) {
	s := newSvc()
	ctx := context.Background()

	resp, err := s.Search(ctx, &pb.SearchRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetTotal() != 600 || len(resp.GetItems()) != DefaultLimit || resp.NextOffset == nil || resp.GetNextOffset() != DefaultLimit {
		t.Fatalf("по умолчанию: total=%d items=%d next=%v", resp.GetTotal(), len(resp.GetItems()), resp.NextOffset)
	}
	first := resp.GetItems()[0]
	if first.GetPhotoUrl() != "/s3/cars/car-1.png" {
		t.Errorf("photo_url = %q", first.GetPhotoUrl())
	}
	if first.GetCreatedAt() == nil || first.GetCreatedAt().AsTime().IsZero() {
		t.Errorf("created_at пуст: %v", first)
	}

	resp, err = s.Search(ctx, &pb.SearchRequest{Plate: "А123ВС 777"})
	if err != nil || resp.GetTotal() != 1 || resp.GetItems()[0].GetId() != 1 || resp.NextOffset != nil {
		t.Fatalf("номер: %v, %v", resp, err)
	}

	resp, err = s.Search(ctx, &pb.SearchRequest{Offset: 595, Limit: 10})
	if err != nil || len(resp.GetItems()) != 5 || resp.NextOffset != nil {
		t.Fatalf("последняя страница: items=%d next=%v err=%v", len(resp.GetItems()), resp.NextOffset, err)
	}
}

func TestSearchValidation(t *testing.T) {
	s := newSvc()
	for name, req := range map[string]*pb.SearchRequest{
		"limit > max":   {Limit: MaxLimit + 1},
		"limit < 0":     {Limit: -1},
		"offset < 0":    {Offset: -1},
		"длинный номер": {Plate: "АААААААААААААААААААААААААААААААААААААА"},
	} {
		if _, err := s.Search(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: code = %v, want InvalidArgument", name, status.Code(err))
		}
	}
}
