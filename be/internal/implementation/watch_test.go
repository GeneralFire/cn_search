package implementation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/implementation/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

const watchPlateStr = "А123ВС 777"

func newWatchSvc(t *testing.T) (*CarSearch, *mocks.AccountsMock, *mocks.WatchesMock) {
	acc, w := mocks.NewAccountsMock(t), mocks.NewWatchesMock(t)
	return NewCarSearch(mocks.NewPlateSightingsMock(t), "/s3", WithAccounts(acc), WithWatches(w)), acc, w
}

func TestWatchRequiresUser(t *testing.T) {
	svc, _, _ := newWatchSvc(t)
	ctx := context.Background() // без токена
	if _, err := svc.Watch(ctx, &pb.WatchRequest{Plate: watchPlateStr}); status.Code(err) != codes.Unauthenticated {
		t.Errorf("Watch: %v", err)
	}
	if _, err := svc.Unwatch(ctx, &pb.UnwatchRequest{Plate: watchPlateStr}); status.Code(err) != codes.Unauthenticated {
		t.Errorf("Unwatch: %v", err)
	}
	if _, err := svc.ListWatches(ctx, &pb.ListWatchesRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Errorf("ListWatches: %v", err)
	}
	if _, err := svc.MarkWatchSeen(ctx, &pb.MarkWatchSeenRequest{Plate: watchPlateStr}); status.Code(err) != codes.Unauthenticated {
		t.Errorf("MarkWatchSeen: %v", err)
	}
}

func TestWatchOff(t *testing.T) {
	svc := NewCarSearch(mocks.NewPlateSightingsMock(t), "/s3")
	if _, err := svc.ListWatches(context.Background(), &pb.ListWatchesRequest{}); status.Code(err) != codes.Unimplemented {
		t.Errorf("подписки выключены: %v", err)
	}
}

func TestWatchAndUnwatch(t *testing.T) {
	svc, acc, w := newWatchSvc(t)
	acc.UserByTokenMock.Return(demoUser, nil)
	ctx := withMD("cookie", "cs_token=t")

	for _, bad := range []string{"", "  ", "---", strings.Repeat("A", maxWatchPlate+1)} {
		if _, err := svc.Watch(ctx, &pb.WatchRequest{Plate: bad}); status.Code(err) != codes.InvalidArgument {
			t.Errorf("номер %q: %v", bad, err)
		}
	}
	w.WatchMock.Expect(minimock.AnyContext, "u1", watchPlateStr).Return(nil)
	if _, err := svc.Watch(ctx, &pb.WatchRequest{Plate: watchPlateStr}); err != nil {
		t.Errorf("Watch: %v", err)
	}
	w.UnwatchMock.Expect(minimock.AnyContext, "u1", watchPlateStr).Return(nil)
	if _, err := svc.Unwatch(ctx, &pb.UnwatchRequest{Plate: watchPlateStr}); err != nil {
		t.Errorf("Unwatch: %v", err)
	}
	if _, err := svc.Unwatch(ctx, &pb.UnwatchRequest{Plate: " "}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("Unwatch без номера: %v", err)
	}
}

func TestListWatches(t *testing.T) {
	svc, acc, w := newWatchSvc(t)
	acc.UserByTokenMock.Return(demoUser, nil)
	at := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	w.ListMock.Expect(minimock.AnyContext, models.ListWatchesOpts{UserID: "u1", IncludeTestData: true}).
		Return([]models.Watch{
			{PlateDisplay: watchPlateStr, Latest: &models.PlateSighting{ID: "s1", Plate: watchPlateStr, ThumbKey: "t.jpg", CreatedAt: at}, NewCount: 3, WatchedAt: at},
			{PlateDisplay: "К777КК 77", WatchedAt: at}, // номера ещё нет
		}, nil)

	resp, err := svc.ListWatches(withMD("cookie", "cs_token=t", "include-test-data", "true"), &pb.ListWatchesRequest{})
	if err != nil || len(resp.Items) != 2 {
		t.Fatalf("ListWatches: %v %v", resp, err)
	}
	it := resp.Items[0]
	if it.NewCount != 3 || it.Plate.GetThumbUrl() != "/s3/t.jpg" || it.WatchedAt == nil || it.PlateDisplay != watchPlateStr {
		t.Errorf("элемент: %v", it)
	}
	if wait := resp.Items[1]; wait.Plate != nil || wait.PlateDisplay != "К777КК 77" {
		t.Errorf("ожидание: %v", wait)
	}
}

func TestMarkWatchSeen(t *testing.T) {
	svc, acc, w := newWatchSvc(t)
	acc.UserByTokenMock.Return(demoUser, nil)
	ctx := withMD("authorization", "Bearer t")
	w.MarkSeenMock.Expect(minimock.AnyContext, "u1", watchPlateStr).Return(nil)
	if _, err := svc.MarkWatchSeen(ctx, &pb.MarkWatchSeenRequest{Plate: watchPlateStr}); err != nil {
		t.Errorf("MarkWatchSeen: %v", err)
	}
	w.MarkSeenMock.Return(models.ErrNotFound)
	if _, err := svc.MarkWatchSeen(ctx, &pb.MarkWatchSeenRequest{Plate: watchPlateStr}); status.Code(err) != codes.NotFound {
		t.Errorf("нет подписки: %v", err)
	}
}
