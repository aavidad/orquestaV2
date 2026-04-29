package cmd

import (
	"path/filepath"
	"testing"
	"time"

	"orquesta/db"
)

func TestProcesarCompactacionExclusividadPremiumSesionActivaReactivacionAutomatica(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}

	generalID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente heredado amplio",
		Descripcion: "Sin contrato premium acotado.",
		ProyectoID:  &proyectoID,
		Modulo:      "capacidadapp",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea general: %v", err)
	}
	if err := db.TomarTarea(generalID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea general: %v", err)
	}
	if err := db.IniciarTarea(generalID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea general: %v", err)
	}

	premiumID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente premium acotado",
		Descripcion: "Slice bounded.\nWrite-set preferente: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarCompactacionExclusividadPremiumSesionActivaReactivacionAutomatica$'",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea premium: %v", err)
	}
	if err := db.TomarTarea(premiumID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea premium: %v", err)
	}
	if err := db.IniciarTarea(premiumID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea premium: %v", err)
	}

	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarCompactacionExclusividadPremiumSesionActiva(sesion, nil)
	if err != nil {
		t.Fatalf("compactar exclusividad premium: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia compactar exactamente un frente heredado amplio tras la reactivacion, got=%d", n)
	}

	general, err := db.GetTarea(generalID)
	if err != nil {
		t.Fatalf("recargar tarea general: %v", err)
	}
	if general == nil || general.Estado != db.TareaBacklog {
		t.Fatalf("la tarea general deberia volver a backlog: %+v", general)
	}

	premium, err := db.GetTarea(premiumID)
	if err != nil {
		t.Fatalf("recargar tarea premium: %v", err)
	}
	if premium == nil || premium.Estado != db.TareaEnProgreso {
		t.Fatalf("el frente premium acotado deberia seguir en progreso: %+v", premium)
	}
}

func TestReactivarSesionAutonomiaPorTrabajoOmiteSesionSinTrabajoArrancable(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("GeminiIdle", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-idle",
		Nombre:  "Orquestador Idle",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "GeminiIdle",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	sesionRecargada, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionRecargada == nil {
		t.Fatalf("recargar sesion: %+v err=%v", sesionRecargada, err)
	}

	n, err := reactivarSesionAutonomiaPorTrabajo(sesionRecargada)
	if err != nil {
		t.Fatalf("reactivar sesion por trabajo: %v", err)
	}
	if n != 0 {
		t.Fatalf("sin trabajo arrancable no deberia relanzar premium, got=%d", n)
	}

	agente := "GeminiIdle"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("sin trabajo arrancable no deberia encolar runtime orders: %+v", orders)
	}
}

func TestReactivarSesionAutonomiaPorTrabajoRespetaCooldownRecienteDeStartResume(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("GeminiCooldown", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-cooldown-reactivacion",
		Nombre:  "Orquestador Cooldown",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar premium con cooldown",
		Descripcion: "Hay trabajo real, pero el relanzamiento debe esperar el cooldown.",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "GeminiCooldown"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "GeminiCooldown"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO runtime_orders (agente, proyecto_id, tipo, estado, payload_json, created_at, updated_at)
		VALUES (?, ?, 'resume', 'completada', ?, ?, ?)
	`, "GeminiCooldown", proyectoID, `{"accion":"resume","motivo":"desbloqueo_humano_auto"}`, time.Now().UTC(), time.Now().UTC()); err != nil {
		t.Fatalf("insert runtime order reciente: %v", err)
	}

	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "GeminiCooldown",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	sesionRecargada, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionRecargada == nil {
		t.Fatalf("recargar sesion: %+v err=%v", sesionRecargada, err)
	}

	n, err := reactivarSesionAutonomiaPorTrabajo(sesionRecargada)
	if err != nil {
		t.Fatalf("reactivar sesion por trabajo: %v", err)
	}
	if n != 0 {
		t.Fatalf("deberia respetar cooldown reciente de start/resume, got=%d", n)
	}

	agente := "GeminiCooldown"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia encolar runtime orders durante cooldown: %+v", orders)
	}
}

func TestReactivarSesionAutonomiaPorTrabajoCompactaFrentesHeredados(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion inicial: %v", err)
	}

	generalID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente heredado amplio",
		Descripcion: "Sin contrato premium acotado.",
		ProyectoID:  &proyectoID,
		Modulo:      "capacidadapp",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea general: %v", err)
	}
	if err := db.TomarTarea(generalID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea general: %v", err)
	}
	if err := db.IniciarTarea(generalID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea general: %v", err)
	}

	premiumID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente premium acotado",
		Descripcion: "Slice bounded.\nWrite-set preferente: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestReactivarSesionAutonomiaPorTrabajoCompactaFrentesHeredados$'",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea premium: %v", err)
	}
	if err := db.TomarTarea(premiumID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea premium: %v", err)
	}
	if err := db.IniciarTarea(premiumID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea premium: %v", err)
	}

	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	sesionRecargada, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionRecargada == nil {
		t.Fatalf("recargar sesion: %+v err=%v", sesionRecargada, err)
	}

	n, err := reactivarSesionAutonomiaPorTrabajo(sesionRecargada)
	if err != nil {
		t.Fatalf("reactivar sesion por trabajo: %v", err)
	}
	if n < 2 {
		t.Fatalf("deberia compactar al menos un frente heredado y encolar control, got=%d", n)
	}

	general, err := db.GetTarea(generalID)
	if err != nil {
		t.Fatalf("recargar tarea general: %v", err)
	}
	if general == nil || general.Estado != db.TareaBacklog {
		t.Fatalf("la tarea general deberia volver a backlog tras la reactivacion: %+v", general)
	}

	premium, err := db.GetTarea(premiumID)
	if err != nil {
		t.Fatalf("recargar tarea premium: %v", err)
	}
	if premium == nil || premium.Estado != db.TareaEnProgreso {
		t.Fatalf("el frente premium acotado deberia seguir en progreso: %+v", premium)
	}
}
