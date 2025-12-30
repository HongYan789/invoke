# Dubbo Invoke 使用说明

一个支持命令行与 Web UI 的 Dubbo 泛化调用工具，内置智能参数解析、类型提示修正、注册中心环境选择与结果可视化。

## 最新版本更新 (v2.0)

Dubbo 测试工具进行了重大升级，主要更新如下：

1.  **页面更简洁**：默认启用全新的精简版 Web UI，专注于服务调用与结果展示，操作更加流畅高效。
    *   保留了老版（完整版）入口，可通过页面底部链接跳转。
2.  **支持复杂的 Dubbo 参数调用**：
    *   全面升级了参数解析引擎，支持复杂的嵌套对象、泛型集合等复杂结构。
    *   支持大整数（如 `Long` 类型 ID）的无损传输与展示，解决了 JavaScript `Number` 类型精度丢失问题。
3.  **优化数字类型跟字符串类型错误匹配的问题**：
    *   增强了类型推断机制，智能识别并修正数字与字符串之间的类型不匹配问题，减少了 `argument type mismatch` 错误。

## 功能总览

- 支持 Zookeeper、Nacos、Dubbo 直连等多种注册中心
- Web UI 与 CLI 双模式，均可执行泛化调用
- 表达式格式与传统格式两种参数输入方式
- 大整数与 `Java long` 字面量精度保留
- 基于类型提示的参数修正（字符串强制保留、复杂对象递归修正）
- 预置 Zookeeper 环境，快速选择正确地址

## 快速开始

- 启动 Web UI
  - `go run .` 或 `dubbo-invoke web`
  - 访问 `http://localhost:8080`
- 连接注册中心
  - 在 Web UI 顶部选择注册中心类型
  - 选择 Zookeeper 的环境，如“开发环境 (dev)”会自动填入 `10.7.8.40:2181`
  - 构造完整地址例如：`zookeeper://10.7.8.40:2181`

## 调用方式

- 表达式格式（推荐）
  - 示例：`invoke com.example.Service.method({"class":"com.example.Param","districtCode":"35058300"})`
  - Web UI 中的表达式框支持对象、数组、字符串等；类型将自动推断并随请求传递
- 传统格式（兼容）
  - 服务名、方法名分别输入
  - 参数以 JSON 数组输入，如 `[{"class":"com.example.Param","districtCode":35058300}]`
  - 在“类型”框填写参数类型列表，例如 `com.example.Param`

## 参数修正与精度

- 类型提示优先生效
  - 当类型提示为 `java.lang.String` 时，参数将被强制按字符串处理
  - 当类型提示为 `java.lang.Object` 或自定义类（如 `com.xxx.Param`）时，复杂对象中的数值会递归转为字符串，避免 Java Bean 设值时发生类型不匹配
- 大整数与 `Java long` 字面量
  - 前端参数解析保留 16 位及以上整数为字符串，避免精度丢失
  - 后端使用 `json.Number` 解析并在返回结果中保留大整数精度
- 传统与表达式两种格式均已启用上述修正策略

## 复杂对象示例与错误修复

- 问题示例
  - 错误：`argument type mismatch`（如为 Java Bean 属性 `districtCode` 期望 `String` 却传入 `Integer`）
- 解决方式
  - 表达式格式：在对象中将 `districtCode` 明确写为字符串
    - `invoke com.xxx.Api.dzsyRegister({"class":"com.xxx.DzsyRegParam","districtCode":"35058300"})`
  - 传统格式：在“类型”中填写 `com.xxx.DzsyRegParam`，工具会递归把对象里的数值转为字符串，消除类型不匹配

## CLI 用法

- 传统格式
  - `dubbo-invoke invoke com.xxx.Api dzsyRegister '{"class":"com.xxx.DzsyRegParam","districtCode":35058300}' -T com.xxx.DzsyRegParam`
- 表达式格式
  - `dubbo-invoke invoke 'com.xxx.Api.dzsyRegister({"class":"com.xxx.DzsyRegParam","districtCode":"35058300"})'`
