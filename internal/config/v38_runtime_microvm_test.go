package config

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

const validRuntimeMicroVMTOML = `[runtime]
provider = "codex"
isolation = "microvm"

[runtime.codex]
model = "gpt-5.6"
credential_ref = "credential:codex-account-1"

[runtime.microvm]
placement_ref = "placement:codex:account-1"
socket_path = "/run/orquesta/agente-microvm.sock"
profile_descriptor_path = "/srv/orquesta/profiles/codex-v1.json"
expected_profile_descriptor_sha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
launch_grant_key_id = "clave-publica:orquesta-01"
launch_grant_signing_credential_ref = "credential:microvm-launch-signing"
expired_launch_continuation_authority_signing_credential_ref = "credential:microvm-continuation-signing"
expired_launch_continuation_authority_key_id = "continuation-orquesta-01"
expired_launch_continuation_authority_key_epoch = 1
expired_launch_continuation_authority_trust_revision = 1
expired_launch_continuation_authority_public_key_sha256 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
expired_launch_continuation_authority_validity = "2m"
credential_broker_socket_path = "/run/orquesta/credential-broker.sock"
credential_broker_peer_uid = 0
credential_broker_exchange_timeout = "30s"
credential_broker_max_connections = 16
`

const validRuntimeMicroVMEgressTOML = validRuntimeMicroVMTOML + `egress_policy_ref = "egreso:codex-controlado"
egress_policy_path = "/srv/orquesta/policies/codex-controlado.json"
egress_policy_expected_sha256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
egress_policy_max_bytes = 4096
`

