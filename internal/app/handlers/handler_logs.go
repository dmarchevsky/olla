package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/thushan/olla/internal/core/constants"
	"github.com/thushan/olla/internal/logger"
)

type LogEntryResponse struct {
	Time     time.Time         `json:"time"`
	Level    string            `json:"level"`
	Message  string            `json:"message"`
	Endpoint string            `json:"endpoint,omitempty"`
	Attrs    map[string]string `json:"attrs,omitempty"`
	Seq      uint64            `json:"seq"`
}

type LogsResponse struct {
	Entries   []LogEntryResponse `json:"entries"`
	NextSince uint64             `json:"next_since"`
	HeadSeq   uint64             `json:"head_seq"`
	Capacity  int                `json:"capacity"`
	Truncated bool               `json:"truncated"`
}

// logsHandler serves the dashboard's log browser from the in-memory ring
// buffer populated by every slog call app-wide (internal/logger).
//
// This handler must never log per-request: doing so would feed its own
// output back into the ring buffer it's serving, and under fast "follow"
// polling that would spiral. It also needs no addition to the quiet-poll-route
// list in middleware/logging.go - that gate is a prefix match on "/internal/",
// so /internal/logs already qualifies automatically.
func (a *Application) logsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)

	rb := logger.GlobalRingBuffer()
	if rb == nil {
		_ = json.NewEncoder(w).Encode(LogsResponse{Entries: []LogEntryResponse{}})
		return
	}

	q := parseLogQuery(r)
	entries, headSeq, oldestSeq := rb.Query(q)

	resp := LogsResponse{
		Entries:   make([]LogEntryResponse, len(entries)),
		NextSince: headSeq,
		HeadSeq:   headSeq,
		Capacity:  rb.Capacity(),
		Truncated: q.Since > 0 && q.Since < oldestSeq,
	}
	for i, e := range entries {
		resp.Entries[i] = LogEntryResponse{
			Seq:      e.Seq,
			Time:     e.Time,
			Level:    e.Level,
			Message:  e.Message,
			Endpoint: e.Endpoint,
			Attrs:    e.Attrs,
		}
	}
	if len(entries) > 0 {
		resp.NextSince = entries[len(entries)-1].Seq
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// parseLogQuery reads since/from/to/level/endpoint/limit query params.
// Unrecognised level values are dropped rather than erroring, since a stale
// bookmarked URL or a filter typo should degrade to "no level filter" instead
// of a 400.
func parseLogQuery(r *http.Request) logger.QueryParams {
	q := r.URL.Query()

	var since uint64
	if v := q.Get("since"); v != "" {
		since, _ = strconv.ParseUint(v, 10, 64)
	}

	var from, to time.Time
	if v := q.Get("from"); v != "" {
		from, _ = time.Parse(time.RFC3339, v)
	}
	if v := q.Get("to"); v != "" {
		to, _ = time.Parse(time.RFC3339, v)
	}

	var levels map[string]struct{}
	if v := q.Get("level"); v != "" {
		levels = make(map[string]struct{})
		for _, l := range strings.Split(v, ",") {
			l = strings.ToLower(strings.TrimSpace(l))
			// "warning" is an accepted alias for "warn" in config (see
			// logger.IsValidLevel), but captured entries only ever carry
			// "warn" (see levelString) - normalise so the alias still matches.
			if l == logger.LogLevelWarning {
				l = logger.LogLevelWarn
			}
			if logger.IsValidLevel(l) {
				levels[l] = struct{}{}
			}
		}
	}

	limit, _ := strconv.Atoi(q.Get("limit"))

	return logger.QueryParams{
		Since:    since,
		From:     from,
		To:       to,
		Levels:   levels,
		Endpoint: q.Get("endpoint"),
		Limit:    limit,
	}
}
