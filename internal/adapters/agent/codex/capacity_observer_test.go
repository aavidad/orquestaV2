package codex

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestCapacityObserverPreservesStructuredFacts(t *testing.T) {
	observer, report := capacityFixture(t)
	writeCapacityReport(t, observer.ReportPath, report)
	got, err := observer.ObserveCapacity(context.Background(), observer.SourceRef, observer.PoolRef)
	same, sameErr := observer.ObserveCapacity(context.Background(), observer.SourceRef, observer.PoolRef)
	if err != nil || sameErr != nil {
		t.Fatalf("observe errors: %v %v", err, sameErr)
	}
	if got.Status != application.AgentCapacityAvailable || got.Quality != "measured" || got.WindowRef != "window:one" || got.ArtifactRef.String() != "artifact:capacity:one" ||
		got.Resources.Slots.Remaining.Value != 0 || !got.Resources.Slots.Remaining.Present || got.Resources.Messages.Applicability != application.AgentCapacityApplicabilityUnknown || got.Resources.Tokens.Applicability != application.AgentCapacityApplicabilityNotApplicable || got.ObservedAt != same.ObservedAt || got.ExpiresAt != same.ExpiresAt || got.ResetAt != report.ResetAt || got.RetryAt != report.TryAgainAt {
		t.Fatalf("facts lost or unstable: %#v %#v", got, same)
	}
	report.WindowRef, report.Status, report.ArtifactRef = "window:two", "unknown", ""
	report.ObservedAt = report.ObservedAt.Add(2 * time.Hour)
	report.ExpiresAt, report.ResetAt, report.TryAgainAt = report.ObservedAt.Add(time.Minute), report.ObservedAt.Add(time.Hour), report.ObservedAt.Add(30*time.Minute)
	writeCapacityReport(t, observer.ReportPath, report)
	next, err := observer.ObserveCapacity(context.Background(), observer.SourceRef, observer.PoolRef)
	if err != nil || next.WindowRef == got.WindowRef || next.Status != application.AgentCapacityUnavailable || next.ObservedAt != report.ObservedAt || next.ArtifactRef.String() != "" {
		t.Fatalf("new skewed window not preserved: %#v %v", next, err)
	}
}

func TestCapacityObserverFailsClosedWithBoundedRedactedErrors(t *testing.T) {
	observer, report := capacityFixture(t)
	if _, err := observer.ObserveCapacity(context.Background(), observer.SourceRef, observer.PoolRef); ErrorCode(err) != CodeUnavailable {
		t.Fatalf("missing code=%q", ErrorCode(err))
	}
	valid, _ := json.Marshal(report)
	report.Status = "invalid"
	invalidStatus, _ := json.Marshal(report)
	report.Status, report.RateLimits.Slots.Unit = "exhausted", "messages"
	invalidUnit, _ := json.Marshal(report)
	cases := []struct {
		data []byte
		max  int64
		code string
	}{
		{append(valid, []byte(" {}")...), 4096, CodeOutputInvalid},
		{[]byte(`12345`), 4, CodeOutputTooLarge},
		{[]byte(`{"schema_version":"unsupported"}`), 4096, CodeOutputInvalid},
		{invalidUnit, 4096, CodeOutputInvalid},
		{invalidStatus, 4096, CodeOutputInvalid},
		{[]byte(`{"schema_version":"codex_usage_accounting.v1","extra":true}`), 4096, CodeOutputInvalid},
	}
	for index, test := range cases {
		if err := os.WriteFile(observer.ReportPath, test.data, 0o600); err != nil {
			t.Fatal(err)
		}
		observer.MaxBytes = test.max
		_, err := observer.ObserveCapacity(context.Background(), observer.SourceRef, observer.PoolRef)
		if ErrorCode(err) != test.code || err.Error() != test.code {
			t.Fatalf("case %d error=%q code=%q", index, err, ErrorCode(err))
		}
	}
	if _, err := observer.ObserveCapacity(context.Background(), observer.SourceRef, "pool:other"); ErrorCode(err) != CodeStateInvalid {
		t.Fatalf("scope code=%q", ErrorCode(err))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := observer.ObserveCapacity(ctx, observer.SourceRef, observer.PoolRef); err != context.Canceled {
		t.Fatalf("cancellation hidden: %v", err)
	}
	if err := os.Remove(observer.ReportPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), observer.ReportPath); err != nil {
		t.Fatal(err)
	}
	if _, err := observer.ObserveCapacity(context.Background(), observer.SourceRef, observer.PoolRef); ErrorCode(err) != CodeUnavailable {
		t.Fatalf("symlink code=%q", ErrorCode(err))
	}
}

func capacityFixture(t *testing.T) (CapacityObserver, capacityReport) {
	base := time.Date(2026, 7, 30, 1, 0, 0, 0, time.UTC)
	zero, four, governs, notGoverning := int64(0), int64(4), true, false
	notApplicable := capacityReportDimension{Governs: &notGoverning}
	report := capacityReport{
		SchemaVersion: "codex_usage_accounting.v1", WindowRef: "window:one", Status: "exhausted", Quality: "measured", ArtifactRef: "artifact:capacity:one",
		ObservedAt: base, ExpiresAt: base.Add(time.Minute), ResetAt: base.Add(time.Hour), TryAgainAt: base.Add(30 * time.Minute),
		RateLimits: capacityReportLimits{
			Slots:   capacityReportDimension{Governs: &governs, Unit: "executions", Limit: &four, Remaining: &zero},
			Seconds: notApplicable, Messages: capacityReportDimension{}, Tokens: notApplicable, Credits: notApplicable,
		},
	}
	return CapacityObserver{ReportPath: t.TempDir() + "/capacity.json", MaxBytes: 4096, SourceRef: "source:codex", PoolRef: "pool:codex"}, report
}

func writeCapacityReport(t *testing.T, path string, report capacityReport) {
	data, _ := json.Marshal(report)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
