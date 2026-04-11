/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

type FaseProyecto struct {
	ID          int64
	Proyecto    string
	Nombre      string
	Descripcion string
	Orden       int64
	Peso        float64
	Estado      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AvanceTarea struct {
	TareaID        int64
	Proyecto       string
	FaseID         *int64
	ProgresoPct    float64
	ActualizadoPor string
	UpdatedAt      time.Time
}

type TareaProgresoDetalle struct {
	Tarea       *Tarea
	FaseID      *int64
	FaseNombre  string
	Proyecto    string
	ProgresoPct float64
	Actualizado *time.Time
	Manual      bool
}

type FaseProgresoDetalle struct {
	Fase              *FaseProyecto
	ProgresoPct       float64
	Tareas            []*TareaProgresoDetalle
	TareasTotales     int
	TareasCompletadas int
}

type ResumenProgresoProyecto struct {
	Proyecto          string
	ProgresoPct       float64
	TareasTotales     int
	TareasCompletadas int
	Fases             []*FaseProgresoDetalle
	TareasSinFase     []*TareaProgresoDetalle
}

func ListarProyectosConProgreso() ([]string, error) {
	rows, err := DB.Query(`
		SELECT proyecto FROM (
			SELECT proyecto FROM fases_proyecto
			UNION
			SELECT proyecto FROM avance_tareas
			UNION
			SELECT proyecto FROM memoria_proyectos
			UNION
			SELECT p.slug AS proyecto
			FROM decisiones_proyecto dp
			JOIN proyectos p ON p.id = dp.proyecto_id
			UNION
			SELECT p.slug AS proyecto
			FROM documentos_externos de
			JOIN proyectos p ON p.id = de.proyecto_id
		)
		WHERE proyecto != ''
		ORDER BY proyecto`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []string
	for rows.Next() {
		var proyecto string
		if err := rows.Scan(&proyecto); err != nil {
			return nil, err
		}
		list = append(list, proyecto)
	}
	return list, rows.Err()
}

func RegistrarFaseProyecto(f *FaseProyecto) (int64, error) {
	if f == nil {
		return 0, fmt.Errorf("fase nula")
	}
	if f.Proyecto == "" || f.Nombre == "" {
		return 0, fmt.Errorf("proyecto y nombre son obligatorios")
	}
	if f.Orden == 0 {
		f.Orden = 100
	}
	if f.Peso <= 0 {
		f.Peso = 1
	}
	if f.Estado == "" {
		f.Estado = "pendiente"
	}
	res, err := DB.Exec(`
		INSERT INTO fases_proyecto (proyecto, nombre, descripcion, orden, peso, estado)
		VALUES (?,?,?,?,?,?)`,
		f.Proyecto, f.Nombre, f.Descripcion, f.Orden, f.Peso, f.Estado,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit("orquesta", "registrar_fase_proyecto", "fase_proyecto", id, f.Proyecto+":"+f.Nombre)
	return id, nil
}

func GetFaseProyecto(id int64) (*FaseProyecto, error) {
	row := DB.QueryRow(`
		SELECT id, proyecto, nombre, descripcion, orden, peso, estado, created_at, updated_at
		FROM fases_proyecto
		WHERE id = ?`, id)
	return escanearFaseProyecto(row)
}

func ListarFasesProyecto(proyecto string) ([]*FaseProyecto, error) {
	rows, err := DB.Query(`
		SELECT id, proyecto, nombre, descripcion, orden, peso, estado, created_at, updated_at
		FROM fases_proyecto
		WHERE proyecto = ?
		ORDER BY orden, id`, proyecto)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*FaseProyecto
	for rows.Next() {
		f, err := escanearFaseProyecto(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func ObtenerFaseActivaProyecto(proyecto string) (string, error) {
	var nombre string
	err := DB.QueryRow(`
		SELECT nombre FROM fases_proyecto
		WHERE proyecto = ? AND estado = 'activa'
		LIMIT 1`, strings.TrimSpace(proyecto)).Scan(&nombre)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return nombre, err
}

func ActualizarFaseProyecto(f *FaseProyecto) error {
	if f == nil {
		return fmt.Errorf("fase nula")
	}
	if f.ID <= 0 {
		return fmt.Errorf("id de fase obligatorio")
	}
	if f.Proyecto == "" || f.Nombre == "" {
		return fmt.Errorf("proyecto y nombre son obligatorios")
	}
	if f.Peso <= 0 {
		return fmt.Errorf("peso debe ser > 0")
	}
	res, err := DB.Exec(`
		UPDATE fases_proyecto
		SET proyecto=?, nombre=?, descripcion=?, orden=?, peso=?, estado=?
		WHERE id=?`,
		f.Proyecto, f.Nombre, f.Descripcion, f.Orden, f.Peso, f.Estado, f.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("fase #%d no encontrada", f.ID)
	}
	Audit("orquesta", "actualizar_fase_proyecto", "fase_proyecto", f.ID, f.Proyecto+":"+f.Nombre)
	return nil
}

func RegistrarAvanceTarea(a *AvanceTarea) error {
	if a == nil {
		return fmt.Errorf("avance nulo")
	}
	if a.TareaID <= 0 || a.Proyecto == "" {
		return fmt.Errorf("tarea_id y proyecto son obligatorios")
	}
	if a.ProgresoPct < 0 || a.ProgresoPct > 100 {
		return fmt.Errorf("progreso_pct debe estar entre 0 y 100")
	}
	if _, err := GetTarea(a.TareaID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("tarea #%d no encontrada", a.TareaID)
		}
		return err
	}
	if a.FaseID != nil {
		f, err := GetFaseProyecto(*a.FaseID)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("fase #%d no encontrada", *a.FaseID)
			}
			return err
		}
		if f.Proyecto != a.Proyecto {
			return fmt.Errorf("la fase #%d no pertenece al proyecto %s", *a.FaseID, a.Proyecto)
		}
	}
	_, err := DB.Exec(`
		INSERT INTO avance_tareas (tarea_id, proyecto, fase_id, progreso_pct, actualizado_por)
		VALUES (?,?,?,?,?)
		ON CONFLICT(tarea_id) DO UPDATE SET
			proyecto=excluded.proyecto,
			fase_id=excluded.fase_id,
			progreso_pct=excluded.progreso_pct,
			actualizado_por=excluded.actualizado_por,
			updated_at=CURRENT_TIMESTAMP`,
		a.TareaID, a.Proyecto, nullableInt64(a.FaseID), a.ProgresoPct, a.ActualizadoPor,
	)
	if err != nil {
		return err
	}
	Audit(a.ActualizadoPor, "registrar_avance_tarea", "tarea", a.TareaID, fmt.Sprintf("%s %.1f%%", a.Proyecto, a.ProgresoPct))
	return nil
}

func ListarAvanceTareasProyecto(proyecto string) ([]*TareaProgresoDetalle, error) {
	rows, err := DB.Query(`
		SELECT t.id, t.titulo, t.descripcion, t.modulo, t.estado, t.agente, t.propuesta_id, t.prioridad,
		       t.dependencias, t.creado_por, t.commit_cierre, t.notas, t.created_at, t.updated_at, t.completada_at,
		       at.proyecto, at.fase_id, COALESCE(fp.nombre,''), at.progreso_pct, at.actualizado_por, at.updated_at
		FROM avance_tareas at
		JOIN tareas t ON t.id = at.tarea_id
		LEFT JOIN fases_proyecto fp ON fp.id = at.fase_id
		WHERE at.proyecto = ?
		ORDER BY COALESCE(fp.orden, 9999), t.id`, proyecto)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*TareaProgresoDetalle
	for rows.Next() {
		item, err := escanearTareaProgresoDetalle(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func CalcularResumenProgresoProyecto(proyecto string) (*ResumenProgresoProyecto, error) {
	fases, err := ListarFasesProyecto(proyecto)
	if err != nil {
		return nil, err
	}
	tareas, err := ListarAvanceTareasProyecto(proyecto)
	if err != nil {
		return nil, err
	}
	resumen := &ResumenProgresoProyecto{Proyecto: proyecto}

	porFase := make(map[int64]*FaseProgresoDetalle)
	for _, fase := range fases {
		det := &FaseProgresoDetalle{Fase: fase}
		porFase[fase.ID] = det
		resumen.Fases = append(resumen.Fases, det)
	}

	var fasesConPeso float64
	var progresoPonderado float64
	for _, tarea := range tareas {
		resumen.TareasTotales++
		if tarea.ProgresoPct >= 100 {
			resumen.TareasCompletadas++
		}
		if tarea.FaseID != nil {
			if fase := porFase[*tarea.FaseID]; fase != nil {
				fase.Tareas = append(fase.Tareas, tarea)
				fase.TareasTotales++
				if tarea.ProgresoPct >= 100 {
					fase.TareasCompletadas++
				}
				continue
			}
		}
		resumen.TareasSinFase = append(resumen.TareasSinFase, tarea)
	}

	for _, fase := range resumen.Fases {
		if len(fase.Tareas) == 0 {
			fase.ProgresoPct = progresoEstadoFase(fase.Fase.Estado)
		} else {
			var sum float64
			for _, tarea := range fase.Tareas {
				sum += tarea.ProgresoPct
			}
			fase.ProgresoPct = redondearProgreso(sum / float64(len(fase.Tareas)))
		}
		fasesConPeso += fase.Fase.Peso
		progresoPonderado += fase.ProgresoPct * fase.Fase.Peso
	}

	if fasesConPeso > 0 {
		resumen.ProgresoPct = redondearProgreso(progresoPonderado / fasesConPeso)
		return resumen, nil
	}
	if len(tareas) > 0 {
		var sum float64
		for _, tarea := range tareas {
			sum += tarea.ProgresoPct
		}
		resumen.ProgresoPct = redondearProgreso(sum / float64(len(tareas)))
	}
	return resumen, nil
}

func progresoEstadoFase(estado string) float64 {
	switch estado {
	case "completada":
		return 100
	case "activa":
		return 50
	case "bloqueada":
		return 25
	default:
		return 0
	}
}

func progresoEstadoTarea(estado EstadoTarea) float64 {
	switch estado {
	case TareaCompletada:
		return 100
	case TareaEnProgreso:
		return 50
	case TareaAsignada:
		return 10
	case TareaBloqueada:
		return 25
	case TareaCancelada:
		return 100
	default:
		return 0
	}
}

func redondearProgreso(v float64) float64 {
	return math.Round(v*10) / 10
}

func escanearFaseProyecto(s scanner) (*FaseProyecto, error) {
	f := &FaseProyecto{}
	if err := s.Scan(&f.ID, &f.Proyecto, &f.Nombre, &f.Descripcion, &f.Orden, &f.Peso, &f.Estado, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, err
	}
	return f, nil
}

func escanearTareaProgresoDetalle(s scanner) (*TareaProgresoDetalle, error) {
	tarea := &Tarea{}
	var depsJSON string
	var agente sql.NullString
	var propuestaID sql.NullInt64
	var completadaAt sql.NullTime
	var proyecto string
	var faseID sql.NullInt64
	var faseNombre string
	var progreso sql.NullFloat64
	var actualizadoPor string
	var updatedAt sql.NullTime
	if err := s.Scan(
		&tarea.ID, &tarea.Titulo, &tarea.Descripcion, &tarea.Modulo, &tarea.Estado, &agente, &propuestaID,
		&tarea.Prioridad, &depsJSON, &tarea.CreadoPor, &tarea.CommitCierre, &tarea.Notas,
		&tarea.CreatedAt, &tarea.UpdatedAt, &completadaAt,
		&proyecto, &faseID, &faseNombre, &progreso, &actualizadoPor, &updatedAt,
	); err != nil {
		return nil, err
	}
	if agente.Valid {
		tarea.Agente = &agente.String
	}
	if propuestaID.Valid {
		tarea.PropuestaID = &propuestaID.Int64
	}
	_ = json.Unmarshal([]byte(depsJSON), &tarea.Dependencias)
	if completadaAt.Valid {
		tarea.CompletadaAt = &completadaAt.Time
	}
	item := &TareaProgresoDetalle{Tarea: tarea, ProgresoPct: progresoEstadoTarea(tarea.Estado)}
	item.Proyecto = proyecto
	item.FaseNombre = faseNombre
	if faseID.Valid {
		item.FaseID = &faseID.Int64
	}
	if progreso.Valid {
		item.ProgresoPct = progreso.Float64
		item.Manual = true
	}
	if updatedAt.Valid {
		item.Actualizado = &updatedAt.Time
	}
	return item, nil
}
