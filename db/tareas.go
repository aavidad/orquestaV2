package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"orquesta/tareaspolicy"
)

type EstadoTarea string

const (
	// Constantes principales
	TareaLibre      EstadoTarea = "libre"
	TareaAsignada   EstadoTarea = "asignada"
	TareaEnProgreso EstadoTarea = "en_progreso"
	TareaCompletada EstadoTarea = "completada"
	TareaBloqueada  EstadoTarea = "bloqueada"
	TareaCancelada  EstadoTarea = "cancelada"
	TareaBacklog    EstadoTarea = "backlog" // pendiente para el futuro

	// Aliases con prefijo Estado (usados en cmd/)
	EstadoLibre      = TareaLibre
	EstadoAsignada   = TareaAsignada
	EstadoEnProgreso = TareaEnProgreso
	EstadoCompletada = TareaCompletada
	EstadoBloqueada  = TareaBloqueada
	EstadoCancelada  = TareaCancelada
	EstadoBacklog    = TareaBacklog
)

type TaskTransitionCoordinator interface {
	AfterTomarTarea(t *Tarea, agente string) error
	AfterIniciarTarea(t *Tarea, agente string) error
	AfterCompletarTarea(t *Tarea, agente, commit string) error
	AfterCancelarTarea(t *Tarea, agente, motivo string) error
	AfterBloquearTarea(t *Tarea, agente, motivo string) error
	AfterDesbloquearTarea(t *Tarea, agente, resolucion string) error
}

type defaultTaskTransitionCoordinator struct{}

var defaultTaskTransitioner TaskTransitionCoordinator = defaultTaskTransitionCoordinator{}

func SetTaskTransitionCoordinator(next TaskTransitionCoordinator) {
	if next == nil {
		next = defaultTaskTransitionCoordinator{}
	}
	defaultTaskTransitioner = next
}

func (defaultTaskTransitionCoordinator) AfterTomarTarea(t *Tarea, agente string) error {
	if t == nil {
		return nil
	}
	_ = agente
	_ = t
	return nil
}

func (defaultTaskTransitionCoordinator) AfterIniciarTarea(t *Tarea, agente string) error {
	if t == nil {
		return nil
	}
	_ = agente
	_ = t
	return nil
}

func (defaultTaskTransitionCoordinator) AfterCompletarTarea(t *Tarea, agente, commit string) error {
	if t == nil {
		return nil
	}
	_ = agente
	_ = t
	_ = commit
	return nil
}

func (defaultTaskTransitionCoordinator) AfterCancelarTarea(t *Tarea, agente, motivo string) error {
	if t == nil {
		return nil
	}
	_ = agente
	_ = t
	_ = motivo
	return nil
}

func (defaultTaskTransitionCoordinator) AfterBloquearTarea(t *Tarea, agente, motivo string) error {
	if t == nil {
		return nil
	}
	_ = agente
	_ = t
	_ = motivo
	return nil
}

func (defaultTaskTransitionCoordinator) AfterDesbloquearTarea(t *Tarea, agente, resolucion string) error {
	if t == nil {
		return nil
	}
	_ = agente
	_ = t
	_ = resolucion
	return nil
}

type PrioridadTarea string

const (
	PrioridadAlta  PrioridadTarea = "alta"
	PrioridadMedia PrioridadTarea = "media"
	PrioridadBaja  PrioridadTarea = "baja"
)

type Tarea struct {
	ID               int64
	Titulo           string
	Descripcion      string
	ProyectoID       *int64
	Modulo           string
	Estado           EstadoTarea
	Agente           *string // nil = sin asignar
	PropuestaID      *int64  // nil = no vinculada a propuesta
	Prioridad        PrioridadTarea
	Dependencias     []int64
	ContratoDefinido bool   // true = interfaz/contrato de E/S documentado (OP-069)
	BlueprintKey     string // clave única dentro del proyecto (evita duplicados en re-generación)
	CreadoPor        string
	CommitCierre     string
	Notas            string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CompletadaAt     *time.Time
}

