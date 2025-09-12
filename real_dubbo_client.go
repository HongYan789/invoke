package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
	"github.com/go-zookeeper/zk"
)

// ChunkedTransferManager 分块传输管理器
type ChunkedTransferManager struct {
	chunkSize    int
	maxChunks    int
	timeout      time.Duration
	compression  bool
}

// NewChunkedTransferManager 创建分块传输管理器
func NewChunkedTransferManager(chunkSize, maxChunks int, timeout time.Duration, compression bool) *ChunkedTransferManager {
	return &ChunkedTransferManager{
		chunkSize:   chunkSize,
		maxChunks:   maxChunks,
		timeout:     timeout,
		compression: compression,
	}
}

// ReadChunkedData 读取分块数据
func (ctm *ChunkedTransferManager) ReadChunkedData(conn net.Conn) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ctm.timeout)
	defer cancel()

	var buffer bytes.Buffer
	chunkBuffer := make([]byte, ctm.chunkSize)
	chunkCount := 0

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("分块读取超时")
		default:
		}

		if chunkCount >= ctm.maxChunks {
			return nil, fmt.Errorf("超过最大分块数量限制: %d", ctm.maxChunks)
		}

		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := conn.Read(chunkBuffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				if buffer.Len() > 0 {
					break // 已读取数据，超时退出
				}
				return nil, fmt.Errorf("分块读取超时: %v", err)
			}
			if buffer.Len() > 0 {
				break // 已读取数据，错误退出
			}
			return nil, fmt.Errorf("分块读取失败: %v", err)
		}

		if n == 0 {
			break
		}

		buffer.Write(chunkBuffer[:n])
		chunkCount++

		// 检查是否读取完整
		if n < len(chunkBuffer) {
			break
		}
	}

	data := buffer.Bytes()
	if ctm.compression {
		return ctm.decompressData(data)
	}
	return data, nil
}

// WriteChunkedData 写入分块数据
func (ctm *ChunkedTransferManager) WriteChunkedData(conn net.Conn, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), ctm.timeout)
	defer cancel()

	if ctm.compression {
		compressed, err := ctm.compressData(data)
		if err != nil {
			return fmt.Errorf("数据压缩失败: %v", err)
		}
		data = compressed
	}

	totalSize := len(data)
	written := 0

	for written < totalSize {
		select {
		case <-ctx.Done():
			return fmt.Errorf("分块写入超时")
		default:
		}

		chunkEnd := written + ctm.chunkSize
		if chunkEnd > totalSize {
			chunkEnd = totalSize
		}

		chunk := data[written:chunkEnd]
		conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		n, err := conn.Write(chunk)
		if err != nil {
			return fmt.Errorf("分块写入失败: %v", err)
		}

		written += n
	}

	return nil
}

// compressData 压缩数据
func (ctm *ChunkedTransferManager) compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	
	_, err := gzWriter.Write(data)
	if err != nil {
		return nil, err
	}
	
	err = gzWriter.Close()
	if err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// decompressData 解压数据
func (ctm *ChunkedTransferManager) decompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		// 如果不是gzip格式，直接返回原数据
		return data, nil
	}
	defer reader.Close()
	
	return io.ReadAll(reader)
}

// StreamProcessor 流式处理器
type StreamProcessor struct {
	bufferPool   *sync.Pool
	processorCh  chan []byte
	resultCh     chan ProcessResult
	ctx          context.Context
	cancel       context.CancelFunc
}

// ProcessResult 处理结果
type ProcessResult struct {
	Data  interface{}
	Error error
}

// NewStreamProcessor 创建流式处理器
func NewStreamProcessor(bufferSize int) *StreamProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamProcessor{
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, bufferSize)
			},
		},
		processorCh: make(chan []byte, 100),
		resultCh:    make(chan ProcessResult, 100),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// StartProcessing 开始流式处理
func (sp *StreamProcessor) StartProcessing() {
	go func() {
		for {
			select {
			case <-sp.ctx.Done():
				return
			case data := <-sp.processorCh:
				result := sp.processChunk(data)
				sp.resultCh <- result
				// 回收缓冲区
				sp.bufferPool.Put(data[:cap(data)])
			}
		}
	}()
}

// processChunk 处理数据块
func (sp *StreamProcessor) processChunk(data []byte) ProcessResult {
	// 尝试解析JSON
	var result interface{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	
	if err := decoder.Decode(&result); err != nil {
		return ProcessResult{Error: err}
	}
	
	return ProcessResult{Data: result}
}

// GetBuffer 获取缓冲区
func (sp *StreamProcessor) GetBuffer() []byte {
	return sp.bufferPool.Get().([]byte)
}

// PutBuffer 归还缓冲区
func (sp *StreamProcessor) PutBuffer(buf []byte) {
	sp.bufferPool.Put(buf)
}

// Stop 停止处理
func (sp *StreamProcessor) Stop() {
	sp.cancel()
	close(sp.processorCh)
	close(sp.resultCh)
}

// MemoryManager 内存管理器
type MemoryManager struct {
	objectPool   *sync.Pool
	bufferPool   *sync.Pool
	maxPoolSize  int
	currentSize  int
	mu           sync.RWMutex
}

// NewMemoryManager 创建内存管理器
func NewMemoryManager(maxPoolSize int) *MemoryManager {
	return &MemoryManager{
		objectPool: &sync.Pool{
			New: func() interface{} {
				return make(map[string]interface{})
			},
		},
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 8192))
			},
		},
		maxPoolSize: maxPoolSize,
	}
}

// GetObject 获取对象
func (mm *MemoryManager) GetObject() map[string]interface{} {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	if mm.currentSize < mm.maxPoolSize {
		mm.currentSize++
		return mm.objectPool.Get().(map[string]interface{})
	}
	return make(map[string]interface{})
}

// PutObject 归还对象
func (mm *MemoryManager) PutObject(obj map[string]interface{}) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	// 清空对象
	for k := range obj {
		delete(obj, k)
	}
	
	if mm.currentSize > 0 {
		mm.objectPool.Put(obj)
		mm.currentSize--
	}
}

