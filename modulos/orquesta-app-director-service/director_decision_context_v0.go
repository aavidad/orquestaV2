package orquestaappdirectorservice

import (
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func directorDecisionObjectiveHintsV0(
	req orquestafactory.AppSpecRequestV0,
) []string {
	hints := []string{
		req.Nombre,
		req.Objetivo,
		req.Descripcion,
		req.TipoApp,
		req.PreferenciasTecnicas.Lenguaje,
		req.PreferenciasTecnicas.Framework,
		req.PreferenciasTecnicas.Arquitectura,
		req.Datos.NecesidadFuncional,
		req.Deploy.Target,
	}
	hints = append(hints, req.UsuariosObjetivo...)
	hints = append(hints, req.Plataformas...)
	hints = append(hints, req.PreferenciasTecnicas.Restricciones...)
	hints = append(hints, req.PreferenciasTecnicas.Preferencias...)
	hints = append(hints, req.Datos.TiposDatos...)
	hints = append(hints, req.Restricciones...)
	for _, integration := range req.Integraciones {
		hints = append(hints, integration.Tipo, integration.Nombre, integration.Proposito)
		hints = append(hints, integration.Restricciones...)
	}
	return compactDirectorDecisionHintsV0(hints)
}

func compactDirectorDecisionHintsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
