package orquestadirectoroperativo

import "strings"

func DefaultOperationalDirectorRepairPolicyV0() OperationalDirectorRepairPolicyV0 {
	return OperationalDirectorRepairPolicyV0{
		PreferRepair: true,
		RepairActions: []OperationalDirectorRepairActionV0{
			OperationalDirectorRepairNormalizeV0,
			OperationalDirectorRepairRequestCorrectionV0,
			OperationalDirectorRepairDelegateReviewV0,
			OperationalDirectorRepairSequenceFollowupV0,
			OperationalDirectorRepairPostponeV0,
		},
		HardRejectTriggers: []string{
			"security",
			"causality_broken",
			"impossible_refs",
			"unauthorized_external_effect",
		},
	}
}

func DecideOperationalDirectorRepairPolicyV0(
	request OperationalDirectorRepairRequestV0,
) OperationalDirectorRepairDecisionV0 {
	request = normalizeOperationalDirectorRepairRequestV0(request)
	if issues := operationalDirectorHardRejectIssuesV0(request); len(issues) > 0 {
		return OperationalDirectorRepairDecisionV0{
			Action:         OperationalDirectorRepairHardRejectV0,
			StepStatus:     OperationalDirectorStepBlockedV0,
			PreserveOutput: false,
			Cause:          compactOperationalDirectorRepairCauseV0(request.Cause, issues[0].Code),
			Issues:         issues,
		}
	}
	if len(request.NormalizableAliases) > 0 {
		return repairDecisionV0(OperationalDirectorRepairNormalizeV0, request, "normalizar salida razonable")
	}
	if request.NeedsCorrection || len(request.MissingEvidence) > 0 {
		return repairDecisionV0(OperationalDirectorRepairRequestCorrectionV0, request, "pedir correccion dirigida")
	}
	if request.NeedsReview || request.Ambiguous {
		return repairDecisionV0(OperationalDirectorRepairDelegateReviewV0, request, "delegar revision de ambiguedad")
	}
	if request.NeedsSequencedStep {
		return repairDecisionV0(OperationalDirectorRepairSequenceFollowupV0, request, "secuenciar paso posterior")
	}
	if len(request.MissingContext) > 0 {
		decision := repairDecisionV0(OperationalDirectorRepairPostponeV0, request, "posponer hasta contexto suficiente")
		decision.StepStatus = OperationalDirectorStepBlockedV0
		return decision
	}
	if request.Reasonable {
		return repairDecisionV0(OperationalDirectorRepairRequestCorrectionV0, request, "conservar trabajo aprovechable")
	}
	return OperationalDirectorRepairDecisionV0{
		Action:         OperationalDirectorRepairNoActionV0,
		StepStatus:     OperationalDirectorStepAcceptedV0,
		PreserveOutput: true,
		Cause:          compactOperationalDirectorRepairCauseV0(request.Cause, "sin reparacion requerida"),
	}
}

func normalizeOperationalDirectorRepairRequestV0(
	request OperationalDirectorRepairRequestV0,
) OperationalDirectorRepairRequestV0 {
	request.OutputRef = strings.TrimSpace(request.OutputRef)
	request.TaskRef = strings.TrimSpace(request.TaskRef)
	request.Cause = strings.TrimSpace(request.Cause)
	request.NormalizableAliases = compactStringsV0(request.NormalizableAliases)
	request.MissingContext = compactStringsV0(request.MissingContext)
	request.MissingEvidence = compactStringsV0(request.MissingEvidence)
	return request
}

func operationalDirectorHardRejectIssuesV0(
	request OperationalDirectorRepairRequestV0,
) []OperationalDirectorIssueV0 {
	var issues []OperationalDirectorIssueV0
	if request.SafetyRisk {
		issues = append(issues, issueV0("security", "safety_risk", "riesgo de seguridad"))
	}
	if request.CausalityBroken {
		issues = append(issues, issueV0("causality_broken", "causality_broken", "causalidad rota"))
	}
	if request.ImpossibleRefs {
		issues = append(issues, issueV0("impossible_refs", "impossible_refs", "refs imposibles"))
	}
	if request.UnauthorizedExternalEffect {
		issues = append(issues, issueV0("unauthorized_external_effect", "unauthorized_external_effect", "efecto externo no autorizado"))
	}
	return issues
}

func repairDecisionV0(
	action OperationalDirectorRepairActionV0,
	request OperationalDirectorRepairRequestV0,
	fallbackCause string,
) OperationalDirectorRepairDecisionV0 {
	return OperationalDirectorRepairDecisionV0{
		Action:           action,
		StepStatus:       OperationalDirectorStepChangesRequestedV0,
		PreserveOutput:   true,
		RequiresFollowup: true,
		Cause:            compactOperationalDirectorRepairCauseV0(request.Cause, fallbackCause),
	}
}

func compactOperationalDirectorRepairCauseV0(cause string, fallback string) string {
	cause = strings.TrimSpace(cause)
	if cause != "" {
		return cause
	}
	return strings.TrimSpace(fallback)
}
