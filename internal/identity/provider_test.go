package identity

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestCredentialIsDetachedRedactedAndNotSerializable(t *testing.T) {
	material := []byte("V11_SECRET_MATERIAL")
	credential, err := NewCredential(material)
	if err != nil {
		t.Fatalf("NewCredential: %v", err)
	}
	clear(material)
	if err := credential.Use(func(detached []byte) error {
		if string(detached) != "V11_SECRET_MATERIAL" {
			t.Fatalf("detached material = %q", detached)
		}
		detached[0] = 'x'
		return nil
	}); err != nil {
		t.Fatalf("Use: %v", err)
	}
	if err := credential.Use(func(detached []byte) error {
		if string(detached) != "V11_SECRET_MATERIAL" {
			t.Fatal("callback mutated retained credential")
		}
		return nil
	}); err != nil {
		t.Fatalf("second Use: %v", err)
	}
	if got := fmt.Sprintf("%v/%#v", credential, credential); strings.Contains(got, "V11_SECRET_MATERIAL") {
		t.Fatalf("formatted credential leaked: %s", got)
	}
	if _, err := json.Marshal(credential); err == nil {
		t.Fatal("credential serialized")
	}
	if _, err := NewCredential(nil); err == nil {
		t.Fatal("empty credential accepted")
	}
	if err := (Credential{}).Use(func([]byte) error { return nil }); err == nil {
		t.Fatal("empty credential used")
	}
	if err := credential.Use(nil); err == nil {
		t.Fatal("nil credential callback accepted")
	}
}

func TestCanonicalIssuerRejectsAmbiguousOrInsecureNamespaces(t *testing.T) {
	for _, issuer := range []string{
		"https://idp.example.test",
		"https://idp.example.test/tenant/v2.0",
		"http://localhost:5556/dex",
		"http://127.0.0.1:5556",
		"http://[::1]:5556",
	} {
		if got, err := CanonicalIssuer(issuer); err != nil || got != issuer {
			t.Errorf("CanonicalIssuer(%q) = %q, %v", issuer, got, err)
		}
	}
	for _, issuer := range []string{
		"", " https://idp.example.test", "https://IDP.example.test",
		"HTTPS://idp.example.test", "https://user@idp.example.test",
		"https://idp.example.test?tenant=x", "https://idp.example.test#fragment",
		"http://idp.example.test", "ftp://idp.example.test", "relative/path",
	} {
		if got, err := CanonicalIssuer(issuer); err == nil || got != "" {
			t.Errorf("invalid issuer accepted: %q -> %q", issuer, got)
		}
	}
}

func TestStablePrincipalUsesOnlyMethodIssuerAndExactSubject(t *testing.T) {
	wantRef := "principal:sha256:a8c85f0b46c9fb75a6acd64dc6aa4a0eafc5771fbff3c190b9f2dfe0e70f7477"
	wantActor := "actor:sha256:a8c85f0b46c9fb75a6acd64dc6aa4a0eafc5771fbff3c190b9f2dfe0e70f7477"
	first, err := StablePrincipal("oidc", "https://idp.v11.invalid", "00u-v11-alice", PrincipalKindHuman)
	if err != nil {
		t.Fatalf("StablePrincipal: %v", err)
	}
	second, err := StablePrincipal("oidc", "https://idp.v11.invalid", "00u-v11-alice", PrincipalKindHuman)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("stable mapping changed: %+v %+v %v", first, second, err)
	}
	if first.Ref.String() != wantRef || first.ActorRef.String() != wantActor || first.Method != "oidc" {
		t.Fatalf("stable principal = %+v", first)
	}
	different, err := StablePrincipal("oidc", "https://idp.v11.invalid", "00u-v11-Alice", PrincipalKindHuman)
	if err != nil || different.Ref == first.Ref {
		t.Fatal("exact case-sensitive subject did not affect identity")
	}
	for _, input := range []struct {
		method  AuthenticationMethod
		issuer  string
		subject string
	}{
		{"", "https://idp.v11.invalid", "subject"},
		{"oidc", "http://remote.invalid", "subject"},
		{"oidc", "https://idp.v11.invalid", ""},
		{"oidc", "https://idp.v11.invalid", "line\nsubject"},
		{"oidc", "https://idp.v11.invalid", strings.Repeat("x", 256)},
	} {
		if _, err := StablePrincipal(input.method, input.issuer, input.subject, PrincipalKindHuman); err == nil {
			t.Errorf("invalid stable identity accepted: %+v", input)
		}
	}
}
