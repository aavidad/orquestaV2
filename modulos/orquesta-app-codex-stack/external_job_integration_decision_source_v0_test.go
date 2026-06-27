package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestExternalJobIntegrationDecisionSourceV0CreaIntegradorParaPadreLegacyEstrecho(t *testing.T) {
	fixture := newExternalJobIntegrationDecisionSourceFixtureV0(true)
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			fixture.source,
		},
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: fixture.run},
	)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 1 ||
		decisions[0].CommandType != orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 ||
		decisions[0].CreateMicrotask == nil {
		t.Fatalf("decisions=%+v", decisions)
	}
	task := decisions[0].CreateMicrotask.Task
	if task.TaskID != externalJobIntegrationTaskRefV0(fixture.parent.TaskID) ||
		task.ParentTaskRef != fixture.parent.TaskID ||
		task.WorkProfileKind != string(orquestacoreworkflow.WorkProfileDomainWorkV0) ||
		!codexStackStringInSetForTestV0(task.WriteSet, "temas/tema_032") ||
		!codexStackStringInSetForTestV0(task.WriteSet, "temas/tema_032/coordinacion") ||
		!codexStackStringInSetForTestV0(task.DependsOn, fixture.parent.TaskID) ||
		!codexStackStringInSetForTestV0(task.DependsOn, fixture.parent.ChildTaskRefs[0]) ||
		!codexStackStringInSetForTestV0(task.ContextRefs, "external-job-integration-required") ||
		!codexStackStringInSetForTestV0(task.ContextRefs, externalJobIntegrationReasonV0) {
		t.Fatalf("task=%+v parent=%+v", task, fixture.parent)
	}
	if len(task.FunctionContractRefs) != 1 ||
		task.FunctionContractRefs[0].ContractRef != fixture.contractRef ||
		task.FunctionContractRefs[0].FunctionName != "ApplyExternalDomainWorkV0" {
		t.Fatalf("function_contract_refs=%+v", task.FunctionContractRefs)
	}
}

func TestExternalJobIntegrationDecisionSourceV0CreaIntegradorGenericoSinOPES(t *testing.T) {
	fixture := newExternalJobIntegrationDecisionSourceFixtureV0(true)
	fixture.record.Request.AppRef = "crm"
	fixture.record.Request.ExternalWork.ProjectRef = "project-ref-crm"
	fixture.record.Request.ExternalWork.InterfaceRefs = []string{"crm.document-bundle.v1"}
	fixture.record.Request.ExternalWork.InputFields = nil
	fixture.source.AppChangeStore = orquestaappchange.NewInMemoryAppChangeStoreV0(fixture.record)

	decisions, err := fixture.source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: fixture.run},
	)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 1 || decisions[0].CreateMicrotask == nil {
		t.Fatalf("decisions=%+v", decisions)
	}
	task := decisions[0].CreateMicrotask.Task
	if task.Title != "Integrar producto externo canonico" ||
		!codexStackStringInSetForTestV0(task.WriteSet, "temas/tema_032") ||
		!codexStackStringInSetForTestV0(task.ContextRefs, "external-job-integration-required") ||
		codexStackStringInSetForTestV0(task.ContextRefs, "opes-integration-required") {
		t.Fatalf("task=%+v", task)
	}
}

func TestExternalJobIntegrationDecisionSourceV0NoDuplicaPadreConProducto(t *testing.T) {
	fixture := newExternalJobIntegrationDecisionSourceFixtureV0(false)

	decisions, err := fixture.source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: fixture.run},
	)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestDrainRunV0CreaYLanzaIntegradorProductoParaPadreLegacyEstrecho(t *testing.T) {
	ctx := context.Background()
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	fixture := newExternalJobIntegrationDecisionSourceFixtureV0(true)
	fixture.run.DeliveredTasks = append(append([]string{}, fixture.parent.ChildTaskRefs...), fixture.parent.TaskID)
	fixture.run.ClosedTasks = append(append([]string{}, fixture.parent.ChildTaskRefs...), fixture.parent.TaskID)
	if err := stack.Stores.AppChangeStore.SaveAppChangeRequestV0(ctx, fixture.record); err != nil {
		t.Fatalf("SaveAppChangeRequestV0: %v", err)
	}
	if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(ctx, fixture.parent); err != nil {
		t.Fatalf("SaveWorkflowTaskV0 parent: %v", err)
	}
	for _, child := range fixture.children {
		if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(ctx, child); err != nil {
			t.Fatalf("SaveWorkflowTaskV0 child %s: %v", child.TaskID, err)
		}
	}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, fixture.run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}

	_, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               fixture.run.RunID,
		CorrelationID:        "corr-external-job-integration-flow-001",
		OccurredAt:           "2026-06-26T14:20:00Z",
		MaxBursts:            8,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 4,
		MaxCommands:          16,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     0,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v", err)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(ctx, fixture.run.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	integrationRef := externalJobIntegrationTaskRefV0(fixture.parent.TaskID)
	integrationAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(integrationRef)
	if !codexStackStringInSetForTestV0(run.Tasks, integrationRef) ||
		!codexStackStringInSetForTestV0(run.StartedAgents, integrationAgent) {
		t.Fatalf("run=%+v integration_ref=%s agent=%s", run, integrationRef, integrationAgent)
	}
	descriptor := mustCodexStackDescriptorByTaskRefV0(t, stack, integrationRef)
	if !codexStackStringInSetForTestV0(descriptor.Spec.AgentPacket.Task.WriteSet, "temas/tema_032") ||
		!codexStackStringInSetForTestV0(descriptor.Spec.AgentPacket.Task.WriteSet, "temas/tema_032/coordinacion") {
		t.Fatalf("packet write_set=%+v", descriptor.Spec.AgentPacket.Task.WriteSet)
	}
}

