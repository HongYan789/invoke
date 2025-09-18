# Dubbo Invoke Tool

一个功能强大的Dubbo服务泛化调用工具，支持命令行和Web UI两种使用方式，兼容Windows、macOS和Linux平台。

[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org/)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey.svg)](https://github.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 功能特性

- 🚀 **泛化调用**: 支持动态调用Dubbo服务，无需依赖接口定义
- 🔧 **多注册中心**: 支持Zookeeper、Nacos等主流注册中心
- 📝 **配置管理**: 支持配置文件管理，简化重复操作
- 🎯 **类型推断**: 自动推断参数类型，支持复杂对象和数组
- 💡 **示例生成**: 自动生成参数示例，快速上手
- 📋 **服务发现**: 列出可用服务和方法
- 🔍 **详细日志**: 支持详细模式，便于调试
- 🌐 **Web UI**: 提供图形化界面，支持浏览器访问

## 快速开始

### Windows双击启动（推荐）

在Windows环境下，您可以直接双击 `dubbo-invoke.exe` 文件启动Web UI界面：

1. 双击 `dubbo-invoke.exe` 文件
2. 程序会自动启动Web服务并在默认浏览器中打开界面
3. 命令行窗口会自动保持打开状态，无需手动操作
4. 程序会每30秒显示一次运行状态，确保服务正常运行
5. 使用 `Ctrl+C` 可以安全停止服务

或者使用批处理文件启动：
1. 双击 `start-web-ui.bat` 文件
2. 程序会自动启动Web服务并在默认浏览器中打开界面

### 1. 基本调用

#### 传统格式
```bash
# 调用用户服务的getUserById方法
./dubbo-invoke invoke com.example.UserService getUserById \
  --registry nacos://127.0.0.1:8848 \
  --app test-app \
  --types java.lang.Long \
  123
```

#### 新格式（表达式调用）
```bash
# 简单参数调用
./dubbo-invoke invoke 'com.example.UserService.getUserById(123)'

# 复杂对象参数调用
./dubbo-invoke invoke 'com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi.getCompanyInfoFromDb({"class":"com.jzt.zhcai.user.companyinfo.dto.request.UserCompanyInfoDetailReq","companyId":1})'

# 无参数调用
./dubbo-invoke invoke 'com.example.UserService.getAllUsers()'

# 多参数调用
./dubbo-invoke invoke 'com.example.UserService.updateUser({"id":1,"name":"张三"}, true)'
```

### 2. 自动类型推断

```bash
# 系统会自动推断参数类型
./dubbo-invoke invoke com.example.UserService updateUser \
  --registry nacos://127.0.0.1:8848 \
  --app test-app \
  '{"id":1,"name":"张三"}' true
```

### 3. 复杂参数调用

```bash
# 支持数组和对象参数
./dubbo-invoke invoke com.example.UserService batchUpdate \
  --registry nacos://127.0.0.1:8848 \
  --app test-app \
  '[{"id":1,"name":"用户1"},{"id":2,"name":"用户2"}]' \
  '{"updateTime":"2024-01-15 10:30:00","operator":"admin"}'
```

### 4. 使用配置文件

```bash
# 初始化配置文件
./dubbo-invoke config init --config ./my-config.yaml

# 查看配置
./dubbo-invoke config show --config ./my-config.yaml

# 使用配置文件调用
./dubbo-invoke invoke com.example.UserService getUserById \
  --config ./my-config.yaml \
  --types java.lang.Long \
  456
```

### 5. 服务发现

```bash
# 列出所有可用服务
./dubbo-invoke list --registry nacos://127.0.0.1:8848 --app test-app

# 列出服务的所有方法
./dubbo-invoke list com.example.UserService \
  --registry nacos://127.0.0.1:8848 \
  --app test-app
```

### 6. 生成示例参数

```bash
# 生成指定类型的示例参数
./dubbo-invoke invoke com.example.UserService createUser \
  --registry nacos://127.0.0.1:8848 \
  --app test-app \
  --example \
  --types 'java.lang.String,java.lang.Integer,java.lang.Boolean'
```

## 文件说明

- `dubbo-invoke` - macOS/Linux可执行文件
- `dubbo-invoke.exe` - Windows可执行文件
- `start-web-ui.bat` - Windows批处理启动文件
- `test-config.yaml` - 示例配置文件

## 支持的参数类型

- `java.lang.String` - 字符串
- `java.lang.Integer` - 整数
- `java.lang.Long` - 长整数
- `java.lang.Double` - 双精度浮点数
- `java.lang.Float` - 单精度浮点数
- `java.lang.Boolean` - 布尔值
- `java.util.Date` - 日期
- `java.util.Map` - 映射对象
- `java.util.List` - 列表数组

## 注册中心支持

- Zookeeper: `zookeeper://127.0.0.1:2181`
- Nacos: `nacos://127.0.0.1:8848`
- Consul: `consul://127.0.0.1:8500`

## Web UI 功能

Web界面提供了图形化的操作方式：

1. **服务调用**: 通过表单填写服务名、方法名和参数进行调用
2. **服务发现**: 自动列出注册中心中的可用服务
3. **调用历史**: 记录最近的调用历史，支持一键回填
4. **参数示例**: 自动生成参数示例，方便快速上手
5. **结果展示**: 格式化显示调用结果，支持大整数精度保持

## 命令参考

### invoke - 调用服务
```bash
# 传统格式
dubbo-invoke invoke [service] [method] [params...] [flags]

# 新格式（表达式）
dubbo-invoke invoke [expression] [flags]

# 标志:
  -e, --example          生成示例参数
  -G, --generic          使用泛化调用 (default true)
  -g, --group string     服务分组
  -T, --types strings    参数类型列表
  -V, --version string   服务版本

# 表达式格式:
  service.method(param1, param2, ...)
  
# 示例:
  'com.example.UserService.getUserById(123)'
  'com.example.UserService.createUser({"name":"张三","age":25})'
```

### web - 启动Web UI
```bash
# 启动Web UI服务器
dubbo-invoke web [flags]

# 标志:
  -p, --port int      Web服务器端口 (default 8080)
  -t, --timeout int   调用超时时间(毫秒) (default 30000)

# 示例:
  dubbo-invoke web                    # 使用默认端口8080
  dubbo-invoke web --port 9090       # 使用指定端口
```

## 版本信息

当前版本: 1.0.0

```bash
./dubbo-invoke version
```

## 🏗️ 项目构建

### 开发环境要求

- Go 1.19 或更高版本
- Git（用于获取版本信息）
- 支持的操作系统：Windows、macOS、Linux

### 快速构建

```bash
# 克隆项目
git clone <repository-url>
cd invoke

# 安装依赖
go mod tidy

# 构建当前平台版本
go build -o invoke .

# 遇到ARM64兼容性问题，需要临时移除resource.syso文件并重新构建。
mv resource.syso resource.syso.bak && go build -o invoke .

# 构建成功后，恢复resource.syso文件以保持项目完整性。
mv resource.syso.bak resource.syso 

# 或使用 Makefile
make build
```

### 跨平台构建

项目提供了自动化的跨平台构建脚本：

```bash
# 使用构建脚本（推荐）
./build_release.sh

# 或使用 Makefile
make build-all
```

构建完成后，可执行文件将生成在 `release/` 目录中：
- `invoke-linux-amd64` - Linux 64位版本
- `invoke-darwin-amd64` - macOS Intel版本
- `invoke-darwin-arm64` - macOS Apple Silicon版本
- `invoke-windows-amd64.exe` - Windows 64位版本

### 构建参数说明

构建时会自动注入版本信息：
- `Version`: 从 `version.go` 获取或通过 ldflags 注入
- `BuildTime`: 构建时间戳
- `GitCommit`: Git提交哈希（如果可用）

```bash
# 自定义版本构建
go build -ldflags "-s -w -X main.Version=v1.2.0 -X main.BuildTime=$(date +%Y-%m-%d_%H:%M:%S)" -o invoke .
```

## 📁 项目结构

```
invoke/
├── README.md                 # 项目说明文档
├── WINDOWS_USAGE.md         # Windows使用说明
├── Makefile                 # 构建管理文件
├── build_release.sh         # 跨平台构建脚本
├── go.mod                   # Go模块依赖
├── go.sum                   # 依赖校验文件
├── config.yaml              # 默认配置文件
├── versioninfo.json         # Windows版本信息
├── resource.syso            # Windows资源文件
├── start-web-ui.bat         # Windows启动脚本
├── main.go                  # 程序入口
├── commands.go              # 命令行命令定义
├── config.go                # 配置管理
├── version.go               # 版本信息管理
├── utils.go                 # 工具函数
├── web_server.go            # Web服务器实现
├── dubbo_client.go          # Dubbo客户端接口
├── real_dubbo_client.go     # 真实Dubbo客户端实现
├── nacos_client.go          # Nacos注册中心客户端
├── icons/                   # 图标资源
│   ├── dubbo.ico           # Windows图标
│   └── dubbo.png           # 通用图标
├── log/                     # 日志目录
└── release/                 # 发布文件目录
    ├── README.md           # 发布说明
    ├── invoke-linux-amd64  # Linux版本
    ├── invoke-darwin-amd64 # macOS Intel版本
    ├── invoke-darwin-arm64 # macOS ARM版本
    └── invoke-windows-amd64.exe # Windows版本
```

### 核心文件说明

| 文件 | 作用 |
|------|------|
| `main.go` | 程序入口，初始化CLI应用 |
| `commands.go` | 定义所有CLI命令（invoke、web、config等） |
| `web_server.go` | Web UI服务器，包含前端页面和API |
| `dubbo_client.go` | Dubbo客户端抽象接口 |
| `real_dubbo_client.go` | 真实Dubbo服务调用实现 |
| `nacos_client.go` | Nacos注册中心集成 |
| `config.go` | 配置文件管理和解析 |
| `version.go` | 版本信息管理 |
| `utils.go` | 通用工具函数 |
| `resource.syso` | Windows资源文件（图标、版本信息） |
| `build_release.sh` | 自动化跨平台构建脚本 |

## 🚀 开发指南

### 本地开发

```bash
# 启动开发模式
go run . web --port 8080

# 或使用热重载（需要安装air）
air
```

### 代码格式化

```bash
# 格式化代码
make fmt
# 或
go fmt ./...
```

### 代码检查

```bash
# 运行代码检查（需要安装golangci-lint）
make lint
```

### 添加新功能

1. 在 `commands.go` 中添加新的CLI命令
2. 在 `web_server.go` 中添加对应的Web API
3. 更新配置结构（如需要）
4. 添加相应的测试
5. 更新文档

## ⚠️ 重要注意事项

### ARM64兼容性

- **macOS Apple Silicon**: 构建时会自动处理 `resource.syso` 文件兼容性问题
- 构建脚本会在ARM64构建时临时移动 `resource.syso` 文件，构建完成后自动恢复
- 如果手动构建ARM64版本遇到问题，请临时移除 `resource.syso` 文件

### Windows资源文件

- `resource.syso`: 包含Windows图标和版本信息
- `versioninfo.json`: Windows版本信息配置
- 修改Windows图标需要重新生成 `resource.syso` 文件

### 构建优化

- 使用 `-ldflags "-s -w"` 参数减小可执行文件大小
- 生产构建会自动注入版本信息和构建时间
- 支持交叉编译，无需在目标平台构建

### Web UI开发

- 前端代码嵌入在 `web_server.go` 的HTML模板中
- 修改前端代码后需要重新编译Go程序
- JavaScript代码支持表达式格式的参数解析

### 配置管理

- 默认配置文件：`config.yaml`
- 支持通过 `--config` 参数指定自定义配置文件
- 配置文件支持注册中心、应用信息、默认参数等设置

### 日志管理

- 日志文件存储在 `log/` 目录
- 支持详细模式调试（`--verbose` 参数）
- Web UI调用日志会记录在服务器日志中

---

**注意**: 这是一个基于模拟数据的演示工具，实际使用时需要连接真实的Dubbo服务提供者。