package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestCodexDeliveryObservationSourceV0IngiereACKTardioDeAgenteMarcadoLost(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	path := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-lost-late-ack",
			Spec:          spec,
			AckPath:       path,
		}},
	}
	request := codexDeliveryRequestForTestV0(spec, nil)
	request.Run.LostAgents = []string{spec.RequestID}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 || observations[0].AgentRef != spec.RequestID {
		t.Fatalf("ACK tardio lost no observado: %+v", observations)
	}
}

func TestCodexDeliveryObservationSourceV0ACKPadreConChildTaskRefNoTumbaTick(t *testing.T) {
	spec := codexDeliverySpecWithRefsForTestV0(
		"agent-ref-parent-child-collision-001",
		"task-ref-parent-child-collision-001",
		"ack-ref-parent-child-collision-001",
	)
	spec.AgentPacket.Task.ChildTaskRefs = []string{"task-ref-child-redaccion-collision-001"}
	ack := codexDeliveryAckForTestV0(spec)
	ack.TaskRef = "task-ref-child-redaccion-collision-001"
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-parent-child-collision-001",
			Spec:          spec,
			AckPath:       path,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))

	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0 no debe tumbar el tick por child ACK collision: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef ||
		observations[0].TaskID != spec.AgentPacket.Task.TaskRef ||
		observations[0].AgentRef != spec.RequestID {
		t.Fatalf("observations=%+v", observations)
	}
	if !codexDeliveryEvidenceContainsForTestV0(
		observations[0].EvidenceRefs,
		"gate-issue:"+orquestaruntimecodex.CodexAgentAckInvalidParentChildTaskCollisionEvidenceV0,
	) {
		t.Fatalf("evidence_refs=%+v", observations[0].EvidenceRefs)
	}
}

func TestCodexDeliveryObservationSourceV0IngiereACKTardioSinProcesoVivoConWaitAgentRefs(t *testing.T) {
	scopedSpec := codexDeliverySpecWithRefsForTestV0(
		"agent-ref-late-scoped-001",
		"task-ref-late-scoped-001",
		"ack-ref-late-scoped-001",
	)
	outOfScopeSpec := codexDeliverySpecWithRefsForTestV0(
		"agent-ref-late-out-of-scope-001",
		"task-ref-late-out-of-scope-001",
		"ack-ref-late-out-of-scope-001",
	)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{
			{
				DescriptorRef: "receipt-ref-late-out-of-scope-001",
				Spec:          outOfScopeSpec,
				AckPath:       writeCodexDeliveryAckForTestV0(t, outOfScopeSpec, codexDeliveryAckForTestV0(outOfScopeSpec)),
			},
			{
				DescriptorRef: "receipt-ref-late-scoped-001",
				Spec:          scopedSpec,
				AckPath:       writeCodexDeliveryAckForTestV0(t, scopedSpec, codexDeliveryAckForTestV0(scopedSpec)),
			},
		},
	}
	request := codexDeliveryRequestForTestV0(scopedSpec, nil)
	request.Run.Agents = []string{scopedSpec.RequestID, outOfScopeSpec.RequestID}
	request.Run.StartedAgents = nil
	request.Run.StoppedAgents = []string{scopedSpec.RequestID}
	request.Run.ConfirmedStoppedAgents = []string{scopedSpec.RequestID}
	request.Run.LostAgents = []string{scopedSpec.RequestID}
	request.Run.AgentAssessments = []string{"agent-assessment-ref-stalled-previo-001"}
	request.WaitAgentRefs = []string{" " + scopedSpec.RequestID + " "}

	observations, err := (CodexDeliveryObservationSourceV0{
		Store: store,
		WorktreeVerifier: staticCodexReceiptWorktreeEvidenceVerifierV0{
			Refs: []string{"gate-issue:file_outside_write_set"},
		},
	}).BuildAgentDeliveryObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].AgentRef != scopedSpec.RequestID ||
		observations[0].DeliveryRef != scopedSpec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("ACK tardio scoped no observado: %+v", observations)
	}
	if !codexDeliveryEvidenceContainsForTestV0(
		observations[0].EvidenceRefs,
		"gate-issue:file_outside_write_set",
	) {
		t.Fatalf("rail blando write-set no conservado: %+v", observations[0].EvidenceRefs)
	}
	if len(store.LastRequest.StartedAgents) != 1 ||
		store.LastRequest.StartedAgents[0] != scopedSpec.RequestID {
		t.Fatalf("scope store=%+v", store.LastRequest.StartedAgents)
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
