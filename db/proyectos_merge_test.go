/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"strings"
	"testing"
)

func TestFusionarProyectosReasignaReferenciasYArchivaOrigen(t *testing.T) {
	prepararDBTemporal(t)

	destinoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto destino: %v", err)
	}
	origenID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquesta",
		Nombre:  "orquesta",
		RutaAbs: "/tmp/orquesta",
		Tipo:    ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto origen: %v", err)
	}

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}

	if _, err := DB.Exec(`INSERT INTO memoria_proyectos (proyecto, resumen, contexto, preguntas_abiertas, actualizado_por) VALUES (?,?,?,?,?)`,
		"orquestador", "resumen destino", "", "", "Codex1",
	); err != nil {
		t.Fatalf("insert memoria destino: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO memoria_proyectos (proyecto, resumen, contexto, preguntas_abiertas, actualizado_por) VALUES (?,?,?,?,?)`,
		"orquesta", "resumen origen", "contexto origen", "", "Codex2",
	); err != nil {
		t.Fatalf("insert memoria origen: %v", err)
	}
	usaProyectoSlug, err := ColumnExists("decisiones_proyecto", "proyecto")
	if err != nil {
		t.Fatalf("ColumnExists decisiones_proyecto.proyecto: %v", err)
	}
	if usaProyectoSlug {
		if _, err := DB.Exec(`INSERT INTO decisiones_proyecto (proyecto, titulo) VALUES (?,?)`,
			"orquesta", "Unificar proyecto canonico",
		); err != nil {
			t.Fatalf("insert decision origen legacy: %v", err)
		}
	} else {
		if _, err := DB.Exec(`INSERT INTO decisiones_proyecto (proyecto_id, titulo, solucion) VALUES (?,?,?)`,
			origenID, "Unificar proyecto canonico", "Fusionar proyecto duplicado",
		); err != nil {
			t.Fatalf("insert decision origen: %v", err)
		}
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Limpiar continuidad",
		Descripcion: "tarea de prueba",
		ProyectoID:  &origenID,
		Modulo:      "sesiones",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "Codex1",
	})
	if err != nil {
		t.Fatalf("crear tarea origen: %v", err)
	}

	if _, err := DB.Exec(`INSERT INTO asignaciones (agente, proyecto_id, estado, nota) VALUES (?,?,?,?)`,
		"Codex2", destinoID, "pausada", "destino previo",
	); err != nil {
		t.Fatalf("insert asignacion destino: %v", err)
	}
	if err := ActivarAsignacion("Codex2", origenID, "origen activo"); err != nil {
		t.Fatalf("activar asignacion origen: %v", err)
	}

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &origenID,
		CWD:         "/tmp/orquesta",
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion origen: %v", err)
	}
	if err := UpsertRuntimeDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert runtime: %v", err)
	}
	if err := UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert handle: %v", err)
	}
	if hot, err := GetRuntimeHandleCanonicoRecienteAgenteProyecto("Codex2", &origenID); err != nil {
		t.Fatalf("precargar cache hot origen: %v", err)
	} else if hot == nil || hot.ProyectoID == nil || *hot.ProyectoID != origenID {
		t.Fatalf("cache hot origen inesperada: %+v", hot)
	}

	resultado, err := FusionarProyectos("orquesta", "orquestador", FusionProyectosOptions{ArchivarOrigen: true})
	if err != nil {
		t.Fatalf("FusionarProyectos: %v", err)
	}
	if resultado == nil {
		t.Fatalf("resultado nil")
	}
	if resultado.OrigenID != origenID || resultado.DestinoID != destinoID {
		t.Fatalf("ids inesperados: %+v", resultado)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.ProyectoID == nil || *tarea.ProyectoID != destinoID {
		t.Fatalf("tarea no movida al destino: %+v", tarea.ProyectoID)
	}

	sesionActual, err := GetSesionActiva("Codex2", &destinoID)
	if err != nil {
		t.Fatalf("get sesion activa destino: %v", err)
	}
	if sesionActual == nil || sesionActual.ProyectoID == nil || *sesionActual.ProyectoID != destinoID {
		t.Fatalf("sesion no movida al destino: %+v", sesionActual)
	}

	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get runtime por sesion: %v", err)
	}
	if runtime == nil || runtime.ProyectoID == nil || *runtime.ProyectoID != destinoID {
		t.Fatalf("runtime no movido al destino: %+v", runtime)
	}

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get handle por sesion: %v", err)
	}
	if handle == nil || handle.ProyectoID == nil || *handle.ProyectoID != destinoID {
		t.Fatalf("handle no movido al destino: %+v", handle)
	}
	if hotOrigen, err := GetRuntimeHandleCanonicoRecienteAgenteProyecto("Codex2", &origenID); err != nil {
		t.Fatalf("hot cache origen tras fusion: %v", err)
	} else if hotOrigen != nil {
		t.Fatalf("la cache hot no deberia seguir devolviendo handle del proyecto origen: %+v", hotOrigen)
	}
	if hotDestino, err := GetRuntimeHandleCanonicoRecienteAgenteProyecto("Codex2", &destinoID); err != nil {
		t.Fatalf("hot cache destino tras fusion: %v", err)
	} else if hotDestino == nil || hotDestino.ProyectoID == nil || *hotDestino.ProyectoID != destinoID {
		t.Fatalf("la cache hot deberia apuntar al proyecto destino: %+v", hotDestino)
	}

	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{Agente: ptrStringProyectoMerge("Codex2")})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	var activasDestino int
	for _, item := range asignaciones {
		if item.ProyectoID == destinoID && item.Estado == AsignacionActiva {
			activasDestino++
		}
		if item.ProyectoID == origenID && item.Estado == AsignacionActiva {
			t.Fatalf("quedó asignación activa en el origen archivado: %+v", item)
		}
	}
	if activasDestino != 1 {
		t.Fatalf("se esperaba una única asignación activa en destino, hay %d", activasDestino)
	}

	destinoProyecto, err := GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto destino: %v", err)
	}
	if destinoProyecto == nil || !destinoProyecto.Activo {
		t.Fatalf("proyecto destino inesperado: %+v", destinoProyecto)
	}

	origenArchivado, err := GetProyecto(resultado.SlugArchivado)
	if err != nil {
		t.Fatalf("get proyecto archivado: %v", err)
	}
	if origenArchivado == nil || origenArchivado.Activo {
		t.Fatalf("origen no archivado correctamente: %+v", origenArchivado)
	}
	if !strings.HasPrefix(origenArchivado.Slug, "orquesta-archivado-") {
		t.Fatalf("slug archivado inesperado: %s", origenArchivado.Slug)
	}
	if origenArchivado.RutaAbs == "/tmp/orquesta" {
		t.Fatalf("ruta archivada no diferenciada: %s", origenArchivado.RutaAbs)
	}

	var resumen string
	if err := DB.QueryRow(`SELECT resumen FROM memoria_proyectos WHERE proyecto = ?`, "orquestador").Scan(&resumen); err != nil {
		t.Fatalf("memoria destino: %v", err)
	}
	if !strings.Contains(resumen, "resumen destino") || !strings.Contains(resumen, "resumen origen") {
		t.Fatalf("memoria fusionada inesperada: %q", resumen)
	}

	if usaProyectoSlug {
		var proyectoDecision string
		if err := DB.QueryRow(`SELECT proyecto FROM decisiones_proyecto WHERE titulo = ?`, "Unificar proyecto canonico").Scan(&proyectoDecision); err != nil {
			t.Fatalf("decision fusionada legacy: %v", err)
		}
		if proyectoDecision != "orquestador" {
			t.Fatalf("decision legacy no movida al destino: %q", proyectoDecision)
		}
	} else {
		var proyectoDecisionID int64
		if err := DB.QueryRow(`SELECT proyecto_id FROM decisiones_proyecto WHERE titulo = ?`, "Unificar proyecto canonico").Scan(&proyectoDecisionID); err != nil {
			t.Fatalf("decision fusionada: %v", err)
		}
		if proyectoDecisionID != destinoID {
			t.Fatalf("decision no movida al destino: %d", proyectoDecisionID)
		}
	}
}

