package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestTareaListarTSVEsEstableParaScripts(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	taskID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Refactor\tPTY\nconnector",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
		Descripcion: "salida estable para scripts",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(taskID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	if err := tareaListarCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := tareaListarCmd.Flags().Set("estado", "asignada"); err != nil {
		t.Fatalf("set estado: %v", err)
	}
	if err := tareaListarCmd.Flags().Set("tsv", "true"); err != nil {
		t.Fatalf("set tsv: %v", err)
	}
	t.Cleanup(func() {
		_ = tareaListarCmd.Flags().Set("agente", "")
		_ = tareaListarCmd.Flags().Set("estado", "")
		_ = tareaListarCmd.Flags().Set("tsv", "false")
	})

	out := capturarStdout(t, func() {
		if err := tareaListarCmd.RunE(tareaListarCmd, nil); err != nil {
			t.Fatalf("run tarea listar --tsv: %v", err)
		}
	})

	line := strings.TrimSpace(out)
	fields := strings.Split(line, "\t")
	if len(fields) != 7 {
		t.Fatalf("esperaba 7 columnas TSV, got=%d line=%q", len(fields), line)
	}
	if fields[0] != itoa(taskID) || fields[1] != "asignada" || fields[2] != string(db.PrioridadAlta) {
		t.Fatalf("cabecera TSV inesperada: %v", fields[:3])
	}
	if fields[3] != "Codex1" || fields[4] != "orquestador" || fields[5] != "controlplane" {
		t.Fatalf("metadatos TSV inesperados: %v", fields[3:6])
	}
	if strings.Contains(fields[6], "\n") || strings.Contains(fields[6], "\t") {
		t.Fatalf("el titulo TSV no deberia contener tabs ni saltos de linea: %q", fields[6])
	}
	if fields[6] != "Refactor PTY connector" {
		t.Fatalf("titulo TSV inesperado: %q", fields[6])
	}
}
