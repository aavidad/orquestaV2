package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Agente struct {
	Nombre                 string
	Rol                    string
	Activo                 bool   // en sesión ahora mismo
	Habilitado             bool   // false = retirado por Alberto
	EstadoSesion           string // disponible | programando | esperando | votando
	UltimaSesion           *time.Time
	ConsumoDiaSegundos     int
	ConsumoSemanalSegundos int
	LimiteDiaSegundos      int
	LimiteSemanalSegundos  int
	LastUsageResetAt       *time.Time
	EstadoCuota            string
	ReanimarAt             *time.Time
	MotivoPausa            string
}

type Sesion struct {
	ID                 int64
	Agente             string
	ConectorID         *int64
	ConectorSlug       string
	ConectorNombre     string
	ProyectoID         *int64
	ProyectoSlug       string
	ProyectoNombre     string
	Inicio             time.Time
	Fin                *time.Time
	Activa             bool
	Estado             string
	CWD                string
	Herramienta        string
	ExternalSessionID  string
	ResumePayloadJSON  string
	ResumenContinuidad string
	Branch             string
	HeartbeatAt        *time.Time
	Host               string
	PID                *int64
}

type SesionInicio struct {
	Agente             string
	ConectorID         *int64
	ProyectoID         *int64
	CWD                string
	Herramienta        string
	ExternalSessionID  string
	ResumePayloadJSON  string
	ResumenContinuidad string
	Branch             string
	Host               string
	PID                *int64
}

type SesionUpdate struct {
	CWD                *string
	Herramienta        *string
	ExternalSessionID  *string
	ResumePayloadJSON  *string
	ResumenContinuidad *string
	Branch             *string
	Host               *string
	PID                *int64
	Heartbeat          bool
	Estado             *string
}

// IniciarSesion marca al agente como activo y crea una sesión.
func IniciarSesion(agente string) (int64, error) {
	s, err := IniciarSesionContexto(SesionInicio{Agente: agente})
	if err != nil {
		return 0, err
	}
	return s.ID, nil
}

func IniciarSesionContexto(in SesionInicio) (*Sesion, error) {
	agente := strings.TrimSpace(in.Agente)
	// Verificar que el agente existe y está habilitado
	var rol string
	var habilitado bool
	err := DB.QueryRow(`SELECT rol, habilitado FROM agentes WHERE nombre = ?`, agente).Scan(&rol, &habilitado)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agente '%s' no registrado; usa 'orquesta config agente-nuevo' para registrarlo", agente)
	}
	if err != nil {
		return nil, err
	}
	if !habilitado {
		return nil, fmt.Errorf("agente '%s' está retirado y no puede iniciar sesión", agente)
	}

	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Cerrar sesiones anteriores abiertas
	_, _ = tx.Exec(`UPDATE sesiones SET activa=0, estado='cerrada', fin=CURRENT_TIMESTAMP WHERE agente=? AND activa=1`, agente)

	// Marcar activo y estado inicial
	_, err = tx.Exec(`UPDATE agentes SET activo=1, estado_sesion='disponible', ultima_sesion=CURRENT_TIMESTAMP WHERE nombre=?`, agente)
	if err != nil {
		return nil, err
	}

	// Crear nueva sesión
<<<<<<< HEAD
	res, err := tx.Exec(`
		INSERT INTO sesiones (agente, conector_id, proyecto_id, cwd, herramienta, external_session_id, resume_payload_json, resumen_continuidad, branch, heartbeat_at, host, pid)
		VALUES (?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP,?,?)`,
		agente, in.ConectorID, in.ProyectoID, in.CWD, in.Herramienta, in.ExternalSessionID, in.ResumePayloadJSON, in.ResumenContinuidad, in.Branch, in.Host, in.PID,
	)
=======
	id, err := insertReturningIDWith(tx, `INSERT INTO sesiones (agente) VALUES (?)`, agente)
