package endpoint

import "errors"

// ErrHashCommandUnavailable 表示远端不支持某个 hash 命令
var ErrHashCommandUnavailable = errors.New("hash command unavailable")

// RemoteHashFS 表示具备在远端执行 hash 命令能力的文件系统
type RemoteHashFS interface {
	ComputeRemoteHash(relPath string, algo ChecksumAlgo) ([]byte, error)
}

