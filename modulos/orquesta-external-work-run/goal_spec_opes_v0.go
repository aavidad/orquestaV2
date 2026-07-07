package orquestaexternalworkrun

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func externalWorkGoalIsOPESDomainV0(domainRequest orquestadomainwork.DomainWorkJobRequestV0) bool {
	for _, value := range []string{domainRequest.DomainRef, domainRequest.WorkKind} {
		if externalWorkGoalLooksOPESRefV0(value) {
			return true
		}
	}
	for _, ref := range domainRequest.InterfaceRefs {
		if externalWorkGoalLooksOPESRefV0(ref) {
			return true
		}
	}
	return false
}

func externalWorkGoalLooksOPESRefV0(value string) bool {
	normalized := compactExternalWorkRunRefV0(value)
	return normalized == "opes" ||
		strings.HasPrefix(normalized, "opes-") ||
		strings.HasPrefix(normalized, "opes_")
}
