package logger

import (
	"log/slog"
	"testing"

	"github.com/thushan/olla/internal/core/domain"
	"github.com/thushan/olla/theme"
)

func newTestStyledLogger(pretty bool) (*RingBuffer, StyledLogger) {
	rb := NewRingBuffer(64)
	l := slog.New(newRingBufferHandler(rb, slog.LevelDebug))
	if pretty {
		return rb, NewPrettyStyledLogger(l, theme.GetTheme("default"))
	}
	return rb, NewPlainStyledLogger(l)
}

func TestStyledLoggerEndpointAttr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		pretty bool
	}{
		{"plain", false},
		{"pretty", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rb, sl := newTestStyledLogger(tc.pretty)

			sl.InfoWithEndpoint("Applied model filter", "ryzen", "original_count", 10)
			sl.WarnWithEndpoint("Pausing discovery for endpoint", "ryzen", "failures", 5)
			sl.ErrorWithEndpoint("Failed to register discovered models", "ryzen", "error", "boom")
			sl.InfoHealthy("Endpoint healthy", "ryzen")
			sl.InfoWithHealthCheck("Health checked", "ryzen")

			entries := queryAll(t, rb)
			if len(entries) != 5 {
				t.Fatalf("expected 5 entries, got %d", len(entries))
			}
			for _, e := range entries {
				if e.Endpoint != "ryzen" {
					t.Errorf("entry %q: Endpoint = %q, want %q", e.Message, e.Endpoint, "ryzen")
				}
			}
			if entries[0].Message != "Applied model filter ryzen" {
				t.Errorf("message composition changed: %q", entries[0].Message)
			}
			if entries[0].Attrs["original_count"] != "10" {
				t.Errorf("caller args lost: %+v", entries[0].Attrs)
			}
		})
	}
}

func TestStyledLoggerInfoHealthStatusEndpointAttr(t *testing.T) {
	t.Parallel()

	rb, sl := newTestStyledLogger(false)
	sl.InfoHealthStatus("Endpoint recovered:", "ryzen", domain.StatusHealthy, "was", "Offline")

	entries := queryAll(t, rb)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Endpoint != "ryzen" {
		t.Errorf("Endpoint = %q, want %q", entries[0].Endpoint, "ryzen")
	}
}

func TestStyledLoggerWithContextEmitsEndpointAndDedups(t *testing.T) {
	t.Parallel()

	rb, sl := newTestStyledLogger(false)
	sl.WarnWithContext("Regular discovery failed (timeout)", "brainiac-halogen", LogContext{
		UserArgs:     []interface{}{"attempts", 3},
		DetailedArgs: []interface{}{"detailed_error", "context deadline exceeded"},
	})

	entries := queryAll(t, rb)
	if len(entries) != 1 {
		t.Fatalf("expected only the user record (detailed twin skipped), got %d entries", len(entries))
	}
	e := entries[0]
	if e.Endpoint != "brainiac-halogen" {
		t.Errorf("Endpoint = %q, want %q", e.Endpoint, "brainiac-halogen")
	}
	if e.Message != "Regular discovery failed (timeout) brainiac-halogen" {
		t.Errorf("message composition changed: %q", e.Message)
	}
	if e.Attrs["attempts"] != "3" {
		t.Errorf("UserArgs lost: %+v", e.Attrs)
	}
}
