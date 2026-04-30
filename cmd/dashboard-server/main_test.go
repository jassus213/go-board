package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jassus213/go-board/dashboard/auth"
	"github.com/jassus213/go-board/dashboard/core/usecase"
	"github.com/jassus213/go-board/dashboard/repo/mocks"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func TestGetEnv(t *testing.T) {
	const key = "UNIT_TEST_GET_ENV_KEY"

	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset env: %v", err)
	}

	if got := getEnv(key, "fallback"); got != "fallback" {
		t.Fatalf("expected fallback value, got %q", got)
	}

	t.Setenv(key, "from-env")
	if got := getEnv(key, "fallback"); got != "from-env" {
		t.Fatalf("expected env value, got %q", got)
	}
}

func TestGetBoolEnv(t *testing.T) {
	const key = "UNIT_TEST_GET_BOOL_ENV_KEY"
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset env: %v", err)
	}

	tests := []struct {
		name     string
		value    string
		exists   bool
		fallback bool
		want     bool
	}{
		{name: "missing uses fallback true", exists: false, fallback: true, want: true},
		{name: "missing uses fallback false", exists: false, fallback: false, want: false},
		{name: "true literal", value: "true", exists: true, fallback: false, want: true},
		{name: "yes literal", value: "yes", exists: true, fallback: false, want: true},
		{name: "one literal", value: "1", exists: true, fallback: false, want: true},
		{name: "on literal", value: "on", exists: true, fallback: false, want: true},
		{name: "false literal", value: "false", exists: true, fallback: true, want: false},
		{name: "no literal", value: "no", exists: true, fallback: true, want: false},
		{name: "zero literal", value: "0", exists: true, fallback: true, want: false},
		{name: "off literal", value: "off", exists: true, fallback: true, want: false},
		{name: "trim and lower", value: "  TrUe  ", exists: true, fallback: false, want: true},
		{name: "invalid falls back", value: "not-a-bool", exists: true, fallback: true, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.exists {
				t.Setenv(key, tc.value)
			} else if err := os.Unsetenv(key); err != nil {
				t.Fatalf("unset env: %v", err)
			}

			got := getBoolEnv(key, tc.fallback)
			if got != tc.want {
				t.Fatalf("getBoolEnv()=%v, want=%v", got, tc.want)
			}
		})
	}
}

func TestApplyRuntimeModeFlags(t *testing.T) {
	tests := []struct {
		name     string
		enableWS bool
		enableGR bool
		args     []string
		wantWS   bool
		wantGR   bool
	}{
		{
			name:     "no flags keeps values",
			enableWS: true, enableGR: false,
			args:   []string{},
			wantWS: true, wantGR: false,
		},
		{
			name:     "disable both",
			enableWS: true, enableGR: true,
			args:   []string{"--no-ws", "--no-grpc"},
			wantWS: false, wantGR: false,
		},
		{
			name:     "enable aliases",
			enableWS: false, enableGR: false,
			args:   []string{"--enable-ws", "--grpc"},
			wantWS: true, wantGR: true,
		},
		{
			name:     "last flag wins",
			enableWS: true, enableGR: false,
			args:   []string{"--no-ws", "--ws", "--grpc", "--disable-grpc"},
			wantWS: true, wantGR: false,
		},
		{
			name:     "unknown flag ignored",
			enableWS: false, enableGR: true,
			args:   []string{"--unknown"},
			wantWS: false, wantGR: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotWS, gotGR := applyRuntimeModeFlags(tc.enableWS, tc.enableGR, tc.args)
			if gotWS != tc.wantWS || gotGR != tc.wantGR {
				t.Fatalf("applyRuntimeModeFlags()=(%v,%v), want=(%v,%v)", gotWS, gotGR, tc.wantWS, tc.wantGR)
			}
		})
	}
}

