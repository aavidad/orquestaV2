package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	codexWaveProcessProofSchemaVersionV0 = "orquesta_codex_wave_process_proof.v0"
	codexWaveProcessProofFileNameV0      = "codex_wave_process_proof_v0.json"
)

type codexWaveProcessProofV0 struct {
	SchemaVersion   string `json:"schema_version"`
	WaveRef         string `json:"wave_ref"`
	AgentRef        string `json:"agent_ref"`
	ProcessRef      string `json:"process_ref"`
	SessionRef      string `json:"session_ref"`
	LaunchRef       string `json:"launch_ref"`
	ProcessProofRef string `json:"process_proof_ref"`
	CommandRef      string `json:"command_ref"`
	RuntimeWorkDir  string `json:"runtime_work_dir"`
	StartRef        string `json:"start_ref"`
	OwnerRef        string `json:"owner_ref"`
	PID             int    `json:"pid"`
	StartedAt       string `json:"started_at"`
	Digest          string `json:"digest"`
}

func codexWaveAttachProcessProofV0(
	agent *codexWaveAgentSummaryV0,
	config codexWaveConfigV0,
	index int,
) error {
	if agent == nil {
		return errors.New("process_proof_agent_missing")
	}
	proof := codexWaveBuildProcessProofV0(*agent, config, index)
	proof.Digest = codexWaveProcessProofDigestV0(proof)
	data, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		return err
	}
	path := codexWaveProcessProofPathV0(agent.RuntimeWorkDir)
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return err
	}
	agent.ProcessProofRef = proof.ProcessProofRef
	agent.ProcessProofDigest = proof.Digest
	return nil
}

func codexWaveBuildProcessProofV0(
	agent codexWaveAgentSummaryV0,
	config codexWaveConfigV0,
	index int,
) codexWaveProcessProofV0 {
	part := safeCodexWavePurgeRefPartV0(config.WaveRef) + "-" + strconv.Itoa(index)
	return codexWaveProcessProofV0{
		SchemaVersion:   codexWaveProcessProofSchemaVersionV0,
		WaveRef:         config.WaveRef,
		AgentRef:        agent.AgentRef,
		ProcessRef:      agent.ProcessRef,
		SessionRef:      agent.SessionRef,
		LaunchRef:       agent.LaunchRef,
		ProcessProofRef: "process-proof-ref-" + part,
		CommandRef:      "command-ref-codex-wave-v0",
		RuntimeWorkDir:  agent.RuntimeWorkDir,
		StartRef:        "process-start-ref-" + part,
		OwnerRef:        "process-owner-ref-" + part,
		PID:             agent.PID,
		StartedAt:       agent.StartedAt,
	}
}

func codexWaveValidateAgentProcessProofV0(
	summary codexWaveLaunchSummaryV0,
	agent codexWaveAgentSummaryV0,
) error {
	if !codexWaveRuntimeUnderAllowedRootV0(summary.RuntimeWorkDir, summary.ProjectWorkDir) {
		return errors.New("blocked_registry_untrusted")
	}
	if strings.TrimSpace(agent.ProcessProofRef) == "" ||
		strings.TrimSpace(agent.ProcessProofDigest) == "" {
		return errors.New("blocked_registry_untrusted")
	}
	path := codexWaveProcessProofPathV0(agent.RuntimeWorkDir)
	if !pathUnderDirV0(summary.RuntimeWorkDir, path) || !pathUnderDirV0(agent.RuntimeWorkDir, path) {
		return errors.New("blocked_registry_untrusted")
	}
	proof, err := codexWaveReadProcessProofV0(path)
	if err != nil {
		return errors.New("blocked_registry_untrusted")
	}
	if proof.Digest != codexWaveProcessProofDigestV0(proof) ||
		proof.Digest != agent.ProcessProofDigest {
		return errors.New("blocked_registry_untrusted")
	}
	if !codexWaveProofMatchesAgentV0(summary, agent, proof) {
		return errors.New("blocked_registry_untrusted")
	}
	return nil
}

func codexWaveReadProcessProofV0(path string) (codexWaveProcessProofV0, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return codexWaveProcessProofV0{}, err
	}
	var proof codexWaveProcessProofV0
	if err := json.Unmarshal(data, &proof); err != nil {
		return codexWaveProcessProofV0{}, err
	}
	return proof, nil
}

func codexWaveProofMatchesAgentV0(
	summary codexWaveLaunchSummaryV0,
	agent codexWaveAgentSummaryV0,
	proof codexWaveProcessProofV0,
) bool {
	return proof.SchemaVersion == codexWaveProcessProofSchemaVersionV0 &&
		proof.WaveRef == summary.WaveRef &&
		proof.AgentRef == agent.AgentRef &&
		proof.ProcessRef == agent.ProcessRef &&
		proof.SessionRef == agent.SessionRef &&
		proof.LaunchRef == agent.LaunchRef &&
		proof.ProcessProofRef == agent.ProcessProofRef &&
		proof.RuntimeWorkDir == agent.RuntimeWorkDir &&
		proof.PID == agent.PID &&
		proof.StartedAt == agent.StartedAt &&
		strings.TrimSpace(proof.CommandRef) != "" &&
		strings.TrimSpace(proof.StartRef) != "" &&
		strings.TrimSpace(proof.OwnerRef) != ""
}

func codexWaveProcessProofDigestV0(proof codexWaveProcessProofV0) string {
	proof.Digest = ""
	canonical := strings.Join([]string{
		proof.SchemaVersion,
		proof.WaveRef,
		proof.AgentRef,
		proof.ProcessRef,
		proof.SessionRef,
		proof.LaunchRef,
		proof.ProcessProofRef,
		proof.CommandRef,
		proof.RuntimeWorkDir,
		proof.StartRef,
		proof.OwnerRef,
		strconv.Itoa(proof.PID),
		proof.StartedAt,
	}, "\x00")
	sum := sha256.Sum256([]byte(canonical))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func codexWaveProcessProofPathV0(agentRuntimeDir string) string {
	return filepath.Join(agentRuntimeDir, codexWaveProcessProofFileNameV0)
}

func codexWaveProofErrorV0(agentRef string, err error) codexWavePublicErrorV0 {
	code := "blocked_registry_untrusted"
	if err != nil && err.Error() != "" {
		code = err.Error()
	}
	return codexWavePublicErrorV0{AgentRef: agentRef, Code: code}
}

func codexWaveProcessProofAttachErrorV0(agentRef string, err error) codexWavePublicErrorV0 {
	return codexWavePublicErrorV0{
		AgentRef: agentRef,
		Code:     "process_proof_write_failed",
		Message:  fmt.Sprintf("%T", err),
	}
}
