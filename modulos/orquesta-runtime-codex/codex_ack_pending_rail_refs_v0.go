package orquestaruntimecodex

func CodexAgentAckPendingRailEvidenceRefsV0(ack CodexAgentAckV0) []string {
	// Pending rail categories are kept as historical/advisory helpers, but they
	// must not enter the runtime evidence path. Generic words such as provider,
	// model, runtime, token or prompt caused false blockers in real OPES work.
	return nil
}
