package logger

import (
	"context"
	"log/slog"
)

// ringBufferHandler fans slog records into a RingBuffer for the dashboard's
// log browser. It never returns an error from Handle: a full or degraded
// ring buffer must never break the real log output the other handlers in
// the chain are responsible for.
type ringBufferHandler struct {
	rb    *RingBuffer
	level slog.Level
	attrs []slog.Attr
}

func newRingBufferHandler(rb *RingBuffer, level slog.Level) *ringBufferHandler {
	return &ringBufferHandler{rb: rb, level: level}
}

func (h *ringBufferHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *ringBufferHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := make(map[string]string, len(h.attrs)+r.NumAttrs())
	endpoint := ""

	for _, a := range h.attrs {
		if a.Key == "endpoint" {
			endpoint = a.Value.String()
			continue
		}
		attrs[a.Key] = a.Value.String()
	}
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "endpoint" {
			endpoint = a.Value.String()
			return true
		}
		attrs[a.Key] = a.Value.String()
		return true
	})

	h.rb.Append(Entry{
		Time:     r.Time,
		Level:    levelString(r.Level),
		Message:  r.Message,
		Endpoint: endpoint,
		Attrs:    attrs,
	})
	return nil
}

func (h *ringBufferHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	newAttrs = append(newAttrs, attrs...)
	return &ringBufferHandler{rb: h.rb, level: h.level, attrs: newAttrs}
}

func (h *ringBufferHandler) WithGroup(_ string) slog.Handler {
	// Groups would namespace attrs (e.g. "group.key"); the log browser doesn't
	// need that structure, so grouped attrs are captured under their bare key
	// same as ungrouped ones. No caller in this codebase uses WithGroup today.
	return h
}

// levelString maps a slog.Level back to the lowercase level strings this
// package already defines (LogLevelDebug etc), so captured entries use the
// same vocabulary as logger.IsValidLevel/parseLevel.
func levelString(l slog.Level) string {
	switch {
	case l < slog.LevelInfo:
		return LogLevelDebug
	case l < slog.LevelWarn:
		return LogLevelInfo
	case l < slog.LevelError:
		return LogLevelWarn
	default:
		return LogLevelError
	}
}
