package orquestaruntimecodexdelivery

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexDeliveryObservationSourceV0ConstruyeObservacionNeutral(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	path := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			Spec:          spec,
			AckPath:       path,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations=%d", len(observations))
	}
	got := observations[0]
	if got.DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef ||
		got.ArtifactRef != spec.AgentPacket.DeliveryRefs.AckRef ||
		got.AgentRef != spec.RequestID ||
		got.TaskID != spec.AgentPacket.Task.TaskRef {
		t.Fatalf("observacion no correlada: %+v", got)
	}
	if observationLeaksCodexDeliveryPathV0(got, path) {
		t.Fatalf("observacion filtra path: %+v", got)
	}
	if store.LastRequest.RunID != "run-ref-001" ||
		len(store.LastRequest.StartedAgents) != 1 {
		t.Fatalf("request store=%+v", store.LastRequest)
	}
}

func TestCodexDeliveryObservationSourceV0OmiteDeliveryYaRegistrada(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	path := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			Spec:          spec,
			AckPath:       path,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(
			context.Background(),
			codexDeliveryRequestForTestV0(spec, []string{spec.AgentPacket.DeliveryRefs.AckRef}),
		)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexDeliveryObservationSourceV0IngiereACKCompletoAunqueAgenteEsteCerrado(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	path := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			Spec:          spec,
			AckPath:       path,
		}},
	}
	request := codexDeliveryRequestForTestV0(spec, nil)
	request.Run.StoppedAgents = []string{spec.RequestID}
	request.Run.ConfirmedStoppedAgents = []string{spec.RequestID}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 || observations[0].AgentRef != spec.RequestID {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexDeliveryObservationSourceV0CompactaEvidenciasParaCore(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	path := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:  "receipt-ref-001",
			Spec:           spec,
			AckPath:        path,
			ProjectWorkDir: t.TempDir(),
		}},
	}
	source := CodexDeliveryObservationSourceV0{
		Store: store,
		WorktreeVerifier: staticCodexReceiptWorktreeEvidenceVerifierV0{
			Refs: manyCodexDeliveryEvidenceRefsForTestV0(),
		},
	}

	observations, err := source.BuildAgentDeliveryObservationsV0(
		context.Background(),
		codexDeliveryRequestForTestV0(spec, nil),
	)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations=%+v", observations)
	}
	if len(observations[0].EvidenceRefs) > 20 ||
		!codexDeliveryEvidenceContainsForTestV0(
			observations[0].EvidenceRefs,
			"evidence-ref-codex-delivery-evidence-truncated",
		) {
		t.Fatalf("evidence_refs=%+v", observations[0].EvidenceRefs)
	}
	for _, ref := range observations[0].EvidenceRefs {
		if len([]rune(ref)) > 260 {
			t.Fatalf("evidence ref demasiado larga: %d %q", len([]rune(ref)), ref)
		}
	}
}

func TestCodexDeliveryObservationSourceV0WaitAgentRefsAcotaACKs(t *testing.T) {
	scopedSpec := codexDeliverySpecWithRefsForTestV0(
		"agent-ref-scoped-001",
		"task-ref-scoped-001",
		"ack-ref-scoped-001",
	)
	outOfScopeSpec := codexDeliverySpecWithRefsForTestV0(
		"agent-ref-out-of-scope-001",
		"task-ref-out-of-scope-001",
		"ack-ref-out-of-scope-001",
	)
	outOfScopeAckPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	if err := os.WriteFile(outOfScopeAckPath, []byte(`{"schema_version":"codex_agent_ack.v0"}`), 0o600); err != nil {
		t.Fatalf("write ack out of scope: %v", err)
	}
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{
			{
				DescriptorRef: "receipt-ref-out-of-scope-001",
				Spec:          outOfScopeSpec,
				AckPath:       outOfScopeAckPath,
			},
			{
				DescriptorRef: "receipt-ref-scoped-001",
				Spec:          scopedSpec,
				AckPath:       writeCodexDeliveryAckForTestV0(t, scopedSpec, codexDeliveryAckForTestV0(scopedSpec)),
			},
		},
	}
	request := codexDeliveryRequestForTestV0(scopedSpec, nil)
	request.Run.Agents = []string{outOfScopeSpec.RequestID, scopedSpec.RequestID}
	request.Run.StartedAgents = []string{
		outOfScopeSpec.RequestID,
		" " + scopedSpec.RequestID + " ",
		scopedSpec.RequestID,
	}
	request.WaitAgentRefs = []string{" " + scopedSpec.RequestID + " ", scopedSpec.RequestID}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].AgentRef != scopedSpec.RequestID ||
		observations[0].DeliveryRef != scopedSpec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("observations=%+v", observations)
	}
	if len(store.LastRequest.StartedAgents) != 1 ||
		store.LastRequest.StartedAgents[0] != scopedSpec.RequestID {
		t.Fatalf("started_agents store=%+v", store.LastRequest.StartedAgents)
	}
}

