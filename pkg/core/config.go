package core

import (
	"fmt"
	"time"

	"zbackup/pkg/endpoint"
)

// BackupConfig 表示一次备份任务的配置
type BackupConfig struct {
	Source       endpoint.Endpoint
	Dest         endpoint.Endpoint
	Mode         endpoint.BackupMode
	Checksum     endpoint.ChecksumAlgo
	Excludes     []string
	DryRun       bool
	SnapshotName string
	LogFile      string
	LogLevel     string
	NoProgress   bool
}

// Validate 进行基础校验
func (c *BackupConfig) Validate() error {
	if c.Source.Type == endpoint.EndpointRemote && c.Dest.Type == endpoint.EndpointRemote {
		return fmt.Errorf("暂不支持远端到远端的备份")
	}
	if c.Source.Path == "" || c.Dest.Path == "" {
		return fmt.Errorf("源和目标路径均不能为空")
	}
	if c.SnapshotName == "" {
		c.SnapshotName = time.Now().UTC().Format("20060102T150405Z")
	}
	return nil
}
