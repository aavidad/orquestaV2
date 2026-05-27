package main

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestarails "orquesta/modulos/orquesta-rails"
)

const (
	codexWavePublicSummarySchemaVersionV0         = "orquesta_codex_wave_public_summary.v0"
	codexDirectorWavePublicSummarySchemaVersionV0 = "orquesta_codex_director_wave_public_summary.v0"
)

type codexWavePublicSummaryV0 struct {
	SchemaVersion       string                            `json:"schema_version"`
	SourceSchemaVersion string                            `json:"source_schema_version"`
	WaveRef             string                            `json:"wave_ref"`
	RuntimeRef          string                            `json:"runtime_ref"`
	RegistryRef         string                            `json:"registry_ref,omitempty"`
	AgentCount          int                               `json:"agent_count"`
	StatusCounts        map[string]int                    `json:"status_counts,omitempty"`
	Sandbox             string                            `json:"sandbox"`
	ApprovalPolicy      string                            `json:"approval_policy"`
	CreatedAt           string                            `json:"created_at,omitempty"`
	UpdatedAt           string                            `json:"updated_at,omitempty"`
	DryRun              bool                              `json:"dry_run,omitempty"`
	RedactionLevel      string                            `json:"redaction_level"`
	Freshness           string                            `json:"freshness"`
	DiagnosticsMode     string                            `json:"diagnostics_mode"`
	CanonicalSource     string                            `json:"canonical_source"`
	PurgeReport         *codexWavePurgeReportV0           `json:"purge_report,omitempty"`
	OperatorInputs      []codexWaveOperatorInputReceiptV0 `json:"operator_inputs,omitempty"`
	Agents              []codexWavePublicAgentSummaryV0   `json:"agents"`
	Errors              []codexWavePublicErrorV0          `json:"errors,omitempty"`
}

type codexWavePublicAgentSummaryV0 struct {
	AgentRef             string                                  `json:"agent_ref"`
	RuntimeRef           string                                  `json:"runtime_ref"`
	PromptRef            string                                  `json:"prompt_ref,omitempty"`
	WrapperRef           string                                  `json:"wrapper_ref,omitempty"`
	StdoutRef            string                                  `json:"stdout_ref,omitempty"`
	StderrRef            string                                  `json:"stderr_ref,omitempty"`
	LastMessageRef       string                                  `json:"last_message_ref,omitempty"`
	ProcessRef           string                                  `json:"process_ref,omitempty"`
	ProcessProofRef      string                                  `json:"process_proof_ref,omitempty"`
	ProcessProofDigest   string                                  `json:"process_proof_digest,omitempty"`
	SessionRef           string                                  `json:"session_ref,omitempty"`
	LaunchRef            string                                  `json:"launch_ref,omitempty"`
	StartedAt            string                                  `json:"started_at,omitempty"`
	StopRequestedAt      string                                  `json:"stop_requested_at,omitempty"`
	StdoutBytes          int64                                   `json:"stdout_bytes,omitempty"`
	StderrBytes          int64                                   `json:"stderr_bytes,omitempty"`
	LastMessageBytes     int64                                   `json:"last_message_bytes,omitempty"`
	Status               string                                  `json:"status"`
	CredentialProjection *codexWaveCredentialProjectionReceiptV0 `json:"credential_projection,omitempty"`
}

type codexWavePublicErrorV0 struct {
	AgentRef string `json:"agent_ref,omitempty"`
	Code     string `json:"code"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message,omitempty"`
}

