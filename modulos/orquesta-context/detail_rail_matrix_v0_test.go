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

func TestContextBundleDetailRailMatrixRejectsSensitiveValuesV0(t *testing.T) {
	t.Setenv(orquestarails.DetailProhibitedRailsEnvV0, "on")
	t.Setenv(orquestarails.DetailProhibitedRailsScopeEnvV0, "context_bundle_request.*")

	request := validContextBundleRequestV0()
	request.ReadSet = []string{"/home/user/.codex/auth.json"}
	request.EvidenceRefs = []string{"evidence-ref-client_secret=valor"}

	bundle := BuildContextBundleV0(request)
	requireContextIssueV0(t, bundle.Issues, ErrContextBundleDetalleProhibidoV0)
}

func TestContextMaterializationDetailRailExternalMatrixV0(t *testing.T) {
	t.Setenv(orquestarails.DetailProhibitedRailsEnvV0, "on")
	t.Setenv(orquestarails.DetailProhibitedRailsScopeEnvV0, "context_materialization.content")

	if contextMaterializedContentHasForbiddenDetailV0("runtime provider model prompt policy transcript policy") {
		t.Fatal("contenido operacional opaco bloqueado")
	}
	for _, value := range []string{"prompt=raw text", "/home/user/private", "sk-test-value"} {
		if !contextMaterializedContentHasForbiddenDetailV0(value) {
			t.Fatalf("contenido sensible aceptado: %q", value)
		}
	}
}
