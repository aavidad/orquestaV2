package main

import (
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

// EL FALLO: el ciclo de vida obtiene la autoridad CAS con un type assert sobre el
// MISMO store (ports.StateStore.(GoalWorkStateCASStorePortV0)). El decorador de
// wakeup envolvia el store y se dejaba el CAS por el camino, asi que el assert
// fallaba y NINGUN goal podia arrancar: 'ports.goal_state_cas_store'. Ese es el
// error exacto que nos mordio y que parcheamos en un test creyendo que era un fake
// incompleto.
//
// Este guard vigila la propiedad que importa: cualquier decorador del store de
// goals debe SEGUIR SIENDO un store con CAS. Envolver no puede quitar capacidades
// en silencio.
func TestDecoradorDeWakeupConservaLaAutoridadCASV0(t *testing.T) {
	var decorado orquestagoal.GoalWorkStateStorePortV0 = serverWakeupGoalStateStoreV0{}
	if _, ok := decorado.(orquestagoal.GoalWorkStateCASStorePortV0); !ok {
		t.Fatal(
			"el decorador de wakeup perdio el CAS: el ciclo de vida haria el type assert, fallaria, " +
				"y ningun goal podria arrancar (ports.goal_state_cas_store)",
		)
	}
}
