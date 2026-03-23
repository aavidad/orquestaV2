package db

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestRuntimeHandleSeSincronizaDesdeSesion(t *testing.T) {
	tmp := prepararDBTemporal(t)

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

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-001",
		ResumenContinuidad: "continuidad inicial",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}
	if handle == nil {
		t.Fatalf("runtime handle nil")
	}
	if handle.Estado != "activo" {
		t.Fatalf("estado inesperado: %s", handle.Estado)
	}
	if handle.HandleRef != "sess-001" {
		t.Fatalf("handle_ref inesperado: %s", handle.HandleRef)
	}

	pid := int64(4242)
	branch := "feature/controlplane"
	if err := GuardarSesionActiva("Codex1", &proyectoID, SesionUpdate{
		PID:       &pid,
		Branch:    &branch,
		Heartbeat: true,
	}); err != nil {
		t.Fatalf("guardar sesion: %v", err)
	}

	handle, err = GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("reload handle: %v", err)
	}
	if handle.HandleKind != "process" {
		t.Fatalf("handle_kind inesperado: %s", handle.HandleKind)
	}
	if handle.HandleRef != "4242" {
		t.Fatalf("handle_ref tras pid inesperado: %s", handle.HandleRef)
	}
	if handle.RuntimeID == nil {
		t.Fatalf("runtime_id no sincronizado")
	}

	if err := FinSesion("Codex1"); err != nil {
		t.Fatalf("fin sesion: %v", err)
	}
	handle, err = GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("handle final: %v", err)
	}
	if handle == nil || handle.Estado != "cerrado" {
		t.Fatalf("handle no cerrado: %+v", handle)
	}
}

func TestRuntimeOrdersMailboxYCheckpoint(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
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

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{"motivo":"test"}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}

	claimed, err := ClaimNextRuntimeOrder("Codex1")
	if err != nil {
		t.Fatalf("claim runtime order: %v", err)
	}
	if claimed == nil || claimed.ID != orderID {
		t.Fatalf("runtime order inesperada: %+v", claimed)
	}
	if claimed.Estado != "tomada" {
		t.Fatalf("estado inesperado tras claim: %s", claimed.Estado)
	}
	secondClaim, err := ClaimNextRuntimeOrder("Codex1")
	if err != nil {
		t.Fatalf("segundo claim runtime order: %v", err)
	}
	if secondClaim != nil {
		t.Fatalf("no esperaba segunda order claimable: %+v", secondClaim)
	}

	if err := MarcarRuntimeOrderEstado(orderID, "completada", `{"ok":true}`, ""); err != nil {
		t.Fatalf("completar runtime order: %v", err)
	}
	claimed, err = GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get runtime order: %v", err)
	}
	if claimed.Estado != "completada" || claimed.FinishedAt == nil {
		t.Fatalf("runtime order no completada: %+v", claimed)
	}

	mailID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "Codex1",
		ToAgente:    "Codex2",
		ProyectoID:  &proyectoID,
		Kind:        "consulta",
		PayloadJSON: `{"texto":"revisa esto"}`,
	})
	if err != nil {
		t.Fatalf("enviar mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailID); err != nil {
		t.Fatalf("marcar entregado: %v", err)
	}
	if err := MarcarRuntimeMailboxConsumido(mailID); err != nil {
		t.Fatalf("marcar consumido: %v", err)
	}
	estadoConsumido := "consumido"
	toAgente := "Codex2"
	mensajes, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente: &toAgente,
		Estado:   &estadoConsumido,
	})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mensajes) != 1 || mensajes[0].ID != mailID {
		t.Fatalf("mailbox inesperado: %+v", mensajes)
	}

	cpID, err := CrearRuntimeCheckpoint(&RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		CheckpointKind: "handoff_prepare",
		Resumen:        "checkpoint de prueba",
		Branch:         "main",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"archivos":["a.go"]}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "test",
	})
	if err != nil {
		t.Fatalf("crear runtime checkpoint: %v", err)
	}

	cp, err := UltimoRuntimeCheckpoint("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("ultimo checkpoint: %v", err)
	}
	if cp == nil || cp.ID != cpID {
		t.Fatalf("checkpoint inesperado: %+v", cp)
	}
	if cp.ResumeStrategy != "resumen_y_payload" {
		t.Fatalf("resume strategy inesperada: %s", cp.ResumeStrategy)
	}
}

