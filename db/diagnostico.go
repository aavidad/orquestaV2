package db

import (
	"sort"
	"time"

	"orquesta/coordinacion"
)

type PropuestaDiagnostico struct {
	Propuesta  *Propuesta
	Acuerdo    int
	Desacuerdo int
	Abstencion int
	Pendiente  int
}

type SnapshotDiagnostico struct {
	GeneradoEn         time.Time
	Agentes            []*Agente
	SesionesActivas    []*Sesion
	ConteoTareas       map[string]int
	TareasBloqueadas   []*Tarea
	TareasEnProgreso   []*Tarea
	TareasLibres       []*Tarea
	PropuestasAbiertas []PropuestaDiagnostico
	LocksActivos       []*coordinacion.Lock
	WorktreesActivos   []*coordinacion.Worktree
	Config             map[string]string
	AuditoriaReciente  []AuditEntry
}

func ConstruirSnapshotDiagnostico(limitAudit int) (*SnapshotDiagnostico, error) {
	if limitAudit <= 0 {
		limitAudit = 20
	}

	agentes, err := ListarAgentes()
	if err != nil {
		return nil, err
	}
	sesiones, err := ListarSesionesActivas()
	if err != nil {
		return nil, err
	}
	conteoTareas, err := ContarTareasPorEstado()
	if err != nil {
		return nil, err
	}

	estadoBloqueada := TareaBloqueada
	tareasBloqueadas, err := ListarTareas(FiltroTareas{Estado: &estadoBloqueada})
	if err != nil {
		return nil, err
	}
	estadoEnProgreso := TareaEnProgreso
	tareasEnProgreso, err := ListarTareas(FiltroTareas{Estado: &estadoEnProgreso})
	if err != nil {
		return nil, err
	}
	tareasLibres, err := ListarTareas(FiltroTareas{Libre: true})
	if err != nil {
		return nil, err
	}

	estadoAbierta := PropuestaAbierta
	propuestas, err := ListarPropuestas(&estadoAbierta, nil)
	if err != nil {
		return nil, err
	}
	propuestasAbiertas := make([]PropuestaDiagnostico, 0, len(propuestas))
	for _, propuesta := range propuestas {
		acuerdo, desacuerdo, abstencion, pendiente, err := ContarVotos(propuesta.ID)
		if err != nil {
			return nil, err
		}
		propuestasAbiertas = append(propuestasAbiertas, PropuestaDiagnostico{
			Propuesta:  propuesta,
			Acuerdo:    acuerdo,
			Desacuerdo: desacuerdo,
			Abstencion: abstencion,
			Pendiente:  pendiente,
		})
	}

	lockState := coordinacion.LockState("activa")
	locksActivos, err := (SQLiteLockRepository{}).List(coordinacion.LockFilter{State: &lockState})
	if err != nil {
		return nil, err
	}
	worktreeState := coordinacion.WorktreeState("activa")
	worktreesActivos, err := (SQLiteWorktreeRepository{}).List(coordinacion.WorktreeFilter{State: &worktreeState})
	if err != nil {
		return nil, err
	}
	config, err := ConfigAll()
	if err != nil {
		return nil, err
	}
	auditoria, err := AuditLog(limitAudit)
	if err != nil {
		return nil, err
	}

	return &SnapshotDiagnostico{
		GeneradoEn:         time.Now(),
		Agentes:            agentes,
		SesionesActivas:    sesiones,
		ConteoTareas:       conteoTareas,
		TareasBloqueadas:   tareasBloqueadas,
		TareasEnProgreso:   tareasEnProgreso,
		TareasLibres:       tareasLibres,
		PropuestasAbiertas: propuestasAbiertas,
		LocksActivos:       locksActivos,
		WorktreesActivos:   worktreesActivos,
		Config:             config,
		AuditoriaReciente:  auditoria,
	}, nil
}

func ClavesOrdenadasConfig(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
