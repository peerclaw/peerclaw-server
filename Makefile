.PHONY: build test lint run clean proto fmt vet dashboard install uninstall

BINARY := peerclawd
BUILD_DIR := bin
GO := go
GOFLAGS := -v

dashboard:
	cd web/app && npm install && npm run build

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

install: build
	sudo deploy/systemd/install.sh $(BUILD_DIR)/$(BINARY)

uninstall:
	sudo deploy/systemd/uninstall.sh

docker-build:
	docker build -t peerclaw-server:latest .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
