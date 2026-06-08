package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestStartAppDirectorV0ConsumesDecisionsAndRerunsProgrammingLoop(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := &serviceAutonomyDecisionSourceForTestV0{}

	request := validStartAppDirectorRequestForTestV0()
	request.MaxDecisionCycles = 2
	request.MaxBursts = 8

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:               store,
			EventSink:              sink,
			OutboxLedger:           ledger,
			DeliverySource:         serviceDirectorArtifactSourceForTestV0{},
			DirectorDecisionSource: source,
			DirectorTaskStore:      taskStore,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if source.calls < 2 {
		t.Fatalf("decision_source_calls=%d", source.calls)
	}
	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s", result.Run.CurrentPhase)
	}
	if !serviceStringInSetV0(result.Run.Tasks, "task-ref-agenda-autonomy-001") {
		t.Fatalf("tasks=%v", result.Run.Tasks)
	}
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-agenda-autonomy-001")
	if !serviceStringInSetV0(result.Run.StartedAgents, agentRef) {
		pendingRefs := servicePendingOutboxRefsForTestV0(t, ledger, result.Run.RunID)
		t.Fatalf(
			"started_agents=%v missing=%s capacity=%v decisions=%v gates=%v agents=%v loop=%s pending=%v",
			result.Run.StartedAgents,
			agentRef,
			result.Run.CapacityRequests,
			result.Run.CapacityDecisions,
			result.Run.ConcurrencyGates,
			result.Run.Agents,
			result.LoopStatus,
			pendingRefs,
		)
	}
}

func TestStartAppDirectorV0ConsumesPlanTeamDecisionAndRerunsProgrammingLoop(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := &servicePlanTeamDecisionSourceForTestV0{}

	request := validStartAppDirectorRequestForTestV0()
	request.MaxDecisionCycles = 2
	request.MaxBursts = 12

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:               store,
			EventSink:              sink,
			OutboxLedger:           ledger,
			DeliverySource:         serviceDirectorArtifactSourceForTestV0{},
			DirectorDecisionSource: source,
			DirectorTaskStore:      taskStore,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if source.calls < 2 {
		t.Fatalf("decision_source_calls=%d", source.calls)
	}
	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s", result.Run.CurrentPhase)
	}
	taskRefs := []string{
		"task-ref-agenda-plan-team-domain-001",
		"task-ref-agenda-plan-team-api-001",
	}
	for _, taskRef := range taskRefs {
		if !serviceStringInSetV0(result.Run.Tasks, taskRef) {
			t.Fatalf("tasks=%v missing=%s", result.Run.Tasks, taskRef)
		}
		agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
		if !serviceStringInSetV0(result.Run.StartedAgents, agentRef) {
			pendingRefs := servicePendingOutboxRefsForTestV0(t, ledger, result.Run.RunID)
			t.Fatalf(
				"started_agents=%v missing=%s capacity=%v decisions=%v gates=%v agents=%v loop=%s pending=%v",
				result.Run.StartedAgents,
				agentRef,
				result.Run.CapacityRequests,
				result.Run.CapacityDecisions,
				result.Run.ConcurrencyGates,
				result.Run.Agents,
				result.LoopStatus,
				pendingRefs,
			)
		}
	}
	loaded, err := taskStore.LoadWorkflowTasksV0(context.Background(), result.Run.RunID, taskRefs)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(loaded) != len(taskRefs) {
		t.Fatalf("loaded_tasks=%d want=%d tasks=%+v", len(loaded), len(taskRefs), loaded)
	}
	if loaded[0].FunctionContractRefs[0].ContractRef != "contract:function:agenda-autonomy:v0" ||
		loaded[1].FunctionContractRefs[0].FunctionName != "AgendaAPI" {
		t.Fatalf("function_contract_refs=%+v %+v", loaded[0].FunctionContractRefs, loaded[1].FunctionContractRefs)
	}
}

func servicePendingOutboxRefsForTestV0(
	t *testing.T,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	runRef string,
) []string {
	t.Helper()
	pending, issues := ledger.ListPending(context.Background(),
		orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{RunRef: runRef},
	)
	if len(issues) > 0 {
		t.Fatalf("ListPending: %+v", issues)
	}
	refs := make([]string, 0, len(pending))
	for _, item := range pending {
		refs = append(refs, item.MessageID)
	}
	return refs
}

type serviceAutonomyDecisionSourceForTestV0 struct {
	calls int
}

type servicePlanTeamDecisionSourceForTestV0 struct {
	calls int
}

