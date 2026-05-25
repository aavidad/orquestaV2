package orquestacontext

func ContextRequiredRefOnlyEntriesV0(
	bundle ContextMaterializedBundleV0,
) []ContextMaterializedEntryV0 {
	entries := make([]ContextMaterializedEntryV0, 0)
	for _, entry := range bundle.Entries {
		if entry.Required && entry.Mode == ContextMaterializationModeRefOnlyV0 {
			entries = append(entries, entry)
		}
	}
	return entries
}

func ContextBundleHasRequiredRefOnlyV0(bundle ContextMaterializedBundleV0) bool {
	return len(ContextRequiredRefOnlyEntriesV0(bundle)) > 0
}
