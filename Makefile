# Dubbo Invoke CLI Makefile

# 变量定义
APP_NAME=dubbo-invoke
VERSION=1.0.0
BUILD_TIME=$(shell date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 编译参数
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# 默认目标
.PHONY: all
all: build

# 安装依赖
.PHONY: deps
deps:
	@echo "正在安装依赖..."
	go mod tidy
	go mod download

# 构建
.PHONY: build
build: deps
	@echo "正在构建 $(APP_NAME)..."
	go build $(LDFLAGS) -o $(APP_NAME) .

# Windows 构建
.PHONY: build-windows
build-windows: deps
	@echo "正在构建 Windows 版本..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(APP_NAME).exe .

# Linux 构建
.PHONY: build-linux
build-linux: deps
	@echo "正在构建 Linux 版本..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(APP_NAME)-linux .

# macOS 构建
.PHONY: build-darwin
build-darwin: deps
	@echo "正在构建 macOS 版本..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(APP_NAME)-darwin .

# 交叉编译所有平台
.PHONY: build-all
build-all: build-windows build-linux build-darwin
	@echo "所有平台构建完成"

# 清理
.PHONY: clean
clean:
	@echo "正在清理构建文件..."
	rm -f $(APP_NAME) $(APP_NAME).exe $(APP_NAME)-linux $(APP_NAME)-darwin

# 删除test目标
# # 测试
# .PHONY: test
# test:
# 	@echo "正在运行测试..."
# 	go test -v ./...

# 格式化代码
.PHONY: fmt
fmt:
	@echo "正在格式化代码..."
	go fmt ./...

# 代码检查
.PHONY: lint
lint:
	@echo "正在进行代码检查..."
	golangci-lint run

# 运行示例
.PHONY: example
example: build
	@echo "运行示例..."
	./$(APP_NAME) --help

# 安装到系统
.PHONY: install
install: build
	@echo "正在安装到系统..."
	cp $(APP_NAME) /usr/local/bin/

# 卸载
.PHONY: uninstall
uninstall:
	@echo "正在从系统卸载..."
	rm -f /usr/local/bin/$(APP_NAME)

# 显示帮助
.PHONY: help
help:
	@echo "可用的 make 目标:"
	@echo "  build         - 构建当前平台版本"
	@echo "  build-windows - 构建 Windows 版本"
	@echo "  build-linux   - 构建 Linux 版本"
	@echo "  build-darwin  - 构建 macOS 版本"
	@echo "  build-all     - 构建所有平台版本"
	@echo "  deps          - 安装依赖"
	# 删除test相关的帮助信息
	# @echo "  test          - 运行测试"
	@echo "  fmt           - 格式化代码"
	@echo "  lint          - 代码检查"
	@echo "  clean         - 清理构建文件"
	@echo "  install       - 安装到系统"
	@echo "  uninstall     - 从系统卸载"
	@echo "  example       - 运行示例"
	@echo "  help          - 显示此帮助信息"