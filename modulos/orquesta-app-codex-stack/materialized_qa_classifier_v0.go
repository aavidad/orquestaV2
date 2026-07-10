package orquestaappcodexstack

import (
	"encoding/json"
	"path/filepath"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

type goalMaterializedQAClassificationV0 struct {
	Passed               bool
	Failed               bool
	ValidArtifactPaths   []string
	InvalidArtifactPaths []string
}

// goalMaterializedClassifyQAPayloadV0 only classifies an already-read QA sidecar.
func goalMaterializedClassifyQAPayloadV0(
	projectRoot string,
	path string,
	base string,
	state orquestagoal.GoalWorkStateV0,
	raw []byte,
) goalMaterializedQAClassificationV0 {
	var payload any
	if json.Unmarshal(raw, &payload) != nil {
		return goalMaterializedQAClassificationV0{}
	}
	isOPES := goalMaterializedStateOrReportLooksLikeOPESV0(projectRoot, path, base, state, payload)
	passed := goalMaterializedJSONLooksLikeQAPassForContextV0(projectRoot, path, base, state, payload)
	failed := goalMaterializedJSONLooksLikeQAFailForContextV0(projectRoot, path, base, state, payload)
	classification := goalMaterializedQAClassificationV0{
		Passed: passed,
		Failed: failed,
	}
	if !isOPES && !goalMaterializedJSONLooksLikeQAPassV0(payload) && !goalMaterializedJSONLooksLikeQAFailV0(payload) {
		return classification
	}
	classification.ValidArtifactPaths, classification.InvalidArtifactPaths = goalMaterializedCollectQAArtifactPathListsV0(payload)
	classification.ValidArtifactPaths = goalMaterializedNormalizeQAArtifactPathsV0(classification.ValidArtifactPaths)
	classification.InvalidArtifactPaths = goalMaterializedNormalizeQAArtifactPathsV0(classification.InvalidArtifactPaths)
	return classification
}

func goalMaterializedCollectQAArtifactPathListsV0(value any) ([]string, []string) {
	valid := []string{}
	invalid := []string{}
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			key = goalMaterializedCanonicalQAKeyV0(key)
			switch {
			case goalMaterializedQAArtifactListKeyV0(key, true):
				valid = append(valid, goalMaterializedStringsFromQAArtifactValueV0(item)...)
			case goalMaterializedQAArtifactListKeyV0(key, false):
				invalid = append(invalid, goalMaterializedStringsFromQAArtifactValueV0(item)...)
			default:
				if path := goalMaterializedPathFromQAArtifactObjectV0(typed); path != "" {
					if goalMaterializedQAArtifactObjectValidV0(typed) {
						valid = append(valid, path)
					} else if goalMaterializedQAArtifactObjectInvalidV0(typed) {
						invalid = append(invalid, path)
					}
				}
			}
			nestedValid, nestedInvalid := goalMaterializedCollectQAArtifactPathListsV0(item)
			valid = append(valid, nestedValid...)
			invalid = append(invalid, nestedInvalid...)
		}
	case []any:
		for _, item := range typed {
			nestedValid, nestedInvalid := goalMaterializedCollectQAArtifactPathListsV0(item)
			valid = append(valid, nestedValid...)
			invalid = append(invalid, nestedInvalid...)
		}
	}
	return compactStringsV0(valid), compactStringsV0(invalid)
}

func goalMaterializedQAArtifactListKeyV0(key string, valid bool) bool {
	validKeys := map[string]bool{
		"valid_artifact_paths": true, "valid_artifacts": true, "valid_files": true,
		"artifact_paths_valid": true, "artifact_refs_valid": true,
		"artefactos_validos": true, "archivos_validos": true, "ficheros_validos": true,
	}
	invalidKeys := map[string]bool{
		"invalid_artifact_paths": true, "invalid_artifacts": true, "invalid_files": true,
		"artifact_paths_invalid": true, "artifact_refs_invalid": true,
		"rejected_artifact_paths": true, "rejected_artifacts": true,
		"artefactos_invalidos": true, "archivos_invalidos": true, "ficheros_invalidos": true,
		"artefactos_rechazados": true, "archivos_rechazados": true,
	}
	if valid {
		return validKeys[key]
	}
	return invalidKeys[key]
}

