# 版本号（可通过 make VERSION=1.0.0 指定）
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_REVISION=$(shell git rev-parse HEAD)
GO_VERSION=$(shell go version)
BUILD_TIME=$(shell date +%Y-%m-%d_%H:%M:%S)

# Go 构建参数
LDFLAGS=-s -w -X 'github.com/dushixiang/pika/pkg/version.Version=$(VERSION)'
GOFLAGS=CGO_ENABLED=0

# 构建前端
build-web:
	cd web && yarn && yarn build

# 构建服务端（开发）
build-server:
	$(GOFLAGS) GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/pika-linux-amd64 cmd/serv/main.go
	upx bin/pika-linux-amd64

build-servers:
	$(GOFLAGS) GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/pika-linux-amd64 cmd/serv/main.go
	$(GOFLAGS) GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/pika-linux-arm64 cmd/serv/main.go

	upx bin/pika-linux-amd64
	upx bin/pika-linux-arm64

# 构建所有平台的 Agent
build-agents:
	@echo "Building agents for all platforms (version: $(VERSION))..."
	@mkdir -p bin/agents

	# Linux
	$(GOFLAGS) GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/agents/pika-agent-linux-amd64 cmd/agent/*.go
	$(GOFLAGS) GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/agents/pika-agent-linux-arm64 cmd/agent/*.go
	$(GOFLAGS) GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="$(LDFLAGS)" -o bin/agents/pika-agent-linux-armv7 cmd/agent/*.go

	# macOS
	$(GOFLAGS) GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/agents/pika-agent-darwin-amd64 cmd/agent/*.go
	$(GOFLAGS) GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/agents/pika-agent-darwin-arm64 cmd/agent/*.go

	# Windows
	$(GOFLAGS) GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/agents/pika-agent-windows-amd64.exe cmd/agent/*.go

	@echo "All agents built successfully!"
	@echo "Compressing agents with UPX..."
	@upx bin/agents/pika-agent-linux-amd64
	@upx bin/agents/pika-agent-linux-arm64
	@upx bin/agents/pika-agent-linux-armv7
	@upx bin/agents/pika-agent-darwin-amd64
	@upx bin/agents/pika-agent-darwin-arm64
	@upx bin/agents/pika-agent-windows-amd64.exe
	@echo "All agents compressed successfully!"
	@ls -lh bin/agents/

# 构建所有
build-release:
	make build-web
	make build-agents
	make build-servers

# 清理编译产物
clean:
	rm -rf bin/*
	rm -rf web/dist

# 运行测试
test:
	go test -v ./...

# 代码格式化
fmt:
	go fmt ./...
	cd web && yarn format

# 代码检查
lint:
	golangci-lint run

# 生成 Wire 代码
wire:
	cd internal && wire

# 显示版本信息
version:
	@echo "Version: $(VERSION)"
	@echo "Git Revision: $(GIT_REVISION)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "Build Time: $(BUILD_TIME)"