package orquestaautoprogramming

import (
	"sort"
	"strings"
	"time"
)

const AutoprogrammingPrepareRunEnvelopeSchemaV0 = "orquesta_autoprogramming_prepare_run_envelope.v0"

const (
	AutoprogrammingPrepareRunExecutionModeGoalFirstV0          = "goal_first"
	AutoprogrammingPrepareRunExecutionModeLegacyDirectorLoopV0 = "legacy_director_loop"
)

// AutoprogrammingPrepareRunRequiredSettingV0 deliberately preserves only the
// canonical setting key. Values are validation inputs and may be secrets; they
// must never become durable intent evidence.
type AutoprogrammingPrepareRunRequiredSettingV0 struct {
	Key string `json:"key"`
}

// AutoprogrammingPrepareRunEnvelopeV0 is the durable, provider-neutral receipt
// of the public prepare_run command. It is stored inside the existing intent
// manifest so the manifest remains the single authority linked from Goal.
type AutoprogrammingPrepareRunEnvelopeV0 struct {
	SchemaVersion          string                                       `json:"schema_version"`
	RequestID              string                                       `json:"request_id"`
	CorrelationID          string                                       `json:"correlation_id"`
	IdempotencyKey         string                                       `json:"idempotency_key"`
	OccurredAt             string                                       `json:"occurred_at,omitempty"`
	RequestedBy            string                                       `json:"requested_by,omitempty"`
	DirectorExecutionMode  string                                       `json:"director_execution_mode,omitempty"`
	AutoprogrammingRequest AutoprogrammingRequestV0                     `json:"autoprogramming_request"`
	MaxBursts              int                                          `json:"max_bursts,omitempty"`
	MaxStepsPerBurst       int                                          `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait   int                                          `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands            int                                          `json:"max_commands,omitempty"`
	MaxOutboxPerCycle      int                                          `json:"max_outbox_per_cycle,omitempty"`
	PriorityScore          int                                          `json:"priority_score,omitempty"`
	RequiredSettings       []AutoprogrammingPrepareRunRequiredSettingV0 `json:"required_settings,omitempty"`
}

