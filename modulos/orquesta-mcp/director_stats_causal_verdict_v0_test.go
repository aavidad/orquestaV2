package orquestamcp

import (
	"context"
	"testing"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestMCPDirectorStatsToolExecutorV0PublicaVeredictoCausalProcesoMuertoV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-causal-dead-001")
	estadoVivo := &fakeMCPAutoprogrammingEstadoVivoSourceV0{
		evidencias: []orquestaestadovivo.EvidenciaEstadoV0{{
			RunRef: run.RunID,
			Fuente: "state",
			Estado: "running",
		}, {
			RunRef:                      run.RunID,
			Fuente:                      "process_snapshot",
			Scope:                       orquestaestadovivo.ScopeGoalExecutionV0,
			RuntimeIdentityRef:          "runtime-ref-stats-causal-dead-001",
			RuntimeObservationAttempted: true,
			RuntimeObservado:            true,
			ProcesoVivo:                 false,
			ObservadoEn:                 "2026-07-10T09:59:00Z",
			EvidenceRefs:                []string{"evidence-ref-stats-causal-dead-snapshot"},
		}},
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:         orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		EstadoVivoSource: estadoVivo,
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RunRef:     run.RunID,
		OccurredAt: "2026-07-10T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.CausalVerdict != string(orquestaestadovivo.VeredictoProcessDeadStateStaleV0) ||
		result.CausalReasonCode != orquestaestadovivo.RazonVeredictoProcesoMuertoEstadoStaleV0 {
		t.Fatalf("veredicto causal no publicado: %+v", result)
	}
	if result.Stats == nil ||
		result.Stats.Status == "running" ||
		result.Stats.Status == mcpDirectorStatsEstadoVivoProcesoVivoV0 ||
		!result.Stats.Closure.Blocked {
		t.Fatalf("proceso muerto publicado como running: %+v", result.Stats)
	}
}

func TestMCPDirectorStatsToolExecutorV0SinFuenteNoPublicaVeredictoCausalV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-causal-nosource-001")

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RunRef:     run.RunID,
		OccurredAt: "2026-07-10T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.CausalVerdict != "" || result.CausalReasonCode != "" {
		t.Fatalf("veredicto causal sin fuente: %+v", result)
	}
}
