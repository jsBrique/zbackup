package endpoint

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

// RemoteFS 使用 ssh 在远端执行命令
type RemoteFS struct {
	endpoint Endpoint
}

// NewRemoteFS 创建远端文件系统
func NewRemoteFS(ep Endpoint) *RemoteFS {
	return &RemoteFS{endpoint: ep}
}

func (r *RemoteFS) Root() string {
	return r.endpoint.Path
}

func (r *RemoteFS) List(excludes []string) ([]FileMeta, error) {
	script := fmt.Sprintf("cd %s && find . -type f -printf '%%P|%%s|%%T@|%%m\\n'", shellQuote(r.endpoint.Path))
	output, err := r.runSSHCommand(script)
	if err != nil {
		return nil, err
	}
	var metas []FileMeta
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		rel := parts[0]
		if rel == "" || rel == "." {
			continue
		}
		if shouldExclude(rel, excludes) {
			continue
		}
		size, _ := strconv.ParseInt(parts[1], 10, 64)
		modEpoch, _ := strconv.ParseFloat(parts[2], 64)
		mode64, _ := strconv.ParseUint(parts[3], 8, 32)
		metas = append(metas, FileMeta{
			RelPath: rel,
			Size:    size,
			Mode:    uint32(mode64),
			ModTime: time.Unix(int64(modEpoch), 0),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return metas, nil
}

func (r *RemoteFS) Open(relPath string) (io.ReadCloser, error) {
	remote := path.Join(r.endpoint.Path, filepathToPosix(relPath))
	cmd := r.sshCommand(fmt.Sprintf("cat %s", shellQuote(remote)))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &cmdReadCloser{Cmd: cmd, Reader: stdout}, nil
}

func (r *RemoteFS) Create(relPath string, perm fs.FileMode) (io.WriteCloser, error) {
	remote := path.Join(r.endpoint.Path, filepathToPosix(relPath))
	dir := path.Dir(remote)
	script := fmt.Sprintf("set -euo pipefail; mkdir -p %s; cat > %s; chmod %04o %s",
		shellQuote(dir), shellQuote(remote), perm&0o777, shellQuote(remote))
	cmd := r.sshCommand(script)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &cmdWriteCloser{Cmd: cmd, Writer: stdin}, nil
}

func (r *RemoteFS) MkdirAll(relPath string) error {
	remote := path.Join(r.endpoint.Path, filepathToPosix(relPath))
	_, err := r.runSSHCommand(fmt.Sprintf("mkdir -p %s", shellQuote(remote)))
	return err
}

func (r *RemoteFS) Remove(relPath string) error {
	remote := path.Join(r.endpoint.Path, filepathToPosix(relPath))
	_, err := r.runSSHCommand(fmt.Sprintf("rm -rf %s", shellQuote(remote)))
	return err
}

func (r *RemoteFS) Stat(relPath string) (FileMeta, error) {
	remote := path.Join(r.endpoint.Path, filepathToPosix(relPath))
	script := fmt.Sprintf("stat -c '%%s|%%Y|%%f' %s", shellQuote(remote))
	out, err := r.runSSHCommand(script)
	if err != nil {
		return FileMeta{}, err
	}
	line := strings.TrimSpace(string(out))
	parts := strings.Split(line, "|")
	if len(parts) < 3 {
		return FileMeta{}, fmt.Errorf("stat output invalid: %s", line)
	}
	size, _ := strconv.ParseInt(parts[0], 10, 64)
	mod, _ := strconv.ParseInt(parts[1], 10, 64)
	mode, _ := strconv.ParseUint(parts[2], 16, 32)
	return FileMeta{
		RelPath: relPath,
		Size:    size,
		Mode:    uint32(mode),
		ModTime: time.Unix(mod, 0),
	}, nil
}

func (r *RemoteFS) runSSHCommand(cmd string) ([]byte, error) {
	command := r.sshCommand(cmd)
	return command.CombinedOutput()
}

func (r *RemoteFS) sshCommand(cmd string) *exec.Cmd {
	args := buildSSHArgs(r.endpoint, cmd)
	return exec.Command("ssh", args...)
}

func buildSSHArgs(ep Endpoint, remoteCmd string) []string {
	var args []string
	if ep.SSHOpts.Identity != "" {
		args = append(args, "-i", ep.SSHOpts.Identity)
	}
	if ep.SSHOpts.Port != 0 {
		args = append(args, "-p", fmt.Sprintf("%d", ep.SSHOpts.Port))
	}
	for _, extra := range ep.SSHOpts.ExtraOpts {
		if strings.TrimSpace(extra) == "" {
			continue
		}
		args = append(args, "-o", extra)
	}
	target := ep.Host
	if ep.User != "" {
		target = fmt.Sprintf("%s@%s", ep.User, ep.Host)
	}
	args = append(args, target, remoteCmd)
	return args
}

func shellQuote(val string) string {
	return "'" + strings.ReplaceAll(val, "'", `'\''`) + "'"
}

func filepathToPosix(rel string) string {
	return strings.ReplaceAll(rel, "\\", "/")
}

type cmdReadCloser struct {
	Cmd    *exec.Cmd
	Reader io.ReadCloser
}

func (c *cmdReadCloser) Read(p []byte) (int, error) {
	return c.Reader.Read(p)
}

func (c *cmdReadCloser) Close() error {
	if err := c.Reader.Close(); err != nil {
		return err
	}
	return c.Cmd.Wait()
}

type cmdWriteCloser struct {
	Cmd    *exec.Cmd
	Writer io.WriteCloser
}

func (c *cmdWriteCloser) Write(p []byte) (int, error) {
	return c.Writer.Write(p)
}

func (c *cmdWriteCloser) Close() error {
	if err := c.Writer.Close(); err != nil {
		return err
	}
	return c.Cmd.Wait()
}
