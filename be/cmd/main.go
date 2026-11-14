// be — сервис поиска фиксаций госномеров: HTTP (ручки clay) + gRPC.
//
// Схема БД — миграции goose (be/migrations), тестовые данные — be/internal/test_utils;
// и то и другое применяется снаружи (docker-compose, Makefile), не этим бинарником.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-openapi/spec"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/utrack/clay/v3/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/GeneralFire/cn_search/be/internal/apitoken"
	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/health"
	"github.com/GeneralFire/cn_search/be/internal/implementation"
	"github.com/GeneralFire/cn_search/be/internal/ocr"
	"github.com/GeneralFire/cn_search/be/internal/ratelimit"
	"github.com/GeneralFire/cn_search/be/internal/repository"
	"github.com/GeneralFire/cn_search/be/internal/storage"
	"github.com/GeneralFire/cn_search/be/internal/usecase"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// apiPrefix — под этим префиксом смонтированы ручки clay: /api/carsearch.CarSearchService/Search.
const apiPrefix = "/api"

type config struct {
	HTTPPort    string
	GRPCPort    string
	DatabaseURL string
	PhotoBase   string // публичный URL бакета с картинками для браузера
	AdminToken  string // пароль админ-панели (заголовок admin-token); пусто — админские методы выключены
	SigningKey  string // TOKEN_SIGNING_KEY — ключ HMAC-подписи API-токенов; пусто — токены не проходят проверку

	// rpc Ingest (фото → OCR → фиксация): включён, если заданы OCR_URL и S3_ENDPOINT.
	OCRURL      string // сервис распознавания номеров (ocr/), напр. http://ocr:8000
	S3Endpoint  string // host:port S3 без схемы, напр. s3:9000
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3UseSSL    bool
}

func loadConfig() config {
	return config{
		HTTPPort:    envOr("PORT", "8080"),
		GRPCPort:    envOr("GRPC_PORT", "8082"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		PhotoBase:   envOr("S3_PUBLIC_URL", "/s3/cars"),
		AdminToken:  os.Getenv("ADMIN_TOKEN"),
		SigningKey:  os.Getenv("TOKEN_SIGNING_KEY"),

		OCRURL:      os.Getenv("OCR_URL"),
		S3Endpoint:  os.Getenv("S3_ENDPOINT"),
		S3AccessKey: os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey: os.Getenv("S3_SECRET_KEY"),
		S3Bucket:    envOr("S3_BUCKET", "cars"),
		S3UseSSL:    os.Getenv("S3_USE_SSL") == "true",
	}
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := serve(ctx, log, loadConfig()); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func withPool(ctx context.Context, cfg config, fn func(*pgxpool.Pool) error) error {
	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL не задан")
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("db ping: %w", err)
	}
	return fn(pool)
}

// ---------------------------------------------------------------- serve

func serve(ctx context.Context, log *slog.Logger, cfg config) error {
	return withPool(ctx, cfg, func(pool *pgxpool.Pool) error {
		// implementation (RPC) → usecase (транзакции) → repository (SQL) → db (pgx).
		cluster, repo := db.NewCluster(pool), repository.New()
		uc := usecase.New(cluster, repo)
		if cfg.AdminToken == "" {
			log.Warn("ADMIN_TOKEN не задан — админ-панель выключена")
		}
		opts := []implementation.Option{
			implementation.WithComplaints(usecase.NewComplaints(cluster, repo), cfg.AdminToken),
			implementation.WithAdmin(usecase.NewAdmin(cluster, repo), cfg.AdminToken),
			implementation.WithAccounts(usecase.NewAccounts(cluster, repo)),
			implementation.WithCameraRequests(usecase.NewCameraRequests(cluster, repo)),
			implementation.WithApiTokens(usecase.NewApiTokens(cluster, repo, apitoken.NewSigner(cfg.SigningKey))),
		}
		// Готовность (/api/ready): БД и, если настроен, бакет S3.
		ready := health.New(health.DefaultTimeout).With("db", pool)
		if cfg.OCRURL != "" && cfg.S3Endpoint != "" {
			photos, err := storage.NewS3(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3UseSSL)
			if err != nil {
				return err
			}
			ready.With("s3", photos)
			opts = append(opts, implementation.WithIngest(usecase.NewIngest(cluster, repo, ocr.New(cfg.OCRURL), photos)))
		} else {
			log.Warn("OCR_URL или S3_ENDPOINT не заданы — загрузка фото (rpc Ingest) выключена")
		}
		svc := implementation.NewCarSearch(uc, cfg.PhotoBase, opts...)
		if cfg.SigningKey == "" {
			log.Warn("TOKEN_SIGNING_KEY не задан — API-токены не проходят проверку")
		}
		desc := pb.NewCarSearchServiceServiceDesc(svc)

		// Ограничение частоты пишущих ручек — один интерсептор на оба транспорта: clay вызывает его
		// из HTTP-ручек (HTTP-заголовки уже в metadata, retry-after уходит в заголовок ответа).
		limit := ratelimit.Interceptor(ratelimit.New(), ratelimit.Limits, implementation.RateLimitKey, svc.IsAdmin)
		desc.Apply(transport.WithUnaryInterceptor(limit))

		// HTTP: ручки clay под /api + health + swagger.
		api := chi.NewRouter()
		desc.RegisterHTTP(api)
		root := chi.NewRouter()
		root.Get(apiPrefix+"/health", health.Live)
		root.Get(apiPrefix+"/ready", ready.Ready)
		swaggerJSON := desc.SwaggerDef(func(s *spec.Swagger) { s.BasePath = apiPrefix })
		root.Get(apiPrefix+"/swagger.json", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(swaggerJSON)
		})
		root.Mount(apiPrefix, implementation.WithPeerFromRequest(api))

		httpSrv := &http.Server{
			Addr:              ":" + cfg.HTTPPort,
			Handler:           logRequests(log, root),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       60 * time.Second, // загрузка фото (rpc Ingest) с телефона
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
		}

		// gRPC — тот же сервис.
		// Фото в rpc Ingest — до 8 МБ: стандартного предела gRPC (4 МБ) не хватает.
		grpcSrv := grpc.NewServer(grpc.MaxRecvMsgSize(implementation.MaxPhotoBytes+1<<20), grpc.UnaryInterceptor(limit))
		desc.RegisterGRPC(grpcSrv)
		reflection.Register(grpcSrv)
		grpcLis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
		if err != nil {
			return fmt.Errorf("grpc listen: %w", err)
		}

		errc := make(chan error, 2)
		go func() {
			log.Info("http listening", "addr", httpSrv.Addr)
			errc <- httpSrv.ListenAndServe()
		}()
		go func() {
			log.Info("grpc listening", "addr", grpcLis.Addr().String())
			errc <- grpcSrv.Serve(grpcLis)
		}()

		select {
		case err := <-errc:
			return err
		case <-ctx.Done():
		}

		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		grpcSrv.GracefulStop()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}
		return nil
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func logRequests(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Info("request", "method", r.Method, "uri", r.URL.RequestURI(), "status", rec.status,
			"duration_ms", time.Since(start).Milliseconds())
	})
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
