package cmd

import "orquesta/db"

// StatusService encapsulates the data needed by /api/status.
type StatusService interface {
	FetchStatus() (apiStatusResponse, error)
}

var statusService StatusService = dbStatusService{}

type dbStatusService struct{}

func (dbStatusService) FetchStatus() (apiStatusResponse, error) {
	agentes, err := db.ListarAgentes()
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
	sesiones, err := db.ListarSesionesActivas()
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
	for _, propuesta := range abiertas {
		if propuesta == nil {
			continue
		}
		votos, err := db.ResumenVotos(propuesta.ID)
		if err != nil {
			return apiStatusResponse{}, err
		}
		propuesta.Votos = votos
	}
	return apiStatusResponse{
		Agentes:             agentes,
		ConteoTareas:        cuentas,
		Proyectos:           proyectos,
		AsignacionesActivas: asignaciones,
		SesionesActivas:     sesionesPorProyecto,
		PropuestasAbiertas:  abiertas,
	}, nil
}
