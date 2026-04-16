package runtimepolicy

import (
	"encoding/json"
	"strings"
)

func RuntimeHandleDeliveryContextScore(metadataJSON, capabilitiesJSON string) int {
	meta := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(metadataJSON)), &meta); err != nil {
		meta = map[string]any{}
	}
	caps := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(capabilitiesJSON)), &caps); err != nil {
		caps = map[string]any{}
	}

	score := 0
	for _, key := range []string{
		"driver",
		"rendered_command",
		"wrapped_command",
		"working_dir",
		"external_session_id",
		"stdin_path",
		"supervisor_ref",
		"tmux_command",
		"tmux_session",
		"mailbox_delivery_mode",
	} {
		if strings.TrimSpace(mapValueString(meta, key, "")) != "" {
			score++
		}
	}
	for key := range caps {
		if key == "mailbox_delivery_mode" {
			score++
		}
	}
	return score
}

func RuntimeHandleHasDeliveryContext(metadataJSON, capabilitiesJSON string) bool {
	return RuntimeHandleDeliveryContextScore(metadataJSON, capabilitiesJSON) > 0
}