type externalJobIntegrationDecisionSourceFixtureV0 struct {
	source      ExternalJobIntegrationDecisionSourceV0
	run         orquestacoreworkflow.OrchestrationRunV0
	parent      orquestacoreworkflow.WorkflowTaskV0
	children    []orquestacoreworkflow.WorkflowTaskV0
	record      orquestaappchange.AppChangeRecordV0
	contractRef string
}

func newExternalJobIntegrationDecisionSourceFixtureV0(
	legacyNarrowParent bool,
) externalJobIntegrationDecisionSourceFixtureV0 {
	const (
		runRef     = "run-ref-external-job-integration-001"
		changeRef  = "opes-job-tema-integration-032"
		contract   = "contract:function:app-change:integration-032:v0"
		jobRef     = "job-ref-opes-integration-032"
		parentBase = "temas/tema_032"
	)
	parentRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	childRefs := []string{
		parentRef + "-subrole-fuentes",
		parentRef + "-subrole-reutilizacion",
		parentRef + "-subrole-redaccion",
		parentRef + "-subrole-visuales",
		parentRef + "-subrole-tests-tutor",
		parentRef + "-subrole-html-rag-audio-qa",
	}
	parentWriteSet := []string{parentBase, parentBase + "/coordinacion"}
	if legacyNarrowParent {
		parentWriteSet = []string{parentBase + "/coordinacion"}
	}
	parent := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             parentRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileDomainWorkV0,
		Title:              "Padre OPES legacy",
		WriteSet:           parentWriteSet,
		AcceptanceCriteria: []string{"entrega externa trazable"},
		RequiredTests:      []string{"validar contrato externo"},
		ChildTaskRefs:      childRefs,
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contract,
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
	children := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(childRefs))
	for _, childRef := range childRefs {
		children = append(children, orquestacoreworkflow.WorkflowTaskV0{
			SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
			TaskID:             childRef,
			RunID:              runRef,
			PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			WorkProfileKind:    orquestacoreworkflow.WorkProfileDomainWorkV0,
			Title:              "Subrol OPES legacy",
			Summary:            "Entrega de subrol usada como insumo del integrador.",
			WriteSet:           []string{parentBase + "/coordinacion"},
			AcceptanceCriteria: []string{"entrega de subrol trazable"},
			RequiredTests:      []string{"validar entrega de subrol"},
			ParentTaskRef:      parentRef,
			FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
				ContractRef:  contract,
				FunctionName: "ApplyExternalDomainWorkV0",
			}},
		})
	}
	record := orquestaappchange.AppChangeRecordV0{
		Request: orquestaappchange.AppChangeRequestV0{
			RunRef:          runRef,
			AppRef:          "opes",
			ChangeRef:       changeRef,
			AllowedWriteSet: []string{parentBase},
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef:    "opes",
				JobRef:        jobRef,
				InterfaceRefs: []string{"opes-rest-v0", "opes.padre-tema-6-subroles.v1"},
				WorkKind:      "draft_content_block",
				InputFields: []orquestadomainwork.DomainWorkFieldV0{{
					Name:  "subroles_required",
					Value: "6",
				}},
			},
		},
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "opes",
		AppSpecRef:    "opes",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases: []orquestacoreworkflow.OrchestrationPhaseV0{{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			EntryCriteria:       []string{"app-change externa aceptada"},
			ExitCriteria:        []string{"producto integrado o bloqueo causal"},
			EvidenceRequired:    []string{"entregas de subroles e integracion canonica"},
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		}},
		FunctionContracts: []string{contract},
		Tasks:             append([]string{parentRef}, childRefs...),
	}
	return externalJobIntegrationDecisionSourceFixtureV0{
		source: ExternalJobIntegrationDecisionSourceV0{
			AppChangeStore: orquestaappchange.NewInMemoryAppChangeStoreV0(record),
			TaskStore:      orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(append([]orquestacoreworkflow.WorkflowTaskV0{parent}, children...)...),
		},
		run:         run,
		parent:      parent,
		children:    children,
		record:      record,
		contractRef: contract,
	}
}
