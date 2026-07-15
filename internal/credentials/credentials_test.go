package credentials

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestSecretCopiesAndRedactsEveryPublicProjection(t *testing.T) {
	input := []byte("live-secret-material")
	secret, err := NewSecret(input)
	if err != nil {
		t.Fatal(err)
	}
	input[0] = 'X'
	first := secret.Bytes()
	first[0] = 'Y'
	if got := secret.Bytes(); string(got) != "live-secret-material" {
		t.Fatalf("secret storage was aliased: %q", got)
	}
	payload, err := json.Marshal(secret)
	if err != nil {
		t.Fatal(err)
	}
	for _, projection := range [][]byte{payload, []byte(fmt.Sprintf("%v %#v", secret, secret))} {
		if bytes.Contains(projection, []byte("live-secret-material")) || !bytes.Contains(projection, []byte("[REDACTED]")) {
			t.Fatalf("unsafe projection: %s", projection)
		}
	}
	alias := secret
	secret.Destroy()
	secret.Destroy()
	if bytes.Contains(alias.Bytes(), []byte("live-secret-material")) || !bytes.Equal(alias.Bytes(), make([]byte, len(input))) {
		t.Fatal("destroy did not clear a shallow secret copy")
	}
}

func TestCredentialErrorProjectionsNeverRenderCause(t *testing.T) {
	material := []byte("cause-contained-secret")
	err := WrapError(ErrorStoreIO, "store", errors.New(string(material)))
	payload, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	for _, projection := range [][]byte{payload, []byte(fmt.Sprintf("%v %+v %#v", err, err, err))} {
		if bytes.Contains(projection, material) {
			t.Fatalf("credential error projection leaked its cause: %s", projection)
		}
	}
	if !errors.Is(err, errors.Unwrap(err)) || errors.Unwrap(err) == nil {
		t.Fatal("redaction removed error-chain diagnostics")
	}
}

func TestValidationRejectsTraversalAndAcceptsCurrentVersion(t *testing.T) {
	request := validUseRequest()
	request.Version = 0
	if err := ValidateUseRequest(request); err != nil {
		t.Fatalf("current-version request rejected: %v", err)
	}
	request.CredentialRef = "credential:../escape"
	if err := ValidateUseRequest(request); !HasErrorCode(err, ErrorInvalidRef) {
		t.Fatalf("traversal ref accepted: %v", err)
	}
}

func TestTypedOpaqueRefsDoNotInventPrefixMappings(t *testing.T) {
	for name, err := range map[string]error{
		"owner": ValidateOwnerRef("actor:alice"), "scope": ValidateScopeRef("project:orquesta"),
		"purpose": ValidatePurposeRef("provider:codex"),
	} {
		if err != nil {
			t.Fatalf("%s opaque ref rejected: %v", name, err)
		}
	}
	if err := ValidateCredentialRef("provider:codex"); !HasErrorCode(err, ErrorInvalidRef) {
		t.Fatalf("credential ref without credential namespace accepted: %v", err)
	}
}

func TestLeakGuardDetectsAndRedactsWithoutLeakingError(t *testing.T) {
	secret, _ := NewSecret([]byte("needle-secret"))
	guard, err := NewLeakGuard(secret)
	if err != nil {
		t.Fatal(err)
	}
	contaminated := []byte("before needle-secret after")
	err = guard.Scan([]LeakSurface{{Name: "result", Content: contaminated}})
	if !HasErrorCode(err, ErrorSecretLeak) || bytes.Contains([]byte(err.Error()), secret.Bytes()) {
		t.Fatalf("unsafe leak error: %v", err)
	}
	if got := guard.Redact(contaminated); string(got) != "before [REDACTED] after" {
		t.Fatalf("redacted = %q", got)
	}
	guard.Destroy()
	guard.Destroy()
	if err := guard.Scan(nil); !HasErrorCode(err, ErrorInvalidRequest) {
		t.Fatalf("destroyed guard remained usable: %v", err)
	}
}

func TestLeakGuardDetectsCommonEncodedMaterial(t *testing.T) {
	material := []byte("encoded-secret-material")
	secret, _ := NewSecret(material)
	guard, err := NewLeakGuard(secret)
	secret.Destroy()
	if err != nil {
		t.Fatal(err)
	}
	for name, encoded := range map[string][]byte{
		"base64":        []byte(base64.StdEncoding.EncodeToString(material)),
		"base64url_raw": []byte(base64.RawURLEncoding.EncodeToString(material)),
		"hex":           []byte(hex.EncodeToString(material)),
	} {
		if err := guard.Scan([]LeakSurface{{Name: name, Content: encoded}}); !HasErrorCode(err, ErrorSecretLeak) {
			t.Errorf("%s encoding escaped leak gate: %v", name, err)
		}
		if redacted := guard.Redact(encoded); bytes.Contains(redacted, encoded) || !bytes.Contains(redacted, []byte("[REDACTED]")) {
			t.Errorf("%s encoding was not redacted: %q", name, redacted)
		}
	}
	guard.Destroy()
}

