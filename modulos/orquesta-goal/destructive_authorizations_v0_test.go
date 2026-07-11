package orquestagoal

import "testing"

func TestValidateGoalWorkSpecV0AceptaAutorizacionRenameDentroDelWriteSetV0(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef: "goal-ref-destructive-rename-001", Objective: "Renombrar un documento.",
		DirectorKind: GoalDirectorKindRuntimeGoalV0,
		WriteSet:     []GoalWriteScopeV0{{Path: "docs"}},
		DestructiveAuthorizations: []GoalDestructiveChangeAuthorizationV0{{
			Kind: GoalDestructiveChangeAuthorizationRenameV0, PreviousPath: "docs/original.md", CurrentPath: "docs/renombrado.md",
		}},
	}
	if issues := ValidateGoalWorkSpecV0(spec); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestValidateGoalWorkSpecV0RechazaAutorizacionDestructivaFueraDelWriteSetV0(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef: "goal-ref-destructive-outside-001", Objective: "Eliminar un documento.",
		DirectorKind: GoalDirectorKindRuntimeGoalV0,
		WriteSet:     []GoalWriteScopeV0{{Path: "docs"}},
		DestructiveAuthorizations: []GoalDestructiveChangeAuthorizationV0{{
			Kind: GoalDestructiveChangeAuthorizationRemoveV0, Path: "fuera.md",
		}},
	}
	if issues := ValidateGoalWorkSpecV0(spec); !hasGoalIssueV0(issues, ErrGoalDestructiveAuthorizationOutsideWriteSetV0) {
		t.Fatalf("issues=%+v", issues)
	}
}
