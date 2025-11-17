package logging

import (
	"bytes"
	"testing"
)

func TestNewLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, err := New("debug", buf)
	if err != nil {
		t.Fatalf("create logger failed: %v", err)
	}
	logger.Debug("hello", "key", "value")
	if err := logger.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatalf("expected log output")
	}
}