func TestCodexDeliveryObservationSourceV0WaitAgentRefsVacioMantieneRunCompleto(t *testing.T) {
	firstSpec := codexDeliverySpecWithRefsForTestV0(
		"agent-ref-legacy-a-001",
		"task-ref-legacy-a-001",
		"ack-ref-legacy-a-001",
	)
	secondSpec := codexDeliverySpecWithRefsForTestV0(
		"agent-ref-legacy-b-001",
		"task-ref-legacy-b-001",
		"ack-ref-legacy-b-001",
	)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{
			{
				DescriptorRef: "receipt-ref-legacy-a-001",
				Spec:          firstSpec,
				AckPath:       writeCodexDeliveryAckForTestV0(t, firstSpec, codexDeliveryAckForTestV0(firstSpec)),
			},
			{
				DescriptorRef: "receipt-ref-legacy-b-001",
				Spec:          secondSpec,
				AckPath:       writeCodexDeliveryAckForTestV0(t, secondSpec, codexDeliveryAckForTestV0(secondSpec)),
			},
		},
	}
	request := codexDeliveryRequestForTestV0(firstSpec, nil)
	request.Run.Agents = []string{firstSpec.RequestID, secondSpec.RequestID}
	request.Run.StartedAgents = []string{firstSpec.RequestID, secondSpec.RequestID}
	request.WaitAgentRefs = []string{" ", ""}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 2 ||
		!codexDeliveryObservationRefsForTestV0(observations, firstSpec.RequestID) ||
		!codexDeliveryObservationRefsForTestV0(observations, secondSpec.RequestID) {
		t.Fatalf("observations=%+v", observations)
	}
	if len(store.LastRequest.StartedAgents) != 2 {
		t.Fatalf("started_agents store=%+v", store.LastRequest.StartedAgents)
	}
}

func TestCodexDeliveryObservationSourceV0OmiteACKNoListoSinError(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	path := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			Spec:          spec,
			AckPath:       path,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexDeliveryObservationSourceV0PropagaACKInvalidoSinPath(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	dir := t.TempDir()
	path := filepath.Join(dir, orquestaruntimecodex.CodexAgentAckFileNameV0)
	if err := os.WriteFile(path, []byte(`{"schema_version":"codex_agent_ack.v0"}`), 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			Spec:          spec,
			AckPath:       path,
		}},
	}

	_, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err == nil {
		t.Fatalf("esperaba error")
	}
	if strings.Contains(err.Error(), dir) || strings.Contains(err.Error(), path) {
		t.Fatalf("error filtra path: %v", err)
	}
}

type staticCodexReceiptStoreV0 struct {
	Descriptors []CodexReceiptDescriptorV0
	LastRequest CodexReceiptDescriptorRequestV0
}

type staticCodexReceiptWorktreeEvidenceVerifierV0 struct {
	Refs []string
}

func (verifier staticCodexReceiptWorktreeEvidenceVerifierV0) VerifyCodexReceiptWorktreeV0(
	context.Context,
	CodexReceiptWorktreeVerificationRequestV0,
) error {
	return nil
}

func (verifier staticCodexReceiptWorktreeEvidenceVerifierV0) VerifyCodexReceiptWorktreeEvidenceRefsV0(
	context.Context,
	CodexReceiptWorktreeVerificationRequestV0,
) ([]string, error) {
	return append([]string(nil), verifier.Refs...), nil
}

func (store *staticCodexReceiptStoreV0) ListCodexReceiptDescriptorsV0(
	_ context.Context,
	request CodexReceiptDescriptorRequestV0,
) ([]CodexReceiptDescriptorV0, error) {
	store.LastRequest = request
	return store.Descriptors, nil
}

func codexDeliveryRequestForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	deliveries []string,
) orquestacionnucleoapp.AgentDeliveryObservationRequestV0 {
	return orquestacionnucleoapp.AgentDeliveryObservationRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:         "run-ref-001",
			Agents:        []string{spec.RequestID},
			StartedAgents: []string{spec.RequestID},
			Deliveries:    deliveries,
		},
		CorrelationID: "corr-receipt-source-001",
	}
}

func codexDeliverySpecForTestV0() orquestaruntime.ExternalAgentLaunchSpecV0 {
	return orquestaruntime.ExternalAgentLaunchSpecV0{
		RequestID:     "agent-ref-001",
		CorrelationID: "corr-agent-001",
		AgentPacket: orquestaruntime.AgentStartPacketV0{
			RequestID:     "agent-ref-001",
			CorrelationID: "corr-agent-001",
			TargetModule:  "agenda-app",
			Phase:         string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Task: orquestaruntime.AgentStartTaskV0{
				TaskRef:       "task-ref-001",
				WriteSet:      []string{"README.md"},
				RequiredTests: []string{"go test ./..."},
			},
			Context: orquestacontext.ContextMaterializedBundleV0{
				SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
				BundleRef:     "bundle-ref-001",
				WorkOrderRef:  "task-ref-001",
				TargetModule:  "agenda-app",
			},
			DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
				MailboxRef:   "mailbox-ref-001",
				AckRef:       "ack-ref-001",
				ReadinessRef: "readiness-ref-001",
			},
		},
	}
}

func codexDeliverySpecWithRefsForTestV0(
	agentRef string,
	taskRef string,
	ackRef string,
) orquestaruntime.ExternalAgentLaunchSpecV0 {
	spec := codexDeliverySpecForTestV0()
	spec.RequestID = agentRef
	spec.CorrelationID = "corr-" + agentRef
	spec.AgentPacket.RequestID = agentRef
	spec.AgentPacket.CorrelationID = spec.CorrelationID
	spec.AgentPacket.Task.TaskRef = taskRef
	spec.AgentPacket.Context.WorkOrderRef = taskRef
	spec.AgentPacket.DeliveryRefs.AckRef = ackRef
	return spec
}

func codexDeliveryAckForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) orquestaruntimecodex.CodexAgentAckV0 {
	return orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     spec.RequestID,
		CorrelationID: spec.CorrelationID,
		AckRef:        spec.AgentPacket.DeliveryRefs.AckRef,
		TargetModule:  spec.AgentPacket.TargetModule,
		TaskRef:       spec.AgentPacket.Task.TaskRef,
		Status:        "completed",
		Files:         orquestaruntimecodex.EvidenceListV0{"README.md"},
		Tests:         orquestaruntimecodex.EvidenceListV0{"go test ./..."},
		Notes:         orquestaruntimecodex.EvidenceListV0{"contexto_ref_only_resuelto: fixture local sin contexto externo"},
	}
}

func writeCodexDeliveryAckForTestV0(
	t *testing.T,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
) string {
	t.Helper()
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	path := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write ack for %s: %v", spec.RequestID, err)
	}
	return path
}

func observationLeaksCodexDeliveryPathV0(
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	path string,
) bool {
	values := append([]string{
		observation.CandidateRef,
		observation.DeliveryRef,
		observation.PhaseID,
		observation.TaskID,
		observation.AgentRef,
		observation.Summary,
	}, observation.EvidenceRefs...)
	for _, value := range values {
		if strings.Contains(value, path) {
			return true
		}
	}
	return false
}

func codexDeliveryObservationRefsForTestV0(
	observations []orquestacionnucleoapp.AgentDeliveryObservationV0,
	agentRef string,
) bool {
	for _, observation := range observations {
		if observation.AgentRef == agentRef {
			return true
		}
	}
	return false
}

func manyCodexDeliveryEvidenceRefsForTestV0() []string {
	refs := make([]string, 0, 40)
	for index := 0; index < 40; index++ {
		refs = append(refs, strings.Repeat("evidence-ref-worktree-soft-issue-", 12)+string(rune('a'+index%20)))
	}
	return refs
}

func codexDeliveryEvidenceContainsForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
