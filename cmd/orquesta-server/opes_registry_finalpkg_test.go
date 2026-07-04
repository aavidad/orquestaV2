package main

import (
	"context"
	"encoding/json"
	"net/http"
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

func TestOPESRegistryFinalPkgConfigLiveRequiresExplicitCourseAndTemplateV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := filepath.Join(projectDir, "state")
	registry := filepath.Join(projectDir, "registry.json")
	courseRoot := filepath.Join(projectDir, "course")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envOPESRegistryFinalPkgCourseIDV0, "")
	t.Setenv(envOPESRegistryFinalPkgTemplateRunV0, "")
	t.Setenv(envOPESRegistryFinalPkgTemplateTopicV0, "")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server":{"state_dir":"` + filepath.ToSlash(stateDir) + `"},
		"opes_registry_finalpkg":{
			"enabled":true,
			"confirm":true,
			"dry_run":false,
			"registry_path":"` + filepath.ToSlash(registry) + `",
			"course_root":"` + filepath.ToSlash(courseRoot) + `"
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	serverConfig, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	config, err := opesRegistryFinalPkgLoopConfigFromEnvV0(serverConfig, "http://127.0.0.1:8787")

	if err == nil ||
		!strings.Contains(err.Error(), "opes_registry_finalpkg_config_incomplete") ||
		!strings.Contains(err.Error(), envOPESRegistryFinalPkgCourseIDV0) ||
		!strings.Contains(err.Error(), envOPESRegistryFinalPkgTemplateRunV0) ||
		!strings.Contains(err.Error(), envOPESRegistryFinalPkgTemplateTopicV0) ||
		config.Loop.Enabled {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}

func TestOPESRegistryFinalPkgConfigLiveLeeFicheroCanonicoExplicitoV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := filepath.Join(projectDir, "state")
	registry := filepath.Join(projectDir, "registry.json")
	courseRoot := filepath.Join(projectDir, "course")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server":{"state_dir":"` + filepath.ToSlash(stateDir) + `"},
		"opes_registry_finalpkg":{
			"enabled":true,
			"confirm":true,
			"dry_run":false,
			"registry_path":"` + filepath.ToSlash(registry) + `",
			"course_id":"course-ref-live-file",
			"course_root":"` + filepath.ToSlash(courseRoot) + `",
			"template_run_ref":"run-ref-live-template-file",
			"template_topic_id":"007"
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	serverConfig, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	config, err := opesRegistryFinalPkgLoopConfigFromEnvV0(serverConfig, "http://127.0.0.1:8787")
	if err != nil {
		t.Fatalf("opesRegistryFinalPkgLoopConfigFromEnvV0: %v", err)
	}
	if !config.Loop.Enabled ||
		config.Producer.DryRun ||
		config.Producer.CourseID != "course-ref-live-file" ||
		config.Producer.TemplateRunRef != "run-ref-live-template-file" ||
		config.Producer.TemplateTopicID != "007" {
		t.Fatalf("config=%+v", config)
	}
	settings := serverConfig.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envOPESRegistryFinalPkgConfirmV0:       "true",
		envOPESRegistryFinalPkgDryRunV0:        "false",
		envOPESRegistryFinalPkgCourseIDV0:      "course-ref-live-file",
		envOPESRegistryFinalPkgTemplateRunV0:   "run-ref-live-template-file",
		envOPESRegistryFinalPkgTemplateTopicV0: "007",
	} {
		setting := effectiveSettingForTestV0(settings, key)
		if setting.Value != want || setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("%s setting=%+v want value=%q source=config_file", key, setting, want)
		}
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

func TestOPESRegistryFinalPkgConfigLeeFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := filepath.Join(projectDir, "state")
	registry := filepath.Join(projectDir, "registry.json")
	courseRoot := filepath.Join(projectDir, "course")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server":{"state_dir":"` + filepath.ToSlash(stateDir) + `"},
		"opes_registry_finalpkg":{
			"enabled":true,
			"dry_run":true,
			"registry_path":"` + filepath.ToSlash(registry) + `",
			"course_id":"course-ref-file",
			"course_root":"` + filepath.ToSlash(courseRoot) + `",
			"template_run_ref":"run-ref-template-file",
			"template_topic_id":"099",
			"batch_size":4,
			"max_in_flight":5,
			"interval_seconds":7,
			"max_ticks":8,
			"queue_ref":"queue-ref-file",
			"reconcile_enabled":true,
			"reconcile_limit":9
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	serverConfig, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	config, err := opesRegistryFinalPkgLoopConfigFromEnvV0(serverConfig, "http://127.0.0.1:8787")
	if err != nil {
		t.Fatalf("opesRegistryFinalPkgLoopConfigFromEnvV0: %v", err)
	}
	if !config.Loop.Enabled ||
		config.Producer.RegistryPath != registry ||
		config.Producer.CourseID != "course-ref-file" ||
		config.Producer.CourseRoot != courseRoot ||
		config.Producer.TemplateRunRef != "run-ref-template-file" ||
		config.Producer.TemplateTopicID != "099" ||
		config.Producer.BatchSize != 4 ||
		config.Producer.MaxInFlight != 5 ||
		config.Producer.QueueRef != "queue-ref-file" ||
		!config.Producer.ReconcileCompleted ||
		config.Producer.ReconcileLimit != 9 ||
		config.Loop.Interval != 7*time.Second ||
		config.Loop.MaxTicks != 8 {
		t.Fatalf("config=%+v", config)
	}
	settings := serverConfig.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envOPESRegistryFinalPkgEnabledV0:        "true",
		envOPESRegistryFinalPkgDryRunV0:         "true",
		envOPESRegistryFinalPkgCourseIDV0:       "course-ref-file",
		envOPESRegistryFinalPkgTemplateRunV0:    "run-ref-template-file",
		envOPESRegistryFinalPkgTemplateTopicV0:  "099",
		envOPESRegistryFinalPkgBatchSizeV0:      "4",
		envOPESRegistryFinalPkgMaxInFlightV0:    "5",
		envOPESRegistryFinalPkgIntervalV0:       "7",
		envOPESRegistryFinalPkgMaxTicksV0:       "8",
		envOPESRegistryFinalPkgQueueRefV0:       "queue-ref-file",
		envOPESRegistryFinalPkgReconcileV0:      "true",
		envOPESRegistryFinalPkgReconcileLimitV0: "9",
	} {
		setting := effectiveSettingForTestV0(settings, key)
		if setting.Value != want || setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("%s setting=%+v want value=%q source=config_file", key, setting, want)
		}
	}
	for key, want := range map[string]string{
		envOPESRegistryFinalPkgRegistryV0:   "opes-registry-finalpkg-registry-path-configured",
		envOPESRegistryFinalPkgCourseRootV0: "opes-registry-finalpkg-course-root-configured",
	} {
		setting := effectiveSettingForTestV0(settings, key)
		if setting.Value != want || setting.Source != configSettingSourceConfigFileV0 || !setting.Sensitive {
			t.Fatalf("%s setting=%+v want sensitive config_file", key, setting)
		}
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

func TestOPESRegistryFinalPkgPackageCompleteRejectsQuestionCountersOnlyV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	path := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final", "tests.json")
	if err := os.WriteFile(path, []byte(`{"total_questions":50}`), 0o644); err != nil {
		t.Fatalf("write counter tests: %v", err)
	}

	validation := validateOPESRegistryFinalPkgPackageV0(
		filepath.Join(fixture.courseRoot, "tema_002", "paquete_final"),
	)

	if validation.Complete ||
		!containsStringForTestV0(validation.Issues, "tests_json_without_questions") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteRejectsQuestionArrayWithoutShapeV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	path := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final", "tests.json")
	if err := os.WriteFile(path, []byte(`{"questions":["pregunta sin opciones ni respuesta"]}`), 0o644); err != nil {
		t.Fatalf("write malformed tests: %v", err)
	}

	validation := validateOPESRegistryFinalPkgPackageV0(
		filepath.Join(fixture.courseRoot, "tema_002", "paquete_final"),
	)

	if validation.Complete ||
		!containsStringForTestV0(validation.Issues, "tests_json_without_questions") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteRequiresEvidenceManifestV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	path := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final", "manifest_cierre.json")
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove manifest: %v", err)
	}

	validation := validateOPESRegistryFinalPkgPackageV0(
		filepath.Join(fixture.courseRoot, "tema_002", "paquete_final"),
	)

	if validation.Complete ||
		!containsStringForTestV0(validation.Issues, "missing_or_empty:manifest_cierre.json") ||
		!containsStringForTestV0(validation.Issues, "manifest_cierre_missing") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteRequiresManifestEvidenceKeysV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	path := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final", "manifest_cierre.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"opes_final_package_evidence_manifest.v0","package_ref":"package-ref-finalpkg-002","manifest_ref":"manifest-cierre-ref-finalpkg-002","checksum_refs":["checksum-ref-finalpkg-002"],"validation_report_ref":"validation-report-ref-finalpkg-002","review_matrix_ref":"review-matrix-ref-finalpkg-002","required_evidence_refs":{"html":["opes-final-evidence:html:002"],"rag":["opes-final-evidence:rag:002"],"audio":["opes-final-evidence:audio:002"],"tests":["opes-final-evidence:tests:002"],"visual":["opes-final-evidence:visual:002"]}}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	validation := validateOPESRegistryFinalPkgPackageV0(
		filepath.Join(fixture.courseRoot, "tema_002", "paquete_final"),
	)

	if validation.Complete ||
		!containsStringForTestV0(validation.Issues, "manifest_cierre_required_evidence_missing:qa") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteRequiresStructuredClosureRefsV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	path := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final", "manifest_cierre.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"opes_final_package_evidence_manifest.v0","required_evidence_refs":{"html":["opes-final-evidence:html:002"],"rag":["opes-final-evidence:rag:002"],"audio":["opes-final-evidence:audio:002"],"tests":["opes-final-evidence:tests:002"],"visual":["opes-final-evidence:visual:002"],"qa":["opes-final-evidence:qa:002"]}}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	validation := validateOPESRegistryFinalPkgPackageV0(
		filepath.Join(fixture.courseRoot, "tema_002", "paquete_final"),
	)

	if validation.Complete ||
		!containsStringForTestV0(validation.Issues, "manifest_cierre_package_ref_missing") ||
		!containsStringForTestV0(validation.Issues, "manifest_cierre_manifest_ref_missing") ||
		!containsStringForTestV0(validation.Issues, "manifest_cierre_checksum_refs_missing") ||
		!containsStringForTestV0(validation.Issues, "manifest_cierre_validation_report_ref_missing") ||
		!containsStringForTestV0(validation.Issues, "manifest_cierre_review_matrix_ref_missing") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteRejectsEvidenceRefsFallbackV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	path := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final", "manifest_cierre.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"opes_final_package_evidence_manifest.v0","package_ref":"package-ref-finalpkg-002","manifest_ref":"manifest-cierre-ref-finalpkg-002","checksum_refs":["checksum-ref-finalpkg-002"],"validation_report_ref":"validation-report-ref-finalpkg-002","review_matrix_ref":"review-matrix-ref-finalpkg-002","evidence_refs":["opes-final-evidence:html:002","opes-final-evidence:rag:002","opes-final-evidence:audio:002","opes-final-evidence:tests:002","opes-final-evidence:visual:002","opes-final-evidence:qa:002"]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	validation := validateOPESRegistryFinalPkgPackageV0(
		filepath.Join(fixture.courseRoot, "tema_002", "paquete_final"),
	)

	if validation.Complete ||
		!containsStringForTestV0(validation.Issues, "manifest_cierre_required_evidence_missing:html") ||
		!containsStringForTestV0(validation.Issues, "manifest_cierre_required_evidence_missing:qa") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteRejectsLooseRAGCorpusV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	base := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final")
	if err := os.RemoveAll(filepath.Join(base, "rag", "corpus")); err != nil {
		t.Fatalf("remove corpus: %v", err)
	}
	for _, relative := range []string{"rag/chunks.jsonl", "rag/summary.json"} {
		path := filepath.Join(base, relative)
		if err := os.WriteFile(path, []byte("legacy\n"), 0o644); err != nil {
			t.Fatalf("write loose rag: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(base, "rag", "manifest.json"), []byte(`{"chunks_ref":"rag/chunks.jsonl","summary_ref":"rag/summary.json"}`), 0o644); err != nil {
		t.Fatalf("write rag manifest: %v", err)
	}

	validation := validateOPESRegistryFinalPkgPackageV0(base)

	for _, want := range []string{
		"rag_corpus_canonical_missing:rag/corpus/chunks.jsonl",
		"rag_corpus_canonical_missing:rag/corpus/summary.json",
		"rag_corpus_loose_legacy_present:rag/chunks.jsonl",
		"rag_manifest_missing_ref:rag/corpus/chunks.jsonl",
	} {
		if validation.Complete || !containsStringForTestV0(validation.Issues, want) {
			t.Fatalf("want %s validation=%+v", want, validation)
		}
	}
}

func TestOPESRegistryFinalPkgPackageCompleteRequiresCanonicalRAGManifestRefsV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	base := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final")
	if err := os.WriteFile(filepath.Join(base, "rag", "manifest.json"), []byte(`{"chunks_ref":"artifact-ref-rag-chunks","summary_ref":"artifact-ref-rag-summary"}`), 0o644); err != nil {
		t.Fatalf("write rag manifest: %v", err)
	}

	validation := validateOPESRegistryFinalPkgPackageV0(base)

	if validation.Complete ||
		!containsStringForTestV0(validation.Issues, "rag_manifest_missing_ref:rag/corpus/chunks.jsonl") ||
		!containsStringForTestV0(validation.Issues, "rag_manifest_missing_ref:rag/corpus/summary.json") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteCountsOnlyTopicHTMLAudioManifestsV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	base := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final")
	if err := os.RemoveAll(filepath.Join(base, "audio", "manifests")); err != nil {
		t.Fatalf("remove audio manifests: %v", err)
	}
	writeOPESRegistryFinalPkgTestFileV0(t, base, "html_final/index.html", "<html>indice</html>")
	writeOPESRegistryFinalPkgTestFileV0(t, base, "html_final/portada.html", "<html>portada</html>")
	writeOPESRegistryFinalPkgTestFileV0(t, base, "html_final/tema_002.html", "<html>tema</html>")
	writeOPESRegistryFinalPkgTestFileV0(t, base, "audio/manifests/index.json", `{"material_path":"html_final/index.html"}`)

	validation := validateOPESRegistryFinalPkgPackageV0(base)

	if validation.Complete ||
		!containsStringForTestV0(validation.Issues, "audio_manifest_missing:html_final/tema_002.html") ||
		containsStringForTestV0(validation.Issues, "audio_manifest_missing:html_final/index.html") ||
		containsStringForTestV0(validation.Issues, "audio_manifest_missing:html_final/portada.html") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteRejectsLegacyAudioManifestIndexHTMLV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	base := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final")
	writeOPESRegistryFinalPkgTestFileV0(t, base, "audio/manifest.json", `{"expected_count":3,"items":[{"source_html":"html_final/index.html"},{"source_html":"html_final/tema_002.html"},{"source_html":"html_ampliado/tema_002.html"}]}`)

	validation := validateOPESRegistryFinalPkgPackageV0(base)

	if validation.Complete ||
		!containsStringForTestV0(validation.Issues, "audio_legacy_manifest_references_index_html") ||
		!containsStringForTestV0(validation.Issues, "audio_legacy_manifest_count_includes_non_topic_html") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteAcceptsNeutralLegacyAudioManifestV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	base := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final")
	writeOPESRegistryFinalPkgTestFileV0(t, base, "audio/manifest.json", `{"expected_count":2,"items":[{"source_html":"html_final/tema_002.html"},{"source_html":"html_ampliado/tema_002.html"}]}`)

	validation := validateOPESRegistryFinalPkgPackageV0(base)

	if !validation.Complete ||
		containsStringForTestV0(validation.Issues, "audio_legacy_manifest_references_index_html") ||
		containsStringForTestV0(validation.Issues, "audio_legacy_manifest_count_includes_non_topic_html") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteRejectsLegacyOnlyPackageV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	base := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final")
	writeOPESRegistryFinalPkgLegacyPackageV0(t, base, "002")

	validation := validateOPESRegistryFinalPkgPackageV0(base)

	if validation.Complete ||
		!containsStringForTestV0(validation.Issues, "missing_or_empty:index.html") ||
		!containsStringForTestV0(validation.Issues, "html_topic_missing:html_final/tema_*.html") ||
		!containsStringForTestV0(validation.Issues, "html_topic_missing:html_ampliado/tema_*.html") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteRequiresFinalAndExpandedTopicHTMLV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	base := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final")
	if err := os.RemoveAll(filepath.Join(base, "html_ampliado")); err != nil {
		t.Fatalf("remove html ampliado: %v", err)
	}

	validation := validateOPESRegistryFinalPkgPackageV0(base)

	if validation.Complete ||
		!containsStringForTestV0(validation.Issues, "html_topic_missing:html_ampliado/tema_*.html") ||
		containsStringForTestV0(validation.Issues, "html_topic_missing:html_final/tema_*.html") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestOPESRegistryFinalPkgPackageCompleteAcceptsCanonicalRAGAndTopicAudioV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	fixture.writeCompletePackage("002")
	base := filepath.Join(fixture.courseRoot, "tema_002", "paquete_final")
	writeOPESRegistryFinalPkgTestFileV0(t, base, "audio/manifests/html_final/tema_002.html.json", `{"material_path":"html_final/tema_002.html"}`)
	writeOPESRegistryFinalPkgTestFileV0(t, base, "audio/manifests/html_ampliado/tema_002.html.json", `{"material_path":"html_ampliado/tema_002.html"}`)

	validation := validateOPESRegistryFinalPkgPackageV0(base)

	if !validation.Complete {
		t.Fatalf("validation=%+v", validation)
	}
}

