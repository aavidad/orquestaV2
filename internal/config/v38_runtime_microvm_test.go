package config

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const validRuntimeMicroVMTOML = `[runtime]
provider = "codex"
isolation = "microvm"

[runtime.codex]
model = "gpt-5.6"

[runtime.microvm]
placement_ref = "placement:codex:account-1"
socket_path = "/run/orquesta/agente-microvm.sock"
profile_descriptor_path = "/srv/orquesta/profiles/codex-v1.json"
expected_profile_descriptor_sha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
launch_grant_key_id = "clave-publica:orquesta-01"
launch_grant_signing_credential_ref = "credential:microvm-launch-signing"
`

func TestV38RuntimeMicroVMConfigurationIsCanonicalAndRedacted(t *testing.T) {
	snapshot := resolveTOML(t, validRuntimeMicroVMTOML, nil)
	if snapshot.RuntimeProvider() != "codex" || snapshot.RuntimeIsolation() != "microvm" ||
		snapshot.RuntimeCodexModel() != "gpt-5.6" ||
		snapshot.RuntimeMicroVMPlacementRef() != "placement:codex:account-1" ||
		snapshot.RuntimeMicroVMSocketPath() != "/run/orquesta/agente-microvm.sock" ||
		snapshot.RuntimeMicroVMProfileDescriptorPath() != "/srv/orquesta/profiles/codex-v1.json" ||
		snapshot.RuntimeMicroVMExpectedProfileDescriptorSHA256() != strings.Repeat("a", 64) ||
		snapshot.RuntimeMicroVMLaunchGrantKeyID() != "clave-publica:orquesta-01" ||
		snapshot.RuntimeMicroVMLaunchGrantSigningCredentialRef() != "credential:microvm-launch-signing" {
		t.Fatal("runtime microVM configuration drifted")
	}

	definition, found := Definition(KeyRuntimeMicroVMLaunchGrantSigningCredentialRef)
	if !found || definition.Type != "credential_ref" || !definition.Sensitive || definition.Scope != "runtime" {
		t.Fatalf("launch signing credential definition = %+v/%v", definition, found)
	}
	placement, found := Definition(KeyRuntimeMicroVMPlacementRef)
	if !found || placement.Type != "string" || placement.Sensitive || placement.Scope != "runtime" ||
		!placement.RestartRequired || placement.EnvAlias != "ORQUESTA_RUNTIME_MICROVM_PLACEMENT_REF" ||
		!reflect.DeepEqual(placement.ValidatorIDs, []string{"trimmed_optional_string"}) {
		t.Fatalf("placement definition = %+v/%v", placement, found)
	}
	effective, err := snapshot.EffectiveJSON()
	if err != nil {
		t.Fatalf("effective config: %v", err)
	}
	if strings.Contains(string(effective), "credential:microvm-launch-signing") {
		t.Fatal("effective config leaks launch signing credential reference")
	}
	var document effectiveDocument
	if err := json.Unmarshal(effective, &document); err != nil {
		t.Fatalf("decode effective config: %v", err)
	}
	var foundPlacement, foundCredential bool
	for _, entry := range document.Entries {
		switch entry.Key {
		case KeyRuntimeMicroVMPlacementRef:
			foundPlacement = true
			if entry.Value != "placement:codex:account-1" || entry.Sensitive || entry.Source != SourceFile {
				t.Fatalf("effective placement = %+v", entry)
			}
		case KeyRuntimeMicroVMLaunchGrantSigningCredentialRef:
			foundCredential = true
			if entry.Value != redactedValue || !entry.Sensitive || entry.Source != SourceFile {
				t.Fatalf("effective signing credential = %+v", entry)
			}
		}
	}
	if !foundPlacement || !foundCredential {
		t.Fatalf("effective microVM identity missing: placement=%v credential=%v", foundPlacement, foundCredential)
	}

	environmentSnapshot := resolveTOML(t, validRuntimeMicroVMTOML, map[string]string{
		"ORQUESTA_RUNTIME_MICROVM_PLACEMENT_REF": "placement:codex:environment",
	})
	if environmentSnapshot.RuntimeMicroVMPlacementRef() != "placement:codex:environment" {
		t.Fatal("canonical placement environment alias did not override TOML")
	}
	assertSource(t, environmentSnapshot, KeyRuntimeMicroVMPlacementRef, SourceEnv)
}

func TestV38RuntimeMicroVMCrossValidatorIdentityAndOrderAreExact(t *testing.T) {
	wantKeys := []Key{
		KeyRuntimeProvider,
		KeyRuntimeIsolation,
		KeyRuntimeCodexModel,
		KeyRuntimeMicroVMPlacementRef,
		KeyRuntimeMicroVMSocketPath,
		KeyRuntimeMicroVMProfileDescriptorPath,
		KeyRuntimeMicroVMExpectedProfileDescriptorSHA256,
		KeyRuntimeMicroVMLaunchGrantKeyID,
		KeyRuntimeMicroVMLaunchGrantSigningCredentialRef,
	}
	for _, validator := range CrossValidators() {
		if validator.ID != "runtime_microvm_requirements" {
			continue
		}
		if !reflect.DeepEqual(validator.Keys, wantKeys) {
			t.Fatalf("runtime microVM cross-validator keys = %#v, want %#v", validator.Keys, wantKeys)
		}
		return
	}
	t.Fatal("runtime_microvm_requirements cross-validator missing")
}

