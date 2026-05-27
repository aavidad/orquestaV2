package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGuardianV0PromocionaCandidatoYGuardaLastGood(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:             dir,
		StateDir:               filepath.Join(dir, "guardian"),
		CurrentBin:             current,
		BuildCommand:           "printf candidate > {candidate_bin}",
		TestCommands:           []string{"true"},
		SkipHealth:             true,
		SkipHealthEvidenceRefs: testGuardianSkipHealthEvidenceRefsV0(),
		Promote:                true,
		CommandTimeout:         time.Second,
		OccurredAt:             time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusPromotedBreakglassV0 || !result.Promoted {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "candidate" {
		t.Fatalf("current=%q", got)
	}
	if got := mustReadGuardianTestFileV0(t, config.LastGoodBin); got != "old" {
		t.Fatalf("last_good=%q", got)
	}
	if _, err := os.Stat(config.ManifestPath); err != nil {
		t.Fatalf("manifest: %v", err)
	}
}

func TestGuardianV0BuildFallidoNoPromocionaYCreaRepairPacket(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	repairSeen := filepath.Join(dir, "repair_seen.txt")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		BuildCommand:   "printf build-failed >&2; exit 7",
		RepairCommand:  "printf \"$ORQUESTA_GUARDIAN_REPAIR_PACKET\" > " + shellQuoteV0(repairSeen),
		SkipHealth:     true,
		Promote:        true,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 24, 12, 1, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusCandidateFailedV0 ||
		result.Phase != "build" ||
		result.RepairPacketPath == "" ||
		!result.RepairStarted {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "old" {
		t.Fatalf("current modificado=%q", got)
	}
	if got := mustReadGuardianTestFileV0(t, repairSeen); got != result.RepairPacketPath {
		t.Fatalf("repair_seen=%q packet=%q", got, result.RepairPacketPath)
	}
	packet := mustReadGuardianRepairPacketForTestV0(t, result.RepairPacketPath)
	if packet.FailurePhase != "build" || !strings.Contains(packet.Summary, "build") {
		t.Fatalf("packet=%+v", packet)
	}
	encodedPacket := mustReadGuardianTestFileV0(t, result.RepairPacketPath)
	for _, leaked := range []string{dir, current, config.CandidateBin, config.LastGoodBin, config.BuildCommand} {
		if strings.Contains(encodedPacket, leaked) {
			t.Fatalf("repair packet contiene diagnostico local no redactado %q: %s", leaked, encodedPacket)
		}
	}
	if packet.RedactionLevel != "refs_only" ||
		packet.Freshness == "" ||
		len(packet.RequiredCommandRefs) == 0 ||
		len(packet.LocalDiagnostics) == 0 {
		t.Fatalf("repair packet publico incompleto: %+v", packet)
	}
}

func TestGuardianV0CLIEmiteResultadoPublicoRedactado(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	var stdout bytes.Buffer
	exitCode := runMain([]string{
		"check-promote",
		"--project-dir", dir,
		"--state-dir", filepath.Join(dir, "guardian"),
		"--current-bin", current,
		"--build-command", "printf token-value >&2; exit 7",
		"--skip-health",
	}, &stdout, io.Discard)
	if exitCode != 1 {
		t.Fatalf("exitCode=%d stdout=%s", exitCode, stdout.String())
	}
	var public guardianPublicResultV0
	if err := json.Unmarshal(stdout.Bytes(), &public); err != nil {
		t.Fatalf("decode public: %v stdout=%s", err, stdout.String())
	}
	encoded := stdout.String()
	for _, leaked := range []string{dir, current, "printf token-value", "token-value"} {
		if strings.Contains(encoded, leaked) {
			t.Fatalf("resultado publico contiene valor no redactado %q: %s", leaked, encoded)
		}
	}
	if public.RedactionLevel != "refs_only" ||
		public.ConfigEffective == nil ||
		!public.ConfigEffective.Promote ||
		!public.ConfigEffective.SkipHealth ||
		public.ConfigEffective.CommandTimeoutMS == 0 ||
		public.ManifestRef == "" ||
		public.RepairPacketRef == "" ||
		len(public.Commands) == 0 ||
		public.Commands[0].CommandRef == "" ||
		public.Commands[0].CommandProfile != "build" ||
		public.Commands[0].TemplateRef == "" ||
		public.Commands[0].TemplateStatus != guardianCommandTemplateChangedReasonV0 ||
		len(public.Commands[0].TemplateEvidenceRefs) == 0 ||
		public.Commands[0].OutputRef == "" ||
		len(public.LocalDiagnostics) == 0 {
		t.Fatalf("public=%+v", public)
	}
}

func TestGuardianV0ConfigEnvEstrictoBloqueaValoresInvalidos(t *testing.T) {
	t.Setenv("ORQUESTA_GUARDIAN_PROMOTE", "tru")
	var stderr bytes.Buffer
	exitCode := runMain([]string{"check-promote", "--current-bin", "orquesta-server"}, io.Discard, &stderr)
	if exitCode != 2 {
		t.Fatalf("exitCode=%d stderr=%s", exitCode, stderr.String())
	}
	if got := stderr.String(); !strings.Contains(got, guardianConfigInvalidBoolV0) ||
		strings.Contains(got, "tru") {
		t.Fatalf("stderr no es reason compacto: %s", got)
	}
}

func TestGuardianV0ConfigFlagEstrictoBloqueaBudgetInvalido(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := runMain([]string{
		"shutdown-server",
		"--command-output-max-bytes", "0",
	}, io.Discard, &stderr)
	if exitCode != 2 {
		t.Fatalf("exitCode=%d stderr=%s", exitCode, stderr.String())
	}
	if got := stderr.String(); !strings.Contains(got, guardianConfigInvalidBudgetV0) {
		t.Fatalf("stderr=%s", got)
	}
}

func TestGuardianV0HealthcheckFallidoNoPromociona(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		BuildCommand:   "printf '#!/bin/sh\\nexit 0\\n' > {candidate_bin}; chmod +x {candidate_bin}",
		Promote:        true,
		SkipHealth:     false,
		HealthTimeout:  100 * time.Millisecond,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 24, 12, 2, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusCandidateFailedV0 || result.Phase != "healthcheck" {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "old" {
		t.Fatalf("current modificado=%q", got)
	}
	if result.RepairPacketPath == "" {
		t.Fatalf("repair packet vacio: %+v", result)
	}
}

func TestGuardianV0SkipHealthPromoteRequiereBreakglassV0(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		BuildCommand:   "printf candidate > {candidate_bin}",
		SkipHealth:     true,
		Promote:        true,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 27, 16, 0, 0, 0, time.UTC),
	})
	config.SkipHealthEvidenceRefs = nil

	result := runGuardianCheckPromoteV0(context.Background(), config)
	public := publicGuardianResultV0(config, result)
	if result.Status != guardianStatusCandidateFailedV0 ||
		result.Phase != "healthcheck" ||
		result.Promoted ||
		!containsStringForGuardianTestV0(public.ReasonCodes, "guardian_healthcheck_required") {
		t.Fatalf("result=%+v public=%+v", result, public)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "old" {
		t.Fatalf("current modificado=%q", got)
	}
}

func TestGuardianV0RestauraLastGood(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 24, 12, 3, 0, 0, time.UTC),
	})
	mustWriteGuardianTestFileV0(t, config.LastGoodBin, "last-good")

	result := restoreLastGoodCommandV0(config)
	if result.Status != guardianStatusRestoredV0 || !result.Restored {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "last-good" {
		t.Fatalf("current=%q", got)
	}
}

