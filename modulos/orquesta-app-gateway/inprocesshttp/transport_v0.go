package inprocesshttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
)

const (
	DefaultMaxResponseBytesV0 = int64(1 << 20)

	CodeCancelledV0        = "inprocess_cancelled"
	CodeHandlerPanicV0     = "inprocess_handler_panic"
	CodeResponseTooLargeV0 = "inprocess_response_too_large"
	CodeTimeoutV0          = "inprocess_timeout"
	CodeWriteClosedV0      = "inprocess_write_closed"
)

var ErrUnconfiguredV0 = errors.New("orquesta_app_gateway_inprocess_transport_unconfigured")

type PublicErrorV0 struct {
	Code string
}

func (err PublicErrorV0) Error() string {
	if err.Code == "" {
		return "inprocess_error"
	}
	return err.Code
}

type TransportV0 struct {
	Handler          http.Handler
	MaxResponseBytes int64
}

func (transport TransportV0) RoundTrip(req *http.Request) (*http.Response, error) {
	if transport.Handler == nil {
		return nil, ErrUnconfiguredV0
	}
	if req.Body != nil {
		defer req.Body.Close()
	}
	if err := req.Context().Err(); err != nil {
		return nil, contextPublicErrorV0(err)
	}

	recorder := newRecorderV0(maxResponseBytesV0(transport.MaxResponseBytes))
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- serveAndRecoverV0(transport.Handler, recorder, req)
	}()

	select {
	case err := <-resultCh:
		if err != nil {
			return nil, err
		}
		return recorder.responseV0(req), nil
	case <-req.Context().Done():
		err := contextPublicErrorV0(req.Context().Err())
		recorder.closeWithErrorV0(err)
		return nil, err
	}
}

func serveAndRecoverV0(handler http.Handler, recorder *recorderV0, req *http.Request) (err error) {
	defer func() {
		if recover() != nil {
			err = PublicErrorV0{Code: CodeHandlerPanicV0}
			recorder.closeWithErrorV0(err)
		}
	}()
	handler.ServeHTTP(recorder, req)
	return recorder.errV0()
}

func maxResponseBytesV0(value int64) int64 {
	if value <= 0 {
		return DefaultMaxResponseBytesV0
	}
	return value
}

func contextPublicErrorV0(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return PublicErrorV0{Code: CodeTimeoutV0}
	}
	return PublicErrorV0{Code: CodeCancelledV0}
}

type recorderV0 struct {
	mutex sync.Mutex
	max   int64

	header http.Header
	status int
	body   bytes.Buffer
	err    error
	closed bool
}

func newRecorderV0(max int64) *recorderV0 {
	return &recorderV0{max: max, header: http.Header{}}
}

func (recorder *recorderV0) Header() http.Header {
	return recorder.header
}

func (recorder *recorderV0) WriteHeader(status int) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	if recorder.status == 0 {
		recorder.status = status
	}
}

func (recorder *recorderV0) Write(data []byte) (int, error) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	if recorder.closed {
		return 0, recorder.setErrLockedV0(PublicErrorV0{Code: CodeWriteClosedV0})
	}
	if recorder.status == 0 {
		recorder.status = http.StatusOK
	}
	remaining := recorder.max - int64(recorder.body.Len())
	if int64(len(data)) > remaining {
		if remaining > 0 {
			_, _ = recorder.body.Write(data[:int(remaining)])
		}
		return 0, recorder.setErrLockedV0(PublicErrorV0{Code: CodeResponseTooLargeV0})
	}
	n, err := recorder.body.Write(data)
	if err != nil {
		return n, recorder.setErrLockedV0(PublicErrorV0{Code: CodeWriteClosedV0})
	}
	return n, nil
}

func (recorder *recorderV0) errV0() error {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	return recorder.err
}

func (recorder *recorderV0) closeWithErrorV0(err error) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	recorder.closed = true
	recorder.setErrLockedV0(err)
}

func (recorder *recorderV0) setErrLockedV0(err error) error {
	if recorder.err == nil {
		recorder.err = err
	}
	return recorder.err
}

func (recorder *recorderV0) responseV0(req *http.Request) *http.Response {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	status := recorder.status
	if status == 0 {
		status = http.StatusOK
	}
	body := append([]byte(nil), recorder.body.Bytes()...)
	return &http.Response{
		StatusCode:    status,
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:        recorder.header.Clone(),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}