func TestV38RuntimeMicroVMConfigurationIsCanonicalAndRedacted(t *testing.T) {
	snapshot := resolveTOML(t, validRuntimeMicroVMTOML, nil)
	if snapshot.RuntimeProvider() != "codex" || snapshot.RuntimeIsolation() != "microvm" ||
		snapshot.RuntimeCodexModel() != "gpt-5.6" ||
		snapshot.RuntimeCodexCredentialRef() != "credential:codex-account-1" ||
		snapshot.RuntimeMicroVMPlacementRef() != "placement:codex:account-1" ||
		snapshot.RuntimeMicroVMSocketPath() != "/run/orquesta/agente-microvm.sock" ||
		snapshot.RuntimeMicroVMProfileDescriptorPath() != "/srv/orquesta/profiles/codex-v1.json" ||
		snapshot.RuntimeMicroVMExpectedProfileDescriptorSHA256() != strings.Repeat("a", 64) ||
		snapshot.RuntimeMicroVMLaunchGrantKeyID() != "clave-publica:orquesta-01" ||
		snapshot.RuntimeMicroVMLaunchGrantSigningCredentialRef() != "credential:microvm-launch-signing" ||
		snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthoritySigningCredentialRef() != "credential:microvm-continuation-signing" ||
		snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthorityKeyID() != "continuation-orquesta-01" ||
		snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthorityKeyEpoch() != 1 ||
		snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthorityTrustRevision() != 1 ||
		snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthorityPublicKeySHA256() != strings.Repeat("c", 64) ||
		snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthorityValidity() != 2*time.Minute ||
		snapshot.RuntimeMicroVMCredentialBrokerSocketPath() != "/run/orquesta/credential-broker.sock" ||
		snapshot.RuntimeMicroVMCredentialBrokerPeerUID() != 0 ||
		snapshot.RuntimeMicroVMCredentialBrokerExchangeTimeout() != 30*time.Second ||
		snapshot.RuntimeMicroVMCredentialBrokerMaxConnections() != 16 {
		t.Fatal("runtime microVM configuration drifted")
	}

	definition, found := Definition(KeyRuntimeMicroVMLaunchGrantSigningCredentialRef)
	if !found || definition.Type != "credential_ref" || !definition.Sensitive || definition.Scope != "runtime" {
		t.Fatalf("launch signing credential definition = %+v/%v", definition, found)
	}
	continuationDefinition, found := Definition(KeyRuntimeMicroVMExpiredLaunchContinuationAuthoritySigningCredentialRef)
	if !found || continuationDefinition.Type != "credential_ref" || !continuationDefinition.Sensitive || continuationDefinition.Scope != "runtime" {
		t.Fatalf("continuation signing credential definition = %+v/%v", continuationDefinition, found)
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
	if strings.Contains(string(effective), "credential:microvm-launch-signing") ||
		strings.Contains(string(effective), "credential:microvm-continuation-signing") ||
		strings.Contains(string(effective), "credential:codex-account-1") {
		t.Fatal("effective config leaks a microVM credential reference")
	}
	var document effectiveDocument
	if err := json.Unmarshal(effective, &document); err != nil {
		t.Fatalf("decode effective config: %v", err)
	}
	var foundPlacement, foundSigningCredential, foundProviderCredential, foundBrokerSocket bool
	for _, entry := range document.Entries {
		switch entry.Key {
		case KeyRuntimeMicroVMPlacementRef:
			foundPlacement = true
			if entry.Value != "placement:codex:account-1" || entry.Sensitive || entry.Source != SourceFile {
				t.Fatalf("effective placement = %+v", entry)
			}
		case KeyRuntimeMicroVMLaunchGrantSigningCredentialRef:
			foundSigningCredential = true
			if entry.Value != redactedValue || !entry.Sensitive || entry.Source != SourceFile {
				t.Fatalf("effective signing credential = %+v", entry)
			}
		case KeyRuntimeCodexCredentialRef:
			foundProviderCredential = true
			if entry.Value != redactedValue || !entry.Sensitive || entry.Source != SourceFile {
				t.Fatalf("effective provider credential = %+v", entry)
			}
		case KeyRuntimeMicroVMCredentialBrokerSocketPath:
			foundBrokerSocket = true
			if entry.Value != "/run/orquesta/credential-broker.sock" || entry.Sensitive || entry.Source != SourceFile {
				t.Fatalf("effective broker socket = %+v", entry)
			}
		}
	}
	if !foundPlacement || !foundSigningCredential || !foundProviderCredential || !foundBrokerSocket {
		t.Fatalf("effective microVM identity missing: placement=%v signing=%v provider=%v broker_socket=%v",
			foundPlacement, foundSigningCredential, foundProviderCredential, foundBrokerSocket)
	}

	environmentSnapshot := resolveTOML(t, validRuntimeMicroVMTOML, map[string]string{
		"ORQUESTA_RUNTIME_MICROVM_PLACEMENT_REF":                      "placement:codex:environment",
		"ORQUESTA_RUNTIME_MICROVM_CREDENTIAL_BROKER_PEER_UID":         "109",
		"ORQUESTA_RUNTIME_MICROVM_CREDENTIAL_BROKER_EXCHANGE_TIMEOUT": "5s",
		"ORQUESTA_RUNTIME_MICROVM_CREDENTIAL_BROKER_MAX_CONNECTIONS":  "256",
	})
	if environmentSnapshot.RuntimeMicroVMPlacementRef() != "placement:codex:environment" ||
		environmentSnapshot.RuntimeMicroVMCredentialBrokerPeerUID() != 109 ||
		environmentSnapshot.RuntimeMicroVMCredentialBrokerExchangeTimeout() != 5*time.Second ||
		environmentSnapshot.RuntimeMicroVMCredentialBrokerMaxConnections() != 256 {
		t.Fatal("canonical microVM environment aliases did not override TOML")
	}
	assertSource(t, environmentSnapshot, KeyRuntimeMicroVMPlacementRef, SourceEnv)
	assertSource(t, environmentSnapshot, KeyRuntimeMicroVMCredentialBrokerPeerUID, SourceEnv)
	assertSource(t, environmentSnapshot, KeyRuntimeMicroVMCredentialBrokerExchangeTimeout, SourceEnv)
	assertSource(t, environmentSnapshot, KeyRuntimeMicroVMCredentialBrokerMaxConnections, SourceEnv)
}

func TestV38MicroVMSeparatesHostQuotaProfileFromGuestCredential(t *testing.T) {
	source := strings.Replace(validRuntimeMicroVMTOML, "[runtime.codex]\n", `[runtime.codex]
account_home_root = "/srv/orquesta/codex-accounts"
account_profile = "C46Pilot"
max_concurrent_executions = 1
`, 1)
	snapshot, err := Resolve(ResolveOptions{TOML: []byte(source)})
	if err != nil {
		t.Fatalf("Resolve(microVM quota profile) error = %v", err)
	}
	if snapshot.RuntimeIsolation() != "microvm" ||
		snapshot.RuntimeCodexCredentialRef() != "credential:codex-account-1" ||
		snapshot.RuntimeCodexAccountHomeRoot() != "/srv/orquesta/codex-accounts" ||
		snapshot.RuntimeCodexAccountProfile() != "C46Pilot" {
		t.Fatal("microVM mixed host/guest authority drifted")
	}
}

func TestCredentialProvisioningResolverOmitsOnlyFutureProfileDescriptor(t *testing.T) {
	withoutDescriptor := strings.ReplaceAll(
		strings.ReplaceAll(
			validRuntimeMicroVMTOML,
			`profile_descriptor_path = "/srv/orquesta/profiles/codex-v1.json"`+"\n",
			"",
		),
		`expected_profile_descriptor_sha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`+"\n",
		"",
	)
	snapshot, err := ResolveForCredentialProvisioning(ResolveOptions{TOML: []byte(withoutDescriptor)})
	if err != nil {
		t.Fatalf("ResolveForCredentialProvisioning() error=%v", err)
	}
	if snapshot.RuntimeMicroVMProfileDescriptorPath() != "" ||
		snapshot.RuntimeMicroVMExpectedProfileDescriptorSHA256() != "" ||
		snapshot.RuntimeMicroVMPlacementRef() != "placement:codex:account-1" ||
		snapshot.RuntimeMicroVMLaunchGrantKeyID() != "clave-publica:orquesta-01" {
		t.Fatalf("credential provisioning snapshot drifted")
	}
	if _, err := Resolve(ResolveOptions{TOML: []byte(withoutDescriptor)}); err == nil ||
		!HasErrorCode(err, ErrorCrossValidation) {
		t.Fatalf("runtime Resolve accepted configuration without physical descriptor: %v", err)
	}
	if _, err := ResolveForCredentialProvisioning(ResolveOptions{
		TOML: []byte(withoutDescriptor), Environment: map[string]string{"UNDECLARED_BOOTSTRAP_VALUE": "secret"},
	}); err == nil || !HasErrorCode(err, ErrorUnknownKey) {
		t.Fatalf("credential provisioning accepted an ad hoc environment key: %v", err)
	}

	mutations := []struct {
		name string
		old  string
	}{
		{name: "only descriptor digest remains", old: `profile_descriptor_path = "/srv/orquesta/profiles/codex-v1.json"` + "\n"},
		{name: "only descriptor path remains", old: `expected_profile_descriptor_sha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"` + "\n"},
		{name: "provider model", old: `model = "gpt-5.6"` + "\n"},
		{name: "provider credential", old: `credential_ref = "credential:codex-account-1"` + "\n"},
		{name: "placement", old: `placement_ref = "placement:codex:account-1"` + "\n"},
		{name: "agent socket", old: `socket_path = "/run/orquesta/agente-microvm.sock"` + "\n"},
		{name: "launch key", old: `launch_grant_key_id = "clave-publica:orquesta-01"` + "\n"},
		{name: "signing credential", old: `launch_grant_signing_credential_ref = "credential:microvm-launch-signing"` + "\n"},
		{name: "broker socket", old: `credential_broker_socket_path = "/run/orquesta/credential-broker.sock"` + "\n"},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			source := strings.Replace(validRuntimeMicroVMTOML, mutation.old, "", 1)
			if source == validRuntimeMicroVMTOML {
				t.Fatal("fixture mutation did not apply")
			}
			_, err := ResolveForCredentialProvisioning(ResolveOptions{TOML: []byte(source)})
			if err == nil || !HasErrorCode(err, ErrorCrossValidation) {
				t.Fatalf("provisioning resolver accepted missing %s: %v", mutation.name, err)
			}
		})
	}
}

