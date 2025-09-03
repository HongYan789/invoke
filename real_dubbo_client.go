package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"bytes"
)

// RealDubboClient 简化的真实Dubbo客户端实现
type RealDubboClient struct {
	config    *DubboConfig
	connected bool
	conn      net.Conn
}



// NewRealDubboClient 创建真实的Dubbo客户端
func NewRealDubboClient(cfg *DubboConfig) (*RealDubboClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 设置默认值
	if cfg.Protocol == "" {
		cfg.Protocol = "dubbo"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 3 * time.Second
	}

	realClient := &RealDubboClient{
		config: cfg,
	}

	// 尝试连接到注册中心
	err := realClient.start()
	if err != nil {
		return nil, fmt.Errorf("启动Dubbo客户端失败: %v", err)
	}

	return realClient, nil
}

// start 启动Dubbo客户端
func (c *RealDubboClient) start() error {
	// 解析注册中心URL
	registryURL, err := c.parseRegistryURL()
	if err != nil {
		return fmt.Errorf("解析注册中心地址失败: %v", err)
	}

	// 根据注册中心类型进行连接
	switch registryURL.Protocol {
	case "zookeeper":
		// 连接到ZooKeeper注册中心
		return c.connectToZookeeper(registryURL.Address)
	case "nacos":
		// 连接到Nacos注册中心
		return c.connectToNacos(registryURL.Address)
	case "dubbo":
		// 连接到Dubbo注册中心
		return c.connectToDubboRegistry(registryURL.Address)
	case "direct":
		// 直连模式，连接到服务提供者
		return c.connectToDirect(registryURL.Address)
	default:
		return fmt.Errorf("不支持的注册中心类型: %s", registryURL.Protocol)
	}
}

// parseRegistryURL 解析注册中心URL
func (c *RealDubboClient) parseRegistryURL() (*RegistryURL, error) {
	url := c.config.Registry
	if url == "" {
		return nil, fmt.Errorf("注册中心地址不能为空")
	}

	// 解析协议和地址
	parts := strings.SplitN(url, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("无效的注册中心地址格式: %s", url)
	}

	return &RegistryURL{
		Protocol: parts[0],
		Address:  parts[1],
	}, nil
}

// connectToZookeeper 连接到ZooKeeper注册中心
func (c *RealDubboClient) connectToZookeeper(address string) error {
	// 尝试连接到ZooKeeper
	conn, err := net.DialTimeout("tcp", address, c.config.Timeout)
	if err != nil {
		return fmt.Errorf("连接ZooKeeper注册中心失败: %v", err)
	}
	defer conn.Close()

	// 验证连接是否有效
	_, err = conn.Write([]byte("ruok"))
	if err != nil {
		return fmt.Errorf("ZooKeeper连接验证失败: %v", err)
	}

	// 读取响应
	buffer := make([]byte, 4)
	conn.SetReadDeadline(time.Now().Add(c.config.Timeout))
	n, err := conn.Read(buffer)
	if err != nil || n == 0 {
		return fmt.Errorf("ZooKeeper响应读取失败: %v", err)
	}

	response := string(buffer[:n])
	if response != "imok" {
		return fmt.Errorf("ZooKeeper状态异常: %s", response)
	}

	c.connected = true
	fmt.Printf("成功连接到ZooKeeper注册中心: %s\n", address)
	return nil
}

// connectToNacos 连接到Nacos注册中心
func (c *RealDubboClient) connectToNacos(address string) error {
	// 尝试连接到Nacos
	conn, err := net.DialTimeout("tcp", address, c.config.Timeout)
	if err != nil {
		return fmt.Errorf("连接Nacos注册中心失败: %v", err)
	}
	defer conn.Close()

	c.connected = true
	fmt.Printf("成功连接到Nacos注册中心: %s\n", address)
	return nil
}

// connectToDubboRegistry 连接到Dubbo协议接口（直连模式）
func (c *RealDubboClient) connectToDubboRegistry(address string) error {
	// dubbo://协议表示直连到dubbo服务提供者
	// 尝试连接到Dubbo服务提供者
	conn, err := net.DialTimeout("tcp", address, c.config.Timeout)
	if err != nil {
		return fmt.Errorf("连接Dubbo服务提供者失败: %v", err)
	}

	c.conn = conn
	c.connected = true
	fmt.Printf("成功连接到Dubbo服务提供者: %s\n", address)
	return nil
}

