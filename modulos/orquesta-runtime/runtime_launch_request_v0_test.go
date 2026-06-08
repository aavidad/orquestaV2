package orquestaruntime

import (
	"encoding/json"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestValidateRuntimeLaunchRequestV0AceptaRequestMinima(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()

	issues := ValidateRuntimeLaunchRequestV0(req)
	if len(issues) != 0 {
		t.Fatalf("errores inesperados: %#v", issues)
	}
}

func TestRuntimeLaunchRequestV0JSONRoundTripMantieneContrato(t *testing.T) {
	raw, err := json.Marshal(runtimeLaunchRequestValidaV0())
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var req RuntimeLaunchRequestV0
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	if issues := ValidateRuntimeLaunchRequestV0(req); len(issues) != 0 {
		t.Fatalf("errores inesperados tras JSON round-trip: %#v", issues)
	}
	if req.SchemaVersion != RuntimeLaunchRequestSchemaVersionV0 {
		t.Fatalf("schema_version = %q", req.SchemaVersion)
	}
}

func TestValidateRuntimeLaunchRequestV0RechazaFunctionContractAusente(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.FunctionContract = nil

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), FunctionContractRequeridoV0)
}

func TestValidateRuntimeLaunchRequestV0RechazaFunctionContractNoActiva(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.FunctionContract.State = "borrador"

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), FunctionContractNoActivaV0)
}

func TestValidateRuntimeLaunchRequestV0RechazaWriteSetVacio(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.FunctionContract.WriteSet = nil

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), WriteSetRequeridoV0)
}

func TestValidateRuntimeLaunchRequestV0RechazaWriteSetInvalido(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.FunctionContract.WriteSet = []string{"modulos/orquesta-runtime/runtime_launch_request_v0.go", "../fuera.go"}

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), WriteSetInvalidoV0)
}

func TestValidateRuntimeLaunchRequestV0RechazaCapacityDecisionAusente(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.CapacityDecision = nil

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), CapacityDecisionRequeridaV0)
}

func TestValidateRuntimeLaunchRequestV0RechazaContextBundleAusente(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.ContextBundle = nil

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), ContextBundleRequeridoV0)
}

func TestValidateRuntimeLaunchRequestV0RechazaContextBundleDeOtroModulo(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.ContextBundle.TargetModule = "orquesta-web"

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), ContextBundleInvalidoV0)
}

func TestValidateRuntimeLaunchRequestV0RechazaHomeRefConFormaDeRuta(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.RuntimeBinding.HomeRef = "~runtime-home"

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), RutaHomeRealDetectadaV0)
}

func TestValidateRuntimeLaunchRequestV0RechazaReferenciasNoOpacas(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.RuntimeBinding.ProviderRef = "provider/ref"

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), ReferenciaNoOpacaV0)
}

func TestValidateRuntimeLaunchRequestV0ValidaSkillRefsOpacas(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.Task.SkillRefs = []string{"skill-ref-orquesta-programacion-v0"}

	if issues := ValidateRuntimeLaunchRequestV0(req); len(issues) != 0 {
		t.Fatalf("skill_refs opacas rechazadas: %#v", issues)
	}

	req.Task.SkillRefs = []string{"../skills/orquesta-programacion"}
	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), ReferenciaNoOpacaV0)
}

func TestValidateRuntimeLaunchRequestV0RechazaSecretosEnRefs(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.RuntimeBinding.CredentialRef = "sk-test-token"

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), SecretoDetectadoV0)
}

func TestValidateRuntimeLaunchRequestV0RechazaSafetyAbierta(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.Safety.WriteSetPolicy = "open"

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), RuntimeLaunchRequestInvalidaV0)
}

func TestValidateRuntimeLaunchRequestV0RechazaSafetyAbiertaAunqueRailsDetalleOff(t *testing.T) {
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "off")
	req := runtimeLaunchRequestValidaV0()
	req.Safety.WriteSetPolicy = "open"
	req.Safety.SecretsPolicy = "open"
	req.Safety.HomePathsPolicy = "open"
	req.Safety.ProviderPolicy = "open"

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), RuntimeLaunchRequestInvalidaV0)
}

func TestValidateRuntimeLaunchRequestV0RechazaModeloDistintoACapacity(t *testing.T) {
	req := runtimeLaunchRequestValidaV0()
	req.RuntimeBinding.ModelRef = "model-ref-distinto"

	requireRuntimeLaunchCodeV0(t, ValidateRuntimeLaunchRequestV0(req), CapacityDecisionNoSoportadaV0)
}

func requireRuntimeLaunchCodeV0(t *testing.T, issues []RuntimeLaunchErrorV0, code RuntimeLaunchErrorCodeV0) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("no se encontro codigo %q en %#v", code, issues)
}

