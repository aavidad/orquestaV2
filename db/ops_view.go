/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"database/sql"
	"strings"
	"time"
)

type SesionActiva struct {
	ID                 int64
	Agente             string
	Inicio             time.Time
	Fin                *time.Time
	Activa             bool
	ConectorID         *int64
	ConectorSlug       string
	ProyectoID         *int64
	ProyectoSlug       string
	Estado             string
	Cwd                string
	Herramienta        string
	ExternalSessionID  string
	ResumePayloadJSON  string
	ResumenContinuidad string
	Branch             string
	HeartbeatAt        *time.Time
	Host               string
	PID                *int64
}

func ListarAsignacionesOpsView(estado, agente string) ([]*Asignacion, error) {
	filtro := FiltroAsignaciones{}
	if agente = strings.TrimSpace(agente); agente != "" {
		filtro.Agente = &agente
	}
	if estado = strings.TrimSpace(estado); estado != "" {
		estadoAsignacion := EstadoAsignacion(estado)
		filtro.Estado = &estadoAsignacion
	}
	return ListarAsignaciones(filtro)
}

func ListarSesionesActivasOpsView() ([]*SesionActiva, error) {
	rows, err := DB.Query(`
		SELECT s.id, s.agente, s.inicio, s.fin, s.activa, s.conector_id, COALESCE(c.slug,''), s.proyecto_id, COALESCE(p.slug,''),
		       s.estado, s.cwd, s.herramienta, s.external_session_id, s.resume_payload_json,
		       s.resumen_continuidad, s.branch, s.heartbeat_at, s.host, s.pid
		FROM sesiones s
		LEFT JOIN conectores c ON c.id = s.conector_id
		LEFT JOIN proyectos p ON p.id = s.proyecto_id
		WHERE s.activa = 1
		ORDER BY s.id DESC`)
	if err != nil {
		return nil, err
	}
	items := make([]*SesionActiva, 0)
	for rows.Next() {
		item, err := scanSesionActiva(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	list := make([]*SesionActiva, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if !sesionOperativaPorHeartbeat(item.Activa, item.HeartbeatAt, item.Inicio, now) {
			activo, err := sesionTieneHandleActivoOperativo(item.Agente, item.ProyectoID)
			if err != nil {
				return nil, err
			}
			if !activo {
				continue
			}
		}
		list = append(list, item)
	}
	return list, nil
}

type OpsViewRepository struct{}

func (OpsViewRepository) ListAgents() ([]*Agente, error) {
	return ListarAgentes()
}

func (OpsViewRepository) ListConnectors() ([]*Conector, error) {
	return ListarConectores()
}

func (OpsViewRepository) ListAssignments(estado, agente string) ([]*Asignacion, error) {
	return ListarAsignacionesOpsView(estado, agente)
}

func (OpsViewRepository) ListActiveSessions() ([]*SesionActiva, error) {
	return ListarSesionesActivasOpsView()
}

func (OpsViewRepository) AuditLog(limit int) ([]AuditEntry, error) {
	return AuditLog(limit)
}

func (OpsViewRepository) RegisterAgent(nombre, rol string) error {
	return RegistrarAgente(nombre, rol)
}

func (OpsViewRepository) RetireAgent(nombre string) error {
	return RetirarAgente(nombre)
}

func (OpsViewRepository) RehabilitateAgent(nombre string) error {
	return RehabilitarAgente(nombre)
}

func scanSesionActiva(s scanner) (*SesionActiva, error) {
	var item SesionActiva
	var fin, heartbeat sql.NullTime
	var conectorID, proyectoID, pid sql.NullInt64
	if err := s.Scan(
		&item.ID, &item.Agente, &item.Inicio, &fin, &item.Activa, &conectorID, &item.ConectorSlug,
		&proyectoID, &item.ProyectoSlug, &item.Estado, &item.Cwd, &item.Herramienta,
		&item.ExternalSessionID, &item.ResumePayloadJSON, &item.ResumenContinuidad,
		&item.Branch, &heartbeat, &item.Host, &pid,
	); err != nil {
		return nil, err
	}
	if fin.Valid {
		item.Fin = &fin.Time
	}
	if heartbeat.Valid {
		item.HeartbeatAt = &heartbeat.Time
	}
	if conectorID.Valid {
		item.ConectorID = &conectorID.Int64
	}
	if proyectoID.Valid {
		item.ProyectoID = &proyectoID.Int64
	}
	if pid.Valid {
		item.PID = &pid.Int64
	}
	return &item, nil
}
