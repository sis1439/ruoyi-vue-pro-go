.PHONY: all build build-linux run dev deps wire gen test verify clean help setup lint vet ci

APP_NAME = server
CMD_PATH = cmd/server/main.go
WIRE_GEN_PATH = cmd/server/wire_gen.go

# 默认目标
all: build

# 编译项目
build: gen
	@echo "Building $(APP_NAME)..."
	go build -o $(APP_NAME) $(CMD_PATH) $(WIRE_GEN_PATH)

# 直接运行 (如果不使用 wire_gen.go，请确保 wire.go 不被编译排除，但通常 wire.go 有 build tag wireinject)

build-linux: gen
	@echo "Building $(APP_NAME)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(APP_NAME)-linux $(CMD_PATH) $(WIRE_GEN_PATH)

run: gen
	@echo "Running $(APP_NAME)..."
	go run $(CMD_PATH) $(WIRE_GEN_PATH)

# 使用 air 热重载运行
dev:
	@if ! command -v air > /dev/null; then \
		echo "Installing air..."; \
		go install github.com/air-verse/air@latest; \
	fi
	air

# 下载依赖
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify

# 重新生成 wire 依赖注入
wire:
	@echo "Regenerating wire..."
	cd cmd/server && go run github.com/google/wire/cmd/wire@v0.7.0

# 重新生成 GORM DAO 代码
gen:
	@echo "Generating DAO code..."
	go run cmd/gen/generate.go

test: gen
	go test ./...

verify: gen
	go mod verify
	go build ./...
	go vet ./...
	go test ./...

vet: gen
	go vet ./...

# 静态检查。直接跑固定版本，不复用 PATH 上的任意版本，避免结果漂移
lint: gen
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 run ./...

# CI 入口：生成代码 -> 编译 -> 静态检查 -> 单元测试
ci: gen build vet test

# 清理构建产物
clean:
	@echo "Cleaning..."
	rm -f $(APP_NAME)
	rm -rf tmp

# 帮助信息
help:
	@echo "Available commands:"
	@echo "  make build  - Build the application"
	@echo "  make run    - Run the application directly"
	@echo "  make dev    - Run with air (live reload)"
	@echo "  make deps   - Clean and download dependencies"
	@echo "  make wire   - Regenerate wire dependencies"
	@echo "  make gen    - Generate GORM DAO code"
	@echo "  make test   - Generate DAO and run tests"
	@echo "  make verify - Verify dependencies, build, vet and test"
	@echo "  make lint   - Run golangci-lint"
	@echo "  make ci     - gen + build + vet + test"
	@echo "  make clean  - Clean build artifacts"
