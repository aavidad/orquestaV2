package candidateseal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

const (
	PhysicalGateBReceiptSchema = "orquesta.candidate-gate-b-physical-receipt.v1"
	PhysicalGateBReceiptStatus = "passed"
)

var ErrPhysicalGateBReceiptInvalid = errors.New("candidate_gate_b.physical_receipt_invalid")

var physicalGateBSubjectRoles = []string{
	"agent_microvm_candidate_manifest",
	"agent_microvm_physical_receipt",
	"cleanup",
	"continuation_authority",
	"continuation_manifest",
	"host_inventory_before",
	"negative_cases",
	"orquesta_build_receipt",
	"orquesta_ledger_snapshot",
	"orquesta_state_migration_receipt",
	"preservation_inventory",
	"preservation_manifest",
	"restart_after_seal",
	"restart_before_seal",
	"run_result",
}

// PhysicalGateBSubject seals one recoverable evidence object. Role is a
// logical name, never a host path, so the receipt remains portable.
type PhysicalGateBSubject struct {
	Role   string `json:"role"`
	SHA256 string `json:"sha256"`
	Bytes  uint64 `json:"bytes"`
}

// PhysicalGateBRun binds Gate B to the sole pre-existing Orquesta effect
// attempt and to the external execution continued by Agente MicroVM.
type PhysicalGateBRun struct {
	ProjectRef                  string `json:"project_ref"`
	GoalRef                     string `json:"goal_ref"`
	WorkItemRef                 string `json:"work_item_ref"`
	ExecutionRef                string `json:"execution_ref"`
	ActionRef                   string `json:"action_ref"`
	EffectIntentRef             string `json:"effect_intent_ref"`
	EffectIntentDigest          string `json:"effect_intent_digest"`
	EffectAttemptRef            string `json:"effect_attempt_ref"`
	PlanGeneration              uint64 `json:"plan_generation"`
	WorkItemGeneration          uint64 `json:"work_item_generation"`
	ActionFence                 uint64 `json:"action_fence"`
	ReconciliationAuthorityRef  string `json:"reconciliation_authority_ref"`
	ReconciliationAttemptRef    string `json:"reconciliation_attempt_ref"`
	ReconciliationReceiptRef    string `json:"reconciliation_receipt_ref"`
	ContinuationSubjectRef      string `json:"continuation_subject_ref"`
	ContinuationAuthorityRef    string `json:"continuation_authority_ref"`
	ContinuationManifestSHA256  string `json:"continuation_manifest_sha256"`
	ContinuationAuthoritySHA256 string `json:"continuation_authority_sha256"`
	EffectReceiptRef            string `json:"effect_receipt_ref"`
	AMVLaunchRef                string `json:"amv_launch_ref"`
	AMVExecutionRef             string `json:"amv_execution_ref"`
	AMVRunRef                   string `json:"amv_run_ref"`
	AMVCID                      uint32 `json:"amv_cid"`
	ResultArtifactRef           string `json:"result_artifact_ref"`
	ResultArtifactSHA256        string `json:"result_artifact_sha256"`
	ResultMarker                string `json:"result_marker"`
	PreservationReceiptRef      string `json:"preservation_receipt_ref"`
	PhysicalManifestRef         string `json:"physical_manifest_ref"`
	PhysicalManifestSHA256      string `json:"physical_manifest_sha256"`
	InventoryArtifactRef        string `json:"inventory_artifact_ref"`
	InventoryArtifactSHA256     string `json:"inventory_artifact_sha256"`
	ReconciliationOutcome       string `json:"reconciliation_outcome"`
	PreservationState           string `json:"preservation_state"`
	LifecycleState              string `json:"lifecycle_state"`
	GoalState                   string `json:"goal_state"`
	WorkItemState               string `json:"work_item_state"`
	ExecutionState              string `json:"execution_state"`
}

