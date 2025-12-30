package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// (favicon 由磁盘读取)

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
	Namespace   string        `json:"namespace"`
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
	Namespace   string          `json:"namespace"`
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
	http.HandleFunc("/", ws.handleSimple)
	http.HandleFunc("/full", ws.handleIndex)
	http.HandleFunc("/simple", ws.handleSimple)
	http.HandleFunc("/api/invoke", ws.handleInvoke)
	http.HandleFunc("/api/list", ws.handleList)
	http.HandleFunc("/api/check-connection", ws.handleCheckConnection)
	http.HandleFunc("/api/methods", ws.handleMethods)
	http.HandleFunc("/api/example", ws.handleExample)
	http.HandleFunc("/api/history", ws.handleHistory)
	http.HandleFunc("/api/clear-history", ws.handleClearHistory)
	http.HandleFunc("/api/zk-environments", ws.handleZkEnvironments)

	// 添加静态文件服务
	http.Handle("/test_download.html", http.HandlerFunc(ws.handleStaticFile))
	// 添加favicon路由
	http.HandleFunc("/favicon.ico", ws.handleFavicon)

	// enhanceWebServerWithCompleteData(ws)
	http.HandleFunc("/api/test-precision", ws.handleTestPrecision)

	addr := fmt.Sprintf(":%d", ws.port)
	color.Green("🚀 Web UI服务器启动成功!")
	color.Cyan("📱 访问地址: http://localhost:%d", ws.port)
	color.Yellow("⚙️  默认注册中心: %s", ws.registry)
	color.Yellow("📦 默认应用名: %s", ws.app)
	color.Green("✨ 数据完整性增强: 已启用")
	fmt.Println()

	// 在Windows平台下，启动一个goroutine来保持控制台活跃
	// 注意：浏览器打开逻辑已移至main.go中统一处理
	if runtime.GOOS == "windows" && len(os.Args) <= 2 {
		// 保持控制台活跃
		go func() {
			for {
				time.Sleep(30 * time.Second)
				color.Green("💓 Web服务运行中... (按 Ctrl+C 停止)")
			}
		}()
	}

	// 启动Web服务器
	server := &http.Server{Addr: addr}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		color.Red("❌ Web服务器启动失败: %v", err)
		return err
	}
	return nil
}

