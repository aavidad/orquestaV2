package orquestagoal

import (
	"strings"
	"testing"
)

func TestGoalWorkspaceRefForGoalV0EsEstableYNeutralAlProveedorV0(t *testing.T) {
	first := GoalWorkspaceRefForGoalV0(" goal-ref-workspace-authority-001 ")
	second := GoalWorkspaceRefForGoalV0("goal-ref-workspace-authority-001")
	if first == "" || first != second || first == GoalWorkspaceRefForGoalV0("goal-ref-workspace-authority-002") {
		t.Fatalf("first=%q second=%q", first, second)
	}
}

func TestGoalExecutionAuthorityForProviderV0EsCompletaEstableYDistingueProveedorV0(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef: "goal-ref-execution-authority-001", RequestRef: "request-ref-execution-authority-001",
		IntentManifestRef:    "intent-manifest-ref-request-ref-execution-authority-001",
		IntentManifestSHA256: strings.Repeat("a", 64),
	}
	providerA := GoalExecutionAuthorityForProviderV0(spec, "provider-ref-a")
	replay := GoalExecutionAuthorityForProviderV0(spec, "provider-ref-a")
	providerB := GoalExecutionAuthorityForProviderV0(spec, "provider-ref-b")
	if !GoalExecutionAuthorityMatchesV0(providerA, replay) || providerA.RuntimeGenerationRef == "" ||
		providerA.WorkspaceAuthoritySchemaVersion != GoalWorkspaceAuthoritySchemaV0 ||
		providerA.WorkspaceRef != GoalWorkspaceRefForGoalV0(spec.GoalRef) ||
		providerA.IntentManifestRef != spec.IntentManifestRef || providerA.IntentManifestSHA256 != spec.IntentManifestSHA256 ||
		len(GoalExecutionAuthorityIssuesV0(providerA, false)) != 0 {
		t.Fatalf("authority incompleta o no estable: provider_a=%+v replay=%+v", providerA, replay)
	}
	if providerA.RuntimeGenerationRef == providerB.RuntimeGenerationRef || providerA.ProviderRef == providerB.ProviderRef {
		t.Fatalf("generation no distingue proveedor: provider_a=%+v provider_b=%+v", providerA, providerB)
	}
	request := GoalObservationRequestForAuthorityV0(providerA, "external-goal-ref-001")
	if !GoalExecutionAuthorityMatchesV0(providerA, GoalExecutionAuthorityFromObservationV0(request)) ||
		request.IntentManifestRef != spec.IntentManifestRef || request.IntentManifestSHA256 != spec.IntentManifestSHA256 {
		t.Fatalf("observation perdio autoridad: %+v", request)
	}
}

