//go:build linux

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"orquesta/internal/adapters/attestor/firecrackerclient"
	"orquesta/internal/e2e/firecrackerattestor"
	"orquesta/internal/e2e/firecrackerattestorworkload"

	"golang.org/x/sys/unix"
)

const maxSpecBytes = int64(64 << 10)

type specDocument struct {
	Candidate struct {
		UnitName             string `json:"unit_name"`
		UnitPath             string `json:"unit_path"`
		UnitSHA256           string `json:"unit_sha256"`
		PrimitivesUnitPath   string `json:"primitives_unit_path"`
		PrimitivesUnitSHA256 string `json:"primitives_unit_sha256"`
		LauncherPath         string `json:"launcher_path"`
		LauncherSHA256       string `json:"launcher_sha256"`
		LauncherSocketPath   string `json:"launcher_socket_path"`
		ConfigPath           string `json:"config_path"`
		ConfigSHA256         string `json:"config_sha256"`
		SupervisorPath       string `json:"supervisor_path"`
		SupervisorSHA256     string `json:"supervisor_sha256"`
		RuntimeRoot          string `json:"runtime_root"`
		CgroupRoot           string `json:"cgroup_root"`
		ParentCgroup         string `json:"parent_cgroup"`
		NetNSPath            string `json:"netns_path"`
		AssetDigest          string `json:"asset_digest"`
	} `json:"candidate"`
	PolicyDigest   string           `json:"policy_digest"`
	EvidencePath   string           `json:"evidence_path"`
	ReceiptPath    string           `json:"receipt_path"`
	PhaseTimeout   string           `json:"phase_timeout"`
	CleanupTimeout string           `json:"cleanup_timeout"`
	StableFor      string           `json:"stable_for"`
	PollInterval   string           `json:"poll_interval"`
	ChildUID       uint32           `json:"child_uid"`
	ChildGID       uint32           `json:"child_gid"`
	Workload       workloadDocument `json:"workload"`
}

