// be — сервис поиска машин.
//
//	be serve    — HTTP (ручки clay) + gRPC
//	be migrate  — создать схему БД
//	be seed     — схема + тестовые данные в БД + картинки в S3
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
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-openapi/spec"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/GeneralFire/cn_search/be/assets"
	"github.com/GeneralFire/cn_search/be/internal/cars"
	"github.com/GeneralFire/cn_search/be/internal/service"
	"github.com/GeneralFire/cn_search/be/internal/storage"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch/v1"
)

// apiPrefix — под этим префиксом смонтированы ручки clay: /api/carsearch.v1.CarSearchService/SearchCars.
const apiPrefix = "/api"

type config struct {
	HTTPPort    string
	GRPCPort    string
	DatabaseURL string
	S3          storage.Config
	PhotoBase   string // публичный URL бакета для браузера
	SeedTotal   int
}

func loadConfig() config {
	total, _ := strconv.Atoi(envOr("SEED_TOTAL", "600"))
	return config{
		HTTPPort:    envOr("PORT", "8080"),
		GRPCPort:    envOr("GRPC_PORT", "8082"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		S3: storage.Config{
			Endpoint:  envOr("S3_ENDPOINT", "localhost:9000"),
			AccessKey: os.Getenv("S3_ACCESS_KEY"),
			SecretKey: os.Getenv("S3_SECRET_KEY"),
			UseSSL:    os.Getenv("S3_USE_SSL") == "true",
			Bucket:    envOr("S3_BUCKET", "cars"),
		},
		PhotoBase: envOr("S3_PUBLIC_URL", "/s3/cars"),
		SeedTotal: total,
	}
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := loadConfig()
	var err error
	switch cmd {
	case "serve":
		err = serve(ctx, log, cfg)
	case "migrate":
		err = withStore(ctx, cfg, func(s *cars.PostgresStore) error { return s.Migrate(ctx) })
	case "seed":
		err = seed(ctx, log, cfg)
	default:
		err = fmt.Errorf("неизвестная команда %q; доступны: serve, migrate, seed", cmd)
	}
	if err != nil {
		log.Error("fatal", "cmd", cmd, "err", err)
		os.Exit(1)
	}
}

func withStore(ctx context.Context, cfg config, fn func(*cars.PostgresStore) error) error {
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
	return fn(cars.NewPostgresStore(pool))
}

// ---------------------------------------------------------------- serve

func serve(ctx context.Context, log *slog.Logger, cfg config) error {
	return withStore(ctx, cfg, func(store *cars.PostgresStore) error {
		desc := pb.NewCarSearchServiceServiceDesc(service.NewCarSearch(store, cfg.PhotoBase))
		configureHTTPRuntime()

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

// ---------------------------------------------------------------- seed

func seed(ctx context.Context, log *slog.Logger, cfg config) error {
	// 1. Картинки → S3.
	s3, err := storage.NewS3(cfg.S3)
	if err != nil {
		return err
	}
	if err := s3.EnsurePublicBucket(ctx); err != nil {
		return err
	}
	for i := 1; i <= assets.PhotoCount; i++ {
		name := fmt.Sprintf("car-%d.png", i)
		data, err := assets.Images.ReadFile("cars/" + name)
		if err != nil {
			return fmt.Errorf("read asset %s: %w", name, err)
		}
		if err := s3.Put(ctx, name, data, "image/png"); err != nil {
			return err
		}
	}
	log.Info("images uploaded", "bucket", cfg.S3.Bucket, "count", assets.PhotoCount)

	// 2. Машины → Postgres.
	return withStore(ctx, cfg, func(store *cars.PostgresStore) error {
		if err := store.Migrate(ctx); err != nil {
			return err
		}
		opt := cars.DefaultGenerateOptions(func(id int) string {
			return fmt.Sprintf("car-%d.png", (id*7)%assets.PhotoCount+1)
		})
		opt.Total = cfg.SeedTotal
		all := cars.Generate(opt)
		if err := store.ReplaceAll(ctx, all); err != nil {
			return err
		}
		log.Info("cars seeded", "count", len(all))
		return nil
	})
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
