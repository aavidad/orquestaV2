package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestAutoprogrammingPromotionGuardianTimeoutEnvPrecedenceV0(t *testing.T) {
	fileValue := "30s"
	tests := []struct {
		name         string
		canonicalKey string
		legacyKey    string
		canonical    string
		legacy       string
		fileValue    *string
		fallback     string
		want         string
	}{
		{
			name:         "canonical env wins",
			canonicalKey: envServerAutoprogrammingPromotionGuardianHealthTimeoutMSV0,
			legacyKey:    envServerAutoprogrammingPromotionGuardianHealthTimeoutV0,
			canonical:    "1200",
			legacy:       "15s",
			fileValue:    &fileValue,
			fallback:     "5s",
			want:         "1200ms",
		},
		{
			name:         "legacy env wins over file",
			canonicalKey: envServerAutoprogrammingPromotionGuardianCommandTimeoutMSV0,
			legacyKey:    envServerAutoprogrammingPromotionGuardianCommandTimeoutV0,
			legacy:       "45s",
			fileValue:    &fileValue,
			fallback:     "5s",
			want:         "45s",
		},
		{
			name:         "file wins over default",
			canonicalKey: envServerAutoprogrammingPromotionGuardianCommandTimeoutMSV0,
			legacyKey:    envServerAutoprogrammingPromotionGuardianCommandTimeoutV0,
			fileValue:    &fileValue,
			fallback:     "5s",
			want:         "30s",
		},
		{
			name:         "default when unset",
			canonicalKey: envServerAutoprogrammingPromotionGuardianHealthTimeoutMSV0,
			legacyKey:    envServerAutoprogrammingPromotionGuardianHealthTimeoutV0,
			fallback:     "5s",
			want:         "5s",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			unsetEnvForTestV0(t, test.canonicalKey)
			unsetEnvForTestV0(t, test.legacyKey)
			if test.canonical != "" {
				t.Setenv(test.canonicalKey, test.canonical)
			}
			if test.legacy != "" {
				t.Setenv(test.legacyKey, test.legacy)
			}
			if got := autoprogrammingPromotionGuardianTimeoutFromEnvOrProjectConfigV0(
				test.canonicalKey, test.legacyKey, test.fileValue, test.fallback,
			); got != test.want {
				t.Fatalf("timeout=%q want=%q", got, test.want)
			}
		})
	}
}

func unsetEnvForTestV0(t *testing.T, key string) {
	t.Helper()
	previous, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, previous)
			return
		}
		_ = os.Unsetenv(key)
	})
}

func TestAutoprogrammingPromotionGuardianEnvV0TodasClavesRegistradasComoChildProcess(t *testing.T) {
	request := fullAutoprogrammingPromotionGuardianEnvRequestForTestV0()
	env := autoprogrammingPromotionGuardianEnvV0(nil, request)
	seen := map[string]bool{}

	for _, item := range env {
		key, _, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			t.Fatalf("env invalido: %q", item)
		}
		metadata, registered := autoprogrammingPromotionGuardianChildEnvRegistryV0[key]
		if !registered {
			t.Fatalf("env guardian sin registry child-process: %s", key)
		}
		if metadata.Scope != autoprogrammingPromotionGuardianChildEnvScopeV0 {
			t.Fatalf("env guardian %s con scope=%q", key, metadata.Scope)
		}
		if strings.TrimSpace(metadata.Classification) == "" ||
			strings.TrimSpace(metadata.Label) == "" ||
			strings.TrimSpace(metadata.Description) == "" {
			t.Fatalf("metadata incompleta para %s: %+v", key, metadata)
		}
		seen[key] = true
	}

	for _, key := range autoprogrammingPromotionGuardianChildEnvRegistryKeysV0 {
		if !seen[key] {
			t.Fatalf("registry child-process no cubierto por env guardian completo: %s", key)
		}
	}
	if len(seen) != len(autoprogrammingPromotionGuardianChildEnvRegistryV0) {
		t.Fatalf("env=%d registry=%d", len(seen), len(autoprogrammingPromotionGuardianChildEnvRegistryV0))
	}
}

