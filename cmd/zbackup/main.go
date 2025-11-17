package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"zbackup/pkg/core"
	"zbackup/pkg/endpoint"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "zbackup 错误: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		sourcePath   string
		destPath     string
		port         int
		identity     string
		sshOptions   []string
		mode         string
		checksum     string
		noProgress   bool
		logFile      string
		logLevel     string
		excludes     []string
		dryRun       bool
		snapshotName string
	)

	cmd := &cobra.Command{
		Use:   "zbackup",
		Short: "基于 SSH 的增量备份工具",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sourcePath == "" || destPath == "" {
				return errors.New("必须同时指定 --source 与 --dest")
			}
			sshOpts := endpoint.SSHOptions{
				Port:      port,
				Identity:  identity,
				ExtraOpts: sshOptions,
			}
			srcEndpoint, err := endpoint.ParseEndpoint(sourcePath, port, sshOpts)
			if err != nil {
				return err
			}
			destEndpoint, err := endpoint.ParseEndpoint(destPath, port, sshOpts)
			if err != nil {
				return err
			}
			cfg := &core.BackupConfig{
				Source:       srcEndpoint,
				Dest:         destEndpoint,
				Mode:         parseMode(mode),
				Checksum:     parseChecksum(checksum),
				Excludes:     excludes,
				DryRun:       dryRun,
				SnapshotName: snapshotName,
				LogFile:      logFile,
				LogLevel:     logLevel,
				NoProgress:   noProgress,
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return core.Run(ctx, cfg)
		},
	}

	cmd.Flags().StringVarP(&sourcePath, "source", "s", "", "源路径 (本地路径或 user@host:/path)")
	cmd.Flags().StringVarP(&destPath, "dest", "d", "", "目标路径 (本地路径或 user@host:/path)")
	cmd.Flags().IntVarP(&port, "port", "p", 22, "SSH 端口")
	cmd.Flags().StringVarP(&identity, "identity", "i", "", "SSH 私钥路径")
	cmd.Flags().StringArrayVarP(&sshOptions, "ssh-option", "o", nil, "透传 ssh 参数，可多次指定")
	cmd.Flags().StringVarP(&mode, "mode", "m", string(endpoint.ModeIncr), "备份模式：full / incr")
	cmd.Flags().StringVar(&checksum, "checksum", string(endpoint.ChecksumSHA256), "校验算法：none / md5 / sha1 / sha256")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "禁用进度条显示")
	cmd.Flags().StringVar(&logFile, "log-file", "", "指定日志文件，不填则写入目标端 .zbackup/logs/")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "日志级别：debug / info / warn / error")
	cmd.Flags().StringArrayVar(&excludes, "exclude", nil, "排除模式，可多次指定")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "演示模式，不执行真正传输")
	cmd.Flags().StringVar(&snapshotName, "snapshot-name", "", "自定义快照名，默认为当前 UTC 时间戳")

	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("dest")
	return cmd
}

func parseMode(val string) endpoint.BackupMode {
	switch endpoint.BackupMode(val) {
	case endpoint.ModeFull:
		return endpoint.ModeFull
	default:
		return endpoint.ModeIncr
	}
}

func parseChecksum(val string) endpoint.ChecksumAlgo {
	switch endpoint.ChecksumAlgo(val) {
	case endpoint.ChecksumNone:
		return endpoint.ChecksumNone
	case endpoint.ChecksumMD5:
		return endpoint.ChecksumMD5
	case endpoint.ChecksumSHA1:
		return endpoint.ChecksumSHA1
	default:
		return endpoint.ChecksumSHA256
	}
}
