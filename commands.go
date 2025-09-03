package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

// runInvokeCommand 执行Dubbo服务调用
func runInvokeCommand(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("需要指定调用表达式或服务名和方法名")
	}

	var serviceName, methodName string
	var params []string

	// 检查是否使用新的调用格式: service.method(params)
	if strings.Contains(args[0], "(") && strings.Contains(args[0], ")") {
		// 解析新格式: com.example.Service.method({"param":"value"})
		serviceName, methodName, params = parseInvokeExpression(args[0])
		if serviceName == "" || methodName == "" {
			return fmt.Errorf("无效的调用表达式格式，期望格式: service.method(params)")
		}
	} else {
		// 使用原有格式: service method params...
		if len(args) < 2 {
			return fmt.Errorf("需要至少指定服务名和方法名")
		}
		serviceName = args[0]
		methodName = args[1]
		params = args[2:]
	}

	// 获取命令行参数
	registry, _ := cmd.Flags().GetString("registry")
	appName, _ := cmd.Flags().GetString("app")
	timeout, _ := cmd.Flags().GetInt("timeout")
	version, _ := cmd.Flags().GetString("version")
	group, _ := cmd.Flags().GetString("group")
	generic, _ := cmd.Flags().GetBool("generic")
	types, _ := cmd.Flags().GetStringSlice("types")
	example, _ := cmd.Flags().GetBool("example")
	verbose, _ := cmd.Flags().GetBool("verbose")

	if verbose {
		color.Cyan("调用参数:")
		color.Cyan("  服务: %s", serviceName)
		color.Cyan("  方法: %s", methodName)
		color.Cyan("  注册中心: %s", registry)
		color.Cyan("  应用名: %s", appName)
		color.Cyan("  超时: %dms", timeout)
		if version != "" {
			color.Cyan("  版本: %s", version)
		}
		if group != "" {
			color.Cyan("  分组: %s", group)
		}
		color.Cyan("  泛化调用: %t", generic)
		color.Cyan("  参数: %v", params)
	}

	// 如果需要生成示例参数
	if example {
		exampleParams := generateExampleParams(types)
		color.Yellow("示例参数:")
		for i, param := range exampleParams {
			color.Yellow("  参数%d: %s", i+1, param)
		}
		return nil
	}

	// 创建Dubbo客户端配置
	config := &DubboConfig{
		Registry:    registry,
		Application: appName,
		Timeout:     time.Duration(timeout) * time.Millisecond,
		Version:     version,
		Group:       group,
	}

	// 创建Dubbo客户端
	client, err := NewDubboClient(config)
	if err != nil {
		return fmt.Errorf("创建Dubbo客户端失败: %v", err)
	}
	defer client.Close()

	// 解析参数
	parsedParams, err := parseParams(params, types)
	if err != nil {
		return fmt.Errorf("解析参数失败: %v", err)
	}

	// 执行调用
	var result interface{}
	if generic {
		result, err = client.GenericInvoke(serviceName, methodName, types, parsedParams)
	} else {
		result, err = client.DirectInvoke(serviceName, methodName, parsedParams)
	}

	if err != nil {
		return fmt.Errorf("调用失败: %v", err)
	}

	// 使用List结果处理器处理返回结果，传递参数信息
	listHandler := NewListResultHandler()
	processedResult := listHandler.HandleListResult(result, methodName, parsedParams)

	// 输出结果
	color.Green("调用成功:")
	resultJson, _ := json.MarshalIndent(processedResult, "", "  ")
	fmt.Println(string(resultJson))

	return nil
}

