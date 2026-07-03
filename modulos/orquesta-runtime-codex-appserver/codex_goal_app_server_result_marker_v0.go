package orquestaruntimecodexappserver

import (
	"encoding/json"
	"errors"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

type codexAppServerGoalResultMarkerV0 struct {
	SchemaVersion         string                                    `json:"schema_version,omitempty"`
	Status                string                                    `json:"status,omitempty"`
	Estado                string                                    `json:"estado,omitempty"`
	GoalRef               string                                    `json:"goal_ref,omitempty"`
	ExternalGoalRef       string                                    `json:"external_goal_ref,omitempty"`
	Summary               string                                    `json:"summary,omitempty"`
	ArtifactRefs          []string                                  `json:"artifact_refs,omitempty"`
	ArtifactPaths         []string                                  `json:"artifact_paths,omitempty"`
	MaterializedArtifacts []orquestagoal.GoalMaterializedArtifactV0 `json:"materialized_artifacts,omitempty"`
	Checklist             orquestagoal.GoalWorkChecklistV0          `json:"checklist,omitempty"`
	MissingRefs           []string                                  `json:"missing_refs,omitempty"`
	RequiredTestResults   []orquestagoal.GoalRequiredTestResultV0   `json:"required_test_results,omitempty"`
	DomainReceiptRefs     []string                                  `json:"domain_receipt_refs,omitempty"`
	ReworkPlanRefs        []string                                  `json:"rework_plan_refs,omitempty"`
	EvidenceRefs          []string                                  `json:"evidence_refs,omitempty"`
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
	var marked codexAppServerGoalResultMarkerV0
	if err := json.Unmarshal([]byte(payload), &marked); err != nil {
		return codexAppServerGoalResultMarkerV0{}, true, err
	}
	marked = normalizeCodexAppServerGoalResultMarkerV0(marked)
	if issue := codexAppServerGoalResultContractIssueCodeV0(marked); issue != "" {
		return marked, true, codexAppServerGoalResultContractErrorV0{Code: issue}
	}
	return marked, true, nil
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
	return markedGoalRef != "" && markedGoalRef != strings.TrimSpace(goalRef)
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
