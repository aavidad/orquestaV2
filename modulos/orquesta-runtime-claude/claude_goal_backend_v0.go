package orquestaruntimeclaude

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
	ClaudeGoalWorkSpecFilePrefixV0 = "claude_goal_work_spec_"
	ClaudeGoalPromptFilePrefixV0   = "claude_goal_prompt_"

	ClaudeGoalResultFileNameV0   = "orquesta_goal_result_v0.json"
	ClaudeGoalResultFilePrefixV0 = "orquesta_goal_result_"

	ClaudeGoalEvidenceLaunchedV0      = "evidence-ref-claude-goal-launch-control-files-written"
	ClaudeGoalEvidenceResultReadV0    = "evidence-ref-claude-goal-result-durable-read"
	ClaudeGoalEvidenceResultPendingV0 = "evidence-ref-claude-goal-result-durable-pending"

	ErrClaudeGoalSpecInvalidV0        = "claude_goal_spec_invalid"
	ErrClaudeGoalRuntimeDirInvalidV0  = "claude_goal_runtime_dir_invalid"
	ErrClaudeGoalProjectDirInvalidV0  = "claude_goal_project_dir_invalid"
	ErrClaudeGoalSpecNotFoundV0       = "claude_goal_spec_not_found"
	ErrClaudeGoalResultInvalidV0      = "claude_goal_result_invalid"
	ErrClaudeGoalResultReadFailedV0   = "claude_goal_result_read_failed"
	ErrClaudeGoalControlWriteFailedV0 = "claude_goal_control_write_failed"
	ErrClaudeGoalControlPathInvalidV0 = "claude_goal_control_path_invalid"
	ErrClaudeGoalContextCanceledV0    = "claude_goal_context_canceled"
)

type ClaudeGoalBackendV0 struct {
	ProjectWorkDir string
	RuntimeWorkDir string
}

func (backend ClaudeGoalBackendV0) LaunchGoalWorkV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return claudeGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrClaudeGoalContextCanceledV0, "context"), err
	}
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:        orquestagoal.GoalStatusInvalidV0,
			GoalRef:       strings.TrimSpace(spec.GoalRef),
			Issues: append([]orquestagoal.GoalWorkIssueV0{{
				Code:  ErrClaudeGoalSpecInvalidV0,
				Field: "spec",
			}}, issues...),
		}, errors.New(ErrClaudeGoalSpecInvalidV0)
	}
	if err := backend.validateDirsV0(); err != nil {
		return claudeGoalInvalidLaunchReceiptV0(spec.GoalRef, err.Error(), "runtime"), err
	}
	if err := os.MkdirAll(backend.RuntimeWorkDir, 0o700); err != nil {
		return claudeGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrClaudeGoalRuntimeDirInvalidV0, "runtime_work_dir"), err
	}
	specPath := filepath.Join(backend.RuntimeWorkDir, claudeGoalSpecFileNameV0(spec.GoalRef))
	if err := writeClaudeJSONFileV0(backend.RuntimeWorkDir, specPath, spec); err != nil {
		return claudeGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrClaudeGoalControlWriteFailedV0, "spec"), err
	}
	prompt := BuildClaudeGoalPromptV0(spec)
	promptPath := filepath.Join(backend.RuntimeWorkDir, claudeGoalPromptFileNameV0(spec.GoalRef))
	if err := writeClaudeControlFileV0(backend.RuntimeWorkDir, promptPath, filepath.Base(promptPath), []byte(prompt), 0o600); err != nil {
		return claudeGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrClaudeGoalControlWriteFailedV0, "prompt"), err
	}
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: claudeGoalExternalRefV0(spec.GoalRef),
		ContextBudget: orquestagoal.NormalizeGoalContextBudgetV0(orquestagoal.GoalContextBudgetV0{
			ContextBudgetTotalBytes: int64(len([]byte(prompt))),
			StaticPromptBytes:       int64(len([]byte(prompt))),
		}),
		EvidenceRefs: []string{ClaudeGoalEvidenceLaunchedV0},
	}, nil
}

func (backend ClaudeGoalBackendV0) ObserveGoalWorkV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return claudeGoalInvalidResultV0(request, ErrClaudeGoalContextCanceledV0, "context"), err
	}
	request = orquestagoal.NormalizeGoalObservationRequestV0(request)
	if strings.TrimSpace(request.GoalRef) == "" {
		return claudeGoalInvalidResultV0(request, ErrClaudeGoalSpecInvalidV0, "goal_ref"), errors.New(ErrClaudeGoalSpecInvalidV0)
	}
	if err := backend.validateDirsV0(); err != nil {
		return claudeGoalInvalidResultV0(request, err.Error(), "runtime"), err
	}
	spec, err := backend.loadSpecV0(request.GoalRef)
	if err != nil {
		return claudeGoalInvalidResultV0(request, ErrClaudeGoalSpecNotFoundV0, "spec"), err
	}
	result, ok, err := backend.readResultFromWriteSetV0(request, spec)
	if err != nil {
		return claudeGoalInvalidResultV0(request, ErrClaudeGoalResultReadFailedV0, "result"), err
	}
	if !ok {
		return orquestagoal.NormalizeGoalWorkResultV0(orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         request.GoalRef,
			ExternalGoalRef: firstNonEmptyClaudeGoalV0(request.ExternalGoalRef, claudeGoalExternalRefV0(request.GoalRef)),
			Summary:         "claude_goal_result_pending",
			EvidenceRefs:    []string{ClaudeGoalEvidenceResultPendingV0},
		}), nil
	}
	return result, nil
}

