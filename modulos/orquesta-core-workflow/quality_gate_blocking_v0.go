package orquestacoreworkflow

import "strings"

func PendingBlockingQualityGateRefsForSubjectV0(run OrchestrationRunV0, subjectRef string) []string {
	subjectRef = strings.TrimSpace(subjectRef)
	if subjectRef == "" {
		return nil
	}
	pending := []string{}
	for _, projection := range run.QualityGates {
		parts, ok := qualityGateProjectionPartsFromRefV0(projection)
		if !ok || parts.SubjectRef != subjectRef {
			continue
		}
		if parts.Decision == QualityGateDecisionAcceptedV0 {
			pending = nil
			continue
		}
		if qualityGateDecisionBlocksSubjectV0(parts.Decision) {
			pending = appendUniqueCompactRefV0(pending, parts.GateRef)
		}
	}
	return pending
}

func QualityGateSubjectHasPendingBlockersV0(run OrchestrationRunV0, subjectRef string) bool {
	return len(PendingBlockingQualityGateRefsForSubjectV0(run, subjectRef)) > 0
}

func qualityGateDecisionBlocksSubjectV0(decision QualityGateDecisionV0) bool {
	switch decision {
	case QualityGateDecisionBlockedV0,
		QualityGateDecisionReworkRequiredV0,
		QualityGateDecisionAskDirectorV0:
		return true
	default:
		return false
	}
}
