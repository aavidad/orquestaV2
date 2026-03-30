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
	"path/filepath"
	"strings"
	"time"
)

type FusionProyectosOptions struct {
	ArchivarOrigen bool `json:"archivar_origen"`
}

type FusionProyectosResultado struct {
	OrigenID      int64            `json:"origen_id"`
	OrigenSlug    string           `json:"origen_slug"`
	DestinoID     int64            `json:"destino_id"`
	DestinoSlug   string           `json:"destino_slug"`
	SlugArchivado string           `json:"slug_archivado,omitempty"`
	RutaArchivada string           `json:"ruta_archivada,omitempty"`
	Actualizadas  map[string]int64 `json:"actualizadas,omitempty"`
	Conflictos    map[string]int64 `json:"conflictos,omitempty"`
}

func (r *FusionProyectosResultado) TotalActualizaciones() int64 {
	if r == nil {
		return 0
	}
	var total int64
	for _, n := range r.Actualizadas {
		total += n
	}
	return total
}

func FusionarProyectos(origenRef, destinoRef string, opts FusionProyectosOptions) (*FusionProyectosResultado, error) {
	origenRef = strings.TrimSpace(origenRef)
	destinoRef = strings.TrimSpace(destinoRef)
	switch {
	case origenRef == "":
		return nil, fmt.Errorf("debes indicar el proyecto origen")
	case destinoRef == "":
		return nil, fmt.Errorf("debes indicar el proyecto destino")
	}

	origen, err := GetProyecto(origenRef)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("proyecto origen '%s' no encontrado", origenRef)
		}
		return nil, err
	}
	destino, err := GetProyecto(destinoRef)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("proyecto destino '%s' no encontrado", destinoRef)
		}
		return nil, err
	}
	if origen.ID == destino.ID {
		return nil, fmt.Errorf("el proyecto origen y destino no pueden ser el mismo")
	}
	if !opts.ArchivarOrigen {
		return nil, fmt.Errorf("fusionar proyectos sin archivar el origen todavía no está soportado")
	}

	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	resultado := &FusionProyectosResultado{
		OrigenID:     origen.ID,
		OrigenSlug:   strings.TrimSpace(origen.Slug),
		DestinoID:    destino.ID,
		DestinoSlug:  strings.TrimSpace(destino.Slug),
		Actualizadas: map[string]int64{},
		Conflictos:   map[string]int64{},
	}

	archivedSlug, archivedPath, err := prepararArchivoProyectoTx(tx, origen, destino)
	if err != nil {
		return nil, err
	}
	resultado.SlugArchivado = archivedSlug
	resultado.RutaArchivada = archivedPath

	simples := []struct {
		nombre        string
		table         string
		column        string
		query         string
		projectBySlug bool
	}{
		{nombre: "propuestas.proyecto_id", table: "propuestas", column: "proyecto_id", query: `UPDATE propuestas SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "sesiones.proyecto_id", table: "sesiones", column: "proyecto_id", query: `UPDATE sesiones SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "locks.proyecto_id", table: "locks", column: "proyecto_id", query: `UPDATE locks SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "worktrees.proyecto_id", table: "worktrees", column: "proyecto_id", query: `UPDATE worktrees SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "review_gates.proyecto_id", table: "review_gates", column: "proyecto_id", query: `UPDATE review_gates SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "runtime_instances.proyecto_id", table: "runtime_instances", column: "proyecto_id", query: `UPDATE runtime_instances SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "runtime_transcript.proyecto_id", table: "runtime_transcript", column: "proyecto_id", query: `UPDATE runtime_transcript SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "runtime_handles.proyecto_id", table: "runtime_handles", column: "proyecto_id", query: `UPDATE runtime_handles SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "runtime_orders.proyecto_id", table: "runtime_orders", column: "proyecto_id", query: `UPDATE runtime_orders SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "runtime_mailbox.proyecto_id", table: "runtime_mailbox", column: "proyecto_id", query: `UPDATE runtime_mailbox SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "runtime_checkpoints.proyecto_id", table: "runtime_checkpoints", column: "proyecto_id", query: `UPDATE runtime_checkpoints SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "autonomia_ciclos.proyecto_id", table: "autonomia_ciclos", column: "proyecto_id", query: `UPDATE autonomia_ciclos SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "git_merges.proyecto_id", table: "git_merges", column: "proyecto_id", query: `UPDATE git_merges SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "refineria_solicitudes.proyecto_id", table: "refineria_solicitudes", column: "proyecto_id", query: `UPDATE refineria_solicitudes SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "asignaciones.proyecto_id", table: "asignaciones", column: "proyecto_id", query: `UPDATE asignaciones SET proyecto_id = ? WHERE proyecto_id = ?`},
		{nombre: "avance_tareas.proyecto", table: "avance_tareas", column: "proyecto", query: `UPDATE avance_tareas SET proyecto = ? WHERE proyecto = ?`, projectBySlug: true},
		{nombre: "memoria_fuentes.proyecto", table: "memoria_fuentes", column: "proyecto", query: `UPDATE memoria_fuentes SET proyecto = ? WHERE proyecto = ?`, projectBySlug: true},
		{nombre: "memoria_hallazgos.proyecto", table: "memoria_hallazgos", column: "proyecto", query: `UPDATE memoria_hallazgos SET proyecto = ? WHERE proyecto = ?`, projectBySlug: true},
		{nombre: "memoria_derivas.proyecto", table: "memoria_derivas", column: "proyecto", query: `UPDATE memoria_derivas SET proyecto = ? WHERE proyecto = ?`, projectBySlug: true},
	}
	for _, op := range simples {
		ok, err := tablaConColumnasMerge(tx, op.table, op.column)
		if err != nil {
			return nil, fmt.Errorf("inspeccionando %s: %w", op.nombre, err)
		}
		if !ok {
			continue
		}
		args := []any{destino.ID, origen.ID}
		if op.projectBySlug {
			args = []any{destino.Slug, origen.Slug}
		}
		n, err := execRowsAfectadas(tx, op.query, args...)
		if err != nil {
			return nil, fmt.Errorf("actualizando %s: %w", op.nombre, err)
		}
		if n > 0 {
			resultado.Actualizadas[op.nombre] = n
		}
	}

	if ok, err := tablaConColumnasMerge(tx, "tareas", "proyecto_id", "blueprint_key"); err != nil {
		return nil, fmt.Errorf("inspeccionando tareas con blueprint_key: %w", err)
	} else if ok {
		if moved, conflicted, err := moverTareasProyectoTx(tx, origen.ID, destino.ID); err != nil {
			return nil, fmt.Errorf("fusionando tareas por blueprint_key: %w", err)
		} else {
			anotarResultadoFusion(resultado, "tareas.proyecto_id", moved, conflicted)
		}
	} else if ok, err := tablaConColumnasMerge(tx, "tareas", "proyecto_id"); err != nil {
		return nil, fmt.Errorf("inspeccionando tareas legacy: %w", err)
	} else if ok {
		if n, err := execRowsAfectadas(tx, `UPDATE tareas SET proyecto_id = ? WHERE proyecto_id = ?`, destino.ID, origen.ID); err != nil {
			return nil, fmt.Errorf("actualizando tareas.proyecto_id: %w", err)
		} else if n > 0 {
			resultado.Actualizadas["tareas.proyecto_id"] = n
		}
	}

	if ok, err := tablaConColumnasMerge(tx, "decisiones_proyecto", "proyecto_id", "titulo"); err != nil {
		return nil, fmt.Errorf("inspeccionando decisiones_proyecto: %w", err)
	} else if ok {
		if moved, conflicted, err := moverRowsProyectoUniqueKeyTx(tx, "decisiones_proyecto", "titulo", origen.ID, destino.ID); err != nil {
			return nil, fmt.Errorf("fusionando decisiones_proyecto: %w", err)
		} else {
			anotarResultadoFusion(resultado, "decisiones_proyecto.proyecto_id", moved, conflicted)
		}
	} else if ok, err := tablaConColumnasMerge(tx, "decisiones_proyecto", "proyecto", "titulo"); err != nil {
		return nil, fmt.Errorf("inspeccionando decisiones_proyecto legacy: %w", err)
	} else if ok {
		if moved, conflicted, err := moverRowsProyectoStringUniqueKeyTx(tx, "decisiones_proyecto", "titulo", origen.Slug, destino.Slug, archivedSlug); err != nil {
			return nil, fmt.Errorf("fusionando decisiones_proyecto legacy: %w", err)
		} else {
			anotarResultadoFusion(resultado, "decisiones_proyecto.proyecto", moved, conflicted)
		}
	}
	if ok, err := tablaConColumnasMerge(tx, "documentos_externos", "proyecto_id", "ruta_ref"); err != nil {
		return nil, fmt.Errorf("inspeccionando documentos_externos: %w", err)
	} else if ok {
		if moved, conflicted, err := moverRowsProyectoUniqueKeyTx(tx, "documentos_externos", "ruta_ref", origen.ID, destino.ID); err != nil {
			return nil, fmt.Errorf("fusionando documentos_externos: %w", err)
		} else {
			anotarResultadoFusion(resultado, "documentos_externos.proyecto_id", moved, conflicted)
		}
	}
	if ok, err := tablaConColumnasMerge(tx, "entidades_memoria", "proyecto_id", "nombre"); err != nil {
		return nil, fmt.Errorf("inspeccionando entidades_memoria: %w", err)
	} else if ok {
		if moved, conflicted, err := moverRowsProyectoUniqueKeyTx(tx, "entidades_memoria", "nombre", origen.ID, destino.ID); err != nil {
			return nil, fmt.Errorf("fusionando entidades_memoria: %w", err)
		} else {
			anotarResultadoFusion(resultado, "entidades_memoria.proyecto_id", moved, conflicted)
		}
	}
	if ok, err := tablaConColumnasMerge(tx, "fases_proyecto", "proyecto", "nombre"); err != nil {
		return nil, fmt.Errorf("inspeccionando fases_proyecto: %w", err)
	} else if ok {
		if moved, conflicted, err := moverRowsProyectoStringUniqueKeyTx(tx, "fases_proyecto", "nombre", origen.Slug, destino.Slug, archivedSlug); err != nil {
			return nil, fmt.Errorf("fusionando fases_proyecto: %w", err)
		} else {
			anotarResultadoFusion(resultado, "fases_proyecto.proyecto", moved, conflicted)
		}
	}
	if ok, err := tablaConColumnasMerge(tx, "memoria_proyectos", "proyecto"); err != nil {
		return nil, fmt.Errorf("inspeccionando memoria_proyectos: %w", err)
	} else if ok {
		if moved, conflicted, err := fusionarMemoriaProyectoTx(tx, origen.Slug, destino.Slug, archivedSlug); err != nil {
			return nil, fmt.Errorf("fusionando memoria_proyectos: %w", err)
		} else {
			anotarResultadoFusion(resultado, "memoria_proyectos.proyecto", moved, conflicted)
		}
	}

	if ok, err := tablaConColumnasMerge(tx, "proyectos_operacion", "proyecto_id"); err != nil {
		return nil, fmt.Errorf("inspeccionando proyectos_operacion: %w", err)
	} else if ok {
		if moved, err := moverSingletonProyectoTx(tx, "proyectos_operacion", origen.ID, destino.ID); err != nil {
			return nil, fmt.Errorf("fusionando proyectos_operacion: %w", err)
		} else if moved > 0 {
			resultado.Actualizadas["proyectos_operacion.proyecto_id"] = moved
		}
	}
	if ok, err := tablaConColumnasMerge(tx, "proyectos_autonomia", "proyecto_id"); err != nil {
		return nil, fmt.Errorf("inspeccionando proyectos_autonomia: %w", err)
	} else if ok {
		if moved, err := moverSingletonProyectoTx(tx, "proyectos_autonomia", origen.ID, destino.ID); err != nil {
			return nil, fmt.Errorf("fusionando proyectos_autonomia: %w", err)
		} else if moved > 0 {
			resultado.Actualizadas["proyectos_autonomia.proyecto_id"] = moved
		}
	}

	if ok, err := tablaConColumnasMerge(tx, "asignaciones", "proyecto_id", "estado", "agente"); err != nil {
		return nil, fmt.Errorf("inspeccionando asignaciones: %w", err)
	} else if ok {
		if n, err := normalizarAsignacionesProyectoTx(tx, destino.ID, origen.Slug, destino.Slug); err != nil {
			return nil, fmt.Errorf("normalizando asignaciones: %w", err)
		} else if n > 0 {
			resultado.Actualizadas["asignaciones.normalizadas"] = n
		}
	}

	if _, err := tx.Exec(`
		UPDATE proyectos
		SET activo = 1
		WHERE id = ?`, destino.ID); err != nil {
		return nil, fmt.Errorf("activar proyecto destino: %w", err)
	}
	if _, err := tx.Exec(`
		UPDATE proyectos
		SET slug = ?, nombre = ?, ruta_abs = ?, activo = 0
		WHERE id = ?`,
		archivedSlug,
		fmt.Sprintf("%s [archivado en favor de %s]", strings.TrimSpace(origen.Nombre), strings.TrimSpace(destino.Slug)),
		archivedPath,
		origen.ID,
	); err != nil {
		return nil, fmt.Errorf("archivar proyecto origen: %w", err)
	}
	if ok, err := tablaConColumnasMerge(tx, "proyectos_operacion", "proyecto_id", "estado_operativo", "motivo", "resume_automatico"); err != nil {
		return nil, fmt.Errorf("inspeccionando cierre de proyectos_operacion: %w", err)
	} else if ok {
		if _, err := tx.Exec(`
			UPDATE proyectos_operacion
			SET estado_operativo = 'cerrado',
			    motivo = ?,
			    resume_automatico = 0
			WHERE proyecto_id = ?`,
			fmt.Sprintf("archivado tras fusion con %s", strings.TrimSpace(destino.Slug)),
			origen.ID,
		); err != nil {
			return nil, fmt.Errorf("cerrar operacion del proyecto origen: %w", err)
		}
	}
	if ok, err := tablaConColumnasMerge(tx, "proyectos_autonomia", "proyecto_id", "enabled", "estado_autonomia"); err != nil {
		return nil, fmt.Errorf("inspeccionando cierre de proyectos_autonomia: %w", err)
	} else if ok {
		if _, err := tx.Exec(`
			UPDATE proyectos_autonomia
			SET enabled = 0,
			    estado_autonomia = 'cerrado'
			WHERE proyecto_id = ?`, origen.ID); err != nil {
			return nil, fmt.Errorf("cerrar autonomia del proyecto origen: %w", err)
		}
	}

	detalle := fmt.Sprintf(
		"%s(%d)->%s(%d) actualizaciones=%d conflictos=%d",
		resultado.OrigenSlug, resultado.OrigenID,
		resultado.DestinoSlug, resultado.DestinoID,
		resultado.TotalActualizaciones(), totalMapa(resultado.Conflictos),
	)
	if _, err := tx.Exec(`
		INSERT INTO audit_log (agente, accion, entidad, entidad_id, detalle)
		VALUES ('orquesta', 'fusionar_proyecto', 'proyecto', ?, ?)`,
		destino.ID, detalle,
	); err != nil {
		return nil, fmt.Errorf("auditar fusion de proyecto: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return resultado, nil
}

func anotarResultadoFusion(resultado *FusionProyectosResultado, nombre string, moved, conflicted int64) {
	if resultado == nil {
		return
	}
	if moved > 0 {
		resultado.Actualizadas[nombre] = moved
	}
	if conflicted > 0 {
		resultado.Conflictos[nombre] = conflicted
	}
}

func totalMapa(items map[string]int64) int64 {
	var total int64
	for _, n := range items {
		total += n
	}
	return total
}

func tablaConColumnasMerge(tx *Tx, table string, columns ...string) (bool, error) {
	if tx == nil {
		return false, fmt.Errorf("tx nil")
	}
	table = strings.TrimSpace(table)
	if table == "" {
		return false, fmt.Errorf("tabla vacía")
	}
	var count int
	if err := tx.QueryRow(tableExistsQuery(tx.driver), table).Scan(&count); err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}
	for _, column := range columns {
		if err := tx.QueryRow(columnExistsQuery(tx.driver), table, strings.TrimSpace(column)).Scan(&count); err != nil {
			return false, err
		}
		if count == 0 {
			return false, nil
		}
	}
	return true, nil
}

