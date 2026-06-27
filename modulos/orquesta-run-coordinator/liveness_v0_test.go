package orquestaruncoordinator

import "testing"

func TestClassifyRunLivenessV0MarcaRunningLiveConProcesoVivo(t *testing.T) {
	got := ClassifyRunLivenessV0(RunLivenessInputV0{
		AgentsInFlight: 1,
		Agents: []RunLivenessAgentV0{{
			Status:        "running",
			InFlight:      true,
			ProcessRef:    "process-ref-live-001",
			ProcessStatus: "running",
		}},
	})

	if got.Class != RunLivenessClassRunningLiveV0 ||
		got.AgentsLive != 1 ||
		!got.ProcessObserved ||
		!got.Live ||
		!got.Verifiable ||
		got.ConfirmedNoLiveProcess ||
		got.UnknownProcess ||
		!got.Running {
		t.Fatalf("classification=%+v", got)
	}
}

func TestClassifyRunLivenessV0MarcaRunningStaleNoProcessConParadoConfirmado(t *testing.T) {
	got := ClassifyRunLivenessV0(RunLivenessInputV0{
		AgentsInFlight: 1,
		Agents: []RunLivenessAgentV0{{
			Status:        "running",
			InFlight:      true,
			ProcessRef:    "process-ref-stopped-001",
			ProcessStatus: "stopped",
		}},
	})

	if got.Class != RunLivenessClassRunningStaleNoProcessV0 ||
		got.AgentsLive != 0 ||
		!got.ProcessObserved ||
		!got.ConfirmedNoLiveProcess ||
		!got.Stale ||
		!got.Verifiable ||
		!got.SafeToReconcile ||
		got.QueueStatusSuggestion != "stopped" ||
		got.UnknownProcess ||
		!got.Running {
		t.Fatalf("classification=%+v", got)
	}
}

func TestClassifyRunLivenessV0NoReconciliaSinProcesoObservado(t *testing.T) {
	got := ClassifyRunLivenessV0(RunLivenessInputV0{
		AgentsInFlight: 1,
		Agents: []RunLivenessAgentV0{{
			Status:   "running",
			InFlight: true,
		}},
	})

	if got.Class != RunLivenessClassRunningWithoutRecentStatsV0 ||
		got.ProcessObserved ||
		got.ConfirmedNoLiveProcess ||
		got.UnknownProcess ||
		!got.Ambiguous ||
		got.SafeToReconcile ||
		!got.Running {
		t.Fatalf("classification=%+v", got)
	}
}

func TestClassifyRunLivenessV0UsaProgresoRecienteComoLive(t *testing.T) {
	got := ClassifyRunLivenessV0(RunLivenessInputV0{
		AgentsInFlight: 1,
		Agents: []RunLivenessAgentV0{{
			Status:             "running",
			InFlight:           true,
			LastProgressStatus: "working",
		}},
	})

	if got.Class != RunLivenessClassRunningLiveV0 ||
		got.AgentsLive != 1 ||
		!got.Live ||
		got.ProcessObserved {
		t.Fatalf("classification=%+v", got)
	}
}

func TestClassifyRunLivenessV0CuentaProgressingAgents(t *testing.T) {
	got := ClassifyRunLivenessV0(RunLivenessInputV0{
		AgentsInFlight:    2,
		ProgressingAgents: 2,
		Agents: []RunLivenessAgentV0{{
			Status:   "running",
			InFlight: true,
		}},
	})

	if got.Class != RunLivenessClassRunningLiveV0 ||
		got.AgentsLive != 2 {
		t.Fatalf("classification=%+v", got)
	}
}

func TestClassifyRunLivenessV0AckPendienteNoEsSeguroReconciliar(t *testing.T) {
	got := ClassifyRunLivenessV0(RunLivenessInputV0{
		AgentsInFlight:  1,
		PendingAckCount: 1,
		Agents: []RunLivenessAgentV0{{
			Status:        "running",
			InFlight:      true,
			ProcessRef:    "process-ref-stopped-ack-001",
			ProcessStatus: "stopped",
		}},
	})

	if got.Class != RunLivenessClassRunningStaleNoProcessV0 ||
		!got.Stale ||
		got.SafeToReconcile ||
		got.QueueStatusSuggestion != "" ||
		got.RecommendedAction != "wait_for_ack_before_reconcile" {
		t.Fatalf("classification=%+v", got)
	}
}

func TestClassifyRunLivenessV0TrabajoLogicoActivoPrefiereLive(t *testing.T) {
	got := ClassifyRunLivenessV0(RunLivenessInputV0{
		LogicalWorkActive: true,
		AgentsInFlight:    1,
		Agents: []RunLivenessAgentV0{{
			Status:        "running",
			InFlight:      true,
			ProcessRef:    "process-ref-logical-stopped-001",
			ProcessStatus: "stopped",
		}},
	})

	if got.Class != RunLivenessClassRunningLiveV0 ||
		!got.Live ||
		got.Stale ||
		got.SafeToReconcile ||
		got.RecommendedAction != "observe_logical_work" {
		t.Fatalf("classification=%+v", got)
	}
}

func TestClassifyRunLivenessV0MixtoVivoYParadoPrefiereLive(t *testing.T) {
	got := ClassifyRunLivenessV0(RunLivenessInputV0{
		AgentsInFlight: 2,
		Agents: []RunLivenessAgentV0{
			{
				Status:        "running",
				InFlight:      true,
				ProcessRef:    "process-ref-stopped-mixed-001",
				ProcessStatus: "stopped",
			},
			{
				Status:        "running",
				InFlight:      true,
				ProcessRef:    "process-ref-live-mixed-001",
				ProcessStatus: "running",
			},
		},
	})

	if got.Class != RunLivenessClassRunningLiveV0 ||
		got.AgentsLive != 1 ||
		!got.Live ||
		got.Stale ||
		got.SafeToReconcile {
		t.Fatalf("classification=%+v", got)
	}
}
