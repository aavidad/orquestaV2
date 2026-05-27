package orquestaruntime

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type failingEntropyV0 struct{}

func (failingEntropyV0) Read([]byte) (int, error) {
	return 0, errors.New("entropy unavailable")
}

type fixedEntropyV0 struct {
	value byte
}

func (entropy fixedEntropyV0) Read(p []byte) (int, error) {
	for index := range p {
		p[index] = entropy.value
	}
	return len(p), nil
}

func TestRefGeneratorV0UsaEntropiaYScopeOpacoV0(t *testing.T) {
	generator := NewRefGeneratorV0(ClockFuncV0(func() time.Time {
		return time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)
	}), fixedEntropyV0{value: 0xab})

	ref, issues := generator.NextRefV0("run-ref", "App A", "request/1")
	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
	if ref != "run-ref-app-a-request-1-abababababababab" {
		t.Fatalf("ref=%q", ref)
	}
}

func TestRefGeneratorV0FallbackDegradadoNoEsSoloUnixNanoV0(t *testing.T) {
	generator := NewRefGeneratorV0(ClockFuncV0(func() time.Time {
		return time.Date(2026, 5, 24, 10, 0, 0, 123, time.UTC)
	}), failingEntropyV0{})

	first, firstIssues := generator.NextRefV0("req-cli", "mutation")
	second, secondIssues := generator.NextRefV0("req-cli", "mutation")
	if first == second || !strings.Contains(first, "degraded") || !strings.Contains(second, "000002") {
		t.Fatalf("fallback no observable o no monotonico: first=%q second=%q", first, second)
	}
	if len(firstIssues) != 1 || firstIssues[0].Code != RefGenerationEntropyUnavailableV0 {
		t.Fatalf("first issues=%+v", firstIssues)
	}
	if len(secondIssues) != 1 || secondIssues[0].Code != RefGenerationEntropyUnavailableV0 {
		t.Fatalf("second issues=%+v", secondIssues)
	}
}

func TestRefGeneratorV0DetectaColisionV0(t *testing.T) {
	generator := NewRefGeneratorV0(SystemClockV0{}, fixedEntropyV0{value: 0x01})

	first, _ := generator.NextRefV0("wave-ref", "same")
	second, issues := generator.NextRefV0("wave-ref", "same")
	if first == second {
		t.Fatalf("collision no resuelta: %q", first)
	}
	if len(issues) != 1 || issues[0].Code != RefGenerationCollisionDetectedV0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestNowUTCV0NormalizaUTCYZeroUsaSistemaV0(t *testing.T) {
	local := time.FixedZone("test", 3600)
	got := NowUTCV0(ClockFuncV0(func() time.Time {
		return time.Date(2026, 5, 24, 11, 0, 0, 0, local)
	}))
	if got.Location() != time.UTC || FormatRFC3339UTCV0(got) != "2026-05-24T10:00:00Z" {
		t.Fatalf("utc=%s loc=%s", got, got.Location())
	}
	if NowUTCV0(ClockFuncV0(func() time.Time { return time.Time{} })).IsZero() {
		t.Fatalf("zero clock no debe propagarse")
	}
}

var _ io.Reader = failingEntropyV0{}
