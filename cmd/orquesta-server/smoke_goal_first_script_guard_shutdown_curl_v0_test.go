package main

import (
	"strings"
	"testing"
)

func TestScriptShutdownCurlCommandsV0DetectaRequestPostLargoV0(t *testing.T) {
	text := strings.Join([]string{
		`shutdown_status="$(curl -sS \`,
		`  --request \`,
		`  POST \`,
		`  "$base_url/api/v0/server/shutdown" \`,
		`  --data-binary "{\"request_id\":\"req-shutdown\",\"cleanup_goal_backends\":true}"`,
		`)"`,
		`shutdown_status_2="$(curl -sS \`,
		`  --request=POST \`,
		`  "$base_url/api/v0/server/shutdown" \`,
		`  --data-binary "{\"request_id\":\"req-shutdown-2\",\"cleanup_goal_backends\":true}"`,
		`)"`,
	}, "\n")

	commands := scriptShutdownCurlCommandsV0(text)
	if len(commands) != 2 {
		t.Fatalf("detector debe cubrir --request POST y --request=POST: %+v", commands)
	}
	for _, command := range commands {
		if !strings.Contains(command, "cleanup_goal_backends") {
			t.Fatalf("detector no conservo payload cleanup: %s", command)
		}
	}
}
