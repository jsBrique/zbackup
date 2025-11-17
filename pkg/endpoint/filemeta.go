package endpoint

import "time"

// FileMeta 描述文件元数据
type FileMeta struct {
	RelPath  string    `json:"rel_path"`
	Size     int64     `json:"size"`
	Mode     uint32    `json:"mode"`
	ModTime  time.Time `json:"mod_time"`
	Checksum string    `json:"checksum,omitempty"`
}
