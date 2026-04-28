package cmd

import (
	"strings"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/db"
)

type fakeRowsProvider struct {
	rows []agentesapp.Row
	err  error
}

type fakeScoreProvider struct {
	ensureCalls   []string
	fitnessByName map[string]float64
	err           error
}

func (f *fakeScoreProvider) GarantizarScoresLocalesAgente(agente, conector string) error {
	f.ensureCalls = append(f.ensureCalls, strings.TrimSpace(agente)+"@"+strings.TrimSpace(conector))
	return f.err
}

func (f *fakeScoreProvider) FitnessAgenteLocal(agente, conector string, pesos map[string]float64) (float64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.fitnessByName[strings.TrimSpace(agente)], nil
}

func (f fakeRowsProvider) BuildPanelRows() ([]agentesapp.Row, error) {
	return f.rows, f.err
}

type blockingRowsProvider struct {
	release <-chan struct{}
}

func (b blockingRowsProvider) BuildPanelRows() ([]agentesapp.Row, error) {
	<-b.release
	return nil, nil
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

func TestResolvedorAgentePipelineOperativoPrefiereCodexNoPrimeAntesQuePg(t *testing.T) {
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: fakeRowsProvider{rows: []agentesapp.Row{
			{Agente: &db.Agente{Nombre: "CodexPg1", Habilitado: true}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "Codex1", Habilitado: true}, EstadoOperativo: "disponible"},
		}},
	}

	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril: "premium_worktree",
	})
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "Codex1" {
		t.Fatalf("deberia preferir codex no-prime antes que pg, got=%q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoUsaPgSiEsElUnicoCodexDisponible(t *testing.T) {
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: fakeRowsProvider{rows: []agentesapp.Row{
			{Agente: &db.Agente{Nombre: "CodexPg1", Habilitado: true}, EstadoOperativo: "disponible"},
		}},
	}

	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril: "premium_worktree",
	})
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "CodexPg1" {
		t.Fatalf("deberia seguir permitiendo pg si es el unico codex disponible, got=%q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoUsaCodexTambienEnEspecificacion(t *testing.T) {
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
	if agente != "Codex1" {
		t.Fatalf("agente especificacion inesperado: %q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoUsaCodexParaRevision(t *testing.T) {
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
	if agente != "Codex1" {
		t.Fatalf("agente revision inesperado: %q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoNoReusaPremiumTrabajando(t *testing.T) {
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: fakeRowsProvider{rows: []agentesapp.Row{
			{Agente: &db.Agente{Nombre: "Claude1", Habilitado: true}, EstadoOperativo: "trabajando"},
			{Agente: &db.Agente{Nombre: "Codex1", Habilitado: true}, EstadoOperativo: "trabajando"},
		}},
	}

	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril: "premium_worktree",
		Fase:   "implementacion",
	})
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "" {
		t.Fatalf("no deberia reutilizar premium trabajando, got=%q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoNoReusaReviewerTrabajando(t *testing.T) {
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: fakeRowsProvider{rows: []agentesapp.Row{
			{Agente: &db.Agente{Nombre: "Claude1", Habilitado: true}, EstadoOperativo: "trabajando"},
		}},
	}

	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril: "revision_diff",
		Fase:   "revision",
	})
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "" {
		t.Fatalf("no deberia reutilizar reviewer trabajando, got=%q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoUsaCodexParaMicroprogramacionLocal(t *testing.T) {
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
	if agente != "Codex1" {
		t.Fatalf("agente local inesperado: %q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoUsaLocalConScoreAltoParaMicroprogramacion(t *testing.T) {
	scoreProvider := &fakeScoreProvider{
		fitnessByName: map[string]float64{
			"Gemma1": 7.4,
		},
	}
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: fakeRowsProvider{rows: []agentesapp.Row{
			{Agente: &db.Agente{Nombre: "Codex1", Habilitado: true}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "Gemma1", Habilitado: true}, EstadoOperativo: "disponible"},
		}},
		scoreProvider: scoreProvider,
	}

	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril: "microprogramacion_local",
	})
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "Gemma1" {
		t.Fatalf("deberia elegir local con fitness alto, got=%q", agente)
	}
	if len(scoreProvider.ensureCalls) == 0 {
		t.Fatalf("deberia garantizar scores locales antes de elegir")
	}
}

func TestResolvedorAgentePipelineOperativoUsaLocalConScoreAltoParaPremium(t *testing.T) {
	scoreProvider := &fakeScoreProvider{
		fitnessByName: map[string]float64{
			"Gemma1": 7.6,
		},
	}
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: fakeRowsProvider{rows: []agentesapp.Row{
			{Agente: &db.Agente{Nombre: "Codex1", Habilitado: true}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "Gemma1", Habilitado: true}, EstadoOperativo: "disponible"},
		}},
		scoreProvider: scoreProvider,
	}

	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril:      "premium_worktree",
		Fase:        "implementacion",
		PerfilTarea: "implementacion",
	})
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "Gemma1" {
		t.Fatalf("deberia elegir local-first con fitness alto en premium, got=%q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoUsaLocalConScoreAltoParaRevision(t *testing.T) {
	scoreProvider := &fakeScoreProvider{
		fitnessByName: map[string]float64{
			"QwenCoder1": 7.2,
		},
	}
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: fakeRowsProvider{rows: []agentesapp.Row{
			{Agente: &db.Agente{Nombre: "Codex1", Habilitado: true}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "QwenCoder1", Habilitado: true}, EstadoOperativo: "disponible"},
		}},
		scoreProvider: scoreProvider,
	}

	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril:      "revision_diff",
		Fase:        "revision",
		PerfilTarea: "revision",
	})
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "QwenCoder1" {
		t.Fatalf("deberia elegir local-first con fitness alto en revision, got=%q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoNoDesplazaCodexSiLocalNoSuperaUmbral(t *testing.T) {
	scoreProvider := &fakeScoreProvider{
		fitnessByName: map[string]float64{
			"Gemma1": 5.9,
		},
	}
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: fakeRowsProvider{rows: []agentesapp.Row{
			{Agente: &db.Agente{Nombre: "Codex1", Habilitado: true}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "Gemma1", Habilitado: true}, EstadoOperativo: "disponible"},
		}},
		scoreProvider: scoreProvider,
	}

	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril: "microprogramacion_local",
	})
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "Codex1" {
		t.Fatalf("no deberia desplazar a Codex sin fitness suficiente, got=%q", agente)
	}
}

func TestResolvedorAgentePipelineOperativoOmiteTimeoutDeRows(t *testing.T) {
	prevTimeout := capacidadAgentResolverRowsTimeout
	defer func() { capacidadAgentResolverRowsTimeout = prevTimeout }()
	capacidadAgentResolverRowsTimeout = 20 * time.Millisecond

	release := make(chan struct{})
	resolvedor := resolvedorAgentePipelineOperativo{
		rowsProvider: blockingRowsProvider{release: release},
	}

	start := time.Now()
	agente, err := resolvedor.ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline{
		Carril: "premium_worktree",
	})
	close(release)
	if err != nil {
		t.Fatalf("ResolverAgentePipeline: %v", err)
	}
	if agente != "" {
		t.Fatalf("deberia degradar limpio si rows timeouta, got=%q", agente)
	}
	if elapsed := time.Since(start); elapsed > 150*time.Millisecond {
		t.Fatalf("deberia degradar rapido, elapsed=%s", elapsed)
	}
}
