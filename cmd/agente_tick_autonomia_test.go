package cmd

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/propuestasapp"
)

func TestConstruirAgenteTickOutputPriorizaSupervisionParaSupervisorOperativo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "Codex1",
		ReserveSupervisor:    true,
		AutoCreateTasks:      true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if err := db.ActivarAsignacion("Codex2", proyectoID, "worker"); err != nil {
		t.Fatalf("activar asignacion worker: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	if _, _, err := propuestasService.Create(propuestasapp.CreateProposalInput{
		Codigo:       "PROP-1",
		Titulo:       "Propuesta de prueba",
		Descripcion:  "detalle",
		Tipo:         "otro",
		Proyecto:     "orquestador",
		PropuestoPor: "tester",
		Distribuidor: "orquesta",
	}); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	out, err := construirAgenteTickOutput("Codex1", proyecto, nil, 100)
	if err != nil {
		t.Fatalf("construir tick: %v", err)
	}
	if out.AccionRecomendada != "supervisar_proyecto" {
		t.Fatalf("accion recomendada inesperada: %+v", out)
	}
	if !strings.Contains(strings.ToLower(out.Motivo), "supervisor operativo") {
		t.Fatalf("motivo inesperado: %q", out.Motivo)
	}
}

func TestConstruirAgenteTickOutputPausaPorPresupuestoCriticoFresco(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex5", "programador"); err != nil {
		t.Fatalf("registrar Codex5: %v", err)
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
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex5",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.ActivarAsignacion("Codex5", proyectoID, "worker"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "Tarea critica",
		Modulo:    "orquestador",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex5"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex5"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.GetSesionActiva("Codex5", &proyectoID)
	if err != nil {
		t.Fatalf("get sesion activa: %v", err)
	}
	credits := 0.0
	started := time.Now().UTC().Add(-10 * time.Minute)
	reset := started.Add(5 * time.Hour)
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:         sesion.ID,
		WindowKind:       "5h",
		WindowStartedAt:  &started,
		ResetAt:          &reset,
		RemainingCredits: &credits,
		BudgetSource:     "codex_token_count_observed",
		CheckedAt:        time.Now().UTC(),
	}); err != nil {
		t.Fatalf("registrar presupuesto: %v", err)
	}

	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	out, err := construirAgenteTickOutput("Codex5", proyecto, sesion, 0)
	if err != nil {
		t.Fatalf("construir tick: %v", err)
	}
	if out.AccionRecomendada != "pausar_por_cuota" || !out.DebePausar {
		t.Fatalf("deberia pausar por presupuesto critico fresco: %+v", out)
	}
	if !strings.Contains(strings.ToLower(out.Motivo), "presupuesto crítico") {
		t.Fatalf("motivo inesperado: %q", out.Motivo)
	}
}

func TestConstruirAgenteTickOutputDescribePausaOperativaPorRuntimePanic(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
		t.Fatalf("registrar Codex4: %v", err)
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
	if err := db.ActivarAsignacion("Codex4", proyectoID, "worker"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	reanimarAt := time.Now().UTC().Add(10 * time.Minute)
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', reanimar_at=?, motivo_pausa='Auto-pausa por runtime_panic: transcript=10576' WHERE nombre='Codex4'`, reanimarAt); err != nil {
		t.Fatalf("actualizar agente: %v", err)
	}

	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	out, err := construirAgenteTickOutput("Codex4", proyecto, nil, 99)
	if err != nil {
		t.Fatalf("construir tick: %v", err)
	}
	if out.AccionRecomendada != "pausar_por_cuota" || !out.DebePausar {
		t.Fatalf("salida inesperada: %+v", out)
	}
	if !strings.Contains(out.Motivo, "Pausa operativa") {
		t.Fatalf("deberia describir pausa operativa: %q", out.Motivo)
	}
	if strings.Contains(out.Motivo, "Cuota agotada") {
		t.Fatalf("no deberia mentir con cuota agotada: %q", out.Motivo)
	}
}

func TestResolverAccionTickAutonomiaPrioridades(t *testing.T) {
	t.Run("presupuesto manda sobre trabajo", func(t *testing.T) {
		accion, pausar, motivo := resolverAccionTickAutonomia(decisionTickAutonomiaInput{
			PausarPorPresupuesto: true,
			MotivoPresupuesto:    "presupuesto crítico",
			TieneTrabajo:         true,
		})
		if accion != "pausar_por_cuota" || !pausar || motivo != "presupuesto crítico" {
			t.Fatalf("decision inesperada: accion=%s pausar=%v motivo=%q", accion, pausar, motivo)
		}
	})

	t.Run("supervision manda sobre propuestas", func(t *testing.T) {
		accion, pausar, motivo := resolverAccionTickAutonomia(decisionTickAutonomiaInput{
			EsSupervisorOperativo: true,
			AgenteNombre:          "Codex1",
			PropuestasPendientes:  3,
		})
		if accion != "supervisar_proyecto" || pausar {
			t.Fatalf("decision inesperada: accion=%s pausar=%v motivo=%q", accion, pausar, motivo)
		}
	})

	t.Run("bloqueos mandan sobre continuar trabajo", func(t *testing.T) {
		accion, pausar, motivo := resolverAccionTickAutonomia(decisionTickAutonomiaInput{
			TieneBloqueos: true,
			TieneTrabajo:  true,
		})
		if accion != "pedir_intervencion" || pausar || !strings.Contains(strings.ToLower(motivo), "bloqueadas") {
			t.Fatalf("decision inesperada: accion=%s pausar=%v motivo=%q", accion, pausar, motivo)
		}
	})

	t.Run("sin trabajo cae a esperar_o_pedir_tarea", func(t *testing.T) {
		accion, pausar, motivo := resolverAccionTickAutonomia(decisionTickAutonomiaInput{})
		if accion != "esperar_o_pedir_tarea" || pausar || !strings.Contains(strings.ToLower(motivo), "no hay tarea activa") {
			t.Fatalf("decision inesperada: accion=%s pausar=%v motivo=%q", accion, pausar, motivo)
		}
	})
}

func TestConstruirAgenteTickOutputNoPideIntervencionPorBloqueoAutorecuperableConSesionActiva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar Gemini1: %v", err)
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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "worker"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente bloqueado autorecuperable",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Gemini1", "Agente Gemini1 en estado bloqueado_por_runtime: worker recuperado"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	out, err := construirAgenteTickOutput("Gemini1", proyecto, nil, 100)
	if err != nil {
		t.Fatalf("construir tick: %v", err)
	}
	if out.AccionRecomendada == "pedir_intervencion" {
		t.Fatalf("no deberia pedir intervencion humana por un bloqueo autorecuperable: %+v", out)
	}
}
