package orquestadirectorscheduler

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

// SchedulerReplanRetryLimitV0 es el numero maximo de replans por tarea antes de
// escalar a decision (ask_director) en vez de seguir reintentando. Evita bucles
// infinitos retry/split observados en la auditoria del Director. Coincide con
// orquesta-core-replanner.DefaultReplanRetryLimitV0; se mantiene aqui como
// constante local para no acoplar el scheduler al paquete replanner.
const SchedulerReplanRetryLimitV0 = 3

const (
	schedulerReplanProjectionTaskSeparatorV0   = "#task:"
	schedulerReplanProjectionActionSeparatorV0 = "#action:"
)

// schedulerReplanRetryLimitExceededV0 indica si la tarea ya acumula
// SchedulerReplanRetryLimitV0 replans previos en el snapshot. La accion actual
// (la que se va a ejecutar) NO cuenta: solo los replans ya registrados.
func schedulerReplanRetryLimitExceededV0(
	snapshot RunSchedulingSnapshotV0,
	taskRef string,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) bool {
	taskRef = strings.TrimSpace(taskRef)
	if taskRef == "" {
		return false
	}
	// Acciones ya terminales/escalado no se capan (ya escalan o abortan).
	switch action {
	case orquestacoreworkflow.ReplanDecisionActionAskDirectorV0,
		orquestacoreworkflow.ReplanDecisionActionAbortTaskV0:
		return false
	}
	return schedulerReplanCountForTaskV0(snapshot.ReplanRefs, taskRef) >= SchedulerReplanRetryLimitV0
}

// schedulerReplanCountForTaskV0 cuenta cuantos replans previos hay para una tarea
// en las proyecciones compactas del snapshot
// (formato: ...#task:<taskRef>#action:<action>#...).
func schedulerReplanCountForTaskV0(replanRefs []string, taskRef string) int {
	count := 0
	for _, ref := range replanRefs {
		if schedulerReplanProjectionTaskRefV0(ref) == taskRef {
			count++
		}
	}
	return count
}

func schedulerReplanProjectionTaskRefV0(ref string) string {
	_, tail, ok := strings.Cut(strings.TrimSpace(ref), schedulerReplanProjectionTaskSeparatorV0)
	if !ok {
		return ""
	}
	taskRef, _, ok := strings.Cut(tail, schedulerReplanProjectionActionSeparatorV0)
	if !ok {
		return ""
	}
	return strings.TrimSpace(taskRef)
}
