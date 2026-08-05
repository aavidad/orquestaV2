package credentials

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestDescribeUseAuthorityRequestValidatesCompleteAuthorizationTuple(t *testing.T) {
	valid := validDescribeUseAuthorityRequest()
	if err := ValidateDescribeUseAuthorityRequest(valid); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	tests := map[string]func(*DescribeUseAuthorityRequest){
		"actor":      func(request *DescribeUseAuthorityRequest) { request.ActorRef = "" },
		"request":    func(request *DescribeUseAuthorityRequest) { request.RequestRef = "other:test" },
		"credential": func(request *DescribeUseAuthorityRequest) { request.CredentialRef = "credential:../escape" },
		"owner":      func(request *DescribeUseAuthorityRequest) { request.OwnerRef = "" },
		"scope":      func(request *DescribeUseAuthorityRequest) { request.ScopeRef = "" },
		"purpose":    func(request *DescribeUseAuthorityRequest) { request.PurposeRef = "" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			if err := ValidateDescribeUseAuthorityRequest(candidate); !HasErrorCode(err, ErrorInvalidRef) {
				t.Fatalf("invalid %s accepted: %v", name, err)
			}
		})
	}
}

func TestDescribedUseAuthorityMustMatchRequestAndPinConcreteVersion(t *testing.T) {
	request := validDescribeUseAuthorityRequest()
	valid := describedUseAuthorityFor(request, 3)
	if err := ValidateDescribedUseAuthority(request, valid); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}

	tests := map[string]func(*DescribedUseAuthority){
		"credential": func(result *DescribedUseAuthority) { result.CredentialRef = "credential:other" },
		"owner":      func(result *DescribedUseAuthority) { result.OwnerRef = "owner:other" },
		"scope":      func(result *DescribedUseAuthority) { result.ScopeRef = "scope:other" },
		"purpose":    func(result *DescribedUseAuthority) { result.PurposeRef = "purpose:other" },
		"version":    func(result *DescribedUseAuthority) { result.Version = 0 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			if err := ValidateDescribedUseAuthority(request, candidate); !HasErrorCode(err, ErrorInvalidRequest) {
				t.Fatalf("invalid %s accepted: %v", name, err)
			}
		})
	}
}

func TestDescribeUseAuthorityContractIsMaterialFreeAndReaderIsSeparate(t *testing.T) {
	for _, contract := range []reflect.Type{
		reflect.TypeOf(DescribeUseAuthorityRequest{}),
		reflect.TypeOf(DescribedUseAuthority{}),
	} {
		assertDescribeUseAuthorityMaterialFree(t, contract, map[reflect.Type]bool{})
	}

	reader := reflect.TypeOf((*UseAuthorityReader)(nil)).Elem()
	if reader.NumMethod() != 1 || reader.Method(0).Name != "DescribeUseAuthority" {
		t.Fatalf("unexpected reader surface: %v", reader)
	}
	var _ UseAuthorityReader = (*describeUseAuthorityReaderStub)(nil)

	store := reflect.TypeOf((*Store)(nil)).Elem()
	oneShot := reflect.TypeOf((*OneShotStore)(nil)).Elem()
	if _, found := store.MethodByName("DescribeUseAuthority"); found {
		t.Fatal("legacy Store was widened with DescribeUseAuthority")
	}
	if _, found := oneShot.MethodByName("DescribeUseAuthority"); found {
		t.Fatal("OneShotStore was widened with DescribeUseAuthority")
	}
}

type describeUseAuthorityReaderStub struct{}

func (*describeUseAuthorityReaderStub) DescribeUseAuthority(
	context.Context,
	DescribeUseAuthorityRequest,
) (DescribedUseAuthority, error) {
	return DescribedUseAuthority{}, nil
}

func assertDescribeUseAuthorityMaterialFree(
	t *testing.T,
	contract reflect.Type,
	visited map[reflect.Type]bool,
) {
	t.Helper()
	if visited[contract] {
		return
	}
	visited[contract] = true
	if contract == reflect.TypeOf(Secret{}) || contract.Kind() == reflect.Slice && contract.Elem().Kind() == reflect.Uint8 {
		t.Fatalf("credential-bearing type exposed: %v", contract)
	}
	if contract.Kind() != reflect.Struct || contract.PkgPath() != reflect.TypeOf(Receipt{}).PkgPath() {
		return
	}
	for index := 0; index < contract.NumField(); index++ {
		field := contract.Field(index)
		name := strings.ToLower(field.Name)
		for _, forbidden := range []string{"secret", "material", "digest", "path", "endpoint", "payload"} {
			if strings.Contains(name, forbidden) {
				t.Fatalf("forbidden field exposed: %s.%s", contract, field.Name)
			}
		}
		assertDescribeUseAuthorityMaterialFree(t, field.Type, visited)
	}
}

func validDescribeUseAuthorityRequest() DescribeUseAuthorityRequest {
	return DescribeUseAuthorityRequest{
		ActorRef: "actor:execution", RequestRef: "request:describe-use-authority",
		CredentialRef: "credential:codex", OwnerRef: "owner:codex",
		ScopeRef: "project:one", PurposeRef: "codex",
	}
}

func describedUseAuthorityFor(
	request DescribeUseAuthorityRequest,
	version Version,
) DescribedUseAuthority {
	return DescribedUseAuthority{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
		ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef, Version: version,
	}
}
