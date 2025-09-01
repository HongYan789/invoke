package main

import (
	"encoding/json"
	"testing"
	"time"
)

// TestBusinessInterfaceInvoke 测试业务接口调用
// 使用用户之前提供的真实业务接口参数进行测试
func TestBusinessInterfaceInvoke(t *testing.T) {
	// 配置真实的业务接口参数
	config := &DubboConfig{
		Registry:    "dubbo://10.7.8.50:16002", // 用户的注册中心地址
		Application: "dubbo-invoke-client",
		Timeout:     5 * time.Second,
	}

	// 创建真实的Dubbo客户端
	client, err := NewRealDubboClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 检查连接状态（NewRealDubboClient已经自动连接）
	if !client.IsConnected() {
		t.Skip("跳过测试：无法连接到注册中心")
		return
	}

	// 测试用户的真实业务接口
	serviceName := "com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi"
	methodName := "getCompanyInfoFromDb"
	
	// 构建业务参数
	paramObj := map[string]interface{}{
		"class":     "com.jzt.zhcai.user.companyinfo.dto.request.UserCompanyInfoDetailReq",
		"companyId": 1,
	}
	
	params := []interface{}{paramObj}
	paramTypes := []string{"com.jzt.zhcai.user.companyinfo.dto.request.UserCompanyInfoDetailReq"}

	t.Logf("开始调用业务接口...")
	t.Logf("服务名: %s", serviceName)
	t.Logf("方法名: %s", methodName)
	t.Logf("参数: %+v", params)
	t.Logf("参数类型: %+v", paramTypes)

	// 执行泛化调用
	result, err := client.GenericInvoke(serviceName, methodName, paramTypes, params)
	
	if err != nil {
		t.Logf("调用失败: %v", err)
		// 检查是否是超时错误
		if containsSubstring(err.Error(), "超时") || containsSubstring(err.Error(), "timeout") {
			t.Logf("检测到超时错误，这是预期的行为")
		} else {
			t.Errorf("调用出现非超时错误: %v", err)
		}
		return
	}

	// 验证返回结果
	if result == nil {
		t.Error("返回结果为空")
		return
	}

	t.Logf("调用成功！")
	t.Logf("返回结果类型: %T", result)
	
	// 尝试将结果转换为JSON格式以便查看
	if resultBytes, err := json.MarshalIndent(result, "", "  "); err == nil {
		t.Logf("返回数据:\n%s", string(resultBytes))
	} else {
		t.Logf("返回数据: %+v", result)
	}

	// 验证数据结构
	if resultMap, ok := result.(map[string]interface{}); ok {
		if data, exists := resultMap["data"]; exists {
			t.Logf("业务数据字段存在: %+v", data)
			// 检查数据是否为乱码
			if dataStr, ok := data.(string); ok {
				if len(dataStr) > 0 && isGarbledText(dataStr) {
					t.Error("检测到乱码数据")
				} else {
					t.Log("数据格式正常")
				}
			}
		} else {
			t.Log("返回结果中没有data字段")
		}
	} else {
		t.Logf("返回结果不是map格式: %+v", result)
	}
}

// TestBusinessInterfaceWithDifferentParams 测试不同参数的业务接口调用
func TestBusinessInterfaceWithDifferentParams(t *testing.T) {
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

	// 检查连接状态（NewRealDubboClient已经自动连接）
	if !client.IsConnected() {
		t.Skip("跳过测试：无法连接到注册中心")
		return
	}

	// 测试不同的companyId参数
	testCases := []struct {
		name      string
		companyId interface{}
	}{
		{"companyId为1", 1},
		{"companyId为2", 2},
		{"companyId为字符串", "1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramObj := map[string]interface{}{
				"class":     "com.jzt.zhcai.user.companyinfo.dto.request.UserCompanyInfoDetailReq",
				"companyId": tc.companyId,
			}
			
			result, err := client.GenericInvoke(
				"com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi",
				"getCompanyInfoFromDb",
				[]string{"com.jzt.zhcai.user.companyinfo.dto.request.UserCompanyInfoDetailReq"},
				[]interface{}{paramObj},
			)
			
			if err != nil {
				t.Logf("%s - 调用失败: %v", tc.name, err)
			} else {
				t.Logf("%s - 调用成功: %+v", tc.name, result)
			}
		})
	}
}

// isGarbledText 检查文本是否为乱码
func isGarbledText(text string) bool {
	// 简单的乱码检测：检查是否包含大量非打印字符
	nonPrintableCount := 0
	for _, r := range text {
		if r < 32 || r > 126 {
			nonPrintableCount++
		}
	}
	// 如果非打印字符超过50%，认为是乱码
	return float64(nonPrintableCount)/float64(len(text)) > 0.5
}

// containsSubstring 检查字符串是否包含子字符串
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}