package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestExternalWorkContextPolicyV0RespetaPerfilExplicito(t *testing.T) {
	compact := externalWorkContextPolicyForWorkV0(
		&orquestaappchange.AppChangeExternalWorkV0{
			WorkKind: "draft_content_block",
			InputFields: []orquestadomainwork.DomainWorkFieldV0{{
				Name:  "CONTEXT_PROFILE",
				Value: "compact",
			}},
		},
	)
	if compact.FieldMaxBytes != externalWorkContextCompactFieldBytesV0 ||
		compact.TotalMaxBytes != externalWorkContextCompactTotalBytesV0 {
		t.Fatalf("compact=%+v", compact)
	}

	large := externalWorkContextPolicyForWorkV0(
		&orquestaappchange.AppChangeExternalWorkV0{
			WorkKind: "summarize_topic",
			InputFields: []orquestadomainwork.DomainWorkFieldV0{{
				Name:  "editorial_granularity",
				Value: "chapter",
			}},
		},
	)
	if large.FieldMaxBytes != externalWorkContextLargeFieldBytesV0 ||
		large.TotalMaxBytes != externalWorkContextLargeTotalBytesV0 {
		t.Fatalf("large=%+v", large)
	}
}

func TestCodexLaunchSpecResolverV0PermiteContextoAmplioEnTrabajoDocumentalLargo(t *testing.T) {
	changeRef := "opes-job-job-ref-longform-001"
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	runRef := "run-ref-opes-context-longform-001"
	longContext := strings.Repeat("contenido editorial ", 180)
	task := externalContextPolicyTaskForTestV0(runRef, taskRef)
	resolver := codexLaunchSpecResolverForExternalContextPolicyTestV0(
		t,
		task,
		orquestaappchange.AppChangeRecordV0{
			Request: orquestaappchange.AppChangeRequestV0{
				RunRef:     runRef,
				AppRef:     "opes",
				ChangeRef:  changeRef,
				UserIntent: "Crear capitulo de temario.",
				ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
					ProjectRef: "opes",
					JobRef:     "job-ref-longform-001",
					WorkKind:   "draft_content_block",
					InputFields: []orquestadomainwork.DomainWorkFieldV0{{
						Name:  "syllabus_full",
						Value: longContext,
					}},
				},
			},
		},
	)

	resolution, err := resolver.ResolveExternalAgentLaunchSpecV0(
		context.Background(),
		agentLaunchInboundForExternalContextPolicyTestV0(runRef, taskRef, "agent-ref-opes-context-longform-001"),
	)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}

	packet := resolution.Spec.AgentPacket
	if contextBundleHasRequiredTruncatedEntryV0(packet.Context) {
		t.Fatalf("contexto largo razonable no debe truncarse: %+v", packet.Context.Entries)
	}
	if !codexStackContextContainsForTestV0(packet.Context.Entries, strings.Repeat("contenido editorial ", 80)) {
		t.Fatalf("contexto amplio no materializado: %+v", packet.Context.Entries)
	}
}

func TestCodexLaunchSpecResolverV0MarcaPresupuestoTotalAgotado(t *testing.T) {
	changeRef := "opes-job-job-ref-budget-001"
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	runRef := "run-ref-opes-context-budget-001"
	task := externalContextPolicyTaskForTestV0(runRef, taskRef)
	resolver := codexLaunchSpecResolverForExternalContextPolicyTestV0(
		t,
		task,
		orquestaappchange.AppChangeRecordV0{
			Request: orquestaappchange.AppChangeRequestV0{
				RunRef:     runRef,
				AppRef:     "opes",
				ChangeRef:  changeRef,
				UserIntent: "Crear bloque largo.",
				ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
					ProjectRef:  "opes",
					JobRef:      "job-ref-budget-001",
					WorkKind:    "draft_content_block",
					InputFields: externalContextBudgetFieldsForTestV0(),
				},
			},
		},
	)

	resolution, err := resolver.ResolveExternalAgentLaunchSpecV0(
		context.Background(),
		agentLaunchInboundForExternalContextPolicyTestV0(runRef, taskRef, "agent-ref-opes-context-budget-001"),
	)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}

	packet := resolution.Spec.AgentPacket
	if !contextBundleHasRequiredTruncatedEntryV0(packet.Context) ||
		!codexStackContextContainsForTestV0(packet.Context.Entries, "external_context_budget") ||
		!stringInSetV0(packet.Task.RequiredTests, externalContextTruncatedRequiredTestV0) {
		t.Fatalf("packet=%+v", packet)
	}
}

