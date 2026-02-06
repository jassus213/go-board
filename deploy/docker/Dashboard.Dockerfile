FROM golang:alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED 0

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o /app cmd/dashboard-server/main.go

FROM scratch

WORKDIR /app
COPY --from=builder /app /app

EXPOSE 8080

CMD ["./app"]