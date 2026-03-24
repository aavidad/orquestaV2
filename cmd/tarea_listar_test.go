package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestTareaListarTSVParaScripts(t *testing.T) {
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
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Revisar conector hexagonal",
		Descripcion: "Sin acceso directo a SQLite",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	for _, item := range []struct {
		name  string
		value string
	}{
		{name: "agente", value: "Codex1"},
		{name: "estado", value: string(db.EstadoAsignada)},
		{name: "tsv", value: "true"},
		{name: "json", value: "false"},
	} {
		if err := tareaListarCmd.Flags().Set(item.name, item.value); err != nil {
			t.Fatalf("set flag %s: %v", item.name, err)
		}
	}
	t.Cleanup(func() {
		_ = tareaListarCmd.Flags().Set("agente", "")
		_ = tareaListarCmd.Flags().Set("estado", "")
		_ = tareaListarCmd.Flags().Set("tsv", "false")
		_ = tareaListarCmd.Flags().Set("json", "false")
	})

	out := capturarStdout(t, func() {
		if err := tareaListarCmd.RunE(tareaListarCmd, nil); err != nil {
			t.Fatalf("run listar tsv: %v", err)
		}
	})
	want := "Revisar conector hexagonal"
	if !strings.Contains(out, "\t"+want) {
		t.Fatalf("salida TSV inesperada:\n%s", out)
	}
	if !strings.Contains(out, "\tasignada\talta\tCodex1\torquestador\tcontrolplane\t") {
		t.Fatalf("salida TSV sin columnas esperadas:\n%s", out)
	}
}
