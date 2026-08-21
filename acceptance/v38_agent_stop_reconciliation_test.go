package acceptance_test

import (
	"reflect"
	"testing"

	"orquesta/internal/adapters/agent/agentmicrovm"
	"orquesta/internal/application"
)

func TestV38B11PublishedAdapterStopsButCannotInventReconciliation(t *testing.T) {
	var _ application.AgentController = (*agentmicrovm.Adapter)(nil)
	if _, found := reflect.TypeOf((*agentmicrovm.Adapter)(nil)).MethodByName("ReconcileStop"); found {
		t.Fatal("published adapter invented read-only reconciliation absent from the sibling protocol")
	}
}