func (source *serviceAutonomyDecisionSourceForTestV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source.calls++
	if source.calls > 1 {
		return nil, nil
	}
	return serviceDirectorPlanDecisionsForTestV0(request.Run), nil
}

func (source *servicePlanTeamDecisionSourceForTestV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source.calls++
	if source.calls > 1 {
		return nil, nil
	}
	return serviceDirectorPlanTeamDecisionsForTestV0(request.Run), nil
}

func serviceDirectorPlanDecisionsForTestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	brainstormRef := "brainstorm-ref-autonomy-001"
	if len(run.Brainstorms) > 0 {
		brainstormRef = run.Brainstorms[0]
	}
	return []orquestadirectoragent.DirectorAgentDecisionV0{
		serviceOpenPhaseDecisionForTestV0(
			run.RunID,
			"director-decision-autonomy-open-vote-001",
			"command-ref-autonomy-open-vote-001",
			orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
			orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
		),
		serviceVoteDecisionForTestV0(run.RunID, brainstormRef),
		serviceAcceptDecisionForTestV0(run.RunID),
		serviceOpenPhaseDecisionForTestV0(
			run.RunID,
			"director-decision-autonomy-open-plan-001",
			"command-ref-autonomy-open-plan-001",
			orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
			orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		),
		serviceContractDecisionForTestV0(run.RunID),
		serviceMicrotaskDecisionForTestV0(run.RunID),
		serviceOpenPhaseDecisionForTestV0(
			run.RunID,
			"director-decision-autonomy-open-program-001",
			"command-ref-autonomy-open-program-001",
			orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
			orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		),
	}
}

func serviceDirectorPlanTeamDecisionsForTestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	decisions := serviceDirectorPlanDecisionsForTestV0(run)
	for index := range decisions {
		if decisions[index].CommandType == orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 {
			decisions[index] = servicePlanTeamDecisionForTestV0(run.RunID)
			break
		}
	}
	return decisions
}

func serviceOpenPhaseDecisionForTestV0(
	runRef string,
	decisionRef string,
	commandRef string,
	currentPhase orquestacoreworkflow.OrchestrationPhaseIDV0,
	nextPhase orquestacoreworkflow.OrchestrationPhaseIDV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   decisionRef,
		RunID:         runRef,
		PhaseID:       string(currentPhase),
		CommandType:   orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		CommandRef:    commandRef,
		Summary:       "Avanzar fase del flujo.",
		EvidenceRefs:  []string{"evidence-ref-" + decisionRef},
		OpenPhase: &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
			PhaseID: string(nextPhase),
			Reason:  "Fase anterior con salida suficiente.",
		},
	}
}

func serviceVoteDecisionForTestV0(
	runRef string,
	brainstormRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-autonomy-vote-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandRequestVoteV0,
		CommandRef:    "command-ref-autonomy-vote-001",
		Summary:       "Solicitar voto tecnico.",
		EvidenceRefs:  []string{"evidence-ref-autonomy-vote-001"},
		RequestVote: &orquestadirectoragent.DirectorAgentVoteCommandV0{
			VoteRequestID:              "vote-ref-autonomy-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			DecisionTopicRef:           "topic-ref-agenda-autonomy-001",
			BrainstormRef:              brainstormRef,
			Summary:                    "Elegir arquitectura compacta.",
			MinimumRecommendedCapacity: orquestadirectoragent.DirectorAgentCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-autonomy-vote-001"},
		},
	}
}

func serviceAcceptDecisionForTestV0(runRef string) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-autonomy-accept-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandAcceptDecisionV0,
		CommandRef:    "command-ref-autonomy-accept-001",
		Summary:       "Aceptar opcion de arquitectura compacta.",
		EvidenceRefs:  []string{"evidence-ref-autonomy-accept-001"},
		AcceptDecision: &orquestadirectoragent.DirectorAgentAcceptDecisionCommandV0{
			DecisionRef:       "decision-ref-autonomy-001",
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			VoteRef:           "vote-ref-autonomy-001",
			AcceptedOptionRef: "option-ref-autonomy-001",
			Summary:           "Arquitectura con puertos y textos externos.",
			EvidenceRefs:      []string{"evidence-ref-autonomy-accept-001"},
		},
	}
}

