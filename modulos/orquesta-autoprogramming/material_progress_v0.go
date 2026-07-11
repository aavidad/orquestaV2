package orquestaautoprogramming

import "strings"

func NormalizeMaterialProgressInputV0(input MaterialProgressInputV0) MaterialProgressInputV0 {
	input.Segment.ContextRevisionRef = strings.TrimSpace(input.Segment.ContextRevisionRef)
	input.Segment.EvidenceRefs = compactStringsV0(input.Segment.EvidenceRefs)
	input.Checkpoint.ContextRevisionRef = strings.TrimSpace(input.Checkpoint.ContextRevisionRef)
	input.Checkpoint.EvidenceRefs = compactStringsV0(input.Checkpoint.EvidenceRefs)
	return input
}

func ValidateMaterialProgressV0(input MaterialProgressInputV0) MaterialProgressValidationResultV0 {
	input = NormalizeMaterialProgressInputV0(input)
	issues := materialProgressPolicyIssuesV0(input.Policy)
	issues = append(issues, materialProgressSegmentIssuesV0(input.Segment)...)
	issues = append(issues, materialProgressCheckpointIssuesV0(input.Checkpoint)...)
	if input.Checkpoint.Sequence < input.Segment.StartSequence {
		issues = append(issues, materialProgressIssueV0(
			"checkpoint_before_segment",
			"checkpoint.sequence",
			"checkpoint debe pertenecer al tramo actual",
		))
	}
	if input.Checkpoint.TokensAccumulated < input.Segment.StartTokensAccumulated {
		issues = append(issues, materialProgressIssueV0(
			"checkpoint_tokens_regressed",
			"checkpoint.tokens_accumulated",
			"tokens acumulados no pueden retroceder dentro del tramo",
		))
	}
	if input.Checkpoint.ContextRevisionRef != "" &&
		input.Segment.ContextRevisionRef != "" &&
		input.Checkpoint.ContextRevisionRef != input.Segment.ContextRevisionRef {
		issues = append(issues, materialProgressIssueV0(
			"checkpoint_context_revision_ref_changed",
			"checkpoint.context_revision_ref",
			"el cambio de revision de contexto requiere iniciar un tramo nuevo",
		))
	}
	return MaterialProgressValidationResultV0{
		Accepted: len(issues) == 0,
		Input:    input,
		Issues:   issues,
	}
}

func DecideMaterialProgressV0(input MaterialProgressInputV0) MaterialProgressDecisionV0 {
	validation := ValidateMaterialProgressV0(input)
	if !validation.Accepted {
		return MaterialProgressDecisionV0{
			Action:  MaterialProgressActionInvalidV0,
			Segment: validation.Input.Segment,
			Issues:  validation.Issues,
		}
	}

	input = validation.Input
	if materialProgressRenewsSegmentV0(input.Segment, input.Checkpoint) {
		return MaterialProgressDecisionV0{
			Accepted:           true,
			Action:             MaterialProgressActionContinueV0,
			MaterialProgressed: true,
			Segment: MaterialProgressSegmentV0{
				StartSequence:          input.Checkpoint.Sequence,
				StartTokensAccumulated: input.Checkpoint.TokensAccumulated,
				ReplansUsed:            input.Segment.ReplansUsed,
				ContextRevisionRef:     input.Checkpoint.ContextRevisionRef,
				EvidenceRefs: compactStringsV0(append(
					append([]string{}, input.Segment.EvidenceRefs...),
					input.Checkpoint.EvidenceRefs...,
				)),
			},
		}
	}

	tokensWithoutMaterial := input.Checkpoint.TokensAccumulated - input.Segment.StartTokensAccumulated
	return MaterialProgressDecisionV0{
		Accepted:              true,
		Action:                materialProgressActionV0(input.Policy, input.Segment, tokensWithoutMaterial),
		TokensWithoutMaterial: tokensWithoutMaterial,
		Segment:               input.Segment,
	}
}

func materialProgressPolicyIssuesV0(policy MaterialProgressPolicyV0) []MaterialProgressIssueV0 {
	var issues []MaterialProgressIssueV0
	for _, threshold := range []struct {
		field string
		value int64
	}{
		{"warning_after_tokens", policy.WarningAfterTokens},
		{"replan_required_after_tokens", policy.ReplanRequiredAfterTokens},
		{"hard_stop_required_after_tokens", policy.HardStopRequiredAfterTokens},
	} {
		if threshold.value <= 0 {
			issues = append(issues, materialProgressIssueV0(
				threshold.field+"_invalid",
				"policy."+threshold.field,
				"umbral de tokens positivo requerido",
			))
		}
	}
	if policy.WarningAfterTokens >= policy.ReplanRequiredAfterTokens ||
		policy.ReplanRequiredAfterTokens >= policy.HardStopRequiredAfterTokens {
		issues = append(issues, materialProgressIssueV0(
			"policy_threshold_order_invalid",
			"policy",
			"warning, replan y hard stop deben tener umbrales estrictamente crecientes",
		))
	}
	if policy.MaxReplans < 0 {
		issues = append(issues, materialProgressIssueV0(
			"policy_max_replans_invalid",
			"policy.max_replans",
			"max replans no puede ser negativo",
		))
	}
	return issues
}