func TestMarcarRuntimeHandlesCerradosPorAgenteRespetaSesionActivaMasNueva(t *testing.T) {
	tmp := prepararDBTemporal(t)

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

	s1, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 1: %v", err)
	}
	s2, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "feature/nueva",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 2: %v", err)
	}
	if err := MarcarRuntimeHandlesCerradosPorAgente("Codex1"); err != nil {
		t.Fatalf("marcar handles cerrados: %v", err)
	}

	handle1, err := GetRuntimeHandleBySesionID(s1.ID)
	if err != nil {
		t.Fatalf("handle1: %v", err)
	}
	handle2, err := GetRuntimeHandleBySesionID(s2.ID)
	if err != nil {
		t.Fatalf("handle2: %v", err)
	}
	if handle1 == nil || handle1.Estado != "cerrado" {
		t.Fatalf("handle1 no cerrado: %+v", handle1)
	}
	if handle2 == nil || handle2.Estado != "activo" {
		t.Fatalf("handle2 no deberia cerrarse: %+v", handle2)
	}
}

func TestReconciliarRuntimeHandlesStaleMarcaHandleFallido(t *testing.T) {
	tmp := prepararDBTemporal(t)

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
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,'activo',?)`,
		"Codex1", proyectoID, "cli", "process", "pid:1", time.Now().UTC().Add(-10*time.Minute)); err != nil {
		t.Fatalf("insert handle stale: %v", err)
	}

	n, err := ReconciliarRuntimeHandlesStale()
	if err != nil {
		t.Fatalf("reconciliar handles stale: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 handle reconciliado, got=%d", n)
	}

	agente := "Codex1"
	handles, err := ListarRuntimeHandles(&agente)
	if err != nil {
		t.Fatalf("listar handles: %v", err)
	}
	if len(handles) != 1 || handles[0].Estado != "fallido" {
		t.Fatalf("handle no marcado como fallido: %+v", handles)
	}
}

func TestReconciliarRuntimeHandlesStaleMarcaHandleConSesionActivaPeroSinHeartbeat(t *testing.T) {
	tmp := prepararDBTemporal(t)

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
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET last_seen_at = ?
		WHERE sesion_id = ?`, time.Now().UTC().Add(-10*time.Minute), sesion.ID); err != nil {
		t.Fatalf("envejecer handle: %v", err)
	}

	n, err := ReconciliarRuntimeHandlesStale()
	if err != nil {
		t.Fatalf("reconciliar handles stale: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 handle stale aunque la sesion siga abierta, got=%d", n)
	}

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}
	if handle == nil || handle.Estado != "fallido" {
		t.Fatalf("handle no marcado como fallido: %+v", handle)
	}
}

func TestProcesarRuntimeOrdersBatchCompletaSyncStatusYCheckpoint(t *testing.T) {
	tmp := prepararDBTemporal(t)

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
	if _, err := DB.Exec(`UPDATE config SET valor = '1' WHERE clave = 'runtime_order_batch_size'`); err != nil {
		t.Fatalf("config batch size: %v", err)
	}
	_, err = IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-runner-001",
		ResumenContinuidad: "seguir por aqui",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync_status: %v", err)
	}
	checkpointID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{"checkpoint_kind":"handoff_prepare","resumen":"checkpoint desde batch"}`,
	})
	if err != nil {
		t.Fatalf("encolar checkpoint: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar batch 1: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 orden procesada en batch 1, got=%d", n)
	}
	n, err = ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar batch 2: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 orden procesada en batch 2, got=%d", n)
	}

	syncOrder, err := GetRuntimeOrder(syncID)
	if err != nil {
		t.Fatalf("get sync order: %v", err)
	}
	if syncOrder.Estado != "completada" {
		t.Fatalf("sync order no completada: %+v", syncOrder)
	}
	var syncResult map[string]any
	if err := json.Unmarshal([]byte(syncOrder.ResultadoJSON), &syncResult); err != nil {
		t.Fatalf("parse sync result: %v", err)
	}
	if ok, _ := syncResult["ok"].(bool); !ok {
		t.Fatalf("sync result no ok: %+v", syncResult)
	}

	checkpointOrder, err := GetRuntimeOrder(checkpointID)
	if err != nil {
		t.Fatalf("get checkpoint order: %v", err)
	}
	if checkpointOrder.Estado != "completada" {
		t.Fatalf("checkpoint order no completada: %+v", checkpointOrder)
	}
	cp, err := UltimoRuntimeCheckpoint("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("ultimo checkpoint: %v", err)
	}
	if cp == nil || cp.CheckpointKind != "handoff_prepare" || cp.Resumen != "checkpoint desde batch" {
		t.Fatalf("checkpoint inesperado: %+v", cp)
	}
}

func TestReconciliarRuntimeHandlesStaleMarcaHandleActivoConSesionAbierta(t *testing.T) {
	tmp := prepararDBTemporal(t)

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
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-stale-open",
		ResumenContinuidad: "seguir",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET last_seen_at = ?
		WHERE sesion_id = ?`,
		time.Now().UTC().Add(-10*time.Minute), sesion.ID,
	); err != nil {
		t.Fatalf("envejecer handle: %v", err)
	}

	n, err := ReconciliarRuntimeHandlesStale()
	if err != nil {
		t.Fatalf("reconciliar handles stale: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 handle stale reconciliado, got=%d", n)
	}

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}
	if handle == nil || handle.Estado != "fallido" {
		t.Fatalf("handle no marcado como fallido: %+v", handle)
	}
}

