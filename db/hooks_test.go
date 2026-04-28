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

func recibirNotificacionHook(t *testing.T) EventoNotificacion {
	t.Helper()
	select {
	case ev := <-CanalNotificaciones:
		return ev
	case <-time.After(2 * time.Second):
		t.Fatal("timeout esperando notificacion hook")
	}
	return EventoNotificacion{}
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

	EmitirHookCicloVida("Codex1", HookTaskStart, proyectoID, "tarea", 42, "Codex1", "Hooks de ciclo de vida")
	ev := recibirNotificacionHook(t)
	if ev.Tipo != "hook:task_start" || ev.ProyectoID != proyectoID || ev.Agente != "Codex1" {
		t.Fatalf("hook task_start inesperado: %+v", ev)
	}
	if !existeAccionAuditoria(t, "hook_task_start") {
		t.Fatalf("no aparece hook_task_start en auditoria")
	}

	EmitirHookCicloVida("Codex1", HookProjectBlocked, proyectoID, "tarea", 42, "Codex1", "esperando validacion humana")
	var vistoBloqueo bool
	for !vistoBloqueo {
		ev = recibirNotificacionHook(t)
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

	EmitirHookCicloVida("Codex1", HookProjectUnblocked, proyectoID, "tarea", 42, "Codex1", "respuesta recibida")
	var vistoDesbloqueo bool
	for !vistoDesbloqueo {
		ev = recibirNotificacionHook(t)
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

func TestCanonicalLifecycleHooksContratoMinimo(t *testing.T) {
	got := CanonicalLifecycleHooks()
	want := []EventoHook{
		HookBeforeTool,
		HookAfterTool,
		HookOnWorkerFail,
		HookOnHandoff,
		HookOnStop,
		HookOnRecovery,
	}
	if len(got) != len(want) {
		t.Fatalf("hooks canonicos inesperados: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("hook canonico[%d] inesperado: got=%q want=%q", i, got[i], want[i])
		}
		if !IsCanonicalLifecycleHook(want[i]) {
			t.Fatalf("hook canonico no reconocido: %q", want[i])
		}
	}
	if IsCanonicalLifecycleHook(HookTaskStart) {
		t.Fatalf("task_start es legacy y no debe formar parte del contrato canonico minimo")
	}
	got[0] = HookTaskStart
	if CanonicalLifecycleHooks()[0] != HookBeforeTool {
		t.Fatalf("CanonicalLifecycleHooks debe devolver copia defensiva")
	}
}

func TestHooksCanonicosPersistenAvisanYNotificanSubscriptores(t *testing.T) {
	tmp := prepararDBTemporal(t)
	drenarNotificacionesHook()

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "hooks-canonicos",
		Nombre:  "Hooks canonicos",
		RutaAbs: filepath.Join(tmp, "hooks-canonicos"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	got := make(chan LifecycleHookEvent, 1)
	unregister := RegisterLifecycleHookSubscriber(func(ev LifecycleHookEvent) {
		got <- ev
	})
	defer unregister()

	EmitirHookCicloVida("Codex2", HookBeforeTool, proyectoID, "tool_call", 99, "Codex2", "rg hooks")

	ev := recibirNotificacionHook(t)
	if ev.Tipo != "hook:before_tool" || ev.ProyectoID != proyectoID || ev.ID != 99 || ev.Agente != "Codex2" {
		t.Fatalf("notificacion before_tool inesperada: %+v", ev)
	}
	if !existeAccionAuditoria(t, "hook_before_tool") {
		t.Fatalf("no aparece hook_before_tool en auditoria")
	}
	select {
	case hook := <-got:
		if hook.Evento != HookBeforeTool || hook.Entidad != "tool_call" || hook.EntidadID != 99 || hook.Detalle != "rg hooks" {
			t.Fatalf("subscriber hook canonico inesperado: %+v", hook)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("subscriber no recibio hook canonico")
	}
}

func TestListarAuditoriaFiltraDesde(t *testing.T) {
	prepararDBTemporal(t)
	Audit("Codex1", "accion_antigua", "tarea", 1, "antes")
	if _, err := DB.Exec(`UPDATE audit_log SET created_at = ? WHERE accion = ?`, time.Now().UTC().Add(-2*time.Hour), "accion_antigua"); err != nil {
		t.Fatalf("retrofechar audit antigua: %v", err)
	}
	Audit("Codex1", "accion_reciente", "tarea", 2, "despues")

	desde := time.Now().UTC().Add(-30 * time.Minute)
	items, err := ListarAuditoria(FiltroAuditoria{Desde: &desde, Limite: 20})
	if err != nil {
		t.Fatalf("listar auditoria: %v", err)
	}
	if len(items) != 1 || items[0] == nil || items[0].Accion != "accion_reciente" {
		t.Fatalf("auditoria filtrada inesperada: %+v", items)
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

func TestLifecycleHookSubscriberRecibeEvento(t *testing.T) {
	got := make(chan LifecycleHookEvent, 1)
	unregister := RegisterLifecycleHookSubscriber(func(ev LifecycleHookEvent) {
		got <- ev
	})
	defer unregister()

	EmitirHookCicloVida("Codex1", HookTaskStart, 7, "tarea", 42, "Codex1", "arranque")

	select {
	case ev := <-got:
		if ev.Evento != HookTaskStart || ev.ProyectoID != 7 || ev.EntidadID != 42 || ev.Agente != "Codex1" {
			t.Fatalf("evento lifecycle inesperado: %+v", ev)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("subscriber lifecycle no recibio evento")
	}
}
