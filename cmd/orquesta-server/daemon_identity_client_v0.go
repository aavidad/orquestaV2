package main

import (
	"encoding/json"
	"fmt"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func verifyLiveDaemonIdentityV0(snapshot orquestaserver.StateV0) (orquestaserver.ServerPublicStatusV0, error) {
	if strings.TrimSpace(snapshot.Addr) == "" ||
		strings.TrimSpace(snapshot.StartedAt) == "" ||
		strings.TrimSpace(snapshot.ProcessRef) == "" ||
		strings.TrimSpace(snapshot.DaemonEpochRef) == "" {
		return orquestaserver.ServerPublicStatusV0{}, fmt.Errorf("daemon_identity_unavailable")
	}
	body, err := getStatusBodyV0(snapshot.Addr)
	if err != nil {
		return orquestaserver.ServerPublicStatusV0{}, fmt.Errorf("daemon_identity_unavailable")
	}
	var live orquestaserver.ServerPublicStatusV0
	if err := json.Unmarshal(body, &live); err != nil {
		return orquestaserver.ServerPublicStatusV0{}, fmt.Errorf("daemon_identity_invalid")
	}
	if strings.TrimSpace(live.Addr) != strings.TrimSpace(snapshot.Addr) ||
		strings.TrimSpace(live.StartedAt) != strings.TrimSpace(snapshot.StartedAt) ||
		strings.TrimSpace(live.ProcessRef) != strings.TrimSpace(snapshot.ProcessRef) ||
		strings.TrimSpace(live.DaemonEpochRef) != strings.TrimSpace(snapshot.DaemonEpochRef) {
		return live, fmt.Errorf("daemon_identity_mismatch")
	}
	if !liveDaemonRuntimeIdentityMatchesV0(snapshot, live) {
		return live, fmt.Errorf("daemon_runtime_identity_mismatch")
	}
	return live, nil
}

func liveDaemonRuntimeIdentityMatchesV0(
	snapshot orquestaserver.StateV0,
	live orquestaserver.ServerPublicStatusV0,
) bool {
	expected := orquestaserver.NewServerPublicRuntimeIdentityV0(snapshot)
	if expected.BinarySHA256 == "" && expected.BuildRef == "" && expected.CommitRef == "" {
		return true
	}
	if expected.BinarySHA256 != "" &&
		strings.TrimSpace(live.RuntimeIdentity.BinarySHA256) != expected.BinarySHA256 {
		return false
	}
	if expected.BuildRef != "" &&
		strings.TrimSpace(live.RuntimeIdentity.BuildRef) != expected.BuildRef {
		return false
	}
	if expected.CommitRef != "" &&
		strings.TrimSpace(live.RuntimeIdentity.CommitRef) != expected.CommitRef {
		return false
	}
	return true
}
