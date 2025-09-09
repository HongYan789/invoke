package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NacosService 表示Nacos中的服务信息
type NacosService struct {
	ServiceName string `json:"serviceName"`
	GroupName   string `json:"groupName"`
	Clusters    string `json:"clusters"`
	CacheMillis int64  `json:"cacheMillis"`
	Hosts       []NacosHost `json:"hosts"`
	LastRefTime int64  `json:"lastRefTime"`
	Checksum    string `json:"checksum"`
	AllIPs      bool   `json:"allIPs"`
	ReachProtectionThreshold bool `json:"reachProtectionThreshold"`
	Valid       bool   `json:"valid"`
}

// NacosHost 表示Nacos中的主机信息
type NacosHost struct {
	InstanceId  string            `json:"instanceId"`
	IP          string            `json:"ip"`
	Port        int               `json:"port"`
	Weight      float64           `json:"weight"`
	Healthy     bool              `json:"healthy"`
	Enabled     bool              `json:"enabled"`
	Ephemeral   bool              `json:"ephemeral"`
	ClusterName string            `json:"clusterName"`
	ServiceName string            `json:"serviceName"`
	Metadata    map[string]string `json:"metadata"`
}

// NacosServiceList 表示Nacos服务列表响应
type NacosServiceList struct {
	Count    int      `json:"count"`
	Services []string `json:"doms"`
}

// NamespaceInfo 表示命名空间信息
type NamespaceInfo struct {
	Namespace         string `json:"namespace"`
	NamespaceShowName string `json:"namespaceShowName"`
	Quota             int    `json:"quota"`
	ConfigCount       int    `json:"configCount"`
	Type              int    `json:"type"`
}

// ServiceInfo 表示服务信息
type ServiceInfo struct {
	Name      string         `json:"name"`
	Namespace string         `json:"namespace"`
	GroupName string         `json:"groupName"`
	Instances []InstanceInfo `json:"instances"`
	Status    string         `json:"status"`
}

// InstanceInfo 表示实例信息
type InstanceInfo struct {
	IP       string            `json:"ip"`
	Port     int               `json:"port"`
	Healthy  bool              `json:"healthy"`
	Weight   float64           `json:"weight"`
	Metadata map[string]string `json:"metadata"`
}

// NacosClient Nacos客户端
type NacosClient struct {
	ServerAddr string
	Namespace  string
	GroupName  string
	Username   string
	Password   string
	Client     *http.Client
}

