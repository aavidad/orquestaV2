package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

type opesRegistryFinalPkgFixtureV0 struct {
	t              *testing.T
	root           string
	registryPath   string
	courseRoot     string
	appChangeState string
	runsDir        string
}

func newOPESRegistryFinalPkgFixtureV0(t *testing.T) opesRegistryFinalPkgFixtureV0 {
	root := t.TempDir()
	fixture := opesRegistryFinalPkgFixtureV0{
		t:              t,
		root:           root,
		registryPath:   filepath.Join(root, "REGISTRO_TRABAJO_TEMAS_OPES.json"),
		courseRoot:     filepath.Join(root, "informatica_a1_72_padres"),
		appChangeState: filepath.Join(root, "run-state", "app_change_v0.json"),
		runsDir:        filepath.Join(root, "orchestration-state", "runs"),
	}
	if err := os.MkdirAll(filepath.Dir(fixture.appChangeState), 0o755); err != nil {
		t.Fatalf("mkdir app state: %v", err)
	}
	if err := os.MkdirAll(fixture.runsDir, 0o755); err != nil {
		t.Fatalf("mkdir runs: %v", err)
	}
	return fixture
}

func (fixture opesRegistryFinalPkgFixtureV0) config(dryRun bool) opesRegistryFinalPkgConfigV0 {
	return opesRegistryFinalPkgConfigV0{
		RegistryPath:           fixture.registryPath,
		CourseID:               defaultOPESRegistryFinalPkgCourseIDV0,
		CourseRoot:             fixture.courseRoot,
		AppChangeState:         fixture.appChangeState,
		OrchestrationStateRoot: fixture.orchestrationStateRoot(),
		OrchestrationRuns:      fixture.runsDir,
		TemplateRunRef:         defaultOPESRegistryFinalPkgTemplateRunRefV0,
		TemplateTopicID:        defaultOPESRegistryFinalPkgTemplateTopicV0,
		OrquestaBaseURL:        "http://127.0.0.1:8787",
		QueueRef:               "global",
		BatchSize:              6,
		MaxInFlight:            6,
		ReconcileLimit:         defaultOPESRegistryFinalPkgReconcileLimitV0,
		DryRun:                 dryRun,
		HTTPTimeout:            time.Second,
	}
}

func (fixture opesRegistryFinalPkgFixtureV0) orchestrationStateRoot() string {
	return filepath.Dir(fixture.runsDir)
}

func (fixture opesRegistryFinalPkgFixtureV0) writeRegistry(topics map[string]any) {
	fixture.writeJSON(fixture.registryPath, map[string]any{
		"courses": map[string]any{
			defaultOPESRegistryFinalPkgCourseIDV0: map[string]any{
				"topics": topics,
			},
		},
	})
}

func (fixture opesRegistryFinalPkgFixtureV0) writeTemplateState() {
	fixture.writeJSON(fixture.appChangeState, map[string]any{
		"records": []map[string]any{{
			"request": map[string]any{
				"schema_version": "app_change_request.v0",
				"request_id":     "request-ref-opes-a1-t001-finalpkg-20260612",
				"correlation_id": "corr-opes-a1-t001-finalpkg-20260612",
				"run_ref":        "run-ref-opes-a1-t001-finalpkg-20260612",
				"app_ref":        "opes-a1-informatica",
				"change_ref":     "change-ref-opes-a1-t001-finalpkg-20260612",
				"user_intent":    "Finalizar paquete publicable del tema 001 de Informatica A1.",
				"allowed_write_set": []string{
					"opes-salidas/coordinacion_temarios/a1_maestros/tema_001",
				},
				"acceptance_criteria": []string{"criterio plantilla"},
				"external_work": map[string]any{
					"project_ref": "opes-a1-informatica",
					"job_ref":     "job-ref-opes-a1-t001-finalpkg-20260612",
					"work_kind":   "finalize_topic_package",
					"work_refs": []string{
						"course:informatica-a1-72-padres",
						"topic:001",
					},
				},
			},
		}},
	})
}

func (fixture opesRegistryFinalPkgFixtureV0) writeCompletePackage(topicID string) {
	base := filepath.Join(fixture.courseRoot, "tema_"+topicID, "paquete_final")
	for _, relative := range requiredOPESRegistryFinalPkgFilesV0 {
		path := filepath.Join(base, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fixture.t.Fatalf("mkdir package: %v", err)
		}
		content := []byte("ok")
		if relative == "tests.json" {
			content = []byte(`{"questions":[]}`)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			fixture.t.Fatalf("write package: %v", err)
		}
	}
}

func (fixture opesRegistryFinalPkgFixtureV0) writeActiveRun(runRef string, status string) {
	fixture.writeJSON(filepath.Join(fixture.runsDir, runRef+".json"), map[string]any{
		"run_ref": runRef,
		"run": map[string]any{
			"run_id": runRef,
			"status": status,
		},
	})
}

func (fixture opesRegistryFinalPkgFixtureV0) writeActiveDeliveredRun(runRef string, status string) {
	fixture.writeJSON(filepath.Join(fixture.runsDir, runRef+".json"), map[string]any{
		"run_ref": runRef,
		"run": map[string]any{
			"run_id":           runRef,
			"status":           status,
			"deliveries":       []string{"delivery-" + runRef},
			"accepted_reviews": []string{"accepted-review-" + runRef},
			"blockers":         []string{"technical-blocker"},
		},
	})
}

func (fixture opesRegistryFinalPkgFixtureV0) writeReconciliableRun(runRef string) {
	phases := orquestacoreworkflow.OrchestrationPhaseCatalogV0()
	for index := range phases {
		if phases[index].ID == orquestacoreworkflow.OrchestrationPhaseRevisionV0 {
			phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
			phases[index].OpenedAt = "2026-06-12T10:00:00Z"
		}
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:   orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:           runRef,
		ProjectRef:      "project-ref-opes-a1-informatica",
		AppSpecRef:      "app-spec-ref-opes-a1-finalpkg",
		Status:          orquestacoreworkflow.OrchestrationRunStatusBlockedV0,
		CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Phases:          phases,
		Tasks:           []string{"task-ref-" + runRef},
		Deliveries:      []string{"delivery-ref-" + runRef},
		AcceptedReviews: []string{"accepted-review-ref-" + runRef},
		Blockers:        []string{"blocker-ref-finalpkg"},
		LastEventID:     "event-ref-seed-" + runRef,
		LastSequence:    10,
	}
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: fixture.orchestrationStateRoot()})
	if err != nil {
		fixture.t.Fatalf("state store: %v", err)
	}
	if err := store.SaveRunV0(context.Background(), run); err != nil {
		fixture.t.Fatalf("save run: %v", err)
	}
}

func (fixture opesRegistryFinalPkgFixtureV0) writeJSON(path string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fixture.t.Fatalf("marshal: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fixture.t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fixture.t.Fatalf("write %s: %v", path, err)
	}
}
