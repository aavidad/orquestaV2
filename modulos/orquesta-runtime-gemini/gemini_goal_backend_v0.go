package orquestaruntimegemini

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	GeminiGoalWorkSpecFilePrefixV0 = "gemini_goal_work_spec_"
	GeminiGoalPromptFilePrefixV0   = "gemini_goal_prompt_"

	GeminiGoalResultFileNameV0   = "orquesta_goal_result_v0.json"
	GeminiGoalResultFilePrefixV0 = "orquesta_goal_result_"

	GeminiGoalEvidenceLaunchedV0      = "evidence-ref-gemini-goal-launch-control-files-written"
	GeminiGoalEvidenceResultReadV0    = "evidence-ref-gemini-goal-result-durable-read"
	GeminiGoalEvidenceResultPendingV0 = "evidence-ref-gemini-goal-result-durable-pending"

	ErrGeminiGoalSpecInvalidV0        = "gemini_goal_spec_invalid"
	ErrGeminiGoalRuntimeDirInvalidV0  = "gemini_goal_runtime_dir_invalid"
	ErrGeminiGoalProjectDirInvalidV0  = "gemini_goal_project_dir_invalid"
	ErrGeminiGoalSpecNotFoundV0       = "gemini_goal_spec_not_found"
	ErrGeminiGoalResultInvalidV0      = "gemini_goal_result_invalid"
	ErrGeminiGoalResultReadFailedV0   = "gemini_goal_result_read_failed"
	ErrGeminiGoalControlWriteFailedV0 = "gemini_goal_control_write_failed"
	ErrGeminiGoalControlPathInvalidV0 = "gemini_goal_control_path_invalid"
	ErrGeminiGoalContextCanceledV0    = "gemini_goal_context_canceled"
)

type GeminiGoalBackendV0 struct {
	ProjectWorkDir string
	RuntimeWorkDir string
}

func (backend GeminiGoalBackendV0) LaunchGoalWorkV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return geminiGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrGeminiGoalContextCanceledV0, "context"), err
	}
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:        orquestagoal.GoalStatusInvalidV0,
			GoalRef:       strings.TrimSpace(spec.GoalRef),
			Issues: append([]orquestagoal.GoalWorkIssueV0{{
				Code:  ErrGeminiGoalSpecInvalidV0,
				Field: "spec",
			}}, issues...),
		}, errors.New(ErrGeminiGoalSpecInvalidV0)
	}
	if err := backend.validateDirsV0(); err != nil {
		return geminiGoalInvalidLaunchReceiptV0(spec.GoalRef, err.Error(), "runtime"), err
	}
	if err := os.MkdirAll(backend.RuntimeWorkDir, 0o700); err != nil {
		return geminiGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrGeminiGoalRuntimeDirInvalidV0, "runtime_work_dir"), err
	}
	specPath := filepath.Join(backend.RuntimeWorkDir, geminiGoalSpecFileNameV0(spec.GoalRef))
	if err := writeGeminiJSONFileV0(backend.RuntimeWorkDir, specPath, spec); err != nil {
		return geminiGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrGeminiGoalControlWriteFailedV0, "spec"), err
	}
	prompt := BuildGeminiGoalPromptV0(spec)
	promptPath := filepath.Join(backend.RuntimeWorkDir, geminiGoalPromptFileNameV0(spec.GoalRef))
	if err := writeGeminiControlFileV0(backend.RuntimeWorkDir, promptPath, filepath.Base(promptPath), []byte(prompt), 0o600); err != nil {
		return geminiGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrGeminiGoalControlWriteFailedV0, "prompt"), err
	}
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: geminiGoalExternalRefV0(spec.GoalRef),
		ContextBudget: orquestagoal.NormalizeGoalContextBudgetV0(orquestagoal.GoalContextBudgetV0{
			ContextBudgetTotalBytes: int64(len([]byte(prompt))),
			StaticPromptBytes:       int64(len([]byte(prompt))),
		}),
		EvidenceRefs: []string{GeminiGoalEvidenceLaunchedV0},
	}, nil
}

func (backend GeminiGoalBackendV0) ObserveGoalWorkV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return geminiGoalInvalidResultV0(request, ErrGeminiGoalContextCanceledV0, "context"), err
	}
	request = orquestagoal.NormalizeGoalObservationRequestV0(request)
	if strings.TrimSpace(request.GoalRef) == "" {
		return geminiGoalInvalidResultV0(request, ErrGeminiGoalSpecInvalidV0, "goal_ref"), errors.New(ErrGeminiGoalSpecInvalidV0)
	}
	if err := backend.validateDirsV0(); err != nil {
		return geminiGoalInvalidResultV0(request, err.Error(), "runtime"), err
	}
	spec, err := backend.loadSpecV0(request.GoalRef)
	if err != nil {
		return geminiGoalInvalidResultV0(request, ErrGeminiGoalSpecNotFoundV0, "spec"), err
	}
	result, ok, err := backend.readResultFromWriteSetV0(request, spec)
	if err != nil {
		return geminiGoalInvalidResultV0(request, ErrGeminiGoalResultReadFailedV0, "result"), err
	}
	if !ok {
		return orquestagoal.NormalizeGoalWorkResultV0(orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         request.GoalRef,
			ExternalGoalRef: firstNonEmptyGeminiGoalV0(request.ExternalGoalRef, geminiGoalExternalRefV0(request.GoalRef)),
			Summary:         "gemini_goal_result_pending",
			EvidenceRefs:    []string{GeminiGoalEvidenceResultPendingV0},
		}), nil
	}
	return result, nil
}

