package main

import (
	"strconv"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func idleSelfImprovementBacklogScannerRequestV0(
	base orquestaserver.IdleSelfImprovementRequestV0,
	plan orquestaserver.IdleSelfImprovementPlanRequestV0,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request := base
	refSuffix := "scanner-" + idleSelfImprovementBacklogHashV0(base.ProjectRef+"|"+strings.TrimSpace(plan.Trigger))
	request.RequestRef = "request-ref-autoprogramming-backlog-" + refSuffix
	request.CorrelationID = "corr-" + request.RequestRef
	request.FailureKind = "backlog_scan"
	request.FailureSummary = "revisar Orquesta, detectar huecos concretos pendientes y anadirlos al backlog de automejora"
	request.SuggestedArea = "backlog-scan"
	request.WriteSet = []string{
		idleSelfImprovementBacklogDocRelV0,
		"docs/rail_errors_observados_2026-05-23.md",
		"docs/duplicaciones_railes_pendientes_2026-05-24.md",
	}
	var testContextRefs []string
	request.RequiredTests, testContextRefs = idleSelfImprovementBacklogScannerRequiredTestsV0(base.RequiredTests, request.WriteSet)
	request.AcceptanceCriteria = compactServerStackStringsV0(append(append([]string(nil), base.AcceptanceCriteria...),
		"detectar huecos reales de nucleo/director/adaptadores sin duplicar tareas ya en cola",
		"anadir secciones Txx concretas al backlog con objetivo, alcance, criterios y tests",
		"no programar cambios amplios desde el scanner; dejar tareas ejecutables para el siguiente ciclo",
		"si no hay huecos nuevos, documentar evidencia breve y no inventar trabajo",
	))
	request.CompactRules = compactServerStackStringsV0(append(append([]string(nil), base.CompactRules...),
		"scanner de backlog: salida compacta y tareas concretas",
	))
	request.ContextRefs = compactServerStackStringsV0(append(append([]string(nil), base.ContextRefs...),
		"backlog-doc-autoprogramacion-2026-05-23",
		"trigger:"+firstNonEmptyServerStackV0(strings.TrimSpace(plan.Trigger), "scanner"),
		"queue_size:"+strconv.Itoa(plan.QueueSize),
		"free_capacity:"+strconv.Itoa(plan.FreeCapacity),
	))
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs, testContextRefs...))
	request.EvidenceRefs = compactServerStackStringsV0(append(append([]string(nil), base.EvidenceRefs...),
		"evidence-ref-autoprogramming-backlog-scanner",
	))
	return request
}

func idleSelfImprovementRequestForBacklogSectionV0(
	base orquestaserver.IdleSelfImprovementRequestV0,
	section idleSelfImprovementBacklogSectionV0,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request := idleSelfImprovementBaseRequestForBacklogSectionV0(base, section)
	request.FailureKind = "backlog_autoprogramming"
	request.FailureSummary = idleSelfImprovementBacklogSummaryV0(section)
	request.WriteSet = idleSelfImprovementBacklogWriteSetV0(base.WriteSet, section.Scope)
	request.AcceptanceChecks = append([]orquestaautoprogramming.AutoprogrammingAcceptanceCheckV0(nil), section.AcceptanceChecks...)
	request.AcceptanceCriteria = compactServerStackStringsV0(append(
		append([]string(nil), section.Criteria...),
		append(
			idleSelfImprovementBacklogDependencyAcceptanceCriteriaV0(section),
			append(
				idleSelfImprovementBacklogManualVerificationCriteriaV0(section),
				idleSelfImprovementBacklogTaskInstanceCriteriaV0(section)...,
			)...,
		)...,
	))
	request.CompactRules = compactServerStackStringsV0(append(append([]string(nil), base.CompactRules...),
		"el director revisa backlog y genera tareas concretas; no una tarea generica",
		"un agente padre por tarea; subagentes hasta 6 si ayudan",
		"si aparece otro hueco general, registrarlo como nueva automejora y seguir",
	))
	return request
}

