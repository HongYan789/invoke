package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-zookeeper/zk"
)

// ZooKeeperServiceDiscovery ZooKeeper服务发现客户端
type ZooKeeperServiceDiscovery struct {
	conn     *zk.Conn
	servers  []string
	timeout  time.Duration
	basePath string
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name      string            `json:"name"`
	Path      string            `json:"path"`
	Instances []InstanceInfo    `json:"instances"`
	Metadata  map[string]string `json:"metadata"`
}

// InstanceInfo 实例信息
type InstanceInfo struct {
	Host     string            `json:"host"`
	Port     int               `json:"port"`
	Protocol string            `json:"protocol"`
	Metadata map[string]string `json:"metadata"`
}

// NewZooKeeperServiceDiscovery 创建ZooKeeper服务发现客户端
func NewZooKeeperServiceDiscovery(servers []string, timeout time.Duration) *ZooKeeperServiceDiscovery {
	return &ZooKeeperServiceDiscovery{
		servers:  servers,
		timeout:  timeout,
		basePath: "/dubbo", // Dubbo默认路径
	}
}

// Connect 连接到ZooKeeper
func (zsd *ZooKeeperServiceDiscovery) Connect() error {
	fmt.Printf("🔗 正在连接ZooKeeper: %v\n", zsd.servers)
	
	conn, _, err := zk.Connect(zsd.servers, zsd.timeout)
	if err != nil {
		return fmt.Errorf("连接ZooKeeper失败: %v", err)
	}
	
	zsd.conn = conn
	fmt.Println("✅ ZooKeeper连接成功")
	return nil
}

// DiscoverServices 发现所有服务
func (zsd *ZooKeeperServiceDiscovery) DiscoverServices() ([]ServiceInfo, error) {
	if zsd.conn == nil {
		return nil, fmt.Errorf("ZooKeeper连接未建立")
	}
	
	fmt.Printf("🔍 开始扫描服务路径: %s\n", zsd.basePath)
	
	// 检查根路径是否存在
	exists, _, err := zsd.conn.Exists(zsd.basePath)
	if err != nil {
		return nil, fmt.Errorf("检查根路径失败: %v", err)
	}
	
	if !exists {
		fmt.Printf("⚠️  根路径 %s 不存在\n", zsd.basePath)
		return []ServiceInfo{}, nil
	}
	
	// 递归扫描所有服务
	services, err := zsd.scanServices(zsd.basePath)
	if err != nil {
		return nil, err
	}
	
	fmt.Printf("\n🎯 总共发现 %d 个真实服务\n", len(services))
	return services, nil
}

// scanServices 递归扫描服务
func (zsd *ZooKeeperServiceDiscovery) scanServices(path string) ([]ServiceInfo, error) {
	var services []ServiceInfo
	
	// 获取子节点
	children, _, err := zsd.conn.Children(path)
	if err != nil {
		return nil, fmt.Errorf("获取子节点失败 [%s]: %v", path, err)
	}
	
	for _, child := range children {
		childPath := filepath.Join(path, child)
		
		// 检查是否是服务路径（通常包含providers）
		if zsd.isServicePath(childPath) {
			service, err := zsd.parseService(childPath)
			if err != nil {
				fmt.Printf("  ⚠️  解析服务失败 [%s]: %v\n", childPath, err)
				continue
			}
			
			if service != nil {
				services = append(services, *service)
				fmt.Printf("  ✅ 发现服务: %s (实例数: %d)\n", service.Name, len(service.Instances))
			}
		} else {
			// 递归扫描子目录
			subServices, err := zsd.scanServices(childPath)
			if err != nil {
				fmt.Printf("  ⚠️  扫描子目录失败 [%s]: %v\n", childPath, err)
				continue
			}
			services = append(services, subServices...)
		}
	}
	
	return services, nil
}

// isServicePath 判断是否是服务路径
func (zsd *ZooKeeperServiceDiscovery) isServicePath(path string) bool {
	// 检查是否有providers子节点
	providersPath := filepath.Join(path, "providers")
	exists, _, err := zsd.conn.Exists(providersPath)
	return err == nil && exists
}

