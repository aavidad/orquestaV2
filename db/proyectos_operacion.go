package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type EstadoOperativoProyecto string

const (
	ProyectoOperativoActivo           EstadoOperativoProyecto = "activo"
	ProyectoOperativoEsperandoHumano  EstadoOperativoProyecto = "esperando_humano"
	ProyectoOperativoBloqueadoExterno EstadoOperativoProyecto = "bloqueado_externo"
	ProyectoOperativoCerrado          EstadoOperativoProyecto = "cerrado"
)

type ProyectoOperacion struct {
	ProyectoID       int64
	EstadoOperativo  EstadoOperativoProyecto
	Motivo           string
	ObjetivoPct      int
	MinAgentes       int
	MaxAgentes       int
	Prioridad        int
	ResumeAutomatico bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func GetProyectoOperacion(proyectoID int64) (*ProyectoOperacion, error) {
	exists, err := SchemaObjectExists("table", "proyectos_operacion")
	if err != nil {
		return nil, err
	}
	if !exists {
		return proyectoOperacionDefault(proyectoID), nil
	}
	var (
		op     ProyectoOperacion
		resume int
	)
	err = DB.QueryRow(`
		SELECT proyecto_id, estado_operativo, motivo, objetivo_pct, min_agentes, max_agentes,
		       prioridad, resume_automatico, created_at, updated_at
		FROM proyectos_operacion
		WHERE proyecto_id = ?`, proyectoID,
	).Scan(
		&op.ProyectoID, &op.EstadoOperativo, &op.Motivo, &op.ObjetivoPct, &op.MinAgentes,
		&op.MaxAgentes, &op.Prioridad, &resume, &op.CreatedAt, &op.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return proyectoOperacionDefault(proyectoID), nil
	}
	if err != nil {
		return nil, err
	}
	op.ResumeAutomatico = resume == 1
	normalizarProyectoOperacion(&op)
	return &op, nil
}

func UpsertProyectoOperacion(op *ProyectoOperacion) error {
	if err := ensureProyectoOperacionSchema(); err != nil {
		return err
	}
	if op == nil {
		return fmt.Errorf("proyecto_operacion nil")
	}
	if op.ProyectoID == 0 {
		return fmt.Errorf("proyecto_id obligatorio")
	}
	normalizarProyectoOperacion(op)
	resume := 0
	if op.ResumeAutomatico {
		resume = 1
	}
	_, err := DB.Exec(`
		INSERT INTO proyectos_operacion (
			proyecto_id, estado_operativo, motivo, objetivo_pct, min_agentes, max_agentes, prioridad, resume_automatico
		) VALUES (?,?,?,?,?,?,?,?)
		ON CONFLICT(proyecto_id) DO UPDATE SET
			estado_operativo=excluded.estado_operativo,
			motivo=excluded.motivo,
			objetivo_pct=excluded.objetivo_pct,
			min_agentes=excluded.min_agentes,
			max_agentes=excluded.max_agentes,
			prioridad=excluded.prioridad,
			resume_automatico=excluded.resume_automatico,
			updated_at=CURRENT_TIMESTAMP`,
		op.ProyectoID, op.EstadoOperativo, strings.TrimSpace(op.Motivo), op.ObjetivoPct,
		op.MinAgentes, op.MaxAgentes, op.Prioridad, resume,
	)
	return err
}

func MarcarProyectoEsperandoHumano(proyectoID int64, motivo string) error {
	op, err := GetProyectoOperacion(proyectoID)
	if err != nil {
		return err
	}
	op.EstadoOperativo = ProyectoOperativoEsperandoHumano
	op.Motivo = strings.TrimSpace(motivo)
	return UpsertProyectoOperacion(op)
}

func MarcarProyectoActivo(proyectoID int64, motivo string) error {
	op, err := GetProyectoOperacion(proyectoID)
	if err != nil {
		return err
	}
	op.EstadoOperativo = ProyectoOperativoActivo
	op.Motivo = strings.TrimSpace(motivo)
	return UpsertProyectoOperacion(op)
}

func MarcarProyectoBloqueadoExterno(proyectoID int64, motivo string) error {
	op, err := GetProyectoOperacion(proyectoID)
	if err != nil {
		return err
	}
	op.EstadoOperativo = ProyectoOperativoBloqueadoExterno
	op.Motivo = strings.TrimSpace(motivo)
	return UpsertProyectoOperacion(op)
}

func MarcarProyectoCerrado(proyectoID int64, motivo string) error {
	op, err := GetProyectoOperacion(proyectoID)
	if err != nil {
		return err
	}
	op.EstadoOperativo = ProyectoOperativoCerrado
	op.Motivo = strings.TrimSpace(motivo)
	return UpsertProyectoOperacion(op)
}

func ensureProyectoOperacionSchema() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS proyectos_operacion (
			proyecto_id       INTEGER PRIMARY KEY REFERENCES proyectos(id) ON DELETE CASCADE,
			estado_operativo  TEXT    NOT NULL DEFAULT 'activo'
			                         CHECK (estado_operativo IN ('activo','esperando_humano','bloqueado_externo','cerrado')),
			motivo            TEXT    NOT NULL DEFAULT '',
			objetivo_pct      INTEGER NOT NULL DEFAULT 100,
			min_agentes       INTEGER NOT NULL DEFAULT 0,
			max_agentes       INTEGER NOT NULL DEFAULT 0,
			prioridad         INTEGER NOT NULL DEFAULT 100,
			resume_automatico INTEGER NOT NULL DEFAULT 1,
			created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	return err
}

func ProyectoDisponibleParaAutonomia(proyectoID int64) (bool, error) {
	op, err := GetProyectoOperacion(proyectoID)
	if err != nil {
		return false, err
	}
	switch op.EstadoOperativo {
	case ProyectoOperativoActivo:
		return true, nil
	case ProyectoOperativoEsperandoHumano, ProyectoOperativoBloqueadoExterno:
		if op.EstadoOperativo == ProyectoOperativoBloqueadoExterno && strings.HasPrefix(strings.TrimSpace(op.Motivo), "conector:") {
			disponible, err := bloqueoExternoConectorResuelto(strings.TrimSpace(op.Motivo))
			if err != nil {
				return false, err
			}
			if !disponible {
				return false, nil
			}
			if !op.ResumeAutomatico {
				return false, nil
			}
			op.EstadoOperativo = ProyectoOperativoActivo
			op.Motivo = ""
			if err := UpsertProyectoOperacion(op); err != nil {
				return false, err
			}
			Audit("orquesta", "proyecto_reactivado_automaticamente", "proyecto", proyectoID, "conector operativo de nuevo")
			return true, nil
		}
		bloqueado, motivo, err := ResolverBloqueoProyecto(proyectoID)
		if err != nil {
			return false, err
		}
		if bloqueado {
			if strings.TrimSpace(motivo) != "" && strings.TrimSpace(op.Motivo) != strings.TrimSpace(motivo) {
				op.Motivo = strings.TrimSpace(motivo)
				if err := UpsertProyectoOperacion(op); err != nil {
					return false, err
				}
			}
			return false, nil
		}
		if !op.ResumeAutomatico {
			return false, nil
		}
		op.EstadoOperativo = ProyectoOperativoActivo
		op.Motivo = ""
		if err := UpsertProyectoOperacion(op); err != nil {
			return false, err
		}
		Audit("orquesta", "proyecto_reactivado_automaticamente", "proyecto", proyectoID, "desbloqueo detectado")
		return true, nil
	case ProyectoOperativoCerrado:
		return false, nil
	default:
		return true, nil
	}
}

func bloqueoProyectoRequiereIntervencionHumana(motivo string) bool {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return true
	}
	switch {
	case strings.HasPrefix(motivo, "Agente "):
		return false
	case strings.HasPrefix(motivo, "Agente degradado:"):
		return false
	case strings.HasPrefix(motivo, "Sobrecarga operativa:"):
		return false
	default:
		return true
	}
}