func writeOPESRegistryFinalPkgLegacyPackageV0(t *testing.T, root string, topicID string) {
	t.Helper()
	for _, relative := range []string{
		"manifest_cierre.json",
		"tema_final.md",
		"tests.json",
		"visuales_plan.md",
		"html/index.html",
		"rag/manifest.json",
		"rag/corpus/chunks.jsonl",
		"rag/corpus/summary.json",
		"audio/guion_audio.md",
		"tutor/tutor_prompt.md",
		"qa_final.md",
	} {
		content := "ok"
		switch relative {
		case "manifest_cierre.json":
			content = `{"schema_version":"opes_final_package_evidence_manifest.v0","package_ref":"package-ref-finalpkg-` + topicID + `","manifest_ref":"manifest-cierre-ref-finalpkg-` + topicID + `","checksum_refs":["checksum-ref-finalpkg-` + topicID + `"],"validation_report_ref":"validation-report-ref-finalpkg-` + topicID + `","review_matrix_ref":"review-matrix-ref-finalpkg-` + topicID + `","required_evidence_refs":{"html":["opes-final-evidence:html:` + topicID + `"],"rag":["opes-final-evidence:rag:` + topicID + `"],"audio":["opes-final-evidence:audio:` + topicID + `"],"tests":["opes-final-evidence:tests:` + topicID + `"],"visual":["opes-final-evidence:visual:` + topicID + `"],"qa":["opes-final-evidence:qa:` + topicID + `"]}}`
		case "rag/manifest.json":
			content = `{"schema_version":"opes_rag_manifest.v1","chunks_ref":"rag/corpus/chunks.jsonl","summary_ref":"rag/corpus/summary.json"}`
		case "tests.json":
			content = `{"questions":[{"id":"q1","prompt":"pregunta verificable","options":["a","b"],"answer":"a"}]}`
		}
		writeOPESRegistryFinalPkgTestFileV0(t, root, relative, content)
	}
}

