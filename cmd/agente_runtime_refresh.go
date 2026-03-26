package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"orquesta/db"
)

const agenteRuntimeRefreshPollInterval = 5 * time.Second

func construirInstruccionRefreshRuntime(msg *db.RuntimeMailboxMessage) (string, bool) {
	return construirInstruccionRefreshRuntimeConCatalogo(msg, nil)
}

func construirInstruccionRefreshRuntimeConCatalogo(msg *db.RuntimeMailboxMessage, catalogo *db.GovernanceCatalog) (string, bool) {
	if msg == nil {
		return "", false
	}

	payload := map[string]any{}
	if strings.TrimSpace(msg.PayloadJSON) != "" {
		_ = json.Unmarshal([]byte(msg.PayloadJSON), &payload)
	}

	switch strings.TrimSpace(msg.Kind) {
	case db.MailboxKindGovernanceRefresh:
		tipoAgente := stringMapValue(payload, "tipo_agente")
		motivo := stringMapValue(payload, "motivo")
		hash := stringMapValue(payload, "hash")
		reglas := intMapValue(payload, "reglas")
		skills := intMapValue(payload, "skills")
		workflows := intMapValue(payload, "workflows")
		scopeTipo := stringMapValue(payload, "scope_tipo")
		scopeRef := stringMapValue(payload, "scope_ref")
		parts := []string{
			"Actualiza en caliente tu gobernanza efectiva antes de continuar.",
		}
		if tipoAgente != "" {
			parts = append(parts, fmt.Sprintf("Rol objetivo: %s.", tipoAgente))
		}
		if motivo != "" {
			parts = append(parts, fmt.Sprintf("Motivo: %s.", motivo))
		}
		if hash != "" {
			parts = append(parts, fmt.Sprintf("Hash: %s.", hash))
		}
		if reglas > 0 || skills > 0 || workflows > 0 {
			parts = append(parts, fmt.Sprintf("Catálogo efectivo: reglas=%d skills=%d workflows=%d.", reglas, skills, workflows))
		}
		if scopeTipo != "" || scopeRef != "" {
			parts = append(parts, fmt.Sprintf("Ámbito del cambio: %s %s.", strings.TrimSpace(scopeTipo), strings.TrimSpace(scopeRef)))
		}
		if catalogo != nil {
			if resolucion := strings.TrimSpace(catalogo.ResolucionActual); resolucion != "" {
				parts = append(parts, fmt.Sprintf("Resolución efectiva: %s.", resolucion))
			}
			if resumen := resumirCatalogoGobernanza(catalogo); resumen != "" {
				parts = append(parts, resumen)
			}
		}
		parts = append(parts, "Ajusta el trabajo en curso sin reiniciar la sesión usando este catálogo efectivo.")
		return strings.Join(parts, " "), true
	case db.MailboxKindSkillsRefresh:
		nombre := stringMapValue(payload, "nombre")
		motivo := stringMapValue(payload, "motivo")
		origen := stringMapValue(payload, "origen")
		parts := []string{
			"Actualiza en caliente el catálogo de skills aplicable antes de continuar.",
		}
		if nombre != "" {
			parts = append(parts, fmt.Sprintf("Skill afectada: %s.", nombre))
		}
		if motivo != "" {
			parts = append(parts, fmt.Sprintf("Motivo: %s.", motivo))
		}
		if origen != "" {
			parts = append(parts, fmt.Sprintf("Origen: %s.", origen))
		}
		parts = append(parts, "Reevalúa si debes usar, dejar de usar o reconfigurar skills durante la tarea actual.")
		return strings.Join(parts, " "), true
	default:
		return "", false
	}
}

func resumirCatalogoGobernanza(catalogo *db.GovernanceCatalog) string {
	if catalogo == nil {
		return ""
	}
	parts := []string{}
	if reglas := resumirTitulosReglas(catalogo.Reglas, 4); reglas != "" {
		parts = append(parts, "Reglas clave: "+reglas+".")
	}
	if skills := resumirNombresSkills(catalogo.Skills, 4); skills != "" {
		parts = append(parts, "Skills clave: "+skills+".")
	}
	if workflows := resumirNombresWorkflows(catalogo.Workflows, 3); workflows != "" {
		parts = append(parts, "Workflows: "+workflows+".")
	}
	return strings.Join(parts, " ")
}

func resumirTitulosReglas(reglas []*db.Regla, max int) string {
	nombres := make([]string, 0, max)
	for _, item := range reglas {
		if item == nil || strings.TrimSpace(item.Titulo) == "" {
			continue
		}
		nombres = append(nombres, strings.TrimSpace(item.Titulo))
		if len(nombres) >= max {
			break
		}
	}
	return strings.Join(nombres, ", ")
}

