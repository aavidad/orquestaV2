package orquestadocumentplanexpander

import orquestadomainwork "orquesta/modulos/orquesta-domain-work"

func ExpectedArtifactTypeForDocumentPlanWorkKindV0(workKind string) string {
	return orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
}