func goalMaterializedStringsFromQAArtifactValueV0(value any) []string {
	out := []string{}
	switch typed := value.(type) {
	case string:
		out = append(out, typed)
	case []any:
		for _, item := range typed {
			out = append(out, goalMaterializedStringsFromQAArtifactValueV0(item)...)
		}
	case map[string]any:
		if path := goalMaterializedPathFromQAArtifactObjectV0(typed); path != "" {
			out = append(out, path)
		}
	}
	return compactStringsV0(out)
}

func goalMaterializedPathFromQAArtifactObjectV0(value map[string]any) string {
	for _, key := range []string{"path", "artifact_path", "file", "file_path", "rel_path", "relative_path", "ruta", "fichero"} {
		if text, ok := value[key].(string); ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
}

func goalMaterializedQAArtifactObjectValidV0(value map[string]any) bool {
	for _, key := range []string{"valid", "passed", "ok"} {
		if item, ok := value[key]; ok && goalMaterializedQAPassValueTruthyV0(item) {
			return true
		}
	}
	if status, ok := value["status"]; ok && goalMaterializedQAPassValueTruthyV0(status) {
		return true
	}
	return false
}

func goalMaterializedQAArtifactObjectInvalidV0(value map[string]any) bool {
	for _, key := range []string{"valid", "passed", "ok"} {
		if item, ok := value[key]; ok && goalMaterializedQAPassValueFalseyV0(item) {
			return true
		}
	}
	if status, ok := value["status"]; ok && goalMaterializedQAPassValueFalseyV0(status) {
		return true
	}
	return false
}

func goalMaterializedNormalizeQAArtifactPathsV0(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
		if path == "" || path == "." || strings.HasPrefix(path, "../") || filepath.IsAbs(path) {
			continue
		}
		out = append(out, path)
	}
	return compactStringsV0(out)
}

func goalMaterializedPathLooksLikeQAReportV0(projectRoot string, path string, base string) bool {
	rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path))
	if err != nil {
		rel = path
	}
	rel = strings.ToLower(filepath.ToSlash(filepath.Clean(rel)))
	base = strings.ToLower(strings.TrimSpace(base))
	return strings.Contains(rel, "09_validacion/") ||
		strings.Contains(rel, "/validacion/") ||
		strings.Contains(base, "qa") ||
		strings.Contains(base, "validacion") ||
		strings.Contains(base, "validation") ||
		strings.Contains(base, "informe")
}

func goalMaterializedJSONLooksLikeQAPassV0(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			key = strings.ToLower(strings.TrimSpace(key))
			switch v := item.(type) {
			case bool:
				if v && (key == "passed" || key == "ok" || strings.HasSuffix(key, "_pass")) {
					return true
				}
			case string:
				text := strings.ToLower(strings.TrimSpace(v))
				if (key == "status" || key == "estado" || strings.HasSuffix(key, "_status")) &&
					(text == "pass" || text == "passed" || text == "ok" || text == "success") {
					return true
				}
				if strings.HasSuffix(key, "_pass") && (text == "true" || text == "ok" || text == "passed") {
					return true
				}
			}
			if goalMaterializedJSONLooksLikeQAPassV0(item) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if goalMaterializedJSONLooksLikeQAPassV0(item) {
				return true
			}
		}
	}
	return false
}

func goalMaterializedJSONLooksLikeQAPassForContextV0(projectRoot string, path string, base string, state orquestagoal.GoalWorkStateV0, value any) bool {
	if goalMaterializedStateOrReportLooksLikeOPESV0(projectRoot, path, base, state, value) {
		return goalMaterializedJSONLooksLikeOPESQAPassV0(value)
	}
	return goalMaterializedJSONLooksLikeQAPassV0(value)
}

func goalMaterializedJSONLooksLikeQAFailForContextV0(projectRoot string, path string, base string, state orquestagoal.GoalWorkStateV0, value any) bool {
	if goalMaterializedStateOrReportLooksLikeOPESV0(projectRoot, path, base, state, value) {
		return goalMaterializedJSONLooksLikeOPESQAFailV0(value)
	}
	return goalMaterializedJSONLooksLikeQAFailV0(value)
}

