package cmd

import (
	"fmt"
	"strings"

	"orquesta/microprogramacionapp"
	"orquesta/runtimesapp"
)

type microprogramacionDespachadorRuntime struct{}

func (microprogramacionDespachadorRuntime) DespacharMicrotarea(solicitud microprogramacionapp.SolicitudDespachoMicrotarea) (*microprogramacionapp.MicrotareaDespachada, error) {
	agenteDestino := strings.TrimSpace(solicitud.AgenteDestino)
	if agenteDestino == "" {
		return nil, fmt.Errorf("agente destino obligatorio")
	}
	if solicitud.Microtarea == nil {
		return nil, fmt.Errorf("microtarea obligatoria")
	}
	if solicitud.ProyectoID == nil || *solicitud.ProyectoID <= 0 {
		return nil, fmt.Errorf("proyecto obligatorio para despachar microtarea")
	}
	resultado, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
		AgenteDestino:     agenteDestino,
		ProyectoID:        solicitud.ProyectoID,
		Mensaje:           strings.TrimSpace(solicitud.Microtarea.Mensaje),
		EspecificacionID:  solicitud.EspecificacionID,
		ArchivoObjetivo:   strings.TrimSpace(solicitud.Microtarea.ArchivoObjetivo),
		SimboloObjetivo:   strings.TrimSpace(solicitud.Microtarea.SimboloObjetivo),
		WriteSet:          solicitud.Microtarea.WriteSet,
		TestsObligatorios: solicitud.Microtarea.TestsObligatorios,
		FormatoSalida:     strings.TrimSpace(solicitud.Microtarea.FormatoSalida),
	})
	if err != nil {
		return nil, err
	}
	return &microprogramacionapp.MicrotareaDespachada{
		EspecificacionID: solicitud.EspecificacionID,
		ProyectoID:       solicitud.ProyectoID,
		AgenteDestino:    agenteDestino,
		RuntimeOrderID:   resultado.RuntimeOrderID,
		Microtarea:       solicitud.Microtarea,
	}, nil
}

func init() {
	microprogramacionService.SetDespachador(microprogramacionDespachadorRuntime{})
}
