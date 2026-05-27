package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const guardianCandidateDynamicAddrV0 = "127.0.0.1:0"

type guardianCandidateAddressPlanV0 struct {
	Dynamic bool
}

type guardianCandidateStateV0 struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Addr          string `json:"addr"`
}

func guardianCandidateAddressPolicyV0(raw string) (string, guardianCandidateAddressPlanV0, error) {
	addr := strings.TrimSpace(raw)
	if addr == "" {
		return guardianCandidateDynamicAddrV0, guardianCandidateAddressPlanV0{Dynamic: true}, nil
	}
	if strings.Contains(addr, "://") || strings.ContainsAny(addr, "/?#@") {
		return "", guardianCandidateAddressPlanV0{}, fmt.Errorf("candidate_addr_unowned")
	}
	if err := validateGuardianCandidateLoopbackAddrV0(addr); err != nil {
		return "", guardianCandidateAddressPlanV0{}, err
	}
	return addr, guardianCandidateAddressPlanV0{}, nil
}

func validateGuardianCandidateLoopbackAddrV0(addr string) error {
	host, port, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return fmt.Errorf("candidate_addr_unowned")
	}
	if strings.TrimSpace(port) == "" || strings.TrimSpace(port) == "0" {
		return fmt.Errorf("candidate_addr_unowned")
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return fmt.Errorf("candidate_addr_unowned")
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("candidate_addr_unowned")
	}
	return nil
}

func waitGuardianCandidateStateAddrV0(
	ctx context.Context,
	stateDir string,
	timeout time.Duration,
) (string, error) {
	deadline := time.Now().Add(timeout)
	path := filepath.Join(stateDir, "orquesta_server_state_v0.json")
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		addr, err := readGuardianCandidateStateAddrV0(path)
		if err == nil {
			return addr, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", fmt.Errorf("candidate_addr_unowned")
}

func readGuardianCandidateStateAddrV0(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("candidate_addr_unowned")
	}
	var state guardianCandidateStateV0
	if err := json.Unmarshal(data, &state); err != nil {
		return "", fmt.Errorf("candidate_addr_unowned")
	}
	if state.SchemaVersion != "orquesta_server_state.v0" || strings.TrimSpace(state.Status) != "running" {
		return "", fmt.Errorf("candidate_addr_unowned")
	}
	addr := strings.TrimSpace(state.Addr)
	if err := validateGuardianCandidateLoopbackAddrV0(addr); err != nil {
		return "", err
	}
	return addr, nil
}

func guardianCandidateProcessStillOwnedV0(waitDone <-chan error) error {
	select {
	case err := <-waitDone:
		if err != nil {
			return fmt.Errorf("candidate_addr_unowned")
		}
		return fmt.Errorf("candidate_addr_unowned")
	default:
		return nil
	}
}

func guardianCandidateReadinessReasonWithOwnershipV0(err error) string {
	if err == nil {
		return "candidate_address_owned"
	}
	if strings.Contains(err.Error(), "candidate_addr_unowned") {
		return "candidate_addr_unowned"
	}
	return guardianCandidateReadinessReasonCodeV0(err)
}
