package agentmicrovm

import (
	"context"
	"reflect"
	"strings"
	"testing"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestBuildHostLaunchAuthorityV1MapsExactSignedControlAndProxy(t *testing.T) {
	request := withEgressAuthority(t, validLaunchRequest(t), validEgressGrant())
	compiled, signed := compiledSignedHostLaunchRequest(t, request, true)
	resolved := resolvedHostLaunchClaim(t, request)
	claim := resolved.OneShotUseRequest()
	extracted, err := microvm.ExtraerAutoridadServiciosHostLanzamientoV1(signed)
	if err != nil {
		t.Fatal(err)
	}

	authority, err := BuildHostLaunchAuthorityV1(request, compiled, signed, resolved)
	if err != nil {
		t.Fatalf("BuildHostLaunchAuthorityV1() = %v", err)
	}
	if err := ports.ValidateMicroVMHostLaunchAuthorityPreparedV1(authority); err != nil {
		t.Fatalf("prepared authority invalid: %v", err)
	}
	if authority.Key.RunRef != request.ExecutionRef ||
		authority.Key.ActionFence != request.EffectAuthority.ActionFence ||
		authority.EffectAttemptRef != request.EffectAuthority.EffectAttemptRef ||
		authority.SessionRef != request.SessionRef || authority.OneShotClaim != claim ||
		authority.PlanSHA256 != extracted.PlanSHA256 || authority.ConcessionSHA256 != extracted.ConcesionSHA256 ||
		authority.ExternalRef != "" || len(authority.Services) != 2 {
		t.Fatalf("authority=%+v extracted=%+v", authority, extracted)
	}
	wantServices := []ports.MicroVMHostServiceAuthorityV1{
		{Role: ports.MicroVMHostServiceControlBroker, ServiceRef: "servicio:control", Port: 10_001,
			IdentityRef: "identidad-servicio:control", IdentitySHA256: strings.Repeat("4", 64)},
		{Role: ports.MicroVMHostServiceControlledEgressProxy, ServiceRef: "servicio:egreso", Port: 10_002,
			IdentityRef: "identidad-servicio:egreso", IdentitySHA256: strings.Repeat("5", 64)},
	}
	if !reflect.DeepEqual(authority.Services, wantServices) {
		t.Fatalf("services=%+v want=%+v", authority.Services, wantServices)
	}
}

func TestBuildHostLaunchAuthorityV1RejectsSignedRunOrFenceCrossedWithRequest(t *testing.T) {
	base := validLaunchRequest(t)
	compiled, signed := compiledSignedHostLaunchRequest(t, base, false)
	otherRun, _ := goal.NewExecutionRef("execution:microvm-other")
	tests := map[string]func(*ports.AgentLaunchRequest){
		"run":   func(request *ports.AgentLaunchRequest) { request.ExecutionRef = otherRun },
		"fence": func(request *ports.AgentLaunchRequest) { request.EffectAuthority.ActionFence++ },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			request := base
			mutate(&request)
			resolved := resolvedHostLaunchClaim(t, request)
			if authority, err := BuildHostLaunchAuthorityV1(request, compiled, signed, resolved); !reflect.DeepEqual(authority, ports.MicroVMHostLaunchAuthorityV1{}) || ErrorCode(err) != CodeLaunchAuthorityInvalid {
				t.Fatalf("authority=%+v err=%v", authority, err)
			}
		})
	}
}

func TestBuildHostLaunchAuthorityV1RejectsCompiledSignedCausalityCrossedSameRunFence(t *testing.T) {
	base := validLaunchRequest(t)
	compiled, signed := compiledSignedHostLaunchRequest(t, base, false)
	otherSession, err := ports.NewExecutionSessionRef("execution-session:microvm-other")
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*ports.AgentLaunchRequest){
		"effect attempt": func(request *ports.AgentLaunchRequest) {
			request.EffectAuthority.EffectAttemptRef = "effect-attempt:microvm-other"
		},
		"session": func(request *ports.AgentLaunchRequest) { request.SessionRef = otherSession },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			request := base
			mutate(&request)
			resolved := resolvedHostLaunchClaim(t, request)
			if authority, err := BuildHostLaunchAuthorityV1(request, compiled, signed, resolved); !reflect.DeepEqual(authority, ports.MicroVMHostLaunchAuthorityV1{}) || ErrorCode(err) != CodeLaunchAuthorityInvalid {
				t.Fatalf("authority=%+v err=%v", authority, err)
			}
		})
	}
}

