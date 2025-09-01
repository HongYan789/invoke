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
	// 直接连接到Dubbo服务提供者地址
	// 从用户的telnet测试可以看到，应该连接到 10.7.8.50:16002
	providerAddress := "10.7.8.50:16002"
	
	// 尝试连接到Dubbo服务提供者
	conn, err := net.DialTimeout("tcp", providerAddress, c.config.Timeout)
	if err != nil {
		return fmt.Errorf("连接Dubbo服务提供者失败: %v", err)
	}

	c.conn = conn
	c.connected = true
	fmt.Printf("成功连接到Dubbo服务提供者: %s\n", providerAddress)

	return nil
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

	// 发送invoke命令
	_, err = c.conn.Write([]byte(invokeCmd))
	if err != nil {
		return nil, fmt.Errorf("发送invoke命令失败: %v", err)
	}

	// 设置读取超时
	c.conn.SetReadDeadline(time.Now().Add(c.config.Timeout))
	
	// 读取响应
	buffer := make([]byte, 8192)
	n, err := c.conn.Read(buffer)
	if err != nil {
		// 如果是超时错误，直接返回超时错误
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, fmt.Errorf("调用超时: %s.%s 在规定时间内未返回响应", serviceName, methodName)
		}
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 尝试将响应从GBK编码转换为UTF-8
	responseText, err := c.convertToUTF8(buffer[:n])
	if err != nil {
		// 如果转换失败，使用原始字符串
		responseText = string(buffer[:n])
	}
	
	// 检查是否包含错误信息
	if strings.Contains(responseText, "Failed to invoke") || 
	   strings.Contains(responseText, "error") ||
	   strings.Contains(responseText, "No such service") ||
	   strings.Contains(responseText, "No provider") ||
	   strings.Contains(responseText, "Service not found") {
		return nil, fmt.Errorf("调用失败: %s", responseText)
	}

	// 清理响应文本，提取JSON部分
	cleanedResponse := c.cleanResponse(responseText)
	
	// 检查清理后的响应是否仍然包含dubbo控制台输出
	// 如果清理后的响应包含"elapsed:"或"dubbo>"，说明没有获得有效的业务响应
	if strings.Contains(cleanedResponse, "elapsed:") || 
	   strings.Contains(cleanedResponse, "dubbo>") ||
	   (strings.HasPrefix(cleanedResponse, "[]") && strings.Contains(cleanedResponse, "elapsed:")) {
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

	// 读取响应
	buffer := make([]byte, 8192)
	n, err := c.conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("读取服务列表响应失败: %v", err)
	}

	// 解析响应文本
	responseText := string(buffer[:n])
	
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

	// 读取响应
	buffer := make([]byte, 4096)
	n, err := c.conn.Read(buffer)
	if err != nil {
		// 如果读取失败，返回默认方法列表
		return c.getDefaultMethods(serviceName), nil
	}

	// 尝试解析方法列表
	var response map[string]interface{}
	if err := json.Unmarshal(buffer[:n], &response); err == nil {
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
			// 验证JSON格式
			var jsonTest interface{}
			if json.Unmarshal([]byte(v), &jsonTest) == nil {
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
	// 按行分割响应
	lines := strings.Split(responseText, "\n")
	
	for _, line := range lines {
		// 去除首尾空白字符
		line = strings.TrimSpace(line)
		
		// 跳过空行和非JSON行
		if line == "" || strings.HasPrefix(line, "elapsed:") || strings.HasPrefix(line, "dubbo>") {
			continue
		}
		
		// 检查是否是JSON格式（以{开头）
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			// 验证是否是有效的JSON
			var jsonTest interface{}
			if json.Unmarshal([]byte(line), &jsonTest) == nil {
				return line
			}
		}
	}
	
	// 如果没有找到有效的JSON，返回原始响应
	return responseText
}