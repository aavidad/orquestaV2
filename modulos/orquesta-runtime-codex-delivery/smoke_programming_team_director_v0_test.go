package orquestaruntimecodexdelivery

import (
	"encoding/json"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func programmingTeamDirectorPacketV0(
	agentRef string,
	taskRef string,
	runRef string,
	phaseID string,
	correlationID string,
	tasks []programmingTeamTaskV0,
) orquestaruntime.AgentStartPacketV0 {
	if phaseID == "" {
		phaseID = string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	}
	return orquestaruntime.AgentStartPacketV0{
		SchemaVersion: orquestaruntime.AgentStartPacketSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: correlationID,
		WorkOrderRef:  taskRef,
		TargetModule:  "agenda-plan-director",
		Phase:         phaseID,
		CapacityLevel: "high",
		Locale:        "es-ES",
		Task: orquestaruntime.AgentStartTaskV0{
			TaskRef:      taskRef,
			Priority:     "alta",
			Title:        "Dirigir plan ejecutable de Agenda",
			Objective:    programmingTeamDirectorObjectiveV0(runRef, tasks),
			TargetSymbol: "AgendaDirector",
			WriteSet: []string{
				"docs/arquitectura.md",
				"docs/plan_microtareas.md",
			},
			DoneCriteria: []string{
				"Documentos de arquitectura y microtareas creados.",
				"director_decisions.json escrito con decisiones ejecutables.",
				"agent_ack.json escrito con status completed.",
			},
		},
		Context: orquestacontext.ContextMaterializedBundleV0{
			SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
			BundleRef:     "bundle-ref-agenda-plan-director",
			WorkOrderRef:  taskRef,
			TargetModule:  "agenda-plan-director",
			Entries: []orquestacontext.ContextMaterializedEntryV0{{
				EntryRef:  "entry-ref-agenda-plan-director",
				Layer:     orquestacontext.ContextLayerTaskContextV0,
				Kind:      orquestacontext.ContextEntryDocRefV0,
				SourceRef: "source-ref-agenda-plan-director",
				Mode:      orquestacontext.ContextMaterializationModeRefOnlyV0,
				Required:  true,
			}},
		},
		DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
			MailboxRef:   "mailbox-ref-agenda-plan-director",
			AckRef:       "ack-ref-agenda-plan-director",
			ReadinessRef: "readiness-ref-agenda-plan-director",
		},
		Policies: []string{"write_set_closed", "ack_required"},
	}
}

func programmingTeamDirectorObjectiveV0(runRef string, tasks []programmingTeamTaskV0) string {
	return strings.Join([]string{
		"Prepara una app pequena de agenda con API REST Go y web.",
		"Usa arquitectura hexagonal, textos visibles es/en y cortes pequenos.",
		"Escribe docs/arquitectura.md y docs/plan_microtareas.md con decisiones breves.",
		"Escribe tambien director_decisions.json con el JSON exacto indicado.",
		"Las decisiones deben crear microtareas y abrir programacion.",
		"JSON requerido:",
		programmingTeamDirectorDecisionsJSONV0(runRef, tasks),
	}, " ")
}

func programmingTeamDirectorDecisionsJSONV0(runRef string, tasks []programmingTeamTaskV0) string {
	data, err := json.Marshal(programmingTeamDirectorDecisionEnvelopeV0(runRef, tasks))
	if err != nil {
		return "{}"
	}
	return string(data)
}

func programmingTeamDirectorDecisionEnvelopeV0(
	runRef string,
	tasks []programmingTeamTaskV0,
) map[string]any {
	return map[string]any{
		"schema_version": "director_agent_decisions_file.v0",
		"decisions":      programmingTeamDirectorDecisionsV0(runRef, tasks),
	}
}

func programmingTeamDirectorDecisionsV0(
	runRef string,
	tasks []programmingTeamTaskV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	decisions := []orquestadirectoragent.DirectorAgentDecisionV0{
		programmingTeamOpenPhaseDecisionV0(
			runRef,
			"open-vote",
			orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
			orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
		),
		programmingTeamVoteDecisionV0(runRef),
		programmingTeamAcceptDecisionV0(runRef),
		programmingTeamOpenPhaseDecisionV0(
			runRef,
			"open-plan",
			orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
			orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		),
		programmingTeamContractDecisionV0(runRef),
	}
	for _, task := range tasks {
		decisions = append(decisions, programmingTeamMicrotaskDecisionV0(runRef, task))
	}
	return append(decisions, programmingTeamOpenPhaseDecisionV0(
		runRef,
		"open-programming",
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
	))
}

