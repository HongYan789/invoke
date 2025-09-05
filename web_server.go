package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// CallHistory 调用历史记录
type CallHistory struct {
	ID          string        `json:"id"`
	ServiceName string        `json:"serviceName"`
	MethodName  string        `json:"methodName"`
	Parameters  []interface{} `json:"parameters"`
	Types       []string      `json:"types"`
	Registry    string        `json:"registry"`
	App         string        `json:"app"`
	Success     bool          `json:"success"`
	Timestamp   time.Time     `json:"timestamp"`
	Result      string        `json:"result"`
	Duration    int64         `json:"duration"` // 调用耗时，单位毫秒
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
	ServiceName string          `json:"serviceName"`
	MethodName  string          `json:"methodName"`
	Parameters  json.RawMessage `json:"parameters"` // 使用json.RawMessage支持多种类型
	Types       []string        `json:"types"`
	Registry    string          `json:"registry"`
	App         string          `json:"app"`
	Timeout     int             `json:"timeout"`
	Group       string          `json:"group"`
	Version     string          `json:"version"`
}

// InvokeResponse Web调用响应
type InvokeResponse struct {
	Success  bool        `json:"success"`
	Data     interface{} `json:"data"`
	Error    string      `json:"error"`
	Message  string      `json:"message"`
	Duration int64       `json:"duration"` // 后端处理耗时，单位毫秒
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
	http.HandleFunc("/api/clear-history", ws.handleClearHistory)

	// 添加静态文件服务
	http.Handle("/test_download.html", http.HandlerFunc(ws.handleStaticFile))

	// enhanceWebServerWithCompleteData(ws)
	http.HandleFunc("/api/test-precision", ws.handleTestPrecision)

	addr := fmt.Sprintf(":%d", ws.port)
	color.Green("🚀 Web UI服务器启动成功!")
	color.Cyan("📱 访问地址: http://localhost:%d", ws.port)
	color.Yellow("⚙️  默认注册中心: %s", ws.registry)
	color.Yellow("📦 默认应用名: %s", ws.app)
	color.Green("✨ 数据完整性增强: 已启用")
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
	color.Green("[WEB] 收到服务调用请求: %s %s", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理OPTIONS预检请求
	if r.Method == "OPTIONS" {
		color.Yellow("[WEB] 处理OPTIONS预检请求")
		w.WriteHeader(http.StatusOK)
		return
	}

	var req InvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		color.Red("[WEB] 请求解析失败: %v", err)
		ws.writeError(w, fmt.Sprintf("请求解析失败: %v", err))
		return
	}

	color.Cyan("[WEB] 解析请求成功 - 服务: %s, 方法: %s, 参数: %s", req.ServiceName, req.MethodName, string(req.Parameters))

	// 解析参数，保持Long类型精度
	var params []interface{}
	if len(req.Parameters) > 0 {
		// 尝试解析为参数数组
		var paramArray []interface{}
		decoder := json.NewDecoder(strings.NewReader(string(req.Parameters)))
		decoder.UseNumber()
		err := decoder.Decode(&paramArray)
		if err == nil {
			// 成功解析为数组
			params = convertJSONNumbers(paramArray)
			color.Green("[WEB] 解析为多参数格式，参数数量: %d", len(params))
		} else {
			// 如果不是数组格式，尝试解析为单个参数
			var singleParam interface{}
			decoder = json.NewDecoder(strings.NewReader(string(req.Parameters)))
			decoder.UseNumber()
			err = decoder.Decode(&singleParam)
			if err == nil {
				params = []interface{}{convertJSONNumber(singleParam)}
				color.Green("[WEB] 解析为单参数格式，参数数量: 1")
			} else {
				// 如果都失败了，作为字符串处理
				params = []interface{}{string(req.Parameters)}
				color.Yellow("[WEB] 参数解析失败，作为字符串处理: %s", string(req.Parameters))
			}
		}
	}

	color.Blue("[WEB] 开始执行Dubbo调用: %s.%s", req.ServiceName, req.MethodName)
	// 记录开始时间
	startTime := time.Now()
	// 执行调用
	result, err := ws.executeInvoke(req)
	// 计算耗时
	duration := time.Since(startTime).Milliseconds()
	color.Cyan("[WEB] 调用耗时: %d ms", duration)

	// 保存调用历史
	history := CallHistory{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		ServiceName: req.ServiceName,
		MethodName:  req.MethodName,
		Parameters:  safeCopyParameters(params), // 使用解析后的参数数组，保持Long类型精度
		Types:       req.Types,
		Registry:    req.Registry,
		App:         req.App,
		Success:     err == nil,
		Timestamp:   time.Now(),
		Duration:    duration,
	}

	if err != nil {
		color.Red("[WEB] 调用失败: %v", err)
		history.Result = err.Error()
		ws.history = append(ws.history, history)
		color.Cyan("[WEB] 已保存失败调用历史, 历史记录总数: %d", len(ws.history))
		// 直接返回原始错误信息，不进行JSON包装
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	// 保存成功结果，对结果中的大整数进行安全处理
	safeResult := safeCopyValue(result)
	color.Green("[WEB] 调用成功，结果已进行安全处理")

	// 使用自定义编码器来处理大整数，确保它们在JSON序列化过程中不会丢失精度
	// 创建一个自定义的JSON编码器，使用SetEscapeHTML(false)来避免HTML转义
	var resultBuffer bytes.Buffer
	encoder := json.NewEncoder(&resultBuffer)
	encoder.SetEscapeHTML(false)

	if jsonErr := encoder.Encode(safeResult); jsonErr == nil {
		// 去除末尾的换行符
		resultStr := strings.TrimSuffix(resultBuffer.String(), "\n")
		history.Result = resultStr
		color.Cyan("[WEB] 结果序列化成功, 长度: %d 字符", len(history.Result))
	} else {
		history.Result = fmt.Sprintf("%v", safeResult)
		color.Yellow("[WEB] 结果序列化失败，使用字符串格式: %v", jsonErr)
	}
	ws.history = append(ws.history, history)
	color.Cyan("[WEB] 已保存成功调用历史, 历史记录总数: %d", len(ws.history))

	// 成功时返回标准的InvokeResponse格式，确保结果中的大整数已安全处理
	response := InvokeResponse{
		Success:  true,
		Data:     safeResult, // 使用安全处理后的结果
		Error:    "",
		Message:  "调用成功",
		Duration: duration,
	}

	w.Header().Set("Content-Type", "application/json")
	// 使用自定义编码器来确保大整数正确序列化
	var responseBuffer bytes.Buffer
	responseEncoder := json.NewEncoder(&responseBuffer)
	responseEncoder.SetEscapeHTML(false)
	responseEncoder.Encode(response)
	w.Write(responseBuffer.Bytes())
}

// handleList 处理服务列表
func (ws *WebServer) handleList(w http.ResponseWriter, r *http.Request) {
	color.Green("[WEB] 收到服务列表请求: %s %s", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 处理POST请求的JSON数据
	var registry, app string
	if r.Method == "POST" {
		var requestData struct {
			Registry string `json:"registry"`
			App      string `json:"app"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
			color.Red("[WEB] 解析请求数据失败: %v", err)
			response := ListServicesResponse{
				Success: false,
				Error:   fmt.Sprintf("解析请求数据失败: %v", err),
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		registry = requestData.Registry
		app = requestData.App
	} else {
		// 处理GET请求的查询参数
		registry = r.URL.Query().Get("registry")
		app = r.URL.Query().Get("app")
	}

	if registry == "" {
		registry = ws.registry
	}
	if app == "" {
		app = ws.app
	}

	// 创建dubbo客户端配置
	config := &DubboConfig{
		Registry:    registry,
		Application: app,
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

	// 如果是空字符串，返回nil
	if param == "" {
		color.Green("[WEB] 参数为空，返回nil")
		return nil, nil
	}

	// 如果不是JSON格式，尝试智能转换
	if !strings.HasPrefix(param, "{") && !strings.HasPrefix(param, "[") {
		// 尝试转换为数字
		if param == "null" {
			color.Green("[WEB] 参数为null，返回nil")
			return nil, nil
		}

		// 尝试转换为布尔值
		if param == "true" {
			color.Green("[WEB] 参数为布尔值true")
			return true, nil
		}
		if param == "false" {
			color.Green("[WEB] 参数为布尔值false")
			return false, nil
		}

		// 尝试转换为整数
		if strings.Contains(param, ".") {
			// 可能是浮点数
			if f, err := strconv.ParseFloat(param, 64); err == nil {
				color.Green("[WEB] 参数转换为浮点数: %f", f)
				return f, nil
			}
		} else {
			// 可能是整数
			if i, err := strconv.ParseInt(param, 10, 64); err == nil {
				color.Green("[WEB] 参数转换为整数: %d", i)
				return i, nil
			}
		}

		color.Green("[WEB] 参数保持为字符串")
		return param, nil
	}

	// 尝试解析为JSON，使用json.Number保持大整数精度
	color.Blue("[WEB] 尝试解析JSON格式参数")
	decoder := json.NewDecoder(strings.NewReader(param))
	decoder.UseNumber() // 使用json.Number保持大整数精度
	var result interface{}
	err := decoder.Decode(&result)
	if err != nil {
		color.Red("[WEB] JSON解析失败: %v", err)
		return nil, err
	}
	color.Green("[WEB] JSON解析成功，使用json.Number保持精度")

	// 特别处理JSON中的null值
	if result == nil {
		color.Green("[WEB] JSON解析结果为null")
		return nil, nil
	}

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

	// 解析字符串参数为interface{}类型
	color.Blue("[WEB] 开始解析调用参数")
	var params []interface{}
	if len(req.Parameters) > 0 {
		// 尝试解析为参数数组
		var paramArray []interface{}
		decoder := json.NewDecoder(strings.NewReader(string(req.Parameters)))
		decoder.UseNumber()
		err := decoder.Decode(&paramArray)
		if err != nil {
			color.Red("[WEB] 参数解析失败: %v", err)
			return nil, fmt.Errorf("参数解析失败: %v", err)
		}

		// 将json.Number转换为适当的类型
		params = convertJSONNumbers(paramArray)
		color.Green("[WEB] 解析参数完成，参数数量: %d", len(params))
	}
	color.Green("[WEB] 参数解析完成，最终参数数量: %d", len(params))

	// 构建并打印dubbo invoke命令，方便用户验证
	invokeCmd := ws.buildDubboInvokeCommand(req.ServiceName, req.MethodName, params)
	color.Yellow("[DUBBO CMD] %s", invokeCmd)

	// 尝试使用真实的Dubbo客户端
	color.Blue("[WEB] 尝试创建真实Dubbo客户端")
	realClient, err := NewRealDubboClient(cfg)
	if err != nil {
		color.Red("[WEB] 真实Dubbo客户端创建失败: %v", err)
		return nil, fmt.Errorf("无法连接到Dubbo注册中心: %v", err)
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

	// 检查result是否为JSON字符串，如果是则解析为对象
	if resultStr, ok := result.(string); ok {
		// 尝试解析JSON字符串为对象，使用UseNumber()保持大整数精度
		var parsedResult interface{}
		decoder := json.NewDecoder(strings.NewReader(resultStr))
		decoder.UseNumber()
		if err := decoder.Decode(&parsedResult); err == nil {
			color.Green("[WEB] JSON字符串解析成功，返回解析后的对象")

			// 转换json.Number为适当的类型
			result = convertJSONNumber(parsedResult)

		} else {
			color.Yellow("[WEB] JSON解析失败，返回原始字符串: %v", err)
		}
	}

	// 直接返回原始结果，不进行额外的数据包装处理
	color.Green("[WEB] 返回原始结果，数据类型: %T", result)
	return result, nil
}

// buildDubboInvokeCommand 构建dubbo invoke命令，用于调试和验证
func (ws *WebServer) buildDubboInvokeCommand(serviceName, methodName string, params []interface{}) string {
	// 创建临时客户端用于格式化参数
	tempClient := &RealDubboClient{}

	// 格式化参数
	paramStr, err := tempClient.formatParameters(params)
	if err != nil {
		// 如果格式化失败，使用简单格式
		var simpleParams []string
		for _, param := range params {
			simpleParams = append(simpleParams, fmt.Sprintf("%v", param))
		}
		paramStr = strings.Join(simpleParams, ", ")
	}

	return fmt.Sprintf("invoke %s.%s(%s)", serviceName, methodName, paramStr)
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

// handleClearHistory 处理清空历史记录
func (ws *WebServer) handleClearHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		ws.writeError(w, "只支持POST方法")
		return
	}

	// 清空历史记录
	ws.history = make([]CallHistory, 0)

	response := map[string]interface{}{
		"success": true,
		"message": "历史记录已清空",
	}

	json.NewEncoder(w).Encode(response)
}

// handleMethods 处理获取服务方法列表
func (ws *WebServer) handleMethods(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理OPTIONS预检请求
	if r.Method == "OPTIONS" {

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

	// 使用默认值
	registry := ws.registry
	app := ws.app
	timeout := ws.timeout

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

	response := ListMethodsResponse{
		Success: true,
		Methods: methods,
	}

	json.NewEncoder(w).Encode(response)
}

// writeError 写入错误响应
// safeCopyParameters 安全复制参数，将大整数转换为字符串以避免精度丢失
// convertJSONNumbers 将json.Number转换为适当的类型，保持大整数精度
func convertJSONNumbers(params []interface{}) []interface{} {
	result := make([]interface{}, len(params))
	for i, param := range params {
		result[i] = convertJSONNumber(param)
	}
	return result
}

// convertJSONNumber 递归转换json.Number类型
func convertJSONNumber(value interface{}) interface{} {
	switch v := value.(type) {
	case json.Number:
		// 检查是否为大整数（超过JavaScript安全整数范围或超过15位数字）
		numStr := string(v)

		if len(numStr) > 15 {
			// 超过15位数字，直接返回字符串避免精度丢失

			return numStr
		}

		// 尝试转换为int64
		if intVal, err := v.Int64(); err == nil {
			// 检查是否超过JavaScript安全整数范围
			if intVal > 9007199254740991 || intVal < -9007199254740991 {
				return numStr // 返回字符串避免精度丢失
			}
			return intVal
		}
		// 如果无法转换为int64，尝试转换为float64
		if floatVal, err := v.Float64(); err == nil {
			return floatVal
		}
		// 如果都失败，返回原始字符串
		return numStr
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertJSONNumber(item)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, item := range v {
			result[k] = convertJSONNumber(item)
		}
		return result
	default:
		return value
	}
}

func safeCopyParameters(params []interface{}) []interface{} {
	result := make([]interface{}, len(params))
	for i, param := range params {
		result[i] = safeCopyValue(param)
	}
	return result
}

// safeCopyValue 安全复制单个值，处理大整数精度问题
func safeCopyValue(value interface{}) interface{} {

	switch v := value.(type) {
	case json.Number:
		// 优先处理json.Number类型，保持原始精度
		numStr := string(v)
		// 尝试解析为整数
		if intVal, err := v.Int64(); err == nil {
			// 检查是否超过JavaScript安全整数范围或大于15位
			if intVal > 9007199254740991 || intVal < -9007199254740991 ||
				intVal >= 1000000000000000 || intVal <= -1000000000000000 {
				return numStr // 返回原始字符串保持精度
			}
			return intVal
		}
		// 如果不是整数，尝试解析为浮点数
		if floatVal, err := v.Float64(); err == nil {
			return floatVal
		}
		// 如果都解析失败，返回原始字符串
		return numStr
	case float64:
		// 检查是否为整数且超过JavaScript安全整数范围
		if v == float64(int64(v)) && (v > 9007199254740991 || v < -9007199254740991) {
			return strconv.FormatFloat(v, 'f', 0, 64)
		}
		// 对于大于15位的整数，也转换为字符串以防止精度丢失
		if v == float64(int64(v)) && (v >= 1000000000000000 || v <= -1000000000000000) {
			return strconv.FormatFloat(v, 'f', 0, 64)
		}
		return v
	case int64:
		// 检查是否超过JavaScript安全整数范围
		if v > 9007199254740991 || v < -9007199254740991 {
			return strconv.FormatInt(v, 10)
		}
		// 对于大于15位的整数，也转换为字符串以防止精度丢失
		if v >= 1000000000000000 || v <= -1000000000000000 {
			return strconv.FormatInt(v, 10)
		}
		return v
	case int:
		// 处理int类型
		if int64(v) > 9007199254740991 || int64(v) < -9007199254740991 {
			return strconv.Itoa(v)
		}
		if int64(v) >= 1000000000000000 || int64(v) <= -1000000000000000 {
			return strconv.Itoa(v)
		}
		return v
	case []interface{}:
		// 递归处理数组
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = safeCopyValue(item)
		}
		return result
	case map[string]interface{}:
		// 递归处理对象
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = safeCopyValue(val)
		}
		return result
	default:
		return v
	}
}

func (ws *WebServer) writeError(w http.ResponseWriter, message string) {
	response := InvokeResponse{
		Success: false,
		Error:   message,
	}
	json.NewEncoder(w).Encode(response)
}

// handleTestPrecision 测试精度处理的接口
func (ws *WebServer) handleTestPrecision(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 创建包含大整数的测试数据
	testData := map[string]interface{}{
		"largeInt1":   json.Number("1954894705928892456"),
		"largeInt2":   json.Number("9223372036854775807"),
		"normalInt":   json.Number("12345"),
		"floatValue":  json.Number("123.456"),
		"stringValue": "test string",
		"nestedData": map[string]interface{}{
			"innerLargeInt": json.Number("1954894705928892456"),
			"innerArray": []interface{}{
				json.Number("1954894705928892456"),
				json.Number("123"),
				"string in array",
			},
		},
	}

	// 使用safeCopyValue处理数据
	processedData := safeCopyValue(testData)

	response := InvokeResponse{
		Success: true,
		Data:    processedData,
		Message: "精度测试数据",
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(response)
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
            width: 100%;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
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
        
        .top-row {
            display: flex;
            gap: 20px;
            flex: 0 0 auto;
            height: 800px;
        }
        
        /* 左列：服务调用面板 */
        .left-column {
            flex: 0 0 50%;
            width: 50%;
            display: flex;
            flex-direction: column;
        }
        /* 右列：可用服务和历史记录 */
        .right-column {
            flex: 0 0 50%;
            width: 50%;
            display: flex;
            flex-direction: column;
            gap: 20px;
        }
        .service-call-panel { 
            flex: 0 0 auto;
            height: 820px;
            min-height: 500px;
            max-height: 820px;
        }
        .available-services-panel { 
            flex: 0 0 auto;
            height: 400px;
            min-height: 300px;
            max-height: 500px;
        }
        .history-panel { 
            flex: 0 0 auto;
            height: 400px;
            min-height: 300px;
            max-height: 500px;
            overflow: hidden;
            max-width: 100%;
            contain: layout;
        }
        .history-list {
            flex: 1;
            min-height: 150px;
            max-height: 300px;
            overflow-y: auto;
            border: 1px solid #e0e0e0;
            border-radius: 3px;
            background: white;
            word-wrap: break-word;
            overflow-wrap: break-word;
        }
        /* 调用结果面板独立显示在底部 */
        .result-panel { 
            min-height: 200px;
            flex-shrink: 0;
            margin-top: 20px;
            width: 100%;
            max-width: 100%;
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
            margin-right: 8px;
            font-size: 1.1em;
        }
        .service-call-panel h2::before {
            content: '🔧'; /* 工具图标 - 服务调用 */
        }
        .available-services-panel h2::before {
            content: '📋'; /* 列表图标 - 可用服务 */
        }
        .history-panel h2::before {
            content: '📜'; /* 卷轴图标 - 调用历史 */
        }
        .history-panel h2 {
            justify-content: space-between;
            flex-wrap: nowrap;
            min-width: 0;
        }
        .history-panel h2 span {
            flex-shrink: 1;
            min-width: 0;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }
        .history-actions {
            display: flex;
            gap: 8px;
            align-items: center;
            flex-shrink: 0;
            margin-left: 10px;
        }
        .icon-btn {
            background: none;
            border: none;
            cursor: pointer;
            padding: 6px;
            border-radius: 4px;
            font-size: 16px;
            transition: background-color 0.2s ease;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .icon-btn:hover {
            background-color: #f0f0f0;
        }
        .icon-btn.download:hover {
            background-color: #e3f2fd;
        }
        .icon-btn.clear:hover {
            background-color: #ffebee;
        }
        .result-panel h2::before {
            content: '📊'; /* 图表图标 - 调用结果 */
        }
        .result-panel h2 {
            justify-content: space-between;
        }
        .result-actions {
            display: flex;
            gap: 8px;
            align-items: center;
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
            border-radius: 8px; 
            padding: 20px; 
            border: 1px solid #e1e5e9;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            display: flex;
            flex-direction: column;
        }
        .panel:hover {
            box-shadow: 0 4px 20px rgba(0,0,0,0.15);
            transform: translateY(-2px);
            transition: all 0.3s ease;
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
            max-height: 300px;
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
            max-width: 100%;
            min-width: 0;
            flex-shrink: 1;
            overflow: hidden;
        }
        .service-item .service-name {
            font-weight: 500; 
            color: #3949ab;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            max-width: 100%;
        }
        .service-item .service-name:hover {
            white-space: normal;
            word-wrap: break-word;
        }

        .history-list::-webkit-scrollbar {
            width: 6px;
        }
        .history-list::-webkit-scrollbar-track {
            background: #f1f1f1;
            border-radius: 3px;
        }
        .history-list::-webkit-scrollbar-thumb {
            background: #c1c1c1;
            border-radius: 3px;
        }
        .history-list::-webkit-scrollbar-thumb:hover {
            background: #a8a8a8;
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
                margin-top: 20px;
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
            <div class="top-row">
                <!-- 左列：服务调用面板 -->
                <div class="left-column">
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
                            <button class="btn btn-success" onclick="loadServices()" style="display: none;">📋 加载服务列表</button>
                        </div>
                    </div>
                </div>
                
                <!-- 右列：可用服务和历史记录 -->
                <div class="right-column">
                    <div class="panel available-services-panel">
                        <h2>可用服务</h2>
                        <div id="serviceList" class="service-list">
                            <div style="padding: 20px; text-align: center; color: #6c757d;">
                                <p>请先连接注册中心</p>
                            </div>
                        </div>
                    </div>
                    
                    <div class="panel history-panel">
                        <h2>
                            <span>最近调用历史</span>
                            <div class="history-actions">
                                <button class="icon-btn download" onclick="downloadHistory()" title="下载日志">
                                    📥
                                </button>
                                <button class="icon-btn clear" onclick="clearHistory()" title="清空日志">
                                    🗑️
                                </button>
                            </div>
                        </h2>
                        <div id="historyList" class="service-list history-list">
                            <div style="padding: 20px; text-align: center; color: #6c757d;">
                                <p>暂无调用历史</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            
            <!-- 调用结果面板独立显示在底部 -->
            <div class="panel result-panel">
                <h2>
                    <span>调用结果</span>
                    <div class="result-actions">
                        <button class="icon-btn copy" onclick="copyResult()" title="复制结果">
                            📋
                        </button>
                    </div>
                </h2>
                <div id="loading" class="loading">
                    <div class="spinner"></div>
                    正在调用服务...
                </div>
                <div id="result" class="result" style="display: none;"></div>
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
                    // 解析参数为真正的JavaScript对象/数组，而不是字符串
                    parameters = paramsText ? JSON.parse(paramsText) : [];
                } catch (e) { alert('参数格式错误，请使用JSON数组格式: ' + e.message); return; }
            }
            const types = format === 'traditional' ? document.getElementById('types').value.trim() : '';
            const registry = format === 'expression' ? 
                document.getElementById('registryExpr').value.trim() : 
                document.getElementById('registry').value.trim();
            const request = {
                serviceName: serviceName, methodName: methodName,
                parameters: parameters,
                types: types ? types.split(',').map(t => t.trim()) : [],
                registry: registry, app: '{{.App}}', timeout: 10000
            };
            showLoading(true);
            const startTime = Date.now(); // 记录前端调用开始时间
            fetch('/api/invoke', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(request)
            })
            .then(response => {
                if (response.ok) {
                    return response.json();
                } else {
                    // 对于错误响应，直接返回文本内容
                    return response.text().then(text => ({
                        success: false,
                        error: text
                    }));
                }
            })
            .then(data => { 
                showLoading(false); 
                const totalTime = Date.now() - startTime; // 计算总耗时
                data.totalTime = totalTime; // 添加总耗时到响应数据
                displayResult(data); 
            })
            .catch(error => {
                showLoading(false);
                const totalTime = Date.now() - startTime;
                displayResult({ success: false, error: '网络错误: ' + error.message, totalTime: totalTime });
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
            
            fetch('/api/list?registry=' + encodeURIComponent(registry) + '&app={{.App}}&timeout=10000')
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
            
            // 如果是成功调用，显示data字段的内容；如果是失败，显示error信息
            if (data.success && data.data !== undefined) {
                // 格式化显示数据，提供优雅的输出格式
                if (typeof data.data === 'string') {
                    try {
                        // 如果是JSON字符串，尝试解析并格式化
                        const parsed = JSON.parse(data.data, function(key, value) {
                            // 检查是否为大整数（超过JavaScript安全整数范围）
                            if (typeof value === 'number' && (value > Number.MAX_SAFE_INTEGER || value < Number.MIN_SAFE_INTEGER)) {
                                return value.toString();
                            }
                            // 处理19位及以上的整数
                            if (typeof value === 'number' && value >= 1000000000000000) {
                                return value.toString();
                            }
                            return value;
                        });
                        result.textContent = JSON.stringify(parsed, null, 2);
                    } catch (e) {
                        // 如果不是JSON字符串，直接显示
                        result.textContent = data.data;
                    }
                } else if (typeof data.data === 'object' && data.data !== null) {
                    // 如果是对象或数组，格式化显示，并处理其中的大整数
                    const processedData = processLargeIntegers(data.data);
                    result.textContent = JSON.stringify(processedData, null, 2);
                } else {
                    // 如果是基础数据类型（数字、布尔值、null等），直接显示
                    result.textContent = String(data.data);
                }
            } else if (!data.success && data.error) {
                result.textContent = data.error;
            } else {
                // 兼容旧格式或其他情况
                result.textContent = JSON.stringify(data, null, 2);
            }
            
            // 更新结果面板标题的状态指示器
            const resultPanelTitle = document.querySelector('.result-panel h2');
            if (resultPanelTitle) {
                const statusIndicator = data.success ? 
                    '<span style="color: #4caf50; margin-left: 8px;">●</span>' : 
                    '<span style="color: #f44336; margin-left: 8px;">●</span>';
                const statusText = data.success ? '调用成功' : '调用失败';
                
                // 构建耗时信息
                let timeInfo = '';
                if (data.totalTime) {
                    timeInfo += ' (总耗时: ' + data.totalTime + 'ms';
                    if (data.duration) {
                        timeInfo += ', 后端: ' + data.duration + 'ms';
                    }
                    timeInfo += ')';
                } else if (data.duration) {
                    timeInfo += ' (后端耗时: ' + data.duration + 'ms)';
                }
                
                // 保留复制按钮，只更新标题文本
                const titleSpan = resultPanelTitle.querySelector('span');
                if (titleSpan) {
                    titleSpan.innerHTML = '调用结果 - ' + statusText + timeInfo + statusIndicator;
                } else {
                    // 如果没有找到span，创建一个并保留原有结构
                    const actionsDiv = resultPanelTitle.querySelector('.result-actions');
                    resultPanelTitle.innerHTML = '<span>调用结果 - ' + statusText + timeInfo + statusIndicator + '</span>';
                    if (actionsDiv) {
                        resultPanelTitle.appendChild(actionsDiv);
                    }
                }
            }
            
            // 调用后自动刷新历史（无论成功失败）
            setTimeout(loadHistory, 500);
        }
        
        // 处理对象中的大整数，确保它们以字符串形式显示
        function processLargeIntegers(obj) {
            if (obj === null || obj === undefined) {
                return obj;
            }
            
            if (typeof obj === 'object' && !Array.isArray(obj)) {
                // 处理对象
                const result = {};
                for (const key in obj) {
                    if (obj.hasOwnProperty(key)) {
                        result[key] = processLargeIntegers(obj[key]);
                    }
                }
                return result;
            } else if (Array.isArray(obj)) {
                // 处理数组
                return obj.map(item => processLargeIntegers(item));
            } else if (typeof obj === 'number') {
                // 处理数字，检查是否为大整数
                // 检查是否超过JavaScript安全整数范围
                if (obj > Number.MAX_SAFE_INTEGER || obj < Number.MIN_SAFE_INTEGER) {
                    return obj.toString();
                }
                // 处理15位及以上的整数（即使在安全范围内也可能有精度问题）
                if ((obj >= 1000000000000000 && obj <= Number.MAX_SAFE_INTEGER) || 
                    (obj <= -1000000000000000 && obj >= Number.MIN_SAFE_INTEGER)) {
                    return obj.toString();
                }
                return obj;
            } else if (typeof obj === 'string') {
                // 尝试将字符串转换为数字，如果转换后超过安全范围，则保持为字符串
                const num = Number(obj);
                if (!isNaN(num)) {
                    // 检查是否超过JavaScript安全整数范围
                    if (num > Number.MAX_SAFE_INTEGER || num < Number.MIN_SAFE_INTEGER) {
                        return obj; // 保持为字符串
                    }
                    // 处理15位及以上的整数
                    if ((num >= 1000000000000000 && num <= Number.MAX_SAFE_INTEGER) || 
                        (num <= -1000000000000000 && num >= Number.MIN_SAFE_INTEGER)) {
                        return obj; // 保持为字符串
                    }
                    return num; // 转换为数字
                }
                return obj;
            }
            
            return obj;
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
        function clearHistory() {
            if (confirm('确定要清空所有历史记录吗？此操作不可恢复。')) {
                fetch('/api/clear-history', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    }
                })
                .then(response => response.json())
                .then(data => {
                    if (data.success) {
                        alert('历史记录已清空');
                        loadHistory(); // 重新加载历史记录
                    } else {
                        alert('清空失败: ' + (data.error || '未知错误'));
                    }
                })
                .catch(error => { alert('清空失败: ' + error.message); });
            }
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
                const fullServiceName = item.serviceName + '.' + item.methodName;
                
                // 处理参数显示，限制长度并添加滚动
                let paramDisplay = '';
                if (item.parameters) {
                    let paramText = '';
                    if (Array.isArray(item.parameters)) {
                        // 数组格式的参数，转换为字符串显示
                        paramText = JSON.stringify(item.parameters);
                    } else if (typeof item.parameters === 'string' && item.parameters.trim() !== '') {
                        // 兼容旧的字符串格式
                        paramText = item.parameters;
                    }
                    
                    if (paramText && paramText.length > 15) {
                        paramDisplay = '<div style="font-size: 0.75em; margin-top: 2px; color: #9aa0a6; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; word-break: break-all;" title="' + paramText.replace(/"/g, '&quot;') + '">' +
                            paramText.substring(0, 15) + '...' +
                        '</div>';
                    } else if (paramText) {
                        paramDisplay = '<div style="font-size: 0.75em; margin-top: 2px; color: #9aa0a6; word-break: break-all; max-width: 100%;">' + paramText + '</div>';
                    } else {
                        paramDisplay = '<div style="font-size: 0.75em; margin-top: 2px; color: #9aa0a6;">无参数</div>';
                    }
                } else {
                    paramDisplay = '<div style="font-size: 0.75em; margin-top: 2px; color: #9aa0a6;">无参数</div>';
                }
                
                historyItem.innerHTML = 
                    '<div class="service-name" style="max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; word-break: break-all;" title="' + fullServiceName + '">' + fullServiceName + '</div>' +
                    '<div style="font-size: 0.8em; margin-top: 3px; color: #5f6368; max-width: 100%; word-break: break-all;">' +
                        '<span class="' + statusClass + '">' + status + '</span> ' + timestamp +
                    '</div>' +
                    paramDisplay;
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
            document.getElementById('serviceName').value = item.serviceName || '';
            document.getElementById('methodName').value = item.methodName || '';
            
            // 处理参数：parameters现在是数组格式
            if (item.parameters) {
                if (Array.isArray(item.parameters)) {
                    // 直接处理数组格式的参数，处理其中的大整数
                    const processedParams = processLargeIntegers(item.parameters);
                    document.getElementById('parameters').value = JSON.stringify(processedParams);
                } else {
                    // 兼容旧的字符串格式
                    try {
                        const parsed = JSON.parse(item.parameters);
                        if (Array.isArray(parsed)) {
                            // 处理其中的大整数
                            const processedParams = processLargeIntegers(parsed);
                            document.getElementById('parameters').value = JSON.stringify(processedParams);
                        } else {
                            document.getElementById('parameters').value = item.parameters;
                        }
                    } catch (e) {
                        document.getElementById('parameters').value = item.parameters;
                    }
                }
            } else {
                document.getElementById('parameters').value = '';
            }
            
            // 处理参数类型
            if (item.types) {
                if (Array.isArray(item.types)) {
                    document.getElementById('types').value = item.types.join(', ');
                } else {
                    try {
                        const parsed = JSON.parse(item.types);
                        if (Array.isArray(parsed)) {
                            document.getElementById('types').value = parsed.join(', ');
                        } else {
                            document.getElementById('types').value = item.types;
                        }
                    } catch (e) {
                        document.getElementById('types').value = item.types;
                    }
                }
            } else {
                document.getElementById('types').value = '';
            }
            
            // 填充注册中心地址
            document.getElementById('registry').value = item.registry || '';
            
            // 填充调用结果
            if (item.result) {
                const resultElement = document.getElementById('result');
                if (resultElement) {
                    // 智能格式化结果数据，处理大整数
                    try {
                        // 尝试解析为JSON并美化显示
                        let resultData = item.result;
                        
                        // 处理双重转义的JSON字符串
                        if (typeof resultData === 'string' && resultData.startsWith('"') && resultData.endsWith('"')) {
                            try {
                                // 先解析一次去掉外层引号和转义
                                resultData = JSON.parse(resultData);
                            } catch (e) {
                                // 如果解析失败，保持原样
                            }
                        }
                        
                        // 再次尝试解析为JSON对象，使用reviver保持大整数精度
                        const parsed = JSON.parse(resultData, function(key, value) {
                            // 检查是否为大整数（超过JavaScript安全整数范围）
                            if (typeof value === 'number' && (value > Number.MAX_SAFE_INTEGER || value < Number.MIN_SAFE_INTEGER)) {
                                return value.toString();
                            }
                            // 处理15位及以上的整数
                            if (typeof value === 'number' && (value >= 1000000000000000 || value <= -1000000000000000)) {
                                return value.toString();
                            }
                            return value;
                        });
                        resultElement.textContent = JSON.stringify(parsed, null, 2);
                    } catch (e) {
                        // 如果不是JSON格式，直接显示原内容
                        resultElement.textContent = item.result;
                    }
                    resultElement.className = 'result ' + (item.success ? 'success' : 'error');
                    
                    // 更新结果面板标题
                    const resultPanelTitle = document.querySelector('.result-panel h2');
                    if (resultPanelTitle) {
                        const statusIndicator = item.success ? 
                            '<span style="color: #4caf50; margin-left: 8px;">●</span>' : 
                            '<span style="color: #f44336; margin-left: 8px;">●</span>';
                        const statusText = item.success ? '调用成功' : '调用失败';
                        
                        // 保留复制按钮，只更新标题文本
                        const titleSpan = resultPanelTitle.querySelector('span');
                        if (titleSpan) {
                            titleSpan.innerHTML = '调用结果 - ' + statusText + statusIndicator;
                        } else {
                            // 如果没有找到span，创建一个并保留原有结构
                            const actionsDiv = resultPanelTitle.querySelector('.result-actions');
                            resultPanelTitle.innerHTML = '<span>调用结果 - ' + statusText + statusIndicator + '</span>';
                            if (actionsDiv) {
                                resultPanelTitle.appendChild(actionsDiv);
                            }
                        }
                    }
                }
            }
            
            // 切换到传统格式
            document.getElementById('callFormat').value = 'traditional';
            toggleCallFormat();
            
            // 重新设置注册中心地址（因为toggleCallFormat可能会重置它）
            document.getElementById('registry').value = item.registry || '';
        }
        
        function copyResult() {
            const resultElement = document.getElementById('result');
            if (!resultElement || !resultElement.textContent.trim()) {
                alert('暂无结果数据可复制');
                return;
            }
            
            // 创建临时文本区域用于复制
            const textarea = document.createElement('textarea');
            textarea.value = resultElement.textContent;
            document.body.appendChild(textarea);
            textarea.select();
            
            try {
                document.execCommand('copy');
                alert('结果已复制到剪贴板');
            } catch (err) {
                // 如果复制失败，提供下载选项
                const blob = new Blob([resultElement.textContent], { type: 'application/json' });
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = 'dubbo-invoke-result-' + new Date().toISOString().slice(0,19).replace(/:/g, '-') + '.json';
                document.body.appendChild(a);
                a.click();
                document.body.removeChild(a);
                URL.revokeObjectURL(url);
                alert('复制失败，已自动下载结果文件');
            } finally {
                document.body.removeChild(textarea);
            }
        }
        
        function testConnection() {
            const registryInput = document.getElementById('registry') || document.getElementById('registryExpr');
            if (!registryInput || !registryInput.value.trim()) {
                showConnectionResult('请先输入注册中心地址', false);
                return;
            }

            const registry = registryInput.value.trim();
            const servicesList = document.getElementById('serviceList');
            
            // 找到所有测试连接按钮
            const testButtons = document.querySelectorAll('button[onclick="testConnection()"]');
            const originalTexts = [];
            
            // 显示测试中状态
            testButtons.forEach((button, index) => {
                originalTexts[index] = button.textContent;
                button.textContent = '测试中...';
                button.disabled = true;
            });
            
            // 在服务列表中显示测试状态
            servicesList.innerHTML = '<div style="padding: 20px; text-align: center; color: #666;"><div style="display: inline-block; width: 20px; height: 20px; border: 2px solid #f3f3f3; border-top: 2px solid #4a90e2; border-radius: 50%; animation: spin 1s linear infinite; margin-right: 10px;"></div>正在测试连接...</div>';
            
            fetch('/api/list', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    registry: registry,
                    app: document.getElementById('app') ? document.getElementById('app').value : 'dubbo-invoke-cli'
                })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    const serviceCount = data.services ? data.services.length : 0;
                    showConnectionResult('连接成功！发现 ' + serviceCount + ' 个服务', true);
                    // 显示服务列表
                    if (data.services && data.services.length > 0) {
                        displayServices(data.services);
                    }
                } else {
                    showConnectionResult('连接失败：' + (data.error || '未知错误'), false);
                }
            })
            .catch(error => {
                showConnectionResult('连接失败：' + error.message, false);
            })
            .finally(() => {
                // 恢复按钮状态
                testButtons.forEach((button, index) => {
                    button.textContent = originalTexts[index];
                    button.disabled = false;
                });
            });
        }
        
        function showConnectionResult(message, isSuccess) {
             const servicesList = document.getElementById('serviceList');
             const iconColor = isSuccess ? '#4caf50' : '#f44336';
             const icon = isSuccess ? '✅' : '❌';
             const bgColor = isSuccess ? '#e8f5e8' : '#ffeaea';
             const borderColor = isSuccess ? '#4caf50' : '#f44336';
             
             servicesList.innerHTML = 
                 '<div style="' +
                     'padding: 20px; ' +
                     'text-align: center; ' +
                     'background: ' + bgColor + '; ' +
                     'border: 1px solid ' + borderColor + '; ' +
                     'border-radius: 8px; ' +
                     'margin: 10px 0;' +
                     'color: ' + iconColor + ';' +
                     'font-weight: 500;' +
                 '">' +
                     '<div style="font-size: 24px; margin-bottom: 8px;">' + icon + '</div>' +
                     '<div>' + message + '</div>' +
                 '</div>';
         }
        
        window.onload = function() { loadHistory(); };
    </script>
</body>
</html>`

func (ws *WebServer) handleStaticFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		ws.writeError(w, "只支持GET方法")
		return
	}

	// 读取test_download.html文件
	filePath := "./test_download.html"
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Write(content)
}
