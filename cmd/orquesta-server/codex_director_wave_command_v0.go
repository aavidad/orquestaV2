package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

const codexDirectorWaveSummarySchemaVersionV0 = "orquesta_codex_director_wave_launch.v0"

type codexDirectorWaveSummaryV0 struct {
	SchemaVersion     string                                                     `json:"schema_version"`
	Request           orquestadirectoroperativo.OperationalDirectorRequestV0     `json:"request"`
	WorktreeIsolation codexDirectorWorktreeIsolationSummaryV0                    `json:"worktree_isolation,omitempty"`
	Plan              orquestadirectoroperativo.OperationalDirectorPlanV0        `json:"plan"`
	WaveWork          orquestadirectoroperativo.OperationalDirectorWaveWorkV0    `json:"wave_work"`
	Launch            codexWaveLaunchSummaryV0                                   `json:"launch"`
	ChildLaunches     []codexDirectorChildWaveSummaryV0                          `json:"child_launches,omitempty"`
	AgentBudget       orquestadirectoroperativo.OperationalDirectorAgentBudgetV0 `json:"agent_budget"`
	GuardOptIn        codexDirectorGuardOptInSummaryV0                           `json:"guard_opt_in,omitempty"`
	OperatorInputs    []codexWaveOperatorInputReceiptV0                          `json:"operator_inputs,omitempty"`
	Issues            []orquestadirectoroperativo.OperationalDirectorIssueV0     `json:"issues,omitempty"`
}

type codexDirectorChildWaveSummaryV0 struct {
	ParentAgentRef            string                            `json:"parent_agent_ref"`
	ParentWaveRef             string                            `json:"parent_wave_ref,omitempty"`
	ParentIndex               int                               `json:"parent_index"`
	DelegationDepth           int                               `json:"delegation_depth"`
	MaxDelegationDepth        int                               `json:"max_delegation_depth"`
	MaxSubagentsPerAgent      int                               `json:"max_subagents_per_agent"`
	SubtreeAgentBudget        int                               `json:"subtree_agent_budget"`
	ReviewRequiredBeforeClose bool                              `json:"review_required_before_close"`
	Launch                    codexWaveLaunchSummaryV0          `json:"launch"`
	ChildLaunches             []codexDirectorChildWaveSummaryV0 `json:"child_launches,omitempty"`
}

type codexDirectorWaveConfigV0 struct {
	Wave codexWaveConfigV0

	RequestRef                string
	RunRef                    string
	ProjectRef                string
	DomainRefs                []string
	Objective                 string
	WorktreeRef               string
	BranchRef                 string
	WriteSet                  []string
	RequiredTests             []string
	AllowRecursiveDelegation  bool
	MaxDelegationDepth        int
	MaxSubagentsPerAgent      int
	RecursiveAgentBudget      int
	StrictDirectorGuards      bool
	AllowGlobalWriteSet       bool
	AllowPlaceholderTests     bool
	GuardOverrideReason       string
	GuardOverrideEvidenceRefs []string
	DomainContextBlocks       []codexDirectorDomainContextBlockV0
}

type codexDirectorGuardOptInSummaryV0 struct {
	AllowGlobalWriteSet      bool     `json:"allow_global_write_set,omitempty"`
	AllowPlaceholderTests    bool     `json:"allow_placeholder_tests,omitempty"`
	Reason                   string   `json:"reason,omitempty"`
	EvidenceRefs             []string `json:"evidence_refs,omitempty"`
	NoDeleteGuard            bool     `json:"no_delete_guard,omitempty"`
	ProjectBoundaryGuard     bool     `json:"project_boundary_guard,omitempty"`
	DirectorDecisionRequired bool     `json:"director_decision_required,omitempty"`
}

type codexDirectorDomainContextBlockV0 struct {
	SourceRef string
	Text      string
}

type codexDirectorStringListFlagV0 []string

func (flagValue *codexDirectorStringListFlagV0) String() string {
	if flagValue == nil {
		return ""
	}
	return strings.Join(*flagValue, ",")
}

func (flagValue *codexDirectorStringListFlagV0) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*flagValue = append(*flagValue, value)
	return nil
}

func codexLaunchDirectorWaveCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	config, err := codexDirectorWaveConfigFromArgsV0(args, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-launch-director-wave: %v\n", err)
		return 2
	}
	summary, err := runCodexLaunchDirectorWaveV0(context.Background(), config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-launch-director-wave: %v\n", err)
		return 1
	}
	if err := writeCommandJSONOutputV0(stdout, codexDirectorWavePublicSummaryFromV0(summary)); err != nil {
		return reportCommandStdioWriteFailureV0(stderr, "codex-launch-director-wave", "stdout", "json_encode", err)
	}
	if len(summary.Issues) > 0 || len(summary.Launch.Errors) > 0 || codexDirectorChildLaunchHasErrorsV0(summary.ChildLaunches) {
		return 1
	}
	return 0
}

func codexDirectorChildLaunchHasErrorsV0(childLaunches []codexDirectorChildWaveSummaryV0) bool {
	for _, child := range childLaunches {
		if len(child.Launch.Errors) > 0 {
			return true
		}
		if codexDirectorChildLaunchHasErrorsV0(child.ChildLaunches) {
			return true
		}
	}
	return false
}
