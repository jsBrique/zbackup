package transfer

import "zbackup/pkg/endpoint"

// TransferAction 定义传输动作类型
type TransferAction string

const (
	ActionUpload   TransferAction = "upload"
	ActionDownload TransferAction = "download"
	ActionDelete   TransferAction = "delete"
	ActionSkip     TransferAction = "skip"
	ActionMkdir    TransferAction = "mkdir"
)

// TransferItem 表示一次对单个文件的操作
type TransferItem struct {
	RelPath string
	Meta    endpoint.FileMeta
	Action  TransferAction
	Reason  string
}

// Plan 描述所有需要操作的集合
type Plan struct {
	Items      []TransferItem
	TotalBytes int64
	TotalFiles int
}

// AddItem 加入计划
func (p *Plan) AddItem(item TransferItem) {
	p.Items = append(p.Items, item)
	if item.Action == ActionDelete || item.Action == ActionSkip || item.Action == ActionMkdir {
		return
	}
	p.TotalFiles++
	p.TotalBytes += item.Meta.Size
}
