package orquestahttpgateway

import "strings"

const (
	PublicRouteMutationV0 = "mutation"
	PublicRouteReadV0     = "read"
)

func PublicRouteMutabilityV0(route string) string {
	route = strings.TrimSpace(route)
	if route == RouteAppIntakeGuidedTurnV0 ||
		route == RouteAppIntakeWizardBotV0 ||
		route == RouteAppDirectorPreviewV0 {
		return PublicRouteReadV0
	}
	if strings.HasPrefix(route, RouteAppChangeV0) {
		return PublicRouteMutationV0
	}
	switch route {
	case RouteAutoprogrammingPrepareRunV0,
		RouteAutoprogrammingObserveGoalV0,
		RouteAutoprogrammingObserveActiveGoalsV0,
		RouteAutoprogrammingSelfImprovementV0,
		RouteAutoprogrammingSuperviseV0,
		RouteRunControlV0,
		RouteRunQueuePriorityV0,
		RouteRunSupervisorV0,
		RouteServerShutdownV0,
		RouteNuevaAppV0,
		RouteAppChangePageV0,
		RouteRunControlPageV0,
		RouteRunQueuePageV0,
		RouteAppSpecV0,
		RouteAppDirectorV0,
		RouteAppDirectorGoalObserveV0,
		RouteAppChangeV0,
		RouteExternalWorkObserveV0,
		RouteExternalWorkRunV0,
		RouteDomainWorkV0:
		return PublicRouteMutationV0
	default:
		return PublicRouteReadV0
	}
}
