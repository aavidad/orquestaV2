//go:build linux

package codex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"orquesta/internal/ports"
)

func TestStopLegacyV4ThroughV6RejectsMissingFenceAndPreservesDurableTerminal(t *testing.T) {
	for _, schemaVersion := range []int{
		intermediateStateSchemaVersion,
		accountlessStateSchemaVersion,
		profileStateSchemaVersion,
	} {
		t.Run(schemaName(schemaVersion), func(t *testing.T) {
			config := testConfig(t)
			request := testRequest(
				t, "stop-terminal-"+schemaName(schemaVersion), "helper:success", 1024,
			)
			if schemaVersion == profileStateSchemaVersion {
				seedPersistedV6Launch(t, config, request)
			} else {
				seedAccountlessLaunchRecord(t, config, request, schemaVersion)
			}
			adapter := openTestAdapter(t, config)
			record, runPath, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
			if err != nil || !found {
				t.Fatalf("load V%d found=%v error=%v", schemaVersion, found, err)
			}
			original := terminalRecord{
				SchemaVersion: stateSchemaVersion, RequestHash: record.RequestHash,
				Status: ports.AgentCompleted, MediaType: request.ArtifactMediaType,
				Artifact:   "terminal:" + schemaName(schemaVersion),
				ObservedAt: config.Now(),
			}
			if _, err := adapter.persistTerminal(
				runPath, original, record.SpecHash, record.MaxOutputBytes,
			); err != nil {
				t.Fatal(err)
			}
			launch, err := record.receipt(request.ExecutionRef)
			if err != nil {
				t.Fatal(err)
			}
			stop := stopRequestForLaunch(
				launch, ports.AgentStopForced, "stop:terminal:"+schemaName(schemaVersion),
			)
			receipt, err := adapter.Stop(context.Background(), stop)
			if ports.AgentContractErrorCode(err) != "agent.stop_launch_action_fence_required" ||
				receipt != (ports.AgentStopReceipt{}) {
				t.Fatalf("Stop(V%d) receipt=%+v error=%v", schemaVersion, receipt, err)
			}
			persisted := readPersistedTerminal(t, config, runPath)
			if !reflect.DeepEqual(persisted, original) {
				t.Fatalf("Stop(V%d) changed terminal: got=%+v want=%+v", schemaVersion, persisted, original)
			}
			if _, err := os.Stat(filepath.Join(
				config.WorkRoot, filepath.FromSlash(runPath), quarantineIntentFileName,
			)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("Stop(V%d) created quarantine intent: %v", schemaVersion, err)
			}
			if _, err := os.Stat(filepath.Join(
				config.WorkRoot, filepath.FromSlash(runPath), launchUpgradeFileName,
			)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("Stop(V%d) created V7 binding: %v", schemaVersion, err)
			}
		})
	}
}

func schemaName(schemaVersion int) string {
	switch schemaVersion {
	case intermediateStateSchemaVersion:
		return "v4"
	case accountlessStateSchemaVersion:
		return "v5"
	case profileStateSchemaVersion:
		return "v6"
	default:
		return "unknown"
	}
}
