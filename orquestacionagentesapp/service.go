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

	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/runtimesapp"
)

type AgentPreparer interface {
	BuildPrepare(input agentesapp.PrepareInput) (*agentesapp.PrepareOutput, error)
}

type RuntimeController interface {
	EnqueueAgentControl(req runtimesapp.AgentControlRequest) (int64, string, error)
	ResolveControlHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error)
	ResolveDeliveryHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error)
	ResolveTranscriptDeliveryHandle(item *db.RuntimeTranscriptEntry) (*db.RuntimeHandle, error)
}

type Service struct {
	agents   AgentPreparer
	runtimes RuntimeController
}

func NewService(agents AgentPreparer, runtimes RuntimeController) *Service {
	return &Service{agents: agents, runtimes: runtimes}
}

type ControlRequest struct {
	Agente       string
	Proyecto     string
	Accion       string
	Conector     string
	Modelo       string
	Razonamiento string
	Perfil       string
	Motivo       string
	Por          string
	TareaID      *int64
}

type LaunchRequest struct {
	Agente       string
	Proyecto     string
	Conector     string
	Modelo       string
	Razonamiento string
	Perfil       string
	Motivo       string
	Por          string
}

type LaunchResult struct {
	OrderID  int64
	Agente   string
	Proyecto agentesapp.ProjectBundle
	Conector agentesapp.ConnectorBundle
}

func (s *Service) EnqueueControl(req ControlRequest) (int64, string, error) {
	if s == nil || s.runtimes == nil {
		return 0, "", fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	return s.runtimes.EnqueueAgentControl(runtimesapp.AgentControlRequest{
		Agente:       strings.TrimSpace(req.Agente),
		Proyecto:     strings.TrimSpace(req.Proyecto),
		Accion:       strings.TrimSpace(req.Accion),
		Conector:     strings.TrimSpace(req.Conector),
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		Perfil:       strings.TrimSpace(req.Perfil),
		Motivo:       strings.TrimSpace(req.Motivo),
		Por:          strings.TrimSpace(req.Por),
		TareaID:      req.TareaID,
	})
}

func (s *Service) Launch(req LaunchRequest) (*LaunchResult, error) {
	if s == nil || s.agents == nil || s.runtimes == nil {
		return nil, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	req.Agente = strings.TrimSpace(req.Agente)
	req.Proyecto = strings.TrimSpace(req.Proyecto)
	if req.Agente == "" || req.Proyecto == "" {
		return nil, fmt.Errorf("agente y proyecto son obligatorios")
	}
	prep, err := s.agents.BuildPrepare(agentesapp.PrepareInput{
		Agente:       req.Agente,
		Proyecto:     req.Proyecto,
		Conector:     strings.TrimSpace(req.Conector),
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		Perfil:       strings.TrimSpace(req.Perfil),
	})
	if err != nil {
		return nil, err
	}
	por := strings.TrimSpace(req.Por)
	if por == "" {
		por = "orquesta"
	}
	orderID, _, err := s.EnqueueControl(ControlRequest{
		Agente:       req.Agente,
		Proyecto:     req.Proyecto,
		Accion:       "start",
		Conector:     prep.Conector.Slug,
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		Perfil:       strings.TrimSpace(req.Perfil),
		Motivo:       strings.TrimSpace(req.Motivo),
		Por:          por,
	})
	if err != nil {
		return nil, err
	}
	return &LaunchResult{
		OrderID:  orderID,
		Agente:   prep.Agente,
		Proyecto: prep.Proyecto,
		Conector: prep.Conector,
	}, nil
}

func (s *Service) ResolveControlHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	if s == nil || s.runtimes == nil {
		return nil, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	return s.runtimes.ResolveControlHandle(strings.TrimSpace(agente), proyectoID)
}

func (s *Service) ResolveDeliveryHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	if s == nil || s.runtimes == nil {
		return nil, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	return s.runtimes.ResolveDeliveryHandle(strings.TrimSpace(agente), proyectoID)
}

func (s *Service) ResolveTranscriptDeliveryHandle(item *db.RuntimeTranscriptEntry) (*db.RuntimeHandle, error) {
	if s == nil || s.runtimes == nil {
		return nil, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	return s.runtimes.ResolveTranscriptDeliveryHandle(item)
}