// handleTestConnection 处理连接测试
func (ws *WebServer) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		ws.writeError(w, "只支持POST方法")
		return
	}

	var req struct {
		Address string `json:"address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ws.writeError(w, fmt.Sprintf("请求解析失败: %v", err))
		return
	}

	if req.Address == "" {
		ws.writeError(w, "地址不能为空")
		return
	}

	// 解析地址，如果包含协议前缀则去除
	address := req.Address
	if idx := strings.Index(address, "://"); idx != -1 {
		address = address[idx+3:]
	}

	// 尝试建立TCP连接
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("连接失败: %v", err),
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	defer conn.Close()

	response := map[string]interface{}{
		"success": true,
		"message": "连接成功",
	}
	json.NewEncoder(w).Encode(response)
}

// handleIndex 处理首页
func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 添加缓存控制头，防止浏览器缓存HTML页面
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	t := template.Must(template.New("index").Parse(indexHTML))
	data := map[string]interface{}{
		"Registry": ws.registry,
		"App":      ws.app,
		"Timeout":  ws.timeout,
		"Version":  time.Now().Unix(), // 添加时间戳作为版本号
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

	// 使用统一的参数处理中间件，支持BigInt和各种类型转换
	params, err := ws.processParametersWithBigIntSupport(req.Parameters)
	if err != nil {
		color.Red("[WEB] 参数处理失败: %v", err)
		ws.writeError(w, fmt.Sprintf("参数处理失败: %v", err))
		return
	}

	color.Blue("[WEB] 开始执行Dubbo调用: %s.%s", req.ServiceName, req.MethodName)
	// 记录开始时间
	startTime := time.Now()
	// 执行调用
	result, err := ws.executeInvoke(req, params)
	// 计算耗时
	duration := time.Since(startTime).Milliseconds()
	color.Cyan("[WEB] 调用耗时: %d ms", duration)

	// 保存调用历史
	// 直接使用解析后的参数，这样可以保存正确的对象格式
	historyParams := safeCopyParameters(params)
	color.Green("[WEB] 历史记录保存解析后的参数格式")

	history := CallHistory{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		ServiceName: req.ServiceName,
		MethodName:  req.MethodName,
		Parameters:  historyParams, // 使用正确处理的参数
		Types:       req.Types,
		Registry:    req.Registry,
		App:         req.App,
		Success:     err == nil,
		Timestamp:   time.Now(),
		Duration:    duration,
		Namespace:   req.Namespace,
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
	var registry, app, namespace string
	if r.Method == "POST" {
		var requestData struct {
			Registry  string `json:"registry"`
			App       string `json:"app"`
			Namespace string `json:"namespace"`
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
		namespace = requestData.Namespace
	} else {
		// 处理GET请求的查询参数
		registry = r.URL.Query().Get("registry")
		app = r.URL.Query().Get("app")
		namespace = r.URL.Query().Get("namespace")
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
		Namespace:   namespace,
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

// handleCheckConnection 处理连接检查（轻量级）
func (ws *WebServer) handleCheckConnection(w http.ResponseWriter, r *http.Request) {
	color.Green("[WEB] 收到连接检查请求: %s %s", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 处理POST请求的JSON数据
	var registry, app, namespace string
	if r.Method == "POST" {
		var requestData struct {
			Registry  string `json:"registry"`
			App       string `json:"app"`
			Namespace string `json:"namespace"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
			color.Red("[WEB] 解析请求数据失败: %v", err)
			response := map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("解析请求数据失败: %v", err),
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		registry = requestData.Registry
		app = requestData.App
		namespace = requestData.Namespace
	} else {
		// 处理GET请求的查询参数
		registry = r.URL.Query().Get("registry")
		app = r.URL.Query().Get("app")
		namespace = r.URL.Query().Get("namespace")
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
		Namespace:   namespace,
	}
	color.Cyan("[WEB] 创建Dubbo客户端配置(仅检查连接): 注册中心=%s, 应用=%s, 超时=%dms", config.Registry, config.Application, ws.timeout)

	// 创建真实的dubbo客户端
	// NewRealDubboClient 内部会尝试连接注册中心，如果连接失败会返回错误
	client, err := NewRealDubboClient(config)
	if err != nil {
		color.Red("[WEB] 连接注册中心失败: %v", err)
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("连接注册中心失败: %v", err),
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
		response := map[string]interface{}{
			"success": false,
			"error":   "无法连接到dubbo注册中心",
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	color.Green("[WEB] Dubbo客户端连接成功")

	response := map[string]interface{}{
		"success": true,
		"message": "连接成功",
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
// removeLSuffix 移除JSON字符串中数字的L后缀
func (ws *WebServer) removeLSuffix(jsonStr string) string {
	// 使用正则表达式匹配数字后面的L后缀
	// 匹配模式：数字(可能包含小数点)后跟L，但L后面必须是非字母数字字符或字符串结尾
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)L([^a-zA-Z0-9]|$)`)
	return re.ReplaceAllString(jsonStr, "${1}${2}")
}

func (ws *WebServer) parseParameter(param string) (interface{}, error) {
	color.Cyan("[WEB] 开始解析参数: %s", param)

	// 去除首尾空格
	param = strings.TrimSpace(param)
	color.Cyan("[WEB] 去除空格后的参数: %s", param)

	// 预处理：移除JSON中的L后缀
	param = ws.removeLSuffix(param)
	color.Cyan("[WEB] 移除L后缀后的参数: %s", param)

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
func (ws *WebServer) executeInvoke(req InvokeRequest, params []interface{}) (interface{}, error) {
	color.Blue("[WEB] 开始执行Dubbo调用: %s.%s", req.ServiceName, req.MethodName)
	color.Cyan("[WEB] 调用参数: Registry=%s, App=%s, Timeout=%dms", req.Registry, req.App, req.Timeout)

	// 创建Dubbo客户端配置
	cfg := &DubboConfig{
		Registry:    req.Registry,
		Application: req.App,
		Timeout:     time.Duration(req.Timeout) * time.Millisecond,
		Namespace:   req.Namespace,
	}
	color.Green("[WEB] Dubbo客户端配置创建成功")

	// 使用传入的已解析参数
	color.Green("[WEB] 使用已解析的参数，参数数量: %d", len(params))
	// 根据类型提示进行参数修正（例如明确指定为字符串类型时，强制使用字符串）
	if len(req.Types) > 0 {
		params = applyTypeHints(req.Types, params)
	}

	// 智能类型补全：当无类型信息且仅一个参数为List时，自动设置为java.util.List
	if len(req.Types) == 0 && len(params) == 1 {
		if _, ok := params[0].([]interface{}); ok {
			color.Yellow("[WEB] 检测到单参数为列表，自动设置类型为 java.util.List")
			req.Types = []string{"java.util.List"}
		}
	}

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
func maybeParseJSONString(s string) interface{} {
	st := strings.TrimSpace(s)
	if (strings.HasPrefix(st, "{") && strings.HasSuffix(st, "}")) ||
		(strings.HasPrefix(st, "[") && strings.HasSuffix(st, "]")) {
		var v interface{}
		decoder := json.NewDecoder(strings.NewReader(st))
		decoder.UseNumber()
		if err := decoder.Decode(&v); err == nil {
			return v
		}
	}
	return s
}

func convertJSONNumbers(params []interface{}) []interface{} {
	result := make([]interface{}, len(params))
	for i, param := range params {
		if str, ok := param.(string); ok {
			param = maybeParseJSONString(str)
		}
		result[i] = convertJSONNumber(param)
	}
	return result
}

// processParametersWithBigIntSupport 统一的参数处理中间件，支持BigInt和各种类型转换
func (ws *WebServer) processParametersWithBigIntSupport(rawParams json.RawMessage) ([]interface{}, error) {
	if len(rawParams) == 0 {
		return []interface{}{}, nil
	}

	// 移除Java long类型的L后缀
	processedParams := ws.removeLSuffix(string(rawParams))
	color.Cyan("[WEB] 处理后的参数: %s", processedParams)

	// 首先尝试解析为参数数组
	var paramArray []interface{}
	decoder := json.NewDecoder(strings.NewReader(processedParams))
	decoder.UseNumber()
	err := decoder.Decode(&paramArray)
	if err == nil {
		// 成功解析为数组，应用类型转换
		result := convertJSONNumbers(paramArray)
		color.Green("[WEB] 解析为多参数格式，参数数量: %d", len(result))
		return result, nil
	}

	// 如果不是数组格式，尝试解析为单个参数
	var singleParam interface{}
	decoder = json.NewDecoder(strings.NewReader(processedParams))
	decoder.UseNumber()
	err = decoder.Decode(&singleParam)
	if err == nil {
		if str, ok := singleParam.(string); ok {
			singleParam = maybeParseJSONString(str)
		}
		result := []interface{}{convertJSONNumber(singleParam)}
		color.Green("[WEB] 解析为单参数格式，参数数量: 1")
		return result, nil
	}

	// 如果JSON解析失败，检查是否为表达式格式的参数字符串
	if processedParams != "" {
		// 检查是否为JSON字符串格式
		if strings.HasPrefix(processedParams, `"`) && strings.HasSuffix(processedParams, `"`) {
			// 去除外层引号
			unquoted := processedParams[1 : len(processedParams)-1]
			// 尝试将去除引号后的内容再次解析为数组
			var innerArray []interface{}
			decoder := json.NewDecoder(strings.NewReader(unquoted))
			decoder.UseNumber()
			err := decoder.Decode(&innerArray)
			if err == nil {
				// 成功解析为数组
				result := convertJSONNumbers(innerArray)
				color.Green("[WEB] 解析为JSON字符串内的数组格式，参数数量: %d", len(result))
				return result, nil
			}

			// 如果不是数组，尝试使用表达式解析器解析参数
			color.Yellow("[WEB] 尝试使用表达式解析器解析参数: %s", unquoted)
			params := parseParametersFromExpression(unquoted)
			if len(params) > 0 {
				result := make([]interface{}, len(params))
				for i, param := range params {
					// 尝试解析每个参数
					if parsed, err := ws.parseParameter(param); err == nil {
						result[i] = parsed
					} else {
						result[i] = param
					}
				}
				color.Green("[WEB] 表达式解析成功，参数数量: %d", len(result))
				return result, nil
			}

			// 如果表达式解析也失败，作为单个字符串参数
			result := []interface{}{unquoted}
			color.Yellow("[WEB] 解析为JSON字符串格式，参数数量: 1")
			return result, nil
		}

		// 尝试使用表达式解析器解析参数（处理非引号包围的情况）
		color.Yellow("[WEB] 尝试使用表达式解析器解析参数: %s", processedParams)
		params := parseParametersFromExpression(processedParams)
		if len(params) > 0 {
			result := make([]interface{}, len(params))
			for i, param := range params {
				// 尝试解析每个参数
				if parsed, err := ws.parseParameter(param); err == nil {
					result[i] = parsed
				} else {
					result[i] = param
				}
			}
			color.Green("[WEB] 表达式解析成功，参数数量: %d", len(result))
			return result, nil
		}

		// 作为普通字符串处理
		result := []interface{}{processedParams}
		color.Yellow("[WEB] 参数解析失败，作为字符串处理: %s", processedParams)
		return result, nil
	}

	return []interface{}{}, nil
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
	case string:
		// 处理BigInt格式的字符串（来自前端JSONBig.parse）
		return convertBigIntString(v)
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

// convertBigIntString 处理来自前端BigInt的字符串格式
func convertBigIntString(s string) interface{} {
	return s
}

func applyTypeHints(types []string, params []interface{}) []interface{} {
	if len(types) == 0 || len(params) == 0 {
		return params
	}
	res := make([]interface{}, len(params))
	var deepToString func(v interface{}) interface{}
	deepToString = func(v interface{}) interface{} {
		switch x := v.(type) {
		case json.Number:
			return string(x)
		case float64, float32, int64, int32, int, uint64, uint32, uint:
			return fmt.Sprintf("%v", x)
		case []interface{}:
			out := make([]interface{}, len(x))
			for i := range x {
				out[i] = deepToString(x[i])
			}
			return out
		case map[string]interface{}:
			out := make(map[string]interface{}, len(x))
			for k, vv := range x {
				out[k] = deepToString(vv)
			}
			return out
		default:
			return v
		}
	}
	for i := range params {
		t := ""
		if i < len(types) {
			t = types[i]
		}
		p := params[i]
		switch t {
		case "java.lang.String", "string":
			switch v := p.(type) {
			case string:
				res[i] = v
			case json.Number:
				res[i] = string(v)
			default:
				res[i] = fmt.Sprintf("%v", v)
			}
		case "java.lang.Object":
			res[i] = deepToString(p)
		default:
			if strings.HasPrefix(t, "com.") || strings.Contains(t, ".") {
				res[i] = deepToString(p)
			} else {
				res[i] = p
			}
		}
	}
	return res
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

// ZookeeperEnvironment Zookeeper环境配置
type ZookeeperEnvironment struct {
	Name        string `json:"name"`
	Address     string `json:"address"`
	ServicePath string `json:"servicePath"`
}

// getZookeeperEnvironments 获取Zookeeper环境配置
func getZookeeperEnvironments() map[string]ZookeeperEnvironment {
	return map[string]ZookeeperEnvironment{
		"dev": {
			Name:        "开发环境",
			Address:     "10.7.8.40:2181",
			ServicePath: "dubbo",
		},
		"uat": {
			Name:        "用户验收测试",
			Address:     "10.7.8.42:2181",
			ServicePath: "uat",
		},
		"tat": {
			Name:        "技术验收测试",
			Address:     "10.6.12.153:2181",
			ServicePath: "tat",
		},
		"fat": {
			Name:        "功能验收测试",
			Address:     "10.6.12.205:2181",
			ServicePath: "fat",
		},
		"pre": {
			Name:        "预生产环境",
			Address:     "mse-4ec83a20-zk.mse.aliyuncs.com:2181",
			ServicePath: "pre",
		},
		"prod": {
			Name:        "生产环境",
			Address:     "mse-2cd54c90-zk.mse.aliyuncs.com:2181",
			ServicePath: "prod",
		},
	}
}

// indexHTML 首页HTML模板
const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dubbo Invoke Web UI</title>
    <link rel="icon" type="image/png" href="/favicon.ico">
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
            gap: 10px; 
            padding: 10px;
            min-height: calc(100vh - 200px);
        }
        
        .top-row {
            display: flex;
            gap: 10px;
            flex: 0 0 auto;
            height: 800px;
            margin: 0 10px;
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
            gap: 10px;
            padding-right: 10px;
        }
        .service-call-panel { 
            flex: 0 0 auto;
            height: 810px;
            min-height: 500px;
            max-height: 810px;
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
            overflow-x: hidden;
            border: 1px solid #e0e0e0;
            border-radius: 3px;
            background: white;
        }
        .history-list .history-item {
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            min-width: 0;
            position: relative;
        }
        .history-list .history-item:hover {
            overflow-x: auto;
            overflow-y: hidden;
            white-space: nowrap;
            z-index: 10;
            background: #f8f9fa;
        }
        .history-list .history-item:hover .service-name {
            white-space: nowrap;
            overflow-x: auto;
            overflow-y: hidden;
            text-overflow: unset;
        }
        /* 调用结果面板独立显示在底部 */
        .result-panel { 
            min-height: 200px;
            flex-shrink: 0;
            margin-top: 10px;
            margin-left: 10px;
            margin-right: 10px;
            width: calc(100% - 20px);
            max-width: calc(100% - 20px);
        }
        
        /* 响应式布局 - 窄屏时单列显示 */
        @media (max-width: 1024px) {
            .top-row {
                flex-direction: column;
                height: auto;
                gap: 20px;
            }
            
            .left-column, .right-column {
                flex: none;
                width: 100%;
                padding-right: 0;
            }
            
            .service-call-panel {
                height: auto;
                min-height: 400px;
                max-height: none;
            }
            
            .available-services-panel, .history-panel {
                height: 350px;
                min-height: 300px;
                max-height: 400px;
            }
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

        /* 行号显示样式 */
        .result.show-line-numbers {
            counter-reset: line-number;
            padding-left: 50px;
            position: relative;
        }

        .result.show-line-numbers::before {
            content: '';
            position: absolute;
            left: 0;
            top: 0;
            bottom: 0;
            width: 40px;
            background: #f8f9fa;
            border-right: 1px solid #e0e0e0;
        }

        .result.show-line-numbers .line {
            counter-increment: line-number;
            position: relative;
        }

        .result.show-line-numbers .line::before {
            content: counter(line-number);
            position: absolute;
            left: -45px;
            width: 35px;
            text-align: right;
            color: #666;
            font-size: 11px;
            line-height: inherit;
            padding-right: 8px;
        }

        /* JSON树形展示样式 */
        .json-tree {
            font-family: monospace;
            font-size: 13px;
            line-height: 1.4;
            color: #333;
            white-space: normal;
            word-break: normal;
            overflow-wrap: normal;
        }

        .json-tree-item {
            margin: 0;
            padding: 0;
        }

        .json-tree-toggle {
            display: inline-block;
            width: 12px;
            height: 12px;
            margin-right: 4px;
            cursor: pointer;
            user-select: none;
            font-size: 10px;
            line-height: 12px;
            text-align: center;
            color: #666;
            border: 1px solid #ccc;
            background: #fff;
            border-radius: 2px;
            vertical-align: middle;
        }

        .json-tree-toggle:hover {
            background: #f0f0f0;
            border-color: #999;
        }

        .json-tree-toggle.collapsed::before {
            content: '▶';
        }

        .json-tree-toggle.expanded::before {
            content: '▼';
        }

        .json-tree-toggle.leaf {
            visibility: hidden;
        }

        .json-tree-key {
            color: #0066cc;
            font-weight: bold;
        }

        .json-tree-string {
            color: #008000;
        }

        .json-tree-number {
            color: #ff6600;
        }

        .json-tree-boolean {
            color: #cc0066;
            font-style: italic;
        }

        .json-tree-null {
            color: #999;
            font-style: italic;
        }

        .json-tree-bracket {
            color: #666;
            font-weight: bold;
        }

        .json-tree-children {
            margin-left: 20px;
            border-left: 1px dotted #ccc;
            padding-left: 10px;
        }

        .json-tree-children.collapsed {
            display: none;
        }

        .json-tree-item-line {
            padding: 1px 0;
        }

        .json-tree-summary {
            color: #999;
            font-style: italic;
            margin-left: 4px;
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
            overflow-x: hidden;
            border: 1px solid #e0e0e0;
            border-radius: 3px;
            background: white;
        }
        .service-item {
            padding: 12px 16px; border-bottom: 1px solid #e9ecef;
            cursor: pointer; transition: all 0.2s ease;
            position: relative;
            max-width: 100%;
            min-width: 0;
            flex-shrink: 1;
            overflow: hidden;
            white-space: nowrap;
            text-overflow: ellipsis;
        }
        .service-item:hover {
            overflow-x: auto;
            overflow-y: hidden;
            white-space: nowrap;
            background: #f8f9fa;
            z-index: 10;
        }
        }
        .service-item::-webkit-scrollbar {
            height: 4px;
        }
        .service-item::-webkit-scrollbar-track {
            background: #f1f1f1;
            border-radius: 2px;
        }
        .service-item::-webkit-scrollbar-thumb {
            background: #c1c1c1;
            border-radius: 2px;
        }
        .service-item::-webkit-scrollbar-thumb:hover {
            background: #a8a8a8;
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
                                <option value="expression" selected>表达式格式 (service.method(params))</option>
                            </select>
                        </div>
                        <div id="traditionalFormat">
                            <div class="form-group">
                                <label>注册中心配置:</label>
                                <div style="display: flex; gap: 10px; align-items: center; margin-bottom: 10px;">
                                    <select id="registryType" onchange="onRegistryTypeChange()" style="width: 120px; flex-shrink: 0;">
                                        <option value="zookeeper">ZooKeeper</option>
                                        <option value="nacos">Nacos</option>
                                        <option value="dubbo">Dubbo</option>
                                    </select>
                                    <div id="zkEnvironmentContainer" style="display: none; flex: 1;">
                                        <select id="zkEnvironment" onchange="onZkEnvironmentChange()" style="width: 100%;">
                                            <option value="">请选择环境</option>
                                            <option value="dev">开发环境 (dev)</option>
                                            <option value="uat">用户验收测试 (uat)</option>
                                            <option value="tat">技术验收测试 (tat)</option>
                                            <option value="fat">功能验收测试 (fat)</option>
                                            <option value="pre">预生产环境 (pre)</option>
                                            <option value="prod">生产环境 (prod)</option>
                                        </select>
                                    </div>
                                    <input type="text" id="registryAddress" placeholder="127.0.0.1:2181" value="127.0.0.1:2181" style="flex: 1;" readonly>
                                    <button class="btn btn-secondary" onclick="testConnection()" style="margin: 0; white-space: nowrap;">🔗 测试连接</button>
                                </div>
                            </div>
                            <div class="form-group" id="namespaceGroup" style="display: none;">
                                <label for="namespace">命名空间 (可选):</label>
                                <input type="text" id="namespace" placeholder="public" value="public">
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
                                <label>注册中心配置:</label>
                                <div style="display: flex; gap: 10px; align-items: center; margin-bottom: 10px;">
                                    <select id="registryTypeExpr" onchange="onRegistryTypeChangeExpr()" style="width: 120px; flex-shrink: 0;">
                                        <option value="zookeeper">ZooKeeper</option>
                                        <option value="nacos">Nacos</option>
                                        <option value="dubbo">Dubbo</option>
                                    </select>
                                    <div id="zkEnvironmentContainerExpr" style="display: none; flex: 1;">
                                        <select id="zkEnvironmentExpr" onchange="onZkEnvironmentChangeExpr()" style="width: 100%;">
                                            <option value="">请选择环境</option>
                                            <option value="dev">开发环境 (dev)</option>
                                            <option value="uat">用户验收测试 (uat)</option>
                                            <option value="tat">技术验收测试 (tat)</option>
                                            <option value="fat">功能验收测试 (fat)</option>
                                            <option value="pre">预生产环境 (pre)</option>
                                            <option value="prod">生产环境 (prod)</option>
                                        </select>
                                    </div>
                                    <input type="text" id="registryAddressExpr" value="{{.Registry}}" placeholder="127.0.0.1:2181" style="flex: 1;" readonly>
                                    <button class="btn btn-secondary" onclick="testConnection()" style="margin: 0; white-space: nowrap;">🔗 测试连接</button>
                                </div>
                            </div>
                            <div class="form-group" id="namespaceGroupExpr" style="display: block;">
                                <label for="namespaceExpr">命名空间 (可选):</label>
                                <input type="text" id="namespaceExpr" placeholder="public" value="public">
                            </div>
                            <div class="form-group">
                                <label for="expression">调用表达式: <span style="font-size: 0.8em; color: #5c6bc0;">(service.method(params))</span></label>
                                <textarea id="expression" placeholder='invoke com.example.UserService.getUserById(123)'>invoke com.example.UserService.getUserById(123)</textarea>
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
                        <div class="service-search-container" style="margin-bottom: 10px; display: none;" id="serviceSearchContainer">
                            <input type="text" id="serviceSearch" placeholder="搜索服务名称..." 
                                   style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px;"
                                   oninput="filterServices()">
                        </div>
                        <div id="serviceList" class="service-list">
                            <div style="padding: 20px; text-align: center; color: #6c757d;">
                                <p>请先连接注册中心</p>
                            </div>
                        </div>
                        <div class="service-pagination" style="display: none; text-align: center; margin-top: 10px;" id="servicePagination">
                            <button id="loadMoreBtn" class="btn btn-secondary" onclick="loadMoreServices()" style="padding: 8px 16px; font-size: 14px;">加载更多服务</button>
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
                        <button class="icon-btn compress" onclick="toggleJsonFormat()" title="压缩/美化JSON">
                            🗜️
                        </button>
                        <button class="icon-btn line-numbers" onclick="toggleLineNumbers()" title="显示/隐藏行号">
                            🔢
                        </button>
                        <button class="icon-btn expand-all" onclick="toggleExpandAll()" title="全部展开/收缩">
                            📂
                        </button>
                        <button class="icon-btn copy" onclick="copyResult()" title="复制结果">
                            📋
                        </button>
                        <button class="icon-btn save" onclick="saveResult()" title="保存结果">
                            💾
                        </button>
                        <button class="icon-btn clear" onclick="clearResult()" title="清空结果">
                            🗑️
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
        // Zookeeper环境配置
        const ZOOKEEPER_ENVIRONMENTS = {
            dev: {
                name: '开发环境',
                address: '10.7.8.40:2181',
                servicePath: 'dubbo'
            },
            uat: {
                name: '用户验收测试',
                address: '10.7.8.42:2181',
                servicePath: 'uat'
            },
            tat: {
                name: '技术验收测试',
                address: '10.6.12.153:2181',
                servicePath: 'tat'
            },
            fat: {
                name: '功能验收测试',
                address: '10.6.12.205:2181',
                servicePath: 'fat'
            },
            pre: {
                name: '预生产环境',
                address: 'mse-4ec83a20-zk.mse.aliyuncs.com:2181',
                servicePath: 'pre'
            },
            prod: {
                name: '生产环境',
                address: 'mse-2cd54c90-zk.mse.aliyuncs.com:2181',
                servicePath: 'prod'
            }
        };

        // 全局变量存储原始JSON数据用于复制
        let originalJsonData = null;
        let isJsonCompressed = false;
        let showLineNumbers = false;
        let isAllExpanded = true;
        
        function toggleCallFormat() {
            const callFormatEl = document.getElementById('callFormat');
            if (!callFormatEl) return;
            
            const format = callFormatEl.value;
            const traditional = document.getElementById('traditionalFormat');
            const expression = document.getElementById('expressionFormat');
            const traditionalTypes = document.getElementById('traditionalTypes');
            
            if (!traditional || !expression || !traditionalTypes) return;
            
            if (format === 'expression') {
                traditional.style.display = 'none';
                expression.style.display = 'block';
                traditionalTypes.style.display = 'none';
                // 同步注册中心类型、地址和命名空间值
                const registryTypeEl = document.getElementById('registryType');
                const registryAddressEl = document.getElementById('registryAddress');
                const namespaceEl = document.getElementById('namespace');
                const registryTypeExprEl = document.getElementById('registryTypeExpr');
                const registryAddressExprEl = document.getElementById('registryAddressExpr');
                const namespaceExprEl = document.getElementById('namespaceExpr');
                
                if (registryTypeEl && registryTypeExprEl) {
                    registryTypeExprEl.value = registryTypeEl.value;
                }
                if (registryAddressEl && registryAddressExprEl) {
                    registryAddressExprEl.value = registryAddressEl.value;
                }
                if (namespaceEl && namespaceExprEl) {
                    namespaceExprEl.value = namespaceEl.value;
                }
                // 触发表达式模式的注册中心类型变化事件
                onRegistryTypeChangeExpr();
            } else {
                traditional.style.display = 'block';
                expression.style.display = 'none';
                traditionalTypes.style.display = 'block';
                // 同步注册中心类型、地址和命名空间值
                const registryTypeEl = document.getElementById('registryType');
                const registryAddressEl = document.getElementById('registryAddress');
                const namespaceEl = document.getElementById('namespace');
                const registryTypeExprEl = document.getElementById('registryTypeExpr');
                const registryAddressExprEl = document.getElementById('registryAddressExpr');
                const namespaceExprEl = document.getElementById('namespaceExpr');
                
                if (registryTypeEl && registryTypeExprEl) {
                    registryTypeEl.value = registryTypeExprEl.value;
                }
                if (registryAddressEl && registryAddressExprEl) {
                    registryAddressEl.value = registryAddressExprEl.value;
                }
                if (namespaceEl && namespaceExprEl) {
                    namespaceEl.value = namespaceExprEl.value;
                }
                // 触发传统模式的注册中心类型变化事件
                onRegistryTypeChange();
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
                // 先应用removeLSuffix处理，确保L后缀被正确移除
                const processedParamsPart = removeLSuffix(paramsPart.trim());
                
                try {
                    // 首先尝试将整个参数部分作为JSON数组解析（只有当它是完整的JSON数组格式时）
                    if (processedParamsPart.startsWith('[') && processedParamsPart.endsWith(']')) {
                        try {
                            // 将整体数组视为单个参数（List），避免被拆分为多个独立参数
                            parameters = [JSONBig.parse(processedParamsPart)];
                        } catch (e) {
                            // 如果解析失败，说明不是有效的JSON数组，使用参数分割逻辑
                            throw e;
                        }
                    } else {
                        // 不是完整的JSON数组格式，使用参数分割逻辑
                        throw new Error('Not a complete JSON array');
                    }
                } catch (e) {
                    // 使用智能参数分割逻辑
                    const paramStrings = parseParametersFromExpression(processedParamsPart);
                    parameters = paramStrings.map(paramStr => {
                        try {
                            return JSONBig.parse(paramStr);
                        } catch (e) {
                            // 如果不是JSON格式，去除引号后返回字符串
                            if (paramStr.startsWith('"') && paramStr.endsWith('"')) {
                                return paramStr.substring(1, paramStr.length - 1);
                            }
                            // 尝试解析为数字
                            if (!isNaN(paramStr) && !isNaN(parseFloat(paramStr))) {
                                return parseFloat(paramStr);
                            }
                            return paramStr;
                        }
                    });
                }
            }
            return { serviceName, methodName, parameters };
        }
        
        // 从表达式中解析参数的函数，与后端逻辑保持一致
        function parseParametersFromExpression(paramsPart) {
            if (!paramsPart || paramsPart.trim() === '') {
                return [];
            }
            
            const params = [];
            let current = '';
            let braceCount = 0;
            let bracketCount = 0;
            let inQuotes = false;
            let escapeNext = false;
            
            for (let i = 0; i < paramsPart.length; i++) {
                const char = paramsPart[i];
                
                if (escapeNext) {
                    current += char;
                    escapeNext = false;
                    continue;
                }
                
                if (char === '\\') {
                    escapeNext = true;
                    current += char;
                    continue;
                }
                
                if (char === '"') {
                    inQuotes = !inQuotes;
                }
                
                if (!inQuotes) {
                    if (char === '{') {
                        braceCount++;
                    } else if (char === '}') {
                        braceCount--;
                    } else if (char === '[') {
                        bracketCount++;
                    } else if (char === ']') {
                        bracketCount--;
                    } else if (char === ',' && braceCount === 0 && bracketCount === 0) {
                        // 找到参数分隔符
                        const param = current.trim();
                        if (param !== '') {
                            params.push(param);
                        }
                        current = '';
                        continue;
                    }
                }
                
                current += char;
            }
            
            // 添加最后一个参数
            const param = current.trim();
            if (param !== '') {
                params.push(param);
            }
            
            return params;
        }
        // JSON-BigInt 库的简化实现，用于处理大整数
        const JSONBig = {
            parse: function(text, reviver) {
                // 首先移除Java long字面量的L后缀
                let cleanText = text.replace(/(\d+(?:\.\d+)?)L([^a-zA-Z0-9]|$)/g, '$1$2');
                
                // 仅在非字符串上下文中将超过15位的整数包装为字符串
                (function() {
                    let s = cleanText;
                    let res = '';
                    let inQuotes = false;
                    let escapeNext = false;
                    let i = 0;
                    while (i < s.length) {
                        const ch = s[i];
                        if (escapeNext) {
                            res += ch;
                            escapeNext = false;
                            i++;
                            continue;
                        }
                        if (ch === '\\') {
                            res += ch;
                            escapeNext = true;
                            i++;
                            continue;
                        }
                        if (ch === '"') {
                            res += ch;
                            inQuotes = !inQuotes;
                            i++;
                            continue;
                        }
                        if (!inQuotes && (ch === '-' || (ch >= '0' && ch <= '9'))) {
                            let j = i;
                            if (ch === '-') j++;
                            while (j < s.length && s[j] >= '0' && s[j] <= '9') j++;
                            const num = s.slice(i, j);
                            let hasL = false;
                            if (j < s.length && s[j] === 'L') {
                                hasL = true;
                                j++;
                            }
                            const prev = res.length ? res[res.length - 1] : '';
                            const next = j < s.length ? s[j] : '';
                            const prevWord = /[A-Za-z0-9_]/.test(prev);
                            const nextWord = /[A-Za-z0-9_.]/.test(next);
                            if (num.length >= 16 && !prevWord && !nextWord) {
                                res += '"' + num + '"';
                            } else {
                                res += s.slice(i, j);
                            }
                            i = j;
                            continue;
                        }
                        res += ch;
                        i++;
                    }
                    cleanText = res;
                })();
                
                try {
                    return JSON.parse(cleanText, function(key, value) {
                        // 检查是否为大整数字符串（被我们转换的）
                        if (typeof value === 'string' && /^-?\d{16,}$/.test(value)) {
                            // 超过15位的整数，保持为字符串避免精度丢失
                            return value;
                        }
                        
                        // 检查是否为超出安全范围的数字
                        if (typeof value === 'number') {
                            if (value > Number.MAX_SAFE_INTEGER || value < Number.MIN_SAFE_INTEGER) {
                                return value.toString();
                            }
                        }
                        
                        return reviver ? reviver(key, value) : value;
                    });
                } catch (error) {
                    // 降级处理：如果解析失败，尝试原始字符串
                    console.warn('JSON解析失败，使用原始字符串:', error);
                    return cleanText;
                }
            },
            
            stringify: function(value, replacer, space) {
                return JSON.stringify(value, function(key, val) {
                    // 处理大整数，确保序列化时保持精度
                    if (typeof val === 'bigint') {
                        return val.toString();
                    }
                    return replacer ? replacer(key, val) : val;
                }, space);
            }
        };
        
        // 兼容性函数：保持向后兼容
        function removeLSuffix(jsonStr) {
            return jsonStr.replace(/(\d+(?:\.\d+)?)L([^a-zA-Z0-9]|$)/g, '$1$2');
        }
        
        function invokeService() {
            const format = document.getElementById('callFormat').value;
            let serviceName, methodName, parameters;
            if (format === 'expression') {
                let expr = document.getElementById('expression').value.trim();
                if (!expr) { alert('请输入调用表达式'); return; }
                // 如果表达式以"invoke "开头，去除该前缀
                if (expr.startsWith('invoke ')) {
                    expr = expr.substring(7); // 去除"invoke "前缀
                }
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
                    // 使用JSONBig解析参数，支持大整数和Java long类型
                    parameters = paramsText ? JSONBig.parse(paramsText) : [];
                } catch (e) { alert('参数格式错误，请使用JSON数组格式: ' + e.message); return; }
            }
            // 获取参数类型信息
            let types = '';
            if (format === 'traditional') {
                types = document.getElementById('types').value.trim();
            } else {
                // 表达式格式：根据参数自动推断类型
                if (parameters && Array.isArray(parameters) && parameters.length > 0) {
                    types = parameters.map(param => {
                        if (typeof param === 'string') {
                            return 'java.lang.String';
                        } else if (typeof param === 'number') {
                            if (Number.isInteger(param)) {
                                // 检查是否超出Integer范围，如果超出则使用Long
                                if (param > 2147483647 || param < -2147483648) {
                                    return 'java.lang.Long';
                                } else {
                                    return 'java.lang.Integer';
                                }
                            } else {
                                return 'java.lang.Double';
                            }
                        } else if (typeof param === 'boolean') {
                            return 'java.lang.Boolean';
                        } else if (Array.isArray(param)) {
                            return 'java.util.List';
                        } else if (typeof param === 'object' && param !== null) {
                            return 'java.lang.Object';
                        } else {
                            return 'java.lang.Object';
                        }
                    }).join(',');
                }
            }
            
            // 获取注册中心类型和地址
            let registryType, registryAddress;
            if (format === 'expression') {
                registryType = document.getElementById('registryTypeExpr').value;
                registryAddress = document.getElementById('registryAddressExpr').value.trim();
            } else {
                registryType = document.getElementById('registryType').value;
                registryAddress = document.getElementById('registryAddress').value.trim();
            }
            
            if (!registryAddress) {
                alert('请先输入注册中心地址');
                return;
            }
            
            // 根据注册中心类型构建完整的registry地址
            let registry;
            if (registryType === 'zookeeper') {
                registry = 'zookeeper://' + registryAddress;
            } else if (registryType === 'nacos') {
                registry = 'nacos://' + registryAddress;
            } else if (registryType === 'dubbo') {
                registry = 'dubbo://' + registryAddress;
            } else {
                registry = registryAddress;
            }
            
            let namespace;
            if (format === 'expression') {
                const namespaceExprElement = document.getElementById('namespaceExpr');
                namespace = namespaceExprElement ? namespaceExprElement.value.trim() : 'public';
            } else {
                const namespaceElement = document.getElementById('namespace');
                namespace = namespaceElement ? namespaceElement.value.trim() : 'public';
            }
            const request = {
                serviceName: serviceName, methodName: methodName,
                parameters: parameters,
                types: types ? types.split(',').map(t => t.trim()) : [],
                registry: registry, app: '{{.App}}', timeout: 10000,
                namespace: namespace
            };
            showLoading(true);
            const startTime = Date.now(); // 记录前端调用开始时间
            fetch('/api/invoke', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSONBig.stringify(request)
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
                        document.getElementById('expression').value = 'invoke ' + serviceName + '.' + methodName + '(' + params + ')';
                    } else {
                        document.getElementById('parameters').value = JSON.stringify(data.examples, null, 2);
                    }
                } else { alert('生成示例失败: ' + data.error); }
            })
            .catch(error => { alert('生成示例失败: ' + error.message); });
        }
        function loadServices() {
            const currentFormat = document.getElementById('callFormat').value;
            let registryType, registryAddress;
            
            if (currentFormat === 'expression') {
                registryType = document.getElementById('registryTypeExpr').value;
                registryAddress = document.getElementById('registryAddressExpr').value.trim();
            } else {
                registryType = document.getElementById('registryType').value;
                registryAddress = document.getElementById('registryAddress').value.trim();
            }
            
            if (!registryAddress) {
                document.getElementById('serviceList').innerHTML = 
                    '<div style="padding: 20px; text-align: center; color: #6c757d;">请先配置注册中心</div>';
                return;
            }
            
            // 根据注册中心类型构建完整的registry地址
            let registry;
            if (registryType === 'zookeeper') {
                registry = 'zookeeper://' + registryAddress;
            } else if (registryType === 'nacos') {
                registry = 'nacos://' + registryAddress;
            } else if (registryType === 'dubbo') {
                registry = 'dubbo://' + registryAddress;
            } else {
                registry = registryAddress;
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
        // 全局变量用于分页和搜索
        let allServices = [];
        let displayedServices = [];
        let currentPage = 0;
        const pageSize = 20;
        let filteredServices = [];
        
        function displayServices(services) {
            allServices = services || [];
            filteredServices = [...allServices];
            currentPage = 0;
            displayedServices = [];
            
            const serviceList = document.getElementById('serviceList');
            const searchContainer = document.getElementById('serviceSearchContainer');
            const pagination = document.getElementById('servicePagination');
            
            if (!services || services.length === 0) {
                serviceList.innerHTML = '<div style="padding: 20px; text-align: center; color: #6c757d;"><i>暂无可用服务</i></div>';
                searchContainer.style.display = 'none';
                pagination.style.display = 'none';
                return;
            }
            
            // 显示搜索框
            searchContainer.style.display = 'block';
            
            // 加载第一页
            loadMoreServices();
        }
        
        function loadMoreServices() {
            const serviceList = document.getElementById('serviceList');
            const pagination = document.getElementById('servicePagination');
            const loadMoreBtn = document.getElementById('loadMoreBtn');
            
            const startIndex = currentPage * pageSize;
            const endIndex = Math.min(startIndex + pageSize, filteredServices.length);
            const newServices = filteredServices.slice(startIndex, endIndex);
            
            if (currentPage === 0) {
                serviceList.innerHTML = '';
                displayedServices = [];
            }
            
            newServices.forEach(service => {
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
            
            displayedServices = displayedServices.concat(newServices);
            currentPage++;
            
            // 更新分页按钮
            if (endIndex >= filteredServices.length) {
                pagination.style.display = 'none';
            } else {
                pagination.style.display = 'block';
                loadMoreBtn.textContent = '加载更多服务 (' + displayedServices.length + '/' + filteredServices.length + ')';
            }
        }
        
        function filterServices() {
            const searchTerm = document.getElementById('serviceSearch').value.toLowerCase().trim();
            
            if (!searchTerm) {
                filteredServices = [...allServices];
            } else {
                filteredServices = allServices.filter(service => 
                    service.toLowerCase().includes(searchTerm)
                );
            }
            
            currentPage = 0;
            displayedServices = [];
            
            const serviceList = document.getElementById('serviceList');
            const pagination = document.getElementById('servicePagination');
            
            if (filteredServices.length === 0) {
                 serviceList.innerHTML = '<div style="padding: 20px; text-align: center; color: #6c757d;"><i>未找到匹配的服务</i></div>';
                 pagination.style.display = 'none';
             } else {
                 loadMoreServices();
             }
         }
        function loadMethods(serviceName) {
            const currentFormat = document.getElementById('callFormat').value;
            const registryExprElement = document.getElementById('registryExpr');
            const registryElement = document.getElementById('registry');
            const registry = currentFormat === 'expression' ? 
                (registryExprElement ? registryExprElement.value.trim() : '') : 
                (registryElement ? registryElement.value.trim() : '');
            
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
                        // 使用JSON树形展示
                        renderJsonTree(parsed, result);
                    } catch (e) {
                        // 如果不是JSON字符串，直接显示
                        result.className = 'result';
                        result.textContent = data.data;
                        originalJsonData = data.data;
                    }
                } else if (typeof data.data === 'object' && data.data !== null) {
                    // 如果是对象或数组，使用JSON树形展示，并处理其中的大整数
                    const processedData = processLargeIntegers(data.data);
                    renderJsonTree(processedData, result);
                } else {
                    // 如果是基础数据类型（数字、布尔值、null等），直接显示
                    result.className = 'result';
                    result.textContent = String(data.data);
                    originalJsonData = data.data;
                }
            } else if (!data.success && data.error) {
                result.className = 'result';
                result.textContent = data.error;
                originalJsonData = data.error;
            } else {
                // 兼容旧格式或其他情况
                renderJsonTree(data, result);
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
                // 如果是空字符串，直接返回，不进行数字转换
                if (obj === '') {
                    return obj;
                }
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

        // JSON树形展示功能
        function createJsonTree(data, key = null, isRoot = true) {
            const container = document.createElement('div');
            container.className = 'json-tree-item';
            
            if (data === null) {
                container.innerHTML = (key ? '<span class="json-tree-key">"' + key + '"</span>: ' : '') + '<span class="json-tree-null">null</span>';
                return container;
            }
            
            if (typeof data === 'string') {
                container.innerHTML = (key ? '<span class="json-tree-key">"' + key + '"</span>: ' : '') + '<span class="json-tree-string">"' + escapeHtml(data) + '"</span>';
                return container;
            }
            
            if (typeof data === 'number') {
                container.innerHTML = (key ? '<span class="json-tree-key">"' + key + '"</span>: ' : '') + '<span class="json-tree-number">' + data + '</span>';
                return container;
            }
            
            if (typeof data === 'boolean') {
                container.innerHTML = (key ? '<span class="json-tree-key">"' + key + '"</span>: ' : '') + '<span class="json-tree-boolean">' + data + '</span>';
                return container;
            }
            
            const isArray = Array.isArray(data);
            const entries = isArray ? data.map((item, index) => [index, item]) : Object.entries(data);
            const isEmpty = entries.length === 0;
            
            const itemLine = document.createElement('div');
            itemLine.className = 'json-tree-item-line';
            
            if (isEmpty) {
                itemLine.innerHTML = (key ? '<span class="json-tree-key">"' + key + '"</span>: ' : '') + '<span class="json-tree-bracket">' + (isArray ? '[]' : '{}') + '</span>';
                container.appendChild(itemLine);
                return container;
            }
            
            const toggle = document.createElement('span');
            toggle.className = 'json-tree-toggle expanded';
            
            const keySpan = key ? '<span class="json-tree-key">"' + key + '"</span>: ' : '';
            const openBracket = '<span class="json-tree-bracket">' + (isArray ? '[' : '{') + '</span>';
            const summary = '<span class="json-tree-summary">' + entries.length + ' ' + (isArray ? 'items' : 'properties') + '</span>';
            
            itemLine.innerHTML = keySpan + openBracket + summary;
            itemLine.insertBefore(toggle, itemLine.firstChild);
            
            const childrenContainer = document.createElement('div');
            childrenContainer.className = 'json-tree-children';
            
            entries.forEach(([childKey, childValue], index) => {
                // 对于数组，不显示索引键，直接显示值
                const childElement = isArray ? 
                    createJsonTree(childValue, null, false) : 
                    createJsonTree(childValue, childKey, false);
                if (index < entries.length - 1) {
                    const comma = document.createElement('span');
                    comma.innerHTML = ',';
                    comma.style.color = '#666';
                    childElement.appendChild(comma);
                }
                childrenContainer.appendChild(childElement);
            });
            
            const closeLine = document.createElement('div');
            closeLine.className = 'json-tree-item-line';
            closeLine.innerHTML = '<span class="json-tree-bracket">' + (isArray ? ']' : '}') + '</span>';
            childrenContainer.appendChild(closeLine);
            
            toggle.addEventListener('click', function() {
                if (toggle.classList.contains('expanded')) {
                    toggle.classList.remove('expanded');
                    toggle.classList.add('collapsed');
                    childrenContainer.classList.add('collapsed');
                    itemLine.innerHTML = keySpan + openBracket + '<span class="json-tree-bracket">' + (isArray ? '...]' : '...}') + '</span>' + summary;
                    itemLine.insertBefore(toggle, itemLine.firstChild);
                } else {
                    toggle.classList.remove('collapsed');
                    toggle.classList.add('expanded');
                    childrenContainer.classList.remove('collapsed');
                    itemLine.innerHTML = keySpan + openBracket + summary;
                    itemLine.insertBefore(toggle, itemLine.firstChild);
                }
            });
            
            container.appendChild(itemLine);
            container.appendChild(childrenContainer);
            
            return container;
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function renderJsonTree(data, container) {
            container.innerHTML = '';
            container.className = 'result json-tree';
            
            // 存储原始JSON数据用于复制功能
            originalJsonData = data;
            
            try {
                const tree = createJsonTree(data);
                container.appendChild(tree);
            } catch (error) {
                container.className = 'result';
                container.textContent = 'JSON树形展示错误: ' + error.message + '\n\n原始数据:\n' + JSON.stringify(data, null, 2);
                // 错误情况下也要存储数据
                originalJsonData = data;
            }
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
                        paramDisplay = '<div style="font-size: 0.75em; margin-top: 2px; color: #9aa0a6; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + paramText.replace(/"/g, '&quot;') + '">' +
                            paramText.substring(0, 15) + '...' +
                        '</div>';
                    } else if (paramText) {
                        paramDisplay = '<div style="font-size: 0.75em; margin-top: 2px; color: #9aa0a6; max-width: 100%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">' + paramText + '</div>';
                    } else {
                        paramDisplay = '<div style="font-size: 0.75em; margin-top: 2px; color: #9aa0a6;">无参数</div>';
                    }
                } else {
                    paramDisplay = '<div style="font-size: 0.75em; margin-top: 2px; color: #9aa0a6;">无参数</div>';
                }
                
                historyItem.innerHTML = 
                    '<div class="service-name" style="max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + fullServiceName + '">' + fullServiceName + '</div>' +
                    '<div style="font-size: 0.8em; margin-top: 3px; color: #5f6368; max-width: 100%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">' +
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
            const serviceNameEl = document.getElementById('serviceName');
            if (serviceNameEl) serviceNameEl.value = item.serviceName || '';
            
            const methodNameEl = document.getElementById('methodName');
            if (methodNameEl) methodNameEl.value = item.methodName || '';
            
            // 填充命名空间字段
            const callFormatEl = document.getElementById('callFormat');
            if (callFormatEl) {
                const currentFormat = callFormatEl.value;
                const namespaceInput = currentFormat === 'expression' ? 
                    document.getElementById('namespaceExpr') : 
                    document.getElementById('namespace');
                if (namespaceInput) {
                    namespaceInput.value = item.namespace || 'public';
                }
            }
            
            // 处理参数：parameters现在是数组格式
            const parametersEl = document.getElementById('parameters');
            if (parametersEl) {
                if (item.parameters) {
                    if (Array.isArray(item.parameters)) {
                        // 直接处理数组格式的参数，处理其中的大整数
                        const processedParams = processLargeIntegers(item.parameters);
                        parametersEl.value = JSON.stringify(processedParams);
                    } else {
                        // 兼容旧的字符串格式
                        try {
                            const parsed = JSON.parse(item.parameters);
                            if (Array.isArray(parsed)) {
                                // 处理其中的大整数
                                const processedParams = processLargeIntegers(parsed);
                                parametersEl.value = JSON.stringify(processedParams);
                            } else {
                                parametersEl.value = item.parameters;
                            }
                        } catch (e) {
                            parametersEl.value = item.parameters;
                        }
                    }
                } else {
                    parametersEl.value = '';
                }
            }
            
            // 处理参数类型
            const typesEl = document.getElementById('types');
            if (typesEl) {
                if (item.types) {
                    if (Array.isArray(item.types)) {
                        typesEl.value = item.types.join(', ');
                    } else {
                        try {
                            const parsed = JSON.parse(item.types);
                            if (Array.isArray(parsed)) {
                                typesEl.value = parsed.join(', ');
                            } else {
                                typesEl.value = item.types;
                            }
                        } catch (e) {
                            typesEl.value = item.types;
                        }
                    }
                } else {
                    typesEl.value = '';
                }
            }
            
            // 填充注册中心地址 - 解析URL格式
            if (item.registry) {
                const registryUrl = item.registry;
                let registryType = 'zookeeper';
                let registryAddress = registryUrl;
                
                // 解析注册中心URL格式 (如: zookeeper://127.0.0.1:2181)
                if (registryUrl.includes('://')) {
                    const parts = registryUrl.split('://');
                    if (parts.length === 2) {
                        registryType = parts[0];
                        registryAddress = parts[1];
                    }
                }
                
                // 设置注册中心类型下拉框
                const registryTypeEl = document.getElementById('registryType');
                if (registryTypeEl) {
                    registryTypeEl.value = registryType;
                    // 触发类型变化事件以更新相关UI
                    onRegistryTypeChange();
                }
                
                // 如果是ZooKeeper类型，尝试匹配环境
                if (registryType === 'zookeeper') {
                    const zkEnvironmentEl = document.getElementById('zkEnvironment');
                    if (zkEnvironmentEl) {
                        // 根据地址匹配环境
                        let matchedEnv = '';
                        for (const [envKey, envConfig] of Object.entries(ZOOKEEPER_ENVIRONMENTS)) {
                            if (envConfig.address === registryAddress) {
                                matchedEnv = envKey;
                                break;
                            }
                        }
                        
                        if (matchedEnv) {
                            // 找到匹配的环境，设置下拉框
                            zkEnvironmentEl.value = matchedEnv;
                            // 触发环境变化事件以更新地址和命名空间
                            onZkEnvironmentChange();
                        } else {
                            // 没有找到匹配的环境，清空选择并手动设置地址
                            zkEnvironmentEl.value = '';
                            const registryAddressEl = document.getElementById('registryAddress');
                            if (registryAddressEl) {
                                registryAddressEl.value = registryAddress;
                            }
                        }
                    }
                } else {
                    // 非ZooKeeper类型，直接设置地址
                    const registryAddressEl = document.getElementById('registryAddress');
                    if (registryAddressEl) {
                        registryAddressEl.value = registryAddress;
                    }
                }
            }
            
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
                        // 使用JSON树形展示
                        renderJsonTree(parsed, resultElement);
                        resultElement.classList.add(item.success ? 'success' : 'error');
                    } catch (e) {
                        // 如果不是JSON格式，直接显示原内容
                        resultElement.className = 'result ' + (item.success ? 'success' : 'error');
                        resultElement.textContent = item.result;
                    }
                    
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
            const callFormatEl3 = document.getElementById('callFormat');
            if (callFormatEl3) {
                callFormatEl3.value = 'traditional';
                toggleCallFormat();
            }
            
            // 重新设置注册中心地址（因为toggleCallFormat可能会重置它）
            if (item.registry) {
                const registryUrl = item.registry;
                let registryType = 'zookeeper';
                let registryAddress = registryUrl;
                
                // 解析注册中心URL格式 (如: zookeeper://127.0.0.1:2181)
                if (registryUrl.includes('://')) {
                    const parts = registryUrl.split('://');
                    if (parts.length === 2) {
                        registryType = parts[0];
                        registryAddress = parts[1];
                    }
                }
                
                // 设置注册中心类型下拉框
                const registryTypeEl = document.getElementById('registryType');
                if (registryTypeEl) {
                    registryTypeEl.value = registryType;
                    // 触发类型变化事件以更新相关UI
                    onRegistryTypeChange();
                }
                
                // 如果是ZooKeeper类型，重新匹配环境（因为toggleCallFormat可能重置了选择）
                if (registryType === 'zookeeper') {
                    const zkEnvironmentEl = document.getElementById('zkEnvironment');
                    if (zkEnvironmentEl) {
                        // 根据地址匹配环境
                        let matchedEnv = '';
                        for (const [envKey, envConfig] of Object.entries(ZOOKEEPER_ENVIRONMENTS)) {
                            if (envConfig.address === registryAddress) {
                                matchedEnv = envKey;
                                break;
                            }
                        }
                        
                        if (matchedEnv) {
                            // 找到匹配的环境，设置下拉框
                            zkEnvironmentEl.value = matchedEnv;
                            // 触发环境变化事件以更新地址和命名空间
                            onZkEnvironmentChange();
                        } else {
                            // 没有找到匹配的环境，清空选择并手动设置地址
                            zkEnvironmentEl.value = '';
                            const registryAddressEl = document.getElementById('registryAddress');
                            if (registryAddressEl) {
                                registryAddressEl.value = registryAddress;
                            }
                        }
                    }
                } else {
                    // 非ZooKeeper类型，直接设置地址
                    const registryAddressEl = document.getElementById('registryAddress');
                    if (registryAddressEl) {
                        registryAddressEl.value = registryAddress;
                    }
                }
                
                // 设置命名空间（如果有的话）
                if (item.namespace) {
                    const namespaceEl = document.getElementById('namespace');
                    if (namespaceEl) {
                        namespaceEl.value = item.namespace;
                    }
                }
            }
        }
        
        function copyResult() {
            // 优先使用存储的原始JSON数据
            let textToCopy = '';
            
            if (originalJsonData !== null) {
                // 使用存储的原始JSON数据，格式化输出
                textToCopy = JSON.stringify(originalJsonData, null, 2);
            } else {
                // 如果没有存储的数据，回退到使用元素的文本内容
                const resultElement = document.getElementById('result');
                if (!resultElement) {
                    alert('暂无结果数据可复制');
                    return;
                }
                
                // 如果是树形展示，提示无法复制
                if (resultElement.classList.contains('json-tree')) {
                    alert('暂无结果数据可复制');
                    return;
                } else {
                    textToCopy = resultElement.textContent.trim();
                }
            }
            
            if (!textToCopy) {
                alert('暂无结果数据可复制');
                return;
            }
            
            // 创建临时文本区域用于复制
            const textarea = document.createElement('textarea');
            textarea.value = textToCopy;
            document.body.appendChild(textarea);
            textarea.select();
            
            try {
                document.execCommand('copy');
                alert('结果已复制到剪贴板');
            } catch (err) {
                // 如果复制失败，提供下载选项
                const blob = new Blob([textToCopy], { type: 'application/json' });
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

        // 压缩/美化JSON显示切换
        function toggleJsonFormat() {
            if (!originalJsonData) {
                alert('暂无JSON数据');
                return;
            }
            
            const resultElement = document.getElementById('result');
            isJsonCompressed = !isJsonCompressed;
            
            if (isJsonCompressed) {
                // 压缩显示
                resultElement.className = 'result';
                resultElement.textContent = JSON.stringify(originalJsonData);
            } else {
                // 美化显示（树形展示）
                renderJsonTree(originalJsonData, resultElement);
            }
            
            // 更新按钮图标
            const btn = document.querySelector('.compress');
            btn.innerHTML = isJsonCompressed ? '📄' : '🗜️';
            btn.title = isJsonCompressed ? '美化JSON' : '压缩JSON';
        }

        // 显示/隐藏行号
        function toggleLineNumbers() {
            const resultElement = document.getElementById('result');
            showLineNumbers = !showLineNumbers;
            
            if (showLineNumbers) {
                resultElement.classList.add('show-line-numbers');
                // 如果是JSON树形展示，转换为格式化的纯文本显示
                if (resultElement.classList.contains('json-tree') && originalJsonData) {
                    const formattedJson = JSON.stringify(originalJsonData, null, 2);
                    resultElement.className = 'result show-line-numbers';
                    addLineNumbers(resultElement, formattedJson);
                } else {
                    // 对于纯文本内容，直接添加行号
                    addLineNumbers(resultElement);
                }
            } else {
                resultElement.classList.remove('show-line-numbers');
                // 恢复原始显示模式
                if (originalJsonData && typeof originalJsonData === 'object') {
                    // 恢复JSON树形展示
                    renderJsonTree(originalJsonData, resultElement);
                } else {
                    // 移除行号，恢复原始内容
                    removeLineNumbers(resultElement);
                }
            }
            
            // 更新按钮图标
            const btn = document.querySelector('.line-numbers');
            btn.innerHTML = showLineNumbers ? '🔢' : '🔢';
            btn.title = showLineNumbers ? '隐藏行号' : '显示行号';
        }

        // 添加行号到结果内容
        function addLineNumbers(element, content = null) {
            const textContent = content || element.textContent || element.innerText;
            const lines = textContent.split('\n');
            
            // 清空元素并重新构建带行号的内容
            element.innerHTML = '';
            lines.forEach((line, index) => {
                const lineDiv = document.createElement('div');
                lineDiv.className = 'line';
                lineDiv.textContent = line || ' '; // 空行显示空格
                element.appendChild(lineDiv);
            });
        }

        // 移除行号，恢复原始内容
        function removeLineNumbers(element) {
            const lines = element.querySelectorAll('.line');
            if (lines.length > 0) {
                const content = Array.from(lines).map(line => line.textContent).join('\n');
                element.textContent = content;
            }
        }

        // 全部展开/收缩
        function toggleExpandAll() {
            const resultElement = document.getElementById('result');
            if (!resultElement.classList.contains('json-tree')) {
                alert('当前不是树形展示模式');
                return;
            }
            
            isAllExpanded = !isAllExpanded;
            const toggles = resultElement.querySelectorAll('.json-tree-toggle');
            const children = resultElement.querySelectorAll('.json-tree-children');
            
            toggles.forEach((toggle, index) => {
                if (isAllExpanded) {
                    toggle.classList.remove('collapsed');
                    toggle.classList.add('expanded');
                    if (children[index]) {
                        children[index].classList.remove('collapsed');
                    }
                } else {
                    toggle.classList.remove('expanded');
                    toggle.classList.add('collapsed');
                    if (children[index]) {
                        children[index].classList.add('collapsed');
                    }
                }
            });
            
            // 更新按钮图标
            const btn = document.querySelector('.expand-all');
            btn.innerHTML = isAllExpanded ? '📁' : '📂';
            btn.title = isAllExpanded ? '全部收缩' : '全部展开';
        }

        // 保存结果
        function saveResult() {
            if (!originalJsonData) {
                alert('暂无结果数据可保存');
                return;
            }
            
            const jsonString = JSON.stringify(originalJsonData, null, 2);
            const blob = new Blob([jsonString], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'dubbo-invoke-result-' + new Date().toISOString().slice(0,19).replace(/:/g, '-') + '.json';
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
        }

        // 清空结果
        function clearResult() {
            const resultElement = document.getElementById('result');
            resultElement.innerHTML = '';
            resultElement.className = 'result';
            resultElement.style.display = 'none';
            originalJsonData = null;
            
            // 重置状态
            isJsonCompressed = false;
            showLineNumbers = false;
            isAllExpanded = true;
            
            // 重置按钮状态
            const compressBtn = document.querySelector('.compress');
            const lineNumbersBtn = document.querySelector('.line-numbers');
            const expandBtn = document.querySelector('.expand-all');
            
            if (compressBtn) {
                compressBtn.innerHTML = '🗜️';
                compressBtn.title = '压缩JSON';
            }
            if (lineNumbersBtn) {
                lineNumbersBtn.innerHTML = '🔢';
                lineNumbersBtn.title = '显示行号';
            }
            if (expandBtn) {
                expandBtn.innerHTML = '📂';
                expandBtn.title = '全部展开/收缩';
            }
            
            // 隐藏结果面板标题的状态指示器
            const resultPanelTitle = document.querySelector('.result-panel h2');
            if (resultPanelTitle) {
                const titleSpan = resultPanelTitle.querySelector('span');
                if (titleSpan) {
                    titleSpan.innerHTML = '调用结果';
                }
            }
        }
        
        function onRegistryTypeChange() {
            const registryType = document.getElementById('registryType').value;
            const namespaceGroup = document.getElementById('namespaceGroup');
            const zkEnvironmentContainer = document.getElementById('zkEnvironmentContainer');
            const addressInput = document.getElementById('registryAddress');
            
            // 只有nacos时才显示命名空间
            if (registryType === 'nacos') {
                namespaceGroup.style.display = 'block';
            } else {
                namespaceGroup.style.display = 'none';
            }
            
            // 只有zookeeper时才显示环境选择
            if (registryType === 'zookeeper') {
                zkEnvironmentContainer.style.display = 'block';
                addressInput.style.display = 'none';
                addressInput.readOnly = true;
                addressInput.placeholder = '请先选择环境';
                addressInput.value = '';
                // 重置环境选择
                document.getElementById('zkEnvironment').value = '';
            } else {
                zkEnvironmentContainer.style.display = 'none';
                addressInput.style.display = 'block';
                addressInput.readOnly = false;
                
                // 根据注册中心类型设置默认端口
                if (registryType === 'nacos') {
                    addressInput.placeholder = '127.0.0.1:8848';
                    if (!addressInput.value || addressInput.value === '127.0.0.1:2181' || addressInput.value === '127.0.0.1:8080') {
                        addressInput.value = '127.0.0.1:8848';
                    }
                } else if (registryType === 'dubbo') {
                    addressInput.placeholder = '127.0.0.1:8080';
                    if (!addressInput.value || addressInput.value === '127.0.0.1:2181' || addressInput.value === '127.0.0.1:8848') {
                        addressInput.value = '127.0.0.1:8080';
                    }
                }
            }
        }
        
        function onZkEnvironmentChange() {
            const selectedEnv = document.getElementById('zkEnvironment').value;
            const addressInput = document.getElementById('registryAddress');
            const namespaceInput = document.getElementById('namespace');
            
            if (selectedEnv && ZOOKEEPER_ENVIRONMENTS[selectedEnv]) {
                const envConfig = ZOOKEEPER_ENVIRONMENTS[selectedEnv];
                addressInput.value = envConfig.address;
                // 自动设置namespace为对应环境的servicePath
                if (namespaceInput) {
                    namespaceInput.value = envConfig.servicePath;
                }
                console.log('选择环境:', envConfig.name, '地址:', envConfig.address, '服务路径:', envConfig.servicePath);
            } else {
                addressInput.value = '';
                if (namespaceInput) {
                    namespaceInput.value = 'public';
                }
            }
        }
        
        function onRegistryTypeChangeExpr() {
            const registryTypeExpr = document.getElementById('registryTypeExpr').value;
            const namespaceGroupExpr = document.getElementById('namespaceGroupExpr');
            const zkEnvironmentContainerExpr = document.getElementById('zkEnvironmentContainerExpr');
            const addressInputExpr = document.getElementById('registryAddressExpr');
            
            // 只有nacos时才显示命名空间
            if (registryTypeExpr === 'nacos') {
                namespaceGroupExpr.style.display = 'block';
            } else {
                namespaceGroupExpr.style.display = 'none';
            }
            
            // 只有zookeeper时才显示环境选择
            if (registryTypeExpr === 'zookeeper') {
                zkEnvironmentContainerExpr.style.display = 'block';
                addressInputExpr.style.display = 'none';
                addressInputExpr.readOnly = true;
                addressInputExpr.placeholder = '请先选择环境';
                addressInputExpr.value = '';
                // 重置环境选择
                document.getElementById('zkEnvironmentExpr').value = '';
            } else {
                zkEnvironmentContainerExpr.style.display = 'none';
                addressInputExpr.style.display = 'block';
                addressInputExpr.readOnly = false;
                
                // 根据注册中心类型设置默认端口
                if (registryTypeExpr === 'nacos') {
                    addressInputExpr.placeholder = '127.0.0.1:8848';
                    if (!addressInputExpr.value || addressInputExpr.value === '127.0.0.1:2181' || addressInputExpr.value === '127.0.0.1:8080') {
                        addressInputExpr.value = '127.0.0.1:8848';
                    }
                } else if (registryTypeExpr === 'dubbo') {
                    addressInputExpr.placeholder = '127.0.0.1:8080';
                    if (!addressInputExpr.value || addressInputExpr.value === '127.0.0.1:2181' || addressInputExpr.value === '127.0.0.1:8848') {
                        addressInputExpr.value = '127.0.0.1:8080';
                    }
                }
            }
        }
        
        function onZkEnvironmentChangeExpr() {
            const selectedEnv = document.getElementById('zkEnvironmentExpr').value;
            const addressInputExpr = document.getElementById('registryAddressExpr');
            const namespaceInputExpr = document.getElementById('namespaceExpr');
            
            if (selectedEnv && ZOOKEEPER_ENVIRONMENTS[selectedEnv]) {
                const envConfig = ZOOKEEPER_ENVIRONMENTS[selectedEnv];
                addressInputExpr.value = envConfig.address;
                // 自动设置namespace为对应环境的servicePath
                if (namespaceInputExpr) {
                    namespaceInputExpr.value = envConfig.servicePath;
                }
                console.log('选择环境:', envConfig.name, '地址:', envConfig.address, '服务路径:', envConfig.servicePath);
            } else {
                addressInputExpr.value = '';
                if (namespaceInputExpr) {
                    namespaceInputExpr.value = 'public';
                }
            }
        }
        
        function testConnection() {
            const format = document.getElementById('callFormat').value;
            let registryType, registryAddress;
            
            if (format === 'expression') {
                registryType = document.getElementById('registryTypeExpr').value;
                registryAddress = document.getElementById('registryAddressExpr').value.trim();
            } else {
                registryType = document.getElementById('registryType').value;
                registryAddress = document.getElementById('registryAddress').value.trim();
            }
            
            if (!registryAddress) {
                showConnectionResult('请先输入注册中心地址', false);
                return;
            }

            // 构建完整的注册中心URL
            let registry;
            if (registryType === 'zookeeper') {
                registry = 'zookeeper://' + registryAddress;
            } else if (registryType === 'nacos') {
                registry = 'nacos://' + registryAddress;
            } else if (registryType === 'dubbo') {
                registry = 'dubbo://' + registryAddress;
            } else {
                registry = registryAddress;
            }
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
            
            let namespace;
            if (format === 'expression') {
                const namespaceExprElement = document.getElementById('namespaceExpr');
                namespace = namespaceExprElement ? namespaceExprElement.value.trim() : 'public';
            } else {
                const namespaceElement = document.getElementById('namespace');
                namespace = namespaceElement ? namespaceElement.value.trim() : 'public';
            }
            
            fetch('/api/list', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    registry: registry,
                    namespace: namespace,
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
        
        window.onload = function() { 
            loadHistory(); 
            // 默认切换到表达式格式
            toggleCallFormat();
            // 初始化注册中心类型变化
            onRegistryTypeChange();
            onRegistryTypeChangeExpr();
        };
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

// handleZkEnvironments 处理获取Zookeeper环境配置
func (ws *WebServer) handleZkEnvironments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		ws.writeError(w, "只支持GET方法")
		return
	}

	environments := getZookeeperEnvironments()

	response := map[string]interface{}{
		"success":      true,
		"environments": environments,
	}

	json.NewEncoder(w).Encode(response)
}

// handleFavicon 处理favicon请求
func (ws *WebServer) handleFavicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400") // 缓存1天

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// 从磁盘读取图标文件（要求运行目录下有 icons/dubbo.png）
	filePath := "icons/dubbo.png"
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Write(content)
}