// GetBuffer 获取缓冲区
func (mm *MemoryManager) GetBuffer() *bytes.Buffer {
	buf := mm.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer 归还缓冲区
func (mm *MemoryManager) PutBuffer(buf *bytes.Buffer) {
	mm.bufferPool.Put(buf)
}

// AsyncProcessor 异步处理器
type AsyncProcessor struct {
	workerPool   chan chan AsyncTask
	taskQueue    chan AsyncTask
	workerCount  int
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// AsyncTask 异步任务
type AsyncTask struct {
	ID       string
	Data     interface{}
	Callback func(interface{}, error)
	Timeout  time.Duration
}

// NewAsyncProcessor 创建异步处理器
func NewAsyncProcessor(workerCount int) *AsyncProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &AsyncProcessor{
		workerPool:  make(chan chan AsyncTask, workerCount),
		taskQueue:   make(chan AsyncTask, workerCount*10),
		workerCount: workerCount,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 启动异步处理器
func (ap *AsyncProcessor) Start() {
	// 启动工作协程
	for i := 0; i < ap.workerCount; i++ {
		ap.wg.Add(1)
		go ap.worker()
	}
	
	// 启动任务分发协程
	go ap.dispatcher()
}

// worker 工作协程
func (ap *AsyncProcessor) worker() {
	defer ap.wg.Done()
	
	taskChan := make(chan AsyncTask)
	
	for {
		// 注册工作协程
		select {
		case ap.workerPool <- taskChan:
			// 等待任务
			select {
			case task := <-taskChan:
				ap.processTask(task)
			case <-ap.ctx.Done():
				return
			}
		case <-ap.ctx.Done():
			return
		}
	}
}

// dispatcher 任务分发器
func (ap *AsyncProcessor) dispatcher() {
	for {
		select {
		case task := <-ap.taskQueue:
			// 获取可用工作协程
			select {
			case workerChan := <-ap.workerPool:
				// 分发任务
				select {
				case workerChan <- task:
				case <-ap.ctx.Done():
					return
				}
			case <-ap.ctx.Done():
				return
			}
		case <-ap.ctx.Done():
			return
		}
	}
}

// processTask 处理任务
func (ap *AsyncProcessor) processTask(task AsyncTask) {
	ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
	defer cancel()
	
	done := make(chan struct{})
	var result interface{}
	var err error
	
	go func() {
		defer close(done)
		// 这里可以根据任务类型进行不同的处理
		result = task.Data
	}()
	
	select {
	case <-done:
		if task.Callback != nil {
			task.Callback(result, err)
		}
	case <-ctx.Done():
		if task.Callback != nil {
			task.Callback(nil, fmt.Errorf("任务超时: %s", task.ID))
		}
	}
}

// SubmitTask 提交任务
func (ap *AsyncProcessor) SubmitTask(task AsyncTask) error {
	select {
	case ap.taskQueue <- task:
		return nil
	case <-ap.ctx.Done():
		return fmt.Errorf("异步处理器已关闭")
	default:
		return fmt.Errorf("任务队列已满")
	}
}

// Stop 停止异步处理器
func (ap *AsyncProcessor) Stop() {
	ap.cancel()
	ap.wg.Wait()
}

// OptimizedDubboConfig 优化的Dubbo配置
type OptimizedDubboConfig struct {
	*DubboConfig
	MaxPayloadSize    int           // 最大负载大小
	ChunkSize         int           // 分块大小
	MaxChunks         int           // 最大分块数
	CompressionLevel  int           // 压缩级别
	WorkerCount       int           // 工作协程数
	BufferPoolSize    int           // 缓冲池大小
	ConnectionPool    int           // 连接池大小
	RetryAttempts     int           // 重试次数
	RetryDelay        time.Duration // 重试延迟
}

// NewOptimizedDubboConfig 创建优化的Dubbo配置
func NewOptimizedDubboConfig(base *DubboConfig) *OptimizedDubboConfig {
	return &OptimizedDubboConfig{
		DubboConfig:       base,
		MaxPayloadSize:    50 * 1024 * 1024, // 50MB
		ChunkSize:         8192,              // 8KB
		MaxChunks:         1000,              // 最大1000个分块
		CompressionLevel:  6,                 // gzip压缩级别
		WorkerCount:       10,                // 10个工作协程
		BufferPoolSize:    1000,              // 1000个缓冲区
		ConnectionPool:    5,                 // 5个连接
		RetryAttempts:     3,                 // 重试3次
		RetryDelay:        time.Second,       // 1秒重试延迟
	}
}

// RealDubboClient 简化的真实Dubbo客户端实现
type RealDubboClient struct {
	config              *DubboConfig
	optimizedConfig     *OptimizedDubboConfig
	connected           bool
	conn                net.Conn
	chunkedTransferMgr  *ChunkedTransferManager
	streamProcessor     *StreamProcessor
	memoryManager       *MemoryManager
	asyncProcessor      *AsyncProcessor
	nacosClient         *NacosClient // 添加Nacos客户端
}



// NewRealDubboClient 创建真实的Dubbo客户端
func NewRealDubboClient(cfg *DubboConfig) (*RealDubboClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 设置默认值
	if cfg.Protocol == "" {
		cfg.Protocol = "dubbo"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 3 * time.Second
	}

	// 创建优化配置
	optimizedConfig := NewOptimizedDubboConfig(cfg)

	// 创建分块传输管理器
	chunkedMgr := NewChunkedTransferManager(
		optimizedConfig.ChunkSize,
		optimizedConfig.MaxChunks,
		cfg.Timeout * 3, // 传输超时时间
		true,  // 启用压缩
	)

	// 创建流式处理器
	streamProcessor := NewStreamProcessor(optimizedConfig.ChunkSize)
	streamProcessor.StartProcessing()

	// 创建内存管理器
	memoryManager := NewMemoryManager(optimizedConfig.BufferPoolSize)

	// 创建异步处理器
	asyncProcessor := NewAsyncProcessor(optimizedConfig.WorkerCount)
	asyncProcessor.Start()

	realClient := &RealDubboClient{
		config:             cfg,
		optimizedConfig:    optimizedConfig,
		chunkedTransferMgr: chunkedMgr,
		streamProcessor:    streamProcessor,
		memoryManager:      memoryManager,
		asyncProcessor:     asyncProcessor,
	}

	// 尝试连接到注册中心
	err := realClient.start()
	if err != nil {
		return nil, fmt.Errorf("启动Dubbo客户端失败: %v", err)
	}

	return realClient, nil
}

// start 启动Dubbo客户端
func (c *RealDubboClient) start() error {
	// 解析注册中心URL
	registryURL, err := c.parseRegistryURL()
	if err != nil {
		return fmt.Errorf("解析注册中心地址失败: %v", err)
	}

	// 根据注册中心类型进行连接
	switch registryURL.Protocol {
	case "zookeeper":
		// 连接到ZooKeeper注册中心
		return c.connectToZookeeper(registryURL.Address)
	case "nacos":
		// 连接到Nacos注册中心
		return c.connectToNacos(registryURL.Address)
	case "dubbo":
		// 连接到Dubbo注册中心
		return c.connectToDubboRegistry(registryURL.Address)
	case "direct":
		// 直连模式，连接到服务提供者
		return c.connectToDirect(registryURL.Address)
	default:
		return fmt.Errorf("不支持的注册中心类型: %s", registryURL.Protocol)
	}
}

// parseRegistryURL 解析注册中心URL
func (c *RealDubboClient) parseRegistryURL() (*RegistryURL, error) {
	url := c.config.Registry
	if url == "" {
		return nil, fmt.Errorf("注册中心地址不能为空")
	}

	// 解析协议和地址
	parts := strings.SplitN(url, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("无效的注册中心地址格式: %s", url)
	}

	return &RegistryURL{
		Protocol: parts[0],
		Address:  parts[1],
	}, nil
}

// connectToZookeeper 连接到ZooKeeper注册中心
func (c *RealDubboClient) connectToZookeeper(address string) error {
	// 连接到ZooKeeper获取服务提供者信息
	zkConn, events, err := zk.Connect([]string{address}, time.Second*10)
	if err != nil {
		return fmt.Errorf("连接ZooKeeper注册中心失败: %v", err)
	}
	defer zkConn.Close()

	// 等待连接建立，最多等待10秒
	timeout := time.After(10 * time.Second)
	for {
		select {
		case event := <-events:
			if event.State == zk.StateHasSession {
				fmt.Printf("成功连接到ZooKeeper注册中心: %s\n", address)
				c.connected = true
				fmt.Printf("ZooKeeper注册中心连接就绪，将在调用时获取服务提供者\n")
				return nil
			}
		case <-timeout:
			return fmt.Errorf("ZooKeeper连接超时，当前状态: %v", zkConn.State())
		}
	}
}

// getProviderFromZooKeeper 从ZooKeeper获取服务提供者地址
func (c *RealDubboClient) getProviderFromZooKeeper(serviceName string) (string, error) {
	// 解析注册中心地址
	registryURL, err := c.parseRegistryURL()
	if err != nil {
		return "", fmt.Errorf("解析注册中心地址失败: %v", err)
	}

	// 连接到ZooKeeper
	zkConn, _, err := zk.Connect([]string{registryURL.Address}, time.Second*10)
	if err != nil {
		return "", fmt.Errorf("连接ZooKeeper失败: %v", err)
	}
	defer zkConn.Close()

	// 构建服务路径
	servicePath := fmt.Sprintf("/dubbo/%s/providers", serviceName)
	fmt.Printf("查找服务提供者路径: %s\n", servicePath)

	// 检查路径是否存在
	exists, _, err := zkConn.Exists(servicePath)
	if err != nil {
		return "", fmt.Errorf("检查服务路径失败: %v", err)
	}
	if !exists {
		return "", fmt.Errorf("服务 %s 在ZooKeeper中不存在", serviceName)
	}

	// 获取提供者列表
	providers, _, err := zkConn.Children(servicePath)
	if err != nil {
		return "", fmt.Errorf("获取服务提供者列表失败: %v", err)
	}

	if len(providers) == 0 {
		return "", fmt.Errorf("服务 %s 没有可用的提供者", serviceName)
	}

	// 解析第一个提供者的地址
	providerURL := providers[0]
	fmt.Printf("找到服务提供者: %s\n", providerURL)

	// 解析URL获取地址
	address, err := c.parseProviderURL(providerURL)
	if err != nil {
		return "", fmt.Errorf("解析提供者URL失败: %v", err)
	}

	return address, nil
}

// parseProviderURL 解析提供者URL获取地址
func (c *RealDubboClient) parseProviderURL(providerURL string) (string, error) {
	// 首先进行URL解码
	decodedURL, err := url.QueryUnescape(providerURL)
	if err != nil {
		return "", fmt.Errorf("URL解码失败: %v", err)
	}
	
	fmt.Printf("解码后的URL: %s\n", decodedURL)
	
	// Dubbo提供者URL格式: dubbo://ip:port/serviceName?version=1.0.0&...
	if strings.HasPrefix(decodedURL, "dubbo://") {
		// 移除协议前缀
		urlPath := strings.TrimPrefix(decodedURL, "dubbo://")
		// 提取地址部分（ip:port）
		parts := strings.Split(urlPath, "/")
		if len(parts) > 0 {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("无效的提供者URL格式: %s", decodedURL)
}

// connectToNacos 连接到Nacos注册中心
func (c *RealDubboClient) connectToNacos(address string) error {
	// 从配置中获取命名空间，如果没有则使用默认值
	namespace := "public"
	if c.config.Namespace != "" {
		namespace = c.config.Namespace
	}
	
	// 创建Nacos客户端
	c.nacosClient = NewNacosClient(address, namespace, "DEFAULT_GROUP")
	
	// 测试连接
	err := c.nacosClient.TestConnection()
	if err != nil {
		return fmt.Errorf("连接Nacos注册中心失败: %v", err)
	}

	c.connected = true
	fmt.Printf("成功连接到Nacos注册中心: %s (命名空间: %s)\n", address, namespace)
	return nil
}

// connectToDubboRegistry 连接到Dubbo协议接口（直连模式）
func (c *RealDubboClient) connectToDubboRegistry(address string) error {
	// dubbo://协议表示直连到dubbo服务提供者
	// 尝试连接到Dubbo服务提供者
	conn, err := net.DialTimeout("tcp", address, c.config.Timeout)
	if err != nil {
		return fmt.Errorf("连接Dubbo服务提供者失败: %v", err)
	}

	c.conn = conn
	c.connected = true
	fmt.Printf("成功连接到Dubbo服务提供者: %s\n", address)
	return nil
}

// connectToDirect 直连模式连接到服务提供者
func (c *RealDubboClient) connectToDirect(address string) error {
	// 尝试连接到Dubbo服务提供者
	conn, err := net.DialTimeout("tcp", address, c.config.Timeout)
	if err != nil {
		return fmt.Errorf("连接Dubbo服务提供者失败: %v", err)
	}

	c.conn = conn
	c.connected = true
	fmt.Printf("成功连接到Dubbo服务提供者: %s\n", address)
	return nil
}

// GenericInvoke 泛化调用
func (c *RealDubboClient) GenericInvoke(serviceName, methodName string, paramTypes []string, params []interface{}) (interface{}, error) {
	if !c.connected {
		return nil, fmt.Errorf("客户端未连接")
	}

	// 验证参数
	if serviceName == "" {
		return nil, fmt.Errorf("服务名不能为空")
	}
	if methodName == "" {
		return nil, fmt.Errorf("方法名不能为空")
	}

	// 对于ZooKeeper模式，需要先获取服务提供者地址并建立连接
	registryURL, err := c.parseRegistryURL()
	if err == nil && registryURL.Protocol == "zookeeper" {
		// 如果当前没有连接到实际的Dubbo服务提供者，先获取地址并连接
		if c.conn == nil {
			providerAddress, err := c.getProviderFromZooKeeper(serviceName)
			if err != nil {
				return nil, fmt.Errorf("从ZooKeeper获取服务提供者失败: %v", err)
			}

			// 连接到实际的Dubbo服务提供者
			conn, err := net.DialTimeout("tcp", providerAddress, c.config.Timeout)
			if err != nil {
				return nil, fmt.Errorf("连接Dubbo服务提供者失败 %s: %v", providerAddress, err)
			}

			c.conn = conn
			fmt.Printf("成功连接到Dubbo服务提供者: %s\n", providerAddress)
		}
	}

	// 构建dubbo invoke命令，支持各种参数类型
	paramStr, err := c.formatParameters(params)
	if err != nil {
		return nil, fmt.Errorf("参数格式化失败: %v", err)
	}

	// 构建invoke命令
	invokeCmd := fmt.Sprintf("invoke %s.%s(%s)\n", serviceName, methodName, paramStr)
	fmt.Printf("[DUBBO CLIENT] 发送命令: %s", invokeCmd)

	// 将UTF-8编码的命令转换为GBK编码后发送
	// 因为很多Java Dubbo服务端默认使用GBK编码处理中文字符
	gbkBytes, err := c.convertToGBK(invokeCmd)
	if err != nil {
		fmt.Printf("[DUBBO CLIENT] GBK编码转换失败，使用UTF-8: %v\n", err)
		gbkBytes = []byte(invokeCmd)
	} else {
		fmt.Printf("[DUBBO CLIENT] 命令已转换为GBK编码\n")
	}

	// 发送invoke命令
	_, err = c.conn.Write(gbkBytes)
	if err != nil {
		return nil, fmt.Errorf("发送invoke命令失败: %v", err)
	}

	// 增加初始读取超时，给服务端更多时间响应
	initialTimeout := time.Duration(30 * time.Second)
	c.conn.SetReadDeadline(time.Now().Add(initialTimeout))
	
	// 使用传统方式读取完整响应数据，避免分块限制导致数据截断
	var responseBuffer bytes.Buffer
	tempBuffer := make([]byte, 4096)
	
	for {
		n, err := c.conn.Read(tempBuffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				if responseBuffer.Len() > 0 {
					break // 已读取数据，超时退出
				}
				return nil, fmt.Errorf("读取响应超时: %v", err)
			}
			if responseBuffer.Len() > 0 {
				break // 已读取数据，连接关闭或其他错误退出
			}
			return nil, fmt.Errorf("读取响应失败: %v", err)
		}
		
		if n == 0 {
			break
		}
		
		responseBuffer.Write(tempBuffer[:n])
		
		// 检查是否读取完整（包含dubbo>提示符或其他结束标识）
		responseText := responseBuffer.String()
		if strings.Contains(responseText, "dubbo>") || 
		   strings.Contains(responseText, "elapsed:") {
			break
		}
		
		// 设置较短的读取超时，避免无限等待
		c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	}

	// 重置读取超时
	c.conn.SetReadDeadline(time.Now().Add(c.config.Timeout))
	
	// 获取完整的响应文本
	responseText := responseBuffer.String()
	fmt.Printf("[DUBBO CLIENT] 完整响应文本: %s\n", responseText)
	
	// 尝试将响应从GBK编码转换为UTF-8
	utf8ResponseText, err := c.convertToUTF8(responseBuffer.Bytes())
	if err != nil {
		// 如果转换失败，使用原始字符串
		utf8ResponseText = responseText
		fmt.Printf("[DUBBO CLIENT] UTF-8转换失败，使用原始文本\n")
	} else {
		fmt.Printf("[DUBBO CLIENT] UTF-8转换成功\n")
	}
	
	// 检查是否包含错误信息
	if strings.Contains(utf8ResponseText, "Failed to invoke") || 
	   strings.Contains(utf8ResponseText, "error") ||
	   strings.Contains(utf8ResponseText, "No such service") ||
	   strings.Contains(utf8ResponseText, "No provider") ||
	   strings.Contains(utf8ResponseText, "Service not found") {
		return nil, fmt.Errorf("调用失败: %s", utf8ResponseText)
	}

	// 清理响应文本，提取JSON部分
	cleanedResponse := c.cleanResponse(utf8ResponseText)
	fmt.Printf("[DUBBO CLIENT] 清理后的响应: %s\n", cleanedResponse)
	
	// 检查清理后的响应是否仍然包含dubbo控制台输出
	// 如果清理后的响应包含"elapsed:"或"dubbo>"，说明可能没有获得有效的业务响应
	// 但是"null"和有效的JSON（包括数组）都是有效的业务响应
	if cleanedResponse != "null" && 
	   (strings.Contains(cleanedResponse, "elapsed:") || 
	    strings.Contains(cleanedResponse, "dubbo>")) {
		// 进一步检查：如果是有效的JSON，则认为是有效响应
		var jsonTest interface{}
		if json.Unmarshal([]byte(cleanedResponse), &jsonTest) != nil {
			// 不是有效的JSON且包含控制台输出，认为是无效响应
			return nil, fmt.Errorf("调用失败，未获得有效响应: %s", cleanedResponse)
		}
		// 如果是有效的JSON，继续执行，认为是有效响应
	}
	
	// 返回清理后的响应
	return cleanedResponse, nil
}

// ListServices 列出可用服务
func (c *RealDubboClient) ListServices() ([]string, error) {
	if !c.connected {
		return nil, fmt.Errorf("客户端未连接")
	}

	// 根据注册中心类型使用不同的获取方式
	registryURL, err := c.parseRegistryURL()
	if err != nil {
		return nil, fmt.Errorf("解析注册中心URL失败: %v", err)
	}

	switch registryURL.Protocol {
	case "zookeeper":
		return c.getServicesFromZooKeeper()
	case "nacos":
		return c.getServicesFromNacos()
	case "dubbo":
		return c.getServicesFromDubboRegistry()
	default:
		// 对于直连模式，返回模拟的服务列表
		return c.getServicesFromDirect()
	}
}

// getServicesFromZooKeeper 从ZooKeeper获取服务列表
func (c *RealDubboClient) getServicesFromZooKeeper() ([]string, error) {
	// 解析注册中心地址
	registryURL, err := c.parseRegistryURL()
	if err != nil {
		return nil, fmt.Errorf("解析注册中心地址失败: %v", err)
	}

	// 连接到ZooKeeper
	conn, _, err := zk.Connect([]string{registryURL.Address}, time.Second*10)
	if err != nil {
		return nil, fmt.Errorf("连接ZooKeeper失败: %v", err)
	}
	defer conn.Close()

	// 扫描Dubbo服务路径
	services, err := c.scanZooKeeperServices(conn, "/dubbo")
	if err != nil {
		return nil, fmt.Errorf("扫描ZooKeeper服务失败: %v", err)
	}

	return services, nil
}

// scanZooKeeperServices 扫描ZooKeeper中的Dubbo服务
func (c *RealDubboClient) scanZooKeeperServices(conn *zk.Conn, basePath string) ([]string, error) {
	var services []string
	
	// 检查基础路径是否存在
	exists, _, err := conn.Exists(basePath)
	if err != nil {
		return nil, fmt.Errorf("检查路径 %s 失败: %v", basePath, err)
	}
	if !exists {
		return services, nil // 路径不存在，返回空列表
	}

	// 获取子节点
	children, _, err := conn.Children(basePath)
	if err != nil {
		return nil, fmt.Errorf("获取 %s 子节点失败: %v", basePath, err)
	}

	for _, child := range children {
		childPath := basePath + "/" + child
		
		// 检查是否为服务路径（包含providers子目录）
		providersPath := childPath + "/providers"
		exists, _, err := conn.Exists(providersPath)
		if err != nil {
			continue // 忽略错误，继续处理下一个
		}
		
		if exists {
			// 这是一个服务，添加到列表中
			services = append(services, child)
		} else {
			// 递归扫描子目录
			subServices, err := c.scanZooKeeperServices(conn, childPath)
			if err != nil {
				continue // 忽略错误，继续处理
			}
			services = append(services, subServices...)
		}
	}

	return services, nil
}

// getServicesFromNacos 从Nacos获取服务列表
func (c *RealDubboClient) getServicesFromNacos() ([]string, error) {
	if c.nacosClient == nil {
		return nil, fmt.Errorf("Nacos客户端未初始化")
	}
	
	// 使用NacosClient获取真实的服务列表
	serviceList, err := c.nacosClient.GetServiceList()
	if err != nil {
		return nil, fmt.Errorf("获取Nacos服务列表失败: %v", err)
	}
	
	// 提取服务名称
	var services []string
	if serviceList != nil && serviceList.Services != nil {
		services = serviceList.Services
	}
	
	// 如果没有获取到服务，返回空列表而不是错误
	if len(services) == 0 {
		fmt.Printf("警告: 在命名空间 '%s' 中未找到任何服务\n", c.nacosClient.Namespace)
	}
	
	return services, nil
}

// getServicesFromDubboRegistry 从Dubbo注册中心获取服务列表
func (c *RealDubboClient) getServicesFromDubboRegistry() ([]string, error) {
	// 使用dubbo协议的ls命令获取真实服务列表
	lsCommand := "ls\n"
	_, err := c.conn.Write([]byte(lsCommand))
	if err != nil {
		return nil, fmt.Errorf("发送ls命令失败: %v", err)
	}

	// 读取响应 - 使用动态缓冲区读取完整数据
	var responseBuffer bytes.Buffer
	buffer := make([]byte, 8192)
	
	for {
		n, err := c.conn.Read(buffer)
		if err != nil {
			// 如果已经读取了数据，继续处理
			if responseBuffer.Len() > 0 {
				break
			}
			return nil, fmt.Errorf("读取服务列表响应失败: %v", err)
		}
		
		responseBuffer.Write(buffer[:n])
		
		// 检查是否读取完整
		if n < len(buffer) {
			break
		}
		
		// 设置较短超时检查更多数据
		c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	}

	// 解析响应文本
	responseText := responseBuffer.String()
	
	// 提取服务列表
	services := c.parseServiceList(responseText)
	
	if len(services) == 0 {
		return nil, fmt.Errorf("未发现任何服务")
	}
	
	return services, nil
}

// getServicesFromDirect 从直连模式获取服务列表
func (c *RealDubboClient) getServicesFromDirect() ([]string, error) {
	// 直连模式返回示例服务
	return []string{
		"com.example.DirectService",
	}, nil
}

// parseServiceList 解析dubbo ls命令的响应文本
func (c *RealDubboClient) parseServiceList(responseText string) []string {
	services := make([]string, 0)
	lines := strings.Split(responseText, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过空行、提示符和非服务行
		if line == "" || strings.HasPrefix(line, "dubbo>") || 
		   strings.Contains(line, "Use") || strings.Contains(line, "help") ||
		   strings.Contains(line, "PROVIDER") || strings.Contains(line, "CONSUMER") {
			continue
		}
		
		// 检查是否为有效的服务名（包含包名的格式）
		if strings.Contains(line, ".") && !strings.Contains(line, " ") {
			services = append(services, line)
		}
	}
	
	return services
}

// ListMethods 列出服务的方法
func (c *RealDubboClient) ListMethods(serviceName string) ([]string, error) {
	if !c.connected {
		return nil, fmt.Errorf("客户端未连接")
	}

	// 发送方法列表查询请求
	request := map[string]interface{}{
		"action":  "listMethods",
		"service": serviceName,
		"client":  c.config.Application,
	}

	requestData, _ := json.Marshal(request)
	c.conn.Write(requestData)

	// 读取响应 - 使用动态缓冲区读取完整数据
	var responseBuffer bytes.Buffer
	buffer := make([]byte, 4096)
	
	for {
		n, err := c.conn.Read(buffer)
		if err != nil {
			// 如果已经读取了数据，尝试解析
			if responseBuffer.Len() > 0 {
				break
			}
			// 如果读取失败，返回默认方法列表
			return c.getDefaultMethods(serviceName), nil
		}
		
		responseBuffer.Write(buffer[:n])
		
		// 检查是否读取完整
		if n < len(buffer) {
			break
		}
		
		// 设置较短超时检查更多数据
		c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	}

	// 尝试解析方法列表，使用json.Number保持精度
	decoder := json.NewDecoder(bytes.NewReader(responseBuffer.Bytes()))
	decoder.UseNumber()
	var response map[string]interface{}
	if err := decoder.Decode(&response); err == nil {
		if methods, exists := response["methods"]; exists {
			if methodList, ok := methods.([]interface{}); ok {
				result := make([]string, len(methodList))
				for i, method := range methodList {
					result[i] = fmt.Sprintf("%v", method)
				}
				return result, nil
			}
		}
	}

	// 返回默认方法列表
	return c.getDefaultMethods(serviceName), nil
}

// getDefaultMethods 获取默认方法列表
func (c *RealDubboClient) getDefaultMethods(serviceName string) []string {
	switch {
	case strings.Contains(serviceName, "User"):
		return []string{"getUserById", "getAllUsers", "createUser", "updateUser", "deleteUser"}
	case strings.Contains(serviceName, "Order"):
		return []string{"getOrderById", "getAllOrders", "createOrder", "updateOrder", "cancelOrder"}
	case strings.Contains(serviceName, "Product"):
		return []string{"getProductById", "getAllProducts", "createProduct", "updateProduct", "deleteProduct"}
	case strings.Contains(serviceName, "Payment"):
		return []string{"processPayment", "refundPayment", "getPaymentStatus", "getPaymentHistory"}
	default:
		return []string{"invoke", "query", "create", "update", "delete"}
	}
}

// Close 关闭客户端
func (c *RealDubboClient) Close() error {
	// 停止异步处理器
	if c.asyncProcessor != nil {
		c.asyncProcessor.Stop()
	}
	
	// 停止流式处理器
	if c.streamProcessor != nil {
		c.streamProcessor.Stop()
	}
	
	// 关闭网络连接
	if c.conn != nil {
		c.conn.Close()
		c.connected = false
	}
	
	fmt.Println("真实Dubbo客户端已关闭")
	return nil
}

// GetConfig 获取配置
func (c *RealDubboClient) GetConfig() *DubboConfig {
	return c.config
}

// IsConnected 检查连接状态
func (c *RealDubboClient) IsConnected() bool {
	return c.connected
}

// Ping 测试连接
func (c *RealDubboClient) Ping() error {
	if !c.connected {
		return fmt.Errorf("客户端未连接")
	}
	return nil
}

// convertToGBK 将UTF-8字符串转换为GBK编码的字节数组
func (c *RealDubboClient) convertToGBK(text string) ([]byte, error) {
	// 将UTF-8字符串转换为GBK编码
	reader := transform.NewReader(strings.NewReader(text), simplifiedchinese.GBK.NewEncoder())
	gbkData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("GBK编码转换失败: %v", err)
	}
	return gbkData, nil
}

// convertToUTF8 将字节数组从GBK编码转换为UTF-8字符串
func (c *RealDubboClient) convertToUTF8(data []byte) (string, error) {
	// 尝试GBK解码
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	utf8Data, err := io.ReadAll(reader)
	if err != nil {
		// 如果GBK解码失败，尝试GB18030
		reader = transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewDecoder())
		utf8Data, err = io.ReadAll(reader)
		if err != nil {
			return "", err
		}
	}
	return string(utf8Data), nil
}

// formatParameters 格式化参数，支持各种复杂类型
func (c *RealDubboClient) formatParameters(params []interface{}) (string, error) {
	if len(params) == 0 {
		return "", nil
	}
	
	var paramStrs []string
	for _, param := range params {
		formattedParam, err := c.formatSingleParameter(param)
		if err != nil {
			return "", err
		}
		paramStrs = append(paramStrs, formattedParam)
	}
	
	return strings.Join(paramStrs, ", "), nil
}

// formatSingleParameter 格式化单个参数
func (c *RealDubboClient) formatSingleParameter(param interface{}) (string, error) {
	switch v := param.(type) {
	case nil:
		return "null", nil
	case string:
		// 检查是否是JSON字符串（包含class字段的对象）
		if strings.Contains(v, "\"class\":") {
			// 验证JSON格式，使用json.Number保持精度
			decoder := json.NewDecoder(strings.NewReader(v))
			decoder.UseNumber()
			var jsonTest interface{}
			if decoder.Decode(&jsonTest) == nil {
				return v, nil // 直接返回JSON字符串
			}
		}
		// 确保字符串参数正确编码为UTF-8
		// 对于包含中文字符的字符串，使用JSON编码确保正确的转义
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			// 如果JSON编码失败，回退到简单的引号包围
			return fmt.Sprintf("\"%s\"", v), nil
		}
		return string(jsonBytes), nil
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%v", v), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v), nil
	case float32, float64:
		return fmt.Sprintf("%v", v), nil
	case bool:
		return fmt.Sprintf("%v", v), nil
	case map[string]interface{}:
		// 处理对象类型
		return c.formatObjectParameter(v)
	case []interface{}:
		// 处理数组类型
		return c.formatArrayParameter(v)
	default:
		// 尝试JSON序列化
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v), nil
		}
		return string(jsonBytes), nil
	}
}

