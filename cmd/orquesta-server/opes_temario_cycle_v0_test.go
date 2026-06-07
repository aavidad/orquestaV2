package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
)

func TestOPESTemarioCycleConfigUsaSecuenciaCompletaPorDefectoV0(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "http://127.0.0.1:18080")
	t.Setenv("ORQUESTA_OPES_BRIDGE_DRY_RUN", "1")

	config, err := opesTemarioCycleConfigFromEnvV0()

	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if len(config.DrainConfig.JobTypeSequence) < 20 ||
		config.FinalJobType != "finalize_temario_package" ||
		config.MaxTicks != defaultOPESTemarioCycleMaxTicksV0 ||
		config.TickSleep != defaultOPESTemarioCycleIntervalSeconds*time.Second {
		t.Fatalf("config=%+v", config)
	}
}

func TestRunOPESTemarioCycleV0CierraCuandoSecuenciaVaciaTrasFinalV0(t *testing.T) {
	sequence := []string{"draft_content_block", "finalize_temario_package"}
	calls := 0
	summary := runOPESTemarioCycleV0(context.Background(), opesTemarioCycleConfigV0{
		DrainConfig: opesDrainConfigV0{
			Limit:           1,
			JobTypeSequence: sequence,
			HTTPTimeout:     time.Second,
		},
		MaxTicks:     5,
		TickSleep:    0,
		FinalJobType: "finalize_temario_package",
	}, func(_ context.Context, _ opesDrainConfigV0) (opesDrainSummaryV0, error) {
		calls++
		switch calls {
		case 1:
			return opesTemarioCycleTickSummaryForTestV0(sequence, "draft_content_block", 1), nil
		case 2:
			return opesTemarioCycleTickSummaryForTestV0(sequence, "finalize_temario_package", 1), nil
		default:
			return opesTemarioCycleEmptySummaryForTestV0(sequence), nil
		}
	})

	if !summary.Completed ||
		!summary.FinalSeen ||
		summary.StopReason != "sequence_empty_after_final" ||
		summary.Ticks != 3 ||
		strings.Join(summary.SelectedJobTypes, ",") != "draft_content_block,finalize_temario_package" {
		t.Fatalf("summary=%+v calls=%d", summary, calls)
	}
}

func TestRunOPESTemarioCycleV0ConservaErroresRecuperablesYContinuaV0(t *testing.T) {
	sequence := []string{"draft_content_block", "finalize_temario_package"}
	calls := 0
	summary := runOPESTemarioCycleV0(context.Background(), opesTemarioCycleConfigV0{
		DrainConfig: opesDrainConfigV0{
			Limit:           1,
			JobTypeSequence: sequence,
			HTTPTimeout:     time.Second,
		},
		MaxTicks:     5,
		TickSleep:    0,
		FinalJobType: "finalize_temario_package",
	}, func(_ context.Context, _ opesDrainConfigV0) (opesDrainSummaryV0, error) {
		calls++
		switch calls {
		case 1:
			tick := opesTemarioCycleTickSummaryForTestV0(sequence, "draft_content_block", 1)
			tick.Errors = []opesDrainPublicErrorV0{{JobRef: "job-ref-draft-001", Code: "context_temporal_unavailable"}}
			return tick, nil
		case 2:
			return opesTemarioCycleTickSummaryForTestV0(sequence, "finalize_temario_package", 1), nil
		default:
			return opesTemarioCycleEmptySummaryForTestV0(sequence), nil
		}
	})

	if !summary.Completed ||
		len(summary.Errors) != 1 ||
		summary.Errors[0].Code != "context_temporal_unavailable" ||
		summary.StopReason != "sequence_empty_after_final" {
		t.Fatalf("summary=%+v", summary)
	}
}

