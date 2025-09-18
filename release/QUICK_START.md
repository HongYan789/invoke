# Dubbo Invoke v1.2.0 快速开始指南

## 🚀 快速部署

### 1. 下载对应平台的可执行文件

根据你的操作系统选择对应的文件：

| 操作系统 | 架构 | 文件名 | 大小 |
|---------|------|--------|------|
| Linux | x86_64 | `invoke-linux-amd64` | 10M |
| macOS | Intel | `invoke-darwin-amd64` | 10M |
| macOS | Apple Silicon | `invoke-darwin-arm64` | 9.9M |
| Windows | x86_64 | `invoke-windows-amd64.exe` | 11M |

### 2. 启动服务

#### Linux/macOS
```bash
# 添加执行权限
chmod +x invoke-*

# 启动Web UI服务（默认端口8080）
./invoke-linux-amd64
# 或
./invoke-darwin-amd64
# 或
./invoke-darwin-arm64
```

#### Windows
```cmd
# 直接运行
invoke-windows-amd64.exe
```

### 3. 访问Web界面

打开浏览器访问：http://localhost:8080

## 🎯 新功能体验

### JSON树形展示
1. 执行任意Dubbo调用
2. 在结果区域查看新的树形JSON展示
3. 点击节点前的 `▼` 或 `▶` 图标展开/收缩

### 工具栏功能
在"调用结果"标题右侧，你会看到6个功能图标：

| 图标 | 功能 | 说明 |
|------|------|------|
| 🔄 | 压缩/美化 | 切换JSON显示格式 |
| 📝 | 行号 | 显示/隐藏行号 |
| 🗑️ | 清空 | 清空结果区域 |
| 💾 | 保存 | 下载JSON文件 |
| 📋 | 复制 | 复制纯JSON数据 |
| 📂 | 展开/收缩 | 全部展开或收缩 |

### 表达式格式（默认）
现在默认使用表达式格式，更加直观：
```
com.example.service.UserService.getUserById(123)
```

## ⚙️ 配置说明

### 默认配置
- **端口**: 8080
- **注册中心**: zookeeper://127.0.0.1:2181
- **应用名**: dubbo-invoke-client

### 自定义配置
创建 `config.yaml` 文件：
```yaml
# 注册中心配置
registry:
  protocol: zookeeper
  address: 10.7.8.40:2181

# 应用配置
application:
  name: dubbo-invoke-client

# Web服务配置
web:
  port: 8080
  host: 0.0.0.0
```

## 🔧 常见问题

### macOS安全提示
首次运行可能提示"无法验证开发者"：
1. 系统偏好设置 → 安全性与隐私
2. 点击"仍要打开"
3. 或使用命令：`sudo spctl --master-disable`

### Windows杀毒软件误报
将可执行文件添加到杀毒软件白名单。

### Linux依赖问题
确保系统已安装基础运行库：
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install libc6

# CentOS/RHEL
sudo yum install glibc
```

## 📋 使用示例

### 1. 基础调用
```
服务名: com.example.UserService
方法名: getUserById
参数: 123
```

### 2. 表达式格式调用
```
com.example.UserService.getUserById(123)
```

### 3. 复杂参数调用
```
com.example.OrderService.createOrder({"userId": 123, "items": [{"id": 1, "count": 2}]})
```

### 4. 多参数调用
```
com.example.UserService.updateUser(123, {"name": "张三", "age": 25}, true)
```

## 🎉 享受新功能

现在你可以：
- ✅ 使用树形结构查看复杂JSON结果
- ✅ 一键展开/收缩所有JSON节点
- ✅ 复制纯净的JSON数据
- ✅ 将结果保存为文件
- ✅ 切换压缩/美化显示
- ✅ 显示行号便于定位

祝你使用愉快！🚀