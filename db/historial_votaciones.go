/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "strings"

type HistorialVotacionProyecto struct {
	Propuesta  *Propuesta
	Acuerdo    int
	Desacuerdo int
	Abstencion int
	Pendiente  int
	Votos      []*Voto
}

func ListarHistorialVotacionesProyecto(selector string) ([]*HistorialVotacionProyecto, error) {
	propuestas, err := ListarPropuestasProyecto(strings.TrimSpace(selector), nil)
	if err != nil {
		return nil, err
	}

	historial := make([]*HistorialVotacionProyecto, 0, len(propuestas))
	for _, propuesta := range propuestas {
		acuerdo, desacuerdo, abstencion, pendiente, err := ContarVotos(propuesta.ID)
		if err != nil {
			return nil, err
		}
		votos, err := ResumenVotos(propuesta.ID)
		if err != nil {
			return nil, err
		}
		historial = append(historial, &HistorialVotacionProyecto{
			Propuesta:  propuesta,
			Acuerdo:    acuerdo,
			Desacuerdo: desacuerdo,
			Abstencion: abstencion,
			Pendiente:  pendiente,
			Votos:      votos,
		})
	}
	return historial, nil
}
