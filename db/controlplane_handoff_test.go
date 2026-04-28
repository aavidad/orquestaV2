package db

import (
	"encoding/json"
	"os"
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

func TestCrearHandoffAgenteStalePermiteSesionAbiertaSinHandleActivo(t *testing.T) {
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
	if _, err := IniciarSesion("Codex2"); err != nil {
		t.Fatalf("IniciarSesion destino: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE agente=?`, "Codex1"); err != nil {
		t.Fatalf("cerrar handles origen: %v", err)
	}
	watchdogID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		Kind:        "watchdog",
		PayloadJSON: `{"texto":"heartbeat obsoleto"}`,
	})
	if err != nil {
		t.Fatalf("crear watchdog pendiente: %v", err)
	}
	if watchdogID == 0 {
		t.Fatalf("watchdog id inesperado: %d", watchdogID)
	}

	orderID, err := CrearHandoffAgenteStale("Codex1", "Codex2", nil, "watchdog", "continuar", "")
	if err != nil {
		t.Fatalf("CrearHandoffAgenteStale sin handle activo: %v", err)
	}
	if orderID == 0 {
		t.Fatalf("order id inesperado: %d", orderID)
	}

	sesionOrigen, err := GetSesionByID(sesionOrigenID)
	if err != nil {
		t.Fatalf("GetSesionByID origen: %v", err)
	}
	if sesionOrigen.Activa || sesionOrigen.Estado != "pausada" {
		t.Fatalf("sesion origen deberia quedar aparcada: %+v", sesionOrigen)
	}

	toAgente := "Codex1"
	estadoPendiente := "pendiente"
	mailboxPendiente, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente: &toAgente,
		Estado:   &estadoPendiente,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	for _, msg := range mailboxPendiente {
		if msg != nil && msg.Kind == "watchdog" {
			t.Fatalf("no deberia quedar watchdog pendiente tras handoff stale: %+v", mailboxPendiente)
		}
	}
	estadoConsumido := "consumido"
	mailboxConsumido, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente: &toAgente,
		Estado:   &estadoConsumido,
	})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	encontradoWatchdog := false
	for _, msg := range mailboxConsumido {
		if msg != nil && msg.ID == watchdogID && msg.Kind == "watchdog" {
			encontradoWatchdog = true
			break
		}
	}
	if !encontradoWatchdog {
		t.Fatalf("watchdog deberia quedar consumido: %+v", mailboxConsumido)
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

func TestCrearHandoffAgenteVivoToleraRuntimeDestinoStaleEnHandle(t *testing.T) {
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
	sesionDestinoID, err := IniciarSesion("Codex2")
	if err != nil {
		t.Fatalf("IniciarSesion destino: %v", err)
	}
	handleDestino, err := GetRuntimeHandleBySesionID(sesionDestinoID)
	if err != nil || handleDestino == nil {
		t.Fatalf("get handle destino: %+v err=%v", handleDestino, err)
	}
	if _, err := DB.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("disable fk: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET runtime_id = ? WHERE id = ?`, int64(999999), handleDestino.ID); err != nil {
		t.Fatalf("inject stale runtime_id: %v", err)
	}
	if _, err := DB.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable fk: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:    "Revisar handoff con runtime stale",
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

	orderID, err := CrearHandoffAgenteVivo("Codex1", "Codex2", &tareaID, "runtime stale", "Continuar", "")
	if err != nil {
		t.Fatalf("CrearHandoffAgenteVivo con runtime stale destino: %v", err)
	}
	if orderID == 0 {
		t.Fatalf("order id inesperado: %d", orderID)
	}
}

func TestCrearHandoffAgenteStaleConDestinoActivoProyectoEncolaReinicioBootstrap(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "handoff-bootstrap",
		Nombre:  "handoff-bootstrap",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{Agente: "Codex1", ProyectoID: &proyectoID}); err != nil {
		t.Fatalf("IniciarSesionContexto origen: %v", err)
	}
	pidDestino := int64(os.Getpid())
	sesionDestino, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		Herramienta: "codex-cli",
		PID:         &pidDestino,
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto destino: %v", err)
	}
	handleDestino, err := GetRuntimeHandleBySesionID(sesionDestino.ID)
	if err != nil || handleDestino == nil {
		t.Fatalf("GetRuntimeHandleBySesionID destino: %+v err=%v", handleDestino, err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET transporte = 'tmux', metadata_json = ?, capabilities_json = ?
		WHERE id = ?`,
		`{"driver":"tmux_cli_session","rendered_command":"`+exe+`","wrapped_command":"`+exe+`","working_dir":"`+cwd+`","tmux_session":"orq-codex2-handoff","tmux_pane_id":"%42","mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`,
		`{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only","can_checkpoint":true,"can_resume":true,"can_capture_pid":true,"can_track_continuity":true,"can_pause":true,"can_stop":true}`,
		handleDestino.ID,
	); err != nil {
		t.Fatalf("actualizar handle destino: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE agente=?`, "Codex1"); err != nil {
		t.Fatalf("cerrar handles origen: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:     "Revisar handoff bootstrap destino vivo",
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

	if _, err := CrearHandoffAgenteStale("Codex1", "Codex2", &tareaID, "watchdog", "continuar", ""); err != nil {
		t.Fatalf("CrearHandoffAgenteStale: %v", err)
	}

	agente := "Codex2"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders: %v", err)
	}
	if len(orders) != 3 {
		t.Fatalf("ordenes destino inesperadas: %+v", orders)
	}
	tipos := map[string]int{}
	for _, order := range orders {
		if order == nil {
			continue
		}
		tipos[order.Tipo]++
	}
	if tipos["handoff"] != 1 || tipos["stop"] != 1 || tipos["start"] != 1 {
		t.Fatalf("tipos de orden destino inesperados: %+v", tipos)
	}
}

func TestCrearHandoffAgenteStaleToleraRuntimeDestinoStale(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "handoff-runtime-stale",
		Nombre:  "handoff-runtime-stale",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{Agente: "Codex1", ProyectoID: &proyectoID}); err != nil {
		t.Fatalf("IniciarSesionContexto origen: %v", err)
	}
	pidDestino := int64(os.Getpid())
	sesionDestino, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		Herramienta: "codex-cli",
		PID:         &pidDestino,
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto destino: %v", err)
	}
	handleDestino, err := GetRuntimeHandleBySesionID(sesionDestino.ID)
	if err != nil || handleDestino == nil {
		t.Fatalf("GetRuntimeHandleBySesionID destino: %+v err=%v", handleDestino, err)
	}
	staleRuntimeID := int64(999999)
	if _, err := DB.Exec(`UPDATE runtime_handles SET runtime_id=? WHERE id=?`, staleRuntimeID, handleDestino.ID); err != nil {
		t.Fatalf("forzar runtime_id stale destino: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE agente=?`, "Codex1"); err != nil {
		t.Fatalf("cerrar handles origen: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:     "Revisar handoff destino stale",
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

	if _, err := CrearHandoffAgenteStale("Codex1", "Codex2", &tareaID, "watchdog", "continuar", ""); err != nil {
		t.Fatalf("CrearHandoffAgenteStale con runtime destino stale: %v", err)
	}

	agente := "Codex2"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders: %v", err)
	}
	if len(orders) != 3 {
		t.Fatalf("ordenes destino inesperadas con runtime stale: %+v", orders)
	}
	tipos := map[string]int{}
	for _, order := range orders {
		if order != nil {
			tipos[order.Tipo]++
		}
	}
	if tipos["handoff"] != 1 || tipos["stop"] != 1 || tipos["start"] != 1 {
		t.Fatalf("tipos de orden destino inesperados con runtime stale: %+v", tipos)
	}
}

func TestCrearHandoffAgenteStaleDestinoSinHandleOperativoEncolaStart(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "handoff-sin-handle",
		Nombre:  "handoff-sin-handle",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{Agente: "Codex1", ProyectoID: &proyectoID}); err != nil {
		t.Fatalf("IniciarSesionContexto origen: %v", err)
	}
	sesionDestino, err := IniciarSesionContexto(SesionInicio{
		Agente:     "Codex2",
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto destino: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE sesion_id=?`, sesionDestino.ID); err != nil {
		t.Fatalf("cerrar handles destino: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE agente=?`, "Codex1"); err != nil {
		t.Fatalf("cerrar handles origen: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:     "Revisar handoff destino sin handle",
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

	if _, err := CrearHandoffAgenteStale("Codex1", "Codex2", &tareaID, "watchdog", "continuar", ""); err != nil {
		t.Fatalf("CrearHandoffAgenteStale destino sin handle operativo: %v", err)
	}

	agente := "Codex2"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("ordenes destino inesperadas sin handle operativo: %+v", orders)
	}
	tipos := map[string]int{}
	for _, order := range orders {
		if order != nil {
			tipos[order.Tipo]++
		}
	}
	if tipos["handoff"] != 1 || tipos["start"] != 1 || tipos["stop"] != 0 {
		t.Fatalf("tipos de orden destino inesperados sin handle operativo: %+v", tipos)
	}
}

func TestProcesarBootstrapRuntimeOrdersActivosBatchEncolaStartParaHandoffSinHandleOperativo(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "handoff-bootstrap-legacy",
		Nombre:  "handoff-bootstrap-legacy",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{Agente: "Codex1", ProyectoID: &proyectoID}); err != nil {
		t.Fatalf("IniciarSesionContexto origen: %v", err)
	}
	sesionDestino, err := IniciarSesionContexto(SesionInicio{
		Agente:     "Codex2",
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto destino: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE sesion_id=?`, sesionDestino.ID); err != nil {
		t.Fatalf("cerrar handles destino: %v", err)
	}
	payloadJSON, err := json.Marshal(HandoffPayload{
		AgenteOrigen:       "Codex1",
		AgenteDestino:      "Codex2",
		Motivo:             "legacy_without_start",
		ResumenContinuidad: "continuar",
	})
	if err != nil {
		t.Fatalf("marshal handoff payload: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		PayloadJSON: string(payloadJSON),
	}); err != nil {
		t.Fatalf("EncolarRuntimeOrder handoff legacy: %v", err)
	}

	processed, err := procesarBootstrapRuntimeOrdersActivosBatch()
	if err != nil {
		t.Fatalf("procesarBootstrapRuntimeOrdersActivosBatch: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed inesperado: %d", processed)
	}

	agente := "Codex2"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders: %v", err)
	}
	tipos := map[string]int{}
	for _, order := range orders {
		if order != nil {
			tipos[order.Tipo]++
		}
	}
	if tipos["handoff"] != 1 || tipos["start"] != 1 {
		t.Fatalf("tipos de orden tras bootstrap legacy inesperados: %+v", tipos)
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
