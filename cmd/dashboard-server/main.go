// Package main serves as the entry point for the GoBoard service.
// It utilizes the Uber Fx framework for Dependency Injection (DI) and lifecycle management,
// orchestrating the initialization of Redis, Business Logic, and transport layers (gRPC/HTTP).
package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jassus213/go-board/dashboard/auth"
	"github.com/jassus213/go-board/dashboard/core/usecase"
	"github.com/jassus213/go-board/dashboard/delivery/grpc"
	pb "github.com/jassus213/go-board/dashboard/delivery/grpc/gen"
	"github.com/jassus213/go-board/dashboard/delivery/rest"
	"github.com/jassus213/go-board/dashboard/delivery/ws"
	"github.com/jassus213/go-board/dashboard/repo/redis"

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
			newUseCase,
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
	// CORSAllowedOrigins is a comma-separated list of allowed origins for WebSocket connections.
	CORSAllowedOrigins string
	// CORSAllowCredentials indicates whether credentials (cookies, auth headers) are allowed.
	CORSAllowCredentials bool
	// EnableWebSocket toggles WebSocket endpoint and hub runtime.
	EnableWebSocket bool
	// EnableGRPC toggles gRPC server startup.
	EnableGRPC bool
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

// newUseCase creates a usecase.
func newUseCase(repo *redis.DashboardRedisRepository) *usecase.BoardUseCase {
	return usecase.New(repo)
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
func runWSHub(cfg Config, hub *ws.Hub, logger *zap.Logger) {
	if !cfg.EnableWebSocket {
		logger.Info("WebSocket mode is disabled")
		return
	}
	go hub.Run()
}

// runHTTPServer configures and starts the HTTP server, including WebSocket and health check endpoints.
// It registers OnStart and OnStop hooks to ensure the server starts after DI and shuts down gracefully.
func runHTTPServer(lc fx.Lifecycle, cfg Config, hub *ws.Hub, uc *usecase.BoardUseCase, verifier *auth.StaticVerifier, logger *zap.Logger) {
	router := rest.NewRouter(&rest.Params{
		Hub:      hub,
		UseCase:  uc,
		Verifier: verifier,
		Config: rest.Config{
			EnableWebSocket:      cfg.EnableWebSocket,
			CORSAllowedOrigins:   parseCORSOrigins(cfg.CORSAllowedOrigins),
			CORSAllowCredentials: cfg.CORSAllowCredentials,
		},
	})

	srv := &http.Server{
		Addr:              cfg.HttpPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("HTTP server starting", zap.String("port", cfg.HttpPort))
			go func() {
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("HTTP server stopped unexpectedly", zap.Error(err))
				}
			}()
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
func runGRPCServer(lc fx.Lifecycle, cfg Config, uc *usecase.BoardUseCase, verifier *auth.StaticVerifier, logger *zap.Logger) {
	if !cfg.EnableGRPC {
		logger.Info("gRPC mode is disabled")
		return
	}

	opts := []googlegrpc.ServerOption{
		googlegrpc.StreamInterceptor(grpc.AuthInterceptor(verifier)),
		googlegrpc.UnaryInterceptor(grpc.AuthUnaryInterceptor(verifier)),
	}

	grpcServer := googlegrpc.NewServer(opts...)
	dashboardService := grpc.NewServer(uc)
	pb.RegisterDashboardServiceServer(grpcServer, dashboardService)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			lis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.GrpcPort)
			if err != nil {
				return err
			}
			logger.Info("gRPC server starting", zap.String("port", cfg.GrpcPort))
			go func() {
				if err := grpcServer.Serve(lis); err != nil {
					logger.Error("gRPC server stopped unexpectedly", zap.Error(err))
				}
			}()
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

	allowCredentials := false
	if credStr := getEnv("CORS_ALLOW_CREDENTIALS", "false"); credStr == "true" || credStr == "1" {
		allowCredentials = true
	}

	enableWebSocket := getBoolEnv("ENABLE_WEBSOCKET", true)
	enableGRPC := getBoolEnv("ENABLE_GRPC", true)
	enableWebSocket, enableGRPC = applyRuntimeModeFlags(enableWebSocket, enableGRPC, os.Args[1:])

	return Config{
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:            getEnv("REDIS_PASS", ""),
		HttpPort:             getEnv("HTTP_PORT", ":8080"),
		GrpcPort:             getEnv("GRPC_PORT", ":50051"),
		AuthSecret:           getEnv("AUTH_SECRET", "super-secret-key"),
		CORSAllowedOrigins:   getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
		CORSAllowCredentials: allowCredentials,
		EnableWebSocket:      enableWebSocket,
		EnableGRPC:           enableGRPC,
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

func getBoolEnv(key string, fallback bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func applyRuntimeModeFlags(enableWS, enableGRPC bool, args []string) (bool, bool) {
	for _, arg := range args {
		switch arg {
		case "--ws", "--enable-ws":
			enableWS = true
		case "--no-ws", "--disable-ws":
			enableWS = false
		case "--grpc", "--enable-grpc":
			enableGRPC = true
		case "--no-grpc", "--disable-grpc":
			enableGRPC = false
		}
	}

	return enableWS, enableGRPC
}

// parseCORSOrigins parses a comma-separated list of allowed origins.
// Returns a slice of origin strings with whitespace trimmed.
func parseCORSOrigins(origins string) []string {
	if origins == "" {
		return []string{}
	}

	parts := strings.Split(origins, ",")
	result := make([]string, 0, len(parts))

	for _, origin := range parts {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
