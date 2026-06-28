package main

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

type serverCodexAppServerGoalRuntimeV0 struct {
	mu        sync.Mutex
	startedAt map[string]time.Time
}

func (backend serverCodexAppServerGoalBackendV0) codexAppServerActiveGoalElapsedV0(
	goal *serverCodexAppServerThreadGoalV0,
) (time.Duration, bool) {
	if goal == nil {
		return 0, false
	}
	if goal.TimeUsedSeconds > 0 {
		return time.Duration(goal.TimeUsedSeconds) * time.Second, true
	}
	now := backend.nowCodexAppServerGoalV0()
	if startedAt, ok := goal.CreatedAt.TimeV0(now); ok {
		return now.Sub(startedAt), true
	}
	threadID := strings.TrimSpace(goal.ThreadID)
	if threadID == "" || backend.Runtime == nil {
		return 0, false
	}
	startedAt := backend.Runtime.ensureStartedAtV0(threadID, now)
	if startedAt.IsZero() {
		return 0, false
	}
	return now.Sub(startedAt), true
}

func (backend serverCodexAppServerGoalBackendV0) recordCodexAppServerGoalStartedAtV0(
	threadID string,
	startedAt time.Time,
) {
	if backend.Runtime == nil {
		return
	}
	backend.Runtime.recordStartedAtV0(threadID, startedAt)
}

func (backend serverCodexAppServerGoalBackendV0) nowCodexAppServerGoalV0() time.Time {
	if backend.Now != nil {
		return backend.Now().UTC()
	}
	return time.Now().UTC()
}

func (runtime *serverCodexAppServerGoalRuntimeV0) recordStartedAtV0(threadID string, startedAt time.Time) {
	threadID = strings.TrimSpace(threadID)
	if runtime == nil || threadID == "" || startedAt.IsZero() {
		return
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.startedAt == nil {
		runtime.startedAt = map[string]time.Time{}
	}
	if _, exists := runtime.startedAt[threadID]; !exists {
		runtime.startedAt[threadID] = startedAt.UTC()
	}
}

func (runtime *serverCodexAppServerGoalRuntimeV0) ensureStartedAtV0(threadID string, now time.Time) time.Time {
	threadID = strings.TrimSpace(threadID)
	if runtime == nil || threadID == "" || now.IsZero() {
		return time.Time{}
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.startedAt == nil {
		runtime.startedAt = map[string]time.Time{}
	}
	if startedAt, exists := runtime.startedAt[threadID]; exists {
		return startedAt
	}
	now = now.UTC()
	runtime.startedAt[threadID] = now
	return now
}

type serverCodexAppServerTimestampV0 struct {
	set   bool
	value int64
}

func (timestamp *serverCodexAppServerTimestampV0) UnmarshalJSON(data []byte) error {
	if timestamp == nil {
		return nil
	}
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*timestamp = serverCodexAppServerTimestampV0{}
		return nil
	}
	if strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"") {
		unquoted, err := strconv.Unquote(raw)
		if err != nil {
			return err
		}
		if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(unquoted)); err == nil {
			timestamp.set = true
			timestamp.value = parsed.Unix()
			return nil
		}
		raw = strings.TrimSpace(unquoted)
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		parsedFloat, floatErr := strconv.ParseFloat(raw, 64)
		if floatErr != nil {
			return err
		}
		value = int64(parsedFloat)
	}
	timestamp.set = true
	timestamp.value = value
	return nil
}

func (timestamp serverCodexAppServerTimestampV0) TimeV0(now time.Time) (time.Time, bool) {
	if !timestamp.set || timestamp.value <= 0 {
		return time.Time{}, false
	}
	value := timestamp.value
	var parsed time.Time
	switch {
	case value > 1_000_000_000_000_000_000:
		parsed = time.Unix(0, value)
	case value > 1_000_000_000_000_000:
		parsed = time.UnixMicro(value)
	case value > 1_000_000_000_000:
		parsed = time.UnixMilli(value)
	case value > 1_000_000_000:
		parsed = time.Unix(value, 0)
	default:
		return time.Time{}, false
	}
	parsed = parsed.UTC()
	now = now.UTC()
	if now.IsZero() ||
		parsed.After(now.Add(5*time.Minute)) ||
		parsed.Before(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return time.Time{}, false
	}
	return parsed, true
}

func serverCodexAppServerTimestampFromTimeV0(value time.Time) serverCodexAppServerTimestampV0 {
	return serverCodexAppServerTimestampV0{set: true, value: value.Unix()}
}
