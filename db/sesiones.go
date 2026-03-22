package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Agente struct {
	Nombre       string
	Rol          string
	Activo       bool   // en sesión ahora mismo
	Habilitado   bool   // false = retirado por Alberto
	EstadoSesion string // disponible | programando | esperando | votando
	UltimaSesion *time.Time
}

// IniciarSesion marca al agente como activo y crea una sesión.
func IniciarSesion(agente string) (int64, error) {
	// Verificar que el agente existe y está habilitado
	var rol string
	var habilitado bool
	err := DB.QueryRow(`SELECT rol, habilitado FROM agentes WHERE nombre = ?`, agente).Scan(&rol, &habilitado)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("agente '%s' no registrado; usa 'orquesta config agente-nuevo' para registrarlo", agente)
	}
	if err != nil {
		return 0, err
	}
	if !habilitado {
		return 0, fmt.Errorf("agente '%s' está retirado y no puede iniciar sesión", agente)
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Cerrar sesiones anteriores abiertas
	_, _ = tx.Exec(`UPDATE sesiones SET activa=0, fin=CURRENT_TIMESTAMP WHERE agente=? AND activa=1`, agente)

	// Marcar activo y estado inicial
	_, err = tx.Exec(`UPDATE agentes SET activo=1, estado_sesion='disponible', ultima_sesion=CURRENT_TIMESTAMP WHERE nombre=?`, agente)
	if err != nil {
		return 0, err
	}

	// Crear nueva sesión
	res, err := tx.Exec(`INSERT INTO sesiones (agente) VALUES (?)`, agente)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	Audit(agente, "inicio_sesion", "sesion", id, "")
	return id, nil
}

// FinSesion marca al agente como inactivo y cierra su sesión.
func FinSesion(agente string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE agentes SET activo=0, estado_sesion=NULL WHERE nombre=?`, agente)
	if err != nil {
		return err
	}
	res, err := tx.Exec(`
		UPDATE sesiones SET activa=0, fin=CURRENT_TIMESTAMP WHERE agente=? AND activa=1`, agente)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("el agente '%s' no tenía sesión activa", agente)
	}
	Audit(agente, "fin_sesion", "sesion", 0, "")
	return nil
}

// ListarAgentes devuelve todos los agentes registrados.
func ListarAgentes() ([]*Agente, error) {
	rows, err := DB.Query(`SELECT nombre, rol, activo, habilitado, COALESCE(estado_sesion,''), ultima_sesion FROM agentes ORDER BY nombre`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Agente
	for rows.Next() {
		a := &Agente{}
		var ultima sql.NullTime
		if err := rows.Scan(&a.Nombre, &a.Rol, &a.Activo, &a.Habilitado, &a.EstadoSesion, &ultima); err != nil {
			return nil, err
		}
		if ultima.Valid {
			a.UltimaSesion = &ultima.Time
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// SetEstadoSesion actualiza el estado de actividad de un agente en sesión.
func SetEstadoSesion(agente, estado string) {
	if DB == nil || agente == "" {
		return
	}
	_, _ = DB.Exec(`UPDATE agentes SET estado_sesion=? WHERE nombre=? AND activo=1`, estado, agente)
}

// RegistrarAgente añade un nuevo agente al sistema.
func RegistrarAgente(nombre, rol string) error {
	_, err := DB.Exec(
		upsertValuesSQL(
			"agentes",
			[]string{"nombre", "rol"},
			[]string{"nombre"},
			[]upsertAssignment{
				{Column: "rol"},
				{Column: "habilitado", Expr: "1"},
			},
		),
		nombre, rol,
	)
	return err
}

// RetirarAgente deshabilita a un agente: no puede votar ni trabajar.
// Sus votos pendientes en propuestas abiertas se eliminan para que no bloqueen el consenso.
func RetirarAgente(nombre string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Deshabilitar + cerrar sesión activa
	if _, err = tx.Exec(
		`UPDATE agentes SET habilitado=0, activo=0 WHERE nombre=?`, nombre); err != nil {
		return err
	}
	if _, err = tx.Exec(
		`UPDATE sesiones SET activa=0, fin=CURRENT_TIMESTAMP WHERE agente=? AND activa=1`, nombre); err != nil {
		return err
	}
	// Eliminar votos pendientes en propuestas abiertas (para no bloquear consenso)
	if _, err = tx.Exec(`
		DELETE FROM votos
		WHERE agente=? AND posicion='pendiente'
		  AND propuesta_id IN (SELECT id FROM propuestas WHERE estado='abierta')`, nombre); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	Audit("alberto", "retirar_agente", "agente", 0, nombre)
	return nil
}

// RehabilitarAgente reactiva a un agente retirado.
func RehabilitarAgente(nombre string) error {
	res, err := DB.Exec(`UPDATE agentes SET habilitado=1 WHERE nombre=?`, nombre)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agente '%s' no encontrado", nombre)
	}
	Audit("alberto", "rehabilitar_agente", "agente", 0, nombre)
	return nil
}

// RegistrarCodex registra un agente Codex y devuelve el nombre asignado (codex1, codex2…).
// Si el agente ya existe con ese nombre, lo devuelve tal cual.
// El nombre se asigna por orden: el siguiente número libre tras los existentes.
func RegistrarCodex() (string, error) {
	// Buscar todos los agentes cuyo nombre empieza por "codex"
	rows, err := DB.Query(`SELECT nombre FROM agentes WHERE nombre LIKE 'codex%' ORDER BY nombre`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	maxN := 0
	for rows.Next() {
		var nombre string
		if err := rows.Scan(&nombre); err != nil {
			return "", err
		}
		var n int
		// codex1, codex2… extraemos el número
		suffix := strings.TrimPrefix(nombre, "codex")
		if _, err := fmt.Sscanf(suffix, "%d", &n); err == nil && n > maxN {
			maxN = n
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	nombre := fmt.Sprintf("codex%d", maxN+1)
	if err := RegistrarAgente(nombre, "programador"); err != nil {
		return "", err
	}
	return nombre, nil
}

// AuditEntry es una entrada del log de auditoría.
type AuditEntry struct {
	Agente    string
	Accion    string
	Entidad   string
	EntidadID int64
	Detalle   string
	CreatedAt time.Time
}

// AuditLog devuelve las últimas N entradas del log.
func AuditLog(limit int) ([]AuditEntry, error) {
	rows, err := DB.Query(
		`SELECT agente, accion, entidad, entidad_id, detalle, created_at
		 FROM audit_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.Agente, &e.Accion, &e.Entidad, &e.EntidadID, &e.Detalle, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}
