/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"orquesta/db"
	"orquesta/gitgobernanza"
	"orquesta/notificaciones"
	"orquesta/planocontrol"
	"orquesta/reviewapp"
	"orquesta/runtimeagente"
	"orquesta/tareasapp"
)

type dbAutomationService struct{}

const autonomiaReplanTaskTitle = "Autonomía: replanificar backlog y abrir siguiente frente útil"

func (dbAutomationService) CheckReanimaciones() ([]*db.Agente, error) {
	return db.CheckReanimaciones()
}

func (dbAutomationService) ResetReanimacion(nombre string) error {
	nombre = strings.TrimSpace(nombre)
	if err := reactivarAgenteTrasReanimacion(nombre); err != nil {
		return err
	}
	return db.ResetReanimacion(nombre)
}

func (dbAutomationService) GarantizarSaludAgentes() error {
	return db.GarantizarSaludAgentes()
}

func (dbAutomationService) PlanificarTareasAutomaticamente() error {
	return db.PlanificarTareasAutomaticamente()
}

func (dbAutomationService) ProcesarAutonomiaAgentesBatch() (int, error) {
	return procesarAutonomiaAgentesBatch()
}

func (dbAutomationService) ProcesarRuntimeSupervisionBatch() (int, error) {
	return db.ProcesarRuntimeSupervisionBatch()
}

func (dbAutomationService) ReconciliarRuntimeHandlesStale() (int, error) {
	return db.ReconciliarRuntimeHandlesStale()
}

func (dbAutomationService) ReconciliarRuntimeOrdersStale() (int, error) {
	return db.ReconciliarRuntimeOrdersStale()
}

func (dbAutomationService) ProcesarRuntimeTranscriptBatch() (int, error) {
	return procesarRuntimeTranscriptBatch()
}

func (dbAutomationService) ProcesarRuntimeMailboxBatch() (int, error) {
	return procesarRuntimeMailboxBatch()
}

func (dbAutomationService) ProcesarRuntimeOrdersBatch() (int, error) {
	return db.ProcesarRuntimeOrdersBatch()
}

func (dbAutomationService) ProcesarRefineriaBatch() (int, error) {
	return db.ProcesarRefineriaBatch()
}

func (dbAutomationService) ProcesarHandoffsBatch() (int, error) {
	return db.ProcesarHandoffsBatch()
}

func (dbAutomationService) Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	db.Audit(agente, accion, entidad, entidadID, detalle)
}

func newControlPlaneRunner(debugLogger *log.Logger, debugControlPlane bool) *planocontrol.Runner {
	runner := &planocontrol.Runner{
		Automation:        dbAutomationService{},
		NotificationFeed:  db.CanalNotificaciones,
		InitNotifications: notificaciones.InicializarDesdeConfig,
		Notifier: func() notificaciones.Notificador {
			return notificaciones.GlobalNotificador
		},
	}
	if debugControlPlane && debugLogger != nil {
		runner.Debugf = debugLogger.Printf
	}
	return runner
}

func procesarRuntimeTranscriptBatch() (int, error) {
	ingested, err := db.IngestarRuntimeTranscriptActivos()
	if err != nil {
		return ingested, err
	}
	if !controlPlaneConfigBoolOrDefault("runtime_transcript_auto_guidance_enabled", true) {
		return ingested, nil
	}
	signals, err := db.ListarRuntimeTranscript(db.FiltroRuntimeTranscript{
		SoloSenalesPend: true,
		Limit:           50,
	})
	if err != nil {
		return ingested, err
	}
	processedSignals := 0
	for i := len(signals) - 1; i >= 0; i-- {
		item := signals[i]
		if item == nil || strings.TrimSpace(item.Classification) == "" {
			continue
		}
		note, err := procesarSignalTranscript(item)
		if err != nil {
			return ingested + processedSignals, err
		}
		if strings.TrimSpace(note) != "" {
			if err := db.MarcarRuntimeTranscriptManejado(item.ID, note); err != nil {
				return ingested + processedSignals, err
			}
		}
		processedSignals++
	}
	return ingested + processedSignals, nil
}

func procesarRuntimeMailboxBatch() (int, error) {
	reconciled, err := reconciliarRuntimeMailboxAgenteSinVidaBatch()
	if err != nil {
		return reconciled, err
	}
	watchdogReconciled, err := reconciliarRuntimeMailboxWatchdogSinHandleBatch()
	if err != nil {
		return reconciled + watchdogReconciled, err
	}
	reconciled += watchdogReconciled
	interactive, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		return reconciled + interactive, err
	}
	supervisorLocal, err := procesarRuntimeMailboxSupervisorLocalBatch()
	if err != nil {
		return reconciled + interactive + supervisorLocal, err
	}
	sessionResume, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		return reconciled + interactive + supervisorLocal + sessionResume, err
	}
	restarts, err := procesarRuntimeMailboxCoordinatedRestartBatch()
	if err != nil {
		return reconciled + interactive + supervisorLocal + sessionResume + restarts, err
	}
	return reconciled + interactive + supervisorLocal + sessionResume + restarts, nil
}

func reconciliarRuntimeMailboxAgenteSinVidaBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	total := 0
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		debeConsumirse, detalle, err := runtimeMailboxDebeConsumirsePorAgenteSinVida(msg)
		if err != nil {
			return total, err
		}
		if !debeConsumirse {
			continue
		}
		if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return total, err
		}
		if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return total, err
		}
		db.Audit("orquesta", "runtime_mailbox_agente_sin_vida", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s kind=%s %s", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), detalle))
		total++
	}
	return total, nil
}

func reconciliarRuntimeMailboxWatchdogSinHandleBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	total := 0
	for _, msg := range mailbox {
		if msg == nil || strings.TrimSpace(msg.Kind) != "watchdog" {
			continue
		}
		handle, err := runtimesService.GetActiveRuntimeHandleAgentProject(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return total, err
		}
		if watchdogPuedeConsumirsePorEnfriamiento(msg, handle) {
			if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
				return total, err
			}
			if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
				return total, err
			}
			db.Audit("orquesta", "runtime_mailbox_watchdog_enfriamiento", "runtime_mailbox", msg.ID,
				fmt.Sprintf("agente=%s proyecto_id=%s watchdog consumido por agente en enfriamiento",
					strings.TrimSpace(msg.ToAgente), runtimeMailboxProyectoDetalle(msg.ProyectoID)))
			total++
			continue
		}
		if handle != nil {
			continue
		}
		if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return total, err
		}
		if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return total, err
		}
		db.Audit("orquesta", "runtime_mailbox_watchdog_sin_handle", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s proyecto_id=%s watchdog consumido por ausencia de handle activo",
				strings.TrimSpace(msg.ToAgente), runtimeMailboxProyectoDetalle(msg.ProyectoID)))
		total++
	}
	return total, nil
}

func watchdogPuedeConsumirsePorEnfriamiento(msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle) bool {
	if msg == nil || handle == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
		return false
	}
	agente, err := db.GetAgente(strings.TrimSpace(msg.ToAgente))
	if err != nil || agente == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(agente.EstadoCuota), "enfriamiento")
}

func runtimeMailboxProyectoDetalle(proyectoID *int64) string {
	if proyectoID == nil || *proyectoID <= 0 {
		return "-"
	}
	return strconv.FormatInt(*proyectoID, 10)
}

func runtimeMailboxDebeConsumirsePorAgenteSinVida(msg *db.RuntimeMailboxMessage) (bool, string, error) {
	if msg == nil {
		return false, "", nil
	}
	agente := strings.TrimSpace(msg.ToAgente)
	if agente == "" {
		return false, "", nil
	}
	if strings.EqualFold(strings.TrimSpace(msg.Kind), "watchdog") {
		return false, "", nil
	}
	handle, err := runtimesService.GetActiveRuntimeHandleAgentProject(agente, msg.ProyectoID)
	if err != nil {
		return false, "", err
	}
	if handle != nil {
		return false, "", nil
	}
	sesiones, err := db.ListarSesionesActivasOperativas()
	if err != nil {
		return false, "", err
	}
	for _, sesion := range sesiones {
		if sesion == nil || strings.TrimSpace(sesion.Agente) != agente {
			continue
		}
		if msg.ProyectoID == nil || sesion.ProyectoID == nil || *sesion.ProyectoID == *msg.ProyectoID {
			return false, "", nil
		}
	}
	proyectoActivoID, err := db.ObtenerProyectoActivoAgente(agente)
	if err != nil {
		return false, "", err
	}
	if proyectoActivoID > 0 && agentePerteneceAFlotaAutobootstrap(agente) {
		return false, "", nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{Agente: &agente})
	if err != nil {
		return false, "", err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaAsignada, db.TareaEnProgreso, db.TareaBloqueada:
			return false, "", nil
		}
	}
	if abierta, err := existeRuntimeOrderAbiertaAgente(agente, msg.ProyectoID); err != nil {
		return false, "", err
	} else if abierta {
		return false, "", nil
	}
	return true, fmt.Sprintf("proyecto_id=%s sin runtime, sesion, asignacion, tareas ni ordenes abiertas", runtimeMailboxProyectoDetalle(msg.ProyectoID)), nil
}

func agentePerteneceAFlotaAutobootstrap(agente string) bool {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false
	}
	cfg := loadServerAutobootstrapConfig()
	if strings.EqualFold(agente, strings.TrimSpace(cfg.SupervisorAgent)) {
		return true
	}
	for _, worker := range cfg.WorkerAgents {
		if strings.EqualFold(agente, strings.TrimSpace(worker)) {
			return true
		}
	}
	return false
}