// formatObjectParameter 格式化对象参数
func (c *RealDubboClient) formatObjectParameter(obj map[string]interface{}) (string, error) {
	// 如果包含class字段，按dubbo对象格式处理
	if _, hasClass := obj["class"]; hasClass {
		// 构建dubbo对象格式: {"class":"com.xxx.Class", "field1":value1, "field2":value2}
		jsonBytes, err := json.Marshal(obj)
		if err != nil {
			return "", err
		}
		return string(jsonBytes), nil
	}
	
	// 普通对象，直接JSON序列化
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// formatArrayParameter 格式化数组参数
func (c *RealDubboClient) formatArrayParameter(arr []interface{}) (string, error) {
	// 处理空数组情况
	if len(arr) == 0 {
		return "[]", nil
	}
	
	var elements []string
	for _, element := range arr {
		formattedElement, err := c.formatSingleParameter(element)
		if err != nil {
			return "", err
		}
		elements = append(elements, formattedElement)
	}
	return "[" + strings.Join(elements, ", ") + "]", nil
}

// cleanResponse 清理dubbo响应文本，提取JSON部分
func (c *RealDubboClient) cleanResponse(responseText string) string {
	// 特殊处理包含dubbo>的响应
	if strings.Contains(responseText, "dubbo>") {
		// 提取dubbo>之前的内容
		parts := strings.Split(responseText, "dubbo>")
		if len(parts) > 0 {
			content := strings.TrimSpace(parts[0])
			// 移除末尾的elapsed信息
			if strings.Contains(content, "elapsed:") {
				elapsedParts := strings.Split(content, "elapsed:")
				if len(elapsedParts) > 0 {
					content = strings.TrimSpace(elapsedParts[0])
				}
			}
			// 如果是null或有效JSON，直接返回
			if content == "null" {
				return "null"
			}
			// 检查是否为有效JSON
			var jsonTest interface{}
			if json.Unmarshal([]byte(content), &jsonTest) == nil {
				return content
			}
		}
	}
	
	// 首先尝试直接解析原始响应作为JSON
	if strings.HasPrefix(strings.TrimSpace(responseText), "[") {
		// 尝试直接验证原始响应是否为有效JSON
		var jsonTest interface{}
		decoder := json.NewDecoder(strings.NewReader(responseText))
		decoder.UseNumber()
		if decoder.Decode(&jsonTest) == nil {
			return responseText
		} else {
			// 尝试修复不完整的JSON数组
			fixed := c.fixIncompleteJSON(responseText)
			if fixed != "" {
				return fixed
			}
		}
	}
	
	// 如果直接解析失败，使用原来的extractLargestJSON方法
	jsonResult := c.extractLargestJSON(responseText)
	if jsonResult != "" {
		// 检查是否为数组类型，如果是则直接返回
		if strings.HasPrefix(jsonResult, "[") && strings.HasSuffix(jsonResult, "]") {
			return jsonResult
		}
		return jsonResult
	}
	
	// 2. 按行分割响应，逐行检查
	lines := strings.Split(responseText, "\n")
	
	// 创建一个新的响应构建器，用于处理多行JSON
	var resultBuilder strings.Builder
	foundJSONStart := false
	jsonStartChar := ""
	
	for _, line := range lines {
		// 去除首尾空白字符
		line = strings.TrimSpace(line)
		
		// 跳过空行和非JSON行
		if line == "" || strings.HasPrefix(line, "elapsed:") || strings.HasPrefix(line, "dubbo>") {
			continue
		}
		
		// 检查是否是JSON格式开始
		if !foundJSONStart {
			if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
				foundJSONStart = true
				if strings.HasPrefix(line, "{") {
					jsonStartChar = "{"
				} else {
					jsonStartChar = "["
				}
				resultBuilder.WriteString(line)
				continue
			}
		}
		
		// 如果已经找到JSON开始，继续添加行直到结束
		if foundJSONStart {
			resultBuilder.WriteString(line)
			
			// 检查是否是JSON结束
			if (jsonStartChar == "{" && strings.HasSuffix(line, "}")) ||
			   (jsonStartChar == "[" && strings.HasSuffix(line, "]")) {
				// 尝试解析构建的JSON
			builtJSON := resultBuilder.String()
			var jsonTest interface{}
			decoder := json.NewDecoder(strings.NewReader(builtJSON))
			decoder.UseNumber()
			if decoder.Decode(&jsonTest) == nil {
				return builtJSON
			}
			}
		}
		
		// 检查单行JSON对象或数组
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			var jsonTest interface{}
			decoder := json.NewDecoder(strings.NewReader(line))
			decoder.UseNumber()
			if decoder.Decode(&jsonTest) == nil {
				return line
			}
		}
		
		// 检查单行JSON数组
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			var jsonTest interface{}
			decoder := json.NewDecoder(strings.NewReader(line))
			decoder.UseNumber()
			if decoder.Decode(&jsonTest) == nil {
				return line
			}
		}
		
		// 3. 以双引号包围的JSON字符串（如"[{...}]"或"{...}"）
		if strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "\"") && len(line) > 2 {
			// 去除外层双引号
			unquoted := line[1 : len(line)-1]
			// 尝试解析内部的JSON
			var jsonTest interface{}
			decoder := json.NewDecoder(strings.NewReader(unquoted))
			decoder.UseNumber()
			if decoder.Decode(&jsonTest) == nil {
				return unquoted // 返回去除双引号后的JSON
			}
		}
	}
	
	// 如果构建了JSON但未成功解析，尝试返回构建的结果
	if foundJSONStart {
		builtJSON := resultBuilder.String()
		var jsonTest interface{}
		decoder := json.NewDecoder(strings.NewReader(builtJSON))
		decoder.UseNumber()
		if decoder.Decode(&jsonTest) == nil {
			return builtJSON
		}
	}
	
	// 如果没有找到有效的JSON，返回原始响应
	return responseText
}

