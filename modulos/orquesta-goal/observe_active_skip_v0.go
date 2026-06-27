package orquestagoal

import "strings"

type GoalObservationFingerprintV0 struct {
	RunRef       string `json:"run_ref,omitempty"`
	GoalRef      string `json:"goal_ref,omitempty"`
	AckFilesHash string `json:"ack_files_hash,omitempty"`
	ProcessAlive bool   `json:"process_alive,omitempty"`
	LastStatus   string `json:"last_status,omitempty"`
	EvidenceHash string `json:"evidence_hash,omitempty"`
}

func NormalizeGoalObservationFingerprintV0(
	fingerprint GoalObservationFingerprintV0,
) GoalObservationFingerprintV0 {
	fingerprint.RunRef = strings.TrimSpace(fingerprint.RunRef)
	fingerprint.GoalRef = strings.TrimSpace(fingerprint.GoalRef)
	fingerprint.AckFilesHash = strings.TrimSpace(fingerprint.AckFilesHash)
	fingerprint.LastStatus = strings.TrimSpace(fingerprint.LastStatus)
	fingerprint.EvidenceHash = strings.TrimSpace(fingerprint.EvidenceHash)
	return fingerprint
}

func GoalObservationUnchangedV0(
	prev GoalObservationFingerprintV0,
	current GoalObservationFingerprintV0,
) bool {
	prev = NormalizeGoalObservationFingerprintV0(prev)
	current = NormalizeGoalObservationFingerprintV0(current)
	if prev.RunRef == "" ||
		current.RunRef == "" ||
		prev.RunRef != current.RunRef ||
		prev.GoalRef != current.GoalRef {
		return false
	}
	return prev.AckFilesHash == current.AckFilesHash &&
		prev.ProcessAlive == current.ProcessAlive &&
		prev.LastStatus == current.LastStatus &&
		prev.EvidenceHash == current.EvidenceHash
}
