package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type reasoningEffortLaunchJournal struct {
	SchemaVersion   int                        `json:"schema_version"`
	RequestHash     string                     `json:"request_hash"`
	ExecutionRef    string                     `json:"execution_ref"`
	GoalRef         string                     `json:"goal_ref"`
	WorkItemRef     string                     `json:"work_item_ref"`
	ModelRef        string                     `json:"model_ref"`
	ReasoningEffort governance.ReasoningEffort `json:"reasoning_effort"`
}

func TestReasoningEffortXHighSurvivesProductionCompositionAndRestart(t *testing.T) {
	root := t.TempDir()
	invocationsPath := filepath.Join(root, "codex-invocations")
	argumentsPath := filepath.Join(root, "codex-arguments")
	helperPath := filepath.Join(root, "codex-helper.sh")
	writeReasoningEffortHelper(t, helperPath, invocationsPath, argumentsPath)

	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[runtime.codex]\ntimeout = \"1s\"",
		"[runtime.codex]\ncommand = "+strconv.Quote(helperPath)+"\nreasoning = \"high\"\ntimeout = \"5s\"",
	)
	configureTestCodexRuntimeCgroup(t, configPath)

	first := buildReasoningEffortRuntime(t, configPath)
	t.Cleanup(func() {
		shutdownReasoningEffortRuntime(t, first)
	})
	if err := first.Start(context.Background()); err != nil {
		t.Fatalf("start first production composition: %v", err)
	}
	result, err := first.Orchestrator().Submit(
		context.Background(),
		testRuntimeAccess(t, first),
		application.SubmitRequest{
			RequestRef: "request:bug459-v7-evidence",
			Statement:  "prove exact xhigh propagation",
			Confirm:    true,
			Plan: &application.PlanSpec{
				Phases: []application.PhaseSpec{{
					Ref: "phase-instance:bug459-v7", Key: goal.DefaultPhaseKey().String(),
					TemplateRef: "phase-template:default",
				}},
				WorkItems: []application.WorkItemSpec{{
					Key: "work:bug459-v7", Objective: "produce BUG459 V7 evidence",
					Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(),
					OutputContract:  goal.OutputContractEvidenceBundle,
					ReasoningEffort: governance.ReasoningEffortXHigh,
				}},
			},
		},
	)
	if err != nil {
		t.Fatalf("submit xhigh Goal: %v", err)
	}
	terminal := waitTerminalGoal(t, first, result.Record.Goal.Ref())
	request := requireReasoningEffortCausality(t, terminal)
	requirePhysicalReasoningEffort(t, argumentsPath, governance.ReasoningEffortXHigh)
	requireReasoningEffortLaunchCount(t, invocationsPath, 1)

	journalPath, journalPayload, journal := readReasoningEffortJournal(t, root)
	if journal.SchemaVersion != 7 ||
		journal.ExecutionRef != request.ExecutionRef.String() ||
		journal.GoalRef != request.GoalRef.String() ||
		journal.WorkItemRef != request.WorkItemRef.String() ||
		journal.ReasoningEffort != governance.ReasoningEffortXHigh ||
		!strings.HasPrefix(journal.RequestHash, "sha256:") {
		t.Fatalf("V7 launch journal lost xhigh causality: %+v", journal)
	}
	providerReceiptRef := "codex-launch:" + journal.RequestHash
	requireReasoningEffortReceiptBinding(t, terminal, request.ExecutionRef, providerReceiptRef)

	shutdownReasoningEffortRuntime(t, first)

	second := buildReasoningEffortRuntime(t, configPath)
	t.Cleanup(func() {
		shutdownReasoningEffortRuntime(t, second)
	})
	if err := second.Start(context.Background()); err != nil {
		t.Fatalf("start restarted production composition: %v", err)
	}
	replayed, err := second.agent.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("replay exact V7 launch after restart: %v", err)
	}
	if replayed.ReceiptRef != providerReceiptRef || replayed.ModelRef != journal.ModelRef {
		t.Fatalf("replayed receipt is not bound to V7 request: %+v journal=%+v", replayed, journal)
	}
	requireReasoningEffortLaunchCount(t, invocationsPath, 1)

	conflicting := request
	conflicting.ReasoningEffort = governance.ReasoningEffortHigh
	if _, err := second.agent.Launch(context.Background(), conflicting); codex.ErrorCode(err) != codex.CodeExecutionConflict {
		t.Fatalf("same execution with changed effort error=%v code=%q", err, codex.ErrorCode(err))
	}
	requireReasoningEffortLaunchCount(t, invocationsPath, 1)
	requireUnchangedReasoningEffortJournal(t, journalPath, journalPayload)

	processed, err := second.Orchestrator().ProcessNext(context.Background(), "worker:bug459-v7-restart")
	if err != nil || processed.Processed {
		t.Fatalf("restart replayed terminal work: result=%+v error=%v", processed, err)
	}
	reloaded, err := second.Orchestrator().GetGoal(
		context.Background(), testRuntimeAccess(t, second), terminal.Goal.Ref(),
	)
	if err != nil || reloaded.Goal.State() != goal.GoalStateSucceeded {
		t.Fatalf("reloaded terminal Goal: state=%s error=%v", reloaded.Goal.State(), err)
	}
	requireReasoningEffortLaunchCount(t, invocationsPath, 1)
	requireUnchangedReasoningEffortJournal(t, journalPath, journalPayload)
}

