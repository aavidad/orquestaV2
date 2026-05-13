package orquestaappgateway

import (
	"net/http"

	orquestaweb "orquesta/modulos/orquesta-web"
)

func httpClientForWebV0(config ConfigV0, apiMux http.Handler) *http.Client {
	if config.HTTPClient != nil {
		return config.HTTPClient
	}
	return &http.Client{
		Transport: InProcessTransportV0{Handler: apiMux},
		Timeout:   config.Timeout,
	}
}

func newSpecClientV0(
	config ConfigV0,
	client *http.Client,
) *orquestaweb.RESTSolicitarNuevaAppClientV0 {
	out := orquestaweb.NewRESTSolicitarNuevaAppClientV0(InternalBaseURLV0, config.Timeout)
	out.HTTPClient = client
	return out
}

func newDirectorClientV0(
	config ConfigV0,
	client *http.Client,
) *orquestaweb.RESTArrancarDirectorAppClientV0 {
	out := orquestaweb.NewRESTArrancarDirectorAppClientV0(InternalBaseURLV0, config.Timeout)
	out.HTTPClient = client
	out.Limits = config.DirectorLimits
	return out
}

func newStatsClientV0(
	config ConfigV0,
	client *http.Client,
) *orquestaweb.RESTDirectorStatsClientV0 {
	out := orquestaweb.NewRESTDirectorStatsClientV0(InternalBaseURLV0, config.Timeout)
	out.HTTPClient = client
	return out
}

func newRunQueueClientV0(
	config ConfigV0,
	client *http.Client,
) *orquestaweb.RESTRunQueueClientV0 {
	out := orquestaweb.NewRESTRunQueueClientV0(InternalBaseURLV0, config.Timeout)
	out.HTTPClient = client
	return out
}

func newAppChangeClientV0(
	config ConfigV0,
	client *http.Client,
) *orquestaweb.RESTAppChangeClientV0 {
	out := orquestaweb.NewRESTAppChangeClientV0(InternalBaseURLV0, config.Timeout)
	out.HTTPClient = client
	return out
}
