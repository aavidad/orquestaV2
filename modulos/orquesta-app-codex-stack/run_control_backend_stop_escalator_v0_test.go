package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

type stackEscalatorActiveWorkReaderForTestV0 struct {
	results []orquestaservershutdown.ActiveShutdownWorkResultV0
	calls   int
}

func (reader *stackEscalatorActiveWorkReaderForTestV0) ReadActiveShutdownWorkV0(
	_ context.Context,
	_ orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	index := reader.calls
	reader.calls++
	if index >= len(reader.results) {
		index = len(reader.results) - 1
	}
	if index < 0 {
		return orquestaservershutdown.ActiveShutdownWorkResultV0{}, nil
	}
	return reader.results[index], nil
}

type stackEscalatorActiveWorkCleanerForTestV0 struct {
	commands []orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0
}

func (cleaner *stackEscalatorActiveWorkCleanerForTestV0) CleanupActiveShutdownWorkV0(
	_ context.Context,
	command orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	cleaner.commands = append(cleaner.commands, command)
	return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{
		CleanedWorkCount: len(command.ActiveWorks),
		EvidenceRefs:     []string{"evidence-ref-escalator-cleanup-executed"},
	}, nil
}

func TestStackRunControlBackendStopEscalatorLimpiaSoloElRunEscaladoV0(t *testing.T) {
	targetWork := orquestaservershutdown.ActiveShutdownWorkV0{
		Kind:            "codex_goal",
		RunRef:          "run-ref-escalator-target-001",
		WorkRef:         "goal-ref-escalator-target-001",
		ExternalWorkRef: "thread-ref-escalator-target-001",
		Status:          "running",
	}
	otherWork := orquestaservershutdown.ActiveShutdownWorkV0{
		Kind:   "codex_goal",
		RunRef: "run-ref-escalator-other-001",
		Status: "running",
	}
	reader := &stackEscalatorActiveWorkReaderForTestV0{
		results: []orquestaservershutdown.ActiveShutdownWorkResultV0{
			{ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{targetWork, otherWork}},
			{ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{otherWork}},
		},
	}
	cleaner := &stackEscalatorActiveWorkCleanerForTestV0{}

	result, err := stackRunControlBackendStopEscalatorV0{Reader: reader, Cleaner: cleaner}.EscalateBackendStopV0(
		context.Background(),
		orquestamcp.MCPRunControlBackendStopEscalationRequestV0{
			RunRef:          "run-ref-escalator-target-001",
			GoalRef:         "goal-ref-escalator-target-001",
			ExternalGoalRef: "thread-ref-escalator-target-001",
			Action:          "stop",
		},
	)
	if err != nil {
		t.Fatalf("EscalateBackendStopV0: %v", err)
	}
	if !result.Stopped || len(result.ResidualRefs) != 0 {
		t.Fatalf("escalada no confirmada: %+v", result)
	}
	if len(cleaner.commands) != 1 ||
		len(cleaner.commands[0].ActiveWorks) != 1 ||
		cleaner.commands[0].ActiveWorks[0].RunRef != "run-ref-escalator-target-001" ||
		!cleaner.commands[0].CleanupGoalBackends {
		t.Fatalf("cleanup no filtrado por identidad: %+v", cleaner.commands)
	}
	if !stringInSetV0(result.EvidenceRefs, stackRunControlEscalatorEvidenceCleanedV0) {
		t.Fatalf("sin evidencia de limpieza: %+v", result.EvidenceRefs)
	}
}

func TestStackRunControlBackendStopEscalatorResidualNoConfirmaV0(t *testing.T) {
	targetWork := orquestaservershutdown.ActiveShutdownWorkV0{
		Kind:    "codex_goal",
		RunRef:  "run-ref-escalator-residual-001",
		WorkRef: "goal-ref-escalator-residual-001",
		Status:  "running",
	}
	reader := &stackEscalatorActiveWorkReaderForTestV0{
		results: []orquestaservershutdown.ActiveShutdownWorkResultV0{
			{ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{targetWork}},
			{ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{targetWork}},
		},
	}
	cleaner := &stackEscalatorActiveWorkCleanerForTestV0{}

	result, err := stackRunControlBackendStopEscalatorV0{Reader: reader, Cleaner: cleaner}.EscalateBackendStopV0(
		context.Background(),
		orquestamcp.MCPRunControlBackendStopEscalationRequestV0{
			RunRef: "run-ref-escalator-residual-001",
			Action: "stop",
		},
	)
	if err != nil {
		t.Fatalf("EscalateBackendStopV0: %v", err)
	}
	if result.Stopped ||
		len(result.ResidualRefs) != 1 ||
		result.ResidualRefs[0] != "goal-ref-escalator-residual-001" {
		t.Fatalf("residual mal reportado: %+v", result)
	}
}

func TestStackRunControlBackendStopEscalatorSinTrabajoActivoConfirmaV0(t *testing.T) {
	reader := &stackEscalatorActiveWorkReaderForTestV0{
		results: []orquestaservershutdown.ActiveShutdownWorkResultV0{{}},
	}
	cleaner := &stackEscalatorActiveWorkCleanerForTestV0{}

	result, err := stackRunControlBackendStopEscalatorV0{Reader: reader, Cleaner: cleaner}.EscalateBackendStopV0(
		context.Background(),
		orquestamcp.MCPRunControlBackendStopEscalationRequestV0{
			RunRef: "run-ref-escalator-empty-001",
			Action: "stop",
		},
	)
	if err != nil {
		t.Fatalf("EscalateBackendStopV0: %v", err)
	}
	if !result.Stopped ||
		len(cleaner.commands) != 0 ||
		!stringInSetV0(result.EvidenceRefs, stackRunControlEscalatorEvidenceNoActiveWorkV0) {
		t.Fatalf("sin trabajo activo mal manejado: %+v commands=%+v", result, cleaner.commands)
	}
}

func TestBuildStackV0CableaBackendStopEscalatorEnRunControlV0(t *testing.T) {
	config := ConfigV0{}
	stack := &StackV0{}
	bindings := buildStackMCPTransportBindingsV0(
		config,
		buildDirectorPortsV0(config),
		normalizeRunQueueConfigV0(config.RunQueue),
		stack,
	)
	executor, ok := bindings.RunControl.(orquestamcp.MCPRunControlToolExecutorV0)
	if !ok {
		t.Fatalf("run control no es el executor MCP esperado: %T", bindings.RunControl)
	}
	if executor.BackendStopEscalator == nil {
		t.Fatalf("run control sin escalador de parada del backend: %+v", executor)
	}
}
