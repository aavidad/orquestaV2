package orquestahttpgateway

import "testing"

func TestPublicRouteMutabilityV0ClasificaMutacionesPublicas(t *testing.T) {
	for _, route := range []string{
		RouteAutoprogrammingPrepareRunV0,
		RouteAutoprogrammingSuperviseV0,
		RouteRunControlV0,
		RouteRunQueuePriorityV0,
		RouteRunSupervisorV0,
		RouteServerShutdownV0,
		RouteNuevaAppV0,
		RouteAppChangePageV0,
		RouteRunControlPageV0,
		RouteRunQueuePageV0,
		RouteAppChangeV0 + "agenda/changes",
	} {
		if got := PublicRouteMutabilityV0(route); got != PublicRouteMutationV0 {
			t.Fatalf("route %s mutability=%s", route, got)
		}
	}
}

func TestPublicRouteMutabilityV0LecturaPorDefecto(t *testing.T) {
	if got := PublicRouteMutabilityV0(RouteOperationalStatusV0); got != PublicRouteReadV0 {
		t.Fatalf("mutability=%s", got)
	}
}