type codexDirectorWavePublicSummaryV0 struct {
	SchemaVersion       string                                                     `json:"schema_version"`
	SourceSchemaVersion string                                                     `json:"source_schema_version"`
	Request             orquestadirectoroperativo.OperationalDirectorRequestV0     `json:"request"`
	WorktreeIsolation   codexDirectorWorktreeIsolationSummaryV0                    `json:"worktree_isolation,omitempty"`
	Plan                orquestadirectoroperativo.OperationalDirectorPlanV0        `json:"plan"`
	WaveWork            orquestadirectoroperativo.OperationalDirectorWaveWorkV0    `json:"wave_work"`
	Launch              codexWavePublicSummaryV0                                   `json:"launch"`
	ChildLaunches       []codexDirectorChildWavePublicSummaryV0                    `json:"child_launches,omitempty"`
	AgentBudget         orquestadirectoroperativo.OperationalDirectorAgentBudgetV0 `json:"agent_budget"`
	GuardOptIn          codexDirectorGuardOptInSummaryV0                           `json:"guard_opt_in,omitempty"`
	OperatorInputs      []codexWaveOperatorInputReceiptV0                          `json:"operator_inputs,omitempty"`
	Issues              []orquestadirectoroperativo.OperationalDirectorIssueV0     `json:"issues,omitempty"`
	RedactionLevel      string                                                     `json:"redaction_level"`
	Freshness           string                                                     `json:"freshness"`
	DiagnosticsMode     string                                                     `json:"diagnostics_mode"`
	CanonicalSource     string                                                     `json:"canonical_source"`
}

type codexDirectorChildWavePublicSummaryV0 struct {
	ParentAgentRef            string                                  `json:"parent_agent_ref"`
	ParentWaveRef             string                                  `json:"parent_wave_ref,omitempty"`
	ParentIndex               int                                     `json:"parent_index"`
	DelegationDepth           int                                     `json:"delegation_depth"`
	MaxDelegationDepth        int                                     `json:"max_delegation_depth"`
	MaxSubagentsPerAgent      int                                     `json:"max_subagents_per_agent"`
	SubtreeAgentBudget        int                                     `json:"subtree_agent_budget"`
	ReviewRequiredBeforeClose bool                                    `json:"review_required_before_close"`
	Launch                    codexWavePublicSummaryV0                `json:"launch"`
	ChildLaunches             []codexDirectorChildWavePublicSummaryV0 `json:"child_launches,omitempty"`
}

func codexWavePublicSummaryFromV0(summary codexWaveLaunchSummaryV0) codexWavePublicSummaryV0 {
	out := codexWavePublicSummaryV0{
		SchemaVersion:       codexWavePublicSummarySchemaVersionV0,
		SourceSchemaVersion: summary.SchemaVersion,
		WaveRef:             summary.WaveRef,
		RuntimeRef:          codexWaveOpaqueRefV0("runtime", summary.WaveRef, summary.RuntimeWorkDir),
		RegistryRef:         codexWaveOpaqueRefV0("registry", summary.WaveRef, summary.RegistryPath),
		AgentCount:          summary.AgentCount,
		StatusCounts:        map[string]int{},
		Sandbox:             summary.Sandbox,
		ApprovalPolicy:      summary.ApprovalPolicy,
		CreatedAt:           summary.CreatedAt,
		UpdatedAt:           summary.UpdatedAt,
		DryRun:              summary.DryRun,
		RedactionLevel:      "refs_only",
		Freshness:           commandPublicFreshnessSnapshotV0,
		DiagnosticsMode:     "use_codex_wave_tail_with_reason",
		CanonicalSource:     commandPublicCanonicalSourceV0,
		PurgeReport:         summary.PurgeReport,
		OperatorInputs:      codexWaveOperatorInputReceiptsCopyV0(summary.OperatorInputs),
		Agents:              make([]codexWavePublicAgentSummaryV0, 0, len(summary.Agents)),
		Errors:              codexWavePublicErrorsFromV0(summary.Errors),
	}
	for _, agent := range summary.Agents {
		out.StatusCounts[agent.Status]++
		out.Agents = append(out.Agents, codexWavePublicAgentFromV0(summary, agent))
	}
	if len(out.StatusCounts) == 0 {
		out.StatusCounts = nil
	}
	return out
}

