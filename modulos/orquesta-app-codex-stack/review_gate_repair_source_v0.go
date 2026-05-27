package orquestaappcodexstack

import (
	"context"
	"sort"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

const codexStackReviewGateNormalizedTestsEvidenceV0 = "gate-normalized:required-tests-equivalent"

type codexStackReviewGateRepairSourceV0 struct {
	Store orquestaruntimecodexdelivery.CodexReceiptDescriptorStorePortV0
	Inner orquestacionnucleoapp.ReviewGateObservationProviderPortV0
}

var _ orquestacionnucleoapp.ReviewGateObservationProviderPortV0 = codexStackReviewGateRepairSourceV0{}

func (source codexStackReviewGateRepairSourceV0) BuildReviewGateObservationsV0(
	ctx context.Context,
	request orquestacionnucleoapp.ReviewGateObservationRequestV0,
) ([]orquestacionnucleoapp.ReviewGateObservationV0, error) {
	observations, err := source.Inner.BuildReviewGateObservationsV0(ctx, request)
	if err != nil || source.Store == nil {
		return observations, err
	}
	descriptors, err := source.Store.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: request.Run.RunID},
	)
	if err != nil {
		return nil, err
	}
	for index := range observations {
		observations[index] = source.repairObservationV0(observations[index], descriptors, request.Run)
	}
	observations = append(observations, codexStackReviewGateReworkAcceptanceObservationsV0(request, descriptors)...)
	return observations, nil
}

func (source codexStackReviewGateRepairSourceV0) repairObservationV0(
	observation orquestacionnucleoapp.ReviewGateObservationV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestacionnucleoapp.ReviewGateObservationV0 {
	if observation.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		!codexStackReviewGateHasRequiredTestIssueV0(observation.EvidenceRefs) {
		return codexStackReviewGateSoftReworkAcceptedObservationV0(run, observation)
	}
	descriptor, ok := codexStackReviewGateDescriptorForDeliveryV0(descriptors, observation.DeliveryRef)
	if !ok || !codexStackReviewGateAckTestsEquivalentV0(descriptor) {
		return observation
	}
	cleaned := codexStackReviewGateWithoutRequiredTestIssuesV0(observation.EvidenceRefs)
	if !codexStackReviewGateEvidenceOnlyAdvisoryV0(cleaned) {
		return observation
	}
	observation.Status = orquestacoreworkflow.ReviewResultStatusAcceptedV0
	observation.AcceptedReviewRef = "accepted-review-ref-" + strings.TrimSpace(observation.DeliveryRef)
	observation.Summary = "Entrega aceptada por gate de revision."
	observation.EvidenceRefs = compactStringsV0(append(cleaned, codexStackReviewGateNormalizedTestsEvidenceV0))
	return codexStackReviewGateSoftReworkAcceptedObservationV0(run, observation)
}

func codexStackReviewGateDescriptorForDeliveryV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	deliveryRef string,
) (orquestaruntimecodexdelivery.CodexReceiptDescriptorV0, bool) {
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, descriptor := range descriptors {
		if strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef) == deliveryRef {
			return descriptor, true
		}
	}
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}, false
}

func codexStackReviewGateAckTestsEquivalentV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	ack, issues := orquestaruntimecodex.ReadAndValidateCodexAgentAckFileV0(
		strings.TrimSpace(descriptor.AckPath),
		descriptor.Spec,
	)
	if strings.TrimSpace(ack.Status) != "completed" || codexStackReviewGateHasNonTestIssueV0(issues) {
		return false
	}
	return codexStackReviewGateTestsCoverRequiredV0(
		descriptor.Spec.AgentPacket.Task.RequiredTests,
		ack.Tests,
	)
}

func codexStackReviewGateTestsCoverRequiredV0(required []string, actual []string) bool {
	actualSet := map[string]bool{}
	for _, command := range compactStringsV0(actual) {
		actualSet[codexStackReviewGateNormalizeTestCommandV0(command)] = true
	}
	for _, command := range compactStringsV0(required) {
		if !actualSet[codexStackReviewGateNormalizeTestCommandV0(command)] {
			return false
		}
	}
	return len(required) > 0
}

func codexStackReviewGateNormalizeTestCommandV0(command string) string {
	parts := strings.Fields(strings.TrimSpace(command))
	if len(parts) < 2 || parts[0] != "go" || parts[1] != "test" {
		return strings.Join(parts, " ")
	}
	flags := []string{}
	packages := []string{}
	for index := 2; index < len(parts); index++ {
		part := parts[index]
		if part == "-count=1" || part == "-v" {
			continue
		}
		if part == "-count" && index+1 < len(parts) && parts[index+1] == "1" {
			index++
			continue
		}
		if codexStackReviewGateFlagNeedsValueV0(part) && index+1 < len(parts) {
			flags = append(flags, part+"="+parts[index+1])
			index++
			continue
		}
		if strings.HasPrefix(part, "-") {
			flags = append(flags, part)
			continue
		}
		packages = append(packages, part)
	}
	sort.Strings(flags)
	sort.Strings(packages)
	return "go test|" + strings.Join(packages, " ") + "|" + strings.Join(flags, " ")
}

