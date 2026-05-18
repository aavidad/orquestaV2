package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

type CodexReviewGateFileEvidenceProviderPortV0 interface {
	BuildCodexReviewGateFilesV0(
		context.Context,
		CodexReceiptDescriptorV0,
		orquestaruntimecodex.CodexAgentAckV0,
	) ([]orquestaautoprogramming.AutoprogrammingReviewGateFileV0, error)
}

type CodexReviewGateFileEvidenceResultProviderPortV0 interface {
	BuildCodexReviewGateFileEvidenceV0(
		context.Context,
		CodexReceiptDescriptorV0,
		orquestaruntimecodex.CodexAgentAckV0,
	) (CodexReviewGateFileEvidenceV0, error)
}

type CodexReviewGateFileEvidenceV0 struct {
	Files  []orquestaautoprogramming.AutoprogrammingReviewGateFileV0
	Issues []orquestaautoprogramming.AutoprogrammingReviewGateIssueV0
}

type CodexReviewGateObservationSourceV0 struct {
	Store              CodexReceiptDescriptorStorePortV0
	FileEvidence       CodexReviewGateFileEvidenceProviderPortV0
	FileEvidenceResult CodexReviewGateFileEvidenceResultProviderPortV0
	MaxLinesPerFile    int
	FailureStatus      orquestacoreworkflow.ReviewResultStatusV0
	QualityGateRefFn   func(deliveryRef string) string
}

var _ orquestacionnucleoapp.ReviewGateObservationProviderPortV0 = CodexReviewGateObservationSourceV0{}

func (source CodexReviewGateObservationSourceV0) BuildReviewGateObservationsV0(
	ctx context.Context,
	request orquestacionnucleoapp.ReviewGateObservationRequestV0,
) ([]orquestacionnucleoapp.ReviewGateObservationV0, error) {
	if source.Store == nil {
		return nil, fmt.Errorf("codex_review_gate_observation_source: store requerido")
	}
	descriptors, err := source.Store.ListCodexReceiptDescriptorsV0(
		ctx,
		codexReviewGateDescriptorRequestV0(request),
	)
	if err != nil {
		return nil, err
	}
	observations := make([]orquestacionnucleoapp.ReviewGateObservationV0, 0, len(descriptors))
	for _, descriptor := range descriptors {
		observation, ready, err := source.observationFromReviewGateDescriptorV0(ctx, request, descriptor)
		if err != nil {
			return nil, err
		}
		if ready {
			observations = append(observations, observation)
		}
	}
	return observations, nil
}

func codexReviewGateDescriptorRequestV0(
	request orquestacionnucleoapp.ReviewGateObservationRequestV0,
) CodexReceiptDescriptorRequestV0 {
	startedAgents := compactCodexDeliveryRefsV0(request.Run.StartedAgents)
	if len(request.WaitAgentRefs) > 0 {
		startedAgents = codexReviewGateScopedStartedAgentsV0(startedAgents, request.WaitAgentRefs)
	}
	return CodexReceiptDescriptorRequestV0{
		RunID:          strings.TrimSpace(request.Run.RunID),
		StartedAgents:  startedAgents,
		Deliveries:     nil,
		PhaseArtifacts: nil,
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		EvidenceRefs:   compactCodexDeliveryRefsV0(request.EvidenceRefs),
	}
}

