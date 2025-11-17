package core

import (
	"path/filepath"
	"sort"
	"strings"

	"zbackup/pkg/endpoint"
	"zbackup/pkg/meta"
	"zbackup/pkg/transfer"
)

// BuildPlan 根据当前扫描结果与上一快照生成增量计划
func BuildPlan(files []endpoint.FileMeta, last *meta.Snapshot, cfg BackupConfig) transfer.Plan {
	current := make(map[string]endpoint.FileMeta, len(files))
	var dirs []endpoint.FileMeta
	var fileMetas []endpoint.FileMeta
	for _, meta := range files {
		rel := normRel(meta.RelPath)
		meta.RelPath = rel
		current[rel] = meta
		if meta.IsDir {
			dirs = append(dirs, meta)
		} else {
			fileMetas = append(fileMetas, meta)
		}
	}
	sort.Slice(dirs, func(i, j int) bool {
		if depth(dirs[i].RelPath) == depth(dirs[j].RelPath) {
			return dirs[i].RelPath < dirs[j].RelPath
		}
		return depth(dirs[i].RelPath) < depth(dirs[j].RelPath)
	})
	sort.Slice(fileMetas, func(i, j int) bool { return fileMetas[i].RelPath < fileMetas[j].RelPath })

	plan := transfer.Plan{}
	for _, dir := range dirs {
		if shouldSkip(dir.RelPath, dir, last, cfg) {
			continue
		}
		plan.AddItem(transfer.TransferItem{
			RelPath: dir.RelPath,
			Meta:    dir,
			Action:  transfer.ActionMkdir,
		})
	}
	action := transfer.ActionUpload
	if cfg.Source.Type == endpoint.EndpointRemote {
		action = transfer.ActionDownload
	}
	for _, meta := range fileMetas {
		if shouldSkip(meta.RelPath, meta, last, cfg) {
			plan.AddItem(transfer.TransferItem{
				RelPath: meta.RelPath,
				Meta:    meta,
				Action:  transfer.ActionSkip,
				Reason:  "文件未变化",
			})
			continue
		}
		plan.AddItem(transfer.TransferItem{
			RelPath: meta.RelPath,
			Meta:    meta,
			Action:  action,
		})
	}
	if last != nil && cfg.Mode == endpoint.ModeFull {
		var deleteFiles []transfer.TransferItem
		var deleteDirs []transfer.TransferItem
		for rel, old := range last.Files {
			if _, ok := current[rel]; ok {
				continue
			}
			item := transfer.TransferItem{
				RelPath: rel,
				Meta:    old,
				Action:  transfer.ActionDelete,
			}
			if old.IsDir {
				deleteDirs = append(deleteDirs, item)
			} else {
				deleteFiles = append(deleteFiles, item)
			}
		}
		sort.Slice(deleteFiles, func(i, j int) bool { return deleteFiles[i].RelPath < deleteFiles[j].RelPath })
		sort.Slice(deleteDirs, func(i, j int) bool {
			if depth(deleteDirs[i].RelPath) == depth(deleteDirs[j].RelPath) {
				return deleteDirs[i].RelPath > deleteDirs[j].RelPath
			}
			return depth(deleteDirs[i].RelPath) > depth(deleteDirs[j].RelPath)
		})
		for _, item := range deleteFiles {
			plan.AddItem(item)
		}
		for _, item := range deleteDirs {
			plan.AddItem(item)
		}
	}
	return plan
}

func shouldSkip(rel string, meta endpoint.FileMeta, last *meta.Snapshot, cfg BackupConfig) bool {
	if meta.IsDir {
		if last == nil {
			return false
		}
		old, ok := last.Files[rel]
		return ok && old.IsDir
	}
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

func depth(rel string) int {
	if rel == "" {
		return 0
	}
	return strings.Count(rel, "/") + 1
}
