package orquestacapacity

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

const (
	AgentHomeSchemaVersionV0           = "agent_home.v0"
	AgentHomeCollectionSchemaVersionV0 = "agent_home_collection.v0"
)

type AgentHomeV0 struct {
	SchemaVersion         string           `json:"schema_version"`
	LogicalAgentRef       string           `json:"logical_agent_ref"`
	HomeRef               string           `json:"home_ref"`
	PoolID                string           `json:"pool_id"`
	RuntimeRef            string           `json:"runtime_ref"`
	ProviderRef           string           `json:"provider_ref"`
	AccountRef            string           `json:"account_ref"`
	CredentialRef         string           `json:"credential_ref,omitempty"`
	CredentialKind        string           `json:"credential_kind"`
	AccountKind           string           `json:"account_kind"`
	EntitlementKind       string           `json:"entitlement_kind"`
	Enabled               bool             `json:"enabled"`
	PausedUntil           string           `json:"paused_until,omitempty"`
	CooldownUntil         string           `json:"cooldown_until,omitempty"`
	QuotaState            string           `json:"quota_state"`
	Quota                 *QuotaSnapshotV0 `json:"quota,omitempty"`
	ActiveSessions        int              `json:"active_sessions"`
	ReservedSessions      int              `json:"reserved_sessions"`
	MaxConcurrentSessions int              `json:"max_concurrent_sessions"`
	DailyLimitSeconds     *int             `json:"daily_limit_seconds,omitempty"`
	WeeklyLimitSeconds    *int             `json:"weekly_limit_seconds,omitempty"`
	UsageObserved         *UsageObservedV0 `json:"usage_observed,omitempty"`
	LastCheckedAt         string           `json:"last_checked_at,omitempty"`
}

type AgentHomeCollectionV0 struct {
	SchemaVersion string        `json:"schema_version"`
	Homes         []AgentHomeV0 `json:"homes"`
}

type UsageObservedV0 struct {
	Seconds  int `json:"seconds,omitempty"`
	Messages int `json:"messages,omitempty"`
	Tokens   int `json:"tokens,omitempty"`
	Credits  int `json:"credits,omitempty"`
}

type AgentHomeValidationErrorV0 struct {
	Issues []CapacityDecisionIssueV0 `json:"issues"`
}

func (err AgentHomeValidationErrorV0) Error() string {
	if len(err.Issues) == 0 {
		return string(ErrCapacityDecisionInvalidaV0)
	}
	return string(err.Issues[0].Code)
}

func DecodeAgentHomeV0(data []byte) (AgentHomeV0, error) {
	var home AgentHomeV0
	if err := decodeAgentHomeJSONV0(data, &home); err != nil {
		return AgentHomeV0{}, err
	}
	if issues := ValidateAgentHomeV0(home); len(issues) > 0 {
		return AgentHomeV0{}, AgentHomeValidationErrorV0{Issues: issues}
	}
	return home, nil
}

func DecodeAgentHomeCollectionV0(data []byte) (AgentHomeCollectionV0, error) {
	var collection AgentHomeCollectionV0
	if err := decodeAgentHomeJSONV0(data, &collection); err != nil {
		return AgentHomeCollectionV0{}, err
	}
	if collection.SchemaVersion != AgentHomeCollectionSchemaVersionV0 || len(collection.Homes) < 2 || len(collection.Homes) > 8 {
		return AgentHomeCollectionV0{}, AgentHomeValidationErrorV0{Issues: []CapacityDecisionIssueV0{{
			Code:  ErrCapacityDecisionSchemaNoSoportadoV0,
			Field: "schema_version",
		}}}
	}
	var issues []CapacityDecisionIssueV0
	for _, home := range collection.Homes {
		issues = append(issues, ValidateAgentHomeV0(home)...)
	}
	if len(issues) > 0 {
		return AgentHomeCollectionV0{}, AgentHomeValidationErrorV0{Issues: issues}
	}
	return collection, nil
}

