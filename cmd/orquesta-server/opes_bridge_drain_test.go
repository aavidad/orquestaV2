package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

func TestRunOPESDrainOnceV0LedgerEvitaReenviarJobPendienteV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-expansion-001",
			"type":           "expand_topic_from_summary",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","summary_payload_json":{"markdown":"Resumen"}}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	posts := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeOrquestaRunSuperviseOKForDrainTestV0(t, w, r) {
			return
		}
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		posts++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_ref": "run-ref-expansion-001",
			"estado":  "accepted",
		})
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	config := opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           1,
		HTTPTimeout:     time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	}

	first, err := runOPESDrainOnceV0(context.Background(), config)
	if err != nil {
		t.Fatalf("first drain: %v", err)
	}
	second, err := runOPESDrainOnceV0(context.Background(), config)
	if err != nil {
		t.Fatalf("second drain: %v", err)
	}

	if posts != 1 {
		t.Fatalf("posts=%d first=%+v second=%+v", posts, first, second)
	}
	if first.Submitted != 1 || second.AlreadySubmitted != 1 ||
		second.Results[0].Status != "already_submitted" ||
		second.Results[0].RunRef != "run-ref-expansion-001" {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
}

func TestRunOPESDrainOnceV0GoalFirstPropagaMetadataYLedgerV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-goal-first-001",
			"type":           "draft_content_block",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque goal"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	posts := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v0/runs/supervise" {
			t.Fatalf("goal-first no debe llamar a runs/supervise")
		}
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		posts++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"estado":                  "ok",
			"route_policy":            "goal_first",
			"director_execution_mode": "goal_first",
			"run_ref":                 "run-ref-goal-first-001",
			"goal_ref":                "goal-ref-goal-first-001",
			"external_goal_ref":       "thread-ref-goal-first-001",
			"next_actions":            []string{"observe_goal"},
		})
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	config := opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           1,
		JobType:         "draft_content_block",
		HTTPTimeout:     time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	}

	first, err := runOPESDrainOnceV0(context.Background(), config)
	if err != nil {
		t.Fatalf("first drain: %v", err)
	}
	second, err := runOPESDrainOnceV0(context.Background(), config)
	if err != nil {
		t.Fatalf("second drain: %v", err)
	}

	if posts != 1 ||
		first.Submitted != 1 ||
		first.Results[0].RoutePolicy != "goal_first" ||
		first.Results[0].GoalRef != "goal-ref-goal-first-001" ||
		first.Results[0].SupervisionStatus != "goal_first_observe_pending" ||
		second.AlreadySubmitted != 1 ||
		second.Results[0].Status != "already_submitted" ||
		second.Results[0].RoutePolicy != "goal_first" ||
		second.Results[0].GoalRef != "goal-ref-goal-first-001" ||
		second.Results[0].ExternalGoalRef != "thread-ref-goal-first-001" {
		t.Fatalf("posts=%d first=%+v second=%+v", posts, first, second)
	}
}

func TestRunOPESDrainOnceV0GoalFirstSupervisionObservaGoalSinSupervisorLegacyV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-goal-observe-001",
			"type":           "draft_content_block",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque goal observe"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	observes := 0
	supervisions := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/apps/director/goal/observe":
			observes++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode observe: %v", err)
			}
			if body["run_ref"] != "run-ref-goal-observe-001" {
				t.Fatalf("observe body=%+v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado":                  "ok",
				"run_ref":                 "run-ref-goal-observe-001",
				"run_status":              "closed",
				"director_execution_mode": "goal_first",
				"goal_ref":                "goal-ref-goal-observe-001",
				"external_goal_ref":       "thread-ref-goal-observe-001",
				"goal_status":             "complete",
				"closure_status":          "accepted",
				"closure_accepted":        true,
				"evidence_refs":           []string{"evidence-ref-goal-observe-accepted"},
			})
		case "/api/v0/runs/supervise":
			supervisions++
			t.Fatalf("goal-first no debe llamar a runs/supervise")
		default:
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if err := opesBridgeRecordSubmittedV0(
		context.Background(),
		ledger,
		orquestaopesconnector.ExternalJobV0{ID: "job-ref-goal-observe-001"},
		"run-ref-goal-observe-001",
		"change-ref-goal-observe-001",
		externalBridgeInputRunMetadataV0{
			RoutePolicy:           "goal_first",
			DirectorExecutionMode: "goal_first",
			GoalRef:               "goal-ref-goal-observe-001",
			ExternalGoalRef:       "thread-ref-goal-observe-001",
			NextActions:           []string{"observe_goal"},
		},
	); err != nil {
		t.Fatalf("record ledger: %v", err)
	}

	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:        opesServer.URL,
		OrquestaBaseURL:    orquestaServer.URL,
		Limit:              1,
		JobType:            "draft_content_block",
		HTTPTimeout:        time.Second,
		SuperviseSubmitted: true,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil ||
		observes != 1 ||
		supervisions != 0 ||
		summary.AlreadySubmitted != 1 ||
		summary.Results[0].SupervisionStatus != "closed" ||
		summary.Results[0].SupervisionStopReason != "accepted" ||
		summary.Results[0].SupervisionEvidenceRef != "evidence-ref-goal-observe-accepted" {
		t.Fatalf("observes=%d supervisions=%d summary=%+v err=%v", observes, supervisions, summary, err)
	}
}

func TestRunOPESDrainOnceV0AlreadySubmittedGoalFirstLedgerAntiguoPivotaPorStatsAntesDeSupervisorV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-goal-legacy-ledger-001",
			"type":           "draft_content_block",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque goal ledger antiguo"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	observes := 0
	statsCalls := 0
	supervisions := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/director/stats":
			statsCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado": "ok",
				"goal": map[string]any{
					"director_execution_mode": "goal_first",
					"goal_ref":                "goal-ref-goal-legacy-ledger-001",
					"external_goal_ref":       "thread-ref-goal-legacy-ledger-001",
					"status":                  "running",
					"evidence_refs":           []string{"evidence-ref-run-stats-goal-first"},
				},
				"stats": map[string]any{
					"run_ref": "run-ref-goal-legacy-ledger-001",
					"status":  "running",
				},
			})
		case "/api/v0/apps/director/goal/observe":
			observes++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode observe: %v", err)
			}
			if body["run_ref"] != "run-ref-goal-legacy-ledger-001" {
				t.Fatalf("observe body=%+v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado":                  "ok",
				"run_ref":                 "run-ref-goal-legacy-ledger-001",
				"run_status":              "running",
				"director_execution_mode": "goal_first",
				"goal_ref":                "goal-ref-goal-legacy-ledger-001",
				"external_goal_ref":       "thread-ref-goal-legacy-ledger-001",
				"goal_status":             "running",
				"evidence_refs":           []string{"evidence-ref-goal-legacy-ledger-observe"},
			})
		case "/api/v0/runs/supervise":
			supervisions++
			t.Fatalf("goal-first hidratado por stats no debe llamar a runs/supervise")
		default:
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if err := opesBridgeRecordSubmittedV0(
		context.Background(),
		ledger,
		orquestaopesconnector.ExternalJobV0{ID: "job-ref-goal-legacy-ledger-001"},
		"run-ref-goal-legacy-ledger-001",
		"change-ref-goal-legacy-ledger-001",
	); err != nil {
		t.Fatalf("record ledger: %v", err)
	}

	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:        opesServer.URL,
		OrquestaBaseURL:    orquestaServer.URL,
		Limit:              1,
		JobType:            "draft_content_block",
		HTTPTimeout:        time.Second,
		SuperviseSubmitted: true,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil ||
		statsCalls != 1 ||
		observes != 1 ||
		supervisions != 0 ||
		summary.AlreadySubmitted != 1 ||
		summary.Results[0].Status != "already_submitted" ||
		summary.Results[0].RoutePolicy != "goal_first" ||
		summary.Results[0].GoalRef != "goal-ref-goal-legacy-ledger-001" ||
		summary.Results[0].SupervisionStatus != "running" ||
		summary.Results[0].SupervisionStopReason != "running" {
		t.Fatalf("stats=%d observes=%d supervisions=%d summary=%+v err=%v", statsCalls, observes, supervisions, summary, err)
	}
	entry, found, err := ledger.LookupExternalBridgeInputV0(
		context.Background(),
		opesBridgeInputLedgerKeyV0("job-ref-goal-legacy-ledger-001"),
	)
	if err != nil ||
		!found ||
		entry.RoutePolicy != "goal_first" ||
		entry.GoalRef != "goal-ref-goal-legacy-ledger-001" ||
		entry.ExternalGoalRef != "thread-ref-goal-legacy-ledger-001" ||
		!containsStringForTestV0(entry.NextActions, "observe_goal") {
		t.Fatalf("entry=%+v found=%v err=%v", entry, found, err)
	}
}

