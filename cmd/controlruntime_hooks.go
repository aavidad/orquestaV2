package cmd

import (
	"strings"

	"orquesta/agentesapp"
	"orquesta/internal/controlruntime"
)

func init() {
	controlruntime.SetSupervisorSignalHandler(processSupervisorSignal)
}

func processSupervisorSignal(signal controlruntime.SupervisorSignal) error {
	agente := strings.TrimSpace(signal.Agente)
	proyecto := strings.TrimSpace(signal.Proyecto)
	if agente == "" || proyecto == "" {
		return nil
	}
	_, err := agentesService.ProcessTick(agentesapp.TickInput{
		Agente:     agente,
		Proyecto:   proyecto,
		Host:       strings.TrimSpace(signal.Host),
		PID:        int64(signal.PID),
		CuotaPct:   100,
		Finalizado: signal.Finalizado,
		Motivo:     strings.TrimSpace(signal.Motivo),
	})
	return err
}
