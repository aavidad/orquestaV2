package main

import (
	"strings"
	"time"
)

type opesTemarioCyclePublicSummaryV0 struct {
	SchemaVersion          string                     `json:"schema_version"`
	Command                string                     `json:"command"`
	Status                 string                     `json:"status"`
	GeneratedAt            string                     `json:"generated_at"`
	Freshness              string                     `json:"freshness"`
	RedactionLevel         string                     `json:"redaction_level"`
	DiagnosticsMode        string                     `json:"diagnostics_mode"`
	CanonicalSource        string                     `json:"canonical_source"`
	OPESBaseURLRef         string                     `json:"opes_base_url_ref,omitempty"`
	OrquestaBaseURLRef     string                     `json:"orquesta_base_url_ref,omitempty"`
	OPESDestination        string                     `json:"opes_destination,omitempty"`
	OrquestaDestination    string                     `json:"orquesta_destination,omitempty"`
	DestinationEvidenceRef string                     `json:"destination_evidence_ref,omitempty"`
	Filter                 opesDrainPublicFilterV0    `json:"filter"`
	Limit                  int                        `json:"limit"`
	JobTypeSequence        []string                   `json:"job_type_sequence"`
	FinalJobType           string                     `json:"final_job_type"`
	ProgramID              string                     `json:"program_id,omitempty"`
	TopicID                string                     `json:"topic_id,omitempty"`
	CorrelationID          string                     `json:"correlation_id,omitempty"`
	DryRun                 bool                       `json:"dry_run,omitempty"`
	MaxTicks               int                        `json:"max_ticks"`
	Ticks                  int                        `json:"ticks"`
	Completed              bool                       `json:"completed"`
	FinalSeen              bool                       `json:"final_seen"`
	StopReason             string                     `json:"stop_reason"`
	SelectedJobTypes       []string                   `json:"selected_job_types,omitempty"`
	TickSummaries          []opesDrainPublicSummaryV0 `json:"tick_summaries,omitempty"`
	ErrorCodes             []string                   `json:"error_codes,omitempty"`
}

func commandPublicOPESTemarioCyclePayloadV0(
	summary opesTemarioCycleSummaryV0,
) opesTemarioCyclePublicSummaryV0 {
	errorCodes := make([]string, 0, len(summary.Errors))
	for _, item := range summary.Errors {
		errorCodes = append(errorCodes, commandPublicMessageCodeV0(item.Code))
	}
	status := "completed"
	if !summary.Completed {
		status = "pending"
	}
	if summary.StopReason == "max_ticks_reached" || summary.StopReason == "context_cancelled" {
		status = "continuable"
	}
	if summary.StopReason == "dry_run_preview" {
		status = "dry_run"
	}
	ticks := make([]opesDrainPublicSummaryV0, 0, len(summary.TickSummaries))
	for _, tickSummary := range summary.TickSummaries {
		ticks = append(ticks, commandPublicOPESDrainPayloadV0(tickSummary))
	}
	filter := opesTemarioCyclePublicFilterV0(summary)
	return opesTemarioCyclePublicSummaryV0{
		SchemaVersion:          "orquesta_opes_temario_cycle_public_summary.v0",
		Command:                opesTemarioCycleCommandNameV0,
		Status:                 status,
		GeneratedAt:            time.Now().UTC().Format(time.RFC3339),
		Freshness:              commandPublicFreshnessLiveV0,
		RedactionLevel:         commandPublicRedactionLevelV0,
		DiagnosticsMode:        "local_diagnostics_require_bridge_ledgers_and_domain_logs",
		CanonicalSource:        commandPublicCanonicalSourceV0,
		OPESBaseURLRef:         firstNonEmptyV0(summary.Destination.OPESDestination.URLRef, commandPublicOpaqueRefV0("opes-base-url", summary.OPESBaseURL)),
		OrquestaBaseURLRef:     firstNonEmptyV0(summary.Destination.OrquestaDestination.URLRef, commandPublicOpaqueRefV0("orquesta-base-url", summary.OrquestaBaseURL)),
		OPESDestination:        strings.TrimSpace(summary.Destination.OPESDestination.Category),
		OrquestaDestination:    strings.TrimSpace(summary.Destination.OrquestaDestination.Category),
		DestinationEvidenceRef: strings.TrimSpace(summary.Destination.DestinationEvidenceRef),
		Filter:                 filter,
		Limit:                  summary.Limit,
		JobTypeSequence:        append([]string(nil), summary.JobTypeSequence...),
		FinalJobType:           strings.TrimSpace(summary.FinalJobType),
		ProgramID:              strings.TrimSpace(summary.ProgramID),
		TopicID:                strings.TrimSpace(summary.TopicID),
		CorrelationID:          strings.TrimSpace(summary.CorrelationID),
		DryRun:                 summary.DryRun,
		MaxTicks:               summary.MaxTicks,
		Ticks:                  summary.Ticks,
		Completed:              summary.Completed,
		FinalSeen:              summary.FinalSeen,
		StopReason:             strings.TrimSpace(summary.StopReason),
		SelectedJobTypes:       append([]string(nil), summary.SelectedJobTypes...),
		TickSummaries:          ticks,
		ErrorCodes:             errorCodes,
	}
}

func opesTemarioCyclePublicFilterV0(summary opesTemarioCycleSummaryV0) opesDrainPublicFilterV0 {
	filter := opesDrainPublicFilterV0{
		Mode:            "job_type_sequence",
		JobTypeSequence: append([]string(nil), summary.JobTypeSequence...),
		ProgramID:       strings.TrimSpace(summary.ProgramID),
		TopicID:         strings.TrimSpace(summary.TopicID),
		CorrelationID:   strings.TrimSpace(summary.CorrelationID),
	}
	if filter.ProgramID != "" {
		filter.Mode = "program_id"
	}
	if filter.TopicID != "" {
		filter.Mode = "topic_id"
	}
	if filter.CorrelationID != "" {
		filter.Mode = "correlation_id"
	}
	return filter
}