func resumirNombresSkills(skills []*db.Skill, max int) string {
	nombres := make([]string, 0, max)
	for _, item := range skills {
		if item == nil || strings.TrimSpace(item.Nombre) == "" {
			continue
		}
		nombres = append(nombres, strings.TrimSpace(item.Nombre))
		if len(nombres) >= max {
			break
		}
	}
	return strings.Join(nombres, ", ")
}

func resumirNombresWorkflows(workflows []*db.Workflow, max int) string {
	nombres := make([]string, 0, max)
	for _, item := range workflows {
		if item == nil || strings.TrimSpace(item.Nombre) == "" {
			continue
		}
		nombres = append(nombres, strings.TrimSpace(item.Nombre))
		if len(nombres) >= max {
			break
		}
	}
	return strings.Join(nombres, ", ")
}

func stringMapValue(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func intMapValue(payload map[string]any, key string) int {
	if payload == nil {
		return 0
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func encolarInstruccionRefreshRuntime(msg *db.RuntimeMailboxMessage, proyecto string) (int64, error) {
	texto, ok := construirInstruccionRefreshRuntime(msg)
	if !ok {
		return 0, nil
	}
	if strings.TrimSpace(msg.Kind) == db.MailboxKindGovernanceRefresh {
		payload := map[string]any{}
		if strings.TrimSpace(msg.PayloadJSON) != "" {
			_ = json.Unmarshal([]byte(msg.PayloadJSON), &payload)
		}
		tipoAgente := stringMapValue(payload, "tipo_agente")
		if catalogo, okAPI, err := cargarGobernanzaCatalogoDesdeAPI(tipoAgente, proyecto, strings.TrimSpace(msg.ToAgente)); okAPI && err == nil && catalogo != nil {
			if enriched, ok := construirInstruccionRefreshRuntimeConCatalogo(msg, catalogo); ok {
				texto = enriched
			}
		}
	}
	payload := map[string]any{
		"to_agente":    strings.TrimSpace(msg.ToAgente),
		"from_agente":  strings.TrimSpace(msg.FromAgente),
		"texto":        texto,
		"refresh_kind": strings.TrimSpace(msg.Kind),
		"mailbox_id":   msg.ID,
	}
	if msg.ProyectoID != nil && *msg.ProyectoID > 0 {
		payload["proyecto_id"] = *msg.ProyectoID
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	orderID, okAPI, err := crearRuntimeOrderDesdeAPI(strings.TrimSpace(msg.ToAgente), "send_instruction", "", string(data))
	if !okAPI {
		return 0, serverFirstCommandError("agente runtime refresh")
	}
	if err != nil {
		return 0, err
	}
	if ok, err := marcarRuntimeMailboxEntregadoPorAPI(msg.ID); !ok {
		return 0, serverFirstCommandError("runtime mailbox entregar")
	} else if err != nil {
		return 0, err
	}
	if ok, err := marcarRuntimeMailboxConsumidoPorAPI(msg.ID); !ok {
		return 0, serverFirstCommandError("runtime mailbox consumir")
	} else if err != nil {
		return 0, err
	}
	return orderID, nil
}

func procesarRefreshRuntimeMailbox(agente, proyecto string) (int, error) {
	query := url.Values{}
	query.Set("to_agente", strings.TrimSpace(agente))
	query.Set("estado", "pendiente")
	if strings.TrimSpace(proyecto) != "" {
		query.Set("proyecto", strings.TrimSpace(proyecto))
	}
	mailbox, ok, err := cargarRuntimeMailboxDesdeAPI(query)
	if !ok {
		return 0, serverFirstCommandError("runtime mailbox")
	}
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, msg := range mailbox {
		if _, ok := construirInstruccionRefreshRuntime(msg); !ok {
			continue
		}
		if _, err := encolarInstruccionRefreshRuntime(msg, proyecto); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func vigilarRefreshRuntimeMailbox(stop <-chan struct{}, agente, proyecto string) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(agenteRuntimeRefreshPollInterval)
		defer ticker.Stop()
		for {
			if processed, err := procesarRefreshRuntimeMailbox(agente, proyecto); err != nil {
				fmt.Fprintf(os.Stderr, "\n[Orquesta] aviso: no se pudo procesar refresh runtime (%v)\n", err)
			} else if processed > 0 {
				fmt.Printf("\n🔄 [Orquesta] Refresh en caliente aplicado (%d).\n", processed)
			}
			select {
			case <-stop:
				return
			case <-ticker.C:
			}
		}
	}()
	return done
}
