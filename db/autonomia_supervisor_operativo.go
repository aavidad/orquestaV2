package db

import (
	"database/sql"
	"sort"
	"strings"

	"orquesta/autonomiapolicy"
)

type autonomiaSupervisorCandidato struct {
	agente   *Agente
	snapshot autonomiapolicy.SupervisorCandidateSnapshot
}

func SeleccionarSupervisorAutonomiaOperativo(proyectoID int64, exclude string) (*Agente, error) {
	if DB == nil {
		return nil, nil
	}
	if proyectoID <= 0 {
		return nil, nil
	}
	policy, err := GetProyectoAutonomia(proyectoID)
	if err != nil || policy == nil || !policy.Enabled {
		return nil, err
	}

	exclude = strings.TrimSpace(exclude)
	preferred := strings.TrimSpace(policy.SupervisorAgente)
	if preferred != "" {
		nombreCanonico, _, _, err := resolverAgentePorNombreCI(preferred)
		if err == nil {
			preferred = nombreCanonico
		} else if err != sql.ErrNoRows {
			return nil, err
		}
	}
	seen := map[string]struct{}{}
	candidatos := make([]autonomiaSupervisorCandidato, 0, 8)

	add := func(nombre string) error {
		nombre = strings.TrimSpace(nombre)
		if nombre == "" || strings.EqualFold(nombre, exclude) {
			return nil
		}
		agente, activo, err := resolverCandidatoSupervisorAutonomia(nombre, proyectoID)
		if err != nil {
			return err
		}
		if agente == nil {
			return nil
		}
		key := strings.ToLower(strings.TrimSpace(agente.Nombre))
		if _, ok := seen[key]; ok {
			return nil
		}
		seen[key] = struct{}{}
		candidatos = append(candidatos, autonomiaSupervisorCandidato{
			agente: agente,
			snapshot: autonomiapolicy.SupervisorCandidateSnapshot{
				AgentName: strings.TrimSpace(agente.Nombre),
				Preferred: preferred != "" && strings.EqualFold(strings.TrimSpace(agente.Nombre), preferred),
				Active:    activo,
				RoleScore: autonomiapolicy.SupervisorRoleScore(agente.Rol),
				CostTier:  autonomiapolicy.AgentCostTier(agente.Nombre),
			},
		})
		return nil
	}

	if err := add(preferred); err != nil {
		return nil, err
	}

	sesiones, err := ListarSesionesActivasOperativas()
	if err != nil {
		return nil, err
	}
	for _, sesion := range sesiones {
		if sesion == nil || sesion.ProyectoID == nil || *sesion.ProyectoID != proyectoID {
			continue
		}
		if err := add(sesion.Agente); err != nil {
			return nil, err
		}
	}

	estado := AsignacionActiva
	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{
		ProyectoID: &proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return nil, err
	}
	for _, asignacion := range asignaciones {
		if asignacion == nil {
			continue
		}
		if err := add(asignacion.Agente); err != nil {
			return nil, err
		}
	}
	tareas, err := ListarTareas(FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		return nil, err
	}
	for _, tarea := range tareas {
		if tarea == nil || tarea.Agente == nil {
			continue
		}
		switch tarea.Estado {
		case TareaAsignada, TareaEnProgreso, TareaBloqueada:
		default:
			continue
		}
		if err := add(strings.TrimSpace(*tarea.Agente)); err != nil {
			return nil, err
		}
	}

	sort.Slice(candidatos, func(i, j int) bool {
		return autonomiapolicy.PreferSupervisorCandidate(
			&candidatos[i].snapshot,
			&candidatos[j].snapshot,
		)
	})
	if len(candidatos) == 0 {
		return nil, nil
	}
	return candidatos[0].agente, nil
}

func EsSupervisorAutonomiaOperativo(proyectoID int64, agente string) (bool, *Agente, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return false, nil, nil
	}
	supervisor, err := SeleccionarSupervisorAutonomiaOperativo(proyectoID, "")
	if err != nil || supervisor == nil {
		return false, supervisor, err
	}
	return strings.EqualFold(strings.TrimSpace(supervisor.Nombre), agente), supervisor, nil
}

func resolverCandidatoSupervisorAutonomia(nombre string, proyectoID int64) (*Agente, bool, error) {
	agente, err := GetAgente(strings.TrimSpace(nombre))
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	disponible, err := agenteAutonomiaDisponible(agente, proyectoID)
	if err != nil {
		return nil, false, err
	}
	if !disponible {
		return nil, false, nil
	}
	activo, err := agenteAutonomiaActivoEnProyecto(strings.TrimSpace(agente.Nombre), proyectoID)
	if err != nil {
		return nil, false, err
	}
	return agente, activo, nil
}

func agenteAutonomiaDisponible(agente *Agente, proyectoID int64) (bool, error) {
	if agente == nil || !agente.Habilitado || proyectoID <= 0 {
		return false, nil
	}
	estadoCuota := strings.ToLower(strings.TrimSpace(agente.EstadoCuota))
	if estadoCuota != "" && estadoCuota != "activo" {
		return false, nil
	}
	proyectoActivoID, err := ObtenerProyectoActivoAgente(strings.TrimSpace(agente.Nombre))
	if err != nil {
		return false, err
	}
	if proyectoActivoID != 0 && proyectoActivoID != proyectoID {
		return false, nil
	}
	if activa, err := agenteAutonomiaActivoEnProyecto(strings.TrimSpace(agente.Nombre), proyectoID); err != nil {
		return false, err
	} else if activa {
		// Si el agente ya está vivo en el proyecto, reutilizarlo como supervisor
		// efectivo no requiere otra activación sobre la cuenta compartida.
		return true, nil
	}
	disponible, _, err := cuentaCompartidaPermiteActivacion(agente)
	return disponible, err
}

func agenteAutonomiaActivoEnProyecto(agente string, proyectoID int64) (bool, error) {
	if agente == "" || proyectoID <= 0 {
		return false, nil
	}
	if sesion, err := GetSesionActiva(agente, &proyectoID); err != nil && err != sql.ErrNoRows {
		return false, err
	} else if sesion != nil {
		return true, nil
	}
	if handle, err := runtimeHandleOperativoRecienteConFallback(agente, &proyectoID); err != nil {
		return false, err
	} else if handle != nil {
		return true, nil
	}
	if runtime, err := runtimePrincipalAgenteProyecto(agente, &proyectoID); err != nil {
		return false, err
	} else if runtime != nil {
		switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
		case "activo", "active", "running", "iniciando", "starting", "esperando_io", "waiting_io", "pausado", "paused":
			return true, nil
		}
	}
	return false, nil
}
