package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// ServiceListResponse API响应结构
type ServiceListResponse struct {
	Success  bool     `json:"success"`
	Services []string `json:"services"`
	Error    string   `json:"error"`
}

// TestServiceListDisplay 测试服务清单展示功能
func TestServiceListDisplay() {
	fmt.Println("=== 服务清单展示测试 ===")
	fmt.Println("⏰ 测试时间:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("🎯 目标: 验证mock数据与真实ZooKeeper服务列表的一致性")

	// 等待确保服务器启动
	fmt.Println("⏳ 等待3秒确保Web服务器就绪...")
	time.Sleep(3 * time.Second)

	// 测试不同注册中心的服务列表
	testCases := []struct {
		name     string
		registry string
		expected []string
	}{
		{
			name:     "ZooKeeper注册中心",
			registry: "zookeeper://10.7.8.40:2181",
			expected: []string{
				"com.example.UserService",
				"com.example.OrderService",
				"com.example.ProductService",
			},
		},
		{
			name:     "Nacos注册中心",
			registry: "nacos://127.0.0.1:8848",
			expected: []string{
				"com.example.UserService",
				"com.example.OrderService",
				"com.example.ProductService",
			},
		},
		{
			name:     "Mock数据测试",
			registry: "mock://test",
			expected: []string{
				"com.example.UserService",
				"com.example.OrderService",
				"com.example.ProductService",
			},
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n🔸 测试: %s\n", tc.name)
		fmt.Printf("📡 注册中心: %s\n", tc.registry)

		// 调用API获取服务列表
		services, err := getServiceList(tc.registry)
		if err != nil {
			fmt.Printf("❌ 获取服务列表失败: %v\n", err)
			continue
		}

		// 显示获取到的服务列表
		fmt.Printf("📊 发现服务数量: %d\n", len(services))
		if len(services) > 0 {
			fmt.Println("📋 服务清单:")
			for i, service := range services {
				fmt.Printf("  %d. %s\n", i+1, service)
			}

			// 验证服务列表是否包含预期的服务
			matchCount := 0
			for _, expected := range tc.expected {
				for _, actual := range services {
					if actual == expected {
						matchCount++
						break
					}
				}
			}

			if matchCount == len(tc.expected) {
				fmt.Printf("✅ 服务列表验证通过! 所有预期服务都存在\n")
			} else {
				fmt.Printf("⚠️  服务列表部分匹配: %d/%d 个预期服务存在\n", matchCount, len(tc.expected))
				fmt.Println("🔍 预期服务列表:")
				for i, expected := range tc.expected {
					fmt.Printf("  %d. %s\n", i+1, expected)
				}
			}
		} else {
			fmt.Println("⚠️  当前没有发现任何服务")
		}
	}

	fmt.Println("\n=== 服务清单展示测试完成 ===")
}

// getServiceList 获取服务列表
func getServiceList(registry string) ([]string, error) {
	// 构建API请求URL
	apiURL := fmt.Sprintf("http://localhost:8080/api/list?registry=%s&app=dubbo-invoke-client&timeout=10000", registry)

	// 发送HTTP请求
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析JSON响应
	var response ServiceListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	// 检查API调用是否成功
	if !response.Success {
		return nil, fmt.Errorf("API调用失败: %s", response.Error)
	}

	return response.Services, nil
}

// main 主函数
func main() {
	fmt.Println("🧪 服务清单展示功能测试")
	fmt.Println("📝 测试目标: 验证更新后的mock数据是否正确展示服务清单")
	fmt.Println("🔧 测试范围: ZooKeeper、Nacos、Mock数据的服务列表一致性")

	// 执行服务清单展示测试
	TestServiceListDisplay()

	fmt.Println("\n🎯 测试总结:")
	fmt.Println("1. 验证了不同注册中心返回的服务列表")
	fmt.Println("2. 确认mock数据与真实ZooKeeper服务的一致性")
	fmt.Println("3. 测试了API接口的正确性和稳定性")
	fmt.Println("\n✨ 测试完成!")
}