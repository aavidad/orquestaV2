package orquestaopesdirector

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	topicRegistrySettlementContractV0    = "not_settled|settled_text|settled_final|needs_rework|blocked"
	topicRegistrySettlementNotSettledV0  = "not_settled"
	topicRegistrySettlementTextV0        = "settled_text"
	topicRegistrySettlementFinalV0       = "settled_final"
	topicRegistrySettlementNeedsReworkV0 = "needs_rework"
	topicRegistrySettlementBlockedV0     = "blocked"
)

type topicRegistrySettlementV0 struct {
	Status        string
	Scope         string
	Reason        string
	Refs          []string
	NextWorkKinds []string
}

func topicRegistrySettlementFieldsForRecordV0(
	record OPESCausalArtifactRecordV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	settlement := topicRegistrySettlementForRecordV0(record)
	fields := []orquestadomainwork.DomainWorkFieldV0{
		{Name: "settlement_status", Value: settlement.Status},
		{Name: "settlement_scope", Value: settlement.Scope},
		{Name: "settlement_reason", Value: settlement.Reason},
		{Name: "settlement_contract", Value: topicRegistrySettlementContractV0},
		{Name: "settlement_refs", Values: settlement.Refs},
	}
	if len(settlement.NextWorkKinds) > 0 {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "next_required_work_kinds",
			Values: settlement.NextWorkKinds,
		})
	}
	if settlement.Status == topicRegistrySettlementTextV0 ||
		settlement.Status == topicRegistrySettlementFinalV0 {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "settled_refs",
			Values: settlement.Refs,
		})
	}
	return fields
}

func topicRegistrySettlementForRecordV0(record OPESCausalArtifactRecordV0) topicRegistrySettlementV0 {
	baseRefs := compactStringsV0([]string{record.ArtifactRef, record.ReceiptRef})
	qualityResult, qualityOK := topicRegistryQualityResultForRecordV0(record)
	qualityPendingRefs := topicRegistryQualityPendingRefsForRecordV0(record)
	if len(qualityPendingRefs) > 0 {
		return topicRegistrySettlementV0{
			Status:        topicRegistrySettlementNeedsReworkV0,
			Scope:         "topic_quality",
			Reason:        "topic_quality_contract_failed",
			Refs:          compactStringsV0(append(append(baseRefs, qualityPendingRefs...), qualityResult.EvidenceRefs...)),
			NextWorkKinds: []string{"review_director_consolidation"},
		}
	}
	operationalStatus := topicRegistryOperationalStatusForRecordV0(record)
	switch operationalStatus {
	case "blocked":
		return topicRegistrySettlementV0{
			Status: topicRegistrySettlementBlockedV0,
			Scope:  "topic",
			Reason: "operational_status_blocked",
			Refs:   baseRefs,
		}
	case "needs_rework":
		return topicRegistrySettlementV0{
			Status:        topicRegistrySettlementNeedsReworkV0,
			Scope:         "topic",
			Reason:        "operational_status_needs_rework",
			Refs:          baseRefs,
			NextWorkKinds: []string{"review_director_consolidation"},
		}
	}
	if lifecyclePendingRefs := topicRegistryLifecyclePendingRefsForRecordV0(record); len(lifecyclePendingRefs) > 0 {
		lifecycle := topicRegistryLifecycleForRecordV0(record)
		return topicRegistrySettlementV0{
			Status:        topicRegistrySettlementNotSettledV0,
			Scope:         "goal_first_lifecycle",
			Reason:        "goal_first_checkpoint_required",
			Refs:          compactStringsV0(append(append(baseRefs, lifecyclePendingRefs...), lifecycle.HeartbeatRefs...)),
			NextWorkKinds: []string{"review_director_consolidation"},
		}
	}
	pendingRefs := topicRegistryPendingRefsForRecordV0(record)
	if len(pendingRefs) > 0 {
		return topicRegistrySettlementV0{
			Status: topicRegistrySettlementNotSettledV0,
			Scope:  "pending_followups",
			Reason: "pending_refs_open",
			Refs:   compactStringsV0(append(baseRefs, pendingRefs...)),
		}
	}
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 &&
		record.CompleteJob &&
		topicRegistryFinalPackageHasClosureEvidenceV0(record) {
		return topicRegistrySettlementV0{
			Status: topicRegistrySettlementFinalV0,
			Scope:  "final_package",
			Reason: "final_package_closure_evidence_complete",
			Refs:   compactStringsV0(append(baseRefs, topicRegistryFinalPackageEvidenceRefsV0(record)...)),
		}
	}
	if topicRegistryTextSettlementCandidateV0(record) {
		if qualityOK && qualityResult.Status == OPESTopicQualityStatusCompleteV0 {
			return topicRegistrySettlementV0{
				Status:        topicRegistrySettlementTextV0,
				Scope:         "topic_text",
				Reason:        "topic_quality_contract_passed",
				Refs:          compactStringsV0(append(baseRefs, qualityResult.EvidenceRefs...)),
				NextWorkKinds: topicRegistryNextWorkKindsAfterTextSettledV0(),
			}
		}
		return topicRegistrySettlementV0{
			Status: topicRegistrySettlementNotSettledV0,
			Scope:  "topic_text",
			Reason: "topic_quality_contract_required",
			Refs:   baseRefs,
		}
	}
	if operationalStatus == "waiting" {
		return topicRegistrySettlementV0{
			Status: topicRegistrySettlementNotSettledV0,
			Scope:  "topic",
			Reason: "operational_status_waiting",
			Refs:   baseRefs,
		}
	}
	return topicRegistrySettlementV0{
		Status: topicRegistrySettlementNotSettledV0,
		Scope:  "artifact_phase",
		Reason: "artifact_phase_not_terminal",
		Refs:   baseRefs,
	}
}

func topicRegistryTextSettlementCandidateV0(record OPESCausalArtifactRecordV0) bool {
	switch record.ArtifactType {
	case orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		orquestadomainwork.DomainWorkArtifactTypeTopicExpansionPackageV0,
		orquestadomainwork.DomainWorkArtifactTypeAssembledTopicV0:
		return true
	}
	switch strings.TrimSpace(fieldStringV0(record.PayloadFields, "source_work_kind", "work_kind")) {
	case "draft_content_block",
		"expand_topic_from_summary",
		"review_director_consolidation",
		"validate_topic",
		"assemble_topic":
		return true
	default:
		return false
	}
}

func topicRegistryNextWorkKindsAfterTextSettledV0() []string {
	return []string{
		"generate_visual_asset",
		"generate_question_bank",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"review_director_consolidation",
		"assemble_topic",
		"generate_audio_asset",
		"generate_tutor_assets",
		"generate_html_site",
		"finalize_temario_package",
	}
}
