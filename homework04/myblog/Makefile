BINARY := blog-server
JWT_SECRET ?= dev-secret-change-me

.PHONY: run build test gotest fmt vet tidy clean

run: ## 本地启动（SQLite）
	JWT_SECRET=$(JWT_SECRET) go run ./cmd/server

build: ## 编译到 bin/
	go build -o bin/$(BINARY) ./cmd/server

test: ## 运行接口冒烟测试（需先启动服务）
	./tests/api_test.sh

gotest: ## 运行 Go 单元/集成测试（无需启动服务）
	go test ./... -count=1

fmt:
	gofmt -w cmd internal

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/ $(BINARY) *.db *.db-shm *.db-wal
