package orquestaappcodexstack

import (
	"strconv"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func assessmentReplanSafeRefV0(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("\\", "-", "/", "-", " ", "-", "#", "-", ":", "-")
	value = strings.Trim(replacer.Replace(value), "-")
	if value == "" {
		return "sin-ref"
	}
	if len(value) > 64 {
		value = strings.Trim(value[:64], "-")
	}
	if value == "" {
		return "sin-ref"
	}
	return value
}

func assessmentReplanSuffixV0(
	runRef string,
	projection orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
	taskRef string,
	failedFollowupCount int,
) string {
	base := assessmentReplanSafeRefV0(taskRef)
	if base == "sin-ref" {
		base = assessmentReplanSafeRefV0(projection.AgentRequestID)
	}
	return base + "-" + assessmentReplanDigestV0(
		runRef,
		assessmentReplanSemanticKeyV0(runRef, projection, taskRef),
		taskRef,
		strconv.Itoa(failedFollowupCount),
	)
}

func assessmentReplanSemanticKeyV0(
	runRef string,
	projection orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
	taskRef string,
) string {
	return assessmentReplanDigestV0(
		runRef,
		taskRef,
		projection.DeliveryRef,
		projection.Verdict,
		projection.Action,
		projection.Severity,
	)
}

func assessmentReplanDigestV0(fields ...string) string {
	digest := codexStackDeterministicDigestV0(fields...)
	if len(digest) > 32 {
		return digest[:32]
	}
	return digest
}