func enfriarAgentePorRuntimePanic(agente string, item *db.RuntimeTranscriptEntry) (string, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return "", nil
	}
	notes := make([]string, 0, 2)
	if item != nil && item.HandleID != nil && *item.HandleID > 0 {
		handle, err := db.GetRuntimeHandle(*item.HandleID)
		if err != nil {
			return "", err
		}
		if handle != nil {
			degradado, err := db.DeshabilitarEntregaCalienteSupervisadaHandle(handle, "runtime_panic", item.ID)
			if err != nil {
				return "", err
			}
			if degradado != nil && degradado.ID > 0 && strings.Contains(degradado.MetadataJSON, `"disable_supervisor_hot_input":true`) {
				notes = append(notes, "supervisor_hot_input_disabled")
			}
		}
	}
	infoAgente, err := db.GetAgente(agente)
	if err != nil {
		return "", err
	}
	cooldownSeconds := controlPlaneConfigIntOrDefault("runtime_panic_cooldown_seconds", 180)
	if cooldownSeconds <= 0 {
		cooldownSeconds = 180
	}
	reanimarAt := time.Now().UTC().Add(time.Duration(cooldownSeconds) * time.Second)
	if infoAgente != nil &&
		strings.EqualFold(strings.TrimSpace(infoAgente.EstadoCuota), "enfriamiento") &&
		infoAgente.ReanimarAt != nil &&
		infoAgente.ReanimarAt.After(reanimarAt) {
		notes = append(notes, "runtime_panic_cooldown_exists")
		return strings.Join(notes, ";"), nil
	}
	motivo := "Auto-pausa por runtime_panic"
	if item != nil && item.ID > 0 {
		motivo = fmt.Sprintf("Auto-pausa por runtime_panic: transcript=%d", item.ID)
	}
	if err := db.PausarAgenteHasta(agente, reanimarAt, motivo); err != nil {
		return "", err
	}
	db.Audit("orquesta", "runtime_panic_cooldown", "agente", 0,
		fmt.Sprintf("agente=%s cooldown_until=%s motivo=%s", agente, reanimarAt.Format(time.RFC3339), motivo))
	notes = append(notes, "runtime_panic_cooldown")
	return strings.Join(notes, ";"), nil
}

