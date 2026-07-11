package orquestaruntimecodexappserver

import (
	"errors"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

type codexAppServerGoalResultMarkerV0 struct {
	SchemaVersion         string                                      `json:"schema_version,omitempty"`
	Status                string                                      `json:"status,omitempty"`
	Estado                string                                      `json:"estado,omitempty"`
	GoalRef               string                                      `json:"goal_ref,omitempty"`
	ExternalGoalRef       string                                      `json:"external_goal_ref,omitempty"`
	ReasonCode            string                                      `json:"reason_code,omitempty"`
	Summary               string                                      `json:"summary,omitempty"`
	ArtifactRefs          []string                                    `json:"artifact_refs,omitempty"`
	ArtifactPaths         []string                                    `json:"artifact_paths,omitempty"`
	MaterializedArtifacts []orquestagoal.GoalMaterializedArtifactV0   `json:"materialized_artifacts,omitempty"`
	Checklist             orquestagoal.GoalWorkChecklistV0            `json:"checklist,omitempty"`
	MissingRefs           []string                                    `json:"missing_refs,omitempty"`
	RequiredTestResults   []orquestagoal.GoalRequiredTestResultV0     `json:"required_test_results,omitempty"`
	DomainReceiptRefs     []string                                    `json:"domain_receipt_refs,omitempty"`
	ReworkPlanRefs        []string                                    `json:"rework_plan_refs,omitempty"`
	EvidenceRefs          []string                                    `json:"evidence_refs,omitempty"`
	RepairReceipt         *orquestagoal.GoalWorkResultRepairReceiptV0 `json:"repair_receipt,omitempty"`
}

func codexAppServerGoalResultFromThreadV0(
	thread serverCodexAppServerThreadReadV0,
) (codexAppServerGoalResultMarkerV0, bool, error) {
	if text := codexAppServerFinalMarkerTextV0(thread, true); text != "" {
		return parseCodexAppServerGoalResultMarkerV0(text)
	}
	if text := codexAppServerFinalMarkerTextV0(thread, false); text != "" {
		return parseCodexAppServerGoalResultMarkerV0(text)
	}
	return codexAppServerGoalResultMarkerV0{}, false, nil
}

func codexAppServerFinalMarkerTextV0(thread serverCodexAppServerThreadReadV0, finalOnly bool) string {
	for turnIndex := len(thread.Turns) - 1; turnIndex >= 0; turnIndex-- {
		items := thread.Turns[turnIndex].Items
		for itemIndex := len(items) - 1; itemIndex >= 0; itemIndex-- {
			item := items[itemIndex]
			if strings.TrimSpace(item.Type) != "agentMessage" {
				continue
			}
			if finalOnly && strings.TrimSpace(item.Phase) != "final_answer" {
				continue
			}
			text := strings.TrimSpace(item.Text)
			if markerText := codexAppServerGoalResultMarkerWindowV0(text); markerText != "" {
				return markerText
			}
		}
	}
	return ""
}

func codexAppServerGoalResultMarkerWindowV0(text string) string {
	markerIndex := strings.LastIndex(text, orquestaruntimecodexgoal.CodexGoalResultMarkerV0)
	if markerIndex < 0 {
		return ""
	}
	return strings.TrimSpace(text[markerIndex:])
}

func parseCodexAppServerGoalResultMarkerV0(text string) (codexAppServerGoalResultMarkerV0, bool, error) {
	if !strings.Contains(text, orquestaruntimecodexgoal.CodexGoalResultMarkerV0) {
		return codexAppServerGoalResultMarkerV0{}, false, nil
	}
	payload, ok := codexAppServerGoalResultMarkerJSONV0(text)
	if !ok {
		return codexAppServerGoalResultMarkerV0{}, true, errors.New("codex_app_server_goal_result_marker_json_missing")
	}
	decoded, err := orquestagoal.DecodeGoalWorkResultJSONV0([]byte(payload))
	if err != nil {
		return codexAppServerGoalResultMarkerV0{}, true, err
	}
	if decoded.Disposition == orquestagoal.GoalWorkResultJSONDispositionIrrecoverableV0 {
		return codexAppServerGoalResultMarkerV0{}, true, errors.New("codex_app_server_goal_result_irrecoverable")
	}
	marked := codexAppServerGoalResultMarkerFromNeutralV0(decoded.Result)
	marked = normalizeCodexAppServerGoalResultMarkerV0(marked)
	if issue := codexAppServerGoalResultContractIssueCodeV0(marked); issue != "" {
		return marked, true, codexAppServerGoalResultContractErrorV0{Code: issue}
	}
	return marked, true, nil
}

func codexAppServerGoalResultMarkerFromNeutralV0(result orquestagoal.GoalWorkResultV0) codexAppServerGoalResultMarkerV0 {
	marked := codexAppServerGoalResultMarkerV0{
		SchemaVersion:         result.SchemaVersion,
		Status:                result.Status,
		GoalRef:               result.GoalRef,
		ExternalGoalRef:       result.ExternalGoalRef,
		Summary:               result.Summary,
		ArtifactRefs:          result.ArtifactRefs,
		ArtifactPaths:         result.ArtifactPaths,
		MaterializedArtifacts: result.MaterializedArtifacts,
		Checklist:             result.Checklist,
		RequiredTestResults:   result.RequiredTestResults,
		DomainReceiptRefs:     result.DomainReceiptRefs,
		ReworkPlanRefs:        result.ReworkPlanRefs,
		EvidenceRefs:          result.EvidenceRefs,
		RepairReceipt:         result.RepairReceipt,
	}
	for _, issue := range result.Issues {
		if strings.TrimSpace(issue.Field) == "reason_code" && strings.TrimSpace(issue.Code) != "" {
			marked.ReasonCode = issue.Code
			break
		}
	}
	return marked
}

func codexAppServerGoalResultMarkerJSONV0(text string) (string, bool) {
	markerIndex := strings.LastIndex(text, orquestaruntimecodexgoal.CodexGoalResultMarkerV0)
	if markerIndex < 0 {
		return "", false
	}
	afterMarker := text[markerIndex+len(orquestaruntimecodexgoal.CodexGoalResultMarkerV0):]
	start := strings.Index(afterMarker, "{")
	if start < 0 {
		return "", false
	}
	depth := 0
	inString := false
	escaped := false
	for index := start; index < len(afterMarker); index++ {
		ch := afterMarker[index]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return afterMarker[start : index+1], true
			}
		}
	}
	return "", false
}

