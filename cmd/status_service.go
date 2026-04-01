package cmd

import (
	"strings"
	"time"

	"orquesta/db"
)

// StatusService encapsulates the data needed by /api/status.
type StatusService interface {
	FetchStatus() (apiStatusResponse, error)
}

var statusService StatusService = dbStatusService{}

type dbStatusService struct{}

func agenteTieneActividadRecienteVisible(agente *db.Agente) bool {
	if agente == nil || agente.UltimaSesion == nil || agente.UltimaSesion.IsZero() {
		return false
	}
	return agente.UltimaSesion.After(time.Now().UTC().Add(-5 * time.Minute))
}

func sesionesVisiblesParaEstado() ([]*db.Sesion, error) {
	return db.ListarSesionesActivasOperativas()
}

func nombresSesionesVisibles() (map[string]bool, error) {
	sesiones, err := sesionesVisiblesParaEstado()
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(sesiones))
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		nombre := strings.ToLower(strings.TrimSpace(sesion.Agente))
		if nombre != "" {
			out[nombre] = true
		}
	}
	return out, nil
}

func agenteCuentaComoConectado(agente *db.Agente) bool {
	if agente == nil {
		return false
	}
	if agenteBloqueadoPorCuotaVisible(agente) {
		return false
	}
	if agente.Activo {
		return true
	}
	estadoSesion := strings.TrimSpace(agente.EstadoSesion)
	if estadoSesion != "" && !strings.EqualFold(estadoSesion, "cerrada") && !strings.EqualFold(estadoSesion, "pausada") {
		return true
	}
	return agenteTieneActividadRecienteVisible(agente)
}

func (dbStatusService) FetchStatus() (apiStatusResponse, error) {
	sesiones, err := sesionesVisiblesParaEstado()
	if err != nil {
		return apiStatusResponse{}, err
	}
	agentes, err := db.ListarAgentesConSesionesActivas(sesiones)
	if err != nil {
		return apiStatusResponse{}, err
	}
	cuentas, err := db.ContarTareasPorEstado()
	if err != nil {
		return apiStatusResponse{}, err
	}
	proyectos, err := db.ListarProyectosConRutaEfectiva(db.FiltroProyectos{}, "")
	if err != nil {
		return apiStatusResponse{}, err
	}
	asignaciones, err := db.ContarAsignacionesActivasPorProyecto()
	if err != nil {
		return apiStatusResponse{}, err
	}
	sesionesPorProyecto := map[int64]int{}
	for _, sesion := range sesiones {
		if sesion.ProyectoID != nil {
			sesionesPorProyecto[*sesion.ProyectoID]++
		}
	}
	estadoAbierta := db.PropuestaAbierta
	abiertas, err := db.ListarPropuestas(&estadoAbierta, nil)
	if err != nil {
		return apiStatusResponse{}, err
	}
	propuestaIDs := make([]int64, 0, len(abiertas))
	for _, propuesta := range abiertas {
		if propuesta == nil {
			continue
		}
		propuestaIDs = append(propuestaIDs, propuesta.ID)
	}
	votosPorPropuesta, err := db.ResumenVotosPorPropuestas(propuestaIDs)
	if err != nil {
		return apiStatusResponse{}, err
	}
	propuestasResumen := make([]propuestaLite, 0, len(abiertas))
	for _, propuesta := range abiertas {
		if propuesta == nil {
			continue
		}
		votos := votosPorPropuesta[propuesta.ID]
		propuesta.Votos = votos
		lite := propuestaLite{
			ID:           propuesta.ID,
			Codigo:       propuesta.Codigo,
			Titulo:       propuesta.Titulo,
			Estado:       propuesta.Estado,
			PropuestoPor: propuesta.PropuestoPor,
		}
		for _, voto := range votos {
			if voto == nil {
				continue
			}
			switch voto.Posicion {
			case db.VotoAcuerdo:
				lite.Acuerdo++
			case db.VotoDesacuerdo:
				lite.Desacuerdo++
			case db.VotoAbstencion:
				lite.Abstencion++
			default:
				lite.Pendiente++
			}
		}
		propuestasResumen = append(propuestasResumen, lite)
	}
	todasLasTareas, err := db.ListarTareas(db.FiltroTareas{})
	if err != nil {
		return apiStatusResponse{}, err
	}
	tareasActivas := make([]tareaLite, 0, len(todasLasTareas))
	trabajandoNombres := make(map[string]bool)
	for _, tarea := range todasLasTareas {
		if tarea == nil {
			continue
		}
		if tarea.ProyectoID == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaAsignada, db.TareaEnProgreso, db.TareaBloqueada:
		default:
			continue
		}
		lite := tareaLite{
			ID:        tarea.ID,
			Titulo:    tarea.Titulo,
			Estado:    tarea.Estado,
			Modulo:    tarea.Modulo,
			Prioridad: tarea.Prioridad,
		}
		if tarea.Agente != nil {
			lite.Agente = *tarea.Agente
			if tarea.Estado == db.TareaEnProgreso && strings.TrimSpace(*tarea.Agente) != "" {
				trabajandoNombres[strings.TrimSpace(*tarea.Agente)] = true
			}
		}
		tareasActivas = append(tareasActivas, lite)
	}
	agentesActivos := make([]*db.Agente, 0, len(agentes))
	agentesPorNombre := make(map[string]*db.Agente, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		agentesPorNombre[agente.Nombre] = agente
		if agenteCuentaComoConectado(agente) {
			agentesActivos = append(agentesActivos, agente)
		}
	}
	trabajandoPorNombre := make(map[string]*db.Agente)
	for nombre := range trabajandoNombres {
		agente := agentesPorNombre[nombre]
		if !agenteCuentaComoConectado(agente) {
			continue
		}
		trabajandoPorNombre[agente.Nombre] = agente
	}
	agentesTrabajando := make([]*db.Agente, 0, len(trabajandoPorNombre))
	for _, agente := range agentesActivos {
		if agente == nil {
			continue
		}
		if trabajandoPorNombre[agente.Nombre] != nil {
			agentesTrabajando = append(agentesTrabajando, agente)
		}
	}
	return apiStatusResponse{
		Agentes:             agentes,
		ConteoTareas:        cuentas,
		ResumenTareas:       cuentas,
		Proyectos:           proyectos,
		AsignacionesActivas: asignaciones,
		SesionesActivas:     sesionesPorProyecto,
		PropuestasAbiertas:  abiertas,
		Generado:            time.Now().UTC().Format(time.RFC3339),
		TareasPorEstado:     cuentas,
		AgentesActivos:      agentesActivos,
		AgentesTrabajando:   agentesTrabajando,
		PropuestasResumen:   propuestasResumen,
		TareasActivas:       tareasActivas,
	}, nil
}