func CanonicalAutoprogrammingPrepareRunEnvelopeV0(
	envelope AutoprogrammingPrepareRunEnvelopeV0,
) (AutoprogrammingPrepareRunEnvelopeV0, []AutoprogrammingRequestIssueV0) {
	envelope.SchemaVersion = strings.TrimSpace(envelope.SchemaVersion)
	if envelope.SchemaVersion == "" {
		envelope.SchemaVersion = AutoprogrammingPrepareRunEnvelopeSchemaV0
	}
	envelope.RequestID = strings.TrimSpace(envelope.RequestID)
	envelope.CorrelationID = strings.TrimSpace(envelope.CorrelationID)
	envelope.IdempotencyKey = strings.TrimSpace(envelope.IdempotencyKey)
	envelope.OccurredAt = strings.TrimSpace(envelope.OccurredAt)
	envelope.RequestedBy = strings.TrimSpace(envelope.RequestedBy)
	envelope.DirectorExecutionMode = strings.ToLower(strings.TrimSpace(envelope.DirectorExecutionMode))
	if envelope.DirectorExecutionMode == "" {
		envelope.DirectorExecutionMode = AutoprogrammingPrepareRunExecutionModeGoalFirstV0
	}

	request, issues := CanonicalAutoprogrammingIntentRequestV0(envelope.AutoprogrammingRequest)
	if len(issues) != 0 {
		return AutoprogrammingPrepareRunEnvelopeV0{}, issues
	}
	envelope.AutoprogrammingRequest = request

	seen := map[string]struct{}{}
	settings := make([]AutoprogrammingPrepareRunRequiredSettingV0, 0, len(envelope.RequiredSettings))
	for _, setting := range envelope.RequiredSettings {
		key := strings.TrimSpace(setting.Key)
		if !autoprogrammingPrepareRunIdentityValueV0(key) {
			return AutoprogrammingPrepareRunEnvelopeV0{}, []AutoprogrammingRequestIssueV0{
				autoprogrammingRequestIssueV0("prepare_run_required_setting_key_invalid", "required_settings.key", "prepare_run_required_setting_key_invalid"),
			}
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		settings = append(settings, AutoprogrammingPrepareRunRequiredSettingV0{Key: key})
	}
	sort.Slice(settings, func(i, j int) bool { return settings[i].Key < settings[j].Key })
	envelope.RequiredSettings = settings

	var envelopeIssues []AutoprogrammingRequestIssueV0
	if envelope.SchemaVersion != AutoprogrammingPrepareRunEnvelopeSchemaV0 {
		envelopeIssues = append(envelopeIssues, autoprogrammingRequestIssueV0("prepare_run_envelope_schema_invalid", "schema_version", "prepare_run_envelope_schema_invalid"))
	}
	if !autoprogrammingPrepareRunIdentityValueV0(envelope.RequestID) {
		envelopeIssues = append(envelopeIssues, autoprogrammingRequestIssueV0("prepare_run_envelope_request_id_invalid", "request_id", "prepare_run_envelope_request_id_invalid"))
	}
	if !autoprogrammingPrepareRunIdentityValueV0(envelope.CorrelationID) {
		envelopeIssues = append(envelopeIssues, autoprogrammingRequestIssueV0("prepare_run_envelope_correlation_id_invalid", "correlation_id", "prepare_run_envelope_correlation_id_invalid"))
	}
	if !autoprogrammingPrepareRunIdentityValueV0(envelope.IdempotencyKey) {
		envelopeIssues = append(envelopeIssues, autoprogrammingRequestIssueV0("prepare_run_envelope_idempotency_key_invalid", "idempotency_key", "prepare_run_envelope_idempotency_key_invalid"))
	}
	if !autoprogrammingPrepareRunIdentityValueV0(envelope.RequestedBy) {
		envelopeIssues = append(envelopeIssues, autoprogrammingRequestIssueV0("prepare_run_envelope_requested_by_invalid", "requested_by", "prepare_run_envelope_requested_by_invalid"))
	}
	if envelope.OccurredAt != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, envelope.OccurredAt); err != nil {
			envelopeIssues = append(envelopeIssues, autoprogrammingRequestIssueV0("prepare_run_envelope_occurred_at_invalid", "occurred_at", "prepare_run_envelope_occurred_at_invalid"))
		} else {
			envelope.OccurredAt = parsed.UTC().Format(time.RFC3339Nano)
		}
	}
	switch envelope.DirectorExecutionMode {
	case AutoprogrammingPrepareRunExecutionModeGoalFirstV0, AutoprogrammingPrepareRunExecutionModeLegacyDirectorLoopV0:
	default:
		envelopeIssues = append(envelopeIssues, autoprogrammingRequestIssueV0("prepare_run_envelope_execution_mode_invalid", "director_execution_mode", "prepare_run_envelope_execution_mode_invalid"))
	}
	for _, limit := range []struct {
		field string
		value int
	}{
		{field: "max_bursts", value: envelope.MaxBursts},
		{field: "max_steps_per_burst", value: envelope.MaxStepsPerBurst},
		{field: "max_dispatches_per_wait", value: envelope.MaxDispatchesPerWait},
		{field: "max_commands", value: envelope.MaxCommands},
		{field: "max_outbox_per_cycle", value: envelope.MaxOutboxPerCycle},
		{field: "priority_score", value: envelope.PriorityScore},
	} {
		if limit.value < 0 {
			envelopeIssues = append(envelopeIssues, autoprogrammingRequestIssueV0("prepare_run_envelope_limit_invalid", limit.field, "prepare_run_envelope_limit_invalid"))
		}
	}
	if len(envelopeIssues) != 0 {
		return AutoprogrammingPrepareRunEnvelopeV0{}, envelopeIssues
	}
	return envelope, nil
}
