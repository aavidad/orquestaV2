package db

import (
	"database/sql"
	"testing"
)

func TestGuardarYListarGitMerges(t *testing.T) {
	prepararDBTemporal(t)
	var proyectoID int64
	if err := DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}

	id, err := GuardarGitMerge(&GitMerge{
		ProyectoID:   proyectoID,
		SourceBranch: "feature/mcp",
		TargetBranch: "main",
		RequestedBy:  "codex2",
		Estado:       "pendiente",
		CommitOrigen: "abc123",
		Notas:        "Merge coordinado tras validacion",
		MetadataJSON: `{"origen":"worktree-codex2"}`,
	})
	if err != nil {
		t.Fatalf("GuardarGitMerge: %v", err)
	}
	if id == 0 {
		t.Fatalf("id invalido")
	}

	items, err := ListarGitMerges(&proyectoID, "pendiente")
	if err != nil {
		t.Fatalf("ListarGitMerges: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("esperaba 1 merge, tengo %d", len(items))
	}
	if items[0].SourceBranch != "feature/mcp" || items[0].TargetBranch != "main" {
		t.Fatalf("ramas inesperadas: %+v", items[0])
	}
}

func TestGuardarGitMergeActualizaEstadoExistente(t *testing.T) {
	prepararDBTemporal(t)
	var proyectoID int64
	if err := DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}

	id, err := GuardarGitMerge(&GitMerge{
		ProyectoID:   proyectoID,
		SourceBranch: "feature/gitgov",
		TargetBranch: "main",
		RequestedBy:  "codex2",
	})
	if err != nil {
		t.Fatalf("GuardarGitMerge insert: %v", err)
	}

	if _, err := GuardarGitMerge(&GitMerge{
		ID:           id,
		ProyectoID:   proyectoID,
		SourceBranch: "feature/gitgov",
		TargetBranch: "main",
		RequestedBy:  "codex2",
		Estado:       "fusionado",
		CommitOrigen: "def456",
		CommitMerge:  "fedcba",
		Notas:        "Integrado en ventana controlada",
	}); err != nil {
		t.Fatalf("GuardarGitMerge update: %v", err)
	}

	item, err := GetGitMerge(id)
	if err != nil {
		t.Fatalf("GetGitMerge: %v", err)
	}
	if item.Estado != "fusionado" || item.CommitMerge != "fedcba" {
		t.Fatalf("merge actualizado inesperado: %+v", item)
	}
}

func TestGuardarGitMergeValidaCampos(t *testing.T) {
	prepararDBTemporal(t)
	if _, err := GuardarGitMerge(&GitMerge{ProyectoID: 1, RequestedBy: "codex2"}); err == nil {
		t.Fatalf("esperaba error por ramas vacias")
	}
	if _, err := GuardarGitMerge(&GitMerge{
		ProyectoID:   1,
		SourceBranch: "a",
		TargetBranch: "b",
		RequestedBy:  "codex2",
		Estado:       "inventado",
	}); err == nil {
		t.Fatalf("esperaba error por estado invalido")
	}
	if _, err := GuardarGitMerge(&GitMerge{
		ID:           999,
		ProyectoID:   1,
		SourceBranch: "a",
		TargetBranch: "b",
		RequestedBy:  "codex2",
	}); err != sql.ErrNoRows {
		t.Fatalf("esperaba sql.ErrNoRows, tengo %v", err)
	}
}
