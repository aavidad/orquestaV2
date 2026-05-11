package orquestaruntime

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type fakeExternalAgentBatchResolverV0 struct {
	expectedProfileRef string
	req                ProcessRuntimeLaunchRequestV0
	issues             []ExternalAgentConnectorErrorV0
}

func (f fakeExternalAgentBatchResolverV0) ResolveExternalAgentProcessCommandV0(
	_ context.Context,
	spec ExternalAgentLaunchSpecV0,
) (ProcessRuntimeLaunchRequestV0, []ExternalAgentConnectorErrorV0) {
	if spec.ProfileRef != f.expectedProfileRef {
		return f.req, []ExternalAgentConnectorErrorV0{
			externalAgentProcessErrorV0(ExternalAgentCommandResolutionInvalidaV0, "profile_ref", "", false),
		}
	}
	return f.req, f.issues
}

type controlledExternalAgentBatchRuntimeV0 struct {
	mu        sync.Mutex
	active    int
	maxActive int
	next      int
	entered   chan string
	release   chan struct{}
}

func newControlledExternalAgentBatchRuntimeV0() *controlledExternalAgentBatchRuntimeV0 {
	return &controlledExternalAgentBatchRuntimeV0{
		entered: make(chan string, 8),
		release: make(chan struct{}, 8),
	}
}

func (r *controlledExternalAgentBatchRuntimeV0) LaunchV0(
	ctx context.Context,
	req ProcessRuntimeLaunchRequestV0,
) (ProcessRuntimeSnapshotV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateProcessRuntimeLaunchRequestV0(req); err != nil {
		return ProcessRuntimeSnapshotV0{}, err
	}

	processRef, sessionRef, launchRef := r.enterLaunchV0()
	defer r.leaveLaunchV0()
	r.entered <- processRef

	select {
	case <-r.release:
	case <-ctx.Done():
		return ProcessRuntimeSnapshotV0{}, processRuntimeErrorV0(ProcessRuntimeContextDoneV0, "context")
	}
	return ProcessRuntimeSnapshotV0{
		SchemaVersion: ProcessRuntimeConnectorVersionV0,
		ProcessRef:    processRef,
		SessionRef:    sessionRef,
		LaunchRef:     launchRef,
		Status:        ProcessRuntimeRunningV0,
	}, nil
}

func (r *controlledExternalAgentBatchRuntimeV0) enterLaunchV0() (string, string, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	r.active++
	if r.active > r.maxActive {
		r.maxActive = r.active
	}
	return fmt.Sprintf("process-ref-batch-%03d", r.next),
		fmt.Sprintf("session-ref-batch-%03d", r.next),
		fmt.Sprintf("launch-ref-batch-%03d", r.next)
}

func (r *controlledExternalAgentBatchRuntimeV0) leaveLaunchV0() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active--
}

func (r *controlledExternalAgentBatchRuntimeV0) releaseLaunchesV0(count int) {
	for i := 0; i < count; i++ {
		r.release <- struct{}{}
	}
}

func (r *controlledExternalAgentBatchRuntimeV0) maxActiveSeenV0() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.maxActive
}

type retryExternalAgentBatchRuntimeV0 struct {
	mu    sync.Mutex
	calls int
}

func (r *retryExternalAgentBatchRuntimeV0) LaunchV0(
	_ context.Context,
	req ProcessRuntimeLaunchRequestV0,
) (ProcessRuntimeSnapshotV0, error) {
	if err := validateProcessRuntimeLaunchRequestV0(req); err != nil {
		return ProcessRuntimeSnapshotV0{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.calls == 1 {
		return ProcessRuntimeSnapshotV0{}, processRuntimeErrorV0(ProcessRuntimeLaunchFallidoV0, "command_path")
	}
	return ProcessRuntimeSnapshotV0{
		SchemaVersion: ProcessRuntimeConnectorVersionV0,
		ProcessRef:    "process-ref-batch-retry",
		SessionRef:    "session-ref-batch-retry",
		LaunchRef:     "launch-ref-batch-retry",
		Status:        ProcessRuntimeRunningV0,
	}, nil
}

func (r *retryExternalAgentBatchRuntimeV0) callsV0() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func externalAgentBatchItemV0(
	t *testing.T,
	itemRef string,
	spec ExternalAgentLaunchSpecV0,
	issues []ExternalAgentConnectorErrorV0,
	runtime ExternalAgentProcessRuntimePortV0,
) ExternalAgentProcessBatchItemV0 {
	t.Helper()
	return ExternalAgentProcessBatchItemV0{
		ItemRef: itemRef,
		Spec:    spec,
		Resolver: fakeExternalAgentBatchResolverV0{
			expectedProfileRef: spec.ProfileRef,
			req:                processRuntimeLaunchRequestForTestV0(t, "exit"),
			issues:             issues,
		},
		Runtime: runtime,
	}
}

func externalAgentLaunchSpecPerfilV0(t *testing.T, suffix string) ExternalAgentLaunchSpecV0 {
	t.Helper()
	request := runtimeLaunchRequestValidaV0()
	request.RequestID = "req-runtime-batch-" + suffix
	request.CorrelationID = "corr-runtime-batch-" + suffix
	request.IdempotencyKey = "idem-runtime-batch-" + suffix
	packet := BuildAgentStartPacketV0(request, runtimeMaterializedContextValidoV0(t, *request.ContextBundle))
	profile := externalAgentConnectorProfileValidoV0()
	profile.ProfileRef = "external-agent-profile-" + suffix
	profile.Command.CommandRef = "external-agent-command-" + suffix
	profile.Command.ExecutableRef = "external-agent-executable-" + suffix
	profile.Command.ArgRefs = []string{"external-agent-arg-" + suffix}
	profile.Command.EnvRefs = []string{"external-agent-env-" + suffix}
	profile.Command.WorkingDirRef = "external-agent-workdir-" + suffix
	spec := BuildExternalAgentLaunchSpecV0(request, packet, profile)
	if !spec.Valid() {
		t.Fatalf("spec batch invalida: %+v", spec.Issues)
	}
	return spec
}

func waitExternalAgentBatchRuntimeEntryV0(
	t *testing.T,
	runtime *controlledExternalAgentBatchRuntimeV0,
) {
	t.Helper()
	select {
	case <-runtime.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout esperando llamada al runtime fake")
	}
}

func waitExternalAgentBatchResultsV0(
	t *testing.T,
	done <-chan []ExternalAgentProcessBatchResultV0,
) []ExternalAgentProcessBatchResultV0 {
	t.Helper()
	select {
	case results := <-done:
		return results
	case <-time.After(2 * time.Second):
		t.Fatal("timeout esperando batch")
		return nil
	}
}

func requireExternalAgentBatchCorrelationV0(
	t *testing.T,
	result ExternalAgentProcessBatchResultV0,
	index int,
	itemRef string,
	spec ExternalAgentLaunchSpecV0,
) {
	t.Helper()
	if result.Index != index ||
		result.ItemRef != itemRef ||
		result.RequestID != spec.RequestID ||
		result.CorrelationID != spec.CorrelationID ||
		result.ProfileRef != spec.ProfileRef ||
		result.ConnectorRef != spec.ConnectorRef ||
		result.RuntimeKind != spec.RuntimeKind {
		t.Fatalf("resultado no correlacionado: %+v spec=%+v", result, spec)
	}
}
