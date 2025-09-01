package main

import (
	"fmt"
	"testing"
	"time"
)

// TestRealServiceList 测试获取真实服务列表
func TestRealServiceList(t *testing.T) {
	config := &DubboConfig{
		Registry:    "zookeeper://10.7.8.50:16002",
		Application: "dubbo-invoke-client",
		Timeout:     30 * time.Second,
	}

	// 创建真实的dubbo客户端
	client, err := NewRealDubboClient(config)
	if err != nil {
		t.Fatalf("创建dubbo客户端失败: %v", err)
	}
	defer client.Close()

	// 检查连接状态
	if !client.IsConnected() {
		t.Fatal("无法连接到dubbo注册中心")
	}

	// 获取服务列表
	services, err := client.ListServices()
	if err != nil {
		t.Fatalf("获取服务列表失败: %v", err)
	}

	fmt.Printf("获取到 %d 个服务:\n", len(services))
	for i, service := range services {
		fmt.Printf("%d. %s\n", i+1, service)
	}

	// 验证是否包含目标服务
	targetService := "com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi"
	found := false
	for _, service := range services {
		if service == targetService {
			found = true
			break
		}
	}

	if found {
		fmt.Printf("✅ 找到目标服务: %s\n", targetService)
	} else {
		fmt.Printf("❌ 未找到目标服务: %s\n", targetService)
		fmt.Println("可用服务列表:")
		for _, service := range services {
			fmt.Printf("  - %s\n", service)
		}
	}
}

// TestRealBusinessInvoke 测试真实业务接口调用
func TestRealBusinessInvoke(t *testing.T) {
	config := &DubboConfig{
		Registry:    "zookeeper://10.7.8.50:16002",
		Application: "dubbo-invoke-client",
		Timeout:     30 * time.Second,
	}

	// 创建真实的dubbo客户端
	client, err := NewRealDubboClient(config)
	if err != nil {
		t.Fatalf("创建dubbo客户端失败: %v", err)
	}
	defer client.Close()

	// 检查连接状态
	if !client.IsConnected() {
		t.Fatal("无法连接到dubbo注册中心")
	}

	// 调用真实的业务接口
	serviceName := "com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi"
	methodName := "getCompanyInfoByCompanyId"
	paramTypes := []string{"int"}
	params := []interface{}{1}

	fmt.Printf("调用服务: %s\n", serviceName)
	fmt.Printf("调用方法: %s\n", methodName)
	fmt.Printf("调用参数: %v\n", params)
	fmt.Printf("参数类型: %v\n", paramTypes)

	result, err := client.GenericInvoke(serviceName, methodName, paramTypes, params)
	if err != nil {
		t.Fatalf("调用业务接口失败: %v", err)
	}

	fmt.Printf("调用结果: %v\n", result)

	// 验证返回数据
	if result != nil {
		fmt.Println("✅ 成功获取到业务数据")
		
		// 检查返回数据是否包含预期字段
		if resultStr, ok := result.(string); ok {
			if validateRealBusinessData(resultStr) {
				fmt.Println("✅ 返回数据格式验证通过")
			} else {
				fmt.Println("❌ 返回数据格式验证失败")
			}
		}
	} else {
		fmt.Println("❌ 未获取到业务数据")
	}
}

// validateRealBusinessData 验证真实业务数据格式
func validateRealBusinessData(data string) bool {
	// 检查是否包含公司信息相关字段
	businessFields := []string{
		"companyId",
		"companyName", 
		"companyInfo",
		"bigCompanyLabel",
		"createTime",
	}
	
	validFieldCount := 0
	for _, field := range businessFields {
		if containsRealSubstring(data, field) {
			validFieldCount++
		}
	}
	
	// 如果包含至少2个业务字段，认为是有效的业务数据
	return validFieldCount >= 2
}

// containsRealSubstring 检查字符串是否包含子字符串
func containsRealSubstring(str, substr string) bool {
	return len(str) >= len(substr) && 
		   func() bool {
			   for i := 0; i <= len(str)-len(substr); i++ {
				   if str[i:i+len(substr)] == substr {
					   return true
				   }
			   }
			   return false
		   }()
}