package orquestaserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

const (
	DefaultHTTPReadHeaderTimeoutV0 = 5 * time.Second
	DefaultHTTPReadTimeoutV0       = 20 * time.Second
	// Control-plane requests can supervise real agents; 30s cuts valid work before ACK.
	DefaultHTTPWriteTimeoutV0     = 10 * time.Minute
	DefaultHTTPIdleTimeoutV0      = 60 * time.Second
	DefaultHTTPMaxHeaderBytesV0   = 1 << 20
	DefaultHTTPControlBodyBytesV0 = 256 << 10
)

type HTTPResourceLimitsV0 struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	ControlBodyBytes  int64
}

func NormalizeHTTPResourceLimitsV0(limits HTTPResourceLimitsV0) HTTPResourceLimitsV0 {
	if limits.ReadHeaderTimeout <= 0 {
		limits.ReadHeaderTimeout = DefaultHTTPReadHeaderTimeoutV0
	}
	if limits.ReadTimeout <= 0 {
		limits.ReadTimeout = DefaultHTTPReadTimeoutV0
	}
	if limits.WriteTimeout <= 0 {
		limits.WriteTimeout = DefaultHTTPWriteTimeoutV0
	}
	if limits.IdleTimeout <= 0 {
		limits.IdleTimeout = DefaultHTTPIdleTimeoutV0
	}
	if limits.MaxHeaderBytes <= 0 {
		limits.MaxHeaderBytes = DefaultHTTPMaxHeaderBytesV0
	}
	if limits.ControlBodyBytes <= 0 {
		limits.ControlBodyBytes = DefaultHTTPControlBodyBytesV0
	}
	return limits
}

func (runtime *RuntimeV0) httpServerV0() *http.Server {
	limits := NormalizeHTTPResourceLimitsV0(runtime.config.HTTPResourceLimits)
	return &http.Server{
		Handler:           runtime.HandlerV0(),
		ReadHeaderTimeout: limits.ReadHeaderTimeout,
		ReadTimeout:       limits.ReadTimeout,
		WriteTimeout:      limits.WriteTimeout,
		IdleTimeout:       limits.IdleTimeout,
		MaxHeaderBytes:    limits.MaxHeaderBytes,
	}
}

func decodeServerControlJSONV0(w http.ResponseWriter, r *http.Request, dst any) string {
	limits := NormalizeHTTPResourceLimitsV0(HTTPResourceLimitsV0{})
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limits.ControlBodyBytes))
	if err := decoder.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return "request_body_too_large"
		}
		return "request_body_invalido"
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return "request_body_trailing_data"
	}
	return ""
}