func prepararArchivoProyectoTx(tx *Tx, origen, destino *Proyecto) (string, string, error) {
	if origen == nil || destino == nil {
		return "", "", fmt.Errorf("proyectos inválidos")
	}
	baseSlug := slugProyecto(origen.Slug)
	if baseSlug == "" {
		baseSlug = slugProyecto(origen.Nombre)
	}
	if baseSlug == "" {
		baseSlug = "proyecto"
	}
	baseSlug = baseSlug + "-archivado-" + time.Now().UTC().Format("20060102-150405")
	candidateSlug := baseSlug
	for idx := 1; ; idx++ {
		var total int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM proyectos WHERE slug = ?`, candidateSlug).Scan(&total); err != nil {
			return "", "", err
		}
		if total == 0 {
			break
		}
		candidateSlug = fmt.Sprintf("%s-%d", baseSlug, idx)
	}

	baseDir := filepath.Join(filepath.Clean(destino.RutaAbs), ".orquesta-archived-projects")
	candidatePath := filepath.Join(baseDir, candidateSlug)
	for idx := 1; ; idx++ {
		var total int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM proyectos WHERE ruta_abs = ?`, candidatePath).Scan(&total); err != nil {
			return "", "", err
		}
		if total == 0 {
			break
		}
		candidatePath = filepath.Join(baseDir, fmt.Sprintf("%s-%d", candidateSlug, idx))
	}
	return candidateSlug, candidatePath, nil
}