func idleSelfImprovementDocumentReviewRequestForBacklogSectionV0(
	base orquestaserver.IdleSelfImprovementRequestV0,
	section idleSelfImprovementBacklogSectionV0,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request := idleSelfImprovementBaseRequestForBacklogSectionV0(base, section)
	request.FailureKind = "backlog_documentation_review"
	request.FailureSummary = "sincronizar estado documental ambiguo de " + section.Heading
	request.SuggestedArea = firstNonEmptyServerStackV0(section.Ref+"-documentacion", base.SuggestedArea)
	request.WriteSet = idleSelfImprovementBacklogDocumentReviewWriteSetV0(section)
	request.AcceptanceCriteria = compactServerStackStringsV0([]string{
		"revisar evidencia declarada y marcar estado canonico o pendiente verificable",
		"no ejecutar cambios amplios de codigo desde una seccion con evidencia ambigua",
	})
	request.CompactRules = compactServerStackStringsV0(append(append([]string(nil), base.CompactRules...),
		"revision documental acotada; no tocar runtime ni worktrees",
	))
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs,
		"backlog_state:ambiguous_evidence",
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(request.EvidenceRefs,
		section.StateEvidenceRefs...,
	))
	return request
}

func idleSelfImprovementBaseRequestForBacklogSectionV0(
	base orquestaserver.IdleSelfImprovementRequestV0,
	section idleSelfImprovementBacklogSectionV0,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request := base
	request.RequestRef = idleSelfImprovementRequestRefForBacklogSectionV0(section)
	request.CorrelationID = "corr-" + request.RequestRef
	request.SuggestedArea = firstNonEmptyServerStackV0(section.Ref, base.SuggestedArea)
	request.WriteSet = idleSelfImprovementBacklogWriteSetV0(base.WriteSet, section.Scope)
	var testContextRefs []string
	request.RequiredTests, testContextRefs = idleSelfImprovementBacklogSectionRequiredTestsV0(
		base.RequiredTests,
		section.Tests,
		section.ManualVerifications,
		request.WriteSet,
	)
	request.ContextRefs = compactServerStackStringsV0(append(append([]string(nil), base.ContextRefs...),
		"backlog-doc-autoprogramacion-2026-05-23",
		"backlog_doc:"+firstNonEmptyServerStackV0(section.SourcePath, idleSelfImprovementBacklogDocRelV0),
		"backlog_section:"+section.Ref,
		"backlog_line:"+strconv.Itoa(section.SourceLine),
	))
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs,
		idleSelfImprovementBacklogTaskInstanceContextRefsV0(section)...))
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs, testContextRefs...))
	request.ContextRefs = compactServerStackStringsV0(append(
		request.ContextRefs,
		idleSelfImprovementBacklogDependencyContextRefsV0(section)...,
	))
	request.ContextRefs = compactServerStackStringsV0(append(
		request.ContextRefs,
		idleSelfImprovementBacklogIOContextRefsV0(section)...,
	))
	request.ContextRefs = compactServerStackStringsV0(append(
		request.ContextRefs,
		idleSelfImprovementFederatedBacklogContextRefsV0(section)...,
	))
	request.ContextRefs = compactServerStackStringsV0(append(
		request.ContextRefs,
		idleSelfImprovementBacklogManualVerificationContextRefsV0(section)...,
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(append([]string(nil), base.EvidenceRefs...),
		"evidence-ref-autoprogramming-backlog-doc",
		"evidence-ref-autoprogramming-backlog-section-"+section.Ref,
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(request.EvidenceRefs,
		section.DiagnosticEvidenceRefs...,
	))
	return request
}

func idleSelfImprovementBacklogFallbackRequestV0(
	base orquestaserver.IdleSelfImprovementRequestV0,
	plan orquestaserver.IdleSelfImprovementPlanRequestV0,
	reason string,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request := idleSelfImprovementBacklogScannerRequestV0(base, plan)
	request.FailureSummary = "revisar backlog de automejora degradado antes de programar cambios de codigo"
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs,
		"backlog_planner_fallback:"+strings.TrimSpace(reason),
	))
	request.AcceptanceCriteria = compactServerStackStringsV0(append(request.AcceptanceCriteria,
		"si el backlog no se puede leer, dejar bloqueo documental publico y no preparar trabajo de codigo generico",
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(request.EvidenceRefs,
		"evidence-ref-autoprogramming-backlog-planner-fallback",
	))
	return request
}

