package codex

import (
	"errors"
	"reflect"
	"testing"
)

func TestSyncDirectoryAndParentUsesCausalChildFirstOrder(t *testing.T) {
	var synced []string
	err := syncDirectoryAndParent("executions/abc", func(directory string) error {
		synced = append(synced, directory)
		return nil
	})
	if err != nil {
		t.Fatalf("syncDirectoryAndParent() error = %v", err)
	}
	if want := []string{"executions/abc", "executions"}; !reflect.DeepEqual(synced, want) {
		t.Fatalf("sync order = %#v, want %#v", synced, want)
	}

	wantErr := errors.New("sync failed")
	synced = nil
	err = syncDirectoryAndParent("executions/abc", func(directory string) error {
		synced = append(synced, directory)
		if directory == "executions/abc" {
			return wantErr
		}
		return nil
	})
	if !errors.Is(err, wantErr) || !reflect.DeepEqual(synced, []string{"executions/abc"}) {
		t.Fatalf("failed sync: order=%#v error=%v", synced, err)
	}
}

func TestTerminalPostMortemKeepsHistoricalAbsenceCompatible(t *testing.T) {
	for _, schemaVersion := range []int{
		legacyStateSchemaVersion,
		intermediateStateSchemaVersion,
		accountlessStateSchemaVersion,
		profileStateSchemaVersion,
		stateSchemaVersion,
	} {
		if err := validateTerminalPostMortem(terminalRecord{
			SchemaVersion:       schemaVersion,
			DiagnosticTruncated: true,
		}); err != nil {
			t.Fatalf("historical V%d terminal rejected: %v", schemaVersion, err)
		}
	}

	if err := validateTerminalPostMortem(terminalRecord{
		SchemaVersion:   stateSchemaVersion,
		SupervisorCause: supervisorCauseNatural,
	}); ErrorCode(err) != CodeStateInvalid {
		t.Fatalf("partial post-mortem error=%v code=%q", err, ErrorCode(err))
	}
}