// NewNacosClient 创建新的Nacos客户端
func NewNacosClient(serverAddr, namespace, groupName string) *NacosClient {
	return &NacosClient{
		ServerAddr: serverAddr,
		Namespace:  namespace,
		GroupName:  groupName,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NewNacosClientWithAuth 创建带认证的Nacos客户端
func NewNacosClientWithAuth(serverAddr, namespace, groupName, username, password string) *NacosClient {
	return &NacosClient{
		ServerAddr: serverAddr,
		Namespace:  namespace,
		GroupName:  groupName,
		Username:   username,
		Password:   password,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// TestConnection 测试与Nacos服务器的连接
func (nc *NacosClient) TestConnection() error {
	// 构建健康检查URL
	healthURL := fmt.Sprintf("http://%s/nacos/v1/ns/operator/metrics", nc.ServerAddr)
	
	fmt.Printf("正在测试Nacos连接: %s\n", healthURL)
	
	resp, err := nc.Client.Get(healthURL)
	if err != nil {
		return fmt.Errorf("连接Nacos服务器失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Nacos服务器响应异常，状态码: %d", resp.StatusCode)
	}
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}
	
	fmt.Printf("Nacos连接成功，响应: %s\n", string(body))
	return nil
}

// GetServiceList 获取服务列表
func (nc *NacosClient) GetServiceList() (*NacosServiceList, error) {
	// 首先获取正确的命名空间ID
	realNamespaceId, err := nc.getRealNamespaceId()
	if err != nil {
		fmt.Printf("⚠️  获取命名空间ID失败: %v\n", err)
		realNamespaceId = nc.Namespace // 使用原始值作为fallback
	}
	
	fmt.Printf("使用命名空间ID: %s\n", realNamespaceId)

	// 尝试多个API端点和参数组合
	type EndpointConfig struct {
		Path   string
		Params map[string]string
	}
	
	endpoints := []EndpointConfig{
		// Nacos 官方推荐的API端点
		{"/nacos/v1/ns/service/list", map[string]string{"pageNo": "1", "pageSize": "100"}},
		{"/nacos/v2/ns/service/list", map[string]string{"pageNo": "1", "pageSize": "100"}},
		// 控制台API端点
		{"/nacos/v1/console/namespaces", map[string]string{}},
	}
	
	for i, endpoint := range endpoints {
		fmt.Printf("\n尝试API端点 %d: %s\n", i+1, endpoint.Path)
		
		// 构建服务列表查询URL
		serviceListURL := fmt.Sprintf("http://%s%s", nc.ServerAddr, endpoint.Path)
		
		// 构建查询参数
		params := url.Values{}
		
		// 使用配置中的参数
		for key, value := range endpoint.Params {
			params.Add(key, value)
		}
		
		// 根据不同端点添加额外参数
		if strings.Contains(endpoint.Path, "/service/list") {
			if realNamespaceId != "" && realNamespaceId != "public" {
				params.Add("namespaceId", realNamespaceId)
			}
			if nc.GroupName != "" {
				params.Add("groupName", nc.GroupName)
			}
		} else if strings.Contains(endpoint.Path, "/console/namespaces") {
			// 命名空间端点不需要额外参数
		}
		
		// 添加认证参数
		if nc.Username != "" && nc.Password != "" {
			params.Add("username", nc.Username)
			params.Add("password", nc.Password)
		}
		
		fullURL := fmt.Sprintf("%s?%s", serviceListURL, params.Encode())
		fmt.Printf("请求URL: %s\n", fullURL)
		
		resp, err := nc.Client.Get(fullURL)
		if err != nil {
			fmt.Printf("❌ 请求失败: %v\n", err)
			continue
		}
		defer resp.Body.Close()
		
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("❌ 读取响应失败: %v\n", err)
			continue
		}
		
		fmt.Printf("响应状态码: %d\n", resp.StatusCode)
		fmt.Printf("响应内容: %s\n", string(body))
		
		if resp.StatusCode == http.StatusOK {
			// 尝试解析不同格式的响应
			var serviceList NacosServiceList
			err = json.Unmarshal(body, &serviceList)
			if err != nil {
				// 尝试解析Nacos 2.x格式的响应
				var nacosV2Response struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
					Data    struct {
						Count    int      `json:"count"`
						Services []string `json:"services"`
					} `json:"data"`
				}
				err2 := json.Unmarshal(body, &nacosV2Response)
				if err2 == nil && nacosV2Response.Code == 0 {
					fmt.Printf("✅ 使用Nacos 2.x格式解析成功，找到 %d 个服务\n", nacosV2Response.Data.Count)
					return &NacosServiceList{
						Count:    nacosV2Response.Data.Count,
						Services: nacosV2Response.Data.Services,
					}, nil
				}
				fmt.Printf("⚠️  JSON解析失败: %v\n原始响应: %s\n", err, string(body))
				// 返回空结果而不是错误
				return &NacosServiceList{
					Count:    0,
					Services: []string{},
				}, nil
			}
			fmt.Printf("✅ 使用Nacos 1.x格式解析成功，找到 %d 个服务\n", serviceList.Count)
			return &serviceList, nil
		} else {
			fmt.Printf("❌ API调用失败，状态码: %d\n", resp.StatusCode)
		}
	}
	
	return nil, fmt.Errorf("所有API端点都调用失败")
}

// getRealNamespaceId 根据命名空间显示名称获取真实的命名空间ID
func (nc *NacosClient) getRealNamespaceId() (string, error) {
	// 如果命名空间为空或者已经是UUID格式，直接返回
	if nc.Namespace == "" || nc.Namespace == "public" {
		return "public", nil
	}
	
	// 检查是否已经是UUID格式（包含连字符的长字符串）
	if len(nc.Namespace) > 30 && strings.Contains(nc.Namespace, "-") {
		return nc.Namespace, nil
	}
	
	// 获取所有命名空间来查找匹配的ID
	namespaces, err := nc.GetNamespaces()
	if err != nil {
		return nc.Namespace, err
	}
	
	// 查找匹配的命名空间
	for _, ns := range namespaces {
		if ns.NamespaceShowName == nc.Namespace {
			fmt.Printf("找到命名空间映射: %s -> %s\n", nc.Namespace, ns.Namespace)
			return ns.Namespace, nil
		}
	}
	
	// 如果没找到匹配的，返回原始值
	fmt.Printf("未找到命名空间 '%s' 的映射，使用原始值\n", nc.Namespace)
	return nc.Namespace, nil
}

// GetNamespaces 获取所有命名空间
func (nc *NacosClient) GetNamespaces() ([]NamespaceInfo, error) {
	// 尝试多个命名空间API端点
	endpoints := []string{
		"/v1/console/namespaces",
		"/nacos/v1/console/namespaces",
		"/v1/ns/namespace",
		"/nacos/v1/ns/namespace",
	}
	
	for i, endpoint := range endpoints {
		// 构建查询参数
		params := url.Values{}
		if nc.Username != "" && nc.Password != "" {
			params.Add("username", nc.Username)
			params.Add("password", nc.Password)
		}
		
		namespaceURL := fmt.Sprintf("http://%s%s", nc.ServerAddr, endpoint)
		if len(params) > 0 {
			namespaceURL = fmt.Sprintf("%s?%s", namespaceURL, params.Encode())
		}
		fmt.Printf("\n尝试命名空间API端点 %d: %s\n", i+1, namespaceURL)
		
		resp, err := nc.Client.Get(namespaceURL)
		if err != nil {
			fmt.Printf("❌ 请求失败: %v\n", err)
			continue
		}
		defer resp.Body.Close()
		
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("❌ 读取响应失败: %v\n", err)
			continue
		}
		
		fmt.Printf("响应状态码: %d\n", resp.StatusCode)
		if resp.StatusCode == 502 {
			fmt.Printf("❌ 502错误，尝试下一个端点\n")
			continue
		}
		
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ 状态码异常: %d，响应: %s\n", resp.StatusCode, string(body))
			continue
		}
		
		fmt.Printf("✅ 成功获取响应\n")
		
		// 尝试解析响应
		var namespaces struct {
			Code int             `json:"code"`
			Data []NamespaceInfo `json:"data"`
		}
		
		err = json.Unmarshal(body, &namespaces)
		if err != nil {
			fmt.Printf("❌ 解析JSON失败: %v，响应内容: %s\n", err, string(body))
			continue
		}
		
		return namespaces.Data, nil
	}
	
	return nil, fmt.Errorf("所有命名空间API端点都调用失败")
}

