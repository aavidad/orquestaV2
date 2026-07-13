package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type codexAckRequiredTestRunnerV0 struct {
	Inner          orquestacionnucleoapp.RequiredTestRunnerPortV0
	ReceiptStore   orquestaruntimecodexdelivery.CodexReceiptDescriptorStorePortV0
	EvidenceReader orquestacionnucleoapp.RequiredTestEvidenceReaderPortV0
	EvidenceWriter orquestacionnucleoapp.RequiredTestEvidenceWriterPortV0
}

var _ orquestacionnucleoapp.RequiredTestRunnerPortV0 = codexAckRequiredTestRunnerV0{}

func codexAckRequiredTestRunnerFromConfigV0(
	config ConfigV0,
	inner orquestacionnucleoapp.RequiredTestRunnerPortV0,
) orquestacionnucleoapp.RequiredTestRunnerPortV0 {
	store := requiredTestEvidenceStoreV0(config)
	if config.Stores.ReceiptStore == nil || store == nil {
		return inner
	}
	return codexAckRequiredTestRunnerV0{
		Inner:          inner,
		ReceiptStore:   config.Stores.ReceiptStore,
		EvidenceReader: store,
		EvidenceWriter: store,
	}
}

func (runner codexAckRequiredTestRunnerV0) RunRequiredTestsV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestExecutionResultV0, error) {
	result, complete, err := runner.runFromCodexAckReceiptsV0(ctx, request)
	if err != nil || complete {
		return result, err
	}
	if runner.Inner != nil {
		return runner.Inner.RunRequiredTestsV0(ctx, request)
	}
	return result, nil
}

// Close preserves ownership of resources held by the wrapped runner after
// stack composition replaces the original port with the ACK-aware adapter.
func (runner codexAckRequiredTestRunnerV0) Close() error {
	return closeRequiredTestRunnerV0(runner.Inner)
}

func closeRequiredTestRunnerV0(runner orquestacionnucleoapp.RequiredTestRunnerPortV0) error {
	closer, ok := runner.(interface{ Close() error })
	if !ok || closer == nil {
		return nil
	}
	return closer.Close()
}

func (runner codexAckRequiredTestRunnerV0) runFromCodexAckReceiptsV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestExecutionResultV0, bool, error) {
	request = codexAckRequiredTestNormalizeRequestV0(request)
	if runner.ReceiptStore == nil || runner.EvidenceWriter == nil ||
		request.RunRef == "" || request.TaskRef == "" ||
		request.DeliveryRef == "" || len(request.TestCommands) == 0 {
		return orquestacionnucleoapp.RequiredTestExecutionResultV0{}, false, nil
	}
	descriptors, err := runner.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: request.RunRef},
	)
	if err != nil {
		return orquestacionnucleoapp.RequiredTestExecutionResultV0{}, false, err
	}
	for _, descriptor := range descriptors {
		result, complete, err := runner.resultFromCodexAckDescriptorV0(ctx, request, descriptor)
		if err != nil || complete {
			return result, complete, err
		}
	}
	return orquestacionnucleoapp.RequiredTestExecutionResultV0{}, false, nil
}

