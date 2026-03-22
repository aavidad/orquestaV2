/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func prepararDBTemporalTareasDependencias(t *testing.T) string {
	t.Helper()

	anteriorDB := os.Getenv("ORQUESTA_DB")
	anteriorRoot := os.Getenv("ORQUESTA_WORKSPACE_ROOT")
	t.Cleanup(func() {
		Close()
		DB = nil
		if anteriorDB == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", anteriorDB)
		}
		if anteriorRoot == "" {
			_ = os.Unsetenv("ORQUESTA_WORKSPACE_ROOT")
		} else {
			_ = os.Setenv("ORQUESTA_WORKSPACE_ROOT", anteriorRoot)
		}
	})

	Close()
	DB = nil

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "orquesta-tareas-dependencias-test.db")
	if err := os.Setenv("ORQUESTA_DB", dbPath); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	if err := os.Setenv("ORQUESTA_WORKSPACE_ROOT", tmp); err != nil {
		t.Fatalf("setenv ORQUESTA_WORKSPACE_ROOT: %v", err)
	}
	if err := Open(); err != nil {
		t.Fatalf("open db temporal: %v", err)
	}
	return tmp
}

func crearTareaPrueba(t *testing.T, titulo string, deps []int64) int64 {
	t.Helper()

	id, err := CrearTarea(&Tarea{
		Titulo:       titulo,
		Descripcion:  "tarea de prueba",
		Modulo:       "orquestador",
		Prioridad:    PrioridadAlta,
		CreadoPor:    "Codex1",
		Dependencias: deps,
	})
	if err != nil {
		t.Fatalf("crear tarea %q: %v", titulo, err)
	}
	return id
}

func prepararAgentePrueba(t *testing.T) {
	t.Helper()
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
}

func completarTareaPrueba(t *testing.T, id int64) {
	t.Helper()
	if err := TomarTarea(id, "Codex1"); err != nil {
		t.Fatalf("tomar tarea #%d: %v", id, err)
	}
	if err := IniciarTarea(id, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea #%d: %v", id, err)
	}
	if err := CompletarTarea(id, "Codex1", "feat(orquestador): cierre de prueba"); err != nil {
		t.Fatalf("completar tarea #%d: %v", id, err)
	}
}

func TestTomarTareaExigeDependenciasCompletadasYConContrato(t *testing.T) {
	prepararDBTemporalTareasDependencias(t)
	prepararAgentePrueba(t)

	t.Run("bloquea_dependencia_con_contrato_pero_no_completada", func(t *testing.T) {
		depID := crearTareaPrueba(t, "Dependencia sin terminar", nil)
		if err := DefinirContrato(depID, "Codex1"); err != nil {
			t.Fatalf("definir contrato: %v", err)
		}
		dependienteID := crearTareaPrueba(t, "Tarea dependiente", []int64{depID})

		err := TomarTarea(dependienteID, "Codex1")
		if err == nil {
			t.Fatalf("esperaba error al tomar tarea dependiente")
		}
		if !strings.Contains(err.Error(), "aún no está completada") {
			t.Fatalf("error inesperado: %v", err)
		}
	})

	t.Run("bloquea_dependencia_completada_sin_contrato", func(t *testing.T) {
		depID := crearTareaPrueba(t, "Dependencia completada sin contrato", nil)
		completarTareaPrueba(t, depID)
		dependienteID := crearTareaPrueba(t, "Tarea dependiente sin contrato", []int64{depID})

		err := TomarTarea(dependienteID, "Codex1")
		if err == nil {
			t.Fatalf("esperaba error al tomar tarea dependiente")
		}
		if !strings.Contains(err.Error(), "no tiene contrato/interfaz definido") {
			t.Fatalf("error inesperado: %v", err)
		}
	})

	t.Run("permite_dependencia_completada_y_con_contrato", func(t *testing.T) {
		depID := crearTareaPrueba(t, "Dependencia resuelta", nil)
		if err := DefinirContrato(depID, "Codex1"); err != nil {
			t.Fatalf("definir contrato: %v", err)
		}
		completarTareaPrueba(t, depID)
		dependienteID := crearTareaPrueba(t, "Tarea dependiente valida", []int64{depID})

		if err := TomarTarea(dependienteID, "Codex1"); err != nil {
			t.Fatalf("tomar tarea dependiente: %v", err)
		}
		tarea, err := GetTarea(dependienteID)
		if err != nil {
			t.Fatalf("get tarea dependiente: %v", err)
		}
		if tarea.Estado != TareaAsignada {
			t.Fatalf("estado inesperado: %s", tarea.Estado)
		}
		if err := IniciarTarea(dependienteID, "Codex1"); err != nil {
			t.Fatalf("iniciar tarea dependiente: %v", err)
		}
	})
}
