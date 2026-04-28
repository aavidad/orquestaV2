package db

import (
	"testing"
)

func TestActivarAsignacionCanonicalizaNombreAgente(t *testing.T) {
	prepararDBTemporal(t)
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo",
		Nombre:  "Demo",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, nombre := range []string{"Codex2", "codex2"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", nombre, err)
		}
	}
	if err := ActivarAsignacion("codex2", proyectoID, "test canonico"); err != nil {
		t.Fatalf("ActivarAsignacion: %v", err)
	}
	asignacion, err := GetAsignacionActivaAgente("Codex2")
	if err != nil {
		t.Fatalf("GetAsignacionActivaAgente canonico: %v", err)
	}
	if asignacion == nil || asignacion.Agente != "Codex2" {
		t.Fatalf("asignacion inesperada: %+v", asignacion)
	}
	var totalLower int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM asignaciones WHERE agente = 'codex2' AND estado = 'activa'`).Scan(&totalLower); err != nil {
		t.Fatalf("count lower asignaciones: %v", err)
	}
	if totalLower != 0 {
		t.Fatalf("no debian existir asignaciones activas para alias lowercase, total=%d", totalLower)
	}
}

func TestTomarTareaCanonicalizaNombreAgente(t *testing.T) {
	prepararDBTemporal(t)
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo",
		Nombre:  "Demo",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, nombre := range []string{"Codex2", "codex2"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", nombre, err)
		}
	}
	id, err := CrearTarea(&Tarea{
		Titulo:      "Tarea demo",
		Descripcion: "Demo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "test",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(id, "codex2"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	tarea, err := GetTarea(id)
	if err != nil {
		t.Fatalf("GetTarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex2" {
		t.Fatalf("tarea con agente inesperado: %+v", tarea)
	}
	gotID, err := GetTareaActivaIDPorAgenteProyecto("codex2", &proyectoID)
	if err != nil {
		t.Fatalf("GetTareaActivaIDPorAgenteProyecto: %v", err)
	}
	if gotID != id {
		t.Fatalf("id activo inesperado: got=%d want=%d", gotID, id)
	}
}

func TestRegistrarAgenteCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("codex7", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	agente, err := GetAgente("Codex7")
	if err != nil {
		t.Fatalf("GetAgente canonico: %v", err)
	}
	if agente.Nombre != "Codex7" {
		t.Fatalf("nombre canonico inesperado: %+v", agente)
	}
	var totalLower int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM agentes WHERE nombre = 'codex7'`).Scan(&totalLower); err != nil {
		t.Fatalf("count lower agentes: %v", err)
	}
	if totalLower != 0 {
		t.Fatalf("no debia existir fila lowercase, total=%d", totalLower)
	}
}

func TestCanonicalizeAgentNameAliasCodexSinPersistenciaDevuelveCanonico(t *testing.T) {
	prepararDBTemporal(t)
	got, err := CanonicalizeAgentName("codex9")
	if err != nil {
		t.Fatalf("CanonicalizeAgentName: %v", err)
	}
	if got != "Codex9" {
		t.Fatalf("nombre canonico inesperado: got=%q want=%q", got, "Codex9")
	}
}

func TestCanonicalizeAgentNameResuelveCamelCasePersistido(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("CodexPg1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	got, err := CanonicalizeAgentName("codexpg1")
	if err != nil {
		t.Fatalf("CanonicalizeAgentName: %v", err)
	}
	if got != "CodexPg1" {
		t.Fatalf("nombre canonico inesperado: got=%q want=%q", got, "CodexPg1")
	}
}
