package db

import (
	"path/filepath"
	"testing"
	"time"
)

func drenarNotificacionesHook() {
	for {
		select {
		case <-CanalNotificaciones:
		default:
			return
		}
	}
}

func existeAccionAuditoria(t *testing.T, accion string) bool {
	t.Helper()
	logs, err := ListarAuditoria(FiltroAuditoria{Limite: 20})
	if err != nil {
		t.Fatalf("listar auditoria: %v", err)
	}
	for _, item := range logs {
		if item != nil && item.Accion == accion {
			return true
		}
	}
	return false
}

func TestHooksCicloVidaPersistenYAvisan(t *testing.T) {
	tmp := prepararDBTemporal(t)
	drenarNotificacionesHook()

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
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
		Titulo:      "Hooks de ciclo de vida",
		Descripcion: "Cobertura de hooks",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	ev := <-CanalNotificaciones
	if ev.Tipo != "hook:task_start" || ev.ProyectoID != proyectoID || ev.Agente != "Codex1" {
		t.Fatalf("hook task_start inesperado: %+v", ev)
	}
	if !existeAccionAuditoria(t, "hook_task_start") {
		t.Fatalf("no aparece hook_task_start en auditoria")
	}

	if err := BloquearTarea(tareaID, "Codex1", "esperando validacion humana"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	var vistoBloqueo bool
	for !vistoBloqueo {
		ev = <-CanalNotificaciones
		if ev.Tipo == "hook:project_blocked" {
			vistoBloqueo = true
			if ev.ProyectoID != proyectoID || ev.Agente != "Codex1" {
				t.Fatalf("hook project_blocked inesperado: %+v", ev)
			}
		}
	}
	if !existeAccionAuditoria(t, "hook_project_blocked") {
		t.Fatalf("no aparece hook_project_blocked en auditoria")
	}

	if err := DesbloquearTarea(tareaID, "Codex1", "respuesta recibida"); err != nil {
		t.Fatalf("desbloquear tarea: %v", err)
	}
	var vistoDesbloqueo bool
	for !vistoDesbloqueo {
		ev = <-CanalNotificaciones
		if ev.Tipo == "hook:project_unblocked" {
			vistoDesbloqueo = true
			if ev.ProyectoID != proyectoID || ev.Agente != "Codex1" {
				t.Fatalf("hook project_unblocked inesperado: %+v", ev)
			}
		}
	}
	if !existeAccionAuditoria(t, "hook_project_unblocked") {
		t.Fatalf("no aparece hook_project_unblocked en auditoria")
	}
}

func TestEmitirNotificacionNoBloqueaSiCanalEstaLleno(t *testing.T) {
	drenarNotificacionesHook()
	for i := 0; i < cap(CanalNotificaciones); i++ {
		CanalNotificaciones <- EventoNotificacion{Tipo: "fill"}
	}
	defer drenarNotificacionesHook()

	done := make(chan bool, 1)
	go func() {
		done <- EmitirNotificacion(EventoNotificacion{Tipo: "overflow"})
	}()

	select {
	case delivered := <-done:
		if delivered {
			t.Fatalf("no deberia entregar cuando el canal esta saturado")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("EmitirNotificacion se bloqueo con el canal lleno")
	}
}
