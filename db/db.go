package db

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

var DB *sql.DB
var dbMu sync.Mutex

// Open abre (o crea) la base de datos SQLite y aplica el schema.
// Orden de resolución de la ruta:
//  1. Variable de entorno ORQUESTA_DB
//  2. Repositorio `orquesta` del workspace actual (../orquesta/orquesta.db)
//  3. Si el git-root ya es el repo `orquesta`, <git-root>/orquesta.db
//  4. ./orquesta.db
func Open() error {
	dbMu.Lock()
	defer dbMu.Unlock()
	if DB != nil {
		return nil
	}
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
	postMigraciones()
	return nil
}

// postMigraciones ejecuta ALTER TABLE idempotentes para columnas añadidas tras el schema inicial.
func postMigraciones() {
	migraciones := []string{
		`ALTER TABLE agentes ADD COLUMN estado_sesion TEXT DEFAULT NULL`,
		`ALTER TABLE propuestas ADD COLUMN proyecto_id INTEGER REFERENCES proyectos(id)`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('pool_handoff_threshold_seconds', '1800')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('pool_handoff_threshold_ratio', '0.10')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('pool_default_budget_source', 'manual')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('model_policy_default_profile', 'implementacion')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('model_policy_default_reasoning', 'high')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('language.policy.default_language', 'es')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('language.policy.documentation_multilang', '1')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('language.policy.apps_multilang', '1')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('language.policy.documentation_default_language', 'es')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('language.policy.apps_default_language', 'es')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('language.policy.allowed_languages', 'es,en')`,
		`UPDATE reglas
		 SET descripcion='No escribir código sin propuesta OP-XXX aprobada en la app de orquestación.'
		 WHERE tipo_agente='programador' AND categoria='calidad' AND titulo='Propuesta antes de código'`,
		`UPDATE reglas
		 SET descripcion='Nueva propuesta: orquesta propuesta nueva "Título" --descripcion "Descripción" --agente <nombre>  |  Votar: orquesta votar <OP-XXX> <acuerdo|desacuerdo|abstencion> --agente <nombre> --comentario "razón"  |  Ver propuesta: orquesta propuesta ver <OP-XXX>  |  Ver pendientes de voto: incluidas en el briefing de sesion inicio.'
		 WHERE tipo_agente='programador' AND categoria='comandos' AND titulo='Propuestas y votación'`,
		`UPDATE workflows
		 SET pasos='["1. Leer la propuesta completa: orquesta propuesta ver <codigo>",
   "2. Analizar impacto técnico en módulos asignados",
   "3. Votar: orquesta votar <codigo> <acuerdo|desacuerdo|abstencion> --agente <mi-nombre> --comentario \"razón\"",
   "4. Si desacuerdo: añadir comentario técnico con alternativa concreta"]'
		 WHERE tipo_agente='programador' AND nombre='votar-propuesta'`,
		`UPDATE workflows
		 SET pasos='["1. Ejecutar: orquesta sesion inicio <mi-nombre>",
   "2. Ver tareas asignadas: orquesta tarea listar --agente <mi-nombre>",
   "3. Votar todas las propuestas con posicion pendiente para mi agente",
   "4. Iniciar la tarea en la app: orquesta tarea iniciar <id> <mi-nombre>",
   "5. Leer el doc del módulo asignado en docs/modulos/MXX_*.md"]'
		 WHERE tipo_agente='programador' AND nombre='inicio-sesion'`,
		`UPDATE workflows
		 SET pasos='["1. Asegurar que todo el trabajo está commiteado (git status limpio)",
   "2. Completar o bloquear mis tareas en la app de orquestación según corresponda",
   "3. Ejecutar: orquesta sesion fin <mi-nombre>"]'
		 WHERE tipo_agente='programador' AND nombre='fin-sesion'`,
		`UPDATE workflows
		 SET pasos='["1. Crear propuesta OP-XXX con orquesta propuesta nueva y esperar consenso en la app",
   "2. Crear fichero de dominio: internal/domain/<modulo>_entities.go",
   "3. Crear interfaces: internal/domain/<modulo>_interfaces.go",
   "4. Crear migración SQL: migrations/XXXXXX_<modulo>.up.sql",
   "5. Crear repositorio: internal/repository/postgres_<modulo>.go",
   "6. Commit: feat(MXX): dominio e interfaces",
   "7. Crear servicio: internal/service/<modulo>_service.go",
   "8. Commit: feat(MXX): servicio",
   "9. Crear tests: internal/service/<modulo>_service_test.go",
   "10. Ejecutar go test ./internal/service/... → debe pasar",
   "11. Commit: feat(MXX): tests servicio",
   "12. Crear handler REST: internal/api/<modulo>_handler.go",
   "13. Commit: feat(MXX): handler REST",
   "14. Gate final: go build ./... && go vet ./... && go test ./... → notificar a Antigravity"]'
		 WHERE tipo_agente='programador' AND nombre='crear-modulo'`,
		`UPDATE workflows
		 SET pasos='["1. Ejecutar: orquesta sesion inicio antigravity",
   "2. Ver tareas asignadas: orquesta tarea listar --agente antigravity",
   "3. Votar propuestas con posicion pendiente para antigravity",
   "4. Revisar docs/00_INDICE.md para detectar gaps"]'
		 WHERE tipo_agente='documentador' AND nombre='inicio-sesion'`,
	}
	for _, m := range migraciones {
		_, _ = DB.Exec(m) // ignorar "duplicate column name"
	}
	ensureLanguagePolicyDefaults()
}

func Close() {
	dbMu.Lock()
	defer dbMu.Unlock()
	if DB != nil {
		_ = DB.Close()
		DB = nil
	}
}

func IsOpen() bool {
	dbMu.Lock()
	defer dbMu.Unlock()
	return DB != nil
}

func resolverRuta() string {
	if v := os.Getenv("ORQUESTA_DB"); strings.TrimSpace(v) != "" {
		return v
	}
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return resolverRutaDesdeGitRoot(strings.TrimSpace(string(out)))
	}
	if wd, err := os.Getwd(); err == nil {
		if ruta := buscarRutaRepoOrquesta(wd); ruta != "" {
			return ruta
		}
	}
	return "orquesta.db"
}

func CurrentDBPath() string {
	return resolverRuta()
}

func resolverRutaDesdeGitRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return "orquesta.db"
	}
	if filepath.Base(root) == "orquesta" {
		return filepath.Join(root, "orquesta.db")
	}
	if ruta := rutaRepoOrquestaEnDirectorio(filepath.Dir(root)); ruta != "" {
		return ruta
	}
	return filepath.Join(root, "orquesta.db")
}

func buscarRutaRepoOrquesta(inicio string) string {
	actual := filepath.Clean(inicio)
	for {
		if ruta := rutaRepoOrquestaEnDirectorio(actual); ruta != "" {
			return ruta
		}
		siguiente := filepath.Dir(actual)
		if siguiente == actual {
			return ""
		}
		actual = siguiente
	}
}

func rutaRepoOrquestaEnDirectorio(base string) string {
	candidato := filepath.Join(base, "orquesta")
	if existeFichero(filepath.Join(candidato, "go.mod")) {
		return filepath.Join(candidato, "orquesta.db")
	}
	return ""
}

func existeFichero(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
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
