package db

import (
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
	_, err := DB.Exec(
		upsertValuesSQL(
			"votos",
			[]string{"propuesta_id", "agente", "posicion", "comentario"},
			[]string{"propuesta_id", "agente"},
			[]upsertAssignment{
				{Column: "posicion"},
				{Column: "comentario"},
			},
		),
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
