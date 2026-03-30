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

type OpenClawGatewayNotificador struct {
	URL      string
	Token    string
	Operator string
	Client   httpDoer
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
		items        []Notificador
		telegram     *TelegramNotificador
		labels       []string
		gatewayURL   = strings.TrimSpace(configValue("openclaw_gateway_url"))
		gatewayToken = strings.TrimSpace(configValue("openclaw_gateway_token"))
		operator     = strings.TrimSpace(configValue("openclaw_gateway_operator"))
	)
	if gatewayURL != "" {
		items = append(items, &OpenClawGatewayNotificador{
			URL:      gatewayURL,
			Token:    gatewayToken,
			Operator: operator,
		})
		labels = append(labels, "OpenClaw Gateway activo.")
	}
	if bot := configuredTelegramNotificador(); bot != nil {
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

func configuredTelegramNotificador() *TelegramNotificador {
	token := strings.TrimSpace(configValue("telegram_token"))
	chatID := strings.TrimSpace(configValue("telegram_chat_id"))
	if token == "" || chatID == "" {
		return nil
	}
	return &TelegramNotificador{Token: token, ChatID: chatID}
}

func configValue(key string) string {
	v, _ := db.ConfigGet(key)
	return v
}