func codexStackReviewGateFlagNeedsValueV0(flag string) bool {
	switch flag {
	case "-run", "-timeout", "-tags", "-parallel", "-coverprofile", "-coverpkg":
		return true
	default:
		return false
	}
}

func codexStackReviewGateHasRequiredTestIssueV0(evidenceRefs []string) bool {
	for _, ref := range evidenceRefs {
		code, ok := strings.CutPrefix(strings.TrimSpace(ref), "gate-issue:")
		if ok && codexStackReviewGateRequiredTestIssueV0(code) {
			return true
		}
	}
	return false
}

func codexStackReviewGateWithoutRequiredTestIssuesV0(evidenceRefs []string) []string {
	out := make([]string, 0, len(evidenceRefs))
	for _, ref := range compactStringsV0(evidenceRefs) {
		code, ok := strings.CutPrefix(ref, "gate-issue:")
		if ok && codexStackReviewGateRequiredTestIssueV0(code) {
			continue
		}
		out = append(out, ref)
	}
	return out
}

func codexStackReviewGateRequiredTestIssueV0(code string) bool {
	switch strings.TrimSpace(code) {
	case "required_test_missing", "missing_required_test", "required_test_not_executed":
		return true
	default:
		return false
	}
}

func codexStackReviewGateEvidenceOnlyAdvisoryV0(evidenceRefs []string) bool {
	for _, ref := range evidenceRefs {
		code, ok := strings.CutPrefix(strings.TrimSpace(ref), "gate-issue:")
		if !ok {
			continue
		}
		if !codexStackReviewGateAdvisoryIssueV0(code) {
			return false
		}
	}
	return true
}

func codexStackReviewGateAdvisoryIssueV0(code string) bool {
	return orquestaautoprogramming.AutoprogrammingReviewGateIssueCodeIsAdvisoryV0(code)
}

func codexStackReviewGateHasNonTestIssueV0(issues []orquestaruntime.ExternalAgentConnectorErrorV0) bool {
	for _, issue := range issues {
		if len(issue.Evidence) == 0 {
			return true
		}
		for _, evidence := range issue.Evidence {
			evidence = strings.TrimSpace(evidence)
			if evidence != "" &&
				!codexStackReviewGateRequiredTestIssueV0(evidence) &&
				!codexStackReviewGateAdvisoryIssueV0(evidence) {
				return true
			}
		}
	}
	return false
}

func codexStackReviewGateSoftReworkAcceptedObservationV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observation orquestacionnucleoapp.ReviewGateObservationV0,
) orquestacionnucleoapp.ReviewGateObservationV0 {
	if observation.Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		codexStackReviewGateSoftReworkAcceptanceAlreadyDoneV0(run, observation.DeliveryRef) ||
		!codexStackReviewGateDeliveryHasReworkV0(run, observation.DeliveryRef) {
		return observation
	}
	safe := codexStackOperationalClosureSafeRefV0(observation.DeliveryRef)
	observation.CandidateRef = "review-gate-candidate-ref-soft-rework-accepted-" + safe
	observation.ReviewResultRef = "review-result-ref-soft-rework-accepted-" + safe
	observation.AcceptedReviewRef = "accepted-review-ref-soft-rework-accepted-" + safe
	observation.QualityGateRef = "quality-gate-ref-soft-rework-accepted-" + safe
	observation.Summary = "Entrega aceptada tras normalizar rails blandos de revision."
	evidenceRefs := append(observation.EvidenceRefs, "gate-normalized:soft-rework-accepted")
	evidenceRefs = append(evidenceRefs, codexStackReviewGateReworkRefsForDeliveryV0(run, observation.DeliveryRef)...)
	observation.EvidenceRefs = compactStringsV0(evidenceRefs)
	return observation
}

func codexStackReviewGateDeliveryHasReworkV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
) bool {
	return len(codexStackReviewGateReworkRefsForDeliveryV0(run, deliveryRef)) > 0
}

func codexStackReviewGateReworkRefsForDeliveryV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
) []string {
	deliveryRef = strings.TrimSpace(deliveryRef)
	refs := []string{}
	for _, rawRework := range run.ReworkRequests {
		rework, ok := codexStackReviewGateParseReworkProjectionV0(rawRework)
		if ok && rework.DeliveryRef == deliveryRef {
			refs = append(refs, rework.ReworkRequestRef)
		}
	}
	return compactStringsV0(refs)
}

func codexStackReviewGateSoftReworkAcceptanceAlreadyDoneV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
) bool {
	safe := codexStackOperationalClosureSafeRefV0(deliveryRef)
	return codexStackStringInSetV0(run.AcceptedReviews, "accepted-review-ref-soft-rework-accepted-"+safe)
}