func moverRowsProyectoUniqueKeyTx(tx *Tx, table, keyColumn string, origenID, destinoID int64) (int64, int64, error) {
	rows, err := tx.Query(fmt.Sprintf(`SELECT id, %s FROM %s WHERE proyecto_id = ? ORDER BY id`, keyColumn, table), origenID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var (
		moved      int64
		conflicted int64
	)
	for rows.Next() {
		var id int64
		var key string
		if err := rows.Scan(&id, &key); err != nil {
			return moved, conflicted, err
		}
		var total int
		if err := tx.QueryRow(
			fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE proyecto_id = ? AND %s = ?`, table, keyColumn),
			destinoID, key,
		).Scan(&total); err != nil {
			return moved, conflicted, err
		}
		if total > 0 {
			conflicted++
			continue
		}
		n, err := execRowsAfectadas(tx, fmt.Sprintf(`UPDATE %s SET proyecto_id = ? WHERE id = ?`, table), destinoID, id)
		if err != nil {
			return moved, conflicted, err
		}
		moved += n
	}
	return moved, conflicted, rows.Err()
}

func moverTareasProyectoTx(tx *Tx, origenID, destinoID int64) (int64, int64, error) {
	rows, err := tx.Query(`SELECT id, COALESCE(blueprint_key, '') FROM tareas WHERE proyecto_id = ? ORDER BY id`, origenID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var (
		moved      int64
		conflicted int64
	)
	for rows.Next() {
		var id int64
		var blueprintKey string
		if err := rows.Scan(&id, &blueprintKey); err != nil {
			return moved, conflicted, err
		}
		blueprintKey = strings.TrimSpace(blueprintKey)
		if blueprintKey != "" {
			var total int
			if err := tx.QueryRow(
				`SELECT COUNT(*) FROM tareas WHERE proyecto_id = ? AND blueprint_key = ?`,
				destinoID, blueprintKey,
			).Scan(&total); err != nil {
				return moved, conflicted, err
			}
			if total > 0 {
				conflicted++
				continue
			}
		}
		n, err := execRowsAfectadas(tx, `UPDATE tareas SET proyecto_id = ? WHERE id = ?`, destinoID, id)
		if err != nil {
			return moved, conflicted, err
		}
		moved += n
	}
	return moved, conflicted, rows.Err()
}

func moverRowsProyectoStringUniqueKeyTx(tx *Tx, table, keyColumn, origenSlug, destinoSlug, archiveSlug string) (int64, int64, error) {
	rows, err := tx.Query(fmt.Sprintf(`SELECT id, %s FROM %s WHERE proyecto = ? ORDER BY id`, keyColumn, table), origenSlug)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var (
		moved      int64
		conflicted int64
	)
	for rows.Next() {
		var id int64
		var key string
		if err := rows.Scan(&id, &key); err != nil {
			return moved, conflicted, err
		}
		destinoFinal := destinoSlug
		var total int
		if err := tx.QueryRow(
			fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE proyecto = ? AND %s = ?`, table, keyColumn),
			destinoSlug, key,
		).Scan(&total); err != nil {
			return moved, conflicted, err
		}
		if total > 0 {
			destinoFinal = archiveSlug
			conflicted++
		}
		n, err := execRowsAfectadas(tx, fmt.Sprintf(`UPDATE %s SET proyecto = ? WHERE id = ?`, table), destinoFinal, id)
		if err != nil {
			return moved, conflicted, err
		}
		moved += n
	}
	return moved, conflicted, rows.Err()
}

