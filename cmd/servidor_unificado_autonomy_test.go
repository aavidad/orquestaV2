package cmd

import (
	"context"
	"testing"
	"time"

	"orquesta/db"
)

func TestLaunchServerAutonomyMaintenanceLoopEjecutaPasadaInmediata(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.ConfigSet("server_autobootstrap_enabled", "true"); err != nil {
		t.Fatalf("config set server_autobootstrap_enabled: %v", err)
	}
	resetControlPlaneConfigCache()

	prevBatch := runtimeProcessDegradadosBatchDetailed
	t.Cleanup(func() {
		runtimeProcessDegradadosBatchDetailed = prevBatch
		runtimeProcessDegradadosEnCurso.Store(false)
	})

	called := make(chan struct{}, 1)
	runtimeProcessDegradadosBatchDetailed = func() (runtimeProcessDegradadosSummary, error) {
		select {
		case called <- struct{}{}:
		default:
		}
		return runtimeProcessDegradadosSummary{}, nil
	}
	runtimeProcessDegradadosEnCurso.Store(false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	launchServerAutonomyMaintenanceLoop(ctx, nil)

	select {
	case <-called:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("deberia ejecutar una pasada inmediata de runtime_process_degradados al arrancar")
	}
}
