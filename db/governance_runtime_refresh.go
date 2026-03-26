package db

import "encoding/json"

const MailboxKindGovernanceRefresh = "governance_refresh"

func notificarRefreshGobernanza(actor, tipoAgente, motivo string, extra map[string]any) {
	if tipoAgente == "" {
		return
	}
	sesiones, err := ListarSesionesActivas()
	if err != nil {
		Audit(actor, "governance_refresh_error", "regla", 0, err.Error())
		return
	}

	seen := map[string]struct{}{}
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		agente, err := GetAgente(sesion.Agente)
		if err != nil || agente == nil || agente.Rol != tipoAgente {
			continue
		}
		if _, ok := seen[sesion.Agente]; ok {
			continue
		}
		seen[sesion.Agente] = struct{}{}

		catalogo, err := ResolveGovernanceCatalogForContext(tipoAgente, sesion.ProyectoID, sesion.Agente)
		if err != nil || catalogo == nil {
			if err != nil {
				Audit(actor, "governance_refresh_error", "regla", 0, err.Error())
			}
			continue
		}
		payloadMap := map[string]any{
			"tipo_agente": tipoAgente,
			"motivo":      motivo,
			"hash":        catalogo.Hash,
			"reglas":      len(catalogo.Reglas),
			"skills":      len(catalogo.Skills),
			"workflows":   len(catalogo.Workflows),
		}
		for key, value := range extra {
			payloadMap[key] = value
		}
		payload, _ := json.Marshal(payloadMap)
		if _, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
			FromAgente:  actor,
			ToAgente:    sesion.Agente,
			ProyectoID:  sesion.ProyectoID,
			Kind:        MailboxKindGovernanceRefresh,
			PayloadJSON: string(payload),
		}); err != nil {
			Audit(actor, "governance_refresh_error", "regla", 0, err.Error())
		}
	}
}
