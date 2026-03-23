/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

// Package db — Canal de Refinería (OP-093)
//
// La Refinería es la "aduana" de calidad que interpone Orquesta entre el trabajo
// de un agente y la rama principal. El flujo es:
//
//  1. El agente llama a SolicitarRefineria(tareaID, …) en lugar de CompletarTarea.
//  2. El control plane recoge la solicitud (ProcesarRefineriaBatch), ejecuta el
//     comando de tests en el directorio de trabajo indicado y:
//     - Si los tests pasan  → llama a AprobarRefineria: completa la tarea y envía
//       un mensaje de éxito al buzón del agente.
//     - Si los tests fallan → llama a RechazarRefineria: deja la tarea en
//       en_progreso y envía el output de error al buzón del agente.
//  3. El agente consulta su buzón y actúa en consecuencia.
package db

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RefineriaSolicitud representa una petición de validación en la cola de la Refinería.
type RefineriaSolicitud struct {
	ID          int64      `json:"id"`
	TareaID     int64      `json:"tarea_id"`
	Agente      string     `json:"agente"`
	ProyectoID  *int64     `json:"proyecto_id,omitempty"`
	Rama        string     `json:"rama"`
	DirTrabajo  string     `json:"dir_trabajo"`
	CmdTest     string     `json:"cmd_test"`
	Estado      string     `json:"estado"`
	Resultado   string     `json:"resultado"`
	ErrorTexto  string     `json:"error_texto"`
	CommitMerge string     `json:"commit_merge"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

// SolicitarRefineria encola una solicitud de validación para una tarea en progreso.
// No modifica el estado de la tarea; el agente continúa en en_progreso hasta que la
// Refinería la apruebe o rechace.
func SolicitarRefineria(tareaID int64, agente, rama, dirTrabajo, cmdTest string) (*RefineriaSolicitud, error) {
	t, err := GetTarea(tareaID)
	if err != nil {
		return nil, fmt.Errorf("tarea #%d no encontrada", tareaID)
	}
	if t.Estado != TareaEnProgreso && t.Estado != TareaAsignada {
		return nil, fmt.Errorf("la tarea #%d está en estado '%s'; solo se puede solicitar refinería desde en_progreso o asignada", tareaID, t.Estado)
	}
	if strings.TrimSpace(cmdTest) == "" {
		cmdTest = "go test ./..."
	}

	res, err := DB.Exec(`
		INSERT INTO refineria_solicitudes
			(tarea_id, agente, proyecto_id, rama, dir_trabajo, cmd_test, estado)
		VALUES (?,?,?,?,?,?,'pendiente')`,
		tareaID, agente, t.ProyectoID, rama, dirTrabajo, cmdTest,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	Audit(agente, "solicitar_refineria", "refineria_solicitud", id,
		fmt.Sprintf("tarea #%d rama=%s dir=%s cmd=%s", tareaID, rama, dirTrabajo, cmdTest))

	return &RefineriaSolicitud{
		ID:         id,
		TareaID:    tareaID,
		Agente:     agente,
		ProyectoID: t.ProyectoID,
		Rama:       rama,
		DirTrabajo: dirTrabajo,
		CmdTest:    cmdTest,
		Estado:     "pendiente",
		CreatedAt:  time.Now(),
	}, nil
}

// ListarRefineriasPendientes devuelve las solicitudes en estado pendiente, ordenadas
// por antigüedad. Usada por el control plane para procesar en FIFO.
func ListarRefineriasPendientes() ([]*RefineriaSolicitud, error) {
	rows, err := DB.Query(`
		SELECT id, tarea_id, agente, proyecto_id, rama, dir_trabajo, cmd_test,
		       estado, resultado, error_texto, commit_merge,
		       created_at, started_at, finished_at
		FROM refineria_solicitudes
		WHERE estado = 'pendiente'
		ORDER BY created_at, id
		LIMIT 10`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return escanearSolicitudes(rows)
}

// ProcesarRefineriaBatch toma las solicitudes pendientes, ejecuta los tests y
// aprueba o rechaza cada una. Devuelve el número de solicitudes procesadas.
func ProcesarRefineriaBatch() (int, error) {
	solicitudes, err := ListarRefineriasPendientes()
	if err != nil {
		return 0, err
	}
	procesadas := 0
	for _, s := range solicitudes {
		if err := procesarSolicitud(s); err != nil {
			Audit("server", "refineria_error", "refineria_solicitud", s.ID, err.Error())
			continue
		}
		procesadas++
	}
	return procesadas, nil
}

// procesarSolicitud ejecuta el pipeline de validación para una solicitud concreta.
func procesarSolicitud(s *RefineriaSolicitud) error {
	// Marcar como ejecutando
	if _, err := DB.Exec(`
		UPDATE refineria_solicitudes
		SET estado='ejecutando', started_at=CURRENT_TIMESTAMP
		WHERE id=? AND estado='pendiente'`, s.ID); err != nil {
		return err
	}

	salida, testErr := ejecutarTests(s.DirTrabajo, s.CmdTest)

	if testErr != nil {
		return rechazarRefineria(s, salida, testErr.Error())
	}
	return aprobarRefineria(s, salida)
}

// ejecutarTests corre el comando de tests en el directorio indicado con un timeout
// de 10 minutos. Devuelve la salida combinada (stdout+stderr) y el error si falla.
func ejecutarTests(dir, cmdTest string) (string, error) {
	partes := strings.Fields(cmdTest)
	if len(partes) == 0 {
		partes = []string{"go", "test", "./..."}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, partes[0], partes[1:]...) //nolint:gosec
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	return buf.String(), err
}

// aprobarRefineria marca la solicitud como aprobada, completa la tarea y avisa al agente.
func aprobarRefineria(s *RefineriaSolicitud, salida string) error {
	if _, err := DB.Exec(`
		UPDATE refineria_solicitudes
		SET estado='aprobada', resultado=?, finished_at=CURRENT_TIMESTAMP
		WHERE id=?`, salida, s.ID); err != nil {
		return err
	}

	// Completar la tarea; rama como pseudo-commit si no hay hash real.
	commit := s.Rama
	if strings.TrimSpace(commit) == "" {
		commit = fmt.Sprintf("refineria#%d", s.ID)
	}
	if err := CompletarTarea(s.TareaID, s.Agente, commit); err != nil {
		return fmt.Errorf("refinería aprobada pero CompletarTarea falló: %w", err)
	}

	Audit("server", "refineria_aprobada", "refineria_solicitud", s.ID,
		fmt.Sprintf("tarea #%d aprobada por refinería", s.TareaID))

	// Notificar al agente por el buzón.
	payload, _ := json.Marshal(map[string]any{
		"refineria_id": s.ID,
		"tarea_id":     s.TareaID,
		"resultado":    "aprobada",
		"rama":         s.Rama,
		"salida":       truncarSalida(salida),
	})
	_, _ = EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    s.Agente,
		ProyectoID:  s.ProyectoID,
		Kind:        "refineria_resultado",
		PayloadJSON: string(payload),
	})
	return nil
}

// rechazarRefineria marca la solicitud como rechazada y devuelve la tarea a en_progreso.
func rechazarRefineria(s *RefineriaSolicitud, salida, errorTexto string) error {
	if _, err := DB.Exec(`
		UPDATE refineria_solicitudes
		SET estado='rechazada', resultado=?, error_texto=?, finished_at=CURRENT_TIMESTAMP
		WHERE id=?`, salida, errorTexto, s.ID); err != nil {
		return err
	}

	Audit("server", "refineria_rechazada", "refineria_solicitud", s.ID,
		fmt.Sprintf("tarea #%d rechazada: %s", s.TareaID, errorTexto))

	// Notificar al agente con el error para que corrija.
	payload, _ := json.Marshal(map[string]any{
		"refineria_id": s.ID,
		"tarea_id":     s.TareaID,
		"resultado":    "rechazada",
		"rama":         s.Rama,
		"error":        errorTexto,
		"salida":       truncarSalida(salida),
	})
	_, _ = EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    s.Agente,
		ProyectoID:  s.ProyectoID,
		Kind:        "refineria_resultado",
		PayloadJSON: string(payload),
	})
	return nil
}

// CancelarRefineria cancela una solicitud pendiente (p.ej. si el agente abandona la tarea).
func CancelarRefineria(id int64, agente string) error {
	res, err := DB.Exec(`
		UPDATE refineria_solicitudes
		SET estado='cancelada', finished_at=CURRENT_TIMESTAMP
		WHERE id=? AND estado IN ('pendiente','ejecutando')`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("solicitud #%d no existe o ya no está activa", id)
	}
	Audit(agente, "cancelar_refineria", "refineria_solicitud", id, "")
	return nil
}

// GetRefineriaSolicitud devuelve una solicitud por ID.
func GetRefineriaSolicitud(id int64) (*RefineriaSolicitud, error) {
	row := DB.QueryRow(`
		SELECT id, tarea_id, agente, proyecto_id, rama, dir_trabajo, cmd_test,
		       estado, resultado, error_texto, commit_merge,
		       created_at, started_at, finished_at
		FROM refineria_solicitudes WHERE id=?`, id)
	ss, err := escanearSolicitudes(wrapRow(row))
	if err != nil {
		return nil, err
	}
	if len(ss) == 0 {
		return nil, sql.ErrNoRows
	}
	return ss[0], nil
}

// ListarRefineriaPorTarea devuelve el historial de solicitudes de una tarea.
func ListarRefineriaPorTarea(tareaID int64) ([]*RefineriaSolicitud, error) {
	rows, err := DB.Query(`
		SELECT id, tarea_id, agente, proyecto_id, rama, dir_trabajo, cmd_test,
		       estado, resultado, error_texto, commit_merge,
		       created_at, started_at, finished_at
		FROM refineria_solicitudes
		WHERE tarea_id=?
		ORDER BY id DESC`, tareaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return escanearSolicitudes(rows)
}

// ─── helpers ────────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

// rowWrapper adapta *sql.Row para que implemente rows.Next()/Close().
type rowWrapper struct {
	row *sql.Row
	ok  bool
}

func wrapRow(r *sql.Row) *rowWrapper { return &rowWrapper{row: r} }
func (w *rowWrapper) Next() bool {
	if !w.ok {
		w.ok = true
		return true
	}
	return false
}
func (w *rowWrapper) Close() error { return nil }
func (w *rowWrapper) Scan(dest ...any) error { return w.row.Scan(dest...) }

type solicitudRows interface {
	Next() bool
	Close() error
	Scan(dest ...any) error
}

func escanearSolicitudes(rows solicitudRows) ([]*RefineriaSolicitud, error) {
	defer rows.Close()
	var list []*RefineriaSolicitud
	for rows.Next() {
		s, err := escanearSolicitud(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, nil
}

func escanearSolicitud(s solicitudRows) (*RefineriaSolicitud, error) {
	var r RefineriaSolicitud
	var proyectoID sql.NullInt64
	var startedAt, finishedAt sql.NullTime
	err := s.Scan(
		&r.ID, &r.TareaID, &r.Agente, &proyectoID, &r.Rama, &r.DirTrabajo, &r.CmdTest,
		&r.Estado, &r.Resultado, &r.ErrorTexto, &r.CommitMerge,
		&r.CreatedAt, &startedAt, &finishedAt,
	)
	if err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		r.ProyectoID = &proyectoID.Int64
	}
	if startedAt.Valid {
		r.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		r.FinishedAt = &finishedAt.Time
	}
	return &r, nil
}

func truncarSalida(s string) string {
	const maxBytes = 4096
	if len(s) <= maxBytes {
		return s
	}
	return "…(truncado)\n" + s[len(s)-maxBytes:]
}
