/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package orquestacionagentesapp

import (
	"fmt"
	"strings"
	"time"
)

type TestRuntimeOps interface {
	HasLiveRuntimeActivity(agente, proyecto string, limit int) (bool, error)
	WakeRuntimeBatches(orders, mailbox, warm bool) error
}

type testRuntimeControlOps interface {
	EnqueueStopControl(req ControlRequest) error
}

type TestRuntimeCleanupService struct {
	orchestrator *Service
	ops          TestRuntimeOps
}

func NewTestRuntimeCleanupService(orchestrator *Service, ops TestRuntimeOps) *TestRuntimeCleanupService {
	return &TestRuntimeCleanupService{orchestrator: orchestrator, ops: ops}
}

type StopAgentRuntimeIfActiveInput struct {
	Agente       string
	Proyecto     string
	Actor        string
	Reason       string
	Timeout      time.Duration
	PollInterval time.Duration
}

type StopAgentRuntimeIfActiveResult struct {
	HadLiveActivity bool
	StopEnqueued    bool
	Stopped         bool
}

func (s *TestRuntimeCleanupService) StopAgentRuntimeIfActive(input StopAgentRuntimeIfActiveInput) (*StopAgentRuntimeIfActiveResult, error) {
	if s == nil || s.ops == nil {
		return nil, fmt.Errorf("servicio de limpieza de runtime no inicializado")
	}
	if s.orchestrator == nil {
		if _, ok := s.ops.(testRuntimeControlOps); !ok {
			return nil, fmt.Errorf("servicio de limpieza de runtime no inicializado")
		}
	}
	agente := strings.TrimSpace(input.Agente)
	if agente == "" {
		return &StopAgentRuntimeIfActiveResult{}, nil
	}
	proyecto := strings.TrimSpace(input.Proyecto)
	live, err := s.ops.HasLiveRuntimeActivity(agente, proyecto, 3)
	if err != nil {
		return nil, err
	}
	if !live {
		return &StopAgentRuntimeIfActiveResult{}, nil
	}
	actor := strings.TrimSpace(input.Actor)
	if actor == "" {
		actor = "orquesta"
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "runtime limpiar-pruebas"
	}
	control := ControlRequest{
		Agente: agente,
		Accion: "stop",
		Motivo: reason,
		Por:    actor,
	}
	if runtimeOps, ok := s.ops.(testRuntimeControlOps); ok {
		if err := runtimeOps.EnqueueStopControl(control); err != nil {
			return nil, err
		}
	} else if _, _, err := s.orchestrator.EnqueueControl(control); err != nil {
		return nil, err
	}
	if err := s.ops.WakeRuntimeBatches(true, false, true); err != nil {
		return nil, err
	}
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	poll := input.PollInterval
	if poll <= 0 {
		poll = 250 * time.Millisecond
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(poll)
		live, err := s.ops.HasLiveRuntimeActivity(agente, proyecto, 3)
		if err != nil {
			return nil, err
		}
		if !live {
			return &StopAgentRuntimeIfActiveResult{
				HadLiveActivity: true,
				StopEnqueued:    true,
				Stopped:         true,
			}, nil
		}
	}
	return nil, fmt.Errorf("timeout esperando parada de runtime de prueba para %s", agente)
}
