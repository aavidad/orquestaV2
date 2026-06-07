package orquestacontext

import (
	"testing"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func TestContextBundleDetailRailExternalMatrixV0(t *testing.T) {
	t.Setenv(orquestarails.DetailProhibitedRailsEnvV0, "on")
	t.Setenv(orquestarails.DetailProhibitedRailsScopeEnvV0, "context_bundle_request.*")

	request := validContextBundleRequestV0()
	request.TaskKind = "web_application"
	request.Objective = "Usar runtime provider model codex git db sql con prompt policy y transcript policy por refs opacas."
	request.ReadSet = []string{"modulos/orquesta-runtime"}
	request.WriteSet = []string{"modulos/orquesta-runtime-codex"}
	request.EvidenceRefs = []string{"evidence-ref-token-budget", "evidence-ref-prompt-policy"}

	bundle := BuildContextBundleV0(request)
	if !bundle.Valid() {
		t.Fatalf("bundle invalido por falso positivo: %+v", bundle.Issues)
	}
}

func TestContextBundleDetailRailMatrixNoBloqueaVocabularioOperativoV0(t *testing.T) {
	t.Setenv(orquestarails.DetailProhibitedRailsEnvV0, "on")
	t.Setenv(orquestarails.DetailProhibitedRailsScopeEnvV0, "context_bundle_request.*")

	request := validContextBundleRequestV0()
	request.Objective = "Usar refs opacas de runtime provider model home prompt policy."
	request.ReadSet = []string{"read-ref-runtime-provider-home"}
	request.EvidenceRefs = []string{"evidence-ref-token-budget"}

	bundle := BuildContextBundleV0(request)
	if !bundle.Valid() {
		t.Fatalf("vocabulario operativo no debe bloquear contexto: %+v", bundle.Issues)
	}
}

func TestContextBundleDetailRailMatrixBloqueaValoresSensiblesEfectivosV0(t *testing.T) {
	request := validContextBundleRequestV0()
	request.EvidenceRefs = []string{"evidence-ref-client_secret=valor"}

	bundle := BuildContextBundleV0(request)
	if bundle.Valid() {
		t.Fatalf("valor sensible efectivo no debe validarse: %+v", bundle)
	}
}

func TestContextMaterializationDetailRailExternalMatrixV0(t *testing.T) {
	t.Setenv(orquestarails.DetailProhibitedRailsEnvV0, "on")
	t.Setenv(orquestarails.DetailProhibitedRailsScopeEnvV0, "context_materialization.content")

	if contextMaterializedContentHasForbiddenDetailV0("runtime provider model prompt policy transcript policy") {
		t.Fatal("contenido operacional opaco bloqueado")
	}
	for _, value := range []string{"prompt policy ref", "transcript policy ref", "home ref opaca"} {
		if contextMaterializedContentHasForbiddenDetailV0(value) {
			t.Fatalf("vocabulario opaco no debe bloquear contenido: %q", value)
		}
	}
	for _, value := range []string{"prompt=raw text", "sk-test-value"} {
		if !contextMaterializedContentHasForbiddenDetailV0(value) {
			t.Fatalf("contenido sensible efectivo debe bloquearse: %q", value)
		}
	}
}
