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

### 🚀 核心功能
- ✅ **Dubbo泛化调用**：支持无需接口类的服务调用
- ✅ **多注册中心支持**：ZooKeeper、Nacos
- ✅ **中文参数处理**：完美支持中文参数，无乱码问题
- ✅ **多种调用格式**：传统格式和表达式格式
- ✅ **参数类型推断**：智能识别和转换参数类型

### 🎨 Web UI界面
- ✅ **现代化界面**：美观易用的Web管理界面
- ✅ **表达式格式优先**：默认使用更直观的表达式格式
- ✅ **服务发现**：自动获取注册中心的服务列表
- ✅ **参数示例生成**：自动生成方法参数示例
- ✅ **调用历史记录**：保存和管理调用历史

### 📊 JSON结果展示增强
- ✅ **树形结构显示**：可折叠/展开的JSON树形展示
- ✅ **多功能工具栏**：
  - 🔄 **压缩/美化切换**：在压缩和格式化JSON之间切换
  - 📝 **行号显示**：显示/隐藏行号功能
  - 🗑️ **清空结果**：一键清空结果区域
  - 💾 **保存结果**：将结果保存为JSON文件
  - 📋 **复制结果**：复制原始JSON数据（不含HTML）
  - 📂 **全部展开/收缩**：一键展开或收缩所有JSON节点
  - 💡 **鼠标悬停提示**：每个图标都有功能说明

### 🔧 技术特性
- ✅ **智能参数解析**：支持复杂参数格式如 `[],[],1`
- ✅ **大整数支持**：正确处理JavaScript大整数
- ✅ **数据完整性**：确保数据传输的完整性和准确性
- ✅ **错误处理**：友好的错误提示和处理机制

## 版本信息

**当前版本：v1.2.0** 🎉

### 🆕 v1.2.0 新特性 (2025-09-19)
- 🎨 **JSON树形展示**：全新的可折叠JSON结果展示
- 🛠️ **多功能工具栏**：6个实用工具提升操作效率
- 🎯 **表达式格式默认**：更符合用户使用习惯
- 🔧 **参数解析优化**：修复复杂参数格式解析问题
- 💾 **数据保存功能**：支持将结果保存为文件
- 📋 **智能复制**：复制纯JSON数据，不含HTML标签

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