// runListCommand 列出可用服务
func runListCommand(cmd *cobra.Command, args []string) error {
	registry, _ := cmd.Flags().GetString("registry")
	appName, _ := cmd.Flags().GetString("app")
	showMethods, _ := cmd.Flags().GetBool("methods")
	filter, _ := cmd.Flags().GetString("filter")
	verbose, _ := cmd.Flags().GetBool("verbose")

	if verbose {
		color.Cyan("连接注册中心: %s", registry)
	}

	// 创建Dubbo客户端配置
	config := &DubboConfig{
		Registry:    registry,
		Application: appName,
		Timeout:     5 * time.Second,
	}

	// 创建Dubbo客户端
	client, err := NewDubboClient(config)
	if err != nil {
		return fmt.Errorf("创建Dubbo客户端失败: %v", err)
	}
	defer client.Close()

	// 获取服务列表
	services, err := client.ListServices()
	if err != nil {
		return fmt.Errorf("获取服务列表失败: %v", err)
	}

	// 过滤服务
	if filter != "" {
		filteredServices := make([]string, 0)
		for _, service := range services {
			if strings.Contains(service, filter) {
				filteredServices = append(filteredServices, service)
			}
		}
		services = filteredServices
	}

	// 如果指定了特定服务，显示其方法
	if len(args) > 0 {
		serviceName := args[0]
		methods, err := client.ListMethods(serviceName)
		if err != nil {
			return fmt.Errorf("获取服务方法失败: %v", err)
		}

		color.Green("服务 %s 的方法:", serviceName)
		for _, method := range methods {
			color.White("  %s", method)
		}
		return nil
	}

	// 显示服务列表
	color.Green("可用服务列表 (共%d个):", len(services))
	for _, service := range services {
		color.White("  %s", service)
		if showMethods {
			methods, err := client.ListMethods(service)
			if err == nil {
				for _, method := range methods {
					color.Cyan("    └─ %s", method)
				}
			}
		}
	}

	return nil
}

// runConfigInitCommand 初始化配置文件
func runConfigInitCommand(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("config")

	// 默认配置
	defaultConfig := map[string]interface{}{
		"registry": map[string]interface{}{
			"address":  "zookeeper://127.0.0.1:2181",
			"timeout":  "5s",
			"username": "",
			"password": "",
		},
		"application": map[string]interface{}{
			"name":    "dubbo-invoke-client",
			"version": "1.0.0",
			"owner":   "",
		},
		"consumer": map[string]interface{}{
			"timeout":     "3s",
			"retries":     0,
			"loadbalance": "random",
			"generic":     true,
		},
		"protocol": map[string]interface{}{
			"name": "dubbo",
			"port": 20880,
		},
	}

	// 检查文件是否已存在
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("配置文件 %s 已存在", configFile)
	}

	// 创建配置文件
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	color.Green("配置文件已创建: %s", configFile)
	return nil
}

// runConfigShowCommand 显示当前配置
func runConfigShowCommand(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("config")

	// 读取配置文件
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 显示配置
	color.Green("当前配置 (%s):", configFile)
	allSettings := viper.AllSettings()
	data, _ := yaml.Marshal(allSettings)
	fmt.Println(string(data))

	return nil
}

// parseParams 解析命令行参数
func parseParams(params []string, types []string) ([]interface{}, error) {
	result := make([]interface{}, len(params))

	for i, param := range params {
		// 尝试解析为JSON
		var jsonValue interface{}
		if err := json.Unmarshal([]byte(param), &jsonValue); err == nil {
			result[i] = jsonValue
			continue
		}

		// 如果指定了类型，按类型解析
		if i < len(types) {
			parsed, err := parseByType(param, types[i])
			if err != nil {
				return nil, fmt.Errorf("解析参数%d失败: %v", i+1, err)
			}
			result[i] = parsed
		} else {
			// 默认作为字符串处理
			result[i] = param
		}
	}

	return result, nil
}

// parseByType 按指定类型解析参数
func parseByType(param, paramType string) (interface{}, error) {
	switch paramType {
	case "java.lang.String", "string":
		return param, nil
	case "java.lang.Integer", "int":
		var value int
		err := json.Unmarshal([]byte(param), &value)
		return value, err
	case "java.lang.Long", "long":
		var value int64
		err := json.Unmarshal([]byte(param), &value)
		return value, err
	case "java.lang.Boolean", "boolean":
		var value bool
		err := json.Unmarshal([]byte(param), &value)
		return value, err
	case "java.lang.Double", "double":
		var value float64
		err := json.Unmarshal([]byte(param), &value)
		return value, err
	default:
		// 尝试解析为JSON对象
		var value interface{}
		err := json.Unmarshal([]byte(param), &value)
		return value, err
	}
}

