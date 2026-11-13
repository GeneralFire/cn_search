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
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/implementation"
	"github.com/GeneralFire/cn_search/be/internal/repository"
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
}

func loadConfig() config {
	return config{
		HTTPPort:    envOr("PORT", "8080"),
		GRPCPort:    envOr("GRPC_PORT", "8082"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		PhotoBase:   envOr("S3_PUBLIC_URL", "/s3/cars"),
		AdminToken:  os.Getenv("ADMIN_TOKEN"),
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
		svc := implementation.NewCarSearch(uc, cfg.PhotoBase,
			implementation.WithComplaints(usecase.NewComplaints(cluster, repo), cfg.AdminToken),
			implementation.WithAdmin(usecase.NewAdmin(cluster, repo), cfg.AdminToken),
			implementation.WithAccounts(usecase.NewAccounts(cluster, repo)))
		desc := pb.NewCarSearchServiceServiceDesc(svc)

		// HTTP: ручки clay под /api + health + swagger.
		api := chi.NewRouter()
		desc.RegisterHTTP(api)
		root := chi.NewRouter()
		root.Get(apiPrefix+"/health", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
		swaggerJSON := desc.SwaggerDef(func(s *spec.Swagger) { s.BasePath = apiPrefix })
		root.Get(apiPrefix+"/swagger.json", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(swaggerJSON)
		})
		root.Mount(apiPrefix, api)

		httpSrv := &http.Server{
			Addr:              ":" + cfg.HTTPPort,
			Handler:           logRequests(log, root),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
		}

		// gRPC — тот же сервис.
		grpcSrv := grpc.NewServer()
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
