package runtimesapp

import (
	"encoding/json"
	"strings"
)

type EstadoDispatchDurable string

const (
	DispatchPendiente  EstadoDispatchDurable = "pending"
	DispatchNotificada EstadoDispatchDurable = "notified"
	DispatchEntregada  EstadoDispatchDurable = "delivered"
	DispatchFallida    EstadoDispatchDurable = "failed"
)

type EntradaResultadoDispatchDurable struct {
	EstadoDispatch EstadoDispatchDurable
	EstadoEntrega  string
	ReceiptSource  string
	UltimaRazon    string
}

func (e EstadoDispatchDurable) normalizado() EstadoDispatchDurable {
	switch strings.ToLower(strings.TrimSpace(string(e))) {
	case string(DispatchPendiente):
		return DispatchPendiente
	case string(DispatchNotificada):
		return DispatchNotificada
	case string(DispatchEntregada):
		return DispatchEntregada
	case string(DispatchFallida):
		return DispatchFallida
	default:
		return ""
	}
}

func fusionarResultadoDispatchDurable(base string, entrada EntradaResultadoDispatchDurable) string {
	resultado := map[string]any{}
	if strings.TrimSpace(base) != "" {
		_ = json.Unmarshal([]byte(base), &resultado)
	}
	if estado := entrada.EstadoDispatch.normalizado(); estado != "" {
		resultado["dispatch_state"] = string(estado)
	}
	if v := strings.TrimSpace(entrada.EstadoEntrega); v != "" {
		resultado["delivery_state"] = v
	}
	if v := strings.TrimSpace(entrada.ReceiptSource); v != "" {
		resultado["receipt_source"] = v
	}
	if v := strings.TrimSpace(entrada.UltimaRazon); v != "" {
		resultado["last_reason"] = v
	}
	raw, err := json.Marshal(resultado)
	if err != nil {
		return strings.TrimSpace(base)
	}
	return string(raw)
}
