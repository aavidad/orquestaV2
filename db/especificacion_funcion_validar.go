/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// ValidarEntregaWriteSet comprueba que todos los ficheros entregados están dentro
// del write-set declarado por la especificación de función.
//
// Devuelve:
//   - (true, nil, nil)  si todos los ficherosCambiados están permitidos
//   - (false, lista, nil) si alguno no está en el write-set
//   - (false, nil, error) si la especificación no existe o hay error de BD
func ValidarEntregaWriteSet(especID int64, ficherosCambiados []string) (bool, []string, error) {
	var writeSetJSON string
	err := DB.QueryRow(
		`SELECT write_set_json FROM especificaciones_funcion WHERE id = ?`, especID,
	).Scan(&writeSetJSON)
	if err == sql.ErrNoRows {
		return false, nil, fmt.Errorf("especificacion #%d no encontrada", especID)
	}
	if err != nil {
		return false, nil, err
	}

	var writeSet []string
	if err := json.Unmarshal([]byte(writeSetJSON), &writeSet); err != nil {
		return false, nil, fmt.Errorf("write_set_json inválido en especificacion #%d: %w", especID, err)
	}

	permitidos := make(map[string]struct{}, len(writeSet))
	for _, f := range writeSet {
		permitidos[f] = struct{}{}
	}

	var noPermitidos []string
	for _, f := range ficherosCambiados {
		if _, ok := permitidos[f]; !ok {
			noPermitidos = append(noPermitidos, f)
		}
	}
	if len(noPermitidos) > 0 {
		return false, noPermitidos, nil
	}
	return true, nil, nil
}