func TestParseCORSOrigins(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty string", in: "", want: []string{}},
		{name: "single origin", in: "http://localhost:3000", want: []string{"http://localhost:3000"}},
		{
			name: "trim and skip empty",
			in:   " https://a.com, ,https://b.com  ,",
			want: []string{"https://a.com", "https://b.com"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseCORSOrigins(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseCORSOrigins()=%v, want=%v", got, tc.want)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("uses defaults when env is absent", func(t *testing.T) {
		restore := unsetEnvForLoadConfig(t)
		defer restore()

		origArgs := os.Args
		os.Args = []string{"dashboard-server"}
		t.Cleanup(func() { os.Args = origArgs })

		cfg := loadConfig()
		if cfg.RedisAddr != "localhost:6379" ||
			cfg.RedisPass != "" ||
			cfg.HttpPort != ":8080" ||
			cfg.GrpcPort != ":50051" ||
			cfg.AuthSecret != "super-secret-key" ||
			cfg.JWTSecret != "" ||
			cfg.AuthMode != "static" ||
			cfg.CORSAllowedOrigins != "http://localhost:3000" {
			t.Fatalf("unexpected defaults: %+v", cfg)
		}
		if cfg.CORSAllowCredentials {
			t.Fatalf("CORSAllowCredentials should be false by default")
		}
		if !cfg.EnableWebSocket || !cfg.EnableGRPC {
			t.Fatalf("EnableWebSocket and EnableGRPC should be true by default")
		}
	})

	t.Run("uses env and runtime args overrides", func(t *testing.T) {
		restore := unsetEnvForLoadConfig(t)
		defer restore()

		t.Setenv("REDIS_ADDR", "redis:6380")
		t.Setenv("REDIS_PASS", "pass")
		t.Setenv("HTTP_PORT", ":18080")
		t.Setenv("GRPC_PORT", ":15051")
		t.Setenv("AUTH_SECRET", "custom-secret")
		t.Setenv("JWT_SECRET", "jwt-secret")
		t.Setenv("AUTH_MODE", "jwt")
		t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
		t.Setenv("CORS_ALLOW_CREDENTIALS", "1")
		t.Setenv("ENABLE_WEBSOCKET", "false")
		t.Setenv("ENABLE_GRPC", "true")

		origArgs := os.Args
		os.Args = []string{"dashboard-server", "--ws", "--no-grpc"}
		t.Cleanup(func() { os.Args = origArgs })

		cfg := loadConfig()
		if cfg.RedisAddr != "redis:6380" ||
			cfg.RedisPass != "pass" ||
			cfg.HttpPort != ":18080" ||
			cfg.GrpcPort != ":15051" ||
			cfg.AuthSecret != "custom-secret" ||
			cfg.JWTSecret != "jwt-secret" ||
			cfg.AuthMode != "jwt" ||
			cfg.CORSAllowedOrigins != "https://app.example.com" {
			t.Fatalf("unexpected env config: %+v", cfg)
		}
		if !cfg.CORSAllowCredentials {
			t.Fatalf("CORSAllowCredentials should be true for value 1")
		}
		if !cfg.EnableWebSocket || cfg.EnableGRPC {
			t.Fatalf("runtime flags not applied correctly: %+v", cfg)
		}
	})
}

type lifecycleStub struct {
	hooks []fx.Hook
}

func (l *lifecycleStub) Append(h fx.Hook) {
	l.hooks = append(l.hooks, h)
}

func TestProviders(t *testing.T) {
	t.Run("newLogger", func(t *testing.T) {
		logger, err := newLogger()
		if err != nil {
			t.Fatalf("newLogger() error = %v", err)
		}
		if logger == nil {
			t.Fatalf("newLogger() returned nil")
		}
		_ = logger.Sync()
	})

	t.Run("newRedisClient registers close hook", func(t *testing.T) {
		lc := &lifecycleStub{}
		cfg := Config{RedisAddr: "localhost:6379", RedisPass: ""}
		logger := zap.NewNop()

		client := newRedisClient(lc, cfg, logger)
		if client == nil {
			t.Fatalf("newRedisClient() returned nil")
		}
		if len(lc.hooks) != 1 {
			t.Fatalf("expected 1 lifecycle hook, got %d", len(lc.hooks))
		}
		if lc.hooks[0].OnStop == nil {
			t.Fatalf("expected OnStop hook to be set")
		}
		if err := lc.hooks[0].OnStop(context.Background()); err != nil {
			t.Fatalf("OnStop returned error: %v", err)
		}
	})

	t.Run("newRepository and newUseCase", func(t *testing.T) {
		client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
		t.Cleanup(func() { _ = client.Close() })

		repo := newRepository(client)
		if repo == nil {
			t.Fatalf("newRepository() returned nil")
		}

		uc := newUseCase(repo)
		if uc == nil {
			t.Fatalf("newUseCase() returned nil")
		}
	})

	t.Run("newAuthVerifier", func(t *testing.T) {
		staticVerifier, err := newAuthVerifier(Config{AuthMode: "static", AuthSecret: "secret"})
		if err != nil || staticVerifier == nil {
			t.Fatalf("newAuthVerifier(static) failed: %v", err)
		}
		if _, ok := staticVerifier.(*auth.StaticVerifier); !ok {
			t.Fatalf("expected *auth.StaticVerifier, got %T", staticVerifier)
		}

		jwtVerifier, err := newAuthVerifier(Config{AuthMode: "jwt", JWTSecret: "jwt-secret"})
		if err != nil || jwtVerifier == nil {
			t.Fatalf("newAuthVerifier(jwt) failed: %v", err)
		}
		if _, ok := jwtVerifier.(*auth.JWTVerifier); !ok {
			t.Fatalf("expected *auth.JWTVerifier, got %T", jwtVerifier)
		}

		noopVerifier, err := newAuthVerifier(Config{AuthMode: "noop"})
		if err != nil || noopVerifier == nil {
			t.Fatalf("newAuthVerifier(noop) failed: %v", err)
		}
		if _, ok := noopVerifier.(*auth.NoOpVerifier); !ok {
			t.Fatalf("expected *auth.NoOpVerifier, got %T", noopVerifier)
		}

		_, err = newAuthVerifier(Config{AuthMode: "jwt"})
		if err == nil {
			t.Fatalf("expected jwt mode without secret to fail")
		}
	})

	t.Run("newWSHub and runWSHub disabled", func(t *testing.T) {
		hub := newWSHub()
		if hub == nil {
			t.Fatalf("newWSHub() returned nil")
		}

		runWSHub(Config{EnableWebSocket: false}, hub, zap.NewNop())
	})

	t.Run("runWSHub enabled", func(t *testing.T) {
		hub := newWSHub()
		runWSHub(Config{EnableWebSocket: true}, hub, zap.NewNop())
	})
}

func TestRunHTTPServerLifecycle(t *testing.T) {
	lc := &lifecycleStub{}
	cfg := Config{
		HttpPort:             ":0",
		EnableWebSocket:      false,
		CORSAllowedOrigins:   "http://localhost:3000",
		CORSAllowCredentials: false,
	}

	runHTTPServer(lc, cfg, newWSHub(), nil, nil, zap.NewNop())
	if len(lc.hooks) != 1 {
		t.Fatalf("expected exactly one lifecycle hook, got %d", len(lc.hooks))
	}

	hook := lc.hooks[0]
	if hook.OnStart == nil || hook.OnStop == nil {
		t.Fatalf("expected both OnStart and OnStop hooks")
	}

	startCtx, cancelStart := context.WithTimeout(context.Background(), time.Second)
	defer cancelStart()
	if err := hook.OnStart(startCtx); err != nil {
		t.Fatalf("http OnStart returned error: %v", err)
	}

	stopCtx, cancelStop := context.WithTimeout(context.Background(), time.Second)
	defer cancelStop()
	if err := hook.OnStop(stopCtx); err != nil {
		t.Fatalf("http OnStop returned error: %v", err)
	}
}

func TestRunHTTPServerHandlers(t *testing.T) {
	t.Run("websocket disabled returns 503 and health is ok", func(t *testing.T) {
		lc := &lifecycleStub{}
		addr := reserveTCPAddress(t)
		cfg := Config{
			HttpPort:        addr,
			EnableWebSocket: false,
		}

		runHTTPServer(lc, cfg, newWSHub(), nil, nil, zap.NewNop())
		if len(lc.hooks) != 1 {
			t.Fatalf("expected lifecycle hook for http server")
		}
		hook := lc.hooks[0]

		requireNoError(t, hook.OnStart(context.Background()))
		t.Cleanup(func() {
			_ = hook.OnStop(context.Background())
		})

		resp := waitAndGet(t, "http://"+addr+"/health")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected /health to return 200, got %d", resp.StatusCode)
		}
		_ = resp.Body.Close()

		resp = waitAndGet(t, "http://"+addr+"/ws")
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("expected /ws to return 503 when disabled, got %d", resp.StatusCode)
		}
		_ = resp.Body.Close()
	})

	t.Run("websocket enabled returns 401 without token", func(t *testing.T) {
		lc := &lifecycleStub{}
		addr := reserveTCPAddress(t)
		cfg := Config{
			HttpPort:        addr,
			EnableWebSocket: true,
		}

		runHTTPServer(lc, cfg, newWSHub(), nil, nil, zap.NewNop())
		hook := lc.hooks[0]

		requireNoError(t, hook.OnStart(context.Background()))
		t.Cleanup(func() {
			_ = hook.OnStop(context.Background())
		})

		resp := waitAndGet(t, "http://"+addr+"/ws")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected /ws to return 401 without token, got %d", resp.StatusCode)
		}
		_ = resp.Body.Close()
	})
}

