package db

import "testing"

func TestConfigSetActualizaCacheCorta(t *testing.T) {
	prepararDBTemporal(t)

	if err := ConfigSet("agent_tick_seconds", "30"); err != nil {
		t.Fatalf("ConfigSet inicial: %v", err)
	}
	if got, err := ConfigGet("agent_tick_seconds"); err != nil || got != "30" {
		t.Fatalf("ConfigGet inicial got=%q err=%v", got, err)
	}
	if err := ConfigSet("agent_tick_seconds", "45"); err != nil {
		t.Fatalf("ConfigSet actualizado: %v", err)
	}
	if got, err := ConfigGet("agent_tick_seconds"); err != nil || got != "45" {
		t.Fatalf("ConfigGet actualizado got=%q err=%v", got, err)
	}
}

func TestConfigSetInvalidaCacheEnteraDerivada(t *testing.T) {
	prepararDBTemporal(t)

	if err := ConfigSet("runtime_transcript_batch_budget_ms", "1500"); err != nil {
		t.Fatalf("ConfigSet inicial: %v", err)
	}
	if got := configIntOrDefault("runtime_transcript_batch_budget_ms", 0); got != 1500 {
		t.Fatalf("configIntOrDefault inicial=%d", got)
	}
	if err := ConfigSet("runtime_transcript_batch_budget_ms", "900"); err != nil {
		t.Fatalf("ConfigSet actualizado: %v", err)
	}
	if got := configIntOrDefault("runtime_transcript_batch_budget_ms", 0); got != 900 {
		t.Fatalf("configIntOrDefault actualizado=%d", got)
	}
}
