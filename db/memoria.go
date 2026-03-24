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
	"time"
)

type EstadoDeriva string

const (
	DerivaAbierta    EstadoDeriva = "abierta"
	DerivaEnRevision EstadoDeriva = "en_revision"
	DerivaResuelta   EstadoDeriva = "resuelta"
	DerivaDescartada EstadoDeriva = "descartada"
)

type MemoriaProyecto struct {
	ID                int64
	Proyecto          string
	Resumen           string
	Contexto          string
	PreguntasAbiertas string
	ActualizadoPor    string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type FuenteMemoria struct {
	ID            int64
	Proyecto      string
	Tipo          string
	Referencia    string
	Titulo        string
	URL           string
	Confianza     string
	Detalle       string
	RegistradoPor string
	CreatedAt     time.Time
}

type HallazgoMemoria struct {
	ID            int64
	Proyecto      string
	FuenteID      *int64
	Tipo          string
	Titulo        string
	Descripcion   string
	Impacto       string
	Confianza     string
	RegistradoPor string
	CreatedAt     time.Time
}

type DerivaMemoria struct {
	ID           int64
	Proyecto     string
	HallazgoID   *int64
	Tipo         string
	Severidad    string
	Estado       EstadoDeriva
	Descripcion  string
	Evidencia    string
	DetectadaPor string
	Resolucion   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ResueltaAt   *time.Time
}

func GuardarMemoriaProyecto(m *MemoriaProyecto) (int64, error) {
	if m == nil {
		return 0, fmt.Errorf("memoria nula")
	}
	if m.Proyecto == "" {
		return 0, fmt.Errorf("el proyecto es obligatorio")
	}
	_, err := DB.Exec(`
		INSERT INTO memoria_proyectos (proyecto, resumen, contexto, preguntas_abiertas, actualizado_por)
		VALUES (?,?,?,?,?)
		ON CONFLICT(proyecto) DO UPDATE SET
			resumen=excluded.resumen,
			contexto=excluded.contexto,
			preguntas_abiertas=excluded.preguntas_abiertas,
			actualizado_por=excluded.actualizado_por`,
		m.Proyecto, m.Resumen, m.Contexto, m.PreguntasAbiertas, m.ActualizadoPor,
	)
	if err != nil {
		return 0, err
	}
	var id int64
	if err := DB.QueryRow(`SELECT id FROM memoria_proyectos WHERE proyecto = ?`, m.Proyecto).Scan(&id); err != nil {
		return 0, err
	}
	Audit(m.ActualizadoPor, "guardar_memoria_proyecto", "memoria_proyecto", id, m.Proyecto)
	return id, nil
}

func GetMemoriaProyecto(proyecto string) (*MemoriaProyecto, error) {
	row := DB.QueryRow(`
		SELECT id, proyecto, resumen, contexto, preguntas_abiertas, actualizado_por, created_at, updated_at
		FROM memoria_proyectos
		WHERE proyecto = ?`, proyecto)
	return escanearMemoriaProyecto(row)
}

func RegistrarFuenteMemoria(f *FuenteMemoria) (int64, error) {
	if f == nil {
		return 0, fmt.Errorf("fuente nula")
	}
	if f.Proyecto == "" {
		return 0, fmt.Errorf("el proyecto es obligatorio")
	}
	if f.Referencia == "" {
		return 0, fmt.Errorf("la referencia es obligatoria")
	}
	res, err := DB.Exec(`
		INSERT INTO memoria_fuentes (proyecto, tipo, referencia, titulo, url, confianza, detalle, registrado_por)
		VALUES (?,?,?,?,?,?,?,?)`,
		f.Proyecto, f.Tipo, f.Referencia, f.Titulo, f.URL, f.Confianza, f.Detalle, f.RegistradoPor,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit(f.RegistradoPor, "registrar_fuente_memoria", "memoria_fuente", id, f.Proyecto+": "+f.Referencia)
	return id, nil
}

func ListarFuentesMemoria(proyecto string) ([]*FuenteMemoria, error) {
	rows, err := DB.Query(`
		SELECT id, proyecto, tipo, referencia, titulo, url, confianza, detalle, registrado_por, created_at
		FROM memoria_fuentes
		WHERE proyecto = ?
		ORDER BY id DESC`, proyecto)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*FuenteMemoria
	for rows.Next() {
		f, err := escanearFuenteMemoria(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func RegistrarHallazgoMemoria(h *HallazgoMemoria) (int64, error) {
	if h == nil {
		return 0, fmt.Errorf("hallazgo nulo")
	}
	if h.Proyecto == "" {
		return 0, fmt.Errorf("el proyecto es obligatorio")
	}
	if h.Titulo == "" {
		return 0, fmt.Errorf("el título es obligatorio")
	}
	res, err := DB.Exec(`
		INSERT INTO memoria_hallazgos (proyecto, fuente_id, tipo, titulo, descripcion, impacto, confianza, registrado_por)
		VALUES (?,?,?,?,?,?,?,?)`,
		h.Proyecto, h.FuenteID, h.Tipo, h.Titulo, h.Descripcion, h.Impacto, h.Confianza, h.RegistradoPor,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit(h.RegistradoPor, "registrar_hallazgo_memoria", "memoria_hallazgo", id, h.Proyecto+": "+h.Titulo)
	return id, nil
}

func ListarHallazgosMemoria(proyecto string) ([]*HallazgoMemoria, error) {
	rows, err := DB.Query(`
		SELECT id, proyecto, fuente_id, tipo, titulo, descripcion, impacto, confianza, registrado_por, created_at
		FROM memoria_hallazgos
		WHERE proyecto = ?
		ORDER BY id DESC`, proyecto)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*HallazgoMemoria
	for rows.Next() {
		h, err := escanearHallazgoMemoria(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

func RegistrarDerivaMemoria(d *DerivaMemoria) (int64, error) {
	if d == nil {
		return 0, fmt.Errorf("deriva nula")
	}
	if d.Proyecto == "" {
		return 0, fmt.Errorf("el proyecto es obligatorio")
	}
	if d.Descripcion == "" {
		return 0, fmt.Errorf("la descripción es obligatoria")
	}
	res, err := DB.Exec(`
		INSERT INTO memoria_derivas (proyecto, hallazgo_id, tipo, severidad, estado, descripcion, evidencia, detectada_por, resolucion, resuelta_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		d.Proyecto, d.HallazgoID, d.Tipo, d.Severidad, d.Estado, d.Descripcion, d.Evidencia, d.DetectadaPor, d.Resolucion, nullableTimePtr(d.ResueltaAt),
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit(d.DetectadaPor, "registrar_deriva_memoria", "memoria_deriva", id, d.Proyecto+": "+d.Descripcion)
	return id, nil
}

func ListarDerivasMemoria(proyecto string, soloAbiertas bool) ([]*DerivaMemoria, error) {
	q := `
		SELECT id, proyecto, hallazgo_id, tipo, severidad, estado, descripcion, evidencia, detectada_por, resolucion, created_at, updated_at, resuelta_at
		FROM memoria_derivas
		WHERE proyecto = ?`
	args := []any{proyecto}
	if soloAbiertas {
		q += ` AND estado IN ('abierta','en_revision')`
	}
	q += ` ORDER BY id DESC`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DerivaMemoria
	for rows.Next() {
		d, err := escanearDerivaMemoria(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func ResolverDerivaMemoria(id int64, agente, estado, resolucion string) error {
	if estado != string(DerivaEnRevision) && estado != string(DerivaResuelta) && estado != string(DerivaDescartada) {
		return fmt.Errorf("estado '%s' no válido para deriva", estado)
	}
	var resueltaAt any
	if estado == string(DerivaResuelta) || estado == string(DerivaDescartada) {
		resueltaAt = time.Now()
	}
	res, err := DB.Exec(`
		UPDATE memoria_derivas
		SET estado = ?, resolucion = ?, resuelta_at = ?
		WHERE id = ?`,
		estado, resolucion, resueltaAt, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("deriva #%d no encontrada", id)
	}
	Audit(agente, "resolver_deriva_memoria", "memoria_deriva", id, estado+": "+resolucion)
	return nil
}

func ListarMemoriaProyectos() ([]*MemoriaProyecto, error) {
	rows, err := DB.Query(`
		SELECT id, proyecto, resumen, contexto, preguntas_abiertas, actualizado_por, created_at, updated_at
		FROM memoria_proyectos
		ORDER BY proyecto`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*MemoriaProyecto
	for rows.Next() {
		m, err := escanearMemoriaProyecto(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func ContarMemoriaProyecto(proyecto string) (fuentes, hallazgos, derivas, derivasAbiertas int, err error) {
	if err = DB.QueryRow(`SELECT COUNT(*) FROM memoria_fuentes WHERE proyecto = ?`, proyecto).Scan(&fuentes); err != nil {
		return
	}
	if err = DB.QueryRow(`SELECT COUNT(*) FROM memoria_hallazgos WHERE proyecto = ?`, proyecto).Scan(&hallazgos); err != nil {
		return
	}
	if err = DB.QueryRow(`SELECT COUNT(*) FROM memoria_derivas WHERE proyecto = ?`, proyecto).Scan(&derivas); err != nil {
		return
	}
	err = DB.QueryRow(`SELECT COUNT(*) FROM memoria_derivas WHERE proyecto = ? AND estado IN ('abierta','en_revision')`, proyecto).Scan(&derivasAbiertas)
	return
}

func escanearMemoriaProyecto(s scanner) (*MemoriaProyecto, error) {
	m := &MemoriaProyecto{}
	if err := s.Scan(&m.ID, &m.Proyecto, &m.Resumen, &m.Contexto, &m.PreguntasAbiertas, &m.ActualizadoPor, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	return m, nil
}

func escanearFuenteMemoria(s scanner) (*FuenteMemoria, error) {
	f := &FuenteMemoria{}
	if err := s.Scan(&f.ID, &f.Proyecto, &f.Tipo, &f.Referencia, &f.Titulo, &f.URL, &f.Confianza, &f.Detalle, &f.RegistradoPor, &f.CreatedAt); err != nil {
		return nil, err
	}
	return f, nil
}

func escanearHallazgoMemoria(s scanner) (*HallazgoMemoria, error) {
	h := &HallazgoMemoria{}
	var fuenteID sql.NullInt64
	if err := s.Scan(&h.ID, &h.Proyecto, &fuenteID, &h.Tipo, &h.Titulo, &h.Descripcion, &h.Impacto, &h.Confianza, &h.RegistradoPor, &h.CreatedAt); err != nil {
		return nil, err
	}
	if fuenteID.Valid {
		h.FuenteID = &fuenteID.Int64
	}
	return h, nil
}

func escanearDerivaMemoria(s scanner) (*DerivaMemoria, error) {
	d := &DerivaMemoria{}
	var hallazgoID sql.NullInt64
	var resueltaAt sql.NullTime
	if err := s.Scan(&d.ID, &d.Proyecto, &hallazgoID, &d.Tipo, &d.Severidad, &d.Estado, &d.Descripcion, &d.Evidencia, &d.DetectadaPor, &d.Resolucion, &d.CreatedAt, &d.UpdatedAt, &resueltaAt); err != nil {
		return nil, err
	}
	if hallazgoID.Valid {
		d.HallazgoID = &hallazgoID.Int64
	}
	if resueltaAt.Valid {
		d.ResueltaAt = &resueltaAt.Time
	}
	return d, nil
}

func nullableTimePtr(v *time.Time) any {
	if v == nil {
		return nil
	}
	return *v
}