// generateExampleParams 生成示例参数
func generateExampleParams(types []string) []string {
	color.Blue("[EXAMPLE] 开始生成示例参数，类型数量: %d", len(types))
	color.Cyan("[EXAMPLE] 输入类型列表: %v", types)
	
	examples := make([]string, len(types))

	for i, paramType := range types {
		color.Cyan("[EXAMPLE] 处理第%d个参数，类型: %s", i+1, paramType)
		
		switch paramType {
		case "java.lang.String", "string":
			examples[i] = `"example"`
			color.Green("[EXAMPLE] 生成字符串示例: %s", examples[i])
		case "java.lang.Integer", "int":
			examples[i] = "0"
			color.Green("[EXAMPLE] 生成整数示例: %s", examples[i])
		case "java.lang.Long", "long":
			examples[i] = "0"
			color.Green("[EXAMPLE] 生成长整数示例: %s", examples[i])
		case "java.lang.Boolean", "boolean":
			examples[i] = "false"
			color.Green("[EXAMPLE] 生成布尔示例: %s", examples[i])
		case "java.lang.Double", "double":
			examples[i] = "0.0"
			color.Green("[EXAMPLE] 生成双精度示例: %s", examples[i])
		case "java.util.List":
			examples[i] = `["item1", "item2"]`
			color.Green("[EXAMPLE] 生成列表示例: %s", examples[i])
		case "java.util.Map":
			examples[i] = `{"key": "value"}`
			color.Green("[EXAMPLE] 生成映射示例: %s", examples[i])
		default:
			if strings.Contains(paramType, "List") {
				examples[i] = `[{"class":"` + paramType + `"}]`
				color.Yellow("[EXAMPLE] 生成自定义列表示例: %s", examples[i])
			} else {
				examples[i] = `{"class":"` + paramType + `"}`
				color.Yellow("[EXAMPLE] 生成自定义对象示例: %s", examples[i])
			}
		}
	}

	color.Green("[EXAMPLE] 示例参数生成完成，结果: %v", examples)
	return examples
}

// parseInvokeExpression 解析调用表达式
// 格式: com.example.Service.method({"param":"value"})
func parseInvokeExpression(expression string) (serviceName, methodName string, params []string) {
	// 查找方法调用的开始位置
	parenIndex := strings.Index(expression, "(")
	if parenIndex == -1 {
		return "", "", nil
	}

	// 提取方法部分 (service.method)
	methodPart := expression[:parenIndex]
	lastDotIndex := strings.LastIndex(methodPart, ".")
	if lastDotIndex == -1 {
		return "", "", nil
	}

	serviceName = methodPart[:lastDotIndex]
	methodName = methodPart[lastDotIndex+1:]

	// 提取参数部分
	paramsPart := expression[parenIndex+1:]
	if strings.HasSuffix(paramsPart, ")") {
		paramsPart = paramsPart[:len(paramsPart)-1]
	}

	// 如果参数部分为空，返回空参数列表
	if strings.TrimSpace(paramsPart) == "" {
		return serviceName, methodName, []string{}
	}

	// 解析参数 - 支持JSON对象和简单参数
	params = parseParametersFromExpression(paramsPart)
	return serviceName, methodName, params
}

// parseParametersFromExpression 从表达式中解析参数
func parseParametersFromExpression(paramsPart string) []string {
	paramsPart = strings.TrimSpace(paramsPart)
	if paramsPart == "" {
		return []string{}
	}

	var params []string
	var current strings.Builder
	var braceCount, bracketCount int
	var inQuotes bool
	var escapeNext bool

	for _, char := range paramsPart {
		if escapeNext {
			current.WriteRune(char)
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			current.WriteRune(char)
			continue
		}

		if char == '"' {
			inQuotes = !inQuotes
		}

		if !inQuotes {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
			} else if char == '[' {
				bracketCount++
			} else if char == ']' {
				bracketCount--
			} else if char == ',' && braceCount == 0 && bracketCount == 0 {
				// 找到参数分隔符
				param := strings.TrimSpace(current.String())
				if param != "" {
					params = append(params, param)
				}
				current.Reset()
				continue
			}
		}

		current.WriteRune(char)
	}

	// 添加最后一个参数
	param := strings.TrimSpace(current.String())
	if param != "" {
		params = append(params, param)
	}

	return params
}