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
	if err != nil || len(observations) == 0 || source.Store == nil {
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
		observations[index] = source.repairObservationV0(observations[index], descriptors)
	}
	return observations, nil
}

func (source codexStackReviewGateRepairSourceV0) repairObservationV0(
	observation orquestacionnucleoapp.ReviewGateObservationV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestacionnucleoapp.ReviewGateObservationV0 {
	if observation.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		!codexStackReviewGateHasRequiredTestIssueV0(observation.EvidenceRefs) {
		return observation
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
	return observation
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
	case "required_test_missing", "missing_required_test":
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
			if evidence != "" && !codexStackReviewGateRequiredTestIssueV0(evidence) {
				return true
			}
		}
	}
	return false
}
