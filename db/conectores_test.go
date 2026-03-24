/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "testing"

func TestUpsertConectorYListarConectores(t *testing.T) {
	abrirDBTemporalMemoria(t)

	id, err := UpsertConector(&Conector{
		Slug:         "codex-cli-test",
		Nombre:       "Codex CLI Test",
		Transporte:   "cli",
		Comando:      "codex",
		ArgsJSON:     `["--sandbox","workspace-write"]`,
		EnvJSON:      `{"OPENAI_API_KEY":"$OPENAI_API_KEY"}`,
		MetadataJSON: `{"familia":"openai"}`,
		Activo:       true,
	})
	if err != nil {
		t.Fatalf("UpsertConector: %v", err)
	}
	if id == 0 {
		t.Fatalf("id de conector inesperado")
	}

	conector, err := GetConector("codex-cli-test")
	if err != nil {
		t.Fatalf("GetConector: %v", err)
	}
	if conector.Nombre != "Codex CLI Test" || conector.Transporte != "cli" {
		t.Fatalf("conector inesperado: %+v", conector)
	}

	id2, err := UpsertConector(&Conector{
		Slug:         "claude-cli-test",
		Nombre:       "Claude CLI Test",
		Transporte:   "cli",
		Comando:      "claude",
		MetadataJSON: `{"familia":"anthropic"}`,
		Activo:       true,
	})
	if err != nil {
		t.Fatalf("UpsertConector segundo: %v", err)
	}
	if id2 == 0 || id2 == id {
		t.Fatalf("id de segundo conector inesperado: %d", id2)
	}

	items, err := ListarConectores()
	if err != nil {
		t.Fatalf("ListarConectores: %v", err)
	}
	if len(items) < 2 {
		t.Fatalf("conectores inesperados: %+v", items)
	}
	if items[0].Slug == "" || items[1].Slug == "" {
		t.Fatalf("slugs inesperados: %+v", items)
	}
}
