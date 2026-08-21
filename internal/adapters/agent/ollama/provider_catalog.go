// Package ollama translates Ollama's read-only model discovery into the
// neutral provider catalog. It never starts, stops, serves or pulls models.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"orquesta/internal/adapters/agent/localhttp"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

const ProviderRef = "provider:ollama"

var (
	ErrProviderCatalogInvalid          = errors.New("ollama.provider_catalog_invalid")
	ErrProviderCatalogUnavailable      = errors.New("ollama.provider_catalog_unavailable")
	ErrProviderCatalogResponseTooLarge = errors.New("ollama.provider_catalog_response_too_large")
)

// ProviderCatalogConfig contains only bounded, non-secret discovery inputs.
// BaseURL must identify an explicit loopback endpoint.
type ProviderCatalogConfig struct {
	BaseURL          string
	Now              func() time.Time
	Freshness        time.Duration
	RequestTimeout   time.Duration
	MaxResponseBytes int64
}

type ProviderCatalogSource struct {
	endpoint         *localhttp.Endpoint
	now              func() time.Time
	freshness        time.Duration
	maxResponseBytes int64
}

func NewProviderCatalogSource(config ProviderCatalogConfig) (*ProviderCatalogSource, error) {
	if config.Now == nil || config.Freshness <= 0 || config.RequestTimeout <= 0 ||
		config.MaxResponseBytes <= 0 || config.MaxResponseBytes > localhttp.MaxResponseBodyBytes {
		return nil, ErrProviderCatalogInvalid
	}
	endpoint, err := localhttp.NewEndpoint(config.BaseURL, "/api/tags", config.RequestTimeout)
	if err != nil {
		return nil, errors.Join(ErrProviderCatalogInvalid, err)
	}
	return &ProviderCatalogSource{
		endpoint:         endpoint,
		now:              config.Now,
		freshness:        config.Freshness,
		maxResponseBytes: config.MaxResponseBytes,
	}, nil
}

func (source *ProviderCatalogSource) ProviderRef() string {
	if source == nil {
		return ""
	}
	return ProviderRef
}

func (source *ProviderCatalogSource) ObserveProviderCatalog(ctx context.Context) (ports.ProviderCatalogObservation, error) {
	if source == nil || ctx == nil {
		return ports.ProviderCatalogObservation{}, ErrProviderCatalogUnavailable
	}
	if err := ctx.Err(); err != nil {
		return ports.ProviderCatalogObservation{}, err
	}
	observedAt, expiresAt, err := observationWindow(source.now(), source.freshness)
	if err != nil {
		return ports.ProviderCatalogObservation{}, err
	}
	response, err := source.endpoint.Get(ctx)
	if err != nil {
		return ports.ProviderCatalogObservation{}, mapHTTPError(err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ports.ProviderCatalogObservation{}, fmt.Errorf("%w: http_status_%d", ErrProviderCatalogUnavailable, response.StatusCode)
	}
	payload, err := localhttp.ReadResponseBody(ctx, response, source.maxResponseBytes)
	if err != nil {
		return ports.ProviderCatalogObservation{}, mapHTTPError(err)
	}
	models, err := decodeModels(payload)
	if err != nil {
		return ports.ProviderCatalogObservation{}, err
	}
	completedAt := source.now().Round(0).UTC()
	if completedAt.Before(observedAt) || !completedAt.Before(expiresAt) {
		return ports.ProviderCatalogObservation{}, ErrProviderCatalogInvalid
	}
	observation := ports.ProviderCatalogObservation{
		ProviderRef:  ProviderRef,
		Models:       models,
		Availability: ports.ProviderAvailabilityUnknown,
		Quota:        ports.ProviderQuotaUnknown,
		Usage:        governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
		ObservedAt:   observedAt,
		ExpiresAt:    expiresAt,
	}
	if err := ports.ValidateProviderCatalogObservation(observation); err != nil {
		return ports.ProviderCatalogObservation{}, errors.Join(ErrProviderCatalogInvalid, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.ProviderCatalogObservation{}, err
	}
	return observation, nil
}

func decodeModels(payload []byte) ([]ports.ProviderModel, error) {
	var tags struct {
		Models []ollamaTag `json:"models"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&tags); err != nil || tags.Models == nil {
		return nil, errors.Join(ErrProviderCatalogInvalid, err)
	}
	if err := requireJSONEnd(decoder); err != nil {
		return nil, errors.Join(ErrProviderCatalogInvalid, err)
	}
	models := make([]ports.ProviderModel, 0, len(tags.Models))
	seen := make(map[string]struct{}, len(tags.Models))
	for _, discovered := range tags.Models {
		model := ports.ProviderModel{ProviderRef: ProviderRef, ModelRef: discovered.Name}
		if _, duplicate := seen[discovered.Name]; duplicate {
			return nil, ErrProviderCatalogInvalid
		}
		if err := ports.ValidateProviderModel(model); err != nil {
			return nil, errors.Join(ErrProviderCatalogInvalid, err)
		}
		seen[discovered.Name] = struct{}{}
		models = append(models, model)
	}
	return models, nil
}

func mapHTTPError(err error) error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, localhttp.ErrResponseTooLarge):
		return errors.Join(ErrProviderCatalogResponseTooLarge, err)
	case errors.Is(err, localhttp.ErrEndpointInvalid):
		return errors.Join(ErrProviderCatalogInvalid, err)
	default:
		return errors.Join(ErrProviderCatalogUnavailable, err)
	}
}

func observationWindow(now time.Time, freshness time.Duration) (time.Time, time.Time, error) {
	observedAt := now.Round(0).UTC()
	expiresAt := observedAt.Add(freshness)
	if observedAt.IsZero() || freshness <= 0 || !expiresAt.After(observedAt) || expiresAt.Year() > 9999 {
		return time.Time{}, time.Time{}, ErrProviderCatalogInvalid
	}
	return observedAt, expiresAt, nil
}

// ollamaTag declares the documented /api/tags shape while deliberately
// ignoring descriptive values. None proves capability or reasoning effort.
type ollamaTag struct {
	Name       string           `json:"name"`
	Model      string           `json:"model"`
	ModifiedAt string           `json:"modified_at"`
	Size       int64            `json:"size"`
	Digest     string           `json:"digest"`
	Details    ollamaTagDetails `json:"details"`
}

type ollamaTagDetails struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

func requireJSONEnd(decoder *json.Decoder) error {
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("ollama.provider_catalog_trailing_json")
		}
		return err
	}
	return nil
}
