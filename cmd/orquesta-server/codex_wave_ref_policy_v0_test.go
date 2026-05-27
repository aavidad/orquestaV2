package main

import (
	"strings"
	"testing"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type fixedWaveEntropyV0 struct {
	value byte
}

func (entropy fixedWaveEntropyV0) Read(p []byte) (int, error) {
	for index := range p {
		p[index] = entropy.value
	}
	return len(p), nil
}

func TestCodexWaveRefV0NoUsaTimestampDeSegundoComoIdentidadV0(t *testing.T) {
	previous := codexWaveRefGeneratorV0
	codexWaveRefGeneratorV0 = orquestaruntime.NewRefGeneratorV0(
		orquestaruntime.ClockFuncV0(func() time.Time {
			return time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)
		}),
		fixedWaveEntropyV0{value: 0x01},
	)
	defer func() { codexWaveRefGeneratorV0 = previous }()

	first, err := codexWaveRefV0("")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := codexWaveRefV0("")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first == second {
		t.Fatalf("refs colisionaron: %q", first)
	}
	if first == "codex-wave-v0-20260524T100000Z" {
		t.Fatalf("wave_ref usa timestamp de segundo: %q", first)
	}
}

func TestCodexDirectorDefaultRefV0UsaGeneradorCompartidoSiNoHayWaveRefV0(t *testing.T) {
	previous := codexWaveRefGeneratorV0
	codexWaveRefGeneratorV0 = orquestaruntime.NewRefGeneratorV0(
		orquestaruntime.ClockFuncV0(func() time.Time {
			return time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)
		}),
		fixedWaveEntropyV0{value: 0x02},
	)
	defer func() { codexWaveRefGeneratorV0 = previous }()

	ref := codexDirectorDefaultRefV0("", "run", "")
	if ref == "run-20260524T100000Z" || ref == "run-" {
		t.Fatalf("ref por defecto insegura: %q", ref)
	}
	if !strings.HasPrefix(ref, "run-codex-wave-v0-director-wave-") {
		t.Fatalf("ref=%q", ref)
	}
}