func (runner codexAckRequiredTestRunnerV0) resultFromCodexAckDescriptorV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (orquestacionnucleoapp.RequiredTestExecutionResultV0, bool, error) {
	if !codexAckRequiredTestDescriptorMatchesRequestV0(descriptor, request) {
		return orquestacionnucleoapp.RequiredTestExecutionResultV0{}, false, nil
	}
	ack, issues := orquestaruntimecodex.ReadAndValidateStrictCompletedCodexAgentAckFileV0(
		descriptor.AckPath,
		descriptor.Spec,
	)
	if len(issues) > 0 || !codexAckRequiredTestAckMatchesRequestV0(ack, descriptor, request) {
		return orquestacionnucleoapp.RequiredTestExecutionResultV0{}, false, nil
	}
	result := orquestacionnucleoapp.RequiredTestExecutionResultV0{}
	for _, command := range request.TestCommands {
		evidenceRef := codexAckRequiredTestEvidenceRefV0(request, command)
		if runner.codexAckRequiredTestEvidenceAlreadyValidV0(ctx, request, command, evidenceRef) {
			result = codexAckRequiredTestResultWithEvidenceV0(
				result,
				evidenceRef,
				orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
			)
			continue
		}
		evidenceRefs, occurredAt, ok := codexAckRequiredTestReceiptEvidenceRefsV0(ack, command)
		if !ok {
			evidenceRefs, ok = codexAckRequiredTestContextualRefOnlyEvidenceRefsV0(descriptor, ack, request, command)
			occurredAt = request.OccurredAt
		}
		if !ok {
			continue
		}
		evidence := orquestacionnucleoapp.RequiredTestEvidenceV0{
			SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
			EvidenceRef:       evidenceRef,
			RunRef:            request.RunRef,
			TaskRef:           request.TaskRef,
			TestCommand:       command,
			Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
			DeliveryRef:       request.DeliveryRef,
			ReviewRequestID:   request.ReviewRequestID,
			ReviewResultRef:   request.ReviewResultRef,
			AcceptedReviewRef: request.AcceptedReviewRef,
			OccurredAt:        firstNonEmptyQueuedSourceV0(occurredAt, request.OccurredAt),
			EvidenceRefs:      evidenceRefs,
		}
		if err := runner.EvidenceWriter.SaveRequiredTestEvidenceV0(ctx, evidence); err != nil {
			return result, false, err
		}
		result = codexAckRequiredTestResultWithEvidenceV0(result, evidenceRef, evidence.Status)
	}
	complete := len(compactStringsV0(result.EvidenceRefs)) == len(compactStringsV0(request.TestCommands))
	return result, complete, nil
}

func codexAckRequiredTestNormalizeRequestV0(
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) orquestacionnucleoapp.RequiredTestExecutionRequestV0 {
	return orquestacionnucleoapp.RequiredTestExecutionRequestV0{
		RunRef:            strings.TrimSpace(request.RunRef),
		TaskRef:           strings.TrimSpace(request.TaskRef),
		TestCommands:      compactStringsV0(request.TestCommands),
		DeliveryRef:       strings.TrimSpace(request.DeliveryRef),
		ReviewRequestID:   strings.TrimSpace(request.ReviewRequestID),
		ReviewResultRef:   strings.TrimSpace(request.ReviewResultRef),
		AcceptedReviewRef: strings.TrimSpace(request.AcceptedReviewRef),
		OccurredAt:        strings.TrimSpace(request.OccurredAt),
		CorrelationID:     strings.TrimSpace(request.CorrelationID),
		EvidenceRefs:      compactStringsV0(request.EvidenceRefs),
	}
}

func codexAckRequiredTestDescriptorMatchesRequestV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) bool {
	if strings.TrimSpace(descriptor.RunID) != strings.TrimSpace(request.RunRef) {
		return false
	}
	packet := descriptor.Spec.AgentPacket
	if strings.TrimSpace(packet.Task.TaskRef) != "" &&
		strings.TrimSpace(packet.Task.TaskRef) != strings.TrimSpace(request.TaskRef) {
		return false
	}
	if strings.TrimSpace(packet.DeliveryRefs.AckRef) != "" &&
		strings.TrimSpace(packet.DeliveryRefs.AckRef) != strings.TrimSpace(request.DeliveryRef) {
		return false
	}
	return strings.TrimSpace(descriptor.AckPath) != ""
}

func codexAckRequiredTestAckMatchesRequestV0(
	ack orquestaruntimecodex.CodexAgentAckV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) bool {
	if !strings.EqualFold(strings.TrimSpace(ack.Status), "completed") ||
		strings.TrimSpace(ack.AckRef) != strings.TrimSpace(request.DeliveryRef) ||
		strings.TrimSpace(ack.TaskRef) != strings.TrimSpace(request.TaskRef) {
		return false
	}
	if strings.TrimSpace(descriptor.Spec.RequestID) != "" &&
		strings.TrimSpace(ack.RequestID) != strings.TrimSpace(descriptor.Spec.RequestID) {
		return false
	}
	return true
}

func codexAckRequiredTestReceiptEvidenceRefsV0(
	ack orquestaruntimecodex.CodexAgentAckV0,
	command string,
) ([]string, string, bool) {
	command = strings.TrimSpace(command)
	for _, receipt := range ack.TestReceipts {
		if !codexAckRequiredTestReceiptMatchesV0(receipt, command) {
			continue
		}
		return compactStringsV0(receipt.EvidenceRefs), strings.TrimSpace(receipt.OccurredAt), true
	}
	return nil, "", false
}