func buildReasoningEffortRuntime(t *testing.T, configPath string) *Runtime {
	t.Helper()
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath,
		Version:    "bug459-v7-evidence",
	})
	if err != nil {
		t.Fatalf("build production composition: %v", err)
	}
	return runtime
}

func shutdownReasoningEffortRuntime(t *testing.T, runtime *Runtime) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := runtime.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown production composition: %v", err)
	}
}

func writeReasoningEffortHelper(t *testing.T, helperPath, invocationsPath, argumentsPath string) {
	t.Helper()
	helper := "#!/bin/sh\n" +
		"set -eu\n" +
		"invocations=" + strconv.Quote(invocationsPath) + "\n" +
		"arguments=" + strconv.Quote(argumentsPath) + "\n" +
		"arguments_tmp=" + strconv.Quote(argumentsPath+".tmp") + "\n" +
		": > \"$arguments_tmp\"\n" +
		"for argument in \"$@\"; do printf '%s\\n' \"$argument\" >> \"$arguments_tmp\"; done\n" +
		"/bin/mv \"$arguments_tmp\" \"$arguments\"\n" +
		"output=\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--output-last-message\" ]; then shift; output=$1; fi\n" +
		"  shift\n" +
		"done\n" +
		"prompt=$(/bin/cat)\n" +
		"test -n \"$prompt\" && test -n \"$output\"\n" +
		"printf 'launch\\n' >> \"$invocations\"\n" +
		"printf '%s\\n' '{\"artifact\":\"artifact:bug459-v7-evidence\"}' > \"$output\"\n"
	if err := os.WriteFile(helperPath, []byte(helper), 0o700); err != nil {
		t.Fatalf("write Codex helper: %v", err)
	}
}

func requireReasoningEffortCausality(t *testing.T, record application.GoalRecord) ports.AgentLaunchRequest {
	t.Helper()
	if record.Goal.State() != goal.GoalStateSucceeded ||
		len(record.Goal.WorkItems()) != 1 ||
		len(record.Executions) != 1 ||
		len(record.EffectIntents) != 1 {
		t.Fatalf("unexpected terminal causal record: %+v", record)
	}
	item := record.Goal.WorkItems()[0]
	execution := record.Executions[0]
	intent := record.EffectIntents[0]
	if item.ReasoningEffort() != governance.ReasoningEffortXHigh ||
		intent.Kind != application.EffectKindAgentLaunch ||
		intent.Subject.ExecutionRef != execution.Ref ||
		intent.ReasoningEffort != governance.ReasoningEffortXHigh {
		t.Fatalf("WorkItem/EffectIntent lost xhigh: item=%q intent=%+v", item.ReasoningEffort(), intent)
	}
	phase, found := reasoningEffortPhase(record.Goal, item.Phase())
	if !found {
		t.Fatalf("phase %s not found", item.Phase())
	}
	request := ports.AgentLaunchRequest{
		ExecutionRef: execution.Ref, SessionRef: execution.ExecutionSessionRef,
		ExecutionWorkspaceRef: execution.ExecutionWorkspaceRef,
		GoalRef:               record.Goal.Ref(), WorkItemRef: item.Ref(),
		PlanGeneration: execution.PlanGeneration, AppSpecGeneration: execution.AppSpecGeneration,
		ExecutionAttempt: execution.AttemptNo, SpecHash: execution.SpecHash,
		ActorRef: record.Goal.Actor(), ProjectRef: record.Goal.Project(),
		Objective: item.Objective(), PhaseRef: phase.Ref().String(), PhaseKey: item.Phase().String(),
		PhaseTemplateRef: phase.TemplateRef().String(),
		PhaseInputRefs:   reasoningEffortRefs(phase.InputRefs()), PhaseCriterionRefs: reasoningEffortRefs(phase.CriterionRefs()),
		RoleKey:   item.Role().String(),
		SkillRefs: reasoningEffortRefs(item.SkillRefs()), ToolRefs: reasoningEffortRefs(item.ToolRefs()),
		CapabilityRefs: reasoningEffortRefs(item.CapabilityRefs()), WriteSet: reasoningEffortRefs(item.WriteSet()),
		OutputContract: string(item.OutputContract().Kind()), ArtifactMediaType: execution.ArtifactMediaType,
		IdempotencyKey: execution.IdempotencyKey, MaxOutputBytes: execution.MaxOutputBytes,
		BudgetDemand: item.BudgetDemand(), SecurityCriticality: item.SecurityCriticality(),
		ReasoningEffort: item.ReasoningEffort(),
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		t.Fatalf("reconstructed durable launch request: %v", err)
	}
	return request
}