func TestExternalWorkFieldContentV0RedactaValoresSensiblesAntesDelPacket(t *testing.T) {
	content := externalWorkFieldContentV0(orquestadomainwork.DomainWorkFieldV0{
		Name:      "api_key",
		Value:     "live-secret-value",
		Values:    []string{"otro-secreto"},
		ValueJSON: []byte(`{"api_key":"json-secret-value"}`),
	})

	if content == "" ||
		strings.Contains(content, "live-secret-value") ||
		strings.Contains(content, "otro-secreto") ||
		strings.Contains(content, "json-secret-value") ||
		!strings.Contains(content, "redacted-sensitive-field") {
		t.Fatalf("contexto sensible no redactado: %s", content)
	}

	content = externalWorkFieldContentV0(orquestadomainwork.DomainWorkFieldV0{
		Name:      "runtime_ref",
		Value:     "usar ref opaca adapter-runtime-001",
		Values:    []string{"authorization: Bearer live-token-001"},
		ValueJSON: []byte(`{"client_secret":"json-secret-value"}`),
	})
	if !strings.Contains(content, "adapter-runtime-001") ||
		strings.Contains(content, "live-token-001") ||
		strings.Contains(content, "json-secret-value") {
		t.Fatalf("contexto opaco/sensible mal tratado: %s", content)
	}
}

func externalContextPolicyTaskForTestV0(
	runRef string,
	taskRef string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        taskRef,
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Redactar unidad documental OPES",
		Summary:       "Redactar unidad editorial amplia.",
		WriteSet:      []string{"external/opes/draft_content_block"},
		AcceptanceCriteria: []string{
			"usar paquete de dominio suficiente",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  "contract:function:app-change:opes:v0",
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
}

func externalContextBudgetFieldsForTestV0() []orquestadomainwork.DomainWorkFieldV0 {
	fields := make([]orquestadomainwork.DomainWorkFieldV0, 0, 8)
	for i := 0; i < 8; i++ {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:  fmt.Sprintf("long_field_%02d", i+1),
			Value: strings.Repeat(fmt.Sprintf("contenido_%02d ", i+1), externalWorkContextLargeFieldBytesV0),
		})
	}
	return fields
}

func codexLaunchSpecResolverForExternalContextPolicyTestV0(
	t *testing.T,
	task orquestacoreworkflow.WorkflowTaskV0,
	record orquestaappchange.AppChangeRecordV0,
) CodexLaunchSpecResolverV0 {
	t.Helper()
	return CodexLaunchSpecResolverV0{
		Config: CodexRuntimeConfigV0{
			RuntimeWorkDir: t.TempDir(),
			ProjectWorkDir: t.TempDir(),
		},
		TaskStore:      orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
		AppChangeStore: orquestaappchange.NewInMemoryAppChangeStoreV0(record),
	}
}

func agentLaunchInboundForExternalContextPolicyTestV0(
	runRef string,
	taskRef string,
	agentRef string,
) orquestaruntime.AgentLauncherInboundV0 {
	return orquestaruntime.AgentLauncherInboundV0{
		CorrelationID: "corr-" + agentRef,
		Payload: &orquestaruntime.LaunchRuntimeAgentRequestV0{
			RunID:              runRef,
			AgentRequestID:     agentRef,
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Role:               "implementacion",
			TaskRef:            taskRef,
			CapacityRequestRef: "capacity-ref-" + agentRef,
			Summary:            "external_work",
		},
	}
}
