package endpoint

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// RemoteFS 使用 ssh 在远端执行命令
type RemoteFS struct {
	endpoint    Endpoint
	controlPath string
}

// NewRemoteFS 创建远端文件系统
func NewRemoteFS(ep Endpoint) *RemoteFS {
	control := ""
	if supportsControlMaster() {
		control = buildControlPath(ep)
	}
	return &RemoteFS{
		endpoint:    ep,
		controlPath: control,
	}
}

func (r *RemoteFS) Root() string {
	return r.endpoint.Path
}

func (r *RemoteFS) List(excludes []string) ([]FileMeta, error) {
	metas, unsupported, err := r.listWithFindPrintf(excludes)
	if err == nil {
		return metas, nil
	}
	if unsupported {
		return r.listWithFindStat(excludes)
	}
	return nil, err
}

func (r *RemoteFS) listWithFindPrintf(excludes []string) ([]FileMeta, bool, error) {
	script := fmt.Sprintf("cd %s && find . -mindepth 1 -printf '%%P|%%s|%%T@|%%m|%%y\\n'", shellQuote(r.endpoint.Path))
	output, err := r.runSSHCommand(script)
	if err != nil {
		if isFindPrintfUnsupported(output) {
			return nil, true, fmt.Errorf("远端 find 不支持 -printf")
		}
		return nil, false, fmt.Errorf("远端列举失败: %w: %s", err, string(output))
	}
	metas, err := parseRemoteListOutput(output, excludes)
	return metas, false, err
}

func (r *RemoteFS) listWithFindStat(excludes []string) ([]FileMeta, error) {
	script := fmt.Sprintf(`cd %[1]s && find . -mindepth 1 -print0 | while IFS= read -r -d '' file; do
rel="${file#./}"
[ -z "$rel" ] && continue
stat_out=$(stat -c '%%s|%%Y|%%f' "$file" 2>/dev/null || stat -f '%%z|%%m|%%p' "$file" 2>/dev/null)
[ -z "$stat_out" ] && continue
if [ -d "$file" ]; then type="d"; else type="f"; fi
printf '%%s|%%s|%%s\n' "$rel" "$stat_out" "$type"
done`, shellQuote(r.endpoint.Path))
	output, err := r.runSSHCommand(script)
	if err != nil {
		return nil, fmt.Errorf("远端列举失败: %w: %s", err, string(output))
	}
	return parseRemoteListOutput(output, excludes)
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
	script := fmt.Sprintf("mkdir -p %s && cat > %s && chmod %04o %s",
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
	script := fmt.Sprintf("stat -c '%%s|%%Y|%%f|%%F' %s 2>/dev/null || stat -f '%%z|%%m|%%p|%%HT' %s", shellQuote(remote), shellQuote(remote))
	out, err := r.runSSHCommand(script)
	if err != nil {
		return FileMeta{}, err
	}
	line := strings.TrimSpace(string(out))
	parts := strings.Split(line, "|")
	if len(parts) < 4 {
		return FileMeta{}, fmt.Errorf("stat output invalid: %s", line)
	}
	size, _ := strconv.ParseInt(parts[0], 10, 64)
	mod := parseEpoch(parts[1])
	mode := parseMode(parts[2])
	isDir := strings.Contains(strings.ToLower(parts[3]), "directory")
	return FileMeta{
		RelPath: relPath,
		Size:    size,
		Mode:    mode,
		ModTime: mod,
		IsDir:   isDir,
	}, nil
}

func (r *RemoteFS) runSSHCommand(cmd string) ([]byte, error) {
	command := r.sshCommand(cmd)
	return command.CombinedOutput()
}

func (r *RemoteFS) sshCommand(cmd string) *exec.Cmd {
	args := buildSSHArgs(r.endpoint, r.controlPath, cmd)
	return exec.Command("ssh", args...)
}

func buildSSHArgs(ep Endpoint, controlPath, remoteCmd string) []string {
	args := baseSSHArgs(ep, controlPath)
	args = append(args, targetHost(ep), remoteCmd)
	return args
}

func shellQuote(val string) string {
	return "'" + strings.ReplaceAll(val, "'", `'\''`) + "'"
}

func filepathToPosix(rel string) string {
	return strings.ReplaceAll(rel, "\\", "/")
}

func buildControlPath(ep Endpoint) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s@%s:%s-%d", ep.User, ep.Host, ep.Path, time.Now().UnixNano())))
	name := fmt.Sprintf("zbackup-ssh-%x.sock", sum[:6])
	return filepath.Join(os.TempDir(), name)
}

