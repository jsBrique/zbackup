package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Progress 定义统一的进度更新接口
type Progress interface {
	Start(totalFiles int, totalBytes int64)
	NextFile(path string, size int64)
	AddBytes(n int64)
	Finish()
}

// BarProgress 实现单行文本进度条，并与日志输出互斥
type BarProgress struct {
	mu             sync.Mutex
	writer         io.Writer
	totalFiles     int
	totalBytes     int64
	completedFiles int
	completedBytes int64
	currentFile    string
	lastLine       string
	active         bool
	startTime      time.Time
}

const (
	progressWidth = 30
	maxDescLen    = 50
)

// NewBarProgress 创建进度条实例
func NewBarProgress(writer io.Writer) *BarProgress {
	return &BarProgress{writer: writer}
}

func (p *BarProgress) Start(totalFiles int, totalBytes int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalFiles = totalFiles
	p.totalBytes = totalBytes
	p.completedFiles = 0
	p.completedBytes = 0
	p.currentFile = ""
	p.lastLine = ""
	p.active = true
	p.startTime = time.Now()
	p.renderLocked(false)
}

func (p *BarProgress) NextFile(path string, size int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.active {
		return
	}
	p.completedFiles++
	p.currentFile = shortenPath(path, maxDescLen)
	p.renderLocked(false)
}

func (p *BarProgress) AddBytes(n int64) {
	if n == 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.active {
		return
	}
	p.completedBytes += n
	if p.completedBytes > p.totalBytes && p.totalBytes > 0 {
		p.completedBytes = p.totalBytes
	}
	p.renderLocked(false)
}

func (p *BarProgress) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.active {
		return
	}
	p.renderLocked(true)
	p.active = false
	p.lastLine = ""
}

// WrapWriter 返回一个 writer，保证日志输出前清除进度条，结束后重新绘制
func (p *BarProgress) WrapWriter(w io.Writer) io.Writer {
	if p == nil {
		return w
	}
	return &progressAwareWriter{
		progress: p,
		writer:   w,
	}
}

// NoopProgress 在 --no-progress 下使用
type NoopProgress struct{}

func (n NoopProgress) Start(totalFiles int, totalBytes int64) {}
func (n NoopProgress) NextFile(path string, size int64)       {}
func (n NoopProgress) AddBytes(delta int64)                   {}
func (n NoopProgress) Finish()                                {}

type progressAwareWriter struct {
	progress *BarProgress
	writer   io.Writer
}

func (pw *progressAwareWriter) Write(b []byte) (int, error) {
	pw.progress.mu.Lock()
	pw.progress.clearLocked()
	n, err := pw.writer.Write(b)
	pw.progress.renderLocked(false)
	pw.progress.mu.Unlock()
	return n, err
}

func (p *BarProgress) clearLocked() {
	if !p.active || p.lastLine == "" || p.writer == nil {
		return
	}
	spaceCount := runeCount(p.lastLine) + 2
	fmt.Fprintf(p.writer, "\r%s\r", strings.Repeat(" ", spaceCount))
	p.lastLine = ""
}

func (p *BarProgress) renderLocked(final bool) {
	if p.writer == nil || !p.active {
		return
	}
	var percent float64
	if p.totalBytes > 0 {
		percent = float64(p.completedBytes) / float64(p.totalBytes)
		if percent > 1 {
			percent = 1
		}
	}
	filled := int(percent * progressWidth)
	if filled > progressWidth {
		filled = progressWidth
	}
	bar := fmt.Sprintf("[%s%s]", strings.Repeat("#", filled), strings.Repeat("-", progressWidth-filled))
	current := ""
	if p.currentFile != "" {
		current = " " + p.currentFile
	}
	speed := p.calcSpeedMbps()
	line := fmt.Sprintf("%s %6.2f%% %d/%d files %5.2f Mbps%s",
		bar, percent*100, p.completedFiles, p.totalFiles, speed, current)
	if final {
		fmt.Fprintf(p.writer, "\r%s\n", line)
		p.lastLine = ""
		return
	}
	fmt.Fprintf(p.writer, "\r%s", line)
	p.lastLine = line
}

func shortenPath(path string, maxLen int) string {
	clean := strings.NewReplacer("\n", " ", "\r", " ").Replace(path)
	runes := []rune(clean)
	if len(runes) <= maxLen {
		return clean
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	keep := maxLen - 3
	head := keep / 2
	tail := keep - head
	return string(runes[:head]) + "..." + string(runes[len(runes)-tail:])
}

func runeCount(s string) int {
	return utf8.RuneCountInString(s)
}

func (p *BarProgress) calcSpeedMbps() float64 {
	elapsed := time.Since(p.startTime).Seconds()
	if elapsed <= 0 {
		return 0
	}
	bytesPerSec := float64(p.completedBytes) / elapsed
	return bytesPerSec * 8 / 1_000_000
}
