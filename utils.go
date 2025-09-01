package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ParameterType 参数类型枚举
type ParameterType string

const (
	TypeString  ParameterType = "string"
	TypeInt     ParameterType = "int"
	TypeLong    ParameterType = "long"
	TypeDouble  ParameterType = "double"
	TypeFloat   ParameterType = "float"
	TypeBoolean ParameterType = "boolean"
	TypeObject  ParameterType = "object"
	TypeArray   ParameterType = "array"
	TypeMap     ParameterType = "map"
	TypeDate    ParameterType = "date"
)

// Parameter 参数定义
type Parameter struct {
	Name         string        `json:"name"`
	Type         ParameterType `json:"type"`
	JavaType     string        `json:"javaType"`
	Value        interface{}   `json:"value"`
	DefaultValue interface{}   `json:"defaultValue,omitempty"`
	Required     bool          `json:"required"`
	Description  string        `json:"description,omitempty"`
}

// TypeInferrer 类型推断器
type TypeInferrer struct{}

// NewTypeInferrer 创建类型推断器
func NewTypeInferrer() *TypeInferrer {
	return &TypeInferrer{}
}

// InferType 推断参数类型
func (ti *TypeInferrer) InferType(javaType string) ParameterType {
	// 清理Java类型字符串
	javaType = strings.TrimSpace(javaType)
	javaType = strings.ToLower(javaType)

	// 基本类型映射
	switch javaType {
	case "string", "java.lang.string":
		return TypeString
	case "int", "integer", "java.lang.integer":
		return TypeInt
	case "long", "java.lang.long":
		return TypeLong
	case "double", "java.lang.double":
		return TypeDouble
	case "float", "java.lang.float":
		return TypeFloat
	case "boolean", "java.lang.boolean":
		return TypeBoolean
	case "date", "java.util.date", "java.time.localdatetime", "java.time.localdate":
		return TypeDate
	}

	// 集合类型
	if strings.Contains(javaType, "list") || strings.Contains(javaType, "array") || strings.HasSuffix(javaType, "[]") {
		return TypeArray
	}

	// Map类型
	if strings.Contains(javaType, "map") || strings.Contains(javaType, "hashmap") {
		return TypeMap
	}

	// 默认为对象类型
	return TypeObject
}

// GenerateDefaultValue 生成默认值
func (ti *TypeInferrer) GenerateDefaultValue(paramType ParameterType, javaType string) interface{} {
	switch paramType {
	case TypeString:
		return "示例字符串"
	case TypeInt:
		return 0
	case TypeLong:
		return int64(0)
	case TypeDouble:
		return 0.0
	case TypeFloat:
		return float32(0.0)
	case TypeBoolean:
		return false
	case TypeDate:
		return time.Now().Format("2006-01-02 15:04:05")
	case TypeArray:
		return []interface{}{}
	case TypeMap:
		return map[string]interface{}{}
	case TypeObject:
		return map[string]interface{}{
			"field1": "value1",
			"field2": "value2",
		}
	default:
		return nil
	}
}

// ParseParameterValue 解析参数值
func (ti *TypeInferrer) ParseParameterValue(value string, paramType ParameterType) (interface{}, error) {
	if value == "" {
		return ti.GenerateDefaultValue(paramType, ""), nil
	}

	switch paramType {
	case TypeString:
		return value, nil
	case TypeInt:
		return strconv.Atoi(value)
	case TypeLong:
		return strconv.ParseInt(value, 10, 64)
	case TypeDouble:
		return strconv.ParseFloat(value, 64)
	case TypeFloat:
		v, err := strconv.ParseFloat(value, 32)
		return float32(v), err
	case TypeBoolean:
		return strconv.ParseBool(value)
	case TypeDate:
		// 尝试多种日期格式
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02",
			"15:04:05",
			time.RFC3339,
		}
		for _, format := range formats {
			if t, err := time.Parse(format, value); err == nil {
				return t.Format("2006-01-02 15:04:05"), nil
			}
		}
		return value, nil // 如果解析失败，返回原始字符串
	case TypeArray, TypeMap, TypeObject:
		// 尝试解析JSON
		var result interface{}
		if err := json.Unmarshal([]byte(value), &result); err != nil {
			return nil, fmt.Errorf("无效的JSON格式: %v", err)
		}
		return result, nil
	default:
		return value, nil
	}
}

// FormatParameterForDisplay 格式化参数用于显示
func (ti *TypeInferrer) FormatParameterForDisplay(param *Parameter) string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("参数名: %s\n", param.Name))
	builder.WriteString(fmt.Sprintf("类型: %s (%s)\n", param.Type, param.JavaType))
	
	if param.Required {
		builder.WriteString("必填: 是\n")
	} else {
		builder.WriteString("必填: 否\n")
	}
	
	if param.Description != "" {
		builder.WriteString(fmt.Sprintf("描述: %s\n", param.Description))
	}
	
	if param.DefaultValue != nil {
		builder.WriteString(fmt.Sprintf("默认值: %v\n", param.DefaultValue))
	}
	
	if param.Value != nil {
		builder.WriteString(fmt.Sprintf("当前值: %v\n", param.Value))
	}
	
	return builder.String()
}

