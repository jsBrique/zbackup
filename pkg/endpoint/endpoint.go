package endpoint

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// EndpointType 表示端点类型：本地或远程
type EndpointType int

const (
	EndpointLocal EndpointType = iota
	EndpointRemote
)

// BackupMode 定义备份模式
type BackupMode string

const (
	ModeFull BackupMode = "full"
	ModeIncr BackupMode = "incr"
)

// ChecksumAlgo 定义校验算法类型
type ChecksumAlgo string

const (
	ChecksumNone   ChecksumAlgo = "none"
	ChecksumMD5    ChecksumAlgo = "md5"
	ChecksumSHA1   ChecksumAlgo = "sha1"
	ChecksumSHA256 ChecksumAlgo = "sha256"
)

// SSHOptions 对应用户通过 CLI 传入的 ssh 相关参数
type SSHOptions struct {
	Port      int
	Identity  string
	ExtraOpts []string
}

// Endpoint 表示备份操作中的一端
type Endpoint struct {
	Type    EndpointType
	User    string
	Host    string
	Path    string
	SSHOpts SSHOptions
}

var remotePattern = regexp.MustCompile(`^([a-zA-Z0-9_\-\.]+@)?[^:]+:.+`)

// ParseEndpoint 根据 CLI 入参自动识别本地或远程路径
func ParseEndpoint(raw string, port int, sshOpts SSHOptions) (Endpoint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Endpoint{}, errors.New("空路径")
	}

	if remotePattern.MatchString(raw) {
		colonIdx := strings.Index(raw, ":")
		if colonIdx <= 0 || colonIdx == len(raw)-1 {
			return Endpoint{}, fmt.Errorf("远端路径格式非法: %s", raw)
		}
		userHost := raw[:colonIdx]
		destPath := raw[colonIdx+1:]
		user := ""
		host := userHost
		if strings.Contains(userHost, "@") {
			parts := strings.SplitN(userHost, "@", 2)
			user = parts[0]
			host = parts[1]
		}
		if sshOpts.Port == 0 {
			sshOpts.Port = port
		}
		return Endpoint{
			Type: EndpointRemote,
			User: user,
			Host: host,
			Path: destPath,
			SSHOpts: SSHOptions{
				Port:      sshOpts.Port,
				Identity:  sshOpts.Identity,
				ExtraOpts: append([]string{}, sshOpts.ExtraOpts...),
			},
		}, nil
	}

	absPath := raw
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Clean(raw)
	}

	return Endpoint{
		Type: EndpointLocal,
		Path: absPath,
	}, nil
}

// Join 构造子路径，兼容不同端点
func (e Endpoint) Join(elem ...string) string {
	if e.Type == EndpointRemote {
		return path.Join(append([]string{e.Path}, elem...)...)
	}
	return filepath.Join(append([]string{e.Path}, elem...)...)
}

// DisplayName 返回用于日志/进度显示的端点名
func (e Endpoint) DisplayName() string {
	if e.Type == EndpointRemote {
		target := e.Host
		if e.User != "" {
			target = fmt.Sprintf("%s@%s", e.User, e.Host)
		}
		return fmt.Sprintf("%s:%s", target, e.Path)
	}
	return e.Path
}