func codexAppServerGoalResultMarkerGoalRefMismatchV0(
	marked codexAppServerGoalResultMarkerV0,
	goalRef string,
) bool {
	markedGoalRef := strings.TrimSpace(marked.GoalRef)
	return markedGoalRef != "" && !codexAppServerGoalResultGoalRefMatchesV0(markedGoalRef, goalRef)
}

const codexAppServerGoalResultGoalRefNormalizedEvidenceV0 = "evidence-ref-codex-app-server-goal-result-goal-ref-normalized"

func codexAppServerGoalResultGoalRefMatchesV0(actual string, expected string) bool {
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)
	if actual == expected {
		return true
	}
	if !strings.HasPrefix(actual, "goal-ref-") || !strings.HasPrefix(expected, "goal-ref-") || len(actual) != len(expected) {
		return false
	}
	differences := 0
	for index := range actual {
		if actual[index] == expected[index] {
			continue
		}
		differences++
		if differences > 1 {
			return false
		}
	}
	return differences == 1
}

func normalizeCodexAppServerGoalResultGoalRefForRequestV0(
	marked codexAppServerGoalResultMarkerV0,
	goalRef string,
) (codexAppServerGoalResultMarkerV0, bool) {
	actual := strings.TrimSpace(marked.GoalRef)
	expected := strings.TrimSpace(goalRef)
	if actual == "" || actual == expected || !codexAppServerGoalResultGoalRefMatchesV0(actual, expected) {
		return marked, false
	}
	marked.GoalRef = expected
	marked.EvidenceRefs = compactServerStackStringsV0(append(
		marked.EvidenceRefs,
		codexAppServerGoalResultGoalRefNormalizedEvidenceV0,
	))
	return marked, true
}

func codexAppServerGoalResultMarkerExternalGoalRefMismatchV0(
	marked codexAppServerGoalResultMarkerV0,
	externalGoalRef string,
) bool {
	markedExternalGoalRef := strings.TrimSpace(marked.ExternalGoalRef)
	return markedExternalGoalRef != "" && markedExternalGoalRef != strings.TrimSpace(externalGoalRef)
}

