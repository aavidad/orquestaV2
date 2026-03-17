package db

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// Open abre (o crea) la base de datos SQLite y aplica el schema.
// Orden de resolución de la ruta:
//  1. Variable de entorno ORQUESTA_DB
//  2. <git-root>/orquesta.db
//  3. ./orquesta.db
func Open() error {
	path := resolverRuta()
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("abriendo DB en %s: %w", path, err)
	}
	db.SetMaxOpenConns(1) // SQLite no soporta escrituras concurrentes
	if err := aplicarSchema(db); err != nil {
		db.Close()
		return fmt.Errorf("aplicando schema: %w", err)
	}
	DB = db
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}

func resolverRuta() string {
	if v := os.Getenv("ORQUESTA_DB"); strings.TrimSpace(v) != "" {
		return v
	}
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return filepath.Join(strings.TrimSpace(string(out)), "orquesta.db")
	}
	return "orquesta.db"
}

func aplicarSchema(db *sql.DB) error {
	_, err := db.Exec(Schema)
	return err
}

// Audit registra una acción en el log de auditoría.
func Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	if DB == nil {
		return
	}
	_, _ = DB.Exec(
		`INSERT INTO audit_log (agente, accion, entidad, entidad_id, detalle) VALUES (?,?,?,?,?)`,
		agente, accion, entidad, entidadID, detalle,
	)
}

// ConfigGet devuelve el valor de una clave de configuración.
func ConfigGet(clave string) (string, error) {
	var v string
	err := DB.QueryRow(`SELECT valor FROM config WHERE clave = ?`, clave).Scan(&v)
	return v, err
}

// ConfigSet actualiza o inserta una clave de configuración.
func ConfigSet(clave, valor string) error {
	_, err := DB.Exec(
		`INSERT INTO config (clave, valor) VALUES (?,?) ON CONFLICT(clave) DO UPDATE SET valor=excluded.valor`,
		clave, valor,
	)
	return err
}

// ConfigAll devuelve toda la configuración.
func ConfigAll() (map[string]string, error) {
	rows, err := DB.Query(`SELECT clave, valor FROM config ORDER BY clave`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, rows.Err()
}