func TestRunOPESDrainOnceV0AlreadySubmittedGoalFirstLedgerAntiguoPivotaPorStatsResidenteV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-goal-resident-ledger-001",
			"type":           "draft_content_block",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque goal resident ledger antiguo"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	observes := 0
	statsCalls := 0
	supervisions := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/director/stats":
			statsCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado": "ok",
				"goal": map[string]any{
					"director_execution_mode": "goal_first",
					"goal_ref":                "goal-ref-goal-resident-ledger-001",
					"external_goal_ref":       "thread-ref-goal-resident-ledger-001",
					"status":                  "running",
					"evidence_refs":           []string{"evidence-ref-goal-resident-stats"},
				},
				"stats": map[string]any{
					"run_ref": "run-ref-goal-resident-ledger-001",
					"status":  "running",
				},
			})
		case "/api/v0/apps/director/goal/observe":
			observes++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode observe: %v", err)
			}
			if body["run_ref"] != "run-ref-goal-resident-ledger-001" {
				t.Fatalf("observe body=%+v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado":                  "ok",
				"run_ref":                 "run-ref-goal-resident-ledger-001",
				"run_status":              "closed",
				"director_execution_mode": "goal_first",
				"goal_ref":                "goal-ref-goal-resident-ledger-001",
				"external_goal_ref":       "thread-ref-goal-resident-ledger-001",
				"goal_status":             "complete",
				"closure_status":          "accepted",
				"closure_accepted":        true,
				"evidence_refs":           []string{"evidence-ref-goal-resident-observe"},
			})
		case "/api/v0/runs/supervise":
			supervisions++
			t.Fatalf("goal-first observado por stats no debe llamar a runs/supervise")
		default:
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if err := opesBridgeRecordSubmittedV0(
		context.Background(),
		ledger,
		orquestaopesconnector.ExternalJobV0{ID: "job-ref-goal-resident-ledger-001"},
		"run-ref-goal-resident-ledger-001",
		"change-ref-goal-resident-ledger-001",
	); err != nil {
		t.Fatalf("record ledger: %v", err)
	}

	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:                  opesServer.URL,
		OrquestaBaseURL:              orquestaServer.URL,
		Limit:                        1,
		JobType:                      "draft_content_block",
		HTTPTimeout:                  time.Second,
		ResidentDispatchWait:         time.Second,
		ResidentDispatchPollInterval: time.Millisecond,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil ||
		statsCalls != 1 ||
		observes != 1 ||
		supervisions != 0 ||
		summary.AlreadySubmitted != 1 ||
		summary.Results[0].Status != "already_submitted" ||
		summary.Results[0].RoutePolicy != "goal_first" ||
		summary.Results[0].GoalRef != "goal-ref-goal-resident-ledger-001" ||
		summary.Results[0].SupervisionStatus != "closed" ||
		summary.Results[0].SupervisionStopReason != "accepted" ||
		summary.Results[0].SupervisionEvidenceRef != "evidence-ref-goal-resident-observe" {
		t.Fatalf("stats=%d observes=%d supervisions=%d summary=%+v err=%v", statsCalls, observes, supervisions, summary, err)
	}
	entry, found, err := ledger.LookupExternalBridgeInputV0(
		context.Background(),
		opesBridgeInputLedgerKeyV0("job-ref-goal-resident-ledger-001"),
	)
	if err != nil ||
		!found ||
		entry.RoutePolicy != "goal_first" ||
		entry.GoalRef != "goal-ref-goal-resident-ledger-001" ||
		entry.ExternalGoalRef != "thread-ref-goal-resident-ledger-001" ||
		!containsStringForTestV0(entry.NextActions, "observe_goal") {
		t.Fatalf("entry=%+v found=%v err=%v", entry, found, err)
	}
}

func TestRunOPESDrainOnceV0NoSupervisaPorDefectoTrasEnviarV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-autonomous-default-001",
			"type":           "draft_content_block",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque autonomo"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	submits := 0
	supervisions := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v0/runs/supervise" {
			supervisions++
			t.Fatalf("el bridge OPES no debe supervisar manualmente por defecto")
		}
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		submits++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_ref": "run-ref-autonomous-default-001",
			"estado":  "accepted",
		})
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           1,
		JobType:         "draft_content_block",
		HTTPTimeout:     time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil ||
		submits != 1 ||
		supervisions != 0 ||
		summary.Submitted != 1 ||
		summary.Results[0].Status != "submitted" ||
		summary.Results[0].SupervisionStatus != "resident_director_pending" {
		t.Fatalf("submits=%d supervisions=%d summary=%+v err=%v", submits, supervisions, summary, err)
	}
}

