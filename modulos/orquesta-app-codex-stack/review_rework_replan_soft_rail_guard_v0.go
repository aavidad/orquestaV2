package orquestaappcodexstack

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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
