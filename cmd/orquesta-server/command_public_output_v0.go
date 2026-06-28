package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	commandPublicOutputSchemaVersionV0 = "orquesta_command_public_output.v0"
	commandPublicCanonicalSourceV0     = "ack_events_evidence_and_stores"
	commandPublicFreshnessLiveV0       = "live"
	commandPublicFreshnessSnapshotV0   = "snapshot"
	commandPublicRedactionLevelV0      = "refs_and_counters"
)

type commandPublicOutputV0 struct {
	SchemaVersion   string                 `json:"schema_version"`
	Command         string                 `json:"command"`
	Status          string                 `json:"status"`
	GeneratedAt     string                 `json:"generated_at"`
	Freshness       string                 `json:"freshness"`
	RedactionLevel  string                 `json:"redaction_level"`
	DiagnosticsMode string                 `json:"diagnostics_mode"`
	CanonicalSource string                 `json:"canonical_source"`
	PayloadRef      string                 `json:"payload_ref,omitempty"`
	JobType         string                 `json:"job_type,omitempty"`
	JobTypeSequence []string               `json:"job_type_sequence,omitempty"`
	SelectedJobType string                 `json:"selected_job_type,omitempty"`
	EmptyJobTypes   []string               `json:"empty_job_types,omitempty"`
	DryRun          bool                   `json:"dry_run,omitempty"`
	Seen            int                    `json:"seen,omitempty"`
	Submitted       int                    `json:"submitted,omitempty"`
	Skipped         int                    `json:"skipped,omitempty"`
	Results         []opesDrainJobResultV0 `json:"results,omitempty"`
	Payload         any                    `json:"payload,omitempty"`
}

type opesDrainPublicSummaryV0 struct {
	SchemaVersion          string                  `json:"schema_version"`
	Command                string                  `json:"command"`
	Status                 string                  `json:"status"`
	GeneratedAt            string                  `json:"generated_at"`
	Freshness              string                  `json:"freshness"`
	RedactionLevel         string                  `json:"redaction_level"`
	DiagnosticsMode        string                  `json:"diagnostics_mode"`
	CanonicalSource        string                  `json:"canonical_source"`
	OPESBaseURLRef         string                  `json:"opes_base_url_ref,omitempty"`
	OrquestaBaseURLRef     string                  `json:"orquesta_base_url_ref,omitempty"`
	OPESDestination        string                  `json:"opes_destination,omitempty"`
	OrquestaDestination    string                  `json:"orquesta_destination,omitempty"`
	DestinationEvidenceRef string                  `json:"destination_evidence_ref,omitempty"`
	Filter                 opesDrainPublicFilterV0 `json:"filter"`
	Limit                  int                     `json:"limit"`
	JobType                string                  `json:"job_type,omitempty"`
	JobTypeSequence        []string                `json:"job_type_sequence,omitempty"`
	SelectedJobType        string                  `json:"selected_job_type,omitempty"`
	TransportJobType       string                  `json:"transport_job_type,omitempty"`
	EmptyJobTypes          []string                `json:"empty_job_types,omitempty"`
	JobRef                 string                  `json:"job_ref,omitempty"`
	ProgramID              string                  `json:"program_id,omitempty"`
	TopicID                string                  `json:"topic_id,omitempty"`
	CorrelationID          string                  `json:"correlation_id,omitempty"`
	DryRun                 bool                    `json:"dry_run,omitempty"`
	Seen                   int                     `json:"seen"`
	Submitted              int                     `json:"submitted"`
	AlreadySubmitted       int                     `json:"already_submitted,omitempty"`
	Claimed                int                     `json:"claimed,omitempty"`
	Skipped                int                     `json:"skipped"`
	Results                []opesDrainJobResultV0  `json:"results,omitempty"`
	ErrorCodes             []string                `json:"error_codes,omitempty"`
}

type opesDrainPublicFilterV0 struct {
	Mode             string   `json:"mode"`
	JobType          string   `json:"job_type,omitempty"`
	TransportJobType string   `json:"transport_job_type,omitempty"`
	JobTypeSequence  []string `json:"job_type_sequence,omitempty"`
	SelectedJobType  string   `json:"selected_job_type,omitempty"`
	JobRef           string   `json:"job_ref,omitempty"`
	ProgramID        string   `json:"program_id,omitempty"`
	TopicID          string   `json:"topic_id,omitempty"`
	CorrelationID    string   `json:"correlation_id,omitempty"`
}

type runStatusPublicSummaryV0 struct {
	Estado          string                                     `json:"estado"`
	RequestID       string                                     `json:"request_id,omitempty"`
	CorrelationID   string                                     `json:"correlation_id,omitempty"`
	RunRef          string                                     `json:"run_ref,omitempty"`
	ExternalJob     *orquestamcp.MCPDirectorExternalJobStatsV0 `json:"external_job,omitempty"`
	Stats           any                                        `json:"stats,omitempty"`
	DecisionContext any                                        `json:"decision_context,omitempty"`
	Errores         []orquestamcp.MCPValidationIssueV0         `json:"errores_publicos,omitempty"`
}

