package orquestaruntimecodex

import "testing"

func TestStrictAckPendingRailGenericVocabularyBuildsSafeObservationV0(t *testing.T) {
	spec := codexSpecForTestV0()
	data := []byte(`{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}],"notes":["token budget policy","provider ref opaco","codex home redacted policy"]}`)

	ack, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0(data, spec)
	if len(issues) != 0 {
		t.Fatalf("vocabulario generico no debe bloquear ACK estricto: %+v", issues)
	}
	if !CodexAgentAckHasPendingRailV0(ack) {
		t.Fatalf("rail pendiente generico no conservado ack=%+v", ack)
	}

	observation, issues := BuildCodexDeliveryObservationV0(ack, spec)
	if len(issues) != 0 {
		t.Fatalf("observacion no debe bloquear rail pendiente generico: %+v", issues)
	}
	for _, want := range []string{
		CodexAgentAckPendingRailEvidenceRefV0,
		"ack-pending-rail:token",
		"ack-pending-rail:provider",
		"ack-pending-rail:home",
	} {
		if !evidenceContainsCodexDeliveryObservationTestV0(observation.EvidenceRefs, want) {
			t.Fatalf("evidence_refs sin %q: %v", want, observation.EvidenceRefs)
		}
	}
	if codexDeliveryObservationUnsafeForCoreV0(observation) {
		t.Fatalf("observation marca rail blando como unsafe: %+v", observation)
	}
}
