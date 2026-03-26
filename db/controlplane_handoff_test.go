package db

import (
	"encoding/json"
	"testing"
)

func strPtrHandoffTest(v string) *string { return &v }

func TestCrearHandoffAgenteVivoReasignaTareaYCreaOrden(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}

	sesionOrigenID, err := IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("IniciarSesion origen: %v", err)
	}
	handleOrigen, err := GetRuntimeHandleBySesionID(sesionOrigenID)
	if err != nil {
		t.Fatalf("get handle origen: %v", err)
	}
	if handleOrigen == nil || handleOrigen.Estado != "activo" {
		t.Fatalf("handle origen inesperado: %+v", handleOrigen)
	}

	if _, err := IniciarSesion("Codex2"); err != nil {
		t.Fatalf("IniciarSesion destino: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:    "Revisar handoff",
		Modulo:    "orquestador",
		Prioridad: PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}

	orderID, err := CrearHandoffAgenteVivo("Codex1", "Codex2", &tareaID, "presupuesto en rojo", "Continuar desde checkpoint", "ext-123")
	if err != nil {
		t.Fatalf("CrearHandoffAgenteVivo: %v", err)
	}
	if orderID == 0 {
		t.Fatalf("order id inesperado: %d", orderID)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("GetTarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex2" {
		t.Fatalf("agente reasignado inesperado: %+v", tarea.Agente)
	}
	if tarea.Estado != TareaAsignada {
		t.Fatalf("estado inesperado: %s", tarea.Estado)
	}

	agente := "Codex2"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("ordenes inesperadas: %+v", orders)
	}
	if orders[0].Tipo != "handoff" {
		t.Fatalf("tipo inesperado: %s", orders[0].Tipo)
	}

	var payload HandoffPayload
	if err := json.Unmarshal([]byte(orders[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	if payload.AgenteOrigen != "Codex1" || payload.AgenteDestino != "Codex2" {
		t.Fatalf("payload inesperado: %+v", payload)
	}
	if payload.TareaID == nil || *payload.TareaID != tareaID {
		t.Fatalf("tarea en payload inesperada: %+v", payload.TareaID)
	}

	handleOrigen, err = GetRuntimeHandleBySesionID(sesionOrigenID)
	if err != nil {
		t.Fatalf("handle origen tras handoff: %v", err)
	}
	if handleOrigen == nil || handleOrigen.Estado != "pausado" {
		t.Fatalf("el origen deberia quedar pausado tras el handoff: %+v", handleOrigen)
	}
}

func TestCrearHandoffAgenteVivoExigeHandleActivoEnOrigen(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if _, err := IniciarSesion("Codex1"); err != nil {
		t.Fatalf("IniciarSesion origen: %v", err)
	}
	if _, err := IniciarSesion("Codex2"); err != nil {
		t.Fatalf("IniciarSesion destino: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE agente=?`, "Codex1"); err != nil {
		t.Fatalf("cerrar handles origen: %v", err)
	}

	if _, err := CrearHandoffAgenteVivo("Codex1", "Codex2", nil, "", "", ""); err == nil {
		t.Fatalf("se esperaba error por falta de handle activo")
	}
}

func TestCrearHandoffAgenteVivoReutilizaPendienteEquivalente(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if _, err := IniciarSesion("Codex1"); err != nil {
		t.Fatalf("IniciarSesion origen: %v", err)
	}
	if _, err := IniciarSesion("Codex2"); err != nil {
		t.Fatalf("IniciarSesion destino: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:    "Revisar handoff idempotente",
		Modulo:    "orquestador",
		Prioridad: PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}

	orderA, err := CrearHandoffAgenteVivo("Codex1", "Codex2", &tareaID, "presupuesto", "Continuar", "ext-123")
	if err != nil {
		t.Fatalf("CrearHandoffAgenteVivo A: %v", err)
	}
	orderB, err := CrearHandoffAgenteVivo("Codex1", "Codex2", &tareaID, "presupuesto", "Continuar", "ext-123")
	if err != nil {
		t.Fatalf("CrearHandoffAgenteVivo B: %v", err)
	}
	if orderA != orderB {
		t.Fatalf("se esperaba reutilizar la orden pendiente: %d != %d", orderA, orderB)
	}

	agente := "Codex2"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("deberia existir una sola orden pendiente equivalente: %+v", orders)
	}
}

func TestCrearHandoffAgenteVivoRechazaDestinoConGobernanzaIncompatible(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "handoff-gobernanza-manual",
		Nombre:  "handoff-gobernanza-manual",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	reglaID, err := UpsertRegla(&Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "server-first-manual",
		Descripcion: "usar API",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("UpsertRegla: %v", err)
	}
	if _, err := GuardarGovernanceOverride("test", &GovernanceOverride{
		ScopeTipo:  GovernanceScopeAgente,
		ScopeRef:   "Codex2",
		Entidad:    GovernanceEntityRegla,
		EntidadID:  reglaID,
		Accion:     GovernanceActionDisable,
		TipoAgente: "programador",
	}); err != nil {
		t.Fatalf("GuardarGovernanceOverride: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{Agente: "Codex1", ProyectoID: &proyectoID}); err != nil {
		t.Fatalf("IniciarSesionContexto origen: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{Agente: "Codex2", ProyectoID: &proyectoID}); err != nil {
		t.Fatalf("IniciarSesionContexto destino: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:     "Manual incompatible",
		Modulo:     "orquestador",
		Prioridad:  PrioridadAlta,
		CreadoPor:  "alberto",
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}

	if _, err := CrearHandoffAgenteVivo("Codex1", "Codex2", &tareaID, "manual", "continuar", ""); err == nil {
		t.Fatalf("se esperaba error por gobernanza incompatible")
	}
}

func TestCrearHandoffAgenteVivoSincronizaAsignaciones(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "handoff-proyecto-a",
		Nombre:  "handoff-proyecto-a",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "handoff-proyecto-b",
		Nombre:  "handoff-proyecto-b",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto B: %v", err)
	}
	sesionOrigen, err := IniciarSesionContexto(SesionInicio{Agente: "Codex1", ProyectoID: &proyectoA})
	if err != nil {
		t.Fatalf("IniciarSesion origen: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{Agente: "Codex2", ProyectoID: &proyectoB}); err != nil {
		t.Fatalf("IniciarSesion destino: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoA, "frente origen"); err != nil {
		t.Fatalf("ActivarAsignacion origen: %v", err)
	}
	if err := ActivarAsignacion("Codex2", proyectoB, "frente previo destino"); err != nil {
		t.Fatalf("ActivarAsignacion destino: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:     "Relevar frente",
		Modulo:     "orquestador",
		Prioridad:  PrioridadAlta,
		CreadoPor:  "alberto",
		ProyectoID: &proyectoA,
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}

	if _, err := CrearHandoffAgenteVivo("Codex1", "Codex2", &tareaID, "relevo", "continuar", ""); err != nil {
		t.Fatalf("CrearHandoffAgenteVivo: %v", err)
	}

	asignacionActiva, err := GetAsignacionActivaAgente("Codex2")
	if err != nil {
		t.Fatalf("GetAsignacionActivaAgente destino: %v", err)
	}
	if asignacionActiva.ProyectoID != proyectoA {
		t.Fatalf("el destino deberia quedar activo en el proyecto del handoff: %+v", asignacionActiva)
	}

	estadoPausada := AsignacionPausada
	origenPausadas, err := ListarAsignaciones(FiltroAsignaciones{Agente: strPtrHandoffTest("Codex1"), Estado: &estadoPausada})
	if err != nil {
		t.Fatalf("ListarAsignaciones origen pausadas: %v", err)
	}
	foundOrigen := false
	for _, item := range origenPausadas {
		if item != nil && item.ProyectoID == proyectoA {
			foundOrigen = true
			break
		}
	}
	if !foundOrigen {
		t.Fatalf("el origen deberia quedar pausado en el proyecto cedido: %+v", origenPausadas)
	}

	estadoCerrada := AsignacionCerrada
	destinoCerradas, err := ListarAsignaciones(FiltroAsignaciones{Agente: strPtrHandoffTest("Codex2"), Estado: &estadoCerrada})
	if err != nil {
		t.Fatalf("ListarAsignaciones destino cerradas: %v", err)
	}
	foundPrevio := false
	for _, item := range destinoCerradas {
		if item != nil && item.ProyectoID == proyectoB {
			foundPrevio = true
			break
		}
	}
	if !foundPrevio {
		t.Fatalf("el frente previo del destino deberia cerrarse al activar el nuevo: %+v", destinoCerradas)
	}

	handleOrigen, err := GetRuntimeHandleBySesionID(sesionOrigen.ID)
	if err != nil {
		t.Fatalf("GetRuntimeHandleBySesionID origen: %v", err)
	}
	if handleOrigen == nil || handleOrigen.Estado != "pausado" {
		t.Fatalf("el runtime origen deberia quedar pausado: %+v", handleOrigen)
	}
}
