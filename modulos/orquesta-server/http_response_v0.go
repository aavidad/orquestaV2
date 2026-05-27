package orquestaserver

import (
	"encoding/json"
	"errors"
	"net/http"
)

const (
	serverResponseEncodeFailedCodeV0 = "response_encode_failed"
	serverResponseWriteFailedCodeV0  = "response_write_failed"
)

type serverHTTPResponseObservationV0 struct {
	OK    bool
	Code  string
	Stage string
}

func writeServerJSONResponseV0(w http.ResponseWriter, status int, value any) serverHTTPResponseObservationV0 {
	body, err := json.Marshal(value)
	if err != nil {
		return writeServerJSONFallbackV0(w)
	}
	body = append(body, '\n')
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if n, err := w.Write(body); err != nil {
		return serverHTTPResponseObservationV0{
			Code:  serverResponseWriteFailedCodeV0,
			Stage: "body_after_header",
		}
	} else if n < len(body) {
		return serverHTTPResponseObservationFromShortWriteV0(false)
	}
	return serverHTTPResponseObservationV0{OK: true}
}

func writeServerJSONFallbackV0(w http.ResponseWriter) serverHTTPResponseObservationV0 {
	const fallback = "{\"error\":\"response_encode_failed\"}\n"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	observation := serverHTTPResponseObservationV0{
		Code:  serverResponseEncodeFailedCodeV0,
		Stage: "encode_before_header",
	}
	if _, err := w.Write([]byte(fallback)); err != nil {
		observation.Code = serverResponseWriteFailedCodeV0
		observation.Stage = "fallback_body_after_header"
	}
	return observation
}

func serverHTTPResponseObservationFromWriteV0(err error, wroteBeforeHeader bool) serverHTTPResponseObservationV0 {
	if err == nil {
		return serverHTTPResponseObservationV0{OK: true}
	}
	stage := "body_after_header"
	if wroteBeforeHeader {
		stage = "body_before_header"
	}
	return serverHTTPResponseObservationV0{
		Code:  serverResponseWriteFailedCodeV0,
		Stage: stage,
	}
}

func serverHTTPResponseObservationFromShortWriteV0(wroteBeforeHeader bool) serverHTTPResponseObservationV0 {
	return serverHTTPResponseObservationFromWriteV0(errors.New(serverResponseWriteFailedCodeV0), wroteBeforeHeader)
}
