package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

const codexStackResidentCouncilArchitectureVoteArtifactTypeV0 = "architecture_vote.v0"

type codexStackReceiptDecisionCouncilVoteSourceV0 struct {
	Store orquestaruntimecodexdelivery.CodexReceiptDescriptorStorePortV0
}

func (source codexStackReceiptDecisionCouncilVoteSourceV0) BuildDecisionCouncilVotesV0(
	ctx context.Context,
	request DecisionCouncilVoteBuildRequestV0,
) (DecisionCouncilVoteBuildResultV0, error) {
	result := DecisionCouncilVoteBuildResultV0{}
	if source.Store == nil {
		result.PendingEvidenceRefs = append(result.PendingEvidenceRefs, "evidence-ref-codex-stack-resident-council-vote-store-missing")
		return result, nil
	}
	descriptors, err := source.Store.ListCodexReceiptDescriptorsV0(ctx, orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
		RunID: strings.TrimSpace(request.Run.RunID),
	})
	if err != nil {
		return result, err
	}
	voteTaskRefs := codexStackResidentCouncilVoteTaskRefSetV0(request.VoteTasks)
	for _, descriptor := range descriptors {
		ack, issues := orquestaruntimecodex.ReadAndValidateCodexDeliveryAckFileV0(descriptor.AckPath, descriptor.Spec)
		taskRef := firstNonEmptyQueuedSourceV0(
			ack.TaskRef,
			descriptor.Spec.AgentPacket.Task.TaskRef,
		)
		if !voteTaskRefs[taskRef] {
			continue
		}
		if len(issues) > 0 {
			result.PendingEvidenceRefs = append(result.PendingEvidenceRefs,
				"evidence-ref-codex-stack-resident-council-vote-ack-pending-"+codexStackOperationalClosureSafeRefV0(taskRef),
			)
			continue
		}
		vote, evidenceRefs, ok := codexStackResidentCouncilVoteFromAckFilesV0(descriptor, ack, request)
		result.EvidenceRefs = append(result.EvidenceRefs, evidenceRefs...)
		if !ok {
			result.PendingEvidenceRefs = append(result.PendingEvidenceRefs,
				"evidence-ref-codex-stack-resident-council-vote-artifact-pending-"+codexStackOperationalClosureSafeRefV0(taskRef),
			)
			continue
		}
		result.Votes = append(result.Votes, vote)
	}
	result.EvidenceRefs = compactStringsV0(result.EvidenceRefs)
	result.PendingEvidenceRefs = compactStringsV0(result.PendingEvidenceRefs)
	return result, nil
}

func codexStackResidentCouncilVoteTaskRefSetV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) map[string]bool {
	out := map[string]bool{}
	for _, task := range tasks {
		taskRef := strings.TrimSpace(task.TaskID)
		if taskRef != "" {
			out[taskRef] = true
		}
	}
	return out
}

func codexStackResidentCouncilVoteFromAckFilesV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
	request DecisionCouncilVoteBuildRequestV0,
) (orquestadecisioncouncil.CouncilVoteV0, []string, bool) {
	evidenceRefs := []string{
		"evidence-ref-codex-stack-resident-council-vote-ack-" + codexStackOperationalClosureSafeRefV0(ack.AckRef),
	}
	for _, file := range ack.Files {
		fileRef := strings.TrimSpace(string(file))
		if fileRef == "" {
			continue
		}
		path, ok := safeDomainWorkDeliveryFilePathV0(descriptor.ProjectWorkDir, fileRef)
		if !ok {
			evidenceRefs = append(evidenceRefs, "evidence-ref-codex-stack-resident-council-vote-invalid-path")
			continue
		}
		intake, err := readDomainWorkDeliveryArtifactFileV0(
			path,
			fileRef,
			codexStackResidentCouncilArchitectureVoteArtifactTypeV0,
		)
		if err != nil || intake.ContentKind != "json" {
			continue
		}
		vote, ok := codexStackResidentCouncilVoteFromJSONV0(intake.Body, request)
		if !ok {
			continue
		}
		if vote.TaskRef == "" {
			vote.TaskRef = strings.TrimSpace(ack.TaskRef)
		}
		vote.EvidenceRefs = compactStringsV0(append(vote.EvidenceRefs,
			"evidence-ref-codex-stack-resident-council-vote-artifact-"+codexStackOperationalClosureSafeRefV0(fileRef),
		))
		return vote, evidenceRefs, true
	}
	return orquestadecisioncouncil.CouncilVoteV0{}, evidenceRefs, false
}

