package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestShortenPath(t *testing.T) {
	path := "很长的目录/包含/很多/层级/以及/特殊/字符/测试/文件.txt"
	short := shortenPath(path, 20)
	if len([]rune(short)) > 20 {
		t.Fatalf("path not shortened: %s", short)
	}
	if short == path {
		t.Fatalf("expected truncation for long path")
	}

	orig := "正常路径"
	if shortenPath(orig, 20) != orig {
		t.Fatalf("should not truncate shorter string")
	}
}

func TestBarProgressSingleLine(t *testing.T) {
	buf := &bytes.Buffer{}
	progress := NewBarProgress(buf)
	progress.Start(2, 100)
	progress.NextFile("示例/文件1.txt", 50)
	progress.AddBytes(50)
	progress.NextFile("示例/文件2.txt", 50)
	progress.AddBytes(50)
	progress.Finish()

	output := buf.String()
	if strings.Count(output, "\n") != 1 {
		t.Fatalf("expected single newline, got %q", output)
	}
	if strings.Contains(output[:len(output)-1], "\n") {
		t.Fatalf("newline appears before finish: %q", output)
	}
}
