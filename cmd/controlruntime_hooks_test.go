package cmd

import (
	"testing"

	"orquesta/internal/controlruntime"
)

func TestProcessSupervisorSignalIgnoraSenalesIncompletas(t *testing.T) {
	if err := processSupervisorSignal(controlruntime.SupervisorSignal{}); err != nil {
		t.Fatalf("una señal incompleta debería ignorarse sin error: %v", err)
	}
	if err := processSupervisorSignal(controlruntime.SupervisorSignal{Agente: "Codex1"}); err != nil {
		t.Fatalf("una señal sin proyecto debería ignorarse sin error: %v", err)
	}
	if err := processSupervisorSignal(controlruntime.SupervisorSignal{Proyecto: "orquestador"}); err != nil {
		t.Fatalf("una señal sin agente debería ignorarse sin error: %v", err)
	}
}