func TestLeakGuardDetectsEveryShortReversibleEncoding(t *testing.T) {
	seed := []byte{0xfb, 0xef, 0xff, 0xfa, 0xab, 0xcd, 0xde}
	for length := 1; length <= len(seed); length++ {
		t.Run(fmt.Sprintf("bytes_%d", length), func(t *testing.T) {
			material := append([]byte(nil), seed[:length]...)
			secret, err := NewSecret(material)
			if err != nil {
				t.Fatal(err)
			}
			guard, err := NewLeakGuard(secret)
			secret.Destroy()
			if err != nil {
				t.Fatal(err)
			}
			defer guard.Destroy()
			guardJSON, err := json.Marshal(guard)
			if err != nil {
				t.Fatal(err)
			}
			guardText := []byte(fmt.Sprintf("%v %+v %#v", guard, guard, guard))

			lowerHex := []byte(hex.EncodeToString(material))
			representations := map[string][]byte{
				"raw":            material,
				"base64_std":     []byte(base64.StdEncoding.EncodeToString(material)),
				"base64_raw":     []byte(base64.RawStdEncoding.EncodeToString(material)),
				"base64_url":     []byte(base64.URLEncoding.EncodeToString(material)),
				"base64_url_raw": []byte(base64.RawURLEncoding.EncodeToString(material)),
				"hex_lower":      lowerHex,
				"hex_upper":      bytes.ToUpper(lowerHex),
			}
			for name, representation := range representations {
				if len(representation) == 0 {
					t.Fatalf("%s produced an empty signature", name)
				}
				scanErr := guard.Scan([]LeakSurface{{Name: name, Content: representation}})
				if !HasErrorCode(scanErr, ErrorSecretLeak) {
					t.Errorf("%s encoding of %d-byte material escaped leak gate: %v", name, length, scanErr)
					continue
				}
				redacted := guard.Redact(representation)
				if bytes.Contains(redacted, representation) || !bytes.Contains(redacted, []byte("[REDACTED]")) {
					t.Errorf("%s encoding of %d-byte material was not redacted: %q", name, length, redacted)
				}
				for _, projection := range [][]byte{
					guardJSON,
					guardText,
					[]byte(scanErr.Error()),
					[]byte(fmt.Sprintf("%v %+v %#v", scanErr, scanErr, scanErr)),
				} {
					if bytes.Contains(projection, material) || bytes.Contains(projection, representation) {
						t.Errorf("%s leak error exposed %d-byte material: %q", name, length, projection)
					}
				}
			}
			if err := guard.Scan([]LeakSurface{{Name: "empty", Content: nil}}); err != nil {
				t.Fatalf("empty surface matched an empty signature: %v", err)
			}
		})
	}
}

func TestLeakGuardRedactionMarkerNeverReintroducesShortMaterial(t *testing.T) {
	tests := []struct {
		name     string
		material string
	}{
		{name: "one_A", material: "A"},
		{name: "one_R", material: "R"},
		{name: "one_open_bracket", material: "["},
		{name: "one_close_bracket", material: "]"},
		{name: "two", material: "ED"},
		{name: "three", material: "RED"},
		{name: "four", material: "DACT"},
		{name: "five", material: "DACTE"},
		{name: "six", material: "REDACT"},
		{name: "seven", material: "EDACTED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			secret, err := NewSecret([]byte(test.material))
			if err != nil {
				t.Fatal(err)
			}
			guard, err := NewLeakGuard(secret)
			secret.Destroy()
			if err != nil {
				t.Fatal(err)
			}
			defer guard.Destroy()
			redacted := guard.Redact([]byte("before " + test.material + " after"))
			if err := guard.Scan([]LeakSurface{{Name: "redacted", Content: redacted}}); err != nil {
				t.Fatalf("redaction retained %d-byte signature: %q: %v", len(test.material), redacted, err)
			}
		})
	}

	secret, _ := NewSecret([]byte{0xfb})
	guard, err := NewLeakGuard(secret)
	secret.Destroy()
	if err != nil {
		t.Fatal(err)
	}
	defer guard.Destroy()
	redacted := guard.Redact([]byte{0xfb})
	if !bytes.Equal(redacted, []byte("[REDACTED]")) || guard.Scan([]LeakSurface{{Name: "safe_marker", Content: redacted}}) != nil {
		t.Fatalf("safe marker was not preserved: %q", redacted)
	}
}