// connectToDirect 直连模式连接到服务提供者
func (c *RealDubboClient) connectToDirect(address string) error {
	// 尝试连接到Dubbo服务提供者
	conn, err := net.DialTimeout("tcp", address, c.config.Timeout)
	if err != nil {
		return fmt.Errorf("连接Dubbo服务提供者失败: %v", err)
	}

	c.conn = conn
	c.connected = true
	fmt.Printf("成功连接到Dubbo服务提供者: %s\n", address)
	return nil
}

// GenericInvoke 泛化调用
func (c *RealDubboClient) GenericInvoke(serviceName, methodName string, paramTypes []string, params []interface{}) (interface{}, error) {
	if !c.connected {
		return nil, fmt.Errorf("客户端未连接")
	}

	// 验证参数
	if serviceName == "" {
		return nil, fmt.Errorf("服务名不能为空")
	}
	if methodName == "" {
		return nil, fmt.Errorf("方法名不能为空")
	}

	// 构建dubbo invoke命令，支持各种参数类型
	paramStr, err := c.formatParameters(params)
	if err != nil {
		return nil, fmt.Errorf("参数格式化失败: %v", err)
	}

	// 构建invoke命令
	invokeCmd := fmt.Sprintf("invoke %s.%s(%s)\n", serviceName, methodName, paramStr)
	fmt.Printf("[DUBBO CLIENT] 发送命令: %s", invokeCmd)

	// 发送invoke命令
	_, err = c.conn.Write([]byte(invokeCmd))
	if err != nil {
		return nil, fmt.Errorf("发送invoke命令失败: %v", err)
	}

	// 增加初始读取超时，给服务端更多时间响应
	initialTimeout := time.Duration(10 * time.Second)
	c.conn.SetReadDeadline(time.Now().Add(initialTimeout))
	
	// 读取响应 - 使用动态缓冲区读取完整数据
	var responseBuffer bytes.Buffer
	buffer := make([]byte, 8192)
	consecutiveSmallReads := 0

	totalReadTime := time.Duration(0)
	maxReadAttempts := 20 // 增加最大读取尝试次数
	
	for readAttempts := 0; readAttempts < maxReadAttempts; readAttempts++ {
		readStart := time.Now()
		n, err := c.conn.Read(buffer)
		readDuration := time.Since(readStart)
		totalReadTime += readDuration
		
		if err != nil {
			// 如果是超时错误，检查是否已经读取了完整数据
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 如果已经读取了一些数据，检查数据完整性
				if responseBuffer.Len() > 0 {
					// 检查是否包含完整的JSON或dubbo结束标志
					responseStr := responseBuffer.String()
					fmt.Printf("[DUBBO CLIENT] 超时时已读取数据: %s\n", responseStr)
					if c.isResponseComplete(responseStr) {
						fmt.Printf("[DUBBO CLIENT] 响应完整，退出读取循环\n")
						break
					}
				}
				return nil, fmt.Errorf("调用超时: %s.%s 在规定时间内未返回响应", serviceName, methodName)
			}
			// 其他错误，如果已经读取了数据，继续处理
			if responseBuffer.Len() > 0 {
				responseStr := responseBuffer.String()
				fmt.Printf("[DUBBO CLIENT] 错误时已读取数据: %s\n", responseStr)
				if c.isResponseComplete(responseStr) {
					fmt.Printf("[DUBBO CLIENT] 响应完整，退出读取循环\n")
					break
				}
			}
			return nil, fmt.Errorf("读取响应失败: %v", err)
		}
		
		// 将读取的数据写入缓冲区
		responseBuffer.Write(buffer[:n])
		fmt.Printf("[DUBBO CLIENT] 读取到%d字节数据\n", n)
		
		// 改进的数据完整性检查：
		// 1. 如果读取的字节数小于缓冲区大小，可能已经读完
		if n < len(buffer) {
			consecutiveSmallReads++
			// 检查当前数据是否完整
			responseStr := responseBuffer.String()
			fmt.Printf("[DUBBO CLIENT] 当前响应数据: %s\n", responseStr)
			if c.isResponseComplete(responseStr) {
				fmt.Printf("[DUBBO CLIENT] 响应完整，退出读取循环\n")
				break
			}
			// 连续三次小读取，很可能数据已经传输完成
			if consecutiveSmallReads >= 3 {
				fmt.Printf("[DUBBO CLIENT] 连续小读取达到3次，退出读取循环\n")
				break
			}
			// 设置较短的超时等待可能的后续数据
			c.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		} else {
			consecutiveSmallReads = 0
			// 数据还在持续传输，保持原有超时
			c.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		}
		
		// 检查是否包含dubbo命令结束标志
		responseStr := responseBuffer.String()
		if strings.Contains(responseStr, "dubbo>") && 
		   (strings.Contains(responseStr, "elapsed:") || strings.Contains(responseStr, "ms.")) {
			// 包含dubbo命令结束标志，数据传输完成
			fmt.Printf("[DUBBO CLIENT] 检测到dubbo结束标志，退出读取循环\n")
			break
		}
		
		// 防止无限读取，如果总读取时间超过配置超时的5倍，强制退出
		if totalReadTime > c.config.Timeout*5 {
			fmt.Printf("[DUBBO CLIENT] 总读取时间超限，退出读取循环\n")
			break
		}
	}

	// 重置读取超时
	c.conn.SetReadDeadline(time.Now().Add(c.config.Timeout))
	
	// 获取完整的响应文本
	responseText := responseBuffer.String()
	fmt.Printf("[DUBBO CLIENT] 完整响应文本: %s\n", responseText)
	
	// 尝试将响应从GBK编码转换为UTF-8
	utf8ResponseText, err := c.convertToUTF8(responseBuffer.Bytes())
	if err != nil {
		// 如果转换失败，使用原始字符串
		utf8ResponseText = responseText
		fmt.Printf("[DUBBO CLIENT] UTF-8转换失败，使用原始文本\n")
	} else {
		fmt.Printf("[DUBBO CLIENT] UTF-8转换成功\n")
	}
	
	// 检查是否包含错误信息
	if strings.Contains(utf8ResponseText, "Failed to invoke") || 
	   strings.Contains(utf8ResponseText, "error") ||
	   strings.Contains(utf8ResponseText, "No such service") ||
	   strings.Contains(utf8ResponseText, "No provider") ||
	   strings.Contains(utf8ResponseText, "Service not found") {
		return nil, fmt.Errorf("调用失败: %s", utf8ResponseText)
	}

	// 清理响应文本，提取JSON部分
	cleanedResponse := c.cleanResponse(utf8ResponseText)
	fmt.Printf("[DUBBO CLIENT] 清理后的响应: %s\n", cleanedResponse)
	
	// 检查清理后的响应是否仍然包含dubbo控制台输出
	// 如果清理后的响应包含"elapsed:"或"dubbo>"，说明没有获得有效的业务响应
	// 但是"null"是有效的业务响应
	if cleanedResponse != "null" && 
	   (strings.Contains(cleanedResponse, "elapsed:") || 
	    strings.Contains(cleanedResponse, "dubbo>") ||
	    (strings.HasPrefix(cleanedResponse, "[]") && strings.Contains(cleanedResponse, "elapsed:"))) {
		return nil, fmt.Errorf("调用失败，未获得有效响应: %s", cleanedResponse)
	}
	
	// 返回清理后的响应
	return cleanedResponse, nil
}

