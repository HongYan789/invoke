//go:build ignore
// +build ignore

// 简单的构建脚本，用于测试基本功能
package main

import (
	"fmt"
	"os"
	"time"
)

// 简化的配置结构
type SimpleConfig struct {
	Registry    string
	Application string
	Timeout     time.Duration
}

// 简化的客户端结构
type SimpleClient struct {
	config    *SimpleConfig
	connected bool
}

// 创建简单客户端
func NewSimpleClient(config *SimpleConfig) *SimpleClient {
	return &SimpleClient{
		config:    config,
		connected: false,
	}
}

// 连接
func (c *SimpleClient) Connect() error {
	fmt.Printf("连接到注册中心: %s\n", c.config.Registry)
	c.connected = true
	return nil
}

// 调用服务
func (c *SimpleClient) Invoke(service, method string, params []interface{}) (interface{}, error) {
	if !c.connected {
		return nil, fmt.Errorf("客户端未连接")
	}
	
	fmt.Printf("调用服务: %s.%s(%v)\n", service, method, params)
	
	result := map[string]interface{}{
		"success": true,
		"message": "调用成功",
		"data":    "模拟返回数据",
		"service": service,
		"method":  method,
		"params":  params,
	}
	
	return result, nil
}

// 列出服务
// 简单客户端不提供mock数据，需要使用真实的注册中心客户端
func (c *SimpleClient) ListServices() []string {
	fmt.Println("⚠️  简单客户端不支持服务发现，请使用ZooKeeper或Nacos客户端")
	return []string{}
}

// 关闭连接
func (c *SimpleClient) Close() {
	fmt.Println("关闭连接")
	c.connected = false
}

// 主函数
func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run build_simple.go <command>")
		fmt.Println("命令:")
		fmt.Println("  invoke  - 调用服务")
		fmt.Println("  list    - 列出服务")
		return
	}

	command := os.Args[1]
	
	// 创建配置
	config := &SimpleConfig{
		Registry:    "nacos://127.0.0.1:8848",
		Application: "dubbo-invoke-cli",
		Timeout:     5 * time.Second,
	}
	
	// 创建客户端
	client := NewSimpleClient(config)
	
	switch command {
	case "invoke":
		if len(os.Args) < 4 {
			fmt.Println("用法: go run build_simple.go invoke <service> <method> [params...]")
			return
		}
		
		service := os.Args[2]
		method := os.Args[3]
		params := make([]interface{}, 0)
		
		for i := 4; i < len(os.Args); i++ {
			params = append(params, os.Args[i])
		}
		
		client.Connect()
		result, err := client.Invoke(service, method, params)
		if err != nil {
			fmt.Printf("调用失败: %v\n", err)
		} else {
			fmt.Printf("调用结果: %v\n", result)
		}
		client.Close()
		
	case "list":
		client.Connect()
		services := client.ListServices()
		fmt.Println("可用服务:")
		for i, service := range services {
			fmt.Printf("  %d. %s\n", i+1, service)
		}
		client.Close()
		
	default:
		fmt.Printf("未知命令: %s\n", command)
	}
}