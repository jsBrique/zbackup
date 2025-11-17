package ui

import (
	"fmt"
	"io"
	"sync/atomic"

	"github.com/schollz/progressbar/v3"
)

// Progress 定义统一的进度更新接口
type Progress interface {
	Start(totalFiles int, totalBytes int64)
	NextFile(path string, size int64)
	AddBytes(n int64)
	Finish()
}

// BarProgress 使用 progressbar 库实现
type BarProgress struct {
	bar         *progressbar.ProgressBar
	currentFile atomic.Value
	totalFiles  int
	completed   atomic.Int64
	output      io.Writer
}

// NewBarProgress 创建进度条
func NewBarProgress(output io.Writer) *BarProgress {
	return &BarProgress{output: output}
}

func (p *BarProgress) Start(totalFiles int, totalBytes int64) {
	p.totalFiles = totalFiles
	p.bar = progressbar.NewOptions64(
		totalBytes,
		progressbar.OptionSetWriter(p.output),
		progressbar.OptionShowBytes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetDescription("准备中"),
	)
}

func (p *BarProgress) NextFile(path string, size int64) {
	p.currentFile.Store(path)
	p.bar.Describe(fmt.Sprintf("%s (%d/%d)", path, p.completed.Load()+1, p.totalFiles))
}

func (p *BarProgress) AddBytes(n int64) {
	p.bar.Add64(n)
}

func (p *BarProgress) Finish() {
	p.bar.Finish()
}

// NoopProgress 在 --no-progress 下使用
type NoopProgress struct{}

func (n NoopProgress) Start(totalFiles int, totalBytes int64) {}
func (n NoopProgress) NextFile(path string, size int64)       {}
func (n NoopProgress) AddBytes(delta int64)                   {}
func (n NoopProgress) Finish()                                {}
