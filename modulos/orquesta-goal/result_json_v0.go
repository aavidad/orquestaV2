package orquestagoal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

const (
	GoalWorkResultJSONDispositionCanonicalV0     = "canonical"
	GoalWorkResultJSONDispositionRepairedV0      = "repaired"
	GoalWorkResultJSONDispositionIrrecoverableV0 = "irrecoverable"
	goalWorkResultJSONRepairedEvidenceRefV0      = "evidence-ref-goal-result-json-repaired"
)

// DecodeGoalWorkResultJSONV0 accepts the canonical receipt plus known
// provider-shaped aliases. It never uses free text to decide closure: aliases
// are recorded in a durable receipt and causal identity remains an adapter
// boundary (the adapter owns the expected goal/external refs).
func DecodeGoalWorkResultJSONV0(raw []byte) (GoalWorkResultJSONDecodeV0, error) {
	hash := sha256.Sum256(raw)
	decode := GoalWorkResultJSONDecodeV0{Disposition: GoalWorkResultJSONDispositionCanonicalV0}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return goalWorkResultJSONIrrecoverableV0(hash, "result_json"), err
	}
	root, ok := value.(map[string]any)
	if !ok {
		return goalWorkResultJSONIrrecoverableV0(hash, "result_json"), nil
	}
	repair := &GoalWorkResultRepairReceiptV0{
		SchemaVersion:   GoalWorkResultRepairReceiptSchemaV0,
		OriginalRefHash: hex.EncodeToString(hash[:]),
	}
	if schema, ok := goalWorkResultJSONStringV0(root["schema_version"]); ok {
		repair.OriginalSchemaVersion = schema
	}
	goalWorkResultJSONApplyAliasesV0(root, repair)
	goalWorkResultJSONNormalizeEvidenceV0(root, "", repair)
	goalWorkResultJSONNormalizeIssuesV0(root, repair)
	encoded, err := json.Marshal(root)
	if err != nil {
		return goalWorkResultJSONIrrecoverableV0(hash, "result_json"), err
	}
	var result GoalWorkResultV0
	if err := json.Unmarshal(encoded, &result); err != nil {
		return goalWorkResultJSONIrrecoverableV0(hash, "result_json"), err
	}
	result = NormalizeGoalWorkResultV0(result)
	if len(repair.Transformations) > 0 {
		repair.EvidenceRefs = []string{goalWorkResultJSONRepairedEvidenceRefV0}
		result.RepairReceipt = repair
		result.EvidenceRefs = compactGoalStringsV0(append(result.EvidenceRefs, repair.EvidenceRefs...))
		decode.Disposition = GoalWorkResultJSONDispositionRepairedV0
		decode.RepairReceipt = repair
	}
	if result.Status == "" {
		decode.Disposition = GoalWorkResultJSONDispositionIrrecoverableV0
		decode.Issues = []GoalWorkIssueV0{{Code: ErrGoalResultJSONInvalidV0, Field: "status"}}
		if decode.RepairReceipt == nil {
			decode.RepairReceipt = repair
		}
		return decode, nil
	}
	decode.Result = result
	return decode, nil
}

func goalWorkResultJSONIrrecoverableV0(hash [sha256.Size]byte, field string) GoalWorkResultJSONDecodeV0 {
	return GoalWorkResultJSONDecodeV0{
		Disposition: GoalWorkResultJSONDispositionIrrecoverableV0,
		Issues:      []GoalWorkIssueV0{{Code: ErrGoalResultJSONInvalidV0, Field: field}},
		RepairReceipt: &GoalWorkResultRepairReceiptV0{
			SchemaVersion:   GoalWorkResultRepairReceiptSchemaV0,
			OriginalRefHash: hex.EncodeToString(hash[:]),
		},
	}
}

func goalWorkResultJSONStringV0(value any) (string, bool) {
	text, ok := value.(string)
	return strings.TrimSpace(text), ok && strings.TrimSpace(text) != ""
}