func BuildGeminiGoalPromptV0(spec orquestagoal.GoalWorkSpecV0) string {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	var b strings.Builder
	b.WriteString("Eres un backend goal-first Gemini gobernado por Orquesta.\n")
	b.WriteString("Objetivo: ")
	b.WriteString(spec.Objective)
	b.WriteString("\nGoalRef: ")
	b.WriteString(spec.GoalRef)
	b.WriteString("\nTrabaja solo dentro del write-set del proyecto y conserva evidencia compacta.\n")
	writeGeminiDurableResultProtocolV0(&b)
	if len(spec.AcceptanceCriteria) > 0 {
		b.WriteString("Criterios de aceptacion:\n")
		for _, criterion := range spec.AcceptanceCriteria {
			if strings.TrimSpace(criterion) == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(criterion))
			b.WriteString("\n")
		}
	}
	b.WriteString("Write-set permitido:\n")
	for _, scope := range spec.WriteSet {
		b.WriteString("- ")
		b.WriteString(scope.Path)
		if strings.TrimSpace(scope.Purpose) != "" {
			b.WriteString(" :: ")
			b.WriteString(strings.TrimSpace(scope.Purpose))
		}
		b.WriteString("\n")
	}
	if len(spec.ArtifactContracts) > 0 {
		b.WriteString("Contratos de artefacto para artifact_refs del resultado:\n")
		for _, artifact := range spec.ArtifactContracts {
			if strings.TrimSpace(artifact.ArtifactRef) == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(artifact.ArtifactRef))
			if strings.TrimSpace(artifact.ArtifactType) != "" {
				b.WriteString(" :: ")
				b.WriteString(strings.TrimSpace(artifact.ArtifactType))
			}
			if artifact.Required {
				b.WriteString(" :: required=true")
			}
			b.WriteString("\n")
		}
		b.WriteString("Incluye en artifact_refs todos los artifact_ref requeridos que hayas materializado o deja status blocked con rework_plan_refs si falta alguno.\n")
	}
	b.WriteString("Tests requeridos:\n")
	for _, test := range spec.RequiredTests {
		b.WriteString("- ")
		b.WriteString(firstNonEmptyGeminiGoalV0(test.TestRef, test.CommandRef, test.Command))
		b.WriteString("\n")
	}
	if len(spec.ClosurePolicy.RequiredEvidenceRefs) > 0 {
		b.WriteString("Evidencias requeridas para cierre accepted; incluyelas como strings en evidence_refs si el trabajo queda complete:\n")
		for _, evidenceRef := range spec.ClosurePolicy.RequiredEvidenceRefs {
			if strings.TrimSpace(evidenceRef) == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(evidenceRef))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (backend GeminiGoalBackendV0) validateDirsV0() error {
	runtimeDir := filepath.Clean(strings.TrimSpace(backend.RuntimeWorkDir))
	projectDir := filepath.Clean(strings.TrimSpace(backend.ProjectWorkDir))
	if runtimeDir == "" || runtimeDir == "." || !filepath.IsAbs(runtimeDir) {
		return errors.New(ErrGeminiGoalRuntimeDirInvalidV0)
	}
	if projectDir == "" || projectDir == "." || !filepath.IsAbs(projectDir) {
		return errors.New(ErrGeminiGoalProjectDirInvalidV0)
	}
	if runtimeDir == projectDir || geminiPathInsideV0(projectDir, runtimeDir) {
		return errors.New(ErrGeminiGoalControlPathInvalidV0)
	}
	return nil
}

func (backend GeminiGoalBackendV0) loadSpecV0(goalRef string) (orquestagoal.GoalWorkSpecV0, error) {
	path := filepath.Join(backend.RuntimeWorkDir, geminiGoalSpecFileNameV0(goalRef))
	data, err := os.ReadFile(path)
	if err != nil {
		return orquestagoal.GoalWorkSpecV0{}, err
	}
	var spec orquestagoal.GoalWorkSpecV0
	if err := json.Unmarshal(data, &spec); err != nil {
		return orquestagoal.GoalWorkSpecV0{}, err
	}
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	if spec.GoalRef != strings.TrimSpace(goalRef) {
		return orquestagoal.GoalWorkSpecV0{}, errors.New(ErrGeminiGoalSpecInvalidV0)
	}
	return spec, nil
}

func (backend GeminiGoalBackendV0) readResultFromWriteSetV0(
	request orquestagoal.GoalObservationRequestV0,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalWorkResultV0, bool, error) {
	for _, path := range backend.resultCandidatePathsV0(spec) {
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return orquestagoal.GoalWorkResultV0{}, false, err
		}
		var result orquestagoal.GoalWorkResultV0
		if err := json.Unmarshal(normalizeGeminiGoalResultJSONV0(data), &result); err != nil {
			return geminiGoalInvalidResultV0(request, ErrGeminiGoalResultInvalidV0, "result_json"), true, nil
		}
		result = orquestagoal.NormalizeGoalWorkResultV0(result)
		if result.GoalRef != "" && result.GoalRef != request.GoalRef {
			continue
		}
		if result.GoalRef == "" {
			result.GoalRef = request.GoalRef
		}
		if result.ExternalGoalRef == "" {
			result.ExternalGoalRef = firstNonEmptyGeminiGoalV0(request.ExternalGoalRef, geminiGoalExternalRefV0(request.GoalRef))
		}
		result.EvidenceRefs = compactGeminiGoalStringsV0(append(result.EvidenceRefs, GeminiGoalEvidenceResultReadV0))
		return orquestagoal.NormalizeGoalWorkResultV0(result), true, nil
	}
	return orquestagoal.GoalWorkResultV0{}, false, nil
}