func TestRunOPESDrainOnceV0EsperaDespachoResidenteSinSupervisarV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-resident-dispatch-001",
			"type":           "plan_temario",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"program_id":"program-ref-resident-001","title":"Temario residente"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	statsCalls := 0
	supervisions := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/external-work/run":
			if r.Method != http.MethodPost {
				t.Fatalf("external-work method=%s", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"run_ref": "run-ref-resident-dispatch-001",
				"estado":  "accepted",
			})
		case "/api/v0/director/stats":
			if r.Method != http.MethodPost {
				t.Fatalf("director stats method=%s", r.Method)
			}
			statsCalls++
			if statsCalls == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"estado":  "ok",
					"run_ref": "run-ref-resident-dispatch-001",
					"stats": map[string]any{
						"run_ref": "run-ref-resident-dispatch-001",
						"status":  "running",
						"counts":  map[string]any{},
					},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado":  "ok",
				"run_ref": "run-ref-resident-dispatch-001",
				"stats": map[string]any{
					"run_ref": "run-ref-resident-dispatch-001",
					"status":  "running",
					"counts": map[string]any{
						"agents_started":   1,
						"agents_in_flight": 1,
					},
					"refs": map[string]any{
						"agents_started": []string{"agent-ref-resident-dispatch-001"},
					},
					"agents": []map[string]any{{
						"agent_request_id": "agent-ref-resident-dispatch-001",
						"started":          true,
						"in_flight":        true,
						"process": map[string]any{
							"process_ref":   "process-ref-resident-dispatch-001",
							"evidence_refs": []string{"evidence-ref-resident-dispatch-001"},
						},
					}},
				},
			})
		case "/api/v0/runs/supervise":
			supervisions++
			t.Fatalf("no debe llamar /runs/supervise en espera residente pasiva")
		default:
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:                  opesServer.URL,
		OrquestaBaseURL:              orquestaServer.URL,
		Limit:                        1,
		JobType:                      "plan_temario",
		HTTPTimeout:                  time.Second,
		ResidentDispatchWait:         time.Second,
		ResidentDispatchPollInterval: time.Millisecond,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil ||
		statsCalls < 2 ||
		supervisions != 0 ||
		summary.Submitted != 1 ||
		summary.Results[0].SupervisionStatus != "started" ||
		summary.Results[0].SupervisionProcessRef != "process-ref-resident-dispatch-001" ||
		summary.Results[0].SupervisionEvidenceRef != "evidence-ref-resident-dispatch-001" {
		t.Fatalf("stats=%d supervisions=%d summary=%+v err=%v", statsCalls, supervisions, summary, err)
	}
}

func TestOPESBridgeSupervisionFromDirectorStatsV0PriorizaLostSobreStarted(t *testing.T) {
	decoded := decodeOPESBridgeDirectorStatsForTestV0(t, `{
		"estado": "ok",
		"stats": {
			"status": "running",
			"counts": {
				"agents_started": 1,
				"agents_in_flight": 1,
				"agents_lost": 1
			},
			"refs": {
				"agents_started": ["agent-ref-resident-lost-001"],
				"agents_lost": ["agent-ref-resident-lost-001"]
			},
			"agents": [{
				"agent_request_id": "agent-ref-resident-lost-001",
				"started": true,
				"in_flight": true,
				"lost": true,
				"process": {
					"process_ref": "process-ref-resident-lost-001",
					"evidence_refs": ["evidence-ref-resident-lost-001"]
				}
			}]
		}
	}`)

	supervision := opesBridgeSupervisionFromDirectorStatsV0(decoded)

	if supervision.Status != "blocked" ||
		supervision.StopReason != "agent_failed_or_lost" ||
		supervision.ProcessRef != "process-ref-resident-lost-001" ||
		supervision.EvidenceRef != "evidence-ref-resident-lost-001" {
		t.Fatalf("supervision=%+v", supervision)
	}
}

func TestOPESBridgeSupervisionFromDirectorStatsV0DetectaLostEnAgenteSinContador(t *testing.T) {
	decoded := decodeOPESBridgeDirectorStatsForTestV0(t, `{
		"estado": "ok",
		"stats": {
			"status": "running",
			"counts": {
				"agents_started": 1
			},
			"agents": [{
				"agent_request_id": "agent-ref-resident-agent-lost-001",
				"started": true,
				"lost": true
			}]
		}
	}`)

	supervision := opesBridgeSupervisionFromDirectorStatsV0(decoded)

	if supervision.Status != "blocked" || supervision.StopReason != "agent_failed_or_lost" {
		t.Fatalf("supervision=%+v", supervision)
	}
}

func TestOPESBridgeSupervisionFromDirectorStatsV0DetectaDeliverySinProcesoVivo(t *testing.T) {
	decoded := decodeOPESBridgeDirectorStatsForTestV0(t, `{
		"estado": "ok",
		"stats": {
			"status": "running",
			"counts": {
				"tasks_delivered": 1,
				"agents_delivered": 1,
				"deliveries": 1
			},
			"refs": {
				"delivered_tasks": ["task-ref-opes-stale-lock-001"],
				"agents_delivered": ["agent-ref-opes-stale-lock-001"],
				"deliveries": ["delivery-ref-opes-stale-lock-001"]
			}
		}
	}`)

	supervision := opesBridgeSupervisionFromDirectorStatsV0(decoded)

	if supervision.Status != "needs_reconcile" ||
		supervision.StopReason != "stale_lock_no_process" ||
		supervision.EvidenceRef != "delivery-ref-opes-stale-lock-001" {
		t.Fatalf("supervision=%+v", supervision)
	}
}

func decodeOPESBridgeDirectorStatsForTestV0(
	t *testing.T,
	raw string,
) opesBridgeDirectorStatsResponseV0 {
	t.Helper()
	var decoded opesBridgeDirectorStatsResponseV0
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("decode director stats: %v", err)
	}
	return decoded
}

func TestRunOPESDrainOnceV0EsperaDespachoResidenteTimeoutSinSupervisarV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-resident-timeout-001",
			"type":           "plan_temario",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"program_id":"program-ref-resident-timeout-001","title":"Temario timeout"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	supervisions := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/external-work/run":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"run_ref": "run-ref-resident-timeout-001",
				"estado":  "accepted",
			})
		case "/api/v0/director/stats":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado":  "ok",
				"run_ref": "run-ref-resident-timeout-001",
				"stats": map[string]any{
					"run_ref": "run-ref-resident-timeout-001",
					"status":  "queued",
					"counts":  map[string]any{},
				},
			})
		case "/api/v0/runs/supervise":
			supervisions++
			t.Fatalf("no debe llamar /runs/supervise en espera residente pasiva")
		default:
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:                  opesServer.URL,
		OrquestaBaseURL:              orquestaServer.URL,
		Limit:                        1,
		JobType:                      "plan_temario",
		HTTPTimeout:                  time.Second,
		ResidentDispatchWait:         5 * time.Millisecond,
		ResidentDispatchPollInterval: time.Millisecond,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil ||
		supervisions != 0 ||
		summary.Submitted != 1 ||
		summary.Results[0].SupervisionStatus != "resident_director_pending" ||
		summary.Results[0].SupervisionStopReason != "resident_dispatch_wait_timeout" {
		t.Fatalf("supervisions=%d summary=%+v err=%v", supervisions, summary, err)
	}
}