func fusionarMemoriaProyectoTx(tx *Tx, origenSlug, destinoSlug, archiveSlug string) (int64, int64, error) {
	type memoriaProyectoRow struct {
		ID                int64
		Resumen           string
		Contexto          string
		PreguntasAbiertas string
		ActualizadoPor    string
	}

	cargar := func(slug string) (*memoriaProyectoRow, error) {
		var item memoriaProyectoRow
		err := tx.QueryRow(`
			SELECT id, resumen, contexto, preguntas_abiertas, actualizado_por
			FROM memoria_proyectos
			WHERE proyecto = ?`, slug,
		).Scan(&item.ID, &item.Resumen, &item.Contexto, &item.PreguntasAbiertas, &item.ActualizadoPor)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return &item, nil
	}

	source, err := cargar(origenSlug)
	if err != nil || source == nil {
		return 0, 0, err
	}
	target, err := cargar(destinoSlug)
	if err != nil {
		return 0, 0, err
	}
	if target == nil {
		n, err := execRowsAfectadas(tx, `UPDATE memoria_proyectos SET proyecto = ? WHERE id = ?`, destinoSlug, source.ID)
		return n, 0, err
	}

	mergedResumen := fusionarTextoMemoria(target.Resumen, source.Resumen)
	mergedContexto := fusionarTextoMemoria(target.Contexto, source.Contexto)
	mergedPreguntas := fusionarTextoMemoria(target.PreguntasAbiertas, source.PreguntasAbiertas)
	actualizadoPor := strings.TrimSpace(target.ActualizadoPor)
	if actualizadoPor == "" {
		actualizadoPor = strings.TrimSpace(source.ActualizadoPor)
	}
	n1, err := execRowsAfectadas(tx, `
		UPDATE memoria_proyectos
		SET resumen = ?, contexto = ?, preguntas_abiertas = ?, actualizado_por = ?
		WHERE id = ?`,
		mergedResumen, mergedContexto, mergedPreguntas, actualizadoPor, target.ID,
	)
	if err != nil {
		return 0, 0, err
	}
	n2, err := execRowsAfectadas(tx, `UPDATE memoria_proyectos SET proyecto = ? WHERE id = ?`, archiveSlug, source.ID)
	if err != nil {
		return 0, 0, err
	}
	return n1 + n2, 1, nil
}

