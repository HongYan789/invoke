package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Registry    RegistryConfig `yaml:"registry" mapstructure:"registry"`
	Application AppConfig      `yaml:"application" mapstructure:"application"`
	Defaults    DefaultConfig  `yaml:"defaults" mapstructure:"defaults"`
}

// RegistryConfig 注册中心配置
type RegistryConfig struct {
	Address  string `yaml:"address" mapstructure:"address"`
	Protocol string `yaml:"protocol" mapstructure:"protocol"`
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	Timeout  string `yaml:"timeout" mapstructure:"timeout"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name    string `yaml:"name" mapstructure:"name"`
	Version string `yaml:"version" mapstructure:"version"`
}

// DefaultConfig 默认配置
type DefaultConfig struct {
	Timeout  string `yaml:"timeout" mapstructure:"timeout"`
	Protocol string `yaml:"protocol" mapstructure:"protocol"`
	Version  string `yaml:"version" mapstructure:"version"`
	Group    string `yaml:"group" mapstructure:"group"`
}

// ConfigManager 配置管理器
type ConfigManager struct {
	configPath string
	config     *Config
}

// NewConfigManager 创建配置管理器
func NewConfigManager() *ConfigManager {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".dubbo-invoke", "config.yaml")
	
	return &ConfigManager{
		configPath: configPath,
		config:     getDefaultConfig(),
	}
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() *Config {
	return &Config{
		Registry: RegistryConfig{
			Address:  "zookeeper://127.0.0.1:2181",
			Protocol: "zookeeper",
			Timeout:  "5s",
		},
		Application: AppConfig{
			Name:    "dubbo-invoke-cli",
			Version: "1.0.0",
		},
		Defaults: DefaultConfig{
			Timeout:  "3s",
			Protocol: "dubbo",
			Version:  "",
			Group:    "",
		},
	}
}

// LoadConfig 加载配置
func (cm *ConfigManager) LoadConfig() error {
	// 检查配置文件是否存在
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// 配置文件不存在，创建默认配置
		return cm.SaveConfig()
	}

	// 使用viper加载配置
	viper.SetConfigFile(cm.configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析配置到结构体
	if err := viper.Unmarshal(cm.config); err != nil {
		return fmt.Errorf("解析配置失败: %v", err)
	}

	return nil
}

// SaveConfig 保存配置
func (cm *ConfigManager) SaveConfig() error {
	// 确保配置目录存在
	configDir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 将配置序列化为YAML
	data, err := yaml.Marshal(cm.config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// GetConfig 获取配置
func (cm *ConfigManager) GetConfig() *Config {
	return cm.config
}

// SetRegistryAddress 设置注册中心地址
func (cm *ConfigManager) SetRegistryAddress(address string) {
	cm.config.Registry.Address = address
}

// SetRegistryAuth 设置注册中心认证
func (cm *ConfigManager) SetRegistryAuth(username, password string) {
	cm.config.Registry.Username = username
	cm.config.Registry.Password = password
}

// SetDefaultTimeout 设置默认超时时间
func (cm *ConfigManager) SetDefaultTimeout(timeout string) {
	cm.config.Defaults.Timeout = timeout
}

// SetDefaultVersion 设置默认版本
func (cm *ConfigManager) SetDefaultVersion(version string) {
	cm.config.Defaults.Version = version
}

// SetDefaultGroup 设置默认分组
func (cm *ConfigManager) SetDefaultGroup(group string) {
	cm.config.Defaults.Group = group
}

// GetDubboConfig 获取Dubbo客户端配置
func (cm *ConfigManager) GetDubboConfig() *DubboConfig {
	timeout, _ := time.ParseDuration(cm.config.Defaults.Timeout)
	
	return &DubboConfig{
		Registry:    cm.config.Registry.Address,
		Application: cm.config.Application.Name,
		Timeout:     timeout,
		Version:     cm.config.Defaults.Version,
		Group:       cm.config.Defaults.Group,
		Protocol:    cm.config.Defaults.Protocol,
		Username:    cm.config.Registry.Username,
		Password:    cm.config.Registry.Password,
	}
}

// ShowConfig 显示当前配置
func (cm *ConfigManager) ShowConfig() (string, error) {
	data, err := yaml.Marshal(cm.config)
	if err != nil {
		return "", fmt.Errorf("序列化配置失败: %v", err)
	}
	return string(data), nil
}

// ValidateConfig 验证配置
func (cm *ConfigManager) ValidateConfig() error {
	if cm.config.Registry.Address == "" {
		return fmt.Errorf("注册中心地址不能为空")
	}

	if cm.config.Application.Name == "" {
		return fmt.Errorf("应用名称不能为空")
	}

	// 验证超时时间格式
	if _, err := time.ParseDuration(cm.config.Defaults.Timeout); err != nil {
		return fmt.Errorf("无效的超时时间格式: %s", cm.config.Defaults.Timeout)
	}

	return nil
}

// GetConfigPath 获取配置文件路径
func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}

// ResetConfig 重置为默认配置
func (cm *ConfigManager) ResetConfig() {
	cm.config = getDefaultConfig()
}