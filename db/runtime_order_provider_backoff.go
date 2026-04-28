package db

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func gestionarBackoffProveedorRuntimeOrderSendInstruction(order *RuntimeOrder, payload map[string]any, runtimeErr error) (bool, error) {
	delay, motivo, ok := runtimeOrderProviderBackoff(runtimeErr)
	if !ok {
		return false, nil
	}
	minutos := int((delay + time.Minute - 1) / time.Minute)
	if minutos < 1 {
		minutos = 1
	}
	motivoPausa := "Auto-pausa por agotamiento: " + motivo
	if err := PausarAgente(strings.TrimSpace(order.Agente), minutos, motivoPausa); err != nil {
		return true, err
	}
	if err := registrarPresupuestoSesionProviderBackoff(order, runtimeErr, delay, motivo); err != nil {
		return true, err
	}
	detalle := fmt.Sprintf("agente en enfriamiento por %s", motivo)
	if runtimeOrderSendInstructionProvieneMailbox(payload) {
		if mailboxID := runtimeOrderSendInstructionMailboxID(payload); mailboxID > 0 {
			msg, err := GetRuntimeMailbox(mailboxID)
			if err != nil {
				return true, err
			}
			if msg != nil && !strings.EqualFold(strings.TrimSpace(msg.Estado), "pendiente") {
				return true, completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, detalle+"; mailbox ya "+strings.TrimSpace(msg.Estado))
			}
		}
		return true, reencolarRuntimeOrderSendInstructionAt(order, payload, detalle, time.Now().UTC().Add(delay))
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"ok":               false,
		"provider_backoff": true,
		"pause_minutes":    minutos,
		"pause_reason":     motivo,
	})
	return true, MarcarRuntimeOrderEstado(order.ID, "fallida", resultado, detalle)
}

func runtimeOrderProviderBackoff(err error) (time.Duration, string, bool) {
	if err == nil {
		return 0, "", false
	}
	raw := strings.TrimSpace(err.Error())
	if raw == "" {
		return 0, "", false
	}
	lower := strings.ToLower(raw)
	if !strings.Contains(lower, "hit your usage limit") &&
		!strings.Contains(lower, "usage limit") &&
		!strings.Contains(lower, "rate limit") &&
		!strings.Contains(lower, "purchase more credits") &&
		!strings.Contains(lower, "try again at") {
		return 0, "", false
	}
	motivo := "cuota de proveedor"
	switch {
	case strings.Contains(lower, "rate limit"):
		motivo = "rate limit de proveedor"
	case strings.Contains(lower, "usage limit"), strings.Contains(lower, "purchase more credits"):
		motivo = "usage limit de proveedor"
	}
	delay := 60 * time.Minute
	if parsed, ok := runtimeOrderProviderRetryDelay(lower, time.Now()); ok && parsed > 0 {
		delay = parsed
	}
	return delay, motivo, true
}

func registrarPresupuestoSesionProviderBackoff(order *RuntimeOrder, runtimeErr error, delay time.Duration, motivo string) error {
	if order == nil || runtimeErr == nil || strings.TrimSpace(order.Agente) == "" {
		return nil
	}
	sesionID, modelSlug, err := resolverSesionYModeloRuntimeOrder(order)
	if err != nil || sesionID <= 0 {
		return err
	}
	now := time.Now().UTC()
	if delay <= 0 {
		delay = 60 * time.Minute
	}
	resetAt := now.Add(delay)
	remainingZero := int64(0)
	raw := strings.TrimSpace(runtimeErr.Error())
	snapshot := map[string]any{
		"source":         "runtime_order_send_instruction",
		"runtime_order":  order.ID,
		"agente":         strings.TrimSpace(order.Agente),
		"motivo":         strings.TrimSpace(motivo),
		"window_kind":    "provider",
		"window_start":   now.Format(time.RFC3339Nano),
		"reset_at":       resetAt.Format(time.RFC3339Nano),
		"remaining_zero": true,
		"error":          raw,
	}
	for key, value := range identidadObservadaDesdeErrorProveedor(raw) {
		snapshot[key] = value
	}
	for key, value := range identidadObservadaDesdeOrdenRuntime(order) {
		if _, exists := snapshot[key]; !exists {
			snapshot[key] = value
		}
	}
	rawJSON, _ := json.Marshal(snapshot)
	id, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:          sesionID,
		ModelSlug:         strings.TrimSpace(modelSlug),
		WindowKind:        "provider",
		WindowStartedAt:   &now,
		ResetAt:           &resetAt,
		RemainingMessages: &remainingZero,
		BudgetSource:      "provider_backoff",
		RawSnapshotJSON:   string(rawJSON),
		CheckedAt:         now,
	})
	if err != nil {
		return err
	}
	Audit("orquesta", "registrar_presupuesto_provider_backoff", "presupuesto_sesion", id,
		fmt.Sprintf("runtime_order=%d agente=%s reset_at=%s motivo=%s",
			order.ID, strings.TrimSpace(order.Agente), resetAt.Format(time.RFC3339), strings.TrimSpace(motivo)))
	return nil
}