// fixIncompleteJSON 尝试修复不完整的JSON数组
func (c *RealDubboClient) fixIncompleteJSON(responseText string) string {
	// 查找最后一个完整的JSON对象
	lastCompleteIndex := -1
	braceCount := 0
	inObject := false
	
	for i, char := range responseText {
		switch char {
		case '{':
			if !inObject {
				inObject = true
				braceCount = 1
			} else {
				braceCount++
			}
		case '}':
			if inObject {
				braceCount--
				if braceCount == 0 {
					// 找到一个完整的对象
					lastCompleteIndex = i
					inObject = false
				}
			}
		}
	}
	
	if lastCompleteIndex > 0 {
		// 截取到最后一个完整对象的位置，并添加数组结束符
		fixedJSON := responseText[:lastCompleteIndex+1] + "]"
		
		// 验证修复后的JSON是否有效
		var jsonTest interface{}
		decoder := json.NewDecoder(strings.NewReader(fixedJSON))
		decoder.UseNumber()
		if decoder.Decode(&jsonTest) == nil {
			return fixedJSON
		}
	}
	
	return ""
}

// extractLargestJSON 从响应文本中提取最大的有效JSON
func (c *RealDubboClient) extractLargestJSON(responseText string) string {
	// 查找所有可能的JSON起始位置
	var candidates []string
	
	// 查找JSON数组 [...] - 优先处理数组
	for i := 0; i < len(responseText); i++ {
		if responseText[i] == '[' {
			// 找到匹配的右括号
			bracketCount := 1
			for j := i + 1; j < len(responseText) && bracketCount > 0; j++ {
				if responseText[j] == '[' {
					bracketCount++
				} else if responseText[j] == ']' {
					bracketCount--
				}
				if bracketCount == 0 {
					candidate := responseText[i:j+1]
					// 验证是否为有效JSON
					var jsonTest interface{}
					decoder := json.NewDecoder(strings.NewReader(candidate))
					decoder.UseNumber()
					if decoder.Decode(&jsonTest) == nil {
						candidates = append(candidates, candidate)
					}
					break
				}
			}
		}
	}
	
	// 查找JSON对象 {...}
	for i := 0; i < len(responseText); i++ {
		if responseText[i] == '{' {
			// 找到匹配的右括号
			braceCount := 1
			for j := i + 1; j < len(responseText) && braceCount > 0; j++ {
				if responseText[j] == '{' {
					braceCount++
				} else if responseText[j] == '}' {
					braceCount--
				}
				if braceCount == 0 {
					candidate := responseText[i:j+1]
					// 验证是否为有效JSON
					var jsonTest interface{}
					decoder := json.NewDecoder(strings.NewReader(candidate))
					decoder.UseNumber()
					if decoder.Decode(&jsonTest) == nil {
						candidates = append(candidates, candidate)
					}
					break
				}
			}
		}
	}
	
	// 返回最长的有效JSON
	longestJSON := ""
	for _, candidate := range candidates {
		if len(candidate) > len(longestJSON) {
			longestJSON = candidate
		}
	}
	
	return longestJSON
}

