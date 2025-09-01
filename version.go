package main

import (
	"fmt"
	"runtime"
)

// 版本信息变量，在编译时通过 ldflags 注入
var (
	Version   = "dev"      // 版本号
	BuildTime = "unknown"  // 构建时间
	GitCommit = "unknown"  // Git 提交哈希
)

// VersionInfo 版本信息结构
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitCommit string `json:"git_commit"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// GetVersionInfo 获取版本信息
func GetVersionInfo() *VersionInfo {
	return &VersionInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String 返回版本信息的字符串表示
func (v *VersionInfo) String() string {
	return fmt.Sprintf(
		"Dubbo Invoke CLI\n"+
			"Version: %s\n"+
			"Build Time: %s\n"+
			"Git Commit: %s\n"+
			"Go Version: %s\n"+
			"OS/Arch: %s/%s",
		v.Version,
		v.BuildTime,
		v.GitCommit,
		v.GoVersion,
		v.OS,
		v.Arch,
	)
}

// PrintVersion 打印版本信息
func PrintVersion() {
	fmt.Println(GetVersionInfo().String())
}