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
	return live, nil
}
