package db

import (
	"database/sql"
	"sort"
	"strings"
)

type autonomiaSupervisorCandidato struct {
	agente    *Agente
	preferred bool
	activo    bool
	score     int
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
	if preferred != "" && !strings.EqualFold(preferred, exclude) {
		nombreCanonico, _, _, err := resolverAgentePorNombreCI(preferred)
		if err == nil {
			preferred = nombreCanonico
		} else if err != sql.ErrNoRows {
			return nil, err
		}
		agente, err := GetAgente(preferred)
		if err == sql.ErrNoRows {
			agente = nil
			err = nil
		}
		if err != nil {
			return nil, err
		}
		disponible, err := agenteAutonomiaDisponible(agente, proyectoID)
		if err != nil {
			return nil, err
		}
		if disponible {
			return agente, nil
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
			agente:    agente,
			preferred: preferred != "" && strings.EqualFold(strings.TrimSpace(agente.Nombre), preferred),
			activo:    activo,
			score:     autonomiaSupervisorRoleScore(agente.Rol),
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

	sort.Slice(candidatos, func(i, j int) bool {
		if candidatos[i].preferred != candidatos[j].preferred {
			return candidatos[i].preferred
		}
		if candidatos[i].activo != candidatos[j].activo {
			return candidatos[i].activo
		}
		if candidatos[i].score != candidatos[j].score {
			return candidatos[i].score < candidatos[j].score
		}
		return strings.TrimSpace(candidatos[i].agente.Nombre) < strings.TrimSpace(candidatos[j].agente.Nombre)
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
	return false, nil
}

func autonomiaSupervisorRoleScore(role string) int {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "supervisor", "orquestador":
		return 0
	case "admin":
		return 1
	case "programador":
		return 2
	case "revisor", "reviewer":
		return 3
	default:
		return 4
	}
}
