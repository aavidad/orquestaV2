package orquestaruntimecodex

import (
	"testing"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func enableCodexRailsModeEnforcedForTestV0(t *testing.T) {
	t.Helper()
	t.Setenv(orquestarails.RailsModeEnvV0, orquestarails.RailsModeEnforcedV0)
}
