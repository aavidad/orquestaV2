package main

import (
	"os"
	"path/filepath"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func startupAutoprogrammingRunNeedsFreshAttemptV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	runtimeWorkDir string,
) bool {
	if orquestaappcodexstack.AutoprogrammingRunNeedsFreshAttemptWithRuntimeV0(run, runtimeWorkDir) {
		return true
	}
	if !strings.HasPrefix(strings.TrimSpace(run.RunID), "request-ref-autoprogramming-backlog-") ||
		run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!startupAutoprogrammingRunHasFailureSignalV0(run) {
		return false
	}
	pending := startupAutoprogrammingPendingAgentRefsV0(run)
	return len(pending) > 0 && !startupAutoprogrammingRuntimeDirForAnyAgentV0(runtimeWorkDir, run.RunID, pending)
}

func startupAutoprogrammingRunHasFailureSignalV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	return len(compactStringsV0(run.FailedAgents)) > 0 ||
		len(compactStringsV0(run.LostAgents)) > 0 ||
		len(compactStringsV0(run.StoppedAgents)) > 0 ||
		len(compactStringsV0(run.AgentAssessments)) > 0
}

func startupAutoprogrammingPendingAgentRefsV0(run orquestacoreworkflow.OrchestrationRunV0) []string {
	inactive := map[string]bool{}
	for _, refs := range [][]string{run.DeliveredAgents, run.FailedAgents, run.LostAgents, run.StoppedAgents, run.ConfirmedStoppedAgents} {
		for _, ref := range compactStringsV0(refs) {
			inactive[ref] = true
		}
	}
	out := make([]string, 0)
	for _, ref := range compactStringsV0(run.StartedAgents) {
		if !inactive[ref] {
			out = append(out, ref)
		}
	}
	return out
}

func startupAutoprogrammingRuntimeDirForAnyAgentV0(runtimeWorkDir string, runRef string, agentRefs []string) bool {
	for _, agentRef := range compactStringsV0(agentRefs) {
		info, err := os.Stat(filepath.Join(strings.TrimSpace(runtimeWorkDir), strings.TrimSpace(runRef), agentRef))
		if err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

func startupAutoprogrammingRunDocumentExistsV0(rootDir string, runRef string) bool {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return false
	}
	files, err := filepath.Glob(filepath.Join(strings.TrimSpace(rootDir), "runs", "*.json"))
	if err != nil {
		return false
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err == nil && strings.Contains(string(data), runRef) {
			return true
		}
	}
	return false
}

func compactStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