func TestRunOPESDrainOnceV0PlanTemarioOperadoresEnviaExternalWorkDocumentPlanV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs/job-ref-plan-operadores-001" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":             "job-ref-plan-operadores-001",
			"type":           "plan_temario",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json": `{
				"program_id":"program-ref-operadores-001",
				"document_kind":"temario_oposicion",
				"language_code":"es",
				"title":"Temario operadores",
				"official_outline":"Operadores, tipos, precedencia, asociatividad y usos.",
				"quality_criteria":["completo","sin placeholders"],
				"target_pages_min":20,
				"target_pages_max":40
			}`,
			"correlation_id":  "corr-job-plan-operadores-001",
			"idempotency_key": "idem-job-plan-operadores-001",
			"requested_by":    "opes",
		})
	}))
	defer opesServer.Close()

	var received orquestaexternalworkrun.StartExternalWorkRunRequestV0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeOrquestaRunSuperviseOKForDrainTestV0(t, w, r) {
			return
		}
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		var envelope struct {
			ExternalWorkRunRequest orquestaexternalworkrun.StartExternalWorkRunRequestV0 `json:"external_work_run_request"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		received = envelope.ExternalWorkRunRequest
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_ref": "run-ref-plan-operadores-001",
			"estado":  "ok",
		})
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           1,
		JobType:         "plan_temario",
		JobRef:          "job-ref-plan-operadores-001",
		HTTPTimeout:     time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 90,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	work := received.AppChangeRequest.ExternalWork
	if work == nil {
		t.Fatalf("external_work nil: summary=%+v received=%+v", summary, received)
	}
	fields := work.InputFields
	if summary.Submitted != 1 ||
		summary.Results[0].Status != "submitted" ||
		summary.Results[0].RunRef != "run-ref-plan-operadores-001" ||
		work == nil ||
		work.JobRef != "job-ref-plan-operadores-001" ||
		work.WorkKind != "plan_temario" ||
		received.AppChangeRequest.AllowedWriteSet[0] != "external/opes/plan_temario/job-ref-plan-operadores-001" ||
		!domainWorkFieldValueForDrainTestV0(fields, "expected_artifact_type", orquestadomainwork.DomainDocumentPlanArtifactTypeV0) ||
		!domainWorkFieldValueForDrainTestV0(fields, "context_budget_profile", "large") ||
		!domainWorkFieldValueForDrainTestV0(fields, "document_kind", "temario_oposicion") {
		t.Fatalf("summary=%+v received=%+v work=%+v", summary, received, work)
	}
}

func TestRunOPESDrainOnceV0SecuenciaPasesSaltaTiposSinPendientesV0(t *testing.T) {
	queries := []string{}
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "10" {
			t.Fatalf("limit=%q", r.URL.Query().Get("limit"))
		}
		jobType := r.URL.Query().Get("job_type")
		queries = append(queries, jobType)
		switch jobType {
		case "draft_content_block":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case "generate_visual_asset":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id":             "job-ref-visual-001",
				"type":           "generate_visual_asset",
				"status":         "pending",
				"execution_mode": "external",
				"payload_json":   `{"topic_id":"topic-ref-001","objective":"Crear esquema visual"}`,
				"requested_by":   "opes",
			}})
		default:
			t.Fatalf("job_type inesperado=%q", jobType)
		}
	}))
	defer opesServer.Close()

	posts := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeOrquestaRunSuperviseOKForDrainTestV0(t, w, r) {
			return
		}
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		posts++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_ref": "run-ref-visual-001",
			"estado":  "accepted",
		})
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           1,
		JobTypeSequence: []string{
			"draft_content_block",
			"generate_visual_asset",
			"review_quality",
		},
		HTTPTimeout:        time.Second,
		SuperviseSubmitted: true,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if posts != 1 ||
		strings.Join(queries, ",") != "draft_content_block,generate_visual_asset" ||
		summary.SelectedJobType != "generate_visual_asset" ||
		summary.JobType != "generate_visual_asset" ||
		strings.Join(summary.EmptyJobTypes, ",") != "draft_content_block" ||
		summary.Submitted != 1 ||
		summary.Results[0].Status != "submitted" {
		t.Fatalf("posts=%d queries=%v summary=%+v", posts, queries, summary)
	}
}

func TestRunOPESDrainOnceV0SecuenciaPasesNoAvanzaSiPrimerTipoYaEnviadoV0(t *testing.T) {
	queries := []string{}
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		jobType := r.URL.Query().Get("job_type")
		queries = append(queries, jobType)
		if jobType != "draft_content_block" {
			t.Fatalf("job_type inesperado=%q", jobType)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-draft-001",
			"type":           "draft_content_block",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque 1"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	supervisions := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeOrquestaDirectorStatsNoGoalForDrainTestV0(t, w, r) {
			return
		}
		if writeOrquestaRunSuperviseOKForDrainTestV0(t, w, r) {
			supervisions++
			return
		}
		t.Fatalf("orquesta no debe recibir POST, recibido %s %s", r.Method, r.URL.Path)
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if err := opesBridgeRecordSubmittedV0(
		context.Background(),
		ledger,
		orquestaopesconnector.ExternalJobV0{ID: "job-ref-draft-001"},
		"run-ref-draft-001",
		"change-ref-draft-001",
	); err != nil {
		t.Fatalf("record ledger: %v", err)
	}

	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           1,
		JobTypeSequence: []string{
			"draft_content_block",
			"generate_visual_asset",
			"review_quality",
		},
		HTTPTimeout:        time.Second,
		SuperviseSubmitted: true,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if strings.Join(queries, ",") != "draft_content_block" ||
		summary.SelectedJobType != "draft_content_block" ||
		summary.AlreadySubmitted != 1 ||
		summary.Submitted != 0 ||
		supervisions != 1 ||
		summary.Results[0].Status != "already_submitted" ||
		summary.Results[0].RunRef != "run-ref-draft-001" {
		t.Fatalf("queries=%v supervisions=%d summary=%+v", queries, supervisions, summary)
	}
}

func TestRunOPESDrainOnceV0AlreadySubmittedStoppedReanudaYReencolaV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-draft-stopped-001",
			"type":           "draft_content_block",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque parado"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	supervisions := 0
	resumes := 0
	readyUpdates := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/runs/supervise":
			supervisions++
			if supervisions == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"estado":      "ok",
					"stop_reason": "stopped",
					"last": map[string]any{
						"status":        "stopped",
						"evidence_refs": []string{"evidence-ref-stopped"},
					},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado":      "ok",
				"stop_reason": "max_ticks",
				"last": map[string]any{
					"status":        "running",
					"process_ref":   "agent-ref-recovered",
					"evidence_refs": []string{"evidence-ref-running"},
				},
			})
		case "/api/v0/runs/control":
			resumes++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode resume: %v", err)
			}
			if body["action"] != "resume" {
				t.Fatalf("resume action=%v", body["action"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado":        "ok",
				"action":        "resume",
				"run_ref":       body["run_ref"],
				"status":        "running",
				"evidence_refs": []string{"evidence-ref-resumed"},
			})
		case "/api/v0/runs/queue/priority":
			readyUpdates++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode ready: %v", err)
			}
			if body["action"] != "set_priority" || body["status"] != "ready" {
				t.Fatalf("ready body=%+v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado": "ok",
				"action": "set_priority",
				"updated": map[string]any{
					"status":        "ready",
					"evidence_refs": []string{"evidence-ref-ready"},
				},
			})
		case "/api/v0/director/stats":
			writeOrquestaDirectorStatsNoGoalForDrainTestV0(t, w, r)
		default:
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if err := opesBridgeRecordSubmittedV0(
		context.Background(),
		ledger,
		orquestaopesconnector.ExternalJobV0{ID: "job-ref-draft-stopped-001"},
		"run-ref-draft-stopped-001",
		"change-ref-draft-stopped-001",
	); err != nil {
		t.Fatalf("record ledger: %v", err)
	}

	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:        opesServer.URL,
		OrquestaBaseURL:    orquestaServer.URL,
		Limit:              1,
		JobType:            "draft_content_block",
		HTTPTimeout:        time.Second,
		SuperviseSubmitted: true,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if supervisions != 2 ||
		resumes != 1 ||
		readyUpdates != 1 ||
		summary.AlreadySubmitted != 1 ||
		summary.Results[0].RunRecoveryStatus != "ready" ||
		summary.Results[0].SupervisionStatus != "running" ||
		summary.Results[0].SupervisionProcessRef != "agent-ref-recovered" {
		t.Fatalf("supervisions=%d resumes=%d ready=%d summary=%+v", supervisions, resumes, readyUpdates, summary)
	}
}

