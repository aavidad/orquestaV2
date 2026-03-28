/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"orquesta/db"
	"orquesta/notificaciones"
	"orquesta/planocontrol"
	"orquesta/reviewapp"
	"orquesta/tareasapp"
)

type dbAutomationService struct{}

const autonomiaReplanTaskTitle = "Autonomía: replanificar backlog y abrir siguiente frente útil"

func (dbAutomationService) CheckReanimaciones() ([]*db.Agente, error) {
	return db.CheckReanimaciones()
}

func (dbAutomationService) ResetReanimacion(nombre string) error {
	if err := db.ResetReanimacion(nombre); err != nil {
		return err
	}
	return reactivarAgenteTrasReanimacion(strings.TrimSpace(nombre))
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

func (dbAutomationService) ReconciliarRuntimeHandlesStale() (int, error) {
	return db.ReconciliarRuntimeHandlesStale()
}

func (dbAutomationService) ReconciliarRuntimeOrdersStale() (int, error) {
	return db.ReconciliarRuntimeOrdersStale()
}

func (dbAutomationService) ProcesarRuntimeTranscriptBatch() (int, error) {
	return procesarRuntimeTranscriptBatch()
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
	mailbox, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		return mailbox, err
	}
	ingested, err := db.IngestarRuntimeTranscriptActivos()
	if err != nil {
		return mailbox + ingested, err
	}
	if !controlPlaneConfigBoolOrDefault("runtime_transcript_auto_guidance_enabled", true) {
		return mailbox + ingested, nil
	}
	signals, err := db.ListarRuntimeTranscript(db.FiltroRuntimeTranscript{
		SoloSenalesPend: true,
		Limit:           50,
	})
	if err != nil {
		return mailbox + ingested, err
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
				return mailbox + ingested + processedSignals, err
			}
		}
		processedSignals++
	}
	return mailbox + ingested + processedSignals, nil
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
		if supervisorNote, err := notificarSupervisorSignalTranscript(item); err != nil {
			return "", err
		} else if strings.TrimSpace(supervisorNote) != "" {
			return supervisorNote + ";runtime_failure_signal", nil
		}
		return "runtime_failure_signal", nil
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
	payloadEvent, _ := json.Marshal(map[string]any{
		"transcript_id":    item.ID,
		"runtime_order_id": orderID,
		"classification":   strings.TrimSpace(item.Classification),
	})
	_, _ = db.RegistrarRuntimeEvent(&db.RuntimeEvent{
		RuntimeID:   item.RuntimeID,
		Kind:        "auto_guidance_sent",
		Level:       "info",
		Message:     fmt.Sprintf("Orquesta respondió a la señal %s", strings.TrimSpace(item.Classification)),
		PayloadJSON: string(payloadEvent),
	})
	notes := []string{fmt.Sprintf("auto_guidance_order:%d", orderID)}
	if supervisorNote, err := notificarSupervisorSignalTranscript(item); err != nil {
		return "", err
	} else if strings.TrimSpace(supervisorNote) != "" {
		notes = append(notes, strings.TrimSpace(supervisorNote))
	}
	return strings.Join(notes, ";"), nil
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
	base := "Orquesta: continúa de forma autónoma dentro de la gobernanza efectiva del proyecto. No necesitas aprobación humana salvo que falten credenciales, secretos o un recurso externo real. Si el cambio es delicado, crea checkpoint y sigue."
	switch strings.TrimSpace(item.Classification) {
	case "approval_request":
		return base + " Si dudas entre varias opciones seguras, elige la más alineada con el proyecto y continúa sin detenerte."
	case "waiting_human":
		return base + " No te quedes esperando respuesta: formula el siguiente paso razonable, ejecuta y documenta los supuestos."
	case "ready_for_review":
		return base + " Deja un resumen breve, asegúrate de que el frente queda verificable y sigue disponible para que Orquesta relance review si procede."
	case "needs_replan":
		return base + " Si no ves el siguiente paso, revisa backlog, tareas, propuestas y último checkpoint, y continúa por el frente más útil disponible."
	case "blocked":
		return base + " Si el bloqueo es de contexto, reevalúa tareas, propuestas y estado del proyecto y avanza por el mejor siguiente paso disponible."
	default:
		return base
	}
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
	supervisor, activado, err := prepararAgentePreferidoAutonomia(proyecto.ID, strings.TrimSpace(policy.SupervisorAgente), "supervision_transcript_signal")
	if err != nil {
		return "", err
	}
	if activado {
		notes = append(notes, "supervisor_start")
	}
	if supervisor == nil {
		supervisor, err = seleccionarAgenteActivoProyectoPreferido(proyecto.ID, strings.TrimSpace(policy.SupervisorAgente), []string{"supervisor", "orquestador", "revisor", "reviewer", "admin", "programador"}, strings.TrimSpace(item.Agente))
		if err != nil {
			return "", err
		}
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
		handle, err := runtimesService.GetActiveRuntimeHandleAgentProject(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return total, err
		}
		if handle == nil {
			continue
		}
		payload := map[string]any{
			"to_agente":    strings.TrimSpace(msg.ToAgente),
			"from_agente":  strings.TrimSpace(msg.FromAgente),
			"texto":        texto,
			"mailbox_id":   msg.ID,
			"mailbox_kind": strings.TrimSpace(msg.Kind),
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return total, err
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
		orderID, err := db.EncolarRuntimeOrder(order)
		if err != nil {
			return total, err
		}
		if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return total, err
		}
		if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return total, err
		}
		db.Audit("orquesta", "runtime_mailbox_interactivo", "runtime_order", orderID,
			fmt.Sprintf("mailbox_id=%d agente=%s kind=%s", msg.ID, strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
		total++
	}
	return total, nil
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
		if pendiente, err := existeRuntimeOrderAutonomiaPendiente(sesion.Agente, &proyecto.ID, "pause", ""); err != nil {
			return 0, err
		} else if pendiente {
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
	if pendiente, err := existeRuntimeOrderAutonomiaPendiente(sesion.Agente, &proyecto.ID, "pause", ""); err != nil {
		return 0, err
	} else if pendiente {
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
	return true, motivo, nil
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
	if pendiente, err := existeRuntimeOrderAutonomiaPendiente(sesion.Agente, sesion.ProyectoID, "pause", ""); err != nil {
		return 0, err
	} else if pendiente {
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
			if pendiente, err := existeRuntimeOrderAutonomiaPendiente(sesion.Agente, sesion.ProyectoID, "pause", ""); err != nil {
				return 0, err
			} else if pendiente {
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
	}
	payload := map[string]any{
		"from_agente": "server",
		"to_agente":   strings.TrimSpace(agente),
		"kind":        "autonomia",
		"accion":      strings.TrimSpace(accion),
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
		fmt.Sprintf("agente=%s proyecto=%s accion=%s", strings.TrimSpace(agente), proyecto.Slug, strings.TrimSpace(accion)))
	return true, nil
}
