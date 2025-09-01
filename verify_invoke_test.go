package main

import (
	"fmt"
	"testing"
	"time"
)

// TestVerifyInvokeGetCompanyInfo 验证invoke com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi.getCompanyInfo(1)调用
func TestVerifyInvokeGetCompanyInfo(t *testing.T) {
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

	// 调用getCompanyInfo方法
	serviceName := "com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi"
	methodName := "getCompanyInfo"
	paramTypes := []string{"int"}
	params := []interface{}{1}

	fmt.Printf("=== 验证调用 ===\n")
	fmt.Printf("服务: %s\n", serviceName)
	fmt.Printf("方法: %s\n", methodName)
	fmt.Printf("参数: %v\n", params)
	fmt.Printf("参数类型: %v\n", paramTypes)
	fmt.Printf("超时时间: %v\n", config.Timeout)
	fmt.Println("")

	// 记录开始时间
	startTime := time.Now()

	// 执行调用
	result, err := client.GenericInvoke(serviceName, methodName, paramTypes, params)

	// 记录结束时间
	elapsed := time.Since(startTime)
	fmt.Printf("调用耗时: %v\n", elapsed)

	if err != nil {
		fmt.Printf("❌ 调用失败: %v\n", err)
		t.Fatalf("调用失败: %v", err)
	}

	fmt.Printf("✅ 调用成功!\n")
	fmt.Printf("返回结果: %v\n", result)

	// 验证返回数据
	if result != nil {
		resultStr := fmt.Sprintf("%v", result)
		if len(resultStr) > 0 {
			fmt.Printf("✅ 成功获取到数据，长度: %d 字符\n", len(resultStr))
			
			// 检查是否包含业务数据字段
			businessFields := []string{"companyId", "companyName", "companyInfo"}
			foundFields := 0
			for _, field := range businessFields {
				if containsField(resultStr, field) {
					foundFields++
					fmt.Printf("  ✓ 包含字段: %s\n", field)
				}
			}
			
			if foundFields > 0 {
				fmt.Printf("✅ 数据验证通过，包含 %d 个业务字段\n", foundFields)
			} else {
				fmt.Printf("⚠️  未检测到预期的业务字段\n")
			}
		} else {
			fmt.Printf("⚠️  返回数据为空\n")
		}
	} else {
		fmt.Printf("❌ 返回结果为nil\n")
	}
}

// containsField 检查字符串是否包含指定字段
func containsField(str, field string) bool {
	return len(str) >= len(field) && 
		   func() bool {
			   for i := 0; i <= len(str)-len(field); i++ {
				   if str[i:i+len(field)] == field {
					   return true
				   }
			   }
			   return false
		   }()
}