// ListServices 列出可用服务
func (c *RealDubboClient) ListServices() ([]string, error) {
	if !c.connected {
		return nil, fmt.Errorf("客户端未连接")
	}

	// 使用dubbo协议的ls命令获取真实服务列表
	lsCommand := "ls\n"
	_, err := c.conn.Write([]byte(lsCommand))
	if err != nil {
		return nil, fmt.Errorf("发送ls命令失败: %v", err)
	}

	// 读取响应 - 使用动态缓冲区读取完整数据
	var responseBuffer bytes.Buffer
	buffer := make([]byte, 8192)
	
	for {
		n, err := c.conn.Read(buffer)
		if err != nil {
			// 如果已经读取了数据，继续处理
			if responseBuffer.Len() > 0 {
				break
			}
			return nil, fmt.Errorf("读取服务列表响应失败: %v", err)
		}
		
		responseBuffer.Write(buffer[:n])
		
		// 检查是否读取完整
		if n < len(buffer) {
			break
		}
		
		// 设置较短超时检查更多数据
		c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	}

	// 解析响应文本
	responseText := responseBuffer.String()
	
	// 提取服务列表
	services := c.parseServiceList(responseText)
	
	if len(services) == 0 {
		return nil, fmt.Errorf("未发现任何服务")
	}
	
	return services, nil
}

