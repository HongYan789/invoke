package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// APITestRequest 表示API测试请求
type APITestRequest struct {
	Registry  string `json:"registry"`
	Namespace string `json:"namespace"`
	App       string `json:"app"`
}

// APITestResponse 表示API测试响应
type APITestResponse struct {
	Success  bool     `json:"success"`
	Services []string `json:"services"`
	Error    string   `json:"error"`
}

// TestAPIEndpoint 测试API端点
func TestAPIEndpoint() {
	fmt.Println("=== API端点测试开始 ===")
	
	// 构建测试请求
	request := APITestRequest{
		Registry:  "nacos://yjj-nacos.it.yyjzt.com:28848",
		Namespace: "dev",
		App:       "dubbo-invoke-cli",
	}
	
	// 序列化请求数据
	requestData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("❌ 序列化请求失败: %v\n", err)
		return
	}
	
	fmt.Printf("📤 发送请求到: http://localhost:8080/api/list\n")
	fmt.Printf("📋 请求数据: %s\n", string(requestData))
	
	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// 发送POST请求
	resp, err := client.Post(
		"http://localhost:8080/api/list",
		"application/json",
		bytes.NewBuffer(requestData),
	)
	if err != nil {
		fmt.Printf("❌ 发送请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("📥 响应状态码: %d\n", resp.StatusCode)
	
	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}
	
	fmt.Printf("📄 原始响应: %s\n", string(body))
	
	// 解析响应
	var response APITestResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Printf("❌ 解析响应失败: %v\n", err)
		return
	}
	
	// 分析测试结果
	if response.Success {
		fmt.Printf("✅ API调用成功!\n")
		fmt.Printf("📊 发现服务数量: %d\n", len(response.Services))
		if len(response.Services) > 0 {
			fmt.Println("📋 服务列表:")
			for i, service := range response.Services {
				fmt.Printf("  %d. %s\n", i+1, service)
			}
		} else {
			fmt.Println("⚠️  当前命名空间中没有发现服务")
		}
	} else {
		fmt.Printf("❌ API调用失败: %s\n", response.Error)
	}
	
	fmt.Println("=== API端点测试完成 ===")
}

// main 主函数
func main() {
	fmt.Println("🧪 开始API端点测试")
	fmt.Println("⏰ 测试时间:", time.Now().Format("2006-01-02 15:04:05"))
	
	// 等待一下确保服务器启动
	fmt.Println("⏳ 等待2秒确保服务器就绪...")
	time.Sleep(2 * time.Second)
	
	// 执行API端点测试
	TestAPIEndpoint()
	
	fmt.Println("\n🎯 测试总结:")
	fmt.Println("1. 如果API测试成功，说明Web服务器和Nacos连接都正常")
	fmt.Println("2. 如果API测试失败，需要检查错误信息进行调试")
	fmt.Println("\n✨ 测试完成!")
}