func TestV38ProcessIsolationRejectsDeadMicroVMConfiguration(t *testing.T) {
	defaults := resolveTOML(t, "", nil)
	if defaults.RuntimeIsolation() != "process" || defaults.RuntimeMicroVMSocketPath() != "" ||
		defaults.RuntimeMicroVMPlacementRef() != "" ||
		defaults.RuntimeMicroVMProfileDescriptorPath() != "" ||
		defaults.RuntimeMicroVMExpectedProfileDescriptorSHA256() != "" ||
		defaults.RuntimeMicroVMLaunchGrantKeyID() != "" ||
		defaults.RuntimeMicroVMLaunchGrantSigningCredentialRef() != "" {
		t.Fatal("process defaults retain microVM configuration")
	}
	processWithModel := resolveTOML(t, `[runtime.codex]
model = "gpt-5.6"`, nil)
	if processWithModel.RuntimeIsolation() != "process" || processWithModel.RuntimeCodexModel() != "gpt-5.6" ||
		processWithModel.RuntimeMicroVMPlacementRef() != "" {
		t.Fatal("process isolation incorrectly requires the provider model to be empty")
	}

	tests := []struct {
		name string
		toml string
	}{
		{name: "placement", toml: `[runtime.microvm]
placement_ref = "placement:codex:account-1"`},
		{name: "socket", toml: `[runtime.microvm]
socket_path = "/run/orquesta/agente-microvm.sock"`},
		{name: "profile descriptor", toml: `[runtime.microvm]
profile_descriptor_path = "/srv/orquesta/profiles/codex-v1.json"`},
		{name: "profile digest", toml: `[runtime.microvm]
expected_profile_descriptor_sha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`},
		{name: "launch key", toml: `[runtime.microvm]
launch_grant_key_id = "clave-publica:orquesta-01"`},
		{name: "signing credential", toml: `[runtime.microvm]
launch_grant_signing_credential_ref = "credential:microvm-launch-signing"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Resolve(ResolveOptions{TOML: []byte(test.toml)})
			assertConfigError(t, err, ErrorCrossValidation, "")
		})
	}
}

func TestV38MicroVMIsolationRejectsIncompleteOrNonCanonicalConfiguration(t *testing.T) {
	tests := []struct {
		name string
		old  string
		new  string
	}{
		{name: "missing socket", old: `socket_path = "/run/orquesta/agente-microvm.sock"`, new: `socket_path = ""`},
		{name: "missing placement", old: `placement_ref = "placement:codex:account-1"`, new: `placement_ref = ""`},
		{name: "wrong placement prefix", old: `placement_ref = "placement:codex:account-1"`, new: `placement_ref = "profile:codex:account-1"`},
		{name: "empty placement suffix", old: `placement_ref = "placement:codex:account-1"`, new: `placement_ref = "placement:"`},
		{name: "missing provider model", old: `model = "gpt-5.6"`, new: `model = ""`},
		{name: "oversized provider model", old: `model = "gpt-5.6"`, new: `model = "` + strings.Repeat("m", 129) + `"`},
		{name: "relative socket", old: `socket_path = "/run/orquesta/agente-microvm.sock"`, new: `socket_path = "run/orquesta/agente-microvm.sock"`},
		{name: "missing profile", old: `profile_descriptor_path = "/srv/orquesta/profiles/codex-v1.json"`, new: `profile_descriptor_path = ""`},
		{name: "unclean profile", old: `profile_descriptor_path = "/srv/orquesta/profiles/codex-v1.json"`, new: `profile_descriptor_path = "/srv/orquesta/profiles/../codex-v1.json"`},
		{name: "missing digest", old: `expected_profile_descriptor_sha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`, new: `expected_profile_descriptor_sha256 = ""`},
		{name: "short digest", old: `expected_profile_descriptor_sha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`, new: `expected_profile_descriptor_sha256 = "` + strings.Repeat("a", 63) + `"`},
		{name: "uppercase digest", old: `expected_profile_descriptor_sha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`, new: `expected_profile_descriptor_sha256 = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"`},
		{name: "non hexadecimal digest", old: `expected_profile_descriptor_sha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`, new: `expected_profile_descriptor_sha256 = "gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg"`},
		{name: "missing launch key", old: `launch_grant_key_id = "clave-publica:orquesta-01"`, new: `launch_grant_key_id = ""`},
		{name: "empty launch key suffix", old: `launch_grant_key_id = "clave-publica:orquesta-01"`, new: `launch_grant_key_id = "clave-publica:"`},
		{name: "non canonical launch key", old: `launch_grant_key_id = "clave-publica:orquesta-01"`, new: `launch_grant_key_id = "clave-publica:Orquesta.01"`},
		{name: "missing signing credential", old: `launch_grant_signing_credential_ref = "credential:microvm-launch-signing"`, new: `launch_grant_signing_credential_ref = ""`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(validRuntimeMicroVMTOML, test.old, test.new, 1)
			if source == validRuntimeMicroVMTOML {
				t.Fatal("test mutation did not apply")
			}
			_, err := Resolve(ResolveOptions{TOML: []byte(source)})
			assertConfigError(t, err, ErrorCrossValidation, "")
		})
	}

	invalidCredential := strings.Replace(
		validRuntimeMicroVMTOML,
		`launch_grant_signing_credential_ref = "credential:microvm-launch-signing"`,
		`launch_grant_signing_credential_ref = "secret-inline"`,
		1,
	)
	_, err := Resolve(ResolveOptions{TOML: []byte(invalidCredential)})
	assertConfigError(t, err, ErrorValueInvalid, KeyRuntimeMicroVMLaunchGrantSigningCredentialRef)

	untrimmedPlacement := strings.Replace(
		validRuntimeMicroVMTOML,
		`placement_ref = "placement:codex:account-1"`,
		`placement_ref = " placement:codex:account-1"`,
		1,
	)
	_, err = Resolve(ResolveOptions{TOML: []byte(untrimmedPlacement)})
	assertConfigError(t, err, ErrorValueInvalid, KeyRuntimeMicroVMPlacementRef)
}
