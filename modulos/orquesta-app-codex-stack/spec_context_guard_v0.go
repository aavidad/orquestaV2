package orquestaappcodexstack

import (
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	externalContextTruncatedRequiredTestV0 = "validar contexto externo no truncado o justificar materializacion externa"
	externalContextTruncatedDoneCriteriaV0 = "Si agent_packet.context tiene entradas required=true y truncated=true, no completar salvo que el trabajo sea resoluble con refs/materializacion externa; en ese caso incluir nota contexto_truncado_resuelto: ..."
)

func taskWithContextGuardV0(
	task orquestaruntime.AgentStartTaskV0,
	bundle orquestacontext.ContextMaterializedBundleV0,
) orquestaruntime.AgentStartTaskV0 {
	if !contextBundleHasRequiredTruncatedEntryV0(bundle) {
		return task
	}
	task.RequiredTests = compactStringsV0(append(
		task.RequiredTests,
		externalContextTruncatedRequiredTestV0,
	))
	task.DoneCriteria = compactStringsV0(append(
		task.DoneCriteria,
		externalContextTruncatedDoneCriteriaV0,
	))
	return task
}

func contextBundleHasRequiredTruncatedEntryV0(
	bundle orquestacontext.ContextMaterializedBundleV0,
) bool {
	for _, entry := range bundle.Entries {
		if entry.Required && entry.Truncated {
			return true
		}
	}
	return false
}
