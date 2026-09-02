package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestComposeAgentEnvironmentPreservationBuilderDefaultsToDeploymentCAS(t *testing.T) {
	builder, err := composeAgentEnvironmentPreservationBuilder(nil, newDAGArtifacts())
	if err != nil || builder == nil {
		t.Fatalf("default preservation builder missing: builder=%v err=%v", builder != nil, err)
	}
	if builder, err := composeAgentEnvironmentPreservationBuilder(nil, nil); err == nil || builder != nil {
		t.Fatalf("nil deployment CAS accepted: builder=%v err=%v", builder != nil, err)
	}
}

func TestComposeAgentEnvironmentPreservationBuilderKeepsExplicitTestOverride(t *testing.T) {
	marker := errors.New("test.preservation_builder_override")
	configured := func(context.Context, ports.AgentPreserveReceipt, application.GoalRecord,
		time.Time,
	) (application.ComprobantePreservacionEntornoAgente, error) {
		return application.ComprobantePreservacionEntornoAgente{}, marker
	}
	builder, err := composeAgentEnvironmentPreservationBuilder(configured, nil)
	if err != nil || builder == nil {
		t.Fatalf("configured builder rejected: builder=%v err=%v", builder != nil, err)
	}
	if _, err := builder(context.Background(), ports.AgentPreserveReceipt{}, application.GoalRecord{}, time.Time{}); !errors.Is(err, marker) {
		t.Fatalf("configured builder replaced: %v", err)
	}
}
