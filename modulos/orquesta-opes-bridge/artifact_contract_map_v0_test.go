package orquestaopesbridge

import (
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

func TestOPESBridgeArtifactContractMapConsumeOwnerNeutralV0(t *testing.T) {
	for _, workKind := range []string{
		"draft_content_block",
		"generate_visual_asset",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"validate_topic",
		"expand_topic_from_summary",
		"assemble_topic",
		"generate_audio_asset",
		"generate_topic_audio",
		"unknown_work_kind",
	} {
		t.Run(workKind, func(t *testing.T) {
			expectedArtifact := orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
			if got := expectedArtifactTypeV0(workKind); got != expectedArtifact {
				t.Fatalf("artifact=%s want=%s", got, expectedArtifact)
			}
			req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
				ID:          "job-ref-" + strings.ReplaceAll(workKind, "_", "-") + "-001",
				Type:        workKind,
				PayloadJSON: `{"topic_id":"topic-ref-artifact-map-001"}`,
			}, JobRunConfigV0{})
			if !ok || req.AppChangeRequest.ExternalWork == nil {
				t.Fatalf("request no construida")
			}
			if !fieldValueForTestV0(
				req.AppChangeRequest.ExternalWork.InputFields,
				"expected_artifact_type",
				expectedArtifact,
			) {
				t.Fatalf("input_fields=%+v", req.AppChangeRequest.ExternalWork.InputFields)
			}
		})
	}
}
