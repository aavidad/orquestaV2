package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestReviewReworkReplanSourceV0MantieneRailsGenericosComoEvidencia(t *testing.T) {
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target"),
		),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.EvidenceRefs = []string{
		"evidence-ref-token-budget-policy",
		"evidence-ref-provider-policy",
		"evidence-ref-home-redacted-policy",
		"evidence-ref-secret-policy",
		"evidence-ref-prompt-policy",
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%+v", plans)
	}
	for _, want := range request.EvidenceRefs {
		if !reviewReworkPlanHasEvidenceForTestV0(plans[0].EvidenceRefs, want) {
			t.Fatalf("evidence_refs perdio rail blando %q: %v", want, plans[0].EvidenceRefs)
		}
	}
}

func TestReviewReworkReplanSourceV0FiltraDetalleSensibleEfectivo(t *testing.T) {
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target"),
		),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.EvidenceRefs = []string{
		"evidence-ref-neutral-review-rework",
		"access_token=abc123",
		"HOME=/home/example/.codex",
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%+v", plans)
	}
	evidence := plans[0].EvidenceRefs
	if reviewReworkPlanHasEvidenceForTestV0(evidence, "access_token=abc123") ||
		reviewReworkPlanHasEvidenceForTestV0(evidence, "HOME=/home/example/.codex") {
		t.Fatalf("evidence_refs conserva detalle sensible efectivo: %v", evidence)
	}
	if !reviewReworkPlanHasEvidenceForTestV0(evidence, "evidence-ref-neutral-review-rework") {
		t.Fatalf("evidence_refs perdio evidencia neutral: %v", evidence)
	}
}

func TestReviewReworkReplanSourceV0FiltraDetalleSensibleConRailsDetalleOff(t *testing.T) {
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "off")
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target"),
		),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.EvidenceRefs = []string{
		"evidence-ref-token-policy",
		"access_token=abc123",
		"HOME=/home/example/.codex",
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%+v", plans)
	}
	evidence := plans[0].EvidenceRefs
	if reviewReworkPlanHasEvidenceForTestV0(evidence, "access_token=abc123") ||
		reviewReworkPlanHasEvidenceForTestV0(evidence, "HOME=/home/example/.codex") {
		t.Fatalf("evidence_refs conserva detalle sensible efectivo: %v", evidence)
	}
	if !reviewReworkPlanHasEvidenceForTestV0(evidence, "evidence-ref-token-policy") {
		t.Fatalf("evidence_refs perdio rail blando: %v", evidence)
	}
}

func TestReviewReworkReplanSourceV0NoReplanificaSoloRailBlandoV0(t *testing.T) {
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target"),
		),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.EvidenceRefs = []string{
		"gate-followup-required",
		"gate-action:request_followup_review",
		"gate-issue:file_too_large",
		"gate-issue:write_set_target_missing:web",
		"ack-pending-rail:provider",
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("rail blando no debe relanzar rework: %+v", plans)
	}
}

func TestReviewReworkReplanSourceV0RailBlandoNoOcultaBloqueoV0(t *testing.T) {
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target"),
		),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.EvidenceRefs = []string{
		"gate-issue:file_too_large",
		"gate-issue:required_test_missing",
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("bloqueo real debe conservar rework: %+v", plans)
	}
}