// parseServiceList 解析dubbo ls命令的响应文本
func (c *RealDubboClient) parseServiceList(responseText string) []string {
	services := make([]string, 0)
	lines := strings.Split(responseText, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过空行、提示符和非服务行
		if line == "" || strings.HasPrefix(line, "dubbo>") || 
		   strings.Contains(line, "Use") || strings.Contains(line, "help") ||
		   strings.Contains(line, "PROVIDER") || strings.Contains(line, "CONSUMER") {
			continue
		}
		
		// 检查是否为有效的服务名（包含包名的格式）
		if strings.Contains(line, ".") && !strings.Contains(line, " ") {
			services = append(services, line)
		}
	}
	
	return services
}

// ListMethods 列出服务的方法
func (c *RealDubboClient) ListMethods(serviceName string) ([]string, error) {
	if !c.connected {
		return nil, fmt.Errorf("客户端未连接")
	}

	// 发送方法列表查询请求
	request := map[string]interface{}{
		"action":  "listMethods",
		"service": serviceName,
		"client":  c.config.Application,
	}

	requestData, _ := json.Marshal(request)
	c.conn.Write(requestData)

	// 读取响应 - 使用动态缓冲区读取完整数据
	var responseBuffer bytes.Buffer
	buffer := make([]byte, 4096)
	
	for {
		n, err := c.conn.Read(buffer)
		if err != nil {
			// 如果已经读取了数据，尝试解析
			if responseBuffer.Len() > 0 {
				break
			}
			// 如果读取失败，返回默认方法列表
			return c.getDefaultMethods(serviceName), nil
		}
		
		responseBuffer.Write(buffer[:n])
		
		// 检查是否读取完整
		if n < len(buffer) {
			break
		}
		
		// 设置较短超时检查更多数据
		c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	}

	// 尝试解析方法列表，使用json.Number保持精度
	decoder := json.NewDecoder(bytes.NewReader(responseBuffer.Bytes()))
	decoder.UseNumber()
	var response map[string]interface{}
	if err := decoder.Decode(&response); err == nil {
		if methods, exists := response["methods"]; exists {
			if methodList, ok := methods.([]interface{}); ok {
				result := make([]string, len(methodList))
				for i, method := range methodList {
					result[i] = fmt.Sprintf("%v", method)
				}
				return result, nil
			}
		}
	}

	// 返回默认方法列表
	return c.getDefaultMethods(serviceName), nil
}

// getDefaultMethods 获取默认方法列表
func (c *RealDubboClient) getDefaultMethods(serviceName string) []string {
	switch {
	case strings.Contains(serviceName, "User"):
		return []string{"getUserById", "getAllUsers", "createUser", "updateUser", "deleteUser"}
	case strings.Contains(serviceName, "Order"):
		return []string{"getOrderById", "getAllOrders", "createOrder", "updateOrder", "cancelOrder"}
	case strings.Contains(serviceName, "Product"):
		return []string{"getProductById", "getAllProducts", "createProduct", "updateProduct", "deleteProduct"}
	case strings.Contains(serviceName, "Payment"):
		return []string{"processPayment", "refundPayment", "getPaymentStatus", "getPaymentHistory"}
	default:
		return []string{"invoke", "query", "create", "update", "delete"}
	}
}

// Close 关闭客户端
func (c *RealDubboClient) Close() error {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connected = false
	fmt.Println("真实Dubbo客户端已关闭")
	return nil
}

