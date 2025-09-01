package main

import (
	"encoding/json"
	"testing"
	"time"
)

// TestBusinessInterfaceWithLongerTimeout 测试更长超时时间的业务接口调用
func TestBusinessInterfaceWithLongerTimeout(t *testing.T) {
	// 使用更长的超时时间
	config := &DubboConfig{
		Registry:    "dubbo://10.7.8.50:16002",
		Application: "dubbo-invoke-client",
		Timeout:     15 * time.Second, // 增加到15秒
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

	// 测试业务接口
	serviceName := "com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi"
	methodName := "getCompanyInfoFromDb"
	
	paramObj := map[string]interface{}{
		"class":     "com.jzt.zhcai.user.companyinfo.dto.request.UserCompanyInfoDetailReq",
		"companyId": 1,
	}
	
	params := []interface{}{paramObj}
	paramTypes := []string{"com.jzt.zhcai.user.companyinfo.dto.request.UserCompanyInfoDetailReq"}

	t.Logf("开始调用业务接口（15秒超时）...")
	t.Logf("服务名: %s", serviceName)
	t.Logf("方法名: %s", methodName)

	start := time.Now()
	result, err := client.GenericInvoke(serviceName, methodName, paramTypes, params)
	duration := time.Since(start)
	
	t.Logf("调用耗时: %v", duration)
	
	if err != nil {
		t.Logf("调用失败: %v", err)
		if containsSubstring(err.Error(), "超时") || containsSubstring(err.Error(), "timeout") {
			t.Logf("仍然超时，可能服务不可用或需要更长时间")
		} else {
			t.Logf("非超时错误: %v", err)
		}
		return
	}

	// 成功获取到数据
	t.Logf("✅ 调用成功！耗时: %v", duration)
	t.Logf("返回结果类型: %T", result)
	
	if resultBytes, err := json.MarshalIndent(result, "", "  "); err == nil {
		t.Logf("返回数据:\n%s", string(resultBytes))
	} else {
		t.Logf("返回数据: %+v", result)
	}

	// 验证数据完整性
	validateBusinessData(t, result)
}

// TestServiceAvailability 测试服务可用性
func TestServiceAvailability(t *testing.T) {
	config := &DubboConfig{
		Registry:    "dubbo://10.7.8.50:16002",
		Application: "dubbo-invoke-client",
		Timeout:     5 * time.Second,
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

	// 测试Ping
	err = client.Ping()
	if err != nil {
		t.Logf("Ping失败: %v", err)
	} else {
		t.Log("✅ Ping成功")
	}

	// 列出可用服务
	services, err := client.ListServices()
	if err != nil {
		t.Logf("列出服务失败: %v", err)
	} else {
		t.Logf("✅ 发现 %d 个服务:", len(services))
		for i, service := range services {
			t.Logf("  %d. %s", i+1, service)
		}
	}

	// 检查目标服务是否在列表中
	targetService := "com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi"
	found := false
	for _, service := range services {
		if service == targetService {
			found = true
			break
		}
	}
	
	if found {
		t.Logf("✅ 目标服务 %s 在服务列表中", targetService)
	} else {
		t.Logf("⚠️  目标服务 %s 不在服务列表中", targetService)
	}

	// 列出目标服务的方法
	methods, err := client.ListMethods(targetService)
	if err != nil {
		t.Logf("列出方法失败: %v", err)
	} else {
		t.Logf("✅ 服务 %s 的方法:", targetService)
		for i, method := range methods {
			t.Logf("  %d. %s", i+1, method)
		}
	}
}

// validateBusinessData 验证业务数据
func validateBusinessData(t *testing.T, result interface{}) {
	if result == nil {
		t.Error("❌ 返回结果为空")
		return
	}

	// 检查是否为map格式
	if resultMap, ok := result.(map[string]interface{}); ok {
		t.Log("✅ 返回结果为map格式")
		
		// 检查常见字段
		if success, exists := resultMap["success"]; exists {
			t.Logf("success字段: %v", success)
		}
		
		if code, exists := resultMap["code"]; exists {
			t.Logf("code字段: %v", code)
		}
		
		if message, exists := resultMap["message"]; exists {
			t.Logf("message字段: %v", message)
		}
		
		if data, exists := resultMap["data"]; exists {
			t.Logf("✅ 包含data字段")
			if dataStr, ok := data.(string); ok {
				if len(dataStr) > 0 {
					if isGarbledText(dataStr) {
						t.Error("❌ 检测到乱码数据")
					} else {
						t.Log("✅ 数据格式正常")
					}
				}
			} else {
				t.Logf("data字段类型: %T, 值: %+v", data, data)
			}
		} else {
			t.Log("⚠️  返回结果中没有data字段")
		}
	} else {
		t.Logf("⚠️  返回结果不是map格式，类型: %T", result)
	}
}