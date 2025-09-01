package main

import (
	"fmt"
	"testing"
	"time"
)

// TestRealInvokeGetCompanyInfo 测试真实的invoke调用
func TestRealInvokeGetCompanyInfo(t *testing.T) {
	// 创建配置
	cfg := &DubboConfig{
		Registry:    "zookeeper://127.0.0.1:2181",
		Application: "dubbo-invoke-client",
		Timeout:     30 * time.Second,
	}

	// 创建真实Dubbo客户端
	realClient, err := NewRealDubboClient(cfg)
	if err != nil {
		t.Fatalf("创建真实Dubbo客户端失败: %v", err)
	}
	defer realClient.Close()

	// 调用getCompanyInfo方法
	serviceName := "com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi"
	methodName := "getCompanyInfo"
	params := []interface{}{1}
	paramTypes := []string{"java.lang.Integer"}

	fmt.Printf("开始调用: %s.%s(%v)\n", serviceName, methodName, params)

	start := time.Now()
	result, err := realClient.GenericInvoke(serviceName, methodName, paramTypes, params)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("调用失败: %v", err)
	}

	fmt.Printf("调用成功，耗时: %v\n", elapsed)
	fmt.Printf("返回结果: %v\n", result)

	// 验证返回结果不为空
	if result == nil {
		t.Fatal("返回结果为空")
	}

	// 尝试解析返回的JSON数据
	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("返回结果不是字符串类型: %T", result)
	}

	// 检查是否包含预期的字段
	if len(resultStr) == 0 {
		t.Fatal("返回结果为空字符串")
	}

	fmt.Printf("✅ 真实invoke调用测试通过\n")
}

// TestRealInvokeWithWebInterface 测试Web接口的真实invoke调用
func TestRealInvokeWithWebInterface(t *testing.T) {
	// 模拟Web接口的调用请求
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

	fmt.Printf("开始Web接口调用: %s.%s(%v)\n", req.ServiceName, req.MethodName, req.Parameters)

	start := time.Now()
	result, err := ws.executeInvoke(req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Web接口调用失败: %v", err)
	}

	fmt.Printf("Web接口调用成功，耗时: %v\n", elapsed)
	fmt.Printf("返回结果: %v\n", result)

	// 验证返回结果
	if result == nil {
		t.Fatal("返回结果为空")
	}

	fmt.Printf("✅ Web接口真实invoke调用测试通过\n")
}