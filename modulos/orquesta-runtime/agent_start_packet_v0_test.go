package orquestaruntime

import (
	"encoding/json"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestBuildAgentStartPacketV0GeneraPaqueteNeutral(t *testing.T) {
	request := runtimeLaunchRequestValidaV0()
	materialized := runtimeMaterializedContextValidoV0(t, *request.ContextBundle)

	packet := BuildAgentStartPacketV0(request, materialized)
	if !packet.Valid() {
		t.Fatalf("packet invalid: %+v", packet.Issues)
	}
	if packet.WorkOrderRef != request.ContextBundle.WorkOrderRef ||
		packet.TargetModule != "orquesta-runtime" ||
		packet.CapacityLevel != request.CapacityDecision.NivelCapacidad {
		t.Fatalf("packet refs mismatch: %+v", packet)
	}
	if len(packet.Task.WriteSet) != len(request.FunctionContract.WriteSet) ||
		len(packet.Context.Entries) == 0 ||
		len(packet.Policies) == 0 {
		t.Fatalf("packet incompleto: %+v", packet)
	}
	assertAgentStartPacketNoOperationalDetailsV0(t, packet)
}

func TestAgentStartTaskV0ConservaLinajeRecursivoEnJSON(t *testing.T) {
	packet := AgentStartPacketV0{
		SchemaVersion: AgentStartPacketSchemaVersionV0,
		RequestID:     "agent-ref-child-001",
		WorkOrderRef:  "task-ref-child-001",
		Task: AgentStartTaskV0{
			TaskRef:         "task-ref-child-001",
			ParentTaskRef:   "task-ref-parent-001",
			CohortRef:       "cohort-ref-recursive-001",
			WaveRef:         "wave-ref-recursive-001",
			DelegationDepth: 2,
			MaxChildAgents:  6,
			ChildTaskRefs:   []string{"task-ref-grandchild-001"},
		},
	}

	data, err := json.Marshal(packet)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got AgentStartPacketV0
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Task.ParentTaskRef != "task-ref-parent-001" ||
		got.Task.CohortRef != "cohort-ref-recursive-001" ||
		got.Task.WaveRef != "wave-ref-recursive-001" ||
		got.Task.DelegationDepth != 2 ||
		got.Task.MaxChildAgents != 6 ||
		len(got.Task.ChildTaskRefs) != 1 ||
		got.Task.ChildTaskRefs[0] != "task-ref-grandchild-001" {
		t.Fatalf("linaje perdido en packet: %+v", got.Task)
	}
	assertAgentStartPacketNoOperationalDetailsV0(t, got)
}

func TestBuildAgentStartPacketV0RechazaContextoMaterializadoAusente(t *testing.T) {
	packet := BuildAgentStartPacketV0(runtimeLaunchRequestValidaV0(), orquestacontext.ContextMaterializedBundleV0{})

	requireRuntimeLaunchCodeV0(t, packet.Issues, AgentStartPacketContextMaterializadoRequeridoV0)
}

func TestBuildAgentStartPacketV0RechazaContextoDeOtroBundle(t *testing.T) {
	request := runtimeLaunchRequestValidaV0()
	materialized := runtimeMaterializedContextValidoV0(t, *request.ContextBundle)
	materialized.BundleRef = "context-bundle-otro"

	packet := BuildAgentStartPacketV0(request, materialized)

	requireRuntimeLaunchCodeV0(t, packet.Issues, AgentStartPacketContextMaterializadoInvalidoV0)
}

func TestBuildAgentStartPacketV0RechazaSecretoEnContextoMaterializado(t *testing.T) {
	request := runtimeLaunchRequestValidaV0()
	materialized := runtimeMaterializedContextValidoV0(t, *request.ContextBundle)
	materialized.Entries[0].Content = "sk-test-token"

	packet := BuildAgentStartPacketV0(request, materialized)

	requireRuntimeLaunchCodeV0(t, packet.Issues, AgentStartPacketInvalidoV0)
}

func TestBuildAgentStartPacketV0DeclaraEvidenciaDeSaneamiento(t *testing.T) {
	request := runtimeLaunchRequestValidaV0()
	materialized := runtimeMaterializedContextValidoV0(t, *request.ContextBundle)
	materialized.SanitizationEvidence = []orquestacontext.ContextSanitizationEvidenceV0{{
		SchemaVersion:  orquestacontext.ContextSanitizationEvidenceSchemaVersionV0,
		EvidenceRef:    "context-sanitization-evidence-entry-001",
		BundleRef:      materialized.BundleRef,
		WorkOrderRef:   materialized.WorkOrderRef,
		TargetModule:   materialized.TargetModule,
		EntryRef:       "context-entry-001",
		SourceRef:      "orquesta-common-rules:v0",
		SanitizerRef:   "sanitizer-ref-local-001",
		Status:         orquestacontext.ContextSanitizationStatusReviewRequiredV0,
		ReviewRequired: true,
	}}

	packet := BuildAgentStartPacketV0(request, materialized)
	if !packet.Valid() {
		t.Fatalf("packet invalid: %+v", packet.Issues)
	}
	if !stringInSetRuntimeTestV0(packet.Policies, "context_sanitization_evidence_present") ||
		!stringInSetRuntimeTestV0(packet.Policies, "ask_director_on_sanitization_review") {
		t.Fatalf("policies=%+v", packet.Policies)
	}
	assertAgentStartPacketNoOperationalDetailsV0(t, packet)
}

func runtimeMaterializedContextValidoV0(
	t *testing.T,
	bundle orquestacontext.ContextBundleV0,
) orquestacontext.ContextMaterializedBundleV0 {
	t.Helper()
	materialized := orquestacontext.ContextMaterializedBundleV0{
		SchemaVersion:        orquestacontext.ContextMaterializedBundleSchemaVersionV0,
		BundleRef:            bundle.BundleRef,
		WorkOrderRef:         bundle.WorkOrderRef,
		TargetModule:         bundle.TargetModule,
		TotalBytes:           64,
		DirectorQuestionHint: bundle.Summary.DirectorQuestionPolicy,
		Entries: []orquestacontext.ContextMaterializedEntryV0{
			{
				EntryRef:  "context-entry-001",
				Layer:     orquestacontext.ContextLayerCommonRulesV0,
				Kind:      orquestacontext.ContextEntryRuleRefV0,
				SourceRef: "orquesta-common-rules:v0",
				Mode:      orquestacontext.ContextMaterializationModeContentV0,
				Content:   "Contexto pequeno. CONSULTA AL DIRECTOR si falta informacion.",
				Bytes:     64,
				Required:  true,
			},
		},
	}
	if !materialized.Valid() {
		t.Fatalf("bad materialized fixture: %+v", materialized)
	}
	return materialized
}

func assertAgentStartPacketNoOperationalDetailsV0(t *testing.T, packet AgentStartPacketV0) {
	t.Helper()
	data, err := json.Marshal(packet)
	if err != nil {
		t.Fatalf("marshal packet: %v", err)
	}
	lower := strings.ToLower(string(data))
	for _, fragment := range []string{
		"provider-ref", "home-ref", "credential-ref", "oauth_ref",
		"model-ref", "bearer ", "access_token", "refresh_token",
	} {
		if strings.Contains(lower, fragment) {
			t.Fatalf("packet leaks %q: %s", fragment, string(data))
		}
	}
	if agentStartPacketHasSecretPrefixV0(lower) {
		t.Fatalf("packet leaks secret prefix: %s", string(data))
	}
}

func stringInSetRuntimeTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
