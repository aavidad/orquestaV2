package db

import (
	"path/filepath"
	"testing"
)

func TestGetAsignacionActivaAgenteDevuelveUltimaActiva(t *testing.T) {
	tmp := prepararDBTemporal(t)

	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "alpha",
		Nombre:  "Alpha",
		RutaAbs: filepath.Join(tmp, "alpha"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto alpha: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "beta",
		Nombre:  "Beta",
		RutaAbs: filepath.Join(tmp, "beta"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto beta: %v", err)
	}

	if _, err := DB.Exec(`INSERT INTO asignaciones (agente, proyecto_id, estado, nota) VALUES (?, ?, 'activa', ?)`, "Codex1", proyectoA, "a"); err != nil {
		t.Fatalf("insert asignacion alpha: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO asignaciones (agente, proyecto_id, estado, nota) VALUES (?, ?, 'activa', ?)`, "Codex1", proyectoB, "b"); err != nil {
		t.Fatalf("insert asignacion beta: %v", err)
	}

	got, err := GetAsignacionActivaAgente("Codex1")
	if err != nil {
		t.Fatalf("get asignacion activa: %v", err)
	}
	if got == nil || got.ProyectoID != proyectoB {
		t.Fatalf("asignacion activa inesperada: %+v", got)
	}
}

func TestAgenteTieneTrabajoArrancableDetectaTareaAsignada(t *testing.T) {
	tmp := prepararDBTemporal(t)

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Cerrar backlog",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "test",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	ok, err := agenteTieneTrabajoArrancable("Codex1", proyectoID)
	if err != nil {
		t.Fatalf("agenteTieneTrabajoArrancable: %v", err)
	}
	if !ok {
		t.Fatalf("deberia detectar tarea asignada")
	}
}

func TestExisteRuntimeOrderAbiertaFiltraProyectoYTipo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "alpha",
		Nombre:  "Alpha",
		RutaAbs: filepath.Join(tmp, "alpha"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto alpha: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "beta",
		Nombre:  "Beta",
		RutaAbs: filepath.Join(tmp, "beta"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto beta: %v", err)
	}

	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoA,
		Tipo:       "start",
		Estado:     "pendiente",
	}); err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoB,
		Tipo:       "send_instruction",
		Estado:     "pendiente",
	}); err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}

	if ok, err := existeRuntimeOrderAbierta("Codex1", &proyectoA, "send_instruction"); err != nil || ok {
		t.Fatalf("no deberia contar otro proyecto: ok=%t err=%v", ok, err)
	}
	if ok, err := existeRuntimeOrderAbierta("Codex1", &proyectoB, "send_instruction"); err != nil || !ok {
		t.Fatalf("deberia detectar orden viva por proyecto/tipo: ok=%t err=%v", ok, err)
	}
}

func TestLiberarBacklogNoLiberaFinishAppDuplicadaSiYaHayOtraActiva(t *testing.T) {
	tmp := prepararDBTemporal(t)

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	activaID, err := CrearTarea(&Tarea{
		Titulo:      "Cerrar Orquesta al 100%",
		Descripcion: "finish_app activa",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "test",
		Notas:       "autonomia:finish_app",
	})
	if err != nil {
		t.Fatalf("crear tarea activa: %v", err)
	}
	if err := TomarTarea(activaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea activa: %v", err)
	}
	if err := IniciarTarea(activaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea activa: %v", err)
	}
	duplicadaID, err := CrearTarea(&Tarea{
		Titulo:      "Cerrar Orquesta al 100%",
		Descripcion: "finish_app duplicada",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "test",
		Notas:       "autonomia:finish_app",
	})
	if err != nil {
		t.Fatalf("crear tarea duplicada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE tareas SET estado='backlog' WHERE id=?`, duplicadaID); err != nil {
		t.Fatalf("pasar duplicada a backlog: %v", err)
	}

	if err := liberarBacklog(); err != nil {
		t.Fatalf("liberarBacklog: %v", err)
	}

	duplicada, err := GetTarea(duplicadaID)
	if err != nil {
		t.Fatalf("get tarea duplicada: %v", err)
	}
	if duplicada == nil || duplicada.Estado != EstadoBacklog {
		t.Fatalf("la finish_app duplicada deberia seguir en backlog: %+v", duplicada)
	}
}

func TestBuscarSiguienteTareaLibreDevuelveUnaLibre(t *testing.T) {
	tmp := prepararDBTemporal(t)

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := CrearTarea(&Tarea{
		Titulo:      "Libre",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "test",
	}); err != nil {
		t.Fatalf("crear tarea libre: %v", err)
	}

	tarea, err := buscarSiguienteTareaLibre(proyectoID)
	if err != nil {
		t.Fatalf("buscarSiguienteTareaLibre: %v", err)
	}
	if tarea == nil || tarea.ID <= 0 {
		t.Fatalf("deberia devolver una tarea libre: %+v", tarea)
	}
}
