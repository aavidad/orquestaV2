package cmd

import (
	"testing"

	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/db"
)

type fakeRowsProvider struct {
	rows []agentesapp.Row
	err  error
}

func (f fakeRowsProvider) BuildPanelRows() ([]agentesapp.Row, error) {
	return f.rows, f.err
}

func TestResolvedorAgentePipelineOperativoPrefierePremiumDisponible(t *testing.T) {
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: fakeRowsProvider{rows: []agentesapp.Row{
			{Agente: &db.Agente{Nombre: "Gemma1", Habilitado: true}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "Codex1", Habilitado: true}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "Claude1", Habilitado: true}, EstadoOperativo: "trabajando"},
		}},
	}

	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril: "premium_worktree",
	})
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "Codex1" {
		t.Fatalf("agente premium inesperado: %q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoPrefiereGeminiParaEspecificacion(t *testing.T) {
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: fakeRowsProvider{rows: []agentesapp.Row{
			{Agente: &db.Agente{Nombre: "Codex1", Habilitado: true}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "Gemini1", Habilitado: true}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "Claude1", Habilitado: true}, EstadoOperativo: "disponible"},
		}},
	}

	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril: "premium_worktree",
		Fase:   "especificacion",
	})
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "Gemini1" {
		t.Fatalf("agente especificacion inesperado: %q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoPrefiereClaudeParaRevision(t *testing.T) {
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: fakeRowsProvider{rows: []agentesapp.Row{
			{Agente: &db.Agente{Nombre: "Codex1", Habilitado: true}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "Gemini1", Habilitado: true}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "Claude1", Habilitado: true}, EstadoOperativo: "disponible"},
		}},
	}

	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril: "revision_diff",
		Fase:   "revision",
	})
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "Claude1" {
		t.Fatalf("agente revision inesperado: %q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoPrefiereLocalParaMicroprogramacion(t *testing.T) {
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: fakeRowsProvider{rows: []agentesapp.Row{
			{Agente: &db.Agente{Nombre: "Codex1", Habilitado: true}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "Gemma1", Habilitado: true}, EstadoOperativo: "disponible"},
		}},
	}

	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril: "microprogramacion_local",
	})
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "Gemma1" {
		t.Fatalf("agente local inesperado: %q", agente)
	}
}