func procesarSignalTranscript(item *db.RuntimeTranscriptEntry) (string, error) {
	if item == nil {
		return "", nil
	}
	agente := strings.TrimSpace(item.Agente)
	if agente == "" {
		return "signal_sin_agente", nil
	}
	if handle, err := runtimesService.GetActiveRuntimeHandleAgentProject(agente, item.ProyectoID); err != nil {
		return "", err
	} else if handle == nil {
		return "sin_runtime_activo", nil
	}
	if note, handled, err := resolverReviewGateDesdeSignalTranscript(item); err != nil {
		return "", err
	} else if handled {
		return note, nil
	}
	if esSignalFalloRuntime(item) {
		notes := make([]string, 0, 3)
		if cooldownNote, err := enfriarAgentePorRuntimePanic(agente, item); err != nil {
			return "", err
		} else if strings.TrimSpace(cooldownNote) != "" {
			notes = append(notes, strings.TrimSpace(cooldownNote))
		}
		if supervisorNote, err := notificarSupervisorSignalTranscript(item); err != nil {
			return "", err
		} else if strings.TrimSpace(supervisorNote) != "" {
			notes = append(notes, strings.TrimSpace(supervisorNote))
		}
		notes = append(notes, "runtime_failure_signal")
		return strings.Join(notes, ";"), nil
	}
	payload := map[string]any{
		"to_agente":      agente,
		"from_agente":    "orquesta",
		"texto":          construirRespuestaSignalTranscript(item),
		"classification": strings.TrimSpace(item.Classification),
		"transcript_id":  item.ID,
		"kind":           "instruction",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	order := &db.RuntimeOrder{
		Agente:      agente,
		ProyectoID:  item.ProyectoID,
		RuntimeID:   &item.RuntimeID,
		Tipo:        "send_instruction",
		PayloadJSON: string(raw),
	}
	if item.HandleID != nil && *item.HandleID > 0 {
		order.HandleID = item.HandleID
	}
	orderID, err := db.EncolarRuntimeOrder(order)
	if err != nil {
		return "", err
	}
	detail := fmt.Sprintf("transcript_id=%d agente=%s signal=%s", item.ID, agente, strings.TrimSpace(item.Classification))
	db.Audit("orquesta", "runtime_transcript_signal", "runtime_order", orderID, detail)
	payloadEvent, _ := json.Marshal(construirPayloadEventoAutoGuidance(item, orderID, order.Tipo, payload))
	eventPayload := construirPayloadEventoAutoGuidance(item, orderID, order.Tipo, payload)
	_, _ = db.RegistrarRuntimeEvent(&db.RuntimeEvent{
		RuntimeID:   item.RuntimeID,
		Kind:        "auto_guidance_sent",
		Level:       "info",
		Message:     fmt.Sprintf("Orquesta envió guía automática a %s tras %s", agente, strings.TrimSpace(item.Classification)),
		PayloadJSON: string(payloadEvent),
	})
	select {
	case db.CanalNotificaciones <- db.EventoNotificacion{
		Tipo:       "runtime_auto_guidance",
		ID:         orderID,
		Agente:     agente,
		Texto:      construirMensajeNotificacionAutoGuidance(item),
		ProyectoID: int64ProyectoRuntimeTranscript(item),
		Payload:    eventPayload,
	}:
	default:
	}
	notes := []string{fmt.Sprintf("auto_guidance_order:%d", orderID)}
	if supervisorNote, err := notificarSupervisorSignalTranscript(item); err != nil {
		return "", err
	} else if strings.TrimSpace(supervisorNote) != "" {
		notes = append(notes, strings.TrimSpace(supervisorNote))
	}
	return strings.Join(notes, ";"), nil
}

func construirPayloadEventoAutoGuidance(item *db.RuntimeTranscriptEntry, orderID int64, orderType string, instruction map[string]any) map[string]any {
	if item == nil {
		payload := map[string]any{
			"runtime_order_id":   orderID,
			"runtime_order_type": strings.TrimSpace(orderType),
		}
		if len(instruction) > 0 {
			payload["instruction"] = instruction
		}
		return payload
	}
	payload := map[string]any{
		"agente":             strings.TrimSpace(item.Agente),
		"classification":     strings.TrimSpace(item.Classification),
		"runtime_id":         item.RuntimeID,
		"runtime_order_id":   orderID,
		"runtime_order_type": strings.TrimSpace(orderType),
		"signal_text":        strings.TrimSpace(item.Text),
		"transcript_id":      item.ID,
	}
	if item.ProyectoID != nil && *item.ProyectoID > 0 {
		payload["proyecto_id"] = *item.ProyectoID
	}
	if item.HandleID != nil && *item.HandleID > 0 {
		payload["handle_id"] = *item.HandleID
	}
	if len(instruction) > 0 {
		payload["instruction"] = instruction
	}
	return payload
}

func construirMensajeNotificacionAutoGuidance(item *db.RuntimeTranscriptEntry) string {
	if item == nil {
		return "Orquesta envió guía automática a un agente"
	}
	agente := strings.TrimSpace(item.Agente)
	classification := strings.TrimSpace(item.Classification)
	if agente == "" {
		agente = "agente-desconocido"
	}
	texto := strings.TrimSpace(item.Text)
	if texto == "" {
		return fmt.Sprintf("Orquesta envió guía automática a %s tras %s", agente, classification)
	}
	return fmt.Sprintf("Orquesta envió guía automática a %s tras %s: %s", agente, classification, texto)
}

func int64ProyectoRuntimeTranscript(item *db.RuntimeTranscriptEntry) int64 {
	if item == nil || item.ProyectoID == nil {
		return 0
	}
	return *item.ProyectoID
}

func resolverReviewGateDesdeSignalTranscript(item *db.RuntimeTranscriptEntry) (string, bool, error) {
	if item == nil || item.ProyectoID == nil || *item.ProyectoID <= 0 {
		return "", false, nil
	}
	var estado string
	switch strings.TrimSpace(item.Classification) {
	case "review_approved":
		estado = reviewapp.GateStateApproved
	case "review_changes_requested":
		estado = reviewapp.GateStateChangesAsked
	case "review_blocked":
		estado = reviewapp.GateStateBlocked
	default:
		return "", false, nil
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(*item.ProyectoID, 10))
	if err != nil || proyecto == nil {
		return "", false, err
	}
	gates, err := reviewService.List(reviewapp.ListInput{
		ProyectoRef:    proyecto.Slug,
		ReviewerAgente: strings.TrimSpace(item.Agente),
		Limit:          20,
	})
	if err != nil {
		return "", false, err
	}
	gate := firstOpenGate(gates)
	if gate == nil {
		return "", false, nil
	}
	findingsJSON, err := construirFindingsReviewTranscript(item)
	if err != nil {
		return "", false, err
	}
	resolved, err := reviewService.Resolve(reviewapp.ResolveGateInput{
		ID:             gate.ID,
		Estado:         estado,
		ReviewerAgente: strings.TrimSpace(item.Agente),
		FindingsJSON:   findingsJSON,
	})
	if err != nil {
		return "", false, err
	}
	detail := fmt.Sprintf("gate=%d transcript=%d estado=%s", resolved.ID, item.ID, strings.TrimSpace(estado))
	db.Audit("orquesta", "runtime_transcript_review_resolution", "review_gate", resolved.ID, detail)
	return "review_gate_resolved:" + detail, true, nil
}

func construirFindingsReviewTranscript(item *db.RuntimeTranscriptEntry) (string, error) {
	payload := []map[string]any{{
		"source":         "runtime_transcript",
		"transcript_id":  item.ID,
		"classification": strings.TrimSpace(item.Classification),
		"agente":         strings.TrimSpace(item.Agente),
		"text":           strings.TrimSpace(item.Text),
		"runtime_id":     item.RuntimeID,
		"handle_id":      item.HandleID,
	}}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func esSignalFalloRuntime(item *db.RuntimeTranscriptEntry) bool {
	if item == nil {
		return false
	}
	switch strings.TrimSpace(item.Classification) {
	case "runtime_panic", "runtime_crash":
		return true
	default:
		return false
	}
}

func construirRespuestaSignalTranscript(item *db.RuntimeTranscriptEntry) string {
	base := "Orquesta: continua de forma autonoma dentro de la gobernanza efectiva del proyecto. Usa API/servicios para el flujo normal, no BD local salvo diagnostico o recuperacion, y mantén tests coherentes y eficientes. Si el cambio es delicado, crea checkpoint y sigue."
	normalized := normalizarSignalTranscriptTexto(item)
	politicaPermisos := construirPoliticaPermisosSignalTranscript(strings.TrimSpace(item.Classification), normalized)
	switch strings.TrimSpace(item.Classification) {
	case "approval_request":
		return base + " " + politicaPermisos + " Si dudas entre varias opciones seguras, elige la mas alineada con el proyecto y continua sin detenerte."
	case "waiting_human":
		return base + " " + politicaPermisos + " No te quedes esperando respuesta: formula el siguiente paso razonable, ejecuta y documenta los supuestos."
	case "ready_for_review":
		return base + " Deja un resumen breve, asegurate de que el frente queda verificable y sigue disponible para que Orquesta relance review si procede."
	case "needs_replan":
		return base + " Si no ves el siguiente paso, revisa backlog, tareas, propuestas y ultimo checkpoint, y continua por el frente mas util disponible."
	case "blocked":
		return base + " Si el bloqueo es de contexto, reevalua tareas, propuestas y estado del proyecto y avanza por el mejor siguiente paso disponible."
	default:
		return base
	}
}

func normalizarSignalTranscriptTexto(item *db.RuntimeTranscriptEntry) string {
	if item == nil {
		return ""
	}
	texto := strings.TrimSpace(item.NormalizedText)
	if texto == "" {
		texto = strings.TrimSpace(strings.ToLower(item.Text))
	}
	texto = strings.ReplaceAll(texto, "\n", " ")
	texto = strings.ReplaceAll(texto, "\r", " ")
	return strings.TrimSpace(texto)
}

func construirPoliticaPermisosSignalTranscript(classification, normalized string) string {
	if classification != "approval_request" && classification != "waiting_human" {
		return ""
	}
	if textoPareceAccionDestructiva(normalized) {
		return "No autorizado automaticamente para acciones destructivas o irreversibles. No uses rm -rf, git reset --hard, borrados masivos, drops de base de datos ni cambios fuera del workspace. Replantea una alternativa segura."
	}
	if textoPareceDependenciaExterna(normalized) {
		return "No inventes credenciales, secretos ni permisos externos. Si de verdad faltan, deja trazabilidad del bloqueo y continua por otro frente util del proyecto."
	}
	if textoPareceAccionNormalAutonoma(normalized) {
		return "Aprobado automaticamente para acciones normales dentro del workspace: editar codigo, refactorizar, crear o ajustar archivos, ejecutar build/test/lint, preparar worktrees y continuar el siguiente paso util. No necesitas confirmacion humana para seguir. " + detalleAccionNormalAutonoma(normalized)
	}
	return "Aprobado automaticamente para trabajo normal dentro del proyecto. No necesitas confirmacion humana para seguir por una opcion segura."
}

func textoPareceAccionNormalAutonoma(normalized string) bool {
	for _, token := range []string{
		"refactor",
		"seguir",
		"continuar",
		"go test",
		"npm test",
		"pnpm test",
		"yarn test",
		"pytest",
		"cargo test",
		"build",
		"lint",
		"gofmt",
		"formate",
		"crear archivo",
		"editar",
		"ajustar",
		"implementar",
		"worktree",
		"rama",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func detalleAccionNormalAutonoma(normalized string) string {
	switch {
	case strings.Contains(normalized, "go test"):
		return "Puedes ejecutar go test y seguir con el ciclo normal."
	case strings.Contains(normalized, "pytest"):
		return "Puedes ejecutar pytest y seguir con el ciclo normal."
	case strings.Contains(normalized, "npm test"), strings.Contains(normalized, "pnpm test"), strings.Contains(normalized, "yarn test"):
		return "Puedes ejecutar los tests del frontend y seguir con el ciclo normal."
	case strings.Contains(normalized, "build"):
		return "Puedes ejecutar el build y seguir con el ciclo normal."
	case strings.Contains(normalized, "lint"):
		return "Puedes ejecutar lint y seguir con el ciclo normal."
	case strings.Contains(normalized, "refactor"):
		return "Puedes continuar con el refactor sin esperar confirmacion humana."
	default:
		return "Sigue por la opcion mas segura y util."
	}
}

func textoPareceAccionDestructiva(normalized string) bool {
	for _, token := range []string{
		"rm -rf",
		"rm -f",
		"git reset --hard",
		"git checkout --",
		"drop table",
		"drop database",
		"truncate table",
		"borrar la base de datos",
		"borrar la bd",
		"delete database",
		"wipe",
		"formatear disco",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func textoPareceDependenciaExterna(normalized string) bool {
	for _, token := range []string{
		"credencial",
		"credenciales",
		"secret",
		"secreto",
		"api key",
		"token",
		"ssh key",
		"oauth",
		"permiso externo",
		"acceso externo",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func notificarSupervisorSignalTranscript(item *db.RuntimeTranscriptEntry) (string, error) {
	if item == nil || item.ProyectoID == nil || *item.ProyectoID <= 0 {
		return "", nil
	}
	policy, err := db.GetProyectoAutonomia(*item.ProyectoID)
	if err != nil || policy == nil || !policy.Enabled {
		return "", err
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(*item.ProyectoID, 10))
	if err != nil || proyecto == nil {
		return "", err
	}
	if strings.TrimSpace(item.Classification) == "ready_for_review" {
		if note, err := notificarReviewerSignalTranscript(item, policy, proyecto); err != nil {
			return "", err
		} else if strings.TrimSpace(note) != "" {
			return note, nil
		}
	}
	notes := make([]string, 0, 3)
	supervisor, activado, err := resolverSupervisorAutonomiaOperativo(proyecto.ID, policy, "supervision_transcript_signal", strings.TrimSpace(item.Agente))
	if err != nil {
		return "", err
	}
	if activado {
		notes = append(notes, "supervisor_start")
	}
	if note, err := asegurarTareaReplanAutonomiaSignal(item, policy, proyecto, supervisor); err != nil {
		return "", err
	} else if strings.TrimSpace(note) != "" {
		notes = append(notes, note)
	}
	if supervisor == nil || strings.EqualFold(strings.TrimSpace(supervisor.Nombre), strings.TrimSpace(item.Agente)) {
		return strings.Join(notes, ";"), nil
	}
	if pendiente, err := existeRuntimeOrderAutonomiaPendiente(supervisor.Nombre, &proyecto.ID, "nudge", "inspeccionar_transcript_signal"); err != nil {
		return "", err
	} else if pendiente {
		notes = append(notes, "supervisor_nudge_pendiente")
		return strings.Join(notes, ";"), nil
	}
	resumen := fmt.Sprintf("agente=%s signal=%s transcript=%d", strings.TrimSpace(item.Agente), strings.TrimSpace(item.Classification), item.ID)
	if encolada, err := encolarNudgeAutonomiaDetallado(supervisor.Nombre, proyecto, "inspeccionar_transcript_signal", resumen, construirInstruccionSupervisionSignalTranscript(policy, proyecto, supervisor.Nombre, item), map[string]any{
		"signal_classification": strings.TrimSpace(item.Classification),
		"signal_transcript_id":  item.ID,
		"signal_agente":         strings.TrimSpace(item.Agente),
		"signal_text":           strings.TrimSpace(item.Text),
	}); err != nil {
		return "", err
	} else if !encolada {
		notes = append(notes, "supervisor_nudge_omitido")
		return strings.Join(notes, ";"), nil
	}
	notes = append(notes, "supervisor_nudged")
	return strings.Join(notes, ";"), nil
}

func asegurarTareaReplanAutonomiaSignal(item *db.RuntimeTranscriptEntry, policy *db.ProyectoAutonomia, proyecto *db.Proyecto, supervisor *db.Agente) (string, error) {
	if item == nil || policy == nil || proyecto == nil || !policy.Enabled {
		return "", nil
	}
	if strings.TrimSpace(item.Classification) != "needs_replan" {
		return "", nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{ProyectoID: &proyecto.ID})
	if err != nil {
		return "", err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaCompletada, db.TareaCancelada:
			continue
		}
		if strings.EqualFold(strings.TrimSpace(tarea.Titulo), autonomiaReplanTaskTitle) || strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:needs_replan") {
			return "replan_task_exists", nil
		}
	}
	target := ""
	if supervisor != nil && !strings.EqualFold(strings.TrimSpace(supervisor.Nombre), strings.TrimSpace(item.Agente)) {
		target = strings.TrimSpace(supervisor.Nombre)
	} else if preferred := strings.TrimSpace(policy.SupervisorAgente); preferred != "" && !strings.EqualFold(preferred, strings.TrimSpace(item.Agente)) {
		target = preferred
	}
	id, err := tareasService.Create(tareasapp.CreateTaskInput{
		Titulo:      autonomiaReplanTaskTitle,
		Descripcion: construirDescripcionTareaReplanAutonomia(policy, proyecto, item),
		Modulo:      "autonomia",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Agente:      target,
		Proyecto:    proyecto.Slug,
		Notas:       fmt.Sprintf("autonomia:needs_replan;transcript:%d;agente_origen:%s", item.ID, strings.TrimSpace(item.Agente)),
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(target) != "" {
		if err := tareasService.Start(id, target); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("replan_task_created:%d", id), nil
}

func construirDescripcionTareaReplanAutonomia(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, item *db.RuntimeTranscriptEntry) string {
	partes := []string{
		"Orquesta ha detectado en el transcript una señal de needs_replan: el agente no tiene claro el siguiente paso útil.",
		"Replanifica backlog, prioridades, propuestas, worktrees y checkpoints para abrir o reforzar el siguiente frente útil sin intervención humana.",
	}
	if proyecto != nil && strings.TrimSpace(proyecto.Slug) != "" {
		partes = append(partes, "Proyecto: "+strings.TrimSpace(proyecto.Slug)+".")
	}
	if item != nil && strings.TrimSpace(item.Agente) != "" {
		partes = append(partes, "Agente origen: "+strings.TrimSpace(item.Agente)+".")
	}
	if item != nil && strings.TrimSpace(item.Text) != "" {
		partes = append(partes, "Señal transcript: "+strings.TrimSpace(item.Text)+".")
	}
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		partes = append(partes, "Objetivo general: "+strings.TrimSpace(policy.ObjetivoGeneral)+".")
	}
	if policy != nil && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "" && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "{}" {
		partes = append(partes, "Definition of done: "+strings.TrimSpace(policy.DefinitionOfDoneJSON)+".")
	}
	return strings.Join(partes, " ")
}

func notificarReviewerSignalTranscript(item *db.RuntimeTranscriptEntry, policy *db.ProyectoAutonomia, proyecto *db.Proyecto) (string, error) {
	if item == nil || proyecto == nil || policy == nil || !policy.Enabled {
		return "", nil
	}
	reviewer, activado, err := prepararAgentePreferidoAutonomia(proyecto.ID, strings.TrimSpace(policy.ReviewerAgente), "review_transcript_signal")
	if err != nil {
		return "", err
	}
	if activado {
		return "reviewer_start", nil
	}
	if reviewer == nil {
		reviewer, err = seleccionarAgenteActivoProyectoPreferido(proyecto.ID, strings.TrimSpace(policy.ReviewerAgente), []string{"revisor", "reviewer", "supervisor", "orquestador", "admin", "programador"}, strings.TrimSpace(item.Agente))
		if err != nil {
			return "", err
		}
	}
	if reviewer == nil || strings.EqualFold(strings.TrimSpace(reviewer.Nombre), strings.TrimSpace(item.Agente)) {
		return "", nil
	}
	if pendiente, err := existeRuntimeOrderAutonomiaPendiente(reviewer.Nombre, &proyecto.ID, "nudge", "inspeccionar_ready_for_review"); err != nil {
		return "", err
	} else if pendiente {
		return "reviewer_nudge_pendiente", nil
	}
	resumen := fmt.Sprintf("agente=%s signal=%s transcript=%d", strings.TrimSpace(item.Agente), strings.TrimSpace(item.Classification), item.ID)
	if encolada, err := encolarNudgeAutonomiaDetallado(reviewer.Nombre, proyecto, "inspeccionar_ready_for_review", resumen, construirInstruccionReviewSignalTranscript(policy, proyecto, reviewer.Nombre, item), map[string]any{
		"signal_classification": strings.TrimSpace(item.Classification),
		"signal_transcript_id":  item.ID,
		"signal_agente":         strings.TrimSpace(item.Agente),
		"signal_text":           strings.TrimSpace(item.Text),
	}); err != nil {
		return "", err
	} else if !encolada {
		return "reviewer_nudge_omitido", nil
	}
	return "reviewer_nudged", nil
}

func construirInstruccionSupervisionSignalTranscript(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, supervisor string, item *db.RuntimeTranscriptEntry) string {
	parts := []string{
		"Orquesta: un agente del proyecto ha emitido una señal de duda, espera o bloqueo en su transcript.",
		"Supervisa el frente ahora mismo: revisa el transcript, el contexto del proyecto, las tareas y propuestas activas, y decide el siguiente paso sin escalar a humano salvo que falten credenciales, secretos o un recurso externo real.",
	}
	if proyecto != nil {
		parts = append(parts, fmt.Sprintf("Proyecto: %s.", strings.TrimSpace(proyecto.Slug)))
	}
	if strings.TrimSpace(supervisor) != "" {
		parts = append(parts, "Supervisor responsable: "+strings.TrimSpace(supervisor)+".")
	}
	if item != nil {
		if strings.TrimSpace(item.Agente) != "" {
			parts = append(parts, "Agente origen: "+strings.TrimSpace(item.Agente)+".")
		}
		if strings.TrimSpace(item.Classification) != "" {
			parts = append(parts, "Clasificación: "+strings.TrimSpace(item.Classification)+".")
		}
		if strings.TrimSpace(item.Text) != "" {
			parts = append(parts, "Fragmento detectado: "+strings.TrimSpace(item.Text)+".")
		}
	}
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		parts = append(parts, "Objetivo general: "+strings.TrimSpace(policy.ObjetivoGeneral)+".")
	}
	if policy != nil && policy.AutoCreateTasks {
		parts = append(parts, "Si basta con una instrucción, emítela; si hace falta, crea o reajusta tareas, propuestas o handoff para que el proyecto no se quede parado.")
	} else {
		parts = append(parts, "Si basta con una instrucción, emítela; si hacen falta cambios estructurales, deja el siguiente frente claro sin crear tareas nuevas automáticamente.")
	}
	return strings.Join(parts, " ")
}

func construirInstruccionReviewSignalTranscript(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, reviewer string, item *db.RuntimeTranscriptEntry) string {
	parts := []string{
		"Orquesta: un agente del proyecto ha indicado en su transcript que el frente está listo para revisión.",
		"Valida el estado real del código, la definición de terminado, tests, arquitectura e i18n; si el frente está maduro, impulsa o ejecuta la revisión, y si no, pide cambios concretos sin bloquear el proyecto más de lo necesario.",
	}
	if proyecto != nil {
		parts = append(parts, fmt.Sprintf("Proyecto: %s.", strings.TrimSpace(proyecto.Slug)))
	}
	if strings.TrimSpace(reviewer) != "" {
		parts = append(parts, "Reviewer responsable: "+strings.TrimSpace(reviewer)+".")
	}
	if item != nil {
		if strings.TrimSpace(item.Agente) != "" {
			parts = append(parts, "Agente origen: "+strings.TrimSpace(item.Agente)+".")
		}
		if strings.TrimSpace(item.Text) != "" {
			parts = append(parts, "Fragmento detectado: "+strings.TrimSpace(item.Text)+".")
		}
	}
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		parts = append(parts, "Objetivo general: "+strings.TrimSpace(policy.ObjetivoGeneral)+".")
	}
	parts = append(parts, "Si aún falta trabajo, conviértelo en feedback accionable; si está listo, deja trazabilidad de revisión sin pedir intervención humana por defecto.")
	return strings.Join(parts, " ")
}

func controlPlaneConfigBoolOrDefault(clave string, fallback bool) bool {
	v, err := db.ConfigGet(clave)
	if err != nil {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "si", "sí", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func procesarRuntimeMailboxInteractivoBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	total := 0
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		texto, ok := construirInstruccionMailboxInteractivo(msg)
		if !ok {
			continue
		}
		if err := coalescerRuntimeMailboxPendiente(msg); err != nil {
			return total, err
		}
		handle, err := runtimesService.GetActiveRuntimeHandleAgentProject(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return total, err
		}
		if handle == nil {
			continue
		}
		if db.RuntimeHandleMailboxDeliveryMode(handle) != runtimeagente.MailboxDeliveryInteractive {
			continue
		}
		if abierta, err := existeRuntimeOrderAbiertaPorHandle(handle.ID, "send_instruction"); err != nil {
			return total, err
		} else if abierta {
			continue
		}
		if dedupe, err := existeIntentoSendInstructionMailboxParaHandle(msg, handle, ""); err != nil {
			return total, err
		} else if dedupe {
			continue
		}
		orderID, err := encolarSendInstructionDesdeRuntimeMailbox(msg, handle, texto, "")
		if err != nil {
			return total, err
		}
		if runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
			superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
			if err != nil {
				return total, err
			}
			if superseded > 0 {
				db.Audit("orquesta", "runtime_mailbox_interactivo_supersede", "runtime_mailbox", msg.ID,
					fmt.Sprintf("agente=%s kind=%s superseded=%d", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), superseded))
			}
		}
		db.Audit("orquesta", "runtime_mailbox_interactivo", "runtime_order", orderID,
			fmt.Sprintf("mailbox_id=%d agente=%s kind=%s", msg.ID, strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
		total++
	}
	return total, nil
}

func procesarRuntimeMailboxSupervisorLocalBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	total := 0
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		texto, ok := construirInstruccionMailboxInteractivo(msg)
		if !ok {
			continue
		}
		if err := coalescerRuntimeMailboxPendiente(msg); err != nil {
			return total, err
		}
		handle, err := runtimesService.GetActiveRuntimeHandleAgentProject(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return total, err
		}
		if handle == nil {
			continue
		}
		var runtime *db.RuntimeInstance
		handle, runtime, externalSessionID, err := db.SincronizarRuntimeHandleSupervisado(handle, runtime, "runtime_mailbox_supervisor_local")
		if err != nil {
			return total, err
		}
		if handle == nil {
			continue
		}
		if !db.RuntimeHandlePermiteEntregaCalienteSupervisada(handle) || db.RuntimeHandlePermiteSendInputInteractivo(handle) {
			continue
		}
		if covered, _, _, err := db.RuntimeMailboxCubiertoPorBootstrapPendiente(msg.ID, handle, runtime); err != nil {
			return total, err
		} else if covered {
			continue
		}
		if abierta, err := existeRuntimeOrderAbiertaPorHandle(handle.ID, "send_instruction"); err != nil {
			return total, err
		} else if abierta {
			continue
		}
		if dedupe, err := existeIntentoSendInstructionMailboxParaHandle(msg, handle, externalSessionID); err != nil {
			return total, err
		} else if dedupe {
			continue
		}
		orderID, err := encolarSendInstructionDesdeRuntimeMailbox(msg, handle, texto, externalSessionID)
		if err != nil {
			return total, err
		}
		if runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
			superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
			if err != nil {
				return total, err
			}
			if superseded > 0 {
				db.Audit("orquesta", "runtime_mailbox_supervisor_local_supersede", "runtime_mailbox", msg.ID,
					fmt.Sprintf("agente=%s kind=%s superseded=%d", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), superseded))
			}
		}
		db.Audit("orquesta", "runtime_mailbox_supervisor_local", "runtime_order", orderID,
			fmt.Sprintf("mailbox_id=%d agente=%s kind=%s", msg.ID, strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
		total++
	}
	return total, nil
}

func procesarRuntimeMailboxCoordinatedRestartBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	type recycleKey struct {
		agente     string
		proyectoID int64
	}
	latest := make(map[recycleKey]*db.RuntimeMailboxMessage)
	for _, msg := range mailbox {
		if msg == nil || msg.ProyectoID == nil || *msg.ProyectoID <= 0 {
			continue
		}
		if err := coalescerRuntimeMailboxPendiente(msg); err != nil {
			return 0, err
		}
		if !runtimeMailboxKindRequiereCoordinatedRestart(strings.TrimSpace(msg.Kind)) {
			continue
		}
		handle, err := runtimesService.GetActiveRuntimeHandleAgentProject(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return 0, err
		}
		if !runtimeHandleRequiereCoordinatedRestartMailbox(handle) {
			continue
		}
		if pendiente, err := existeRuntimeOrderAbiertaAutonomia(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, "stop", "start", "restart", "resume", "handoff"); err != nil {
			return 0, err
		} else if pendiente {
			continue
		}
		key := recycleKey{
			agente:     strings.TrimSpace(msg.ToAgente),
			proyectoID: *msg.ProyectoID,
		}
		if prev := latest[key]; prev == nil || msg.ID > prev.ID {
			latest[key] = msg
		}
	}
	keys := make([]recycleKey, 0, len(latest))
	for key := range latest {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].proyectoID == keys[j].proyectoID {
			return keys[i].agente < keys[j].agente
		}
		return keys[i].proyectoID < keys[j].proyectoID
	})
	total := 0
	for _, key := range keys {
		msg := latest[key]
		if msg == nil {
			continue
		}
		proyecto, err := runtimesService.GetProject(strconv.FormatInt(key.proyectoID, 10))
		if err != nil || proyecto == nil {
			return total, err
		}
		handle, err := runtimesService.GetActiveRuntimeHandleAgentProject(key.agente, &key.proyectoID)
		if err != nil {
			return total, err
		}
		if handle == nil {
			continue
		}
		motivo := fmt.Sprintf("runtime_mailbox_coordinated_restart:%s:%d", strings.TrimSpace(msg.Kind), msg.ID)
		stopOrderID, startOrderID, err := encolarReinicioCoordinadoMailbox(handle, proyecto, msg, motivo)
		if err != nil {
			return total, err
		}
		if runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
			superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
			if err != nil {
				return total, err
			}
			if superseded > 0 {
				db.Audit("orquesta", "runtime_mailbox_coordinated_restart_supersede", "runtime_mailbox", msg.ID,
					fmt.Sprintf("agente=%s kind=%s superseded=%d", key.agente, strings.TrimSpace(msg.Kind), superseded))
			}
		}
		db.Audit("orquesta", "runtime_mailbox_coordinated_restart", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s proyecto=%s kind=%s stop_order_id=%d start_order_id=%d", key.agente, proyecto.Slug, strings.TrimSpace(msg.Kind), stopOrderID, startOrderID))
		total++
	}
	return total, nil
}

func procesarRuntimeMailboxSessionResumeBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	total := 0
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		texto, ok := construirInstruccionMailboxInteractivo(msg)
		if !ok {
			continue
		}
		if err := coalescerRuntimeMailboxPendiente(msg); err != nil {
			return total, err
		}
		handle, err := runtimesService.GetActiveRuntimeHandleAgentProject(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return total, err
		}
		if handle == nil {
			continue
		}
		var runtime *db.RuntimeInstance
		if handle.RuntimeID != nil {
			runtime, err = db.GetRuntime(*handle.RuntimeID)
			if err != nil {
				return total, err
			}
		}
		handle, externalSessionID, err := db.SincronizarRuntimeHandleExternalSessionID(handle, runtime)
		if err != nil {
			return total, err
		}
		if handle == nil || strings.TrimSpace(externalSessionID) == "" {
			continue
		}
		if covered, _, _, err := db.RuntimeMailboxCubiertoPorBootstrapPendiente(msg.ID, handle, runtime); err != nil {
			return total, err
		} else if covered {
			continue
		}
		if db.RuntimeHandleMailboxDeliveryMode(handle) != runtimeagente.MailboxDeliverySessionResume {
			continue
		}
		if abierta, err := existeRuntimeOrderAbiertaPorHandle(handle.ID, "send_instruction"); err != nil {
			return total, err
		} else if abierta {
			continue
		}
		if dedupe, err := existeIntentoSendInstructionMailboxParaHandle(msg, handle, externalSessionID); err != nil {
			return total, err
		} else if dedupe {
			continue
		}
		orderID, err := encolarSendInstructionDesdeRuntimeMailbox(msg, handle, texto, externalSessionID)
		if err != nil {
			return total, err
		}
		if runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
			superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
			if err != nil {
				return total, err
			}
			if superseded > 0 {
				db.Audit("orquesta", "runtime_mailbox_session_resume_supersede", "runtime_mailbox", msg.ID,
					fmt.Sprintf("agente=%s kind=%s superseded=%d", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), superseded))
			}
		}
		db.Audit("orquesta", "runtime_mailbox_session_resume", "runtime_order", orderID,
			fmt.Sprintf("mailbox_id=%d agente=%s kind=%s", msg.ID, strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
		total++
	}
	return total, nil
}

