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
	entregaID, err := db.CrearEntregaNotificacion("openclaw_gateway", strings.TrimSpace(n.URL), ev)
	if err != nil {
		return err
	}
	err = n.enviarEventoHTTP(ev)
	if err != nil {
		nextRetryAt := time.Now().UTC().Add(notificationRetryDelay(1))
		_ = db.MarcarEntregaNotificacionFallida(entregaID, err.Error(), nextRetryAt)
		return err
	}
	_ = db.MarcarEntregaNotificacionEntregada(entregaID)
	return nil
}

func (n *OpenClawGatewayNotificador) enviarEventoHTTP(ev db.EventoNotificacion) error {
	url := strings.TrimSpace(n.URL)
	if url == "" {
		return nil
	}
	payload := map[string]any{
		"source":     "orquesta",
		"event_type": mapNotificationEventType(ev.Tipo),
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
	req.Header.Set("X-Orquesta-Event-Type", mapNotificationEventType(ev.Tipo))
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

func mapNotificationEventType(tipo string) string {
	switch strings.TrimSpace(tipo) {
	case "bloqueo":
		return "task_blocked"
	case "propuesta":
		return "proposal_vote_requested"
	case "fin_proyecto":
		return "project_completed"
	case "runtime_auto_guidance":
		return "runtime_auto_guidance"
	default:
		return "message"
	}
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
		if err := gateway.enviarEventoHTTP(item.Evento); err != nil {
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
