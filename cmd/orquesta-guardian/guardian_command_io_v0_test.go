package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGuardianCommandOutputV0AcotaTailYRedactaSensibles(t *testing.T) {
	dir := t.TempDir()
	config := guardianConfigV0{
		ProjectDir:                dir,
		StateDir:                  filepath.Join(dir, "guardian"),
		CurrentBin:                filepath.Join(dir, "current"),
		CandidateBin:              filepath.Join(dir, "candidate"),
		LastGoodBin:               filepath.Join(dir, "last-good"),
		CommandTimeout:            time.Second,
		CommandOutputMaxBytes:     96,
		EnvAllowlist:              []string{"PATH"},
		CommandEffectEvidenceRefs: []string{"evidence-ref-guardian-test-command-effect"},
		OccurredAt:                time.Now().UTC(),
	}
	config, err := normalizeGuardianConfigV0(config)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := ensureGuardianDirsV0(config); err != nil {
		t.Fatalf("dirs: %v", err)
	}
	command := "printf 'api_key=secret-token\\nHOME=" + shellQuoteV0(dir) + "\\n'; yes filler | head -n 40"
	result := runGuardianShellCommandV0(t.Context(), config, "build", command)
	if result.ExitCode != 0 {
		t.Fatalf("exit=%d error=%s", result.ExitCode, result.Error)
	}
	if !result.OutputTruncated || result.OutputReasonCode != guardianOutputTruncatedReasonV0 {
		t.Fatalf("sin truncado esperado: %+v", result)
	}
	body := mustReadGuardianTestFileV0(t, result.OutputPath)
	if strings.Contains(body, "secret-token") || strings.Contains(body, dir) {
		t.Fatalf("log no redactado: %q", body)
	}
	if len(body) > int(config.CommandOutputMaxBytes) {
		t.Fatalf("log excede presupuesto: got=%d max=%d", len(body), config.CommandOutputMaxBytes)
	}
}

func TestGuardianCommandEffectPolicyV0BloqueaShellNoCanonicoSinEvidencia(t *testing.T) {
	dir := t.TempDir()
	config, err := normalizeGuardianConfigV0(guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     filepath.Join(dir, "current"),
		CandidateBin:   filepath.Join(dir, "candidate"),
		LastGoodBin:    filepath.Join(dir, "last-good"),
		CommandTimeout: time.Second,
		EnvAllowlist:   []string{"PATH"},
		OccurredAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	result := runGuardianShellCommandV0(t.Context(), config, "build", "printf candidate > {candidate_bin}")
	if result.ExitCode == 0 ||
		result.ReasonCode != guardianCommandEffectBlockedReasonV0 ||
		result.EffectProfile != "build" ||
		result.EffectAuthorization != guardianCommandEffectBlockedV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestGuardianCommandIdentityV0UsaPerfilTemplateYAttempt(t *testing.T) {
	dir := t.TempDir()
	configA := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "state-a"),
		CurrentBin:     filepath.Join(dir, "current-a"),
		CandidateBin:   filepath.Join(dir, "candidate-a"),
		LastGoodBin:    filepath.Join(dir, "last-good-a"),
		CommandTimeout: time.Second,
		AttemptRef:     "guardian-attempt-ref-same",
		OccurredAt:     time.Now().UTC(),
	})
	configB := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "state-b"),
		CurrentBin:     filepath.Join(dir, "current-b"),
		CandidateBin:   filepath.Join(dir, "candidate-b"),
		LastGoodBin:    filepath.Join(dir, "last-good-b"),
		CommandTimeout: time.Second,
		AttemptRef:     "guardian-attempt-ref-same",
		OccurredAt:     time.Now().UTC(),
	})
	commandA := guardianCommandResultV0{Phase: "build", Command: guardianCanonicalBuildCommandV0(configA)}
	commandB := guardianCommandResultV0{Phase: "build", Command: guardianCanonicalBuildCommandV0(configB)}
	identityA := guardianCommandIdentityV0(configA, commandA)
	identityB := guardianCommandIdentityV0(configB, commandB)
	if identityA.CommandProfile != "build" ||
		identityA.TemplateRef == "" ||
		identityA.TemplateRef != identityB.TemplateRef ||
		identityA.CommandRef != identityB.CommandRef ||
		identityA.TemplateStatus != "" {
		t.Fatalf("identityA=%+v identityB=%+v", identityA, identityB)
	}

	configC := configB
	configC.AttemptRef = "guardian-attempt-ref-other"
	identityC := guardianCommandIdentityV0(configC, commandB)
	if identityC.TemplateRef != identityA.TemplateRef || identityC.CommandRef == identityA.CommandRef {
		t.Fatalf("identityA=%+v identityC=%+v", identityA, identityC)
	}

	changed := guardianCommandIdentityV0(configA, guardianCommandResultV0{
		Phase:   "build",
		Command: "printf candidate > " + shellQuoteV0(configA.CandidateBin),
	})
	if changed.TemplateStatus != guardianCommandTemplateChangedReasonV0 ||
		len(changed.TemplateEvidenceRef) == 0 {
		t.Fatalf("changed=%+v", changed)
	}
}

func TestGuardianCommandEnvV0NoHeredaSecretosPorDefault(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_HOME", "/tmp/private-codex-home")
	t.Setenv("SECRET_TOKEN", "secret-value")
	t.Setenv("HOME", "/tmp/private-home")
	dir := t.TempDir()
	config := guardianConfigV0{
		ProjectDir:            dir,
		StateDir:              filepath.Join(dir, "guardian"),
		CurrentBin:            filepath.Join(dir, "current"),
		CandidateBin:          filepath.Join(dir, "candidate"),
		LastGoodBin:           filepath.Join(dir, "last-good"),
		CommandTimeout:        time.Second,
		CommandOutputMaxBytes: defaultGuardianCommandOutputMaxBytesV0,
		EnvAllowlist:          defaultGuardianEnvAllowlistV0(),
		OccurredAt:            time.Now().UTC(),
	}
	env := guardianCommandEnvV0(config, []string{"ORQUESTA_GUARDIAN_REPAIR_PACKET=packet-ref"})
	joined := strings.Join(env, "\n")
	for _, forbidden := range []string{
		"ORQUESTA_CODEX_HOME=",
		"SECRET_TOKEN=",
		"HOME=/tmp/private-home",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("env hereda %s: %s", forbidden, joined)
		}
	}
	if !strings.Contains(joined, "ORQUESTA_GUARDIAN_REPAIR_PACKET=packet-ref") {
		t.Fatalf("env explicito ausente: %s", joined)
	}
	if os.Getenv("HOME") != "/tmp/private-home" {
		t.Fatalf("test setup mutado")
	}
}
