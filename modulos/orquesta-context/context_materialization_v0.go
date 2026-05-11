package orquestacontext

func MaterializeContextBundleV0(
	bundle ContextBundleV0,
	reader ContextRefReaderV0,
) ContextMaterializedBundleV0 {
	result := ContextMaterializedBundleV0{
		SchemaVersion:        ContextMaterializedBundleSchemaVersionV0,
		BundleRef:            bundle.BundleRef,
		WorkOrderRef:         bundle.WorkOrderRef,
		TargetModule:         bundle.TargetModule,
		DirectorQuestionHint: bundle.Summary.DirectorQuestionPolicy,
	}
	if !bundle.Valid() {
		result.Issues = append(result.Issues,
			contextMaterializationIssueV0(ErrContextMaterializationBundleInvalidoV0, "bundle", "bundle invalido"))
		return result
	}
	for _, entry := range bundle.Entries {
		materialized, issues := materializeContextEntryV0(bundle, entry, reader)
		if len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
			continue
		}
		result.TotalBytes += materialized.Bytes
		if result.TotalBytes > bundle.Limits.MaxTotalBytes {
			result.Issues = append(result.Issues,
				contextMaterializationIssueV0(ErrContextMaterializationTamanoV0, "total_bytes", "limite de bytes excedido"))
			return result
		}
		result.Entries = append(result.Entries, materialized)
	}
	return result
}

func materializeContextEntryV0(
	bundle ContextBundleV0,
	entry ContextBundleEntryV0,
	reader ContextRefReaderV0,
) (ContextMaterializedEntryV0, []ContextMaterializationIssueV0) {
	if entry.Kind == ContextEntryRuleRefV0 {
		return materializeContextRuleEntryV0(entry), nil
	}
	if resolvedRef, ok := contextMaterializationReadableRefV0(bundle, entry); ok {
		return materializeContextReadableEntryV0(entry, resolvedRef, reader)
	}
	return contextMaterializedRefOnlyV0(entry), nil
}

func materializeContextRuleEntryV0(entry ContextBundleEntryV0) ContextMaterializedEntryV0 {
	content, ok := contextCommonRulesContentV0(entry.SourceRef)
	if !ok {
		return contextMaterializedRefOnlyV0(entry)
	}
	return contextMaterializedContentV0(entry, ContextRefContentV0{
		SourceRef: entry.SourceRef,
		Content:   content,
		Bytes:     len(content),
	})
}

func materializeContextReadableEntryV0(
	entry ContextBundleEntryV0,
	resolvedRef string,
	reader ContextRefReaderV0,
) (ContextMaterializedEntryV0, []ContextMaterializationIssueV0) {
	if reader == nil {
		return ContextMaterializedEntryV0{}, []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationReaderRequeridoV0, entry.SourceRef, "reader requerido"),
		}
	}
	content, issues := reader.ReadContextRefV0(resolvedRef, contextMaterializationEntryMaxBytesV0(entry))
	if len(issues) > 0 {
		return ContextMaterializedEntryV0{}, issues
	}
	if contextMaterializedContentHasForbiddenDetailV0(content.Content) {
		return ContextMaterializedEntryV0{}, []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationDetalleProhibidoV0, entry.SourceRef, "detalle prohibido"),
		}
	}
	return contextMaterializedContentV0(entry, content), nil
}