// GenerateExampleJSON 生成示例JSON
func (ti *TypeInferrer) GenerateExampleJSON(params []Parameter) (string, error) {
	example := make(map[string]interface{})
	
	for _, param := range params {
		if param.Value != nil {
			example[param.Name] = param.Value
		} else if param.DefaultValue != nil {
			example[param.Name] = param.DefaultValue
		} else {
			example[param.Name] = ti.GenerateDefaultValue(param.Type, param.JavaType)
		}
	}
	
	data, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return "", fmt.Errorf("生成JSON失败: %v", err)
	}
	
	return string(data), nil
}

// ValidateParameters 验证参数
func (ti *TypeInferrer) ValidateParameters(params []Parameter) []string {
	var errors []string
	
	for _, param := range params {
		if param.Required && param.Value == nil {
			errors = append(errors, fmt.Sprintf("必填参数 '%s' 不能为空", param.Name))
		}
		
		// 类型验证
		if param.Value != nil {
			if err := ti.validateParameterType(param.Value, param.Type); err != nil {
				errors = append(errors, fmt.Sprintf("参数 '%s' 类型错误: %v", param.Name, err))
			}
		}
	}
	
	return errors
}

// validateParameterType 验证参数类型
func (ti *TypeInferrer) validateParameterType(value interface{}, expectedType ParameterType) error {
	valueType := reflect.TypeOf(value)
	if valueType == nil {
		return nil // nil值跳过验证
	}
	
	switch expectedType {
	case TypeString:
		if valueType.Kind() != reflect.String {
			return fmt.Errorf("期望字符串类型，实际为 %s", valueType.Kind())
		}
	case TypeInt:
		if valueType.Kind() != reflect.Int && valueType.Kind() != reflect.Int32 && valueType.Kind() != reflect.Int64 {
			return fmt.Errorf("期望整数类型，实际为 %s", valueType.Kind())
		}
	case TypeBoolean:
		if valueType.Kind() != reflect.Bool {
			return fmt.Errorf("期望布尔类型，实际为 %s", valueType.Kind())
		}
	case TypeArray:
		if valueType.Kind() != reflect.Slice && valueType.Kind() != reflect.Array {
			return fmt.Errorf("期望数组类型，实际为 %s", valueType.Kind())
		}
	case TypeMap, TypeObject:
		if valueType.Kind() != reflect.Map {
			return fmt.Errorf("期望对象类型，实际为 %s", valueType.Kind())
		}
	}
	
	return nil
}

// ConvertToJavaTypes 转换为Java类型数组
func (ti *TypeInferrer) ConvertToJavaTypes(params []Parameter) []string {
	javaTypes := make([]string, len(params))
	for i, param := range params {
		javaTypes[i] = param.JavaType
	}
	return javaTypes
}

// ConvertToValues 转换为值数组
func (ti *TypeInferrer) ConvertToValues(params []Parameter) []interface{} {
	values := make([]interface{}, len(params))
	for i, param := range params {
		if param.Value != nil {
			values[i] = param.Value
		} else if param.DefaultValue != nil {
			values[i] = param.DefaultValue
		} else {
			values[i] = ti.GenerateDefaultValue(param.Type, param.JavaType)
		}
	}
	return values
}

// ParseMethodSignature 解析方法签名
func (ti *TypeInferrer) ParseMethodSignature(signature string) (methodName string, params []Parameter, err error) {
	// 简单的方法签名解析
	// 格式: methodName(type1 param1, type2 param2)
	
	parts := strings.Split(signature, "(")
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("无效的方法签名格式")
	}
	
	methodName = strings.TrimSpace(parts[0])
	paramStr := strings.TrimSuffix(strings.TrimSpace(parts[1]), ")")
	
	if paramStr == "" {
		return methodName, []Parameter{}, nil
	}
	
	paramParts := strings.Split(paramStr, ",")
	params = make([]Parameter, len(paramParts))
	
	for i, paramPart := range paramParts {
		paramPart = strings.TrimSpace(paramPart)
		parts := strings.Fields(paramPart)
		
		if len(parts) < 2 {
			return "", nil, fmt.Errorf("无效的参数格式: %s", paramPart)
		}
		
		javaType := parts[0]
		paramName := parts[1]
		paramType := ti.InferType(javaType)
		
		params[i] = Parameter{
			Name:         paramName,
			Type:         paramType,
			JavaType:     javaType,
			DefaultValue: ti.GenerateDefaultValue(paramType, javaType),
			Required:     true,
		}
	}
	
	return methodName, params, nil
}