func goalWorkResultJSONApplyAliasesV0(root map[string]any, receipt *GoalWorkResultRepairReceiptV0) {
	goalWorkResultJSONAliasV0(root, receipt, "schema_version", "schema", "version")
	goalWorkResultJSONAliasV0(root, receipt, "status", "estado", "state")
	goalWorkResultJSONAliasV0(root, receipt, "goal_ref", "goal_id", "goal")
	goalWorkResultJSONAliasV0(root, receipt, "external_goal_ref", "external_goal_id")
	goalWorkResultJSONAliasV0(root, receipt, "required_test_results", "required_tests", "test_results")
	if missing, present := root["missing_refs"]; present {
		checklist, _ := root["checklist"].(map[string]any)
		if checklist == nil {
			checklist = map[string]any{}
			root["checklist"] = checklist
		}
		checklist["missing_refs"] = goalWorkResultJSONMergeRefsV0(checklist["missing_refs"], missing)
		delete(root, "missing_refs")
		goalWorkResultJSONTransformationV0(receipt, "checklist.missing_refs", "alias_missing_refs")
	}
	if reason, ok := goalWorkResultJSONStringV0(root["reason_code"]); ok {
		var issues []any
		switch typed := root["issues"].(type) {
		case []any:
			issues = typed
		case string:
			issues = []any{typed}
		}
		issues = append(issues, map[string]any{"code": reason, "field": "reason_code"})
		root["issues"] = issues
		delete(root, "reason_code")
		goalWorkResultJSONTransformationV0(receipt, "issues", "reason_code_issue")
	}
	if status, ok := goalWorkResultJSONStringV0(root["status"]); ok {
		canonical := goalWorkResultJSONStatusAliasV0(status)
		if canonical != status {
			root["status"] = canonical
			goalWorkResultJSONTransformationV0(receipt, "status", "status_alias")
		}
	}
	if tests, ok := root["required_test_results"].([]any); ok {
		for _, item := range tests {
			test, ok := item.(map[string]any)
			if !ok {
				continue
			}
			status, ok := goalWorkResultJSONStringV0(test["status"])
			if !ok {
				continue
			}
			canonical := goalWorkResultJSONRequiredTestStatusAliasV0(status)
			if canonical != status {
				test["status"] = canonical
				goalWorkResultJSONTransformationV0(receipt, "required_test_results.status", "status_alias")
			}
		}
	}
}

func goalWorkResultJSONMergeRefsV0(values ...any) []any {
	var out []any
	seen := map[string]bool{}
	for _, value := range values {
		var items []any
		switch typed := value.(type) {
		case []any:
			items = typed
		case string:
			items = []any{typed}
		}
		for _, item := range items {
			ref, ok := goalWorkResultJSONStringV0(item)
			if !ok || seen[ref] {
				continue
			}
			seen[ref] = true
			out = append(out, ref)
		}
	}
	return out
}

func goalWorkResultJSONAliasV0(root map[string]any, receipt *GoalWorkResultRepairReceiptV0, canonical string, aliases ...string) {
	if value, ok := root[canonical]; ok && value != nil {
		return
	}
	for _, alias := range aliases {
		if value, ok := root[alias]; ok {
			root[canonical] = value
			delete(root, alias)
			goalWorkResultJSONTransformationV0(receipt, canonical, "alias_"+alias)
			return
		}
	}
}

func goalWorkResultJSONStatusAliasV0(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "done", "passed", "pass", "success", "succeeded":
		return GoalStatusCompleteV0
	case "in_progress", "in-progress", "pending":
		return GoalStatusRunningV0
	case "failed", "failure", "error":
		return GoalStatusInvalidV0
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func goalWorkResultJSONRequiredTestStatusAliasV0(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "complete", "done", "pass", "success", "succeeded":
		return "passed"
	case "failure", "error":
		return "failed"
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func goalWorkResultJSONNormalizeEvidenceV0(value any, path string, receipt *GoalWorkResultRepairReceiptV0) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			field := key
			if path != "" {
				field = path + "." + key
			}
			if key == "evidence_refs" {
				typed[key] = goalWorkResultJSONEvidenceRefsV0(nested, field, receipt)
				continue
			}
			goalWorkResultJSONNormalizeEvidenceV0(nested, field, receipt)
		}
	case []any:
		for _, nested := range typed {
			goalWorkResultJSONNormalizeEvidenceV0(nested, path, receipt)
		}
	}
}

