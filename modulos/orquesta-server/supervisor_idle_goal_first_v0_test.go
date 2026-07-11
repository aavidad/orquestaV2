package orquestaserver

import (
	"context"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestRuntimeV0LaunchIdleSelfImprovementGoalsV0UsesRequiredTestBinderV0(t *testing.T) {
	binder := &idleSelfImprovementRequiredTestBinderForTestV0{}
	launcher := &idleSelfImprovementGoalLauncherForTestV0{}
	goalStates := newMemoryGoalStateStoreV0()
	runtime, err := NewRuntimeV0(ConfigV0{StateDir: t.TempDir(), AuditDisabled: true}, RuntimeDepsV0{
		GoalStateStore:             goalStates,
		GoalRequiredTestSpecBinder: binder,
		StateStore:                 &memoryStateStoreV0{},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	request := idleSelfImprovementAcceptanceRequestForGoalFirstTestV0()
	runtime.launchIdleSelfImprovementGoalsV0(context.Background(), launcher, []IdleSelfImprovementRequestV0{request})

	if binder.calls != 1 || launcher.calls != 1 ||
		len(launcher.spec.RequiredTests) != 1 ||
		!launcher.spec.ClosurePolicy.RequireIndependentRequiredTestAttestation ||
		!containsStringForTestV0(launcher.spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs, "criterion-ref-self-audit-launch-001") {
		t.Fatalf("binder_calls=%d launcher_calls=%d spec=%+v", binder.calls, launcher.calls, launcher.spec)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), request.RequestRef)
	if err != nil || state.Status != orquestagoal.GoalStatusRunningV0 ||
		state.Spec.ImplementerAgentRef != "agent-ref-idle-self-improvement-binder-001" {
		t.Fatalf("state=%+v err=%v", state, err)
	}
}

func TestRuntimeV0LaunchIdleSelfImprovementGoalsV0FailsClosedWithoutRequiredTestBinderV0(t *testing.T) {
	launcher := &idleSelfImprovementGoalLauncherForTestV0{}
	runtime, err := NewRuntimeV0(ConfigV0{StateDir: t.TempDir(), AuditDisabled: true}, RuntimeDepsV0{
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     &memoryStateStoreV0{},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.launchIdleSelfImprovementGoalsV0(
		context.Background(),
		launcher,
		[]IdleSelfImprovementRequestV0{idleSelfImprovementAcceptanceRequestForGoalFirstTestV0()},
	)
	if launcher.calls != 0 {
		t.Fatalf("launcher debe quedar sin invocar sin binder: calls=%d spec=%+v", launcher.calls, launcher.spec)
	}
}

func TestIdleSelfImprovementGoalWorkSpecV0CompilaAcceptanceChecksTipadosV0(t *testing.T) {
	request := IdleSelfImprovementRequestV0{
		RequestRef:     "request-ref-idle-acceptance-checks-001",
		ProjectRef:     "project-ref-orquesta",
		FailureSummary: "Corregir hallazgo determinado.",
		WriteSet:       []string{"modulos/orquesta-server"},
		RequiredTests: []string{
			"go test -count=1 ./modulos/orquesta-server",
		},
		AcceptanceCriteria: []string{"texto legacy advisory que no crea checks"},
		AcceptanceChecks: []orquestaautoprogramming.AutoprogrammingAcceptanceCheckV0{
			{
				CriterionRef: "criterion-ref-self-audit-001",
				Description:  "El hallazgo A queda cubierto.",
				Command:      "go test -count=1 ./modulos/orquesta-server",
			},
			{
				CriterionRef: "criterion-ref-self-audit-002",
				Description:  "El hallazgo B queda cubierto.",
				Command:      "go test -count=1 ./modulos/orquesta-server",
			},
		},
	}

	spec := (&RuntimeV0{}).idleSelfImprovementGoalWorkSpecV0(request)
	if len(spec.RequiredTests) != 1 ||
		spec.RequiredTests[0].Command != "go test -count=1 ./modulos/orquesta-server" ||
		spec.RequiredTests[0].CommandRef == "" ||
		spec.RequiredTests[0].CommandSHA256 == "" ||
		spec.RequiredTests[0].DefinitionSHA256 == "" ||
		!containsStringForTestV0(spec.RequiredTests[0].AcceptanceCriteriaRefs, "criterion-ref-self-audit-001") ||
		!containsStringForTestV0(spec.RequiredTests[0].AcceptanceCriteriaRefs, "criterion-ref-self-audit-002") ||
		!containsStringForTestV0(spec.RequiredTests[0].AcceptanceCriteria, "El hallazgo A queda cubierto.") ||
		!containsStringForTestV0(spec.RequiredTests[0].AcceptanceCriteria, "El hallazgo B queda cubierto.") ||
		!spec.ClosurePolicy.RequireIndependentRequiredTestAttestation ||
		!containsStringForTestV0(spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs, "criterion-ref-self-audit-001") ||
		!containsStringForTestV0(spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs, "criterion-ref-self-audit-002") {
		t.Fatalf("spec=%+v", spec)
	}
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) != 0 {
		t.Fatalf("spec invalid issues=%+v spec=%+v", issues, spec)
	}
}

func TestIdleSelfImprovementGoalWorkSpecV0RechazaAcceptanceChecksInvalidosV0(t *testing.T) {
	spec := (&RuntimeV0{}).idleSelfImprovementGoalWorkSpecV0(IdleSelfImprovementRequestV0{
		RequestRef:     "request-ref-idle-invalid-acceptance-check-001",
		ProjectRef:     "project-ref-orquesta",
		FailureSummary: "No aceptar contrato incompleto.",
		WriteSet:       []string{"modulos/orquesta-server"},
		AcceptanceChecks: []orquestaautoprogramming.AutoprogrammingAcceptanceCheckV0{{
			CriterionRef: "criterion-ref-incomplete-001",
			Description:  "No hay comando determinista.",
		}},
	})
	if !spec.ClosurePolicy.RequireIndependentRequiredTestAttestation || len(orquestagoal.ValidateGoalWorkSpecV0(spec)) == 0 {
		t.Fatalf("spec invalid acceptance check must fail closed: %+v", spec)
	}
}

func idleSelfImprovementAcceptanceRequestForGoalFirstTestV0() IdleSelfImprovementRequestV0 {
	return IdleSelfImprovementRequestV0{
		RequestRef:     "request-ref-idle-acceptance-launch-001",
		ProjectRef:     "project-ref-orquesta",
		FailureSummary: "Corregir hallazgo determinado.",
		WriteSet:       []string{"modulos/orquesta-server"},
		AcceptanceChecks: []orquestaautoprogramming.AutoprogrammingAcceptanceCheckV0{{
			CriterionRef: "criterion-ref-self-audit-launch-001",
			Description:  "El hallazgo queda cubierto.",
			Command:      "go test -count=1 ./modulos/orquesta-server",
		}},
	}
}

type idleSelfImprovementRequiredTestBinderForTestV0 struct{ calls int }

func (binder *idleSelfImprovementRequiredTestBinderForTestV0) BindGoalRequiredTestSpecV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalWorkSpecV0, error) {
	binder.calls++
	spec.ImplementerAgentRef = "agent-ref-idle-self-improvement-binder-001"
	spec.ImplementerCredentialRef = "credential-ref-idle-self-improvement-binder-001"
	spec.ClosurePolicy.RequiredAttestorTrustPolicyRef = "trust-policy-ref-idle-self-improvement-binder-001"
	return spec, nil
}

type idleSelfImprovementGoalLauncherForTestV0 struct {
	calls int
	spec  orquestagoal.GoalWorkSpecV0
}

func (launcher *idleSelfImprovementGoalLauncherForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	launcher.calls++
	launcher.spec = spec
	return orquestagoal.GoalLaunchReceiptV0{
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "external-" + spec.GoalRef,
	}, nil
}