func goalMaterializedJSONLooksLikeQAFailV0(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			key = goalMaterializedCanonicalQAKeyV0(key)
			switch v := item.(type) {
			case bool:
				if !v && (key == "passed" || key == "ok" || strings.HasSuffix(key, "_pass")) {
					return true
				}
			case string:
				text := strings.ToLower(strings.TrimSpace(v))
				if (key == "status" || key == "estado" || strings.HasSuffix(key, "_status")) &&
					(text == "fail" || text == "failed" || text == "error" || text == "blocked") {
					return true
				}
				if strings.HasSuffix(key, "_pass") && (text == "false" || text == "fail" || text == "failed" || text == "error") {
					return true
				}
			case float64:
				if v == 0 && strings.HasSuffix(key, "_pass") {
					return true
				}
			}
			if goalMaterializedJSONLooksLikeQAFailV0(item) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if goalMaterializedJSONLooksLikeQAFailV0(item) {
				return true
			}
		}
	}
	return false
}

func goalMaterializedStateOrReportLooksLikeOPESV0(projectRoot string, path string, base string, state orquestagoal.GoalWorkStateV0, value any) bool {
	if goalMaterializedTextLooksLikeOPESV0(base) || goalMaterializedTextLooksLikeOPESV0(path) {
		return true
	}
	if rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path)); err == nil && goalMaterializedTextLooksLikeOPESV0(rel) {
		return true
	}
	if goalMaterializedStateLooksLikeOPESV0(state) {
		return true
	}
	return goalMaterializedJSONLooksLikeOPESReportV0(value)
}

func goalMaterializedStateLooksLikeOPESV0(state orquestagoal.GoalWorkStateV0) bool {
	values := []string{state.RunRef, state.GoalRef, state.ExternalGoalRef, state.Spec.GoalRef, state.Spec.RequestRef, state.Spec.RunRef, state.Spec.ProjectRef, state.Spec.DomainRef, state.Spec.WorkKind, state.Spec.WorkProfileKind, state.Spec.Objective}
	values = append(values, state.EvidenceRefs...)
	values = append(values, state.LaunchReceipt.EvidenceRefs...)
	values = append(values, state.Spec.SkillRefs...)
	values = append(values, state.Spec.AcceptanceCriteria...)
	values = append(values, state.Spec.EvidenceRefs...)
	values = append(values, state.Spec.ClosurePolicy.RequiredEvidenceRefs...)
	for _, ref := range state.Spec.ContextRefs {
		values = append(values, ref.Kind, ref.Ref, ref.Purpose)
	}
	for _, ref := range state.Spec.RuleRefs {
		values = append(values, ref.Kind, ref.Ref, ref.Enforcement)
	}
	for _, scope := range state.Spec.WriteSet {
		values = append(values, scope.Path, scope.Purpose)
	}
	for _, test := range state.Spec.RequiredTests {
		values = append(values, test.TestRef, test.CommandRef, test.Command)
		values = append(values, test.AcceptanceCriteria...)
		values = append(values, test.AcceptanceCriteriaRefs...)
		values = append(values, test.EvidenceRefs...)
	}
	for _, contract := range state.Spec.ArtifactContracts {
		values = append(values, contract.ArtifactRef, contract.ArtifactType)
		values = append(values, contract.EvidenceRefs...)
	}
	if state.LastResult != nil {
		values = append(values, state.LastResult.ArtifactRefs...)
		values = append(values, state.LastResult.ArtifactPaths...)
		values = append(values, state.LastResult.DomainReceiptRefs...)
		values = append(values, state.LastResult.EvidenceRefs...)
	}
	for _, value := range values {
		if goalMaterializedTextLooksLikeOPESV0(value) {
			return true
		}
	}
	return false
}

func goalMaterializedJSONLooksLikeOPESReportV0(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if goalMaterializedTextLooksLikeOPESV0(key) {
				return true
			}
			if text, ok := item.(string); ok && goalMaterializedTextLooksLikeOPESV0(text) {
				return true
			}
			if goalMaterializedJSONLooksLikeOPESReportV0(item) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if goalMaterializedJSONLooksLikeOPESReportV0(item) {
				return true
			}
		}
	}
	return false
}