>>>>>>> origin/orq-orquestador-codex2
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	Audit(agente, "inicio_sesion", "sesion", id, "")
	sesion, err := GetSesionByID(id)
	if err != nil {
		return nil, err
	}
	if err := UpsertRuntimeDesdeSesion(sesion); err != nil {
		return nil, err
	}
	if err := UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		return nil, err
	}
	return sesion, nil
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
		UPDATE sesiones SET activa=0, estado='cerrada', fin=CURRENT_TIMESTAMP WHERE agente=? AND activa=1`, agente)
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
	if err := MarcarRuntimesCerradosPorAgente(agente); err != nil {
		return err
	}
	if err := MarcarRuntimeHandlesCerradosPorAgente(agente); err != nil {
		return err
	}
	Audit(agente, "fin_sesion", "sesion", 0, "")
	return nil
}

func GetSesionByID(id int64) (*Sesion, error) {
	row := DB.QueryRow(`
		SELECT s.id, s.agente, s.conector_id, COALESCE(c.slug,''), COALESCE(c.nombre,''),
		       s.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
		       s.inicio, s.fin, s.activa, s.estado, s.cwd, s.herramienta,
		       s.external_session_id, s.resume_payload_json, s.resumen_continuidad,
		       s.branch, s.heartbeat_at, s.host, s.pid
		FROM sesiones s
		LEFT JOIN conectores c ON c.id = s.conector_id
		LEFT JOIN proyectos p ON p.id = s.proyecto_id
		WHERE s.id = ?`, id)
	return escanearSesion(row)
}

func ObtenerUltimaSesion(agente string, proyectoID *int64) (*Sesion, error) {
	return ObtenerUltimaSesionConFiltro(agente, proyectoID, "")
}

func ObtenerUltimaSesionConFiltro(agente string, proyectoID *int64, cwd string) (*Sesion, error) {
	q := `
		SELECT s.id, s.agente, s.conector_id, COALESCE(c.slug,''), COALESCE(c.nombre,''),
		       s.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
		       s.inicio, s.fin, s.activa, s.estado, s.cwd, s.herramienta,
		       s.external_session_id, s.resume_payload_json, s.resumen_continuidad,
		       s.branch, s.heartbeat_at, s.host, s.pid
		FROM sesiones s
		LEFT JOIN conectores c ON c.id = s.conector_id
		LEFT JOIN proyectos p ON p.id = s.proyecto_id
		WHERE s.agente = ?`
	args := []any{agente}
	if proyectoID != nil {
		q += ` AND s.proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	if strings.TrimSpace(cwd) != "" {
		q += ` AND s.cwd = ?`
		args = append(args, strings.TrimSpace(cwd))
	}
	q += ` ORDER BY s.id DESC LIMIT 1`
	return escanearSesion(DB.QueryRow(q, args...))
}