func idleSelfImprovementBacklogWriteSetV0(base []string, scope []string) []string {
	if len(scope) == 0 {
		return append([]string(nil), base...)
	}
	out := make([]string, 0, len(scope))
	for _, value := range scope {
		out = append(out, idleSelfImprovementBacklogWriteSetEntriesV0(value)...)
	}
	return compactServerStackStringsV0(out)
}

func idleSelfImprovementBacklogDocumentReviewWriteSetV0(section idleSelfImprovementBacklogSectionV0) []string {
	out := []string{idleSelfImprovementBacklogDocRelV0, "docs/runbooks"}
	for _, scope := range idleSelfImprovementBacklogWriteSetV0(nil, section.Scope) {
		if strings.HasPrefix(scope, "modulos/") {
			out = append(out, scope+"/docs/tareas.md", scope+"/docs/decisiones.md", scope+"/docs/pruebas.md", scope+"/README.md")
		}
		if strings.HasPrefix(scope, "docs/") {
			out = append(out, scope)
		}
	}
	return compactServerStackStringsV0(out)
}

func idleSelfImprovementBacklogIOContextRefsV0(
	section idleSelfImprovementBacklogSectionV0,
) []string {
	inputs := section.Inputs
	if len(inputs) == 0 {
		inputs = []string{"backlog_section:" + section.Ref}
		if len(section.Scope) > 0 {
			inputs = append(inputs, "write_set:"+strings.Join(section.Scope, "+"))
		}
	}
	outputs := section.Outputs
	if len(outputs) == 0 {
		outputs = []string{"entrega verificable para " + section.Ref}
		if len(section.Tests) > 0 {
			outputs = append(outputs, "tests:"+strings.Join(section.Tests, "+"))
		}
		if len(section.ManualVerifications) > 0 {
			outputs = append(outputs, "manual_verification:"+strings.Join(section.ManualVerifications, "+"))
		}
	}
	refs := make([]string, 0, len(inputs)+len(outputs))
	for _, input := range inputs {
		refs = append(refs, "backlog_input:"+strings.TrimSpace(input))
	}
	for _, output := range outputs {
		refs = append(refs, "backlog_output:"+strings.TrimSpace(output))
	}
	return compactServerStackStringsV0(refs)
}

func idleSelfImprovementBacklogDependencyContextRefsV0(
	section idleSelfImprovementBacklogSectionV0,
) []string {
	refs := make([]string, 0, len(section.Dependencies))
	for _, dependency := range section.Dependencies {
		refs = append(refs, "backlog_dependency:"+dependency)
		if idleSelfImprovementBacklogHasRunnableDependencyContractV0(section) {
			refs = append(refs, "backlog_dependency_contract_ready:"+dependency)
		}
	}
	return compactServerStackStringsV0(refs)
}

func idleSelfImprovementBacklogDependencyAcceptanceCriteriaV0(
	section idleSelfImprovementBacklogSectionV0,
) []string {
	if len(section.Dependencies) == 0 {
		return nil
	}
	if idleSelfImprovementBacklogHasRunnableDependencyContractV0(section) {
		return []string{
			"si alguna dependencia sigue abierta, programar solo contra las entradas/salidas declaradas y dejar el ajuste final como tarea separada si cambia el contrato",
		}
	}
	return []string{
		"respetar las dependencias declaradas antes de cerrar esta mejora",
	}
}
