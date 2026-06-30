package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestExternalWorkRunRuntimeCompatibilityGuardPermiteSinOptInV0(t *testing.T) {
	next := &fakeExternalWorkRunGuardNextV0{
		result: orquestamcp.MCPExternalWorkRunToolResultV0{Estado: orquestamcp.MCPExternalWorkRunEstadoOKV0},
	}
	executor := NewExternalWorkRunRuntimeCompatibilityGuardExecutorV0(next, ExternalWorkRunRuntimeCompatibilityGuardConfigV0{
		RuntimeIdentity: ExternalWorkRunRuntimeIdentityV0{BinarySHA256: "sha-runtime-vivo"},
	})

	result, err := executor.Execute(context.Background(), orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID: "req-no-optin",
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			AppRef: "opes",
		},
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 || next.calls != 1 {
		t.Fatalf("result=%+v calls=%d", result, next.calls)
	}
}

func TestExternalWorkRunRuntimeCompatibilityGuardBloqueaSHADistintoV0(t *testing.T) {
	next := &fakeExternalWorkRunGuardNextV0{}
	executor := NewExternalWorkRunRuntimeCompatibilityGuardExecutorV0(next, ExternalWorkRunRuntimeCompatibilityGuardConfigV0{
		RuntimeIdentity: ExternalWorkRunRuntimeIdentityV0{
			BinarySHA256: "sha-runtime-obsoleto",
			BuildRef:     "build-local",
			CommitRef:    "commit-local",
		},
	})

	result, err := executor.Execute(context.Background(), orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID: "req-sha",
		AppChangeRequest: appChangeWithRuntimeCompatibilityFieldsV0([]orquestadomainwork.DomainWorkFieldV0{{
			Name:  ExternalWorkRunRuntimeCompatibilityFieldBinarySHA256V0,
			Value: "sha-runtime-aprobado",
		}}),
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != ExternalWorkRunRuntimeCompatibilityMismatchV0 ||
		result.Errores[0].Field != ExternalWorkRunRuntimeCompatibilityFieldBinarySHA256V0+":action_use_approved_orquesta_runtime" ||
		next.calls != 0 {
		t.Fatalf("result=%+v calls=%d", result, next.calls)
	}
	if len(result.NextActions) == 0 {
		t.Fatalf("falta next action accionable: %+v", result)
	}
}

func TestExternalWorkRunRuntimeCompatibilityGuardPermiteIdentidadCoincidenteV0(t *testing.T) {
	next := &fakeExternalWorkRunGuardNextV0{
		result: orquestamcp.MCPExternalWorkRunToolResultV0{Estado: orquestamcp.MCPExternalWorkRunEstadoOKV0},
	}
	executor := NewExternalWorkRunRuntimeCompatibilityGuardExecutorV0(next, ExternalWorkRunRuntimeCompatibilityGuardConfigV0{
		RuntimeIdentity: ExternalWorkRunRuntimeIdentityV0{
			BinarySHA256: "sha-runtime-aprobado",
			BuildRef:     "build-aprobado",
			CommitRef:    "commit-aprobado",
		},
	})

	result, err := executor.Execute(context.Background(), orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID: "req-ok",
		AppChangeRequest: appChangeWithRuntimeCompatibilityFieldsV0([]orquestadomainwork.DomainWorkFieldV0{
			{Name: ExternalWorkRunRuntimeCompatibilityFieldRequiredV0, Value: "true"},
			{Name: ExternalWorkRunRuntimeCompatibilityFieldBinarySHA256V0, Value: "sha-runtime-aprobado"},
			{Name: ExternalWorkRunRuntimeCompatibilityFieldBuildRefV0, Value: "build-aprobado"},
			{Name: ExternalWorkRunRuntimeCompatibilityFieldCommitRefV0, Value: "commit-aprobado"},
		}),
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 || next.calls != 1 {
		t.Fatalf("result=%+v calls=%d", result, next.calls)
	}
}

func TestExternalWorkRunRuntimeCompatibilityGuardRequiereIdentidadEsperadaSiOptInEstrictoV0(t *testing.T) {
	next := &fakeExternalWorkRunGuardNextV0{}
	executor := NewExternalWorkRunRuntimeCompatibilityGuardExecutorV0(next, ExternalWorkRunRuntimeCompatibilityGuardConfigV0{
		RuntimeIdentity: ExternalWorkRunRuntimeIdentityV0{BinarySHA256: "sha-runtime-aprobado"},
	})

	result, err := executor.Execute(context.Background(), orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID: "req-required",
		AppChangeRequest: appChangeWithRuntimeCompatibilityFieldsV0([]orquestadomainwork.DomainWorkFieldV0{{
			Name:  ExternalWorkRunRuntimeCompatibilityFieldRequiredV0,
			Value: "true",
		}}),
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != ExternalWorkRunRuntimeCompatibilityRequiredMissingV0 ||
		next.calls != 0 {
		t.Fatalf("result=%+v calls=%d", result, next.calls)
	}
}

func appChangeWithRuntimeCompatibilityFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) orquestaappchange.AppChangeRequestV0 {
	return orquestaappchange.AppChangeRequestV0{
		AppRef: "opes",
		ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
			ProjectRef:  "opes",
			WorkKind:    "generate_audio_asset",
			InputFields: fields,
		},
	}
}
