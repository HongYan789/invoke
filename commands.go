package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	// 直接使用原始结果，不进行额外的数据包装处理
	processedResult := result

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
// removeLSuffix 移除JSON字符串中数字的L后缀
func removeLSuffix(jsonStr string) string {
	// 使用正则表达式匹配数字后面的L后缀
	// 匹配模式：数字(可能包含小数点)后跟L，但L后面必须是非字母数字字符或字符串结尾
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)L([^a-zA-Z0-9]|$)`)
	return re.ReplaceAllString(jsonStr, "${1}${2}")
}

// hasLSuffixBigInt 检查参数是否包含L后缀的大整数
func hasLSuffixBigInt(param string) bool {
	// 检查是否包含可能导致精度丢失的大整数L后缀
	re := regexp.MustCompile(`\d{16,}L`) // 16位及以上的数字后跟L
	return re.MatchString(param)
}

// processLSuffixParam 处理包含L后缀的参数，保持大整数精度
func processLSuffixParam(param string) interface{} {
	// 使用正则表达式找到所有大整数L后缀并替换为字符串格式
	re := regexp.MustCompile(`(\d{16,})L([^a-zA-Z0-9]|$)`)
	processedParam := re.ReplaceAllStringFunc(param, func(match string) string {
		// 提取数字部分
		numMatch := regexp.MustCompile(`(\d{16,})L`).FindStringSubmatch(match)
		if len(numMatch) >= 2 {
			number := numMatch[1]
			// 替换为字符串格式，保持原有的分隔符
			suffix := match[len(numMatch[0]):]
			return `"` + number + `"` + suffix
		}
		return match
	})

	// 尝试解析为JSON
	decoder := json.NewDecoder(strings.NewReader(processedParam))
	decoder.UseNumber()
	var jsonValue interface{}
	if err := decoder.Decode(&jsonValue); err == nil {
		return convertJSONNumber(jsonValue)
	}

	// 如果解析失败，返回原始参数
	return param
}

func parseParams(params []string, types []string) ([]interface{}, error) {
	result := make([]interface{}, len(params))
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

	for i, param := range params {
		// 检查原始参数是否包含L后缀的大整数
		if hasLSuffixBigInt(param) {
			// 对于包含L后缀的大整数，特殊处理以保持精度
			processed := processLSuffixParam(param)
			result[i] = processed
			continue
		}

		// 预处理：移除JSON中的L后缀
		processedParam := removeLSuffix(param)

		// 尝试解析为JSON，使用json.Number保持精度
		decoder := json.NewDecoder(strings.NewReader(processedParam))
		decoder.UseNumber()
		var jsonValue interface{}
		if err := decoder.Decode(&jsonValue); err == nil {
			// 根据类型提示进行修正
			t := ""
			if i < len(types) {
				t = strings.TrimSpace(types[i])
			}
			if t == "java.lang.String" || t == "string" {
				result[i] = fmt.Sprintf("%v", jsonValue)
			} else if t == "java.lang.Object" || strings.HasPrefix(t, "com.") || strings.Contains(t, ".") {
				result[i] = deepToString(jsonValue)
			} else {
				result[i] = convertJSONNumber(jsonValue)
			}
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
		decoder := json.NewDecoder(strings.NewReader(param))
		decoder.UseNumber()
		var value json.Number
		err := decoder.Decode(&value)
		if err != nil {
			return nil, err
		}
		return convertJSONNumber(value), nil
	case "java.lang.Long", "long":
		decoder := json.NewDecoder(strings.NewReader(param))
		decoder.UseNumber()
		var value json.Number
		err := decoder.Decode(&value)
		if err != nil {
			return nil, err
		}
		return convertJSONNumber(value), nil
	case "java.lang.Boolean", "boolean":
		// 使用json.Number保持精度
		decoder := json.NewDecoder(strings.NewReader(param))
		decoder.UseNumber()
		var value bool
		err := decoder.Decode(&value)
		return value, err
	case "java.lang.Double", "double":
		// 使用json.Number保持精度
		decoder := json.NewDecoder(strings.NewReader(param))
		decoder.UseNumber()
		var value float64
		err := decoder.Decode(&value)
		return value, err
	default:
		// 尝试解析为JSON对象，使用json.Number保持精度
		decoder := json.NewDecoder(strings.NewReader(param))
		decoder.UseNumber()
		var value interface{}
		err := decoder.Decode(&value)
		if err != nil {
			return nil, err
		}
		if paramType == "java.lang.Object" || strings.HasPrefix(paramType, "com.") || strings.Contains(paramType, ".") {
			// 深度将数值转换为字符串，避免对象字段类型不匹配
			return func(v interface{}) interface{} {
				switch x := v.(type) {
				case json.Number:
					return string(x)
				case float64, float32, int64, int32, int, uint64, uint32, uint:
					return fmt.Sprintf("%v", x)
				case []interface{}:
					out := make([]interface{}, len(x))
					for i := range x {
						out[i] = func(y interface{}) interface{} {
							switch yy := y.(type) {
							case json.Number:
								return string(yy)
							case float64, float32, int64, int32, int, uint64, uint32, uint:
								return fmt.Sprintf("%v", yy)
							case []interface{}:
								inner := make([]interface{}, len(yy))
								for j := range yy {
									inner[j] = func(z interface{}) interface{} {
										switch zz := z.(type) {
										case json.Number:
											return string(zz)
										case float64, float32, int64, int32, int, uint64, uint32, uint:
											return fmt.Sprintf("%v", zz)
										case []interface{}:
											return yy // unreachable in nested inline; keep simple
										case map[string]interface{}:
											m := make(map[string]interface{}, len(zz))
											for k, vv := range zz {
												m[k] = func(tv interface{}) interface{} {
													switch tt := tv.(type) {
													case json.Number:
														return string(tt)
													case float64, float32, int64, int32, int, uint64, uint32, uint:
														return fmt.Sprintf("%v", tt)
													case []interface{}:
														ii := make([]interface{}, len(tt))
														for p := range tt {
															ii[p] = tt[p]
														}
														return ii
													case map[string]interface{}:
														return tt
													default:
														return tv
													}
												}(vv)
											}
											return m
										default:
											return z
										}
									}(yy[j])
								}
								return inner
							case map[string]interface{}:
								m := make(map[string]interface{}, len(yy))
								for k, vv := range yy {
									m[k] = func(tv interface{}) interface{} {
										switch tt := tv.(type) {
										case json.Number:
											return string(tt)
										case float64, float32, int64, int32, int, uint64, uint32, uint:
											return fmt.Sprintf("%v", tt)
										case []interface{}:
											return tt
										case map[string]interface{}:
											return tt
										default:
											return tv
										}
									}(vv)
								}
								return m
							default:
								return y
							}
						}(x[i])
					}
					return out
				case map[string]interface{}:
					out := make(map[string]interface{}, len(x))
					for k, vv := range x {
						out[k] = func(tv interface{}) interface{} {
							switch tt := tv.(type) {
							case json.Number:
								return string(tt)
							case float64, float32, int64, int32, int, uint64, uint32, uint:
								return fmt.Sprintf("%v", tt)
							case []interface{}:
								return tt
							case map[string]interface{}:
								return tt
							default:
								return tv
							}
						}(vv)
					}
					return out
				default:
					return v
				}
			}(value), nil
		}
		return convertJSONNumber(value), nil
	}
}

// generateExampleParams 生成示例参数
func generateExampleParams(types []string) []interface{} {
	color.Blue("[EXAMPLE] 开始生成示例参数，类型数量: %d", len(types))
	color.Cyan("[EXAMPLE] 输入类型列表: %v", types)

	examples := make([]interface{}, len(types))

	for i, paramType := range types {
		color.Cyan("[EXAMPLE] 处理第%d个参数，类型: %s", i+1, paramType)

		switch paramType {
		case "java.lang.String", "string":
			examples[i] = "example"
			color.Green("[EXAMPLE] 生成字符串示例: %v", examples[i])
		case "java.lang.Integer", "int":
			examples[i] = 0
			color.Green("[EXAMPLE] 生成整数示例: %v", examples[i])
		case "java.lang.Long", "long":
			examples[i] = int64(0)
			color.Green("[EXAMPLE] 生成长整数示例: %v", examples[i])
		case "java.lang.Boolean", "boolean":
			examples[i] = false
			color.Green("[EXAMPLE] 生成布尔示例: %v", examples[i])
		case "java.lang.Double", "double":
			examples[i] = 0.0
			color.Green("[EXAMPLE] 生成双精度示例: %v", examples[i])
		case "java.util.List":
			examples[i] = []interface{}{"item1", "item2"}
			color.Green("[EXAMPLE] 生成列表示例: %v", examples[i])
		case "java.util.Map":
			examples[i] = map[string]interface{}{"key": "value"}
			color.Green("[EXAMPLE] 生成映射示例: %v", examples[i])
		default:
			if strings.Contains(paramType, "List") {
				examples[i] = []interface{}{map[string]interface{}{"class": paramType}}
				color.Yellow("[EXAMPLE] 生成自定义列表示例: %v", examples[i])
			} else {
				examples[i] = map[string]interface{}{"class": paramType}
				color.Yellow("[EXAMPLE] 生成自定义对象示例: %v", examples[i])
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