func serviceContractDecisionForTestV0(runRef string) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-autonomy-contract-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandPublishContractV0,
		CommandRef:    "command-ref-autonomy-contract-001",
		Summary:       "Publicar contrato funcional.",
		EvidenceRefs:  []string{"evidence-ref-autonomy-contract-001"},
		PublishContract: &orquestadirectoragent.DirectorAgentPublishContractCommandV0{
			ContractRef:   "contract:function:agenda-autonomy:v0",
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
			DecisionRef:   "decision-ref-autonomy-001",
			Summary:       "Contrato para agenda compacta.",
			FunctionNames: []string{"AgendaUseCases"},
			EvidenceRefs:  []string{"evidence-ref-autonomy-contract-001"},
		},
	}
}

func servicePlanTeamDecisionForTestV0(runRef string) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-autonomy-plan-team-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandProposePlanTeamV0,
		CommandRef:    "command-ref-autonomy-plan-team-001",
		Summary:       "Crear dos microtareas coordinadas para agenda.",
		EvidenceRefs:  []string{"evidence-ref-autonomy-plan-team-001"},
		ProposePlanTeam: &orquestadirectoragent.DirectorAgentPlanTeamCommandV0{
			Plan: orquestadirectoragent.DirectorAgentAutonomousPlanTeamV0{
				SchemaVersion: orquestadirectoragent.DirectorAgentAutonomousPlanTeamSchemaVersionV0,
				PlanRef:       "plan-ref-autonomy-plan-team-001",
				RunID:         runRef,
				PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
				GoalRef:       "goal-ref-agenda-autonomy-001",
				Summary:       "Dividir el trabajo de agenda en dominio e interfaz.",
				Team: []orquestadirectoragent.DirectorAgentTeamMemberV0{
					{
						MemberRef:          "member-ref-agenda-domain-001",
						Role:               "dominio",
						Capacity:           orquestadirectoragent.DirectorAgentCapacityMediumV0,
						ResponsibilityRefs: []string{"responsibility-ref-agenda-domain-001"},
					},
					{
						MemberRef:          "member-ref-agenda-api-001",
						Role:               "api",
						Capacity:           orquestadirectoragent.DirectorAgentCapacityMediumV0,
						ResponsibilityRefs: []string{"responsibility-ref-agenda-api-001"},
					},
				},
				WorkUnits: []orquestadirectoragent.DirectorAgentAutonomousWorkUnitV0{
					{
						WorkUnitRef:       "task-ref-agenda-plan-team-domain-001",
						PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
						WorkProfileKind:   string(orquestacoreworkflow.WorkProfileImplementationV0),
						Title:             "Implementar dominio de agenda",
						Summary:           "Crear entidades y casos de uso de agenda.",
						AssignedMemberRef: "member-ref-agenda-domain-001",
						WriteSet:          []string{"internal/agenda/domain"},
						AcceptanceCriteria: []string{
							"Compila pruebas unitarias.",
							"Respeta contrato funcional publicado.",
						},
						FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{
							{ContractRef: "contract:function:agenda-autonomy:v0", FunctionName: "AgendaUseCases"},
						},
					},
					{
						WorkUnitRef:       "task-ref-agenda-plan-team-api-001",
						PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
						WorkProfileKind:   string(orquestacoreworkflow.WorkProfileImplementationV0),
						Title:             "Implementar API de agenda",
						Summary:           "Exponer operaciones de agenda para la aplicacion.",
						AssignedMemberRef: "member-ref-agenda-api-001",
						WriteSet:          []string{"internal/agenda/api"},
						AcceptanceCriteria: []string{
							"Compila pruebas unitarias.",
							"Respeta contrato funcional publicado.",
						},
						FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{
							{ContractRef: "contract:function:agenda-autonomy:v0", FunctionName: "AgendaAPI"},
						},
					},
				},
				EvidenceRefs: []string{"evidence-ref-autonomy-plan-team-002"},
			},
		},
	}
}

func serviceMicrotaskDecisionForTestV0(runRef string) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-autonomy-task-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CommandRef:    "command-ref-autonomy-task-001",
		Summary:       "Crear microtarea de agenda.",
		EvidenceRefs:  []string{"evidence-ref-autonomy-task-001"},
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
				SchemaVersion:      orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0,
				TaskID:             "task-ref-agenda-autonomy-001",
				RunID:              runRef,
				PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				Title:              "Crear agenda minima",
				Summary:            "Implementar casos de uso de agenda.",
				WriteSet:           []string{"internal/agenda"},
				AcceptanceCriteria: []string{"Compila pruebas unitarias.", "Expone interfaz funcional."},
				RequiredTests:      []string{"go test ./..."},
				FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{{
					ContractRef:  "contract:function:agenda-autonomy:v0",
					FunctionName: "AgendaUseCases",
				}},
			},
		},
	}
}
