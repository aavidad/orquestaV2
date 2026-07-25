package firecrackerlauncher

import (
	"path/filepath"
	"strings"
	"time"
)

const (
	runtimeMarkerName      = ".orquesta-firecracker-launcher-root"
	maxDiagnosticBytesHard = int64(64 << 20)
	maxConcurrentRunsHard  = uint32(256)
)

type Config struct {
	SocketPath             string
	RuntimeRoot            string
	FirecrackerCommand     string
	JailerCommand          string
	KernelImage            string
	GuestImage             string
	CgroupRoot             string
	ParentCgroup           string
	AllowedUID             uint32
	AllowedGID             uint32
	JailUID                uint32
	JailGID                uint32
	MaxInputBytes          int64
	MaxOutputDriveBytes    uint64
	MaxCapturedOutputBytes uint64
	MaxMemoryBytes         uint64
	MaxPIDs                uint32
	MaxCPUQuotaMicros      uint64
	MaxConcurrentRuns      uint32
	MaxTimeout             time.Duration
	CleanupTimeout         time.Duration
	MaxDiagnosticBytes     int64
}

func ValidateConfig(config Config) error {
	return validateConfig(config)
}

func validateConfig(config Config) error {
	if !canonicalAbsolute(config.SocketPath) || !canonicalAbsolute(config.RuntimeRoot) ||
		!canonicalAbsolute(config.FirecrackerCommand) || !canonicalAbsolute(config.JailerCommand) ||
		!canonicalAbsolute(config.KernelImage) || !canonicalAbsolute(config.GuestImage) ||
		!canonicalAbsolute(config.CgroupRoot) || config.RuntimeRoot == "/" ||
		!pathWithin(config.RuntimeRoot, config.SocketPath) || !validRelativeCgroup(config.ParentCgroup) ||
		config.AllowedUID == ^uint32(0) || config.AllowedGID == ^uint32(0) ||
		config.JailUID == 0 || config.JailGID == 0 ||
		config.JailUID == ^uint32(0) || config.JailGID == ^uint32(0) ||
		config.JailUID == config.AllowedUID || config.JailGID == config.AllowedGID ||
		config.MaxInputBytes < int64(firecrackerSectorBytes) ||
		config.MaxInputBytes > maxInputDriveBytesHard ||
		config.MaxInputBytes%int64(firecrackerSectorBytes) != 0 ||
		!validOutputDriveBytes(config.MaxOutputDriveBytes) ||
		config.MaxCapturedOutputBytes == 0 ||
		config.MaxCapturedOutputBytes > maxCapturedOutputBytes ||
		config.MaxMemoryBytes < minimumMemoryMaxBytes || config.MaxPIDs == 0 ||
		config.MaxCPUQuotaMicros == 0 || config.MaxConcurrentRuns == 0 ||
		config.MaxConcurrentRuns > maxConcurrentRunsHard ||
		config.MaxTimeout <= 0 || config.CleanupTimeout <= 0 ||
		config.CleanupTimeout >= config.MaxTimeout ||
		config.MaxDiagnosticBytes <= 0 || config.MaxDiagnosticBytes > maxDiagnosticBytesHard {
		return launcherError(CodeConfigInvalid)
	}
	return nil
}

func canonicalAbsolute(value string) bool {
	return value != "" && filepath.IsAbs(value) && filepath.Clean(value) == value &&
		!strings.ContainsRune(value, 0)
}

func pathWithin(root, child string) bool {
	relative, err := filepath.Rel(root, child)
	return err == nil && relative != "." && relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func validRelativeCgroup(value string) bool {
	if value == "" || filepath.IsAbs(value) || filepath.Clean(value) != value ||
		strings.ContainsRune(value, 0) || strings.Contains(value, `\`) {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func (config Config) peerAllowed(uid, gid uint32) bool {
	return config.AllowedUID == uid && config.AllowedGID == gid
}

func (config Config) validateRequest(request LaunchRequest) error {
	guestBytes := uint64(request.GuestMemoryMiB) << 20
	if !validLaunchRequestShape(request) || request.Timeout > config.MaxTimeout ||
		request.MemoryMaxBytes > config.MaxMemoryBytes ||
		request.MemoryMaxBytes < guestBytes+(128<<20) ||
		request.PIDsMax > config.MaxPIDs ||
		request.CPUQuotaMicros > config.MaxCPUQuotaMicros ||
		request.OutputDriveBytes > config.MaxOutputDriveBytes ||
		request.MaxCapturedOutputBytes > config.MaxCapturedOutputBytes {
		return launcherError(CodeResourceUnsafe)
	}
	vcpus := (request.CPUQuotaMicros + request.CPUPeriodMicros - 1) / request.CPUPeriodMicros
	if vcpus == 0 || vcpus > uint64(maxFirecrackerVCPUs) {
		return launcherError(CodeResourceUnsafe)
	}
	return nil
}