func goalWorkResultJSONEvidenceRefsV0(value any, field string, receipt *GoalWorkResultRepairReceiptV0) any {
	var items []any
	switch typed := value.(type) {
	case nil:
		return value
	case []any:
		items = typed
	case string:
		items = []any{typed}
		goalWorkResultJSONTransformationV0(receipt, goalWorkResultJSONReceiptFieldV0(field), "evidence_string")
	case map[string]any:
		items = []any{typed}
	default:
		goalWorkResultJSONTransformationV0(receipt, goalWorkResultJSONReceiptFieldV0(field), "evidence_value_dropped")
		return []any{}
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case string:
			out = append(out, typed)
		case map[string]any:
			if ref, ok := goalWorkResultJSONStringV0(typed["ref"]); ok {
				out = append(out, ref)
				goalWorkResultJSONTransformationV0(receipt, goalWorkResultJSONReceiptFieldV0(field), "evidence_object_ref")
			} else {
				goalWorkResultJSONTransformationV0(receipt, goalWorkResultJSONReceiptFieldV0(field), "evidence_object_dropped")
			}
		default:
			goalWorkResultJSONTransformationV0(receipt, goalWorkResultJSONReceiptFieldV0(field), "evidence_value_dropped")
		}
	}
	return out
}

func goalWorkResultJSONReceiptFieldV0(field string) string {
	switch {
	case field == "evidence_refs":
		return field
	case strings.HasPrefix(field, "checklist."):
		return "checklist.evidence_refs"
	case strings.HasPrefix(field, "required_test_results."):
		return "required_test_results.evidence_refs"
	case strings.HasPrefix(field, "materialized_artifacts."):
		return "materialized_artifacts.evidence_refs"
	default:
		return "evidence_refs"
	}
}

func goalWorkResultJSONNormalizeIssuesV0(root map[string]any, receipt *GoalWorkResultRepairReceiptV0) {
	goalWorkResultJSONNormalizeIssueListV0(root, "issues", receipt)
	if artifacts, ok := root["materialized_artifacts"].([]any); ok {
		for _, artifact := range artifacts {
			if item, ok := artifact.(map[string]any); ok {
				goalWorkResultJSONNormalizeIssueListV0(item, "issues", receipt)
			}
		}
	}
}

func goalWorkResultJSONNormalizeIssueListV0(root map[string]any, key string, receipt *GoalWorkResultRepairReceiptV0) {
	var items []any
	switch typed := root[key].(type) {
	case nil:
		return
	case []any:
		items = typed
	case string:
		items = []any{typed}
	default:
		return
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case string:
			out = append(out, map[string]any{"code": goalWorkResultJSONIssueCodeAliasV0(typed)})
			goalWorkResultJSONTransformationV0(receipt, key, "issue_string")
		case map[string]any:
			if _, exists := typed["code"]; !exists {
				if reason, ok := typed["reason"]; ok {
					typed["code"] = reason
					delete(typed, "reason")
					goalWorkResultJSONTransformationV0(receipt, key, "issue_reason_alias")
				}
			}
			if code, ok := goalWorkResultJSONStringV0(typed["code"]); ok {
				canonical := goalWorkResultJSONIssueCodeAliasV0(code)
				if canonical != code {
					typed["code"] = canonical
					goalWorkResultJSONTransformationV0(receipt, key, "issue_code_alias")
				}
			}
			out = append(out, typed)
		}
	}
	root[key] = out
}

func goalWorkResultJSONIssueCodeAliasV0(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	switch code {
	case "external_test_environment_restriction", "test_environment_unavailable":
		return GoalIssueRequiredTestsEnvironmentUnavailableV0
	default:
		return code
	}
}

func goalWorkResultJSONTransformationV0(receipt *GoalWorkResultRepairReceiptV0, field, kind string) {
	for _, existing := range receipt.Transformations {
		if existing.Field == field && existing.Kind == kind {
			return
		}
	}
	receipt.Transformations = append(receipt.Transformations, GoalWorkResultRepairTransformationV0{Field: field, Kind: kind})
}
