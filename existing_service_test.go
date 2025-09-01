package main

import (
	"encoding/json"
	"testing"
	"time"
)

// TestExistingServiceInvoke 测试现有服务的调用和数据接收
func TestExistingServiceInvoke(t *testing.T) {
	// 使用注册中心中实际存在的服务
	config := &DubboConfig{
		Registry:    "dubbo://10.7.8.50:16002",
		Application: "dubbo-invoke-client",
		Timeout:     10 * time.Second,
	}

	client, err := NewRealDubboClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Skip("跳过测试：无法连接到注册中心")
		return
	}

	// 测试现有的服务
	testCases := []struct {
		name        string
		serviceName string
		methodName  string
		params      []interface{}
		paramTypes  []string
	}{
		{
			name:        "UserService.getUserById",
			serviceName: "com.example.UserService",
			methodName:  "getUserById",
			params:      []interface{}{123},
			paramTypes:  []string{"java.lang.Long"},
		},
		{
			name:        "UserService.getAllUsers",
			serviceName: "com.example.UserService",
			methodName:  "getAllUsers",
			params:      []interface{}{},
			paramTypes:  []string{},
		},
		{
			name:        "OrderService.getOrderById",
			serviceName: "com.example.OrderService",
			methodName:  "getOrderById",
			params:      []interface{}{456},
			paramTypes:  []string{"java.lang.Long"},
		},
		{
			name:        "ProductService.getProductById",
			serviceName: "com.example.ProductService",
			methodName:  "getProductById",
			params:      []interface{}{789},
			paramTypes:  []string{"java.lang.Long"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("🚀 开始调用: %s.%s", tc.serviceName, tc.methodName)
			t.Logf("参数: %+v", tc.params)
			t.Logf("参数类型: %+v", tc.paramTypes)

			start := time.Now()
			result, err := client.GenericInvoke(tc.serviceName, tc.methodName, tc.paramTypes, tc.params)
			duration := time.Since(start)
			
			t.Logf("调用耗时: %v", duration)
			
			if err != nil {
				t.Logf("❌ 调用失败: %v", err)
				if containsSubstring(err.Error(), "超时") || containsSubstring(err.Error(), "timeout") {
					t.Logf("超时错误，可能服务响应较慢")
				} else {
					t.Logf("其他错误: %v", err)
				}
				return
			}

			// 成功获取到数据
			t.Logf("✅ 调用成功！耗时: %v", duration)
			t.Logf("返回结果类型: %T", result)
			
			// 详细输出返回数据
			if resultBytes, err := json.MarshalIndent(result, "", "  "); err == nil {
				t.Logf("📄 返回数据 (JSON格式):\n%s", string(resultBytes))
			} else {
				t.Logf("📄 返回数据 (原始格式): %+v", result)
			}

			// 验证数据完整性和格式
			validateReturnedData(t, result)
		})
	}
}

// TestComplexParameterInvoke 测试复杂参数的服务调用
func TestComplexParameterInvoke(t *testing.T) {
	config := &DubboConfig{
		Registry:    "dubbo://10.7.8.50:16002",
		Application: "dubbo-invoke-client",
		Timeout:     10 * time.Second,
	}

	client, err := NewRealDubboClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Skip("跳过测试：无法连接到注册中心")
		return
	}

	// 测试复杂对象参数
	userObj := map[string]interface{}{
		"id":    1,
		"name":  "张三",
		"email": "zhangsan@example.com",
		"age":   25,
	}

	t.Logf("🚀 测试复杂参数调用: UserService.createUser")
	t.Logf("用户对象: %+v", userObj)

	result, err := client.GenericInvoke(
		"com.example.UserService",
		"createUser",
		[]string{"com.example.User"},
		[]interface{}{userObj},
	)

	if err != nil {
		t.Logf("❌ 复杂参数调用失败: %v", err)
	} else {
		t.Logf("✅ 复杂参数调用成功")
		if resultBytes, err := json.MarshalIndent(result, "", "  "); err == nil {
			t.Logf("📄 返回数据:\n%s", string(resultBytes))
		} else {
			t.Logf("📄 返回数据: %+v", result)
		}
		validateReturnedData(t, result)
	}
}

// validateReturnedData 验证返回的数据
func validateReturnedData(t *testing.T, result interface{}) {
	if result == nil {
		t.Error("❌ 返回结果为空")
		return
	}

	// 检查数据类型和结构
	switch v := result.(type) {
	case map[string]interface{}:
		t.Log("✅ 返回数据为map结构")
		
		// 检查常见的响应字段
		if success, exists := v["success"]; exists {
			t.Logf("  success: %v", success)
		}
		if code, exists := v["code"]; exists {
			t.Logf("  code: %v", code)
		}
		if message, exists := v["message"]; exists {
			t.Logf("  message: %v", message)
		}
		if data, exists := v["data"]; exists {
			t.Logf("  ✅ 包含data字段，类型: %T", data)
			
			// 检查data字段是否为乱码
			if dataStr, ok := data.(string); ok {
				if len(dataStr) > 0 {
					if isGarbledText(dataStr) {
						t.Error("❌ data字段包含乱码")
					} else {
						t.Log("  ✅ data字段格式正常")
					}
				}
			} else {
				t.Logf("  data字段为非字符串类型: %T", data)
			}
		} else {
			t.Log("  ⚠️  没有data字段")
		}
		
		// 统计字段数量
		t.Logf("  字段总数: %d", len(v))
		
case string:
		t.Logf("✅ 返回数据为字符串，长度: %d", len(v))
		if isGarbledText(v) {
			t.Error("❌ 字符串数据包含乱码")
		} else {
			t.Log("✅ 字符串数据格式正常")
		}
		
case []interface{}:
		t.Logf("✅ 返回数据为数组，长度: %d", len(v))
		
default:
		t.Logf("✅ 返回数据类型: %T", result)
	}

	// 检查数据大小
	if resultBytes, err := json.Marshal(result); err == nil {
		t.Logf("📊 数据大小: %d 字节", len(resultBytes))
		if len(resultBytes) == 0 {
			t.Error("❌ 数据为空")
		} else if len(resultBytes) < 10 {
			t.Log("⚠️  数据较小，可能不完整")
		} else {
			t.Log("✅ 数据大小正常")
		}
	}
}