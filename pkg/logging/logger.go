package logging

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Logger 包装 slog.Logger，方便统一关闭资源
type Logger struct {
	*slog.Logger
	closers []io.Closer
}

// New 创建 Logger，输出到给定 writer
func New(level string, writers ...io.Writer) (*Logger, error) {
	if len(writers) == 0 {
		return nil, fmt.Errorf("必须提供至少一个日志输出")
	}
	var closerList []io.Closer
	var output io.Writer
	if len(writers) == 1 {
		output = writers[0]
	} else {
		output = io.MultiWriter(writers...)
	}
	for _, w := range writers {
		if c, ok := w.(io.Closer); ok {
			closerList = append(closerList, c)
		}
	}
	handler := slog.NewTextHandler(output, &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	return &Logger{
		Logger:  slog.New(handler),
		closers: closerList,
	}, nil
}

// Close 关闭所有 writer
func (l *Logger) Close() error {
	var lastErr error
	for _, c := range l.closers {
		if err := c.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func parseLevel(level string) slog.Leveler {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
