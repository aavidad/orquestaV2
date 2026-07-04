package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
)

const (
	opesTemarioCycleCommandNameV0          = "opes-temario-cycle"
	defaultOPESTemarioCycleMaxTicksV0      = 1000
	defaultOPESTemarioCycleIntervalSeconds = 5
)

type opesTemarioCycleConfigV0 struct {
	DrainConfig  opesDrainConfigV0
	MaxTicks     int
	TickSleep    time.Duration
	FinalJobType string
}

type opesTemarioCycleSummaryV0 struct {
	OPESBaseURL      string                       `json:"opes_base_url"`
	OrquestaBaseURL  string                       `json:"orquesta_base_url"`
	Limit            int                          `json:"limit"`
	JobTypeSequence  []string                     `json:"job_type_sequence"`
	FinalJobType     string                       `json:"final_job_type"`
	ProgramID        string                       `json:"program_id,omitempty"`
	TopicID          string                       `json:"topic_id,omitempty"`
	CorrelationID    string                       `json:"correlation_id,omitempty"`
	DryRun           bool                         `json:"dry_run,omitempty"`
	MaxTicks         int                          `json:"max_ticks"`
	Ticks            int                          `json:"ticks"`
	Completed        bool                         `json:"completed"`
	FinalSeen        bool                         `json:"final_seen"`
	StopReason       string                       `json:"stop_reason"`
	SelectedJobTypes []string                     `json:"selected_job_types,omitempty"`
	TickSummaries    []opesDrainSummaryV0         `json:"tick_summaries,omitempty"`
	Errors           []opesDrainPublicErrorV0     `json:"errors,omitempty"`
	Destination      opesDrainDestinationPolicyV0 `json:"-"`
}

func opesTemarioCycleCommandV0(stdout io.Writer, stderr io.Writer) int {
	projectConfig := opesProjectConfigFromEnvBestEffortV0()
	config, err := opesTemarioCycleConfigFromEnvV0()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: %v\n", opesTemarioCycleCommandNameV0, err)
		return 2
	}
	if !config.DrainConfig.DryRun && !opesBridgeConfirmFromProjectConfigFileV0(projectConfig) {
		_, _ = fmt.Fprintf(stderr, "%s: exporta ORQUESTA_OPES_BRIDGE_CONFIRM=1 para crear runs\n", opesTemarioCycleCommandNameV0)
		return 2
	}
	if !config.DrainConfig.DryRun && !opesBridgeHasSafeFilterV0(config.DrainConfig) {
		_, _ = fmt.Fprintf(stderr, "%s: usa secuencia OPES o scope job/program/topic/correlation antes de crear runs\n", opesTemarioCycleCommandNameV0)
		return 2
	}
	summary := runOPESTemarioCycleV0(context.Background(), config, runOPESDrainOnceV0)
	if err := writeCommandJSONOutputV0(stdout, commandPublicOPESTemarioCyclePayloadV0(summary)); err != nil {
		return reportCommandStdioWriteFailureV0(stderr, opesTemarioCycleCommandNameV0, "stdout", "json_encode", err)
	}
	if summary.Completed || config.DrainConfig.DryRun {
		return 0
	}
	return 1
}

func opesTemarioCycleConfigFromEnvV0() (opesTemarioCycleConfigV0, error) {
	projectConfig := opesProjectConfigFromEnvBestEffortV0()
	drainConfig, err := opesDrainConfigFromEnvV0()
	if err != nil {
		return opesTemarioCycleConfigV0{}, err
	}
	if strings.TrimSpace(drainConfig.JobType) != "" {
		return opesTemarioCycleConfigV0{}, fmt.Errorf("%s incompatible con ciclo completo OPES", envOPESBridgeJobTypeV0)
	}
	if strings.TrimSpace(drainConfig.JobRef) != "" {
		return opesTemarioCycleConfigV0{}, fmt.Errorf("%s incompatible con ciclo completo OPES", envOPESBridgeJobRefV0)
	}
	if len(drainConfig.JobTypeSequence) == 0 {
		drainConfig.JobTypeSequence = orquestaopesbridge.OPESFullTemarioJobTypeSequenceV0()
	}
	if len(drainConfig.JobTypeSequence) == 0 {
		return opesTemarioCycleConfigV0{}, fmt.Errorf("opes_temario_sequence_required")
	}
	maxTicks := opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgeMaxTicksV0, defaultOPESTemarioCycleMaxTicksV0)
	tickSleep := opesBridgeDurationSecondsFromProjectConfigFileV0(
		projectConfig,
		envOPESBridgeIntervalSecondsV0,
		defaultOPESTemarioCycleIntervalSeconds*time.Second,
	)
	return opesTemarioCycleConfigV0{
		DrainConfig:  drainConfig,
		MaxTicks:     maxTicks,
		TickSleep:    tickSleep,
		FinalJobType: opesTemarioCycleFinalJobTypeV0(drainConfig.JobTypeSequence),
	}, nil
}