func TestAutoprogrammingPromotionGuardianChildEnvRegistryV0NoPublicaValoresCrudos(t *testing.T) {
	request := fullAutoprogrammingPromotionGuardianEnvRequestForTestV0()
	publicRegistry := autoprogrammingPromotionGuardianChildEnvPublicRegistryV0()
	rawPublicRegistry, err := json.Marshal(publicRegistry)
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	body := string(rawPublicRegistry)

	for _, forbidden := range rawValuesAutoprogrammingPromotionGuardianEnvForTestV0(request) {
		if strings.Contains(body, forbidden) {
			t.Fatalf("registry child-process publica valor crudo %q: %s", forbidden, body)
		}
	}
	if !strings.Contains(body, envAutoprogrammingPromotionGuardianProjectDirV0) ||
		!strings.Contains(body, autoprogrammingPromotionGuardianChildEnvScopeV0) {
		t.Fatalf("registry child-process no publica metadata basica: %s", body)
	}
}

func fullAutoprogrammingPromotionGuardianEnvRequestForTestV0() serverAutoprogrammingPromotionGuardianRequestV0 {
	return serverAutoprogrammingPromotionGuardianRequestV0{
		ProjectDir:                 "/raw-private/guardian/project",
		StateDir:                   "/raw-private/guardian/state",
		CurrentBin:                 "/raw-private/guardian/current-bin",
		CandidateBin:               "/raw-private/guardian/candidate-bin",
		LastGoodBin:                "/raw-private/guardian/last-good-bin",
		ArtifactRoot:               "/raw-private/guardian/artifacts",
		BuildCommand:               "raw-private-build-command --token=secret-build",
		TestCommands:               []string{"raw-private-test-command-one", "raw-private-test-command-two"},
		HealthTimeout:              "17s",
		CommandTimeout:             "19s",
		ArtifactMaxBytes:           "23000",
		RepairCommand:              "raw-private-repair-command --token=secret-repair",
		RepairCodex:                true,
		RepairCodexWriteSet:        []string{"raw-private/write-set-one", "raw-private/write-set-two"},
		RepairCodexRequiredTests:   []string{"raw-private-repair-test-one", "raw-private-repair-test-two"},
		RepairCodexWorktreeRef:     "worktree-ref-raw-private-repair",
		RepairCodexBranchRef:       "branch-ref-raw-private-repair",
		RepairCodexRunRef:          "run-ref-raw-private-repair",
		RepairCodexPromotionRef:    "promotion-ref-raw-private-repair",
		RepairCodexSandbox:         "workspace-write-private",
		RepairCodexReasoning:       "medium-private",
		RepairCodexRuntimeDir:      "/raw-private/guardian/repair-runtime",
		RepairCodexAllowBroad:      true,
		RepairCodexSandboxEvidence: "evidence-ref-raw-private-sandbox",
		CommandEffectEvidenceRefs:  []string{"evidence-ref-raw-private-effect-one", "evidence-ref-raw-private-effect-two"},
		SkipHealthEvidenceRefs:     []string{"evidence-ref-raw-private-skip-one", "evidence-ref-raw-private-skip-two"},
		PromotionRef:               "promotion-ref-raw-private",
		RunRef:                     "run-ref-raw-private",
		WorktreeRef:                "worktree-ref-raw-private",
		BranchRef:                  "branch-ref-raw-private",
		EvidenceRefs:               []string{"evidence-ref-raw-private-promotion-one", "evidence-ref-raw-private-promotion-two"},
	}
}

func rawValuesAutoprogrammingPromotionGuardianEnvForTestV0(
	request serverAutoprogrammingPromotionGuardianRequestV0,
) []string {
	return []string{
		request.ProjectDir,
		request.StateDir,
		request.CurrentBin,
		request.CandidateBin,
		request.LastGoodBin,
		request.ArtifactRoot,
		request.BuildCommand,
		strings.Join(request.TestCommands, "\n"),
		request.HealthTimeout,
		request.CommandTimeout,
		request.ArtifactMaxBytes,
		request.RepairCommand,
		strings.Join(request.RepairCodexWriteSet, ","),
		strings.Join(request.RepairCodexRequiredTests, "\n"),
		request.RepairCodexWorktreeRef,
		request.RepairCodexBranchRef,
		request.RepairCodexRunRef,
		request.RepairCodexPromotionRef,
		request.RepairCodexSandbox,
		request.RepairCodexReasoning,
		request.RepairCodexRuntimeDir,
		request.RepairCodexSandboxEvidence,
		strings.Join(request.CommandEffectEvidenceRefs, ","),
		strings.Join(request.SkipHealthEvidenceRefs, ","),
		request.PromotionRef,
		request.RunRef,
		request.WorktreeRef,
		request.BranchRef,
		strings.Join(request.EvidenceRefs, ","),
	}
}