func goalMaterializedTextLooksLikeOPESV0(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return false
	}
	normalized := goalMaterializedCanonicalQAKeyV0(value)
	return normalized == "opes" || strings.HasPrefix(normalized, "opes_") || strings.HasSuffix(normalized, "_opes") || strings.Contains(normalized, "_opes_") || strings.Contains(normalized, "plan_temario")
}

type goalMaterializedOPESQAPassesV0 struct {
	ExtensionPass         bool
	OfficialTextQAPass    bool
	StrictEditorialQAPass bool
}

func goalMaterializedJSONLooksLikeOPESQAPassV0(value any) bool {
	passes := goalMaterializedOPESQAPassesV0{}
	goalMaterializedCollectOPESQAPassesV0(value, &passes)
	return passes.ExtensionPass && passes.OfficialTextQAPass && passes.StrictEditorialQAPass
}

func goalMaterializedJSONLooksLikeOPESQAFailV0(value any) bool {
	return goalMaterializedCollectOPESQAFailV0(value)
}

func goalMaterializedCollectOPESQAFailV0(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if passKind := goalMaterializedOPESQAPassKindForKeyV0(key); passKind != "" && goalMaterializedQAPassValueFalseyV0(item) {
				return true
			}
			if goalMaterializedCollectOPESQAFailV0(item) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if goalMaterializedCollectOPESQAFailV0(item) {
				return true
			}
		}
	}
	return false
}

func goalMaterializedCollectOPESQAPassesV0(value any, passes *goalMaterializedOPESQAPassesV0) {
	if passes == nil || (passes.ExtensionPass && passes.OfficialTextQAPass && passes.StrictEditorialQAPass) {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if passKind := goalMaterializedOPESQAPassKindForKeyV0(key); passKind != "" && (goalMaterializedQAPassValueTruthyV0(item) || goalMaterializedJSONLooksLikeQAPassV0(item)) {
				goalMaterializedMarkOPESQAPassV0(passes, passKind)
			}
			goalMaterializedCollectOPESQAPassesV0(item, passes)
		}
	case []any:
		for _, item := range typed {
			goalMaterializedCollectOPESQAPassesV0(item, passes)
		}
	}
}

func goalMaterializedOPESQAPassKindForKeyV0(key string) string {
	switch goalMaterializedCanonicalQAKeyV0(key) {
	case "extension", "extension_pass", "extension_qa", "extension_qa_pass", "content_extension_pass":
		return "extension"
	case "official_text", "official_text_pass", "official_text_qa", "official_text_qa_pass", "officialtextpass", "officialtextqapass", "texto_oficial_pass", "texto_oficial_qa_pass":
		return "official_text"
	case "strict_editorial", "strict_editorial_pass", "strict_editorial_qa", "strict_editorial_qa_pass", "stricteditorialpass", "stricteditorialqapass", "editorial_estricta_pass", "editorial_estricta_qa_pass":
		return "strict_editorial"
	default:
		return ""
	}
}

func goalMaterializedMarkOPESQAPassV0(passes *goalMaterializedOPESQAPassesV0, passKind string) {
	switch passKind {
	case "extension":
		passes.ExtensionPass = true
	case "official_text":
		passes.OfficialTextQAPass = true
	case "strict_editorial":
		passes.StrictEditorialQAPass = true
	}
}

func goalMaterializedQAPassValueTruthyV0(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		text := strings.ToLower(strings.TrimSpace(typed))
		return text == "true" || text == "pass" || text == "passed" || text == "ok" || text == "success"
	case float64:
		return typed == 1
	default:
		return false
	}
}

func goalMaterializedQAPassValueFalseyV0(value any) bool {
	switch typed := value.(type) {
	case bool:
		return !typed
	case string:
		text := strings.ToLower(strings.TrimSpace(typed))
		return text == "false" || text == "fail" || text == "failed" || text == "error" || text == "blocked"
	case float64:
		return typed == 0
	default:
		return false
	}
}

func goalMaterializedCanonicalQAKeyV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastSep := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastSep = false
		default:
			if !lastSep {
				builder.WriteByte('_')
				lastSep = true
			}
		}
	}
	return strings.Trim(builder.String(), "_")
}
