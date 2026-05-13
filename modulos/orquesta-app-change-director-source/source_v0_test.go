package orquestaappchangedirectorsource

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

func TestAppChangeDirectorDecisionSourceV0GeneraCadenaCompleta(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 8 {
		t.Fatalf("decisions=%+v", decisions)
	}
	if decisions[0].CommandType != orquestadirectoragent.DirectorAgentCommandAnswerQuestionV0 ||
		decisions[6].CommandType != orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 {
		t.Fatalf("command types=%s %s", decisions[0].CommandType, decisions[6].CommandType)
	}
	task := decisions[6].CreateMicrotask.Task
	if len(task.WriteSet) != 1 || task.WriteSet[0] != "web/agenda" {
		t.Fatalf("write_set=%+v", task.WriteSet)
	}
	if len(task.RequiredTests) == 0 {
		t.Fatalf("required_tests vacio: %+v", task)
	}
	if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(decisions[6]); len(issues) != 0 {
		t.Fatalf("decision invalida: %+v", issues)
	}
}

func TestAppChangeDirectorDecisionSourceV0ExigeGoTestSiTocaCodigoGo(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = []string{"web/agenda/view.go"}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	task := decisions[6].CreateMicrotask.Task
	if len(task.RequiredTests) == 0 || task.RequiredTests[0] != "go test ./..." {
		t.Fatalf("required_tests=%+v", task.RequiredTests)
	}
}

func TestAppChangeDirectorDecisionSourceV0ProyectaTrabajoExterno(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "project-ref-opes",
		InterfaceRefs: []string{"mcp-contract-ref-opes-v0"},
		WorkKind:      "documentation",
		WorkRefs:      []string{"domain-work-ref-opes-topic-001"},
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	contract := decisions[5].PublishContract
	task := decisions[6].CreateMicrotask.Task
	if contract == nil ||
		len(contract.FunctionNames) != 1 ||
		contract.FunctionNames[0] != "ApplyExternalDomainWorkV0" ||
		task.Title != "Resolver trabajo documental externo" ||
		task.Summary == "" ||
		!stringInSetV0(task.RequiredTests, "validar contrato externo de dominio") {
		t.Fatalf("contract=%+v task=%+v", contract, task)
	}
}

func TestAppChangeDirectorDecisionSourceV0NoInventaMicrotareaSinWriteSet(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestAppChangeDirectorDecisionSourceV0ConsumeCambioRecibidoComoEvento(t *testing.T) {
	store := orquestaappchange.NewInMemoryAppChangeStoreV0()
	event := orquestaappchange.AppChangeIntentEventV0{
		SchemaVersion:      orquestaappchange.AppChangeIntentEventSchemaV0,
		EventID:            "event-ref-source-mid-001",
		Source:             "orquesta-mcp",
		OccurredAt:         "2026-05-10T10:25:00Z",
		RunRef:             "run-ref-source-001",
		ChangeRef:          "change-ref-source-mid-001",
		UserIntent:         "Quiero modificar la web para mostrar filtros.",
		AcceptanceCriteria: []string{"filtros visibles"},
		AllowedWriteSet:    []string{"web/agenda"},
	}

	result, err := orquestaappchange.ReceiveAppChangeIntentEventV0(
		context.Background(),
		event,
		orquestaappchange.AppChangePortsV0{
			Store:            store,
			DirectorNotifier: appChangeSourceNoopNotifierV0{},
		},
	)
	if err != nil {
		t.Fatalf("ReceiveAppChangeIntentEventV0: %v", err)
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:             event.RunRef,
		CurrentPhase:      orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		DirectorQuestions: []string{result.DirectorQuestionRef},
		Decisions:         []string{"decision-ref-base-001"},
	}
	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 3 ||
		decisions[2].CommandType != orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func appChangeRecordForSourceTestV0() orquestaappchange.AppChangeRecordV0 {
	return orquestaappchange.AppChangeRecordV0{
		Request: orquestaappchange.AppChangeRequestV0{
			RunRef:             "run-ref-source-001",
			ChangeRef:          "change-ref-source-001",
			UserIntent:         "Cambiar la web para mostrar vista semanal.",
			AcceptanceCriteria: []string{"vista semanal visible"},
			AllowedWriteSet:    []string{"web/agenda"},
		},
	}
}

type appChangeSourceNoopNotifierV0 struct{}

func (appChangeSourceNoopNotifierV0) NotifyAppChangeRequestedV0(
	context.Context,
	orquestaappchange.AppChangeRecordV0,
) (orquestaappchange.AppChangeDirectorNotificationV0, error) {
	return orquestaappchange.AppChangeDirectorNotificationV0{}, nil
}

func appChangeRunForSourceTestV0(
	record orquestaappchange.AppChangeRecordV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		RunID:             record.Request.RunRef,
		CurrentPhase:      orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		DirectorQuestions: []string{appChangeQuestionRefV0(record.Request.ChangeRef)},
	}
}
