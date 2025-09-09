package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// ZKTestRequest 表示ZooKeeper测试请求
type ZKTestRequest struct {
	Registry  string `json:"registry"`
	Namespace string `json:"namespace"`
	App       string `json:"app"`
}

// ZKTestResponse 表示ZooKeeper测试响应
type ZKTestResponse struct {
	Success  bool     `json:"success"`
	Services []string `json:"services"`
	Error    string   `json:"error"`
}

// TestZooKeeperConnection 测试ZooKeeper连接和服务发现
func TestZooKeeperConnection() {
	fmt.Println("=== ZooKeeper服务发现测试开始 ===")
	
	// 构建ZooKeeper测试请求
	request := ZKTestRequest{
		Registry:  "zookeeper://10.7.8.40:2181",
		Namespace: "", // ZooKeeper通常不使用命名空间
		App:       "dubbo-invoke-client",
	}
	
	// 序列化请求数据
	requestData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("❌ 序列化请求失败: %v\n", err)
		return
	}
	
	fmt.Printf("📤 发送请求到: http://localhost:8080/api/list\n")
	fmt.Printf("🔗 ZooKeeper地址: %s\n", request.Registry)
	fmt.Printf("📋 请求数据: %s\n", string(requestData))
	
	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 60 * time.Second, // ZooKeeper连接可能需要更长时间
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
	var response ZKTestResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Printf("❌ 解析响应失败: %v\n", err)
		return
	}
	
	// 分析测试结果
	if response.Success {
		fmt.Printf("✅ ZooKeeper连接成功!\n")
		fmt.Printf("📊 发现服务数量: %d\n", len(response.Services))
		if len(response.Services) > 0 {
			fmt.Println("📋 ZooKeeper中的服务列表:")
			for i, service := range response.Services {
				fmt.Printf("  %d. %s\n", i+1, service)
			}
			fmt.Println("\n🎉 ZooKeeper服务发现功能正常!")
		} else {
			fmt.Println("⚠️  ZooKeeper中当前没有注册的服务")
			fmt.Println("💡 这可能是正常情况，如果没有服务注册到该ZooKeeper实例")
		}
	} else {
		fmt.Printf("❌ ZooKeeper连接失败: %s\n", response.Error)
		fmt.Println("\n🔍 可能的原因:")
		fmt.Println("  1. ZooKeeper服务器不可达 (10.7.8.40:2181)")
		fmt.Println("  2. 网络连接问题")
		fmt.Println("  3. ZooKeeper服务未启动")
		fmt.Println("  4. 防火墙阻止连接")
	}
	
	fmt.Println("=== ZooKeeper服务发现测试完成 ===")
}

// TestMockVsRealZK 对比Mock数据和真实ZooKeeper的差异
func TestMockVsRealZK() {
	fmt.Println("\n=== Mock数据 vs 真实ZooKeeper对比测试 ===")
	
	// 测试Mock数据 (默认zookeeper://127.0.0.1:2181)
	fmt.Println("\n🔸 测试Mock数据 (本地ZooKeeper):")
	mockRequest := ZKTestRequest{
		Registry:  "zookeeper://127.0.0.1:2181",
		Namespace: "",
		App:       "dubbo-invoke-client",
	}
	testConnection(mockRequest, "Mock数据")
	
	// 测试真实ZooKeeper
	fmt.Println("\n🔸 测试真实ZooKeeper (10.7.8.40:2181):")
	realRequest := ZKTestRequest{
		Registry:  "zookeeper://10.7.8.40:2181",
		Namespace: "",
		App:       "dubbo-invoke-client",
	}
	testConnection(realRequest, "真实ZooKeeper")
}

// testConnection 通用连接测试函数
func testConnection(request ZKTestRequest, testType string) {
	requestData, _ := json.Marshal(request)
	client := &http.Client{Timeout: 30 * time.Second}
	
	resp, err := client.Post(
		"http://localhost:8080/api/list",
		"application/json",
		bytes.NewBuffer(requestData),
	)
	if err != nil {
		fmt.Printf("❌ %s连接失败: %v\n", testType, err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	var response ZKTestResponse
	json.Unmarshal(body, &response)
	
	if response.Success {
		fmt.Printf("✅ %s连接成功，发现 %d 个服务\n", testType, len(response.Services))
	} else {
		fmt.Printf("❌ %s连接失败: %s\n", testType, response.Error)
	}
}

// main 主函数
func main() {
	fmt.Println("🧪 ZooKeeper服务发现功能测试")
	fmt.Println("⏰ 测试时间:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("🎯 目标: 验证是否能正常连接真实ZooKeeper并加载服务列表")
	
	// 等待确保服务器启动
	fmt.Println("⏳ 等待3秒确保Web服务器就绪...")
	time.Sleep(3 * time.Second)
	
	// 执行ZooKeeper连接测试
	TestZooKeeperConnection()
	
	// 执行对比测试
	TestMockVsRealZK()
	
	fmt.Println("\n🎯 测试总结:")
	fmt.Println("1. 如果真实ZooKeeper测试成功，说明可以正常连接并获取服务列表")
	fmt.Println("2. 如果测试失败，请检查网络连接和ZooKeeper服务状态")
	fmt.Println("3. 对比Mock数据可以验证项目的ZooKeeper集成是否正常工作")
	fmt.Println("\n✨ 测试完成!")
}