func encolarSendInstructionDesdeRuntimeMailbox(msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle, texto, externalSessionID string) (int64, error) {
	if msg == nil || handle == nil {
		return 0, nil
	}
	payload := map[string]any{
		"to_agente":                  strings.TrimSpace(msg.ToAgente),
		"from_agente":                strings.TrimSpace(msg.FromAgente),
		"texto":                      strings.TrimSpace(texto),
		"mailbox_id":                 msg.ID,
		"mailbox_kind":               strings.TrimSpace(msg.Kind),
		"delivery_attempt_signature": runtimeMailboxDeliveryAttemptSignature(handle, externalSessionID),
	}
	if strings.TrimSpace(externalSessionID) != "" {
		payload["external_session_id"] = strings.TrimSpace(externalSessionID)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	order := &db.RuntimeOrder{
		Agente:      strings.TrimSpace(msg.ToAgente),
		ProyectoID:  msg.ProyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: string(raw),
	}
	if handle.RuntimeID != nil {
		order.RuntimeID = handle.RuntimeID
	}
	order.HandleID = &handle.ID
	return db.EncolarRuntimeOrder(order)
}

func existeIntentoSendInstructionMailboxParaHandle(msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle, externalSessionID string) (bool, error) {
	if msg == nil || handle == nil || msg.ID <= 0 {
		return false, nil
	}
	agente := strings.TrimSpace(msg.ToAgente)
	if agente == "" {
		return false, nil
	}
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: msg.ProyectoID})
	if err != nil {
		return false, err
	}
	currentSessionID := strings.TrimSpace(externalSessionID)
	currentSignature := runtimeMailboxDeliveryAttemptSignature(handle, externalSessionID)
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != "send_instruction" {
			continue
		}
		if runtimeOrderMailboxIDFromJSON(order.PayloadJSON) != msg.ID {
			continue
		}
		if order.HandleID != nil && *order.HandleID != handle.ID {
			continue
		}
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando":
			return true, nil
		case "completada":
			if !runtimeOrderMailboxOnlyResult(order.ResultadoJSON) {
				continue
			}
			orderSignature := runtimeOrderDeliveryAttemptSignature(order.PayloadJSON)
			if currentSignature != "" {
				if orderSignature == "" {
					continue
				}
				if currentSignature != orderSignature {
					continue
				}
				return true, nil
			}
			orderSessionID := strings.TrimSpace(runtimeOrderExternalSessionIDFromJSON(order.PayloadJSON))
			if currentSessionID != "" && orderSessionID != "" && currentSessionID != orderSessionID {
				continue
			}
			return true, nil
		}
	}
	return false, nil
}