func baseSSHArgs(ep Endpoint, controlPath string) []string {
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
	if controlPath != "" {
		args = append(args,
			"-o", "ControlMaster=auto",
			"-o", fmt.Sprintf("ControlPath=%s", controlPath),
			"-o", "ControlPersist=600",
		)
	}
	return args
}

func targetHost(ep Endpoint) string {
	target := ep.Host
	if ep.User != "" {
		target = fmt.Sprintf("%s@%s", ep.User, ep.Host)
	}
	return target
}

// Close 关闭 SSH 控制连接
func (r *RemoteFS) Close() error {
	if r.controlPath == "" {
		return nil
	}
	args := baseSSHArgs(r.endpoint, "")
	args = append(args, "-S", r.controlPath, "-O", "exit", targetHost(r.endpoint))
	cmd := exec.Command("ssh", args...)
	if err := cmd.Run(); err != nil {
		if strings.Contains(err.Error(), "No such file or directory") || strings.Contains(err.Error(), "No control socket") {
			return nil
		}
		return err
	}
	_ = os.Remove(r.controlPath)
	return nil
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

func parseRemoteListOutput(output []byte, excludes []string) ([]FileMeta, error) {
	var metas []FileMeta
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		meta, ok := parseRemoteLine(line)
		if !ok {
			continue
		}
		if shouldExclude(meta.RelPath, excludes) {
			continue
		}
		metas = append(metas, meta)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return metas, nil
}

func parseRemoteLine(line string) (FileMeta, bool) {
	if strings.TrimSpace(line) == "" {
		return FileMeta{}, false
	}
	parts := strings.Split(line, "|")
	if len(parts) < 5 {
		return FileMeta{}, false
	}
	rel := strings.TrimPrefix(parts[0], "./")
	if rel == "" || rel == "." {
		return FileMeta{}, false
	}
	size, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return FileMeta{}, false
	}
	mod := parseEpoch(parts[2])
	mode := parseMode(parts[3])
	isDir := strings.TrimSpace(parts[4]) == "d"
	return FileMeta{
		RelPath: rel,
		Size:    size,
		Mode:    mode,
		ModTime: mod,
		IsDir:   isDir,
	}, true
}

func parseEpoch(val string) time.Time {
	val = strings.TrimSpace(val)
	if val == "" {
		return time.Unix(0, 0)
	}
	if strings.Contains(val, ".") {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			sec, frac := math.Modf(f)
			return time.Unix(int64(sec), int64(frac*1e9))
		}
	}
	if i, err := strconv.ParseInt(val, 10, 64); err == nil {
		return time.Unix(i, 0)
	}
	return time.Unix(0, 0)
}

func parseMode(val string) uint32 {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0
	}
	if v, err := strconv.ParseUint(val, 8, 32); err == nil {
		return uint32(v)
	}
	if v, err := strconv.ParseUint(val, 16, 32); err == nil {
		return uint32(v)
	}
	if v, err := strconv.ParseUint(val, 10, 32); err == nil {
		return uint32(v)
	}
	return 0
}

func isFindPrintfUnsupported(output []byte) bool {
	if len(output) == 0 {
		return false
	}
	text := strings.ToLower(string(output))
	return strings.Contains(text, "-printf") || strings.Contains(text, "unknown predicate") || strings.Contains(text, "busybox")
}

func supportsControlMaster() bool {
	// Windows 版 OpenSSH 尚未实现控制主连接
	return runtime.GOOS != "windows"
}