func codexDirectorWavePublicSummaryFromV0(summary codexDirectorWaveSummaryV0) codexDirectorWavePublicSummaryV0 {
	operatorInputs := codexDirectorWaveOperatorInputsV0(summary)
	request, plan := codexDirectorRedactFileBackedObjectiveV0(summary.Request, summary.Plan, operatorInputs)
	waveWork := codexDirectorRedactWaveWorkFileBackedObjectiveV0(summary.WaveWork, operatorInputs)
	return codexDirectorWavePublicSummaryV0{
		SchemaVersion:       codexDirectorWavePublicSummarySchemaVersionV0,
		SourceSchemaVersion: summary.SchemaVersion,
		Request:             request,
		WorktreeIsolation:   summary.WorktreeIsolation,
		Plan:                plan,
		WaveWork:            waveWork,
		Launch:              codexWavePublicSummaryFromV0(summary.Launch),
		ChildLaunches:       codexDirectorChildWavePublicSummariesFromV0(summary.ChildLaunches),
		AgentBudget:         summary.AgentBudget,
		GuardOptIn:          summary.GuardOptIn,
		OperatorInputs:      codexWaveOperatorInputReceiptsCopyV0(operatorInputs),
		Issues:              append([]orquestadirectoroperativo.OperationalDirectorIssueV0(nil), summary.Issues...),
		RedactionLevel:      "refs_only",
		Freshness:           commandPublicFreshnessSnapshotV0,
		DiagnosticsMode:     "use_codex_wave_tail_with_reason",
		CanonicalSource:     commandPublicCanonicalSourceV0,
	}
}

func codexDirectorRedactWaveWorkFileBackedObjectiveV0(
	waveWork orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	operatorInputs []codexWaveOperatorInputReceiptV0,
) orquestadirectoroperativo.OperationalDirectorWaveWorkV0 {
	receipt, ok := codexWaveOperatorInputHasKindV0(operatorInputs, "objective")
	if !ok || receipt.Origin != "operator_local_file" {
		return waveWork
	}
	for waveIndex := range waveWork.Waves {
		for itemIndex := range waveWork.Waves[waveIndex].Items {
			for criteriaIndex, criteria := range waveWork.Waves[waveIndex].Items[itemIndex].AcceptanceCriteria {
				if strings.HasPrefix(criteria, "Mantener el objetivo actual: ") {
					waveWork.Waves[waveIndex].Items[itemIndex].AcceptanceCriteria[criteriaIndex] = "Mantener el objetivo actual por ref: " + receipt.ContentRef
				}
			}
		}
	}
	return waveWork
}

func codexDirectorWaveOperatorInputsV0(summary codexDirectorWaveSummaryV0) []codexWaveOperatorInputReceiptV0 {
	if len(summary.OperatorInputs) > 0 {
		return codexWaveOperatorInputReceiptsCopyV0(summary.OperatorInputs)
	}
	return codexWaveOperatorInputReceiptsCopyV0(summary.Launch.OperatorInputs)
}

func codexDirectorRedactFileBackedObjectiveV0(
	request orquestadirectoroperativo.OperationalDirectorRequestV0,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	operatorInputs []codexWaveOperatorInputReceiptV0,
) (orquestadirectoroperativo.OperationalDirectorRequestV0, orquestadirectoroperativo.OperationalDirectorPlanV0) {
	receipt, ok := codexWaveOperatorInputHasKindV0(operatorInputs, "objective")
	if !ok || receipt.Origin != "operator_local_file" {
		return request, plan
	}
	redacted := "objective_input_ref:" + receipt.ContentRef
	request.Objective = redacted
	plan.Objective = redacted
	for stepIndex := range plan.Steps {
		for criteriaIndex, criteria := range plan.Steps[stepIndex].AcceptanceCriteria {
			if strings.HasPrefix(criteria, "Mantener el objetivo actual: ") {
				plan.Steps[stepIndex].AcceptanceCriteria[criteriaIndex] = "Mantener el objetivo actual por ref: " + receipt.ContentRef
			}
		}
	}
	return request, plan
}