func TestRunOPESDrainOnceV0AlreadySubmittedDoneConOPESPendienteReanudaYReencolaV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		if jobType := r.URL.Query().Get("job_type"); jobType != "draft_content_block" {
			t.Fatalf("job_type inesperado=%q", jobType)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-draft-done-pending-001",
			"type":           "draft_content_block",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque done pendiente"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	supervisions := 0
	resumes := 0
	readyUpdates := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/runs/supervise":
			supervisions++
			if supervisions == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"estado":      "ok",
					"stop_reason": "stop_quiescent",
					"last": map[string]any{
						"status":        "done",
						"evidence_refs": []string{"evidence-ref-done-without-opes-artifact"},
					},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado":      "ok",
				"stop_reason": "max_ticks",
				"last": map[string]any{
					"status":        "running",
					"process_ref":   "agent-ref-recovered-from-done",
					"evidence_refs": []string{"evidence-ref-running-after-done"},
				},
			})
		case "/api/v0/runs/control":
			resumes++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode resume: %v", err)
			}
			if body["action"] != "resume" {
				t.Fatalf("resume action=%v", body["action"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado":        "ok",
				"status":        "running",
				"evidence_refs": []string{"evidence-ref-resumed-from-done"},
			})
		case "/api/v0/runs/queue/priority":
			readyUpdates++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode ready: %v", err)
			}
			if body["action"] != "set_priority" || body["status"] != "ready" {
				t.Fatalf("ready body=%+v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado": "ok",
				"updated": map[string]any{
					"status":        "ready",
					"evidence_refs": []string{"evidence-ref-ready-from-done"},
				},
			})
		case "/api/v0/director/stats":
			writeOrquestaDirectorStatsNoGoalForDrainTestV0(t, w, r)
		default:
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if err := opesBridgeRecordSubmittedV0(
		context.Background(),
		ledger,
		orquestaopesconnector.ExternalJobV0{ID: "job-ref-draft-done-pending-001"},
		"run-ref-draft-done-pending-001",
		"change-ref-draft-done-pending-001",
	); err != nil {
		t.Fatalf("record ledger: %v", err)
	}

	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           1,
		JobTypeSequence: []string{
			"draft_content_block",
			"generate_visual_asset",
		},
		HTTPTimeout:        time.Second,
		SuperviseSubmitted: true,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if supervisions != 2 ||
		resumes != 1 ||
		readyUpdates != 1 ||
		summary.SelectedJobType != "draft_content_block" ||
		summary.AlreadySubmitted != 1 ||
		summary.Submitted != 0 ||
		summary.Results[0].RunRecoveryStatus != "ready" ||
		summary.Results[0].SupervisionStatus != "running" ||
		summary.Results[0].SupervisionProcessRef != "agent-ref-recovered-from-done" {
		t.Fatalf("supervisions=%d resumes=%d ready=%d summary=%+v", supervisions, resumes, readyUpdates, summary)
	}
}

func TestRunOPESDrainOnceV0AlreadySubmittedDoneConOPESPendienteMarcaRetryPendingSiNoPuedeRecuperarV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-draft-done-retry-001",
			"type":           "draft_content_block",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque retry"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	supervisions := 0
	resumes := 0
	readyUpdates := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/runs/supervise":
			supervisions++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado":      "ok",
				"stop_reason": "stop_quiescent",
				"last": map[string]any{
					"status":        "done",
					"evidence_refs": []string{"evidence-ref-done-retry"},
				},
			})
		case "/api/v0/runs/control":
			resumes++
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"estado": "error"})
		case "/api/v0/runs/queue/priority":
			readyUpdates++
			t.Fatalf("no debe reencolar si resume falla")
		case "/api/v0/director/stats":
			writeOrquestaDirectorStatsNoGoalForDrainTestV0(t, w, r)
		default:
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if err := opesBridgeRecordSubmittedV0(
		context.Background(),
		ledger,
		orquestaopesconnector.ExternalJobV0{ID: "job-ref-draft-done-retry-001"},
		"run-ref-draft-done-retry-001",
		"change-ref-draft-done-retry-001",
	); err != nil {
		t.Fatalf("record ledger: %v", err)
	}

	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:        opesServer.URL,
		OrquestaBaseURL:    orquestaServer.URL,
		Limit:              1,
		JobType:            "draft_content_block",
		HTTPTimeout:        time.Second,
		SuperviseSubmitted: true,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil ||
		supervisions != 1 ||
		resumes != 1 ||
		readyUpdates != 0 ||
		summary.AlreadySubmitted != 1 ||
		summary.Submitted != 0 ||
		summary.Results[0].RunRecoveryStatus != "retry_pending" ||
		summary.Results[0].SupervisionStatus != "retry_pending" ||
		summary.Results[0].SupervisionStopReason != "pending_opes_job_terminal_supervision" {
		t.Fatalf("supervisions=%d resumes=%d ready=%d summary=%+v err=%v", supervisions, resumes, readyUpdates, summary, err)
	}
}

func TestRunOPESDrainOnceV0SupervisionTemporalNoBloqueaBridgeV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-draft-supervision-001",
			"type":           "draft_content_block",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque supervision"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	supervisions := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeOrquestaDirectorStatsNoGoalForDrainTestV0(t, w, r) {
			return
		}
		if r.URL.Path != "/api/v0/runs/supervise" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		supervisions++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"estado": "error"})
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if err := opesBridgeRecordSubmittedV0(
		context.Background(),
		ledger,
		orquestaopesconnector.ExternalJobV0{ID: "job-ref-draft-supervision-001"},
		"run-ref-draft-supervision-001",
		"change-ref-draft-supervision-001",
	); err != nil {
		t.Fatalf("record ledger: %v", err)
	}

	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:        opesServer.URL,
		OrquestaBaseURL:    orquestaServer.URL,
		Limit:              1,
		JobType:            "draft_content_block",
		HTTPTimeout:        time.Second,
		SuperviseSubmitted: true,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil ||
		supervisions != 1 ||
		summary.AlreadySubmitted != 1 ||
		len(summary.Errors) != 0 ||
		summary.Results[0].SupervisionStatus != "retry_pending" ||
		summary.Results[0].SupervisionStopReason != "supervision_unavailable" {
		t.Fatalf("supervisions=%d summary=%+v err=%v", supervisions, summary, err)
	}
}

