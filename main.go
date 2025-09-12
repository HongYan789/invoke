package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"
	buildTime = "unknown"
)

func main() {
	// 如果没有参数，默认启动web模式
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "web")
		
		// 在Windows平台下显示启动信息并设置信号处理
		if runtime.GOOS == "windows" {
			// 显示启动信息
			fmt.Println(strings.Repeat("=", 60))
			color.Green("🚀 Dubbo Invoke Web UI 正在启动...")
			color.Cyan("📱 Web界面将在浏览器中自动打开")
			color.Yellow("⚠️  请保持此窗口打开以维持服务运行")
			color.Red("❌ 关闭此窗口将停止Web服务")
			fmt.Println(strings.Repeat("=", 60))
			fmt.Println()
			
			// 设置信号处理，防止Windows下控制台窗口自动关闭
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			
			// 启动Web服务器
			webStarted := make(chan bool, 1)
			go func() {
				// 直接调用web命令
				webCmd := newWebCommand()
				webStarted <- true
				if err := webCmd.RunE(webCmd, []string{}); err != nil {
					color.Red("错误: %v", err)
					os.Exit(1)
				}
			}()
			
			// 等待Web服务器启动
			<-webStarted
			time.Sleep(2 * time.Second)
			
			// 延迟打开浏览器
			go func() {
				time.Sleep(1 * time.Second)
				openBrowser("http://localhost:8080")
				// 浏览器打开后显示服务运行状态
				time.Sleep(1 * time.Second)
				color.Green("✅ 浏览器已打开，Web服务正在运行中...")
				color.Cyan("💡 按 Ctrl+C 可安全停止服务")
			}()
			
			// 启动保活机制
			go func() {
				for {
					time.Sleep(30 * time.Second)
					color.Green("💓 Web服务运行中... (按 Ctrl+C 停止)")
				}
			}()
			
			// 等待中断信号
			<-c
			color.Yellow("\n🛑 正在安全停止Web服务...")
			os.Exit(0)
		}
	}

	rootCmd := createRootCommand()

	if err := rootCmd.Execute(); err != nil {
		color.Red("错误: %v", err)
		os.Exit(1)
	}
}

// createRootCommand 创建根命令
func createRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "dubbo-invoke",
		Short: "Dubbo接口调用工具",
		Long: `一个用于调用Dubbo服务的命令行工具，支持：
- 连接多种注册中心（Zookeeper、Nacos等）
- 泛化调用任意Dubbo服务
- 智能参数解析和示例生成
- 灵活的配置管理
- List类型返回结果自动处理

双击exe文件将自动启动Web UI模式`,
		Version: fmt.Sprintf("%s (built at %s)", version, buildTime),
	}

	// 添加子命令
	rootCmd.AddCommand(newInvokeCommand())
	rootCmd.AddCommand(newListCommand())
	rootCmd.AddCommand(newConfigCommand())
	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newWebCommand())
	rootCmd.AddCommand(newTestNacosCommand())

	// 全局标志
	rootCmd.PersistentFlags().StringP("config", "c", "config.yaml", "配置文件路径")
	rootCmd.PersistentFlags().StringP("registry", "r", "zookeeper://127.0.0.1:2181", "注册中心地址")
	rootCmd.PersistentFlags().StringP("app", "a", "dubbo-invoke-client", "应用名称")
	rootCmd.PersistentFlags().IntP("timeout", "t", 3000, "调用超时时间(毫秒)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "详细输出")

	return rootCmd
}

// invoke命令 - 调用Dubbo服务
func newInvokeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoke [service] [method] [params...] | [expression]",
		Short: "调用Dubbo服务方法",
		Long: `调用指定的Dubbo服务方法

示例:
  # 传统格式
  dubbo-invoke invoke com.example.UserService getUserById 123
  dubbo-invoke invoke com.example.UserService createUser '{"name":"张三","age":25}'
  
  # 新格式（表达式）
  dubbo-invoke invoke 'com.example.UserService.getUserById(123)'
  dubbo-invoke invoke 'com.jzt.zhcai.user.companyinfo.CompanyInfoDubboApi.getCompanyInfoFromDb({"class":"com.jzt.zhcai.user.companyinfo.dto.request.UserCompanyInfoDetailReq","companyId":1})'`,
		Args: cobra.MinimumNArgs(1),
		RunE: runInvokeCommand,
	}

	cmd.Flags().StringP("version", "V", "", "服务版本")
	cmd.Flags().StringP("group", "g", "", "服务分组")
	cmd.Flags().BoolP("generic", "G", true, "使用泛化调用")
	cmd.Flags().StringSliceP("types", "T", nil, "参数类型列表")
	cmd.Flags().BoolP("example", "e", false, "生成示例参数")

	return cmd
}

// openBrowser 在不同操作系统上打开默认浏览器
func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	default:
		return
	}

	exec.Command(cmd, args...).Start()
}

// test-nacos命令 - 测试Nacos注册中心连接
func newTestNacosCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test-nacos",
		Short: "测试Nacos注册中心连接",
		Long: `测试与Nacos注册中心的连接，包括：
- 连接测试
- 获取服务列表
- 查询服务详情
- 验证服务实例状态`,
		RunE: func(cmd *cobra.Command, args []string) error {
			color.Green("开始测试Nacos注册中心连接...")
			TestNacosRegistry()
			return nil
		},
	}
}

// version命令 - 显示版本信息
func newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Long:  "显示详细的版本信息，包括构建时间和Git提交哈希",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("版本: %s\n", version)
			fmt.Printf("构建时间: %s\n", buildTime)
		},
	}

	return cmd
}

// list命令 - 列出可用服务
func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [service]",
		Short: "列出可用的Dubbo服务",
		Long: `列出注册中心中可用的Dubbo服务和方法

示例:
  dubbo-invoke list                           # 列出所有服务
  dubbo-invoke list com.example.UserService  # 列出指定服务的方法`,
		RunE: runListCommand,
	}

	cmd.Flags().BoolP("methods", "m", false, "显示服务方法")
	cmd.Flags().StringP("filter", "f", "", "过滤服务名称")

	return cmd
}

// config命令 - 配置管理
func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "配置管理",
		Long:  `管理dubbo-invoke的配置文件`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "初始化配置文件",
		RunE:  runConfigInitCommand,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "显示当前配置",
		RunE:  runConfigShowCommand,
	})

	return cmd
}