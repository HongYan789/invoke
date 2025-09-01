package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// CallHistory 调用历史记录
type CallHistory struct {
	ID          string    `json:"id"`
	ServiceName string    `json:"serviceName"`
	MethodName  string    `json:"methodName"`
	Parameters  []string  `json:"parameters"`
	Types       []string  `json:"types"`
	Registry    string    `json:"registry"`
	App         string    `json:"app"`
	Success     bool      `json:"success"`
	Timestamp   time.Time `json:"timestamp"`
	Result      string    `json:"result"`
}

// WebServer Web服务器结构
type WebServer struct {
	port     int
	registry string
	app      string
	timeout  int
	history  []CallHistory // 调用历史记录
}

// InvokeRequest Web调用请求
type InvokeRequest struct {
	ServiceName string   `json:"serviceName"`
	MethodName  string   `json:"methodName"`
	Parameters  []string `json:"parameters"`
	Types       []string `json:"types"`
	Registry    string   `json:"registry"`
	App         string   `json:"app"`
	Timeout     int      `json:"timeout"`
	Group       string   `json:"group"`
	Version     string   `json:"version"`
}

// InvokeResponse Web调用响应
type InvokeResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
	Message string      `json:"message"`
}

// ListServicesResponse 服务列表响应
type ListServicesResponse struct {
	Success  bool     `json:"success"`
	Services []string `json:"services"`
	Error    string   `json:"error"`
}

// ListMethodsResponse 方法列表响应结构
type ListMethodsResponse struct {
	Success bool     `json:"success"`
	Methods []string `json:"methods"`
	Error   string   `json:"error"`
}

// newWebCommand 创建web命令
func newWebCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web",
		Short: "启动Web UI服务器",
		Long: `启动Web UI服务器，提供图形化界面进行Dubbo服务调用

示例:
  dubbo-invoke web                                    # 默认端口8080
  dubbo-invoke web --port 9090                       # 指定端口
  dubbo-invoke web --registry nacos://127.0.0.1:8848 # 指定注册中心
  dubbo-invoke web --timeout 30000                   # 设置超时时间`,
		RunE: runWebCommand,
	}

	cmd.Flags().IntP("port", "p", 8080, "Web服务器端口")
	cmd.Flags().IntP("timeout", "t", 30000, "调用超时时间(毫秒)")

	return cmd
}

// runWebCommand 运行Web服务器
func runWebCommand(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	registry, _ := cmd.Flags().GetString("registry")
	app, _ := cmd.Flags().GetString("app")
	timeout, _ := cmd.Flags().GetInt("timeout")

	server := &WebServer{
		port:     port,
		registry: registry,
		app:      app,
		timeout:  timeout,
	}

	return server.Start()
}

// Start 启动Web服务器
func (ws *WebServer) Start() error {
	// 初始化历史记录
	ws.history = make([]CallHistory, 0)

	// 设置路由
	http.HandleFunc("/", ws.handleIndex)
	http.HandleFunc("/api/invoke", ws.handleInvoke)
	http.HandleFunc("/api/list", ws.handleList)
	http.HandleFunc("/api/methods", ws.handleMethods)
	http.HandleFunc("/api/example", ws.handleExample)
	http.HandleFunc("/api/history", ws.handleHistory)
	http.HandleFunc("/api/test-connection", ws.handleTestConnection)

	addr := fmt.Sprintf(":%d", ws.port)
	color.Green("🚀 Web UI服务器启动成功!")
	color.Cyan("📱 访问地址: http://localhost:%d", ws.port)
	color.Yellow("⚙️  默认注册中心: %s", ws.registry)
	color.Yellow("📦 默认应用名: %s", ws.app)
	fmt.Println()

	return http.ListenAndServe(addr, nil)
}

// handleIndex 处理首页
func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.Must(template.New("index").Parse(indexHTML))
	data := map[string]interface{}{
		"Registry": ws.registry,
		"App":      ws.app,
		"Timeout":  ws.timeout,
	}
	t.Execute(w, data)
}

