/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"database/sql"
	"fmt"
	"strings"

	"orquesta/storage"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresBackend struct{}

func init() {
	RegisterBackend(postgresBackend{})
}

func (postgresBackend) Name() string { return "postgres" }

func (postgresBackend) Open(cfg storage.Config) (*sql.DB, error) {
	db, err := storage.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("abriendo DB postgres en %s: %w", storage.DisplayTarget(cfg), err)
	}
	return db, nil
}

func (postgresBackend) Prepare(db *sql.DB, cfg storage.Config) error {
	if cfg.BootstrapSchema {
		if err := aplicarSchemaPorDriver(db, "postgres"); err != nil {
			return fmt.Errorf("aplicando schema postgres: %w", err)
		}
	}
	if err := postMigracionesPorDriver(db, "postgres"); err != nil {
		return fmt.Errorf("post-migraciones postgres: %w", err)
	}
	return nil
}

func (postgresBackend) Backup(_ *sql.DB, _ string) error {
	return fmt.Errorf("respaldo postgres no soportado todavia desde Orquesta; usa pg_dump o un adaptador de backup dedicado")
}

func (postgresBackend) Verify(raw *sql.DB, cfg storage.Config) *InformePersistencia {
	informe := nuevoInformePersistencia(cfg)
	if !verificarConexionPersistencia(informe, raw) {
		return informe
	}
	verificarTablasCorePersistencia(informe, raw, "postgres")
	return informe
}

func (postgresBackend) IsBusy(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "sqlstate 55p03") ||
		strings.Contains(msg, "lock not available") ||
		strings.Contains(msg, "could not obtain lock on relation")
}