func TestLeakGuardProjectionsNeverRenderMaterial(t *testing.T) {
	material := []byte("guard-projection-secret")
	secret, _ := NewSecret(material)
	guard, err := NewLeakGuard(secret)
	secret.Destroy()
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(guard)
	if err != nil {
		t.Fatal(err)
	}
	for _, projection := range [][]byte{payload, []byte(fmt.Sprintf("%v %+v %#v", guard, guard, guard))} {
		if bytes.Contains(projection, material) || bytes.Contains(projection, []byte("material:")) ||
			bytes.Contains(projection, []byte("0x67")) || !bytes.Contains(projection, []byte("[REDACTED]")) {
			t.Fatalf("unsafe LeakGuard projection: %s", projection)
		}
	}
	guard.Destroy()
}

func TestWithChildEnvironmentRejectsEncodedCredentialOutput(t *testing.T) {
	secret, _ := NewSecret([]byte("child-encoded-secret"))
	store := &callbackStore{secret: secret}
	output, receipts, err := WithChildEnvironment(context.Background(), store, validChildRequest(), func([]string) (ChildOutput, error) {
		return ChildOutput{Artifact: []byte(base64.StdEncoding.EncodeToString([]byte("child-encoded-secret")))}, nil
	})
	if !HasErrorCode(err, ErrorSecretLeak) || len(receipts) != 1 || len(output.Artifact) != 0 {
		t.Fatalf("encoded child leak output=%+v receipts=%+v err=%v", output, receipts, err)
	}
}

func TestWithChildEnvironmentIsExactAndLeakGated(t *testing.T) {
	secret, _ := NewSecret([]byte("child-secret"))
	store := &callbackStore{secret: secret}
	request := validChildRequest()

	output, receipts, err := WithChildEnvironment(context.Background(), store, request, func(environment []string) (ChildOutput, error) {
		if store.active != 1 {
			return ChildOutput{}, fmt.Errorf("credential callback is not active")
		}
		want := []string{"PATH=/usr/bin", "TOKEN=child-secret"}
		if !reflect.DeepEqual(environment, want) {
			return ChildOutput{}, fmt.Errorf("environment mismatch")
		}
		return ChildOutput{Result: []byte("ok")}, nil
	})
	if err != nil || string(output.Result) != "ok" || len(receipts) != 1 || store.last.Version != 0 {
		t.Fatalf("delivery output=%+v receipts=%+v err=%v use=%+v", output, receipts, err, store.last)
	}

	output, receipts, err = WithChildEnvironment(context.Background(), store, request, func([]string) (ChildOutput, error) {
		return ChildOutput{Artifact: []byte("child-secret")}, nil
	})
	if !HasErrorCode(err, ErrorSecretLeak) || len(output.Result)+len(output.Diagnostic)+len(output.Artifact) != 0 || len(receipts) != 1 {
		t.Fatalf("leak escaped output=%+v receipts=%+v err=%v", output, receipts, err)
	}
}

func TestWithChildEnvironmentDropsCallbackCausesAndClearsFailedOutput(t *testing.T) {
	material := []byte("child-hidden-cause-secret")
	secret, _ := NewSecret(material)
	defer secret.Destroy()
	store := &callbackStore{secret: secret}
	request := validChildRequest()

	var hiddenCause error
	output, receipts, err := WithChildEnvironment(context.Background(), store, request, func([]string) (ChildOutput, error) {
		hiddenCause = errors.New(string(material))
		return ChildOutput{}, hiddenCallbackCause{cause: hiddenCause}
	})
	if !HasErrorCode(err, ErrorChildEnvironment) || errors.Is(err, hiddenCause) || errors.Unwrap(err) != nil ||
		bytes.Contains([]byte(err.Error()), material) || len(receipts) != 1 || !emptyChildOutput(output) {
		t.Fatalf("hidden callback cause escaped: output=%+v receipts=%+v err=%v", output, receipts, err)
	}

	output, receipts, err = WithChildEnvironment(context.Background(), store, request, func([]string) (ChildOutput, error) {
		hiddenCause = errors.New(string(material))
		return ChildOutput{}, WrapError(ErrorConsumerFailed, "delivery", hiddenCause)
	})
	if !HasErrorCode(err, ErrorConsumerFailed) || errors.Is(err, hiddenCause) || errors.Unwrap(err) != nil ||
		bytes.Contains([]byte(err.Error()), material) || len(receipts) != 1 || !emptyChildOutput(output) {
		t.Fatalf("typed callback cause escaped: output=%+v receipts=%+v err=%v", output, receipts, err)
	}

	for name, sentinel := range map[string]error{"canceled": context.Canceled, "deadline": context.DeadlineExceeded} {
		t.Run(name, func(t *testing.T) {
			output, receipts, err := WithChildEnvironment(context.Background(), store, request, func([]string) (ChildOutput, error) {
				return ChildOutput{}, sentinel
			})
			if err != sentinel || errors.Unwrap(err) != nil || len(receipts) != 1 || !emptyChildOutput(output) {
				t.Fatalf("context sentinel changed: output=%+v receipts=%+v err=%v", output, receipts, err)
			}
		})
	}

	result := []byte("failed-result")
	diagnostic := []byte("failed-diagnostic")
	artifact := append([]byte(nil), material...)
	output, receipts, err = WithChildEnvironment(context.Background(), store, request, func([]string) (ChildOutput, error) {
		return ChildOutput{Result: result, Diagnostic: diagnostic, Artifact: artifact}, errors.New("safe callback failure")
	})
	if !HasErrorCode(err, ErrorSecretLeak) || len(receipts) != 1 || !emptyChildOutput(output) ||
		!allZero(result) || !allZero(diagnostic) || !allZero(artifact) {
		t.Fatalf("failed callback buffers retained: output=%+v receipts=%+v result=%q diagnostic=%q artifact=%q err=%v",
			output, receipts, result, diagnostic, artifact, err)
	}
}

