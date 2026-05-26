package orquestaappcodexstack

import (
	"errors"
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestValidateDomainWorkDeliveryQualityV0IgnoraMetadataNoPlaceholders(t *testing.T) {
	payload := `{
		"quality_trace":{"no_placeholders":true},
		"chapters":[{"blocks":[{"markdown":"uno dos tres cuatro"}]}],
		"tema_mediano":{"markdown":"texto visible"},
		"resumen":{"markdown":"texto visible"},
		"esquema_repaso":["punto limpio"],
		"plan_visuales":[{"markdown":"visual limpio"}]
	}`

	err := validateDomainWorkDeliveryQualityV0(
		[]orquestadomainwork.DomainWorkFieldV0{{Name: "target_words_min", Value: "4"}},
		"topic_expansion_package",
		`{"artifact_type":"topic_expansion_package","payload_json":`+payload+`}`,
		payload,
	)

	if err != nil {
		t.Fatalf("quality gate rechazo metadata valida: %v", err)
	}
}

func TestValidateDomainWorkDeliveryQualityV0RechazaPlaceholderVisible(t *testing.T) {
	payload := `{
		"chapters":[{"blocks":[{"markdown":"uno dos tres cuatro PLACEHOLDER"}]}]
	}`

	err := validateDomainWorkDeliveryQualityV0(
		[]orquestadomainwork.DomainWorkFieldV0{{Name: "target_words_min", Value: "4"}},
		"topic_expansion_package",
		payload,
		payload,
	)

	if err == nil || !strings.Contains(err.Error(), "domain_work_artifact_quality_gate_failed") {
		t.Fatalf("esperaba rechazo por placeholder visible, err=%v", err)
	}
	var issue domainWorkQualityIssueErrorV0
	if !errors.As(err, &issue) ||
		issue.Issue.Code != "placeholder_visible" ||
		issue.Issue.Field != "payload.visible_text" {
		t.Fatalf("esperaba issue estructurado de placeholder, issue=%+v err=%v", issue, err)
	}
}

func TestValidateDomainWorkDeliveryQualityV0RechazaPendienteConAcento(t *testing.T) {
	payload := `{
		"chapters":[{"blocks":[{"markdown":"uno dos tres cuatro Pendiente de ampliación"}]}]
	}`

	err := validateDomainWorkDeliveryQualityV0(
		[]orquestadomainwork.DomainWorkFieldV0{{Name: "target_words_min", Value: "4"}},
		"topic_expansion_package",
		payload,
		payload,
	)

	if err == nil || !strings.Contains(err.Error(), "domain_work_artifact_quality_gate_failed") {
		t.Fatalf("esperaba rechazo por pendiente visible, err=%v", err)
	}
}

func TestValidateDomainWorkDeliveryQualityV0ReportaIssueEstructuradoDePalabras(t *testing.T) {
	payload := `{
		"chapters":[{"blocks":[{"markdown":"uno dos tres cuatro"}]}]
	}`

	err := validateDomainWorkDeliveryQualityV0(
		[]orquestadomainwork.DomainWorkFieldV0{{Name: "target_words_min", Value: "8"}},
		"topic_expansion_package",
		payload,
		payload,
	)

	var issue domainWorkQualityIssueErrorV0
	if !errors.As(err, &issue) {
		t.Fatalf("esperaba issue estructurado, err=%v", err)
	}
	if issue.Issue.Code != "min_words_not_met" ||
		issue.Issue.Field != "payload.chapters.blocks" ||
		issue.MinWords != 8 ||
		issue.ActualWords != 4 {
		t.Fatalf("issue inesperado: %+v", issue)
	}
	if !strings.Contains(err.Error(), "code=min_words_not_met") {
		t.Fatalf("error sin codigo publico: %v", err)
	}
}

func TestValidateDomainWorkDeliveryQualityV0AceptaDocumentPlanValido(t *testing.T) {
	payload := validDocumentPlanPayloadForTestV0()

	err := validateDomainWorkDeliveryQualityV0(
		nil,
		orquestadomainwork.DomainDocumentPlanArtifactTypeV0,
		payload,
		payload,
	)

	if err != nil {
		t.Fatalf("quality gate rechazo document_plan valido: %v", err)
	}
}

func TestValidateDomainWorkDeliveryQualityV0RechazaDocumentPlanIncompleto(t *testing.T) {
	payload := `{
		"schema_version":"domain_document_plan.v0",
		"plan_ref":"plan_ref_080",
		"domain_ref":"opes",
		"work_kind":"plan_tema",
		"document_kind":"tema_oposicion",
		"language_code":"es",
		"title":"Tema",
		"objective":"Planificar tema.",
		"sections":[]
	}`

	err := validateDomainWorkDeliveryQualityV0(
		nil,
		orquestadomainwork.DomainDocumentPlanArtifactTypeV0,
		payload,
		payload,
	)

	if err == nil || !strings.Contains(err.Error(), "domain_work_artifact_quality_gate_failed") {
		t.Fatalf("esperaba rechazo por document_plan incompleto, err=%v", err)
	}
	var issue domainWorkQualityIssueErrorV0
	if !errors.As(err, &issue) ||
		issue.Issue.Code == "" ||
		issue.Issue.Field == "" {
		t.Fatalf("esperaba issue estructurado de document_plan, issue=%+v err=%v", issue, err)
	}
}