func ValidateAgentHomeV0(home AgentHomeV0) []CapacityDecisionIssueV0 {
	v := capacityDecisionValidatorV0{}
	v.requireConst("schema_version", home.SchemaVersion, AgentHomeSchemaVersionV0, ErrCapacityDecisionSchemaNoSoportadoV0)
	v.requireOpaque("logical_agent_ref", home.LogicalAgentRef)
	v.requireOpaque("home_ref", home.HomeRef)
	v.requireOpaque("pool_id", home.PoolID)
	v.requireOpaque("runtime_ref", home.RuntimeRef)
	v.requireOpaque("provider_ref", home.ProviderRef)
	v.requireOpaque("account_ref", home.AccountRef)
	v.optionalOpaque("credential_ref", home.CredentialRef)
	if !isOneOfV0(home.CredentialKind, "oauth", "api_key", "browser_session", "local_runtime", "none", "unknown") {
		v.add(ErrCapacityDecisionInvalidaV0, "credential_kind")
	}
	if isOneOfV0(home.CredentialKind, "oauth", "api_key", "browser_session") && home.CredentialRef == "" {
		v.add(ErrReferenciaNoOpacaV0, "credential_ref")
	}
	if !isOneOfV0(home.AccountKind, "personal", "service", "pro", "premium", "enterprise", "local", "ephemeral", "unknown") {
		v.add(ErrCapacityDecisionInvalidaV0, "account_kind")
	}
	if !isOneOfV0(home.EntitlementKind, "free", "pro", "premium", "enterprise", "local", "unknown") {
		v.add(ErrCapacityDecisionInvalidaV0, "entitlement_kind")
	}
	if !home.Enabled {
		v.add(ErrPoolNoDisponibleV0, "enabled")
	}
	v.optionalUTCTime("paused_until", home.PausedUntil)
	v.optionalUTCTime("cooldown_until", home.CooldownUntil)
	if !isOneOfV0(home.QuotaState, "active", "limited", "exhausted", "paused", "unknown") {
		v.add(ErrCuotaNoDisponibleV0, "quota_state")
	}
	if home.Quota != nil {
		v.validateQuota(*home.Quota)
	}
	validateAgentHomeSessionsV0(&v, home)
	v.optionalNonNegativeInt("daily_limit_seconds", home.DailyLimitSeconds)
	v.optionalNonNegativeInt("weekly_limit_seconds", home.WeeklyLimitSeconds)
	if home.UsageObserved != nil {
		validateUsageObservedV0(&v, *home.UsageObserved)
	}
	v.optionalUTCTime("last_checked_at", home.LastCheckedAt)
	return v.issues
}

func validateAgentHomeSessionsV0(v *capacityDecisionValidatorV0, home AgentHomeV0) {
	if home.ActiveSessions < 0 || home.ReservedSessions < 0 || home.MaxConcurrentSessions < 1 || home.MaxConcurrentSessions > 16 {
		v.add(ErrConcurrenciaHomeAgotadaV0, "max_concurrent_sessions")
		return
	}
	if home.ActiveSessions+home.ReservedSessions > home.MaxConcurrentSessions {
		v.add(ErrConcurrenciaHomeAgotadaV0, "reserved_sessions")
	}
}

func validateUsageObservedV0(v *capacityDecisionValidatorV0, usage UsageObservedV0) {
	for field, value := range map[string]int{
		"usage_observed.seconds":  usage.Seconds,
		"usage_observed.messages": usage.Messages,
		"usage_observed.tokens":   usage.Tokens,
		"usage_observed.credits":  usage.Credits,
	} {
		if value < 0 {
			v.add(ErrCapacityDecisionInvalidaV0, field)
		}
	}
}

func decodeAgentHomeJSONV0(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return AgentHomeValidationErrorV0{Issues: []CapacityDecisionIssueV0{{Code: ErrCapacityDecisionJSONInvalidoV0}}}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return AgentHomeValidationErrorV0{Issues: []CapacityDecisionIssueV0{{Code: ErrCapacityDecisionJSONInvalidoV0}}}
	}
	return nil
}