func runtimeOrderMailboxIDFromJSON(raw string) int64 {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return 0
	}
	value, _ := payload["mailbox_id"]
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}

func runtimeOrderExternalSessionIDFromJSON(raw string) string {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return ""
	}
	value, _ := payload["external_session_id"]
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func runtimeOrderDeliveryAttemptSignature(raw string) string {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return ""
	}
	if text, ok := payload["delivery_attempt_signature"].(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func runtimeMailboxDeliveryAttemptSignature(handle *db.RuntimeHandle, externalSessionID string) string {
	if handle == nil || handle.ID <= 0 {
		return ""
	}
	mode := strings.TrimSpace(string(db.RuntimeHandleMailboxDeliveryMode(handle)))
	if db.RuntimeHandlePermiteEntregaCalienteSupervisada(handle) && !db.RuntimeHandlePermiteSendInputInteractivo(handle) {
		mode = "supervisor_local"
	}
	if mode == "" {
		return ""
	}
	return fmt.Sprintf("%s|handle:%d|session:%s", mode, handle.ID, strings.TrimSpace(externalSessionID))
}

func runtimeOrderMailboxOnlyResult(raw string) bool {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return false
	}
	value, _ := payload["mailbox_only"]
	flag, ok := value.(bool)
	return ok && flag
}

