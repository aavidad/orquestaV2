/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package orquestacionagentesapp

import (
	"encoding/json"
	"strings"

	"orquesta/db"
)

type Repository struct{}

func (Repository) Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	db.Audit(agente, accion, entidad, entidadID, detalle)
}

func (Repository) GetPersistedAgentQuotaState(agente string) (string, error) {
	return db.GetPersistedAgentQuotaState(agente)
}

func (Repository) GetSessionRuntimeHandle(sessionID int64) (*db.RuntimeHandle, error) {
	handle, err := db.GetRuntimeHandleBySesionID(sessionID)
	if err != nil || handle == nil || handle.ID <= 0 {
		return handle, err
	}
	return db.GetRuntimeHandle(handle.ID)
}

func (Repository) ParkActiveSession(agente string, proyectoID *int64) error {
	return db.AparcarSesionActiva(agente, proyectoID)
}

func (Repository) PauseAssignment(agente string, proyectoID int64, motivo string) error {
	return db.PausarAsignacion(agente, proyectoID, motivo)
}

func (Repository) ResolveSessionConnector(sesion *db.Sesion, runtime *db.RuntimeInstance, handle *db.RuntimeHandle) (*db.Conector, error) {
	if sesion != nil && strings.TrimSpace(sesion.ConectorSlug) != "" {
		return db.GetConector(strings.TrimSpace(sesion.ConectorSlug))
	}
	if runtime != nil && strings.TrimSpace(runtime.Connector) != "" {
		return db.GetConector(strings.TrimSpace(runtime.Connector))
	}
	if handle != nil {
		if slug := metadataString(handle.MetadataJSON, "conector"); slug != "" {
			return db.GetConector(slug)
		}
	}
	return nil, nil
}

func (Repository) IsConnectorAvailable(conectorID int64) (bool, error) {
	disponible, _, err := db.ConectorDisponibleParaArranque(conectorID)
	return disponible, err
}

func (Repository) MarkProjectExternallyBlocked(proyectoID int64, motivo string) error {
	return db.MarcarProyectoBloqueadoExterno(proyectoID, motivo)
}

func metadataString(raw string, key string) string {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed); err != nil || parsed == nil {
		return ""
	}
	value, ok := parsed[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
