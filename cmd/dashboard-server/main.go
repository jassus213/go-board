// Package main serves as the entry point for the GoBoard service.
// It utilizes the Uber Fx framework for Dependency Injection (DI) and lifecycle management,
// orchestrating the initialization of Redis, Business Logic, and transport layers (gRPC/HTTP).
package main

import (
	"context"
	"net"
	"net/http"
	"os"

	"github.com/jassus213/go-board/dashboard/auth"
	"github.com/jassus213/go-board/dashboard/dal/redis"
	"github.com/jassus213/go-board/dashboard/delivery/grpc"
	pb "github.com/jassus213/go-board/dashboard/delivery/grpc/gen"
	"github.com/jassus213/go-board/dashboard/delivery/ws"

	"github.com/joho/godotenv"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
)

// main initializes and runs the Fx application.
// It defines the providers for dependency injection and the invokers
// to start the long-running services.
func main() {
	fx.New(
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),

		fx.Provide(
			loadConfig,
			newLogger,
			newRedisClient,
			newRepository,
			newAuthVerifier,
			newWSHub,
		),

		fx.Invoke(
			runWSHub,
			runGRPCServer,
			runHTTPServer,
		),
	).Run()
}

// Config holds the application configuration parameters loaded from environment variables.
type Config struct {
	// RedisAddr is the host:port address of the Redis instance.
	RedisAddr string
	// RedisPass is the password for the Redis instance (if any).
	RedisPass string
	// HttpPort is the port used for the HTTP and WebSocket server (e.g., ":8080").
	HttpPort string
	// GrpcPort is the port used for the gRPC server (e.g., ":50051").
	GrpcPort string
	// AuthSecret is a shared secret used by the StaticVerifier for simple token validation.
	AuthSecret string
}

// --- Providers (Dependency Injection) ---

func newLogger() (*zap.Logger, error) {
	return zap.NewProduction()
}

// newRedisClient initializes the Redis client and registers a graceful shutdown hook.
func newRedisClient(lc fx.Lifecycle, cfg Config, logger *zap.Logger) *goredis.Client {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
	})

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("Closing Redis connection")
			return rdb.Close()
		},
	})
	return rdb
}

// newRepository creates a new Redis-based dashboard repository.
func newRepository(rdb *goredis.Client) *redis.DashboardRedisRepository {
	return redis.NewDashboardRedisRepository(rdb, "prod:")
}

// newAuthVerifier creates a static token verifier for authentication.
func newAuthVerifier(cfg Config) *auth.StaticVerifier {
	return &auth.StaticVerifier{Secret: cfg.AuthSecret}
}

// newWSHub creates a new WebSocket Hub to manage active connections.
func newWSHub() *ws.Hub {
	return ws.NewHub()
}

// --- Invokers (Lifecycle Hooks & Startup) ---

// runWSHub starts the internal WebSocket hub processing loop in a background goroutine.
func runWSHub(hub *ws.Hub) {
	go hub.Run()
}

// runHTTPServer configures and starts the HTTP server, including WebSocket and health check endpoints.
// It registers OnStart and OnStop hooks to ensure the server starts after DI and shuts down gracefully.
func runHTTPServer(lc fx.Lifecycle, cfg Config, hub *ws.Hub, repo *redis.DashboardRedisRepository, verifier *auth.StaticVerifier, logger *zap.Logger) {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(hub, repo, verifier, w, r)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr:    cfg.HttpPort,
		Handler: mux,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("HTTP server starting", zap.String("port", cfg.HttpPort))
			go srv.ListenAndServe()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping HTTP server")
			return srv.Shutdown(ctx)
		},
	})
}

// runGRPCServer configures and starts the gRPC server with the AuthInterceptor enabled.
// It handles the registration of the DashboardService and registers lifecycle hooks for graceful startup/shutdown.
func runGRPCServer(lc fx.Lifecycle, cfg Config, repo *redis.DashboardRedisRepository, verifier *auth.StaticVerifier, logger *zap.Logger) {
	opts := []googlegrpc.ServerOption{
		googlegrpc.StreamInterceptor(grpc.AuthInterceptor(verifier)),
	}

	grpcServer := googlegrpc.NewServer(opts...)
	dashboardService := grpc.NewServer(repo)
	pb.RegisterDashboardServiceServer(grpcServer, dashboardService)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			lis, err := net.Listen("tcp", cfg.GrpcPort)
			if err != nil {
				return err
			}
			logger.Info("gRPC server starting", zap.String("port", cfg.GrpcPort))
			go grpcServer.Serve(lis)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping gRPC server")
			grpcServer.GracefulStop()
			return nil
		},
	})
}

// --- Configuration Utilities ---

// loadConfig reads configuration from environment variables or a .env file.
// It returns a Config struct with default values as fallbacks.
func loadConfig() Config {
	_ = godotenv.Load()
	return Config{
		RedisAddr:  getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:  getEnv("REDIS_PASS", ""),
		HttpPort:   getEnv("HTTP_PORT", ":8080"),
		GrpcPort:   getEnv("GRPC_PORT", ":50051"),
		AuthSecret: getEnv("AUTH_SECRET", "super-secret-key"),
	}
}

// getEnv retrieves the value of the environment variable named by the key.
// If the variable is not present, it returns the fallback value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
