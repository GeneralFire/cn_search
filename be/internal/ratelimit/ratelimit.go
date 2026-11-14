// Package ratelimit — ограничение частоты пишущих ручек по клиенту: token bucket в памяти процесса.
//
// Подключается одним gRPC unary interceptor (Interceptor) к обоим транспортам: grpc.Server
// (grpc.UnaryInterceptor) и HTTP-ручкам clay (desc.Apply(transport.WithUnaryInterceptor)) —
// сгенерированные clay ручки вызывают тот же интерсептор с HTTP-заголовками в incoming metadata.
package ratelimit

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Limit — не больше Requests запросов за Per (ёмкость ведра Requests, пополнение равномерное).
type Limit struct {
	Requests int
	Per      time.Duration
}

// rate — пополнение ведра, токенов в секунду.
func (l Limit) rate() float64 { return float64(l.Requests) / l.Per.Seconds() }

// bucket — ведро одного клиента для одного метода.
type bucket struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
	limit  Limit
}

// epsilon — допуск на погрешность float при пополнении (5/6 + 1/6 может оказаться < 1).
const epsilon = 1e-9

// cleanupEvery — как часто (не чаще) Allow выметает ведра, простоявшие дольше полного пополнения.
const cleanupEvery = time.Minute

// Limiter — набор вёдер по ключу (метод + клиент). Потокобезопасен.
type Limiter struct {
	buckets sync.Map // string → *bucket
	now     func() time.Time

	cleanMu   sync.Mutex
	lastClean time.Time
}

// New — Limiter на системных часах.
func New() *Limiter { return NewWithClock(time.Now) }

// NewWithClock — Limiter на заданных часах (тесты).
func NewWithClock(now func() time.Time) *Limiter {
	return &Limiter{now: now, lastClean: now()}
}

// Allow списывает токен из ведра key с лимитом l. false — ведро пусто; retryAfter — через сколько
// появится следующий токен.
func (lim *Limiter) Allow(key string, l Limit) (ok bool, retryAfter time.Duration) {
	now := lim.now()
	lim.cleanup(now)

	v, _ := lim.buckets.LoadOrStore(key, &bucket{tokens: float64(l.Requests), last: now, limit: l})
	b := v.(*bucket)
	b.mu.Lock()
	defer b.mu.Unlock()

	if el := now.Sub(b.last).Seconds(); el > 0 {
		b.tokens = math.Min(float64(l.Requests), b.tokens+el*l.rate())
	}
	b.last = now
	if b.tokens >= 1-epsilon {
		b.tokens = math.Max(0, b.tokens-1)
		return true, 0
	}
	wait := (1 - b.tokens) / l.rate()
	return false, time.Duration(wait * float64(time.Second))
}

// cleanup удаляет ведра, которые за время простоя успели бы наполниться целиком: их состояние
// не отличается от нового ведра, а память без чистки росла бы с каждым новым IP.
func (lim *Limiter) cleanup(now time.Time) {
	lim.cleanMu.Lock()
	if now.Sub(lim.lastClean) < cleanupEvery {
		lim.cleanMu.Unlock()
		return
	}
	lim.lastClean = now
	lim.cleanMu.Unlock()

	lim.buckets.Range(func(k, v any) bool {
		b := v.(*bucket)
		b.mu.Lock()
		idle := now.Sub(b.last) >= b.limit.Per
		b.mu.Unlock()
		if idle {
			lim.buckets.Delete(k)
		}
		return true
	})
}

// size — число живых вёдер (тесты).
func (lim *Limiter) size() int {
	n := 0
	lim.buckets.Range(func(_, _ any) bool { n++; return true })
	return n
}

// KeyFunc — ключ клиента из контекста запроса (пользователь или IP).
type KeyFunc func(ctx context.Context) string

// ExemptFunc — true, если запрос не ограничивается (админ).
type ExemptFunc func(ctx context.Context) bool

// Interceptor — unary interceptor: методы из limits (полное имя, /pkg.Service/Method) ограничиваются
// по ключу key; exempt-запросы и прочие методы проходят без проверки. Превышение —
// ResourceExhausted и заголовок retry-after (clay пишет его в HTTP-ответ, gRPC — в header metadata).
func Interceptor(lim *Limiter, limits map[string]Limit, key KeyFunc, exempt ExemptFunc) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		l, ok := limits[info.FullMethod]
		if !ok || (exempt != nil && exempt(ctx)) {
			return handler(ctx, req)
		}
		allowed, wait := lim.Allow(info.FullMethod+"|"+key(ctx), l)
		if allowed {
			return handler(ctx, req)
		}
		secs := int(math.Ceil(wait.Seconds()))
		if secs < 1 {
			secs = 1
		}
		_ = grpc.SetHeader(ctx, metadata.Pairs("retry-after", fmt.Sprint(secs)))
		return nil, status.Errorf(codes.ResourceExhausted, "слишком много запросов, попробуйте через %d с", secs)
	}
}
