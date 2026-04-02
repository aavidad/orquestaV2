package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type AgenteIdentidadObservada struct {
	Agente     string
	Email      string
	Usuario    string
	Fuente     string
	ObservedAt time.Time
	UpdatedAt  time.Time
}

func ensureAgentesIdentidadObservadaSchema() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS agentes_identidad_observada (
			agente      TEXT PRIMARY KEY REFERENCES agentes(nombre) ON DELETE CASCADE,
			email       TEXT    NOT NULL DEFAULT '',
			usuario     TEXT    NOT NULL DEFAULT '',
			fuente      TEXT    NOT NULL DEFAULT '',
			observed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_agentes_identidad_observada_email ON agentes_identidad_observada(email, updated_at DESC)`)
	return err
}

func UpsertAgenteIdentidadObservada(agente, email, usuario, fuente string, observedAt *time.Time) error {
	agente = strings.TrimSpace(agente)
	email = strings.TrimSpace(email)
	usuario = strings.TrimSpace(usuario)
	fuente = strings.TrimSpace(fuente)
	if agente == "" {
		return fmt.Errorf("agente obligatorio")
	}
	if email == "" && usuario == "" {
		return fmt.Errorf("debes indicar email o usuario")
	}
	if _, err := GetAgente(agente); err != nil {
		return err
	}
	if err := ensureAgentesIdentidadObservadaSchema(); err != nil {
		return err
	}
	when := time.Now().UTC()
	if observedAt != nil && !observedAt.IsZero() {
		when = observedAt.UTC()
	}
	sqlText := buildUpsertValuesSQL(
		CurrentStorageDriver(),
		"agentes_identidad_observada",
		[]string{"agente", "email", "usuario", "fuente", "observed_at", "updated_at"},
		[]string{"agente"},
		[]upsertAssignment{
			{Column: "email"},
			{Column: "usuario"},
			{Column: "fuente"},
			{Column: "observed_at"},
			{Column: "updated_at"},
		},
	)
	_, err := DB.Exec(sqlText, agente, email, usuario, fuente, when, time.Now().UTC())
	if err != nil {
		return err
	}
	Audit("orquesta", "observar_cuenta_agente", "agente", 0, fmt.Sprintf("%s %s %s", agente, email, usuario))
	return nil
}

func UltimaIdentidadCuentaObservadaAgente(agente string) identidadCuentaAgente {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return identidadCuentaAgente{}
	}
	if err := ensureAgentesIdentidadObservadaSchema(); err != nil {
		return identidadCuentaAgente{}
	}
	row := DB.QueryRow(`
		SELECT email, usuario, fuente, observed_at
		FROM agentes_identidad_observada
		WHERE agente = ?`, agente)
	var email, usuario, fuente string
	var observedAt sql.NullTime
	if err := row.Scan(&email, &usuario, &fuente, &observedAt); err != nil {
		return identidadCuentaAgente{}
	}
	out := identidadCuentaAgente{
		email:   strings.TrimSpace(email),
		usuario: strings.TrimSpace(usuario),
		fuente:  strings.TrimSpace(fuente),
	}
	if observedAt.Valid {
		out.observedAt = &observedAt.Time
	}
	return out
}
