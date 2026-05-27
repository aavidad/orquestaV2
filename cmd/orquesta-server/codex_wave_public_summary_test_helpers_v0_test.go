package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func mustReadCodexWaveCommandSummaryForTest(
	t *testing.T,
	data []byte,
	runtimeDir string,
) codexWaveLaunchSummaryV0 {
	t.Helper()
	var public codexWavePublicSummaryV0
	if err := json.Unmarshal(data, &public); err != nil {
		t.Fatalf("json publico invalido: %v\n%s", err, string(data))
	}
	if public.SchemaVersion != codexWavePublicSummarySchemaVersionV0 {
		t.Fatalf("schema publico=%q", public.SchemaVersion)
	}
	if codexWavePublicSummaryJSONContainsLocalDetailsV0(data, runtimeDir, "/home/", "auth.json", "codex_stdout.log") {
		t.Fatalf("summary publico filtra detalle local: %s", string(data))
	}
	return mustLoadCodexWaveRegistryOrPublicForTest(t, public, runtimeDir)
}

func mustLoadCodexWaveRegistryOrPublicForTest(
	t *testing.T,
	public codexWavePublicSummaryV0,
	runtimeDir string,
) codexWaveLaunchSummaryV0 {
	t.Helper()
	for _, candidate := range codexWaveRegistryCandidateDirsForTestV0(runtimeDir, public.WaveRef) {
		if summary, err := codexWaveLoadRegistryV0(candidate); err == nil {
			return summary
		}
	}
	return codexWaveSummaryFromPublicForTest(public)
}

func codexWaveRegistryCandidateDirsForTestV0(runtimeDir string, waveRef string) []string {
	if runtimeDir == "" {
		return nil
	}
	out := []string{runtimeDir}
	if waveRef != "" && filepath.Base(filepath.Clean(runtimeDir)) != waveRef {
		out = append(out, filepath.Join(runtimeDir, "codex-waves", waveRef))
		out = append(out, filepath.Join(runtimeDir, "children", waveRef))
	}
	return out
}

func codexWaveSummaryFromPublicForTest(public codexWavePublicSummaryV0) codexWaveLaunchSummaryV0 {
	summary := codexWaveLaunchSummaryV0{
		SchemaVersion:  public.SourceSchemaVersion,
		WaveRef:        public.WaveRef,
		AgentCount:     public.AgentCount,
		Sandbox:        public.Sandbox,
		ApprovalPolicy: public.ApprovalPolicy,
		CreatedAt:      public.CreatedAt,
		UpdatedAt:      public.UpdatedAt,
		DryRun:         public.DryRun,
		PurgeReport:    public.PurgeReport,
		OperatorInputs: public.OperatorInputs,
		Agents:         make([]codexWaveAgentSummaryV0, 0, len(public.Agents)),
		Errors:         codexWaveErrorsFromPublicForTest(public.Errors),
	}
	for _, agent := range public.Agents {
		summary.Agents = append(summary.Agents, codexWaveAgentSummaryV0{
			AgentRef:           agent.AgentRef,
			ProcessRef:         agent.ProcessRef,
			ProcessProofRef:    agent.ProcessProofRef,
			ProcessProofDigest: agent.ProcessProofDigest,
			SessionRef:         agent.SessionRef,
			LaunchRef:          agent.LaunchRef,
			StartedAt:          agent.StartedAt,
			StopRequestedAt:    agent.StopRequestedAt,
			StdoutBytes:        agent.StdoutBytes,
			StderrBytes:        agent.StderrBytes,
			LastMessageBytes:   agent.LastMessageBytes,
			Status:             agent.Status,
		})
	}
	return summary
}

func codexWaveErrorsFromPublicForTest(errorsIn []codexWavePublicErrorV0) []codexWavePublicErrorV0 {
	if len(errorsIn) == 0 {
		return nil
	}
	out := make([]codexWavePublicErrorV0, 0, len(errorsIn))
	for _, item := range errorsIn {
		out = append(out, codexWavePublicErrorV0{
			AgentRef: item.AgentRef,
			Code:     item.Code,
			Field:    item.Field,
			Message:  item.Message,
		})
	}
	return out
}

