package cmd

import (
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestBuildServerOperationalInfoToleraTrabajoConfirmadoSinWorkingAgents(t *testing.T) {
	info := buildServerOperationalInfo(apiStatusResponse{
		TareasEnProgreso: []tareaLite{{ID: 1, Estado: db.TareaEnProgreso}},
		Autonomia: autonomiaResumen{
			WorkConfirmed: 1,
		},
	})
	if !info.Operational || info.State != "ready" {
		t.Fatalf("no deberia degradar si la autonomia ya confirmo trabajo: %+v", info)
	}
}

func TestBuildServerOperationalInfoExponeQuotaBlockedSinWorkersActivos(t *testing.T) {
	reset := time.Now().UTC().Add(30 * time.Minute)
	info := buildServerOperationalInfo(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &reset},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &reset},
		},
	})
	if !info.Operational {
		t.Fatalf("el control plane sigue operativo aunque la flota este en cuota: %+v", info)
	}
	if info.State != "idle" || info.Reason != "workers_quota_blocked" || info.QuotaAgents != 1 {
		t.Fatalf("estado operativo inesperado: %+v", info)
	}
	if info.NextQuotaResetAt == "" {
		t.Fatalf("deberia exponer el siguiente quota reset visible: %+v", info)
	}
}

func TestBuildServerOperationalInfoUsaCuotaVisibleAunqueSnapshotNoTraigaQuotaBlockedExplicito(t *testing.T) {
	reset := time.Now().UTC().Add(20 * time.Minute)
	info := buildServerOperationalInfo(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &reset},
		},
	})
	if info.QuotaAgents != 1 {
		t.Fatalf("deberia derivar quota agents desde el snapshot visible: %+v", info)
	}
	if info.NextQuotaResetAt == "" {
		t.Fatalf("deberia derivar el próximo reset visible aunque no venga agentesQuotaBlocked: %+v", info)
	}
}

func TestFormatServerOperationalSummaryIncluyeDispatch(t *testing.T) {
	summary := formatServerOperationalSummary(&serverOperationalInfo{
		ActiveAgents:        2,
		RegisteredAgents:    4,
		DispatchPending:     1,
		DispatchNotified:    2,
		DispatchFailed:      3,
		DispatchConfirmed:   4,
		AutonomySupervising: 1,
		AutonomyContinuing:  2,
		AutonomyPending:     3,
		AutonomyConfirmed:   4,
		AutonomyHandoffs:    5,
		QuotaAgents:         6,
		NextQuotaResetAt:    "2026-04-20T21:05:00Z",
	})
	if !strings.Contains(summary, "dispatch p:1 n:2 f:3 c:4") {
		t.Fatalf("summary sin dispatch confirmado: %s", summary)
	}
	if !strings.Contains(summary, "autonomia s:1 c:2 p:3 ok:4 h:5") {
		t.Fatalf("summary sin resumen de autonomia: %s", summary)
	}
	if !strings.Contains(summary, "6 bloqueados_cuota") || !strings.Contains(summary, "quota_reset 2026-04-20T21:05:00Z") {
		t.Fatalf("summary sin informacion de cuota visible: %s", summary)
	}
}