func TestContinuationCredentialProvisioningResolverOmitsOnlyDerivedPublicDigest(t *testing.T) {
	withoutDigest := strings.Replace(
		validRuntimeMicroVMTOML,
		`expired_launch_continuation_authority_public_key_sha256 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"`+"\n",
		"",
		1,
	)
	if withoutDigest == validRuntimeMicroVMTOML {
		t.Fatal("continuation digest fixture was not removed")
	}
	snapshot, err := ResolveForContinuationCredentialProvisioning(ResolveOptions{TOML: []byte(withoutDigest)})
	if err != nil {
		t.Fatalf("ResolveForContinuationCredentialProvisioning() error=%v", err)
	}
	if snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthorityPublicKeySHA256() != "" ||
		snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthoritySigningCredentialRef() !=
			"credential:microvm-continuation-signing" {
		t.Fatalf("continuation bootstrap snapshot drifted")
	}
	if _, err := Resolve(ResolveOptions{TOML: []byte(withoutDigest)}); err == nil ||
		!HasErrorCode(err, ErrorCrossValidation) {
		t.Fatalf("runtime Resolve accepted missing continuation digest: %v", err)
	}
	if _, err := ResolveForCredentialProvisioning(ResolveOptions{TOML: []byte(withoutDigest)}); err == nil ||
		!HasErrorCode(err, ErrorCrossValidation) {
		t.Fatalf("launch credential resolver accepted missing continuation digest: %v", err)
	}

	for _, mutation := range []struct {
		name string
		old  string
		new  string
	}{
		{name: "credential ref", old: `expired_launch_continuation_authority_signing_credential_ref = "credential:microvm-continuation-signing"`, new: `expired_launch_continuation_authority_signing_credential_ref = ""`},
		{name: "key id", old: `expired_launch_continuation_authority_key_id = "continuation-orquesta-01"`, new: `expired_launch_continuation_authority_key_id = ""`},
		{name: "key epoch", old: `expired_launch_continuation_authority_key_epoch = 1`, new: `expired_launch_continuation_authority_key_epoch = 0`},
		{name: "trust revision", old: `expired_launch_continuation_authority_trust_revision = 1`, new: `expired_launch_continuation_authority_trust_revision = 0`},
		{name: "profile", old: `profile_descriptor_path = "/srv/orquesta/profiles/codex-v1.json"` + "\n", new: ""},
		{name: "invalid nonempty digest", old: "", new: `expired_launch_continuation_authority_public_key_sha256 = "not-a-digest"` + "\n"},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			source := withoutDigest
			if mutation.old == "" {
				source = strings.Replace(source, "[runtime.microvm]\n", "[runtime.microvm]\n"+mutation.new, 1)
			} else {
				source = strings.Replace(source, mutation.old, mutation.new, 1)
			}
			if source == withoutDigest {
				t.Fatal("fixture mutation did not apply")
			}
			if _, err := ResolveForContinuationCredentialProvisioning(ResolveOptions{TOML: []byte(source)}); err == nil {
				t.Fatal("continuation provisioning resolver accepted non-digest omission")
			}
		})
	}
}

