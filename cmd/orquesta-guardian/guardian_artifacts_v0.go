package main

import (
	"errors"
	"os"
)

const (
	guardianArtifactManifestSchemaVersionV0 = "orquesta_guardian_artifact_manifest.v0"
	defaultGuardianArtifactMaxBytesV0       = int64(512 * 1024 * 1024)
)

type guardianArtifactManifestV0 struct {
	SchemaVersion   string                   `json:"schema_version"`
	Status          string                   `json:"status"`
	MaxBytes        int64                    `json:"max_bytes"`
	Candidate       guardianArtifactDigestV0 `json:"candidate,omitempty"`
	PreviousCurrent guardianArtifactDigestV0 `json:"previous_current,omitempty"`
	LastGood        guardianArtifactDigestV0 `json:"last_good,omitempty"`
	Current         guardianArtifactDigestV0 `json:"current,omitempty"`
	Operation       string                   `json:"operation,omitempty"`
	ReasonCode      string                   `json:"reason_code,omitempty"`
}

type guardianArtifactDigestV0 struct {
	Ref       string `json:"ref,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	SHA256    string `json:"sha256,omitempty"`
	Mode      string `json:"mode,omitempty"`
}

type guardianArtifactCopyResultV0 struct {
	Source guardianArtifactDigestV0
	Dest   guardianArtifactDigestV0
}

type guardianArtifactPolicyErrorV0 struct {
	Code string
	Err  error
}

func (err guardianArtifactPolicyErrorV0) Error() string {
	if err.Err == nil {
		return err.Code
	}
	return err.Code + ": " + err.Err.Error()
}

func (err guardianArtifactPolicyErrorV0) Unwrap() error { return err.Err }

func promoteGuardianCandidateArtifactsV0(
	config guardianConfigV0,
) (*guardianArtifactManifestV0, error) {
	manifest := baseGuardianArtifactManifestV0(config, "promote")
	candidate, err := inspectGuardianArtifactV0(config, config.CandidateBin)
	if err != nil {
		manifest.Status = guardianStatusCandidateFailedV0
		manifest.ReasonCode = guardianArtifactReasonCodeV0(err)
		return manifest, err
	}
	manifest.Candidate = candidate
	if current, err := inspectGuardianArtifactV0(config, config.CurrentBin); err == nil {
		manifest.PreviousCurrent = current
		copyResult, err := copyGuardianArtifactAtomicV0(config, config.CurrentBin, config.LastGoodBin, 0o700)
		if err != nil {
			manifest.Status = guardianStatusLastGoodUnverifiedV0
			manifest.ReasonCode = guardianArtifactReasonCodeV0(err)
			return manifest, err
		}
		manifest.LastGood = copyResult.Dest
	} else if !errors.Is(err, os.ErrNotExist) {
		manifest.Status = guardianStatusLastGoodUnverifiedV0
		manifest.ReasonCode = guardianArtifactReasonCodeV0(err)
		return manifest, err
	}
	copyResult, err := copyGuardianArtifactAtomicV0(config, config.CandidateBin, config.CurrentBin, 0o700)
	if err != nil {
		manifest.Status = guardianStatusPromotionIncompleteV0
		manifest.ReasonCode = guardianArtifactReasonCodeV0(err)
		return manifest, err
	}
	manifest.Candidate = copyResult.Source
	manifest.Current = copyResult.Dest
	manifest.Status = guardianStatusPromotedV0
	return manifest, nil
}

func restoreGuardianLastGoodArtifactsV0(
	config guardianConfigV0,
) (*guardianArtifactManifestV0, error) {
	manifest := baseGuardianArtifactManifestV0(config, "restore")
	lastGood, err := inspectGuardianArtifactV0(config, config.LastGoodBin)
	if err != nil {
		manifest.Status = guardianStatusLastGoodUnverifiedV0
		manifest.ReasonCode = guardianArtifactReasonCodeV0(err)
		return manifest, err
	}
	manifest.LastGood = lastGood
	copyResult, err := copyGuardianArtifactAtomicV0(config, config.LastGoodBin, config.CurrentBin, 0o700)
	if err != nil {
		manifest.Status = guardianStatusPromotionIncompleteV0
		manifest.ReasonCode = guardianArtifactReasonCodeV0(err)
		return manifest, err
	}
	manifest.Current = copyResult.Dest
	manifest.Status = guardianStatusRestoredV0
	return manifest, nil
}

func baseGuardianArtifactManifestV0(config guardianConfigV0, operation string) *guardianArtifactManifestV0 {
	return &guardianArtifactManifestV0{
		SchemaVersion: guardianArtifactManifestSchemaVersionV0,
		Status:        guardianStatusCandidateFailedV0,
		MaxBytes:      config.ArtifactMaxBytes,
		Operation:     operation,
	}
}

func guardianArtifactReasonCodeV0(err error) string {
	var policyErr guardianArtifactPolicyErrorV0
	if errors.As(err, &policyErr) && policyErr.Code != "" {
		return policyErr.Code
	}
	return "guardian_artifact_error"
}

func guardianArtifactErrorV0(err error) error {
	if err == nil {
		return nil
	}
	var policyErr guardianArtifactPolicyErrorV0
	if errors.As(err, &policyErr) {
		return errors.New(policyErr.Code)
	}
	return err
}
