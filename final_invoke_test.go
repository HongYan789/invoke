package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestFinalInvokeGetCompanyInfo 最终验证用户要求的invoke调用
func TestFinalInvokeGetCompanyInfo(t *testing.T) {
	fmt.Println("🎯 开始最终验证：invoke com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi.getCompanyInfo(1)")
	
	// 创建配置
	cfg := &DubboConfig{
		Registry:    "zookeeper://127.0.0.1:2181",
		Application: "dubbo-invoke-client",
		Timeout:     30 * time.Second,
	}

	// 创建真实Dubbo客户端
	realClient, err := NewRealDubboClient(cfg)
	if err != nil {
		t.Fatalf("❌ 创建真实Dubbo客户端失败: %v", err)
	}
	defer realClient.Close()

	// 用户要求的具体调用
	serviceName := "com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi"
	methodName := "getCompanyInfo"
	params := []interface{}{1}
	paramTypes := []string{"java.lang.Integer"}

	fmt.Printf("📞 正在调用: %s.%s(%v)\n", serviceName, methodName, params)
	fmt.Printf("⏱️  超时设置: %v\n", cfg.Timeout)

	start := time.Now()
	result, err := realClient.GenericInvoke(serviceName, methodName, paramTypes, params)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("❌ 调用失败: %v", err)
	}

	fmt.Printf("✅ 调用成功！耗时: %v\n", elapsed)
	fmt.Printf("📊 返回结果类型: %T\n", result)
	fmt.Printf("📄 返回内容: %v\n", result)

	// 验证返回结果
	if result == nil {
		t.Fatal("❌ 返回结果为空")
	}

	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("❌ 返回结果不是字符串类型: %T", result)
	}

	if len(resultStr) == 0 {
		t.Fatal("❌ 返回结果为空字符串")
	}

	// 检查是否包含业务数据字段
	expectedFields := []string{"companyId", "companyName", "success"}
	for _, field := range expectedFields {
		if !strings.Contains(resultStr, field) {
			t.Errorf("❌ 返回结果缺少字段: %s", field)
		} else {
			fmt.Printf("✓ 包含字段: %s\n", field)
		}
	}

	fmt.Println("🎉 最终验证通过！invoke调用成功返回业务数据")
}

// TestFinalWebInvokeGetCompanyInfo 最终验证Web接口的invoke调用
func TestFinalWebInvokeGetCompanyInfo(t *testing.T) {
	fmt.Println("🌐 开始最终验证：Web接口invoke调用")
	
	// 模拟用户在Web界面的调用请求
	req := InvokeRequest{
		ServiceName: "com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi",
		MethodName:  "getCompanyInfo",
		Parameters:  []string{"1"},
		Types:       []string{"java.lang.Integer"},
		Registry:    "zookeeper://127.0.0.1:2181",
		App:         "dubbo-invoke-client",
		Timeout:     30000, // 30秒
	}

	// 创建WebServer实例
	ws := &WebServer{
		port:     8081,
		registry: "zookeeper://127.0.0.1:2181",
		app:      "dubbo-invoke-client",
		timeout:  30000,
	}

	fmt.Printf("📞 Web接口调用: %s.%s(%v)\n", req.ServiceName, req.MethodName, req.Parameters)

	start := time.Now()
	result, err := ws.executeInvoke(req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("❌ Web接口调用失败: %v", err)
	}

	fmt.Printf("✅ Web接口调用成功！耗时: %v\n", elapsed)
	fmt.Printf("📊 返回结果: %v\n", result)

	// 验证返回结果
	if result == nil {
		t.Fatal("❌ Web接口返回结果为空")
	}

	fmt.Println("🎉 Web接口最终验证通过！")
}