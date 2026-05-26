package orquestaserver

import (
	"strings"
	"testing"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestStatusTrackerV0ProyectaMensajesOperativosSensiblesV0(t *testing.T) {
	now := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{}, now)
	sensitive := "fallo con token=abc en /tmp/orquesta-runtime-secret"

	state := tracker.MarkSupervisorErrorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{},
		orquestarunsupervisor.RunSupervisorResultV0{},
		sensitive,
		now,
	)
	if state.LastSupervisorError != serverOperationalMessageRedactedV0 ||
		state.SupervisorLastError != serverOperationalMessageRedactedV0 ||
		state.LastError != serverOperationalMessageRedactedV0 ||
		len(state.RecentErrors) != 1 ||
		state.RecentErrors[0].Message != serverOperationalMessageRedactedV0 {
		t.Fatalf("state supervisor=%+v", state)
	}
	assertOperationalMessageDoesNotLeakV0(t, state.LastSupervisorError, "token=abc", "/tmp/orquesta-runtime-secret")

	state = tracker.MarkStartupBlockedV0(StartupCheckResultV0{
		Status:       StartupCheckStatusBlockedV0,
		Message:      sensitive,
		EvidenceRefs: []string{"evidence-ref-startup-sensitive"},
	}, now)
	if state.StartupMessage != serverStartupOperationalMessageRedactedV0 ||
		state.LastError != serverStartupOperationalMessageRedactedV0 ||
		state.RecentErrors[0].Message != serverStartupOperationalMessageRedactedV0 {
		t.Fatalf("state startup=%+v", state)
	}

	state = tracker.MarkIdleSelfImprovementPreparedV0(IdleSelfImprovementResultV0{
		Accepted:    true,
		RunRef:      "run-ref-idle-self-improvement-001",
		RequestRef:  "request-ref-idle-self-improvement-001",
		Status:      "prepared",
		Message:     sensitive,
		NextActions: []string{"revisar /tmp/orquesta-runtime-secret"},
	}, now)
	if strings.Contains(state.IdleSelfImprovementReason, "token=abc") ||
		strings.Contains(state.IdleSelfImprovementReason, "/tmp/orquesta-runtime-secret") ||
		!strings.Contains(state.IdleSelfImprovementReason, "message="+serverOperationalMessageRedactedV0) ||
		!strings.Contains(state.IdleSelfImprovementReason, "next="+serverOperationalMessageRedactedV0) {
		t.Fatalf("idle reason=%q", state.IdleSelfImprovementReason)
	}
}

func TestStatusTrackerV0TruncaMensajesOperativosLargosV0(t *testing.T) {
	now := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{}, now)
	state := tracker.MarkErrorV0(strings.Repeat("a", 260), now)

	if len([]rune(state.LastError)) > serverOperationalMessageMaxRunesV0 ||
		!strings.HasSuffix(state.LastError, serverOperationalMessageTruncatedSuffixV0) ||
		state.RecentErrors[0].Message != state.LastError {
		t.Fatalf("last_error=%q recent=%+v", state.LastError, state.RecentErrors)
	}
}

func assertOperationalMessageDoesNotLeakV0(t *testing.T, message string, forbidden ...string) {
	t.Helper()
	for _, value := range forbidden {
		if strings.Contains(message, value) {
			t.Fatalf("mensaje filtra %q en %q", value, message)
		}
	}
}
