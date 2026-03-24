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
	"strings"
)

type FusionAgentesResultado struct {
	Origen           string           `json:"origen"`
	Destino          string           `json:"destino"`
	Actualizadas     map[string]int64 `json:"actualizadas"`
	VotosDescartados int64            `json:"votos_descartados"`
}

func (r *FusionAgentesResultado) TotalActualizaciones() int64 {
	if r == nil {
		return 0
	}
	var total int64
	for _, n := range r.Actualizadas {
		total += n
	}
	return total
}

func FusionarAgentes(origen, destino string) (*FusionAgentesResultado, error) {
	origen = strings.TrimSpace(origen)
	destino = strings.TrimSpace(destino)
	switch {
	case origen == "":
		return nil, fmt.Errorf("debes indicar el agente origen")
	case destino == "":
		return nil, fmt.Errorf("debes indicar el agente destino")
	case strings.EqualFold(origen, destino) && origen == destino:
		return nil, fmt.Errorf("el agente origen y destino no pueden ser el mismo")
	}

	if _, err := GetAgente(origen); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agente origen '%s' no encontrado", origen)
		}
		return nil, err
	}
	if _, err := GetAgente(destino); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agente destino '%s' no encontrado", destino)
		}
		return nil, err
	}

	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := validarFusionAgente(tx, origen); err != nil {
		return nil, err
	}

	resultado := &FusionAgentesResultado{
		Origen:       origen,
		Destino:      destino,
		Actualizadas: map[string]int64{},
	}

	descartados, err := execRowsAfectadas(tx, `
		DELETE FROM votos
		WHERE agente = ?
		  AND propuesta_id IN (
			SELECT propuesta_id
			FROM votos
			WHERE agente = ?
		  )`, origen, destino)
	if err != nil {
		return nil, fmt.Errorf("resolver colisiones de votos: %w", err)
	}
	resultado.VotosDescartados = descartados

	operaciones := []struct {
		nombre string
		query  string
	}{
		{nombre: "tareas.agente", query: `UPDATE tareas SET agente = ? WHERE agente = ?`},
		{nombre: "tareas.creado_por", query: `UPDATE tareas SET creado_por = ? WHERE creado_por = ?`},
		{nombre: "propuestas.propuesto_por", query: `UPDATE propuestas SET propuesto_por = ? WHERE propuesto_por = ?`},
		{nombre: "votos.agente", query: `UPDATE votos SET agente = ? WHERE agente = ?`},
		{nombre: "bloqueos.bloqueado_por", query: `UPDATE bloqueos SET bloqueado_por = ? WHERE bloqueado_por = ?`},
		{nombre: "sesiones.agente", query: `UPDATE sesiones SET agente = ? WHERE agente = ?`},
		{nombre: "asignaciones.agente", query: `UPDATE asignaciones SET agente = ? WHERE agente = ?`},
		{nombre: "locks.agente", query: `UPDATE locks SET agente = ? WHERE agente = ?`},
		{nombre: "worktrees.agente", query: `UPDATE worktrees SET agente = ? WHERE agente = ?`},
		{nombre: "runtime_instances.agente", query: `UPDATE runtime_instances SET agente = ? WHERE agente = ?`},
		{nombre: "runtime_handles.agente", query: `UPDATE runtime_handles SET agente = ? WHERE agente = ?`},
		{nombre: "runtime_orders.agente", query: `UPDATE runtime_orders SET agente = ? WHERE agente = ?`},
		{nombre: "runtime_mailbox.from_agente", query: `UPDATE runtime_mailbox SET from_agente = ? WHERE from_agente = ?`},
		{nombre: "runtime_mailbox.to_agente", query: `UPDATE runtime_mailbox SET to_agente = ? WHERE to_agente = ?`},
		{nombre: "runtime_checkpoints.agente", query: `UPDATE runtime_checkpoints SET agente = ? WHERE agente = ?`},
		{nombre: "git_merges.requested_by", query: `UPDATE git_merges SET requested_by = ? WHERE requested_by = ?`},
		{nombre: "memoria_proyectos.actualizado_por", query: `UPDATE memoria_proyectos SET actualizado_por = ? WHERE actualizado_por = ?`},
		{nombre: "memoria_fuentes.registrado_por", query: `UPDATE memoria_fuentes SET registrado_por = ? WHERE registrado_por = ?`},
		{nombre: "memoria_hallazgos.registrado_por", query: `UPDATE memoria_hallazgos SET registrado_por = ? WHERE registrado_por = ?`},
		{nombre: "memoria_derivas.detectada_por", query: `UPDATE memoria_derivas SET detectada_por = ? WHERE detectada_por = ?`},
		{nombre: "avance_tareas.actualizado_por", query: `UPDATE avance_tareas SET actualizado_por = ? WHERE actualizado_por = ?`},
		{nombre: "entidades_memoria.verificado_por", query: `UPDATE entidades_memoria SET verificado_por = ? WHERE verificado_por = ?`},
		{nombre: "refineria_solicitudes.agente", query: `UPDATE refineria_solicitudes SET agente = ? WHERE agente = ?`},
		{nombre: "audit_log.agente", query: `UPDATE audit_log SET agente = ? WHERE agente = ?`},
	}
	for _, op := range operaciones {
		n, err := execRowsAfectadas(tx, op.query, destino, origen)
		if err != nil {
			return nil, fmt.Errorf("actualizando %s: %w", op.nombre, err)
		}
		if n > 0 {
			resultado.Actualizadas[op.nombre] = n
		}
	}

	borrados, err := execRowsAfectadas(tx, `DELETE FROM agentes WHERE nombre = ?`, origen)
	if err != nil {
		return nil, fmt.Errorf("eliminar agente origen: %w", err)
	}
	if borrados != 1 {
		return nil, fmt.Errorf("no se pudo eliminar el agente origen '%s'", origen)
	}

	detalle := fmt.Sprintf("%s->%s actualizadas=%d votos_descartados=%d", origen, destino, resultado.TotalActualizaciones(), resultado.VotosDescartados)
	if _, err := tx.Exec(`
		INSERT INTO audit_log (agente, accion, entidad, entidad_id, detalle)
		VALUES (?, 'fusionar_agente', 'agente', 0, ?)`, destino, detalle); err != nil {
		return nil, fmt.Errorf("auditar fusion de agente: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return resultado, nil
}

func validarFusionAgente(tx queryRower, origen string) error {
	validaciones := []struct {
		query string
		label string
		args  []any
	}{
		{
			query: `SELECT COUNT(*) FROM sesiones WHERE agente = ? AND activa = 1`,
			label: "sesiones activas",
			args:  []any{origen},
		},
		{
			query: `SELECT COUNT(*) FROM locks WHERE agente = ? AND estado = 'activa'`,
			label: "locks activos",
			args:  []any{origen},
		},
		{
			query: `SELECT COUNT(*) FROM worktrees WHERE agente = ? AND estado = 'activa'`,
			label: "worktrees activos",
			args:  []any{origen},
		},
		{
			query: `SELECT COUNT(*) FROM runtime_handles WHERE agente = ? AND estado IN ('activo','pausado')`,
			label: "runtime handles activos",
			args:  []any{origen},
		},
		{
			query: `SELECT COUNT(*) FROM runtime_orders WHERE agente = ? AND estado IN ('pendiente','tomada','ejecutando')`,
			label: "runtime orders vivos",
			args:  []any{origen},
		},
	}
	for _, v := range validaciones {
		var total int
		if err := tx.QueryRow(v.query, v.args...).Scan(&total); err != nil {
			return fmt.Errorf("validar %s: %w", v.label, err)
		}
		if total > 0 {
			return fmt.Errorf("no se puede fusionar '%s': tiene %d %s", origen, total, v.label)
		}
	}
	return nil
}

func execRowsAfectadas(tx execer, query string, args ...any) (int64, error) {
	res, err := tx.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return n, nil
}