func writeCommandPublicOutputV0(
	stdout io.Writer,
	command string,
	status string,
	freshness string,
	diagnosticsMode string,
	payload any,
) error {
	out := commandPublicOutputV0{
		SchemaVersion:   commandPublicOutputSchemaVersionV0,
		Command:         strings.TrimSpace(command),
		Status:          strings.TrimSpace(status),
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		Freshness:       strings.TrimSpace(freshness),
		RedactionLevel:  commandPublicRedactionLevelV0,
		DiagnosticsMode: strings.TrimSpace(diagnosticsMode),
		CanonicalSource: commandPublicCanonicalSourceV0,
		PayloadRef:      commandPublicPayloadRefV0(command, payload),
		Payload:         payload,
	}
	if out.Status == "" {
		out.Status = "ok"
	}
	if out.Freshness == "" {
		out.Freshness = commandPublicFreshnessSnapshotV0
	}
	if opesPayload, ok := payload.(opesDrainPublicSummaryV0); ok {
		out.JobType = opesPayload.JobType
		out.JobTypeSequence = append([]string(nil), opesPayload.JobTypeSequence...)
		out.SelectedJobType = opesPayload.SelectedJobType
		out.EmptyJobTypes = append([]string(nil), opesPayload.EmptyJobTypes...)
		out.DryRun = opesPayload.DryRun
		out.Seen = opesPayload.Seen
		out.Submitted = opesPayload.Submitted
		out.Skipped = opesPayload.Skipped
		out.Results = append([]opesDrainJobResultV0(nil), opesPayload.Results...)
	}
	return writeCommandJSONOutputV0(stdout, out)
}

func commandPublicServerStatusPayloadV0(status orquestaserver.ServerPublicStatusV0) orquestaserver.ServerPublicStatusV0 {
	status.Addr = ""
	status.LastError = commandPublicMessageCodeV0(status.LastError)
	status.LastSupervisorError = commandPublicMessageCodeV0(status.LastSupervisorError)
	status.SupervisorLastError = commandPublicMessageCodeV0(status.SupervisorLastError)
	return status
}

func commandPublicRunStatusPayloadV0(result orquestamcp.MCPDirectorStatsToolResultV0) runStatusPublicSummaryV0 {
	return runStatusPublicSummaryV0{
		Estado:          strings.TrimSpace(result.Estado),
		RequestID:       strings.TrimSpace(result.RequestID),
		CorrelationID:   strings.TrimSpace(result.CorrelationID),
		RunRef:          strings.TrimSpace(result.RunRef),
		ExternalJob:     result.ExternalJob,
		Stats:           result.Stats,
		DecisionContext: result.DecisionContext,
		Errores:         append([]orquestamcp.MCPValidationIssueV0(nil), result.Errores...),
	}
}

func commandPublicOPESDrainPayloadV0(summary opesDrainSummaryV0) opesDrainPublicSummaryV0 {
	errorCodes := make([]string, 0, len(summary.Errors))
	for _, item := range summary.Errors {
		errorCodes = append(errorCodes, commandPublicMessageCodeV0(item.Code))
	}
	status := "completed"
	if len(summary.Errors) > 0 {
		status = "failed"
	}
	return opesDrainPublicSummaryV0{
		SchemaVersion:          "orquesta_opes_drain_once_public_summary.v0",
		Command:                "opes-drain-once",
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
		Filter:                 commandPublicOPESDrainFilterV0(summary),
		Limit:                  summary.Limit,
		JobType:                strings.TrimSpace(summary.JobType),
		JobTypeSequence:        append([]string(nil), summary.JobTypeSequence...),
		SelectedJobType:        strings.TrimSpace(summary.SelectedJobType),
		TransportJobType:       strings.TrimSpace(summary.TransportJobType),
		EmptyJobTypes:          append([]string(nil), summary.EmptyJobTypes...),
		JobRef:                 strings.TrimSpace(summary.JobRef),
		ProgramID:              strings.TrimSpace(summary.ProgramID),
		TopicID:                strings.TrimSpace(summary.TopicID),
		CorrelationID:          strings.TrimSpace(summary.CorrelationID),
		DryRun:                 summary.DryRun,
		Seen:                   summary.Seen,
		Submitted:              summary.Submitted,
		AlreadySubmitted:       summary.AlreadySubmitted,
		Claimed:                summary.Claimed,
		Skipped:                summary.Skipped,
		Results:                append([]opesDrainJobResultV0(nil), summary.Results...),
		ErrorCodes:             errorCodes,
	}
}

func commandPublicOPESDrainFilterV0(summary opesDrainSummaryV0) opesDrainPublicFilterV0 {
	filter := opesDrainPublicFilterV0{
		JobType:          strings.TrimSpace(summary.JobType),
		TransportJobType: strings.TrimSpace(summary.TransportJobType),
		JobTypeSequence:  append([]string(nil), summary.JobTypeSequence...),
		SelectedJobType:  strings.TrimSpace(summary.SelectedJobType),
		JobRef:           strings.TrimSpace(summary.JobRef),
		ProgramID:        strings.TrimSpace(summary.ProgramID),
		TopicID:          strings.TrimSpace(summary.TopicID),
		CorrelationID:    strings.TrimSpace(summary.CorrelationID),
	}
	switch {
	case filter.JobRef != "":
		filter.Mode = "job_ref"
	case len(filter.JobTypeSequence) > 0:
		filter.Mode = "job_type_sequence"
	case filter.ProgramID != "":
		filter.Mode = "program_id"
	case filter.TopicID != "":
		filter.Mode = "topic_id"
	case filter.CorrelationID != "":
		filter.Mode = "correlation_id"
	case filter.JobType != "":
		filter.Mode = "job_type"
	default:
		filter.Mode = "unfiltered"
	}
	return filter
}

func firstNonEmptyV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func commandPublicPayloadRefV0(command string, payload any) string {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 {
		return ""
	}
	sum := sha256.Sum256(data)
	return "command-output-ref-" + commandPublicRefPartV0(command) + "-" + hex.EncodeToString(sum[:])[:16]
}

func commandPublicOpaqueRefV0(kind string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return commandPublicRefPartV0(kind) + "-ref-" + hex.EncodeToString(sum[:])[:16]
}

func commandPublicMessageCodeV0(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	if len(value) > 64 {
		return "external_error"
	}
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
		if !ok {
			return "external_error"
		}
	}
	return value
}

func commandPublicRefPartV0(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