func normalizeGeminiGoalResultJSONV0(data []byte) []byte {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return data
	}
	normalized := normalizeGeminiGoalEvidenceRefsJSONValueV0(value)
	out, err := json.Marshal(normalized)
	if err != nil {
		return data
	}
	return out
}

func normalizeGeminiGoalEvidenceRefsJSONValueV0(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			if key == "evidence_refs" {
				out[key] = normalizeGeminiGoalEvidenceRefsJSONArrayV0(nested)
				continue
			}
			out[key] = normalizeGeminiGoalEvidenceRefsJSONValueV0(nested)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, nested := range typed {
			out = append(out, normalizeGeminiGoalEvidenceRefsJSONValueV0(nested))
		}
		return out
	default:
		return value
	}
}

func normalizeGeminiGoalEvidenceRefsJSONArrayV0(value any) any {
	items, ok := value.([]any)
	if !ok {
		return normalizeGeminiGoalEvidenceRefsJSONValueV0(value)
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case string:
			out = append(out, typed)
		case map[string]any:
			if ref, ok := typed["ref"].(string); ok && strings.TrimSpace(ref) != "" {
				out = append(out, strings.TrimSpace(ref))
				continue
			}
			out = append(out, normalizeGeminiGoalEvidenceRefsJSONValueV0(item))
		default:
			out = append(out, normalizeGeminiGoalEvidenceRefsJSONValueV0(item))
		}
	}
	return out
}

func (backend GeminiGoalBackendV0) resultCandidatePathsV0(spec orquestagoal.GoalWorkSpecV0) []string {
	candidates := []string{}
	for _, scope := range spec.WriteSet {
		root := filepath.Join(backend.ProjectWorkDir, filepath.Clean(strings.TrimSpace(scope.Path)))
		candidates = append(candidates, filepath.Join(root, GeminiGoalResultFileNameV0))
		candidates = append(candidates, filepath.Join(root, GeminiGoalResultFilePrefixV0+geminiGoalSafeRefV0(spec.GoalRef)+".json"))
	}
	return compactGeminiGoalStringsV0(candidates)
}

func geminiGoalInvalidLaunchReceiptV0(goalRef string, code string, field string) orquestagoal.GoalLaunchReceiptV0 {
	return orquestagoal.NormalizeGoalLaunchReceiptV0(orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:        orquestagoal.GoalStatusInvalidV0,
		GoalRef:       strings.TrimSpace(goalRef),
		Issues:        []orquestagoal.GoalWorkIssueV0{{Code: code, Field: field}},
	})
}

func geminiGoalInvalidResultV0(
	request orquestagoal.GoalObservationRequestV0,
	code string,
	field string,
) orquestagoal.GoalWorkResultV0 {
	return orquestagoal.NormalizeGoalWorkResultV0(orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusInvalidV0,
		GoalRef:         strings.TrimSpace(request.GoalRef),
		ExternalGoalRef: strings.TrimSpace(request.ExternalGoalRef),
		Issues:          []orquestagoal.GoalWorkIssueV0{{Code: code, Field: field}},
	})
}

func geminiGoalSpecFileNameV0(goalRef string) string {
	return GeminiGoalWorkSpecFilePrefixV0 + geminiGoalSafeRefV0(goalRef) + ".json"
}

func geminiGoalPromptFileNameV0(goalRef string) string {
	return GeminiGoalPromptFilePrefixV0 + geminiGoalSafeRefV0(goalRef) + ".txt"
}

func geminiGoalExternalRefV0(goalRef string) string {
	return "gemini-goal-" + geminiGoalSafeRefV0(goalRef)
}

func geminiGoalSafeRefV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('-')
	}
	out := strings.Trim(b.String(), "-_")
	if out == "" {
		return "unknown"
	}
	return out
}

func compactGeminiGoalStringsV0(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func firstNonEmptyGeminiGoalV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
