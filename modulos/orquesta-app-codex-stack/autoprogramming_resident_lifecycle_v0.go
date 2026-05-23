package orquestaappcodexstack

import (
	"context"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

const autoprogrammingResidentBacklogContinuesEvidenceV0 = "evidence-ref-autoprogramming-resident-backlog-continues"

type autoprogrammingResidentLifecycleV0 struct {
	inner CodexSupervisorStackLifecycleV0
}

var _ CodexSupervisorAgentLifecyclePortV0 = autoprogrammingResidentLifecycleV0{}

func (lifecycle autoprogrammingResidentLifecycleV0) LaunchV0(
	ctx context.Context,
) (CodexSupervisorRuntimeSnapshotV0, error) {
	snapshot, err := lifecycle.inner.LaunchV0(ctx)
	return autoprogrammingResidentBacklogSnapshotV0(snapshot), err
}

func (lifecycle autoprogrammingResidentLifecycleV0) ContinueV0(
	ctx context.Context,
	message string,
) (CodexSupervisorRuntimeSnapshotV0, error) {
	snapshot, err := lifecycle.inner.ContinueV0(ctx, message)
	return autoprogrammingResidentBacklogSnapshotV0(snapshot), err
}

func autoprogrammingResidentBacklogSnapshotV0(
	snapshot CodexSupervisorRuntimeSnapshotV0,
) CodexSupervisorRuntimeSnapshotV0 {
	if snapshot.Status != CodexSupervisorRuntimeDoneV0 ||
		!codexSupervisorEvidenceRefsContainPartV0(snapshot.EvidenceRefs, "evidence-ref-codex-supervisor-stack-global") ||
		codexSupervisorEvidenceRefsContainPartV0(snapshot.EvidenceRefs, orquestarunsupervisor.RunSupervisorStopNoExecutionV0) {
		return snapshot
	}
	snapshot.Status = CodexSupervisorRuntimeRunningV0
	snapshot.EvidenceRefs = compactStringsV0(append(
		snapshot.EvidenceRefs,
		autoprogrammingResidentBacklogContinuesEvidenceV0,
	))
	return snapshot
}
