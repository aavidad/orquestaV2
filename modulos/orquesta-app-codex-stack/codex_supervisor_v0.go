package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

const DefaultCodexSupervisorContinueMessageV0 = "sigue"

type CodexSupervisorRuntimeStateV0 string

const (
	CodexSupervisorRuntimePendingV0       CodexSupervisorRuntimeStateV0 = "pending"
	CodexSupervisorRuntimeRunningV0       CodexSupervisorRuntimeStateV0 = "running"
	CodexSupervisorRuntimeRunningLiveV0   CodexSupervisorRuntimeStateV0 = "running_live"
	CodexSupervisorRuntimeWaitingOutboxV0 CodexSupervisorRuntimeStateV0 = "waiting_outbox"
	CodexSupervisorRuntimeStalledV0       CodexSupervisorRuntimeStateV0 = "stalled"
	CodexSupervisorRuntimeLaunchFailedV0  CodexSupervisorRuntimeStateV0 = "launch_failed"
	CodexSupervisorRuntimeNeedsReplanV0   CodexSupervisorRuntimeStateV0 = "needs_replan"
	CodexSupervisorRuntimeStoppedV0       CodexSupervisorRuntimeStateV0 = "stopped"
	CodexSupervisorRuntimeDoneV0          CodexSupervisorRuntimeStateV0 = "done"
	CodexSupervisorRuntimeFailedV0        CodexSupervisorRuntimeStateV0 = "failed"
)

type CodexSupervisorStopReasonV0 string

const (
	CodexSupervisorStopDoneV0         CodexSupervisorStopReasonV0 = "done"
	CodexSupervisorStopFailedV0       CodexSupervisorStopReasonV0 = "failed"
	CodexSupervisorStopStoppedV0      CodexSupervisorStopReasonV0 = "stopped"
	CodexSupervisorStopDispatchV0     CodexSupervisorStopReasonV0 = "dispatch_started"
	CodexSupervisorStopMaxTicksV0     CodexSupervisorStopReasonV0 = "max_ticks"
	CodexSupervisorStopContextDoneV0  CodexSupervisorStopReasonV0 = "context_done"
	CodexSupervisorStopRuntimeErrorV0 CodexSupervisorStopReasonV0 = "runtime_error"
	CodexSupervisorStopNoRuntimeV0    CodexSupervisorStopReasonV0 = "no_runtime"
)

var ErrCodexSupervisorRuntimeRequiredV0 = errors.New("codex_supervisor: runtime required")

type CodexSupervisorAgentLifecyclePortV0 interface {
	LaunchV0(context.Context) (CodexSupervisorRuntimeSnapshotV0, error)
	ContinueV0(context.Context, string) (CodexSupervisorRuntimeSnapshotV0, error)
}

type CodexSupervisorRuntimePortV0 = CodexSupervisorAgentLifecyclePortV0

type CodexSupervisorRuntimeSnapshotV0 struct {
	Status       CodexSupervisorRuntimeStateV0                 `json:"status"`
	SessionRef   string                                        `json:"session_ref,omitempty"`
	AgentRef     string                                        `json:"agent_ref,omitempty"`
	ProcessRef   string                                        `json:"process_ref,omitempty"`
	EvidenceRefs []string                                      `json:"evidence_refs,omitempty"`
	Diagnostics  []orquestaruncoordinator.RunDrainDiagnosticV0 `json:"diagnostics,omitempty"`
}

type CodexSupervisorDepsV0 struct {
	AgentLifecycle CodexSupervisorAgentLifecyclePortV0
	Runtime        CodexSupervisorRuntimePortV0
}

type CodexSupervisorCommandV0 struct {
	MaxTicks        int
	ContinueMessage string
}

type CodexSupervisorTickV0 struct {
	TickNumber int                              `json:"tick_number"`
	Action     string                           `json:"action"`
	Snapshot   CodexSupervisorRuntimeSnapshotV0 `json:"snapshot"`
}

type CodexSupervisorResultV0 struct {
	StopReason CodexSupervisorStopReasonV0      `json:"stop_reason"`
	Ticks      int                              `json:"ticks"`
	History    []CodexSupervisorTickV0          `json:"history,omitempty"`
	Last       CodexSupervisorRuntimeSnapshotV0 `json:"last,omitempty"`
}

