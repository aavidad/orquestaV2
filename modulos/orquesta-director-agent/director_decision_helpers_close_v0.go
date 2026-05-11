package orquestadirectoragent

import "strings"

func normalizeDirectorAgentCloseTaskV0(
	payload DirectorAgentCloseTaskCommandV0,
) DirectorAgentCloseTaskCommandV0 {
	return DirectorAgentCloseTaskCommandV0{
		TaskID:            strings.TrimSpace(payload.TaskID),
		PhaseID:           strings.TrimSpace(payload.PhaseID),
		DeliveryRef:       strings.TrimSpace(payload.DeliveryRef),
		AcceptedReviewRef: strings.TrimSpace(payload.AcceptedReviewRef),
		Summary:           strings.TrimSpace(payload.Summary),
		EvidenceRefs:      compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}

func normalizeDirectorAgentFinalValidationV0(
	payload DirectorAgentFinalValidationCommandV0,
) DirectorAgentFinalValidationCommandV0 {
	return DirectorAgentFinalValidationCommandV0{
		ValidationRef:       strings.TrimSpace(payload.ValidationRef),
		PhaseID:             strings.TrimSpace(payload.PhaseID),
		ClosedTaskRef:       strings.TrimSpace(payload.ClosedTaskRef),
		Summary:             strings.TrimSpace(payload.Summary),
		RequestKind:         strings.TrimSpace(payload.RequestKind),
		ExecutionMode:       strings.TrimSpace(payload.ExecutionMode),
		MinimumDeliverables: compactDirectorAgentStringsV0(payload.MinimumDeliverables),
		EvidenceRefs:        compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}

func normalizeDirectorAgentCloseRunV0(
	payload DirectorAgentCloseRunCommandV0,
) DirectorAgentCloseRunCommandV0 {
	return DirectorAgentCloseRunCommandV0{
		ClosureRef:          strings.TrimSpace(payload.ClosureRef),
		PhaseID:             strings.TrimSpace(payload.PhaseID),
		ValidationRef:       strings.TrimSpace(payload.ValidationRef),
		Summary:             strings.TrimSpace(payload.Summary),
		RequestKind:         strings.TrimSpace(payload.RequestKind),
		ExecutionMode:       strings.TrimSpace(payload.ExecutionMode),
		MinimumDeliverables: compactDirectorAgentStringsV0(payload.MinimumDeliverables),
		EvidenceRefs:        compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}
