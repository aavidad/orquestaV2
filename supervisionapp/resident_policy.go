package supervisionapp

import (
	"fmt"
	"strings"
)

// BuildResidentSupervisorEventDrivenPolicyInput normalizes the canonical
// project autonomy profile where a reserved supervisor stays resident and the
// control plane drives the rest of the work through events/tasks.
func BuildResidentSupervisorEventDrivenPolicyInput(in PolicyInput) (PolicyInput, error) {
	out := in
	out.Enabled = true
	out.ObjetivoGeneral = strings.TrimSpace(out.ObjetivoGeneral)
	out.DefinitionOfDoneJSON = strings.TrimSpace(out.DefinitionOfDoneJSON)
	out.SupervisorAgente = strings.TrimSpace(out.SupervisorAgente)
	out.ReviewerAgente = strings.TrimSpace(out.ReviewerAgente)
	out.ReserveSupervisor = true
	out.AutoCreateTasks = true

	if out.DefinitionOfDoneJSON == "" {
		out.DefinitionOfDoneJSON = "{}"
	}
	if out.MaxWorkers < 0 {
		return PolicyInput{}, fmt.Errorf("max_workers no puede ser negativo")
	}
	if out.SupervisorAgente == "" {
		return PolicyInput{}, fmt.Errorf("supervisor_agente obligatorio para politica residente")
	}
	if out.ReviewRequired {
		if out.ReviewerAgente == "" {
			return PolicyInput{}, fmt.Errorf("reviewer_agente obligatorio cuando review_required=true")
		}
		out.ReserveReviewer = true
	}

	estado := strings.TrimSpace(string(out.EstadoAutonomia))
	if estado == "" {
		out.EstadoAutonomia = AutonomiaProyectoActiva
		return out, nil
	}
	estadoNormalizado, err := ParseEstadoAutonomiaProyecto(estado)
	if err != nil {
		return PolicyInput{}, err
	}
	out.EstadoAutonomia = estadoNormalizado
	return out, nil
}