func TestBuildHostLaunchAuthorityV1RejectsResolvedClaimFromOtherPlacementCredential(t *testing.T) {
	requestA := validLaunchRequest(t)
	compiledA, signedA := compiledSignedHostLaunchRequest(t, requestA, false)
	placementB, err := ports.NewAgentPlacementRef("placement:microvm-other-account")
	if err != nil {
		t.Fatal(err)
	}
	requestB := requestA
	requestB.ReferenciaColocacion = placementB
	credentialA := credentials.CredentialRef("credential:provider_codex_primary")
	credentialB := credentials.CredentialRef("credential:provider_codex_other")
	reader := &credentialClaimReaderStub{versions: map[credentials.CredentialRef]credentials.Version{
		credentialA: 3,
		credentialB: 8,
	}}
	resolver := mustCredentialClaimResolver(t, reader, []CredentialClaimBinding{
		{PlacementRef: requestA.ReferenciaColocacion, CredentialRef: credentialA},
		{PlacementRef: placementB, CredentialRef: credentialB},
	})
	resolvedB, err := resolver.Resolve(context.Background(), requestB)
	if err != nil {
		t.Fatal(err)
	}
	if resolvedB.OneShotUseRequest().CredentialRef != credentialB {
		t.Fatalf("resolved B=%+v", resolvedB.OneShotUseRequest())
	}

	authority, err := BuildHostLaunchAuthorityV1(requestA, compiledA, signedA, resolvedB)
	if !reflect.DeepEqual(authority, ports.MicroVMHostLaunchAuthorityV1{}) || ErrorCode(err) != CodeLaunchAuthorityInvalid {
		t.Fatalf("authority=%+v err=%v", authority, err)
	}
}

func TestBuildHostLaunchAuthorityV1RejectsInvalidRequestAndEffectAuthority(t *testing.T) {
	base := validLaunchRequest(t)
	compiled, signed := compiledSignedHostLaunchRequest(t, base, false)
	resolved := resolvedHostLaunchClaim(t, base)
	tests := map[string]func(*ports.AgentLaunchRequest){
		"request": func(request *ports.AgentLaunchRequest) { request.GoalRef = goal.GoalRef{} },
		"effect":  func(request *ports.AgentLaunchRequest) { request.EffectAuthority.EffectApprovalRef = "" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			request := base
			mutate(&request)
			if authority, err := BuildHostLaunchAuthorityV1(request, compiled, signed, resolved); !reflect.DeepEqual(authority, ports.MicroVMHostLaunchAuthorityV1{}) || ErrorCode(err) != CodeLaunchAuthorityInvalid {
				t.Fatalf("authority=%+v err=%v", authority, err)
			}
		})
	}
}

func TestBuildHostLaunchAuthorityV1RejectsMalformedSignedRequestWithoutLeakingInput(t *testing.T) {
	request := validLaunchRequest(t)
	compiled, valid := compiledSignedHostLaunchRequest(t, request, false)
	resolved := resolvedHostLaunchClaim(t, request)
	marker := "signed-private-marker"
	tests := map[string]microvm.SolicitudLanzamiento{
		"plan":       {Plan: []byte(marker), Concesion: append([]byte(nil), valid.Concesion...)},
		"concession": {Plan: append([]byte(nil), valid.Plan...), Concesion: []byte(marker)},
		"empty":      {},
	}
	for name, signed := range tests {
		t.Run(name, func(t *testing.T) {
			authority, err := BuildHostLaunchAuthorityV1(request, compiled, signed, resolved)
			if !reflect.DeepEqual(authority, ports.MicroVMHostLaunchAuthorityV1{}) || ErrorCode(err) != CodeLaunchAuthorityInvalid ||
				err.Error() != CodeLaunchAuthorityInvalid || strings.Contains(err.Error(), marker) {
				t.Fatalf("authority=%+v err=%v", authority, err)
			}
		})
	}
}

