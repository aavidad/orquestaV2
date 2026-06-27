package main

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestOPESRegistryFinalPkgConfigDisabledByDefaultV0(t *testing.T) {
	t.Setenv(envOPESRegistryFinalPkgEnabledV0, "")

	config, err := opesRegistryFinalPkgLoopConfigFromEnvV0(
		orquestaserver.ConfigV0{StateDir: t.TempDir()},
		"http://127.0.0.1:8787",
	)

	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if config.Loop.Enabled {
		t.Fatalf("enabled=%v", config.Loop.Enabled)
	}
}

func TestOPESRegistryFinalPkgConfigRequiresConfirmForEffectsV0(t *testing.T) {
	t.Setenv(envOPESRegistryFinalPkgEnabledV0, "1")
	t.Setenv(envOPESRegistryFinalPkgDryRunV0, "0")

	config, err := opesRegistryFinalPkgLoopConfigFromEnvV0(
		orquestaserver.ConfigV0{StateDir: t.TempDir()},
		"http://127.0.0.1:8787",
	)

	if err == nil || !strings.Contains(err.Error(), envOPESRegistryFinalPkgConfirmV0) || config.Loop.Enabled {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}

func TestOPESRegistryFinalPkgConfigDerivesStatePathsV0(t *testing.T) {
	stateDir := t.TempDir()
	registry := filepath.Join(t.TempDir(), "registry.json")
	courseRoot := filepath.Join(t.TempDir(), "course")
	t.Setenv(envOPESRegistryFinalPkgEnabledV0, "1")
	t.Setenv(envOPESRegistryFinalPkgRegistryV0, registry)
	t.Setenv(envOPESRegistryFinalPkgCourseRootV0, courseRoot)
	t.Setenv(envOPESRegistryFinalPkgBatchSizeV0, "4")
	t.Setenv(envOPESRegistryFinalPkgMaxInFlightV0, "5")
	t.Setenv(envOPESRegistryFinalPkgIntervalV0, "7")

	config, err := opesRegistryFinalPkgLoopConfigFromEnvV0(
		orquestaserver.ConfigV0{StateDir: stateDir},
		"http://127.0.0.1:8787",
	)

	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if !config.Loop.Enabled ||
		!config.Producer.DryRun ||
		config.Producer.AppChangeState != filepath.Join(stateDir, "run-state", "app_change_v0.json") ||
		config.Producer.OrchestrationRuns != filepath.Join(stateDir, "orchestration-state", "runs") ||
		config.Producer.BatchSize != 4 ||
		config.Producer.MaxInFlight != 5 ||
		config.Loop.Interval != 7*time.Second {
		t.Fatalf("config=%+v", config)
	}
}

func TestOPESRegistryFinalPkgDryRunSelectsCandidatesWithoutSubmitV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeRegistry(map[string]any{
		"001": map[string]any{},
		"002": map[string]any{},
		"003": map[string]any{"lock": map[string]any{"agent_id": "agent-live"}},
		"004": map[string]any{},
	})
	fixture.writeTemplateState()
	fixture.writeCompletePackage("004")
	submits := 0

	summary, err := runOPESRegistryFinalPkgOnceV0(
		context.Background(),
		fixture.config(true),
		func(context.Context, *http.Client, string, orquestaexternalworkrun.StartExternalWorkRunRequestV0) (string, error) {
			submits++
			return "", nil
		},
	)

	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if submits != 0 ||
		summary.Submitted != 0 ||
		len(summary.Results) != 1 ||
		summary.Results[0].TopicID != "002" ||
		summary.Results[0].Status != "dry_run" {
		t.Fatalf("submits=%d summary=%+v", submits, summary)
	}
}

