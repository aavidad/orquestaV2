/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"database/sql"
	"strings"
)

func CountProjectOpenProposals(proyectoID int64) (int, error) {
	if proyectoID <= 0 {
		return 0, nil
	}
	return countQueryInt(
		`SELECT COUNT(*) FROM propuestas WHERE proyecto_id = ? AND estado = ?`,
		proyectoID,
		PropuestaAbierta,
	)
}

func CountProjectOpenReviewGates(proyectoID int64) (int, error) {
	if proyectoID <= 0 {
		return 0, nil
	}
	exists, err := SchemaObjectExists("table", "review_gates")
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, nil
	}
	return countQueryInt(
		`SELECT COUNT(*) FROM review_gates WHERE proyecto_id = ? AND estado <> ?`,
		proyectoID,
		ReviewGateAprobado,
	)
}

func CountProjectPendingRuntimeMailbox(proyectoID int64) (int, error) {
	if proyectoID <= 0 {
		return 0, nil
	}
	return countQueryInt(
		`SELECT COUNT(*) FROM runtime_mailbox WHERE proyecto_id = ? AND estado = ?`,
		proyectoID,
		"pendiente",
	)
}

func CountProjectOpenRuntimeOrders(proyectoID int64, estados ...string) (int, error) {
	if proyectoID <= 0 || len(estados) == 0 {
		return 0, nil
	}
	placeholders := make([]string, 0, len(estados))
	args := make([]any, 0, len(estados)+1)
	args = append(args, proyectoID)
	for _, estado := range estados {
		estado = strings.TrimSpace(estado)
		if estado == "" {
			continue
		}
		placeholders = append(placeholders, "?")
		args = append(args, estado)
	}
	if len(placeholders) == 0 {
		return 0, nil
	}
	query := `SELECT COUNT(*) FROM runtime_orders WHERE proyecto_id = ? AND estado IN (` + strings.Join(placeholders, ",") + `)`
	return countQueryInt(query, args...)
}

func GetPersistedAgentQuotaState(nombre string) (string, error) {
	nombreCanonico, _, _, err := resolverAgentePorNombreCI(strings.TrimSpace(nombre))
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return consultarConReintentos(func() (string, error) {
		var estado sql.NullString
		err := DB.QueryRow(
			`SELECT estado_cuota FROM agentes WHERE nombre = ? LIMIT 1`,
			nombreCanonico,
		).Scan(&estado)
		if err == sql.ErrNoRows {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(estado.String), nil
	})
}

func countQueryInt(query string, args ...any) (int, error) {
	return consultarConReintentos(func() (int, error) {
		var total int
		err := DB.QueryRow(query, args...).Scan(&total)
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return total, err
	})
}