// handleInvoke 处理服务调用
func (ws *WebServer) handleInvoke(w http.ResponseWriter, r *http.Request) {
	color.Green("[WEB] 收到调用请求: %s %s", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		color.Blue("[WEB] 处理OPTIONS预检请求")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		color.Red("[WEB] 错误: 不支持的HTTP方法 %s", r.Method)
		ws.writeError(w, "只支持POST方法")
		return
	}

	var req InvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		color.Red("[WEB] 错误: 请求参数解析失败 - %v", err)
		ws.writeError(w, "请求参数解析失败: "+err.Error())
		return
	}
	color.Cyan("[WEB] 解析请求参数成功: 服务=%s, 方法=%s, 参数数量=%d", req.ServiceName, req.MethodName, len(req.Parameters))

	// 使用默认值
	if req.Registry == "" {
		req.Registry = ws.registry
		color.Yellow("[WEB] 使用默认注册中心: %s", req.Registry)
	}
	if req.App == "" {
		req.App = ws.app
		color.Yellow("[WEB] 使用默认应用名: %s", req.App)
	}
	if req.Timeout == 0 {
		req.Timeout = ws.timeout
		color.Yellow("[WEB] 使用默认超时时间: %d ms", req.Timeout)
	}

	color.Blue("[WEB] 开始执行Dubbo调用: %s.%s", req.ServiceName, req.MethodName)
	// 执行调用
	result, err := ws.executeInvoke(req)
	
	// 保存调用历史
	history := CallHistory{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		ServiceName: req.ServiceName,
		MethodName:  req.MethodName,
		Parameters:  req.Parameters,
		Types:       req.Types,
		Registry:    req.Registry,
		App:         req.App,
		Success:     err == nil,
		Timestamp:   time.Now(),
	}
	
	if err != nil {
		color.Red("[WEB] 调用失败: %v", err)
		history.Result = err.Error()
		ws.history = append(ws.history, history)
		color.Cyan("[WEB] 已保存失败调用历史, 历史记录总数: %d", len(ws.history))
		ws.writeError(w, err.Error())
		return
	}
	
	// 保存成功结果
	color.Green("[WEB] 调用成功")
	if resultBytes, jsonErr := json.Marshal(result); jsonErr == nil {
		history.Result = string(resultBytes)
		color.Cyan("[WEB] 结果序列化成功, 长度: %d 字符", len(history.Result))
	} else {
		history.Result = fmt.Sprintf("%v", result)
		color.Yellow("[WEB] 结果序列化失败，使用字符串格式: %v", jsonErr)
	}
	ws.history = append(ws.history, history)
	color.Cyan("[WEB] 已保存成功调用历史, 历史记录总数: %d", len(ws.history))

	// 成功时直接返回原始数据，不包装
	w.Header().Set("Content-Type", "application/json")
	
	// 如果result已经是字符串格式的JSON，直接写入
	if resultStr, ok := result.(string); ok {
		// 检查是否是有效的JSON字符串
		var jsonTest interface{}
		if json.Unmarshal([]byte(resultStr), &jsonTest) == nil {
			// 是有效JSON，直接输出
			w.Write([]byte(resultStr))
			return
		}
	}
	
	// 否则进行JSON编码
	json.NewEncoder(w).Encode(result)
}

