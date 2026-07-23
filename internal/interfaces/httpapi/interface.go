// Package httpapi projects the canonical command registry over HTTP.
package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/identity"
)

type commandBinding struct {
	CommandID      string
	Version        string
	Path           string
	ExecutionBound bool
}

type Config struct {
	Dispatcher commandcore.Executor
	Identity   identity.Provider
}

type Handler struct {
	dispatcher commandcore.Executor
	identity   identity.Provider
	maximum    int64
	bindings   map[string]commandBinding
}

type Request struct {
	Version             string          `json:"version"`
	RequestRef          string          `json:"request_ref"`
	ProjectRef          string          `json:"project_ref"`
	ClaimedExecutionRef string          `json:"claimed_execution_ref,omitempty"`
	Payload             json.RawMessage `json:"payload"`
}

func New(config Config) (*Handler, error) {
	if config.Dispatcher == nil || config.Identity == nil || !config.Dispatcher.Limits().Valid() {
		return nil, errors.New("httpapi.config_invalid")
	}
	definitions := commandcore.CanonicalDefinitions()
	bindings := make(map[string]commandBinding, len(definitions))
	for _, definition := range definitions {
		if definition.ID == "" || definition.Version == "" || definition.HTTP.Method != http.MethodPost ||
			definition.HTTP.Path == "" {
			return nil, errors.New("httpapi.binding_registry_invalid")
		}
		binding := commandBinding{
			CommandID: definition.ID, Version: definition.Version, Path: definition.HTTP.Path,
			ExecutionBound: definition.ExecutionBound,
		}
		if _, duplicate := bindings[binding.Path]; duplicate {
			return nil, errors.New("httpapi.binding_duplicate")
		}
		bindings[binding.Path] = binding
	}
	return &Handler{
		dispatcher: config.Dispatcher, identity: config.Identity,
		maximum: config.Dispatcher.Limits().MaxRequestBytes, bindings: bindings,
	}, nil
}

func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	if handler == nil || request == nil {
		writeResult(writer, http.StatusServiceUnavailable, commandcore.Result{Failure: &commandcore.Failure{Code: commandcore.CodeUnavailable, MessageKey: "error.unavailable"}})
		return
	}
	binding, ok := handler.bindings[request.URL.Path]
	if !ok || request.Method != http.MethodPost {
		writeResult(writer, http.StatusNotFound, commandcore.Result{Failure: &commandcore.Failure{Code: commandcore.CodeNotFound, MessageKey: "error.not_found"}})
		return
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, handler.maximum+1))
	if err != nil || int64(len(body)) > handler.maximum {
		writeResult(writer, http.StatusBadRequest, commandcore.Result{CommandID: binding.CommandID, CommandVersion: binding.Version, Failure: &commandcore.Failure{Code: commandcore.CodeInvalidRequest, MessageKey: "error.invalid_request"}})
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var input Request
	if err := decoder.Decode(&input); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeResult(writer, http.StatusBadRequest, commandcore.Result{CommandID: binding.CommandID, CommandVersion: binding.Version, Failure: &commandcore.Failure{Code: commandcore.CodeInvalidRequest, MessageKey: "error.invalid_request"}})
		return
	}
	if input.Version != binding.Version || input.RequestRef == "" || input.ProjectRef == "" || len(input.Payload) == 0 {
		writeResult(writer, http.StatusBadRequest, commandcore.Result{CommandID: binding.CommandID, CommandVersion: binding.Version, Failure: &commandcore.Failure{Code: commandcore.CodeInvalidRequest, MessageKey: "error.invalid_request"}})
		return
	}
	if binding.ExecutionBound == (input.ClaimedExecutionRef == "") {
		writeResult(writer, http.StatusBadRequest, commandcore.Result{CommandID: binding.CommandID, CommandVersion: input.Version, RequestRef: input.RequestRef, Failure: &commandcore.Failure{Code: commandcore.CodeInvalidRequest, MessageKey: "error.invalid_request"}})
		return
	}
	principal, err := handler.identity.Principal(request.Context())
	if err != nil {
		writeResult(writer, http.StatusUnauthorized, commandcore.Result{CommandID: binding.CommandID, CommandVersion: input.Version, RequestRef: input.RequestRef, Failure: &commandcore.Failure{Code: commandcore.CodeUnauthenticated, MessageKey: "error.unauthenticated"}})
		return
	}
	result := handler.dispatcher.Dispatch(request.Context(), commandcore.Invocation{
		CommandID: binding.CommandID, CommandVersion: input.Version, RequestRef: input.RequestRef,
		ProjectRef: input.ProjectRef, ClaimedExecutionRef: input.ClaimedExecutionRef,
		Principal: principal, Payload: input.Payload,
	})
	writeResult(writer, statusFor(result.Failure), result)
}

func writeResult(writer http.ResponseWriter, status int, result commandcore.Result) {
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(result)
}

func statusFor(failure *commandcore.Failure) int {
	if failure == nil {
		return http.StatusOK
	}
	switch failure.Code {
	case commandcore.CodeInvalidRequest:
		return http.StatusBadRequest
	case commandcore.CodeUnauthenticated:
		return http.StatusUnauthorized
	case commandcore.CodeForbidden:
		return http.StatusForbidden
	case commandcore.CodeNotFound:
		return http.StatusNotFound
	case commandcore.CodeConflict:
		return http.StatusConflict
	case commandcore.CodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
