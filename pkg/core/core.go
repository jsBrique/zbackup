package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"zbackup/pkg/endpoint"
	"zbackup/pkg/logging"
	"zbackup/pkg/meta"
	"zbackup/pkg/transfer"
	"zbackup/pkg/ui"
)

// Run 执行一次备份
func Run(ctx context.Context, cfg *BackupConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	srcFS, err := buildFS(&cfg.Source)
	if err != nil {
		return err
	}
	defer srcFS.Close()
	destFS, err := buildFS(&cfg.Dest)
	if err != nil {
		return err
	}
	defer destFS.Close()

	store := meta.NewStore(destFS)
	lastSnap, err := store.LoadLatest()
	if err != nil {
		return fmt.Errorf("读取历史快照失败: %w", err)
	}
	pendingSnap, err := store.LoadPending()
	if err != nil {
		return fmt.Errorf("读取未完成快照失败: %w", err)
	}
	baseSnap := lastSnap
	if pendingSnap != nil {
		cfg.SnapshotName = pendingSnap.Name
		baseSnap = pendingSnap
	}

	srcFiles, err := srcFS.List(cfg.Excludes)
	if err != nil {
		return fmt.Errorf("扫描源目录失败: %w", err)
	}

	plan := BuildPlan(srcFiles, baseSnap, *cfg)

	logWriter, logPath, err := prepareLogWriter(cfg, destFS)
	if err != nil {
		return err
	}
	var progress ui.Progress
	stdWriter := io.Writer(os.Stdout)
	if cfg.NoProgress {
		progress = ui.NoopProgress{}
	} else {
		bar := ui.NewBarProgress(os.Stdout)
		progress = bar
		stdWriter = bar.WrapWriter(os.Stdout)
	}
	var logWriters []io.Writer
	logWriters = append(logWriters, stdWriter)
	if logWriter != nil {
		logWriters = append(logWriters, logWriter)
	}
	logger, err := logging.New(cfg.LogLevel, logWriters...)
	if err != nil {
		return err
	}
	defer logger.Close()

	if logPath != "" {
		logger.Info("日志写入路径", "dest", logPath)
	}
	if cfg.DryRun {
		logger.Info("Dry-run 模式，只展示计划", "files", plan.TotalFiles, "bytes", plan.TotalBytes)
		for _, item := range plan.Items {
			logger.Info("计划条目", "action", item.Action, "path", item.RelPath, "size", item.Meta.Size, "reason", item.Reason)
		}
		return nil
	}

	checkpoint := newCheckpoint(store, baseSnap, cfg.SnapshotName, cfg.Source, cfg.Dest)

	executor := transfer.Executor{
		SourceFS: srcFS,
		DestFS:   destFS,
		Src:      cfg.Source,
		Dst:      cfg.Dest,
		Checksum: cfg.Checksum,
		Logger:   logger.Logger,
		Progress: progress,
		OnSuccess: func(item transfer.TransferItem, meta endpoint.FileMeta) {
			if err := checkpoint.Record(meta); err != nil {
				logger.Warn("写入增量进度失败", "path", meta.RelPath, "err", err)
			}
		},
	}

	result, execErr := executor.Execute(ctx, plan)
	if err := checkpoint.Flush(); err != nil {
		logger.Warn("刷新进度失败", "err", err)
	}
	if execErr != nil {
		logger.Error("备份过程中出现错误", "err", execErr)
	}

	finalFiles := mergeSnapshot(baseSnap, plan, result)
	snapshot := meta.Snapshot{
		Name:       cfg.SnapshotName,
		CreatedAt:  time.Now().UTC(),
		SourceRoot: cfg.Source.Path,
		DestRoot:   cfg.Dest.Path,
		Files:      finalFiles,
		Completed:  execErr == nil,
	}
	if err := store.Save(snapshot); err != nil {
		logger.Error("保存快照失败", "err", err)
		return err
	}
	if execErr != nil {
		logger.Warn("备份未完成，保留进度以供继续", "snapshot", snapshot.Name)
		return execErr
	}
	if err := store.ClearPending(); err != nil {
		logger.Warn("清理未完成快照失败", "err", err)
	}
	logger.Info("备份完成", "snapshot", snapshot.Name, "files", len(snapshot.Files))
	return nil
}

func mergeSnapshot(last *meta.Snapshot, plan transfer.Plan, result transfer.Result) map[string]endpoint.FileMeta {
	final := make(map[string]endpoint.FileMeta)
	if last != nil {
		for rel, meta := range last.Files {
			final[rel] = meta
		}
	}
	for rel, meta := range result.Success {
		meta.RelPath = rel
		final[rel] = meta
	}
	for _, item := range plan.Items {
		switch item.Action {
		case transfer.ActionDelete:
			delete(final, item.RelPath)
		case transfer.ActionSkip:
			if last != nil {
				if meta, ok := last.Files[item.RelPath]; ok {
					final[item.RelPath] = meta
				}
			}
		}
	}
	return final
}

func buildFS(ep *endpoint.Endpoint) (endpoint.FileSystem, error) {
	switch ep.Type {
	case endpoint.EndpointLocal:
		abs, err := filepath.Abs(ep.Path)
		if err != nil {
			return nil, err
		}
		ep.Path = abs
		return endpoint.NewLocalFS(abs), nil
	case endpoint.EndpointRemote:
		return endpoint.NewRemoteFS(*ep), nil
	default:
		return nil, fmt.Errorf("未知端点类型")
	}
}

func prepareLogWriter(cfg *BackupConfig, destFS endpoint.FileSystem) (io.WriteCloser, string, error) {
	if cfg.LogFile != "" {
		file, err := os.Create(cfg.LogFile)
		if err != nil {
			return nil, "", err
		}
		return file, cfg.LogFile, nil
	}
	logRel := filepath.Join(".zbackup", "logs", fmt.Sprintf("backup-%s.log", cfg.SnapshotName))
	writer, err := destFS.Create(logRel, 0o644)
	if err != nil {
		return nil, "", err
	}
	return writer, logRel, nil
}
