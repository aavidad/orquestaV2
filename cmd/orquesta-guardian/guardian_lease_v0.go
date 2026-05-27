package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const guardianLeaseSchemaVersionV0 = "orquesta_guardian_promotion_lease.v0"

type guardianLeaseReceiptV0 struct {
	SchemaVersion string `json:"schema_version"`
	LeaseRef      string `json:"lease_ref"`
	OwnerRef      string `json:"owner_ref"`
	AttemptRef    string `json:"attempt_ref,omitempty"`
	PromotionRef  string `json:"promotion_ref,omitempty"`
	RunRef        string `json:"run_ref,omitempty"`
	WorktreeRef   string `json:"worktree_ref,omitempty"`
	BranchRef     string `json:"branch_ref,omitempty"`
	Operation     string `json:"operation"`
	Phase         string `json:"phase"`
	Status        string `json:"status"`
	Freshness     string `json:"freshness"`
	AcquiredAt    string `json:"acquired_at,omitempty"`
	DeadlineAt    string `json:"deadline_at"`
	ReleasedAt    string `json:"released_at,omitempty"`
	PayloadHash   string `json:"payload_hash"`
	StateRef      string `json:"state_ref"`
	TargetRef     string `json:"target_ref,omitempty"`
	ReasonCode    string `json:"reason_code,omitempty"`
}

func acquireGuardianPromotionLeaseV0(config guardianConfigV0, operation string) (guardianLeaseReceiptV0, error) {
	now := guardianLeaseNowV0(config)
	lease := newGuardianLeaseReceiptV0(config, operation, now)
	path := guardianLeasePathV0(config)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return lease, err
	}
	if existing, ok := readGuardianLeaseV0(path); ok && existing.activeAtV0(now) {
		existing.Status = guardianStatusLeaseBusyV0
		existing.ReasonCode = guardianStatusLeaseBusyV0
		return existing, errors.New(guardianStatusLeaseBusyV0)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		if existing, ok := readGuardianLeaseV0(path); ok && existing.activeAtV0(now) {
			existing.Status = guardianStatusLeaseBusyV0
			existing.ReasonCode = guardianStatusLeaseBusyV0
			return existing, errors.New(guardianStatusLeaseBusyV0)
		}
		_ = os.Remove(path)
		file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	}
	if err != nil {
		return lease, err
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(lease); err != nil {
		_ = os.Remove(path)
		return lease, err
	}
	_ = file.Sync()
	_ = fsyncGuardianDirV0(filepath.Dir(path))
	return lease, nil
}

func verifyGuardianPromotionLeaseV0(config guardianConfigV0, lease guardianLeaseReceiptV0, phase string) error {
	current, ok := readGuardianLeaseV0(guardianLeasePathV0(config))
	if !ok || current.PayloadHash != lease.PayloadHash || !current.activeAtV0(guardianLeaseNowV0(config)) {
		return errors.New(guardianStatusLeaseLostV0)
	}
	current.Phase = phase
	return nil
}

func releaseGuardianPromotionLeaseV0(config guardianConfigV0, lease *guardianLeaseReceiptV0) {
	if lease == nil || lease.PayloadHash == "" {
		return
	}
	path := guardianLeasePathV0(config)
	current, ok := readGuardianLeaseV0(path)
	if !ok || current.PayloadHash != lease.PayloadHash {
		return
	}
	current.Status = "released"
	current.ReleasedAt = guardianLeaseNowV0(config).Format(time.RFC3339)
	_ = writeJSONFileV0(path+".released", current)
	_ = os.Remove(path)
}

func guardianLeasePathV0(config guardianConfigV0) string {
	return filepath.Join(config.StateDir, "leases", "guardian-promotion-lease.json")
}

func readGuardianLeaseV0(path string) (guardianLeaseReceiptV0, bool) {
	body, err := os.ReadFile(path)
	if err != nil {
		return guardianLeaseReceiptV0{}, false
	}
	var lease guardianLeaseReceiptV0
	if err := json.Unmarshal(body, &lease); err != nil {
		return guardianLeaseReceiptV0{}, false
	}
	return lease, lease.PayloadHash != ""
}

func newGuardianLeaseReceiptV0(config guardianConfigV0, operation string, acquired time.Time) guardianLeaseReceiptV0 {
	deadline := acquired.Add(config.LeaseTTL)
	owner := firstNonEmptyV0(config.PromotionRef, config.AttemptRef)
	payloadHash := guardianLeasePayloadHashV0(config, operation, owner, deadline)
	if owner == "" {
		owner = "attempt-ref-guardian-" + payloadHash[:16]
	}
	return guardianLeaseReceiptV0{
		SchemaVersion: guardianLeaseSchemaVersionV0,
		LeaseRef:      "lease-ref-guardian-promotion-" + payloadHash[:16],
		OwnerRef:      owner,
		AttemptRef:    config.AttemptRef,
		PromotionRef:  config.PromotionRef,
		RunRef:        config.RunRef,
		WorktreeRef:   config.WorktreeRef,
		BranchRef:     config.BranchRef,
		Operation:     operation,
		Phase:         "acquired",
		Status:        "active",
		Freshness:     "deadline_controlled",
		AcquiredAt:    acquired.Format(time.RFC3339),
		DeadlineAt:    deadline.Format(time.RFC3339),
		PayloadHash:   payloadHash,
		StateRef:      guardianPathRefV0("guardian-state-ref", config.StateDir),
		TargetRef:     guardianLeaseTargetRefV0(config, operation),
	}
}

func guardianLeaseNowV0(config guardianConfigV0) time.Time {
	if config.OccurredAt.IsZero() {
		return time.Now().UTC()
	}
	return config.OccurredAt.UTC()
}

func (lease guardianLeaseReceiptV0) activeAtV0(now time.Time) bool {
	if lease.Status != "active" {
		return false
	}
	deadline, err := time.Parse(time.RFC3339, lease.DeadlineAt)
	return err == nil && now.Before(deadline)
}

func guardianLeasePayloadHashV0(
	config guardianConfigV0,
	operation string,
	owner string,
	deadline time.Time,
) string {
	payload := strings.Join([]string{
		operation,
		owner,
		config.PromotionRef,
		config.AttemptRef,
		config.RunRef,
		config.WorktreeRef,
		config.BranchRef,
		guardianPathRefV0("guardian-state-ref", config.StateDir),
		guardianLeaseTargetRefV0(config, operation),
		deadline.Format(time.RFC3339),
	}, "|")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func guardianLeaseTargetRefV0(config guardianConfigV0, operation string) string {
	target := config.CurrentBin + "|" + config.LastGoodBin
	if operation == "shutdown" {
		target = config.ServerAddr
	}
	if operation == "check-promote" {
		target += "|" + config.CandidateBin
	}
	return guardianPathRefV0("guardian-target-ref", target)
}

func firstNonEmptyV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