// PhysicalGateBProof records bounded physical facts. It deliberately cannot
// express a Gate C execution or the accreditation of V38.
type PhysicalGateBProof struct {
	AgentCount                uint64   `json:"agent_count"`
	MicroVMCount              uint64   `json:"microvm_count"`
	Transport                 string   `json:"transport"`
	AllowedServices           []string `json:"allowed_services"`
	GuestIPNetwork            bool     `json:"guest_ip_network"`
	TAP                       bool     `json:"tap"`
	Bridge                    bool     `json:"bridge"`
	NAT                       bool     `json:"nat"`
	Inbound                   bool     `json:"inbound"`
	EastWest                  bool     `json:"east_west"`
	DirectInternet            bool     `json:"direct_internet"`
	RealCodex                 bool     `json:"real_codex"`
	WorkCompleted             bool     `json:"work_completed"`
	Quiesced                  bool     `json:"quiesced"`
	Preserved                 bool     `json:"preserved"`
	Closed                    bool     `json:"closed"`
	OrquestaRestartBeforeSeal bool     `json:"orquesta_restart_before_seal"`
	OrquestaRestartAfterSeal  bool     `json:"orquesta_restart_after_seal"`
	AgentRestartBeforeSeal    bool     `json:"agent_microvm_restart_before_seal"`
	AgentRestartAfterSeal     bool     `json:"agent_microvm_restart_after_seal"`
	NegativeCases             []string `json:"negative_cases"`
	FirecrackerProcessesAfter uint64   `json:"firecracker_processes_after"`
	JailerProcessesAfter      uint64   `json:"jailer_processes_after"`
	OwnedMicroVMsAfter        uint64   `json:"owned_microvms_after"`
	OwnedCgroupsAfter         uint64   `json:"owned_cgroups_after"`
	OwnedTemporaryAssetsAfter uint64   `json:"owned_temporary_assets_after"`
	GateCClaimed              bool     `json:"gate_c_claimed"`
	V38AccreditationClaimed   bool     `json:"v38_accreditation_claimed"`
}

type PhysicalGateBTimeline struct {
	StartedAt        string `json:"started_at"`
	LaunchAcceptedAt string `json:"launch_accepted_at"`
	WorkCompletedAt  string `json:"work_completed_at"`
	PreservedAt      string `json:"preserved_at"`
	ClosedAt         string `json:"closed_at"`
	RecordedAt       string `json:"recorded_at"`
}

// PhysicalGateBReceipt is evidence for Gate B only. ReceiptSHA256 seals every
// preceding field with ReceiptSHA256 empty.
type PhysicalGateBReceipt struct {
	Schema                              string                 `json:"schema"`
	Gate                                string                 `json:"gate"`
	Status                              string                 `json:"status"`
	Protocol                            string                 `json:"protocol"`
	GateACandidateSHA256                string                 `json:"gate_a_candidate_sha256"`
	GateBBindingSHA256                  string                 `json:"gate_b_binding_sha256"`
	AgentMicroVMCandidateManifestSHA256 string                 `json:"agent_microvm_candidate_manifest_sha256"`
	Run                                 PhysicalGateBRun       `json:"run"`
	Proof                               PhysicalGateBProof     `json:"proof"`
	Timeline                            PhysicalGateBTimeline  `json:"timeline"`
	Subjects                            []PhysicalGateBSubject `json:"subjects"`
	ReceiptSHA256                       string                 `json:"receipt_sha256"`
}

// RequiredPhysicalGateBSubjectRoles returns the exact evidence inventory.
func RequiredPhysicalGateBSubjectRoles() []string {
	return append([]string(nil), physicalGateBSubjectRoles...)
}

