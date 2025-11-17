package core

import (
	"path/filepath"

	"zbackup/pkg/endpoint"
	"zbackup/pkg/meta"
	"zbackup/pkg/transfer"
)

// BuildPlan 根据当前扫描结果与上一快照生成增量计划
func BuildPlan(files []endpoint.FileMeta, last *meta.Snapshot, cfg BackupConfig) transfer.Plan {
	current := make(map[string]endpoint.FileMeta, len(files))
	for _, meta := range files {
		current[normRel(meta.RelPath)] = meta
	}
	plan := transfer.Plan{}
	action := transfer.ActionUpload
	if cfg.Source.Type == endpoint.EndpointRemote {
		action = transfer.ActionDownload
	}
	for rel, meta := range current {
		if shouldSkip(rel, meta, last, cfg) {
			plan.AddItem(transfer.TransferItem{
				RelPath: rel,
				Meta:    meta,
				Action:  transfer.ActionSkip,
				Reason:  "文件未变化",
			})
			continue
		}
		meta.RelPath = rel
		plan.AddItem(transfer.TransferItem{
			RelPath: rel,
			Meta:    meta,
			Action:  action,
		})
	}
	if last != nil && cfg.Mode == endpoint.ModeFull {
		for rel := range last.Files {
			if _, ok := current[rel]; !ok {
				plan.AddItem(transfer.TransferItem{
					RelPath: rel,
					Action:  transfer.ActionDelete,
				})
			}
		}
	}
	return plan
}

func shouldSkip(rel string, meta endpoint.FileMeta, last *meta.Snapshot, cfg BackupConfig) bool {
	if cfg.Mode == endpoint.ModeFull || last == nil {
		return false
	}
	old, ok := last.Files[rel]
	if !ok {
		return false
	}
	if old.Size != meta.Size {
		return false
	}
	if !old.ModTime.Equal(meta.ModTime) {
		return false
	}
	if cfg.Checksum != endpoint.ChecksumNone && old.Checksum != "" && meta.Checksum != "" && old.Checksum != meta.Checksum {
		return false
	}
	return true
}

func normRel(rel string) string {
	return filepath.ToSlash(rel)
}
