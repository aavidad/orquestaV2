package orquestaappcodexstack

import orquestadirectoragent "orquesta/modulos/orquesta-director-agent"

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