func writeOPESRegistryFinalPkgTestFileV0(t *testing.T, root string, relative string, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", relative, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relative, err)
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

func TestOPESRegistryFinalPkgReconcilesCompletedRunWithoutAcceptedReviewsV0(t *testing.T) {
	fixture := newOPESRegistryFinalPkgFixtureV0(t)
	runRef := "run-ref-opes-a1-t002-finalpkg-20260612"
	topicID := "002"
	fixture.writeRegistry(map[string]any{
		"001": map[string]any{},
		"002": map[string]any{},
	})
	fixture.writeTemplateState()
	fixture.writeCompletePackage(topicID)
	fixture.writeReconciliableRunWithoutAcceptedReviews(runRef)
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
	reviewRequestRef := opesRegistryFinalPkgReviewRequestRefV0(runRef, topicID)
	reviewResultRef := opesRegistryFinalPkgReviewResultRefV0(runRef, topicID)
	acceptedReviewRef := opesRegistryFinalPkgAcceptedReviewRefV0(runRef, topicID)
	deliveryRef := "delivery-ref-" + runRef
	wantReviewResult := reviewResultRef +
		"#review_result:accepted#review_request:" + reviewRequestRef +
		"#delivery:" + deliveryRef
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(run.Blockers) != 0 ||
		len(run.ClosedTasks) != 1 ||
		len(run.Validations) != 1 ||
		len(run.Closures) != 1 ||
		!containsStringForTestV0(run.Reviews, reviewRequestRef) ||
		!containsStringForTestV0(run.ReviewResults, wantReviewResult) ||
		!containsStringForTestV0(run.AcceptedReviews, acceptedReviewRef) {
		t.Fatalf("run no reconciliado con review determinista: %+v", run)
	}
}