func TestValidateGoalLaunchReceiptV0NoAceptaWorkspaceDePrepareComoAutoridadPersistidaV0(t *testing.T) {
	issues := ValidateGoalLaunchReceiptV0(GoalLaunchReceiptV0{
		Status:                          GoalStatusRunningV0,
		GoalRef:                         "goal-ref-workspace-authority-001",
		WorkspaceAuthoritySchemaVersion: GoalWorkspaceAuthoritySchemaV0,
		WorkspaceRef:                    GoalWorkspaceRefForGoalV0("goal-ref-workspace-authority-001"),
	})
	if !hasGoalIssueFieldV0(issues, "provider_ref") || !hasGoalIssueFieldV0(issues, "runtime_generation_ref") {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestValidateGoalWorkspaceBindingV0PermitePrepareSinRuntimeV0(t *testing.T) {
	issues := ValidateGoalWorkspaceBindingV0(GoalWorkspaceBindingV0{
		WorkspaceRef:   GoalWorkspaceRefForGoalV0("goal-ref-workspace-prepare-001"),
		ProjectWorkDir: "/physical/path-owned-by-adapter",
	}, false)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestValidateGoalWorkspaceAuthorityV0DistingueLegacyDeContratoVersionadoV0(t *testing.T) {
	goalRef := "goal-ref-workspace-authority-shapes-001"
	workspaceRef := GoalWorkspaceRefForGoalV0(goalRef)
	tests := []struct {
		name       string
		schema     string
		workspace  string
		provider   string
		generation string
		valid      bool
	}{
		{name: "legacy empty", valid: true},
		{name: "legacy generation only", generation: "generation-ref-pre057", valid: true},
		{name: "legacy workspace only", workspace: workspaceRef},
		{name: "legacy full unversioned", workspace: workspaceRef, provider: "provider-ref-codex", generation: "generation-ref-unversioned"},
		{name: "versioned empty", schema: GoalWorkspaceAuthoritySchemaV0},
		{name: "versioned workspace only", schema: GoalWorkspaceAuthoritySchemaV0, workspace: workspaceRef},
		{name: "versioned pair", schema: GoalWorkspaceAuthoritySchemaV0, workspace: workspaceRef, provider: "provider-ref-codex"},
		{name: "versioned generation only", schema: GoalWorkspaceAuthoritySchemaV0, generation: "generation-ref-partial"},
		{name: "versioned full", schema: GoalWorkspaceAuthoritySchemaV0, workspace: workspaceRef, provider: "provider-ref-codex", generation: "generation-ref-current", valid: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues := ValidateGoalWorkspaceAuthorityV0(goalRef, test.schema, test.workspace, test.provider, test.generation, true)
			if (len(issues) == 0) != test.valid {
				t.Fatalf("valid=%v issues=%+v", test.valid, issues)
			}
		})
	}
}

func TestGoalWorkStateV0AceptaReceiptPre057GenerationOnlySinMigrarloV0(t *testing.T) {
	state := mustGoalLifecycleStateForTestV0(t)
	state.LaunchReceipt.RuntimeGenerationRef = "generation-ref-pre057-persisted"
	normalized, err := NewGoalWorkStateV0(state)
	if err != nil {
		t.Fatal(err)
	}
	request := GoalObservationRequestFromStateV0(normalized)
	if request.WorkspaceAuthoritySchemaVersion != "" || request.WorkspaceRef != "" || request.ProviderRef != "" ||
		request.RuntimeGenerationRef != "generation-ref-pre057-persisted" || len(ValidateGoalObservationRequestV0(request)) != 0 {
		t.Fatalf("request=%+v", request)
	}
}

func TestValidateGoalWorkspaceAuthorityV0NormalizaEspaciosYLimitaRefsV0(t *testing.T) {
	goalRef := "goal-ref-workspace-authority-whitespace-001"
	request := NormalizeGoalObservationRequestV0(GoalObservationRequestV0{
		GoalRef: " " + goalRef + " ", WorkspaceAuthoritySchemaVersion: " " + GoalWorkspaceAuthoritySchemaV0 + " ",
		WorkspaceRef: " " + GoalWorkspaceRefForGoalV0(goalRef) + " ", ProviderRef: " provider-ref-codex ",
		RuntimeGenerationRef: " generation-ref-current ",
	})
	if len(ValidateGoalObservationRequestV0(request)) != 0 {
		t.Fatalf("normalized request rejected: %+v", request)
	}
	request.ProviderRef = strings.Repeat("p", GoalWorkspaceAuthorityRefMaxBytesV0+1)
	issues := ValidateGoalObservationRequestV0(request)
	if !hasGoalIssueFieldV0(issues, "provider_ref") {
		t.Fatalf("oversize issues=%+v", issues)
	}
}

func TestValidateGoalLaunchReceiptV0ValidaIdentidadDeManifestV0(t *testing.T) {
	base := GoalLaunchReceiptV0{
		GoalRef: "goal-ref-receipt-manifest-001", IntentManifestRef: "intent-manifest-ref-request-ref-receipt-manifest-001",
		IntentManifestSHA256: strings.Repeat("a", 64),
	}
	if issues := ValidateGoalLaunchReceiptV0(base); len(issues) != 0 {
		t.Fatalf("valid receipt issues=%+v", issues)
	}
	for _, test := range []struct {
		name   string
		mutate func(*GoalLaunchReceiptV0)
		field  string
	}{
		{name: "missing sha", mutate: func(receipt *GoalLaunchReceiptV0) { receipt.IntentManifestSHA256 = "" }, field: "intent_manifest_sha256"},
		{name: "missing ref", mutate: func(receipt *GoalLaunchReceiptV0) { receipt.IntentManifestRef = "" }, field: "intent_manifest_ref"},
		{name: "unsafe ref", mutate: func(receipt *GoalLaunchReceiptV0) { receipt.IntentManifestRef = "intent-manifest-ref-../outside" }, field: "intent_manifest_ref"},
		{name: "uppercase sha", mutate: func(receipt *GoalLaunchReceiptV0) { receipt.IntentManifestSHA256 = strings.Repeat("A", 64) }, field: "intent_manifest_sha256"},
	} {
		t.Run(test.name, func(t *testing.T) {
			receipt := base
			test.mutate(&receipt)
			if issues := ValidateGoalLaunchReceiptV0(receipt); !hasGoalIssueFieldV0(issues, test.field) {
				t.Fatalf("issues=%+v", issues)
			}
		})
	}
}

func TestNewGoalWorkStateFromLaunchV0LigaManifestConAutoridadVersionadaV0(t *testing.T) {
	spec := validGoalLifecycleSpecForTestV0()
	spec.RequestRef = "request-ref-state-manifest-001"
	spec.IntentManifestRef = "intent-manifest-ref-" + spec.RequestRef
	spec.IntentManifestSHA256 = strings.Repeat("b", 64)
	receipt := GoalLaunchReceiptV0{
		Status: GoalStatusRunningV0, GoalRef: spec.GoalRef,
		IntentManifestRef: spec.IntentManifestRef, IntentManifestSHA256: spec.IntentManifestSHA256,
		WorkspaceAuthoritySchemaVersion: GoalWorkspaceAuthoritySchemaV0,
		WorkspaceRef:                    GoalWorkspaceRefForGoalV0(spec.GoalRef), ProviderRef: "provider-ref-codex",
		RuntimeGenerationRef: "generation-ref-state-manifest-001",
	}
	request := GoalWorkStateFromLaunchRequestV0{RunRef: "run-ref-state-manifest-001", Spec: spec, LaunchReceipt: receipt}
	if _, err := NewGoalWorkStateFromLaunchV0(request); err != nil {
		t.Fatalf("matching manifest rejected: %v", err)
	}
	for _, mutate := range []func(*GoalLaunchReceiptV0){
		func(receipt *GoalLaunchReceiptV0) {
			receipt.IntentManifestRef = "intent-manifest-ref-request-ref-other"
		},
		func(receipt *GoalLaunchReceiptV0) { receipt.IntentManifestSHA256 = strings.Repeat("c", 64) },
		func(receipt *GoalLaunchReceiptV0) { receipt.IntentManifestRef, receipt.IntentManifestSHA256 = "", "" },
	} {
		mismatch := request
		mismatch.LaunchReceipt = receipt
		mutate(&mismatch.LaunchReceipt)
		if _, err := NewGoalWorkStateFromLaunchV0(mismatch); err == nil {
			t.Fatalf("manifest mismatch accepted: %+v", mismatch.LaunchReceipt)
		}
	}

	legacy := request
	legacy.LaunchReceipt = GoalLaunchReceiptV0{
		Status: GoalStatusRunningV0, GoalRef: spec.GoalRef,
		RuntimeGenerationRef: "generation-ref-pre057-with-spec-manifest",
	}
	if _, err := NewGoalWorkStateFromLaunchV0(legacy); err == nil {
		t.Fatal("new launch downgraded manifest authority to a legacy receipt")
	}
	legacyState := GoalWorkStateV0{
		SchemaVersion: GoalWorkStateSchemaV0, RunRef: legacy.RunRef, GoalRef: spec.GoalRef,
		ExternalGoalRef: spec.GoalRef, Status: GoalStatusRunningV0,
		Spec: spec, LaunchReceipt: legacy.LaunchReceipt,
	}
	if state, err := NewGoalWorkStateV0(legacyState); err != nil || state.LaunchReceipt.IntentManifestRef != "" {
		t.Fatalf("pre-057 durable state no longer loads: state=%+v err=%v", state, err)
	}
	unversionedSubstitution := legacy
	unversionedSubstitution.LaunchReceipt.IntentManifestRef = "intent-manifest-ref-request-ref-substituted"
	unversionedSubstitution.LaunchReceipt.IntentManifestSHA256 = strings.Repeat("d", 64)
	if _, err := NewGoalWorkStateFromLaunchV0(unversionedSubstitution); err == nil {
		t.Fatal("unversioned receipt substituted manifest identity")
	}
}

func TestGoalObservationRequestFromStateV0ProyectaAutoridadInmutableV0(t *testing.T) {
	goalRef := "goal-ref-workspace-authority-001"
	state := GoalWorkStateV0{
		GoalRef:         goalRef,
		ExternalGoalRef: "external-goal-ref-workspace-authority-001",
		LaunchReceipt: GoalLaunchReceiptV0{
			WorkspaceAuthoritySchemaVersion: GoalWorkspaceAuthoritySchemaV0,
			WorkspaceRef:                    GoalWorkspaceRefForGoalV0(goalRef),
			ProviderRef:                     "provider-ref-workspace-authority-001",
			RuntimeGenerationRef:            "runtime-generation-ref-workspace-authority-001",
		},
	}
	request := GoalObservationRequestFromStateV0(state)
	if request.WorkspaceRef != state.LaunchReceipt.WorkspaceRef ||
		request.ProviderRef != state.LaunchReceipt.ProviderRef ||
		request.RuntimeGenerationRef != state.LaunchReceipt.RuntimeGenerationRef {
		t.Fatalf("request=%+v", request)
	}
}

func TestGoalWorkspaceAuthorityIssuesV0ClasificaMutacionesV0(t *testing.T) {
	goalRef := "goal-ref-workspace-authority-001"
	expected := GoalObservationRequestV0{
		GoalRef:                         goalRef,
		WorkspaceAuthoritySchemaVersion: GoalWorkspaceAuthoritySchemaV0,
		WorkspaceRef:                    GoalWorkspaceRefForGoalV0(goalRef),
		ProviderRef:                     "provider-ref-workspace-authority-001",
		RuntimeGenerationRef:            "runtime-generation-ref-workspace-authority-001",
	}
	tests := []struct {
		name    string
		binding GoalWorkspaceBindingV0
		field   string
	}{
		{
			name: "workspace",
			binding: GoalWorkspaceBindingV0{
				WorkspaceRef: "workspace-ref-mutated", ProviderRef: expected.ProviderRef,
				RuntimeGenerationRef: expected.RuntimeGenerationRef,
			},
			field: "workspace_ref",
		},
		{
			name: "provider",
			binding: GoalWorkspaceBindingV0{
				WorkspaceRef: expected.WorkspaceRef, ProviderRef: "provider-ref-mutated",
				RuntimeGenerationRef: expected.RuntimeGenerationRef,
			},
			field: "provider_ref",
		},
		{
			name: "generation",
			binding: GoalWorkspaceBindingV0{
				WorkspaceRef: expected.WorkspaceRef, ProviderRef: expected.ProviderRef,
				RuntimeGenerationRef: "runtime-generation-ref-mutated",
			},
			field: "runtime_generation_ref",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues := GoalWorkspaceAuthorityIssuesV0(expected, test.binding)
			if !hasGoalIssueFieldV0(issues, test.field) {
				t.Fatalf("issues=%+v", issues)
			}
		})
	}
}

func TestGoalWorkStateTerminalCompatibleV0RechazaReplayConOtraAutoridadV0(t *testing.T) {
	goalRef := "goal-ref-workspace-authority-001"
	current := GoalWorkStateV0{
		StoreVersion:    2,
		RunRef:          "run-ref-workspace-authority-001",
		GoalRef:         goalRef,
		ExternalGoalRef: "external-goal-ref-workspace-authority-001",
		Status:          GoalStatusCompleteV0,
		Spec:            GoalWorkSpecV0{GoalRef: goalRef},
		LaunchReceipt: GoalLaunchReceiptV0{
			WorkspaceAuthoritySchemaVersion: GoalWorkspaceAuthoritySchemaV0,
			WorkspaceRef:                    GoalWorkspaceRefForGoalV0(goalRef),
			ProviderRef:                     "provider-ref-workspace-authority-001",
			RuntimeGenerationRef:            "runtime-generation-ref-workspace-authority-001",
		},
		LastResult: &GoalWorkResultV0{
			Status: GoalStatusCompleteV0, GoalRef: goalRef,
			ExternalGoalRef: "external-goal-ref-workspace-authority-001",
		},
	}
	desired := current
	desired.StoreVersion = 1
	desired.LaunchReceipt.ProviderRef = "provider-ref-mutated"
	if goalWorkStateTerminalCompatibleV0(current, desired) {
		t.Fatal("replay con provider distinto aceptado")
	}
}