type FiltroTareas struct {
	Estado      *EstadoTarea
	Agente      *string
	ProyectoID  *int64
	Modulo      *string
	PropuestaID *int64
	Libre       bool // solo las libre (sin agente)
	Limit       int
}

type ResumenBloqueo struct {
	ID     int64
	Titulo string
	Agente string
	Motivo string
}

// CrearTarea inserta una nueva tarea y devuelve su ID.
// Si blueprint_key no está vacío y ya existe una tarea con el mismo (proyecto_id, blueprint_key),
// devuelve 0, nil (operación idempotente: no es un error, la tarea ya existe).
func CrearTarea(t *Tarea) (int64, error) {
	deps, _ := json.Marshal(t.Dependencias)
	blueprintKey := strings.TrimSpace(t.BlueprintKey)
	if blueprintKey != "" && t.ProyectoID != nil && *t.ProyectoID > 0 {
		if existente := GetTareaIDBlueprintKey(*t.ProyectoID, blueprintKey); existente > 0 {
			return 0, nil
		}
	}
	id, err := insertReturningID(`
		INSERT INTO tareas (titulo, descripcion, proyecto_id, modulo, prioridad, dependencias, creado_por, notas, propuesta_id, blueprint_key)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		t.Titulo, t.Descripcion, t.ProyectoID, t.Modulo, t.Prioridad, string(deps), t.CreadoPor, t.Notas, t.PropuestaID, blueprintKey,
	)
	if err != nil {
		return 0, err
	}
	Audit(t.CreadoPor, "crear_tarea", "tarea", id, t.Titulo)
	return id, nil
}

// GetTareaActivaIDPorAgente devuelve el ID de la tarea que el agente tiene actualmente en curso.
// Busca prioritariamente en_progreso y luego asignada, devolviendo la más reciente.
func GetTareaActivaIDPorAgente(agente string) (int64, error) {
	return GetTareaActivaIDPorAgenteProyecto(agente, nil)
}

// GetTareaActivaIDPorAgenteProyecto devuelve la tarea activa más reciente del
// agente, opcionalmente acotada al proyecto.
func GetTareaActivaIDPorAgenteProyecto(agente string, proyectoID *int64) (int64, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return 0, nil
	}
	args := []any{agente}
	q := `
		SELECT id FROM tareas
		WHERE agente = ? AND estado IN ('en_progreso','asignada')`
	if proyectoID != nil && *proyectoID > 0 {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += `
		ORDER BY CASE WHEN estado = 'en_progreso' THEN 1 ELSE 2 END, updated_at DESC LIMIT 1`
	var id int64
	err := DB.QueryRow(q, args...).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

// GetTareaIDBlueprintKey devuelve el ID de una tarea por proyecto_id + blueprint_key.
// Devuelve 0 si no existe.
func GetTareaIDBlueprintKey(proyectoID int64, key string) int64 {
	id, _ := consultarConReintentos(func() (int64, error) {
		var found int64
		err := DB.QueryRow(`SELECT id FROM tareas WHERE proyecto_id=? AND blueprint_key=? LIMIT 1`, proyectoID, key).Scan(&found)
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return found, err
	})
	return id
}

// GetTarea devuelve una tarea por ID.
func GetTarea(id int64) (*Tarea, error) {
	return consultarConReintentos(func() (*Tarea, error) {
		row := DB.QueryRow(`
			SELECT id, titulo, descripcion, proyecto_id, modulo, estado, agente, propuesta_id, prioridad,
			       dependencias, creado_por, commit_cierre, notas,
			       created_at, updated_at, completada_at, contrato_definido
			FROM tareas WHERE id = ?`, id)
		return escanearTarea(row)
	})
}

// ListarTareas devuelve tareas según filtros opcionales.
func ListarTareas(f FiltroTareas) ([]*Tarea, error) {
	return consultarConReintentos(func() ([]*Tarea, error) {
		q := `SELECT id, titulo, descripcion, proyecto_id, modulo, estado, agente, propuesta_id, prioridad,
			         dependencias, creado_por, commit_cierre, notas,
			         created_at, updated_at, completada_at, contrato_definido
			  FROM tareas WHERE 1=1`
		args := []any{}

		if f.Libre {
			q += " AND estado = 'libre'"
		} else if f.Estado != nil {
			q += " AND estado = ?"
			args = append(args, *f.Estado)
		}
		if f.Agente != nil {
			q += " AND agente = ?"
			args = append(args, *f.Agente)
		}
		if f.ProyectoID != nil {
			q += " AND proyecto_id = ?"
			args = append(args, *f.ProyectoID)
		}
		if f.Modulo != nil {
			q += " AND modulo = ?"
			args = append(args, *f.Modulo)
		}
		if f.PropuestaID != nil {
			q += " AND propuesta_id = ?"
			args = append(args, *f.PropuestaID)
		}
		q += " ORDER BY CASE prioridad WHEN 'alta' THEN 1 WHEN 'media' THEN 2 ELSE 3 END, id"
		if f.Limit > 0 {
			q += " LIMIT ?"
			args = append(args, f.Limit)
		}

		rows, err := DB.Query(q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var list []*Tarea
		for rows.Next() {
			t, err := escanearTarea(rows)
			if err != nil {
				return nil, err
			}
			list = append(list, t)
		}
		return list, rows.Err()
	})
}

// ValidarDependencias comprueba que todas las dependencias previas de una tarea
// estén completadas y tengan contrato/interfaz definido. Implementa OP-069.
func ValidarDependencias(t *Tarea) error {
	if t == nil {
		return nil
	}
	return tareaspolicy.ValidateDependencies(&tareaspolicy.TaskSnapshot{
		ID:            t.ID,
		Title:         t.Titulo,
		DependencyIDs: t.Dependencias,
	}, func(depID int64) (*tareaspolicy.TaskSnapshot, error) {
		dep, err := GetTarea(depID)
		if err != nil {
			return nil, err
		}
		if dep == nil {
			return nil, nil
		}
		return &tareaspolicy.TaskSnapshot{
			ID:              dep.ID,
			Title:           dep.Titulo,
			Status:          string(dep.Estado),
			ContractDefined: dep.ContratoDefinido,
		}, nil
	})
}

// TomarTarea asigna una tarea libre (o backlog) a un agente.
// Valida dependencias (OP-069) antes de permitir la asignación.
func TomarTarea(id int64, agente string) error {
	t, err := GetTarea(id)
	if err != nil {
		return fmt.Errorf("tarea #%d no encontrada", id)
	}
	if t.Estado != TareaLibre && t.Estado != TareaBacklog {
		return fmt.Errorf("la tarea #%d está en estado '%s', solo se pueden tomar tareas 'libre' o 'backlog'", id, t.Estado)
	}
	if err := ValidarDependencias(t); err != nil {
		return err
	}
	_, err = DB.Exec(
		`UPDATE tareas SET estado='asignada', agente=? WHERE id=?`,
		agente, id,
	)
	if err == nil {
		err = defaultTaskTransitioner.AfterTomarTarea(t, agente)
	}
	return err
}

// DefinirContrato marca una tarea como con contrato/interfaz de E/S definido (OP-069).
func DefinirContrato(id int64, agente string) error {
	t, err := GetTarea(id)
	if err != nil {
		return fmt.Errorf("tarea #%d no encontrada", id)
	}
	_, err = DB.Exec(`UPDATE tareas SET contrato_definido=1 WHERE id=?`, id)
	if err == nil {
		Audit(agente, "definir_contrato", "tarea", id, t.Titulo)
	}
	return err
}

// IniciarTarea marca una tarea como en_progreso.
// También valida dependencias (OP-069) como segunda barrera de seguridad.
func IniciarTarea(id int64, agente string) error {
	t, err := GetTarea(id)
	if err != nil {
		return fmt.Errorf("tarea #%d no encontrada", id)
	}
	if t.Estado != TareaAsignada {
		return fmt.Errorf("la tarea #%d está en estado '%s', debe estar 'asignada' para iniciarla", id, t.Estado)
	}
	if err := ValidarDependencias(t); err != nil {
		return err
	}
	_, err = DB.Exec(
		`UPDATE tareas SET estado='en_progreso', agente=? WHERE id=?`,
		agente, id,
	)
	if err == nil {
		if err := defaultTaskTransitioner.AfterIniciarTarea(t, agente); err != nil {
			return err
		}
	}
	return err
}

// CompletarTarea marca una tarea como completada.
func CompletarTarea(id int64, agente, commit string) error {
	t, err := GetTarea(id)
	if err != nil {
		return fmt.Errorf("tarea #%d no encontrada", id)
	}
	if t.Estado != TareaEnProgreso && t.Estado != TareaAsignada {
		return fmt.Errorf("la tarea #%d está en estado '%s', debe estar en progreso o asignada", id, t.Estado)
	}
	_, err = DB.Exec(
		`UPDATE tareas SET estado='completada', commit_cierre=?, completada_at=CURRENT_TIMESTAMP WHERE id=?`,
		commit, id,
	)
	if err == nil {
		err = defaultTaskTransitioner.AfterCompletarTarea(t, agente, commit)
	}
	return err
}

// CancelarTarea marca una tarea como cancelada (no se va a realizar).
func CancelarTarea(id int64, agente, motivo string) error {
	t, err := GetTarea(id)
	if err != nil {
		return fmt.Errorf("tarea #%d no encontrada", id)
	}
	anotacion := formatearAnotacionTarea(agente, "CANCELADA: "+motivo, time.Now().UTC())
	_, err = DB.Exec(
		`UPDATE tareas SET estado='cancelada', notas = COALESCE(notas,'') || ? WHERE id=?`,
		anotacion, id,
	)
	if err == nil {
		err = defaultTaskTransitioner.AfterCancelarTarea(t, agente, motivo)
	}
	return err
}

// BloquearTarea marca una tarea como bloqueada y registra el bloqueo.
func BloquearTarea(id int64, agente, motivo string) error {
	t, err := GetTarea(id)
	if err != nil {
		return fmt.Errorf("tarea #%d no encontrada", id)
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`UPDATE tareas SET estado='bloqueada' WHERE id=?`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(
		`INSERT INTO bloqueos (tarea_id, motivo, bloqueado_por) VALUES (?,?,?)`,
		id, motivo, agente,
	); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return defaultTaskTransitioner.AfterBloquearTarea(t, agente, motivo)
}

// DesbloquearTarea resuelve el bloqueo y libera la tarea.
func DesbloquearTarea(id int64, agente, resolucion string) error {
	t, err := GetTarea(id)
	if err != nil {
		return fmt.Errorf("tarea #%d no encontrada", id)
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE tareas SET estado='libre', agente=NULL WHERE id=?`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`
		UPDATE bloqueos SET resuelto=1, resolucion=?, resuelto_at=CURRENT_TIMESTAMP
		WHERE tarea_id=? AND resuelto=0`, resolucion, id); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return defaultTaskTransitioner.AfterDesbloquearTarea(t, agente, resolucion)
}

