package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func parseAutoprogrammingPromotionGuardianOutputV0(
	stdout *autoprogrammingPromotionGuardianOutputBufferV0,
	stderr *autoprogrammingPromotionGuardianOutputBufferV0,
) (serverAutoprogrammingPromotionGuardianResultV0, error) {
	if stdout == nil || strings.TrimSpace(stdout.String()) == "" || stdout.truncated {
		return serverAutoprogrammingPromotionGuardianResultV0{}, fmt.Errorf("guardian_result_invalid")
	}
	if stderr != nil && stderr.truncated {
		return serverAutoprogrammingPromotionGuardianResultV0{}, fmt.Errorf("guardian_result_invalid")
	}
	var public autoprogrammingPromotionGuardianPublicResultV0
	decoder := json.NewDecoder(strings.NewReader(stdout.String()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&public); err != nil {
		return serverAutoprogrammingPromotionGuardianResultV0{}, fmt.Errorf("guardian_result_invalid")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return serverAutoprogrammingPromotionGuardianResultV0{}, fmt.Errorf("guardian_result_invalid")
	}
	if public.SchemaVersion != "orquesta_guardian_result.v0" ||
		!autoprogrammingPromotionGuardianStatusAllowedV0(public.Status) ||
		len(compactGuardianEvidenceRefsV0(public.EvidenceRefs)) == 0 {
		return serverAutoprogrammingPromotionGuardianResultV0{}, fmt.Errorf("guardian_result_invalid")
	}
	return serverAutoprogrammingPromotionGuardianResultV0{
		Status:       strings.TrimSpace(public.Status),
		Phase:        strings.TrimSpace(public.Phase),
		Promote:      public.ConfigEffective == nil || public.ConfigEffective.Promote,
		Promoted:     public.Promoted,
		Restored:     public.Restored,
		VerifiedOnly: public.ConfigEffective != nil && !public.ConfigEffective.Promote,
		BreakglassPromoted: strings.TrimSpace(public.Status) == "candidate_promoted_breakglass" &&
			public.Promoted,
		ManifestRef:       strings.TrimSpace(public.ManifestRef),
		RepairPacketRef:   strings.TrimSpace(public.RepairPacketRef),
		RepairAttemptRef:  strings.TrimSpace(public.RepairAttemptRef),
		FailurePacketHash: strings.TrimSpace(public.FailurePacketHash),
		RepairBlocked:     public.RepairBlocked,
		RepairBlockReason: strings.TrimSpace(public.RepairBlockReason),
		ReasonCodes:       compactGuardianEvidenceRefsV0(public.ReasonCodes),
		EvidenceRefs:      compactGuardianEvidenceRefsV0(public.EvidenceRefs),
	}, nil
}

func autoprogrammingPromotionGuardianStatusAllowedV0(status string) bool {
	switch strings.TrimSpace(status) {
	case "candidate_promoted", "candidate_promoted_breakglass", "candidate_failed", "last_good_restored",
		"shutdown_ready", "promotion_incomplete", "last_good_unverified",
		"guardian_promotion_lease_busy", "guardian_promotion_lease_lost",
		"guardian_manifest_incomplete":
		return true
	default:
		return false
	}
}

func autoprogrammingPromotionGuardianResultAcceptedV0(
	result serverAutoprogrammingPromotionGuardianResultV0,
) bool {
	return (result.Status == "candidate_promoted" && (result.Promoted || result.VerifiedOnly)) ||
		(result.Status == "candidate_promoted_breakglass" && result.Promoted)
}

func autoprogrammingPromotionGuardianResultWithRequestV0(
	result serverAutoprogrammingPromotionGuardianResultV0,
	request serverAutoprogrammingPromotionGuardianRequestV0,
) serverAutoprogrammingPromotionGuardianResultV0 {
	result.ProjectRef = strings.TrimSpace(request.ProjectRef)
	result.AppRef = strings.TrimSpace(request.AppRef)
	result.RepoRef = strings.TrimSpace(request.RepoRef)
	result.PromotionRef = strings.TrimSpace(request.PromotionRef)
	result.RunRef = strings.TrimSpace(request.RunRef)
	result.WorktreeRef = strings.TrimSpace(request.WorktreeRef)
	result.BranchRef = strings.TrimSpace(request.BranchRef)
	result.CandidateHash, result.CandidateSizeBytes = autoprogrammingPromotionGuardianCandidateDigestV0(
		request.CandidateBin,
	)
	return result
}

func autoprogrammingPromotionGuardianCandidateDigestV0(path string) (string, int64) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", 0
	}
	file, err := os.Open(path)
	if err != nil {
		return "", 0
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", 0
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", 0
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), info.Size()
}

func autoprogrammingPromotionGuardianPublicErrorV0(
	result serverAutoprogrammingPromotionGuardianResultV0,
) string {
	switch strings.TrimSpace(result.Status) {
	case "candidate_failed", "last_good_restored", "shutdown_ready",
		"promotion_incomplete", "last_good_unverified",
		"guardian_promotion_lease_busy", "guardian_promotion_lease_lost",
		"guardian_manifest_incomplete":
		return strings.TrimSpace(result.Status)
	case "guardian_healthcheck_required":
		return strings.TrimSpace(result.Status)
	default:
		return "guardian_result_invalid"
	}
}

func autoprogrammingPromotionGuardianErrorCodeV0(err error) string {
	if err == nil {
		return "guardian_candidate_failed"
	}
	code := strings.TrimSpace(err.Error())
	if code == "" {
		return "guardian_candidate_failed"
	}
	return code
}

func compactGuardianEvidenceRefsV0(values []string) []string {
	seen := map[string]struct{}{}
	var refs []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || strings.ContainsAny(value, `/\`) {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		refs = append(refs, value)
	}
	return refs
}