// GetServiceDetail 获取指定服务的详细信息
func (nc *NacosClient) GetServiceDetail(serviceName string) (*NacosService, error) {
	// 构建服务详情查询URL
	serviceDetailURL := fmt.Sprintf("http://%s/nacos/v1/ns/instance/list", nc.ServerAddr)
	
	// 构建查询参数
	params := url.Values{}
	params.Add("serviceName", serviceName)
	if nc.Namespace != "" {
		params.Add("namespaceId", nc.Namespace)
	}
	if nc.GroupName != "" {
		params.Add("groupName", nc.GroupName)
	}
	// 添加认证参数
	if nc.Username != "" && nc.Password != "" {
		params.Add("username", nc.Username)
		params.Add("password", nc.Password)
	}
	
	fullURL := fmt.Sprintf("%s?%s", serviceDetailURL, params.Encode())
	fmt.Printf("正在获取服务详情: %s\n", fullURL)
	
	resp, err := nc.Client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("获取服务详情失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取服务详情响应异常，状态码: %d", resp.StatusCode)
	}
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取服务详情响应失败: %v", err)
	}
	
	fmt.Printf("服务详情响应: %s\n", string(body))
	
	var service NacosService
	err = json.Unmarshal(body, &service)
	if err != nil {
		return nil, fmt.Errorf("解析服务详情失败: %v", err)
	}
	
	return &service, nil
}

// LoadAvailableServices 加载可用服务列表
// 使用真实的Nacos API调用获取服务列表，不使用任何mock数据
func (nc *NacosClient) LoadAvailableServices() ([]ServiceInfo, error) {
	fmt.Println("🔍 开始加载Nacos可用服务...")
	
	fmt.Printf("\n📂 正在扫描命名空间: %s\n", nc.Namespace)
	
	// 使用真实的API调用获取服务列表
	serviceList, err := nc.GetServiceList()
	if err != nil {
		return nil, fmt.Errorf("获取服务列表失败: %v", err)
	}
	
	var allServices []ServiceInfo
	
	if serviceList == nil || len(serviceList.Services) == 0 {
		fmt.Println("⚠️  当前命名空间中没有发现任何服务")
		return allServices, nil
	}
	
	// 遍历真实的服务列表，查询每个服务的详情
	for _, serviceName := range serviceList.Services {
		fmt.Printf("\n🔍 查询服务: %s\n", serviceName)
		
		// 获取服务实例信息
		serviceDetail, err := nc.GetServiceDetail(serviceName)
		if err != nil {
			fmt.Printf("  ❌ 查询失败: %v\n", err)
			continue
		}
		
		// 转换实例信息
		var instances []InstanceInfo
		healthyCount := 0
		for _, host := range serviceDetail.Hosts {
			if host.Healthy {
				healthyCount++
			}
			instances = append(instances, InstanceInfo{
				IP:       host.IP,
				Port:     host.Port,
				Healthy:  host.Healthy,
				Weight:   host.Weight,
				Metadata: host.Metadata,
			})
		}
		
		status := fmt.Sprintf("%d/%d健康", healthyCount, len(instances))
		if len(instances) == 0 {
			status = "无实例"
		}
		
		// 添加真实的服务信息
		serviceInfo := ServiceInfo{
			Name:      serviceName,
			Namespace: nc.Namespace,
			GroupName: nc.GroupName,
			Instances: instances,
			Status:    status,
		}
		
		allServices = append(allServices, serviceInfo)
		fmt.Printf("  ✅ 找到服务 %s [%s]\n", serviceName, status)
	}
	
	fmt.Printf("\n🎯 总共发现 %d 个真实服务\n", len(allServices))
	
	if len(allServices) == 0 {
		fmt.Println("⚠️  当前没有发现任何服务")
	} else {
		fmt.Println("\n📋 真实服务清单:")
		for i, service := range allServices {
			fmt.Printf("  %d. %s (命名空间: %s, 组: %s, 状态: %s)\n",
				i+1, service.Name, service.Namespace, service.GroupName, service.Status)
		}
	}
	
	return allServices, nil
}

