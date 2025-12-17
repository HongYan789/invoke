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
build-windows: build-windows-icon

# Windows 构建（带图标）
.PHONY: build-windows-icon
build-windows-icon: deps
	@echo "正在构建带图标的 Windows 版本..."
	@echo "生成资源文件..."
	@if [ -f "$(HOME)/go/bin/goversioninfo" ]; then \
		$(HOME)/go/bin/goversioninfo -icon=icons/dubbo.ico versioninfo.json; \
	else \
		echo "警告: goversioninfo 未找到，将构建无图标版本"; \
	fi
	@echo "编译 Windows exe..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(APP_NAME).exe .

# Windows 构建（32位）
.PHONY: build-windows-32
build-windows-32: deps
	@echo "正在构建 Windows 32位版本..."
	GOOS=windows GOARCH=386 go build $(LDFLAGS) -o $(APP_NAME)-32.exe .

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

# macOS ARM64 构建 (Apple Silicon)
.PHONY: build-darwin-arm64
build-darwin-arm64: deps
	@echo "正在构建 macOS ARM64 版本..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(APP_NAME)-darwin-arm64 .

# Linux ARM64 构建
.PHONY: build-linux-arm64
build-linux-arm64: deps
	@echo "正在构建 Linux ARM64 版本..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(APP_NAME)-linux-arm64 .

# FreeBSD 构建
.PHONY: build-freebsd
build-freebsd: deps
	@echo "正在构建 FreeBSD 版本..."
	GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(APP_NAME)-freebsd .

# 快速构建常用平台
.PHONY: build-quick
build-quick: deps
	@echo "🚀 快速构建常用平台..."
	@mkdir -p release
	@rm -f release/$(APP_NAME)-*
	@echo "🐧 构建 Linux amd64..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o release/$(APP_NAME)-linux-amd64 .
	@echo "🍎 构建 macOS amd64..."
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o release/$(APP_NAME)-darwin-amd64 .
	@echo "🍎 构建 macOS arm64..."
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o release/$(APP_NAME)-darwin-arm64 .
	@echo "🪟 构建 Windows amd64..."
	@if [ -f "$(HOME)/go/bin/goversioninfo" ]; then \
		$(HOME)/go/bin/goversioninfo -icon=icons/dubbo.ico versioninfo.json; \
		GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o release/$(APP_NAME)-windows-amd64.exe .; \
		rm -f resource.syso; \
	else \
		GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o release/$(APP_NAME)-windows-amd64.exe .; \
	fi
	@echo "✅ 快速构建完成！"
	@ls -la release/$(APP_NAME)-*

# 完整跨平台构建
.PHONY: build-release
build-release:
	@echo "🚀 开始完整跨平台构建..."
	@chmod +x build_release.sh
	@./build_release.sh

# 交叉编译所有平台
.PHONY: build-all
build-all: build-windows build-windows-32 build-linux build-linux-arm64 build-darwin build-darwin-arm64 build-freebsd
	@echo "所有平台构建完成"

# 清理
.PHONY: clean
clean:
	@echo "正在清理构建文件..."
	rm -f $(APP_NAME) $(APP_NAME).exe $(APP_NAME)-32.exe $(APP_NAME)-linux $(APP_NAME)-darwin
	rm -f $(APP_NAME)-*
	rm -rf release/
	rm -f resource.syso resource.syso.bak

# 清理发布文件
.PHONY: clean-release
clean-release:
	@echo "正在清理发布文件..."
	rm -rf release/

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
	@echo "🔧 Dubbo Invoke CLI 构建工具"
	@echo ""
	@echo "📦 构建目标:"
	@echo "  build             - 构建当前平台版本"
	@echo "  build-quick       - 快速构建常用平台 (Linux, macOS, Windows)"
	@echo "  build-release     - 完整跨平台构建 (使用 build_release.sh)"
	@echo "  build-all         - 构建所有支持的平台"
	@echo ""
	@echo "🎯 单平台构建:"
	@echo "  build-windows     - 构建 Windows amd64"
	@echo "  build-windows-icon - 构建带图标的 Windows 版本"
	@echo "  build-windows-32  - 构建 Windows 32位"
	@echo "  build-linux       - 构建 Linux amd64"
	@echo "  build-linux-arm64 - 构建 Linux ARM64"
	@echo "  build-darwin      - 构建 macOS amd64 (Intel)"
	@echo "  build-darwin-arm64 - 构建 macOS ARM64 (Apple Silicon)"
	@echo "  build-freebsd     - 构建 FreeBSD amd64"
	@echo ""
	@echo "🛠️  开发工具:"
	@echo "  deps              - 安装依赖"
	@echo "  fmt               - 格式化代码"
	@echo "  lint              - 代码检查"
	@echo "  example           - 运行示例"
	@echo ""
	@echo "🧹 清理工具:"
	@echo "  clean             - 清理构建文件"
	@echo "  clean-release     - 清理发布文件"
	@echo ""
	@echo "⚙️  系统安装:"
	@echo "  install           - 安装到系统"
	@echo "  uninstall         - 从系统卸载"
	@echo ""
	@echo "💡 推荐使用:"
	@echo "  make build-quick  - 快速构建常用平台"
	@echo "  make build-release - 完整发布构建"