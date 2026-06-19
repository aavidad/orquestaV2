package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureEnsureAckRequiredTestEvidenceRefsV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	result orquestacoreworkflow.ReviewResultV0,
) ([]string, error) {
	if source.RequiredTestEvidenceWriter == nil || strings.TrimSpace(request.OccurredAt) == "" {
		return nil, nil
	}
	descriptors := []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0(nil)
	if source.ReceiptStore != nil {
		items, err := source.ReceiptStore.ListCodexReceiptDescriptorsV0(
			ctx,
			orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: request.Run.RunID},
		)
		if err != nil {
			return nil, err
		}
		descriptors = append(descriptors, items...)
	}
	descriptors = append(descriptors, source.codexStackOperationalClosureRuntimeAckDescriptorsV0(request, task, delivery)...)
	for _, descriptor := range descriptors {
		refs, ok, err := source.codexStackOperationalClosureAckEvidenceRefsFromDescriptorV0(
			ctx,
			request,
			task,
			delivery,
			reviewRequest,
			accepted,
			result,
			descriptor,
		)
		if err != nil || ok {
			return refs, err
		}
	}
	return nil, nil
}

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureRuntimeAckDescriptorsV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
) []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	runtimeRoot := strings.TrimSpace(source.RuntimeWorkDir)
	runRef := strings.TrimSpace(request.Run.RunID)
	agentRef := strings.TrimSpace(delivery.AgentRef)
	if agentRef == "" {
		agentRef = orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID)
	}
	if runtimeRoot == "" || runRef == "" || agentRef == "" {
		return nil
	}
	base := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef: "codex-receipt-ref-runtime-fallback-" + codexStackOperationalClosureSafeRefV0(runRef) + "-" + codexStackOperationalClosureSafeRefV0(agentRef),
		RunID:         runRef,
		AgentRef:      agentRef,
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID: agentRef,
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				RequestID: agentRef,
				Task: orquestaruntime.AgentStartTaskV0{
					TaskRef:       task.TaskID,
					RequiredTests: append([]string(nil), task.RequiredTests...),
				},
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
					AckRef: strings.TrimSpace(delivery.DeliveryRef),
				},
			},
		},
	}
	dirs := codexStackAgentRuntimeDetailRuntimeDirsV0(runtimeRoot, base, agentRef)
	out := make([]orquestaruntimecodexdelivery.CodexReceiptDescriptorV0, 0, len(dirs))
	seen := map[string]bool{}
	for _, dir := range dirs {
		ackPath := filepath.Join(dir, orquestaruntimecodex.CodexAgentAckFileNameV0)
		info, err := os.Stat(ackPath)
		if err != nil || info.IsDir() || seen[ackPath] {
			continue
		}
		seen[ackPath] = true
		descriptor := base
		descriptor.AckPath = ackPath
		descriptor.DescriptorRef += "-" + codexStackOperationalClosureSafeRefV0(filepath.Base(filepath.Dir(dir)))
		out = append(out, descriptor)
	}
	return out
}

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureAckEvidenceRefsFromDescriptorV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	result orquestacoreworkflow.ReviewResultV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) ([]string, bool, error) {
	if !codexStackOperationalClosureDescriptorMatchesDeliveryV0(descriptor, delivery, task) {
		return nil, false, nil
	}
	ack, err := orquestaruntimecodex.ReadCodexAgentAckFileV0(descriptor.AckPath)
	if err != nil ||
		!codexStackOperationalClosureAckMatchesAcceptedDeliveryV0(ack, task, delivery, result) {
		return nil, false, nil
	}
	requiredTests := codexStackOperationalClosureCompactRefsV0(task.RequiredTests)
	refs := make([]string, 0, len(requiredTests))
	for _, command := range requiredTests {
		evidenceRef := codexStackOperationalClosureRequiredTestEvidenceRefV0(request, task.TaskID, command, result, accepted)
		if source.codexStackOperationalClosureRequiredTestEvidenceAlreadyValidV0(ctx, request, task.TaskID, command, evidenceRef, result, accepted) {
			refs = append(refs, evidenceRef)
			continue
		}
		evidenceRefs, occurredAt := codexStackOperationalClosureAckTestEvidenceRefsV0(ack, command)
		if len(evidenceRefs) == 0 {
			evidenceRefs = codexStackOperationalClosureAckContextualRefOnlyEvidenceRefsV0(
				descriptor,
				ack,
				command,
				delivery,
				reviewRequest,
				accepted,
				result,
			)
			occurredAt = request.OccurredAt
		}
		if len(evidenceRefs) == 0 {
			return nil, false, nil
		}
		evidence := orquestacionnucleoapp.RequiredTestEvidenceV0{
			SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
			EvidenceRef:       evidenceRef,
			RunRef:            request.Run.RunID,
			TaskRef:           task.TaskID,
			TestCommand:       command,
			Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
			DeliveryRef:       result.DeliveryRef,
			ReviewRequestID:   reviewRequest.ReviewRequestID,
			ReviewResultRef:   result.ReviewResultRef,
			AcceptedReviewRef: accepted.AcceptedReviewRef,
			OccurredAt:        firstNonEmptyQueuedSourceV0(occurredAt, request.OccurredAt),
			EvidenceRefs:      evidenceRefs,
		}
		if err := source.RequiredTestEvidenceWriter.SaveRequiredTestEvidenceV0(ctx, evidence); err != nil {
			return nil, false, err
		}
		refs = append(refs, evidenceRef)
	}
	return refs, len(refs) == len(requiredTests) && len(refs) > 0, nil
}

