package orquestaserver

import (
	"context"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	publicidentity "orquesta/modulos/orquesta-server/publicidentity"
)

const (
	auditQueryRawMaxBytesV0 = 8 << 10
	auditQueryMaxKeysV0     = 24
	auditQueryMaxKeyBytesV0 = 96
)

const (
	auditClientIdentitySchemaV0    = "orquesta_http_client_identity.v0"
	auditClientIdentityRedactionV0 = "category_only"
)

type auditResponseWriterV0 struct {
	http.ResponseWriter
	status      int
	bytes       int
	observation serverHTTPResponseObservationV0
}

func (writer *auditResponseWriterV0) WriteHeader(status int) {
	if writer.status != 0 {
		return
	}
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *auditResponseWriterV0) Write(data []byte) (int, error) {
	wroteBeforeHeader := writer.status == 0
	if writer.status == 0 {
		writer.status = http.StatusOK
	}
	n, err := writer.ResponseWriter.Write(data)
	writer.bytes += n
	if err != nil {
		writer.observation = serverHTTPResponseObservationFromWriteV0(err, wroteBeforeHeader)
	} else if n < len(data) {
		writer.observation = serverHTTPResponseObservationFromShortWriteV0(wroteBeforeHeader)
	}
	return n, err
}

func (runtime *RuntimeV0) auditHTTPHandlerV0(next http.Handler) http.Handler {
	if next == nil || runtime == nil || runtime.auditSink == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &auditResponseWriterV0{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		auditStatus := strconv.Itoa(status)
		if !recorder.observation.OK && recorder.observation.Code != "" {
			auditStatus = recorder.observation.Code
			if runtime.tracker != nil {
				runtime.persistStateTransitionV0(
					context.Background(),
					runtime.tracker.MarkResponseWriteFailedV0(recorder.observation.Stage, runtime.clock.Now()),
					"response_write_failed",
				)
			}
		}
		runtime.auditEventV0(r.Context(), "http_request", auditStatus, "", map[string]interface{}{
			"method":          r.Method,
			"endpoint":        r.URL.Path,
			"path":            r.URL.Path,
			"query_keys":      auditQueryKeysV0(r),
			"correlation_id":  compactControlPlaneIDV0(r.Header.Get(publicidentity.PublicCorrelationHeaderV0)),
			"client_identity": auditClientIdentityPayloadV0(r, runtime.config),
			"status":          status,
			"response_status": auditStatus,
			"response_write":  responseWriteAuditPayloadV0(recorder.observation),
			"bytes":           recorder.bytes,
			"content_length":  r.ContentLength,
			"duration_millis": time.Since(start).Milliseconds(),
		})
	})
}

func responseWriteAuditPayloadV0(observation serverHTTPResponseObservationV0) map[string]interface{} {
	if observation.OK || observation.Code == "" {
		return map[string]interface{}{"status": "ok"}
	}
	return map[string]interface{}{
		"status": "failed",
		"code":   observation.Code,
		"stage":  compactResponseWriteStageV0(observation.Stage),
	}
}

func auditQueryKeysV0(r *http.Request) []string {
	if r == nil || r.URL == nil || r.URL.RawQuery == "" {
		return []string{}
	}
	if len(r.URL.RawQuery) > auditQueryRawMaxBytesV0 {
		return []string{"query_too_large"}
	}
	query := r.URL.Query()
	keys := make([]string, 0, len(query))
	for key := range query {
		if publicKey := auditPublicQueryKeyV0(key); publicKey != "" {
			keys = append(keys, publicKey)
		}
	}
	sort.Strings(keys)
	if len(keys) > auditQueryMaxKeysV0 {
		keys = append(keys[:auditQueryMaxKeysV0], "query_keys_truncated")
	}
	return keys
}

func auditPublicQueryKeyV0(key string) string {
	key = compactControlPlaneIDV0(key)
	if key == "" {
		return ""
	}
	if len(key) > auditQueryMaxKeyBytesV0 {
		return "query_key_too_large"
	}
	lower := strings.ToLower(key)
	for _, sensitive := range []string{"token", "secret", "credential", "prompt", "raw_q", "raw_query", "url", "path"} {
		if strings.Contains(lower, sensitive) {
			return "sensitive_query_key_redacted"
		}
	}
	return key
}

func auditClientIdentityPayloadV0(r *http.Request, config ConfigV0) map[string]interface{} {
	payload := map[string]interface{}{
		"schema_version":          auditClientIdentitySchemaV0,
		"source":                  "connection",
		"category":                auditRemoteAddrCategoryV0(remoteAddrHostV0(r)),
		"redaction":               auditClientIdentityRedactionV0,
		"raw_address_persisted":   false,
		"forwarded_header_policy": "ignored_untrusted",
		"authorization_scope":     auditAuthorizationScopeV0(config),
	}
	if headers := auditForwardedHeaderNamesV0(r); len(headers) > 0 {
		payload["forwarded_headers_present"] = headers
	}
	return payload
}

func remoteAddrHostV0(r *http.Request) string {
	if r == nil {
		return ""
	}
	remote := strings.TrimSpace(r.RemoteAddr)
	if remote == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(remote)
	if err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(remote, "[]")
}

func auditRemoteAddrCategoryV0(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "unknown"
	}
	if strings.EqualFold(host, "localhost") {
		return "loopback"
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "unknown"
	}
	switch {
	case ip.IsLoopback():
		return "loopback"
	case ip.IsPrivate():
		return "private"
	default:
		return "external"
	}
}

func auditForwardedHeaderNamesV0(r *http.Request) []string {
	if r == nil {
		return []string{}
	}
	names := make([]string, 0, 4)
	for _, name := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Real-IP"} {
		if strings.TrimSpace(r.Header.Get(name)) != "" {
			names = append(names, http.CanonicalHeaderKey(name))
		}
	}
	return names
}

func auditAuthorizationScopeV0(config ConfigV0) string {
	config = NormalizeConfigV0(config)
	if controlPlaneAddrIsLoopbackV0(config.Addr) {
		return "loopback_bind"
	}
	if config.ControlPlane.RemoteAccessOptIn {
		return "remote_bind_opt_in"
	}
	return "remote_bind_unconfirmed"
}