func TestReconciliarRuntimeOrdersStaleReencolaBasicasYExpiraNoSoportadas(t *testing.T) {
	tmp := prepararDBTemporal(t)

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

	syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync order: %v", err)
	}
	otherID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar handoff order: %v", err)
	}
	if _, err := ClaimNextRuntimeOrder("Codex1"); err != nil {
		t.Fatalf("claim sync order: %v", err)
	}
	if _, err := ClaimNextRuntimeOrder("Codex1"); err != nil {
		t.Fatalf("claim handoff order: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET started_at = ?, updated_at = ?
		WHERE id IN (?, ?)`,
		time.Now().UTC().Add(-10*time.Minute),
		time.Now().UTC().Add(-10*time.Minute),
		syncID, otherID,
	); err != nil {
		t.Fatalf("envejecer orders: %v", err)
	}

	n, err := ReconciliarRuntimeOrdersStale()
	if err != nil {
		t.Fatalf("reconciliar orders stale: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba 2 orders stale reconciliadas, got=%d", n)
	}

	syncOrder, err := GetRuntimeOrder(syncID)
	if err != nil {
		t.Fatalf("get sync order: %v", err)
	}
	if syncOrder.Estado != "pendiente" || syncOrder.StartedAt != nil {
		t.Fatalf("sync order no reencolada: %+v", syncOrder)
	}

	otherOrder, err := GetRuntimeOrder(otherID)
	if err != nil {
		t.Fatalf("get handoff order: %v", err)
	}
	if otherOrder.Estado != "expirada" || otherOrder.FinishedAt == nil {
		t.Fatalf("handoff order no expirada: %+v", otherOrder)
	}
}

func TestProcesarRuntimeOrdersBatchIgnoraTiposNoSoportadosYResuelveProyectoDeLaOrden(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquesta-a",
		Nombre:  "Orquesta A",
		RutaAbs: filepath.Join(tmp, "orquesta-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquesta-b",
		Nombre:  "Orquesta B",
		RutaAbs: filepath.Join(tmp, "orquesta-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}

	sesionA, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoA,
		CWD:                filepath.Join(tmp, "orquesta-a"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-a",
		ResumenContinuidad: "seguir en A",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion A: %v", err)
	}
	runtimeA, err := GetRuntimeBySesionID(sesionA.ID)
	if err != nil || runtimeA == nil {
		t.Fatalf("runtime A: %+v err=%v", runtimeA, err)
	}

	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoB,
		CWD:                filepath.Join(tmp, "orquesta-b"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-b",
		ResumenContinuidad: "seguir en B",
		Branch:             "feature/b",
	}); err != nil {
		t.Fatalf("iniciar sesion B: %v", err)
	}

	unsupportedID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoA,
		Tipo:        "handoff",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar unsupported: %v", err)
	}
	syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoA,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 order procesada, got=%d", n)
	}

	unsupportedOrder, err := GetRuntimeOrder(unsupportedID)
	if err != nil {
		t.Fatalf("get unsupported order: %v", err)
	}
	if unsupportedOrder.Estado != "pendiente" {
		t.Fatalf("la order no soportada no deberia reclamarse: %+v", unsupportedOrder)
	}

	syncOrder, err := GetRuntimeOrder(syncID)
	if err != nil {
		t.Fatalf("get sync order: %v", err)
	}
	if syncOrder.Estado != "completada" {
		t.Fatalf("sync order no completada: %+v", syncOrder)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(syncOrder.ResultadoJSON), &result); err != nil {
		t.Fatalf("parse sync result: %v", err)
	}
	if sinHandle, _ := result["sin_handle"].(bool); !sinHandle {
		t.Fatalf("esperaba sin_handle=true por cambio de proyecto: %+v", result)
	}
	runtimeID, ok := result["runtime_id"].(float64)
	if !ok || int64(runtimeID) != runtimeA.ID {
		t.Fatalf("runtime resuelto incorrecto: %+v, runtimeA=%d", result, runtimeA.ID)
	}
}

func TestEjecutarRuntimeOrderCheckpointReutilizaCheckpointExistente(t *testing.T) {
	tmp := prepararDBTemporal(t)

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
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-checkpoint",
		ResumenContinuidad: "seguir por aqui",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("runtime por sesion: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{"checkpoint_kind":"handoff_prepare","resumen":"checkpoint reintentado"}`,
	})
	if err != nil {
		t.Fatalf("encolar checkpoint: %v", err)
	}
	existenteID, err := CrearRuntimeCheckpoint(&RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		SesionID:       &sesion.ID,
		RuntimeID:      &runtime.ID,
		CheckpointKind: "handoff_prepare",
		Resumen:        "checkpoint reintentado",
		Branch:         "main",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"checkpoint_kind":"handoff_prepare","resumen":"checkpoint reintentado"}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "runtime_order:" + jsonNumber(orderID),
	})
	if err != nil {
		t.Fatalf("crear checkpoint existente: %v", err)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if err := ejecutarRuntimeOrderCheckpoint(order); err != nil {
		t.Fatalf("ejecutar checkpoint: %v", err)
	}

	cp, err := GetRuntimeCheckpointBySource("runtime_order:" + jsonNumber(orderID))
	if err != nil {
		t.Fatalf("get checkpoint por source: %v", err)
	}
	if cp == nil || cp.ID != existenteID {
		t.Fatalf("checkpoint reutilizado incorrecto: %+v", cp)
	}

	order, err = GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if order.Estado != "completada" {
		t.Fatalf("order no completada: %+v", order)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(order.ResultadoJSON), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	checkpointID, ok := result["checkpoint_id"].(float64)
	if !ok || int64(checkpointID) != existenteID {
		t.Fatalf("resultado no reutiliza checkpoint existente: %+v", result)
	}
}

func TestProcesarRuntimeOrdersBatchCompletaNudgeEnMailbox(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
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

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		Tipo:        "nudge",
		PayloadJSON: `{"from_agente":"Codex1","to_agente":"Codex2","kind":"nudge","texto":"retoma la tarea y desbloquea el siguiente paso"}`,
	})
	if err != nil {
		t.Fatalf("encolar nudge: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("processed inesperado: %d", n)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order == nil || order.Estado != "completada" {
		t.Fatalf("order no completada: %+v", order)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(order.ResultadoJSON), &result); err != nil {
		t.Fatalf("parse resultado nudge: %v", err)
	}
	mailboxID, ok := result["mailbox_id"].(float64)
	if !ok || int64(mailboxID) <= 0 {
		t.Fatalf("resultado nudge sin mailbox_id: %+v", result)
	}

	toAgente := "Codex2"
	estado := "pendiente"
	mailbox, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente: &toAgente,
		Estado:   &estado,
	})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mailbox) != 1 {
		t.Fatalf("mailbox inesperado: %+v", mailbox)
	}
	if mailbox[0].RuntimeOrderID == nil || *mailbox[0].RuntimeOrderID != orderID {
		t.Fatalf("runtime_order_id inesperado en mailbox: %+v", mailbox[0])
	}
	if mailbox[0].FromAgente != "Codex1" || mailbox[0].Kind != "nudge" {
		t.Fatalf("mailbox nudge inesperado: %+v", mailbox[0])
	}
}

