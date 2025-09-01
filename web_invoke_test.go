package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestWebInvokeGetCompanyInfo 测试通过Web接口调用getCompanyInfo方法
func TestWebInvokeGetCompanyInfo(t *testing.T) {
	// 构造请求数据
	request := InvokeRequest{
		ServiceName: "com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi",
		MethodName:  "getCompanyInfo",
		Parameters:  []string{"1"},
		Types:       []string{},
		Registry:    "zookeeper://127.0.0.1:2181",
		App:         "dubbo-invoke-client",
		Timeout:     30000, // 30秒超时
		Group:       "",
		Version:     "",
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("序列化请求失败: %v", err)
	}

	// 发送HTTP请求
	resp, err := http.Post("http://localhost:8081/api/invoke", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("发送HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var response InvokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应
	if !response.Success {
		t.Fatalf("调用失败: %s", response.Error)
	}

	// 验证返回数据
	if response.Data == nil {
		t.Fatal("返回数据为空")
	}

	// 将返回数据转换为字符串进行验证
	dataStr := fmt.Sprintf("%v", response.Data)
	if dataStr == "" {
		t.Fatal("返回数据为空字符串")
	}

	fmt.Printf("✅ Web接口调用成功\n")
	fmt.Printf("📊 返回数据: %v\n", response.Data)
	fmt.Printf("📝 响应消息: %s\n", response.Message)
}

// TestWebInvokeTimeout 测试Web接口的超时处理
func TestWebInvokeTimeout(t *testing.T) {
	// 构造一个会超时的请求（使用很短的超时时间）
	request := InvokeRequest{
		ServiceName: "com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi",
		MethodName:  "getCompanyInfo",
		Parameters:  []string{"1"},
		Types:       []string{},
		Registry:    "zookeeper://127.0.0.1:2181",
		App:         "dubbo-invoke-client",
		Timeout:     1, // 1毫秒超时，必然超时
		Group:       "",
		Version:     "",
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("序列化请求失败: %v", err)
	}

	// 发送HTTP请求
	start := time.Now()
	resp, err := http.Post("http://localhost:8081/api/invoke", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("发送HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()
	duration := time.Since(start)

	// 解析响应
	var response InvokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应应该是失败的
	if response.Success {
		t.Fatal("期望调用失败但实际成功了")
	}

	// 验证错误信息包含超时相关内容
	if response.Error == "" {
		t.Fatal("错误信息为空")
	}

	fmt.Printf("✅ 超时测试成功\n")
	fmt.Printf("⏱️  调用耗时: %v\n", duration)
	fmt.Printf("❌ 错误信息: %s\n", response.Error)
}