package orquestaappgateway

import (
	"bytes"
	"errors"
	"io"
	"net/http"
)

var ErrInProcessTransportUnconfiguredV0 = errors.New("orquesta_app_gateway_inprocess_transport_unconfigured")

type InProcessTransportV0 struct {
	Handler http.Handler
}

func (transport InProcessTransportV0) RoundTrip(req *http.Request) (*http.Response, error) {
	if transport.Handler == nil {
		return nil, ErrInProcessTransportUnconfiguredV0
	}
	if req.Body != nil {
		defer req.Body.Close()
	}

	recorder := newInProcessRecorderV0()
	transport.Handler.ServeHTTP(recorder, req)
	return recorder.responseV0(req), nil
}

type inProcessRecorderV0 struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newInProcessRecorderV0() *inProcessRecorderV0 {
	return &inProcessRecorderV0{header: http.Header{}}
}

func (recorder *inProcessRecorderV0) Header() http.Header {
	return recorder.header
}

func (recorder *inProcessRecorderV0) WriteHeader(status int) {
	if recorder.status != 0 {
		return
	}
	recorder.status = status
}

func (recorder *inProcessRecorderV0) Write(data []byte) (int, error) {
	if recorder.status == 0 {
		recorder.status = http.StatusOK
	}
	return recorder.body.Write(data)
}

func (recorder *inProcessRecorderV0) responseV0(req *http.Request) *http.Response {
	status := recorder.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     recorder.header.Clone(),
		Body:       io.NopCloser(bytes.NewReader(recorder.body.Bytes())),
		Request:    req,
	}
}
