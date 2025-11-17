package endpoint

import (
	"testing"
	"time"
)

func TestParseRemoteLine(t *testing.T) {
	line := "dir/file.txt|123|1700000000.5|755"
	meta, ok := parseRemoteLine(line)
	if !ok {
		t.Fatalf("line should parse")
	}
	if meta.RelPath != "dir/file.txt" {
		t.Fatalf("unexpected rel path %s", meta.RelPath)
	}
	if meta.Size != 123 {
		t.Fatalf("unexpected size %d", meta.Size)
	}
	if meta.Mode == 0 {
		t.Fatalf("mode should parse")
	}
	if meta.ModTime.Unix() != 1700000000 {
		t.Fatalf("unexpected mod time %s", meta.ModTime)
	}
}

func TestParseRemoteListOutputExclude(t *testing.T) {
	data := []byte("foo.txt|10|1700000000|644\nbar.tmp|11|1700000001|644\n")
	metas, err := parseRemoteListOutput(data, []string{"*.tmp"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(metas) != 1 || metas[0].RelPath != "foo.txt" {
		t.Fatalf("unexpected metas: %+v", metas)
	}
}

func TestParseEpochModeFallback(t *testing.T) {
	tm := parseEpoch("1700000000")
	if tm.Unix() != 1700000000 {
		t.Fatalf("unexpected unix time %v", tm)
	}
	tm = parseEpoch("1700000000.500")
	if tm.UnixNano() != (1700000000*int64(time.Second) + int64(0.5*1e9)) {
		t.Fatalf("fraction lost")
	}
	if parseMode("81ed") == 0 {
		t.Fatalf("hex mode should parse")
	}
	if parseMode("0100644") == 0 {
		t.Fatalf("octal mode should parse")
	}
}

func TestFindPrintfUnsupported(t *testing.T) {
	if !isFindPrintfUnsupported([]byte("find: unrecognized: -printf")) {
		t.Fatal("should detect unsupported")
	}
	if isFindPrintfUnsupported([]byte("normal output")) {
		t.Fatal("should not detect on normal text")
	}
}
