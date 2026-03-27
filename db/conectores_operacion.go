package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type EstadoOperativoConector string

const (
	ConectorOperativoActivo          EstadoOperativoConector = "activo"
	ConectorOperativoCircuitoAbierto EstadoOperativoConector = "circuito_abierto"
)

type ConectorOperacion struct {
	ConectorID         int64
	EstadoOperativo    EstadoOperativoConector
	Motivo             string
	FallosConsecutivos int
	CooldownUntil      *time.Time
	CircuitoAbiertoAt  *time.Time
	UltimoError        string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func ensureConectorOperacionSchema() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS conectores_operacion (
			conector_id          INTEGER PRIMARY KEY REFERENCES conectores(id) ON DELETE CASCADE,
			estado_operativo     TEXT    NOT NULL DEFAULT 'activo'
			                             CHECK (estado_operativo IN ('activo','circuito_abierto')),
			motivo               TEXT    NOT NULL DEFAULT '',
			fallos_consecutivos  INTEGER NOT NULL DEFAULT 0,
			cooldown_until       DATETIME,
			circuito_abierto_at  DATETIME,
			ultimo_error         TEXT    NOT NULL DEFAULT '',
			created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	return err
}

func GetConectorOperacion(conectorID int64) (*ConectorOperacion, error) {
	if err := ensureConectorOperacionSchema(); err != nil {
		return nil, err
	}
	var op ConectorOperacion
	err := DB.QueryRow(`
		SELECT conector_id, estado_operativo, motivo, fallos_consecutivos, cooldown_until,
		       circuito_abierto_at, ultimo_error, created_at, updated_at
		FROM conectores_operacion
		WHERE conector_id = ?`, conectorID,
	).Scan(
		&op.ConectorID,
		&op.EstadoOperativo,
		&op.Motivo,
		&op.FallosConsecutivos,
		&op.CooldownUntil,
		&op.CircuitoAbiertoAt,
		&op.UltimoError,
		&op.CreatedAt,
		&op.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return conectorOperacionDefault(conectorID), nil
	}
	if err != nil {
		return nil, err
	}
	normalizarConectorOperacion(&op)
	return &op, nil
}

func UpsertConectorOperacion(op *ConectorOperacion) error {
	if err := ensureConectorOperacionSchema(); err != nil {
		return err
	}
	if op == nil {
		return fmt.Errorf("conector_operacion nil")
	}
	if op.ConectorID == 0 {
		return fmt.Errorf("conector_id obligatorio")
	}
	normalizarConectorOperacion(op)
	_, err := DB.Exec(`
		INSERT INTO conectores_operacion (
			conector_id, estado_operativo, motivo, fallos_consecutivos, cooldown_until, circuito_abierto_at, ultimo_error
		) VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(conector_id) DO UPDATE SET
			estado_operativo=excluded.estado_operativo,
			motivo=excluded.motivo,
			fallos_consecutivos=excluded.fallos_consecutivos,
			cooldown_until=excluded.cooldown_until,
			circuito_abierto_at=excluded.circuito_abierto_at,
			ultimo_error=excluded.ultimo_error,
			updated_at=CURRENT_TIMESTAMP`,
		op.ConectorID,
		op.EstadoOperativo,
		strings.TrimSpace(op.Motivo),
		op.FallosConsecutivos,
		op.CooldownUntil,
		op.CircuitoAbiertoAt,
		strings.TrimSpace(op.UltimoError),
	)
	return err
}

func RegistrarFalloConector(conectorID int64, motivo string) (*ConectorOperacion, error) {
	op, err := GetConectorOperacion(conectorID)
	if err != nil {
		return nil, err
	}
	op.FallosConsecutivos++
	op.UltimoError = strings.TrimSpace(motivo)
	threshold := configIntOrDefault("connector_circuit_breaker_threshold", 3)
	if threshold <= 0 {
		threshold = 3
	}
	if op.FallosConsecutivos >= threshold {
		now := time.Now().UTC()
		cooldownSeconds := configIntOrDefault("connector_circuit_breaker_cooldown_seconds", 300)
		op.EstadoOperativo = ConectorOperativoCircuitoAbierto
		op.Motivo = "remote_connector_unavailable"
		op.CircuitoAbiertoAt = &now
		if cooldownSeconds > 0 {
			until := now.Add(time.Duration(cooldownSeconds) * time.Second)
			op.CooldownUntil = &until
		} else {
			op.CooldownUntil = nil
		}
	}
	if err := UpsertConectorOperacion(op); err != nil {
		return nil, err
	}
	return op, nil
}

func RegistrarExitoConector(conectorID int64) error {
	op, err := GetConectorOperacion(conectorID)
	if err != nil {
		return err
	}
	op.EstadoOperativo = ConectorOperativoActivo
	op.Motivo = ""
	op.FallosConsecutivos = 0
	op.CooldownUntil = nil
	op.CircuitoAbiertoAt = nil
	op.UltimoError = ""
	return UpsertConectorOperacion(op)
}

func ConectorDisponibleParaArranque(conectorID int64) (bool, *ConectorOperacion, error) {
	op, err := GetConectorOperacion(conectorID)
	if err != nil {
		return false, nil, err
	}
	if op.EstadoOperativo != ConectorOperativoCircuitoAbierto {
		return true, op, nil
	}
	if op.CooldownUntil != nil && time.Now().UTC().Before(*op.CooldownUntil) {
		return false, op, nil
	}
	op.EstadoOperativo = ConectorOperativoActivo
	op.Motivo = ""
	op.FallosConsecutivos = 0
	op.CooldownUntil = nil
	op.CircuitoAbiertoAt = nil
	op.UltimoError = ""
	if err := UpsertConectorOperacion(op); err != nil {
		return false, nil, err
	}
	return true, op, nil
}

func conectorOperacionDefault(conectorID int64) *ConectorOperacion {
	return &ConectorOperacion{
		ConectorID:      conectorID,
		EstadoOperativo: ConectorOperativoActivo,
	}
}

func normalizarConectorOperacion(op *ConectorOperacion) {
	if op == nil {
		return
	}
	switch op.EstadoOperativo {
	case ConectorOperativoActivo, ConectorOperativoCircuitoAbierto:
	default:
		op.EstadoOperativo = ConectorOperativoActivo
	}
	if op.FallosConsecutivos < 0 {
		op.FallosConsecutivos = 0
	}
}
