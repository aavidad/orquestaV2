package main

import (
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

func TestOPESDrainArtifactContractMapConsumeOwnerNeutralV0(t *testing.T) {
	for _, workKind := range []string{
		"draft_content_block",
		"generate_visual_asset",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"validate_topic",
		"expand_topic_from_summary",
		"assemble_topic",
		"unknown_work_kind",
	} {
		expectedArtifact := orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
		req, ok := orquestaopesbridge.BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
			ID:          "job-ref-" + strings.ReplaceAll(workKind, "_", "-") + "-001",
			Type:        workKind,
			PayloadJSON: `{"topic_id":"topic-ref-server-artifact-map-001"}`,
		}, orquestaopesbridge.JobRunConfigV0{})
		if !ok || req.AppChangeRequest.ExternalWork == nil {
			t.Fatalf("%s request no construida", workKind)
		}
		if !domainWorkFieldHasValueForServerArtifactMapTestV0(
			req.AppChangeRequest.ExternalWork.InputFields,
			"expected_artifact_type",
			expectedArtifact,
		) {
			t.Fatalf("%s input_fields=%+v", workKind, req.AppChangeRequest.ExternalWork.InputFields)
		}
	}
}

func domainWorkFieldHasValueForServerArtifactMapTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	value string,
) bool {
	for _, field := range fields {
		if field.Name == name && field.Value == value {
			return true
		}
	}
	return false
}