// AnotarTarea añade una nota a una tarea.
func AnotarTarea(id int64, agente, nota string) error {
	anotacion := formatearAnotacionTarea(agente, nota, time.Now().UTC())
	_, err := DB.Exec(
		`UPDATE tareas SET notas = notas || ? WHERE id=?`,
		anotacion, id,
	)
	return err
}

func formatearAnotacionTarea(agente, nota string, marcaTiempo time.Time) string {
	return fmt.Sprintf(
		"\n%s [%s %s]",
		nota,
		marcaTiempo.UTC().Format("2006-01-02 15:04:05"),
		agente,
	)
}

func EnviarTareaABacklog(id int64) error {
	return MoverTareaABacklog(id)
}

func MoverTareaABacklog(id int64) error {
	_, err := DB.Exec(`UPDATE tareas SET estado='backlog', agente=NULL WHERE id=?`, id)
	return err
}

func ReasignarTarea(id int64, nuevoAgente string) error {
	_, err := DB.Exec(`UPDATE tareas SET estado='asignada', agente=? WHERE id=?`, nuevoAgente, id)
	return err
}

// ContarTareasPorEstado devuelve un mapa estado→cantidad.
func ContarTareasPorEstado() (map[string]int, error) {
	return consultarConReintentos(func() (map[string]int, error) {
		rows, err := DB.Query(`SELECT estado, COUNT(*) FROM tareas GROUP BY estado`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		m := make(map[string]int)
		for rows.Next() {
			var estado string
			var n int
			if err := rows.Scan(&estado, &n); err != nil {
				return nil, err
			}
			m[estado] = n
		}
		return m, rows.Err()
	})
}

// ─── helpers ────────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func escanearTarea(s scanner) (*Tarea, error) {
	var t Tarea
	var depsJSON string
	var agente sql.NullString
	var proyectoID sql.NullInt64
	var propuestaID sql.NullInt64
	var completadaAt sql.NullTime
	var contratoDefinido int
	err := s.Scan(
		&t.ID, &t.Titulo, &t.Descripcion, &proyectoID, &t.Modulo, &t.Estado, &agente, &propuestaID,
		&t.Prioridad, &depsJSON, &t.CreadoPor, &t.CommitCierre, &t.Notas,
		&t.CreatedAt, &t.UpdatedAt, &completadaAt, &contratoDefinido,
	)
	if err != nil {
		return nil, err
	}
	t.ContratoDefinido = contratoDefinido == 1
	if agente.Valid {
		t.Agente = &agente.String
	}
	if proyectoID.Valid {
		t.ProyectoID = &proyectoID.Int64
	}
	if propuestaID.Valid {
		t.PropuestaID = &propuestaID.Int64
	}
	_ = json.Unmarshal([]byte(depsJSON), &t.Dependencias)
	if completadaAt.Valid {
		t.CompletadaAt = &completadaAt.Time
	}
	return &t, nil
}

