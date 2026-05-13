package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexLaunchSpecResolverV0MaterializaContextoDominioExterno(t *testing.T) {
	changeRef := "opes-job-job-ref-001"
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	runRef := "run-ref-opes-context-001"
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        taskRef,
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Redactar bloque documental OPES",
		Summary:       "Redactar bloque con paquete de dominio suficiente.",
		WriteSet:      []string{"external/opes/draft_content_block"},
		AcceptanceCriteria: []string{
			"validar paquete de dominio",
		},
	}
	resolver := CodexLaunchSpecResolverV0{
		Config: CodexRuntimeConfigV0{
			RuntimeWorkDir: t.TempDir(),
			ProjectWorkDir: t.TempDir(),
		},
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
		AppChangeStore: orquestaappchange.NewInMemoryAppChangeStoreV0(
			orquestaappchange.AppChangeRecordV0{
				Request: orquestaappchange.AppChangeRequestV0{
					RunRef:     runRef,
					AppRef:     "opes",
					ChangeRef:  changeRef,
					UserIntent: "Crear bloque de temario.",
					ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
						ProjectRef: "opes",
						JobRef:     "job-ref-001",
						WorkKind:   "draft_content_block",
						InputFields: []orquestadomainwork.DomainWorkFieldV0{
							{Name: "topic_id", Value: "topic-ref-real-001"},
							{Name: "syllabus_full", Value: "Tema completo y esquema aprobado."},
							{Name: "block_position", ValueJSON: []byte(`{"chapter_order":1,"block_order":2,"total_blocks":38}`)},
							{Name: "source_refs", Values: []string{"BOE-A-001", "BOE-A-001"}},
						},
					},
				},
			},
		),
	}

	spec, err := resolver.ResolveExternalAgentLaunchSpecV0(
		context.Background(),
		orquestaruntime.AgentLauncherInboundV0{
			CorrelationID: "corr-opes-context-001",
			Payload: &orquestaruntime.LaunchRuntimeAgentRequestV0{
				RunID:              runRef,
				AgentRequestID:     "agent-ref-opes-context-001",
				PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				Role:               "implementacion",
				TaskRef:            taskRef,
				CapacityRequestRef: "capacity-ref-opes-context",
				Summary:            "external_work",
			},
		},
	)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}

	context := spec.Spec.AgentPacket.Context
	if len(context.Entries) < 4 {
		t.Fatalf("context entries=%+v", context.Entries)
	}
	for _, want := range []string{"topic_id", "topic-ref-real-001", "syllabus_full", "block_position", "chapter_order", "source_refs", "BOE-A-001"} {
		if !codexStackContextContainsForTestV0(context.Entries, want) {
			t.Fatalf("context no contiene %q: %+v", want, context.Entries)
		}
	}
}

func codexStackContextContainsForTestV0(
	entries []orquestacontext.ContextMaterializedEntryV0,
	want string,
) bool {
	for _, entry := range entries {
		if strings.Contains(entry.Content, want) {
			return true
		}
	}
	return false
}
