package main

import (
	"encoding/json"
	"log"
	"strings"
)

// ListResultHandler 专门处理List类型返回结果的处理器
type ListResultHandler struct{}

// NewListResultHandler 创建新的List结果处理器
func NewListResultHandler() *ListResultHandler {
	return &ListResultHandler{}
}

// HandleListResult 处理List类型返回结果
func (lrh *ListResultHandler) HandleListResult(result interface{}, methodName string, params []interface{}) interface{} {
	log.Printf("[ListHandler] 开始处理方法 %s 的返回结果，类型: %T, 参数数量: %d", methodName, result, len(params))
	
	// 记录参数详情
	for i, param := range params {
		log.Printf("[ListHandler] 参数%d: %T = %v", i, param, param)
	}
	
	// 检查方法名是否应该返回List类型
	if !lrh.isListMethod(methodName) {
		log.Printf("[ListHandler] 方法 %s 不是List类型方法，跳过处理", methodName)
		return result
	}
	
	log.Printf("[ListHandler] 方法 %s 是List类型方法，开始处理", methodName)
	
	// 移除所有参数检查逻辑，入参是否为空不影响最终调用结果
	// 直接处理返回结果
	
	// 如果结果已经是数组类型，直接返回
	if lrh.isArrayType(result) {
		log.Printf("[ListHandler] 结果已经是数组类型，直接返回")
		return result
	}
	
	// 如果结果是字符串，尝试解析为JSON对象或数组
	if resultStr, ok := result.(string); ok {
		log.Printf("[ListHandler] 结果是字符串类型，尝试解析为JSON")
		
		// 处理双重转义的JSON字符串
		unquotedStr := lrh.unquoteJSONString(resultStr)
		
		// 尝试解析为JSON数组
		var jsonArray []interface{}
		if err := json.Unmarshal([]byte(unquotedStr), &jsonArray); err == nil {
			log.Printf("[ListHandler] 成功解析为JSON数组，元素数量: %d", len(jsonArray))
			return jsonArray
		}
		
		// 尝试解析为JSON对象
		var jsonObj map[string]interface{}
		if err := json.Unmarshal([]byte(unquotedStr), &jsonObj); err == nil {
			log.Printf("[ListHandler] 成功解析为JSON对象，字段数量: %d", len(jsonObj))
			// 对于应该返回List但实际返回单个对象的情况，包装成数组返回
			return []interface{}{jsonObj}
		}
		
		log.Printf("[ListHandler] 字符串无法解析为JSON，返回原始字符串")
		return result
	}
	
	// 对于应该返回List但实际返回单个对象的情况，包装成数组返回
	log.Printf("[ListHandler] 结果是单个对象，包装成数组返回")
	return []interface{}{result}
}

// isListMethod 检查方法是否应该返回List类型
func (lrh *ListResultHandler) isListMethod(methodName string) bool {
	// 更精确地匹配应该返回List的方法
	// 只有明确包含列表相关关键词的方法才应该返回List
	listPatterns := []string{
		"list", "getlist", "find", "query", "search", "select", "batch",
		"byids", "byidlist", "bycodes", "bycodelist",
		"companyinfos", "infos", // 注意这里移除了"companyinfo"
	}
	
	methodName = strings.ToLower(methodName)
	for _, pattern := range listPatterns {
		if strings.Contains(methodName, pattern) {
			return true
		}
	}
	
	// 特殊处理您提到的方法
	if strings.Contains(strings.ToLower(methodName), "getcompanyinfobycompanyidsanddanwbh") {
		return true
	}
	
	// 明确不应该返回List的方法
	nonListPatterns := []string{
		"getcompanyinfobycompanyid", // 单数形式，应该返回单个对象
	}
	
	for _, pattern := range nonListPatterns {
		if strings.Contains(methodName, pattern) {
			return false
		}
	}
	
	return false
}

// isArrayType 检查是否为数组类型
func (lrh *ListResultHandler) isArrayType(result interface{}) bool {
	switch result.(type) {
	case []interface{}, []map[string]interface{}:
		return true
	default:
		return false
	}
}

// isEmptyList 检查是否为空列表
func (lrh *ListResultHandler) isEmptyList(param interface{}) bool {
	log.Printf("[ListHandler] 检查参数是否为空列表, 类型: %T, 值: %v", param, param)
	
	switch v := param.(type) {
	case []interface{}:
		log.Printf("[ListHandler] 参数是[]interface{}类型, 长度: %d", len(v))
		return len(v) == 0
	case []map[string]interface{}:
		log.Printf("[ListHandler] 参数是[]map[string]interface{}类型, 长度: %d", len(v))
		return len(v) == 0
	case nil:
		log.Printf("[ListHandler] 参数是nil")
		return true
	default:
		log.Printf("[ListHandler] 参数是其他类型: %T", param)
		// 尝试转换为字符串并检查是否为"[]"
		if str, ok := param.(string); ok {
			log.Printf("[ListHandler] 参数是字符串类型: %s", str)
			return str == "[]" || str == ""
		}
		return false
	}
}

// unquoteJSONString 处理双重转义的JSON字符串
func (lrh *ListResultHandler) unquoteJSONString(str string) string {
	// 去除首尾空格
	str = strings.TrimSpace(str)
	
	// 如果是双重引号包围的字符串，去除外层引号
	if strings.HasPrefix(str, "\"") && strings.HasSuffix(str, "\"") && len(str) > 2 {
		// 去除外层引号
		innerStr := str[1 : len(str)-1]
		
		// 处理转义字符
		unquoted, err := json.Marshal(innerStr)
		if err == nil {
			// 去除json.Marshal添加的引号
			unquotedStr := string(unquoted)
			if len(unquotedStr) > 2 && strings.HasPrefix(unquotedStr, "\"") && strings.HasSuffix(unquotedStr, "\"") {
				return unquotedStr[1 : len(unquotedStr)-1]
			}
		}
		
		return innerStr
	}
	
	return str
}

// enhanceWebServerWithListHandling 为Web服务器添加List结果处理功能
// 注意：由于WebServer类型在当前文件中未定义，这里只是声明函数签名
func enhanceWebServerWithListHandling(ws interface{}) {
	log.Println("List结果处理增强: 已启用")
}