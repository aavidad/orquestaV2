package main

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGuardianPublicRefsV0EstablesEntreWorktreesTemporales(t *testing.T) {
	first := guardianPublicRefsFixtureV0(t, filepath.Join(t.TempDir(), "worktree-a"))
	second := guardianPublicRefsFixtureV0(t, filepath.Join(t.TempDir(), "worktree-b"))

	if first.ManifestRef == "" || first.RepairPacketRef == "" || len(first.Commands) == 0 ||
		first.Commands[0].OutputRef == "" || len(first.LocalDiagnostics) == 0 {
		t.Fatalf("refs publicas incompletas: %+v", first)
	}
	if first.ManifestRef != second.ManifestRef ||
		first.RepairPacketRef != second.RepairPacketRef ||
		first.Commands[0].OutputRef != second.Commands[0].OutputRef {
		t.Fatalf("refs cambian entre worktrees: first=%+v second=%+v", first, second)
	}
	if !reflect.DeepEqual(guardianDiagnosticRefsForTestV0(first), guardianDiagnosticRefsForTestV0(second)) {
		t.Fatalf("diagnosticos cambian entre worktrees: first=%+v second=%+v", first.LocalDiagnostics, second.LocalDiagnostics)
	}
	encoded := strings.Join([]string{
		first.ManifestRef,
		first.RepairPacketRef,
		first.Commands[0].OutputRef,
		strings.Join(guardianDiagnosticRefsForTestV0(first), "\n"),
	}, "\n")
	if strings.Contains(encoded, "worktree-a") || strings.Contains(encoded, "worktree-b") {
		t.Fatalf("refs contienen path local: %s", encoded)
	}
}

func TestGuardianPublicRefsV0NoInventaOutputRefSinHash(t *testing.T) {
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:   t.TempDir(),
		CurrentBin:   "orquesta-server",
		AttemptRef:   "guardian-attempt-ref-public-output-missing",
		OccurredAt:   time.Date(2026, 5, 27, 15, 0, 0, 0, time.UTC),
		SkipHealth:   true,
		Promote:      false,
		BuildCommand: "true",
	})
	commands := publicGuardianCommandsV0(config, []guardianCommandResultV0{{
		Phase:      "build",
		Command:    "true",
		ExitCode:   0,
		OutputPath: filepath.Join(config.StateDir, "logs", "missing.log"),
	}})
	if len(commands) != 1 || commands[0].OutputRef != "" {
		t.Fatalf("output_ref sin hash debe quedar vacia: %+v", commands)
	}
}

func guardianPublicRefsFixtureV0(t *testing.T, dir string) guardianPublicResultV0 {
	t.Helper()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		BuildCommand:   "printf stable-diagnostic >&2; exit 7",
		SkipHealth:     true,
		Promote:        true,
		AttemptRef:     "guardian-attempt-ref-public-stable",
		PromotionRef:   "promotion-ref-public-stable",
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 27, 14, 0, 0, 0, time.UTC),
	})
	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusCandidateFailedV0 || result.RepairPacketPath == "" {
		t.Fatalf("result=%+v", result)
	}
	return publicGuardianResultV0(config, result)
}

func guardianDiagnosticRefsForTestV0(public guardianPublicResultV0) []string {
	refs := make([]string, 0, len(public.LocalDiagnostics))
	for _, diagnostic := range public.LocalDiagnostics {
		refs = append(refs, diagnostic.Ref)
	}
	return refs
}
