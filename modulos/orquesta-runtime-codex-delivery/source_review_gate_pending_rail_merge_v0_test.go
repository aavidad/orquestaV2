package orquestaruntimecodexdelivery

import (
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexReviewGateMergeConnectorIssuesV0MantienePendingRailComoAdvisory(t *testing.T) {
	result := codexReviewGateMergeConnectorIssuesV0(
		orquestaautoprogramming.AutoprogrammingReviewGateResultFromIssuesV0(nil),
		[]orquestaruntime.ExternalAgentConnectorErrorV0{{
			Code:     orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckArtifactV0),
			Field:    "notes",
			Evidence: []string{"ack-pending-rail:token"},
		}},
	)

	if !result.Accepted || !result.PreserveOutput || !result.RequiresFollowup ||
		result.RecommendedAction != orquestaautoprogramming.AutoprogrammingReviewGateActionRequestFollowupReviewV0 {
		t.Fatalf("pending rail debe quedar advisory: %+v", result)
	}
	if !codexReviewGateResultHasIssueV0(result, "ack-pending-rail:token") {
		t.Fatalf("pending rail perdido: %+v", result.Issues)
	}
}

func TestCodexReviewGateMergeConnectorIssuesV0NormalizaRailBlandoDeArtifact(t *testing.T) {
	result := codexReviewGateMergeConnectorIssuesV0(
		orquestaautoprogramming.AutoprogrammingReviewGateResultFromIssuesV0(nil),
		[]orquestaruntime.ExternalAgentConnectorErrorV0{{
			Code:     orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckArtifactV0),
			Field:    "files",
			Evidence: []string{"artifact_path_outside_write_set:docs/extra.md"},
		}},
	)

	if !result.Accepted || !result.PreserveOutput || !result.RequiresFollowup ||
		result.RecommendedAction != orquestaautoprogramming.AutoprogrammingReviewGateActionRequestFollowupReviewV0 {
		t.Fatalf("artifact fuera de write-set debe quedar advisory: %+v", result)
	}
	if !codexReviewGateResultHasIssueV0(result, "artifact_path_outside_write_set:docs/extra.md") {
		t.Fatalf("rail blando perdido: %+v", result.Issues)
	}
}

func TestCodexReviewGateMergePendingRailEvidenceV0MantieneACKRailComoFollowup(t *testing.T) {
	ack := orquestaruntimecodex.CodexAgentAckV0{
		Notes: []string{"rail pendiente: token/provider/home como vocabulario operativo"},
	}

	result := codexReviewGateMergePendingRailEvidenceV0(
		orquestaautoprogramming.AutoprogrammingReviewGateResultFromIssuesV0(nil),
		ack,
	)

	if !result.Accepted || !result.PreserveOutput || !result.RequiresFollowup ||
		result.RecommendedAction != orquestaautoprogramming.AutoprogrammingReviewGateActionRequestFollowupReviewV0 {
		t.Fatalf("pending rail debe quedar como followup advisory: %+v", result)
	}
	if !codexReviewGateResultHasIssueV0(result, orquestaruntimecodex.CodexAgentAckPendingRailEvidenceRefV0) ||
		!codexReviewGateResultHasIssueV0(result, "ack-pending-rail:token") {
		t.Fatalf("pending rail evidence perdido: %+v", result.Issues)
	}
}

func TestCodexReviewGateEvidenceRefsV0NormalizaGateIssuePrefijado(t *testing.T) {
	ack := orquestaruntimecodex.CodexAgentAckV0{AckRef: "ack-ref-soft-rail-prefixed-001"}
	result := orquestaautoprogramming.AutoprogrammingReviewGateResultFromIssuesV0(
		[]orquestaautoprogramming.AutoprogrammingReviewGateIssueV0{
			{Code: "gate-issue:file_too_large"},
			{Code: "gate-issue:ack-pending-rail:token"},
		},
	)

	refs := codexReviewGateEvidenceRefsV0(ack, result)

	if !stringInCodexDeliverySetV0(refs, "gate-issue:file_too_large") ||
		!stringInCodexDeliverySetV0(refs, "gate-issue:ack-pending-rail:token") ||
		stringInCodexDeliverySetV0(refs, "gate-issue:gate-issue:file_too_large") {
		t.Fatalf("refs de rail blando mal normalizadas: %v", refs)
	}
}
