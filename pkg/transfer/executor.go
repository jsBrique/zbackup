package transfer

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"os"

	"zbackup/pkg/endpoint"
	"zbackup/pkg/ui"
)

// Executor 负责执行传输计划
type Executor struct {
	SourceFS endpoint.FileSystem
	DestFS   endpoint.FileSystem
	Src      endpoint.Endpoint
	Dst      endpoint.Endpoint
	Checksum endpoint.ChecksumAlgo
	Logger   *slog.Logger
	Progress ui.Progress
}

// Result 描述执行结果
type Result struct {
	Success map[string]endpoint.FileMeta
	Failed  map[string]error
}

// Execute 执行计划
func (e *Executor) Execute(ctx context.Context, plan Plan) (Result, error) {
	result := Result{
		Success: make(map[string]endpoint.FileMeta),
		Failed:  make(map[string]error),
	}
	e.Progress.Start(plan.TotalFiles, plan.TotalBytes)
	var errs []error
	for _, item := range plan.Items {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}
		switch item.Action {
		case ActionUpload, ActionDownload:
			e.Progress.NextFile(item.RelPath, item.Meta.Size)
			if meta, err := e.copyFile(item); err != nil {
				e.Logger.Error("传输失败", "path", item.RelPath, "err", err)
				errs = append(errs, err)
				result.Failed[item.RelPath] = err
			} else {
				e.Logger.Info("传输完成", "path", item.RelPath, "size", item.Meta.Size)
				result.Success[item.RelPath] = meta
			}
		case ActionDelete:
			if err := e.DestFS.Remove(item.RelPath); err != nil {
				e.Logger.Warn("删除失败", "path", item.RelPath, "err", err)
			}
		case ActionSkip:
			e.Logger.Debug("跳过未变化文件", "path", item.RelPath, "reason", item.Reason)
		}
	}
	e.Progress.Finish()
	if len(errs) > 0 {
		return result, fmt.Errorf("%d 个文件传输失败", len(errs))
	}
	return result, nil
}

func (e *Executor) copyFile(item TransferItem) (endpoint.FileMeta, error) {
	reader, err := e.SourceFS.Open(item.RelPath)
	if err != nil {
		return endpoint.FileMeta{}, fmt.Errorf("读取源文件失败: %w", err)
	}
	defer reader.Close()

	perm := os.FileMode(item.Meta.Mode)
	if perm == 0 {
		perm = 0o644
	}
	writer, err := e.DestFS.Create(item.RelPath, perm)
	if err != nil {
		return endpoint.FileMeta{}, fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer writer.Close()

	var writers []io.Writer
	writers = append(writers, writer, progressWriter{progress: e.Progress})
	var srcHash hash.Hash
	if e.Checksum != endpoint.ChecksumNone {
		srcHash = newHash(e.Checksum)
		if srcHash != nil {
			writers = append(writers, srcHash)
		}
	}
	multi := io.MultiWriter(writers...)
	if _, err := io.Copy(multi, reader); err != nil {
		return endpoint.FileMeta{}, err
	}
	if e.Checksum == endpoint.ChecksumNone || srcHash == nil {
		return item.Meta, nil
	}
	srcSum := srcHash.Sum(nil)
	destSum, err := e.computeChecksumOnDest(item.RelPath)
	if err != nil {
		return endpoint.FileMeta{}, err
	}
	if !equalBytes(srcSum, destSum) {
		return endpoint.FileMeta{}, fmt.Errorf("校验失败: %s", item.RelPath)
	}
	meta := item.Meta
	meta.Checksum = fmt.Sprintf("%x", srcSum)
	return meta, nil
}

func (e *Executor) computeChecksumOnDest(relPath string) ([]byte, error) {
	reader, err := e.DestFS.Open(relPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	h := newHash(e.Checksum)
	if h == nil {
		return nil, fmt.Errorf("未知校验算法: %s", e.Checksum)
	}
	if _, err := io.Copy(h, reader); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func newHash(algo endpoint.ChecksumAlgo) hash.Hash {
	switch algo {
	case endpoint.ChecksumMD5:
		return md5.New()
	case endpoint.ChecksumSHA1:
		return sha1.New()
	case endpoint.ChecksumSHA256:
		return sha256.New()
	default:
		return nil
	}
}

type progressWriter struct {
	progress ui.Progress
}

func (p progressWriter) Write(b []byte) (int, error) {
	p.progress.AddBytes(int64(len(b)))
	return len(b), nil
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
