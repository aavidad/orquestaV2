// Este conector traduce cuota Codex estructurada; no decide admisión ni persistencia.
package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

type CapacityObserver struct {
	ReportPath string
	MaxBytes   int64
	SourceRef  application.AgentCapacitySourceRef
	PoolRef    application.AgentCapacityPoolRef
}
type capacityReportDimension struct {
	Governs   *bool  `json:"governs_admission"`
	Unit      string `json:"unit,omitempty"`
	Limit     *int64 `json:"limit,omitempty"`
	Remaining *int64 `json:"remaining,omitempty"`
}
type capacityReportLimits struct {
	Slots    capacityReportDimension `json:"slots"`
	Seconds  capacityReportDimension `json:"seconds"`
	Messages capacityReportDimension `json:"messages"`
	Tokens   capacityReportDimension `json:"tokens"`
	Credits  capacityReportDimension `json:"credits"`
}
type capacityReport struct {
	SchemaVersion string                             `json:"schema_version"`
	WindowRef     application.AgentCapacityWindowRef `json:"window_ref"`
	Status        string                             `json:"status"`
	Quality       governance.UsageQuality            `json:"quality"`
	ObservedAt    time.Time                          `json:"observed_at"`
	ExpiresAt     time.Time                          `json:"expires_at"`
	ResetAt       time.Time                          `json:"reset_at,omitempty"`
	TryAgainAt    time.Time                          `json:"try_again_at,omitempty"`
	ArtifactRef   string                             `json:"artifact_ref,omitempty"`
	RateLimits    capacityReportLimits               `json:"rate_limits"`
}

func (observer CapacityObserver) ObserveCapacity(ctx context.Context, source application.AgentCapacitySourceRef, pool application.AgentCapacityPoolRef) (application.AgentCapacityObservation, error) {
	if ctx == nil || observer.ReportPath == "" || observer.MaxBytes <= 0 ||
		source == "" || pool == "" || source != observer.SourceRef || pool != observer.PoolRef {
		return application.AgentCapacityObservation{}, &Error{Code: CodeStateInvalid}
	}
	if err := ctx.Err(); err != nil {
		return application.AgentCapacityObservation{}, err
	}
	before, err := os.Lstat(observer.ReportPath)
	if err != nil || !before.Mode().IsRegular() {
		return application.AgentCapacityObservation{}, &Error{Code: CodeUnavailable}
	}
	file, err := os.Open(observer.ReportPath)
	if err != nil {
		return application.AgentCapacityObservation{}, &Error{Code: CodeUnavailable}
	}
	defer file.Close()
	if opened, statErr := file.Stat(); statErr != nil || !os.SameFile(before, opened) {
		return application.AgentCapacityObservation{}, &Error{Code: CodeUnavailable}
	}
	defer context.AfterFunc(ctx, func() { _ = file.Close() })()
	data, readErr := io.ReadAll(io.LimitReader(file, observer.MaxBytes+1))
	if ctxErr := ctx.Err(); ctxErr != nil {
		return application.AgentCapacityObservation{}, ctxErr
	}
	if int64(len(data)) > observer.MaxBytes {
		return application.AgentCapacityObservation{}, &Error{Code: CodeOutputTooLarge}
	}
	var report capacityReport
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if readErr != nil || decoder.Decode(&report) != nil || requireJSONEOF(decoder) != nil || report.SchemaVersion != "codex_usage_accounting.v1" {
		return application.AgentCapacityObservation{}, &Error{Code: CodeOutputInvalid}
	}
	var dimensions [5]application.AgentCapacityDimension
	values := [...]capacityReportDimension{report.RateLimits.Slots, report.RateLimits.Seconds, report.RateLimits.Messages, report.RateLimits.Tokens, report.RateLimits.Credits}
	for index, value := range values {
		dimensions[index] = translateCapacityDimension(value, [...]string{"executions", "seconds", "messages", "tokens", "credit_microunits"}[index])
	}
	var artifact goal.ArtifactRef
	var artifactErr error
	if report.ArtifactRef != "" {
		artifact, artifactErr = goal.NewArtifactRef(report.ArtifactRef)
	}
	observation := application.AgentCapacityObservation{
		SourceRef: source, PoolRef: pool, WindowRef: report.WindowRef, Status: map[string]application.AgentCapacityStatus{"available": application.AgentCapacityAvailable, "limited": application.AgentCapacityAvailable, "exhausted": application.AgentCapacityAvailable, "unknown": application.AgentCapacityUnavailable, "unavailable": application.AgentCapacityUnavailable}[report.Status], Quality: report.Quality,
		ObservedAt: report.ObservedAt, ExpiresAt: report.ExpiresAt, ResetAt: report.ResetAt, RetryAt: report.TryAgainAt, ArtifactRef: artifact,
		Resources: application.AgentCapacityResources{Slots: dimensions[0], Seconds: dimensions[1], Messages: dimensions[2], Tokens: dimensions[3], Credits: dimensions[4]},
	}
	if artifactErr != nil || application.ValidateAgentCapacityObservation(observation) != nil {
		return application.AgentCapacityObservation{}, &Error{Code: CodeOutputInvalid}
	}
	return observation, nil
}
func translateCapacityDimension(value capacityReportDimension, unit string) application.AgentCapacityDimension {
	empty := value.Unit == "" && value.Limit == nil && value.Remaining == nil
	if value.Governs == nil && empty {
		return application.AgentCapacityDimension{Applicability: application.AgentCapacityApplicabilityUnknown}
	}
	if value.Governs != nil && !*value.Governs && empty {
		return application.AgentCapacityDimension{Applicability: application.AgentCapacityApplicabilityNotApplicable}
	}
	if value.Governs != nil && *value.Governs && value.Unit == unit &&
		value.Limit != nil && value.Remaining != nil && *value.Limit >= 0 && *value.Remaining >= 0 && *value.Remaining <= *value.Limit {
		return application.AgentCapacityDimension{Applicability: application.AgentCapacityApplicabilityApplicable, Limit: application.AgentCapacityAmount{Present: true, Value: *value.Limit}, Remaining: application.AgentCapacityAmount{Present: true, Value: *value.Remaining}}
	}
	return application.AgentCapacityDimension{}
}
