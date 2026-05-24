package orquestaserver

import (
	"net/http"
	"strconv"
	"time"
)

type auditResponseWriterV0 struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (writer *auditResponseWriterV0) WriteHeader(status int) {
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *auditResponseWriterV0) Write(data []byte) (int, error) {
	if writer.status == 0 {
		writer.status = http.StatusOK
	}
	n, err := writer.ResponseWriter.Write(data)
	writer.bytes += n
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
		runtime.auditEventV0(r.Context(), "http_request", strconv.Itoa(status), "", map[string]interface{}{
			"method":          r.Method,
			"path":            r.URL.Path,
			"raw_query":       r.URL.RawQuery,
			"remote_addr":     r.RemoteAddr,
			"status":          status,
			"bytes":           recorder.bytes,
			"content_length":  r.ContentLength,
			"duration_millis": time.Since(start).Milliseconds(),
		})
	})
}
