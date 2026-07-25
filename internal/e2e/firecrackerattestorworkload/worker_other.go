//go:build !linux

package firecrackerattestorworkload

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/adapters/attestor/firecrackerclient"
)

const codeUnavailable = "firecracker_physical_e2e.unavailable"

type Config struct {
	SocketPath          string
	ExpectedAssetDigest string
	WorkRoot            string
	GitCommand          string
	TestSleep           time.Duration
	Limits              firecrackerclient.Limits
	ConcurrentRuns      int
}

type Summary struct {
	Attempted    int                  `json:"attempted"`
	Passed       int                  `json:"passed"`
	PolicyDigest string               `json:"policy_digest"`
	AttestorRef  string               `json:"attestor_ref"`
	Attestations []AttestationSummary `json:"attestations"`
}

type AttestationSummary struct {
	SubjectDigest          string `json:"subject_digest"`
	WorkspaceBindingDigest string `json:"workspace_binding_digest"`
	ChangeSetDigest        string `json:"change_set_digest"`
	ReceiptRef             string `json:"receipt_ref"`
	RunID                  string `json:"run_id"`
}

type Error struct {
	Code  string
	cause error
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

func ErrorCode(err error) string {
	var workerError *Error
	if errors.As(err, &workerError) {
		return workerError.Code
	}
	return ""
}

func Run(context.Context, Config) (Summary, error) {
	return Summary{}, &Error{Code: codeUnavailable}
}