func encolarReinicioCoordinadoMailbox(handle *db.RuntimeHandle, proyecto *db.Proyecto, msg *db.RuntimeMailboxMessage, motivo string) (int64, int64, error) {
	if handle == nil || msg == nil {
		return 0, 0, nil
	}
	return db.EncolarReinicioCoordinadoRuntimeHandle(handle, proyectoIDPtr(proyecto), motivo, "orquesta")
}

func proyectoIDPtr(proyecto *db.Proyecto) *int64 {
	if proyecto == nil || proyecto.ID <= 0 {
		return nil
	}
	return &proyecto.ID
}

func runtimeMailboxKindCoalescible(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "instruction", "autonomia", "nudge", "watchdog", db.MailboxKindGovernanceRefresh, db.MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
}

func runtimeMailboxKindRequiereCoordinatedRestart(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "instruction", "autonomia", "nudge", "watchdog", db.MailboxKindGovernanceRefresh, db.MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
}

func coalescerRuntimeMailboxPendiente(msg *db.RuntimeMailboxMessage) error {
	if msg == nil || !runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
		return nil
	}
	superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
	if err != nil {
		return err
	}
	if superseded > 0 {
		db.Audit("orquesta", "runtime_mailbox_supersede", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s kind=%s superseded=%d", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), superseded))
	}
	return nil
}

func runtimeHandleRequiereCoordinatedRestartMailbox(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if strings.TrimSpace(handle.Transporte) != "cli" || strings.TrimSpace(handle.HandleKind) != "process" {
		return false
	}
	if db.RuntimeHandleMailboxDeliveryMode(handle) != runtimeagente.MailboxDeliveryCoordinatedRestart {
		return false
	}
	return true
}

func existeRuntimeOrderAbiertaPorHandle(handleID int64, tipos ...string) (bool, error) {
	if handleID <= 0 || len(tipos) == 0 {
		return false, nil
	}
	estados := []string{"pendiente", "tomada", "ejecutando"}
	for _, estado := range estados {
		estado := estado
		orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{Estado: &estado})
		if err != nil {
			return false, err
		}
		for _, order := range orders {
			if order == nil || order.HandleID == nil || *order.HandleID != handleID {
				continue
			}
			for _, tipo := range tipos {
				if strings.TrimSpace(order.Tipo) == strings.TrimSpace(tipo) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func construirInstruccionMailboxInteractivo(msg *db.RuntimeMailboxMessage) (string, bool) {
	if msg == nil {
		return "", false
	}
	payload := map[string]any{}
	if strings.TrimSpace(msg.PayloadJSON) != "" {
		_ = json.Unmarshal([]byte(msg.PayloadJSON), &payload)
	}
	switch strings.TrimSpace(msg.Kind) {
	case "instruction":
		texto := stringMapValue(payload, "texto")
		if texto == "" {
			texto = stringMapValue(payload, "instruction")
		}
		return texto, strings.TrimSpace(texto) != ""
	case "nudge", "watchdog":
		texto := stringMapValue(payload, "instruction")
		if texto == "" {
			texto = stringMapValue(payload, "texto")
		}
		return texto, strings.TrimSpace(texto) != ""
	case "autonomia":
		if instruction := stringMapValue(payload, "instruction"); strings.TrimSpace(instruction) != "" {
			return strings.TrimSpace(instruction), true
		}
		accion := stringMapValue(payload, "accion")
		motivo := stringMapValue(payload, "motivo")
		base := "Orquesta: continúa de forma autónoma dentro de la gobernanza efectiva del proyecto. No necesitas aprobación humana salvo que falten credenciales, secretos o un recurso externo real."
		switch strings.TrimSpace(accion) {
		case "supervisar_proyecto":
			base += " Actúa como supervisor del proyecto: revisa el estado real, comprueba cumplimiento de reglas, detecta flecos, crea o ajusta tareas si falta trabajo y deja el siguiente frente útil encaminado."
		case "ejecutar_review_gate":
			base += " Actúa como revisor del proyecto: inspecciona el código, valida arquitectura, tests y definición de terminado, documenta findings y aprueba o pide cambios sin detener el proyecto."
		case "continuar_trabajo":
			base += " Sigue con el trabajo en curso y cierra el siguiente frente útil."
		case "esperar_o_pedir_tarea":
			base += " No te quedes esperando: revisa backlog, asignación y siguiente paso útil, y continúa."
		case "votar_propuestas_pendientes":
			base += " Revisa y resuelve las propuestas pendientes del proyecto antes de continuar."
		case "pedir_intervencion":
			base += " No escales a humano por defecto: decide el mejor siguiente paso permitido y ejecuta."
		default:
			base += " Evalúa el mejor siguiente paso y ejecútalo."
		}
		if strings.TrimSpace(motivo) != "" {
			base += " Contexto: " + strings.TrimSpace(motivo) + "."
		}
		base += " Si el cambio es delicado, crea checkpoint y sigue."
		return base, true
	case db.MailboxKindGovernanceRefresh, db.MailboxKindSkillsRefresh:
		return construirInstruccionRefreshRuntime(msg)
	default:
		return "", false
	}
}

func procesarAutonomiaAgentesBatch() (int, error) {
	activa := true
	sesiones, err := sesionesAPIService.ListInspectionSessions(db.FiltroSesionesInspeccion{Activa: &activa})
	if err != nil {
		return 0, err
	}
	procesadas := 0
	vistas := map[string]struct{}{}
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		agente := strings.TrimSpace(sesion.Agente)
		if agente == "" {
			continue
		}
		if _, ok := vistas[agente]; ok {
			continue
		}
		vistas[agente] = struct{}{}
		n, err := procesarAutonomiaSesionActiva(sesion)
		if err != nil {
			db.Audit("server", "autonomia_agente_error", "agente", 0, fmt.Sprintf("agente=%s error=%s", agente, err.Error()))
			continue
		}
		procesadas += n
	}
	return procesadas, nil
}

func procesarAutonomiaSesionActiva(sesion *db.Sesion) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	if n, err := procesarCierreProyectoSesion(sesion); err != nil || n > 0 {
		return n, err
	}
	if n, err := procesarAparcadoAutonomoSesion(sesion); err != nil || n > 0 {
		return n, err
	}
	if n, err := procesarRecuperacionRuntimeDegradadoSesion(sesion); err != nil || n > 0 {
		return n, err
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	if err != nil {
		return 0, err
	}
	out, err := construirAgenteTickOutput(sesion.Agente, proyecto, sesion, 0)
	if err != nil {
		return 0, err
	}
	switch strings.TrimSpace(out.AccionRecomendada) {
	case "pausar_por_cuota", "pausar_y_reasignar":
		if satisfecha, err := pausaAutonomiaYaSatisfecha(sesion.Agente, &proyecto.ID, sesion); err != nil {
			return 0, err
		} else if satisfecha {
			return 0, nil
		}
		if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
			Agente:   sesion.Agente,
			Proyecto: proyecto.Slug,
			Accion:   agenteControlAccionPause,
			Motivo:   out.Motivo,
			Por:      "orquesta",
		}); err != nil {
			return 0, err
		}
		return 1, nil
	case "supervisar_proyecto":
		// La supervisión rica del proyecto ya tiene su propio batch y señales
		// dedicadas. Repetir un nudge genérico por cada tick de una sesión viva
		// solo reinyecta guidance redundante sobre un runtime ya activo.
		return 0, nil
	case "votar_propuestas_pendientes", "pedir_intervencion":
		if pendiente, err := existeRuntimeOrderAutonomiaPendiente(sesion.Agente, &proyecto.ID, "nudge", strings.TrimSpace(out.AccionRecomendada)); err != nil {
			return 0, err
		} else if pendiente {
			return 0, nil
		}
		if encolada, err := encolarNudgeAutonomia(sesion.Agente, proyecto, out.AccionRecomendada, out.Motivo); err != nil {
			return 0, err
		} else if !encolada {
			return 0, nil
		}
		return 1, nil
	case "continuar_trabajo", "esperar_o_pedir_tarea":
		// Una sesión ya activa no debe recibir recordatorios periódicos para
		// "seguir trabajando": en conectores no interactivos eso se traduce en
		// churn de mailbox y reinicios artificiales. Las acciones pasivas quedan
		// implícitas en la tarea y el contexto ya adoptado.
		return 0, nil
	default:
		return 0, nil
	}
}

func procesarCierreProyectoSesion(sesion *db.Sesion) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	if err != nil {
		return 0, err
	}
	if policy, err := supervisionService.GetProjectPolicy(proyecto.Slug); err != nil {
		return 0, err
	} else if policy == nil || !policy.Enabled || !policy.AutoCloseProject {
		return 0, nil
	}
	terminado, motivo, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil || !terminado {
		return 0, err
	}
	if err := db.MarcarProyectoCerrado(proyecto.ID, motivo); err != nil {
		return 0, err
	}
	estadoSesion := strings.ToLower(strings.TrimSpace(sesion.Estado))
	if estadoSesion == "pausada" {
		if err := db.AparcarSesionActiva(sesion.Agente, sesion.ProyectoID); err != nil {
			return 0, err
		}
		if err := db.PausarAsignacion(sesion.Agente, proyecto.ID, "proyecto_terminado:"+motivo); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if satisfecha, err := pausaAutonomiaYaSatisfecha(sesion.Agente, &proyecto.ID, sesion); err != nil {
		return 0, err
	} else if satisfecha {
		return 0, nil
	}
	if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
		Agente:   sesion.Agente,
		Proyecto: proyecto.Slug,
		Accion:   agenteControlAccionPause,
		Motivo:   "proyecto_terminado:" + motivo,
		Por:      "orquesta",
	}); err != nil {
		return 0, err
	}
	return 1, nil
}

