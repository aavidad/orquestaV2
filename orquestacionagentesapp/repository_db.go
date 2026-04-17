/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package orquestacionagentesapp

import "orquesta/db"

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