func TestRunOPESTemarioCycleV0SoloParaPorMaxTicksSiFinalNoDesapareceV0(t *testing.T) {
	sequence := []string{"finalize_temario_package"}
	summary := runOPESTemarioCycleV0(context.Background(), opesTemarioCycleConfigV0{
		DrainConfig: opesDrainConfigV0{
			Limit:           1,
			JobTypeSequence: sequence,
			HTTPTimeout:     time.Second,
		},
		MaxTicks:     2,
		TickSleep:    0,
		FinalJobType: "finalize_temario_package",
	}, func(_ context.Context, _ opesDrainConfigV0) (opesDrainSummaryV0, error) {
		tick := opesTemarioCycleTickSummaryForTestV0(sequence, "finalize_temario_package", 1)
		tick.AlreadySubmitted = 1
		tick.Submitted = 0
		return tick, nil
	})

	if summary.Completed ||
		!summary.FinalSeen ||
		summary.StopReason != "max_ticks_reached" ||
		summary.Ticks != 2 {
		t.Fatalf("summary=%+v", summary)
	}
}

func TestRunOPESTemarioCycleV0NoParaPorVacioAntesDelFinalV0(t *testing.T) {
	sequence := []string{"draft_content_block", "finalize_temario_package"}
	summary := runOPESTemarioCycleV0(context.Background(), opesTemarioCycleConfigV0{
		DrainConfig: opesDrainConfigV0{
			Limit:           1,
			JobTypeSequence: sequence,
			HTTPTimeout:     time.Second,
		},
		MaxTicks:     2,
		TickSleep:    0,
		FinalJobType: "finalize_temario_package",
	}, func(_ context.Context, _ opesDrainConfigV0) (opesDrainSummaryV0, error) {
		return opesTemarioCycleEmptySummaryForTestV0(sequence), nil
	})

	if summary.Completed ||
		summary.FinalSeen ||
		summary.StopReason != "max_ticks_reached" ||
		summary.Ticks != 2 {
		t.Fatalf("summary=%+v", summary)
	}
}

func TestRunOPESTemarioCycleV0ConBridgeRealFakeHastaCierreV0(t *testing.T) {
	sequence := []string{"draft_content_block", "finalize_temario_package"}
	activeStage := 0
	runs := map[string]int{}
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		jobType := r.URL.Query().Get("job_type")
		w.Header().Set("Content-Type", "application/json")
		if activeStage >= len(sequence) || jobType != sequence[activeStage] {
			_ = json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-cycle-" + strings.ReplaceAll(jobType, "_", "-") + "-001",
			"type":           jobType,
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   opesDerivedPayloadForDrainTestV0(jobType),
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/external-work/run":
			var envelope map[string]any
			if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
				t.Fatalf("decode run: %v", err)
			}
			runRef := "run-ref-cycle-" + sequence[activeStage]
			runs[runRef] = activeStage
			_ = json.NewEncoder(w).Encode(map[string]string{"run_ref": runRef, "estado": "accepted"})
		case "/api/v0/runs/supervise":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode supervise: %v", err)
			}
			runRef, _ := body["run_ref"].(string)
			if stage, ok := runs[runRef]; ok && stage == activeStage {
				activeStage++
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"estado":      "ok",
				"stop_reason": "max_ticks",
				"last": map[string]any{
					"status":        "running",
					"process_ref":   "process-ref-cycle",
					"evidence_refs": []string{"evidence-ref-cycle"},
				},
			})
		default:
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
	}))
	defer orquestaServer.Close()
	ledger, err := newFileExternalBridgeInputLedgerV0(t.TempDir() + "/ledger.json")
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}

	summary := runOPESTemarioCycleV0(context.Background(), opesTemarioCycleConfigV0{
		DrainConfig: opesDrainConfigV0{
			OPESBaseURL:     opesServer.URL,
			OrquestaBaseURL: orquestaServer.URL,
			Limit:           1,
			JobTypeSequence: sequence,
			HTTPTimeout:     time.Second,
			RunConfig:       orquestaopesbridge.JobRunConfigV0{PriorityScore: 70, RequestedBy: "test"},
			InputLedger:     ledger,
		},
		MaxTicks:     5,
		TickSleep:    0,
		FinalJobType: "finalize_temario_package",
	}, runOPESDrainOnceV0)

	if !summary.Completed ||
		summary.StopReason != "sequence_empty_after_final" ||
		strings.Join(summary.SelectedJobTypes, ",") != "draft_content_block,finalize_temario_package" ||
		activeStage != len(sequence) {
		t.Fatalf("summary=%+v active=%d", summary, activeStage)
	}
}