func codexAckRequiredTestContextualRefOnlyEvidenceRefsV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
	command string,
) ([]string, bool) {
	if !codexStackOperationalClosureRequiredTestIsContextualRefOnlyV0(command) ||
		!orquestaruntimecodex.CodexTestCommandInSetV0([]string(ack.Tests), command) ||
		!codexStackOperationalClosureAckHasNotePrefixV0([]string(ack.Notes), "contexto_ref_only_resuelto") {
		return nil, false
	}
	return compactStringsV0([]string{
		descriptor.DescriptorRef,
		request.DeliveryRef,
		request.ReviewRequestID,
		request.ReviewResultRef,
		request.AcceptedReviewRef,
		codexStackDeterministicRefV0(
			"evidence-ref-codex-context-ref-only-",
			request.DeliveryRef,
			request.ReviewResultRef,
			request.AcceptedReviewRef,
			command,
		),
	}), true
}

func codexAckRequiredTestReceiptMatchesV0(
	receipt orquestaruntimecodex.CodexRequiredTestReceiptV0,
	command string,
) bool {
	return orquestaruntimecodex.CodexSchemaVersionCompatibleV0(
		receipt.SchemaVersion,
		orquestaruntimecodex.CodexRequiredTestReceiptSchemaVersionV0,
	) &&
		orquestaruntimecodex.CodexTestCommandMatchesV0(receipt.Command, command) &&
		strings.EqualFold(strings.TrimSpace(receipt.Status), "passed") &&
		receipt.ExitCode != nil &&
		*receipt.ExitCode == 0 &&
		len(compactStringsV0(receipt.EvidenceRefs)) > 0 &&
		strings.TrimSpace(receipt.OccurredAt) != "" &&
		receipt.Sequence > 0 &&
		receipt.OutputRedacted != nil &&
		*receipt.OutputRedacted
}

func (runner codexAckRequiredTestRunnerV0) codexAckRequiredTestEvidenceAlreadyValidV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
	command string,
	evidenceRef string,
) bool {
	reader := runner.EvidenceReader
	if reader == nil {
		reader, _ = runner.EvidenceWriter.(orquestacionnucleoapp.RequiredTestEvidenceReaderPortV0)
	}
	if reader == nil {
		return false
	}
	items, err := reader.LoadRequiredTestEvidenceV0(ctx, request.RunRef, []string{evidenceRef})
	if err != nil || len(items) == 0 {
		return false
	}
	item := items[0]
	return item.Status == orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 &&
		strings.TrimSpace(item.TaskRef) == request.TaskRef &&
		orquestaruntimecodex.CodexTestCommandMatchesV0(item.TestCommand, command) &&
		strings.TrimSpace(item.DeliveryRef) == request.DeliveryRef &&
		strings.TrimSpace(item.ReviewRequestID) == request.ReviewRequestID &&
		strings.TrimSpace(item.ReviewResultRef) == request.ReviewResultRef &&
		strings.TrimSpace(item.AcceptedReviewRef) == request.AcceptedReviewRef
}

func codexAckRequiredTestResultWithEvidenceV0(
	result orquestacionnucleoapp.RequiredTestExecutionResultV0,
	evidenceRef string,
	status orquestacionnucleoapp.RequiredTestEvidenceStatusV0,
) orquestacionnucleoapp.RequiredTestExecutionResultV0 {
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, evidenceRef))
	if status == orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 {
		result.PassedEvidenceRefs = compactStringsV0(append(result.PassedEvidenceRefs, evidenceRef))
	}
	if status == orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0 {
		result.FailedEvidenceRefs = compactStringsV0(append(result.FailedEvidenceRefs, evidenceRef))
	}
	return result
}

func codexAckRequiredTestEvidenceRefV0(
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
	command string,
) string {
	return codexStackDeterministicRefV0("test-evidence-ref-v0-",
		request.RunRef,
		request.TaskRef,
		command,
		request.DeliveryRef,
		request.ReviewRequestID,
		request.ReviewResultRef,
		request.AcceptedReviewRef,
	)
}
