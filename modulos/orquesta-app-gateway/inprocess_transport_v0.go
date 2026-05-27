package orquestaappgateway

import (
	"net/http"

	"orquesta/modulos/orquesta-app-gateway/inprocesshttp"
)

var ErrInProcessTransportUnconfiguredV0 = inprocesshttp.ErrUnconfiguredV0

type InProcessTransportErrorV0 = inprocesshttp.PublicErrorV0

type InProcessTransportV0 struct {
	Handler          http.Handler
	MaxResponseBytes int64
}

func (transport InProcessTransportV0) RoundTrip(req *http.Request) (*http.Response, error) {
	return inprocesshttp.TransportV0{
		Handler:          transport.Handler,
		MaxResponseBytes: transport.MaxResponseBytes,
	}.RoundTrip(req)
}