func reasoningEffortPhase(aggregate goal.Goal, key goal.PhaseKey) (goal.PhaseInstance, bool) {
	for _, phase := range aggregate.Phases() {
		if phase.Key() == key {
			return phase, true
		}
	}
	return goal.PhaseInstance{}, false
}

func reasoningEffortRefs[T interface{ String() string }](values []T) []string {
	refs := make([]string, len(values))
	for index, value := range values {
		refs[index] = value.String()
	}
	return refs
}

func requirePhysicalReasoningEffort(
	t *testing.T,
	argumentsPath string,
	effort governance.ReasoningEffort,
) {
	t.Helper()
	payload, err := os.ReadFile(argumentsPath)
	if err != nil {
		t.Fatalf("read captured Codex arguments: %v", err)
	}
	arguments := strings.Split(strings.TrimSuffix(string(payload), "\n"), "\n")
	want := `model_reasoning_effort="` + string(effort) + `"`
	found := 0
	for index, argument := range arguments {
		if strings.HasPrefix(argument, "model_reasoning_effort=") && argument != want {
			t.Fatalf("physical Codex effort argument = %q want %q; argv=%q", argument, want, arguments)
		}
		if index > 0 && arguments[index-1] == "--config" && argument == want {
			found++
		}
	}
	if found != 1 {
		t.Fatalf("physical Codex effort argument occurrences=%d want=1; argv=%q", found, arguments)
	}
}

func requireReasoningEffortLaunchCount(t *testing.T, path string, want int) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read physical launch count: %v", err)
	}
	if got := strings.Count(string(payload), "launch\n"); got != want {
		t.Fatalf("physical Codex launches=%d want=%d payload=%q", got, want, payload)
	}
}

func readReasoningEffortJournal(
	t *testing.T,
	root string,
) (string, []byte, reasoningEffortLaunchJournal) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, "work", "executions", "*", "request.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("V7 launch journals=%d error=%v paths=%q", len(matches), err, matches)
	}
	payload, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read V7 launch journal: %v", err)
	}
	var journal reasoningEffortLaunchJournal
	if err := json.Unmarshal(payload, &journal); err != nil {
		t.Fatalf("decode V7 launch journal: %v", err)
	}
	return matches[0], payload, journal
}

func requireReasoningEffortReceiptBinding(
	t *testing.T,
	record application.GoalRecord,
	executionRef goal.ExecutionRef,
	providerReceiptRef string,
) {
	t.Helper()
	if len(record.EffectReceipts) != 1 {
		t.Fatalf("effect receipts=%d want=1", len(record.EffectReceipts))
	}
	execution := record.Executions[0]
	receipt := record.EffectReceipts[0]
	if execution.Ref != executionRef ||
		execution.LaunchReceiptRef != receipt.Ref ||
		receipt.IntentRef != record.EffectIntents[0].Ref ||
		receipt.ExternalRef != providerReceiptRef {
		t.Fatalf("launch receipt/hash binding lost: execution=%+v receipt=%+v want=%q",
			execution, receipt, providerReceiptRef)
	}
}

func requireUnchangedReasoningEffortJournal(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read V7 journal after replay/conflict: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("V7 journal mutated after replay/conflict:\ngot  %s\nwant %s", got, want)
	}
}