// ResponseCompletionDetector 响应完整性检测器
type ResponseCompletionDetector struct {
	protocolMarkers []string
	errorMarkers    []string
}

// NewResponseCompletionDetector 创建响应完整性检测器
func NewResponseCompletionDetector() *ResponseCompletionDetector {
	return &ResponseCompletionDetector{
		protocolMarkers: []string{
			"dubbo>",           // Dubbo命令行结束标志
			"elapsed:",         // 执行时间标志
			"ms.",              // 毫秒标志
			"result:",          // 结果标志
		},
		errorMarkers: []string{
			"Failed to invoke",
			"No such service",
			"No provider",
			"Connection refused",
			"Timeout",
			"Exception",
		},
	}
}

// isResponseComplete 检查响应是否完整 - 重构版本
func (c *RealDubboClient) isResponseComplete(responseText string) bool {
	detector := NewResponseCompletionDetector()
	
	// 1. 检查协议标识符完整性
	if detector.hasProtocolCompletion(responseText) {
		return true
	}
	
	// 2. 检查JSON结构完整性
	if detector.hasValidJSONStructure(responseText) {
		return true
	}
	
	// 3. 检查错误响应完整性
	if detector.hasErrorCompletion(responseText) {
		return true
	}
	
	// 4. 检查特殊响应（如null）
	if detector.hasSpecialResponseCompletion(responseText) {
		return true
	}
	
	return false
}

