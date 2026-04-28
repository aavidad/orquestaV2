package cmd

import (
	"testing"

	"orquesta/db"
)

func TestCanonicalAutonomyCodexName(t *testing.T) {
	cases := map[string]string{
		"":             "",
		"codex2":       "Codex2",
		" Codex3 ":     "Codex3",
		"CODEX4":       "Codex4",
		"codexBudget":  "CodexBudget",
		"gemma1":       "gemma1",
		"Claude1":      "Claude1",
		"  codexTeam ": "CodexTeam",
	}
	for in, want := range cases {
		if got := canonicalAutonomyCodexName(in); got != want {
			t.Fatalf("canonicalAutonomyCodexName(%q)=%q want %q", in, got, want)
		}
	}
}

func TestResolverAgenteAutonomiaOperativoCanonicalizaCodexCase(t *testing.T) {
	prepararDBTemporalCmd(t)
	for _, nombre := range []string{"Codex2", "codex2"} {
		if err := db.RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", nombre, err)
		}
	}
	agente, err := resolverAgenteAutonomiaOperativo("codex2")
	if err != nil {
		t.Fatalf("resolverAgenteAutonomiaOperativo: %v", err)
	}
	if agente == nil || agente.Nombre != "Codex2" {
		t.Fatalf("agente resuelto inesperado: %+v", agente)
	}
}

func TestResolveRepoPersistentAutonomyRequestCanonicalizaCodexCase(t *testing.T) {
	prepararDBTemporalCmd(t)
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil || proyecto == nil {
		t.Fatalf("get proyecto: %v proyecto=%+v", err, proyecto)
	}
	for _, nombre := range []string{"Codex2", "Codex3", "Codex4", "codex2", "codex3"} {
		if err := db.RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", nombre, err)
		}
	}
	req := normalizeRepoMejorarRequest(apiRepoMejorarRequest{
		Proyecto:             "orquestador",
		Titulo:               "Cerrar app",
		Descripcion:          "Cerrar autonomamente hasta terminar",
		AutonomiaPersistente: true,
		SupervisorAgente:     "codex2",
		ReviewerAgente:       "codex3",
		MaxWorkers:           2,
	})
	got, err := resolveRepoPersistentAutonomyRequest(proyecto, req)
	if err != nil {
		t.Fatalf("resolveRepoPersistentAutonomyRequest: %v", err)
	}
	if got.SupervisorAgente != "Codex2" {
		t.Fatalf("supervisor inesperado: %+v", got)
	}
	if got.ReviewerAgente != "Codex3" {
		t.Fatalf("reviewer inesperado: %+v", got)
	}
	if asignacion, err := db.GetAsignacionActivaAgente("Codex2"); err != nil || asignacion == nil {
		t.Fatalf("asignacion Codex2 inesperada: err=%v asignacion=%+v", err, asignacion)
	}
	var totalLower int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM asignaciones WHERE agente = 'codex2' AND estado = 'activa'`).Scan(&totalLower); err != nil {
		t.Fatalf("count lower asignaciones: %v", err)
	}
	if totalLower != 0 {
		t.Fatalf("codex2 no debia activarse como alias lowercase: total=%d", totalLower)
	}
}
