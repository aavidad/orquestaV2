package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestAutoprogrammingPromotionGuardianTimeoutsFromTypedConfigOverrideProcessDefaultsV0(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, serverProjectConfigFileNameV0)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"autoprogramming":{"promotion":{"guardian":{
			"enabled":true,
			"health_timeout":"30s",
			"command_timeout":"2m"
		}}}
	}`
	if err := os.WriteFile(configPath, []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	guardian := autoprogrammingPromotionGuardianFromEnvV0(orquestaserver.ConfigV0{
		ProjectWorkDir:        root,
		ProjectConfigFilePath: configPath,
	})
	if !guardian.Enabled || guardian.HealthTimeout != "30s" || guardian.CommandTimeout != "2m" {
		t.Fatalf("guardian=%+v", guardian)
	}
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