func TestRunOPESDrainOnceV0EscaneaMasQueLimitYEnviaSiguientesNoEnviadosV0(t *testing.T) {
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "30" {
			t.Fatalf("limit=%q", r.URL.Query().Get("limit"))
		}
		jobs := []map[string]any{}
		for _, id := range []string{
			"job-ref-draft-001",
			"job-ref-draft-002",
			"job-ref-draft-003",
			"job-ref-draft-004",
			"job-ref-draft-005",
			"job-ref-draft-006",
			"job-ref-draft-007",
		} {
			jobs = append(jobs, map[string]any{
				"id":             id,
				"type":           "draft_content_block",
				"status":         "pending",
				"execution_mode": "external",
				"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque"}`,
				"requested_by":   "opes",
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jobs)
	}))
	defer opesServer.Close()

	posts := []string{}
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeOrquestaRunSuperviseOKForDrainTestV0(t, w, r) {
			return
		}
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		var envelope struct {
			ExternalWorkRunRequest orquestaexternalworkrun.StartExternalWorkRunRequestV0 `json:"external_work_run_request"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		jobRef := envelope.ExternalWorkRunRequest.AppChangeRequest.ExternalWork.JobRef
		posts = append(posts, jobRef)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_ref": "run-ref-" + jobRef,
			"estado":  "accepted",
		})
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	for _, id := range []string{"job-ref-draft-001", "job-ref-draft-002", "job-ref-draft-003"} {
		if err := opesBridgeRecordSubmittedV0(
			context.Background(),
			ledger,
			orquestaopesconnector.ExternalJobV0{ID: id},
			"run-ref-"+id,
			"change-ref-"+id,
		); err != nil {
			t.Fatalf("record ledger: %v", err)
		}
	}

	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           3,
		JobType:         "draft_content_block",
		HTTPTimeout:     time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if strings.Join(posts, ",") != "job-ref-draft-004,job-ref-draft-005,job-ref-draft-006" ||
		summary.Seen != 7 ||
		summary.AlreadySubmitted != 3 ||
		summary.Submitted != 3 {
		t.Fatalf("posts=%v summary=%+v", posts, summary)
	}
}