func SuperviseCodexV0(
	ctx context.Context,
	deps CodexSupervisorDepsV0,
	command CodexSupervisorCommandV0,
) (CodexSupervisorResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	command = normalizeCodexSupervisorCommandV0(command)
	result := CodexSupervisorResultV0{}
	agentLifecycle := codexSupervisorAgentLifecycleV0(deps)
	if agentLifecycle == nil {
		result.StopReason = CodexSupervisorStopNoRuntimeV0
		return result, ErrCodexSupervisorRuntimeRequiredV0
	}
	for tickNumber := 1; tickNumber <= command.MaxTicks; tickNumber++ {
		if err := ctx.Err(); err != nil {
			result.StopReason = CodexSupervisorStopContextDoneV0
			return result, err
		}
		action := codexSupervisorActionV0(tickNumber)
		snapshot, err := codexSupervisorRuntimeStepV0(ctx, agentLifecycle, action, command.ContinueMessage)
		result.Ticks = tickNumber
		result.Last = snapshot
		result.History = append(result.History, CodexSupervisorTickV0{
			TickNumber: tickNumber,
			Action:     action,
			Snapshot:   snapshot,
		})
		if err != nil {
			result.StopReason = CodexSupervisorStopRuntimeErrorV0
			return result, err
		}
		if codexSupervisorRuntimeDoneV0(snapshot.Status) {
			result.StopReason = CodexSupervisorStopDoneV0
			return result, nil
		}
		if codexSupervisorRuntimeFailedV0(snapshot.Status) {
			result.StopReason = CodexSupervisorStopFailedV0
			return result, nil
		}
		if codexSupervisorRuntimeStoppedV0(snapshot.Status) {
			result.StopReason = CodexSupervisorStopStoppedV0
			return result, nil
		}
		if codexSupervisorRuntimeDispatchStartedV0(snapshot.Status) {
			result.StopReason = CodexSupervisorStopDispatchV0
			return result, nil
		}
	}
	result.StopReason = CodexSupervisorStopMaxTicksV0
	return result, nil
}

func normalizeCodexSupervisorCommandV0(
	command CodexSupervisorCommandV0,
) CodexSupervisorCommandV0 {
	if command.MaxTicks <= 0 {
		command.MaxTicks = 1
	}
	command.ContinueMessage = strings.TrimSpace(command.ContinueMessage)
	if command.ContinueMessage == "" {
		command.ContinueMessage = DefaultCodexSupervisorContinueMessageV0
	}
	return command
}

func codexSupervisorAgentLifecycleV0(
	deps CodexSupervisorDepsV0,
) CodexSupervisorAgentLifecyclePortV0 {
	if deps.AgentLifecycle != nil {
		return deps.AgentLifecycle
	}
	return deps.Runtime
}

func codexSupervisorActionV0(tickNumber int) string {
	if tickNumber <= 1 {
		return "launch"
	}
	return "continue"
}

func codexSupervisorRuntimeStepV0(
	ctx context.Context,
	runtime CodexSupervisorRuntimePortV0,
	action string,
	continueMessage string,
) (CodexSupervisorRuntimeSnapshotV0, error) {
	if action == "launch" {
		return runtime.LaunchV0(ctx)
	}
	return runtime.ContinueV0(ctx, continueMessage)
}

func codexSupervisorRuntimeDoneV0(state CodexSupervisorRuntimeStateV0) bool {
	switch strings.TrimSpace(string(state)) {
	case string(CodexSupervisorRuntimeDoneV0), "completed", "complete":
		return true
	default:
		return false
	}
}

func codexSupervisorRuntimeFailedV0(state CodexSupervisorRuntimeStateV0) bool {
	switch strings.TrimSpace(string(state)) {
	case string(CodexSupervisorRuntimeFailedV0), string(CodexSupervisorRuntimeLaunchFailedV0), "error", "errored":
		return true
	default:
		return false
	}
}

func codexSupervisorRuntimeStoppedV0(state CodexSupervisorRuntimeStateV0) bool {
	switch strings.TrimSpace(string(state)) {
	case string(CodexSupervisorRuntimeNeedsReplanV0), string(CodexSupervisorRuntimeStoppedV0), "blocked", "paused", "stop_requested":
		return true
	default:
		return false
	}
}

func codexSupervisorRuntimeDispatchStartedV0(state CodexSupervisorRuntimeStateV0) bool {
	switch strings.TrimSpace(string(state)) {
	case string(CodexSupervisorRuntimeRunningLiveV0):
		return true
	default:
		return false
	}
}
