package candidateseal

import (
	"strings"
	"testing"
)

func TestPhysicalGateBReceiptBindsExactEvidenceAndCannotAccreditV38(t *testing.T) {
	binding := physicalGateBBinding(t, "physical")
	subjects := physicalGateBSubjects("one")
	receipt, err := SealPhysicalGateBReceipt(physicalGateBDraft(), binding, subjects)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyPhysicalGateBReceipt(receipt, binding, subjects); err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodePhysicalGateBReceipt(receipt)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodePhysicalGateBReceipt(encoded)
	if err != nil || decoded.ReceiptSHA256 != receipt.ReceiptSHA256 {
		t.Fatalf("decode=%+v err=%v", decoded, err)
	}
	if receipt.Proof.GateCClaimed || receipt.Proof.V38AccreditationClaimed {
		t.Fatal("Gate B receipt overclaimed Gate C or V38")
	}
}

func TestPhysicalGateBReceiptRejectsCrossedOrIncompletePhysicalEvidence(t *testing.T) {
	binding := physicalGateBBinding(t, "one")
	other := physicalGateBBinding(t, "two")
	subjects := physicalGateBSubjects("one")
	receipt, err := SealPhysicalGateBReceipt(physicalGateBDraft(), binding, subjects)
	if err != nil {
		t.Fatal(err)
	}
	if VerifyPhysicalGateBReceipt(receipt, other, subjects) == nil {
		t.Fatal("crossed Gate A/B binding accepted")
	}
	changed := clonePhysicalGateBSubjects(subjects)
	changed["preservation_manifest"][0] ^= 1
	if VerifyPhysicalGateBReceipt(receipt, binding, changed) == nil {
		t.Fatal("changed physical evidence accepted")
	}
	missing := clonePhysicalGateBSubjects(subjects)
	delete(missing, "cleanup")
	if VerifyPhysicalGateBReceipt(receipt, binding, missing) == nil {
		t.Fatal("missing cleanup evidence accepted")
	}

	mutations := []func(*PhysicalGateBReceipt){
		func(v *PhysicalGateBReceipt) { v.Status = "prepared_not_executed" },
		func(v *PhysicalGateBReceipt) { v.Run.ActionFence++ },
		func(v *PhysicalGateBReceipt) { v.Run.ReconciliationOutcome = "quarantined" },
		func(v *PhysicalGateBReceipt) { v.Run.LifecycleState = "preserved" },
		func(v *PhysicalGateBReceipt) { v.Proof.MicroVMCount = 2 },
		func(v *PhysicalGateBReceipt) { v.Proof.DirectInternet = true },
		func(v *PhysicalGateBReceipt) { v.Proof.NegativeCases = v.Proof.NegativeCases[:5] },
		func(v *PhysicalGateBReceipt) { v.Proof.FirecrackerProcessesAfter = 1 },
		func(v *PhysicalGateBReceipt) { v.Proof.GateCClaimed = true },
		func(v *PhysicalGateBReceipt) { v.Proof.V38AccreditationClaimed = true },
		func(v *PhysicalGateBReceipt) { v.Timeline.ClosedAt = v.Timeline.StartedAt },
		func(v *PhysicalGateBReceipt) { v.ReceiptSHA256 = testDigest('f') },
	}
	for index, mutate := range mutations {
		changedReceipt := receipt
		changedReceipt.Proof.AllowedServices = append([]string(nil), receipt.Proof.AllowedServices...)
		changedReceipt.Proof.NegativeCases = append([]string(nil), receipt.Proof.NegativeCases...)
		changedReceipt.Subjects = append([]PhysicalGateBSubject(nil), receipt.Subjects...)
		mutate(&changedReceipt)
		if VerifyPhysicalGateBReceipt(changedReceipt, binding, subjects) == nil {
			t.Fatalf("mutation %d accepted", index)
		}
	}
}

func TestPhysicalGateBReceiptDecodeRejectsUnknownDuplicateAndTrailingJSON(t *testing.T) {
	binding := physicalGateBBinding(t, "decode")
	receipt, err := SealPhysicalGateBReceipt(physicalGateBDraft(), binding, physicalGateBSubjects("decode"))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodePhysicalGateBReceipt(receipt)
	if err != nil {
		t.Fatal(err)
	}
	invalid := [][]byte{
		[]byte(strings.Replace(string(encoded), "{", `{"unknown":true,`, 1)),
		[]byte(strings.Replace(string(encoded), `"schema":`, `"schema":"duplicate","schema":`, 1)),
		append(append([]byte(nil), encoded...), []byte("{}")...),
	}
	for index, candidate := range invalid {
		if _, err := DecodePhysicalGateBReceipt(candidate); err == nil {
			t.Fatalf("invalid JSON %d accepted", index)
		}
	}
}