func codexStackOperationalClosureDescriptorMatchesDeliveryV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	agentRef := codexStackAgentRuntimeDetailDescriptorAgentRefV0(descriptor)
	if strings.TrimSpace(delivery.AgentRef) != "" && agentRef != strings.TrimSpace(delivery.AgentRef) {
		return false
	}
	packet := descriptor.Spec.AgentPacket
	packetTaskRef := strings.TrimSpace(packet.Task.TaskRef)
	if packetTaskRef != "" &&
		packetTaskRef != strings.TrimSpace(task.TaskID) &&
		packetTaskRef != strings.TrimSpace(delivery.TaskID) {
		return false
	}
	if strings.TrimSpace(packet.DeliveryRefs.AckRef) != "" && strings.TrimSpace(packet.DeliveryRefs.AckRef) != strings.TrimSpace(delivery.DeliveryRef) {
		return false
	}
	return true
}

func codexStackOperationalClosureAckMatchesAcceptedDeliveryV0(
	ack orquestaruntimecodex.CodexAgentAckV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
	result orquestacoreworkflow.ReviewResultV0,
) bool {
	ackTaskRef := strings.TrimSpace(ack.TaskRef)
	return strings.EqualFold(strings.TrimSpace(ack.Status), "completed") &&
		strings.TrimSpace(ack.AckRef) == strings.TrimSpace(result.DeliveryRef) &&
		(ackTaskRef == strings.TrimSpace(task.TaskID) || ackTaskRef == strings.TrimSpace(delivery.TaskID))
}

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureRequiredTestEvidenceAlreadyValidV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	taskRef string,
	command string,
	evidenceRef string,
	result orquestacoreworkflow.ReviewResultV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
) bool {
	if source.RequiredTestEvidenceStore == nil {
		return false
	}
	items, err := source.RequiredTestEvidenceStore.LoadRequiredTestEvidenceV0(ctx, request.Run.RunID, []string{evidenceRef})
	if err != nil || len(items) == 0 {
		return false
	}
	item := items[0]
	return item.Status == orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 &&
		strings.TrimSpace(item.TaskRef) == strings.TrimSpace(taskRef) &&
		strings.TrimSpace(item.TestCommand) == strings.TrimSpace(command) &&
		strings.TrimSpace(item.DeliveryRef) == strings.TrimSpace(result.DeliveryRef) &&
		strings.TrimSpace(item.ReviewRequestID) == strings.TrimSpace(result.ReviewRequestID) &&
		strings.TrimSpace(item.ReviewResultRef) == strings.TrimSpace(result.ReviewResultRef) &&
		strings.TrimSpace(item.AcceptedReviewRef) == strings.TrimSpace(accepted.AcceptedReviewRef)
}

func codexStackOperationalClosureAckTestEvidenceRefsV0(
	ack orquestaruntimecodex.CodexAgentAckV0,
	command string,
) ([]string, string) {
	for _, receipt := range ack.TestReceipts {
		if strings.TrimSpace(receipt.Command) != strings.TrimSpace(command) ||
			!strings.EqualFold(strings.TrimSpace(receipt.Status), "passed") ||
			strings.TrimSpace(receipt.SchemaVersion) != orquestaruntimecodex.CodexRequiredTestReceiptSchemaVersionV0 ||
			receipt.ExitCode == nil || *receipt.ExitCode != 0 ||
			strings.TrimSpace(receipt.OccurredAt) == "" ||
			receipt.Sequence <= 0 ||
			receipt.OutputRedacted == nil || !*receipt.OutputRedacted ||
			len(codexStackOperationalClosureCompactRefsV0(receipt.EvidenceRefs)) == 0 {
			continue
		}
		return codexStackOperationalClosureCompactRefsV0(receipt.EvidenceRefs), strings.TrimSpace(receipt.OccurredAt)
	}
	return nil, ""
}

func codexStackOperationalClosureAckContextualRefOnlyEvidenceRefsV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
	command string,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	result orquestacoreworkflow.ReviewResultV0,
) []string {
	if !codexStackOperationalClosureRequiredTestIsContextualRefOnlyV0(command) ||
		!codexStackOperationalClosureContainsV0([]string(ack.Tests), command) ||
		!codexStackOperationalClosureAckHasNotePrefixV0([]string(ack.Notes), "contexto_ref_only_resuelto") {
		return nil
	}
	return codexStackOperationalClosureCompactRefsV0([]string{
		descriptor.DescriptorRef,
		delivery.DeliveryRef,
		reviewRequest.ReviewRequestID,
		result.ReviewResultRef,
		accepted.AcceptedReviewRef,
		codexStackDeterministicRefV0(
			"evidence-ref-codex-context-ref-only-",
			delivery.DeliveryRef,
			result.ReviewResultRef,
			accepted.AcceptedReviewRef,
			command,
		),
	})
}

func codexStackOperationalClosureRequiredTestIsContextualRefOnlyV0(command string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(command)), "ref_only")
}

func codexStackOperationalClosureAckHasNotePrefixV0(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == want ||
			strings.HasPrefix(normalized, want+":") ||
			strings.HasPrefix(normalized, want+" ") {
			return true
		}
	}
	return false
}

func codexStackOperationalClosureRequiredTestEvidenceRefV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	taskRef string,
	command string,
	result orquestacoreworkflow.ReviewResultV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
) string {
	return codexStackDeterministicRefV0("test-evidence-ref-v0-",
		request.Run.RunID,
		taskRef,
		command,
		result.DeliveryRef,
		result.ReviewRequestID,
		result.ReviewResultRef,
		accepted.AcceptedReviewRef,
	)
}
