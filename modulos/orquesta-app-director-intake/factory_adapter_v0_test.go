package orquestaappdirectorintake

import (
	"reflect"
	"testing"
)

func TestAppDirectorInputSpecFromFactoryV0PreservesDirectorContext(t *testing.T) {
	spec := validHighAutonomyAppSpecForDirectorIntakeTestV0(t)

	input := AppDirectorInputSpecFromFactoryV0(spec)

	if input.SchemaVersion != AppDirectorInputSpecSchemaVersionV0 {
		t.Fatalf("schema=%q", input.SchemaVersion)
	}
	if input.SpecID != spec.SpecID || input.App.Slug != spec.App.Slug {
		t.Fatalf("refs input=%+v spec=%+v", input, spec)
	}
	if input.App.Nombre != spec.App.Nombre ||
		input.App.Objetivo != spec.App.Objetivo ||
		input.App.TipoApp != spec.App.TipoApp {
		t.Fatalf("app context input=%+v spec=%+v", input.App, spec.App)
	}
	if !reflect.DeepEqual(input.Platforms, spec.Platforms) ||
		!reflect.DeepEqual(input.Data.Needs, spec.Data.Needs) {
		t.Fatalf("context input=%+v spec=%+v", input, spec)
	}
	if input.Data.PersistenceRequired != spec.Data.PersistenceRequired ||
		input.I18N.Enabled != spec.I18N.Enabled ||
		input.Quality.Tests != spec.Quality.Tests ||
		input.AgentPreferences.Autonomy != spec.AgentPreferences.Autonomy {
		t.Fatalf("flags input=%+v spec=%+v", input, spec)
	}
}

func TestPrepareAppDirectorIntakeV0LegacyFactoryMatchesNeutralInput(t *testing.T) {
	spec := validHighAutonomyAppSpecForDirectorIntakeTestV0(t)
	base := PrepareAppDirectorInputRequestV0{
		RunRef:        "run-app-director-neutral-compat-001",
		ProjectRef:    "project-app-director-neutral-compat-001",
		OccurredAt:    "2026-05-23T11:10:00Z",
		CorrelationID: "corr-app-director-neutral-compat-001",
		RequestedBy:   "orquesta-app-director-intake-test",
	}

	legacy, err := PrepareAppDirectorIntakeV0(PrepareAppDirectorIntakeRequestV0{
		RunRef:        base.RunRef,
		ProjectRef:    base.ProjectRef,
		OccurredAt:    base.OccurredAt,
		CorrelationID: base.CorrelationID,
		RequestedBy:   base.RequestedBy,
		AppSpec:       spec,
	})
	if err != nil {
		t.Fatalf("PrepareAppDirectorIntakeV0: %v", err)
	}
	base.AppSpec = AppDirectorInputSpecFromFactoryV0(spec)
	neutral, err := PrepareAppDirectorInputV0(base)
	if err != nil {
		t.Fatalf("PrepareAppDirectorInputV0: %v", err)
	}

	if !reflect.DeepEqual(legacy, neutral) {
		t.Fatalf("legacy and neutral prepared differ:\nlegacy=%+v\nneutral=%+v", legacy, neutral)
	}
}
