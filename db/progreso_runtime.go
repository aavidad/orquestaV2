/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "database/sql"

// ProgresoEstimadoPorSenales estima el porcentaje de progreso (0-100) de una
// tarea a partir de señales observables del runtime: checkpoints del agente
// asignado y mensajes consumidos en su mailbox.
//
// Fórmula:
//   - +20% por cada checkpoint del agente (tope 60%)
//   - +10% por cada 3 mensajes mailbox consumidos del agente (tope 40%)
//
// Si la tarea no existe o no tiene agente asignado, devuelve 0, nil.
func ProgresoEstimadoPorSenales(tareaID int64) (int, error) {
	var agente sql.NullString
	err := DB.QueryRow(`SELECT agente FROM tareas WHERE id = ?`, tareaID).Scan(&agente)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !agente.Valid || agente.String == "" {
		return 0, nil
	}
	nombre := agente.String

	var checkpoints int
	if err := DB.QueryRow(
		`SELECT COUNT(*) FROM runtime_checkpoints WHERE agente = ?`, nombre,
	).Scan(&checkpoints); err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	var consumidos int
	if err := DB.QueryRow(
		`SELECT COUNT(*) FROM runtime_mailbox WHERE to_agente = ? AND estado = 'consumido'`, nombre,
	).Scan(&consumidos); err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	checkpointPct := checkpoints * 20
	if checkpointPct > 60 {
		checkpointPct = 60
	}
	mailboxPct := (consumidos / 3) * 10
	if mailboxPct > 40 {
		mailboxPct = 40
	}
	return checkpointPct + mailboxPct, nil
}
