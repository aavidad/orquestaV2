package orquestaruntimecodex

import (
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func codexPacketHasRequiredTruncatedContextV0(
	packet orquestaruntime.AgentStartPacketV0,
) bool {
	for _, entry := range packet.Context.Entries {
		if entry.Required && entry.Truncated {
			return true
		}
	}
	return false
}

func codexPacketHasRequiredRefOnlyContextV0(
	packet orquestaruntime.AgentStartPacketV0,
) bool {
	return orquestacontext.ContextBundleHasRequiredRefOnlyV0(packet.Context)
}

func codexPacketRequiresRefOnlyAckEvidenceV0(
	packet orquestaruntime.AgentStartPacketV0,
) bool {
	for _, entry := range orquestacontext.ContextRequiredRefOnlyEntriesV0(packet.Context) {
		if entry.RefOnlyReason != orquestacontext.ContextRefOnlyReasonByDesignV0 {
			return true
		}
		switch entry.RequiredRefAction {
		case orquestacontext.ContextRequiredRefActionReadLocalV0,
			orquestacontext.ContextRequiredRefActionAskDirectorV0:
			return true
		}
	}
	return false
}