func MotivoBloqueoActivoTarea(tareaID int64) (string, error) {
	if tareaID <= 0 {
		return "", nil
	}
	return consultarConReintentos(func() (string, error) {
		var motivo sql.NullString
		err := DB.QueryRow(`
			SELECT motivo
			FROM bloqueos
			WHERE tarea_id = ? AND resuelto = 0
			ORDER BY id DESC
			LIMIT 1
		`, tareaID).Scan(&motivo)
		if err == sql.ErrNoRows {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(motivo.String), nil
	})
}

// ListarResumenBloqueos devuelve una lista de tareas bloqueadas con su motivo actual.
func ListarResumenBloqueos() ([]ResumenBloqueo, error) {
	return consultarConReintentos(func() ([]ResumenBloqueo, error) {
		rows, err := DB.Query(`
			SELECT t.id, t.titulo, b.bloqueado_por, b.motivo
			FROM tareas t
			JOIN bloqueos b ON t.id = b.tarea_id
			WHERE t.estado = 'bloqueada' AND b.resuelto = 0
			ORDER BY t.updated_at DESC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var res []ResumenBloqueo
		for rows.Next() {
			var rb ResumenBloqueo
			if err := rows.Scan(&rb.ID, &rb.Titulo, &rb.Agente, &rb.Motivo); err != nil {
				return nil, err
			}
			res = append(res, rb)
		}
		return res, rows.Err()
	})
}