// SealPhysicalGateBReceipt fills only envelope identities and evidence
// digests. Callers must supply the observed run, proof and timeline.
func SealPhysicalGateBReceipt(
	draft PhysicalGateBReceipt,
	binding GateBBinding,
	subjects map[string][]byte,
) (PhysicalGateBReceipt, error) {
	if !validGateBBindingSelf(binding) {
		return PhysicalGateBReceipt{}, ErrPhysicalGateBReceiptInvalid
	}
	draft.Schema, draft.Gate, draft.Status, draft.Protocol =
		PhysicalGateBReceiptSchema, "B", PhysicalGateBReceiptStatus, GateBProtocol
	draft.GateACandidateSHA256 = binding.GateACandidateSHA256
	draft.GateBBindingSHA256 = binding.BindingSHA256
	draft.Subjects = make([]PhysicalGateBSubject, 0, len(physicalGateBSubjectRoles))
	for _, role := range physicalGateBSubjectRoles {
		content, ok := subjects[role]
		if !ok || content == nil {
			return PhysicalGateBReceipt{}, ErrPhysicalGateBReceiptInvalid
		}
		draft.Subjects = append(draft.Subjects, PhysicalGateBSubject{
			Role: role, SHA256: digestPhysicalGateBBytes(content), Bytes: uint64(len(content)),
		})
	}
	if len(subjects) != len(physicalGateBSubjectRoles) {
		return PhysicalGateBReceipt{}, ErrPhysicalGateBReceiptInvalid
	}
	sealedDigests := make(map[string]string, len(draft.Subjects))
	for _, subject := range draft.Subjects {
		sealedDigests[subject.Role] = strings.TrimPrefix(subject.SHA256, "sha256:")
	}
	draft.AgentMicroVMCandidateManifestSHA256 = sealedDigests["agent_microvm_candidate_manifest"]
	draft.Run.ContinuationManifestSHA256 = sealedDigests["continuation_manifest"]
	draft.Run.ContinuationAuthoritySHA256 = sealedDigests["continuation_authority"]
	draft.Run.ResultArtifactSHA256 = sealedDigests["run_result"]
	draft.Run.PhysicalManifestSHA256 = sealedDigests["preservation_manifest"]
	draft.Run.InventoryArtifactSHA256 = sealedDigests["preservation_inventory"]
	draft.ReceiptSHA256 = physicalGateBReceiptDigest(draft)
	if err := VerifyPhysicalGateBReceipt(draft, binding, subjects); err != nil {
		return PhysicalGateBReceipt{}, err
	}
	return draft, nil
}

// VerifyPhysicalGateBReceipt verifies the receipt against the exact recovered
// evidence bytes. A digest-only map or a crossed candidate is insufficient.
func VerifyPhysicalGateBReceipt(
	receipt PhysicalGateBReceipt,
	binding GateBBinding,
	subjects map[string][]byte,
) error {
	if !validGateBBindingSelf(binding) ||
		receipt.Schema != PhysicalGateBReceiptSchema || receipt.Gate != "B" ||
		receipt.Status != PhysicalGateBReceiptStatus || receipt.Protocol != GateBProtocol ||
		receipt.GateACandidateSHA256 != binding.GateACandidateSHA256 ||
		receipt.GateBBindingSHA256 != binding.BindingSHA256 ||
		!validBarePhysicalGateBSHA256(receipt.AgentMicroVMCandidateManifestSHA256) ||
		!validPhysicalGateBRun(receipt.Run) || !validPhysicalGateBProof(receipt.Proof) ||
		!validPhysicalGateBTimeline(receipt.Timeline) ||
		!validPhysicalGateBSubjects(receipt.Subjects, subjects) ||
		!validPhysicalGateBSubjectBindings(receipt) ||
		receipt.ReceiptSHA256 != physicalGateBReceiptDigest(receipt) {
		return ErrPhysicalGateBReceiptInvalid
	}
	return nil
}

func EncodePhysicalGateBReceipt(receipt PhysicalGateBReceipt) ([]byte, error) {
	if !validPhysicalGateBReceiptSelf(receipt) {
		return nil, ErrPhysicalGateBReceiptInvalid
	}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return nil, ErrPhysicalGateBReceiptInvalid
	}
	return append(encoded, '\n'), nil
}