func TestBuildHostLaunchAuthorityV1RejectsCrossedOrInvalidResolvedClaim(t *testing.T) {
	request := validLaunchRequest(t)
	compiled, signed := compiledSignedHostLaunchRequest(t, request, false)
	otherPlacement, err := ports.NewAgentPlacementRef("placement:microvm-other")
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*ResolvedCredentialClaim){
		"placement crossed": func(resolved *ResolvedCredentialClaim) { resolved.placement = otherPlacement },
		"actor crossed":     func(resolved *ResolvedCredentialClaim) { resolved.claim.ActorRef = "actor:other" },
		"owner crossed":     func(resolved *ResolvedCredentialClaim) { resolved.claim.OwnerRef = "actor:other" },
		"scope crossed":     func(resolved *ResolvedCredentialClaim) { resolved.claim.ScopeRef = "project:other" },
		"request crossed":   func(resolved *ResolvedCredentialClaim) { resolved.claim.RequestRef = "request:other" },
		"purpose invalid":   func(resolved *ResolvedCredentialClaim) { resolved.claim.PurposeRef = "" },
		"version invalid":   func(resolved *ResolvedCredentialClaim) { resolved.claim.Version = 0 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			resolved := resolvedHostLaunchClaim(t, request)
			mutate(&resolved)
			if authority, err := BuildHostLaunchAuthorityV1(request, compiled, signed, resolved); !reflect.DeepEqual(authority, ports.MicroVMHostLaunchAuthorityV1{}) || ErrorCode(err) != CodeLaunchAuthorityInvalid {
				t.Fatalf("authority=%+v err=%v", authority, err)
			}
		})
	}
}

func TestBuildHostLaunchAuthorityV1ReturnsDetachedMaterialFreeAuthority(t *testing.T) {
	request := validLaunchRequest(t)
	compiled, signed := compiledSignedHostLaunchRequest(t, request, true)
	resolved := resolvedHostLaunchClaim(t, request)
	first, err := BuildHostLaunchAuthorityV1(request, compiled, signed, resolved)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildHostLaunchAuthorityV1(request, compiled, signed, resolved)
	if err != nil {
		t.Fatal(err)
	}
	first.Services[0].ServiceRef = "servicio:mutated"
	clear(signed.Plan)
	clear(signed.Concesion)
	resolved.claim.ActorRef = "actor:mutated"
	if second.Services[0].ServiceRef != "servicio:control" || second.OneShotClaim.ActorRef != request.ActorRef.String() ||
		second.ExternalRef != "" {
		t.Fatalf("authority aliases caller input: %+v", second)
	}
	assertCredentialClaimMaterialFreeType(t, reflect.TypeOf(second), map[reflect.Type]bool{})
}

func compiledSignedHostLaunchRequest(
	t *testing.T,
	request ports.AgentLaunchRequest,
	withEgress bool,
) (Compilation, microvm.SolicitudLanzamiento) {
	t.Helper()
	compiled := mustCompile(t, request, validDescriptor(t, withEgress))
	signed, err := validSigner().Preparar(
		context.Background(), request, compiled.Context, compiled.Plan, compiled.IssuedAt, compiled.Validity,
	)
	if err != nil {
		t.Fatal(err)
	}
	return compiled, signed
}

func resolvedHostLaunchClaim(t *testing.T, request ports.AgentLaunchRequest) ResolvedCredentialClaim {
	t.Helper()
	credentialRef := credentials.CredentialRef("credential:provider_codex_primary")
	reader := &credentialClaimReaderStub{versions: map[credentials.CredentialRef]credentials.Version{credentialRef: 3}}
	resolver := mustCredentialClaimResolver(t, reader, []CredentialClaimBinding{{
		PlacementRef: request.ReferenciaColocacion, CredentialRef: credentialRef,
	}})
	resolved, err := resolver.Resolve(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}