- 常用全局参数
  - `-r` 指定注册中心地址：`-r zookeeper://10.7.8.40:2181`
  - `-a` 应用名：`-a dubbo-invoke-client`
  - `-t` 超时毫秒：`-t 10000`

## Web API（便于验证）

- 列服务：`GET /api/list?registry=zookeeper://10.7.8.40:2181&app=dubbo-invoke-client&timeout=10000`
- 调用：`POST /api/invoke`（JSON 包含 `serviceName`、`methodName`、`parameters`、`types` 等）
- 精度测试：`GET /api/test-precision`

## 预置 Zookeeper 环境

| 环境 | 地址 | 服务路径 |
|---|---|---|
| dev | 10.7.8.40:2181 | dubbo |
| uat | 10.7.8.42:2181 | uat |
| tat | 10.6.12.153:2181 | tat |
| fat | 10.6.12.205:2181 | fat |
| pre | mse-4ec83a20-zk.mse.aliyuncs.com:2181 | pre |
| prod | mse-2cd54c90-zk.mse.aliyuncs.com:2181 | prod |

在 Web UI 选择环境后会自动填入地址与服务路径；Zookeeper 会以服务路径作为 `namespace` 构建真实查询路径。

## 关键实现位置

- Web 参数修正：`web_server.go:917` 的 `applyTypeHints`
- 传统格式解析：`commands.go:269` 的 `parseParams`
- 大整数精度处理：`web_server.go:871` 的 `convertJSONNumber`
- 构建真实调用命令：`web_server.go:607` 的 `buildDubboInvokeCommand`
- ZK 环境接口：`web_server.go:3760` 的 `handleZkEnvironments`

## 启动与验证

- 启动 Web UI：`go run .`
- 打开浏览器访问：`http://localhost:8080`（默认进入精简版页面）
  - 如需访问完整版页面，请点击页面底部的“体验完整版”链接或访问 `http://localhost:8080/full`
- 选择 Zookeeper → 开发环境 (dev) → 地址自动填入 `10.7.8.40:2181`
- 使用表达式或传统格式进行调用，并观察结果面板中大整数与类型修正效果

## 项目构建

如果你需要编译生成可执行文件（如 `dubbo-invoke` 或 `dubbo-invoke.exe`），可以使用以下方式：

### 1. 基础构建 (当前系统)

```bash
# 构建默认名称 (dubbo-invoke)
go build .

# 如果你想构建为 dubbo-invoke-cli (仅名字不同，功能一致)
go build -o dubbo-invoke-cli .
```

构建完成后，当前目录下会生成 `dubbo-invoke` 或 `dubbo-invoke-cli` 可执行文件。这些文件通常用于**本地开发调试**。

### 2. 使用 Makefile (推荐)

项目提供了 `Makefile`，支持多种平台的构建目标：

- **默认构建**: `make` 或 `make build`
- **构建 Linux 版本**: `make build-linux`
- **构建 macOS 版本**: `make build-darwin` (Intel) / `make build-darwin-arm64` (Apple Silicon)
- **构建 Windows 版本**: `make build-windows`
- **一键构建所有常用平台**: `make build-quick` (产物在 `release/` 目录)

### 3. 使用脚本构建

如果你没有安装 `make`，也可以直接运行脚本：

```bash
./build_quick.sh
```

该脚本会编译 Linux (amd64), macOS (amd64/arm64), Windows (amd64) 的可执行文件并输出到 `release/` 目录。

## 文件说明

- **根目录下的文件** (`dubbo-invoke`, `dubbo-invoke-cli`):
  - 通常由你在本地手动执行 `go build` 生成。
  - 仅适用于你当前的操作系统架构（例如 macOS）。
  - 用于开发过程中的快速验证和使用。

- **release/ 目录下的文件**:
  - 由构建脚本生成。
  - 包含了适用于不同操作系统（Windows, Linux, macOS）的版本。
  - 用于分发给其他用户使用。
