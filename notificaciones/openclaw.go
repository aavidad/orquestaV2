/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package notificaciones

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"orquesta/db"
	"strings"
	"time"
)

type EventAware interface {
	EnviarEvento(ev db.EventoNotificacion) error
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type EstadoCanalNotificacion struct {
	Nombre  string `json:"nombre"`
	Activo  bool   `json:"activo"`
	Detalle string `json:"detalle,omitempty"`
}

type EstadoNotificaciones struct {
	Canales []EstadoCanalNotificacion `json:"canales"`
}

type OpenClawGatewayNotificador struct {
	URL      string
	Token    string
	Operator string
	Client   httpDoer
}

type OutboxSummary struct {
	Activas   []*db.EntregaNotificacion `json:"activas,omitempty"`
	Recientes []*db.EntregaNotificacion `json:"recientes,omitempty"`
}

type notificationConfigSnapshot struct {
	gatewayURL      string
	gatewayToken    string
	gatewayOperator string
	telegramToken   string
	telegramChatID  string
}

type fanoutNotificador struct {
	items []Notificador
}

func (n *fanoutNotificador) EnviarMensaje(texto string) error {
	return n.each(func(item Notificador) error { return item.EnviarMensaje(texto) })
}

func (n *fanoutNotificador) EnviarAlertaBloqueo(tareaID int64, agente, motivo string) error {
	return n.each(func(item Notificador) error { return item.EnviarAlertaBloqueo(tareaID, agente, motivo) })
}

func (n *fanoutNotificador) EnviarPropuestaVotacion(codigo, titulo string) error {
	return n.each(func(item Notificador) error { return item.EnviarPropuestaVotacion(codigo, titulo) })
}

func (n *fanoutNotificador) EnviarAvisoFinProyecto(proyectoID int64, nombre string) error {
	return n.each(func(item Notificador) error { return item.EnviarAvisoFinProyecto(proyectoID, nombre) })
}

func (n *fanoutNotificador) EnviarEvento(ev db.EventoNotificacion) error {
	return n.each(func(item Notificador) error {
		if aware, ok := item.(EventAware); ok {
			return aware.EnviarEvento(ev)
		}
		return dispatchLegacyEvent(item, ev)
	})
}

func (n *fanoutNotificador) each(fn func(item Notificador) error) error {
	if n == nil {
		return nil
	}
	var errs []string
	for _, item := range n.items {
		if item == nil {
			continue
		}
		if err := fn(item); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

func (n *OpenClawGatewayNotificador) EnviarMensaje(texto string) error {
	return n.EnviarEvento(db.EventoNotificacion{
		Tipo:  "mensaje",
		Texto: strings.TrimSpace(texto),
	})
}

func (n *OpenClawGatewayNotificador) EnviarAlertaBloqueo(tareaID int64, agente, motivo string) error {
	return n.EnviarEvento(db.EventoNotificacion{
		Tipo:   "bloqueo",
		ID:     tareaID,
		Agente: strings.TrimSpace(agente),
		Texto:  strings.TrimSpace(motivo),
	})
}

func (n *OpenClawGatewayNotificador) EnviarPropuestaVotacion(codigo, titulo string) error {
	return n.EnviarEvento(db.EventoNotificacion{
		Tipo:   "propuesta",
		Codigo: strings.TrimSpace(codigo),
		Texto:  strings.TrimSpace(titulo),
	})
}

func (n *OpenClawGatewayNotificador) EnviarAvisoFinProyecto(proyectoID int64, nombre string) error {
	return n.EnviarEvento(db.EventoNotificacion{
		Tipo:       "fin_proyecto",
		ID:         proyectoID,
		ProyectoID: proyectoID,
		Texto:      strings.TrimSpace(nombre),
	})
}

func (n *OpenClawGatewayNotificador) EnviarEvento(ev db.EventoNotificacion) error {
	curated, eventType, ok := curateOpenClawEvent(ev)
	if !ok {
		return nil
	}
	entregaID, err := db.CrearEntregaNotificacion("openclaw_gateway", strings.TrimSpace(n.URL), curated)
	if err != nil {
		return err
	}
	err = n.enviarEventoHTTP(curated, eventType)
	if err != nil {
		nextRetryAt := time.Now().UTC().Add(notificationRetryDelay(1))
		_ = db.MarcarEntregaNotificacionFallida(entregaID, err.Error(), nextRetryAt)
		return err
	}
	_ = db.MarcarEntregaNotificacionEntregada(entregaID)
	return nil
}

func (n *OpenClawGatewayNotificador) enviarEventoHTTP(ev db.EventoNotificacion, eventType string) error {
	url := strings.TrimSpace(n.URL)
	if url == "" {
		return nil
	}
	if strings.TrimSpace(eventType) == "" {
		eventType = mapNotificationEventType(ev)
	}
	payload := map[string]any{
		"source":     "orquesta",
		"event_type": eventType,
		"event_id":   ev.ID,
		"text":       strings.TrimSpace(ev.Texto),
		"sent_at":    time.Now().UTC().Format(time.RFC3339),
	}
	if operator := strings.TrimSpace(n.Operator); operator != "" {
		payload["operator"] = operator
	}
	if agente := strings.TrimSpace(ev.Agente); agente != "" {
		payload["agent"] = agente
	}
	if codigo := strings.TrimSpace(ev.Codigo); codigo != "" {
		payload["proposal_code"] = codigo
	}
	if ev.ProyectoID > 0 {
		payload["project_id"] = ev.ProyectoID
	}
	if len(ev.Payload) > 0 {
		payload["payload"] = ev.Payload
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Orquesta-Source", "orquesta")
	req.Header.Set("X-Orquesta-Event-Type", eventType)
	if token := strings.TrimSpace(n.Token); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := n.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error openclaw gateway (status %d): %s", resp.StatusCode, strings.TrimSpace(string(rb)))
	}
	return nil
}

func curateOpenClawEvent(ev db.EventoNotificacion) (db.EventoNotificacion, string, bool) {
	eventType, ok := openClawEventType(ev)
	if !ok {
		return ev, "", false
	}
	ev.Tipo = strings.TrimSpace(ev.Tipo)
	ev.Texto = decorateOpenClawEventText(ev, eventType)
	return ev, eventType, true
}

func openClawEventType(ev db.EventoNotificacion) (string, bool) {
	switch strings.TrimSpace(ev.Tipo) {
	case "bloqueo", "hook:project_blocked":
		return "task_blocked", true
	case "hook:project_unblocked", "progress_summary":
		return "progress_summary", true
	case "hook:task_finish":
		return "task_completed", true
	case "fin_proyecto":
		return "project_completed", true
	case "runtime_handoff":
		return "handoff_completed", true
	case "runtime_failure":
		switch strings.TrimSpace(payloadString(ev.Payload, "classification")) {
		case "runtime_panic":
			return "runtime_panic", true
		case "runtime_crash":
			return "runtime_crash", true
		default:
			return "runtime_failure", true
		}
	case "runtime_review":
		switch strings.TrimSpace(payloadString(ev.Payload, "stage")) {
		case "ready_for_review":
			return "review_ready", true
		case "review_approved", "aprobado":
			return "review_approved", true
		case "review_changes_requested", "cambios_pedidos":
			return "review_changes_requested", true
		case "review_blocked", "bloqueado":
			return "review_blocked", true
		default:
			return "review_update", true
		}
	default:
		return "", false
	}
}

func decorateOpenClawEventText(ev db.EventoNotificacion, eventType string) string {
	base := strings.TrimSpace(ev.Texto)
	agente := strings.TrimSpace(ev.Agente)
	switch eventType {
	case "task_blocked":
		prefix := "Bloqueo detectado"
		if ev.ID > 0 {
			prefix = fmt.Sprintf("Tarea #%d bloqueada", ev.ID)
		}
		return joinOpenClawTextParts(prefix, openClawAgentLabel(agente), base)
	case "task_completed":
		prefix := "Tarea completada"
		if ev.ID > 0 {
			prefix = fmt.Sprintf("Tarea #%d completada", ev.ID)
		}
		return joinOpenClawTextParts(prefix, openClawAgentLabel(agente), base)
	case "progress_summary":
		prefix := "Resumen de progreso"
		if strings.TrimSpace(ev.Tipo) == "hook:project_unblocked" {
			prefix = "Bloqueo resuelto"
		}
		return joinOpenClawTextParts(prefix, openClawAgentLabel(agente), base)
	case "project_completed":
		prefix := "Proyecto completado"
		if ev.ID > 0 {
			prefix = fmt.Sprintf("Proyecto #%d completado", ev.ID)
		}
		return joinOpenClawTextParts(prefix, base)
	default:
		if base != "" {
			return base
		}
		return "Notificacion de operador"
	}
}

func openClawAgentLabel(agente string) string {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return ""
	}
	return "agente=" + agente
}

func joinOpenClawTextParts(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return strings.Join(out, " | ")
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func DescribirConfiguracion() EstadoNotificaciones {
	cfg := readNotificationConfigSnapshot()
	return EstadoNotificaciones{
		Canales: []EstadoCanalNotificacion{
			describirCanalOpenClaw(cfg),
			describirCanalTelegram(cfg),
		},
	}
}

func DescribirOutbox(limit int) OutboxSummary {
	if limit <= 0 {
		limit = 10
	}
	activas, _ := db.ListarEntregasNotificacion(db.FiltroEntregasNotificacion{
		Canal:       "openclaw_gateway",
		ActivasOnly: true,
		Limit:       limit,
	})
	recientes, _ := db.ListarEntregasNotificacion(db.FiltroEntregasNotificacion{
		Canal: "openclaw_gateway",
		Limit: limit,
	})
	return OutboxSummary{Activas: activas, Recientes: recientes}
}

func (n *OpenClawGatewayNotificador) client() httpDoer {
	if n != nil && n.Client != nil {
		return n.Client
	}
	return http.DefaultClient
}

func dispatchLegacyEvent(n Notificador, ev db.EventoNotificacion) error {
	if n == nil {
		return nil
	}
	switch strings.TrimSpace(ev.Tipo) {
	case "bloqueo":
		return n.EnviarAlertaBloqueo(ev.ID, ev.Agente, ev.Texto)
	case "propuesta":
		return n.EnviarPropuestaVotacion(ev.Codigo, ev.Texto)
	case "fin_proyecto":
		nombre := strings.TrimSpace(ev.Texto)
		if nombre == "" {
			nombre = strings.TrimSpace(ev.Codigo)
		}
		return n.EnviarAvisoFinProyecto(ev.ID, nombre)
	default:
		return n.EnviarMensaje(formatNotificationMessage(ev))
	}
}

func formatNotificationMessage(ev db.EventoNotificacion) string {
	texto := strings.TrimSpace(ev.Texto)
	if texto != "" {
		switch strings.TrimSpace(ev.Tipo) {
		case "runtime_auto_guidance":
			return "OpenClaw Gateway: " + texto
		case "":
			return texto
		default:
			return fmt.Sprintf("[%s] %s", strings.TrimSpace(ev.Tipo), texto)
		}
	}
	tipo := strings.TrimSpace(ev.Tipo)
	if tipo == "" {
		return "Notificación de Orquesta"
	}
	return "Notificación de Orquesta: " + tipo
}

func mapNotificationEventType(ev db.EventoNotificacion) string {
	if eventType, ok := openClawEventType(ev); ok {
		return eventType
	}
	return "message"
}

func buildConfiguredNotifiers() (Notificador, *TelegramNotificador, []string) {
	var (
		items    []Notificador
		telegram *TelegramNotificador
		labels   []string
		cfg      = readNotificationConfigSnapshot()
	)
	if cfg.gatewayURL != "" {
		items = append(items, &OpenClawGatewayNotificador{
			URL:      cfg.gatewayURL,
			Token:    cfg.gatewayToken,
			Operator: cfg.gatewayOperator,
		})
		labels = append(labels, "OpenClaw Gateway activo.")
	}
	if bot := configuredTelegramNotificador(cfg); bot != nil {
		telegram = bot
		items = append(items, bot)
		labels = append(labels, "Notificaciones de Telegram activas.")
	}
	switch len(items) {
	case 0:
		return nil, nil, nil
	case 1:
		return items[0], telegram, labels
	default:
		return &fanoutNotificador{items: items}, telegram, labels
	}
}

func RetryDueGatewayDeliveries(n Notificador, limit int) (int, error) {
	if limit <= 0 {
		limit = 10
	}
	gateway := extractOpenClawGatewayNotifier(n)
	if gateway == nil {
		return 0, nil
	}
	items, err := db.ListarEntregasNotificacion(db.FiltroEntregasNotificacion{
		Canal:       "openclaw_gateway",
		ActivasOnly: true,
		DueOnly:     true,
		Limit:       limit,
	})
	if err != nil {
		return 0, err
	}
	processed := 0
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if item == nil {
			continue
		}
		intentos, err := db.IncrementarIntentoEntregaNotificacion(item.ID)
		if err != nil {
			return processed, err
		}
		curated, eventType, ok := curateOpenClawEvent(item.Evento)
		if !ok {
			if err := db.MarcarEntregaNotificacionEntregada(item.ID); err != nil {
				return processed, err
			}
			processed++
			continue
		}
		if err := gateway.enviarEventoHTTP(curated, eventType); err != nil {
			nextRetryAt := time.Now().UTC().Add(notificationRetryDelay(intentos))
			if markErr := db.MarcarEntregaNotificacionFallida(item.ID, err.Error(), nextRetryAt); markErr != nil {
				return processed, markErr
			}
			processed++
			continue
		}
		if err := db.MarcarEntregaNotificacionEntregada(item.ID); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func extractOpenClawGatewayNotifier(n Notificador) *OpenClawGatewayNotificador {
	switch v := n.(type) {
	case *OpenClawGatewayNotificador:
		return v
	case *fanoutNotificador:
		for _, item := range v.items {
			if gateway := extractOpenClawGatewayNotifier(item); gateway != nil {
				return gateway
			}
		}
	}
	return nil
}

func notificationRetryDelay(intentos int) time.Duration {
	if intentos <= 1 {
		return 30 * time.Second
	}
	delay := time.Duration(intentos*intentos) * time.Minute
	if delay > 15*time.Minute {
		return 15 * time.Minute
	}
	return delay
}

func configuredTelegramNotificador(cfg notificationConfigSnapshot) *TelegramNotificador {
	if cfg.telegramToken == "" || cfg.telegramChatID == "" {
		return nil
	}
	return &TelegramNotificador{Token: cfg.telegramToken, ChatID: cfg.telegramChatID}
}

func readNotificationConfigSnapshot() notificationConfigSnapshot {
	return notificationConfigSnapshot{
		gatewayURL:      strings.TrimSpace(configValue("openclaw_gateway_url")),
		gatewayToken:    strings.TrimSpace(configValue("openclaw_gateway_token")),
		gatewayOperator: strings.TrimSpace(configValue("openclaw_gateway_operator")),
		telegramToken:   strings.TrimSpace(configValue("telegram_token")),
		telegramChatID:  strings.TrimSpace(configValue("telegram_chat_id")),
	}
}

func describirCanalOpenClaw(cfg notificationConfigSnapshot) EstadoCanalNotificacion {
	if cfg.gatewayURL == "" {
		return EstadoCanalNotificacion{
			Nombre:  "OpenClaw Gateway",
			Activo:  false,
			Detalle: "sin openclaw_gateway_url",
		}
	}
	detalle := cfg.gatewayURL
	partes := []string{detalle}
	if cfg.gatewayOperator != "" {
		partes = append(partes, "operador="+cfg.gatewayOperator)
	}
	if cfg.gatewayToken != "" {
		partes = append(partes, "token=configurado")
	} else {
		partes = append(partes, "token=vacío")
	}
	return EstadoCanalNotificacion{
		Nombre:  "OpenClaw Gateway",
		Activo:  true,
		Detalle: strings.Join(partes, " · "),
	}
}

func describirCanalTelegram(cfg notificationConfigSnapshot) EstadoCanalNotificacion {
	if cfg.telegramToken == "" || cfg.telegramChatID == "" {
		return EstadoCanalNotificacion{
			Nombre:  "Telegram",
			Activo:  false,
			Detalle: "sin telegram_token o telegram_chat_id",
		}
	}
	return EstadoCanalNotificacion{
		Nombre:  "Telegram",
		Activo:  true,
		Detalle: "chat_id=" + cfg.telegramChatID + " · token=configurado",
	}
}

func configValue(key string) string {
	v, _ := db.ConfigGet(key)
	return v
}