func codexStackResidentCouncilVoteFromJSONV0(
	body string,
	request DecisionCouncilVoteBuildRequestV0,
) (orquestadecisioncouncil.CouncilVoteV0, bool) {
	body = strings.TrimSpace(body)
	if body == "" {
		return orquestadecisioncouncil.CouncilVoteV0{}, false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return orquestadecisioncouncil.CouncilVoteV0{}, false
	}
	payload, ok := codexStackResidentCouncilVoteEnvelopePayloadV0(raw)
	if !ok {
		return orquestadecisioncouncil.CouncilVoteV0{}, false
	}
	raw = payload
	vote := orquestadecisioncouncil.CouncilVoteV0{
		TaskRef:      codexStackResidentCouncilVoteJSONStringV0(raw, "task_ref", "taskRef"),
		VoteRef:      codexStackResidentCouncilVoteJSONStringV0(raw, "vote_ref", "voteRef"),
		VoterRef:     codexStackResidentCouncilVoteJSONStringV0(raw, "voter_ref", "voterRef"),
		FamilyRef:    codexStackResidentCouncilVoteJSONStringV0(raw, "family_ref", "familyRef"),
		OptionRef:    codexStackResidentCouncilVoteJSONStringV0(raw, "option_ref", "optionRef", "selected_option_ref", "selectedOptionRef"),
		Position:     codexStackResidentCouncilVotePositionV0(codexStackResidentCouncilVoteJSONStringV0(raw, "position", "vote", "decision")),
		EvidenceRefs: codexStackResidentCouncilVoteJSONStringListV0(raw, "evidence_refs", "evidenceRefs"),
	}
	if vote.VoteRef == "" {
		vote.VoteRef = codexStackResidentCouncilVoteJSONStringV0(raw, "ref", "id")
	}
	if vote.TaskRef == "" || vote.OptionRef == "" || vote.Position == "" {
		return orquestadecisioncouncil.CouncilVoteV0{}, false
	}
	if len(request.OptionRefs) > 0 && !stringInSetV0(request.OptionRefs, vote.OptionRef) {
		return orquestadecisioncouncil.CouncilVoteV0{}, false
	}
	return vote, true
}

func codexStackResidentCouncilVoteEnvelopePayloadV0(
	raw map[string]json.RawMessage,
) (map[string]json.RawMessage, bool) {
	artifactType := codexStackResidentCouncilVoteJSONStringV0(raw, "artifact_type", "artifactType")
	if artifactType != "" && !codexStackResidentCouncilVoteArtifactTypeAcceptedV0(artifactType) {
		return nil, false
	}
	for _, key := range []string{"payload_json", "payloadJson", "payload"} {
		data, ok := raw[key]
		if !ok {
			continue
		}
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(data, &payload); err == nil && len(payload) > 0 {
			return payload, true
		}
	}
	return raw, true
}

func codexStackResidentCouncilVoteArtifactTypeAcceptedV0(value string) bool {
	switch strings.TrimSpace(value) {
	case "architecture_vote.v0", "architecture_vote", "decision_council_vote.v0", "decision_council_vote":
		return true
	default:
		return false
	}
}

func codexStackResidentCouncilVoteJSONStringV0(
	raw map[string]json.RawMessage,
	keys ...string,
) string {
	for _, key := range keys {
		data, ok := raw[key]
		if !ok {
			continue
		}
		var value string
		if err := json.Unmarshal(data, &value); err == nil {
			if strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
		var number json.Number
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.UseNumber()
		if err := decoder.Decode(&number); err == nil {
			return strings.TrimSpace(number.String())
		}
	}
	return ""
}

func codexStackResidentCouncilVoteJSONStringListV0(
	raw map[string]json.RawMessage,
	keys ...string,
) []string {
	for _, key := range keys {
		data, ok := raw[key]
		if !ok {
			continue
		}
		var values []string
		if err := json.Unmarshal(data, &values); err == nil {
			return compactStringsV0(values)
		}
		var objects []map[string]any
		if err := json.Unmarshal(data, &objects); err == nil {
			out := make([]string, 0, len(objects))
			for _, item := range objects {
				for _, field := range []string{"ref", "path", "name"} {
					if value, ok := item[field].(string); ok && strings.TrimSpace(value) != "" {
						out = append(out, strings.TrimSpace(value))
						break
					}
				}
			}
			return compactStringsV0(out)
		}
	}
	return nil
}

func codexStackResidentCouncilVotePositionV0(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case orquestadecisioncouncil.CouncilVoteApproveV0, "approved", "accept", "accepted", "yes", "si", "sí":
		return orquestadecisioncouncil.CouncilVoteApproveV0
	case orquestadecisioncouncil.CouncilVoteRejectV0, "rejected", "deny", "denied", "no":
		return orquestadecisioncouncil.CouncilVoteRejectV0
	case orquestadecisioncouncil.CouncilVoteBlockV0, "blocked":
		return orquestadecisioncouncil.CouncilVoteBlockV0
	case orquestadecisioncouncil.CouncilVoteAbstainV0, "abstained":
		return orquestadecisioncouncil.CouncilVoteAbstainV0
	default:
		return ""
	}
}

func codexStackDecisionCouncilConfigWithDefaultsV0(config ConfigV0) DecisionCouncilConfigV0 {
	decisionCouncil := config.DecisionCouncil
	if decisionCouncil.VoteSource == nil && config.Stores.ReceiptStore != nil {
		decisionCouncil.VoteSource = codexStackReceiptDecisionCouncilVoteSourceV0{
			Store: config.Stores.ReceiptStore,
		}
	}
	return decisionCouncil
}

var _ DecisionCouncilVoteSourcePortV0 = codexStackReceiptDecisionCouncilVoteSourceV0{}
