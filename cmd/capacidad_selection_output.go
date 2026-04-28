package cmd

import (
	"fmt"
	"strings"

	"orquesta/capacidadapp"
)

func imprimirSeleccionAgenteCLI(seleccion *capacidadapp.SeleccionAgentePipeline) {
	if seleccion == nil {
		return
	}
	if estrategia := strings.TrimSpace(seleccion.Estrategia); estrategia != "" {
		fmt.Printf("Estrategia agente: %s\n", estrategia)
	}
	if motivo := strings.TrimSpace(seleccion.Motivo); motivo != "" {
		fmt.Printf("Motivo agente: %s\n", motivo)
	}
}

func imprimirModoDespachoCLI(despacho *capacidadapp.DespachoPipelineLocal) {
	if despacho == nil {
		return
	}
	if despacho.FinishApp {
		fmt.Printf("Modo dispatch: finish_app\n")
	}
}
