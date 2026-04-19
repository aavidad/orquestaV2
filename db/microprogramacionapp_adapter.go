package db

import (
	"database/sql"
	"encoding/json"
	"strings"

	"orquesta/microprogramacionapp"
)

type MicroprogramacionRepository struct{}

func (MicroprogramacionRepository) CrearEspecificacionFuncion(spec *microprogramacionapp.EspecificacionFuncion) (int64, error) {
	precondicionesJSON, err := json.Marshal(spec.Precondiciones)
	if err != nil {
		return 0, err
	}
	postcondicionesJSON, err := json.Marshal(spec.Postcondiciones)
	if err != nil {
		return 0, err
	}
	depsPermitidasJSON, err := json.Marshal(spec.DependenciasPermitidas)
	if err != nil {
		return 0, err
	}
	depsProhibidasJSON, err := json.Marshal(spec.DependenciasProhibidas)
	if err != nil {
		return 0, err
	}
	testsJSON, err := json.Marshal(spec.TestsObligatorios)
	if err != nil {
		return 0, err
	}
	writeSetJSON, err := json.Marshal(spec.WriteSet)
	if err != nil {
		return 0, err
	}
	res, err := DB.Exec(`
		INSERT INTO especificaciones_funcion (
			tarea_id, proyecto_id, titulo, archivo_objetivo, simbolo_objetivo, descripcion,
			precondiciones_json, postcondiciones_json, dependencias_permitidas_json, dependencias_prohibidas_json,
			tests_obligatorios_json, write_set_json, formato_salida, estado, version, creado_por
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nullInt64Micro(spec.TareaID),
		nullInt64Micro(spec.ProyectoID),
		strings.TrimSpace(spec.Titulo),
		strings.TrimSpace(spec.ArchivoObjetivo),
		strings.TrimSpace(spec.SimboloObjetivo),
		strings.TrimSpace(spec.Descripcion),
		string(precondicionesJSON),
		string(postcondicionesJSON),
		string(depsPermitidasJSON),
		string(depsProhibidasJSON),
		string(testsJSON),
		string(writeSetJSON),
		strings.TrimSpace(spec.FormatoSalida),
		string(spec.Estado),
		spec.Version,
		strings.TrimSpace(spec.CreadoPor),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (MicroprogramacionRepository) ObtenerEspecificacionFuncion(id int64) (*microprogramacionapp.EspecificacionFuncion, error) {
	row := DB.QueryRow(`
		SELECT id, tarea_id, proyecto_id, titulo, archivo_objetivo, simbolo_objetivo, descripcion,
		       precondiciones_json, postcondiciones_json, dependencias_permitidas_json, dependencias_prohibidas_json,
		       tests_obligatorios_json, write_set_json, formato_salida, estado, version, creado_por, created_at, updated_at
		FROM especificaciones_funcion
		WHERE id = ?`, id)
	return scanEspecificacionFuncion(row)
}

func (MicroprogramacionRepository) ListarEspecificacionesFuncion(filtro microprogramacionapp.FiltroEspecificaciones) ([]*microprogramacionapp.EspecificacionFuncion, error) {
	query := `
		SELECT id, tarea_id, proyecto_id, titulo, archivo_objetivo, simbolo_objetivo, descripcion,
		       precondiciones_json, postcondiciones_json, dependencias_permitidas_json, dependencias_prohibidas_json,
		       tests_obligatorios_json, write_set_json, formato_salida, estado, version, creado_por, created_at, updated_at
		FROM especificaciones_funcion
		WHERE 1=1`
	args := make([]any, 0, 3)
	if filtro.TareaID != nil {
		query += ` AND tarea_id = ?`
		args = append(args, *filtro.TareaID)
	}
	if filtro.ProyectoID != nil {
		query += ` AND proyecto_id = ?`
		args = append(args, *filtro.ProyectoID)
	}
	if filtro.Estado != nil {
		query += ` AND estado = ?`
		args = append(args, string(*filtro.Estado))
	}
	query += ` ORDER BY id DESC`
	if filtro.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filtro.Limit)
	}
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var resultado []*microprogramacionapp.EspecificacionFuncion
	for rows.Next() {
		item, err := scanEspecificacionFuncion(rows)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, item)
	}
	return resultado, rows.Err()
}

func scanEspecificacionFuncion(scanner interface{ Scan(dest ...any) error }) (*microprogramacionapp.EspecificacionFuncion, error) {
	var (
		item                microprogramacionapp.EspecificacionFuncion
		tareaID             sql.NullInt64
		proyectoID          sql.NullInt64
		precondicionesJSON  string
		postcondicionesJSON string
		depsPermitidasJSON  string
		depsProhibidasJSON  string
		testsJSON           string
		writeSetJSON        string
		estado              string
	)
	if err := scanner.Scan(
		&item.ID,
		&tareaID,
		&proyectoID,
		&item.Titulo,
		&item.ArchivoObjetivo,
		&item.SimboloObjetivo,
		&item.Descripcion,
		&precondicionesJSON,
		&postcondicionesJSON,
		&depsPermitidasJSON,
		&depsProhibidasJSON,
		&testsJSON,
		&writeSetJSON,
		&item.FormatoSalida,
		&estado,
		&item.Version,
		&item.CreadoPor,
		&item.CreadaAt,
		&item.ActualizadaAt,
	); err != nil {
		return nil, err
	}
	if tareaID.Valid {
		item.TareaID = &tareaID.Int64
	}
	if proyectoID.Valid {
		item.ProyectoID = &proyectoID.Int64
	}
	item.Estado = microprogramacionapp.EstadoEspecificacion(strings.TrimSpace(estado))
	if err := json.Unmarshal([]byte(precondicionesJSON), &item.Precondiciones); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(postcondicionesJSON), &item.Postcondiciones); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(depsPermitidasJSON), &item.DependenciasPermitidas); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(depsProhibidasJSON), &item.DependenciasProhibidas); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(testsJSON), &item.TestsObligatorios); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(writeSetJSON), &item.WriteSet); err != nil {
		return nil, err
	}
	return &item, nil
}

func nullInt64Micro(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

type SqliteMicroprogramacionRepo = MicroprogramacionRepository
