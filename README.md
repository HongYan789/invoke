# Dubbo Invoke Tool

一个用于Dubbo服务泛化调用的命令行工具，支持Windows和macOS平台。

## 功能特性

- 🚀 **泛化调用**: 支持动态调用Dubbo服务，无需依赖接口定义
- 🔧 **多注册中心**: 支持Zookeeper、Nacos等主流注册中心
- 📝 **配置管理**: 支持配置文件管理，简化重复操作
- 🎯 **类型推断**: 自动推断参数类型，支持复杂对象和数组
- 💡 **示例生成**: 自动生成参数示例，快速上手
- 📋 **服务发现**: 列出可用服务和方法
- 🔍 **详细日志**: 支持详细模式，便于调试

## 快速开始

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

## 版本信息

当前版本: 1.0.0

```bash
./dubbo-invoke version
```

---

**注意**: 这是一个基于模拟数据的演示工具，实际使用时需要连接真实的Dubbo服务提供者。