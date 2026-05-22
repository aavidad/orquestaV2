package orquestahttpgateway

import "net/http"

const (
	RouteNuevaAppV0                       = "/nueva-app"
	RouteAppChangePageV0                  = "/app-change"
	RouteDirectorStatsPageV0              = "/director-stats"
	RouteRunControlPageV0                 = "/run-control"
	RouteRunQueuePageV0                   = "/run-queue"
	RouteAppSpecV0                        = "/api/v0/apps/spec"
	RouteAppDirectorV0                    = "/api/v0/apps/director"
	RouteAppChangeV0                      = "/api/v0/apps/"
	RouteDirectorStatsV0                  = "/api/v0/director/stats"
	RouteRunControlV0                     = "/api/v0/runs/control"
	RouteRunQueuePriorityV0               = "/api/v0/runs/queue/priority"
	RouteRunSupervisorV0                  = "/api/v0/runs/supervise"
	RouteServerShutdownV0                 = "/api/v0/server/shutdown"
	RouteAutoprogrammingValidateRequestV0 = "/api/v0/autoprogramming/validate-request"
	RouteAutoprogrammingPrepareRunV0      = "/api/v0/autoprogramming/prepare-run"
	RouteDomainWorkV0                     = "/api/v0/domain-work"
	RouteExternalWorkRunV0                = "/api/v0/external-work/run"
)

type RouteHandlersV0 struct {
	NuevaApp                       http.Handler
	AppChangePage                  http.Handler
	DirectorStatsPage              http.Handler
	RunControlPage                 http.Handler
	RunQueuePage                   http.Handler
	AppSpec                        http.Handler
	AppDirector                    http.Handler
	AppChange                      http.Handler
	DirectorStats                  http.Handler
	RunControl                     http.Handler
	RunQueuePriority               http.Handler
	RunSupervisor                  http.Handler
	ServerShutdown                 http.Handler
	AutoprogrammingValidateRequest http.Handler
	AutoprogrammingPrepareRun      http.Handler
	DomainWork                     http.Handler
	ExternalWorkRun                http.Handler
}

func NewAppGatewayMuxV0(handlers RouteHandlersV0) http.Handler {
	mux := http.NewServeMux()

	handleIfPresent(mux, RouteNuevaAppV0, handlers.NuevaApp)
	handleIfPresent(mux, RouteAppChangePageV0, handlers.AppChangePage)
	handleIfPresent(mux, RouteDirectorStatsPageV0, handlers.DirectorStatsPage)
	handleIfPresent(mux, RouteRunControlPageV0, handlers.RunControlPage)
	handleIfPresent(mux, RouteRunQueuePageV0, handlers.RunQueuePage)
	handleIfPresent(mux, RouteAppSpecV0, handlers.AppSpec)
	handleIfPresent(mux, RouteAppDirectorV0, handlers.AppDirector)
	handleIfPresent(mux, RouteAppChangeV0, handlers.AppChange)
	handleIfPresent(mux, RouteDirectorStatsV0, handlers.DirectorStats)
	handleIfPresent(mux, RouteRunControlV0, handlers.RunControl)
	handleIfPresent(mux, RouteRunQueuePriorityV0, handlers.RunQueuePriority)
	handleIfPresent(mux, RouteRunSupervisorV0, handlers.RunSupervisor)
	handleIfPresent(mux, RouteServerShutdownV0, handlers.ServerShutdown)
	handleIfPresent(mux, RouteAutoprogrammingValidateRequestV0, handlers.AutoprogrammingValidateRequest)
	handleIfPresent(mux, RouteAutoprogrammingPrepareRunV0, handlers.AutoprogrammingPrepareRun)
	handleIfPresent(mux, RouteDomainWorkV0, handlers.DomainWork)
	handleIfPresent(mux, RouteExternalWorkRunV0, handlers.ExternalWorkRun)

	return mux
}

func handleIfPresent(mux *http.ServeMux, route string, handler http.Handler) {
	if handler == nil {
		return
	}

	mux.Handle(route, handler)
}
