package main

import (
	"path/filepath"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
	orquestastatefileoutbox "orquesta/modulos/orquesta-state-file/outbox"
)

func TestBuildStackFromEnvV0UsaConectoresDurablesFileBased(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeDir)
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}

	assertServerStackDurableStoresV0(t, stack)
}

func assertServerStackDurableStoresV0(
	t *testing.T,
	stack orquestaappcodexstack.StackV0,
) {
	t.Helper()
	assertTypeV0[*orquestastatefile.StoreV0](t, "RunStore", stack.Stores.RunStore)
	assertTypeV0[*orquestastatefile.StoreV0](t, "EventSink", stack.Stores.EventSink)
	assertTypeV0[*orquestastatefile.StoreV0](t, "TaskStore", stack.Stores.TaskStore)
	assertTypeV0[*orquestastatefile.StoreV0](t, "ProcessRegistry", stack.Stores.ProcessRegistry)
	assertTypeV0[*orquestastatefileoutbox.FileOutboxLedgerV0](t, "OutboxLedger", stack.Stores.OutboxLedger)
	assertTypeV0[*orquestarunfile.RunFileStoreV0](t, "AppChangeStore", stack.Stores.AppChangeStore)
	assertTypeV0[*orquestarunfile.RunFileStoreV0](t, "RunControl", stack.Stores.RunControl)
	assertTypeV0[*orquestarunfile.RunFileStoreV0](t, "RunQueue", stack.Stores.RunQueue)
	assertTypeV0[*orquestaruntimecodexdelivery.FileCodexReceiptDescriptorStoreV0](t, "ReceiptStore", stack.Stores.ReceiptStore)
	assertTypeV0[*orquestaruntimecodexdelivery.FileCodexProgressStateStoreV0](t, "ProgressState", stack.Stores.ProgressState)
}

func assertTypeV0[T any](t *testing.T, name string, value any) {
	t.Helper()
	if _, ok := value.(T); !ok {
		t.Fatalf("%s usa %T, debe usar %T", name, value, *new(T))
	}
}
