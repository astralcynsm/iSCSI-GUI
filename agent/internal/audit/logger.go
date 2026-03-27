package audit

import (
	"strings"
	"sync"
	"time"
)

type Record struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id,omitempty"`
	Actor     string `json:"actor,omitempty"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	TargetIQN string `json:"target_iqn,omitempty"`
	Result    string `json:"result"`
	Changed   *bool  `json:"changed,omitempty"`
	Message   string `json:"message,omitempty"`
}

type Filter struct {
	Limit     int
	TargetIQN string
	Action    string
}

type Logger struct {
	mu    sync.RWMutex
	max   int
	items []Record
}

func NewLogger(max int) *Logger {
	if max <= 0 {
		max = 200
	}
	return &Logger{max: max, items: make([]Record, 0, max)}
}

func (l *Logger) Add(r Record) {
	if l == nil {
		return
	}
	if strings.TrimSpace(r.Timestamp) == "" {
		r.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	r.Action = strings.TrimSpace(strings.ToLower(r.Action))
	r.Resource = strings.TrimSpace(strings.ToLower(r.Resource))
	r.TargetIQN = strings.TrimSpace(r.TargetIQN)
	r.Result = strings.TrimSpace(strings.ToLower(r.Result))
	r.Actor = strings.TrimSpace(r.Actor)
	r.RequestID = strings.TrimSpace(r.RequestID)
	r.Message = strings.TrimSpace(r.Message)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.items = append(l.items, r)
	if len(l.items) > l.max {
		drop := len(l.items) - l.max
		l.items = append([]Record(nil), l.items[drop:]...)
	}
}

func (l *Logger) List(f Filter) []Record {
	if l == nil {
		return []Record{}
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	targetFilter := strings.TrimSpace(f.TargetIQN)
	actionFilter := strings.TrimSpace(strings.ToLower(f.Action))

	l.mu.RLock()
	defer l.mu.RUnlock()

	out := make([]Record, 0, limit)
	for i := len(l.items) - 1; i >= 0; i-- {
		it := l.items[i]
		if targetFilter != "" && it.TargetIQN != targetFilter {
			continue
		}
		if actionFilter != "" && it.Action != actionFilter {
			continue
		}
		out = append(out, it)
		if len(out) >= limit {
			break
		}
	}
	return out
}