func DecodePhysicalGateBReceipt(encoded []byte) (PhysicalGateBReceipt, error) {
	if rejectPhysicalGateBDuplicateKeys(encoded) != nil {
		return PhysicalGateBReceipt{}, ErrPhysicalGateBReceiptInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var receipt PhysicalGateBReceipt
	if err := decoder.Decode(&receipt); err != nil {
		return PhysicalGateBReceipt{}, ErrPhysicalGateBReceiptInvalid
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF || !validPhysicalGateBReceiptSelf(receipt) {
		return PhysicalGateBReceipt{}, ErrPhysicalGateBReceiptInvalid
	}
	return receipt, nil
}

func rejectPhysicalGateBDuplicateKeys(encoded []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			keys := make(map[string]struct{})
			for decoder.More() {
				keyToken, keyErr := decoder.Token()
				key, stringKey := keyToken.(string)
				if keyErr != nil || !stringKey {
					return ErrPhysicalGateBReceiptInvalid
				}
				if _, found := keys[key]; found {
					return ErrPhysicalGateBReceiptInvalid
				}
				keys[key] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return ErrPhysicalGateBReceiptInvalid
		}
	}
	return walk()
}

func validPhysicalGateBReceiptSelf(receipt PhysicalGateBReceipt) bool {
	return receipt.Schema == PhysicalGateBReceiptSchema && receipt.Gate == "B" &&
		receipt.Status == PhysicalGateBReceiptStatus && receipt.Protocol == GateBProtocol &&
		validDigest(receipt.GateACandidateSHA256) && validDigest(receipt.GateBBindingSHA256) &&
		validBarePhysicalGateBSHA256(receipt.AgentMicroVMCandidateManifestSHA256) && validPhysicalGateBRun(receipt.Run) &&
		validPhysicalGateBProof(receipt.Proof) && validPhysicalGateBTimeline(receipt.Timeline) &&
		validPhysicalGateBSubjectInventory(receipt.Subjects) && validPhysicalGateBSubjectBindings(receipt) &&
		receipt.ReceiptSHA256 == physicalGateBReceiptDigest(receipt)
}

func validPhysicalGateBRun(run PhysicalGateBRun) bool {
	for _, value := range []string{
		run.ProjectRef, run.GoalRef, run.WorkItemRef, run.ExecutionRef, run.ActionRef,
		run.EffectIntentRef, run.EffectAttemptRef, run.ReconciliationAuthorityRef,
		run.ReconciliationAttemptRef, run.ReconciliationReceiptRef, run.ContinuationSubjectRef,
		run.ContinuationAuthorityRef, run.EffectReceiptRef, run.AMVLaunchRef, run.AMVExecutionRef,
		run.AMVRunRef, run.ResultArtifactRef, run.ResultMarker, run.PreservationReceiptRef,
		run.PhysicalManifestRef, run.InventoryArtifactRef,
	} {
		if !validPhysicalGateBText(value) {
			return false
		}
	}
	if !validBarePhysicalGateBSHA256(run.EffectIntentDigest) ||
		!validBarePhysicalGateBSHA256(run.ContinuationManifestSHA256) ||
		!validBarePhysicalGateBSHA256(run.ContinuationAuthoritySHA256) ||
		!validBarePhysicalGateBSHA256(run.ResultArtifactSHA256) ||
		!validBarePhysicalGateBSHA256(run.PhysicalManifestSHA256) ||
		!validBarePhysicalGateBSHA256(run.InventoryArtifactSHA256) || run.PlanGeneration == 0 ||
		run.WorkItemGeneration == 0 || run.ActionFence == 0 || run.AMVCID < 3 {
		return false
	}
	return run.ActionRef == "action:launch:"+run.ExecutionRef &&
		run.EffectIntentRef == "effect-intent:"+run.ActionRef &&
		strings.HasPrefix(run.EffectAttemptRef, "effect-attempt:"+run.ActionRef+":claim:") &&
		run.EffectReceiptRef == "effect-receipt:"+run.EffectIntentRef &&
		run.ReconciliationReceiptRef == "agent-launch-reconciliation-receipt:"+run.ReconciliationAuthorityRef &&
		run.ReconciliationOutcome == "completed" && run.PreservationState == "preserved_pending_review" &&
		run.LifecycleState == "closed" && run.GoalState == "succeeded" &&
		run.WorkItemState == "succeeded" && run.ExecutionState == "succeeded"
}

func validPhysicalGateBProof(proof PhysicalGateBProof) bool {
	wantServices := []string{"controlled_egress_proxy", "orquesta_broker"}
	wantNegatives := []string{
		"crossed_reference", "digest_mismatch", "grant_replay", "restart_after_seal",
		"restart_before_seal", "write_set_mismatch",
	}
	return proof.AgentCount == 1 && proof.MicroVMCount == 1 && proof.Transport == "vsock_only" &&
		equalStrings(proof.AllowedServices, wantServices) && !proof.GuestIPNetwork && !proof.TAP &&
		!proof.Bridge && !proof.NAT && !proof.Inbound && !proof.EastWest && !proof.DirectInternet &&
		proof.RealCodex && proof.WorkCompleted && proof.Quiesced && proof.Preserved && proof.Closed &&
		proof.OrquestaRestartBeforeSeal && proof.OrquestaRestartAfterSeal &&
		proof.AgentRestartBeforeSeal && proof.AgentRestartAfterSeal &&
		equalStrings(proof.NegativeCases, wantNegatives) && proof.FirecrackerProcessesAfter == 0 &&
		proof.JailerProcessesAfter == 0 && proof.OwnedMicroVMsAfter == 0 && proof.OwnedCgroupsAfter == 0 &&
		proof.OwnedTemporaryAssetsAfter == 0 && !proof.GateCClaimed && !proof.V38AccreditationClaimed
}

func validPhysicalGateBTimeline(timeline PhysicalGateBTimeline) bool {
	values := []string{
		timeline.StartedAt, timeline.LaunchAcceptedAt, timeline.WorkCompletedAt,
		timeline.PreservedAt, timeline.ClosedAt, timeline.RecordedAt,
	}
	var previous time.Time
	for _, value := range values {
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil || parsed.Location() != time.UTC || parsed.Format(time.RFC3339Nano) != value ||
			(!previous.IsZero() && parsed.Before(previous)) {
			return false
		}
		previous = parsed
	}
	return true
}

func validPhysicalGateBSubjects(subjects []PhysicalGateBSubject, content map[string][]byte) bool {
	if !validPhysicalGateBSubjectInventory(subjects) || len(content) != len(subjects) {
		return false
	}
	for _, subject := range subjects {
		value, ok := content[subject.Role]
		if !ok || value == nil || uint64(len(value)) != subject.Bytes ||
			digestPhysicalGateBBytes(value) != subject.SHA256 {
			return false
		}
	}
	return true
}

func validPhysicalGateBSubjectInventory(subjects []PhysicalGateBSubject) bool {
	if len(subjects) != len(physicalGateBSubjectRoles) {
		return false
	}
	for index, subject := range subjects {
		if subject.Role != physicalGateBSubjectRoles[index] || subject.Bytes == 0 || !validDigest(subject.SHA256) {
			return false
		}
	}
	return true
}

func validPhysicalGateBSubjectBindings(receipt PhysicalGateBReceipt) bool {
	digests := make(map[string]string, len(receipt.Subjects))
	for _, subject := range receipt.Subjects {
		digests[subject.Role] = strings.TrimPrefix(subject.SHA256, "sha256:")
	}
	return receipt.AgentMicroVMCandidateManifestSHA256 == digests["agent_microvm_candidate_manifest"] &&
		receipt.Run.ContinuationManifestSHA256 == digests["continuation_manifest"] &&
		receipt.Run.ContinuationAuthoritySHA256 == digests["continuation_authority"] &&
		receipt.Run.ResultArtifactSHA256 == digests["run_result"] &&
		receipt.Run.PhysicalManifestSHA256 == digests["preservation_manifest"] &&
		receipt.Run.InventoryArtifactSHA256 == digests["preservation_inventory"]
}

func validBarePhysicalGateBSHA256(value string) bool {
	return len(value) == sha256.Size*2 && strings.ToLower(value) == value &&
		validDigest("sha256:"+value)
}

func validPhysicalGateBText(value string) bool {
	return value != "" && len(value) <= 512 && strings.TrimSpace(value) == value &&
		!strings.ContainsAny(value, "\x00\r\n")
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func digestPhysicalGateBBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func physicalGateBReceiptDigest(receipt PhysicalGateBReceipt) string {
	receipt.ReceiptSHA256 = ""
	encoded, _ := json.Marshal(receipt)
	return digestPhysicalGateBBytes(encoded)
}
