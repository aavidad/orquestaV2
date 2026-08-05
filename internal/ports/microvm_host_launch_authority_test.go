package ports

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/goal"
)

func TestMicroVMHostLaunchAuthorityAcceptsPreparedAndBoundStates(t *testing.T) {
	prepared := validMicroVMHostLaunchAuthority(t)
	if err := ValidateMicroVMHostLaunchAuthorityPreparedV1(prepared); err != nil {
		t.Fatalf("valid prepared authority rejected: %v", err)
	}
	if code := MicroVMHostLaunchAuthorityContractErrorCode(
		ValidateMicroVMHostLaunchAuthorityBoundV1(prepared),
	); code != "microvm_host_launch_authority.bound_external_ref_required" {
		t.Fatalf("prepared authority accepted as bound: %q", code)
	}

	bound := prepared
	bound.ExternalRef = "ejecucion:physical_1"
	if err := ValidateMicroVMHostLaunchAuthorityBoundV1(bound); err != nil {
		t.Fatalf("valid bound authority rejected: %v", err)
	}
	if code := MicroVMHostLaunchAuthorityContractErrorCode(
		ValidateMicroVMHostLaunchAuthorityPreparedV1(bound),
	); code != "microvm_host_launch_authority.prepared_external_ref_present" {
		t.Fatalf("bound authority accepted as prepared: %q", code)
	}
}

func TestMicroVMHostLaunchAuthorityRejectsInvalidCausalityAndDigests(t *testing.T) {
	base := validMicroVMHostLaunchAuthority(t)
	tests := map[string]struct {
		mutate func(*MicroVMHostLaunchAuthorityV1)
		code   string
	}{
		"run":               {func(v *MicroVMHostLaunchAuthorityV1) { v.Key.RunRef = goal.ExecutionRef{} }, "run_ref_invalid"},
		"fence":             {func(v *MicroVMHostLaunchAuthorityV1) { v.Key.ActionFence = 0 }, "action_fence_invalid"},
		"attempt":           {func(v *MicroVMHostLaunchAuthorityV1) { v.EffectAttemptRef = " attempt:one" }, "effect_attempt_ref_invalid"},
		"session":           {func(v *MicroVMHostLaunchAuthorityV1) { v.SessionRef = "execution-session:bad/path" }, "session_ref_invalid"},
		"plan digest":       {func(v *MicroVMHostLaunchAuthorityV1) { v.PlanSHA256 = strings.Repeat("A", 64) }, "plan_digest_invalid"},
		"concession digest": {func(v *MicroVMHostLaunchAuthorityV1) { v.ConcessionSHA256 = strings.Repeat("b", 63) }, "concession_digest_invalid"},
		"physical ref":      {func(v *MicroVMHostLaunchAuthorityV1) { v.ExternalRef = "ejecucion:with/path" }, "external_ref_invalid"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := CloneMicroVMHostLaunchAuthorityV1(base)
			test.mutate(&candidate)
			want := "microvm_host_launch_authority." + test.code
			if got := MicroVMHostLaunchAuthorityContractErrorCode(ValidateMicroVMHostLaunchAuthorityV1(candidate)); got != want {
				t.Fatalf("error code=%q want=%q", got, want)
			}
		})
	}
}

func TestMicroVMHostLaunchAuthorityRejectsOversizeOrMultilineRunRef(t *testing.T) {
	base := validMicroVMHostLaunchAuthority(t)
	for name, value := range map[string]string{
		"oversize": strings.Repeat("r", microVMHostLaunchMaxRunRefBytes+1),
		"newline":  "execution:host\nlaunch",
		"carriage": "execution:host\rlaunch",
	} {
		t.Run(name, func(t *testing.T) {
			runRef, err := goal.NewExecutionRef(value)
			if err != nil {
				t.Fatalf("goal constructor unexpectedly closed the port-specific case: %v", err)
			}
			key := base.Key
			key.RunRef = runRef
			if got := MicroVMHostLaunchAuthorityContractErrorCode(ValidateMicroVMHostLaunchAuthorityKey(key)); got != "microvm_host_launch_authority.run_ref_invalid" {
				t.Fatalf("error code=%q", got)
			}
		})
	}
}

func TestMicroVMHostLaunchAuthorityRequiresCanonicalServices(t *testing.T) {
	base := validMicroVMHostLaunchAuthority(t)
	tests := map[string]func(*MicroVMHostLaunchAuthorityV1){
		"none":              func(v *MicroVMHostLaunchAuthorityV1) { v.Services = nil },
		"proxy only":        func(v *MicroVMHostLaunchAuthorityV1) { v.Services = v.Services[1:] },
		"reverse order":     func(v *MicroVMHostLaunchAuthorityV1) { v.Services[0], v.Services[1] = v.Services[1], v.Services[0] },
		"second broker":     func(v *MicroVMHostLaunchAuthorityV1) { v.Services[1].Role = MicroVMHostServiceControlBroker },
		"third":             func(v *MicroVMHostLaunchAuthorityV1) { v.Services = append(v.Services, v.Services[1]) },
		"zero port":         func(v *MicroVMHostLaunchAuthorityV1) { v.Services[0].Port = 0 },
		"duplicate port":    func(v *MicroVMHostLaunchAuthorityV1) { v.Services[1].Port = v.Services[0].Port },
		"duplicate service": func(v *MicroVMHostLaunchAuthorityV1) { v.Services[1].ServiceRef = v.Services[0].ServiceRef },
		"service path":      func(v *MicroVMHostLaunchAuthorityV1) { v.Services[0].ServiceRef = "servicio:control/path" },
		"identity endpoint": func(v *MicroVMHostLaunchAuthorityV1) { v.Services[0].IdentityRef = "https://host/control" },
		"identity digest":   func(v *MicroVMHostLaunchAuthorityV1) { v.Services[0].IdentitySHA256 = strings.Repeat("g", 64) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := CloneMicroVMHostLaunchAuthorityV1(base)
			mutate(&candidate)
			if MicroVMHostLaunchAuthorityContractErrorCode(ValidateMicroVMHostLaunchAuthorityV1(candidate)) == "" {
				t.Fatalf("invalid services accepted: %+v", candidate.Services)
			}
		})
	}
}

func TestCloneMicroVMHostLaunchAuthorityDetachesServices(t *testing.T) {
	original := validMicroVMHostLaunchAuthority(t)
	clone := CloneMicroVMHostLaunchAuthorityV1(original)
	clone.Services[0].ServiceRef = "servicio:changed"
	if original.Services[0].ServiceRef == clone.Services[0].ServiceRef {
		t.Fatal("clone aliases caller-owned Services")
	}
}

func TestMicroVMHostLaunchAuthorityContractExposesNoSecretOrEndpointSurface(t *testing.T) {
	for _, contract := range []reflect.Type{
		reflect.TypeOf(MicroVMHostLaunchAuthorityKey{}),
		reflect.TypeOf(MicroVMHostServiceAuthorityV1{}),
		reflect.TypeOf(MicroVMHostLaunchAuthorityV1{}),
	} {
		for index := 0; index < contract.NumField(); index++ {
			name := strings.ToLower(contract.Field(index).Name)
			for _, forbidden := range []string{"secret", "credential", "token", "material", "path", "endpoint", "command", "payload"} {
				if strings.Contains(name, forbidden) {
					t.Fatalf("credential-bearing field exposed: %s.%s", contract, contract.Field(index).Name)
				}
			}
		}
	}
	var _ MicroVMHostLaunchAuthorityRegistry = (*microVMHostLaunchRegistryContractStub)(nil)
}

type microVMHostLaunchRegistryContractStub struct{}

func (*microVMHostLaunchRegistryContractStub) Prepare(_ context.Context, authority MicroVMHostLaunchAuthorityV1) (MicroVMHostLaunchAuthorityV1, error) {
	return CloneMicroVMHostLaunchAuthorityV1(authority), nil
}

func (*microVMHostLaunchRegistryContractStub) BindExternal(_ context.Context, _ MicroVMHostLaunchAuthorityKey, _ string) (MicroVMHostLaunchAuthorityV1, error) {
	return MicroVMHostLaunchAuthorityV1{}, nil
}

func (*microVMHostLaunchRegistryContractStub) Resolve(_ context.Context, _ MicroVMHostLaunchAuthorityKey) (MicroVMHostLaunchAuthorityV1, error) {
	return MicroVMHostLaunchAuthorityV1{}, nil
}

func validMicroVMHostLaunchAuthority(t *testing.T) MicroVMHostLaunchAuthorityV1 {
	t.Helper()
	runRef, err := goal.NewExecutionRef("execution:host-launch")
	if err != nil {
		t.Fatal(err)
	}
	sessionRef, err := NewExecutionSessionRef("execution-session:sha256:" + strings.Repeat("d", 64))
	if err != nil {
		t.Fatal(err)
	}
	return MicroVMHostLaunchAuthorityV1{
		Key:              MicroVMHostLaunchAuthorityKey{RunRef: runRef, ActionFence: 7},
		EffectAttemptRef: "effect-attempt:launch:one", SessionRef: sessionRef,
		PlanSHA256: strings.Repeat("a", 64), ConcessionSHA256: strings.Repeat("b", 64),
		Services: []MicroVMHostServiceAuthorityV1{
			{Role: MicroVMHostServiceControlBroker, ServiceRef: "servicio:control", Port: 5001,
				IdentityRef: "identidad-servicio:control", IdentitySHA256: strings.Repeat("c", 64)},
			{Role: MicroVMHostServiceControlledEgressProxy, ServiceRef: "servicio:egress", Port: 5002,
				IdentityRef: "identidad-servicio:egress", IdentitySHA256: strings.Repeat("e", 64)},
		},
	}
}
