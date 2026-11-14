// Package logging — структурные логи с ID запроса.
//
// Interceptor — один gRPC unary interceptor на оба транспорта (как ratelimit): кладёт в контекст
// request_id и логгер, а по завершении пишет итоговую строку запроса. Из любого слоя
// logging.From(ctx) отдаёт логгер с полями request_id, method и user_id.
// Токены (cs_token, cst_, admin-token, Authorization) в логи не попадают: в поля идут только
// request_id, метод, код, длительность, id пользователя и IP.
package logging

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Header — HTTP-заголовок с ID запроса (на входе и в ответе); в gRPC metadata — в нижнем регистре.
const Header = "X-Request-Id"

var safeID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// NewRequestID — incoming, если он выглядит безопасно (до 64 символов [A-Za-z0-9-_]), иначе новый UUID.
func NewRequestID(incoming string) string {
	if safeID.MatchString(incoming) {
		return incoming
	}
	return uuid.NewString()
}

type ctxKey int

const (
	keyID ctxKey = iota
	keyLogger
	keyHTTP
)

// RequestID — ID запроса из контекста ("" — не задан).
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(keyID).(string)
	return id
}

// WithRequestID кладёт ID запроса в контекст (для тестов и фоновых задач).
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyID, id)
}

// From — логгер запроса (request_id, method, user_id); вне запроса — slog.Default().
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(keyLogger).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// HTTP — middleware: ID запроса из X-Request-Id или новый UUID. Кладёт его в заголовок запроса
// (clay переносит заголовки в metadata, так интерсептор видит тот же ID), в ответ и в контекст.
// Ответ с заголовком пишет сам middleware, поэтому интерсептор для HTTP его не дублирует.
func HTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := NewRequestID(r.Header.Get(Header))
		r.Header.Set(Header, id)
		w.Header().Set(Header, id)
		ctx := context.WithValue(r.Context(), keyHTTP, true)
		ctx = context.WithValue(ctx, keyID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Options — то, что logging не может узнать сам (знает implementation).
type Options struct {
	UserID func(ctx context.Context) string // "" — не вошёл
	IP     func(ctx context.Context) string
}

// Interceptor — unary interceptor: request_id и логгер в контекст, ID в заголовок ответа gRPC,
// итоговая строка запроса (request_id, method, code, duration_ms, user_id, ip).
func Interceptor(base *slog.Logger, o Options) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		id := RequestID(ctx)
		if id == "" {
			var in string
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				if v := md.Get("x-request-id"); len(v) > 0 {
					in = v[0]
				}
			}
			id = NewRequestID(in)
			ctx = context.WithValue(ctx, keyID, id)
		}
		if httpCall, _ := ctx.Value(keyHTTP).(bool); !httpCall {
			_ = grpc.SetHeader(ctx, metadata.Pairs("x-request-id", id))
		}

		attrs := []any{"request_id", id, "method", info.FullMethod}
		user := ""
		if o.UserID != nil {
			if user = o.UserID(ctx); user != "" {
				attrs = append(attrs, "user_id", user)
			}
		}
		ctx = context.WithValue(ctx, keyLogger, base.With(attrs...))

		resp, err := handler(ctx, req)

		code := status.Code(err)
		ip := ""
		if o.IP != nil {
			ip = o.IP(ctx)
		}
		From(ctx).Log(ctx, level(code), "rpc", "code", code.String(),
			"duration_ms", time.Since(start).Milliseconds(), "ip", ip)
		return resp, err
	}
}

// level: сбои сервера — Error, ошибки клиента — Warn, остальное — Info.
func level(c codes.Code) slog.Level {
	switch c {
	case codes.OK:
		return slog.LevelInfo
	case codes.Unknown, codes.DeadlineExceeded, codes.Internal, codes.Unavailable, codes.DataLoss:
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}

// Chain — несколько интерсепторов в одном (первый — внешний); clay принимает только один.
func Chain(ints ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		h := handler
		for i := len(ints) - 1; i >= 0; i-- {
			next, in := h, ints[i]
			h = func(ctx context.Context, req any) (any, error) { return in(ctx, req, info, next) }
		}
		return h(ctx, req)
	}
}