func codexDirectorChildWavePublicSummariesFromV0(
	children []codexDirectorChildWaveSummaryV0,
) []codexDirectorChildWavePublicSummaryV0 {
	if len(children) == 0 {
		return nil
	}
	out := make([]codexDirectorChildWavePublicSummaryV0, 0, len(children))
	for _, child := range children {
		out = append(out, codexDirectorChildWavePublicSummaryV0{
			ParentAgentRef:            child.ParentAgentRef,
			ParentWaveRef:             child.ParentWaveRef,
			ParentIndex:               child.ParentIndex,
			DelegationDepth:           child.DelegationDepth,
			MaxDelegationDepth:        child.MaxDelegationDepth,
			MaxSubagentsPerAgent:      child.MaxSubagentsPerAgent,
			SubtreeAgentBudget:        child.SubtreeAgentBudget,
			ReviewRequiredBeforeClose: child.ReviewRequiredBeforeClose,
			Launch:                    codexWavePublicSummaryFromV0(child.Launch),
			ChildLaunches:             codexDirectorChildWavePublicSummariesFromV0(child.ChildLaunches),
		})
	}
	return out
}

func codexWavePublicAgentFromV0(
	summary codexWaveLaunchSummaryV0,
	agent codexWaveAgentSummaryV0,
) codexWavePublicAgentSummaryV0 {
	return codexWavePublicAgentSummaryV0{
		AgentRef:             agent.AgentRef,
		PromptRef:            codexWaveOpaqueRefV0("prompt", agent.AgentRef, agent.PromptPath),
		WrapperRef:           codexWaveOpaqueRefV0("wrapper", agent.AgentRef, agent.WrapperPath),
		StdoutRef:            codexWaveOpaqueRefV0("stdout", agent.AgentRef, agent.StdoutPath),
		StderrRef:            codexWaveOpaqueRefV0("stderr", agent.AgentRef, agent.StderrPath),
		LastMessageRef:       codexWaveOpaqueRefV0("last-message", agent.AgentRef, agent.LastMessagePath),
		ProcessRef:           agent.ProcessRef,
		ProcessProofRef:      agent.ProcessProofRef,
		ProcessProofDigest:   agent.ProcessProofDigest,
		SessionRef:           agent.SessionRef,
		LaunchRef:            agent.LaunchRef,
		StartedAt:            agent.StartedAt,
		StopRequestedAt:      agent.StopRequestedAt,
		StdoutBytes:          agent.StdoutBytes,
		StderrBytes:          agent.StderrBytes,
		LastMessageBytes:     agent.LastMessageBytes,
		Status:               agent.Status,
		CredentialProjection: agent.CredentialProjection,
		// Preserve wave scope in refs through the hash input, never as a path.
		RuntimeRef: codexWaveOpaqueRefV0("agent-runtime", summary.WaveRef, agent.RuntimeWorkDir),
	}
}

func codexWavePublicErrorsFromV0(errorsIn []codexWavePublicErrorV0) []codexWavePublicErrorV0 {
	if len(errorsIn) == 0 {
		return nil
	}
	out := make([]codexWavePublicErrorV0, 0, len(errorsIn))
	for _, item := range errorsIn {
		message, _ := orquestarails.RedactOperationalTextForFieldV0("codex_wave_public_summary", item.Field, item.Message)
		out = append(out, codexWavePublicErrorV0{
			AgentRef: item.AgentRef,
			Code:     strings.TrimSpace(item.Code),
			Field:    strings.TrimSpace(item.Field),
			Message:  strings.TrimSpace(message),
		})
	}
	return out
}

func codexWaveOpaqueRefV0(kind string, scope string, value string) string {
	kind = safeCodexWavePurgeRefPartV0(kind)
	scope = safeCodexWavePurgeRefPartV0(scope)
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(filepath.Clean(value)))
	return kind + "-ref-" + scope + "-" + hex.EncodeToString(sum[:])[:16]
}
