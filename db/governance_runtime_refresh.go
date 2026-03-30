package db

import "encoding/json"

const MailboxKindGovernanceRefresh = "governance_refresh"

func buildGovernanceRefreshPayload(tipoAgente string, proyectoID *int64, agente, motivo string, extra map[string]any) (string, error) {
	catalogo, err := ResolveGovernanceCatalogForContext(tipoAgente, proyectoID, agente)
	if err != nil {
		return "", err
	}
	if catalogo == nil {
		return "", nil
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
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func enviarRefreshGobernanzaAgente(actor, tipoAgente, agente string, proyectoID *int64, motivo string, extra map[string]any) error {
	if tipoAgente == "" || agente == "" {
		return nil
	}
	payload, err := buildGovernanceRefreshPayload(tipoAgente, proyectoID, agente, motivo, extra)
	if err != nil {
		Audit(actor, "governance_refresh_error", "regla", 0, err.Error())
		return err
	}
	if payload == "" {
		return nil
	}
	_, err = EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  actor,
		ToAgente:    agente,
		ProyectoID:  proyectoID,
		Kind:        MailboxKindGovernanceRefresh,
		PayloadJSON: payload,
	})
	if err != nil {
		Audit(actor, "governance_refresh_error", "regla", 0, err.Error())
	}
	return err
}

func notificarRefreshGobernanza(actor, tipoAgente, motivo string, extra map[string]any) {
	if tipoAgente == "" {
		return
	}
	sesiones, err := ListarSesionesActivasOperativas()
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

		payload, err := buildGovernanceRefreshPayload(tipoAgente, sesion.ProyectoID, sesion.Agente, motivo, extra)
		if err != nil || payload == "" {
			if err != nil {
				Audit(actor, "governance_refresh_error", "regla", 0, err.Error())
			}
			continue
		}
		if _, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
			FromAgente:  actor,
			ToAgente:    sesion.Agente,
			ProyectoID:  sesion.ProyectoID,
			Kind:        MailboxKindGovernanceRefresh,
			PayloadJSON: payload,
		}); err != nil {
			Audit(actor, "governance_refresh_error", "regla", 0, err.Error())
		}
	}
}
