package orquestaserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
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
		IdleSelfImprovementGoalSpec: &orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-state-001",
			Objective:     "Validar round-trip durable del goal",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server"}},
			ClosurePolicy: orquestagoal.GoalClosurePolicyV0{RequiredEvidenceRefs: []string{"evidence-ref-state-001"}},
		},
		IdleSelfImprovementGoalReceipt: &orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:          orquestagoal.GoalStatusAcceptedV0,
			GoalRef:         "goal-ref-state-001",
			ExternalGoalRef: "external-goal-ref-state-001",
			EvidenceRefs:    []string{"evidence-ref-state-receipt-001"},
		},
		IdleSelfImprovementGoalResult: &orquestagoal.GoalWorkResultV0{
			SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
			Status:        orquestagoal.GoalStatusCompleteV0,
			GoalRef:       "goal-ref-state-001",
			EvidenceRefs:  []string{"evidence-ref-state-001"},
		},
		IdleSelfImprovementGoalClosure: &orquestagoal.GoalClosureValidationV0{
			Status:       orquestagoal.GoalStatusAcceptedV0,
			Accepted:     true,
			EvidenceRefs: []string{"evidence-ref-state-001"},
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
		got.EffectiveConfig.Settings[0].Key != "ORQUESTA_SERVER_MAX_RUNS_PER_TICK" ||
		got.IdleSelfImprovementGoalSpec == nil ||
		got.IdleSelfImprovementGoalSpec.GoalRef != "goal-ref-state-001" ||
		got.IdleSelfImprovementGoalReceipt == nil ||
		got.IdleSelfImprovementGoalReceipt.ExternalGoalRef != "external-goal-ref-state-001" ||
		got.IdleSelfImprovementGoalResult == nil ||
		got.IdleSelfImprovementGoalResult.Status != orquestagoal.GoalStatusCompleteV0 ||
		got.IdleSelfImprovementGoalClosure == nil ||
		!got.IdleSelfImprovementGoalClosure.Accepted {
		t.Fatalf("state=%+v want=%+v", got, want)
	}
	assertServerDurableFilePolicyV0(t, filepath.Dir(path), filepath.Base(path))
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

func assertServerDurableFilePolicyV0(t *testing.T, dir string, name string) {
	t.Helper()
	info, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("stat durable file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("durable file mode=%#o", got)
	}
	leftovers, err := filepath.Glob(filepath.Join(dir, "."+name+".*.tmp"))
	if err != nil {
		t.Fatalf("glob temp: %v", err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("temps persistidos=%v", leftovers)
	}
	if _, err := os.Stat(filepath.Join(dir, name+".tmp")); !os.IsNotExist(err) {
		t.Fatalf("temp fijo presente err=%v", err)
	}
}