func runOPESTemarioCycleV0(
	ctx context.Context,
	config opesTemarioCycleConfigV0,
	drainer opesBridgeDrainerV0,
) opesTemarioCycleSummaryV0 {
	if ctx == nil {
		ctx = context.Background()
	}
	sequence := append([]string(nil), config.DrainConfig.JobTypeSequence...)
	summary := opesTemarioCycleSummaryV0{
		OPESBaseURL:     config.DrainConfig.OPESBaseURL,
		OrquestaBaseURL: config.DrainConfig.OrquestaBaseURL,
		Limit:           config.DrainConfig.Limit,
		JobTypeSequence: sequence,
		FinalJobType:    firstNonEmptyV0(config.FinalJobType, opesTemarioCycleFinalJobTypeV0(sequence)),
		ProgramID:       config.DrainConfig.ProgramID,
		TopicID:         config.DrainConfig.TopicID,
		CorrelationID:   config.DrainConfig.CorrelationID,
		DryRun:          config.DrainConfig.DryRun,
		MaxTicks:        opesTemarioCycleMaxTicksV0(config.MaxTicks),
		Destination:     config.DrainConfig.Destination,
	}
	if drainer == nil {
		summary.StopReason = "drainer_not_configured"
		return summary
	}
	for tick := 1; tick <= summary.MaxTicks; tick++ {
		if ctx.Err() != nil {
			summary.StopReason = "context_cancelled"
			break
		}
		tickCtx, cancel := context.WithTimeout(ctx, config.DrainConfig.HTTPTimeout)
		tickSummary, err := drainer(tickCtx, config.DrainConfig)
		cancel()
		summary.Ticks = tick
		if err != nil {
			summary.Errors = append(summary.Errors, opesDrainPublicErrorV0{Code: err.Error()})
		} else {
			summary.TickSummaries = append(summary.TickSummaries, tickSummary)
			summary.Errors = append(summary.Errors, tickSummary.Errors...)
			selected := strings.TrimSpace(tickSummary.SelectedJobType)
			if selected != "" {
				summary.SelectedJobTypes = append(summary.SelectedJobTypes, selected)
			}
			if selected == summary.FinalJobType && tickSummary.Seen > 0 {
				summary.FinalSeen = true
			}
			if opesTemarioCycleSequenceEmptyV0(tickSummary, sequence) {
				if summary.FinalSeen {
					summary.Completed = true
					summary.StopReason = "sequence_empty_after_final"
					break
				}
			}
		}
		if config.DrainConfig.DryRun {
			summary.StopReason = "dry_run_preview"
			break
		}
		if tick < summary.MaxTicks && !waitOPESTemarioCycleTickSleepV0(ctx, config.TickSleep) {
			summary.StopReason = "context_cancelled"
			break
		}
	}
	if summary.StopReason == "" {
		summary.StopReason = "max_ticks_reached"
	}
	return summary
}

func opesTemarioCycleFinalJobTypeV0(sequence []string) string {
	for index := len(sequence) - 1; index >= 0; index-- {
		if value := strings.TrimSpace(sequence[index]); value != "" {
			return value
		}
	}
	return orquestaopesbridge.OPESFinalTemarioJobTypeV0()
}

func opesTemarioCycleMaxTicksV0(value int) int {
	if value <= 0 {
		return defaultOPESTemarioCycleMaxTicksV0
	}
	return value
}

func opesTemarioCycleSequenceEmptyV0(summary opesDrainSummaryV0, sequence []string) bool {
	return strings.TrimSpace(summary.SelectedJobType) == "" &&
		summary.Seen == 0 &&
		len(summary.EmptyJobTypes) >= len(sequence)
}

func waitOPESTemarioCycleTickSleepV0(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