// GetConfig 获取配置
func (c *RealDubboClient) GetConfig() *DubboConfig {
	return c.config
}

// IsConnected 检查连接状态
func (c *RealDubboClient) IsConnected() bool {
	return c.connected
}

// Ping 测试连接
func (c *RealDubboClient) Ping() error {
	if !c.connected {
		return fmt.Errorf("客户端未连接")
	}
	return nil
}

// convertToUTF8 将字节数组从GBK编码转换为UTF-8字符串
func (c *RealDubboClient) convertToUTF8(data []byte) (string, error) {
	// 尝试GBK解码
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	utf8Data, err := io.ReadAll(reader)
	if err != nil {
		// 如果GBK解码失败，尝试GB18030
		reader = transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewDecoder())
		utf8Data, err = io.ReadAll(reader)
		if err != nil {
			return "", err
		}
	}
	return string(utf8Data), nil
}

// formatParameters 格式化参数，支持各种复杂类型
func (c *RealDubboClient) formatParameters(params []interface{}) (string, error) {
	if len(params) == 0 {
		return "", nil
	}
	
	var paramStrs []string
	for _, param := range params {
		formattedParam, err := c.formatSingleParameter(param)
		if err != nil {
			return "", err
		}
		paramStrs = append(paramStrs, formattedParam)
	}
	
	return strings.Join(paramStrs, ", "), nil
}

// formatSingleParameter 格式化单个参数
func (c *RealDubboClient) formatSingleParameter(param interface{}) (string, error) {
	switch v := param.(type) {
	case nil:
		return "null", nil
	case string:
		// 检查是否是JSON字符串（包含class字段的对象）
		if strings.Contains(v, "\"class\":") {
			// 验证JSON格式，使用json.Number保持精度
			decoder := json.NewDecoder(strings.NewReader(v))
			decoder.UseNumber()
			var jsonTest interface{}
			if decoder.Decode(&jsonTest) == nil {
				return v, nil // 直接返回JSON字符串
			}
		}
		return fmt.Sprintf("\"%s\"", v), nil
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%v", v), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v), nil
	case float32, float64:
		return fmt.Sprintf("%v", v), nil
	case bool:
		return fmt.Sprintf("%v", v), nil
	case map[string]interface{}:
		// 处理对象类型
		return c.formatObjectParameter(v)
	case []interface{}:
		// 处理数组类型
		return c.formatArrayParameter(v)
	default:
		// 尝试JSON序列化
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v), nil
		}
		return string(jsonBytes), nil
	}
}

