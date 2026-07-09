package orquestaopesdirector

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	topicRegistryLifecycleContractV0              = "not_goal_first|goal_first_checkpoint_missing|goal_first_heartbeat_only|goal_first_checkpoint_recorded"
	topicRegistryLifecycleNotGoalFirstV0          = "not_goal_first"
	topicRegistryLifecycleCheckpointMissingV0     = "goal_first_checkpoint_missing"
	topicRegistryLifecycleHeartbeatOnlyV0         = "goal_first_heartbeat_only"
	topicRegistryLifecycleCheckpointRecordedV0    = "goal_first_checkpoint_recorded"
	topicRegistryGoalFirstCheckpointRequiredRefV0 = "goal-first-topic-checkpoint-required"
)

type topicRegistryLifecycleV0 struct {
	GoalFirst      bool
	Status         string
	Reason         string
	CheckpointRefs []string
	HeartbeatRefs  []string
	ObservedAt     string
}

func topicRegistryLifecycleFieldsForRecordV0(
	record OPESCausalArtifactRecordV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	lifecycle := topicRegistryLifecycleForRecordV0(record)
	fields := []orquestadomainwork.DomainWorkFieldV0{
		{Name: "goal_first_lifecycle_status", Value: lifecycle.Status},
		{Name: "goal_first_lifecycle_reason", Value: lifecycle.Reason},
		{Name: "goal_first_lifecycle_contract", Value: topicRegistryLifecycleContractV0},
		{Name: "goal_first_checkpoint_refs", Values: lifecycle.CheckpointRefs},
		{Name: "goal_first_heartbeat_refs", Values: lifecycle.HeartbeatRefs},
	}
	if lifecycle.ObservedAt != "" {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{Name: "goal_first_observed_at", Value: lifecycle.ObservedAt})
	}
	return fields
}

func topicRegistryLifecycleForRecordV0(record OPESCausalArtifactRecordV0) topicRegistryLifecycleV0 {
	lifecycle := topicRegistryLifecycleV0{
		GoalFirst:      topicRegistryRecordLooksGoalFirstV0(record),
		CheckpointRefs: topicRegistryCheckpointRefsForRecordV0(record),
		HeartbeatRefs:  topicRegistryHeartbeatRefsForRecordV0(record),
		ObservedAt: firstNonEmptyV0(
			fieldStringV0(record.PayloadFields, "goal_first_observed_at", "last_observed_at", "observed_at"),
			record.RecordedAt,
		),
	}
	switch {
	case !lifecycle.GoalFirst:
		lifecycle.Status = topicRegistryLifecycleNotGoalFirstV0
		lifecycle.Reason = "domain_record_not_goal_first"
	case len(lifecycle.CheckpointRefs) > 0:
		lifecycle.Status = topicRegistryLifecycleCheckpointRecordedV0
		lifecycle.Reason = "durable_checkpoint_refs_present"
	case len(lifecycle.HeartbeatRefs) > 0:
		lifecycle.Status = topicRegistryLifecycleHeartbeatOnlyV0
		lifecycle.Reason = "heartbeat_without_terminal_checkpoint"
	default:
		lifecycle.Status = topicRegistryLifecycleCheckpointMissingV0
		lifecycle.Reason = "durable_checkpoint_required"
	}
	return lifecycle
}

func topicRegistryLifecyclePendingRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	if !topicRegistryGoalFirstLifecycleNeedsTerminalCheckpointV0(record) {
		return nil
	}
	lifecycle := topicRegistryLifecycleForRecordV0(record)
	if !lifecycle.GoalFirst || len(lifecycle.CheckpointRefs) > 0 {
		return nil
	}
	return []string{topicRegistryGoalFirstCheckpointRequiredRefV0}
}

func topicRegistryGoalFirstLifecycleNeedsTerminalCheckpointV0(record OPESCausalArtifactRecordV0) bool {
	if opesDirectorIsFinalPackageArtifactTypeV0(record.ArtifactType) {
		return record.CompleteJob && topicRegistryFinalPackageHasClosureEvidenceV0(record)
	}
	return topicRegistryTextSettlementCandidateV0(record)
}

func topicRegistryRecordLooksGoalFirstV0(record OPESCausalArtifactRecordV0) bool {
	values := []string{
		fieldStringV0(record.PayloadFields, "director_execution_mode", "execution_mode", "orquesta_execution_mode"),
		fieldStringV0(record.PayloadFields, "goal_first", "goal_first_status", "goal_runtime"),
		fieldStringV0(record.PayloadFields, "external_goal_ref", "goal_ref", "thread_ref"),
		record.RunRef,
		record.TaskRef,
		record.DeliveryRef,
		record.ArtifactRef,
		record.ReceiptRef,
	}
	values = append(values, record.EvidenceRefs...)
	values = append(values, record.PayloadRefs...)
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if normalized == "goal_first" || normalized == "goal-first" {
			return true
		}
		if strings.Contains(normalized, "goal_first") || strings.Contains(normalized, "goal-first") {
			return true
		}
	}
	return false
}

func topicRegistryCheckpointRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	refs := fieldStringsV0(
		record.PayloadFields,
		"checkpoint_ref",
		"checkpoint_refs",
		"goal_checkpoint_ref",
		"goal_checkpoint_refs",
		"goal_first_checkpoint_ref",
		"goal_first_checkpoint_refs",
		"topic_checkpoint_ref",
		"topic_checkpoint_refs",
		"materialized_checkpoint_ref",
		"materialized_checkpoint_refs",
		"orquesta_goal_result_ref",
		"orquesta_goal_result_refs",
		"goal_result_ref",
		"goal_result_refs",
	)
	refs = append(refs, topicRegistryRefsContainingAnyV0(append(append([]string(nil), record.EvidenceRefs...), record.PayloadRefs...),
		"checkpoint",
		"goal-result",
		"goal_result",
		"orquesta_goal_result",
	)...)
	return compactStringsV0(refs)
}

func topicRegistryHeartbeatRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	refs := fieldStringsV0(
		record.PayloadFields,
		"heartbeat_ref",
		"heartbeat_refs",
		"goal_heartbeat_ref",
		"goal_heartbeat_refs",
		"goal_first_heartbeat_ref",
		"goal_first_heartbeat_refs",
		"topic_heartbeat_ref",
		"topic_heartbeat_refs",
		"progress_ref",
		"progress_refs",
	)
	refs = append(refs, topicRegistryRefsContainingAnyV0(append(append([]string(nil), record.EvidenceRefs...), record.PayloadRefs...),
		"heartbeat",
		"progress",
	)...)
	return compactStringsV0(refs)
}

func topicRegistryRefsContainingAnyV0(values []string, needles ...string) []string {
	var refs []string
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		for _, needle := range needles {
			if needle = strings.ToLower(strings.TrimSpace(needle)); needle != "" && strings.Contains(normalized, needle) {
				refs = append(refs, value)
				break
			}
		}
	}
	return compactStringsV0(refs)
}
