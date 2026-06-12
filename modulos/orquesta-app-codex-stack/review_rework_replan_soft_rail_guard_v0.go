package orquestaappcodexstack

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func reviewResultNeedsReworkV0(status orquestacoreworkflow.ReviewResultStatusV0) bool {
	return status == orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 ||
		status == orquestacoreworkflow.ReviewResultStatusRejectedV0
}

func reviewReworkReplanCapacityV0(
	config CapacityConfigV0,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if strings.TrimSpace(string(config.Tier)) != "" {
		return config.Tier
	}
	return orquestacoreworkflow.OrchestrationCapacityHighV0
}

func reviewReworkReplanSkipSoftRailOnlyV0(
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
	_ reviewReworkProjectionV0,
	_ reviewResultProjectionV0,
) bool {
	hasSoftRail := false
	for _, ref := range compactStringsV0(request.EvidenceRefs) {
		if autoprogrammingResidentEvidenceIsBlockingV0(ref) {
			return false
		}
		if autoprogrammingResidentEvidenceIsSoftRailV0(ref) {
			hasSoftRail = true
		}
	}
	return hasSoftRail
}

func reviewReworkReplanSkipExternalRailDocsLoopV0(
	rework reviewReworkProjectionV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	for _, descriptor := range descriptors {
		if strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef) !=
			strings.TrimSpace(rework.DeliveryRef) {
			continue
		}
		return reviewReworkDescriptorIsCompletedDocsOnlyExternalRailV0(descriptor)
	}
	return false
}

func reviewReworkDescriptorIsCompletedDocsOnlyExternalRailV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	task := descriptor.Spec.AgentPacket.Task
	if !reviewReworkWriteSetDocumentationOnlyV0(task.WriteSet) ||
		!reviewReworkHasGlobalGoTestRequiredV0(task.RequiredTests) {
		return false
	}
	ack, err := orquestaruntimecodex.ReadCodexAgentAckFileV0(descriptor.AckPath)
	if err != nil {
		return false
	}
	if strings.TrimSpace(ack.Status) != "completed" {
		return false
	}
	return reviewReworkAckHasExternalGoTestRailFailureV0(ack)
}

func reviewReworkWriteSetDocumentationOnlyV0(writeSet []string) bool {
	values := compactStringsV0(writeSet)
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		value = strings.Trim(strings.TrimSpace(value), "/")
		switch {
		case value == "docs":
			continue
		case strings.HasPrefix(value, "docs/"):
			continue
		case strings.HasSuffix(value, ".md"):
			continue
		default:
			return false
		}
	}
	return true
}

func reviewReworkHasGlobalGoTestRequiredV0(requiredTests []string) bool {
	for _, command := range compactStringsV0(requiredTests) {
		if reviewReworkCommandIsGlobalGoTestV0(command) {
			return true
		}
	}
	return false
}

func reviewReworkAckHasExternalGoTestRailFailureV0(
	ack orquestaruntimecodex.CodexAgentAckV0,
) bool {
	hasFailedGlobalGoTest := false
	var evidence []string
	for _, receipt := range ack.TestReceipts {
		if reviewReworkCommandIsGlobalGoTestV0(receipt.Command) &&
			reviewReworkTestReceiptFailedV0(receipt) {
			hasFailedGlobalGoTest = true
		}
		evidence = append(evidence, receipt.Command, receipt.Status)
		evidence = append(evidence, receipt.EvidenceRefs...)
	}
	evidence = append(evidence, ack.Tests...)
	evidence = append(evidence, ack.Notes...)
	if !hasFailedGlobalGoTest {
		return false
	}
	return reviewReworkEvidenceLooksExternalRailV0(evidence)
}

func reviewReworkCommandIsGlobalGoTestV0(command string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(command), " "))
	return strings.HasPrefix(normalized, "go test ") && strings.Contains(normalized, "./...")
}

func reviewReworkTestReceiptFailedV0(receipt orquestaruntimecodex.CodexRequiredTestReceiptV0) bool {
	status := strings.TrimSpace(receipt.Status)
	if status != "" && status != "passed" {
		return true
	}
	return receipt.ExitCode != nil && *receipt.ExitCode != 0
}

func reviewReworkEvidenceLooksExternalRailV0(values []string) bool {
	for _, value := range values {
		normalized := strings.ToLower(value)
		for _, marker := range []string{
			"gocache",
			"read-only",
			"readonly",
			"solo lectura",
			"socket",
			"httptest",
			"operation not permitted",
			"exit 75",
			"sandbox",
			"wrapper",
		} {
			if strings.Contains(normalized, marker) {
				return true
			}
		}
	}
	return false
}
