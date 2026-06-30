package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

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
		if relative == "manifest_cierre.json" {
			content = []byte(`{"schema_version":"opes_final_package_evidence_manifest.v0","package_ref":"package-ref-finalpkg-` + topicID + `","manifest_ref":"manifest-cierre-ref-finalpkg-` + topicID + `","checksum_refs":["checksum-ref-finalpkg-` + topicID + `"],"validation_report_ref":"validation-report-ref-finalpkg-` + topicID + `","review_matrix_ref":"review-matrix-ref-finalpkg-` + topicID + `","required_evidence_refs":{"html":["opes-final-evidence:html:` + topicID + `"],"rag":["opes-final-evidence:rag:` + topicID + `"],"audio":["opes-final-evidence:audio:` + topicID + `"],"tests":["opes-final-evidence:tests:` + topicID + `"],"visual":["opes-final-evidence:visual:` + topicID + `"],"qa":["opes-final-evidence:qa:` + topicID + `"]}}`)
		}
		if relative == "tests.json" {
			content = []byte(`{"questions":[{"id":"q1","prompt":"pregunta verificable","options":["a","b"],"answer":"a"}]}`)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			fixture.t.Fatalf("write package: %v", err)
		}
	}
}

func (fixture opesRegistryFinalPkgFixtureV0) writePackageWithEmptyQuestionBank(topicID string) {
	fixture.writeCompletePackage(topicID)
	path := filepath.Join(fixture.courseRoot, "tema_"+topicID, "paquete_final", "tests.json")
	if err := os.WriteFile(path, []byte(`{"questions":[]}`), 0o644); err != nil {
		fixture.t.Fatalf("write empty tests: %v", err)
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
	fixture.writeReconciliableRunWithAcceptedReviews(runRef, []string{"accepted-review-ref-" + runRef})
}

func (fixture opesRegistryFinalPkgFixtureV0) writeReconciliableRunWithoutAcceptedReviews(runRef string) {
	fixture.writeReconciliableRunWithAcceptedReviews(runRef, nil)
}

func (fixture opesRegistryFinalPkgFixtureV0) writeReconciliableRunWithAcceptedReviews(runRef string, acceptedReviews []string) {
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
		AcceptedReviews: acceptedReviews,
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