func TestOPESRegistryFinalPkgPostsExternalWorkRunEnvelopeV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeRegistry(map[string]any{"001": map[string]any{}, "002": map[string]any{}})
	fixture.writeTemplateState()
	var received orquestaexternalworkrun.StartExternalWorkRunRequestV0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("request inesperada %s %s", r.Method, r.URL.Path)
		}
		var envelope struct {
			ExternalWorkRunRequest orquestaexternalworkrun.StartExternalWorkRunRequestV0 `json:"external_work_run_request"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode: %v", err)
		}
		received = envelope.ExternalWorkRunRequest
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"run_ref": received.RunRef, "estado": "ok"})
	}))
	defer server.Close()
	config := fixture.config(false)
	config.OrquestaBaseURL = server.URL

	summary, err := runOPESRegistryFinalPkgOnceV0(context.Background(), config, nil)

	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if summary.Submitted != 1 ||
		received.RunRef != "run-ref-opes-a1-t002-finalpkg-20260612" ||
		received.QueueRef != "global" ||
		!strings.Contains(received.AppChangeRequest.UserIntent, "tema 002") ||
		received.AppChangeRequest.ExternalWork == nil ||
		received.AppChangeRequest.ExternalWork.JobRef != "job-ref-opes-a1-t002-finalpkg-20260612" {
		t.Fatalf("summary=%+v received=%+v", summary, received)
	}
}

func TestOPESRegistryFinalPkgMaxInFlightSkipsLaunchV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeRegistry(map[string]any{"001": map[string]any{}, "002": map[string]any{}})
	fixture.writeTemplateState()
	fixture.writeActiveRun("run-ref-opes-a1-t002-finalpkg-20260612", "activa")
	config := fixture.config(false)
	config.MaxInFlight = 1
	submits := 0

	summary, err := runOPESRegistryFinalPkgOnceV0(
		context.Background(),
		config,
		func(context.Context, *http.Client, string, orquestaexternalworkrun.StartExternalWorkRunRequestV0) (string, error) {
			submits++
			return "", nil
		},
	)

	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if submits != 0 || len(summary.Active) != 1 || summary.Results[0].Status != "max_in_flight" {
		t.Fatalf("submits=%d summary=%+v", submits, summary)
	}
}

func TestOPESRegistryFinalPkgNoCuentaRunNoTerminalConPaqueteCompletoV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeRegistry(map[string]any{
		"001": map[string]any{},
		"002": map[string]any{},
		"003": map[string]any{},
	})
	fixture.writeTemplateState()
	fixture.writeCompletePackage("002")
	fixture.writeActiveDeliveredRun("run-ref-opes-a1-t002-finalpkg-20260612", "bloqueada")
	config := fixture.config(false)
	config.MaxInFlight = 1
	submitted := []string{}

	summary, err := runOPESRegistryFinalPkgOnceV0(
		context.Background(),
		config,
		func(_ context.Context, _ *http.Client, _ string, request orquestaexternalworkrun.StartExternalWorkRunRequestV0) (string, error) {
			submitted = append(submitted, request.RunRef)
			return request.RunRef, nil
		},
	)

	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if summary.Submitted != 1 ||
		len(submitted) != 1 ||
		submitted[0] != "run-ref-opes-a1-t003-finalpkg-20260612" ||
		len(summary.Active) != 0 ||
		summary.CompletedNonTerminal != 1 ||
		len(summary.CompletedNonTerminalRefs) != 1 ||
		summary.CompletedNonTerminalRefs[0].RunRef != "run-ref-opes-a1-t002-finalpkg-20260612" ||
		summary.CompletedNonTerminalRefs[0].TopicID != "002" ||
		summary.CompletedNonTerminalRefs[0].Status != "bloqueada" ||
		summary.CompletedNonTerminalRefs[0].Reason != "package_complete_run_non_terminal" {
		t.Fatalf("submitted=%v summary=%+v", submitted, summary)
	}
	counters := externalBridgeResultCountersV0(summary)
	if counters["completed_nonterminal"] != 1 {
		t.Fatalf("counters=%+v", counters)
	}
	evidenceRefs := externalBridgeResultEvidenceRefsV0(summary)
	if len(evidenceRefs) != 1 || evidenceRefs[0] != "run-ref-opes-a1-t002-finalpkg-20260612" {
		t.Fatalf("evidence_refs=%+v", evidenceRefs)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteRequiresDeterministicQuestionBankV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writePackageWithEmptyQuestionBank("002")

	validation := validateOPESRegistryFinalPkgPackageV0(
		filepath.Join(fixture.courseRoot, "tema_002", "paquete_final"),
	)

	if validation.Complete ||
		!containsStringForTestV0(validation.Issues, "tests_json_without_questions") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgReviewAcceptedDoesNotCompleteEmptyQuestionBankV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeRegistry(map[string]any{
		"001": map[string]any{},
		"002": map[string]any{},
		"003": map[string]any{},
	})
	fixture.writeTemplateState()
	fixture.writePackageWithEmptyQuestionBank("002")
	fixture.writeActiveDeliveredRun("run-ref-opes-a1-t002-finalpkg-20260612", "bloqueada")
	config := fixture.config(false)
	config.MaxInFlight = 1
	submitted := []string{}

	summary, err := runOPESRegistryFinalPkgOnceV0(
		context.Background(),
		config,
		func(_ context.Context, _ *http.Client, _ string, request orquestaexternalworkrun.StartExternalWorkRunRequestV0) (string, error) {
			submitted = append(submitted, request.RunRef)
			return request.RunRef, nil
		},
	)

	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(submitted) != 0 ||
		summary.CompletedNonTerminal != 0 ||
		len(summary.CompletedNonTerminalRefs) != 0 ||
		len(summary.Active) != 1 ||
		summary.Active[0] != "run-ref-opes-a1-t002-finalpkg-20260612" ||
		len(summary.Results) != 1 ||
		summary.Results[0].Status != "max_in_flight" {
		t.Fatalf("submitted=%v summary=%+v", submitted, summary)
	}
}

func TestOPESRegistryFinalPkgReconcilesCompletedRunV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	runRef := "run-ref-opes-a1-t002-finalpkg-20260612"
	fixture.writeRegistry(map[string]any{
		"001": map[string]any{},
		"002": map[string]any{},
	})
	fixture.writeTemplateState()
	fixture.writeCompletePackage("002")
	fixture.writeReconciliableRun(runRef)
	config := fixture.config(false)
	config.ReconcileCompleted = true
	config.ReconcileLimit = 10
	submits := 0

	summary, err := runOPESRegistryFinalPkgOnceV0(
		context.Background(),
		config,
		func(context.Context, *http.Client, string, orquestaexternalworkrun.StartExternalWorkRunRequestV0) (string, error) {
			submits++
			return "", nil
		},
	)

	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if submits != 0 ||
		summary.Reconciled != 1 ||
		summary.ReconcileSkipped != 0 ||
		len(summary.Errors) != 0 ||
		len(summary.Results) != 1 ||
		summary.Results[0].Status != "reconciled" ||
		summary.Results[0].RunRef != runRef {
		t.Fatalf("submits=%d summary=%+v", submits, summary)
	}
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: fixture.orchestrationStateRoot()})
	if err != nil {
		t.Fatalf("state store: %v", err)
	}
	run, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(run.Blockers) != 0 ||
		len(run.ClosedTasks) != 1 ||
		len(run.Validations) != 1 ||
		len(run.Closures) != 1 {
		t.Fatalf("run no reconciliado: %+v", run)
	}
	counters := externalBridgeResultCountersV0(summary)
	if counters["reconciled"] != 1 || counters["completed_nonterminal"] != 1 {
		t.Fatalf("counters=%+v", counters)
	}
}
