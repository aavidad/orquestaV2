package db

import (
	"strings"
	"testing"
	"time"
)

func TestFormatearAnotacionTarea(t *testing.T) {
	t.Parallel()

	got := formatearAnotacionTarea("codex1", "revisar", time.Date(2026, time.March, 22, 15, 4, 5, 0, time.UTC))
	want := "\nrevisar [2026-03-22 15:04:05 codex1]"
	if got != want {
		t.Fatalf("anotacion inesperada: got %q want %q", got, want)
	}
}

func TestAnotarTareaPortatil(t *testing.T) {
	withTempDBPools(t, func() {
		id, err := CrearTarea(&Tarea{
			Titulo:      "tarea de prueba",
			Descripcion: "",
			Modulo:      "db",
			Prioridad:   PrioridadMedia,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("CrearTarea: %v", err)
		}

		if err := AnotarTarea(id, "codex1", "revisar"); err != nil {
			t.Fatalf("AnotarTarea: %v", err)
		}

		var notas string
		if err := DB.QueryRow(`SELECT notas FROM tareas WHERE id = ?`, id).Scan(&notas); err != nil {
			t.Fatalf("query notas: %v", err)
		}
		if !strings.HasPrefix(notas, "\nrevisar [") {
			t.Fatalf("nota sin prefijo esperado: %q", notas)
		}
		if !strings.HasSuffix(notas, " codex1]") {
			t.Fatalf("nota sin sufijo esperado: %q", notas)
		}
	})
}

func TestListarTareasRespetaLimit(t *testing.T) {
	withTempDBPools(t, func() {
		for i := 0; i < 3; i++ {
			if _, err := CrearTarea(&Tarea{
				Titulo:    "tarea",
				Modulo:    "db",
				Prioridad: PrioridadMedia,
				CreadoPor: "alberto",
			}); err != nil {
				t.Fatalf("CrearTarea %d: %v", i, err)
			}
		}

		items, err := ListarTareas(FiltroTareas{Limit: 2})
		if err != nil {
			t.Fatalf("ListarTareas limit: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("len(items)=%d want 2", len(items))
		}
	})
}

func TestCrearTareaBlueprintKeyEsIdempotenteSinONCONFLICTDeMotor(t *testing.T) {
	withTempDBPools(t, func() {
		proyectoID, err := UpsertProyecto(&Proyecto{
			Slug:    "orquesta",
			Nombre:  "Orquesta",
			RutaAbs: t.TempDir(),
			Tipo:    ProyectoRepo,
			Activo:  true,
		})
		if err != nil {
			t.Fatalf("UpsertProyecto: %v", err)
		}

		id, err := CrearTarea(&Tarea{
			Titulo:       "portabilidad postgres",
			Descripcion:  "slice acotado",
			ProyectoID:   &proyectoID,
			Modulo:       "db",
			Prioridad:    PrioridadAlta,
			CreadoPor:    "orquesta",
			BlueprintKey: "postgres-lastinsertid-slice",
		})
		if err != nil {
			t.Fatalf("CrearTarea primera: %v", err)
		}
		if id <= 0 {
			t.Fatalf("id inesperado: %d", id)
		}

		duplicada, err := CrearTarea(&Tarea{
			Titulo:       "portabilidad postgres",
			Descripcion:  "slice acotado",
			ProyectoID:   &proyectoID,
			Modulo:       "db",
			Prioridad:    PrioridadAlta,
			CreadoPor:    "orquesta",
			BlueprintKey: "postgres-lastinsertid-slice",
		})
		if err != nil {
			t.Fatalf("CrearTarea duplicada: %v", err)
		}
		if duplicada != 0 {
			t.Fatalf("duplicada=%d want 0", duplicada)
		}
	})
}
