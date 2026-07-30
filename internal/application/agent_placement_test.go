package application

import (
	"errors"
	"testing"
	"time"
)

func TestAgentQuotaGateIsSeparateAndFailsClosed(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	base := placementQuotaRecord(now)
	exhausted, unknown, stale, unknownQuality := base, base, base, base
	exhausted.Status, unknown.Status, stale.ExpiresAt = AgentQuotaExhausted, AgentQuotaUnknown, now
	unknownQuality.Quality = "unknown"
	tests := []struct {
		record *AgentQuotaObservationRecord
		reason AgentCapacityAdmissionReason
	}{
		{nil, AgentCapacityAdmissionUnknown},
		{&base, AgentCapacityAdmissionAvailable},
		{&exhausted, AgentCapacityAdmissionExhausted},
		{&unknown, AgentCapacityAdmissionUnknown},
		{&unknownQuality, AgentCapacityAdmissionUnknown},
		{&stale, AgentCapacityAdmissionStale},
	}
	for _, test := range tests {
		reason, err := DecideAgentQuotaGate(now, test.record)
		if err != nil || reason != test.reason {
			t.Fatalf("DecideAgentQuotaGate() = %s, %v", reason, err)
		}
	}
	for _, invalidRef := range []string{" quota:one", "quota:\x00"} {
		malformed := base
		malformed.Ref = invalidRef
		if _, err := DecideAgentQuotaGate(now, &malformed); !errors.Is(err, ErrAgentCapacityInvalid) {
			t.Fatalf("cuota no canónica aceptada: %v", err)
		}
	}
	for _, mutate := range []func(*AgentQuotaObservationRecord){
		func(value *AgentQuotaObservationRecord) { value.WindowRef = " window:one" },
		func(value *AgentQuotaObservationRecord) { value.Revision = 0 },
		func(value *AgentQuotaObservationRecord) { value.Status = "unavailable" },
		func(value *AgentQuotaObservationRecord) { value.Status = "stale" },
		func(value *AgentQuotaObservationRecord) { value.Quality = "invented" },
		func(value *AgentQuotaObservationRecord) { value.ObservedAt = time.Time{} },
		func(value *AgentQuotaObservationRecord) {
			value.ObservedAt, value.ExpiresAt = now.Add(time.Minute), now.Add(2*time.Minute)
		},
		func(value *AgentQuotaObservationRecord) { value.ExpiresAt = value.ObservedAt },
		func(value *AgentQuotaObservationRecord) { value.ResetAt = value.ObservedAt },
		func(value *AgentQuotaObservationRecord) { value.RetryAt = value.ObservedAt },
	} {
		invalid := base
		mutate(&invalid)
		if _, err := DecideAgentQuotaGate(now, &invalid); !errors.Is(err, ErrAgentCapacityInvalid) {
			t.Fatalf("registro de cuota inválido aceptado: %+v", invalid)
		}
	}
}

func placementQuotaRecord(now time.Time) AgentQuotaObservationRecord {
	return AgentQuotaObservationRecord{Ref: "quota:one", WindowRef: "window:one", Revision: 7,
		Status: AgentQuotaAvailable, Quality: "exact",
		ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute)}
}