func runtimeLaunchRequestValidaV0() RuntimeLaunchRequestV0 {
	return RuntimeLaunchRequestV0{
		SchemaVersion:  RuntimeLaunchRequestSchemaVersionV0,
		RequestID:      "req-runtime-001-minima",
		CorrelationID:  "corr-runtime-001",
		IdempotencyKey: "idem-runtime-001",
		RequestedAt:    "2026-05-04T10:00:00Z",
		Source: &RuntimeLaunchSourceV0{
			Module:     RuntimeLaunchSourceModuleCoreV0,
			AdapterRef: "core-launch-usecase-v0",
		},
		Locale:     "es-ES",
		LaunchMode: RuntimeLaunchModeNewSessionV0,
		Task: &RuntimeLaunchTaskV0{
			TaskRef:    "task-ref-001",
			ProjectRef: "project-ref-001",
			PhaseRef:   "phase-ref-implementacion",
			Priority:   "normal",
		},
		FunctionContract: &RuntimeFunctionContractV0{
			SourceContract:  FunctionContractSourceV0,
			ContractRef:     "function-contract-ref-001",
			ContractVersion: FunctionContractVersionV0,
			State:           FunctionContractStateActiveV0,
			Titulo:          "Definir validador local RuntimeLaunchRequest v0",
			Objetivo:        "Validar el contrato sin arrancar runtime real.",
			ArchivoObjetivo: "modulos/orquesta-runtime/runtime_launch_request_v0.go",
			SimboloObjetivo: "RuntimeLaunchRequestV0",
			WriteSet: []string{
				"modulos/orquesta-runtime/runtime_launch_request_v0.go",
				"modulos/orquesta-runtime/runtime_launch_request_v0_test.go",
			},
			TestsObligatorios: []string{
				"go test -count=1 ./modulos/orquesta-runtime",
			},
			CriterioCierre: []string{
				"El validador acepta un request minimo valido.",
				"El validador rechaza los invariantes publicos principales.",
			},
		},
		CapacityDecision: &RuntimeCapacityDecisionV0{
			DecisionRef:     "capacity-decision-ref-001",
			ContractVersion: CapacityDecisionVersionV0,
			NivelCapacidad:  "medium",
			ReasoningEffort: "medium",
			PoolRef:         "pool-ref-remoto-a",
			ModelRef:        "model-ref-medium-a",
			QuotaRef:        "quota-ref-001",
		},
		RuntimeBinding: &RuntimeBindingV0{
			LogicalAgentRef: "logical-agent-ref-001",
			RuntimeKind:     "cli",
			ConnectorRef:    "connector-ref-cli-a",
			ProviderRef:     "provider-ref-remoto-a",
			ModelRef:        "model-ref-medium-a",
			HomeRef:         "home-ref-runtime-a",
			CredentialKind:  "oauth_ref",
			CredentialRef:   "credential-ref-oauth-a",
		},
		EvidenceRefs: &RuntimeEvidenceRefsV0{
			MailboxRef:    "mailbox-ref-001",
			AckRef:        "ack-ref-001",
			ReadinessRef:  "readiness-ref-001",
			CheckpointRef: "checkpoint-ref-001",
		},
		ContextBundle: runtimeContextBundleValidoV0(),
		Delivery: &RuntimeDeliveryV0{
			MailboxProtocol:         DeliveryMailboxProtocolV0,
			AckRequired:             true,
			ReadinessTimeoutSeconds: 120,
			MaxStartupSeconds:       180,
		},
		Safety: &RuntimeSafetyV0{
			SecretsPolicy:   SafetyPolicyReferencesOnlyV0,
			HomePathsPolicy: SafetyPolicyOpaqueRefsOnlyV0,
			ProviderPolicy:  SafetyPolicyOpaqueRefsOnlyV0,
			WriteSetPolicy:  SafetyPolicyWriteSetClosedV0,
		},
	}
}

func runtimeContextBundleValidoV0() *orquestacontext.ContextBundleV0 {
	bundle := orquestacontext.BuildContextBundleV0(orquestacontext.ContextBundleRequestV0{
		SchemaVersion: orquestacontext.ContextBundleRequestSchemaVersionV0,
		BundleRef:     "context-bundle-runtime-001",
		WorkOrderRef:  "work-order-runtime-001",
		TargetModule:  "orquesta-runtime",
		Phase:         "programacion",
		TaskKind:      "microtarea_codigo",
		Objective:     "Validar contrato de lanzamiento con contexto pequeno.",
		CapacityLevel: "medium",
		ReadSet:       []string{"runtime_launch_request_types_v0.go"},
		WriteSet:      []string{"runtime_launch_request_v0.go", "runtime_launch_request_v0_test.go"},
		ContractRefs:  []string{"RuntimeLaunchRequestV0", "ContextBundleV0"},
	})
	return &bundle
}
