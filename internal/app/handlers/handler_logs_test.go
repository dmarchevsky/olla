package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/thushan/olla/internal/logger"
)

// withTestRingBuffer installs a fresh global ring buffer for the duration of
// the test and restores the previous one afterwards. Not parallel-safe by
// design: the buffer is process-global, mirroring production wiring.
func withTestRingBuffer(t *testing.T, capacity int) *logger.RingBuffer {
	t.Helper()
	rb := logger.NewRingBuffer(capacity)
	prev := logger.GlobalRingBuffer()
	logger.SetGlobalRingBuffer(rb)
	t.Cleanup(func() { logger.SetGlobalRingBuffer(prev) })
	return rb
}

func decodeLogs(t *testing.T, rec *httptest.ResponseRecorder) LogsResponse {
	t.Helper()
	var resp LogsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding logs response: %v (body: %s)", err, rec.Body.String())
	}
	return resp
}

func TestLogsHandlerEndpointFilter(t *testing.T) {
	rb := withTestRingBuffer(t, 64)
	rb.Append(logger.Entry{Time: time.Now(), Level: "info", Message: "alpha entry", Endpoint: "alpha"})
	rb.Append(logger.Entry{Time: time.Now(), Level: "info", Message: "beta entry", Endpoint: "beta"})

	app := &Application{}
	req := httptest.NewRequest(http.MethodGet, "/internal/logs?endpoint=alpha", nil)
	rec := httptest.NewRecorder()
	app.logsHandler(rec, req)

	resp := decodeLogs(t, rec)
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
	}
	if resp.Entries[0].Endpoint != "alpha" || resp.Entries[0].Message != "alpha entry" {
		t.Errorf("unexpected entry: %+v", resp.Entries[0])
	}
}

func TestLogsHandlerSerializesAttrs(t *testing.T) {
	rb := withTestRingBuffer(t, 64)
	rb.Append(logger.Entry{
		Time:     time.Now(),
		Level:    "info",
		Message:  "Discovered models",
		Endpoint: "ryzen",
		Attrs:    map[string]string{"models": "4"},
	})

	app := &Application{}
	req := httptest.NewRequest(http.MethodGet, "/internal/logs", nil)
	rec := httptest.NewRecorder()
	app.logsHandler(rec, req)

	resp := decodeLogs(t, rec)
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
	}
	if resp.Entries[0].Attrs["models"] != "4" {
		t.Errorf("attrs not serialized: %+v", resp.Entries[0].Attrs)
	}
}

func TestLogsHandlerSinceCursor(t *testing.T) {
	rb := withTestRingBuffer(t, 64)
	rb.Append(logger.Entry{Time: time.Now(), Level: "info", Message: "first"})
	rb.Append(logger.Entry{Time: time.Now(), Level: "info", Message: "second"})

	app := &Application{}
	req := httptest.NewRequest(http.MethodGet, "/internal/logs", nil)
	rec := httptest.NewRecorder()
	app.logsHandler(rec, req)
	first := decodeLogs(t, rec)
	if len(first.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(first.Entries))
	}

	req = httptest.NewRequest(http.MethodGet, "/internal/logs?since="+strconv.FormatUint(first.NextSince, 10), nil)
	rec = httptest.NewRecorder()
	app.logsHandler(rec, req)
	second := decodeLogs(t, rec)
	if len(second.Entries) != 0 {
		t.Errorf("expected no entries after cursor, got %d", len(second.Entries))
	}
}
