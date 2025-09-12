#!/bin/bash

# 跨平台构建脚本
# 为Linux、Mac、Windows生成可执行文件

set -e

echo "🚀 开始构建跨平台可执行文件..."

# 确保release目录存在
mkdir -p release

# 清理之前的构建文件
rm -f release/invoke-*

# 获取版本信息
VERSION=$(grep 'Version.*=' version.go | cut -d'"' -f2 || echo "v1.0.0")
echo "📦 构建版本: $VERSION"

# 构建Linux amd64
echo "🐧 构建Linux amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o release/invoke-linux-amd64 .
echo "✅ Linux amd64 构建完成"

# 构建Mac amd64 (Intel)
echo "🍎 构建Mac amd64 (Intel)..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o release/invoke-darwin-amd64 .
echo "✅ Mac amd64 构建完成"

# 构建Mac arm64 (Apple Silicon)
echo "🍎 构建Mac arm64 (Apple Silicon)..."
# 临时移动resource.syso文件以避免ARM64构建问题
if [ -f "resource.syso" ]; then
    mv resource.syso resource.syso.bak
fi
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o release/invoke-darwin-arm64 .
# 恢复resource.syso文件
if [ -f "resource.syso.bak" ]; then
    mv resource.syso.bak resource.syso
fi
echo "✅ Mac arm64 构建完成"

# 构建Windows amd64
echo "🪟 构建Windows amd64..."
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o release/invoke-windows-amd64.exe .
echo "✅ Windows amd64 构建完成"

echo ""
echo "🎉 所有平台构建完成！"
echo "📁 构建文件位置: release/"
echo ""
echo "📋 构建文件列表:"
ls -la release/invoke-*
echo ""
echo "📊 文件大小统计:"
du -h release/invoke-*