func mustReadCodexDirectorWaveCommandSummaryForTest(
	t *testing.T,
	data []byte,
	rootRuntimeDir string,
) codexDirectorWaveSummaryV0 {
	t.Helper()
	var public codexDirectorWavePublicSummaryV0
	if err := json.Unmarshal(data, &public); err != nil {
		t.Fatalf("json publico invalido: %v\n%s", err, string(data))
	}
	if public.SchemaVersion != codexDirectorWavePublicSummarySchemaVersionV0 {
		t.Fatalf("schema publico=%q", public.SchemaVersion)
	}
	if codexWavePublicSummaryJSONContainsLocalDetailsV0(data, rootRuntimeDir, "/home/", "auth.json", "codex_stdout.log") {
		t.Fatalf("summary director publico filtra detalle local: %s", string(data))
	}
	return codexDirectorWaveSummaryV0{
		SchemaVersion:     public.SourceSchemaVersion,
		Request:           public.Request,
		WorktreeIsolation: public.WorktreeIsolation,
		Plan:              public.Plan,
		WaveWork:          public.WaveWork,
		Launch:            mustLoadCodexWaveRegistryForPublicLaunchForTest(t, public.Launch, rootRuntimeDir),
		ChildLaunches:     mustLoadCodexDirectorChildSummariesForTest(t, public.ChildLaunches, rootRuntimeDir),
		AgentBudget:       public.AgentBudget,
		GuardOptIn:        public.GuardOptIn,
		OperatorInputs:    public.OperatorInputs,
		Issues:            public.Issues,
	}
}

func mustLoadCodexDirectorChildSummariesForTest(
	t *testing.T,
	children []codexDirectorChildWavePublicSummaryV0,
	rootRuntimeDir string,
) []codexDirectorChildWaveSummaryV0 {
	t.Helper()
	if len(children) == 0 {
		return nil
	}
	out := make([]codexDirectorChildWaveSummaryV0, 0, len(children))
	for _, child := range children {
		out = append(out, codexDirectorChildWaveSummaryV0{
			ParentAgentRef:            child.ParentAgentRef,
			ParentWaveRef:             child.ParentWaveRef,
			ParentIndex:               child.ParentIndex,
			DelegationDepth:           child.DelegationDepth,
			MaxDelegationDepth:        child.MaxDelegationDepth,
			MaxSubagentsPerAgent:      child.MaxSubagentsPerAgent,
			SubtreeAgentBudget:        child.SubtreeAgentBudget,
			ReviewRequiredBeforeClose: child.ReviewRequiredBeforeClose,
			Launch:                    mustLoadCodexWaveRegistryForPublicLaunchForTest(t, child.Launch, rootRuntimeDir),
			ChildLaunches:             mustLoadCodexDirectorChildSummariesForTest(t, child.ChildLaunches, rootRuntimeDir),
		})
	}
	return out
}

func mustLoadCodexWaveRegistryForPublicLaunchForTest(
	t *testing.T,
	public codexWavePublicSummaryV0,
	rootRuntimeDir string,
) codexWaveLaunchSummaryV0 {
	t.Helper()
	if public.WaveRef == "" {
		return codexWaveSummaryFromPublicForTest(public)
	}
	runtimeDir := rootRuntimeDir
	if filepath.Base(filepath.Clean(rootRuntimeDir)) != public.WaveRef {
		runtimeDir = filepath.Join(rootRuntimeDir, "children", public.WaveRef)
	}
	return mustLoadCodexWaveRegistryOrPublicForTest(t, public, runtimeDir)
}

func codexWavePublicSummaryJSONContainsLocalDetailsV0(data []byte, values ...string) bool {
	text := string(data)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && strings.Contains(text, value) {
			return true
		}
	}
	return false
}
