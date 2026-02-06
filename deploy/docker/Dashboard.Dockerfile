FROM golang:1.24-alpine AS builder

WORKDIR /build
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
COPY pkg/dashboard/go.mod pkg/dashboard/go.sum pkg/dashboard/
RUN go mod download

COPY . .

RUN go build -mod=readonly -ldflags="-s -w" \
    -o /app/dashboard-server \
    ./cmd/dashboard-server/main.go

FROM scratch
COPY --from=builder /app/dashboard-server /dashboard-server
ENTRYPOINT ["/dashboard-server"]
