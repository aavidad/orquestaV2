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
	stmt := upsertValuesSQL(
		"votos",
		[]string{"propuesta_id", "agente", "posicion", "comentario"},
		[]string{"propuesta_id", "agente"},
		[]upsertAssignment{
			{Column: "posicion"},
			{Column: "comentario"},
		},
	)
	_, err := DB.Exec(stmt,
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
	return consultarConReintentos(func() ([]*Voto, error) {
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
	})
}

// ResumenVotos devuelve todos los votos de una propuesta (detalle por agente).
// Alias de VotosDePropuesta — para uso en cmd/propuesta.go y cmd/exportar.go.
func ResumenVotos(propuestaID int64) ([]*Voto, error) {
	return VotosDePropuesta(propuestaID)
}

// ResumenVotosPorPropuestas devuelve el detalle de votos agrupado por propuesta
// en una sola consulta para evitar N+1 en paneles y resúmenes globales.
func ResumenVotosPorPropuestas(propuestaIDs []int64) (map[int64][]*Voto, error) {
	grouped := make(map[int64][]*Voto, len(propuestaIDs))
	if len(propuestaIDs) == 0 {
		return grouped, nil
	}
	placeholders := joinInt64Placeholders(propuestaIDs)
	args := int64Args(propuestaIDs)
	return consultarConReintentos(func() (map[int64][]*Voto, error) {
		rows, err := DB.Query(`
			SELECT id, propuesta_id, agente, posicion, comentario, created_at, updated_at
			FROM votos
			WHERE propuesta_id IN (`+placeholders+`)
			ORDER BY propuesta_id, agente`, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for _, propuestaID := range propuestaIDs {
			grouped[propuestaID] = nil
		}
		for rows.Next() {
			v := &Voto{}
			if err := rows.Scan(&v.ID, &v.PropuestaID, &v.Agente, &v.Posicion,
				&v.Comentario, &v.CreatedAt, &v.UpdatedAt); err != nil {
				return nil, err
			}
			grouped[v.PropuestaID] = append(grouped[v.PropuestaID], v)
		}
		return grouped, rows.Err()
	})
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
