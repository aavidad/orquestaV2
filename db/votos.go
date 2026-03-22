package db

import (
	"database/sql"
	"fmt"
	"time"
)

type PosicionVoto string

const (
	VotoPendiente  PosicionVoto = "pendiente"
	VotoAcuerdo    PosicionVoto = "acuerdo"
	VotoDesacuerdo PosicionVoto = "desacuerdo"
	VotoAbstencion PosicionVoto = "abstencion"
)

type Voto struct {
	ID          int64
	PropuestaID int64
	Agente      string
	Posicion    PosicionVoto
	Comentario  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Votar registra o actualiza el voto de un agente en una propuesta.
// Devuelve (consensoAlcanzado bool, error).
func Votar(propuestaID int64, agente string, posicion PosicionVoto, comentario string) (bool, error) {
	_, err := DB.Exec(`
		INSERT INTO votos (propuesta_id, agente, posicion, comentario)
		VALUES (?,?,?,?)
		ON CONFLICT(propuesta_id, agente) DO UPDATE SET
		    posicion   = excluded.posicion,
		    comentario = excluded.comentario`,
		propuestaID, agente, posicion, comentario,
	)
	if err != nil {
		return false, err
	}
	Audit(agente, "votar", "propuesta", propuestaID, string(posicion))

	// Evaluar consenso automáticamente
	consenso, err := EvaluarConsenso(propuestaID, agente)
	return consenso, err
}

// VotosDePropuesta devuelve todos los votos de una propuesta.
func VotosDePropuesta(propuestaID int64) ([]*Voto, error) {
	rows, err := DB.Query(`
		SELECT id, propuesta_id, agente, posicion, comentario, created_at, updated_at
		FROM votos WHERE propuesta_id = ? ORDER BY agente`, propuestaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Voto
	for rows.Next() {
		v := &Voto{}
		if err := rows.Scan(&v.ID, &v.PropuestaID, &v.Agente, &v.Posicion,
			&v.Comentario, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// RepararVotosPendientes sincroniza los votos pendientes de una propuesta.
// Si reiniciar=true, todos los votantes habilitados vuelven a pendiente.
// Si reiniciar=false, solo se crean los votos que faltan.
func RepararVotosPendientes(propuestaID int64, agente string, reiniciar bool) (int, error) {
	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	count, err := repararVotosPendientesTx(tx, propuestaID, reiniciar)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	detalle := fmt.Sprintf("propuesta=%d votos=%d reiniciar=%t", propuestaID, count, reiniciar)
	Audit(agente, "reparar_votos_pendientes", "propuesta", propuestaID, detalle)
	return count, nil
}

// ResumenVotos devuelve todos los votos de una propuesta (detalle por agente).
// Alias de VotosDePropuesta — para uso en cmd/propuesta.go y cmd/exportar.go.
func ResumenVotos(propuestaID int64) ([]*Voto, error) {
	return VotosDePropuesta(propuestaID)
}

// ContarVotos devuelve conteo de posiciones para una propuesta.
func ContarVotos(propuestaID int64) (acuerdo, desacuerdo, abstencion, pendiente int, err error) {
	rows, err := DB.Query(
		`SELECT posicion, COUNT(*) FROM votos WHERE propuesta_id=? GROUP BY posicion`,
		propuestaID,
	)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var pos string
		var n int
		_ = rows.Scan(&pos, &n)
		switch PosicionVoto(pos) {
		case VotoAcuerdo:
			acuerdo = n
		case VotoDesacuerdo:
			desacuerdo = n
		case VotoAbstencion:
			abstencion = n
		case VotoPendiente:
			pendiente = n
		}
	}
	return
}

func repararVotosPendientesTx(tx *sql.Tx, propuestaID int64, reiniciar bool) (int, error) {
	rows, err := tx.Query(`
		SELECT nombre
		FROM agentes
		WHERE rol != 'admin' AND habilitado = 1
		ORDER BY nombre`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var votantes []string
	for rows.Next() {
		var agente string
		if err := rows.Scan(&agente); err != nil {
			return 0, err
		}
		votantes = append(votantes, agente)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var count int
	for _, agente := range votantes {
		if reiniciar {
			res, err := tx.Exec(`
				INSERT INTO votos (propuesta_id, agente, posicion, comentario)
				VALUES (?,?, 'pendiente', '')
				ON CONFLICT(propuesta_id, agente) DO UPDATE SET
					posicion='pendiente',
					comentario=''`,
				propuestaID, agente,
			)
			if err != nil {
				return 0, err
			}
			n, _ := res.RowsAffected()
			if n > 0 {
				count++
			}
			continue
		}

		res, err := tx.Exec(
			`INSERT OR IGNORE INTO votos (propuesta_id, agente, posicion) VALUES (?,?, 'pendiente')`,
			propuestaID, agente,
		)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			count++
		}
	}
	return count, nil
}
