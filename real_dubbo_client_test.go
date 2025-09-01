package main

import (
	"fmt"
	"testing"
	"time"
)

// TestRealDubboClientTimeout 测试真实Dubbo客户端的超时处理
func TestRealDubboClientTimeout(t *testing.T) {
	// 创建测试配置
	config := &DubboConfig{
		Registry:    "dubbo://10.7.8.50:16002", // 使用之前超时的地址
		Application: "test-dubbo-client",
		Timeout:     3000, // 3秒超时
	}

	// 创建客户端
	client, err := NewRealDubboClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 测试用例
	testCases := []struct {
		name        string
		serviceName string
		methodName  string
		paramTypes  []string
		params      []interface{}
	}{
		{
			name:        "测试用户服务调用",
			serviceName: "com.example.UserService",
			methodName:  "getUserById",
			paramTypes:  []string{"java.lang.Long"},
			params:      []interface{}{123},
		},
		{
			name:        "测试订单服务调用",
			serviceName: "com.example.OrderService",
			methodName:  "getOrderInfo",
			paramTypes:  []string{"java.lang.String"},
			params:      []interface{}{"ORDER123"},
		},
		{
			name:        "测试产品服务调用",
			serviceName: "com.example.ProductService",
			methodName:  "getProductList",
			paramTypes:  []string{},
			params:      []interface{}{},
		},
	}

	// 执行测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Printf("\n=== 执行测试: %s ===\n", tc.name)
			fmt.Printf("服务: %s\n", tc.serviceName)
			fmt.Printf("方法: %s\n", tc.methodName)
			fmt.Printf("参数: %v\n", tc.params)
			fmt.Printf("开始时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))

			// 记录开始时间
			startTime := time.Now()

			// 执行调用
			result, err := client.GenericInvoke(tc.serviceName, tc.methodName, tc.paramTypes, tc.params)

			// 记录结束时间
			endTime := time.Now()
			duration := endTime.Sub(startTime)

			fmt.Printf("调用耗时: %v\n", duration)
			fmt.Printf("结束时间: %s\n", endTime.Format("2006-01-02 15:04:05"))

			if err != nil {
				fmt.Printf("调用失败: %v\n", err)
				// 检查是否是超时错误
				if duration >= time.Duration(config.Timeout)*time.Millisecond {
					fmt.Printf("✓ 超时处理正确: 在%v后返回超时错误\n", duration)
				} else {
					fmt.Printf("✗ 非超时错误: %v\n", err)
				}
			} else {
				fmt.Printf("调用成功: %v\n", result)
				fmt.Printf("✓ 成功获取响应\n")
			}

			fmt.Printf("=== 测试完成 ===\n\n")
		})
	}
}

// TestRealDubboClientConnection 测试连接状态
func TestRealDubboClientConnection(t *testing.T) {
	config := &DubboConfig{
		Registry:    "dubbo://10.7.8.50:16002",
		Application: "test-connection-client",
		Timeout:     5000,
	}

	fmt.Printf("\n=== 测试连接状态 ===\n")
	fmt.Printf("注册中心: %s\n", config.Registry)

	client, err := NewRealDubboClient(config)
	if err != nil {
		fmt.Printf("✗ 创建客户端失败: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Printf("✓ 客户端创建成功\n")
	fmt.Printf("连接状态: %v\n", client.IsConnected())

	// 测试Ping
	err = client.Ping()
	if err != nil {
		fmt.Printf("✗ Ping失败: %v\n", err)
	} else {
		fmt.Printf("✓ Ping成功\n")
	}

	// 测试服务列表
	services, err := client.ListServices()
	if err != nil {
		fmt.Printf("✗ 获取服务列表失败: %v\n", err)
	} else {
		fmt.Printf("✓ 获取到%d个服务\n", len(services))
		for i, service := range services {
			if i < 5 { // 只显示前5个
				fmt.Printf("  - %s\n", service)
			}
		}
		if len(services) > 5 {
			fmt.Printf("  ... 还有%d个服务\n", len(services)-5)
		}
	}

	fmt.Printf("=== 连接测试完成 ===\n\n")
}

// BenchmarkRealDubboClientInvoke 性能测试
func BenchmarkRealDubboClientInvoke(b *testing.B) {
	config := &DubboConfig{
		Registry:    "dubbo://10.7.8.50:16002",
		Application: "benchmark-client",
		Timeout:     3000,
	}

	client, err := NewRealDubboClient(config)
	if err != nil {
		b.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GenericInvoke(
			"com.example.UserService",
			"getUserById",
			[]string{"java.lang.Long"},
			[]interface{}{int64(i)},
		)
	}
}