func materialProgressSegmentIssuesV0(segment MaterialProgressSegmentV0) []MaterialProgressIssueV0 {
	var issues []MaterialProgressIssueV0
	if segment.StartSequence <= 0 {
		issues = append(issues, materialProgressIssueV0(
			"segment_start_sequence_invalid",
			"segment.start_sequence",
			"secuencia inicial positiva requerida",
		))
	}
	if segment.StartTokensAccumulated < 0 {
		issues = append(issues, materialProgressIssueV0(
			"segment_start_tokens_invalid",
			"segment.start_tokens_accumulated",
			"tokens iniciales no pueden ser negativos",
		))
	}
	if segment.ReplansUsed < 0 {
		issues = append(issues, materialProgressIssueV0(
			"segment_replans_used_invalid",
			"segment.replans_used",
			"replans usados no pueden ser negativos",
		))
	}
	if segment.ContextRevisionRef == "" {
		issues = append(issues, materialProgressIssueV0(
			"segment_context_revision_ref_missing",
			"segment.context_revision_ref",
			"revision de contexto requerida",
		))
	}
	return issues
}

func materialProgressCheckpointIssuesV0(checkpoint MaterialProgressCheckpointV0) []MaterialProgressIssueV0 {
	var issues []MaterialProgressIssueV0
	if checkpoint.Sequence <= 0 {
		issues = append(issues, materialProgressIssueV0(
			"checkpoint_sequence_invalid",
			"checkpoint.sequence",
			"secuencia positiva requerida",
		))
	}
	if checkpoint.TokensAccumulated < 0 {
		issues = append(issues, materialProgressIssueV0(
			"checkpoint_tokens_invalid",
			"checkpoint.tokens_accumulated",
			"tokens acumulados no pueden ser negativos",
		))
	}
	if checkpoint.ContextRevisionRef == "" {
		issues = append(issues, materialProgressIssueV0(
			"checkpoint_context_revision_ref_missing",
			"checkpoint.context_revision_ref",
			"revision de contexto requerida",
		))
	}
	if !materialProgressClassValidV0(checkpoint.MaterialClass) {
		issues = append(issues, materialProgressIssueV0(
			"checkpoint_material_class_invalid",
			"checkpoint.material_class",
			"clase material tipada requerida",
		))
	}
	if materialProgressClassRenewsV0(checkpoint.MaterialClass) && len(checkpoint.EvidenceRefs) == 0 {
		issues = append(issues, materialProgressIssueV0(
			"checkpoint_material_evidence_missing",
			"checkpoint.evidence_refs",
			"evidencia requerida para clase material",
		))
	}
	return issues
}

func materialProgressActionV0(
	policy MaterialProgressPolicyV0,
	segment MaterialProgressSegmentV0,
	tokensWithoutMaterial int64,
) MaterialProgressActionV0 {
	switch {
	case tokensWithoutMaterial >= policy.HardStopRequiredAfterTokens:
		return MaterialProgressActionHardStopRequiredV0
	case tokensWithoutMaterial >= policy.ReplanRequiredAfterTokens:
		if segment.ReplansUsed >= policy.MaxReplans {
			return MaterialProgressActionHardStopRequiredV0
		}
		return MaterialProgressActionReplanRequiredV0
	case tokensWithoutMaterial >= policy.WarningAfterTokens:
		return MaterialProgressActionWarningV0
	default:
		return MaterialProgressActionContinueV0
	}
}

func materialProgressRenewsSegmentV0(
	segment MaterialProgressSegmentV0,
	checkpoint MaterialProgressCheckpointV0,
) bool {
	return materialProgressClassRenewsV0(checkpoint.MaterialClass) &&
		materialProgressHasNewEvidenceV0(segment.EvidenceRefs, checkpoint.EvidenceRefs)
}

func materialProgressHasNewEvidenceV0(existing []string, candidates []string) bool {
	seen := make(map[string]struct{}, len(existing))
	for _, evidenceRef := range existing {
		seen[evidenceRef] = struct{}{}
	}
	for _, evidenceRef := range candidates {
		if _, ok := seen[evidenceRef]; !ok {
			return true
		}
	}
	return false
}

func materialProgressClassValidV0(class MaterialProgressClassV0) bool {
	switch class {
	case MaterialProgressClassDiffV0,
		MaterialProgressClassTestV0,
		MaterialProgressClassResultV0,
		MaterialProgressClassReceiptV0,
		MaterialProgressClassNoneV0:
		return true
	default:
		return false
	}
}

func materialProgressClassRenewsV0(class MaterialProgressClassV0) bool {
	switch class {
	case MaterialProgressClassDiffV0,
		MaterialProgressClassTestV0,
		MaterialProgressClassResultV0,
		MaterialProgressClassReceiptV0:
		return true
	default:
		return false
	}
}

func materialProgressIssueV0(code string, field string, message string) MaterialProgressIssueV0 {
	return MaterialProgressIssueV0{Code: code, Field: field, Message: message}
}
