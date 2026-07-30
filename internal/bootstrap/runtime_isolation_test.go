package bootstrap

import (
	"context"
	"testing"
)

func TestBuildRejectsMicroVMBeforeConstructingProcessAgent(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[runtime]\n", "[runtime]\nisolation = \"microvm\"\n")

	runtime, err := Build(context.Background(), Options{ConfigPath: configPath})
	if runtime != nil || err == nil || err.Error() != "bootstrap.runtime_isolation_not_composed" {
		t.Fatalf("microvm without connector: runtime=%v err=%v", runtime, err)
	}
	assertNoCompositionState(t, root)
}
