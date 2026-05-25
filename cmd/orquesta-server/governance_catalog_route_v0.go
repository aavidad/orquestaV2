package main

import (
	"net/http"

	orquestagovernance "orquesta/modulos/orquesta-governance"
)

func withGovernanceCatalogRouteV0(handler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(
		orquestagovernance.GovernanceCatalogQueryHTTPPathV0,
		orquestagovernance.GovernanceCatalogQueryHTTPHandlerV0(nil),
	)
	mux.Handle("/", handler)
	return mux
}
