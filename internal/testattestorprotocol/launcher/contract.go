// Package launcher defines the narrow, transport-facing contract between an
// unprivileged TestAttestor adapter and the privileged Firecracker launcher.
// It contains no socket, VMM, configuration, policy, or application logic.
package launcher

import (
	"context"
	"os"
	"time"
)

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

// Client is implemented by the authenticated local launcher transport. The
// application adapter depends on this contract and receives the implementation
// from composition; it never imports the privileged launcher adapter.
type Client interface {
	Launch(context.Context, LaunchRequest, *os.File, int64) (ClientResult, error)
}