func TestEjecutarRuntimeOrderNudgeReutilizaMailboxExistente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
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

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		Tipo:        "nudge",
		PayloadJSON: `{"from_agente":"Codex1","to_agente":"Codex2","texto":"reintento nudge"}`,
	})
	if err != nil {
		t.Fatalf("encolar nudge: %v", err)
	}

	runtimeOrderID := orderID
	existenteID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:     "Codex1",
		ToAgente:       "Codex2",
		ProyectoID:     &proyectoID,
		RuntimeOrderID: &runtimeOrderID,
		Kind:           "nudge",
		PayloadJSON:    `{"from_agente":"Codex1","to_agente":"Codex2","texto":"reintento nudge"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox existente: %v", err)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if err := ejecutarRuntimeOrderNudge(order); err != nil {
		t.Fatalf("ejecutar nudge: %v", err)
	}

	reutilizado, err := GetRuntimeMailboxByRuntimeOrderID(orderID)
	if err != nil {
		t.Fatalf("get mailbox por runtime order: %v", err)
	}
	if reutilizado == nil || reutilizado.ID != existenteID {
		t.Fatalf("mailbox reutilizado incorrecto: %+v", reutilizado)
	}

	order, err = GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if order.Estado != "completada" {
		t.Fatalf("order no completada: %+v", order)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(order.ResultadoJSON), &result); err != nil {
		t.Fatalf("parse resultado nudge: %v", err)
	}
	if reused, _ := result["reused"].(bool); !reused {
		t.Fatalf("resultado nudge deberia marcar reused: %+v", result)
	}
	mailboxID, ok := result["mailbox_id"].(float64)
	if !ok || int64(mailboxID) != existenteID {
		t.Fatalf("resultado nudge no reutiliza mailbox existente: %+v", result)
	}
}

func TestReconciliarRuntimeOrdersStaleRecuperaBasicasYExpiraNoSoportadas(t *testing.T) {
	tmp := prepararDBTemporal(t)

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
	if _, err := DB.Exec(`UPDATE config SET valor = '1' WHERE clave = 'runtime_order_stale_seconds'`); err != nil {
		t.Fatalf("config stale seconds: %v", err)
	}

	basicID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar basica: %v", err)
	}
	customID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "custom_dispatch",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar custom: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(basicID, "ejecutando", `{}`, ""); err != nil {
		t.Fatalf("marcar ejecutando basica: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(customID, "tomada", `{}`, ""); err != nil {
		t.Fatalf("marcar tomada custom: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET started_at = ?, updated_at = ?
		WHERE id IN (?, ?)`,
		time.Now().UTC().Add(-10*time.Minute),
		time.Now().UTC().Add(-10*time.Minute),
		basicID, customID,
	); err != nil {
		t.Fatalf("envejecer orders: %v", err)
	}

	n, err := ReconciliarRuntimeOrdersStale()
	if err != nil {
		t.Fatalf("reconciliar runtime orders stale: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba recuperar/expirar 2 ordenes, got=%d", n)
	}

	basicOrder, err := GetRuntimeOrder(basicID)
	if err != nil {
		t.Fatalf("get basic order: %v", err)
	}
	if basicOrder.Estado != "pendiente" || basicOrder.StartedAt != nil {
		t.Fatalf("orden basica no reencolada: %+v", basicOrder)
	}

	customOrder, err := GetRuntimeOrder(customID)
	if err != nil {
		t.Fatalf("get custom order: %v", err)
	}
	if customOrder.Estado != "expirada" || customOrder.FinishedAt == nil {
		t.Fatalf("orden no soportada no expirada: %+v", customOrder)
	}
}

