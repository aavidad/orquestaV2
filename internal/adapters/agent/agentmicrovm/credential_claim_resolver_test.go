package agentmicrovm

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type credentialClaimReaderStub struct {
	mu       sync.Mutex
	versions map[credentials.CredentialRef]credentials.Version
	requests []credentials.DescribeUseAuthorityRequest
	err      error
	mutate   func(*credentials.DescribedUseAuthority)
}

type credentialClaimTypedNilContext struct{ context.Context }

func (reader *credentialClaimReaderStub) DescribeUseAuthority(
	_ context.Context,
	request credentials.DescribeUseAuthorityRequest,
) (credentials.DescribedUseAuthority, error) {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	reader.requests = append(reader.requests, request)
	if reader.err != nil {
		return credentials.DescribedUseAuthority{}, reader.err
	}
	result := credentials.DescribedUseAuthority{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
		ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef,
		Version: reader.versions[request.CredentialRef],
	}
	if reader.mutate != nil {
		reader.mutate(&result)
	}
	return result, nil
}

func TestCredentialClaimResolverSelectsExactPlacementAccounts(t *testing.T) {
	placementA := mustCredentialClaimPlacement(t, "placement:account-a")
	placementB := mustCredentialClaimPlacement(t, "placement:account-b")
	credentialA := credentials.CredentialRef("credential:account-a")
	credentialB := credentials.CredentialRef("credential:account-b")
	reader := &credentialClaimReaderStub{versions: map[credentials.CredentialRef]credentials.Version{
		credentialA: 3, credentialB: 7,
	}}
	resolver := mustCredentialClaimResolver(t, reader, []CredentialClaimBinding{
		{PlacementRef: placementA, CredentialRef: credentialA},
		{PlacementRef: placementB, CredentialRef: credentialB},
	})

	for _, test := range []struct {
		placement  ports.AgentPlacementRef
		credential credentials.CredentialRef
		version    credentials.Version
	}{
		{placementA, credentialA, 3},
		{placementB, credentialB, 7},
	} {
		request := credentialClaimLaunchRequest(t, test.placement)
		resolved, err := resolver.Resolve(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		claim := resolved.OneShotUseRequest()
		wantRequestRef, err := ports.BuildMicroVMHostLaunchOneShotRequestRefV1(
			ports.MicroVMHostLaunchAuthorityKey{RunRef: request.ExecutionRef, ActionFence: request.EffectAuthority.ActionFence},
			request.EffectAuthority.EffectAttemptRef, request.SessionRef,
		)
		if err != nil {
			t.Fatal(err)
		}
		if claim.CredentialRef != test.credential || claim.Version != test.version ||
			claim.ActorRef != request.ActorRef.String() || claim.OwnerRef.String() != request.ActorRef.String() ||
			claim.ScopeRef.String() != request.ProjectRef.String() || claim.PurposeRef != "purpose:microvm-launch" ||
			claim.RequestRef != wantRequestRef || credentials.ValidateOneShotUseRequest(claim) != nil {
			t.Fatalf("claim=%+v", claim)
		}
	}
	reader.mu.Lock()
	defer reader.mu.Unlock()
	if len(reader.requests) != 2 || reader.requests[0].CredentialRef == reader.requests[1].CredentialRef ||
		reader.requests[0].RequestRef == reader.requests[1].RequestRef {
		t.Fatalf("describe requests=%+v", reader.requests)
	}
	for _, request := range reader.requests {
		if !strings.HasPrefix(request.RequestRef, credentialClaimDescribePrefix) ||
			strings.HasPrefix(request.RequestRef, "request:microvm-host-launch-one-shot:") {
			t.Fatalf("describe request not domain-separated: %q", request.RequestRef)
		}
	}
}

func TestNewCredentialClaimResolverRejectsInvalidOrAmbiguousBindings(t *testing.T) {
	placement := mustCredentialClaimPlacement(t, "placement:account-a")
	validReader := &credentialClaimReaderStub{}
	var typedNil *credentialClaimReaderStub
	tests := []struct {
		name     string
		reader   credentials.UseAuthorityReader
		purpose  credentials.PurposeRef
		bindings []CredentialClaimBinding
	}{
		{"nil reader", nil, "purpose:microvm-launch", []CredentialClaimBinding{{placement, "credential:account-a"}}},
		{"typed nil reader", typedNil, "purpose:microvm-launch", []CredentialClaimBinding{{placement, "credential:account-a"}}},
		{"purpose", validReader, " purpose:microvm-launch", []CredentialClaimBinding{{placement, "credential:account-a"}}},
		{"empty bindings", validReader, "purpose:microvm-launch", nil},
		{"placement", validReader, "purpose:microvm-launch", []CredentialClaimBinding{{CredentialRef: "credential:account-a"}}},
		{"credential", validReader, "purpose:microvm-launch", []CredentialClaimBinding{{placement, "credential:../bad"}}},
		{"duplicate", validReader, "purpose:microvm-launch", []CredentialClaimBinding{
			{placement, "credential:account-a"}, {placement, "credential:account-b"},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver, err := NewCredentialClaimResolver(test.reader, test.purpose, test.bindings)
			if resolver != nil || ErrorCode(err) != CodeCredentialClaimResolverInvalid {
				t.Fatalf("resolver=%v err=%v", resolver, err)
			}
		})
	}
}

func TestCredentialClaimResolverRejectsUnknownPlacementAndReaderFailures(t *testing.T) {
	placement := mustCredentialClaimPlacement(t, "placement:account-a")
	reader := &credentialClaimReaderStub{versions: map[credentials.CredentialRef]credentials.Version{"credential:account-a": 1}}
	resolver := mustCredentialClaimResolver(t, reader, []CredentialClaimBinding{{placement, "credential:account-a"}})
	unknown := credentialClaimLaunchRequest(t, mustCredentialClaimPlacement(t, "placement:unknown"))
	if resolved, err := resolver.Resolve(context.Background(), unknown); resolved != (ResolvedCredentialClaim{}) ||
		ErrorCode(err) != CodeCredentialClaimPlacementUnknown {
		t.Fatalf("unknown resolved=%+v err=%v", resolved, err)
	}

	sentinel := errors.New("reader unavailable")
	reader.err = sentinel
	request := credentialClaimLaunchRequest(t, placement)
	if resolved, err := resolver.Resolve(context.Background(), request); resolved != (ResolvedCredentialClaim{}) ||
		ErrorCode(err) != CodeCredentialClaimDescriptionUnavailable || !errors.Is(err, sentinel) ||
		strings.Contains(err.Error(), sentinel.Error()) {
		t.Fatalf("reader error resolved=%+v err=%v", resolved, err)
	}
	reader.err = nil
	reader.mutate = func(result *credentials.DescribedUseAuthority) { result.OwnerRef = "owner:crossed" }
	if resolved, err := resolver.Resolve(context.Background(), request); resolved != (ResolvedCredentialClaim{}) ||
		ErrorCode(err) != CodeCredentialClaimDescriptionInvalid {
		t.Fatalf("crossed reader resolved=%+v err=%v", resolved, err)
	}
}

func TestCredentialClaimResolverRejectsInvalidDurableLaunchAuthorityBeforeDescribe(t *testing.T) {
	placement := mustCredentialClaimPlacement(t, "placement:account-a")
	reader := &credentialClaimReaderStub{versions: map[credentials.CredentialRef]credentials.Version{"credential:account-a": 1}}
	resolver := mustCredentialClaimResolver(t, reader, []CredentialClaimBinding{{placement, "credential:account-a"}})
	tests := []struct {
		name   string
		mutate func(*ports.AgentLaunchRequest)
	}{
		{"authorization receipt", func(request *ports.AgentLaunchRequest) { request.EffectAuthority.AuthorizationReceiptRef = "" }},
		{"effect approval", func(request *ports.AgentLaunchRequest) { request.EffectAuthority.EffectApprovalRef = "" }},
		{"effect attempt", func(request *ports.AgentLaunchRequest) { request.EffectAuthority.EffectAttemptRef = "" }},
		{"action fence", func(request *ports.AgentLaunchRequest) { request.EffectAuthority.ActionFence = 0 }},
		{"started at", func(request *ports.AgentLaunchRequest) { request.EffectAuthority.StartedAt = time.Time{} }},
		{"claim lease", func(request *ports.AgentLaunchRequest) {
			request.EffectAuthority.ClaimLeaseUntil = request.EffectAuthority.StartedAt
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := credentialClaimLaunchRequest(t, placement)
			test.mutate(&request)
			resolved, err := resolver.Resolve(context.Background(), request)
			if resolved != (ResolvedCredentialClaim{}) || ErrorCode(err) != CodeCredentialClaimRequestInvalid {
				t.Fatalf("resolved=%+v err=%v", resolved, err)
			}
		})
	}
	reader.mu.Lock()
	defer reader.mu.Unlock()
	if len(reader.requests) != 0 {
		t.Fatalf("invalid authority reached reader: %+v", reader.requests)
	}
}

func TestCredentialClaimResolverRejectsTypedNilContext(t *testing.T) {
	placement := mustCredentialClaimPlacement(t, "placement:account-a")
	reader := &credentialClaimReaderStub{versions: map[credentials.CredentialRef]credentials.Version{"credential:account-a": 1}}
	resolver := mustCredentialClaimResolver(t, reader, []CredentialClaimBinding{{placement, "credential:account-a"}})
	var typedNil *credentialClaimTypedNilContext
	resolved, err := resolver.Resolve(typedNil, credentialClaimLaunchRequest(t, placement))
	if resolved != (ResolvedCredentialClaim{}) || ErrorCode(err) != CodeCredentialClaimResolverInvalid {
		t.Fatalf("resolved=%+v err=%v", resolved, err)
	}
}

func TestCredentialClaimResolverSealsRunFenceAttemptAndSession(t *testing.T) {
	placement := mustCredentialClaimPlacement(t, "placement:account-a")
	reader := &credentialClaimReaderStub{versions: map[credentials.CredentialRef]credentials.Version{"credential:account-a": 1}}
	resolver := mustCredentialClaimResolver(t, reader, []CredentialClaimBinding{{placement, "credential:account-a"}})
	base := credentialClaimLaunchRequest(t, placement)
	baseResolved, err := resolver.Resolve(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	baseClaim := baseResolved.OneShotUseRequest()
	reader.mu.Lock()
	baseDescribeRef := reader.requests[len(reader.requests)-1].RequestRef
	reader.mu.Unlock()

	otherRun, _ := goal.NewExecutionRef("execution:other")
	otherSession, _ := ports.NewExecutionSessionRef("execution-session:sha256:" + strings.Repeat("e", 64))
	mutations := []func(*ports.AgentLaunchRequest){
		func(request *ports.AgentLaunchRequest) { request.ExecutionRef = otherRun },
		func(request *ports.AgentLaunchRequest) { request.EffectAuthority.ActionFence++ },
		func(request *ports.AgentLaunchRequest) { request.EffectAuthority.EffectAttemptRef = "attempt:other" },
		func(request *ports.AgentLaunchRequest) { request.SessionRef = otherSession },
	}
	for index, mutate := range mutations {
		candidate := base
		mutate(&candidate)
		resolved, err := resolver.Resolve(context.Background(), candidate)
		if err != nil {
			t.Fatalf("mutation %d: %v", index, err)
		}
		claim := resolved.OneShotUseRequest()
		reader.mu.Lock()
		describeRef := reader.requests[len(reader.requests)-1].RequestRef
		reader.mu.Unlock()
		if claim.RequestRef == baseClaim.RequestRef || describeRef == baseDescribeRef {
			t.Fatalf("mutation %d did not change causal refs: claim=%q describe=%q", index, claim.RequestRef, describeRef)
		}
	}
}

func TestCredentialClaimResolverPinsDescribedRotationWithoutChangingClaimIdentity(t *testing.T) {
	placement := mustCredentialClaimPlacement(t, "placement:account-a")
	credentialRef := credentials.CredentialRef("credential:account-a")
	reader := &credentialClaimReaderStub{versions: map[credentials.CredentialRef]credentials.Version{credentialRef: 1}}
	bindings := []CredentialClaimBinding{{placement, credentialRef}}
	resolver := mustCredentialClaimResolver(t, reader, bindings)
	bindings[0].CredentialRef = "credential:mutated-after-construction"
	request := credentialClaimLaunchRequest(t, placement)
	firstResolved, err := resolver.Resolve(context.Background(), request)
	first := firstResolved.OneShotUseRequest()
	if err != nil || first.CredentialRef != credentialRef || first.Version != 1 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	reader.mu.Lock()
	reader.versions[credentialRef] = 2
	reader.mu.Unlock()
	secondResolved, err := resolver.Resolve(context.Background(), request)
	second := secondResolved.OneShotUseRequest()
	if err != nil || second.CredentialRef != credentialRef || second.Version != 2 || second.RequestRef != first.RequestRef {
		t.Fatalf("second=%+v first=%+v err=%v", second, first, err)
	}
}

func TestCredentialClaimResolverSurfaceIsMaterialFree(t *testing.T) {
	for _, contract := range []reflect.Type{
		reflect.TypeOf(CredentialClaimBinding{}), reflect.TypeOf(ResolvedCredentialClaim{}),
		reflect.TypeOf(CredentialClaimResolver{}),
	} {
		assertCredentialClaimMaterialFreeType(t, contract, map[reflect.Type]bool{})
	}
	method, found := reflect.TypeOf((*CredentialClaimResolver)(nil)).MethodByName("Resolve")
	if !found || method.Type.NumOut() != 2 || method.Type.Out(0) != reflect.TypeOf(ResolvedCredentialClaim{}) {
		t.Fatalf("unexpected resolver surface: %v", method.Type)
	}
	resolvedType := reflect.TypeOf(ResolvedCredentialClaim{})
	for index := 0; index < resolvedType.NumField(); index++ {
		if resolvedType.Field(index).PkgPath == "" {
			t.Fatalf("resolved claim field is forgeable outside package: %s", resolvedType.Field(index).Name)
		}
	}
}

func TestResolvedCredentialClaimAccessorReturnsValueCopy(t *testing.T) {
	request := credentialClaimLaunchRequest(t, mustCredentialClaimPlacement(t, "placement:account-a"))
	reader := &credentialClaimReaderStub{versions: map[credentials.CredentialRef]credentials.Version{"credential:account-a": 3}}
	resolver := mustCredentialClaimResolver(t, reader, []CredentialClaimBinding{{request.ReferenciaColocacion, "credential:account-a"}})
	resolved, err := resolver.Resolve(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	copy := resolved.OneShotUseRequest()
	copy.CredentialRef = "credential:mutated"
	if resolved.OneShotUseRequest().CredentialRef != "credential:account-a" {
		t.Fatal("OneShotUseRequest accessor aliases resolved authority")
	}
}

func assertCredentialClaimMaterialFreeType(t *testing.T, contract reflect.Type, seen map[reflect.Type]bool) {
	t.Helper()
	if seen[contract] {
		return
	}
	seen[contract] = true
	if contract == reflect.TypeOf(credentials.Secret{}) ||
		contract.Kind() == reflect.Slice && contract.Elem().Kind() == reflect.Uint8 {
		t.Fatalf("material-bearing type exposed: %v", contract)
	}
	switch contract.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		assertCredentialClaimMaterialFreeType(t, contract.Elem(), seen)
	case reflect.Map:
		assertCredentialClaimMaterialFreeType(t, contract.Key(), seen)
		assertCredentialClaimMaterialFreeType(t, contract.Elem(), seen)
	case reflect.Struct:
		for index := 0; index < contract.NumField(); index++ {
			field := contract.Field(index)
			name := strings.ToLower(field.Name)
			for _, forbidden := range []string{"secret", "material", "token", "payload", "path", "endpoint"} {
				if strings.Contains(name, forbidden) {
					t.Fatalf("material-bearing field exposed: %s.%s", contract, field.Name)
				}
			}
			assertCredentialClaimMaterialFreeType(t, field.Type, seen)
		}
	}
}

func mustCredentialClaimResolver(
	t *testing.T,
	reader credentials.UseAuthorityReader,
	bindings []CredentialClaimBinding,
) *CredentialClaimResolver {
	t.Helper()
	resolver, err := NewCredentialClaimResolver(reader, "purpose:microvm-launch", bindings)
	if err != nil {
		t.Fatal(err)
	}
	return resolver
}

func mustCredentialClaimPlacement(t *testing.T, value string) ports.AgentPlacementRef {
	t.Helper()
	ref, err := ports.NewAgentPlacementRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func credentialClaimLaunchRequest(t *testing.T, placement ports.AgentPlacementRef) ports.AgentLaunchRequest {
	t.Helper()
	request := validLaunchRequest(t)
	request.ReferenciaColocacion = placement
	return request
}
