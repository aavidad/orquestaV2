package orquestacoreconcurrency

import (
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func BuildWorksetClaimFromWorkflowTaskV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	bundle orquestacontext.ContextBundleV0,
) (WorksetClaimV0, []WorksetClaimIssueV0) {
	task = orquestacoreworkflow.NormalizeWorkflowTaskV0(task)
	readSet, readIssues := scopeRefsFromContextBundleV0(bundle, orquestacontext.ContextEntryReadRefV0)
	writeSet, writeIssues := workflowClaimWriteSetV0(task, bundle)
	claim := WorksetClaimV0{
		SchemaVersion: WorksetClaimSchemaVersionV0,
		ClaimRef:      "claim-" + task.TaskID,
		RunRef:        task.RunID,
		TaskRef:       task.TaskID,
		GroupRef:      workflowClaimGroupRefV0(task, bundle),
		ReadSet:       readSet,
		WriteSet:      writeSet,
		EvidenceRefs:  workflowClaimEvidenceRefsV0(bundle),
	}

	normalized, issues := NormalizeWorksetClaimV0(claim)
	issues = append(issues, readIssues...)
	issues = append(issues, writeIssues...)
	if err := orquestacoreworkflow.ValidateWorkflowTaskV0(task); err != nil {
		issues = append(issues, newWorksetClaimIssueV0(ErrWorksetClaimCampoRequeridoV0, "workflow_task", err.Error()))
	}
	if !bundle.Valid() {
		issues = append(issues, newWorksetClaimIssueV0(ErrWorksetClaimCampoRequeridoV0, "context_bundle", "invalid"))
	}
	return normalized, issues
}

func workflowClaimGroupRefV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	bundle orquestacontext.ContextBundleV0,
) string {
	if bundle.WorkOrderRef != "" {
		return bundle.WorkOrderRef
	}
	if task.PhaseID != "" {
		return "phase-" + string(task.PhaseID)
	}
	return ""
}

func workflowClaimWriteSetV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	bundle orquestacontext.ContextBundleV0,
) ([]ScopeRefV0, []WorksetClaimIssueV0) {
	refs := append([]string{}, task.WriteSet...)
	refs = append(refs, contextBundleEntrySourceRefsV0(bundle, orquestacontext.ContextEntryWriteRefV0)...)
	return NormalizeScopeRefsV0(refs)
}

func scopeRefsFromContextBundleV0(
	bundle orquestacontext.ContextBundleV0,
	kind string,
) ([]ScopeRefV0, []WorksetClaimIssueV0) {
	return NormalizeScopeRefsV0(contextBundleEntrySourceRefsV0(bundle, kind))
}

func workflowClaimEvidenceRefsV0(bundle orquestacontext.ContextBundleV0) []string {
	refs := contextBundleEntrySourceRefsV0(bundle, orquestacontext.ContextEntryEvidenceRefV0)
	if bundle.BundleRef != "" {
		refs = append(refs, bundle.BundleRef)
	}
	return normalizeOpaqueRefsV0(refs)
}

func contextBundleEntrySourceRefsV0(bundle orquestacontext.ContextBundleV0, kind string) []string {
	refs := make([]string, 0, len(bundle.Entries))
	for _, entry := range bundle.Entries {
		if entry.Kind == kind {
			refs = append(refs, entry.SourceRef)
		}
	}
	return refs
}
