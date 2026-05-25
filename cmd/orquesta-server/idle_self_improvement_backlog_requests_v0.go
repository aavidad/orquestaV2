package main

import (
	"strconv"
	"strings"

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
	request.RequiredTests = append([]string(nil), base.RequiredTests...)
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
	request.AcceptanceCriteria = compactServerStackStringsV0(append(append(
		append([]string(nil), base.AcceptanceCriteria...),
		section.Criteria...,
	), append(
		idleSelfImprovementBacklogDependencyAcceptanceCriteriaV0(section),
		idleSelfImprovementBacklogManualVerificationCriteriaV0(section)...,
	)...))
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
	request.AcceptanceCriteria = compactServerStackStringsV0(append(append([]string(nil), base.AcceptanceCriteria...),
		"revisar evidencia declarada y marcar estado canonico o pendiente verificable",
		"no ejecutar cambios amplios de codigo desde una seccion con evidencia ambigua",
	))
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
	request.RequiredTests = append([]string(nil), base.RequiredTests...)
	if len(section.Tests) > 0 {
		request.RequiredTests = compactServerStackStringsV0(section.Tests)
	} else if len(section.ManualVerifications) > 0 {
		request.RequiredTests = nil
	}
	request.ContextRefs = compactServerStackStringsV0(append(append([]string(nil), base.ContextRefs...),
		"backlog-doc-autoprogramacion-2026-05-23",
		"backlog_doc:"+firstNonEmptyServerStackV0(section.SourcePath, idleSelfImprovementBacklogDocRelV0),
		"backlog_section:"+section.Ref,
		"backlog_line:"+strconv.Itoa(section.SourceLine),
	))
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
	return request
}

func idleSelfImprovementRequestRefForBacklogSectionV0(
	section idleSelfImprovementBacklogSectionV0,
) string {
	refSuffix := section.Ref + "-" + idleSelfImprovementBacklogHashV0(
		idleSelfImprovementBacklogSectionFingerprintV0(section),
	)
	return "request-ref-autoprogramming-backlog-" + refSuffix
}

func idleSelfImprovementBacklogFallbackRequestV0(
	base orquestaserver.IdleSelfImprovementRequestV0,
	reason string,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request := base
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs,
		"backlog_planner_fallback:"+strings.TrimSpace(reason),
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(request.EvidenceRefs,
		"evidence-ref-autoprogramming-backlog-planner-fallback",
	))
	return request
}

func idleSelfImprovementBacklogSummaryV0(section idleSelfImprovementBacklogSectionV0) string {
	objective := strings.TrimSpace(section.Objective)
	if objective == "" {
		objective = "cerrar seccion pendiente de autoprogramacion"
	}
	return "backlog pendiente " + section.Heading + ": " + objective
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

func idleSelfImprovementBacklogWriteSetEntriesV0(value string) []string {
	value = strings.TrimSpace(value)
	original := value
	var out []string
	for {
		_, rest, ok := strings.Cut(value, "`")
		if !ok {
			break
		}
		entry, next, ok := strings.Cut(rest, "`")
		if !ok {
			break
		}
		out = append(out, idleSelfImprovementCleanWriteSetEntryV0(entry))
		value = next
	}
	if len(out) > 0 {
		return compactServerStackStringsV0(out)
	}
	if entry := idleSelfImprovementCleanWriteSetEntryV0(original); entry != "" {
		return []string{entry}
	}
	return nil
}

func idleSelfImprovementCleanWriteSetEntryV0(value string) string {
	value = strings.TrimSpace(strings.Trim(strings.TrimSpace(value), "`"))
	for len(value) > 1 && strings.ContainsRune(".,;:", rune(value[len(value)-1])) {
		value = strings.TrimSpace(value[:len(value)-1])
	}
	return value
}

func idleSelfImprovementPlanEvidenceRefsV0(requests []orquestaserver.IdleSelfImprovementRequestV0) []string {
	refs := []string{"evidence-ref-autoprogramming-backlog-doc"}
	for _, request := range requests {
		refs = append(refs, request.EvidenceRefs...)
	}
	return compactServerStackStringsV0(refs)
}
