package cmd

import (
	"fmt"
	"testing"
	"time"

	"orquesta/db"
)

func TestRuntimeProcessDegradadosHandleDeferredLoopErrorThrottleaAuditoria(t *testing.T) {
	prepararDBTemporalCmd(t)
	resetRuntimeProcessDegradadosDeferredGate()
	prev := runtimeProcessDegradadosDeferredIntervalOverride
	runtimeProcessDegradadosDeferredIntervalOverride = time.Hour
	t.Cleanup(func() {
		runtimeProcessDegradadosDeferredIntervalOverride = prev
		resetRuntimeProcessDegradadosDeferredGate()
	})

	err := fmt.Errorf("no se pueden purgar handles con runtime orders vivas asociadas: #12150(pendiente)")
	if !runtimeProcessDegradadosHandleDeferredLoopError(err, nil) {
		t.Fatal("deberia diferir el error de purga bloqueada")
	}
	if !runtimeProcessDegradadosHandleDeferredLoopError(err, nil) {
		t.Fatal("el segundo intento dentro del cooldown sigue siendo diferible")
	}

	action := "runtime_process_degradados_deferred"
	logs, auditErr := db.ListarAuditoria(db.FiltroAuditoria{Accion: &action, Limite: 20})
	if auditErr != nil {
		t.Fatalf("listar auditoria: %v", auditErr)
	}
	if len(logs) != 1 {
		t.Fatalf("deberia auditar una sola vez dentro del cooldown, got=%d", len(logs))
	}
}

func TestRuntimeProcessDegradadosHandleDeferredLoopErrorIgnoraErroresAjenos(t *testing.T) {
	if runtimeProcessDegradadosHandleDeferredLoopError(fmt.Errorf("otro error"), nil) {
		t.Fatal("errores ajenos no deberian diferirse")
	}
}