type workloadDocument struct {
	SocketPath                string `json:"socket_path"`
	ExpectedAssetDigest       string `json:"expected_asset_digest"`
	WorkRoot                  string `json:"work_root"`
	GitCommand                string `json:"git_command"`
	TestSleep                 string `json:"test_sleep"`
	AttestationTimeout        string `json:"attestation_timeout"`
	AttestationCleanupTimeout string `json:"attestation_cleanup_timeout"`
	MaxOutputBytes            uint64 `json:"max_output_bytes"`
	MaxSubjectBytes           int64  `json:"max_subject_bytes"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 1 {
		if handled, status := firecrackerattestor.RunAutoExecMode(
			arguments[0], physicalWorkloadHandler{},
		); handled {
			return status
		}
	}
	if len(arguments) != 2 || arguments[0] != "--spec" {
		writeStatus(stderr, "firecracker_attestor_e2e.arguments_invalid")
		return 2
	}
	config, err := loadSpec(arguments[1])
	if err != nil {
		writeStatus(stderr, "firecracker_attestor_e2e.spec_invalid")
		return 2
	}
	unit := firecrackerattestor.LinuxUnit{}
	monitor, err := firecrackerattestor.NewLinuxMonitor(config, unit)
	if err != nil {
		writeStatus(stderr, "firecracker_attestor_e2e.spec_invalid")
		return 2
	}
	supervisor, err := firecrackerattestor.New(
		config,
		unit,
		firecrackerattestor.AutoExecWorkload{
			SelfPath:   config.Candidate.SupervisorPath,
			SelfSHA256: config.Candidate.SupervisorSHA256,
			UID:        config.ChildUID, GID: config.ChildGID,
			Payload:           config.WorkloadPayload,
			CancellationGrace: config.CleanupTimeout,
		},
		monitor,
		firecrackerattestor.LinuxPublisher{
			EvidencePath: config.EvidencePath,
			ReceiptPath:  config.ReceiptPath,
		},
		firecrackerattestor.SystemClock{},
	)
	if err != nil {
		writeStatus(stderr, "firecracker_attestor_e2e.spec_invalid")
		return 2
	}
	supervisorContext, cancelSupervisor := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancelSupervisor()
	publication, err := supervisor.Run(supervisorContext)
	if err != nil {
		writeStatus(stderr, "firecracker_attestor_e2e.failed")
		return 1
	}
	_, _ = fmt.Fprintf(
		stdout,
		"status=passed\nevidence_sha256=%s\nevidence_path=%s\nreceipt_path=%s\n",
		publication.EvidenceSHA256, publication.EvidencePath, publication.ReceiptPath,
	)
	return 0
}

func loadSpec(path string) (firecrackerattestor.Config, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return firecrackerattestor.Config{}, firecrackerattestor.ErrInvalid
	}
	if err := firecrackerattestor.VerifyTrustedAncestors(path); err != nil {
		return firecrackerattestor.Config{}, err
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, path, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return firecrackerattestor.Config{}, err
	}
	file := os.NewFile(uintptr(fd), "firecracker-attestor-e2e-spec")
	defer file.Close()
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Uid != 0 || stat.Gid != 0 || stat.Nlink != 1 ||
		stat.Mode&0o777 != 0o400 ||
		stat.Size <= 0 || stat.Size > maxSpecBytes {
		return firecrackerattestor.Config{}, firecrackerattestor.ErrInvalid
	}
	content, err := io.ReadAll(io.LimitReader(file, maxSpecBytes+1))
	if err != nil || int64(len(content)) != stat.Size {
		return firecrackerattestor.Config{}, firecrackerattestor.ErrInvalid
	}
	var document specDocument
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&document) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return firecrackerattestor.Config{}, firecrackerattestor.ErrInvalid
	}
	phaseTimeout, err := time.ParseDuration(document.PhaseTimeout)
	if err != nil {
		return firecrackerattestor.Config{}, err
	}
	cleanupTimeout, err := time.ParseDuration(document.CleanupTimeout)
	if err != nil {
		return firecrackerattestor.Config{}, err
	}
	stableFor, err := time.ParseDuration(document.StableFor)
	if err != nil {
		return firecrackerattestor.Config{}, err
	}
	pollInterval, err := time.ParseDuration(document.PollInterval)
	if err != nil {
		return firecrackerattestor.Config{}, err
	}
	candidate := document.Candidate
	if document.Workload.SocketPath != candidate.LauncherSocketPath ||
		document.Workload.ExpectedAssetDigest != candidate.AssetDigest {
		return firecrackerattestor.Config{}, firecrackerattestor.ErrInvalid
	}
	return firecrackerattestor.Config{
		Candidate: firecrackerattestor.Candidate{
			UnitName: candidate.UnitName, UnitPath: candidate.UnitPath,
			UnitSHA256:           candidate.UnitSHA256,
			PrimitivesUnitPath:   candidate.PrimitivesUnitPath,
			PrimitivesUnitSHA256: candidate.PrimitivesUnitSHA256,
			LauncherPath:         candidate.LauncherPath,
			LauncherSHA256:       candidate.LauncherSHA256,
			LauncherSocketPath:   candidate.LauncherSocketPath,
			ConfigPath:           candidate.ConfigPath, ConfigSHA256: candidate.ConfigSHA256,
			SupervisorPath:   candidate.SupervisorPath,
			SupervisorSHA256: candidate.SupervisorSHA256,
			RuntimeRoot:      candidate.RuntimeRoot, CgroupRoot: candidate.CgroupRoot,
			ParentCgroup: candidate.ParentCgroup, NetNSPath: candidate.NetNSPath,
			AssetDigest: candidate.AssetDigest,
		},
		PolicyDigest: document.PolicyDigest,
		EvidencePath: document.EvidencePath, ReceiptPath: document.ReceiptPath,
		PhaseTimeout: phaseTimeout, CleanupTimeout: cleanupTimeout,
		StableFor: stableFor, PollInterval: pollInterval,
		ChildUID: document.ChildUID, ChildGID: document.ChildGID,
		WorkloadPayload: workloadPayload(document.Workload),
	}, nil
}

func writeStatus(writer io.Writer, code string) {
	_, _ = fmt.Fprintf(writer, "status=failed\ncode=%s\n", code)
}

type physicalWorkloadHandler struct{}

func (physicalWorkloadHandler) Run(
	ctx context.Context,
	request firecrackerattestor.WorkloadRequest,
	raw json.RawMessage,
) ([]firecrackerattestor.AttestationOutcome, error) {
	var document workloadDocument
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&document) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return nil, firecrackerattestor.ErrInvalid
	}
	testSleep, err := time.ParseDuration(document.TestSleep)
	if err != nil {
		return nil, err
	}
	timeout, err := time.ParseDuration(document.AttestationTimeout)
	if err != nil {
		return nil, err
	}
	cleanupTimeout, err := time.ParseDuration(document.AttestationCleanupTimeout)
	if err != nil {
		return nil, err
	}
	if request.Count == 0 ||
		request.GuestMemoryMiB != firecrackerattestor.ExpectedGuestMemoryMiB ||
		request.MemoryMaxBytes != firecrackerattestor.ExpectedMemoryMaxBytes ||
		request.PIDsMax != firecrackerattestor.ExpectedPIDsMax ||
		request.CPUQuotaMicros != firecrackerattestor.ExpectedCPUQuotaMicros ||
		request.CPUPeriodMicros != firecrackerattestor.ExpectedCPUPeriodMicros {
		return nil, firecrackerattestor.ErrInvalid
	}
	summary, err := firecrackerattestorworkload.Run(
		ctx,
		firecrackerattestorworkload.Config{
			SocketPath:          document.SocketPath,
			ExpectedAssetDigest: document.ExpectedAssetDigest,
			WorkRoot:            document.WorkRoot, GitCommand: document.GitCommand,
			TestSleep: testSleep, ConcurrentRuns: int(request.Count),
			Limits: firecrackerclient.Limits{
				Timeout: timeout, CleanupTimeout: cleanupTimeout,
				MaxOutputBytes:    document.MaxOutputBytes,
				MaxSubjectBytes:   document.MaxSubjectBytes,
				MaxConcurrentRuns: int(firecrackerattestor.ExpectedConcurrentRuns),
				GuestMemoryMiB:    request.GuestMemoryMiB,
				MemoryMaxBytes:    request.MemoryMaxBytes,
				PIDsMax:           request.PIDsMax,
				CPUQuotaMicros:    request.CPUQuotaMicros,
			},
		},
	)
	if err != nil || summary.Attempted != int(request.Count) ||
		summary.Passed != int(request.Count) ||
		len(summary.Attestations) != int(request.Count) ||
		(request.PolicyDigest != "" && summary.PolicyDigest != request.PolicyDigest) {
		return nil, firecrackerattestor.ErrInvalid
	}
	outcomes := make([]firecrackerattestor.AttestationOutcome, len(summary.Attestations))
	for index, attestation := range summary.Attestations {
		outcomes[index] = firecrackerattestor.AttestationOutcome{
			Ref:           attestation.ReceiptRef,
			RunID:         attestation.RunID,
			SubjectDigest: attestation.SubjectDigest,
			ReceiptRef:    attestation.ReceiptRef,
			PolicyDigest:  summary.PolicyDigest,
			Valid:         true,
		}
	}
	return outcomes, nil
}

func workloadPayload(document workloadDocument) json.RawMessage {
	content, err := json.Marshal(document)
	if err != nil {
		return nil
	}
	return content
}
