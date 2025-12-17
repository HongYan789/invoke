#!/bin/bash

# 快速构建脚本 - 只构建常用平台
set -e

echo "🚀 快速构建常用平台..."

# 确保release目录存在
mkdir -p release

# 清理之前的构建文件
rm -f release/dubbo-invoke-*

# 获取构建信息
VERSION="v1.0.0"
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

echo "📦 版本: $VERSION"
echo "🔧 Git提交: $GIT_COMMIT"
echo ""

# 选择 goversioninfo 可执行路径（用于生成 Windows 资源文件）
GOVERSIONINFO=""
if command -v goversioninfo >/dev/null 2>&1; then
    GOVERSIONINFO=$(command -v goversioninfo)
elif [ -x "$HOME/go/bin/goversioninfo" ]; then
    GOVERSIONINFO="$HOME/go/bin/goversioninfo"
else
    echo "⚠️ goversioninfo 未安装，Windows 资源文件将被跳过"
fi

# 构建Linux amd64
echo "🐧 构建 Linux amd64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o release/dubbo-invoke-linux-amd64 .

# 构建Mac amd64 (Intel)
echo "🍎 构建 Mac amd64 (Intel)..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o release/dubbo-invoke-darwin-amd64 .

# 构建Mac arm64 (Apple Silicon)
echo "🍎 构建 Mac arm64 (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o release/dubbo-invoke-darwin-arm64 .

# 构建Windows amd64
echo "🪟 构建 Windows amd64..."
# 生成 Windows 资源文件（图标、版本信息）
if [ -n "$GOVERSIONINFO" ]; then
    echo "🖼 生成 Windows 资源文件（icons/dubbo.ico）..."
    "$GOVERSIONINFO" -icon=icons/dubbo.ico versioninfo.json || echo "⚠️ 生成资源文件失败，继续构建"
fi
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o release/dubbo-invoke-windows-amd64.exe .

# 清理 resource.syso，避免影响后续非 Windows 构建
if [ -f "resource.syso" ]; then
    rm -f resource.syso
fi

echo ""
echo "✅ 快速构建完成！"
echo "📁 构建文件:"
ls -la release/dubbo-invoke-*
echo ""
echo "📊 文件大小:"
du -h release/dubbo-invoke-*