func physicalGateBBinding(t *testing.T, suffix string) GateBBinding {
	t.Helper()
	gateA, err := Build(candidateInput("physical-" + suffix))
	if err != nil {
		t.Fatal(err)
	}
	binding, err := BuildGateBBinding(gateA, GateBConnector{
		Module: "github.com/aavidad/agente_microvm/conectores/orquesta", Version: "v41",
		ModuleSum: "h1:physical", ContractSHA256: testDigest('b'),
	})
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func physicalGateBSubjects(suffix string) map[string][]byte {
	result := make(map[string][]byte, len(physicalGateBSubjectRoles))
	for _, role := range physicalGateBSubjectRoles {
		result[role] = []byte(role + ":" + suffix)
	}
	return result
}

func clonePhysicalGateBSubjects(source map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(source))
	for role, content := range source {
		result[role] = append([]byte(nil), content...)
	}
	return result
}

func physicalGateBDraft() PhysicalGateBReceipt {
	execution := "execution:189f66f40fb6dd5d30301bf211035086"
	action := "action:launch:" + execution
	intent := "effect-intent:" + action
	reconciliation := "agent-launch-reconciliation-authority:one"
	return PhysicalGateBReceipt{
		Run: PhysicalGateBRun{
			ProjectRef: "project:default", GoalRef: "goal:e5aa533c285ee4fdc6727fa86ac367ff",
			WorkItemRef: "work-item:c87fa56888d135e84d3e7017f823f319", ExecutionRef: execution,
			ActionRef: action, EffectIntentRef: intent,
			EffectIntentDigest: strings.Repeat("7", 64),
			EffectAttemptRef:   "effect-attempt:" + action + ":claim:143b6491a2f4795489eced1f91ba78fb",
			PlanGeneration:     1, WorkItemGeneration: 1, ActionFence: 1,
			ReconciliationAuthorityRef: reconciliation,
			ReconciliationAttemptRef:   "agent-launch-reconciliation-attempt:one",
			ReconciliationReceiptRef:   "agent-launch-reconciliation-receipt:" + reconciliation,
			ContinuationSubjectRef:     "expired-launch-continuation-subject:one",
			ContinuationAuthorityRef:   "continuation:one",
			EffectReceiptRef:           "effect-receipt:" + intent,
			AMVLaunchRef:               "lanzamiento:one", AMVExecutionRef: "ejecucion:one", AMVRunRef: "run:one", AMVCID: 42,
			ResultArtifactRef: "artifact:one", ResultMarker: "ORQUESTA_V38_C46_OK_one",
			PreservationReceiptRef: "environment-receipt:one", PhysicalManifestRef: "manifest:one",
			InventoryArtifactRef: "artifact:inventory:one", ReconciliationOutcome: "completed",
			PreservationState: "preserved_pending_review", LifecycleState: "closed",
			GoalState: "succeeded", WorkItemState: "succeeded", ExecutionState: "succeeded",
		},
		Proof: PhysicalGateBProof{
			AgentCount: 1, MicroVMCount: 1, Transport: "vsock_only",
			AllowedServices: []string{"controlled_egress_proxy", "orquesta_broker"},
			RealCodex:       true, WorkCompleted: true, Quiesced: true, Preserved: true, Closed: true,
			OrquestaRestartBeforeSeal: true, OrquestaRestartAfterSeal: true,
			AgentRestartBeforeSeal: true, AgentRestartAfterSeal: true,
			NegativeCases: []string{
				"crossed_reference", "digest_mismatch", "grant_replay", "restart_after_seal",
				"restart_before_seal", "write_set_mismatch",
			},
		},
		Timeline: PhysicalGateBTimeline{
			StartedAt: "2026-09-02T10:00:00Z", LaunchAcceptedAt: "2026-09-02T10:00:01Z",
			WorkCompletedAt: "2026-09-02T10:00:02Z", PreservedAt: "2026-09-02T10:00:03Z",
			ClosedAt: "2026-09-02T10:00:04Z", RecordedAt: "2026-09-02T10:00:05Z",
		},
	}
}
