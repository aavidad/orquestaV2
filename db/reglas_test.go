package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuardarYListarReglas(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()
	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	id, err := GuardarRegla(&Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "Puerto de almacenamiento",
		Descripcion: "No acoplar a SQLite",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("GuardarRegla: %v", err)
	}
	if id == 0 {
		t.Fatalf("id invalido")
	}
	items, err := ListarReglas("programador", nil)
	if err != nil {
		t.Fatalf("ListarReglas: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("sin reglas")
	}
}

func TestGuardarYListarSkills(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()
	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	id, err := GuardarSkill(&Skill{
		TipoAgente:  "programador",
		Nombre:      "consultar-schema",
		Descripcion: "Revisar esquema",
		CuandoUsar:  "Antes de migrar",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("GuardarSkill: %v", err)
	}
	if id == 0 {
		t.Fatalf("id invalido")
	}
	items, err := ListarSkills("programador", nil)
	if err != nil {
		t.Fatalf("ListarSkills: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("sin skills")
	}
}

func TestGuardarYListarWorkflows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()
	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	id, err := GuardarWorkflow(&Workflow{
		TipoAgente:  "programador",
		Nombre:      "workflow-prueba",
		Descripcion: "Flujo de prueba",
		Pasos:       "[\"uno\",\"dos\"]",
		Activo:      true,
	})
	if err != nil {
		t.Fatalf("GuardarWorkflow: %v", err)
	}
	if id == 0 {
		t.Fatalf("id invalido")
	}
	items, err := ListarWorkflows("programador", nil)
	if err != nil {
		t.Fatalf("ListarWorkflows: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("sin workflows")
	}
}