// parseService 解析服务信息
func (zsd *ZooKeeperServiceDiscovery) parseService(servicePath string) (*ServiceInfo, error) {
	// 从路径中提取服务名
	serviceName := filepath.Base(servicePath)
	
	// 获取providers信息
	providersPath := filepath.Join(servicePath, "providers")
	providers, _, err := zsd.conn.Children(providersPath)
	if err != nil {
		return nil, fmt.Errorf("获取providers失败: %v", err)
	}
	
	var instances []InstanceInfo
	for _, provider := range providers {
		// 解析provider URL
		instance, err := zsd.parseProviderURL(provider)
		if err != nil {
			fmt.Printf("    ⚠️  解析provider失败: %v\n", err)
			continue
		}
		
		if instance != nil {
			instances = append(instances, *instance)
		}
	}
	
	return &ServiceInfo{
		Name:      serviceName,
		Path:      servicePath,
		Instances: instances,
		Metadata:  make(map[string]string),
	}, nil
}

// parseProviderURL 解析provider URL
func (zsd *ZooKeeperServiceDiscovery) parseProviderURL(providerURL string) (*InstanceInfo, error) {
	// 简单的URL解析（实际项目中应该使用更完善的URL解析）
	if !strings.HasPrefix(providerURL, "dubbo://") {
		return nil, fmt.Errorf("不支持的协议: %s", providerURL)
	}
	
	// 提取host:port部分
	url := strings.TrimPrefix(providerURL, "dubbo://")
	parts := strings.Split(url, "/")
	if len(parts) < 1 {
		return nil, fmt.Errorf("无效的URL格式: %s", providerURL)
	}
	
	hostPort := parts[0]
	hostPortParts := strings.Split(hostPort, ":")
	if len(hostPortParts) != 2 {
		return nil, fmt.Errorf("无效的host:port格式: %s", hostPort)
	}
	
	host := hostPortParts[0]
	port := 0
	fmt.Sscanf(hostPortParts[1], "%d", &port)
	
	return &InstanceInfo{
		Host:     host,
		Port:     port,
		Protocol: "dubbo",
		Metadata: make(map[string]string),
	}, nil
}

// Close 关闭连接
func (zsd *ZooKeeperServiceDiscovery) Close() {
	if zsd.conn != nil {
		zsd.conn.Close()
		fmt.Println("🔌 ZooKeeper连接已关闭")
	}
}

// 测试ZooKeeper服务发现
func TestZooKeeperServiceDiscovery() {
	fmt.Println("\n=== ZooKeeper服务发现测试 ===")
	
	// 创建ZooKeeper客户端
	servers := []string{"10.7.8.40:2181"}
	zsd := NewZooKeeperServiceDiscovery(servers, 10*time.Second)
	
	// 连接ZooKeeper
	err := zsd.Connect()
	if err != nil {
		log.Printf("❌ 连接失败: %v", err)
		return
	}
	defer zsd.Close()
	
	// 发现服务
	services, err := zsd.DiscoverServices()
	if err != nil {
		log.Printf("❌ 服务发现失败: %v", err)
		return
	}
	
	// 显示结果
	if len(services) == 0 {
		fmt.Println("\n⚠️  没有发现任何服务")
	} else {
		fmt.Println("\n📋 真实服务清单:")
		for i, service := range services {
			fmt.Printf("\n%d. 服务名称: %s\n", i+1, service.Name)
			fmt.Printf("   路径: %s\n", service.Path)
			fmt.Printf("   实例数: %d\n", len(service.Instances))
			
			if len(service.Instances) > 0 {
				fmt.Println("   实例列表:")
				for j, instance := range service.Instances {
					fmt.Printf("     %d) %s:%d (%s)\n", j+1, instance.Host, instance.Port, instance.Protocol)
				}
			}
		}
	}
	
	fmt.Printf("\n✅ 测试完成，共发现 %d 个真实服务\n", len(services))
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "test" {
		TestZooKeeperServiceDiscovery()
	} else {
		fmt.Println("ZooKeeper服务发现测试程序")
		fmt.Println("用法: go run zk_service_discovery_test.go test")
		fmt.Println("")
		fmt.Println("功能:")
		fmt.Println("- 连接到真实的ZooKeeper (10.7.8.40:2181)")
		fmt.Println("- 扫描并发现所有注册的服务")
		fmt.Println("- 显示服务的详细信息和实例列表")
		fmt.Println("- 不使用任何mock数据，完全基于真实数据")
	}
}