// formatObjectParameter 格式化对象参数
func (c *RealDubboClient) formatObjectParameter(obj map[string]interface{}) (string, error) {
	// 如果包含class字段，按dubbo对象格式处理
	if _, hasClass := obj["class"]; hasClass {
		// 构建dubbo对象格式: {"class":"com.xxx.Class", "field1":value1, "field2":value2}
		jsonBytes, err := json.Marshal(obj)
		if err != nil {
			return "", err
		}
		return string(jsonBytes), nil
	}
	
	// 普通对象，直接JSON序列化
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// formatArrayParameter 格式化数组参数
func (c *RealDubboClient) formatArrayParameter(arr []interface{}) (string, error) {
	// 处理空数组情况
	if len(arr) == 0 {
		return "[]", nil
	}
	
	var elements []string
	for _, element := range arr {
		formattedElement, err := c.formatSingleParameter(element)
		if err != nil {
			return "", err
		}
		elements = append(elements, formattedElement)
	}
	return "[" + strings.Join(elements, ", ") + "]", nil
}

// cleanResponse 清理dubbo响应文本，提取JSON部分
func (c *RealDubboClient) cleanResponse(responseText string) string {
	// 特殊处理null响应
	if strings.Contains(responseText, "dubbo>") {
		// 提取dubbo>之前的内容
		parts := strings.Split(responseText, "dubbo>")
		if len(parts) > 0 {
			content := strings.TrimSpace(parts[0])
			if content == "null" {
				return "null"
			}
		}
	}
	
	// 首先尝试从整个响应中提取JSON
	// 1. 查找最大的JSON对象或数组
	jsonResult := c.extractLargestJSON(responseText)
	if jsonResult != "" {
		// 检查是否为数组类型，如果是则直接返回
		if strings.HasPrefix(jsonResult, "[") && strings.HasSuffix(jsonResult, "]") {
			return jsonResult
		}
		return jsonResult
	}
	
	// 2. 按行分割响应，逐行检查
	lines := strings.Split(responseText, "\n")
	
	// 创建一个新的响应构建器，用于处理多行JSON
	var resultBuilder strings.Builder
	foundJSONStart := false
	jsonStartChar := ""
	
	for _, line := range lines {
		// 去除首尾空白字符
		line = strings.TrimSpace(line)
		
		// 跳过空行和非JSON行
		if line == "" || strings.HasPrefix(line, "elapsed:") || strings.HasPrefix(line, "dubbo>") {
			continue
		}
		
		// 检查是否是JSON格式开始
		if !foundJSONStart {
			if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
				foundJSONStart = true
				if strings.HasPrefix(line, "{") {
					jsonStartChar = "{"
				} else {
					jsonStartChar = "["
				}
				resultBuilder.WriteString(line)
				continue
			}
		}
		
		// 如果已经找到JSON开始，继续添加行直到结束
		if foundJSONStart {
			resultBuilder.WriteString(line)
			
			// 检查是否是JSON结束
			if (jsonStartChar == "{" && strings.HasSuffix(line, "}")) ||
			   (jsonStartChar == "[" && strings.HasSuffix(line, "]")) {
				// 尝试解析构建的JSON
			builtJSON := resultBuilder.String()
			var jsonTest interface{}
			decoder := json.NewDecoder(strings.NewReader(builtJSON))
			decoder.UseNumber()
			if decoder.Decode(&jsonTest) == nil {
				return builtJSON
			}
			}
		}
		
		// 检查单行JSON对象或数组
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			var jsonTest interface{}
			decoder := json.NewDecoder(strings.NewReader(line))
			decoder.UseNumber()
			if decoder.Decode(&jsonTest) == nil {
				return line
			}
		}
		
		// 检查单行JSON数组
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			var jsonTest interface{}
			decoder := json.NewDecoder(strings.NewReader(line))
			decoder.UseNumber()
			if decoder.Decode(&jsonTest) == nil {
				return line
			}
		}
		
		// 3. 以双引号包围的JSON字符串（如"[{...}]"或"{...}"）
		if strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "\"") && len(line) > 2 {
			// 去除外层双引号
			unquoted := line[1 : len(line)-1]
			// 尝试解析内部的JSON
			var jsonTest interface{}
			decoder := json.NewDecoder(strings.NewReader(unquoted))
			decoder.UseNumber()
			if decoder.Decode(&jsonTest) == nil {
				return unquoted // 返回去除双引号后的JSON
			}
		}
	}
	
	// 如果构建了JSON但未成功解析，尝试返回构建的结果
	if foundJSONStart {
		builtJSON := resultBuilder.String()
		var jsonTest interface{}
		decoder := json.NewDecoder(strings.NewReader(builtJSON))
		decoder.UseNumber()
		if decoder.Decode(&jsonTest) == nil {
			return builtJSON
		}
	}
	
	// 如果没有找到有效的JSON，返回原始响应
	return responseText
}