func mustGuardianConfigForTestV0(t *testing.T, config guardianConfigV0, requireCurrentBin ...bool) guardianConfigV0 {
	t.Helper()
	if len(config.CommandEffectEvidenceRefs) == 0 {
		config.CommandEffectEvidenceRefs = []string{"evidence-ref-guardian-test-command-effect"}
	}
	if config.SkipHealth && config.Promote && len(config.SkipHealthEvidenceRefs) == 0 {
		config.SkipHealthEvidenceRefs = []string{"evidence-ref-guardian-test-skip-health"}
	}
	require := true
	if len(requireCurrentBin) > 0 {
		require = requireCurrentBin[0]
	}
	normalized, err := normalizeGuardianConfigForCommandV0(config, require)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	return normalized
}

func testGuardianSkipHealthEvidenceRefsV0() []string {
	return []string{"evidence-ref-guardian-skip-health-test-breakglass"}
}

func mustWriteGuardianTestFileV0(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustReadGuardianTestFileV0(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}

func mustReadGuardianRepairPacketForTestV0(t *testing.T, path string) guardianRepairPacketV0 {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read packet: %v", err)
	}
	var packet guardianRepairPacketV0
	if err := json.Unmarshal(body, &packet); err != nil {
		t.Fatalf("decode packet: %v", err)
	}
	return packet
}
