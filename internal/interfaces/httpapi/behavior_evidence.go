package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"orquesta/internal/application"
	"orquesta/internal/identity"
)

const (
	BehaviorEvidenceManifestPath          = "/api/v1/application-behavior-evidence/manifests"
	BehaviorEvidenceMaxRequestBytes int64 = 64 * 1024
)

type BehaviorEvidenceIngestor interface {
	IngestBehaviorEvidenceManifest(
		context.Context,
		application.BehaviorEvidenceManifest,
	) (application.BehaviorEvidenceReceipt, error)
}

type BehaviorEvidenceConfig struct {
	Ingestor BehaviorEvidenceIngestor
	Identity identity.Provider
}

type BehaviorEvidenceHandler struct {
	ingestor BehaviorEvidenceIngestor
	identity identity.Provider
}

type behaviorEvidenceFailure struct {
	Code string `json:"code"`
}

func NewBehaviorEvidenceHandler(
	config BehaviorEvidenceConfig,
) (*BehaviorEvidenceHandler, error) {
	if config.Ingestor == nil || config.Identity == nil {
		return nil, errors.New("httpapi.behavior_evidence_config_invalid")
	}
	return &BehaviorEvidenceHandler{
		ingestor: config.Ingestor,
		identity: config.Identity,
	}, nil
}

func (handler *BehaviorEvidenceHandler) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set("Content-Type", "application/json")
	if handler == nil || handler.ingestor == nil || handler.identity == nil ||
		request == nil {
		writeBehaviorEvidenceFailure(
			writer,
			http.StatusServiceUnavailable,
			"behavior_evidence.unavailable",
		)
		return
	}
	if request.Method != http.MethodPost ||
		request.URL.Path != BehaviorEvidenceManifestPath {
		writeBehaviorEvidenceFailure(
			writer,
			http.StatusNotFound,
			"behavior_evidence.not_found",
		)
		return
	}
	body, err := io.ReadAll(
		io.LimitReader(request.Body, BehaviorEvidenceMaxRequestBytes+1),
	)
	if err != nil || int64(len(body)) > BehaviorEvidenceMaxRequestBytes {
		writeBehaviorEvidenceFailure(
			writer,
			http.StatusBadRequest,
			"behavior_evidence.invalid_request",
		)
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var manifest application.BehaviorEvidenceManifest
	if err := decoder.Decode(&manifest); err != nil ||
		decoder.Decode(&struct{}{}) != io.EOF ||
		application.ValidateBehaviorEvidenceManifest(manifest) != nil {
		writeBehaviorEvidenceFailure(
			writer,
			http.StatusBadRequest,
			"behavior_evidence.invalid_manifest",
		)
		return
	}
	if _, err := handler.identity.Principal(request.Context()); err != nil {
		writeBehaviorEvidenceFailure(
			writer,
			http.StatusUnauthorized,
			"behavior_evidence.unauthenticated",
		)
		return
	}
	receipt, err := handler.ingestor.IngestBehaviorEvidenceManifest(
		request.Context(),
		manifest,
	)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrBehaviorEvidenceInvalid):
			writeBehaviorEvidenceFailure(
				writer,
				http.StatusBadRequest,
				"behavior_evidence.invalid_manifest",
			)
		case errors.Is(err, application.ErrBehaviorEvidenceConflict):
			writeBehaviorEvidenceFailure(
				writer,
				http.StatusConflict,
				"behavior_evidence.conflict",
			)
		default:
			writeBehaviorEvidenceFailure(
				writer,
				http.StatusServiceUnavailable,
				"behavior_evidence.unavailable",
			)
		}
		return
	}
	writer.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(writer).Encode(receipt)
}

func writeBehaviorEvidenceFailure(
	writer http.ResponseWriter,
	status int,
	code string,
) {
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(behaviorEvidenceFailure{Code: code})
}