func ResolverBloqueoProyecto(proyectoID int64) (bool, string, error) {
	if proyectoID == 0 {
		return false, "", nil
	}
	var activas, bloqueadas int
	if err := DB.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN estado IN ('asignada','en_progreso') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN estado = 'bloqueada' THEN 1 ELSE 0 END), 0)
		FROM tareas
		WHERE proyecto_id = ?`, proyectoID,
	).Scan(&activas, &bloqueadas); err != nil {
		return false, "", err
	}
	if activas > 0 || bloqueadas == 0 {
		return false, "", nil
	}
	rows, err := DB.Query(`
		SELECT b.motivo
		FROM bloqueos b
		JOIN tareas t ON t.id = b.tarea_id
		WHERE t.proyecto_id = ?
		  AND t.estado = 'bloqueada'
		  AND b.resuelto = 0
		ORDER BY b.id DESC`, proyectoID)
	if err != nil {
		return false, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var motivo sql.NullString
		if err := rows.Scan(&motivo); err != nil {
			return false, "", err
		}
		if !motivo.Valid || strings.TrimSpace(motivo.String) == "" {
			return true, "esperando_desbloqueo_humano", nil
		}
		resuelto := strings.TrimSpace(motivo.String)
		if bloqueoProyectoRequiereIntervencionHumana(resuelto) {
			return true, resuelto, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, "", err
	}
	return false, "", nil
}

func proyectoOperacionDefault(proyectoID int64) *ProyectoOperacion {
	return &ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  ProyectoOperativoActivo,
		ObjetivoPct:      100,
		Prioridad:        100,
		ResumeAutomatico: true,
	}
}

func normalizarProyectoOperacion(op *ProyectoOperacion) {
	if op == nil {
		return
	}
	switch op.EstadoOperativo {
	case ProyectoOperativoActivo, ProyectoOperativoEsperandoHumano, ProyectoOperativoBloqueadoExterno, ProyectoOperativoCerrado:
	default:
		op.EstadoOperativo = ProyectoOperativoActivo
	}
	if op.ObjetivoPct <= 0 {
		op.ObjetivoPct = 100
	}
	if op.Prioridad <= 0 {
		op.Prioridad = 100
	}
	if op.MinAgentes < 0 {
		op.MinAgentes = 0
	}
	if op.MaxAgentes < 0 {
		op.MaxAgentes = 0
	}
}

func bloqueoExternoConectorResuelto(motivo string) (bool, error) {
	motivo = strings.TrimSpace(motivo)
	if !strings.HasPrefix(motivo, "conector:") {
		return false, nil
	}
	parts := strings.Split(motivo, ":")
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		return false, nil
	}
	conector, err := GetConector(strings.TrimSpace(parts[1]))
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	disponible, _, err := ConectorDisponibleParaArranque(conector.ID)
	return disponible, err
}
