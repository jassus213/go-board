# 🏆 GoBoard - Real-Time Leaderboard Engine

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![CI](https://github.com/jassus213/go-board/actions/workflows/ci.yml/badge.svg)](https://github.com/jassus213/go-board/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/jassus213/go-board/branch/main/graph/badge.svg)](https://codecov.io/gh/jassus213/go-board)
[![Go Report Card](https://goreportcard.com/badge/github.com/jassus213/go-board)](https://goreportcard.com/report/github.com/jassus213/go-board)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/jassus213/go-board)](https://github.com/jassus213/go-board/releases)

A production-ready, high-performance leaderboard engine designed for easy integration into your Go applications. Built with Redis and featuring real-time updates via WebSocket and gRPC streaming.

## ✨ Key Features

- **🔌 Easy Integration**
    - Import as a Go module
    - Run as a standalone SaaS service
    - Well-defined interfaces for custom implementations

- **⚡ High Performance**
    - Redis Sorted Sets (O(log N) operations)
    - Batch operations with pipelining
    - Handles 10K+ updates/sec per instance

- **🎯 Developer-First**
    - Simple repository pattern interface
    - Comprehensive test coverage (>85%)
    - Example code and integration guides

- **🚀 Production Ready**
    - Docker support
    - CI/CD with CircleCI
    - Structured logging and metrics-ready

- **🔐 Security Built-in**
    - JWT/Static token authentication
    - Configurable CORS
    - Rate limiting ready

## 📋 Table of Contents

- [Integration Methods](#-integration-methods)
    - [Method 1: As a Go Module](#method-1-as-a-go-module-recommended)
    - [Method 2: As a Standalone Service (SaaS)](#method-2-as-a-standalone-service-saas)
- [Quick Start](#-quick-start)
- [Architecture](#-architecture)
- [API Reference](#-api-reference)
- [Configuration](#-configuration)
- [Development](#-development)
- [Testing](#-testing)
- [Deployment](#-deployment)
- [Performance](#-performance)
- [Contributing](#-contributing)

## 🔌 Integration Methods

GoBoard can be integrated into your application in two ways:

### Method 1: As a Go Module (Recommended)

Import GoBoard directly into your Go application and use the repository interface.

**Pros:**
- ✅ Full control over the code
- ✅ No network overhead
- ✅ Easy to customize
- ✅ Shared Redis instance with your app

**Use Case:** When you're building a Go application and want leaderboard functionality embedded.

### Method 2: As a Standalone Service (SaaS)

Run GoBoard as a separate microservice and communicate via gRPC or WebSocket.

**Pros:**
- ✅ Language agnostic (any language can use it)
- ✅ Independent scaling
- ✅ Isolated failures
- ✅ Multiple apps can share one instance

**Use Case:** When you have multiple services (different languages) or want to run leaderboards as a separate concern.

---

## 🚀 Quick Start

### Option A: Use as a Go Module

### Option A: Use as a Go Module

**1. Install the module**

```bash
go get github.com/jassus213/go-board/pkg/dashboard
```

**2. Initialize Redis client**

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/jassus213/go-board/pkg/dashboard/dal/redis"
    goredis "github.com/redis/go-redis/v9"
)

func main() {
    ctx := context.Background()
    
    // Connect to Redis
    rdb := goredis.NewClient(&goredis.Options{
        Addr: "localhost:6379",
    })
    
    // Create repository
    repo := redis.NewDashboardRedisRepository(rdb, "myapp:")
    
    // Add a member to leaderboard
    err := repo.AddMemberToDashboard(ctx, "global", "user123", 1000)
    if err != nil {
        panic(err)
    }
    
    // Get top 10 members
    top, _ := repo.GetTopMembers(ctx, "global", 10)
    for _, member := range top {
        fmt.Printf("Rank %d: %s (%.0f points)\n", 
            member.Rank, member.ID, member.Score)
    }
}
```

**3. That's it!** You now have a fully functional leaderboard.

**Advanced: Batch Operations**

```go
// Add multiple members at once (more efficient)
members := []domain.DashboardRecord{
    {ID: "user1", Score: 1500},
    {ID: "user2", Score: 1200},
    {ID: "user3", Score: 980},
}
repo.AddMembersBatch(ctx, "global", members)

// Increment scores in batch (for high-throughput scenarios)
increments := []domain.DashboardRecord{
    {ID: "user1", Score: 10},  // +10 points
    {ID: "user2", Score: 5},   // +5 points
}
repo.IncrementMembersBatch(ctx, "global", increments)
```

**Advanced: Using with your own Business Logic**

```go
import "github.com/jassus213/go-board/pkg/dashboard/bll"

// Commands (Write operations)
err := bll.AddMemberHandler(ctx, repo, bll.AddMemberRequest{
    Dashboard: "global",
    MemberID:  "user123",
    Score:     1000,
})

err = bll.IncrementScoreHandler(ctx, repo, bll.IncrementScoreRequest{
    Dashboard: "global",
    MemberID:  "user123",
    Value:     50,
})

// Queries (Read operations)
topMembers, err := bll.GetTopMembersHandler(ctx, repo, bll.GetTopRequest{
    Dashboard: "global",
    Limit:     10,
})

rank, err := bll.GetMemberRankHandler(ctx, repo, bll.GetRankRequest{
    Dashboard: "global",
    MemberID:  "user123",
})

// Workflows (Combined operations)
rank, err := bll.ProcessScoreUpdate(ctx, repo, bll.IncrementScoreRequest{
    Dashboard: "global",
    MemberID:  "user123",
    Value:     10,
})
// This atomically increments score AND returns new rank
```

### Option B: Run as a Standalone Service

**1. Clone and configure**
**1. Clone and configure**

```bash
git clone https://github.com/jassus213/go-board.git
cd go-board
cp .env.example .env
# Edit .env with your settings
```

2. **Start Redis (using Docker)**
```bash
cd deploy/docker/local
docker-compose up -d redis
```

3. **Run the server**
```bash
go run cmd/dashboard-server/main.go
```

The service will be available at:
- **WebSocket**: `ws://localhost:8080/ws`
- **gRPC**: `localhost:50051`
- **Health Check**: `http://localhost:8080/health`

**4. Connect from any language**

**JavaScript/TypeScript (WebSocket)**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws?token=your-jwt-token');

ws.onopen = () => {
    // Increment score
    ws.send(JSON.stringify({
        dashboard: "global_leaderboard",
        member_id: "user123",
        increment: 10.5
    }));
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('New rank:', data.rank);
};
```

**Python (gRPC)**
```python
import grpc
import dashboard_pb2
import dashboard_pb2_grpc

channel = grpc.insecure_channel('localhost:50051')
stub = dashboard_pb2_grpc.DashboardServiceStub(channel)

# Stream updates
def generate_requests():
    yield dashboard_pb2.UpdateRequest(
        dashboard="global",
        member_id="user123",
        increment=10.5
    )

responses = stub.StreamUpdates(generate_requests())
for response in responses:
    print(f'New rank: {response.rank}')
```

**cURL (Quick Test)**
```bash
# Health check
curl http://localhost:8080/health

# WebSocket (using websocat)
echo '{"dashboard":"global","member_id":"user123","increment":10}' | \
  websocat ws://localhost:8080/ws?token=test-token
```

---

## 📚 Integration Examples

### Example 1: Game Backend Integration

```go
package game

import (
    "context"
    "github.com/jassus213/go-board/pkg/dashboard/dal/redis"
    "github.com/jassus213/go-board/pkg/dashboard/bll"
)

type GameServer struct {
    leaderboard dal.DashboardRepository
}

func (g *GameServer) OnMatchComplete(winner, loser string, points int) {
    ctx := context.Background()
    
    // Update winner's score
    bll.IncrementScoreHandler(ctx, g.leaderboard, bll.IncrementScoreRequest{
        Dashboard: "ranked_pvp",
        MemberID:  winner,
        Value:     float64(points),
    })
    
    // Update loser's score (negative increment)
    bll.IncrementScoreHandler(ctx, g.leaderboard, bll.IncrementScoreRequest{
        Dashboard: "ranked_pvp",
        MemberID:  loser,
        Value:     -float64(points / 2),
    })
}

func (g *GameServer) GetPlayerRank(playerID string) (int64, error) {
    ctx := context.Background()
    return bll.GetMemberRankHandler(ctx, g.leaderboard, bll.GetRankRequest{
        Dashboard: "ranked_pvp",
        MemberID:  playerID,
    })
}
```

### Example 2: E-commerce Integration

```go
package shop

import (
    "context"
    "github.com/jassus213/go-board/pkg/dashboard/dal/redis"
    "github.com/jassus213/go-board/pkg/dashboard/bll"
)

type LoyaltyService struct {
    leaderboard dal.DashboardRepository
}

func (l *LoyaltyService) OnPurchase(customerID string, amount float64) {
    ctx := context.Background()
    
    // Add loyalty points
    points := amount * 0.1 // 10% cashback
    
    bll.IncrementScoreHandler(ctx, l.leaderboard, bll.IncrementScoreRequest{
        Dashboard: "loyalty_points",
        MemberID:  customerID,
        Value:     points,
    })
}

func (l *LoyaltyService) GetTopCustomers(limit int) ([]domain.DashboardRecord, error) {
    ctx := context.Background()
    return bll.GetTopMembersHandler(ctx, l.leaderboard, bll.GetTopRequest{
        Dashboard: "loyalty_points",
        Limit:     int64(limit),
    })
}
```

### Example 3: Social Media Integration

```go
package social

import (
    "context"
    "github.com/jassus213/go-board/pkg/dashboard/dal/redis"
    "github.com/jassus213/go-board/pkg/dashboard/bll"
)

type InfluencerRanking struct {
    leaderboard dal.DashboardRepository
}

func (i *InfluencerRanking) OnPostEngagement(userID string, likes, shares, comments int) {
    ctx := context.Background()
    
    // Calculate engagement score
    score := float64(likes*1 + shares*5 + comments*3)
    
    bll.IncrementScoreHandler(ctx, i.leaderboard, bll.IncrementScoreRequest{
        Dashboard: "influencer_weekly",
        MemberID:  userID,
        Value:     score,
    })
}

func (i *InfluencerRanking) ResetWeeklyRankings() error {
    ctx := context.Background()
    return bll.DeleteDashboardHandler(ctx, i.leaderboard, "influencer_weekly")
}
```

---

## 🎯 Quick Test
```bash
curl http://localhost:8080/health
```

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Client Layer                            │
│  (WebSocket Clients / gRPC Clients / HTTP Clients)          │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                   Delivery Layer                             │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐             │
│  │WebSocket │    │  gRPC    │    │   HTTP   │             │
│  │  Hub     │    │ Streaming│    │ Handlers │             │
│  └──────────┘    └──────────┘    └──────────┘             │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│               Business Logic Layer (BLL)                     │
│  ┌──────────────────┐    ┌─────────────────────┐           │
│  │   Commands       │    │     Queries         │           │
│  │  (Write Ops)     │    │    (Read Ops)       │           │
│  │ - AddMember      │    │ - GetTopMembers     │           │
│  │ - IncrementScore │    │ - GetMemberRank     │           │
│  │ - RemoveMember   │    │ - GetTotalMembers   │           │
│  └──────────────────┘    └─────────────────────┘           │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│              Data Access Layer (DAL)                         │
│  ┌────────────────────────────────────────────┐             │
│  │       Repository Interface                 │             │
│  └────────────────────┬───────────────────────┘             │
│                       │                                      │
│  ┌────────────────────▼───────────────────────┐             │
│  │      Redis Implementation                  │             │
│  │  - Sorted Sets (ZSET)                      │             │
│  │  - Pipeline Operations                      │             │
│  │  - Key Prefixing                           │             │
│  └────────────────────────────────────────────┘             │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                     Redis Storage                            │
│  ┌─────────────────────────────────────────────┐            │
│  │  Sorted Sets (O(log N) operations)          │            │
│  │  - prod:leaderboard1 → {user1: 1000}        │            │
│  │  - prod:leaderboard2 → {user2: 500}         │            │
│  └─────────────────────────────────────────────┘            │
└──────────────────────────────────────────────────────────────┘
```

### Key Design Patterns

- **CQRS**: Commands (writes) and Queries (reads) are separated for better scalability
- **Repository Pattern**: Data access is abstracted behind interfaces
- **Hub Pattern**: WebSocket connections are managed by a central hub
- **Dependency Injection**: Uber Fx manages component lifecycle and dependencies

## ⚙️ Configuration

### Environment Variables

Create a `.env` file based on `.env.example`:

```bash
# Server Configuration
HTTP_PORT=:8080
GRPC_PORT=:50051

# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_PASS=
REDIS_PREFIX=prod:

# Authentication
AUTH_SECRET=your-secret-key
JWT_SECRET=your-jwt-secret-key
AUTH_MODE=jwt  # Options: static, jwt, noop

# CORS Configuration
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://your-domain.com
CORS_ALLOW_CREDENTIALS=true

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### CORS Configuration

⚠️ **Important**: Properly configure CORS for production!

```bash
# Development (allow localhost)
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# Production (specific domains only)
CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com

# ❌ DO NOT USE IN PRODUCTION
CORS_ALLOWED_ORIGINS=*
```

### Authentication Modes

**JWT Mode (Recommended for Production)**
```bash
AUTH_MODE=jwt
JWT_SECRET=your-secure-jwt-secret-at-least-32-characters
```

Generate JWT tokens with:
```go
import "github.com/golang-jwt/jwt/v5"

token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
    "user_id": "user123",
    "exp":     time.Now().Add(24 * time.Hour).Unix(),
})
tokenString, _ := token.SignedString([]byte("your-jwt-secret"))
```

**Static Mode (Internal Services)**
```bash
AUTH_MODE=static
AUTH_SECRET=shared-secret-between-services
```

**NoOp Mode (Development Only)**
```bash
AUTH_MODE=noop  # ⚠️ NO SECURITY - Token is treated as user ID
```

## 📡 API Reference

### WebSocket API

**Connect**
```
ws://localhost:8080/ws?token=<JWT_TOKEN>
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
  "member_id": "user123",
  "rank": 42,
  "error": ""
}
```

### gRPC API

**Service Definition**
```protobuf
service DashboardService {
  rpc StreamUpdates (stream UpdateRequest) returns (stream UpdateResponse);
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
  string error = 4;
}
```

**Example Client (Go)**
```go
conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
client := pb.NewDashboardServiceClient(conn)

stream, _ := client.StreamUpdates(context.Background())

// Send update
stream.Send(&pb.UpdateRequest{
    Dashboard: "global",
    MemberId:  "user123",
    Increment: 10.5,
})

// Receive response
resp, _ := stream.Recv()
fmt.Printf("New rank: %d\n", resp.Rank)
```

### Repository Interface

```go
type DashboardRepository interface {
    // Write operations
    AddMemberToDashboard(ctx, dashboard, member string, score float64) error
    AddMembersBatch(ctx, dashboard string, members []DashboardRecord) error
    IncrementMemberScore(ctx, dashboard, member string, increment float64) error
    RemoveMemberFromDashboard(ctx, dashboard, member string) error
    DeleteDashboard(ctx, dashboard string) error
    
    // Read operations
    GetTopMembers(ctx, dashboard string, top int64) ([]DashboardRecord, error)
    ViewMemberRank(ctx, dashboard, memberId string) (int64, error)
    GetTotalMembers(ctx, dashboard string) (int64, error)
}
```

## 🛠️ Development

### Project Structure

```
go-board/
├── cmd/
│   └── dashboard-server/
│       └── main.go              # Application entry point
├── pkg/
│   └── dashboard/
│       ├── auth/                # Authentication & verification
│       │   ├── verifier.go
│       │   └── verifier_test.go
│       ├── bll/                 # Business Logic Layer
│       │   ├── commands.go      # Write operations
│       │   ├── queries.go       # Read operations
│       │   └── workflow.go      # Composite operations
│       ├── dal/                 # Data Access Layer
│       │   ├── dashboard_repository.go
│       │   └── redis/
│       │       └── repository.go
│       ├── delivery/            # Transport Layer
│       │   ├── grpc/
│       │   │   ├── server.go
│       │   │   └── proto/
│       │   └── ws/
│       │       ├── hub.go
│       │       ├── client.go
│       │       └── messages.go
│       └── domain/              # Domain models
│           ├── record.go
│           └── errors.go
├── deploy/
│   └── docker/
│       ├── Dashboard.Dockerfile
│       └── local/
│           └── docker-compose.yaml
├── .circleci/
│   └── config.yml
├── .env.example
├── Makefile
└── README.md
```

### Running Locally

**1. Start Dependencies**
```bash
cd deploy/docker/local
docker-compose up -d
```

This starts:
- Redis on port `6379`
- Redis Insight on port `8001` (GUI at http://localhost:8001)

**2. Run the Server**
```bash
# Development mode
go run cmd/dashboard-server/main.go

# With hot reload (using air)
air

# Build and run
make build
./bin/dashboard-server
```

### Makefile Commands

```bash
# Generate protobuf files
make gen

# Install protoc plugins
make deps

# Build the application
make build

# Run tests
make test

# Run tests with coverage
make test-cover

# Clean generated files
make clean
```

### Adding New Features

**1. Add a new command (write operation)**

```go
// pkg/dashboard/bll/commands.go
func NewCommandHandler(ctx context.Context, repo dal.DashboardRepository, req Request) error {
    // Validate input
    // Call repository
    // Return result
}
```

**2. Add a new query (read operation)**

```go
// pkg/dashboard/bll/queries.go
func NewQueryHandler(ctx context.Context, repo dal.DashboardRepository, req Request) (Result, error) {
    // Call repository
    // Transform data if needed
    // Return result
}
```

**3. Expose via API**

Add to gRPC server or WebSocket handler in `pkg/dashboard/delivery/`

## 🐳 Deployment

### Using Docker

**Build Image**
```bash
docker build -t goboard:latest -f deploy/docker/Dashboard.Dockerfile .
```

**Run Container**
```bash
docker run -d \
  --name goboard \
  -p 8080:8080 \
  -p 50051:50051 \
  -e REDIS_ADDR=redis:6379 \
  -e AUTH_SECRET=your-secret \
  -e CORS_ALLOWED_ORIGINS=https://yourdomain.com \
  goboard:latest
```

### Using Docker Compose

```yaml
version: '3.8'

services:
  goboard:
    build:
      context: .
      dockerfile: deploy/docker/Dashboard.Dockerfile
    ports:
      - "8080:8080"
      - "50051:50051"
    environment:
      - REDIS_ADDR=redis:6379
      - AUTH_SECRET=${AUTH_SECRET}
      - CORS_ALLOWED_ORIGINS=${CORS_ALLOWED_ORIGINS}
    depends_on:
      - redis

  redis:
    image: redis:7.2-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes

volumes:
  redis_data:
```

**Deploy:**
```bash
docker-compose up -d
```

### Helm Chart

> 📦 **Coming Soon**: Kubernetes Helm chart for production deployments.
>
> Stay tuned for:
> - High availability setup
> - Horizontal pod autoscaling
> - Redis Sentinel integration
> - Prometheus metrics
> - Grafana dashboards

Want to contribute the Helm chart? See [Contributing](#-contributing)!

---

## 🧪 Testing

### Run All Tests
```bash
go test ./...
```

### Run with Coverage
```bash
go test -cover ./...

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Coverage Reporting

This project uses [Codecov](https://codecov.io/) for coverage reporting.

**Setup for your fork:**

1. **Sign up for Codecov** at https://codecov.io (free for open source)
2. **Add your repository** to Codecov (it auto-detects GitHub repos)
3. **Get your upload token** from Codecov dashboard
4. **Add to GitHub Secrets**:
    - Go to repository Settings → Secrets and variables → Actions
    - Click "New repository secret"
    - Name: `CODECOV_TOKEN`
    - Value: `<your-codecov-token>`

5. **GitHub Actions will automatically**:
    - Run tests with coverage on every push
    - Upload results to Codecov
    - Comment coverage diff on PRs

**View coverage:**
- Badge: Already in README (updates automatically)
- Dashboard: https://codecov.io/gh/YOUR_USERNAME/go-board
- PR Comments: Automatic coverage diff on pull requests

### GitHub Actions Setup

**Already configured!** GitHub Actions works out of the box:

1. **Fork the repository**
2. **Push to main branch**
3. **Actions run automatically**

**No additional setup needed** - it just works! 🎉

**What runs automatically:**
- ✅ **Linting** (golangci-lint)
- ✅ **Tests** (with Redis service)
- ✅ **Coverage** (uploaded to Codecov)
- ✅ **Build** (compiles binary)
- ✅ **Docker Build** (tests Docker image)
- ✅ **Benchmarks** (on PRs)

**View pipeline status:**
- Badge in README (shows latest status)
- Actions tab: https://github.com/YOUR_USERNAME/go-board/actions

**Badges in README:**
```markdown
[![CI](https://github.com/YOUR_USERNAME/go-board/actions/workflows/ci.yml/badge.svg)](https://github.com/YOUR_USERNAME/go-board/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/YOUR_USERNAME/go-board/branch/main/graph/badge.svg)](https://codecov.io/gh/YOUR_USERNAME/go-board)
```

### Release Pipeline

When you create a git tag, the release workflow automatically:
1. Builds binaries for all platforms (Linux, macOS, Windows)
2. Creates GitHub Release with changelog
3. Uploads binaries
4. Builds and pushes Docker images to GitHub Container Registry

**Create a release:**
```bash
git tag v1.0.0
git push origin v1.0.0
```

**Result:**
- GitHub Release created at: `https://github.com/YOUR_USERNAME/go-board/releases`
- Docker images at: `ghcr.io/YOUR_USERNAME/go-board:v1.0.0`

### Current Pipeline

**Main CI Pipeline** (`.github/workflows/ci.yml`):
```
lint → test → build → docker
  ↓      ↓       ↓       ↓
  ✅     ✅      ✅      ✅
```

**Release Pipeline** (`.github/workflows/release.yml`):
```
On tag push → Build all platforms → Create GitHub Release → Push Docker images
```

**What each job does:**
- **lint**: Runs golangci-lint for code quality
- **test**: Runs all tests with Redis, uploads coverage
- **build**: Compiles binary, uploads artifact
- **docker**: Builds Docker image (with caching)
- **benchmark**: Runs benchmarks on PRs, comments results

### Run Specific Tests
```bash
# Test specific package
go test ./pkg/dashboard/auth/...

# Test specific function
go test -run TestJWTVerifier ./pkg/dashboard/auth/...

# Verbose output
go test -v ./...
```

### Benchmarks
```bash
go test -bench=. ./pkg/dashboard/auth/...
```

### Test Structure

```go
func TestFeatureName(t *testing.T) {
    // Arrange: Set up test data
    repo := setupTestRepo()
    
    // Act: Execute the function
    result, err := MyFunction(repo, input)
    
    // Assert: Verify results
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

## 📊 Performance

### Benchmarks

```
BenchmarkStaticVerifier_Verify-8      10000000    115 ns/op
BenchmarkJWTVerifier_Verify-8          100000   15234 ns/op
BenchmarkRedisIncrement-8              50000    32145 ns/op
BenchmarkRedisGetRank-8               100000    12456 ns/op
```

### Optimization Tips

1. **Use Batch Operations**
   ```go
   // Instead of
   for _, member := range members {
       repo.AddMemberToDashboard(ctx, dashboard, member.ID, member.Score)
   }
   
   // Use
   repo.AddMembersBatch(ctx, dashboard, members)
   ```

2. **Pipeline Redis Commands**
   ```go
   pipe := client.Pipeline()
   for _, cmd := range commands {
       pipe.ZIncrBy(ctx, key, value, member)
   }
   pipe.Exec(ctx)
   ```

3. **Connection Pooling**
   Redis client automatically manages connection pool. Reuse the client instance.

## 🔒 Security Best Practices

1. **Use JWT Authentication in Production**
   ```bash
   AUTH_MODE=jwt
   JWT_SECRET=<strong-secret-at-least-32-chars>
   ```

2. **Configure CORS Properly**
   ```bash
   # Specific origins only
   CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com
   ```

3. **Use TLS/SSL**
   ```go
   // Configure TLS for gRPC
   creds, _ := credentials.NewServerTLSFromFile("cert.pem", "key.pem")
   grpcServer := grpc.NewServer(grpc.Creds(creds))
   ```

4. **Rate Limiting**
   Consider adding rate limiting middleware (not implemented yet)

5. **Secrets Management**
   Use environment variables or secret management tools (Vault, AWS Secrets Manager)

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Add tests for new features
- Update documentation

### Commit Messages

```
feat: add new feature
fix: bug fix
docs: documentation update
test: add tests
refactor: code refactoring
```

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👤 Author

**Nikita Okhotnikov**

- GitHub: [@jassus213](https://github.com/jassus213)

## 🙏 Acknowledgments

- [Uber Fx](https://github.com/uber-go/fx) - Dependency injection framework
- [go-redis](https://github.com/redis/go-redis) - Redis client for Go
- [Gorilla WebSocket](https://github.com/gorilla/websocket) - WebSocket implementation
- [gRPC](https://grpc.io/) - RPC framework
- [Zap](https://github.com/uber-go/zap) - Fast, structured logging

## 📮 Support

For issues and questions:
- Open an [Issue](https://github.com/jassus213/go-board/issues)
- Check existing [Issues](https://github.com/jassus213/go-board/issues) first

---

**Built with ❤️ using Go and Redis**