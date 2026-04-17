package orquestacionagentesapp

import (
	"testing"

	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/runtimesapp"
)

type stubAgentPreparer struct {
	input agentesapp.PrepareInput
	out   *agentesapp.PrepareOutput
	err   error
}

func (s *stubAgentPreparer) BuildPrepare(input agentesapp.PrepareInput) (*agentesapp.PrepareOutput, error) {
	s.input = input
	return s.out, s.err
}

type stubRuntimeController struct {
	controlReq          runtimesapp.AgentControlRequest
	controlID           int64
	controlAction       string
	controlErr          error
	controlHandle       *db.RuntimeHandle
	controlHandleErr    error
	deliveryHandle      *db.RuntimeHandle
	deliveryHandleErr   error
	transcriptHandle    *db.RuntimeHandle
	transcriptHandleErr error
}

func (s *stubRuntimeController) EnqueueAgentControl(req runtimesapp.AgentControlRequest) (int64, string, error) {
	s.controlReq = req
	return s.controlID, s.controlAction, s.controlErr
}

func (s *stubRuntimeController) ResolveControlHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return s.controlHandle, s.controlHandleErr
}

func (s *stubRuntimeController) ResolveDeliveryHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return s.deliveryHandle, s.deliveryHandleErr
}

func (s *stubRuntimeController) ResolveTranscriptDeliveryHandle(item *db.RuntimeTranscriptEntry) (*db.RuntimeHandle, error) {
	return s.transcriptHandle, s.transcriptHandleErr
}

func TestLaunchBuildsPrepareAndEnqueuesStart(t *testing.T) {
	t.Parallel()

	agents := &stubAgentPreparer{
		out: &agentesapp.PrepareOutput{
			Agente:   "Codex1",
			Proyecto: agentesapp.ProjectBundle{ID: 7, Slug: "orquestador"},
			Conector: agentesapp.ConnectorBundle{Slug: "codex-cli"},
		},
	}
	runtimes := &stubRuntimeController{controlID: 33, controlAction: "start"}
	service := NewService(agents, runtimes)

	result, err := service.Launch(LaunchRequest{
		Agente:       " Codex1 ",
		Proyecto:     " orquestador ",
		Conector:     " codex-cli ",
		Modelo:       " gpt-5.4 ",
		Razonamiento: " high ",
		Perfil:       " implementacion ",
		Motivo:       " cerrar core ",
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if result.OrderID != 33 || result.Agente != "Codex1" {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if agents.input.Agente != "Codex1" || agents.input.Proyecto != "orquestador" {
		t.Fatalf("prepare no normalizado: %+v", agents.input)
	}
	if runtimes.controlReq.Accion != "start" || runtimes.controlReq.Conector != "codex-cli" {
		t.Fatalf("control inesperado: %+v", runtimes.controlReq)
	}
	if runtimes.controlReq.Por != "orquesta" {
		t.Fatalf("actor por defecto inesperado: %+v", runtimes.controlReq)
	}
}

func TestEnqueueControlNormalizesPayload(t *testing.T) {
	t.Parallel()

	runtimes := &stubRuntimeController{controlID: 7, controlAction: "pause"}
	service := NewService(nil, runtimes)

	orderID, accion, err := service.EnqueueControl(ControlRequest{
		Agente:       " Codex2 ",
		Proyecto:     " core ",
		Accion:       " pause ",
		Conector:     " codex ",
		Modelo:       " gpt-5.4 ",
		Razonamiento: " medium ",
		Perfil:       " review ",
		Motivo:       " mantenimiento ",
		Por:          " admin ",
	})
	if err != nil {
		t.Fatalf("EnqueueControl: %v", err)
	}
	if orderID != 7 || accion != "pause" {
		t.Fatalf("respuesta inesperada: %d %s", orderID, accion)
	}
	if runtimes.controlReq.Agente != "Codex2" || runtimes.controlReq.Proyecto != "core" {
		t.Fatalf("payload no normalizado: %+v", runtimes.controlReq)
	}
}
