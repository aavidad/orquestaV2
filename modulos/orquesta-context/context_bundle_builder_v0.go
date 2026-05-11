package orquestacontext

func BuildContextBundleV0(request ContextBundleRequestV0) ContextBundleV0 {
	request = normalizeContextBundleRequestV0(request)
	bundle := ContextBundleV0{
		SchemaVersion: ContextBundleSchemaVersionV0,
		BundleRef:     request.BundleRef,
		WorkOrderRef:  request.WorkOrderRef,
		TargetModule:  request.TargetModule,
		Phase:         request.Phase,
		CapacityLevel: request.CapacityLevel,
		Summary: ContextBundleSummaryV0{
			TaskKind:               request.TaskKind,
			Objective:              request.Objective,
			ContextPolicy:          "contexto_pequeno_por_refs",
			DirectorQuestionPolicy: "emitir_CONSULTA_AL_DIRECTOR_si_falta_contexto_externo",
		},
		Limits: ContextBundleLimitsV0{
			MaxEntries:    request.MaxEntries,
			MaxTotalBytes: request.MaxTotalBytes,
		},
	}
	if issues := ValidateContextBundleRequestV0(request); len(issues) > 0 {
		bundle.Issues = issues
		return bundle
	}

	entries := baseContextEntriesV0(request)
	entries = append(entries, phaseContextEntriesV0(request, len(entries)+1)...)
	entries = append(entries, refContextEntriesV0(request, len(entries)+1)...)
	if len(entries) > request.MaxEntries {
		bundle.Issues = append(bundle.Issues,
			contextBundleIssueV0(ErrContextBundleTamanoInvalidoV0, "entries", "demasiadas entradas de contexto"))
		return bundle
	}
	bundle.Entries = entries
	return bundle
}

func baseContextEntriesV0(request ContextBundleRequestV0) []ContextBundleEntryV0 {
	moduleRoot := "modulos/" + request.TargetModule
	return []ContextBundleEntryV0{
		contextBundleEntryV0(1, ContextLayerCommonRulesV0, ContextEntryRuleRefV0,
			"orquesta-common-rules:v0", "reglas comunes compactas", true),
		contextBundleEntryV0(2, ContextLayerModuleContextV0, ContextEntryDocRefV0,
			moduleRoot+"/AGENTS.md", "contexto primario del modulo", true),
		contextBundleEntryV0(3, ContextLayerModuleContextV0, ContextEntryDocRefV0,
			moduleRoot+"/README.md", "responsabilidad local del modulo", true),
		contextBundleEntryV0(4, ContextLayerModuleContextV0, ContextEntryDocRefV0,
			moduleRoot+"/docs/contratos.md", "contratos locales", true),
		contextBundleEntryV0(5, ContextLayerModuleContextV0, ContextEntryDocRefV0,
			moduleRoot+"/docs/tareas.md", "microtareas locales", true),
	}
}

func phaseContextEntriesV0(request ContextBundleRequestV0, start int) []ContextBundleEntryV0 {
	moduleRoot := "modulos/" + request.TargetModule
	docs := phaseContextDocRefsV0(request.Phase, moduleRoot)
	entries := make([]ContextBundleEntryV0, 0, len(docs))
	for index, doc := range docs {
		entries = append(entries, contextBundleEntryV0(
			start+index,
			ContextLayerPhaseContextV0,
			ContextEntryDocRefV0,
			doc,
			"contexto requerido por fase",
			true,
		))
	}
	return entries
}

func phaseContextDocRefsV0(phase string, moduleRoot string) []string {
	switch phase {
	case "brainstorming_arquitectura", "votacion_y_decision":
		return []string{moduleRoot + "/docs/decisiones.md"}
	case "programacion", "documentacion", "validacion_final":
		return []string{moduleRoot + "/docs/pruebas.md"}
	case "integracion", "revision", "cierre":
		return []string{moduleRoot + "/docs/pruebas.md", moduleRoot + "/docs/decisiones.md"}
	default:
		return nil
	}
}

func refContextEntriesV0(request ContextBundleRequestV0, start int) []ContextBundleEntryV0 {
	var entries []ContextBundleEntryV0
	entries = appendRefEntriesV0(entries, start+len(entries), ContextLayerTaskContextV0, ContextEntryReadRefV0, request.ReadSet, "read-set de la microtarea", false)
	entries = appendRefEntriesV0(entries, start+len(entries), ContextLayerTaskContextV0, ContextEntryWriteRefV0, request.WriteSet, "write-set autorizado", true)
	entries = appendRefEntriesV0(entries, start+len(entries), ContextLayerContractContextV0, ContextEntryContractRefV0, request.ContractRefs, "contrato publico requerido", true)
	entries = appendRefEntriesV0(entries, start+len(entries), ContextLayerContractContextV0, ContextEntryContractRefV0, request.CrossModuleRefs, "ref cruzada publica; no leer internals", false)
	entries = appendRefEntriesV0(entries, start+len(entries), ContextLayerEvidenceContextV0, ContextEntryEvidenceRefV0, request.EvidenceRefs, "evidencia compacta", false)
	return entries
}

func appendRefEntriesV0(
	entries []ContextBundleEntryV0,
	start int,
	layer string,
	kind string,
	refs []string,
	reason string,
	required bool,
) []ContextBundleEntryV0 {
	for index, ref := range refs {
		entries = append(entries, contextBundleEntryV0(start+index, layer, kind, ref, reason, required))
	}
	return entries
}