func mergeCodexAppServerGoalResultV0(
	receipt *orquestaruntimecodexgoal.CodexGoalObservationReceiptV0,
	marked codexAppServerGoalResultMarkerV0,
	sourceEvidenceRef string,
) {
	if status := codexAppServerGoalResultExplicitStatusV0(marked); status != "" {
		receipt.Status = status
	}
	if strings.TrimSpace(marked.Summary) != "" {
		receipt.Summary = strings.TrimSpace(marked.Summary)
	}
	if reasonCode := codexAppServerGoalResultReasonCodeV0(marked); reasonCode != "" &&
		strings.TrimSpace(receipt.IssueCode) == "" {
		receipt.IssueCode = reasonCode
	}
	receipt.ArtifactRefs = compactServerStackStringsV0(append(receipt.ArtifactRefs, marked.ArtifactRefs...))
	receipt.ArtifactPaths = compactServerStackStringsV0(append(receipt.ArtifactPaths, marked.ArtifactPaths...))
	receipt.MaterializedArtifacts = append(receipt.MaterializedArtifacts, marked.MaterializedArtifacts...)
	receipt.Checklist = mergeCodexAppServerGoalResultChecklistV0(receipt.Checklist, marked.Checklist)
	receipt.RequiredTestResults = append(receipt.RequiredTestResults, marked.RequiredTestResults...)
	receipt.DomainReceiptRefs = compactServerStackStringsV0(append(receipt.DomainReceiptRefs, marked.DomainReceiptRefs...))
	receipt.ReworkPlanRefs = compactServerStackStringsV0(append(receipt.ReworkPlanRefs, marked.ReworkPlanRefs...))
	receipt.EvidenceRefs = compactServerStackStringsV0(append(
		append(receipt.EvidenceRefs, sourceEvidenceRef),
		marked.EvidenceRefs...,
	))
	if marked.RepairReceipt != nil {
		repair := *marked.RepairReceipt
		receipt.RepairReceipt = &repair
	}
	if codexAppServerGoalResultMarkerLooksPlaceholderV0(marked) {
		if strings.TrimSpace(receipt.IssueCode) == "" {
			receipt.IssueCode = codexAppServerGoalResultPlaceholderReasonCodeV0
		}
		receipt.EvidenceRefs = compactServerStackStringsV0(append(
			receipt.EvidenceRefs,
			codexAppServerGoalResultPlaceholderEvidenceRefV0,
		))
	}
}

const (
	codexAppServerGoalResultPlaceholderReasonCodeV0  = "goal_result_placeholder_in_progress"
	codexAppServerGoalResultCheckpointReasonCodeV0   = "checkpoint_started"
	codexAppServerGoalResultCheckpointEvidenceRefV0  = "evidence-ref-codex-app-server-checkpoint-started"
	codexAppServerGoalResultPlaceholderEvidenceRefV0 = "evidence-ref-goal-result-placeholder-in-progress"
)

func codexAppServerGoalResultReasonCodeV0(marked codexAppServerGoalResultMarkerV0) string {
	reasonCode := strings.TrimSpace(marked.ReasonCode)
	if reasonCode != "" {
		return reasonCode
	}
	for _, artifact := range marked.MaterializedArtifacts {
		for _, artifactIssue := range artifact.Issues {
			if code := strings.TrimSpace(artifactIssue.Code); code != "" {
				return code
			}
		}
	}
	return ""
}

// Un resultado blocked sin tests requeridos, sin checklist completada y sin
// missing_refs es un placeholder de progreso del agente, no un blocked real:
// los watchers deben discriminar por este reason code y nunca por el texto
// libre del summary (TAREA-D6 / P7).
func codexAppServerGoalResultMarkerLooksPlaceholderV0(marked codexAppServerGoalResultMarkerV0) bool {
	if codexAppServerGoalResultExplicitStatusV0(marked) != orquestagoal.GoalStatusBlockedV0 {
		return false
	}
	return len(marked.RequiredTestResults) == 0 &&
		len(marked.Checklist.CompletedRefs) == 0 &&
		len(marked.Checklist.MissingRefs) == 0 &&
		len(marked.MissingRefs) == 0
}

func mergeCodexAppServerGoalResultChecklistV0(
	current orquestagoal.GoalWorkChecklistV0,
	next orquestagoal.GoalWorkChecklistV0,
) orquestagoal.GoalWorkChecklistV0 {
	return orquestagoal.GoalWorkChecklistV0{
		ExpectedRefs:  compactServerStackStringsV0(append(current.ExpectedRefs, next.ExpectedRefs...)),
		CompletedRefs: compactServerStackStringsV0(append(current.CompletedRefs, next.CompletedRefs...)),
		MissingRefs:   compactServerStackStringsV0(append(current.MissingRefs, next.MissingRefs...)),
		EvidenceRefs:  compactServerStackStringsV0(append(current.EvidenceRefs, next.EvidenceRefs...)),
	}
}
