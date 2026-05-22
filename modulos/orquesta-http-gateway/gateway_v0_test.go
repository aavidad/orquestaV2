package orquestahttpgateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAppGatewayMuxV0RegistersConfiguredRoutes(t *testing.T) {
	cases := []struct {
		name     string
		route    string
		handlers RouteHandlersV0
	}{
		{
			name:  "nueva app",
			route: RouteNuevaAppV0,
			handlers: RouteHandlersV0{
				NuevaApp: markerHandler("nueva-app"),
			},
		},
		{
			name:  "app spec",
			route: RouteAppSpecV0,
			handlers: RouteHandlersV0{
				AppSpec: markerHandler("app-spec"),
			},
		},
		{
			name:  "app change page",
			route: RouteAppChangePageV0,
			handlers: RouteHandlersV0{
				AppChangePage: markerHandler("app-change-page"),
			},
		},
		{
			name:  "director stats page",
			route: RouteDirectorStatsPageV0,
			handlers: RouteHandlersV0{
				DirectorStatsPage: markerHandler("director-stats-page"),
			},
		},
		{
			name:  "run control page",
			route: RouteRunControlPageV0,
			handlers: RouteHandlersV0{
				RunControlPage: markerHandler("run-control-page"),
			},
		},
		{
			name:  "run queue page",
			route: RouteRunQueuePageV0,
			handlers: RouteHandlersV0{
				RunQueuePage: markerHandler("run-queue-page"),
			},
		},
		{
			name:  "app director",
			route: RouteAppDirectorV0,
			handlers: RouteHandlersV0{
				AppDirector: markerHandler("app-director"),
			},
		},
		{
			name:  "app change",
			route: "/api/v0/apps/app-ref-001/changes",
			handlers: RouteHandlersV0{
				AppChange: markerHandler("app-change"),
			},
		},
		{
			name:  "director stats",
			route: RouteDirectorStatsV0,
			handlers: RouteHandlersV0{
				DirectorStats: markerHandler("director-stats"),
			},
		},
		{
			name:  "run control",
			route: RouteRunControlV0,
			handlers: RouteHandlersV0{
				RunControl: markerHandler("run-control"),
			},
		},
		{
			name:  "run queue priority",
			route: RouteRunQueuePriorityV0,
			handlers: RouteHandlersV0{
				RunQueuePriority: markerHandler("run-queue-priority"),
			},
		},
		{
			name:  "run supervisor",
			route: RouteRunSupervisorV0,
			handlers: RouteHandlersV0{
				RunSupervisor: markerHandler("run-supervisor"),
			},
		},
		{
			name:  "server shutdown",
			route: RouteServerShutdownV0,
			handlers: RouteHandlersV0{
				ServerShutdown: markerHandler("server-shutdown"),
			},
		},
		{
			name:  "autoprogramming validate request",
			route: RouteAutoprogrammingValidateRequestV0,
			handlers: RouteHandlersV0{
				AutoprogrammingValidateRequest: markerHandler("autoprogramming-validate-request"),
			},
		},
		{
			name:  "autoprogramming prepare run",
			route: RouteAutoprogrammingPrepareRunV0,
			handlers: RouteHandlersV0{
				AutoprogrammingPrepareRun: markerHandler("autoprogramming-prepare-run"),
			},
		},
		{
			name:  "domain work",
			route: RouteDomainWorkV0,
			handlers: RouteHandlersV0{
				DomainWork: markerHandler("domain-work"),
			},
		},
		{
			name:  "external work run",
			route: RouteExternalWorkRunV0,
			handlers: RouteHandlersV0{
				ExternalWorkRun: markerHandler("external-work-run"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tc.route, nil)

			NewAppGatewayMuxV0(tc.handlers).ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			if response.Body.String() == "" {
				t.Fatal("expected configured handler response")
			}
		})
	}
}

func TestNewAppGatewayMuxV0PrefiereRutasExactasAntesDeCambioDinamico(t *testing.T) {
	mux := NewAppGatewayMuxV0(RouteHandlersV0{
		AppSpec:     markerHandler("app-spec"),
		AppDirector: markerHandler("app-director"),
		AppChange:   markerHandler("app-change"),
	})

	for _, tc := range []struct {
		path string
		want string
	}{
		{RouteAppSpecV0, "app-spec"},
		{RouteAppDirectorV0, "app-director"},
		{"/api/v0/apps/app-ref-001/changes", "app-change"},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, tc.path, nil)

		mux.ServeHTTP(response, request)

		if response.Body.String() != tc.want {
			t.Fatalf("%s body=%q want %q", tc.path, response.Body.String(), tc.want)
		}
	}
}

func TestNewAppGatewayMuxV0PreservesPathAndMethod(t *testing.T) {
	const wantMethod = http.MethodPost
	const wantPath = RouteAppDirectorV0

	seen := make(chan struct {
		method string
		path   string
	}, 1)

	mux := NewAppGatewayMuxV0(RouteHandlersV0{
		AppDirector: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen <- struct {
				method string
				path   string
			}{
				method: r.Method,
				path:   r.URL.Path,
			}
			w.WriteHeader(http.StatusAccepted)
		}),
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(wantMethod, wantPath, nil)

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}

	got := <-seen
	if got.method != wantMethod {
		t.Fatalf("method = %q, want %q", got.method, wantMethod)
	}
	if got.path != wantPath {
		t.Fatalf("path = %q, want %q", got.path, wantPath)
	}
}

func TestNewAppGatewayMuxV0Returns404ForUnconfiguredRoutes(t *testing.T) {
	mux := NewAppGatewayMuxV0(RouteHandlersV0{
		NuevaApp: markerHandler("nueva-app"),
	})

	for _, route := range []string{
		RouteAppSpecV0,
		RouteAppDirectorV0,
		RouteDirectorStatsPageV0,
		RouteDirectorStatsV0,
		RouteDomainWorkV0,
		"/no-existe",
	} {
		t.Run(route, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, route, nil)

			mux.ServeHTTP(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
			}
		})
	}
}

func markerHandler(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	})
}