type hiddenCallbackCause struct{ cause error }

func (err hiddenCallbackCause) Error() string { return "safe callback failure" }
func (err hiddenCallbackCause) Unwrap() error { return err.cause }

func emptyChildOutput(output ChildOutput) bool {
	return len(output.Result) == 0 && len(output.Diagnostic) == 0 && len(output.Artifact) == 0
}

func allZero(content []byte) bool { return bytes.Equal(content, make([]byte, len(content))) }

func TestWithChildEnvironmentRejectsCollisionBeforeUse(t *testing.T) {
	secret, _ := NewSecret([]byte("child-secret"))
	store := &callbackStore{secret: secret}
	request := validChildRequest()
	request.PublicEnvironment["TOKEN"] = "spoof"
	called := false
	_, _, err := WithChildEnvironment(context.Background(), store, request, func([]string) (ChildOutput, error) {
		called = true
		return ChildOutput{}, nil
	})
	if !HasErrorCode(err, ErrorChildEnvironment) || called || store.calls != 0 {
		t.Fatalf("collision err=%v callback=%v use_calls=%d", err, called, store.calls)
	}
}

func TestWithChildEnvironmentKeepsEveryCredentialCallbackActive(t *testing.T) {
	secret, _ := NewSecret([]byte("child-secret"))
	store := &callbackStore{secret: secret}
	request := validChildRequest()
	request.Bindings = append(request.Bindings, ChildBinding{
		EnvironmentName: "TOKEN_TWO", CredentialRef: "credential:second", Version: 2,
	})
	_, receipts, err := WithChildEnvironment(context.Background(), store, request, func([]string) (ChildOutput, error) {
		if store.active != len(request.Bindings) {
			return ChildOutput{}, fmt.Errorf("active callbacks = %d", store.active)
		}
		return ChildOutput{}, nil
	})
	if err != nil || len(receipts) != 2 {
		t.Fatalf("nested delivery receipts=%+v err=%v", receipts, err)
	}
}

type callbackStore struct {
	secret Secret
	last   UseRequest
	calls  int
	active int
	err    error
}

func (*callbackStore) Create(context.Context, CreateRequest) (MutationResult, error) {
	return MutationResult{}, errors.New("unused")
}

func (store *callbackStore) Use(_ context.Context, request UseRequest, callback func(Secret) error) (Receipt, error) {
	store.calls++
	store.last = request
	if store.err != nil {
		return Receipt{}, store.err
	}
	store.active++
	err := callback(store.secret)
	store.active--
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef, ScopeRef: request.ScopeRef,
		PurposeRef: request.PurposeRef, Version: 3, RequestRef: request.RequestRef, ActorRef: request.ActorRef}, nil
}

func (*callbackStore) Rotate(context.Context, RotateRequest) (MutationResult, error) {
	return MutationResult{}, errors.New("unused")
}

func (*callbackStore) Revoke(context.Context, RevokeRequest) (MutationResult, error) {
	return MutationResult{}, errors.New("unused")
}

func validUseRequest() UseRequest {
	return UseRequest{ActorRef: "actor:test", RequestRef: "request:test", CredentialRef: "credential:test",
		OwnerRef: "owner:test", ScopeRef: "scope:test", PurposeRef: "purpose:test", Version: 1}
}

func validChildRequest() ChildEnvironmentRequest {
	return ChildEnvironmentRequest{
		ActorRef: "actor:test", RequestRef: "request:test", OwnerRef: "owner:test",
		ScopeRef: "scope:test", PurposeRef: "purpose:test", PublicEnvironment: map[string]string{"PATH": "/usr/bin"},
		Bindings: []ChildBinding{{EnvironmentName: "TOKEN", CredentialRef: "credential:test", Version: 0}},
	}
}
