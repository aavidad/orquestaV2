package cmd

import (
	"testing"
	"time"

	"orquesta/db"
)

func TestControlPlaneConfigIntOrDefaultAplicaSueloSeguroEnClavesCalientes(t *testing.T) {
	prepararDBTemporalCmd(t)

	casos := []struct {
		clave    string
		valor    string
		fallback int
		want     int
	}{
		{clave: "runtime_mailbox_reevaluation_interval_seconds", valor: "1", fallback: 10, want: 15},
		{clave: "autonomia_continue_nudge_interval_seconds", valor: "5", fallback: 60, want: 30},
		{clave: "runtime_budget_background_observation_interval_seconds", valor: "2", fallback: 120, want: 60},
		{clave: "autonomia_active_sessions_interval_seconds", valor: "3", fallback: 60, want: 30},
		{clave: "autonomia_idle_autoassign_interval_seconds", valor: "10", fallback: 300, want: 120},
		{clave: "pipeline_local_dispatch_interval_seconds", valor: "20", fallback: 180, want: 60},
	}

	for _, tc := range casos {
		if err := db.ConfigSet(tc.clave, tc.valor); err != nil {
			t.Fatalf("config set %s: %v", tc.clave, err)
		}
		resetControlPlaneConfigCache()
		if got := controlPlaneConfigIntOrDefault(tc.clave, tc.fallback); got != tc.want {
			t.Fatalf("%s=%s => got %d want %d", tc.clave, tc.valor, got, tc.want)
		}
	}
}

func TestControlPlaneConfigIntOrDefaultConservaValorSano(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.ConfigSet("runtime_budget_background_observation_interval_seconds", "180"); err != nil {
		t.Fatalf("config set: %v", err)
	}
	resetControlPlaneConfigCache()
	if got := controlPlaneConfigIntOrDefault("runtime_budget_background_observation_interval_seconds", 120); got != 180 {
		t.Fatalf("got %d want 180", got)
	}
}

func TestServerAutonomyMaintenanceIntervalAplicaSueloSeguro(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.ConfigSet("server_autonomy_maintenance_interval_seconds", "5"); err != nil {
		t.Fatalf("config set: %v", err)
	}
	resetControlPlaneConfigCache()
	if got := serverAutonomyMaintenanceInterval(); got != 30*time.Second {
		t.Fatalf("got %s want 30s", got)
	}
}