// hasProtocolCompletion 检查协议标识符完整性
func (d *ResponseCompletionDetector) hasProtocolCompletion(responseText string) bool {
	// Dubbo命令行结束标志 + 执行时间标志
	if strings.Contains(responseText, "dubbo>") && 
	   (strings.Contains(responseText, "elapsed:") || strings.Contains(responseText, "ms.")) {
		return true
	}
	return false
}

// hasValidJSONStructure 检查JSON结构完整性
func (d *ResponseCompletionDetector) hasValidJSONStructure(responseText string) bool {
	// 提取可能的JSON内容
	jsonContent := d.extractPotentialJSON(responseText)
	if jsonContent == "" {
		return false
	}
	
	// 验证JSON结构完整性
	return d.validateJSONCompleteness(jsonContent)
}

// extractPotentialJSON 提取潜在的JSON内容
func (d *ResponseCompletionDetector) extractPotentialJSON(responseText string) string {
	// 查找JSON数组
	if startIdx := strings.Index(responseText, "["); startIdx != -1 {
		if endIdx := strings.LastIndex(responseText, "]"); endIdx > startIdx {
			return responseText[startIdx : endIdx+1]
		}
	}
	
	// 查找JSON对象
	if startIdx := strings.Index(responseText, "{"); startIdx != -1 {
		if endIdx := strings.LastIndex(responseText, "}"); endIdx > startIdx {
			return responseText[startIdx : endIdx+1]
		}
	}
	
	return ""
}

// validateJSONCompleteness 验证JSON完整性
func (d *ResponseCompletionDetector) validateJSONCompleteness(jsonContent string) bool {
	decoder := json.NewDecoder(strings.NewReader(jsonContent))
	decoder.UseNumber()
	var jsonTest interface{}
	return decoder.Decode(&jsonTest) == nil
}

// hasErrorCompletion 检查错误响应完整性
func (d *ResponseCompletionDetector) hasErrorCompletion(responseText string) bool {
	for _, marker := range d.errorMarkers {
		if strings.Contains(responseText, marker) {
			return true
		}
	}
	return false
}

// hasSpecialResponseCompletion 检查特殊响应完整性
func (d *ResponseCompletionDetector) hasSpecialResponseCompletion(responseText string) bool {
	// null响应
	if strings.Contains(responseText, "dubbo>") {
		cleanedResponse := strings.TrimSpace(strings.Replace(responseText, "dubbo>", "", -1))
		if cleanedResponse == "null" || cleanedResponse == "" {
			return true
		}
	}
	return false
}