func programmingTeamOpenPhaseDecisionV0(
	runRef string,
	suffix string,
	current orquestacoreworkflow.OrchestrationPhaseIDV0,
	next orquestacoreworkflow.OrchestrationPhaseIDV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-agenda-" + suffix,
		RunID:         runRef,
		PhaseID:       string(current),
		CommandType:   orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		CommandRef:    "command-ref-agenda-" + suffix,
		Summary:       "Avanzar fase con salida suficiente.",
		EvidenceRefs:  []string{"evidence-ref-agenda-" + suffix},
		OpenPhase: &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
			PhaseID: string(next),
			Reason:  "Siguiente corte listo.",
		},
	}
}

func programmingTeamVoteDecisionV0(runRef string) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-agenda-vote",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandRequestVoteV0,
		CommandRef:    "command-ref-agenda-vote",
		Summary:       "Solicitar seleccion tecnica.",
		EvidenceRefs:  []string{"evidence-ref-agenda-vote"},
		RequestVote: &orquestadirectoragent.DirectorAgentVoteCommandV0{
			VoteRequestID:              "vote-ref-agenda-plan",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			DecisionTopicRef:           "topic-ref-agenda-plan",
			BrainstormRef:              "brainstorm-ref-app-001",
			Summary:                    "Elegir arquitectura por puertos.",
			MinimumRecommendedCapacity: orquestadirectoragent.DirectorAgentCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-agenda-vote"},
		},
	}
}

func programmingTeamAcceptDecisionV0(runRef string) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-agenda-accept",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandAcceptDecisionV0,
		CommandRef:    "command-ref-agenda-accept",
		Summary:       "Aceptar arquitectura compacta.",
		EvidenceRefs:  []string{"evidence-ref-agenda-accept"},
		AcceptDecision: &orquestadirectoragent.DirectorAgentAcceptDecisionCommandV0{
			DecisionRef:       "decision-ref-agenda-plan",
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			VoteRef:           "vote-ref-agenda-plan",
			AcceptedOptionRef: "option-ref-agenda-puertos",
			Summary:           "Arquitectura por puertos y textos externos.",
			EvidenceRefs:      []string{"evidence-ref-agenda-accept"},
		},
	}
}

func programmingTeamContractDecisionV0(runRef string) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-agenda-contract",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandPublishContractV0,
		CommandRef:    "command-ref-agenda-contract",
		Summary:       "Publicar contrato funcional compacto.",
		EvidenceRefs:  []string{"evidence-ref-agenda-contract"},
		PublishContract: &orquestadirectoragent.DirectorAgentPublishContractCommandV0{
			ContractRef:   "contract:function:agenda-work:v0",
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
			DecisionRef:   "decision-ref-agenda-plan",
			Summary:       "Contrato funcional para agenda.",
			FunctionNames: []string{"AgendaWork"},
			EvidenceRefs:  []string{"evidence-ref-agenda-contract"},
		},
	}
}

func programmingTeamMicrotaskDecisionV0(
	runRef string,
	task programmingTeamTaskV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-agenda-task-" + task.Area,
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CommandRef:    "command-ref-agenda-task-" + task.Area,
		Summary:       "Crear microtarea " + task.Area + ".",
		EvidenceRefs:  []string{"evidence-ref-agenda-task-" + task.Area},
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
				SchemaVersion:      orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0,
				TaskID:             task.TaskRef,
				RunID:              runRef,
				PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				Title:              task.Title,
				Summary:            task.Objective,
				WriteSet:           task.WriteSet,
				RequiredTests:      task.RequiredTests,
				AcceptanceCriteria: programmingTeamAcceptanceCriteriaV0(task),
				FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{{
					ContractRef:  "contract:function:agenda-work:v0",
					FunctionName: "AgendaWork",
				}},
			},
		},
	}
}

func programmingTeamAcceptanceCriteriaV0(task programmingTeamTaskV0) []string {
	criteria := []string{
		"Solo se modifica el write-set declarado.",
		"Cada fichero queda por debajo de 300 lineas.",
	}
	if len(task.RequiredTests) > 0 {
		criteria = append(criteria, "Pruebas obligatorias ejecutadas.")
	}
	return criteria
}

func programmingTeamProgrammingAgentsV0(tasks []programmingTeamTaskV0) []string {
	agents := make([]string, 0, len(tasks))
	for _, task := range tasks {
		agents = append(agents, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskRef))
	}
	return agents
}
