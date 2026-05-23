package main

import (
	"encoding/json"
	"os"
	"testing"
)

func resultStartupEvidenceContainsV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeStartupJSONForTestV0(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readStartupQueueForTestV0(t *testing.T, path string) startupQueueSnapshotV0 {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var snapshot startupQueueSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return snapshot
}

func readStartupControlForTestV0(t *testing.T, path string) startupControlSnapshotV0 {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var snapshot startupControlSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return snapshot
}
