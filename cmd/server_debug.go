package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/internal/rpclocal"
)

type serverDebugOptions struct {
	Enabled      bool
	HTTP         bool
	ControlPlane bool
}

func resolveServerDebugOptions(cmd *cobra.Command) serverDebugOptions {
	base := boolFlagOrEnv(cmd, "debug", "ORQUESTA_DEBUG", false)
	httpEnabled := boolFlagOrEnv(cmd, "debug-http", "ORQUESTA_DEBUG_HTTP", base)
	controlEnabled := boolFlagOrEnv(cmd, "debug-control-plane", "ORQUESTA_DEBUG_CONTROL_PLANE", base)
	return serverDebugOptions{
		Enabled:      base || httpEnabled || controlEnabled,
		HTTP:         httpEnabled,
		ControlPlane: controlEnabled,
	}
}

func boolFlagOrEnv(cmd *cobra.Command, name, env string, fallback bool) bool {
	if cmd != nil {
		if binding := cmd.Flags().Lookup(name); binding != nil && binding.Changed {
			value, _ := cmd.Flags().GetBool(name)
			return value
		}
	}
	if raw, ok := os.LookupEnv(env); ok {
		return parseBoolDebug(raw, fallback)
	}
	return fallback
}

func parseBoolDebug(raw string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "on", "si", "sí":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func newServerDebugLogger(kind string) *log.Logger {
	prefix := fmt.Sprintf("orquesta[%s][debug] ", strings.TrimSpace(kind))
	return log.New(os.Stdout, prefix, log.Ldate|log.Ltime|log.Lmicroseconds)
}

func wrapServeMuxWithDebug(next http.Handler, opts serverDebugOptions, logger *log.Logger) http.Handler {
	if next == nil {
		return http.NewServeMux()
	}
	if !opts.HTTP || logger == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusCapturingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		logger.Printf("http method=%s path=%s query=%q status=%d duration=%s remote=%s ua=%q",
			r.Method, r.URL.Path, r.URL.RawQuery, ww.status, time.Since(start).Round(time.Millisecond), r.RemoteAddr, r.UserAgent())
	})
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusCapturingResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func debugDurationsSummary(reanimacionCada, saludCada, planificacionCada, controlPlaneCada time.Duration) string {
	return strings.Join([]string{
		"reanimacion=" + reanimacionCada.String(),
		"salud=" + saludCada.String(),
		"planificacion=" + planificacionCada.String(),
		"control_plane=" + controlPlaneCada.String(),
	}, " ")
}

func localServerLogPath() string {
	return strings.TrimSuffix(rpclocal.DefaultInfoPath(), filepath.Ext(rpclocal.DefaultInfoPath())) + ".log"
}
