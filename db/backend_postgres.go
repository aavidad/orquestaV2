/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

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

func (postgresBackend) Backup(db *sql.DB, path string) error {
	if db == nil {
		return fmt.Errorf("base de datos postgres no inicializada")
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("crear respaldo postgres %s: %w", path, err)
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(map[string]any{
		"type":       "header",
		"driver":     "postgres",
		"generated":  time.Now().UTC().Format(time.RFC3339Nano),
		"format":     "orquesta-postgres-logical-backup-v1",
		"targetFile": path,
	}); err != nil {
		return fmt.Errorf("escribir cabecera de respaldo postgres: %w", err)
	}

	rows, err := db.Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name`)
	if err != nil {
		return fmt.Errorf("listar tablas postgres para respaldo: %w", err)
	}
	defer rows.Close()

	var tablas []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return fmt.Errorf("leer tabla postgres para respaldo: %w", err)
		}
		tablas = append(tablas, strings.TrimSpace(table))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterar tablas postgres para respaldo: %w", err)
	}

	for _, table := range tablas {
		if table == "" {
			continue
		}
		if err := enc.Encode(map[string]any{
			"type":  "table",
			"name":  table,
			"stage": "begin",
		}); err != nil {
			return fmt.Errorf("escribir inicio de tabla %s en respaldo postgres: %w", table, err)
		}

		query := fmt.Sprintf(`SELECT row_to_json(t)::text FROM (SELECT * FROM "%s") t`, strings.ReplaceAll(table, `"`, `""`))
		tableRows, err := db.Query(query)
		if err != nil {
			return fmt.Errorf("leer filas de %s para respaldo postgres: %w", table, err)
		}
		for tableRows.Next() {
			var raw string
			if err := tableRows.Scan(&raw); err != nil {
				tableRows.Close()
				return fmt.Errorf("scan fila de %s para respaldo postgres: %w", table, err)
			}
			var payload any
			if err := json.Unmarshal([]byte(raw), &payload); err != nil {
				tableRows.Close()
				return fmt.Errorf("decodificar fila de %s para respaldo postgres: %w", table, err)
			}
			if err := enc.Encode(map[string]any{
				"type":  "row",
				"table": table,
				"data":  payload,
			}); err != nil {
				tableRows.Close()
				return fmt.Errorf("escribir fila de %s en respaldo postgres: %w", table, err)
			}
		}
		if err := tableRows.Err(); err != nil {
			tableRows.Close()
			return fmt.Errorf("iterar filas de %s para respaldo postgres: %w", table, err)
		}
		if err := tableRows.Close(); err != nil {
			return fmt.Errorf("cerrar cursor de %s en respaldo postgres: %w", table, err)
		}
		if err := enc.Encode(map[string]any{
			"type":  "table",
			"name":  table,
			"stage": "end",
		}); err != nil {
			return fmt.Errorf("escribir cierre de tabla %s en respaldo postgres: %w", table, err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush respaldo postgres: %w", err)
	}
	return nil
}

func (postgresBackend) SupportsBackup() bool { return true }

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
