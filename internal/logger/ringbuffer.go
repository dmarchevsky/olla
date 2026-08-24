package logger

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultRingBufferSize is used whenever a caller passes a zero or negative
	// capacity, so config.RingBufferSize == 0 (unset) behaves sanely rather than
	// producing a zero-capacity buffer that silently drops every entry.
	DefaultRingBufferSize = 5000

	// maxEntryAttrs and maxEntryAttrValueLen bound the memory a single log
	// record's attributes can consume once captured. Append runs on the slog
	// hot path for every log statement app-wide, so a single oversized attr
	// (e.g. a stack trace passed as an "error" value) must not be able to blow
	// the buffer's memory budget.
	maxEntryAttrs        = 16
	maxEntryAttrValueLen = 256
)

// Entry is one captured, structured log record.
type Entry struct {
	Time     time.Time
	Level    string
	Message  string
	Endpoint string
	Attrs    map[string]string
	Seq      uint64
}

// QueryParams filters a RingBuffer.Query call. Zero values mean "no filter"
// for that field, except Since where 0 means "from the oldest retained entry".
type QueryParams struct {
	From     time.Time
	To       time.Time
	Levels   map[string]struct{}
	Endpoint string
	Since    uint64
	Limit    int
}

// RingBuffer is a fixed-capacity, thread-safe circular buffer of log Entry
// values. Oldest entries are silently overwritten once full, so memory stays
// bounded regardless of how much the application logs. It captures the raw
// stream; filtering happens at Query time.
type RingBuffer struct {
	mu       sync.Mutex
	buf      []Entry
	next     int
	filled   bool
	seq      uint64
	capacity int
}

// NewRingBuffer creates a RingBuffer with the given capacity. A non-positive
// capacity falls back to DefaultRingBufferSize.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = DefaultRingBufferSize
	}
	return &RingBuffer{
		buf:      make([]Entry, capacity),
		capacity: capacity,
	}
}

// Capacity returns the buffer's fixed entry capacity.
func (rb *RingBuffer) Capacity() int {
	return rb.capacity
}

// Append records e, assigning it the next monotonic sequence number.
// Capping Attrs happens here (not by the caller) so every entry that reaches
// the buffer is already bounded regardless of caller.
func (rb *RingBuffer) Append(e Entry) {
	e.Attrs = capAttrs(e.Attrs)

	rb.mu.Lock()
	rb.seq++
	e.Seq = rb.seq
	rb.buf[rb.next] = e
	rb.next = (rb.next + 1) % rb.capacity
	if rb.next == 0 {
		rb.filled = true
	}
	rb.mu.Unlock()
}

// capAttrs bounds the number of attribute keys and the length of each value so
// one pathological log call cannot dominate the buffer's memory footprint.
func capAttrs(attrs map[string]string) map[string]string {
	if len(attrs) == 0 {
		return attrs
	}
	out := make(map[string]string, min(len(attrs), maxEntryAttrs))
	count := 0
	for k, v := range attrs {
		if count >= maxEntryAttrs {
			break
		}
		if len(v) > maxEntryAttrValueLen {
			v = v[:maxEntryAttrValueLen]
		}
		out[k] = v
		count++
	}
	return out
}

// Query returns entries matching q in ascending Seq order, along with the
// buffer's current newest sequence (headSeq) and the oldest sequence still
// retained (oldestSeq) — the latter lets the caller detect whether q.Since
// has fallen behind what the buffer still has (i.e. entries were evicted
// before the client could read them).
func (rb *RingBuffer) Query(q QueryParams) (entries []Entry, headSeq, oldestSeq uint64) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	headSeq = rb.seq

	count := rb.capacity
	if !rb.filled {
		count = rb.next
	}
	if count == 0 {
		return nil, headSeq, 0
	}

	// Oldest retained entry is at rb.next when the buffer has wrapped
	// (rb.next is about to overwrite the oldest slot), or at index 0 otherwise.
	startIdx := 0
	if rb.filled {
		startIdx = rb.next
	}
	oldestSeq = rb.buf[startIdx].Seq

	limit := q.Limit
	if limit <= 0 {
		limit = defaultQueryLimit
	}
	if limit > maxQueryLimit {
		limit = maxQueryLimit
	}

	entries = make([]Entry, 0, min(count, limit))
	for i := 0; i < count; i++ {
		idx := (startIdx + i) % rb.capacity
		e := rb.buf[idx]

		if e.Seq <= q.Since {
			continue
		}
		if !q.From.IsZero() && e.Time.Before(q.From) {
			continue
		}
		if !q.To.IsZero() && e.Time.After(q.To) {
			continue
		}
		if len(q.Levels) > 0 {
			if _, ok := q.Levels[e.Level]; !ok {
				continue
			}
		}
		if q.Endpoint != "" && e.Endpoint != q.Endpoint {
			continue
		}

		entries = append(entries, e)
		if len(entries) >= limit {
			break
		}
	}

	return entries, headSeq, oldestSeq
}

const (
	defaultQueryLimit = 500
	maxQueryLimit     = 2000
)

var globalRingBuffer atomic.Pointer[RingBuffer]

// SetGlobalRingBuffer installs rb as the process-wide log ring buffer, mirroring
// the existing slog.SetDefault(logger) pattern used for "the one logger
// instance" (see main.go) rather than threading a new field through
// CreateAndStartServiceManager -> registerServices -> HTTPService ->
// NewApplication.
func SetGlobalRingBuffer(rb *RingBuffer) {
	globalRingBuffer.Store(rb)
}

// GlobalRingBuffer returns the process-wide log ring buffer, or nil if New/
// NewWithTheme has not run yet.
func GlobalRingBuffer() *RingBuffer {
	return globalRingBuffer.Load()
}