// handleList 处理服务列表
func (ws *WebServer) handleList(w http.ResponseWriter, r *http.Request) {
	color.Green("[WEB] 收到服务列表请求: %s %s", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 创建dubbo客户端配置
	config := &DubboConfig{
		Registry:    ws.registry,
		Application: ws.app,
		Timeout:     time.Duration(ws.timeout) * time.Millisecond,
	}
	color.Cyan("[WEB] 创建Dubbo客户端配置: 注册中心=%s, 应用=%s, 超时=%dms", config.Registry, config.Application, ws.timeout)

	// 创建真实的dubbo客户端
	client, err := NewRealDubboClient(config)
	if err != nil {
		color.Red("[WEB] 创建Dubbo客户端失败: %v", err)
		response := ListServicesResponse{
			Success: false,
			Error:   fmt.Sprintf("创建dubbo客户端失败: %v", err),
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	defer client.Close()
	color.Blue("[WEB] Dubbo客户端创建成功")

	// 检查连接状态
	color.Blue("[WEB] 检查Dubbo客户端连接状态")
	if !client.IsConnected() {
		color.Red("[WEB] 无法连接到Dubbo注册中心")
		response := ListServicesResponse{
			Success: false,
			Error:   "无法连接到dubbo注册中心",
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	color.Green("[WEB] Dubbo客户端连接成功")

	// 获取真实的服务列表
	color.Blue("[WEB] 开始获取服务列表")
	services, err := client.ListServices()
	if err != nil {
		color.Red("[WEB] 获取服务列表失败: %v", err)
		response := ListServicesResponse{
			Success: false,
			Error:   fmt.Sprintf("获取服务列表失败: %v", err),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ListServicesResponse{
		Success:  true,
		Services: services,
	}

	json.NewEncoder(w).Encode(response)
}

// handleExample 处理示例参数生成
func (ws *WebServer) handleExample(w http.ResponseWriter, r *http.Request) {
	color.Blue("[WEB] 收到示例参数生成请求")
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	typesParam := r.URL.Query().Get("types")
	color.Cyan("[WEB] 获取types参数: %s", typesParam)
	
	if typesParam == "" {
		color.Red("[WEB] 缺少types参数")
		ws.writeError(w, "缺少types参数")
		return
	}

	types := strings.Split(typesParam, ",")
	color.Green("[WEB] 解析types参数成功，类型数量: %d", len(types))
	
	color.Blue("[WEB] 开始生成示例参数")
	examples := generateExampleParams(types)
	color.Green("[WEB] 示例参数生成成功")

	response := map[string]interface{}{
		"success":  true,
		"examples": examples,
	}

	color.Green("[WEB] 示例参数响应发送成功")
	json.NewEncoder(w).Encode(response)
}

// parseParameter 解析参数，支持JSON格式的智能类型推断
func (ws *WebServer) parseParameter(param string) (interface{}, error) {
	color.Cyan("[WEB] 开始解析参数: %s", param)
	
	// 去除首尾空格
	param = strings.TrimSpace(param)
	color.Cyan("[WEB] 去除空格后的参数: %s", param)
	
	// 如果不是JSON格式，直接返回字符串
	if !strings.HasPrefix(param, "{") && !strings.HasPrefix(param, "[") {
		color.Green("[WEB] 参数不是JSON格式，返回原始字符串")
		return param, nil
	}
	
	// 尝试解析为JSON
	color.Blue("[WEB] 尝试解析JSON格式参数")
	var result interface{}
	err := json.Unmarshal([]byte(param), &result)
	if err != nil {
		color.Red("[WEB] JSON解析失败: %v", err)
		return nil, err
	}
	color.Green("[WEB] JSON解析成功")
	
	return result, nil
}

// executeInvoke 执行调用
func (ws *WebServer) executeInvoke(req InvokeRequest) (interface{}, error) {
	color.Blue("[WEB] 开始执行Dubbo调用: %s.%s", req.ServiceName, req.MethodName)
	color.Cyan("[WEB] 调用参数: Registry=%s, App=%s, Timeout=%dms", req.Registry, req.App, req.Timeout)
	
	// 创建Dubbo客户端配置
	cfg := &DubboConfig{
		Registry:    req.Registry,
		Application: req.App,
		Timeout:     time.Duration(req.Timeout) * time.Millisecond,
	}
	color.Green("[WEB] Dubbo客户端配置创建成功")

	// 智能转换参数为interface{}类型，支持类型推断
	color.Blue("[WEB] 开始解析调用参数，参数数量: %d", len(req.Parameters))
	params := make([]interface{}, len(req.Parameters))
	for i, p := range req.Parameters {
		color.Cyan("[WEB] 解析参数[%d]: %s", i, p)
		// 尝试解析JSON格式的参数
		parsedParam, err := ws.parseParameter(p)
		if err != nil {
			// 如果解析失败，使用原始字符串
			color.Yellow("[WEB] 参数[%d]解析失败，使用原始字符串: %v", i, err)
			params[i] = p
		} else {
			color.Green("[WEB] 参数[%d]解析成功", i)
			params[i] = parsedParam
		}
	}
	color.Green("[WEB] 所有参数解析完成")

	// 尝试使用真实的Dubbo客户端
	color.Blue("[WEB] 尝试创建真实Dubbo客户端")
	realClient, err := NewRealDubboClient(cfg)
	if err != nil {
		// 如果真实客户端创建失败，回退到模拟客户端
		color.Red("[WEB] 真实Dubbo客户端创建失败，回退到模拟客户端: %v", err)
		
		// 创建模拟客户端
		color.Blue("[WEB] 尝试创建模拟Dubbo客户端")
		mockClient, mockErr := NewDubboClient(cfg)
		if mockErr != nil {
			color.Red("[WEB] 创建模拟Dubbo客户端失败: %v", mockErr)
			return nil, fmt.Errorf("创建模拟Dubbo客户端失败: %v", mockErr)
		}
		color.Green("[WEB] 模拟Dubbo客户端创建成功")
		defer mockClient.Close()
		
		// 执行模拟调用
		color.Blue("[WEB] 开始执行模拟调用")
		result, err := mockClient.GenericInvoke(req.ServiceName, req.MethodName, req.Types, params)
		if err != nil {
			color.Red("[WEB] 模拟调用失败: %v", err)
			return nil, fmt.Errorf("模拟调用失败: %v", err)
		}
		color.Green("[WEB] 模拟调用成功")
		return result, nil
	}
	color.Green("[WEB] 真实Dubbo客户端创建成功")
	defer realClient.Close()

	// 执行真实的泛化调用
	color.Blue("[WEB] 开始执行真实Dubbo调用")
	result, err := realClient.GenericInvoke(req.ServiceName, req.MethodName, req.Types, params)
	if err != nil {
		color.Red("[WEB] 真实调用失败: %v", err)
		return nil, fmt.Errorf("真实调用失败: %v", err)
	}
	color.Green("[WEB] 真实调用成功")

	return result, nil
}

// handleHistory 处理调用历史
func (ws *WebServer) handleHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		ws.writeError(w, "只支持GET方法")
		return
	}

	// 返回最近的50条历史记录
	historyCount := len(ws.history)
	start := 0
	if historyCount > 50 {
		start = historyCount - 50
	}

	recentHistory := ws.history[start:]

	response := map[string]interface{}{
		"success": true,
		"history": recentHistory,
		"total":   historyCount,
	}

	json.NewEncoder(w).Encode(response)
}

// handleMethods 处理获取服务方法列表
func (ws *WebServer) handleMethods(w http.ResponseWriter, r *http.Request) {
	color.Cyan("[DEBUG] 收到获取方法列表请求")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理OPTIONS预检请求
	if r.Method == "OPTIONS" {
		color.Yellow("[DEBUG] 处理OPTIONS预检请求")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		color.Red("[ERROR] 不支持的HTTP方法: %s", r.Method)
		ws.writeError(w, "只支持GET方法")
		return
	}

	// 获取服务名参数
	serviceName := r.URL.Query().Get("serviceName")
	if serviceName == "" {
		color.Red("[ERROR] 缺少serviceName参数")
		ws.writeError(w, "缺少serviceName参数")
		return
	}

	color.Green("[DEBUG] 获取服务方法列表: %s", serviceName)

	// 使用默认值
	registry := ws.registry
	app := ws.app
	timeout := ws.timeout

	color.Cyan("[DEBUG] 使用配置 - 注册中心: %s, 应用名: %s, 超时: %d", registry, app, timeout)

	// 创建Dubbo客户端配置
	config := &DubboConfig{
		Registry:    registry,
		Application: app,
		Timeout:     time.Duration(timeout) * time.Millisecond,
	}

	client, err := NewRealDubboClient(config)
	if err != nil {
		color.Red("[ERROR] 创建Dubbo客户端失败: %v", err)
		response := ListMethodsResponse{
			Success: false,
			Error:   fmt.Sprintf("创建Dubbo客户端失败: %v", err),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	color.Green("[DEBUG] Dubbo客户端创建成功")

	// 检查连接状态
	if !client.IsConnected() {
		color.Red("[ERROR] Dubbo客户端连接失败")
		response := ListMethodsResponse{
			Success: false,
			Error:   "无法连接到注册中心",
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	color.Green("[DEBUG] Dubbo客户端连接正常")

	// 获取方法列表
	methods, err := client.ListMethods(serviceName)
	if err != nil {
		color.Red("[ERROR] 获取方法列表失败: %v", err)
		response := ListMethodsResponse{
			Success: false,
			Error:   fmt.Sprintf("获取方法列表失败: %v", err),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	color.Green("[DEBUG] 成功获取方法列表，共 %d 个方法", len(methods))

	response := ListMethodsResponse{
		Success: true,
		Methods: methods,
	}

	json.NewEncoder(w).Encode(response)
}

// handleTestConnection 处理连接测试请求
func (ws *WebServer) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取查询参数
	registry := r.URL.Query().Get("registry")
	app := r.URL.Query().Get("app")
	timeout := r.URL.Query().Get("timeout")

	color.Yellow("[DEBUG] 测试连接请求 - 注册中心: %s, 应用: %s, 超时: %s", registry, app, timeout)

	if registry == "" {
		color.Red("[ERROR] 注册中心地址不能为空")
		ws.writeError(w, "注册中心地址不能为空")
		return
	}

	// 使用默认值
	if app == "" {
		app = ws.app
	}

	// 创建Dubbo配置
	config := &DubboConfig{
		Registry:    registry,
		Application: app,
		Timeout:     time.Duration(ws.timeout) * time.Millisecond,
	}

	// 创建Dubbo客户端进行连接测试
	client, err := NewRealDubboClient(config)
	if err != nil {
		color.Red("[ERROR] 创建Dubbo客户端失败: %v", err)
		ws.writeError(w, fmt.Sprintf("连接失败: %v", err))
		return
	}
	defer client.Close()

	// 尝试获取服务列表来验证连接
	services, err := client.ListServices()
	if err != nil {
		color.Red("[ERROR] 获取服务列表失败: %v", err)
		ws.writeError(w, fmt.Sprintf("连接测试失败: %v", err))
		return
	}

	color.Green("[DEBUG] 连接测试成功，发现 %d 个服务", len(services))

	response := ListServicesResponse{
		Success:  true,
		Services: services,
	}

	json.NewEncoder(w).Encode(response)
}

// writeError 写入错误响应
func (ws *WebServer) writeError(w http.ResponseWriter, message string) {
	response := InvokeResponse{
		Success: false,
		Error:   message,
	}
	json.NewEncoder(w).Encode(response)
}

// indexHTML 首页HTML模板
const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dubbo Invoke Web UI</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: white;
            min-height: 100vh; padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 0 10px rgba(0,0,0,0.05);
            overflow: hidden;
            width: calc(100% - 40px);
        }
        .header {
            background: white;
            color: #333;
            padding: 25px;
            text-align: center;
            border-bottom: 1px solid #eee;
        }
        .header h1 { font-size: 2.5em; margin-bottom: 10px; font-weight: 300; }
        .header p { font-size: 1.1em; opacity: 0.9; }
        /* 布局样式 - 211阵型 */
        .main-content { 
            display: flex; 
            flex-direction: column;
            gap: 20px; 
            padding: 20px;
            min-height: calc(100vh - 200px);
        }
        /* 第一行：2个面板 */
        .first-row {
            display: flex;
            gap: 20px;
            min-height: 400px;
        }
        .service-call-panel { 
            flex: 2; /* 服务调用面板占据2份空间 */
            min-height: 400px;
        }
        .available-services-panel { 
            flex: 1; /* 可用服务面板占据1份空间 */
            min-height: 400px;
        }
        /* 第二行：1个面板 */
        .history-panel { 
            width: 100%;
            min-height: 300px;
        }
        /* 第三行：1个面板 */
        .result-panel { 
            width: 100%;
            min-height: 200px;
        }
        .panel h2 { 
            color: #333; 
            margin-bottom: 15px; 
            font-size: 1.1em; 
            font-weight: 400; 
            text-align: left;
            border-bottom: none;
            padding-left: 5px;
            display: flex;
            align-items: center;
        }
        .panel h2::before {
            content: '\1F4C2'; /* 文件夹图标 Unicode */
            margin-right: 5px;
            font-size: 1.1em;
        }
        /* 表单样式调整 */
        .form-group {
            margin-bottom: 15px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            color: #555;
            font-size: 13px;
            font-weight: normal;
        }
        input, select, textarea {
            width: 100%;
            padding: 8px 10px;
            border: 1px solid #e0e0e0;
            border-radius: 4px;
            font-size: 13px;
            background-color: #fff;
        }
        input:focus, select:focus, textarea:focus {
            outline: none;
            border-color: #4a90e2;
        }
        textarea {
            resize: vertical;
            min-height: 80px;
            font-family: monospace;
        }
        .btn {
            background: #4a90e2;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 13px;
            font-weight: 400;
            transition: background 0.2s ease;
            margin-right: 10px;
            margin-bottom: 10px;
        }
        .btn:hover {
            background: #3a7dca;
        }
        .btn-secondary {
            background: #6c6fe2;
        }
        .btn-secondary:hover {
            background: #5a5dca;
        }
        .btn-success {
            background: #4caf50;
        }
        .btn-success:hover {
            background: #43a047;
        }
        .panel { 
            background: #fff; 
            border-radius: 4px; 
            padding: 15px; 
            border: 2px solid #ff5252; /* 红色边框 */
            box-shadow: none;
            display: flex;
            flex-direction: column;
        }
        .panel:hover {
            box-shadow: none;
        }
        .result {
            background: white;
            border: 1px solid #e0e0e0;
            border-radius: 4px;
            padding: 16px;
            font-family: monospace;
            font-size: 13px;
            white-space: pre-wrap;
            min-height: 150px;
            max-height: 400px;
            overflow-y: auto;
            word-wrap: break-word;
            word-break: break-all;
            overflow-wrap: break-word;
            max-width: 100%;
            overflow-x: auto;
        }
        .success {
            border-color: #4caf50;
            background-color: #f1f8e9;
        }
        .error {
            border-color: #ff5252;
            background-color: #ffebee;
            color: #d32f2f;
        }
        .loading { 
            display: none; 
            text-align: center; 
            padding: 25px; 
            color: #5c6bc0; 
            font-weight: 500;
            background-color: rgba(92, 107, 192, 0.05);
            border-radius: 8px;
        }
        .spinner {
            border: 3px solid rgba(92, 107, 192, 0.1); border-top: 3px solid #5c6bc0;
            border-radius: 50%; width: 30px; height: 30px;
            animation: spin 1s linear infinite; margin: 0 auto 10px;
        }
        @keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }
        .service-list {
            flex: 1;
            min-height: 150px;
            overflow-y: auto;
            border: 1px solid #e0e0e0;
            border-radius: 3px;
            background: white;
            word-wrap: break-word;
            overflow-wrap: break-word;
        }
        .service-item {
            padding: 12px 16px; border-bottom: 1px solid #e9ecef;
            cursor: pointer; transition: all 0.2s ease;
            word-wrap: break-word; /* 确保长服务名能够换行 */
            overflow-wrap: break-word;
            white-space: normal;
            position: relative;
        }
        .service-item::after {
            content: '';
            position: absolute;
            left: 0;
            top: 0;
            height: 100%;
            width: 0;
            background-color: rgba(92, 107, 192, 0.1);
            transition: width 0.2s ease;
        }
        .service-item:hover { background-color: #f5f7ff; }
        .service-item:hover::after { width: 4px; }
        .service-item:last-child { border-bottom: none; }
        .config-info {
            background: #e8eaf6; border: 1px solid #c5cae9; border-radius: 8px;
            padding: 16px; margin-bottom: 20px; font-size: 13px;
        }
        .config-info strong { color: #3949ab; }
        /* 表单布局 */
        .form-row {
            display: flex;
            gap: 15px;
            margin-bottom: 20px;
        }
        .form-col {
            flex: 1;
        }
        .form-col .form-group:last-child {
            margin-bottom: 0;
        }
        /* 按钮组样式 */
        .btn-group {
            display: flex;
            flex-wrap: wrap;
            gap: 10px;
            margin-top: auto;
            padding-top: 10px;
        }
        .btn-group .btn {
            margin: 0;
        }
        @media (max-width: 768px) {
            .main-content { 
                flex-direction: column;
                gap: 16px; 
                padding: 16px; 
            }
            .first-row {
                flex-direction: column;
                gap: 16px;
            }
            .service-call-panel,
            .available-services-panel,
            .history-panel,
            .result-panel {
                width: 100%;
                flex: none;
                margin-top: 0;
                min-height: auto;
            }
            .header h1 { font-size: 2em; }
            .container { width: calc(100% - 20px); margin: 10px auto; }
            .header { padding: 20px; }
        }
        @media (max-width: 480px) {
            .container { width: calc(100% - 10px); margin: 5px auto; }
            .main-content { padding: 15px; gap: 15px; }
            .panel { padding: 15px; }
            .header { padding: 15px; }
            .header h1 { font-size: 1.8em; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 Dubbo Invoke Web UI</h1>
            <p>图形化界面进行Dubbo服务调用</p>
        </div>
        <div class="main-content">
            <!-- 第一行：2个面板 -->
            <div class="first-row">
                <div class="panel service-call-panel">
                    <h2>服务调用</h2>

                    <div class="form-group">
                        <label for="callFormat">调用格式:</label>
                        <select id="callFormat" onchange="toggleCallFormat()">
                            <option value="traditional">传统格式 (服务名 + 方法名)</option>
                            <option value="expression">表达式格式 (service.method(params))</option>
                        </select>
                    </div>
                    <div id="traditionalFormat">
                        <div class="form-group">
                            <label for="registry">注册中心:</label>
                            <div style="display: flex; gap: 10px; align-items: center;">
                                <input type="text" id="registry" value="{{.Registry}}" style="flex: 1;">
                                <button class="btn btn-secondary" onclick="testConnection()" style="margin: 0; white-space: nowrap;">🔗 测试连接</button>
                            </div>
                        </div>
                        <div class="form-row">
                            <div class="form-col">
                                <div class="form-group">
                                    <label for="serviceName">服务名:</label>
                                    <input type="text" id="serviceName" placeholder="com.example.UserService" value="com.example.UserService">
                                </div>
                            </div>
                            <div class="form-col">
                                <div class="form-group">
                                    <label for="methodName">方法名:</label>
                                    <input type="text" id="methodName" placeholder="getUserById" value="getUserById">
                                </div>
                            </div>
                        </div>
                        <div class="form-group">
                            <label for="parameters">参数 (JSON数组格式):</label>
                            <textarea id="parameters" placeholder='[123, "张三", true]'>[123]</textarea>
                        </div>
                    </div>
                    <div id="expressionFormat" style="display: none;">
                        <div class="form-group">
                            <label for="registry">注册中心:</label>
                            <div style="display: flex; gap: 10px; align-items: center;">
                                <input type="text" id="registryExpr" value="{{.Registry}}" style="flex: 1;">
                                <button class="btn btn-secondary" onclick="testConnection()" style="margin: 0; white-space: nowrap;">🔗 测试连接</button>
                            </div>
                        </div>
                        <div class="form-group">
                            <label for="expression">调用表达式: <span style="font-size: 0.8em; color: #5c6bc0;">(service.method(params))</span></label>
                            <textarea id="expression" placeholder='com.example.UserService.getUserById(123)'>com.example.UserService.getUserById(123)</textarea>
                        </div>
                    </div>
                    <div id="traditionalTypes" class="form-group">
                        <label for="types">参数类型 (可选，逗号分隔):</label>
                        <input type="text" id="types" placeholder="java.lang.Long,java.lang.String">
                    </div>
                    <div class="btn-group">
                        <button class="btn" onclick="invokeService()">🚀 调用服务</button>
                        <button class="btn btn-secondary" onclick="generateExample()">📝 生成示例</button>
                        <button class="btn btn-success" onclick="loadServices()">📋 加载服务列表</button>
                    </div>
                </div>
                <div class="panel available-services-panel">
                    <h2>可用服务</h2>
                    <div id="serviceList" class="service-list">
                        <div style="padding: 20px; text-align: center; color: #6c757d;">
                            <p>请先连接注册中心</p>
                        </div>
                    </div>
                </div>
            </div>
            
            <!-- 第二行：1个面板 -->
            <div class="panel history-panel">
                <h2>最近调用历史</h2>
                <div id="historyList" class="service-list">
                    <div style="padding: 20px; text-align: center; color: #6c757d;">
                        <p>暂无调用历史</p>
                    </div>
                </div>
                <div class="btn-group">
                    <button class="btn btn-secondary" onclick="downloadHistory()">下载日志</button>
                </div>
            </div>
            
            <!-- 第三行：1个面板 -->
            <div class="panel result-panel">
                <h2>调用结果</h2>
                <div class="loading" id="loading">
                    <div class="spinner"></div>
                    正在调用服务...
                </div>
                <div id="result" class="result">等待调用结果...</div>
            </div>
        </div>
    </div>
    <script>
        function toggleCallFormat() {
            const format = document.getElementById('callFormat').value;
            const traditional = document.getElementById('traditionalFormat');
            const expression = document.getElementById('expressionFormat');
            const traditionalTypes = document.getElementById('traditionalTypes');
            if (format === 'expression') {
                traditional.style.display = 'none';
                expression.style.display = 'block';
                traditionalTypes.style.display = 'none';
                // 同步注册中心值
                const registryValue = document.getElementById('registry').value;
                document.getElementById('registryExpr').value = registryValue;
            } else {
                traditional.style.display = 'block';
                expression.style.display = 'none';
                traditionalTypes.style.display = 'block';
                // 同步注册中心值
                const registryExprValue = document.getElementById('registryExpr').value;
                document.getElementById('registry').value = registryExprValue;
            }
        }
        function parseExpression(expr) {
            const parenIndex = expr.indexOf('(');
            if (parenIndex === -1) return null;
            const methodPart = expr.substring(0, parenIndex);
            const lastDotIndex = methodPart.lastIndexOf('.');
            if (lastDotIndex === -1) return null;
            const serviceName = methodPart.substring(0, lastDotIndex);
            const methodName = methodPart.substring(lastDotIndex + 1);
            let paramsPart = expr.substring(parenIndex + 1);
            if (paramsPart.endsWith(')')) {
                paramsPart = paramsPart.substring(0, paramsPart.length - 1);
            }
            let parameters = [];
            if (paramsPart.trim()) {
                try {
                    if (paramsPart.trim().startsWith('[')) {
                        parameters = JSON.parse(paramsPart);
                    } else {
                        parameters = [paramsPart.trim()];
                        try {
                            const parsed = JSON.parse(paramsPart.trim());
                            parameters = [parsed];
                        } catch (e) {}
                    }
                } catch (e) {
                    parameters = [paramsPart.trim()];
                }
            }
            return { serviceName, methodName, parameters };
        }
        function invokeService() {
            const format = document.getElementById('callFormat').value;
            let serviceName, methodName, parameters;
            if (format === 'expression') {
                const expr = document.getElementById('expression').value.trim();
                if (!expr) { alert('请输入调用表达式'); return; }
                const parsed = parseExpression(expr);
                if (!parsed) { alert('无效的表达式格式'); return; }
                serviceName = parsed.serviceName;
                methodName = parsed.methodName;
                parameters = parsed.parameters;
            } else {
                serviceName = document.getElementById('serviceName').value.trim();
                methodName = document.getElementById('methodName').value.trim();
                const paramsText = document.getElementById('parameters').value.trim();
                if (!serviceName || !methodName) { alert('请输入服务名和方法名'); return; }
                try {
                    parameters = paramsText ? JSON.parse(paramsText) : [];
                } catch (e) { alert('参数格式错误，请使用JSON数组格式'); return; }
            }
            const types = format === 'traditional' ? document.getElementById('types').value.trim() : '';
            const registry = format === 'expression' ? 
                document.getElementById('registryExpr').value.trim() : 
                document.getElementById('registry').value.trim();
            const request = {
                serviceName: serviceName, methodName: methodName,
                parameters: parameters.map(p => typeof p === 'string' ? p : JSON.stringify(p)),
                types: types ? types.split(',').map(t => t.trim()) : [],
                registry: registry, app: '{{.App}}', timeout: 10000
            };
            showLoading(true);
            fetch('/api/invoke', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(request)
            })
            .then(response => response.json())
            .then(data => { showLoading(false); displayResult(data); })
            .catch(error => {
                showLoading(false);
                displayResult({ success: false, error: '网络错误: ' + error.message });
            });
        }
        function generateExample() {
            const types = document.getElementById('types').value.trim();
            if (!types) { alert('请先输入参数类型'); return; }
            fetch('/api/example?types=' + encodeURIComponent(types))
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    const currentFormat = document.getElementById('callFormat').value;
                    if (currentFormat === 'expression') {
                        const serviceName = 'com.example.Service';
                        const methodName = 'exampleMethod';
                        const params = data.examples.join(', ');
                        document.getElementById('expression').value = serviceName + '.' + methodName + '(' + params + ')';
                    } else {
                        document.getElementById('parameters').value = JSON.stringify(data.examples, null, 2);
                    }
                } else { alert('生成示例失败: ' + data.error); }
            })
            .catch(error => { alert('生成示例失败: ' + error.message); });
        }
        function testConnection() {
            const currentFormat = document.getElementById('callFormat').value;
            const registry = currentFormat === 'expression' ? 
                document.getElementById('registryExpr').value.trim() : 
                document.getElementById('registry').value.trim();
            
            if (!registry) {
                alert('请先输入注册中心地址');
                return;
            }
            
            // 显示测试中状态
            const serviceList = document.getElementById('serviceList');
            serviceList.innerHTML = '<div style="padding: 20px; text-align: center; color: #6c757d;">🔗 正在测试连接...</div>';
            
            fetch('/api/test-connection?registry=' + encodeURIComponent(registry) + '&app={{.App}}&timeout=10000')
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    alert('✅ 连接成功！');
                    // 连接成功后自动加载服务列表
                    loadServices();
                } else {
                    alert('❌ 连接失败: ' + data.error);
                    serviceList.innerHTML = '<div style="padding: 20px; text-align: center; color: #dc3545;">连接失败: ' + data.error + '</div>';
                }
            })
            .catch(error => {
                alert('❌ 连接测试失败: ' + error.message);
                serviceList.innerHTML = '<div style="padding: 20px; text-align: center; color: #dc3545;">网络错误: ' + error.message + '</div>';
            });
        }
        function loadServices() {
            const currentFormat = document.getElementById('callFormat').value;
            const registry = currentFormat === 'expression' ? 
                document.getElementById('registryExpr').value.trim() : 
                document.getElementById('registry').value.trim();
            
            if (!registry) {
                document.getElementById('serviceList').innerHTML = 
                    '<div style="padding: 20px; text-align: center; color: #6c757d;">请先配置注册中心</div>';
                return;
            }
            
            fetch('/api/list')
            .then(response => response.json())
            .then(data => {
                if (data.success) { displayServices(data.services); }
                else { 
                    document.getElementById('serviceList').innerHTML = 
                        '<div style="padding: 20px; text-align: center; color: #dc3545;">连接注册中心失败: ' + data.error + '</div>';
                }
            })
            .catch(error => { 
                document.getElementById('serviceList').innerHTML = 
                    '<div style="padding: 20px; text-align: center; color: #dc3545;">网络错误: ' + error.message + '</div>';
            });
        }
        function displayServices(services) {
            const serviceList = document.getElementById('serviceList');
            serviceList.innerHTML = '';
            
            if (!services || services.length === 0) {
                serviceList.innerHTML = '<div style="padding: 20px; text-align: center; color: #6c757d;"><i>暂无可用服务</i></div>';
                return;
            }
            
            services.forEach(service => {
                const item = document.createElement('div');
                item.className = 'service-item';
                
                // 尝试提取包名和服务名
                const parts = service.split('.');
                const serviceName = parts.pop();
                const packageName = parts.join('.');
                
                if (packageName) {
                    item.innerHTML = 
                        '<div style="font-weight: 500; color: #3949ab;">' + serviceName + '</div>' +
                        '<div style="font-size: 0.8em; margin-top: 3px; color: #5f6368;">' + packageName + '</div>';
                } else {
                    item.textContent = service;
                }
                
                item.onclick = () => {
                    document.getElementById('serviceName').value = service;
                    loadMethods(service);
                };
                serviceList.appendChild(item);
            });
        }
        function loadMethods(serviceName) {
            const currentFormat = document.getElementById('callFormat').value;
            const registry = currentFormat === 'expression' ? 
                document.getElementById('registryExpr').value.trim() : 
                document.getElementById('registry').value.trim();
            
            if (!registry || !serviceName) {
                return;
            }
            
            fetch('/api/methods?serviceName=' + encodeURIComponent(serviceName) + '&registry=' + encodeURIComponent(registry) + '&app={{.App}}&timeout=10000')
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    setupMethodDropdown(data.methods);
                } else {
                    console.log('获取方法列表失败: ' + data.error);
                }
            })
            .catch(error => {
                console.log('获取方法列表失败: ' + error.message);
            });
        }
        function setupMethodDropdown(methods) {
            const methodInput = document.getElementById('methodName');
            const existingDatalist = document.getElementById('methodDatalist');
            if (existingDatalist) {
                existingDatalist.remove();
            }
            
            if (methods && methods.length > 0) {
                const datalist = document.createElement('datalist');
                datalist.id = 'methodDatalist';
                methods.forEach(method => {
                    const option = document.createElement('option');
                    option.value = method;
                    datalist.appendChild(option);
                });
                methodInput.setAttribute('list', 'methodDatalist');
                methodInput.parentNode.appendChild(datalist);
                
                // 如果只有一个方法，自动填充
                if (methods.length === 1) {
                    methodInput.value = methods[0];
                }
            } else {
                methodInput.removeAttribute('list');
            }
        }
        function showLoading(show) {
            const loading = document.getElementById('loading');
            const result = document.getElementById('result');
            if (show) {
                loading.style.display = 'block';
                result.style.display = 'none';
            } else {
                loading.style.display = 'none';
                result.style.display = 'block';
            }
        }
        function displayResult(data) {
            const result = document.getElementById('result');
            result.className = 'result ' + (data.success ? 'success' : 'error');
            result.textContent = JSON.stringify(data, null, 2);
            // 调用后自动刷新历史（无论成功失败）
            setTimeout(loadHistory, 500);
        }
        function downloadHistory() {
            fetch('/api/history')
            .then(response => response.json())
            .then(data => {
                if (data.success && data.history) {
                    const blob = new Blob([JSON.stringify(data.history, null, 2)], 
                        { type: 'application/json' });
                    const url = URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = 'dubbo-invoke-history-' + new Date().toISOString().slice(0,19).replace(/:/g, '-') + '.json';
                    document.body.appendChild(a);
                    a.click();
                    document.body.removeChild(a);
                    URL.revokeObjectURL(url);
                } else {
                    alert('下载失败: ' + (data.error || '无历史数据'));
                }
            })
            .catch(error => { alert('下载失败: ' + error.message); });
        }
        function loadHistory() {
            fetch('/api/history')
            .then(response => response.json())
            .then(data => {
                if (data.success) { displayHistory(data.history); }
                else { alert('加载历史记录失败: ' + data.error); }
            })
            .catch(error => { alert('加载历史记录失败: ' + error.message); });
        }
        function displayHistory(history) {
            const historyList = document.getElementById('historyList');
            historyList.innerHTML = '';
            if (!history || history.length === 0) {
                historyList.innerHTML = '<div style="padding: 20px; text-align: center; color: #6c757d;"><i>暂无调用历史</i></div>';
                return;
            }
            // 按时间倒序显示最近的记录
            history.reverse().forEach(item => {
                const historyItem = document.createElement('div');
                historyItem.className = 'service-item';
                const timestamp = new Date(item.timestamp).toLocaleString();
                const status = item.success ? '✅' : '❌';
                const statusClass = item.success ? 'success-text' : 'error-text';
                historyItem.innerHTML = 
                    '<div style="font-weight: 500; color: #3949ab;">' + item.serviceName + '.' + item.methodName + '</div>' +
                    '<div style="font-size: 0.8em; margin-top: 3px; color: #5f6368;">' +
                        '<span class="' + statusClass + '">' + status + '</span> ' + timestamp +
                    '</div>' +
                    '<div style="font-size: 0.75em; margin-top: 2px; color: #9aa0a6;">' +
                        item.parameters.length + ' 参数' +
                    '</div>';
                historyItem.onclick = () => fillFromHistory(item);
                historyList.appendChild(historyItem);
            });

            // 添加样式
            const style = document.createElement('style');
            style.textContent = 
                '.success-text { color: #43a047; }' +
                '.error-text { color: #e53935; }';
            document.head.appendChild(style);
        }
        function fillFromHistory(item) {
            // 填充表单字段
            document.getElementById('serviceName').value = item.serviceName;
            document.getElementById('methodName').value = item.methodName;
            document.getElementById('parameters').value = JSON.stringify(item.parameters, null, 2);
            document.getElementById('types').value = item.types.join(', ');
            document.getElementById('registry').value = item.registry;
            // 切换到传统格式
            document.getElementById('callFormat').value = 'traditional';
            toggleCallFormat();
        }
        window.onload = function() { loadHistory(); };
    </script>
</body>
</html>`