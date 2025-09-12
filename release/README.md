# Dubbo Invoke 跨平台可执行文件

本目录包含了适用于不同操作系统和架构的 Dubbo Invoke 工具可执行文件。

## 文件说明

| 文件名 | 操作系统 | 架构 | 说明 |
|--------|----------|------|------|
| `invoke-linux-amd64` | Linux | x86_64 | 适用于大多数Linux发行版 |
| `invoke-darwin-amd64` | macOS | x86_64 | 适用于Intel芯片的Mac |
| `invoke-darwin-arm64` | macOS | ARM64 | 适用于Apple Silicon芯片的Mac |
| `invoke-windows-amd64.exe` | Windows | x86_64 | 适用于64位Windows系统 |

## 使用方法

### Linux
```bash
# 添加执行权限
chmod +x invoke-linux-amd64

# 运行程序
./invoke-linux-amd64 --help

# 启动Web UI
./invoke-linux-amd64 web --port 8080
```

### macOS (Intel)
```bash
# 添加执行权限
chmod +x invoke-darwin-amd64

# 运行程序
./invoke-darwin-amd64 --help

# 启动Web UI
./invoke-darwin-amd64 web --port 8080
```

### macOS (Apple Silicon)
```bash
# 添加执行权限
chmod +x invoke-darwin-arm64

# 运行程序
./invoke-darwin-arm64 --help

# 启动Web UI
./invoke-darwin-arm64 web --port 8080
```

### Windows
```cmd
# 直接运行
invoke-windows-amd64.exe --help

# 启动Web UI
invoke-windows-amd64.exe web --port 8080
```

## 功能特性

- ✅ 支持中文参数处理（已修复乱码问题）
- ✅ 支持ZooKeeper和Nacos注册中心
- ✅ 提供Web UI界面
- ✅ 支持泛化调用
- ✅ 支持多种参数类型
- ✅ 调用历史记录

## 版本信息

当前版本：1.0.0

查看版本信息：
```bash
./invoke-[platform] version
```

## 注意事项

1. **macOS用户**：首次运行时可能会提示"无法验证开发者"，请在系统偏好设置 > 安全性与隐私中允许运行
2. **Windows用户**：可能会被杀毒软件误报，请添加到白名单
3. **Linux用户**：确保系统已安装必要的运行时库

## 技术支持

如遇到问题，请检查：
1. 操作系统和架构是否匹配
2. 是否有足够的执行权限
3. 网络连接是否正常
4. 注册中心地址是否可达