func proyectoTerminadoAutonomamente(proyecto *db.Proyecto) (bool, string, error) {
	if proyecto == nil {
		return false, "", nil
	}
	policy, err := supervisionService.GetProjectPolicy(proyecto.Slug)
	if err != nil {
		return false, "", err
	}
	if policy == nil || !policy.Enabled || !policy.AutoCloseProject {
		return false, "", nil
	}
	listo, motivo, err := proyectoSinTrabajoPendiente(proyecto)
	if err != nil || !listo {
		return listo, motivo, err
	}
	reviewOK, reviewMotivo, err := proyectoReviewAutonomoCompletado(proyecto)
	if err != nil || !reviewOK {
		return false, reviewMotivo, err
	}
	if strings.TrimSpace(reviewMotivo) != "" {
		motivo += "; " + strings.TrimSpace(reviewMotivo)
	}
	integrado, integracionMotivo, err := proyectoIntegracionAutonomaCompletada(proyecto)
	if err != nil || !integrado {
		return false, integracionMotivo, err
	}
	if strings.TrimSpace(integracionMotivo) != "" {
		motivo += "; " + strings.TrimSpace(integracionMotivo)
	}
	return true, motivo, nil
}

func proyectoIntegracionAutonomaCompletada(proyecto *db.Proyecto) (bool, string, error) {
	if proyecto == nil {
		return false, "", nil
	}
	merges, err := gitgobernanza.NewService(gitgobernanza.Repository{}).ListRequests(strings.TrimSpace(proyecto.Slug), "")
	if err != nil {
		return false, "", err
	}
	if len(merges) == 0 {
		return true, "", nil
	}
	var latestMerged *db.GitMerge
	for _, merge := range merges {
		if merge == nil {
			continue
		}
		switch strings.TrimSpace(merge.Estado) {
		case "pendiente", "validando", "aprobado", "ejecutando":
			return false, fmt.Sprintf("esperando integración merge #%d (%s)", merge.ID, strings.TrimSpace(merge.Estado)), nil
		case "fallido":
			return false, fmt.Sprintf("integración merge #%d fallida", merge.ID), nil
		case "fusionado":
			if latestMerged == nil || merge.ID > latestMerged.ID {
				latestMerged = merge
			}
		}
	}
	if latestMerged != nil {
		return true, fmt.Sprintf("merge #%d fusionado", latestMerged.ID), nil
	}
	return false, "sin integración fusionada", nil
}

func procesarAparcadoAutonomoSesion(sesion *db.Sesion) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	op, err := db.GetProyectoOperacion(*sesion.ProyectoID)
	if err != nil {
		return 0, err
	}
	motivo := strings.TrimSpace(op.Motivo)
	switch op.EstadoOperativo {
	case db.ProyectoOperativoActivo:
		bloqueado, motivoDetectado, err := db.ResolverBloqueoProyecto(*sesion.ProyectoID)
		if err != nil || !bloqueado {
			return 0, err
		}
		motivo = strings.TrimSpace(motivoDetectado)
		if err := db.MarcarProyectoEsperandoHumano(*sesion.ProyectoID, motivo); err != nil {
			return 0, err
		}
	case db.ProyectoOperativoEsperandoHumano, db.ProyectoOperativoBloqueadoExterno:
		if strings.TrimSpace(motivo) == "" {
			motivo = "esperando_desbloqueo_humano"
		}
	default:
		return 0, nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	if err != nil {
		return 0, err
	}
	estadoSesion := strings.ToLower(strings.TrimSpace(sesion.Estado))
	if estadoSesion == "pausada" {
		if err := db.AparcarSesionActiva(sesion.Agente, sesion.ProyectoID); err != nil {
			return 0, err
		}
		if err := db.PausarAsignacion(sesion.Agente, *sesion.ProyectoID, "bloqueo_humano:"+motivo); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if satisfecha, err := pausaAutonomiaYaSatisfecha(sesion.Agente, sesion.ProyectoID, sesion); err != nil {
		return 0, err
	} else if satisfecha {
		return 0, nil
	}
	if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
		Agente:   sesion.Agente,
		Proyecto: proyecto.Slug,
		Accion:   agenteControlAccionPause,
		Motivo:   "bloqueo_humano:" + motivo,
		Por:      "orquesta",
	}); err != nil {
		return 0, err
	}
	return 1, nil
}

func procesarRecuperacionRuntimeDegradadoSesion(sesion *db.Sesion) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		return 0, err
	}
	if handle == nil {
		return 0, nil
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil {
		return 0, err
	}
	if !esTransporteRemotoAutonomia(handle.Transporte) {
		if !runtimeLocalFallido(handle, runtime) {
			return 0, nil
		}
		if pendiente, err := existeRuntimeOrderAbiertaAutonomia(sesion.Agente, sesion.ProyectoID, "start", "resume", "handoff"); err != nil {
			return 0, err
		} else if pendiente {
			return 0, nil
		}
		proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
		if err != nil {
			return 0, err
		}
		tieneTrabajo, err := dbAgenteTieneTrabajoArrancable(strings.TrimSpace(sesion.Agente), proyecto.ID)
		if err != nil {
			return 0, err
		}
		if !tieneTrabajo {
			return 0, nil
		}
		if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
			Agente:   strings.TrimSpace(sesion.Agente),
			Proyecto: proyecto.Slug,
			Accion:   agenteControlAccionStart,
			Motivo:   "local_runtime_failed",
			Por:      "orquesta",
		}); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if !runtimeRemotoDegradado(handle, runtime) {
		return 0, nil
	}
	conector, err := resolverConectorSesionAutonomia(sesion, runtime, handle)
	if err != nil {
		return 0, err
	}
	if conector != nil {
		disponible, _, err := db.ConectorDisponibleParaArranque(conector.ID)
		if err != nil {
			return 0, err
		}
		if !disponible {
			motivo := "conector:" + strings.TrimSpace(conector.Slug) + ":circuito_abierto"
			if err := db.MarcarProyectoBloqueadoExterno(*sesion.ProyectoID, motivo); err != nil {
				return 0, err
			}
			proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
			if err != nil {
				return 0, err
			}
			if satisfecha, err := pausaAutonomiaYaSatisfecha(sesion.Agente, sesion.ProyectoID, sesion); err != nil {
				return 0, err
			} else if satisfecha {
				return 1, nil
			}
			if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
				Agente:   strings.TrimSpace(sesion.Agente),
				Proyecto: proyecto.Slug,
				Accion:   agenteControlAccionPause,
				Motivo:   motivo,
				Por:      "orquesta",
			}); err != nil {
				return 0, err
			}
			return 1, nil
		}
	}
	if pendiente, err := existeRuntimeOrderAbiertaAutonomia(sesion.Agente, sesion.ProyectoID, "checkpoint", "start", "resume", "handoff"); err != nil {
		return 0, err
	} else if pendiente {
		return 0, nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	if err != nil {
		return 0, err
	}
	checkpointPayload, err := json.Marshal(map[string]any{
		"checkpoint_kind": "remote_recovery",
		"resumen":         "Checkpoint automático antes de recuperación de runtime remoto degradado",
		"motivo":          "remote_runtime_degraded",
	})
	if err != nil {
		return 0, err
	}
	if _, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      strings.TrimSpace(sesion.Agente),
		ProyectoID:  sesion.ProyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "checkpoint",
		PayloadJSON: string(checkpointPayload),
	}); err != nil {
		return 0, err
	}
	accion := agenteControlAccionStart
	if runtimeRemotoReanudable(sesion, handle) {
		accion = agenteControlAccionResume
	}
	if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
		Agente:   strings.TrimSpace(sesion.Agente),
		Proyecto: proyecto.Slug,
		Accion:   accion,
		Motivo:   "remote_runtime_degraded",
		Por:      "orquesta",
	}); err != nil {
		return 0, err
	}
	return 2, nil
}

func runtimeLocalFallido(handle *db.RuntimeHandle, runtime *db.RuntimeInstance) bool {
	if handle == nil || esTransporteRemotoAutonomia(handle.Transporte) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.Estado), "fallido") {
		return true
	}
	if runtime == nil {
		return false
	}
	logical := strings.ToLower(strings.TrimSpace(runtime.LogicalState))
	process := strings.ToLower(strings.TrimSpace(runtime.ProcessState))
	return logical == "fallido" || process == "fallido" || process == "crashed" || process == "exited"
}

func resolverConectorSesionAutonomia(sesion *db.Sesion, runtime *db.RuntimeInstance, handle *db.RuntimeHandle) (*db.Conector, error) {
	if sesion != nil && strings.TrimSpace(sesion.ConectorSlug) != "" {
		return db.GetConector(strings.TrimSpace(sesion.ConectorSlug))
	}
	if runtime != nil && strings.TrimSpace(runtime.Connector) != "" {
		return db.GetConector(strings.TrimSpace(runtime.Connector))
	}
	if handle != nil {
		slug := strings.TrimSpace(stringFromMetadataJSON(handle.MetadataJSON, "conector"))
		if slug != "" {
			return db.GetConector(slug)
		}
	}
	return nil, nil
}

