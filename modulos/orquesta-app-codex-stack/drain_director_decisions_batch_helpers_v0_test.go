package orquestaappcodexstack

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

type codexStackBatchDecisionTaskSpecForTestV0 struct {
	Suffix   string
	Title    string
	Summary  string
	WriteSet []string
}

func codexStackBatchDecisionTaskSpecsForTestV0() []codexStackBatchDecisionTaskSpecForTestV0 {
	return []codexStackBatchDecisionTaskSpecForTestV0{
		{
			Suffix:   "bootstrap",
			Title:    "Bootstrap Go",
			Summary:  "Crear modulo Go y punto de entrada HTTP.",
			WriteSet: []string{"go.mod", "cmd/server/main.go", "internal/config/config.go"},
		},
		{
			Suffix:   "domain",
			Title:    "Dominio agenda",
			Summary:  "Crear entidades, casos de uso y mensajes i18n.",
			WriteSet: []string{"internal/domain", "internal/application", "internal/i18n"},
		},
		{
			Suffix:   "http",
			Title:    "API REST",
			Summary:  "Crear rutas y handlers REST para agenda.",
			WriteSet: []string{"internal/http/router.go", "internal/http/handlers.go"},
		},
		{
			Suffix:   "web",
			Title:    "Web agenda",
			Summary:  "Crear interfaz web estatica conectada a la API.",
			WriteSet: []string{"web/static/index.html", "web/static/app.js", "web/static/styles.css"},
		},
		{
			Suffix:   "docs",
			Title:    "Documentacion y seguridad",
			Summary:  "Crear documentacion minima y pruebas de seguridad.",
			WriteSet: []string{"README.md", "docs/operacion.md", "internal/security"},
		},
	}
}

func codexStackBatchDecisionTaskRefsForTestV0() []string {
	specs := codexStackBatchDecisionTaskSpecsForTestV0()
	refs := make([]string, 0, len(specs))
	for _, spec := range specs {
		refs = append(refs, codexStackBatchTaskRefForTestV0(spec))
	}
	return refs
}

func codexStackMicrotaskDecisionForBatchTestV0(
	runRef string,
	spec codexStackBatchDecisionTaskSpecForTestV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	taskRef := codexStackBatchTaskRefForTestV0(spec)
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-stack-task-" + spec.Suffix,
		RunID:         runRef,
		PhaseID:       "planificacion_microtareas",
		CommandType:   orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CommandRef:    "command-ref-stack-task-" + spec.Suffix,
		Summary:       spec.Summary,
		EvidenceRefs:  []string{"evidence-ref-stack-task-" + spec.Suffix},
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
				SchemaVersion: orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0,
				TaskID:        taskRef,
				RunID:         runRef,
				PhaseID:       "programacion",
				Title:         spec.Title,
				Summary:       spec.Summary,
				WriteSet:      append([]string(nil), spec.WriteSet...),
				AcceptanceCriteria: []string{
					"Respeta arquitectura hexagonal e i18n.",
					"Compila con go test ./...",
					"No usa imports relativos ni acopla adaptadores al nucleo.",
				},
				RequiredTests: []string{"go test ./..."},
				DependsOn:     codexStackBatchTaskDependsOnForTestV0(spec),
				FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{{
					ContractRef:  "contract:function:stack-agenda:v0",
					FunctionName: "AgendaUseCases",
				}},
			},
		},
	}
}

func codexStackBatchTaskRefForTestV0(spec codexStackBatchDecisionTaskSpecForTestV0) string {
	return "task-ref-stack-agenda-" + spec.Suffix
}

func codexStackBatchTaskDependsOnForTestV0(
	spec codexStackBatchDecisionTaskSpecForTestV0,
) []string {
	if spec.Suffix == "bootstrap" {
		return nil
	}
	return []string{"task-ref-stack-agenda-bootstrap"}
}

func codexStackVerticalGoDirectorDecisionsForTestV0(
	runRef string,
	brainstormRef string,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	taskRef := "workflow-task-agenda-programacion-001"
	return []orquestadirectoragent.DirectorAgentDecisionV0{
		codexStackOpenPhaseDecisionForTestV0(
			runRef,
			"decision-ref-agenda-open-vote-001",
			"command-ref-agenda-open-vote-001",
			orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
			orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
		),
		codexStackVoteDecisionForTestV0(runRef, brainstormRef),
		codexStackAcceptDecisionForTestV0(runRef),
		codexStackOpenPhaseDecisionForTestV0(
			runRef,
			"decision-ref-agenda-open-plan-001",
			"command-ref-agenda-open-plan-001",
			orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
			orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		),
		codexStackVerticalGoContractDecisionForTestV0(runRef),
		codexStackVerticalGoMicrotaskDecisionForTestV0(runRef, taskRef),
		codexStackOpenPhaseDecisionForTestV0(
			runRef,
			"decision-ref-agenda-open-programming-001",
			"command-ref-agenda-open-programming-001",
			orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
			orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		),
	}
}

func codexStackVerticalGoContractDecisionForTestV0(
	runRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "decision-ref-agenda-contract-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandPublishContractV0,
		CommandRef:    "command-ref-agenda-contract-001",
		Summary:       "Publicar contrato funcional.",
		EvidenceRefs:  []string{"evidence-ref-docs-decisiones-v0"},
		PublishContract: &orquestadirectoragent.DirectorAgentPublishContractCommandV0{
			ContractRef: "function-contract-agenda-core-001",
			PhaseID: string(
				orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
			),
			DecisionRef: "decision-ref-stack-001",
			Summary:     "Contrato para contactos, citas y web.",
			FunctionNames: []string{
				"contacts.create",
				"contacts.list",
				"appointments.create",
				"appointments.list",
				"admin.web.manage",
			},
			EvidenceRefs: []string{"evidence-ref-docs-decisiones-v0"},
		},
	}
}

func codexStackVerticalGoMicrotaskDecisionForTestV0(
	runRef string,
	taskRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "decision-ref-agenda-task-programming-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CommandRef:    "command-ref-agenda-task-programming-001",
		Summary:       "Crear tarea vertical de programacion.",
		EvidenceRefs:  []string{"evidence-ref-docs-plan-v0"},
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
				SchemaVersion: orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0,
				TaskID:        taskRef,
				RunID:         runRef,
				PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				Title:         "Implementar Agenda API Web vertical",
				Summary:       "Crear modulo Go autonomo con servidor REST, dominio, puertos, web, README y pruebas.",
				WriteSet: []string{
					"go.mod",
					"cmd/server/main.go",
					"internal/domain/**",
					"internal/app/**",
					"internal/ports/**",
					"internal/http/**",
					"internal/memory/**",
					"internal/i18n/**",
					"README.md",
					"docs/**",
				},
				AcceptanceCriteria: []string{
					"objetivo_actual: Gestionar contactos y citas con API REST en Go y una web de administracion; persistencia por puerto.",
					"Modulo Go autonomo con imports desde go.mod y entrypoint cmd/server/main.go.",
					"Dominio no depende de tecnologia externa y usa puertos para persistencia.",
					"Web de administracion servida por el proceso y conectada a la API.",
				},
				FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{
					{ContractRef: "function-contract-agenda-core-001", FunctionName: "contacts.create"},
					{ContractRef: "function-contract-agenda-core-001", FunctionName: "appointments.create"},
					{ContractRef: "function-contract-agenda-core-001", FunctionName: "admin.web.manage"},
				},
				RequiredTests: []string{"go test ./..."},
			},
		},
	}
}
