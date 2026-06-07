package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestOPESBridgeLoopDisabledByDefaultV0(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BRIDGE_ENABLED", "")

	config, err := opesBridgeLoopConfigFromEnvV0("http://127.0.0.1:8787")

	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if config.Loop.Enabled {
		t.Fatalf("enabled=%v", config.Loop.Enabled)
	}
}

func TestOPESBridgeLoopConfigUsesServerFallbackV0(t *testing.T) {
	t.Setenv("ORQUESTA_BASE_URL", "")
	t.Setenv("ORQUESTA_OPES_BRIDGE_ENABLED", "1")
	t.Setenv("ORQUESTA_OPES_BRIDGE_CONFIRM", "1")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "http://127.0.0.1:18082")
	t.Setenv("ORQUESTA_OPES_BRIDGE_LIMIT", "7")
	t.Setenv("ORQUESTA_OPES_BRIDGE_JOB_TYPE", "plan_temario")
	t.Setenv("ORQUESTA_OPES_BRIDGE_INTERVAL_SECONDS", "3")
	t.Setenv("ORQUESTA_OPES_BRIDGE_INITIAL_DELAY_SECONDS", "1")
	t.Setenv("ORQUESTA_OPES_BRIDGE_MAX_TICKS", "2")

	config, err := opesBridgeLoopConfigFromEnvV0("http://127.0.0.1:18100")

	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if !config.Loop.Enabled ||
		config.DrainConfig.OPESBaseURL != "http://127.0.0.1:18082" ||
		config.DrainConfig.OrquestaBaseURL != "http://127.0.0.1:18100" ||
		config.DrainConfig.Limit != 7 ||
		config.DrainConfig.JobType != "plan_temario" ||
		config.Loop.Interval != 3*time.Second ||
		config.Loop.InitialDelay != time.Second ||
		config.Loop.MaxTicks != 2 {
		t.Fatalf("config=%+v", config)
	}
}

func TestOPESBridgeLoopConfigAceptaSecuenciaComoFiltroSeguroV0(t *testing.T) {
	t.Setenv("ORQUESTA_BASE_URL", "")
	t.Setenv("ORQUESTA_OPES_BRIDGE_ENABLED", "1")
	t.Setenv("ORQUESTA_OPES_BRIDGE_CONFIRM", "1")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "http://127.0.0.1:18082")
	t.Setenv("ORQUESTA_OPES_BRIDGE_LIMIT", "1")
	t.Setenv(
		"ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE",
		"draft_content_block, generate_visual_asset; review_quality",
	)
	t.Setenv("ORQUESTA_OPES_BRIDGE_INTERVAL_SECONDS", "3")

	config, err := opesBridgeLoopConfigFromEnvV0("http://127.0.0.1:18100")

	if err != nil {
		t.Fatalf("config: %v", err)
	}
	wantSequence := []string{"draft_content_block", "generate_visual_asset", "review_quality"}
	if !config.Loop.Enabled ||
		config.Loop.Component != "opes_bridge_sequence_loop" ||
		config.DrainConfig.OPESBaseURL != "http://127.0.0.1:18082" ||
		config.DrainConfig.OrquestaBaseURL != "http://127.0.0.1:18100" ||
		config.DrainConfig.Limit != 1 ||
		config.Loop.Interval != 3*time.Second ||
		strings.Join(config.DrainConfig.JobTypeSequence, ",") != strings.Join(wantSequence, ",") {
		t.Fatalf("config=%+v", config)
	}
}

func TestOPESBridgeLoopConfigAceptaProgramIDComoFiltroSeguroV0(t *testing.T) {
	t.Setenv("ORQUESTA_BASE_URL", "")
	t.Setenv("ORQUESTA_OPES_BRIDGE_ENABLED", "1")
	t.Setenv("ORQUESTA_OPES_BRIDGE_CONFIRM", "1")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "http://127.0.0.1:18082")
	t.Setenv("ORQUESTA_OPES_BRIDGE_PROGRAM_ID", "program-conductores")
	t.Setenv("ORQUESTA_OPES_BRIDGE_CORRELATION_ID", "conductores-20260602")

	config, err := opesBridgeLoopConfigFromEnvV0("http://127.0.0.1:18100")

	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if !config.Loop.Enabled ||
		config.DrainConfig.ProgramID != "program-conductores" ||
		config.DrainConfig.CorrelationID != "conductores-20260602" {
		t.Fatalf("config=%+v", config)
	}
	if !containsStringOPESBridgeLoopTestV0(config.Loop.FilterSummary, "program_id=configured") ||
		!containsStringOPESBridgeLoopTestV0(config.Loop.FilterSummary, "correlation_id=configured") {
		t.Fatalf("filter summary=%+v", config.Loop.FilterSummary)
	}
}

