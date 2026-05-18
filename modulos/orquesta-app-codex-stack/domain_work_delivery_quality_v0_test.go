package orquestaappcodexstack

import (
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
}