func TestProcesarRuntimeOrdersBatchDejaPendienteOrdenNoSoportada(t *testing.T) {
	tmp := prepararDBTemporal(t)

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
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-a",
		ResumenContinuidad: "seguir",
		Branch:             "main",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	customID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "custom_dispatch",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar custom: %v", err)
	}
	syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba procesar solo la orden soportada, got=%d", n)
	}

	customOrder, err := GetRuntimeOrder(customID)
	if err != nil {
		t.Fatalf("get custom order: %v", err)
	}
	if customOrder.Estado != "pendiente" {
		t.Fatalf("la orden no soportada no deberia tocarse: %+v", customOrder)
	}

	syncOrder, err := GetRuntimeOrder(syncID)
	if err != nil {
		t.Fatalf("get sync order: %v", err)
	}
	if syncOrder.Estado != "completada" {
		t.Fatalf("la orden soportada deberia completarse: %+v", syncOrder)
	}
}

func TestProcesarRuntimeOrdersBatchResuelveRuntimePorProyecto(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}

	sesionA, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoA,
		CWD:                filepath.Join(tmp, "orquestador-a"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-a",
		ResumenContinuidad: "continuidad A",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion A: %v", err)
	}
	runtimeA, err := GetRuntimeBySesionID(sesionA.ID)
	if err != nil || runtimeA == nil {
		t.Fatalf("runtime A no disponible: runtime=%+v err=%v", runtimeA, err)
	}

	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoB,
		CWD:                filepath.Join(tmp, "orquestador-b"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-b",
		ResumenContinuidad: "continuidad B",
		Branch:             "main",
	}); err != nil {
		t.Fatalf("iniciar sesion B: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoA,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 orden procesada, got=%d", n)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(order.ResultadoJSON), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	gotRuntimeID, ok := result["runtime_id"].(float64)
	if !ok {
		t.Fatalf("runtime_id ausente en resultado: %+v", result)
	}
	if int64(gotRuntimeID) != runtimeA.ID {
		t.Fatalf("runtime incorrecto; esperado proyecto A=%d, got=%v", runtimeA.ID, gotRuntimeID)
	}
}
