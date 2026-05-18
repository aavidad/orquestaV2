package orquestaappcodexstack

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

func domainWorkArtifactRequestIDV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
	artifactType string,
) string {
	return "req-domain-work-artifact-" + domainWorkArtifactStableSuffixV0(work, artifactType)
}

func domainWorkArtifactIdempotencyKeyV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
	artifactType string,
) string {
	return "idem-domain-work-artifact-" + domainWorkArtifactStableSuffixV0(work, artifactType)
}

func domainWorkArtifactStableSuffixV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
	artifactType string,
) string {
	if work == nil {
		return "unknown"
	}
	parts := compactCodexStackStringsV0([]string{
		strings.TrimSpace(work.ProjectRef),
		strings.TrimSpace(work.JobRef),
		strings.TrimSpace(artifactType),
	})
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, "-")
}
