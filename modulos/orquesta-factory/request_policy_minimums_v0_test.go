package orquestafactory

import "testing"

func TestResolveRequestPolicyV0NormalPermiteCierreProductivo(t *testing.T) {
	policy := ResolveRequestPolicyV0(RequestKindCrearAppCompletaV0, ExecutionModeNormalV0)

	if policy.RequestKind != RequestKindCrearAppCompletaV0 ||
		policy.ExecutionMode != ExecutionModeNormalV0 ||
		!policy.ProductiveClosureAllowed ||
		policy.AllowsReducedScope ||
		policy.OmissionReportRequired {
		t.Fatalf("policy inesperada: %+v", policy)
	}
	if !containsStringV0(policy.MinimumDeliverables, "programacion") ||
		!containsStringV0(policy.MinimumDeliverables, "revision_final") {
		t.Fatalf("minimos app completa incompletos: %+v", policy.MinimumDeliverables)
	}
}

func TestResolveRequestPolicyV0DebugExigeReporteDeOmisiones(t *testing.T) {
	policy := ResolveRequestPolicyV0(RequestKindDocumentarAppV0, ExecutionModeDebugV0)

	if !policy.AllowsReducedScope || policy.ProductiveClosureAllowed || !policy.OmissionReportRequired {
		t.Fatalf("debug policy inesperada: %+v", policy)
	}
	if !containsStringV0(policy.MinimumDeliverables, "manual_usuario") ||
		!containsStringV0(policy.MinimumDeliverables, "manual_sistemas_deploy") {
		t.Fatalf("minimos documentacion incompletos: %+v", policy.MinimumDeliverables)
	}
}

func TestMinimumDeliverablesForRequestKindV0DevuelveCopia(t *testing.T) {
	first := MinimumDeliverablesForRequestKindV0(RequestKindSeguridadV0)
	first[0] = "mutado"
	second := MinimumDeliverablesForRequestKindV0(RequestKindSeguridadV0)

	if second[0] == "mutado" {
		t.Fatalf("minimum deliverables comparte backing array: %+v", second)
	}
}
