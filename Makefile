.PHONY: build test lint run clean proto fmt vet dashboard

BINARY := peerclawd
BUILD_DIR := bin
GO := go
GOFLAGS := -v

dashboard:
	cd web/dashboard && npm install && npm run build

build: dashboard
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/peerclawd

run: build
	./$(BUILD_DIR)/$(BINARY) -config configs/peerclaw.example.yaml

test:
	CGO_ENABLED=1 $(GO) test $(GOFLAGS) ./...

test-cover:
	CGO_ENABLED=1 $(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html *.db

docker-build:
	docker build -t peerclaw-server:latest .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