func (source CodexReviewGateObservationSourceV0) observationFromReviewGateDescriptorV0(
	ctx context.Context,
	request orquestacionnucleoapp.ReviewGateObservationRequestV0,
	descriptor CodexReceiptDescriptorV0,
) (orquestacionnucleoapp.ReviewGateObservationV0, bool, error) {
	ack, issues := orquestaruntimecodex.ReadAndValidateCodexAgentAckFileV0(
		strings.TrimSpace(descriptor.AckPath),
		descriptor.Spec,
	)
	if len(issues) > 0 && codexReceiptIssueMeansAckNotReadyV0(issues[0]) {
		return orquestacionnucleoapp.ReviewGateObservationV0{}, false, nil
	}
	if len(issues) > 0 && (!codexReviewGateCanEvaluateAckV0(ack) ||
		!codexReviewGateIssuesEvaluableV0(issues)) {
		return orquestacionnucleoapp.ReviewGateObservationV0{}, false, fmt.Errorf(
			"codex_review_gate_observation: %s:%s",
			issues[0].Code,
			issues[0].Field,
		)
	}
	deliveryRef := strings.TrimSpace(ack.AckRef)
	if !codexReviewGateDeliveryEligibleV0(request, descriptor, deliveryRef) {
		return orquestacionnucleoapp.ReviewGateObservationV0{}, false, nil
	}
	input, fileIssues, err := source.reviewGateInputV0(ctx, descriptor, ack)
	if err != nil {
		return orquestacionnucleoapp.ReviewGateObservationV0{}, false, err
	}
	result := orquestaautoprogramming.EvaluateAutoprogrammingReviewGateV0(input)
	result = codexReviewGateMergeGateIssuesV0(result, fileIssues)
	result = codexReviewGateMergeConnectorIssuesV0(result, issues)
	if codexReviewGateTerminalProjectedV0(request.Run, deliveryRef, source.reviewGateStatusV0(result)) {
		return orquestacionnucleoapp.ReviewGateObservationV0{}, false, nil
	}
	return source.reviewGateObservationV0(ack, result), true, nil
}

func codexReviewGateCanEvaluateAckV0(ack orquestaruntimecodex.CodexAgentAckV0) bool {
	return strings.TrimSpace(ack.AckRef) != "" &&
		strings.TrimSpace(ack.RequestID) != "" &&
		strings.TrimSpace(ack.Status) != ""
}

func codexReviewGateDeliveryEligibleV0(
	request orquestacionnucleoapp.ReviewGateObservationRequestV0,
	descriptor CodexReceiptDescriptorV0,
	deliveryRef string,
) bool {
	agentRef := strings.TrimSpace(descriptor.AgentRef)
	if agentRef == "" {
		agentRef = strings.TrimSpace(descriptor.Spec.RequestID)
	}
	return stringInCodexDeliverySetV0(request.Run.Deliveries, deliveryRef) &&
		codexReviewGateAgentEligibleV0(request, agentRef)
}

func codexReviewGateAgentEligibleV0(
	request orquestacionnucleoapp.ReviewGateObservationRequestV0,
	agentRef string,
) bool {
	if len(request.WaitAgentRefs) > 0 && !stringInCodexDeliverySetV0(request.WaitAgentRefs, agentRef) {
		return false
	}
	return stringInCodexDeliverySetV0(request.Run.Agents, agentRef) &&
		stringInCodexDeliverySetV0(request.Run.StartedAgents, agentRef) &&
		!stringInCodexDeliverySetV0(request.Run.FailedAgents, agentRef)
}

func codexReviewGateScopedStartedAgentsV0(
	startedAgents []string,
	waitAgentRefs []string,
) []string {
	out := make([]string, 0, len(startedAgents))
	for _, agentRef := range compactCodexDeliveryRefsV0(startedAgents) {
		if stringInCodexDeliverySetV0(waitAgentRefs, agentRef) {
			out = append(out, agentRef)
		}
	}
	return compactCodexDeliveryRefsV0(out)
}

func (source CodexReviewGateObservationSourceV0) reviewGateInputV0(
	ctx context.Context,
	descriptor CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
) (orquestaautoprogramming.AutoprogrammingReviewGateInputV0, []orquestaautoprogramming.AutoprogrammingReviewGateIssueV0, error) {
	evidence, err := source.reviewGateFileEvidenceV0(ctx, descriptor, ack)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingReviewGateInputV0{}, nil, err
	}
	return orquestaautoprogramming.AutoprogrammingReviewGateInputV0{
		ACK: orquestaautoprogramming.AutoprogrammingReviewGateACKV0{
			Present: true,
			Status:  strings.TrimSpace(ack.Status),
		},
		RequiredTests:   compactCodexDeliveryRefsV0(descriptor.Spec.AgentPacket.Task.RequiredTests),
		Tests:           codexReviewGateTestsV0(ack.Tests),
		Files:           evidence.Files,
		WriteSet:        codexReviewGateAllowedFilesV0(descriptor.Spec, ack),
		MaxLinesPerFile: source.MaxLinesPerFile,
	}, evidence.Issues, nil
}

