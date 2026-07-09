package orquestaopesdirector

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	opesDirectorArtifactTypeCompletedSyllabusPackageV0 = "completed_syllabus_package"
	opesDirectorArtifactTypeLearningGamesPackageV0     = "learning_games_package"
	opesDirectorArtifactTypeHelpManualPackageV0        = "help_manual_package"
	opesDirectorArtifactTypeQualityAuditReportV0       = "opes_quality_audit_report"
)

func normalizeOPESDirectorArtifactTypeV0(artifactType string) string {
	switch strings.TrimSpace(artifactType) {
	case opesDirectorArtifactTypeCompletedSyllabusPackageV0:
		return orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0
	case opesDirectorArtifactTypeLearningGamesPackageV0:
		return orquestadomainwork.DomainWorkArtifactTypeInteractivePracticeV0
	case opesDirectorArtifactTypeHelpManualPackageV0:
		return orquestadomainwork.DomainWorkArtifactTypeHelpPackageV0
	default:
		return strings.TrimSpace(artifactType)
	}
}

func opesDirectorIsFinalPackageArtifactTypeV0(artifactType string) bool {
	return normalizeOPESDirectorArtifactTypeV0(artifactType) ==
		orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0
}
