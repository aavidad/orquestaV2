package orquestaappcodexstack

import (
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

func TestAppChangeEvidenceRefsV0IncluyeTrabajoExterno(t *testing.T) {
	refs := appChangeEvidenceRefsV0(orquestaappchange.AppChangeRequestV0{
		ChangeRef:        "change-ref-stack-external-001",
		CurrentStateRefs: []string{"delivery-ref-stack-001"},
		MetadataRefs:     []string{"metadata-ref-stack-001"},
		ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
			ProjectRef:    "project-ref-opes",
			InterfaceRefs: []string{"mcp-contract-ref-opes-v0"},
			WorkKind:      "documentation",
			WorkRefs:      []string{"domain-work-ref-opes-topic-001"},
		},
	})

	for _, expected := range []string{
		"change-ref-stack-external-001",
		"delivery-ref-stack-001",
		"metadata-ref-stack-001",
		"project-ref-opes",
		"mcp-contract-ref-opes-v0",
		"documentation",
		"domain-work-ref-opes-topic-001",
	} {
		if !codexStackStringInSetForTestV0(refs, expected) {
			t.Fatalf("refs=%v missing=%s", refs, expected)
		}
	}
}
