package credentials

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestOneShotUseRequestValidatesCompleteAuthorityTuple(t *testing.T) {
	valid := validOneShotUseRequest()
	if err := ValidateOneShotUseRequest(valid); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	floating := valid
	floating.Version = 0
	err := ValidateOneShotUseRequest(floating)
	var typed *Error
	if !HasErrorCode(err, ErrorInvalidRequest) || !errors.As(err, &typed) || typed.Field != "version" {
		t.Fatalf("floating current-version selector accepted or unstable error returned: %v", err)
	}

	independentClaim := valid
	independentClaim.ActorRef = "actor:other-execution"
	independentClaim.RequestRef = "request:other-execution"
	if err := ValidateOneShotUseRequest(independentClaim); err != nil {
		t.Fatalf("independent claim for same credential/version rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*OneShotUseRequest)
	}{
		{name: "actor", mutate: func(request *OneShotUseRequest) { request.ActorRef = "" }},
		{name: "request", mutate: func(request *OneShotUseRequest) { request.RequestRef = "other:test" }},
		{name: "credential", mutate: func(request *OneShotUseRequest) { request.CredentialRef = "credential:../escape" }},
		{name: "owner", mutate: func(request *OneShotUseRequest) { request.OwnerRef = "" }},
		{name: "scope", mutate: func(request *OneShotUseRequest) { request.ScopeRef = "" }},
		{name: "purpose", mutate: func(request *OneShotUseRequest) { request.PurposeRef = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := valid
			test.mutate(&request)
			if err := ValidateOneShotUseRequest(request); !HasErrorCode(err, ErrorInvalidRef) {
				t.Fatalf("invalid %s accepted: %v", test.name, err)
			}
		})
	}
}

func TestOneShotUseResultMustMatchCompleteRequest(t *testing.T) {
	request := validOneShotUseRequest()
	result := validOneShotUseResult(request)
	if err := ValidateOneShotUseResult(request, result); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}
	replayed := result
	replayed.Replayed = true
	if err := ValidateOneShotUseResult(request, replayed); err != nil {
		t.Fatalf("exact replay evidence rejected: %v", err)
	}

	tests := []struct {
		name   string
		code   ErrorCode
		mutate func(*OneShotUseResult)
	}{
		{name: "actor", code: ErrorInvalidRequest, mutate: func(result *OneShotUseResult) { result.Receipt.ActorRef = "actor:other" }},
		{name: "request", code: ErrorInvalidRequest, mutate: func(result *OneShotUseResult) { result.Receipt.RequestRef = "request:other" }},
		{name: "credential", code: ErrorInvalidRequest, mutate: func(result *OneShotUseResult) { result.Receipt.CredentialRef = "credential:other" }},
		{name: "owner", code: ErrorInvalidRequest, mutate: func(result *OneShotUseResult) { result.Receipt.OwnerRef = "owner:other" }},
		{name: "scope", code: ErrorInvalidRequest, mutate: func(result *OneShotUseResult) { result.Receipt.ScopeRef = "scope:other" }},
		{name: "purpose", code: ErrorInvalidRequest, mutate: func(result *OneShotUseResult) { result.Receipt.PurposeRef = "purpose:other" }},
		{name: "operation", code: ErrorInvalidRequest, mutate: func(result *OneShotUseResult) { result.Receipt.Operation = "use" }},
		{name: "zero_version", code: ErrorVersionConflict, mutate: func(result *OneShotUseResult) { result.Receipt.Version = 0 }},
		{name: "version", code: ErrorVersionConflict, mutate: func(result *OneShotUseResult) { result.Receipt.Version++ }},
		{name: "time", code: ErrorInvalidRequest, mutate: func(result *OneShotUseResult) { result.Receipt.OccurredAt = time.Time{} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := result
			test.mutate(&candidate)
			if err := ValidateOneShotUseResult(request, candidate); !HasErrorCode(err, test.code) {
				t.Fatalf("mismatched %s accepted: %v", test.name, err)
			}
		})
	}
}

func TestOneShotContractProjectionsCannotCarryCredentialMaterial(t *testing.T) {
	assertMaterialFreeContractType(t, reflect.TypeOf(OneShotUseRequest{}), map[reflect.Type]bool{})
	assertMaterialFreeContractType(t, reflect.TypeOf(OneShotUseResult{}), map[reflect.Type]bool{})

	material := []byte("one-shot-projection-secret")
	result := validOneShotUseResult(validOneShotUseRequest())
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	replayErr := NewError(ErrorAlreadyConsumed, "request_ref")
	projections := [][]byte{
		payload,
		[]byte(fmt.Sprintf("%v %+v %#v", result, result, result)),
		[]byte(fmt.Sprintf("%v %+v %#v", replayErr, replayErr, replayErr)),
	}
	for _, projection := range projections {
		for _, signature := range [][]byte{
			material,
			[]byte(base64.StdEncoding.EncodeToString(material)),
			[]byte(hex.EncodeToString(material)),
		} {
			if bytes.Contains(projection, signature) {
				t.Fatalf("one-shot projection leaked credential signature: %q", projection)
			}
		}
	}
	if !HasErrorCode(replayErr, ErrorAlreadyConsumed) || replayErr.Error() != "credentials.already_consumed: request_ref" {
		t.Fatalf("unstable replay error: %v", replayErr)
	}
}

func assertMaterialFreeContractType(t *testing.T, contract reflect.Type, visited map[reflect.Type]bool) {
	t.Helper()
	if visited[contract] {
		return
	}
	visited[contract] = true
	if contract == reflect.TypeOf(Secret{}) || contract.Kind() == reflect.Slice && contract.Elem().Kind() == reflect.Uint8 {
		t.Fatalf("credential-bearing type exposed by one-shot contract: %v", contract)
	}
	if contract.Kind() != reflect.Struct || contract.PkgPath() != reflect.TypeOf(Receipt{}).PkgPath() {
		return
	}
	for index := 0; index < contract.NumField(); index++ {
		field := contract.Field(index)
		name := strings.ToLower(field.Name)
		if strings.Contains(name, "secret") || strings.Contains(name, "material") || strings.Contains(name, "digest") {
			t.Fatalf("credential-derived field exposed by one-shot contract: %s.%s", contract, field.Name)
		}
		assertMaterialFreeContractType(t, field.Type, visited)
	}
}

func validOneShotUseRequest() OneShotUseRequest {
	return OneShotUseRequest{
		ActorRef: "actor:test", RequestRef: "request:one-shot", CredentialRef: "credential:test",
		OwnerRef: "owner:test", ScopeRef: "scope:test", PurposeRef: "purpose:test", Version: 3,
	}
}

func validOneShotUseResult(request OneShotUseRequest) OneShotUseResult {
	return OneShotUseResult{Receipt: Receipt{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef, ScopeRef: request.ScopeRef,
		PurposeRef: request.PurposeRef, Version: request.Version, RequestRef: request.RequestRef,
		ActorRef: request.ActorRef, Operation: OperationUseOnce, OccurredAt: time.Unix(1_700_000_000, 0).UTC(),
	}}
}