func (source CodexReviewGateObservationSourceV0) reviewGateFileEvidenceV0(
	ctx context.Context,
	descriptor CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
) (CodexReviewGateFileEvidenceV0, error) {
	if source.FileEvidenceResult != nil {
		return source.FileEvidenceResult.BuildCodexReviewGateFileEvidenceV0(ctx, descriptor, ack)
	}
	if source.FileEvidence != nil {
		files, err := source.FileEvidence.BuildCodexReviewGateFilesV0(ctx, descriptor, ack)
		return CodexReviewGateFileEvidenceV0{Files: files}, err
	}
	files := make([]orquestaautoprogramming.AutoprogrammingReviewGateFileV0, 0, len(ack.Files))
	for _, path := range compactCodexDeliveryRefsV0(ack.Files) {
		files = append(files, orquestaautoprogramming.AutoprogrammingReviewGateFileV0{Path: path})
	}
	return CodexReviewGateFileEvidenceV0{Files: files}, nil
}

func codexReviewGateTestsV0(
	values []string,
) []orquestaautoprogramming.AutoprogrammingReviewGateTestV0 {
	tests := make([]orquestaautoprogramming.AutoprogrammingReviewGateTestV0, 0, len(values))
	for _, command := range compactCodexDeliveryRefsV0(values) {
		tests = append(tests, orquestaautoprogramming.AutoprogrammingReviewGateTestV0{
			Command: command,
			Passed:  true,
			Status:  "passed",
		})
	}
	return tests
}

func codexReviewGateAllowedFilesV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
) []string {
	allowed := compactCodexDeliveryRefsV0(spec.AgentPacket.Task.WriteSet)
	for _, file := range compactCodexDeliveryRefsV0(ack.Files) {
		if codexReviewGatePathAllowedByWriteSetV0(file, allowed) {
			allowed = append(allowed, file)
		}
	}
	return compactCodexDeliveryRefsV0(allowed)
}

func codexReviewGatePathAllowedByWriteSetV0(path string, writeSet []string) bool {
	path = strings.TrimSpace(path)
	for _, allowed := range writeSet {
		allowed = strings.TrimSpace(allowed)
		if path == allowed || strings.HasPrefix(path, allowed+"/") {
			return true
		}
	}
	return false
}

func (source CodexReviewGateObservationSourceV0) reviewGateObservationV0(
	ack orquestaruntimecodex.CodexAgentAckV0,
	result orquestaautoprogramming.AutoprogrammingReviewGateResultV0,
) orquestacionnucleoapp.ReviewGateObservationV0 {
	deliveryRef := strings.TrimSpace(ack.AckRef)
	observation := orquestacionnucleoapp.ReviewGateObservationV0{
		CandidateRef:    "review-gate-candidate-ref-" + deliveryRef,
		ReviewRequestID: codexReviewGateReviewRequestIDV0(deliveryRef),
		ReviewResultRef: codexReviewGateReviewResultRefV0(deliveryRef),
		DeliveryRef:     deliveryRef,
		PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		Status:          source.reviewGateStatusV0(result),
		Summary:         codexReviewGateSummaryV0(result),
		QualityGateRef:  source.qualityGateRefV0(deliveryRef),
		EvidenceRefs:    codexReviewGateEvidenceRefsV0(ack, result),
	}
	if result.Accepted {
		observation.AcceptedReviewRef = codexReviewGateAcceptedReviewRefV0(deliveryRef)
	}
	return observation
}

func (source CodexReviewGateObservationSourceV0) reviewGateStatusV0(
	result orquestaautoprogramming.AutoprogrammingReviewGateResultV0,
) orquestacoreworkflow.ReviewResultStatusV0 {
	if result.Accepted {
		return orquestacoreworkflow.ReviewResultStatusAcceptedV0
	}
	if source.FailureStatus != "" {
		return source.FailureStatus
	}
	return orquestacoreworkflow.ReviewResultStatusChangesRequestedV0
}

func (source CodexReviewGateObservationSourceV0) qualityGateRefV0(deliveryRef string) string {
	if source.QualityGateRefFn != nil {
		if ref := strings.TrimSpace(source.QualityGateRefFn(deliveryRef)); ref != "" {
			return ref
		}
	}
	return "quality-gate-ref-" + strings.TrimSpace(deliveryRef)
}