func TestRunOPESDrainOnceV0SecuenciaDerivadosOPESHastaAssembleTopicV0(t *testing.T) {
	sequence := []string{
		"draft_content_block",
		"generate_visual_asset",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"review_codex",
		"review_gemini",
		"review_claude",
		"review_pair_codex_gemini",
		"review_pair_codex_claude",
		"review_pair_gemini_claude",
		"review_director_consolidation",
		"validate_topic",
		"assemble_topic",
	}
	expectedArtifactByType := map[string]string{
		"draft_content_block":           "content_block",
		"generate_visual_asset":         "visual_asset",
		"review_legal":                  "block_revision",
		"review_pedagogical":            "block_revision",
		"review_quality":                "block_revision",
		"review_codex":                  "agent_review_report",
		"review_gemini":                 "agent_review_report",
		"review_claude":                 "agent_review_report",
		"review_pair_codex_gemini":      "agent_pair_review_report",
		"review_pair_codex_claude":      "agent_pair_review_report",
		"review_pair_gemini_claude":     "agent_pair_review_report",
		"review_director_consolidation": "director_review_matrix",
		"validate_topic":                "block_revision",
		"assemble_topic":                "assembled_topic",
	}
	activeStage := 0
	queries := []string{}
	submittedByType := map[string]int{}

	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		jobType := r.URL.Query().Get("job_type")
		queries = append(queries, jobType)
		if activeStage >= len(sequence) || jobType != sequence[activeStage] {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":              "job-ref-" + strings.ReplaceAll(jobType, "_", "-") + "-001",
			"type":            jobType,
			"status":          "pending",
			"execution_mode":  "external",
			"payload_json":    opesDerivedPayloadForDrainTestV0(jobType),
			"correlation_id":  "corr-ref-" + strings.ReplaceAll(jobType, "_", "-") + "-001",
			"idempotency_key": "idem-ref-" + strings.ReplaceAll(jobType, "_", "-") + "-001",
			"requested_by":    "opes",
		}})
	}))
	defer opesServer.Close()

	type postedRun struct {
		JobRef       string
		WorkKind     string
		ArtifactType string
		WriteSet     string
	}
	posts := []postedRun{}
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeOrquestaRunSuperviseOKForDrainTestV0(t, w, r) {
			return
		}
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		var envelope struct {
			ExternalWorkRunRequest orquestaexternalworkrun.StartExternalWorkRunRequestV0 `json:"external_work_run_request"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		work := envelope.ExternalWorkRunRequest.AppChangeRequest.ExternalWork
		if work == nil {
			t.Fatalf("external_work nil: %+v", envelope.ExternalWorkRunRequest)
		}
		artifactType := domainWorkFieldStringForDrainTestV0(work.InputFields, "expected_artifact_type")
		posts = append(posts, postedRun{
			JobRef:       work.JobRef,
			WorkKind:     work.WorkKind,
			ArtifactType: artifactType,
			WriteSet:     envelope.ExternalWorkRunRequest.AppChangeRequest.AllowedWriteSet[0],
		})
		submittedByType[work.WorkKind]++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_ref": "run-ref-" + work.JobRef,
			"estado":  "accepted",
		})
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	config := opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           1,
		JobTypeSequence: sequence,
		HTTPTimeout:     time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	}

	for index, jobType := range sequence {
		activeStage = index
		first, err := runOPESDrainOnceV0(context.Background(), config)
		if err != nil {
			t.Fatalf("drain %s: %v", jobType, err)
		}
		second, err := runOPESDrainOnceV0(context.Background(), config)
		if err != nil {
			t.Fatalf("drain replay %s: %v", jobType, err)
		}
		if first.SelectedJobType != jobType ||
			first.Submitted != 1 ||
			second.SelectedJobType != jobType ||
			second.Submitted != 0 ||
			second.AlreadySubmitted != 1 ||
			second.Results[0].Status != "already_submitted" {
			t.Fatalf("jobType=%s first=%+v second=%+v", jobType, first, second)
		}
		if submittedByType[jobType] != 1 {
			t.Fatalf("jobType=%s submissions=%d", jobType, submittedByType[jobType])
		}
	}

	if len(posts) != len(sequence) {
		t.Fatalf("posts=%+v", posts)
	}
	for index, post := range posts {
		jobType := sequence[index]
		if post.WorkKind != jobType ||
			post.ArtifactType != expectedArtifactByType[jobType] ||
			post.WriteSet != "external/opes/"+jobType+"/"+post.JobRef {
			t.Fatalf("index=%d post=%+v expected=%s", index, post, expectedArtifactByType[jobType])
		}
	}
	last := posts[len(posts)-1]
	if last.WorkKind != "assemble_topic" || last.ArtifactType != "assembled_topic" {
		t.Fatalf("assemble no entrego assembled_topic: posts=%+v queries=%v", posts, queries)
	}
}

func domainWorkFieldValueForDrainTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	value string,
) bool {
	for _, field := range fields {
		if field.Name == name && field.Value == value {
			return true
		}
	}
	return false
}

func domainWorkFieldStringForDrainTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
) string {
	for _, field := range fields {
		if field.Name == name {
			return field.Value
		}
	}
	return ""
}

func writeOrquestaRunSuperviseOKForDrainTestV0(
	t *testing.T,
	w http.ResponseWriter,
	r *http.Request,
) bool {
	t.Helper()
	if r.URL.Path != "/api/v0/runs/supervise" {
		return false
	}
	if r.Method != http.MethodPost {
		t.Fatalf("supervise method=%s", r.Method)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"estado":      "ok",
		"stop_reason": "max_ticks",
		"last": map[string]any{
			"status":        "running",
			"process_ref":   "process-ref-supervise-test",
			"evidence_refs": []string{"evidence-ref-supervise-test"},
		},
	})
	return true
}

func writeOrquestaDirectorStatsNoGoalForDrainTestV0(
	t *testing.T,
	w http.ResponseWriter,
	r *http.Request,
) bool {
	t.Helper()
	if r.URL.Path != "/api/v0/director/stats" {
		return false
	}
	if r.Method != http.MethodPost {
		t.Fatalf("director stats method=%s", r.Method)
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	runRef, _ := body["run_ref"].(string)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"estado": "ok",
		"stats": map[string]any{
			"run_ref": runRef,
			"status":  "running",
		},
	})
	return true
}

func opesDerivedPayloadForDrainTestV0(jobType string) string {
	switch jobType {
	case "draft_content_block":
		return `{"program_id":"program-ref-operadores-001","topic_id":"topic-ref-operadores-001","section_ref":"section-ref-001","title":"Bloque operadores"}`
	case "generate_visual_asset":
		return `{"program_id":"program-ref-operadores-001","topic_id":"topic-ref-operadores-001","visual_ref":"visual-ref-001","objective":"Diagrama de precedencia"}`
	case "assemble_topic":
		return `{"program_id":"program-ref-operadores-001","topic_id":"topic-ref-operadores-001","document_plan_artifact_id":"artifact-plan-operadores-001","content_block_artifact_refs":["artifact-block-001"],"visual_asset_artifact_refs":["artifact-visual-001"],"review_artifact_refs":["artifact-review-001"],"validation_artifact_ref":"artifact-validation-001"}`
	default:
		return `{"program_id":"program-ref-operadores-001","topic_id":"topic-ref-operadores-001","document_plan_artifact_id":"artifact-plan-operadores-001","scope":"tema completo"}`
	}
}

func TestSmokeOPESDerivativesRESTWrapperFakeServerV0(t *testing.T) {
	requireLocalTCPForTestV0(t)
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 no disponible")
	}
	repoRoot := filepath.Clean("../..")
	cmd := exec.Command("bash", "scripts/smoke_opes_derivatives_rest.sh")
	cmd.Dir = repoRoot
	cmd.Env = cleanOPESSmokeEnvForDrainTestV0(os.Environ())
	cmd.Env = append(cmd.Env,
		"ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1",
		"ORQUESTA_OPES_BRIDGE_LIMIT=1",
		"SMOKE_ID=test-derivatives-rest-fake",
		"SMOKE_OUT_DIR="+filepath.Join(t.TempDir(), "out"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("script err=%v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, `"dry_run":true`) ||
		!strings.Contains(output, `"selected_job_type":"assemble_topic"`) ||
		!strings.Contains(output, `"empty_job_types":["update_topic_registry","research_exam_precedents","draft_content_block","generate_visual_asset","generate_question_bank","review_legal","review_pedagogical","review_quality","review_codex","review_gemini","review_claude","review_pair_codex_gemini","review_pair_codex_claude","review_pair_gemini_claude","review_director_consolidation","validate_topic"]`) ||
		!strings.Contains(output, `"job_ref":"job-ref-fake-assemble-topic-001"`) ||
		!strings.Contains(output, `"status":"dry_run"`) {
		t.Fatalf("stdout=%s stderr=%s", output, stderr.String())
	}
}

func TestSmokeOPESDerivativesRESTWrapperFakeServerRunUntilFinalizeV0(t *testing.T) {
	requireLocalTCPForTestV0(t)
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 no disponible")
	}
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl no disponible")
	}
	repoRoot := filepath.Clean("../..")
	cmd := exec.Command("bash", "scripts/smoke_opes_derivatives_rest.sh")
	cmd.Dir = repoRoot
	cmd.Env = cleanOPESSmokeEnvForDrainTestV0(os.Environ())
	cmd.Env = append(cmd.Env,
		"ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1",
		"ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-finalize",
		"ORQUESTA_OPES_DERIVATIVES_EXECUTE=1",
		"ORQUESTA_OPES_TEMPORAL_CONFIRM=1",
		"ORQUESTA_OPES_BRIDGE_PROGRAM_ID=program-ref-fake-operario-001",
		"ORQUESTA_OPES_BRIDGE_LIMIT=1",
		"ORQUESTA_OPES_BRIDGE_MAX_TICKS=30",
		"ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS=0",
		"SMOKE_ID=test-derivatives-rest-run-until-fake",
		"SMOKE_OUT_DIR="+filepath.Join(t.TempDir(), "out"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("script err=%v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, `"selected_job_type":"update_topic_registry"`) ||
		!strings.Contains(output, `"selected_job_type":"research_exam_precedents"`) ||
		!strings.Contains(output, `"selected_job_type":"draft_content_block"`) ||
		!strings.Contains(output, `"selected_job_type":"generate_visual_asset"`) ||
		!strings.Contains(output, `"selected_job_type":"generate_question_bank"`) ||
		!strings.Contains(output, `"selected_job_type":"review_codex"`) ||
		!strings.Contains(output, `"selected_job_type":"review_gemini"`) ||
		!strings.Contains(output, `"selected_job_type":"review_claude"`) ||
		!strings.Contains(output, `"selected_job_type":"review_pair_codex_gemini"`) ||
		!strings.Contains(output, `"selected_job_type":"review_pair_codex_claude"`) ||
		!strings.Contains(output, `"selected_job_type":"review_pair_gemini_claude"`) ||
		!strings.Contains(output, `"selected_job_type":"review_director_consolidation"`) ||
		!strings.Contains(output, `"selected_job_type":"assemble_topic"`) ||
		!strings.Contains(output, `"selected_job_type":"generate_audio_asset"`) ||
		!strings.Contains(output, `"selected_job_type":"generate_tutor_assets"`) ||
		!strings.Contains(output, `"selected_job_type":"generate_learning_games"`) ||
		!strings.Contains(output, `"selected_job_type":"generate_html_site"`) ||
		!strings.Contains(output, `"selected_job_type":"generate_help_manual_assets"`) ||
		!strings.Contains(output, `"selected_job_type":"finalize_temario_package"`) ||
		!strings.Contains(output, `"route_policy":"goal_first"`) ||
		!strings.Contains(output, `"director_execution_mode":"goal_first"`) ||
		!strings.Contains(output, `"next_actions":["observe_goal"]`) ||
		!strings.Contains(output, `"supervision_status":"goal_first_observe_pending"`) ||
		!strings.Contains(output, `"goal_status": "complete"`) ||
		!strings.Contains(output, `"closure_status": "accepted"`) ||
		!strings.Contains(output, `"closure_accepted": true`) ||
		!strings.Contains(output, `"artifact_refs": ["artifact-ref-fake-finalize_temario_package"]`) ||
		!strings.Contains(output, `run_until_status=completed`) ||
		!strings.Contains(output, `run_until_mode=run-until-finalize`) ||
		!strings.Contains(output, `final_job_type=finalize_temario_package`) {
		t.Fatalf("stdout=%s stderr=%s", output, stderr.String())
	}
}

func TestSmokeOPESDerivativesRESTWrapperPreflightRealAceptaScopeYGoalFirstV0(t *testing.T) {
	stdout, stderr, err := runSmokeOPESDerivativesPreflightForTestV0(t,
		"ORQUESTA_OPES_BRIDGE_PROGRAM_ID=program-ref-operadores-preflight",
		"ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1",
		"ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio",
	)
	if err != nil {
		t.Fatalf("preflight err=%v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "preflight_status=ok") ||
		!strings.Contains(stdout, "preflight_target_mode=run-until-finalize") ||
		!strings.Contains(stdout, "preflight_scope=program_id") {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestSmokeOPESDerivativesRESTWrapperPreflightRealBloqueaProgramIDSinFiltroConfirmadoV0(t *testing.T) {
	stdout, stderr, err := runSmokeOPESDerivativesPreflightForTestV0(t,
		"ORQUESTA_OPES_BRIDGE_PROGRAM_ID=program-ref-operadores-preflight",
		"ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio",
	)
	if err == nil ||
		!strings.Contains(stderr, "program_id documental no basta") {
		t.Fatalf("err=%v stdout=%s stderr=%s", err, stdout, stderr)
	}
}

func TestSmokeOPESDerivativesRESTWrapperPreflightRealBloqueaProductivoV0(t *testing.T) {
	stdout, stderr, err := runSmokeOPESDerivativesPreflightForTestV0(t,
		"ORQUESTA_OPES_BRIDGE_PROGRAM_ID=program-ref-operadores-preflight",
		"ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1",
		"ORQUESTA_OPES_BRIDGE_PRODUCTIVE_CONFIRM=1",
		"ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio",
	)
	if err == nil ||
		!strings.Contains(stderr, "no ejecutar esta cadena contra OPES productivo") {
		t.Fatalf("err=%v stdout=%s stderr=%s", err, stdout, stderr)
	}
}

func TestSmokeOPESDerivativesRESTWrapperPreflightRealBloqueaSinGoalFirstV0(t *testing.T) {
	stdout, stderr, err := runSmokeOPESDerivativesPreflightForTestV0(t,
		"ORQUESTA_OPES_BRIDGE_PROGRAM_ID=program-ref-operadores-preflight",
		"ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1",
		"ORQUESTA_CODEX_GOAL_BACKEND=",
	)
	if err == nil ||
		!strings.Contains(stderr, "falta Orquesta temporal goal-first") {
		t.Fatalf("err=%v stdout=%s stderr=%s", err, stdout, stderr)
	}
}

func TestSmokeOPESPlanTemarioWrapperFakeServerV0(t *testing.T) {
	requireLocalTCPForTestV0(t)
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 no disponible")
	}
	repoRoot := filepath.Clean("../..")
	cmd := exec.Command("bash", "scripts/smoke_opes_plan_temario_operadores.sh")
	cmd.Dir = repoRoot
	cmd.Env = cleanOPESSmokeEnvForDrainTestV0(os.Environ())
	cmd.Env = append(cmd.Env,
		"ORQUESTA_OPES_PLAN_TEMARIO_FAKE_SERVER=1",
		"SMOKE_ID=test-plan-temario-fake",
		"SMOKE_OUT_DIR="+filepath.Join(t.TempDir(), "out"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("script err=%v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, `"dry_run":true`) ||
		!strings.Contains(output, `"job_type":"plan_temario"`) ||
		!strings.Contains(output, `"job_ref":"job-ref-fake-plan-temario-operadores-001"`) ||
		!strings.Contains(output, `"status":"dry_run"`) {
		t.Fatalf("stdout=%s stderr=%s", output, stderr.String())
	}
}

func TestSmokeOPESPlanTemarioWrapperFakeServerDrainOnceV0(t *testing.T) {
	requireLocalTCPForTestV0(t)
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 no disponible")
	}
	repoRoot := filepath.Clean("../..")
	cmd := exec.Command("bash", "scripts/smoke_opes_plan_temario_operadores.sh")
	cmd.Dir = repoRoot
	cmd.Env = cleanOPESSmokeEnvForDrainTestV0(os.Environ())
	cmd.Env = append(cmd.Env,
		"ORQUESTA_OPES_PLAN_TEMARIO_FAKE_SERVER=1",
		"ORQUESTA_OPES_PLAN_TEMARIO_SMOKE_MODE=drain-once",
		"ORQUESTA_OPES_PLAN_TEMARIO_EXECUTE=1",
		"ORQUESTA_OPES_BRIDGE_WAIT_RESIDENT_SECONDS=1",
		"ORQUESTA_OPES_BRIDGE_WAIT_RESIDENT_INTERVAL_MS=50",
		"SMOKE_ID=test-plan-temario-fake-drain",
		"SMOKE_OUT_DIR="+filepath.Join(t.TempDir(), "out"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("script err=%v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	output := stdout.String()
	if strings.Contains(output, `"dry_run":true`) ||
		!strings.Contains(output, `"job_type":"plan_temario"`) ||
		!strings.Contains(output, `"job_ref":"job-ref-fake-plan-temario-operadores-001"`) ||
		!strings.Contains(output, `"status":"submitted"`) ||
		!strings.Contains(output, `"supervision_status":"started"`) ||
		!strings.Contains(output, `"supervision_process_ref":"process-ref-fake-plan-temario-001"`) ||
		!strings.Contains(output, `"supervision_evidence_ref":"evidence-ref-fake-plan-temario-dispatch"`) {
		t.Fatalf("stdout=%s stderr=%s", output, stderr.String())
	}
}

func runSmokeOPESDerivativesPreflightForTestV0(
	t *testing.T,
	extraEnv ...string,
) (string, string, error) {
	t.Helper()
	repoRoot := filepath.Clean("../..")
	cmd := exec.Command("bash", "scripts/smoke_opes_derivatives_rest.sh")
	cmd.Dir = repoRoot
	cmd.Env = cleanOPESSmokeEnvForDrainTestV0(os.Environ())
	cmd.Env = append(cmd.Env,
		"ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=preflight-only",
		"ORQUESTA_OPES_DERIVATIVES_PREFLIGHT_TARGET_MODE=run-until-finalize",
		"ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1",
		"ORQUESTA_OPES_TEMPORAL_CONFIRM=1",
		"ORQUESTA_OPES_DERIVATIVES_EXECUTE=1",
		"ORQUESTA_OPES_BASE_URL=http://127.0.0.1:19090",
		"ORQUESTA_BASE_URL=http://127.0.0.1:19091",
		"ORQUESTA_OPES_BRIDGE_LIMIT=1",
		"SMOKE_ID=test-derivatives-rest-preflight",
		"SMOKE_OUT_DIR="+filepath.Join(t.TempDir(), "out"),
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func cleanOPESSmokeEnvForDrainTestV0(env []string) []string {
	out := make([]string, 0, len(env))
	for _, item := range env {
		if strings.HasPrefix(item, "ORQUESTA_OPES") ||
			strings.HasPrefix(item, "ORQUESTA_CODEX_GOAL_BACKEND=") ||
			strings.HasPrefix(item, "OPES_BASE_URL=") ||
			strings.HasPrefix(item, "ORQUESTA_BASE_URL=") ||
			strings.HasPrefix(item, "SMOKE_ID=") ||
			strings.HasPrefix(item, "SMOKE_OUT_DIR=") {
			continue
		}
		out = append(out, item)
	}
	return out
}
