package logger

import (
	"context"
	"log/slog"
	"testing"
)

func newTestRingLogger(level slog.Level) (*RingBuffer, *slog.Logger) {
	rb := NewRingBuffer(64)
	return rb, slog.New(newRingBufferHandler(rb, level))
}

func queryAll(t *testing.T, rb *RingBuffer) []Entry {
	t.Helper()
	entries, _, _ := rb.Query(QueryParams{})
	return entries
}

func TestRingBufferHandlerCapturesEndpointAttr(t *testing.T) {
	t.Parallel()

	rb, l := newTestRingLogger(slog.LevelInfo)
	l.Info("Applied model filter", "endpoint", "ryzen", "original_count", 10)

	entries := queryAll(t, rb)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Endpoint != "ryzen" {
		t.Errorf("Endpoint = %q, want %q", entries[0].Endpoint, "ryzen")
	}
	if entries[0].Message != "Applied model filter" {
		t.Errorf("Message = %q, want %q", entries[0].Message, "Applied model filter")
	}
	if entries[0].Attrs["original_count"] != "10" {
		t.Errorf("Attrs[original_count] = %q, want %q", entries[0].Attrs["original_count"], "10")
	}
	if _, ok := entries[0].Attrs["endpoint"]; ok {
		t.Error("endpoint attr should be promoted to Endpoint, not kept in Attrs")
	}
}

func TestRingBufferHandlerEndpointNameFallback(t *testing.T) {
	t.Parallel()

	rb, l := newTestRingLogger(slog.LevelInfo)
	l.Info("Model discovery failed", "endpoint_name", "brainiac-halogen")

	entries := queryAll(t, rb)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Endpoint != "brainiac-halogen" {
		t.Errorf("Endpoint = %q, want %q", entries[0].Endpoint, "brainiac-halogen")
	}
}

func TestRingBufferHandlerPrefersEndpointOverEndpointName(t *testing.T) {
	t.Parallel()

	rb, l := newTestRingLogger(slog.LevelInfo)
	l.Info("both keys", "endpoint", "primary", "endpoint_name", "alias")

	entries := queryAll(t, rb)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Endpoint != "primary" {
		t.Errorf("Endpoint = %q, want %q", entries[0].Endpoint, "primary")
	}
}

func TestRingBufferHandlerStripsANSI(t *testing.T) {
	t.Parallel()

	rb, l := newTestRingLogger(slog.LevelInfo)
	l.Info("Applied model filter \x1b[38;5;213mryzen\x1b[0m", "detail", "\x1b[1mbold\x1b[0m value")

	entries := queryAll(t, rb)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Message != "Applied model filter ryzen" {
		t.Errorf("Message = %q, want ANSI-stripped message", entries[0].Message)
	}
	if entries[0].Attrs["detail"] != "bold value" {
		t.Errorf("Attrs[detail] = %q, want ANSI-stripped value", entries[0].Attrs["detail"])
	}
}

func TestRingBufferHandlerSkipsDetailedRecords(t *testing.T) {
	t.Parallel()

	rb, l := newTestRingLogger(slog.LevelInfo)

	l.Info("user record")
	detailedCtx := context.WithValue(context.Background(), DefaultDetailedCookie, true)
	l.InfoContext(detailedCtx, "detailed twin", "detailed_error", "boom")

	entries := queryAll(t, rb)
	if len(entries) != 1 {
		t.Fatalf("expected only the user record, got %d entries", len(entries))
	}
	if entries[0].Message != "user record" {
		t.Errorf("Message = %q, want %q", entries[0].Message, "user record")
	}
}

func TestRingBufferHandlerWithAttrsEndpoint(t *testing.T) {
	t.Parallel()

	rb := NewRingBuffer(64)
	base := newRingBufferHandler(rb, slog.LevelInfo)
	l := slog.New(base.WithAttrs([]slog.Attr{slog.String("endpoint", "pinned")}))

	l.Info("from WithAttrs")

	entries := queryAll(t, rb)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Endpoint != "pinned" {
		t.Errorf("Endpoint = %q, want %q", entries[0].Endpoint, "pinned")
	}
}
