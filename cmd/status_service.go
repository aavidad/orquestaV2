package cmd

import (
	"time"

	"orquesta/db"
)

// StatusService encapsulates the data needed by /api/status.
type StatusService interface {
	FetchStatus() (apiStatusResponse, error)
}

var statusService StatusService = dbStatusService{}

type dbStatusService struct{}

func (dbStatusService) FetchStatus() (apiStatusResponse, error) {
	sesiones, err := db.ListarSesionesActivasOperativas()
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
		}
		tareasActivas = append(tareasActivas, lite)
	}
	agentesActivos := make([]*db.Agente, 0, len(agentes))
	for _, agente := range agentes {
		if agente != nil && agente.Activo {
			agentesActivos = append(agentesActivos, agente)
		}
	}
	return apiStatusResponse{
		Agentes:             agentes,
		ConteoTareas:        cuentas,
		Proyectos:           proyectos,
		AsignacionesActivas: asignaciones,
		SesionesActivas:     sesionesPorProyecto,
		PropuestasAbiertas:  abiertas,
		Generado:            time.Now().UTC().Format(time.RFC3339),
		TareasPorEstado:     cuentas,
		AgentesActivos:      agentesActivos,
		PropuestasResumen:   propuestasResumen,
		TareasActivas:       tareasActivas,
	}, nil
}