func stringFromMetadataJSON(raw string, key string) string {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed); err != nil || parsed == nil {
		return ""
	}
	value, ok := parsed[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func esTransporteRemotoAutonomia(transport string) bool {
	switch strings.TrimSpace(transport) {
	case "api", "mcp_http", "otro":
		return true
	default:
		return false
	}
}

func runtimeRemotoDegradado(handle *db.RuntimeHandle, runtime *db.RuntimeInstance) bool {
	if handle != nil && strings.EqualFold(strings.TrimSpace(handle.Estado), "fallido") {
		return true
	}
	if runtime == nil {
		return false
	}
	logical := strings.ToLower(strings.TrimSpace(runtime.LogicalState))
	process := strings.ToLower(strings.TrimSpace(runtime.ProcessState))
	return logical == "degradado" || process == "remote_status_error"
}

func runtimeRemotoReanudable(sesion *db.Sesion, handle *db.RuntimeHandle) bool {
	if sesion == nil || handle == nil {
		return false
	}
	if !esTransporteRemotoAutonomia(handle.Transporte) {
		return false
	}
	if strings.TrimSpace(handle.HandleKind) == "process" {
		return false
	}
	if ext := strings.TrimSpace(sesion.ExternalSessionID); ext != "" {
		return true
	}
	if ext := strings.TrimSpace(stringFromMetadataJSON(handle.MetadataJSON, "external_session_id")); ext != "" {
		return true
	}
	ref := strings.TrimSpace(handle.HandleRef)
	if ref == "" {
		return false
	}
	return ref != strconv.FormatInt(sesion.ID, 10)
}

func reactivarAgenteTrasReanimacion(agente string) error {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil
	}
	proyectoID, err := db.ObtenerProyectoActivoAgente(agente)
	if err != nil || proyectoID == 0 {
		return err
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		return err
	}
	if pendiente, err := existeRuntimeOrderAbiertaAutonomia(agente, &proyecto.ID, "resume", "start", "handoff"); err != nil {
		return err
	} else if pendiente {
		return nil
	}
	handle, err := resolverHandleControlAgente(agente, &proyecto.ID)
	if err != nil {
		return err
	}
	if handle != nil && strings.TrimSpace(handle.Estado) == "pausado" {
		_, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
			Agente:   agente,
			Proyecto: proyecto.Slug,
			Accion:   agenteControlAccionResume,
			Motivo:   "reanimacion_automatica",
			Por:      "orquesta",
		})
		return err
	}
	tieneTrabajo, err := dbAgenteTieneTrabajoArrancable(agente, proyecto.ID)
	if err != nil {
		return err
	}
	if !tieneTrabajo {
		return nil
	}
	_, _, err = encolarControlAgenteLocal(apiAgenteControlRequest{
		Agente:   agente,
		Proyecto: proyecto.Slug,
		Accion:   agenteControlAccionStart,
		Motivo:   "reanimacion_automatica",
		Por:      "orquesta",
	})
	return err
}

func existeRuntimeOrderAutonomiaPendiente(agente string, proyectoID *int64, tipo string, accion string) (bool, error) {
	estado := "pendiente"
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return false, err
	}
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != strings.TrimSpace(tipo) {
			continue
		}
		if accion == "" {
			return true, nil
		}
		if strings.Contains(strings.ToLower(order.PayloadJSON), `"kind":"autonomia"`) &&
			strings.Contains(strings.ToLower(order.PayloadJSON), fmt.Sprintf(`"accion":"%s"`, strings.ToLower(strings.TrimSpace(accion)))) {
			return true, nil
		}
	}
	return false, nil
}

func pausaAutonomiaYaSatisfecha(agente string, proyectoID *int64, sesion *db.Sesion) (bool, error) {
	if pendiente, err := existeRuntimeOrderAutonomiaPendiente(agente, proyectoID, "pause", ""); err != nil {
		return false, err
	} else if pendiente {
		return true, nil
	}
	if sesion != nil && strings.EqualFold(strings.TrimSpace(sesion.Estado), "pausada") {
		return true, nil
	}
	infoAgente, err := db.GetAgente(strings.TrimSpace(agente))
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	if infoAgente != nil && strings.EqualFold(strings.TrimSpace(infoAgente.EstadoCuota), "enfriamiento") {
		handle, err := resolverHandleControlAgente(strings.TrimSpace(agente), proyectoID)
		if err != nil {
			return false, err
		}
		if handle == nil || strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
			return true, nil
		}
	}
	handle, err := resolverHandleControlAgente(strings.TrimSpace(agente), proyectoID)
	if err != nil {
		return false, err
	}
	if handle != nil && strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
		return true, nil
	}
	return false, nil
}

func existeRuntimeOrderAutonomiaReciente(agente string, proyectoID *int64, tipo string, accion string, within time.Duration) (bool, error) {
	if within <= 0 {
		return false, nil
	}
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
	})
	if err != nil {
		return false, err
	}
	cutoff := time.Now().UTC().Add(-within)
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != strings.TrimSpace(tipo) {
			continue
		}
		if accion != "" && !runtimeOrderAutonomiaTieneAccion(order, accion) {
			continue
		}
		if runtimeOrderTimestamp(order).Before(cutoff) {
			continue
		}
		return true, nil
	}
	return false, nil
}

func runtimeOrderAutonomiaTieneAccion(order *db.RuntimeOrder, accion string) bool {
	if order == nil {
		return false
	}
	if strings.Contains(strings.ToLower(order.PayloadJSON), `"kind":"autonomia"`) &&
		strings.Contains(strings.ToLower(order.PayloadJSON), fmt.Sprintf(`"accion":"%s"`, strings.ToLower(strings.TrimSpace(accion)))) {
		return true
	}
	return false
}

func runtimeOrderTimestamp(order *db.RuntimeOrder) time.Time {
	if order == nil {
		return time.Time{}
	}
	if order.FinishedAt != nil && !order.FinishedAt.IsZero() {
		return order.FinishedAt.UTC()
	}
	if order.StartedAt != nil && !order.StartedAt.IsZero() {
		return order.StartedAt.UTC()
	}
	if !order.UpdatedAt.IsZero() {
		return order.UpdatedAt.UTC()
	}
	return order.CreatedAt.UTC()
}

func existeRuntimeOrderAbiertaAutonomia(agente string, proyectoID *int64, tipos ...string) (bool, error) {
	if len(tipos) == 0 {
		return false, nil
	}
	estados := []string{"pendiente", "tomada", "ejecutando"}
	for _, estado := range estados {
		estado := estado
		orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{
			Agente:     &agente,
			ProyectoID: proyectoID,
			Estado:     &estado,
		})
		if err != nil {
			return false, err
		}
		for _, order := range orders {
			if order == nil {
				continue
			}
			for _, tipo := range tipos {
				if strings.TrimSpace(order.Tipo) == strings.TrimSpace(tipo) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func existeRuntimeOrderAbiertaAgente(agente string, proyectoID *int64) (bool, error) {
	estados := []string{"pendiente", "tomada", "ejecutando"}
	for _, estado := range estados {
		estado := estado
		orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{
			Agente:     &agente,
			ProyectoID: proyectoID,
			Estado:     &estado,
		})
		if err != nil {
			return false, err
		}
		if len(orders) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func dbAgenteTieneTrabajoArrancable(agente string, proyectoID int64) (bool, error) {
	filtro := db.FiltroTareas{
		Agente:     &agente,
		ProyectoID: &proyectoID,
	}
	tareas, err := tareasService.List(filtro)
	if err != nil {
		return false, err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaAsignada, db.TareaEnProgreso:
			return true, nil
		}
	}
	return false, nil
}

func encolarNudgeAutonomia(agente string, proyecto *db.Proyecto, accion, motivo string) (bool, error) {
	return encolarNudgeAutonomiaDetallado(agente, proyecto, accion, motivo, "", nil)
}

func encolarNudgeAutonomiaDetallado(agente string, proyecto *db.Proyecto, accion, motivo, instruction string, extras map[string]any) (bool, error) {
	if proyecto == nil {
		return false, nil
	}
	agente = strings.TrimSpace(agente)
	accion = strings.TrimSpace(accion)
	if abierta, err := existeRuntimeOrderAbiertaAutonomia(strings.TrimSpace(agente), &proyecto.ID, "send_instruction"); err != nil {
		return false, err
	} else if abierta {
		return false, nil
	}
	cooldown := time.Duration(controlPlaneConfigIntOrDefault("autonomia_nudge_cooldown_seconds", 60)) * time.Second
	if reciente, err := existeRuntimeOrderAutonomiaReciente(strings.TrimSpace(agente), &proyecto.ID, "nudge", "", cooldown); err != nil {
		return false, err
	} else if reciente {
		return false, nil
	}
	var runtimeID *int64
	var handleID *int64
	handle, err := resolverHandleControlAgente(strings.TrimSpace(agente), &proyecto.ID)
	if err != nil {
		return false, err
	}
	if handle != nil {
		handleID = &handle.ID
		if handle.RuntimeID != nil {
			runtimeID = handle.RuntimeID
		}
		if !db.RuntimeHandlePermiteSendInputInteractivo(handle) {
			if pendiente, err := existeRuntimeMailboxAutonomiaPendiente(agente, &proyecto.ID, accion); err != nil {
				return false, err
			} else if pendiente {
				return false, nil
			}
		}
	}
	payload := map[string]any{
		"from_agente": "server",
		"to_agente":   agente,
		"kind":        "autonomia",
		"accion":      accion,
		"texto":       strings.TrimSpace(motivo),
	}
	if strings.TrimSpace(instruction) != "" {
		payload["instruction"] = strings.TrimSpace(instruction)
	}
	for key, value := range extras {
		payload[key] = value
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	orderID, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      strings.TrimSpace(agente),
		ProyectoID:  &proyecto.ID,
		RuntimeID:   runtimeID,
		HandleID:    handleID,
		Tipo:        "nudge",
		PayloadJSON: string(payloadJSON),
	})
	if err != nil {
		return false, err
	}
	db.Audit("orquesta", "autonomia_nudge", "runtime_order", orderID,
		fmt.Sprintf("agente=%s proyecto=%s accion=%s", agente, proyecto.Slug, accion))
	return true, nil
}

func existeRuntimeMailboxAutonomiaPendiente(agente string, proyectoID *int64, accion string) (bool, error) {
	estado := "pendiente"
	agente = strings.TrimSpace(agente)
	accion = strings.TrimSpace(accion)
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &agente,
		ProyectoID: proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return false, err
	}
	for _, msg := range mailbox {
		if msg == nil || strings.TrimSpace(msg.Kind) != "autonomia" {
			continue
		}
		payload := map[string]any{}
		_ = json.Unmarshal([]byte(msg.PayloadJSON), &payload)
		if strings.TrimSpace(stringMapValue(payload, "accion")) == accion {
			return true, nil
		}
	}
	return false, nil
}
