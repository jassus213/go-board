PROTO_DIR = pkg/dashboard/delivery/grpc/proto
OUT_DIR = pkg/dashboard/delivery/grpc/gen

PROTO_FILES = $(wildcard $(PROTO_DIR)/*.proto)

.PHONY: all gen deps clean

all: gen

deps:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

gen:
	@echo "Generating Go code..."
	@echo "Proto dir: $(PROTO_DIR)"
	@echo "Out dir:   $(OUT_DIR)"

	if not exist "$(subst /,\,$(OUT_DIR))" mkdir "$(subst /,\,$(OUT_DIR))"

	protoc --proto_path=$(PROTO_DIR) \
		--go_out=$(OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)
	@echo "Done!"

build:
	@echo "Building"

	go build ./...

	@echo "Done!"

test:
	@echo "Running tests"

	go test -v ./... ./pkg/dashboard/...

test-cover:
	go test -coverprofile=coverage.out ./... ./pkg/dashboard/...
	go tool cover -html=coverage.out

clean:
	del /Q "$(subst /,\,$(OUT_DIR))\*.pb.go"