func resolverSesionYModeloRuntimeOrder(order *RuntimeOrder) (int64, string, error) {
	if order == nil {
		return 0, "", nil
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return 0, "", err
	}
	model := ""
	if runtime != nil {
		model = strings.TrimSpace(runtime.Model)
	}
	sesion, err := resolverSesionParaOrden(order)
	if err != nil {
		return 0, "", err
	}
	if sesion != nil {
		return sesion.ID, model, nil
	}
	return 0, model, nil
}

func identidadObservadaDesdeErrorProveedor(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := map[string]any{}
	emailRe := regexp.MustCompile(`(?i)[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}`)
	if email := strings.TrimSpace(emailRe.FindString(raw)); email != "" {
		out["account_email"] = email
	}
	userRe := regexp.MustCompile(`(?im)^(?:perfil activo|active profile|logged in as|usuario|user|username|login)\s*:\s*([^\r\n]+)\s*$`)
	if match := userRe.FindStringSubmatch(raw); len(match) == 2 {
		if user := strings.TrimSpace(match[1]); user != "" {
			out["account_user"] = user
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func identidadObservadaDesdeOrdenRuntime(order *RuntimeOrder) map[string]any {
	if order == nil {
		return nil
	}
	candidates := make([]*RuntimeHandle, 0, 2)
	if order.HandleID != nil && *order.HandleID > 0 {
		handle, err := GetRuntimeHandle(*order.HandleID)
		if err == nil && handle != nil {
			candidates = append(candidates, handle)
		}
	}
	handle, err := resolverHandleParaOrden(order)
	if err == nil && handle != nil {
		if len(candidates) == 0 || candidates[0].ID != handle.ID {
			candidates = append(candidates, handle)
		}
	}
	for _, candidate := range candidates {
		if observed := identidadObservadaDesdeHandleRuntime(candidate); observed != nil {
			return observed
		}
	}
	return nil
}

func identidadObservadaDesdeHandleRuntime(handle *RuntimeHandle) map[string]any {
	if handle == nil {
		return nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if meta == nil {
		return nil
	}
	rendered := strings.TrimSpace(stringFromMap(meta, "rendered_command", ""))
	if rendered == "" {
		return nil
	}
	re := regexp.MustCompile(`(?i)codex-perfil'\s+'([^']+)'`)
	match := re.FindStringSubmatch(rendered)
	if len(match) != 2 || strings.TrimSpace(match[1]) == "" {
		return nil
	}
	return map[string]any{"account_user": strings.TrimSpace(match[1])}
}

func runtimeOrderProviderRetryDelay(lower string, now time.Time) (time.Duration, bool) {
	re := regexp.MustCompile(`try again at\s+(\d{1,2}):(\d{2})\s*([ap]m)`)
	match := re.FindStringSubmatch(strings.ToLower(lower))
	if len(match) != 4 {
		return 0, false
	}
	hour, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}
	minute, err := strconv.Atoi(match[2])
	if err != nil {
		return 0, false
	}
	ampm := match[3]
	if hour == 12 {
		hour = 0
	}
	if ampm == "pm" {
		hour += 12
	}
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !target.After(now) {
		target = target.Add(24 * time.Hour)
	}
	return target.Sub(now), true
}