func BuildClaudeGoalPromptV0(spec orquestagoal.GoalWorkSpecV0) string {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	var b strings.Builder
	b.WriteString("Eres un backend goal-first Claude gobernado por Orquesta.\n")
	b.WriteString("Objetivo: ")
	b.WriteString(spec.Objective)
	b.WriteString("\nGoalRef: ")
	b.WriteString(spec.GoalRef)
	b.WriteString("\nTrabaja solo dentro del write-set del proyecto y conserva evidencia compacta.\n")
	writeClaudeDurableResultProtocolV0(&b)
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
	b.WriteString("Tests requeridos:\n")
	for _, test := range spec.RequiredTests {
		b.WriteString("- ")
		b.WriteString(firstNonEmptyClaudeGoalV0(test.TestRef, test.CommandRef, test.Command))
		b.WriteString("\n")
	}
	return b.String()
}

func (backend ClaudeGoalBackendV0) validateDirsV0() error {
	runtimeDir := filepath.Clean(strings.TrimSpace(backend.RuntimeWorkDir))
	projectDir := filepath.Clean(strings.TrimSpace(backend.ProjectWorkDir))
	if runtimeDir == "" || runtimeDir == "." || !filepath.IsAbs(runtimeDir) {
		return errors.New(ErrClaudeGoalRuntimeDirInvalidV0)
	}
	if projectDir == "" || projectDir == "." || !filepath.IsAbs(projectDir) {
		return errors.New(ErrClaudeGoalProjectDirInvalidV0)
	}
	if runtimeDir == projectDir || claudePathInsideV0(projectDir, runtimeDir) {
		return errors.New(ErrClaudeGoalControlPathInvalidV0)
	}
	return nil
}

func (backend ClaudeGoalBackendV0) loadSpecV0(goalRef string) (orquestagoal.GoalWorkSpecV0, error) {
	path := filepath.Join(backend.RuntimeWorkDir, claudeGoalSpecFileNameV0(goalRef))
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
		return orquestagoal.GoalWorkSpecV0{}, errors.New(ErrClaudeGoalSpecInvalidV0)
	}
	return spec, nil
}

func (backend ClaudeGoalBackendV0) readResultFromWriteSetV0(
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
		if err := json.Unmarshal(data, &result); err != nil {
			return claudeGoalInvalidResultV0(request, ErrClaudeGoalResultInvalidV0, "result_json"), true, nil
		}
		result = orquestagoal.NormalizeGoalWorkResultV0(result)
		if result.GoalRef != "" && result.GoalRef != request.GoalRef {
			continue
		}
		if result.GoalRef == "" {
			result.GoalRef = request.GoalRef
		}
		if result.ExternalGoalRef == "" {
			result.ExternalGoalRef = firstNonEmptyClaudeGoalV0(request.ExternalGoalRef, claudeGoalExternalRefV0(request.GoalRef))
		}
		result.EvidenceRefs = compactClaudeGoalStringsV0(append(result.EvidenceRefs, ClaudeGoalEvidenceResultReadV0))
		return orquestagoal.NormalizeGoalWorkResultV0(result), true, nil
	}
	return orquestagoal.GoalWorkResultV0{}, false, nil
}

func claudeGoalInvalidLaunchReceiptV0(goalRef string, code string, field string) orquestagoal.GoalLaunchReceiptV0 {
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:        orquestagoal.GoalStatusInvalidV0,
		GoalRef:       strings.TrimSpace(goalRef),
		Issues:        []orquestagoal.GoalWorkIssueV0{{Code: strings.TrimSpace(code), Field: strings.TrimSpace(field)}},
	}
}

func claudeGoalInvalidResultV0(request orquestagoal.GoalObservationRequestV0, code string, field string) orquestagoal.GoalWorkResultV0 {
	return orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusInvalidV0,
		GoalRef:         strings.TrimSpace(request.GoalRef),
		ExternalGoalRef: strings.TrimSpace(request.ExternalGoalRef),
		Summary:         strings.TrimSpace(code),
		Issues:          []orquestagoal.GoalWorkIssueV0{{Code: strings.TrimSpace(code), Field: strings.TrimSpace(field)}},
	}
}
