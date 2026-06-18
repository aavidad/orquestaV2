package orquestacorereplanner

// Limite de reintentos de replan por tarea. Evita bucles infinitos de
// retry/split: cuando una tarea ya se ha replanificado este numero de veces sin
// cerrarse, el Director deja de intentar resolverlo automaticamente y escala a
// `ask_director` (decision/humano). Es la frontera "el Director resuelve por
// agente; si no puede tras N intentos, escala". Ver
// docs/orquesta_director_y_entregas_2026-06-18.md.
const DefaultReplanRetryLimitV0 = 3

// CapReplanActionByRetryLimitV0 acota la accion recomendada segun cuantos replans
// previos acumula ya la tarea. Acciones "resolutivas por agente" (retry, split,
// replace_agent, escalate_capacity) se convierten en `ask_director` cuando se
// alcanza el limite, para no reintentar en bucle. `ask_director` y `abort_task`
// no se tocan (ya son terminales/escalado). Es pura y determinista.
//
//   - priorReplanCount: numero de ReplanDecisions ya registrados para la tarea.
//   - limit <= 0 usa DefaultReplanRetryLimitV0.
func CapReplanActionByRetryLimitV0(
	action ReplanRecommendedActionV0,
	priorReplanCount int,
	limit int,
) ReplanRecommendedActionV0 {
	if limit <= 0 {
		limit = DefaultReplanRetryLimitV0
	}
	if priorReplanCount < limit {
		return action
	}
	switch action {
	case ReplanActionRetryTaskV0,
		ReplanActionSplitTaskV0,
		ReplanActionReplaceAgentV0,
		ReplanActionEscalateCapacityV0:
		// Resolucion automatica agotada: escala a decision (Director/humano).
		return ReplanActionAskDirectorV0
	default:
		// ask_director y abort_task ya son terminales/escalado.
		return action
	}
}

// ReplanRetryLimitExceededV0 indica si la tarea ya alcanzo el limite de replans.
func ReplanRetryLimitExceededV0(priorReplanCount int, limit int) bool {
	if limit <= 0 {
		limit = DefaultReplanRetryLimitV0
	}
	return priorReplanCount >= limit
}
