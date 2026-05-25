package orquestaserver

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileStateStoreV0RecuperaStateV0(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultStateFileV0)
	store, err := NewFileStateStoreV0(path)
	if err != nil {
		t.Fatalf("NewFileStateStoreV0: %v", err)
	}
	want := StateV0{
		Status:          "running",
		PID:             1234,
		Addr:            "127.0.0.1:8787",
		StartedAt:       "2026-05-12T10:00:00Z",
		LastHeartbeatAt: "2026-05-12T10:01:00Z",
		EffectiveConfig: ServerEffectiveConfigV0{
			SchemaVersion: ServerEffectiveConfigSchemaVersionV0,
			Settings: []ServerConfigSettingV0{{
				Key:       "ORQUESTA_SERVER_MAX_RUNS_PER_TICK",
				Value:     "10",
				Scope:     "server_supervisor",
				Editable:  true,
				Canonical: true,
			}},
		},
	}
	if err := store.SaveServerStateV0(context.Background(), want); err != nil {
		t.Fatalf("SaveServerStateV0: %v", err)
	}
	got, err := store.LoadServerStateV0(context.Background())
	if err != nil {
		t.Fatalf("LoadServerStateV0: %v", err)
	}
	if got.SchemaVersion != StateSchemaVersionV0 ||
		got.Status != want.Status ||
		got.PID != want.PID ||
		got.Addr != want.Addr ||
		len(got.EffectiveConfig.Settings) != 1 ||
		got.EffectiveConfig.Settings[0].Key != "ORQUESTA_SERVER_MAX_RUNS_PER_TICK" {
		t.Fatalf("state=%+v want=%+v", got, want)
	}
}

func TestFileStateStoreV0NoFiltraPathEnErrorV0(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultStateFileV0)
	store, err := NewFileStateStoreV0(path)
	if err != nil {
		t.Fatalf("NewFileStateStoreV0: %v", err)
	}
	_, err = store.LoadServerStateV0(context.Background())
	if err == nil {
		t.Fatalf("esperaba error")
	}
	if strings.Contains(err.Error(), path) {
		t.Fatalf("error filtra path: %v", err)
	}
}
