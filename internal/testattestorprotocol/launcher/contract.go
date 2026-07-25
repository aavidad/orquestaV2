// Package launcher defines the narrow, transport-facing contract between an
// unprivileged TestAttestor adapter and the privileged Firecracker launcher.
// It contains no socket, VMM, configuration, policy, or application logic.
package launcher

import (
	"context"
	"os"
	"time"
)

// CPUPeriodMicrosV0 is part of the launcher wire contract. Callers configure
// only the quota; both protocol ends must use this fixed period to derive vCPU
// capacity identically.
const CPUPeriodMicrosV0 = uint64(100_000)

type LaunchRequest struct {
	Nonce                  string
	InputDigest            string
	SubjectDigest          string
	PolicyDigest           string
	Timeout                time.Duration
	GuestMemoryMiB         uint32
	MemoryMaxBytes         uint64
	PIDsMax                uint32
	CPUQuotaMicros         uint64
	CPUPeriodMicros        uint64
	OutputDriveBytes       uint64
	MaxCapturedOutputBytes uint64
}

type LaunchResponse struct {
	Nonce        string
	Code         string
	OutputDigest string
	AssetDigest  string
}

type ClientResult struct {
	Response LaunchResponse
	Output   *os.File
}

// Identity is the immutable logical identity of one configured launcher
// transport. It binds transport semantics and its trust target; it is not by
// itself proof that a peer or asset is trusted.
type Identity struct {
	Ref    string
	Digest string
}

// Client is implemented by the authenticated local launcher transport. The
// application adapter depends on this contract and receives the implementation
// from composition; it never imports the privileged launcher adapter. Launch
// must obey context cancellation, release its own transport resources and
// never retain the borrowed input descriptor after returning.
type Client interface {
	Identity() Identity
	Launch(context.Context, LaunchRequest, *os.File, int64) (ClientResult, error)
}