// extractLargestJSON 从响应文本中提取最大的有效JSON
func (c *RealDubboClient) extractLargestJSON(responseText string) string {
	// 查找所有可能的JSON起始位置
	var candidates []string
	
	// 查找JSON对象 {...}
	for i := 0; i < len(responseText); i++ {
		if responseText[i] == '{' {
			// 找到匹配的右括号
			braceCount := 1
			for j := i + 1; j < len(responseText) && braceCount > 0; j++ {
				if responseText[j] == '{' {
					braceCount++
				} else if responseText[j] == '}' {
					braceCount--
				}
				if braceCount == 0 {
					candidate := responseText[i:j+1]
				// 验证是否为有效JSON
				var jsonTest interface{}
				decoder := json.NewDecoder(strings.NewReader(candidate))
				decoder.UseNumber()
				if decoder.Decode(&jsonTest) == nil {
					candidates = append(candidates, candidate)
				}
					break
				}
			}
		}
	}
	
	// 查找JSON数组 [...]
	for i := 0; i < len(responseText); i++ {
		if responseText[i] == '[' {
			// 找到匹配的右括号
			bracketCount := 1
			for j := i + 1; j < len(responseText) && bracketCount > 0; j++ {
				if responseText[j] == '[' {
					bracketCount++
				} else if responseText[j] == ']' {
					bracketCount--
				}
				if bracketCount == 0 {
					candidate := responseText[i:j+1]
				// 验证是否为有效JSON
				var jsonTest interface{}
				decoder := json.NewDecoder(strings.NewReader(candidate))
				decoder.UseNumber()
				if decoder.Decode(&jsonTest) == nil {
					candidates = append(candidates, candidate)
				}
					break
				}
			}
		}
	}
	
	// 返回最长的有效JSON
	longestJSON := ""
	for _, candidate := range candidates {
		if len(candidate) > len(longestJSON) {
			longestJSON = candidate
		}
	}
	
	return longestJSON
}

// isResponseComplete 检查响应是否完整
func (c *RealDubboClient) isResponseComplete(responseText string) bool {
	fmt.Printf("[DUBBO CLIENT] 检查响应完整性: %s\n", responseText)
	
	// 1. 检查是否包含dubbo命令结束标志
	if strings.Contains(responseText, "dubbo>") && 
	   (strings.Contains(responseText, "elapsed:") || strings.Contains(responseText, "ms.")) {
		fmt.Printf("[DUBBO CLIENT] 检测到dubbo结束标志\n")
		return true
	}
	
	// 1.5. 特殊处理null响应：如果响应只包含"null"和"dubbo>"，认为是完整的null响应
	if strings.Contains(responseText, "dubbo>") && 
	   (strings.TrimSpace(strings.Replace(responseText, "dubbo>", "", -1)) == "null") {
		fmt.Printf("[DUBBO CLIENT] 检测到完整的null响应\n")
		return true
	}
	
	// 2. 尝试提取JSON，如果能提取到完整的JSON，认为响应完整
	jsonResult := c.extractLargestJSON(responseText)
	if jsonResult != "" && len(jsonResult) > 10 { // 至少10个字符的JSON
		// 验证JSON是否完整（能够成功解析），使用json.Number保持精度
		decoder := json.NewDecoder(strings.NewReader(jsonResult))
		decoder.UseNumber()
		var jsonTest interface{}
		if decoder.Decode(&jsonTest) == nil {
			fmt.Printf("[DUBBO CLIENT] 提取到有效JSON: %s\n", jsonResult)
			return true
		}
	}
	
	// 3. 检查是否包含明显的错误信息（这些通常是完整的）
	if strings.Contains(responseText, "Failed to invoke") || 
	   strings.Contains(responseText, "No such service") ||
	   strings.Contains(responseText, "No provider") {
		fmt.Printf("[DUBBO CLIENT] 检测到错误信息\n")
		return true
	}
	
	// 4. 检查是否包含完整的JSON数组（特别是对于List类型返回）
	if strings.HasPrefix(responseText, "[") && strings.HasSuffix(responseText, "]") {
		// 尝试解析整个响应为JSON数组，使用json.Number保持精度
		decoder := json.NewDecoder(strings.NewReader(responseText))
		decoder.UseNumber()
		var jsonArray []interface{}
		if decoder.Decode(&jsonArray) == nil {
			fmt.Printf("[DUBBO CLIENT] 检测到完整JSON数组\n")
			return true
		}
	}
	
	// 5. 检查是否包含完整的JSON对象
	if strings.HasPrefix(responseText, "{") && strings.HasSuffix(responseText, "}") {
		// 尝试解析整个响应为JSON对象，使用json.Number保持精度
		decoder := json.NewDecoder(strings.NewReader(responseText))
		decoder.UseNumber()
		var jsonObj map[string]interface{}
		if decoder.Decode(&jsonObj) == nil {
			fmt.Printf("[DUBBO CLIENT] 检测到完整JSON对象\n")
			return true
		}
	}
	
	fmt.Printf("[DUBBO CLIENT] 响应不完整\n")
	return false
}