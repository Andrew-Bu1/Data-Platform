package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

const (
	colorReset  = "\033[0m"
	colorGray   = "\033[90m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[97m"
)

// ColorHandler is a slog.Handler that writes colored, human-readable log lines.
type ColorHandler struct {
	mu  sync.Mutex
	out io.Writer
}

func NewColorHandler(out io.Writer) *ColorHandler {
	return &ColorHandler{out: out}
}

func (h *ColorHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *ColorHandler) Handle(_ context.Context, r slog.Record) error {
	var levelColor, levelStr string
	switch {
	case r.Level >= slog.LevelError:
		levelColor = colorRed
		levelStr = "ERROR"
	case r.Level >= slog.LevelWarn:
		levelColor = colorYellow
		levelStr = "WARN "
	case r.Level >= slog.LevelInfo:
		levelColor = colorGreen
		levelStr = "INFO "
	default:
		levelColor = colorCyan
		levelStr = "DEBUG"
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s%s%s ", colorGray, r.Time.Format("2006/01/02 15:04:05"), colorReset)
	fmt.Fprintf(&buf, "%s%s%s ", levelColor, levelStr, colorReset)
	fmt.Fprintf(&buf, "%s%s%s", colorWhite, r.Message, colorReset)

	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(&buf, " %s%s%s=%v", colorCyan, a.Key, colorReset, a.Value)
		return true
	})
	buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf.Bytes())
	return err
}

func (h *ColorHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *ColorHandler) WithGroup(_ string) slog.Handler      { return h }