func TestOPESBridgeLoopConfigRechazaSecuenciaAmbiguaConJobTypeV0(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BRIDGE_ENABLED", "1")
	t.Setenv("ORQUESTA_OPES_BRIDGE_CONFIRM", "1")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "http://127.0.0.1:18082")
	t.Setenv("ORQUESTA_OPES_BRIDGE_JOB_TYPE", "plan_temario")
	t.Setenv("ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE", "draft_content_block,review_quality")

	config, err := opesBridgeLoopConfigFromEnvV0("http://127.0.0.1:18100")

	if err == nil ||
		!strings.Contains(err.Error(), "ORQUESTA_OPES_BRIDGE_JOB_TYPE incompatible") ||
		config.Loop.Enabled {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}

func containsStringOPESBridgeLoopTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestOPESBridgeLoopConfigRequiereConfirmacionV0(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BRIDGE_ENABLED", "1")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "http://127.0.0.1:18082")
	t.Setenv("ORQUESTA_OPES_BRIDGE_JOB_TYPE", "plan_temario")

	config, err := opesBridgeLoopConfigFromEnvV0("http://127.0.0.1:18100")

	if err == nil ||
		!strings.Contains(err.Error(), "ORQUESTA_OPES_BRIDGE_CONFIRM") ||
		config.Loop.Enabled {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}

func TestOPESBridgeLoopConfigRequiereFiltroSeguroV0(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BRIDGE_ENABLED", "1")
	t.Setenv("ORQUESTA_OPES_BRIDGE_CONFIRM", "1")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "http://127.0.0.1:18082")

	config, err := opesBridgeLoopConfigFromEnvV0("http://127.0.0.1:18100")

	if err == nil ||
		!strings.Contains(err.Error(), "ORQUESTA_OPES_BRIDGE_JOB_TYPE") ||
		config.Loop.Enabled {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}

func TestOPESBridgeLoopRunsBoundedTicksV0(t *testing.T) {
	calls := 0
	drainer := func(context.Context, opesDrainConfigV0) (opesDrainSummaryV0, error) {
		calls++
		return opesDrainSummaryV0{Seen: 1, Submitted: 1}, nil
	}
	var stderr bytes.Buffer

	runOPESBridgeLoopV0(
		context.Background(),
		opesBridgeLoopConfigV0{
			Loop: externalBridgeLoopConfigV0{
				Enabled:      true,
				Component:    "opes_bridge_loop",
				ResultField:  "summary",
				Interval:     time.Millisecond,
				InitialDelay: 0,
				MaxTicks:     2,
			},
		},
		&stderr,
		drainer,
	)

	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
	if got := stderr.String(); !strings.Contains(got, `"component":"opes_bridge_loop"`) ||
		!strings.Contains(got, `"submitted":1`) {
		t.Fatalf("stderr=%s", got)
	}
}

func TestOPESBridgeLoopWaitReportaShutdownTimeoutV0(t *testing.T) {
	events := []externalBridgeLoopEventV0{}
	done := make(chan struct{})
	ok := waitOPESBridgeLoopDoneV0(
		context.Background(),
		opesBridgeLoopConfigV0{
			Loop: externalBridgeLoopConfigV0{
				Component:     "opes_bridge_loop",
				EffectTimeout: time.Nanosecond,
				Observer: func(_ context.Context, event externalBridgeLoopEventV0) {
					events = append(events, event)
				},
			},
		},
		done,
	)

	if ok || len(events) != 1 ||
		events[0].Status != "timeout" ||
		events[0].ErrorCode != "external_bridge_shutdown_timeout" {
		t.Fatalf("ok=%v events=%+v", ok, events)
	}
}
