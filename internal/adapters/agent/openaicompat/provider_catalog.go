// Package openaicompat translates local OpenAI-compatible model discovery
// into Orquesta's neutral, read-only provider catalog contract.
package openaicompat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"orquesta/internal/adapters/agent/localhttp"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

const ProviderRef = "provider:openaicompat"

var (
	ErrProviderCatalogInvalid     = errors.New("openaicompat.provider_catalog_invalid")
	ErrProviderCatalogUnavailable = errors.New("openaicompat.provider_catalog_unavailable")
	ErrProviderCatalogTooLarge    = errors.New("openaicompat.provider_catalog_too_large")
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
	endpoint, err := localhttp.NewEndpoint(config.BaseURL, "/v1/models", config.RequestTimeout)
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
		return ports.ProviderCatalogObservation{}, ErrProviderCatalogUnavailable
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

type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func decodeModels(payload []byte) ([]ports.ProviderModel, error) {
	var decoded modelsResponse
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&decoded); err != nil || decoded.Data == nil {
		return nil, ErrProviderCatalogInvalid
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, ErrProviderCatalogInvalid
	}
	models := make([]ports.ProviderModel, 0, len(decoded.Data))
	seen := make(map[string]struct{}, len(decoded.Data))
	for _, discovered := range decoded.Data {
		model := ports.ProviderModel{ProviderRef: ProviderRef, ModelRef: discovered.ID}
		if _, duplicate := seen[discovered.ID]; duplicate {
			return nil, ErrProviderCatalogInvalid
		}
		if err := ports.ValidateProviderModel(model); err != nil {
			return nil, errors.Join(ErrProviderCatalogInvalid, err)
		}
		seen[discovered.ID] = struct{}{}
		models = append(models, model)
	}
	slices.SortFunc(models, func(left, right ports.ProviderModel) int {
		return strings.Compare(left.ModelRef, right.ModelRef)
	})
	return models, nil
}

func mapHTTPError(err error) error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, localhttp.ErrResponseTooLarge):
		return errors.Join(ErrProviderCatalogTooLarge, err)
	case errors.Is(err, localhttp.ErrEndpointInvalid):
		return errors.Join(ErrProviderCatalogInvalid, err)
	default:
		return errors.Join(ErrProviderCatalogUnavailable, err)
	}
}

func requireJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return ErrProviderCatalogInvalid
}

func observationWindow(now time.Time, freshness time.Duration) (time.Time, time.Time, error) {
	observedAt := now.Round(0).UTC()
	expiresAt := observedAt.Add(freshness)
	if observedAt.IsZero() || freshness <= 0 || !expiresAt.After(observedAt) || expiresAt.Year() > 9999 {
		return time.Time{}, time.Time{}, ErrProviderCatalogInvalid
	}
	return observedAt, expiresAt, nil
}
