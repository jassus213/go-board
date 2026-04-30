# 🏆 GoBoard - Real-Time Leaderboard Engine

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![CI](https://github.com/jassus213/go-board/actions/workflows/ci.yml/badge.svg)](https://github.com/jassus213/go-board/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/jassus213/go-board/branch/main/graph/badge.svg)](https://codecov.io/gh/jassus213/go-board)
[![Go Report Card](https://goreportcard.com/badge/github.com/jassus213/go-board)](https://goreportcard.com/report/github.com/jassus213/go-board)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/jassus213/go-board)](https://github.com/jassus213/go-board/releases)

A production-ready leaderboard engine built with Redis, with real-time updates over WebSocket and gRPC streaming.

## ✨ Key Features

- **🔌 Easy Integration**
    - Import as a Go module
    - Run as a standalone service
    - Clear interfaces for custom implementations

- **⚡ Performance-Oriented**
    - Redis Sorted Sets (`O(log N)`)
    - Batch operations with pipelining
    - Lightweight runtime and transport layers

- **🎯 Developer-First**
    - Clean architecture (`core`, `repo`, `delivery`, `auth`)
    - Unit and integration tests
    - gRPC + WebSocket support out of the box

- **🚀 Tooling**
    - GitHub Actions CI
    - Codecov integration
    - Docker and Docker Compose setup

## 📋 Table of Contents

- [Integration Methods](#-integration-methods)
    - [Method 1: As a Go Module](#method-1-as-a-go-module-recommended)
    - [Method 2: As a Standalone Service](#method-2-as-a-standalone-service)
- [Quick Start](#-quick-start)
- [Architecture](#-architecture)
- [API Reference](#-api-reference)
- [Configuration](#-configuration)
- [Development](#-development)
- [Testing](#-testing)
- [CI and Release](#-ci-and-release)
- [Deployment](#-deployment)
- [Troubleshooting](#-troubleshooting)
- [Contributing](#-contributing)

## 🔌 Integration Methods

GoBoard can be integrated in two ways:

### Method 1: As a Go Module (Recommended)

Import `dashboard` into your Go app and use the repository/use case directly.

**Pros:**
- ✅ Full control over integration
- ✅ No network hop
- ✅ Easy extension inside your service

### Method 2: As a Standalone Service

Run GoBoard as a separate process and connect via WebSocket/gRPC.

**Pros:**
- ✅ Language-agnostic API
- ✅ Independent scaling
- ✅ Shared leaderboard backend for multiple services

---

## 🚀 Quick Start

### Option A: Use as a Go Module

**1. Install the module**

```bash
go get github.com/jassus213/go-board/dashboard
```

**2. Minimal usage**

```go
package main

import (
	"context"
	"fmt"

	"github.com/jassus213/go-board/dashboard/repo/redis"
	goredis "github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	rdb := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
	repo := redis.NewDashboardRedisRepository(rdb, "myapp:")

	if err := repo.AddMemberToDashboard(ctx, "global", "user123", 1000); err != nil {
		panic(err)
	}

	top, _ := repo.GetTopMembers(ctx, "global", 10)
	for _, member := range top {
		fmt.Printf("Rank %d: %s (%.0f points)\n", member.Rank, member.ID, member.Score)
	}
}
```

### Option B: Run as a Standalone Service

**1. Clone and configure**

```bash
git clone https://github.com/jassus213/go-board.git
cd go-board
cp cmd/dashboard-server/env.example .env
```

**2. Start dependencies (Redis + Redis Insight)**

```bash
docker compose -f deploy/docker/local/docker-compose.yml up -d
```

**3. Run the server**

```bash
AUTH_SECRET=super-secret-key go run ./cmd/dashboard-server/main.go --ws --grpc
```

Service endpoints:
- **WebSocket**: `ws://localhost:8080/ws`
- **gRPC**: `localhost:50051`
- **Health**: `http://localhost:8080/health`

**4. Smoke test transports**

**JavaScript/TypeScript (WebSocket)**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws?token=super-secret-key');

ws.onopen = () => {
    ws.send(JSON.stringify({
        dashboard: "global_leaderboard",
        member_id: "user123",
        increment: 10.5
    }));
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('New rank:', data.rank, 'error:', data.error);
};
```

**gRPC (`grpcurl`)**
```bash
grpcurl -plaintext \
  -import-path dashboard/delivery/grpc/proto \
  -proto dashboard.proto \
  -H "authorization: Bearer super-secret-key" \
  -d '{"dashboard":"global","member_id":"user123","increment":10}' \
  localhost:50051 dashboard.DashboardService/StreamUpdates
```

**cURL (Quick Test)**
```bash
curl http://localhost:8080/health
```

---

## 🏗️ Architecture

```text
go-board/
├── cmd/dashboard-server/             # service entrypoint (dashboard-api module)
├── dashboard/                        # reusable leaderboard module
│   ├── auth/                         # token verifiers
│   ├── core/                         # dto, entities, interfaces, usecase
│   ├── delivery/
│   │   ├── grpc/                     # gRPC server + proto/gen
│   │   └── ws/                       # WebSocket hub/clients
│   └── repo/redis/                   # Redis repository implementation
└── deploy/docker/                    # Docker assets
```

### Key Design Patterns

- **Use Case Layer**: business workflows are orchestrated in `dashboard/core/usecase`
- **Repository Pattern**: data access behind interfaces
- **Hub Pattern**: WebSocket connection management
- **Dependency Injection**: Uber Fx runtime composition

## ⚙️ Configuration

### Environment Variables

The runtime (`cmd/dashboard-server/main.go`) reads `.env` if present.

```bash
# Server
HTTP_PORT=:8080
GRPC_PORT=:50051

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASS=

# Auth (current runtime uses StaticVerifier)
AUTH_SECRET=super-secret-key

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000
CORS_ALLOW_CREDENTIALS=false

# Transport toggles
ENABLE_WEBSOCKET=true
ENABLE_GRPC=true
```

> Note: `cmd/dashboard-server/env.example` includes some extra variables from earlier iterations that are not all consumed by current runtime.

## 📡 API Reference

### WebSocket API

**Connect**
```text
ws://localhost:8080/ws?token=<AUTH_SECRET>
```

**Send Score Update**
```json
{
  "dashboard": "global_leaderboard",
  "member_id": "user123",
  "increment": 10.5
}
```

**Receive Response**
```json
{
  "rank": 42
}
```

On error, WebSocket responses include typed Problem Details:
```json
{
  "problem": {
    "type": "urn:goboard:request:invalid-argument",
    "title": "Bad Request",
    "status": 400,
    "detail": "missing dashboard or member_id",
    "instance": "/ws",
    "code": "invalid_argument"
  }
}
```

### gRPC API

**Service Definition**
```protobuf
service DashboardService {
  rpc StreamUpdates (stream UpdateRequest) returns (stream UpdateResponse);
  rpc IncrementScore (IncrementScoreRequest) returns (IncrementScoreResponse);
  rpc GetMemberRank (GetMemberRankRequest) returns (GetMemberRankResponse);
  rpc GetTopMembers (GetTopMembersRequest) returns (GetTopMembersResponse);
  rpc GetDashboardStats (GetDashboardStatsRequest) returns (GetDashboardStatsResponse);
}

message UpdateRequest {
  string dashboard = 1;
  string member_id = 2;
  double increment = 3;
}

message UpdateResponse {
  string member_id = 1;
  int64 rank = 2;
  double score = 3;
  ProblemDetails problem = 5;
}

message ProblemDetails {
  string type = 1;
  string title = 2;
  int32 status = 3;
  string detail = 4;
  string instance = 5;
  string code = 6;
}

message IncrementScoreRequest {
  string dashboard = 1;
  string member_id = 2;
  double increment = 3;
}

message IncrementScoreResponse {
  string member_id = 1;
  int64 rank = 2;
}
```

Auth failures in the gRPC interceptor are returned as `grpc status` errors with:
- code mapped from `ProblemDetails.status` (`Unauthenticated`, `PermissionDenied`, etc.),
- `ProblemDetails` attached to status `details` for typed client-side handling.

### REST API (Gin)

All REST routes are under `http://localhost:8080/api/v1` and require `Authorization: Bearer <AUTH_SECRET>`.

- `POST /dashboards/:dashboard/members/:member_id/increment` with body `{"increment": 10.5}`
- `GET /dashboards/:dashboard/members/:member_id/rank`
- `GET /dashboards/:dashboard/top?limit=10`
- `GET /dashboards/:dashboard/stats`

Errors are returned as `application/problem+json` payloads using `ProblemDetails`.

### API Cheat Sheet

Set shared variables:
```bash
TOKEN=super-secret-key
BASE_URL=http://localhost:8080/api/v1
GRPC_ADDR=localhost:50051
```

**REST (`curl`)**
```bash
# Increment authenticated member score
curl -X POST "$BASE_URL/dashboards/global/members/user123/increment" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"increment":10.5}'

# Get authenticated member rank
curl -X GET "$BASE_URL/dashboards/global/members/user123/rank" \
  -H "Authorization: Bearer $TOKEN"

# Get top members
curl -X GET "$BASE_URL/dashboards/global/top?limit=5" \
  -H "Authorization: Bearer $TOKEN"

# Get dashboard stats
curl -X GET "$BASE_URL/dashboards/global/stats" \
  -H "Authorization: Bearer $TOKEN"
```

**gRPC unary (`grpcurl`)**
```bash
# Increment score
grpcurl -plaintext \
  -import-path dashboard/delivery/grpc/proto \
  -proto dashboard.proto \
  -H "authorization: Bearer $TOKEN" \
  -d '{"dashboard":"global","member_id":"user123","increment":10.5}' \
  $GRPC_ADDR dashboard.DashboardService/IncrementScore

# Get member rank
grpcurl -plaintext \
  -import-path dashboard/delivery/grpc/proto \
  -proto dashboard.proto \
  -H "authorization: Bearer $TOKEN" \
  -d '{"dashboard":"global","member_id":"user123"}' \
  $GRPC_ADDR dashboard.DashboardService/GetMemberRank

# Get top members
grpcurl -plaintext \
  -import-path dashboard/delivery/grpc/proto \
  -proto dashboard.proto \
  -H "authorization: Bearer $TOKEN" \
  -d '{"dashboard":"global","limit":5}' \
  $GRPC_ADDR dashboard.DashboardService/GetTopMembers

# Get dashboard stats
grpcurl -plaintext \
  -import-path dashboard/delivery/grpc/proto \
  -proto dashboard.proto \
  -H "authorization: Bearer $TOKEN" \
  -d '{"dashboard":"global"}' \
  $GRPC_ADDR dashboard.DashboardService/GetDashboardStats
```

**Error example (`ProblemDetails`)**
```bash
curl -i -X GET "$BASE_URL/dashboards/global/stats"
```
```json
{
  "type": "urn:goboard:auth:missing-token",
  "title": "Unauthorized",
  "status": 401,
  "detail": "authentication token is required",
  "instance": "/api/v1/dashboards/global/stats",
  "code": "auth_missing_token"
}
```

### Known Runtime Behavior

- Runtime in `cmd/dashboard-server` currently wires `auth.StaticVerifier`.
- For valid token, authenticated identity resolves to fixed `"admin_user"`.
- Incoming `member_id` is validated/overridden by authenticated identity logic in transport handlers.

### Repository Interface

```go
type DashboardRepository interface {
    AddMemberToDashboard(ctx context.Context, dashboard, member string, score float64) error
    AddMembersBatch(ctx context.Context, dashboard string, members []entity.DashboardRecord) error
    RemoveMemberFromDashboard(ctx context.Context, dashboard, member string) error
    GetTopMembers(ctx context.Context, dashboard string, top int64) ([]entity.DashboardRecord, error)
    ViewMemberRank(ctx context.Context, dashboard string, memberId string) (int64, error)
    IncrementMemberScore(ctx context.Context, dashboard, member string, increment float64) error
    GetTotalMembers(ctx context.Context, dashboard string) (int64, error)
    DeleteDashboard(ctx context.Context, dashboard string) error
    IncrementMembersBatch(ctx context.Context, dashboard string, increments []entity.DashboardRecord) error
}
```

## 🛠️ Development

### Running Locally

```bash
# Start dependencies
docker compose -f deploy/docker/local/docker-compose.yml up -d

# Run service
go run ./cmd/dashboard-server/main.go
```

This repository currently has a **two-module Go setup**:
- root module (`go.mod`) for `dashboard-api` (service runtime),
- nested module (`dashboard/go.mod`) for the reusable `dashboard` package.

### Useful Commands

```bash
# Root module tests (dashboard-api)
go test ./...

# Library module tests
(cd dashboard && go test ./...)

# Build server binary
go build -o bin/dashboard-server ./cmd/dashboard-server/main.go
```

## 🐳 Deployment

### Using Docker

**Build Image**
```bash
docker build -t goboard:latest -f deploy/docker/Dockerfile .
```

**Run Container**
```bash
docker run -d \
  --name goboard \
  -p 8080:8080 \
  -p 50051:50051 \
  -e REDIS_ADDR=host.docker.internal:6379 \
  -e AUTH_SECRET=your-secret \
  goboard:latest
```

### Using Docker Compose (Dependencies)

```bash
docker compose -f deploy/docker/local/docker-compose.yml up -d
```

## 🧪 Testing

### Run All Tests
```bash
go test ./...
(cd dashboard && go test ./...)
```

### Run with Coverage
```bash
(cd dashboard && go test -coverprofile=../coverage.out -covermode=atomic ./...)
```

### Benchmarks
```bash
(cd dashboard && go test -bench=. -benchmem ./auth/...)
```

## 🔄 CI and Release

- CI runs from `.github/workflows/ci.yml` (lint, tests, build, docker, benchmark).
- Releases run from `.github/workflows/release.yml` on tags `v*.*.*`.
- For daily development, badges at the top of this README are the primary status view.

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes
4. Run tests locally
5. Open a Pull Request

### Code Style

- Follow standard Go conventions
- Run `gofmt` before committing
- Add tests for new features
- Update docs when behavior changes

For an expanded contributor checklist, see [`CONTRIBUTING.md`](CONTRIBUTING.md).

## 🛟 Troubleshooting

- **Redis connection errors**
  - Ensure Redis is running on `REDIS_ADDR` (default `localhost:6379`).
  - Quick check: `redis-cli -h localhost -p 6379 ping`.
- **401/403 on WebSocket/gRPC**
  - Ensure token matches `AUTH_SECRET` in current runtime mode.
  - For gRPC, include `authorization` metadata (`Bearer <token>` or plain token).
- **Browser WS rejected by CORS**
  - Add your frontend origin to `CORS_ALLOWED_ORIGINS`.
  - Do not use wildcard `*` in production.

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👤 Author

**Nikita Okhotnikov**

- GitHub: [@jassus213](https://github.com/jassus213)

## 🙏 Acknowledgments

- [Uber Fx](https://github.com/uber-go/fx) - dependency injection and lifecycle
- [go-redis](https://github.com/redis/go-redis) - Redis client for Go
- [Gorilla WebSocket](https://github.com/gorilla/websocket) - WebSocket implementation
- [gRPC](https://grpc.io/) - RPC framework
- [Redis](https://redis.io/) - sorted-set storage model

## 📮 Support

For issues and questions:
- Open an [Issue](https://github.com/jassus213/go-board/issues)
- Check existing [Issues](https://github.com/jassus213/go-board/issues) first

---

**Built with ❤️ using Go and Redis**
