module github.com/jassus213/go-board/dashboard-api

go 1.24.0

require (
	github.com/joho/godotenv v1.5.1
	github.com/redis/go-redis/v9 v9.17.3
	google.golang.org/grpc v1.79.3
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

require (
	github.com/jassus213/go-board/dashboard v0.0.0
	go.uber.org/fx v1.24.0
	go.uber.org/zap v1.27.1
)

replace github.com/jassus213/go-board/dashboard => ./dashboard