func GuardarSesionActiva(agente string, proyectoID *int64, upd SesionUpdate) error {
	q := `SELECT id FROM sesiones WHERE agente = ? AND activa = 1`
	args := []any{agente}
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY id DESC LIMIT 1`

	var id int64
	if err := DB.QueryRow(q, args...).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("el agente '%s' no tiene sesión activa", agente)
		}
		return err
	}

	partes := make([]string, 0, 8)
	updateArgs := make([]any, 0, 10)
	if upd.CWD != nil {
		partes = append(partes, "cwd = ?")
		updateArgs = append(updateArgs, *upd.CWD)
	}
	if upd.Herramienta != nil {
		partes = append(partes, "herramienta = ?")
		updateArgs = append(updateArgs, *upd.Herramienta)
	}
	if upd.ExternalSessionID != nil {
		partes = append(partes, "external_session_id = ?")
		updateArgs = append(updateArgs, *upd.ExternalSessionID)
	}
	if upd.ResumePayloadJSON != nil {
		partes = append(partes, "resume_payload_json = ?")
		updateArgs = append(updateArgs, *upd.ResumePayloadJSON)
	}
	if upd.ResumenContinuidad != nil {
		partes = append(partes, "resumen_continuidad = ?")
		updateArgs = append(updateArgs, *upd.ResumenContinuidad)
	}
	if upd.Branch != nil {
		partes = append(partes, "branch = ?")
		updateArgs = append(updateArgs, *upd.Branch)
	}
	if upd.Host != nil {
		partes = append(partes, "host = ?")
		updateArgs = append(updateArgs, *upd.Host)
	}
	if upd.PID != nil {
		partes = append(partes, "pid = ?")
		updateArgs = append(updateArgs, *upd.PID)
	}
	if upd.Estado != nil {
		partes = append(partes, "estado = ?")
		updateArgs = append(updateArgs, *upd.Estado)
	}
	if upd.Heartbeat {
		partes = append(partes, "heartbeat_at = CURRENT_TIMESTAMP")
		// Solo incrementamos estadísticas de uso estimado, pero ya no bloqueamos localmente (OP-084)
		tick := configIntOrDefault("agent_tick_seconds", 30)
		_, _ = DB.Exec(`
			UPDATE agentes 
			SET consumo_dia_segundos = consumo_dia_segundos + ?, 
			    consumo_semanal_segundos = consumo_semanal_segundos + ?
			WHERE nombre = ?`, tick, tick, agente)
	}
	if len(partes) == 0 {
		return nil
	}

	updateArgs = append(updateArgs, id)
	_, err := DB.Exec(`UPDATE sesiones SET `+strings.Join(partes, ", ")+` WHERE id = ?`, updateArgs...)
	if err == nil {
		sesion, getErr := GetSesionByID(id)
		if getErr != nil {
			err = getErr
		} else {
			err = UpsertRuntimeDesdeSesion(sesion)
			if err == nil {
				err = UpsertRuntimeHandleDesdeSesion(sesion)
			}
		}
	}
	if err == nil {
		Audit(agente, "guardar_sesion", "sesion", id, "")
	}
	return err
}

func ListarSesionesActivas() ([]*Sesion, error) {
	rows, err := DB.Query(`
		SELECT s.id, s.agente, s.conector_id, COALESCE(c.slug,''), COALESCE(c.nombre,''),
		       s.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
		       s.inicio, s.fin, s.activa, s.estado, s.cwd, s.herramienta,
		       s.external_session_id, s.resume_payload_json, s.resumen_continuidad,
		       s.branch, s.heartbeat_at, s.host, s.pid
		FROM sesiones s
		LEFT JOIN conectores c ON c.id = s.conector_id
		LEFT JOIN proyectos p ON p.id = s.proyecto_id
		WHERE s.activa = 1
		ORDER BY s.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Sesion
	for rows.Next() {
		s, err := escanearSesion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListarAgentes devuelve todos los agentes registrados con su estado de cuota.
func ListarAgentes() ([]*Agente, error) {
	rows, err := DB.Query(`
		SELECT nombre, rol, activo, habilitado, COALESCE(estado_sesion,''), ultima_sesion,
		       consumo_dia_segundos, consumo_semanal_segundos, limite_dia_segundos,
		       limite_semanal_segundos, last_usage_reset_at, estado_cuota,
		       reanimar_at, motivo_pausa
		FROM agentes ORDER BY nombre`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Agente
	for rows.Next() {
		a := &Agente{}
		var ultima sql.NullTime
		var lastReset sql.NullTime
		var reanimar sql.NullTime
		var motivo sql.NullString
		if err := rows.Scan(
			&a.Nombre, &a.Rol, &a.Activo, &a.Habilitado, &a.EstadoSesion, &ultima,
			&a.ConsumoDiaSegundos, &a.ConsumoSemanalSegundos, &a.LimiteDiaSegundos,
			&a.LimiteSemanalSegundos, &lastReset, &a.EstadoCuota,
			&reanimar, &motivo,
		); err != nil {
			return nil, err
		}
		if ultima.Valid {
			a.UltimaSesion = &ultima.Time
		}
		if lastReset.Valid {
			a.LastUsageResetAt = &lastReset.Time
		}
		if reanimar.Valid {
			a.ReanimarAt = &reanimar.Time
		}
		a.MotivoPausa = motivo.String
		list = append(list, a)
	}
	return list, rows.Err()
}

func GetAgente(nombre string) (*Agente, error) {
	row := DB.QueryRow(`
		SELECT nombre, rol, activo, habilitado, COALESCE(estado_sesion,''), ultima_sesion,
		       consumo_dia_segundos, consumo_semanal_segundos, limite_dia_segundos,
		       limite_semanal_segundos, last_usage_reset_at, estado_cuota,
		       reanimar_at, motivo_pausa
		FROM agentes WHERE nombre = ?`, nombre)
	a := &Agente{}
	var ultima sql.NullTime
	var lastReset sql.NullTime
	var reanimar sql.NullTime
	var motivo sql.NullString
	if err := row.Scan(
		&a.Nombre, &a.Rol, &a.Activo, &a.Habilitado, &a.EstadoSesion, &ultima,
		&a.ConsumoDiaSegundos, &a.ConsumoSemanalSegundos, &a.LimiteDiaSegundos,
		&a.LimiteSemanalSegundos, &lastReset, &a.EstadoCuota,
		&reanimar, &motivo,
	); err != nil {
		return nil, err
	}
	if ultima.Valid {
		a.UltimaSesion = &ultima.Time
	}
	if lastReset.Valid {
		a.LastUsageResetAt = &lastReset.Time
	}
	if reanimar.Valid {
		a.ReanimarAt = &reanimar.Time
	}
	a.MotivoPausa = motivo.String
	return a, nil
}

// CheckReanimaciones busca agentes cuya fecha de reanimación ha vencido.
func CheckReanimaciones() ([]*Agente, error) {
	rows, err := DB.Query(`
		SELECT nombre, rol, activo, habilitado, COALESCE(estado_sesion,''), ultima_sesion,
		       consumo_dia_segundos, consumo_semanal_segundos, limite_dia_segundos,
		       limite_semanal_segundos, last_usage_reset_at, estado_cuota,
		       reanimar_at, motivo_pausa
		FROM agentes
		WHERE reanimar_at IS NOT NULL AND reanimar_at <= CURRENT_TIMESTAMP`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Agente
	for rows.Next() {
		a := &Agente{}
		var ultima, lastReset, reanimar sql.NullTime
		var motivo sql.NullString
		if err := rows.Scan(
			&a.Nombre, &a.Rol, &a.Activo, &a.Habilitado, &a.EstadoSesion, &ultima,
			&a.ConsumoDiaSegundos, &a.ConsumoSemanalSegundos, &a.LimiteDiaSegundos,
			&a.LimiteSemanalSegundos, &lastReset, &a.EstadoCuota,
			&reanimar, &motivo,
		); err != nil {
			return nil, err
		}
		if ultima.Valid {
			a.UltimaSesion = &ultima.Time
		}
		if lastReset.Valid {
			a.LastUsageResetAt = &lastReset.Time
		}
		if reanimar.Valid {
			a.ReanimarAt = &reanimar.Time
		}
		a.MotivoPausa = motivo.String
		list = append(list, a)
	}
	return list, rows.Err()
}

// ResetReanimacion limpia los campos de reanimación de un agente.
func ResetReanimacion(nombre string) error {
	_, err := DB.Exec(`UPDATE agentes SET reanimar_at = NULL, motivo_pausa = NULL, estado_cuota = 'activo' WHERE nombre = ?`, nombre)
	return err
}

// EliminarAgente borra físicamente un agente de la base de datos.
func EliminarAgente(nombre string) error {
	_, err := DB.Exec(`DELETE FROM agentes WHERE nombre = ?`, nombre)
	return err
}

// PausarAgente establece una pausa forzada (por rate-limit externo) para un agente.
func PausarAgente(nombre string, minutos int, motivo string) error {
	reanimar := time.Now().Add(time.Duration(minutos) * time.Minute)
	_, err := DB.Exec(`
		UPDATE agentes 
		SET estado_cuota = 'enfriamiento', reanimar_at = ?, motivo_pausa = ? 
		WHERE nombre = ?`, reanimar, motivo, nombre)
	return err
}

// SetEstadoSesion actualiza el estado de actividad de un agente en sesión.
func SetEstadoSesion(agente, estado string) {
	if DB == nil || agente == "" {
		return
	}
	_, _ = DB.Exec(`UPDATE agentes SET estado_sesion=? WHERE nombre=? AND activo=1`, estado, agente)
}

func GetSesionActiva(agente string, proyectoID *int64) (*Sesion, error) {
	q := `
		SELECT s.id, s.agente, s.conector_id, COALESCE(c.slug,''), COALESCE(c.nombre,''),
		       s.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
		       s.inicio, s.fin, s.activa, s.estado, s.cwd, s.herramienta,
		       s.external_session_id, s.resume_payload_json, s.resumen_continuidad,
		       s.branch, s.heartbeat_at, s.host, s.pid
		FROM sesiones s
		LEFT JOIN conectores c ON c.id = s.conector_id
		LEFT JOIN proyectos p ON p.id = s.proyecto_id
		WHERE s.agente = ? AND s.activa = 1`
	args := []any{agente}
	if proyectoID != nil {
		q += ` AND s.proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY s.id DESC LIMIT 1`
	return escanearSesion(DB.QueryRow(q, args...))
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

func escanearSesion(s scanner) (*Sesion, error) {
	var sesion Sesion
	var conectorID sql.NullInt64
	var proyectoID sql.NullInt64
	var fin sql.NullTime
	var heartbeat sql.NullTime
	var pid sql.NullInt64
	err := s.Scan(
		&sesion.ID, &sesion.Agente, &conectorID, &sesion.ConectorSlug, &sesion.ConectorNombre,
		&proyectoID, &sesion.ProyectoSlug, &sesion.ProyectoNombre,
		&sesion.Inicio, &fin, &sesion.Activa, &sesion.Estado, &sesion.CWD, &sesion.Herramienta,
		&sesion.ExternalSessionID, &sesion.ResumePayloadJSON, &sesion.ResumenContinuidad,
		&sesion.Branch, &heartbeat, &sesion.Host, &pid,
	)
	if err != nil {
		return nil, err
	}
	if conectorID.Valid {
		sesion.ConectorID = &conectorID.Int64
	}
	if proyectoID.Valid {
		sesion.ProyectoID = &proyectoID.Int64
	}
	if fin.Valid {
		sesion.Fin = &fin.Time
	}
	if heartbeat.Valid {
		sesion.HeartbeatAt = &heartbeat.Time
	}
	if pid.Valid {
		sesion.PID = &pid.Int64
	}
	return &sesion, nil
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