func TestV38RuntimeMicroVMCrossValidatorIdentityAndOrderAreExact(t *testing.T) {
	wantKeys := []Key{
		KeyRuntimeProvider,
		KeyRuntimeIsolation,
		KeyRuntimeCodexModel,
		KeyRuntimeCodexCredentialRef,
		KeyRuntimeMicroVMPlacementRef,
		KeyRuntimeMicroVMSocketPath,
		KeyRuntimeMicroVMProfileDescriptorPath,
		KeyRuntimeMicroVMExpectedProfileDescriptorSHA256,
		KeyRuntimeMicroVMLaunchGrantKeyID,
		KeyRuntimeMicroVMLaunchGrantSigningCredentialRef,
		KeyRuntimeMicroVMExpiredLaunchContinuationAuthoritySigningCredentialRef,
		KeyRuntimeMicroVMExpiredLaunchContinuationAuthorityKeyID,
		KeyRuntimeMicroVMExpiredLaunchContinuationAuthorityKeyEpoch,
		KeyRuntimeMicroVMExpiredLaunchContinuationAuthorityTrustRevision,
		KeyRuntimeMicroVMExpiredLaunchContinuationAuthorityPublicKeySHA256,
		KeyRuntimeMicroVMExpiredLaunchContinuationAuthorityValidity,
		KeyRuntimeMicroVMCredentialBrokerSocketPath,
		KeyRuntimeMicroVMCredentialBrokerPeerUID,
		KeyRuntimeMicroVMCredentialBrokerExchangeTimeout,
		KeyRuntimeMicroVMCredentialBrokerMaxConnections,
		KeyRuntimeMicroVMEgressPolicyRef,
		KeyRuntimeMicroVMEgressPolicyPath,
		KeyRuntimeMicroVMEgressPolicyExpectedSHA256,
		KeyRuntimeMicroVMEgressPolicyMaxBytes,
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

func TestV38RuntimePathAuthorityIncludesEgressPolicyPath(t *testing.T) {
	wantKeys := []Key{
		KeyStateSQLitePath,
		KeyArtifactFilesystemRoot,
		KeyCredentialsLocalPath,
		KeyRuntimeCodexWorkRoot,
		KeyRuntimeCodexCacheRoot,
		KeyRuntimeCodexAccountHomeRoot,
		KeyRuntimeMicroVMEgressPolicyPath,
		KeyWorkspaceLocalRoot,
		KeyConfigEffectivePath,
		KeyIdentityLocalTokenPath,
		KeyIdentityLocalPrincipalsManifestPath,
	}
	for _, validator := range CrossValidators() {
		if validator.ID != "runtime_paths_disjoint" {
			continue
		}
		if !reflect.DeepEqual(validator.Keys, wantKeys) {
			t.Fatalf("runtime path cross-validator keys = %#v, want %#v", validator.Keys, wantKeys)
		}
		return
	}
	t.Fatal("runtime_paths_disjoint cross-validator missing")
}

func TestV38ProcessIsolationRejectsDeadMicroVMConfiguration(t *testing.T) {
	defaults := resolveTOML(t, "", nil)
	if defaults.RuntimeIsolation() != "process" || defaults.RuntimeMicroVMSocketPath() != "" ||
		defaults.RuntimeMicroVMPlacementRef() != "" ||
		defaults.RuntimeMicroVMProfileDescriptorPath() != "" ||
		defaults.RuntimeMicroVMExpectedProfileDescriptorSHA256() != "" ||
		defaults.RuntimeMicroVMLaunchGrantKeyID() != "" ||
		defaults.RuntimeMicroVMLaunchGrantSigningCredentialRef() != "" ||
		defaults.RuntimeMicroVMCredentialBrokerSocketPath() != "" ||
		defaults.RuntimeMicroVMCredentialBrokerPeerUID() != 0 ||
		defaults.RuntimeMicroVMCredentialBrokerExchangeTimeout() != 30*time.Second ||
		defaults.RuntimeMicroVMCredentialBrokerMaxConnections() != 16 ||
		defaults.RuntimeMicroVMEgressPolicyRef() != "" ||
		defaults.RuntimeMicroVMEgressPolicyPath() != "" ||
		defaults.RuntimeMicroVMEgressPolicyExpectedSHA256() != "" ||
		defaults.RuntimeMicroVMEgressPolicyMaxBytes() != 64<<10 ||
		defaults.RuntimeCodexCredentialRef() != "" {
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
		{name: "broker socket", toml: `[runtime.microvm]
credential_broker_socket_path = "/run/orquesta/credential-broker.sock"`},
		{name: "broker peer uid", toml: `[runtime.microvm]
credential_broker_peer_uid = 109`},
		{name: "broker exchange timeout", toml: `[runtime.microvm]
credential_broker_exchange_timeout = "5s"`},
		{name: "broker max connections", toml: `[runtime.microvm]
credential_broker_max_connections = 20`},
		{name: "egress policy ref", toml: `[runtime.microvm]
egress_policy_ref = "egreso:codex-controlado"`},
		{name: "egress policy path", toml: `[runtime.microvm]
egress_policy_path = "/srv/orquesta/policies/codex-controlado.json"`},
		{name: "egress policy digest", toml: `[runtime.microvm]
egress_policy_expected_sha256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"`},
		{name: "egress policy maximum", toml: `[runtime.microvm]
egress_policy_max_bytes = 4096`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Resolve(ResolveOptions{TOML: []byte(test.toml)})
			assertConfigError(t, err, ErrorCrossValidation, "")
		})
	}
}

func TestV38RuntimeMicroVMEgressPolicyIsOptInAndAllOrNone(t *testing.T) {
	snapshot := resolveTOML(t, validRuntimeMicroVMEgressTOML, nil)
	if snapshot.RuntimeMicroVMEgressPolicyRef() != "egreso:codex-controlado" ||
		snapshot.RuntimeMicroVMEgressPolicyPath() != "/srv/orquesta/policies/codex-controlado.json" ||
		snapshot.RuntimeMicroVMEgressPolicyExpectedSHA256() != strings.Repeat("b", 64) ||
		snapshot.RuntimeMicroVMEgressPolicyMaxBytes() != 4096 {
		t.Fatal("runtime microVM egress policy configuration drifted")
	}
	defaultBoundSource := strings.Replace(validRuntimeMicroVMEgressTOML, "egress_policy_max_bytes = 4096\n", "", 1)
	defaultBound := resolveTOML(t, defaultBoundSource, nil)
	if defaultBound.RuntimeMicroVMEgressPolicyRef() == "" || defaultBound.RuntimeMicroVMEgressPolicyMaxBytes() != 64<<10 {
		t.Fatal("configured policy did not retain the canonical default byte bound")
	}
	for _, test := range []struct {
		name string
		old  string
		new  string
	}{
		{name: "missing ref", old: `egress_policy_ref = "egreso:codex-controlado"`, new: `egress_policy_ref = ""`},
		{name: "missing path", old: `egress_policy_path = "/srv/orquesta/policies/codex-controlado.json"`, new: `egress_policy_path = ""`},
		{name: "missing digest", old: `egress_policy_expected_sha256 = "` + strings.Repeat("b", 64) + `"`, new: `egress_policy_expected_sha256 = ""`},
		{name: "relative path", old: `egress_policy_path = "/srv/orquesta/policies/codex-controlado.json"`, new: `egress_policy_path = "policies/codex-controlado.json"`},
		{name: "unclean path", old: `egress_policy_path = "/srv/orquesta/policies/codex-controlado.json"`, new: `egress_policy_path = "/srv/orquesta/policies/../codex-controlado.json"`},
		{name: "uppercase digest", old: `egress_policy_expected_sha256 = "` + strings.Repeat("b", 64) + `"`, new: `egress_policy_expected_sha256 = "` + strings.Repeat("B", 64) + `"`},
		{name: "short digest", old: `egress_policy_expected_sha256 = "` + strings.Repeat("b", 64) + `"`, new: `egress_policy_expected_sha256 = "` + strings.Repeat("b", 63) + `"`},
		{name: "oversized ref", old: `egress_policy_ref = "egreso:codex-controlado"`, new: `egress_policy_ref = "` + strings.Repeat("r", 513) + `"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(validRuntimeMicroVMEgressTOML, test.old, test.new, 1)
			if source == validRuntimeMicroVMEgressTOML {
				t.Fatal("test mutation did not apply")
			}
			_, err := Resolve(ResolveOptions{TOML: []byte(source)})
			assertConfigError(t, err, ErrorCrossValidation, "")
		})
	}

	for _, test := range []struct {
		name  string
		value string
	}{
		{name: "zero maximum", value: "0"},
		{name: "maximum above authority bound", value: "65537"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(validRuntimeMicroVMEgressTOML, "egress_policy_max_bytes = 4096", "egress_policy_max_bytes = "+test.value, 1)
			_, err := Resolve(ResolveOptions{TOML: []byte(source)})
			assertConfigError(t, err, ErrorValueInvalid, KeyRuntimeMicroVMEgressPolicyMaxBytes)
		})
	}

	environment := map[string]string{
		"ORQUESTA_RUNTIME_MICROVM_EGRESS_POLICY_REF":             "egreso:environment",
		"ORQUESTA_RUNTIME_MICROVM_EGRESS_POLICY_PATH":            "/srv/orquesta/policies/environment.json",
		"ORQUESTA_RUNTIME_MICROVM_EGRESS_POLICY_EXPECTED_SHA256": strings.Repeat("c", 64),
		"ORQUESTA_RUNTIME_MICROVM_EGRESS_POLICY_MAX_BYTES":       "8192",
	}
	environmentSnapshot := resolveTOML(t, validRuntimeMicroVMEgressTOML, environment)
	if environmentSnapshot.RuntimeMicroVMEgressPolicyRef() != "egreso:environment" ||
		environmentSnapshot.RuntimeMicroVMEgressPolicyPath() != "/srv/orquesta/policies/environment.json" ||
		environmentSnapshot.RuntimeMicroVMEgressPolicyExpectedSHA256() != strings.Repeat("c", 64) ||
		environmentSnapshot.RuntimeMicroVMEgressPolicyMaxBytes() != 8192 {
		t.Fatal("canonical egress policy environment aliases did not override TOML")
	}
	assertSource(t, environmentSnapshot, KeyRuntimeMicroVMEgressPolicyRef, SourceEnv)
	assertSource(t, environmentSnapshot, KeyRuntimeMicroVMEgressPolicyPath, SourceEnv)
	assertSource(t, environmentSnapshot, KeyRuntimeMicroVMEgressPolicyExpectedSHA256, SourceEnv)
	assertSource(t, environmentSnapshot, KeyRuntimeMicroVMEgressPolicyMaxBytes, SourceEnv)
	_, err := Resolve(ResolveOptions{TOML: []byte(validRuntimeMicroVMTOML), Environment: map[string]string{
		"ORQUESTA_RUNTIME_MICROVM_EGRESS_POLICY_REF": "egreso:partial-environment",
	}})
	assertConfigError(t, err, ErrorCrossValidation, "")
	_, err = Resolve(ResolveOptions{TOML: []byte(validRuntimeMicroVMTOML), Environment: map[string]string{
		"ORQUESTA_RUNTIME_MICROVM_EGRESS_POLICY_MAX_BYTES": "4096",
	}})
	assertConfigError(t, err, ErrorCrossValidation, "")

	definition, found := Definition(KeyRuntimeMicroVMEgressPolicyMaxBytes)
	if !found || definition.Type != "integer" || definition.Sensitive || definition.Scope != "runtime" ||
		definition.Minimum == nil || *definition.Minimum != 1 ||
		definition.Maximum == nil || *definition.Maximum != 64<<10 {
		t.Fatalf("egress policy maximum definition = %+v/%v", definition, found)
	}
}

func TestV38RuntimeMicroVMEgressPolicyPathRejectsProtectedRuntimePairs(t *testing.T) {
	const configuredPath = "/srv/orquesta/policies/codex-controlado.json"
	tests := []struct {
		name       string
		policyPath string
		extraTOML  string
		sourcePath string
	}{
		{name: "effective config", policyPath: "/srv/orquesta/collisions/effective.json",
			extraTOML: "\n[config]\neffective_path = \"/srv/orquesta/collisions/effective.json\"\n"},
		{name: "state", policyPath: "/srv/orquesta/collisions/state/orquesta.sqlite",
			extraTOML: "\n[state.sqlite]\npath = \"/srv/orquesta/collisions/state/orquesta.sqlite\"\n"},
		{name: "state sidecar", policyPath: "/srv/orquesta/collisions/state/orquesta.sqlite-wal",
			extraTOML: "\n[state.sqlite]\npath = \"/srv/orquesta/collisions/state/orquesta.sqlite\"\n"},
		{name: "artifact root", policyPath: "/srv/orquesta/collisions/artifacts/policy.json",
			extraTOML: "\n[artifact.filesystem]\nroot = \"/srv/orquesta/collisions/artifacts\"\n"},
		{name: "credential store", policyPath: "/srv/orquesta/collisions/credentials.json",
			extraTOML: "\n[credentials.local]\npath = \"/srv/orquesta/collisions/credentials.json\"\n"},
		{name: "credential recovery sidecar", policyPath: "/srv/orquesta/collisions/credentials.json.next",
			extraTOML: "\n[credentials.local]\npath = \"/srv/orquesta/collisions/credentials.json\"\n"},
		{name: "workspace root", policyPath: "/srv/orquesta/collisions/workspaces/policy.json",
			extraTOML: "\n[workspace.local]\nroot = \"/srv/orquesta/collisions/workspaces\"\n"},
		{name: "config source sidecar", policyPath: "/srv/orquesta/collisions/orquesta.toml.lock",
			sourcePath: "/srv/orquesta/collisions/orquesta.toml.lock"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(
				validRuntimeMicroVMEgressTOML,
				`egress_policy_path = "`+configuredPath+`"`,
				`egress_policy_path = "`+test.policyPath+`"`,
				1,
			) + test.extraTOML
			_, err := Resolve(ResolveOptions{TOML: []byte(source), SourcePath: test.sourcePath})
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
		{name: "missing provider credential", old: `credential_ref = "credential:codex-account-1"`, new: `credential_ref = ""`},
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
		{name: "missing continuation signing credential", old: `expired_launch_continuation_authority_signing_credential_ref = "credential:microvm-continuation-signing"`, new: `expired_launch_continuation_authority_signing_credential_ref = ""`},
		{name: "missing continuation key id", old: `expired_launch_continuation_authority_key_id = "continuation-orquesta-01"`, new: `expired_launch_continuation_authority_key_id = ""`},
		{name: "missing continuation key epoch", old: `expired_launch_continuation_authority_key_epoch = 1`, new: `expired_launch_continuation_authority_key_epoch = 0`},
		{name: "missing continuation trust revision", old: `expired_launch_continuation_authority_trust_revision = 1`, new: `expired_launch_continuation_authority_trust_revision = 0`},
		{name: "missing continuation public key digest", old: `expired_launch_continuation_authority_public_key_sha256 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"`, new: `expired_launch_continuation_authority_public_key_sha256 = ""`},
		{name: "continuation validity above protocol bound", old: `expired_launch_continuation_authority_validity = "2m"`, new: `expired_launch_continuation_authority_validity = "6m"`},
		{name: "missing broker socket", old: `credential_broker_socket_path = "/run/orquesta/credential-broker.sock"`, new: `credential_broker_socket_path = ""`},
		{name: "relative broker socket", old: `credential_broker_socket_path = "/run/orquesta/credential-broker.sock"`, new: `credential_broker_socket_path = "run/orquesta/credential-broker.sock"`},
		{name: "broker socket equals launcher socket", old: `credential_broker_socket_path = "/run/orquesta/credential-broker.sock"`, new: `credential_broker_socket_path = "/run/orquesta/agente-microvm.sock"`},
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

	invalidTypedValues := []struct {
		name string
		old  string
		new  string
		key  Key
	}{
		{name: "broker peer uid below uint32", old: `credential_broker_peer_uid = 0`, new: `credential_broker_peer_uid = -1`, key: KeyRuntimeMicroVMCredentialBrokerPeerUID},
		{name: "broker peer uid above uint32", old: `credential_broker_peer_uid = 0`, new: `credential_broker_peer_uid = 4294967296`, key: KeyRuntimeMicroVMCredentialBrokerPeerUID},
		{name: "broker exchange timeout is zero", old: `credential_broker_exchange_timeout = "30s"`, new: `credential_broker_exchange_timeout = "0s"`, key: KeyRuntimeMicroVMCredentialBrokerExchangeTimeout},
		{name: "broker max connections below boundary", old: `credential_broker_max_connections = 16`, new: `credential_broker_max_connections = 0`, key: KeyRuntimeMicroVMCredentialBrokerMaxConnections},
		{name: "broker max connections above physical boundary", old: `credential_broker_max_connections = 16`, new: `credential_broker_max_connections = 257`, key: KeyRuntimeMicroVMCredentialBrokerMaxConnections},
	}
	for _, test := range invalidTypedValues {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(validRuntimeMicroVMTOML, test.old, test.new, 1)
			if source == validRuntimeMicroVMTOML {
				t.Fatal("typed test mutation did not apply")
			}
			_, err := Resolve(ResolveOptions{TOML: []byte(source)})
			assertConfigError(t, err, ErrorValueInvalid, test.key)
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

	reusedSigningCredential := strings.Replace(
		validRuntimeMicroVMTOML,
		`credential_ref = "credential:codex-account-1"`,
		`credential_ref = "credential:microvm-launch-signing"`,
		1,
	)
	_, err = Resolve(ResolveOptions{TOML: []byte(reusedSigningCredential)})
	assertConfigError(t, err, ErrorCrossValidation, "")

	untrimmedPlacement := strings.Replace(
		validRuntimeMicroVMTOML,
		`placement_ref = "placement:codex:account-1"`,
		`placement_ref = " placement:codex:account-1"`,
		1,
	)
	_, err = Resolve(ResolveOptions{TOML: []byte(untrimmedPlacement)})
	assertConfigError(t, err, ErrorValueInvalid, KeyRuntimeMicroVMPlacementRef)
}