// TestNacosRegistry 测试Nacos注册中心功能
func TestNacosRegistry() {
	fmt.Println("=== Nacos注册中心测试开始 ===")
	
	// 解析Nacos地址
	nacosAddr := "yjj-nacos.it.yyjzt.com:28848"
	fmt.Printf("测试Nacos地址: %s\n", nacosAddr)
	
	// 创建Nacos客户端，使用项目配置
	client := NewNacosClientWithAuth(
		nacosAddr,
		"c946d14d-cd12-4baf-94a5-55c3adf17a68", // dev环境命名空间
		"DEFAULT_GROUP",
		"nacos",
		"nacos",
	)
	
	// 1. 测试连接
	fmt.Println("\n1. 测试Nacos连接...")
	err := client.TestConnection()
	if err != nil {
		fmt.Printf("❌ 连接测试失败: %v\n", err)
		return
	}
	fmt.Println("✅ Nacos连接测试成功")
	
	// 2. 加载所有可用服务
	fmt.Println("\n2. 加载所有可用服务...")
	services, err := client.LoadAvailableServices()
	if err != nil {
		fmt.Printf("❌ 加载服务失败: %v\n", err)
		return
	}
	
	if len(services) == 0 {
		fmt.Println("\n⚠️  当前没有发现任何服务")
	} else {
		fmt.Println("\n📋 可用服务清单:")
		for i, service := range services {
			fmt.Printf("\n  %d. 🔧 %s\n", i+1, service.Name)
			fmt.Printf("     📂 命名空间: %s\n", service.Namespace)
			fmt.Printf("     👥 分组: %s\n", service.GroupName)
			fmt.Printf("     📊 状态: %s\n", service.Status)
			
			if len(service.Instances) > 0 {
				fmt.Printf("     🖥️  实例列表:\n")
				for j, instance := range service.Instances {
					status := "❌ 不健康"
					if instance.Healthy {
						status = "✅ 健康"
					}
					fmt.Printf("        %d. %s:%d [%s] 权重:%.1f\n", j+1, instance.IP, instance.Port, status, instance.Weight)
				}
			}
		}
		
		fmt.Printf("\n📈 服务统计:\n")
		totalInstances := 0
		healthyInstances := 0
		namespaceCount := make(map[string]int)
		
		for _, service := range services {
			namespaceCount[service.Namespace]++
			totalInstances += len(service.Instances)
			for _, instance := range service.Instances {
				if instance.Healthy {
					healthyInstances++
				}
			}
		}
		
		fmt.Printf("  • 总服务数: %d\n", len(services))
		fmt.Printf("  • 总实例数: %d\n", totalInstances)
		fmt.Printf("  • 健康实例: %d\n", healthyInstances)
		fmt.Printf("  • 命名空间分布:\n")
		for ns, count := range namespaceCount {
			if ns == "" {
				ns = "public"
			}
			fmt.Printf("    - %s: %d个服务\n", ns, count)
		}
	}
	
	// 3. 测试特定服务查询
	fmt.Println("\n3. 测试特定服务查询...")
	testServices := []string{"dubbo-provider", "user-service", "order-service"}
	for _, serviceName := range testServices {
		fmt.Printf("\n查询服务: %s\n", serviceName)
		detail, err := client.GetServiceDetail(serviceName)
		if err != nil {
			if strings.Contains(err.Error(), "状态码: 404") {
				fmt.Printf("  ⚠️  服务 %s 不存在\n", serviceName)
			} else {
				fmt.Printf("  ❌ 查询失败: %v\n", err)
			}
		} else {
			fmt.Printf("  ✅ 找到服务 %s，实例数: %d\n", serviceName, len(detail.Hosts))
		}
	}
	
	fmt.Println("\n=== Nacos注册中心测试完成 ===")
}

// RunNacosTest 运行Nacos测试的入口函数
func RunNacosTest() {
	TestNacosRegistry()
}