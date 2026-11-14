package ratelimit

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/utrack/clay/v3/transport/httptransport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// fakeClock — ручные часы.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time            { return c.t }
func (c *fakeClock) add(d time.Duration)       { c.t = c.t.Add(d) }
func newClock() *fakeClock                     { return &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)} }
func ok(_ context.Context, _ any) (any, error) { return "ok", nil }

func TestAllow_BurstThenRefill(t *testing.T) {
	c := newClock()
	lim := NewWithClock(c.now)
	l := Limit{10, time.Minute}
	for i := range 10 {
		allowed, _ := lim.Allow("k", l)
		if !allowed {
			t.Fatalf("запрос %d: ожидался пропуск", i+1)
		}
	}
	allowed, wait := lim.Allow("k", l)
	if allowed {
		t.Fatal("ожидался отказ")
	}
	eq(t, 6*time.Second, wait) // 10 в минуту — токен каждые 6 с

	c.add(5 * time.Second)
	allowed, wait = lim.Allow("k", l)
	if allowed {
		t.Fatal("ожидался отказ")
	}
	eq(t, time.Second, wait)

	c.add(time.Second)
	allowed, _ = lim.Allow("k", l)
	if !allowed {
		t.Fatal("ожидался пропуск")
	}

	// Другой ключ — своё ведро.
	allowed, _ = lim.Allow("other", l)
	if !allowed {
		t.Fatal("ожидался пропуск")
	}
}

func TestAllow_RefillCappedAtCapacity(t *testing.T) {
	c := newClock()
	lim := NewWithClock(c.now)
	l := Limit{2, time.Minute}
	lim.Allow("k", l)
	c.add(time.Hour)
	for range 2 {
		allowed, _ := lim.Allow("k", l)
		if !allowed {
			t.Fatal("ожидался пропуск")
		}
	}
	allowed, _ := lim.Allow("k", l)
	if allowed {
		t.Fatal("ожидался отказ")
	}
}

func TestCleanup_RemovesIdleBuckets(t *testing.T) {
	c := newClock()
	lim := NewWithClock(c.now)
	lim.Allow("old", Limit{1, time.Minute})
	lim.Allow("long", Limit{1, time.Hour})
	c.add(2 * time.Minute)
	lim.Allow("new", Limit{1, time.Minute}) // запускает чистку
	eq(t, 2, lim.size())                    // "old" выметен, "long" ещё не наполнился
	_, found := lim.buckets.Load("old")
	if found {
		t.Fatal("ожидался отказ")
	}
}

func TestInterceptor(t *testing.T) {
	c := newClock()
	lim := NewWithClock(c.now)
	client := "a"
	admin := false
	icpt := Interceptor(lim, Limits,
		func(context.Context) string { return client },
		func(context.Context) bool { return admin })
	clarify := &grpc.UnaryServerInfo{FullMethod: pb.CarSearchService_Clarify_FullMethodName}

	for i := range ClarifyPerMinute {
		_, err := icpt(context.Background(), nil, clarify, ok)
		if err != nil {
			t.Fatalf("запрос %d: %v", i+1, err)
		}
	}

	// 11-й Clarify за минуту: ResourceExhausted и retry-after — по HTTP (транспорт clay) в заголовок.
	rec := httptest.NewRecorder()
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), httptransport.NewTStream(rec))
	_, err := icpt(ctx, nil, clarify, ok)
	eq(t, codes.ResourceExhausted, status.Code(err))
	eq(t, "слишком много запросов, попробуйте через 6 с", status.Convert(err).Message())
	eq(t, "6", rec.Header().Get("Retry-After"))

	// Другой клиент не задет.
	client = "b"
	_, err = icpt(context.Background(), nil, clarify, ok)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Админ не ограничивается.
	client, admin = "a", true
	_, err = icpt(context.Background(), nil, clarify, ok)
	if err != nil {
		t.Fatalf("%v", err)
	}
	admin = false

	// Методы без лимита не ограничиваются.
	search := &grpc.UnaryServerInfo{FullMethod: pb.CarSearchService_Search_FullMethodName}
	for range 100 {
		_, err = icpt(context.Background(), nil, search, ok)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}

	// У каждого метода своё ведро: Complain клиента "a" не тратит Clarify.
	complain := &grpc.UnaryServerInfo{FullMethod: pb.CarSearchService_Complain_FullMethodName}
	_, err = icpt(context.Background(), nil, complain, ok)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Через 6 с у "a" снова есть токен.
	c.add(6 * time.Second)
	_, err = icpt(context.Background(), nil, clarify, ok)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func eq[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}
