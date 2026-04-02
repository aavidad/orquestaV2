package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"orquesta/db"
	"orquesta/notificaciones"
)

type openClawNormalizedEvent struct {
	Source          string    `json:"source"`
	NormalizedEvent string    `json:"normalized_event"`
	RawType         string    `json:"raw_type,omitempty"`
	Resource        string    `json:"resource,omitempty"`
	Status          string    `json:"status,omitempty"`
	Severity        string    `json:"severity,omitempty"`
	Agent           string    `json:"agent,omitempty"`
	Project         string    `json:"project,omitempty"`
	Message         string    `json:"message,omitempty"`
	SuggestedAction string    `json:"suggested_action,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

func buildOpenClawNormalizedEvents(limit int) ([]openClawNormalizedEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	gates, err := db.ListarReviewGates(db.FiltroReviewGates{Limit: limit})
	if err != nil {
		return nil, err
	}
	signals, err := listarSignalsRevisionSupervisor(limit)
	if err != nil {
		return nil, err
	}
	merges, err := listarMergesRevisionSupervisor(limit)
	if err != nil {
		return nil, err
	}
	outbox := notificaciones.DescribirOutbox(limit)
	return buildOpenClawNormalizedEventsFromData(gates, signals, merges, outbox.Recientes, limit), nil
}

func buildOpenClawNormalizedEventsFromData(gates []*db.ReviewGate, signals []*supervisorReviewSignal, merges []*db.GitMerge, deliveries []*db.EntregaNotificacion, limit int) []openClawNormalizedEvent {
	if limit <= 0 {
		limit = 20
	}
	items := make([]openClawNormalizedEvent, 0, len(gates)+len(signals)+len(merges)+len(deliveries))
	for _, gate := range gates {
		if gate == nil || gate.Estado == db.ReviewGateAprobado {
			continue
		}
		items = append(items, normalizeOpenClawReviewGate(gate))
	}
	for _, signal := range signals {
		if signal == nil || signal.Event == nil {
			continue
		}
		items = append(items, normalizeOpenClawSignal(signal))
	}
	for _, merge := range merges {
		if merge == nil {
			continue
		}
		items = append(items, normalizeOpenClawMerge(merge))
	}
	for _, entrega := range deliveries {
		if entrega == nil {
			continue
		}
		items = append(items, normalizeOpenClawDelivery(entrega))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func normalizeOpenClawReviewGate(gate *db.ReviewGate) openClawNormalizedEvent {
	ev := openClawNormalizedEvent{
		Source:    "review_gate",
		RawType:   strings.TrimSpace(string(gate.Estado)),
		Resource:  fmt.Sprintf("review_gate:%d", gate.ID),
		Severity:  strings.TrimSpace(gate.SeverityMax),
		Agent:     strings.TrimSpace(gate.ReviewerAgente),
		CreatedAt: gate.UpdatedAt,
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = gate.CreatedAt
	}
	if gate.ProyectoID != nil && *gate.ProyectoID > 0 {
		if proyecto, err := db.GetProyecto(fmt.Sprintf("%d", *gate.ProyectoID)); err == nil && proyecto != nil {
			ev.Project = strings.TrimSpace(proyecto.Slug)
		}
	}
	switch gate.Estado {
	case db.ReviewGatePendiente:
		ev.NormalizedEvent = "review.gate_open"
		ev.Status = "open"
		ev.SuggestedAction = "resolver_o_asignar_review"
	case db.ReviewGateEnRevision:
		ev.NormalizedEvent = "review.in_progress"
		ev.Status = "in_progress"
		ev.SuggestedAction = "supervisar_revision"
	case db.ReviewGateCambiosPed:
		ev.NormalizedEvent = "review.changes_requested"
		ev.Status = "changes_requested"
		ev.SuggestedAction = "relanzar_cambios"
	case db.ReviewGateBloqueado:
		ev.NormalizedEvent = "review.blocked"
		ev.Status = "blocked"
		ev.SuggestedAction = "desbloquear_gate"
	default:
		ev.NormalizedEvent = "review.unknown"
		ev.Status = strings.TrimSpace(string(gate.Estado))
		ev.SuggestedAction = "inspeccionar_gate"
	}
	if gate.TareaID != nil && *gate.TareaID > 0 {
		ev.Message = fmt.Sprintf("gate #%d sobre tarea %d", gate.ID, *gate.TareaID)
	} else {
		ev.Message = fmt.Sprintf("gate #%d", gate.ID)
	}
	return ev
}

func normalizeOpenClawSignal(item *supervisorReviewSignal) openClawNormalizedEvent {
	ev := openClawNormalizedEvent{
		Source:    "runtime_signal",
		RawType:   strings.TrimSpace(item.Event.Kind),
		Resource:  fmt.Sprintf("runtime_event:%d", item.Event.ID),
		Agent:     strings.TrimSpace(item.Agent),
		Project:   strings.TrimSpace(item.Project),
		Message:   compactMCPLine(item.Event.Message, 180),
		CreatedAt: item.Event.CreatedAt,
	}
	switch strings.TrimSpace(item.Event.Kind) {
	case "approval_request":
		ev.NormalizedEvent = "supervisor.approval_required"
		ev.Status = "pending"
		ev.SuggestedAction = "arbitrar_y_desbloquear"
	case "waiting_human":
		ev.NormalizedEvent = "supervisor.input_required"
		ev.Status = "blocked"
		ev.SuggestedAction = "reencuadrar_o_pedir_contexto"
	case "ready_for_review":
		ev.NormalizedEvent = "review.ready"
		ev.Status = "ready"
		ev.SuggestedAction = "inspeccionar_y_decidir_gate"
	default:
		ev.NormalizedEvent = "supervisor.signal"
		ev.Status = strings.TrimSpace(item.Event.Level)
		ev.SuggestedAction = "inspeccionar_signal"
	}
	return ev
}

func normalizeOpenClawMerge(merge *db.GitMerge) openClawNormalizedEvent {
	ev := openClawNormalizedEvent{
		Source:    "merge",
		RawType:   strings.TrimSpace(merge.Estado),
		Resource:  fmt.Sprintf("git_merge:%d", merge.ID),
		Agent:     strings.TrimSpace(merge.RequestedBy),
		Project:   strings.TrimSpace(merge.ProyectoSlug),
		Message:   compactMCPLine(strings.TrimSpace(merge.SourceBranch)+" -> "+strings.TrimSpace(merge.TargetBranch), 180),
		CreatedAt: merge.UpdatedAt,
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = merge.CreatedAt
	}
	switch strings.TrimSpace(merge.Estado) {
	case "pendiente":
		ev.NormalizedEvent = "merge.pending"
		ev.Status = "pending"
		ev.SuggestedAction = "priorizar_merge"
	case "validando":
		ev.NormalizedEvent = "merge.validating"
		ev.Status = "validating"
		ev.SuggestedAction = "esperar_validacion"
	case "aprobado":
		ev.NormalizedEvent = "merge.approved"
		ev.Status = "approved"
		ev.SuggestedAction = "ejecutar_merge"
	case "ejecutando":
		ev.NormalizedEvent = "merge.executing"
		ev.Status = "running"
		ev.SuggestedAction = "vigilar_merge"
	case "fallido":
		ev.NormalizedEvent = "merge.failed"
		ev.Status = "failed"
		ev.SuggestedAction = "revisar_error_merge"
	default:
		ev.NormalizedEvent = "merge.unknown"
		ev.Status = strings.TrimSpace(merge.Estado)
		ev.SuggestedAction = "inspeccionar_merge"
	}
	return ev
}

func normalizeOpenClawDelivery(entrega *db.EntregaNotificacion) openClawNormalizedEvent {
	ev := openClawNormalizedEvent{
		Source:    "notification",
		RawType:   strings.TrimSpace(entrega.TipoEvento),
		Resource:  fmt.Sprintf("delivery:%d", entrega.ID),
		Status:    strings.TrimSpace(string(entrega.Estado)),
		Message:   compactMCPLine(strings.TrimSpace(entrega.Evento.Texto), 180),
		CreatedAt: entrega.UpdatedAt,
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = entrega.CreatedAt
	}
	switch entrega.Estado {
	case db.EntregaNotificacionPendiente:
		ev.NormalizedEvent = "notification.pending"
		ev.SuggestedAction = "vigilar_entrega"
	case db.EntregaNotificacionFallida:
		ev.NormalizedEvent = "notification.failed"
		ev.SuggestedAction = "reintentar_o_revisar_gateway"
	case db.EntregaNotificacionEntregada:
		ev.NormalizedEvent = "notification.delivered"
		ev.SuggestedAction = "sin_accion"
	default:
		ev.NormalizedEvent = "notification.unknown"
		ev.SuggestedAction = "inspeccionar_entrega"
	}
	return ev
}