func TestRunHTTPServerRESTHandlers(t *testing.T) {
	lc := &lifecycleStub{}
	addr := reserveTCPAddress(t)
	cfg := Config{
		HttpPort:        addr,
		EnableWebSocket: false,
	}

	repo := mocks.NewDashboardRepository(t)
	uc := usecase.New(repo)
	verifier := &auth.StaticVerifier{Secret: "secret-token"}
	runHTTPServer(lc, cfg, newWSHub(), uc, verifier, zap.NewNop())
	if len(lc.hooks) != 1 {
		t.Fatalf("expected lifecycle hook for http server")
	}

	hook := lc.hooks[0]
	requireNoError(t, hook.OnStart(context.Background()))
	t.Cleanup(func() {
		_ = hook.OnStop(context.Background())
	})

	t.Run("rest_unauthorized_without_token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://"+addr+"/api/v1/dashboards/games/stats", http.NoBody)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		resp := waitAndDo(t, req)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
		_ = resp.Body.Close()
	})

	t.Run("increment_score_success", func(t *testing.T) {
		repo.EXPECT().
			IncrementMemberScore(mock.Anything, "games", "admin_user", 5.0).
			Return(nil).
			Once()
		repo.EXPECT().
			ViewMemberRank(mock.Anything, "games", "admin_user").
			Return(int64(1), nil).
			Once()

		payload := map[string]float64{"increment": 5}
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}

		req, err := http.NewRequest(
			http.MethodPost,
			"http://"+addr+"/api/v1/dashboards/games/members/user123/increment",
			bytes.NewReader(raw),
		)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer secret-token")
		req.Header.Set("Content-Type", "application/json")

		resp := waitAndDo(t, req)
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(body))
		}
		defer func() { _ = resp.Body.Close() }()

		var got struct {
			MemberID string `json:"member_id"`
			Rank     int64  `json:"rank"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.MemberID != "admin_user" || got.Rank != 1 {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
}

func TestRunGRPCServerLifecycle(t *testing.T) {
	t.Run("disabled mode does nothing", func(t *testing.T) {
		lc := &lifecycleStub{}
		runGRPCServer(lc, Config{EnableGRPC: false}, nil, nil, zap.NewNop())
		if len(lc.hooks) != 0 {
			t.Fatalf("expected no hooks when gRPC is disabled, got %d", len(lc.hooks))
		}
	})

	t.Run("enabled mode registers start stop hooks", func(t *testing.T) {
		lc := &lifecycleStub{}
		cfg := Config{
			EnableGRPC: true,
			GrpcPort:   "127.0.0.1:0",
		}
		runGRPCServer(lc, cfg, nil, nil, zap.NewNop())
		if len(lc.hooks) != 1 {
			t.Fatalf("expected exactly one lifecycle hook, got %d", len(lc.hooks))
		}

		hook := lc.hooks[0]
		if hook.OnStart == nil || hook.OnStop == nil {
			t.Fatalf("expected both OnStart and OnStop hooks")
		}

		startCtx, cancelStart := context.WithTimeout(context.Background(), time.Second)
		defer cancelStart()
		if err := hook.OnStart(startCtx); err != nil {
			t.Fatalf("grpc OnStart returned error: %v", err)
		}

		stopCtx, cancelStop := context.WithTimeout(context.Background(), time.Second)
		defer cancelStop()
		if err := hook.OnStop(stopCtx); err != nil {
			t.Fatalf("grpc OnStop returned error: %v", err)
		}
	})
}

func unsetEnvForLoadConfig(t *testing.T) func() {
	t.Helper()

	keys := []string{
		"REDIS_ADDR",
		"REDIS_PASS",
		"HTTP_PORT",
		"GRPC_PORT",
		"AUTH_SECRET",
		"JWT_SECRET",
		"AUTH_MODE",
		"CORS_ALLOWED_ORIGINS",
		"CORS_ALLOW_CREDENTIALS",
		"ENABLE_WEBSOCKET",
		"ENABLE_GRPC",
	}

	type savedValue struct {
		value string
		ok    bool
	}
	saved := make(map[string]savedValue, len(keys))
	for _, key := range keys {
		v, ok := os.LookupEnv(key)
		saved[key] = savedValue{value: v, ok: ok}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}

	return func() {
		for _, key := range keys {
			s := saved[key]
			var err error
			if s.ok {
				err = os.Setenv(key, s.value)
			} else {
				err = os.Unsetenv(key)
			}
			if err != nil {
				t.Fatalf("restore %s: %v", key, err)
			}
		}
	}
}

func reserveTCPAddress(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

func waitAndGet(t *testing.T, url string) *http.Response {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, err := http.Get(url)
		if err == nil {
			return resp
		}
		if time.Now().After(deadline) {
			t.Fatalf("http get failed for %s: %v", url, err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func waitAndDo(t *testing.T, req *http.Request) *http.Response {
	t.Helper()

	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, err := client.Do(req)
		if err == nil {
			return resp
		}
		if time.Now().After(deadline) {
			t.Fatalf("http request failed for %s %s: %v", req.Method, req.URL.String(), err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
