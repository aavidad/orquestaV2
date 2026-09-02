package config

import (
	"errors"
	"testing"
	"time"
)

func TestStateMigrationResolverReadsOnlyCanonicalSQLiteOptions(t *testing.T) {
	source := []byte(`[state.sqlite]
path = "/srv/orquesta/state/orquesta.sqlite"
busy_timeout = "7s"
max_open_connections = 3

[runtime]
provider = 17
isolation = "invalid"

[runtime.microvm]
socket_path = 99

[test_attestor]
provider = 23
`)
	if _, err := Resolve(ResolveOptions{TOML: source}); err == nil {
		t.Fatal("runtime resolver accepted intentionally invalid provider configuration")
	}
	snapshot, err := ResolveForStateMigration(ResolveOptions{
		TOML: source,
		Environment: map[string]string{
			"ORQUESTA_STATE_SQLITE_BUSY_TIMEOUT":                  "9s",
			"ORQUESTA_RUNTIME_MICROVM_CREDENTIAL_BROKER_PEER_UID": "not-an-integer",
			"ORQUESTA_TEST_ATTESTOR_PROVIDER":                     "not-a-provider",
		},
	})
	if err != nil {
		t.Fatalf("ResolveForStateMigration() error=%v", err)
	}
	if snapshot.StateSQLitePath() != "/srv/orquesta/state/orquesta.sqlite" ||
		snapshot.StateSQLiteBusyTimeout() != 9*time.Second ||
		snapshot.StateSQLiteMaxOpenConnections() != 3 {
		t.Fatalf("state options path=%q timeout=%s connections=%d",
			snapshot.StateSQLitePath(), snapshot.StateSQLiteBusyTimeout(),
			snapshot.StateSQLiteMaxOpenConnections())
	}
}

func TestStateMigrationResolverStillRejectsInvalidStateAndUnknownInputs(t *testing.T) {
	for _, options := range []ResolveOptions{
		{TOML: []byte("[state.sqlite]\nbusy_timeout = \"0s\"\n")},
		{TOML: []byte("[unknown]\nvalue = 1\n")},
		{Environment: map[string]string{"ORQUESTA_UNKNOWN": "value"}},
	} {
		if _, err := ResolveForStateMigration(options); err == nil {
			t.Fatalf("accepted invalid state migration options=%+v", options)
		} else {
			var configError *Error
			if !errors.As(err, &configError) {
				t.Fatalf("untyped config error=%v", err)
			}
		}
	}
}