func TestCommandPublicOPESTemarioCyclePayloadV0RedactaURLsYTicksV0(t *testing.T) {
	sequence := []string{"draft_content_block", "finalize_temario_package"}
	payload := commandPublicOPESTemarioCyclePayloadV0(opesTemarioCycleSummaryV0{
		OPESBaseURL:      "http://127.0.0.1:18080/private",
		OrquestaBaseURL:  "http://127.0.0.1:8787",
		Limit:            1,
		JobTypeSequence:  sequence,
		FinalJobType:     "finalize_temario_package",
		MaxTicks:         5,
		Ticks:            2,
		Completed:        true,
		FinalSeen:        true,
		StopReason:       "sequence_empty_after_final",
		SelectedJobTypes: []string{"draft_content_block", "finalize_temario_package"},
		TickSummaries: []opesDrainSummaryV0{{
			OPESBaseURL:     "http://127.0.0.1:18080/private",
			OrquestaBaseURL: "http://127.0.0.1:8787",
			Limit:           1,
			JobTypeSequence: sequence,
			SelectedJobType: "draft_content_block",
			Seen:            1,
			Destination: opesDrainDestinationPolicyV0{
				OPESDestination:        opesDrainDestinationV0{Category: "loopback", URLRef: "opes-base-url-ref-abc"},
				OrquestaDestination:    opesDrainDestinationV0{Category: "loopback", URLRef: "orquesta-base-url-ref-def"},
				DestinationEvidenceRef: "opes-destination-evidence-ref-abc",
			},
		}},
		Errors: []opesDrainPublicErrorV0{{Code: `Get "http://127.0.0.1/private": token=abc`}},
		Destination: opesDrainDestinationPolicyV0{
			OPESDestination:        opesDrainDestinationV0{Category: "loopback", URLRef: "opes-base-url-ref-abc"},
			OrquestaDestination:    opesDrainDestinationV0{Category: "loopback", URLRef: "orquesta-base-url-ref-def"},
			DestinationEvidenceRef: "opes-destination-evidence-ref-abc",
		},
	})

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if payload.Status != "completed" ||
		payload.StopReason != "sequence_empty_after_final" ||
		len(payload.TickSummaries) != 1 ||
		len(payload.ErrorCodes) != 1 ||
		payload.ErrorCodes[0] != "external_error" {
		t.Fatalf("payload=%+v", payload)
	}
	if strings.Contains(string(data), "127.0.0.1") ||
		strings.Contains(string(data), "token=abc") {
		t.Fatalf("payload filtra detalle local: %s", string(data))
	}
}

func opesTemarioCycleTickSummaryForTestV0(
	sequence []string,
	selected string,
	seen int,
) opesDrainSummaryV0 {
	return opesDrainSummaryV0{
		Limit:           1,
		JobType:         selected,
		JobTypeSequence: append([]string(nil), sequence...),
		SelectedJobType: selected,
		Seen:            seen,
		Submitted:       seen,
	}
}

func opesTemarioCycleEmptySummaryForTestV0(sequence []string) opesDrainSummaryV0 {
	return opesDrainSummaryV0{
		Limit:           1,
		JobTypeSequence: append([]string(nil), sequence...),
		EmptyJobTypes:   append([]string(nil), sequence...),
	}
}