func fusionarTextoMemoria(destino, origen string) string {
	destino = strings.TrimSpace(destino)
	origen = strings.TrimSpace(origen)
	switch {
	case origen == "":
		return destino
	case destino == "":
		return origen
	case strings.Contains(destino, origen):
		return destino
	default:
		return destino + "\n\n---\n" + origen
	}
}

func moverSingletonProyectoTx(tx *Tx, table string, origenID, destinoID int64) (int64, error) {
	var sourceTotal int
	if err := tx.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE proyecto_id = ?`, table), origenID).Scan(&sourceTotal); err != nil {
		return 0, err
	}
	if sourceTotal == 0 {
		return 0, nil
	}
	var targetTotal int
	if err := tx.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE proyecto_id = ?`, table), destinoID).Scan(&targetTotal); err != nil {
		return 0, err
	}
	if targetTotal > 0 {
		return 0, nil
	}
	return execRowsAfectadas(tx, fmt.Sprintf(`UPDATE %s SET proyecto_id = ? WHERE proyecto_id = ?`, table), destinoID, origenID)
}

func normalizarAsignacionesProyectoTx(tx *Tx, proyectoID int64, origenSlug, destinoSlug string) (int64, error) {
	rows, err := tx.Query(`
		SELECT agente
		FROM asignaciones
		WHERE proyecto_id = ? AND estado = 'activa'
		GROUP BY agente
		HAVING COUNT(*) > 1`, proyectoID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var total int64
	for rows.Next() {
		var agente string
		if err := rows.Scan(&agente); err != nil {
			return total, err
		}
		ids, err := listarAsignacionesActivasProyectoTx(tx, agente, proyectoID)
		if err != nil {
			return total, err
		}
		for idx, id := range ids {
			if idx == 0 {
				continue
			}
			n, err := execRowsAfectadas(tx, `
				UPDATE asignaciones
				SET estado = 'cerrada',
				    nota = ?,
				    cerrada_at = CURRENT_TIMESTAMP
				WHERE id = ?`,
				fmt.Sprintf("cerrada automáticamente tras fusion de proyectos %s -> %s", origenSlug, destinoSlug),
				id,
			)
			if err != nil {
				return total, err
			}
			total += n
		}
	}
	return total, rows.Err()
}

func listarAsignacionesActivasProyectoTx(tx *Tx, agente string, proyectoID int64) ([]int64, error) {
	rows, err := tx.Query(`
		SELECT id
		FROM asignaciones
		WHERE agente = ? AND proyecto_id = ? AND estado = 'activa'
		ORDER BY updated_at DESC, id DESC`, agente, proyectoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