func TestFusionarProyectosSoportaDecisionesLegacyPorSlug(t *testing.T) {
	prepararDBTemporal(t)

	if _, err := DB.Exec(`DROP TRIGGER IF EXISTS trig_decisiones_proyecto_updated`); err != nil {
		t.Fatalf("drop trigger decisiones_proyecto: %v", err)
	}
	if _, err := DB.Exec(`DROP TABLE IF EXISTS decisiones_proyecto`); err != nil {
		t.Fatalf("drop decisiones_proyecto: %v", err)
	}
	if _, err := DB.Exec(`
		CREATE TABLE decisiones_proyecto (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			proyecto TEXT NOT NULL,
			titulo TEXT NOT NULL,
			solucion_elegida TEXT NOT NULL DEFAULT '',
			motivo TEXT NOT NULL DEFAULT '',
			alternativas_descartadas TEXT NOT NULL DEFAULT '',
			impacto TEXT NOT NULL DEFAULT 'medio',
			propuesta_codigo TEXT NOT NULL DEFAULT '',
			tarea_id INTEGER,
			registrado_por TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		t.Fatalf("create decisiones_proyecto legacy: %v", err)
	}

	destinoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto destino: %v", err)
	}
	_, err = UpsertProyecto(&Proyecto{
		Slug:    "orquesta",
		Nombre:  "orquesta",
		RutaAbs: "/tmp/orquesta",
		Tipo:    ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto origen: %v", err)
	}

	if _, err := DB.Exec(`INSERT INTO decisiones_proyecto (proyecto, titulo, solucion_elegida, motivo, registrado_por) VALUES (?,?,?,?,?)`,
		"orquesta", "Decisión legacy", "Mover por slug", "compatibilidad con DB previa", "Codex2",
	); err != nil {
		t.Fatalf("insert decision legacy: %v", err)
	}

	resultado, err := FusionarProyectos("orquesta", "orquestador", FusionProyectosOptions{ArchivarOrigen: true})
	if err != nil {
		t.Fatalf("FusionarProyectos legacy: %v", err)
	}
	if resultado == nil {
		t.Fatalf("resultado nil")
	}
	if resultado.Actualizadas["decisiones_proyecto.proyecto"] != 1 {
		t.Fatalf("se esperaba mover 1 decision legacy, got %+v", resultado.Actualizadas)
	}

	var proyecto string
	if err := DB.QueryRow(`SELECT proyecto FROM decisiones_proyecto WHERE titulo = ?`, "Decisión legacy").Scan(&proyecto); err != nil {
		t.Fatalf("consultar decision legacy: %v", err)
	}
	if proyecto != "orquestador" {
		t.Fatalf("decision legacy no movida al destino: %q", proyecto)
	}
	if destinoID == 0 {
		t.Fatalf("destinoID no debería ser cero")
	}
}

func TestFusionarProyectosRechazaDesactivarArchivoOrigen(t *testing.T) {
	prepararDBTemporal(t)

	if _, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    ProyectoRaiz,
		Activo:  true,
	}); err != nil {
		t.Fatalf("crear proyecto destino: %v", err)
	}
	if _, err := UpsertProyecto(&Proyecto{
		Slug:    "orquesta",
		Nombre:  "orquesta",
		RutaAbs: "/tmp/orquesta",
		Tipo:    ProyectoRaiz,
		Activo:  true,
	}); err != nil {
		t.Fatalf("crear proyecto origen: %v", err)
	}

	if _, err := FusionarProyectos("orquesta", "orquestador", FusionProyectosOptions{ArchivarOrigen: false}); err == nil {
		t.Fatalf("se esperaba error al desactivar el archivado del origen")
	}
}

func TestFusionarProyectosConservaTareasConflictivasPorBlueprintEnArchivado(t *testing.T) {
	prepararDBTemporal(t)

	destinoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto destino: %v", err)
	}
	origenID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquesta",
		Nombre:  "orquesta",
		RutaAbs: "/tmp/orquesta",
		Tipo:    ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto origen: %v", err)
	}

	if _, err := CrearTarea(&Tarea{
		Titulo:       "Briefing destino",
		ProyectoID:   &destinoID,
		Modulo:       "producto",
		Prioridad:    PrioridadAlta,
		CreadoPor:    "Codex1",
		BlueprintKey: "briefing",
	}); err != nil {
		t.Fatalf("crear tarea destino: %v", err)
	}
	tareaConflictoID, err := CrearTarea(&Tarea{
		Titulo:       "Briefing origen",
		ProyectoID:   &origenID,
		Modulo:       "producto",
		Prioridad:    PrioridadAlta,
		CreadoPor:    "Codex2",
		BlueprintKey: "briefing",
	})
	if err != nil {
		t.Fatalf("crear tarea origen conflictiva: %v", err)
	}
	tareaMovidaID, err := CrearTarea(&Tarea{
		Titulo:       "Investigacion origen",
		ProyectoID:   &origenID,
		Modulo:       "analisis",
		Prioridad:    PrioridadAlta,
		CreadoPor:    "Codex2",
		BlueprintKey: "investigacion",
	})
	if err != nil {
		t.Fatalf("crear tarea origen movible: %v", err)
	}

	resultado, err := FusionarProyectos("orquesta", "orquestador", FusionProyectosOptions{ArchivarOrigen: true})
	if err != nil {
		t.Fatalf("FusionarProyectos: %v", err)
	}
	if got := resultado.Conflictos["tareas.proyecto_id"]; got != 1 {
		t.Fatalf("conflictos de tareas inesperados: %+v", resultado.Conflictos)
	}

	origenArchivado, err := GetProyecto(resultado.SlugArchivado)
	if err != nil {
		t.Fatalf("get origen archivado: %v", err)
	}
	if origenArchivado == nil {
		t.Fatalf("origen archivado nil")
	}

	tareaConflicto, err := GetTarea(tareaConflictoID)
	if err != nil {
		t.Fatalf("get tarea conflicto: %v", err)
	}
	if tareaConflicto.ProyectoID == nil || *tareaConflicto.ProyectoID != origenArchivado.ID {
		t.Fatalf("la tarea conflictiva deberia quedar en el proyecto archivado: %+v", tareaConflicto.ProyectoID)
	}

	tareaMovida, err := GetTarea(tareaMovidaID)
	if err != nil {
		t.Fatalf("get tarea movida: %v", err)
	}
	if tareaMovida.ProyectoID == nil || *tareaMovida.ProyectoID != destinoID {
		t.Fatalf("la tarea no conflictiva deberia moverse al destino: %+v", tareaMovida.ProyectoID)
	}
}

func ptrStringProyectoMerge(v string) *string {
	return &v
}
