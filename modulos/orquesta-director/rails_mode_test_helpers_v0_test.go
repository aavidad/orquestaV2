package orquestadirector

import (
	"testing"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func enableDirectorRailsModeEnforcedForTestV0(t *testing.T) {
	t.Helper()
	t.Setenv(orquestarails.RailsModeEnvV0, orquestarails.RailsModeEnforcedV0)
	t.Setenv(orquestarails.DetailProhibitedRailsEnvV0, "on")
}
