package orquestaruntimecodexappserver

import (
	"strconv"
	"strings"
	"sync"
	"time"

	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

type serverCodexAppServerGoalRuntimeV0 struct {
	mu                sync.Mutex
	startedAt         map[string]time.Time
	timeouts          map[string]time.Duration
	writeSetBaselines map[string]codexAppServerRuntimeWriteSetBaselineV0
}

type codexAppServerRuntimeWriteSetBaselineV0 struct {
	ProjectWorkDir            string
	WriteSet                  []string
	DestructiveAuthorizations []orquestaruntimeworktree.WorktreeDestructiveAuthorizationV0
	Snapshot                  orquestaruntimeworktree.WorktreeSnapshotV0
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

func (backend serverCodexAppServerGoalBackendV0) recordCodexAppServerGoalRuntimeV0(
	threadID string,
	startedAt time.Time,
	timeout time.Duration,
) {
	if backend.Runtime == nil {
		return
	}
	backend.Runtime.recordGoalRuntimeV0(threadID, startedAt, timeout)
}

func (backend serverCodexAppServerGoalBackendV0) recordCodexAppServerRuntimeWriteSetBaselineV0(
	threadID string,
	baseline codexAppServerRuntimeWriteSetBaselineV0,
) {
	if backend.Runtime == nil {
		return
	}
	backend.Runtime.recordWriteSetBaselineV0(threadID, baseline)
}

func (backend serverCodexAppServerGoalBackendV0) codexAppServerGoalTimeoutForThreadV0(threadID string) time.Duration {
	if backend.Runtime != nil {
		if timeout := backend.Runtime.timeoutForThreadV0(threadID); timeout > 0 {
			return timeout
		}
	}
	timeout := backend.Timeout
	if timeout <= 0 {
		timeout = time.Duration(defaultCodexGoalTimeoutMSV0) * time.Millisecond
	}
	return timeout
}

func (backend serverCodexAppServerGoalBackendV0) codexAppServerGoalTimeoutForPacketV0(packet orquestaruntimecodexgoal.CodexGoalStartPacketV0) time.Duration {
	timeout := backend.codexAppServerGoalTimeoutForThreadV0("")
	if packet.Budget.MaxRuntimeSeconds > 0 {
		budgetTimeout := time.Duration(packet.Budget.MaxRuntimeSeconds) * time.Second
		if budgetTimeout > timeout {
			timeout = budgetTimeout
		}
	}
	return timeout
}

func (backend serverCodexAppServerGoalBackendV0) nowCodexAppServerGoalV0() time.Time {
	if backend.Now != nil {
		return backend.Now().UTC()
	}
	return time.Now().UTC()
}

func (runtime *serverCodexAppServerGoalRuntimeV0) recordGoalRuntimeV0(
	threadID string,
	startedAt time.Time,
	timeout time.Duration,
) {
	threadID = strings.TrimSpace(threadID)
	if runtime == nil || threadID == "" {
		return
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.startedAt == nil {
		runtime.startedAt = map[string]time.Time{}
	}
	if runtime.timeouts == nil {
		runtime.timeouts = map[string]time.Duration{}
	}
	if !startedAt.IsZero() {
		if _, exists := runtime.startedAt[threadID]; !exists {
			runtime.startedAt[threadID] = startedAt.UTC()
		}
	}
	if timeout > 0 {
		if _, exists := runtime.timeouts[threadID]; !exists {
			runtime.timeouts[threadID] = timeout
		}
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

func (runtime *serverCodexAppServerGoalRuntimeV0) timeoutForThreadV0(threadID string) time.Duration {
	threadID = strings.TrimSpace(threadID)
	if runtime == nil || threadID == "" {
		return 0
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.timeouts == nil {
		return 0
	}
	return runtime.timeouts[threadID]
}

func (runtime *serverCodexAppServerGoalRuntimeV0) recordWriteSetBaselineV0(
	threadID string,
	baseline codexAppServerRuntimeWriteSetBaselineV0,
) {
	threadID = strings.TrimSpace(threadID)
	if runtime == nil ||
		threadID == "" ||
		strings.TrimSpace(baseline.ProjectWorkDir) == "" ||
		baseline.Snapshot.SchemaVersion == "" ||
		len(baseline.WriteSet) == 0 {
		return
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.writeSetBaselines == nil {
		runtime.writeSetBaselines = map[string]codexAppServerRuntimeWriteSetBaselineV0{}
	}
	if _, exists := runtime.writeSetBaselines[threadID]; exists {
		return
	}
	runtime.writeSetBaselines[threadID] = codexAppServerRuntimeWriteSetBaselineV0{
		ProjectWorkDir:            strings.TrimSpace(baseline.ProjectWorkDir),
		WriteSet:                  append([]string(nil), baseline.WriteSet...),
		DestructiveAuthorizations: append([]orquestaruntimeworktree.WorktreeDestructiveAuthorizationV0(nil), baseline.DestructiveAuthorizations...),
		Snapshot:                  baseline.Snapshot,
	}
}

func (runtime *serverCodexAppServerGoalRuntimeV0) writeSetBaselineForThreadV0(
	threadID string,
) (codexAppServerRuntimeWriteSetBaselineV0, bool) {
	threadID = strings.TrimSpace(threadID)
	if runtime == nil || threadID == "" {
		return codexAppServerRuntimeWriteSetBaselineV0{}, false
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.writeSetBaselines == nil {
		return codexAppServerRuntimeWriteSetBaselineV0{}, false
	}
	baseline, ok := runtime.writeSetBaselines[threadID]
	if !ok {
		return codexAppServerRuntimeWriteSetBaselineV0{}, false
	}
	baseline.WriteSet = append([]string(nil), baseline.WriteSet...)
	baseline.DestructiveAuthorizations = append([]orquestaruntimeworktree.WorktreeDestructiveAuthorizationV0(nil), baseline.DestructiveAuthorizations...)
	return baseline, true
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
