// Package firecrackerattestor runs the privileged, destructive Firecracker
// attestor acceptance gate. It deliberately owns no test workload: callers
// must inject one through WorkloadPort.
package firecrackerattestor

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

const (
	EvidenceSchema = "orquesta.firecracker-attestor.physical-16.evidence.v2"
	ReceiptSchema  = "orquesta_firecracker_activation_receipt.v2"
	Suite          = "orquesta.firecracker-attestor.physical-16.v1"

	ExpectedConcurrentRuns  = uint32(16)
	ExpectedGuestMemoryMiB  = uint32(4096)
	ExpectedMemoryMaxBytes  = uint64(5 << 30)
	ExpectedPIDsMax         = uint32(512)
	ExpectedCPUQuotaMicros  = uint64(200_000)
	ExpectedCPUPeriodMicros = uint64(100_000)
)

var ErrInvalid = errors.New("firecracker_attestor_e2e.invalid")

type Config struct {
	Candidate       Candidate
	PolicyDigest    string
	EvidencePath    string
	ReceiptPath     string
	PhaseTimeout    time.Duration
	CleanupTimeout  time.Duration
	StableFor       time.Duration
	PollInterval    time.Duration
	ChildUID        uint32
	ChildGID        uint32
	WorkloadPayload json.RawMessage
}

type Candidate struct {
	UnitName             string
	UnitPath             string
	UnitSHA256           string
	PrimitivesUnitPath   string
	PrimitivesUnitSHA256 string
	LauncherPath         string
	LauncherSHA256       string
	LauncherSocketPath   string
	ConfigPath           string
	ConfigSHA256         string
	SupervisorPath       string
	SupervisorSHA256     string
	RuntimeRoot          string
	CgroupRoot           string
	ParentCgroup         string
	NetNSPath            string
	AssetDigest          string
}

type CandidateIdentity struct {
	UnitSHA256           string `json:"unit_sha256"`
	PrimitivesUnitSHA256 string `json:"primitives_unit_sha256"`
	LauncherSHA256       string `json:"launcher_sha256"`
	ConfigSHA256         string `json:"config_sha256"`
	SupervisorSHA256     string `json:"supervisor_sha256"`
	AssetDigest          string `json:"asset_digest"`
}

type UnitIdentity struct {
	UnitName         string `json:"unit_name"`
	MainPID          int    `json:"main_pid"`
	InvocationID     string `json:"invocation_id"`
	Active           bool   `json:"active"`
	FragmentPath     string `json:"fragment_path"`
	Loaded           bool   `json:"loaded"`
	NeedDaemonReload bool   `json:"need_daemon_reload"`
}

type WorkloadRequest struct {
	Count           uint32 `json:"count"`
	GuestMemoryMiB  uint32 `json:"guest_memory_mib"`
	MemoryMaxBytes  uint64 `json:"memory_max_bytes"`
	PIDsMax         uint32 `json:"pids_max"`
	CPUQuotaMicros  uint64 `json:"cpu_quota_micros"`
	CPUPeriodMicros uint64 `json:"cpu_period_micros"`
	PolicyDigest    string `json:"policy_digest"`
}

type AttestationOutcome struct {
	Ref           string `json:"ref"`
	RunID         string `json:"run_id"`
	SubjectDigest string `json:"subject_digest"`
	ReceiptRef    string `json:"receipt_ref"`
	PolicyDigest  string `json:"policy_digest"`
	Valid         bool   `json:"valid"`
}

type WorkloadRun interface {
	Done() <-chan struct{}
	Result() ([]AttestationOutcome, error)
	Close() error
}

type WorkloadPort interface {
	StartPhase(context.Context, WorkloadRequest) (WorkloadRun, error)
}

type UnitPort interface {
	Preflight(context.Context, Config) (CandidateIdentity, error)
	Start(context.Context, string) (UnitIdentity, error)
	WaitReady(context.Context, Config, UnitIdentity) (UnitIdentity, error)
	Observe(context.Context, string) (UnitIdentity, error)
	Stop(context.Context, string) error
}

type PhaseEvidence struct {
	RequestedRuns      uint32   `json:"requested_runs"`
	HighWaterRuns      uint32   `json:"high_water_runs"`
	Samples            uint64   `json:"samples"`
	RunIDs             []string `json:"run_ids"`
	FirecrackerPIDs    []int    `json:"firecracker_pids"`
	LimitsExact        bool     `json:"limits_exact"`
	MemorySwapMaxZero  bool     `json:"memory_swap_max_zero"`
	NetworkAbsent      bool     `json:"network_absent"`
	APIAbsent          bool     `json:"api_absent"`
	VsockAbsent        bool     `json:"vsock_absent"`
	SerialAbsent       bool     `json:"serial_absent"`
	UnitIdentityStable bool     `json:"unit_identity_stable"`
}

type CleanupEvidence struct {
	StableSamples     uint64 `json:"stable_samples"`
	ResidualRuns      uint32 `json:"residual_runs"`
	ResidualCgroups   uint32 `json:"residual_cgroups"`
	ResidualProcesses uint32 `json:"residual_processes"`
	UnitStopped       bool   `json:"unit_stopped"`
	SocketAbsent      bool   `json:"socket_absent"`
}

type MonitorPort interface {
	CapturePhase(context.Context, UnitIdentity, WorkloadRun, uint32) (PhaseEvidence, error)
	WaitQuiescent(context.Context, UnitIdentity) (CleanupEvidence, error)
}

type Publication struct {
	EvidenceSHA256 string
	EvidencePath   string
	ReceiptPath    string
}

type PublisherPort interface {
	Publish(Evidence, Receipt) (Publication, error)
}

type Evidence struct {
	Schema       string               `json:"schema"`
	Suite        string               `json:"suite"`
	Status       string               `json:"status"`
	StartedAt    string               `json:"started_at"`
	FinishedAt   string               `json:"finished_at"`
	Candidate    CandidateIdentity    `json:"candidate"`
	PolicyDigest string               `json:"policy_digest"`
	Unit         UnitIdentity         `json:"unit"`
	PhaseOne     PhaseEvidence        `json:"phase_one"`
	PhaseSixteen PhaseEvidence        `json:"phase_sixteen"`
	Attestations []AttestationOutcome `json:"attestations"`
	Cleanup      CleanupEvidence      `json:"cleanup"`
}

type Receipt struct {
	Schema               string
	Status               string
	EvidenceSHA256       string
	ConfigSHA256         string
	UnitSHA256           string
	PrimitivesUnitSHA256 string
	LauncherSHA256       string
	AssetDigest          string
	PolicyDigest         string
	E2ESuite             string
	MaxConcurrentRuns    uint32
	PhysicalMicroVMCount uint32
	ConcurrentHighWater  uint32
	AllAttestationsValid bool
	ZeroResidualRuns     bool
	NetworkAbsent        bool
	MemorySwapMaxZero    bool
	APIAbsent            bool
	VsockAbsent          bool
	SerialAbsent         bool
}

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }
