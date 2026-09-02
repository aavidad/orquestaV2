package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"orquesta/internal/identity"
)

// ResolveOptions contains captured inputs. Environment must contain only
// environment names declared by the canonical registry.
type ResolveOptions struct {
	TOML        []byte
	Environment map[string]string
	SourcePath  string
}

// Resolve resolves captured inputs without reading process-global state.
func Resolve(options ResolveOptions) (Snapshot, error) {
	return resolve(options, resolveProfileRuntime)
}

// ResolveForCredentialProvisioning resolves the canonical configuration for
// the one-shot credential bootstrap. It keeps every normal validation except
// that the future physical profile descriptor and its digest may both be
// absent. Resolve remains the only runtime/server resolver.
func ResolveForCredentialProvisioning(options ResolveOptions) (Snapshot, error) {
	return resolve(options, resolveProfileCredentialProvisioning)
}

// ResolveForContinuationCredentialProvisioning resolves the canonical
// runtime document before the public continuation-key digest is known. It
// relaxes only that derived digest; the normal runtime resolver still requires
// the final value and every other microVM authority remains mandatory.
func ResolveForContinuationCredentialProvisioning(options ResolveOptions) (Snapshot, error) {
	return resolve(options, resolveProfileContinuationCredentialProvisioning)
}

// ResolveForStateMigration parses the canonical document but resolves only
// the three SQLite options needed by the offline maintenance command. Known
// runtime, provider and attestor values are deliberately not interpreted.
func ResolveForStateMigration(options ResolveOptions) (Snapshot, error) {
	return resolve(options, resolveProfileStateMigration)
}

type resolveProfile uint8

const (
	resolveProfileRuntime resolveProfile = iota
	resolveProfileCredentialProvisioning
	resolveProfileContinuationCredentialProvisioning
	resolveProfileStateMigration
)

func resolve(options ResolveOptions, profile resolveProfile) (Snapshot, error) {
	registry, err := loadRegistry()
	if err != nil {
		return Snapshot{}, err
	}
	values, err := defaultResolvedValues(registry)
	if err != nil {
		return Snapshot{}, err
	}
	if options.TOML != nil {
		var explicit map[Key]any
		if profile == resolveProfileStateMigration {
			explicit, err = parseExplicitSelectedWithRegistry(
				options.TOML, registry, isStateMigrationConfigKey,
			)
		} else {
			explicit, err = parseExplicitWithRegistry(options.TOML, registry)
		}
		if err != nil {
			return Snapshot{}, err
		}
		applyExplicit(values, explicit)
	}
	for name := range options.Environment {
		_, canonical := registry.environmentTargets[name]
		_, alias := registry.environmentAliases[name]
		if !canonical && !alias {
			return Snapshot{}, &Error{Code: ErrorUnknownKey, Key: Key(name)}
		}
	}
	if err := applyEnvironmentMapForProfile(values, registry, options.Environment, profile); err != nil {
		return Snapshot{}, err
	}
	if err := validateCrossRegistryValues(registry, values, options.SourcePath, profile); err != nil {
		return Snapshot{}, err
	}
	return buildSnapshot(registry, values)
}

type resolvedValue struct {
	value  any
	source Source
}

func defaultResolvedValues(registry registry) (map[Key]resolvedValue, error) {
	resolved := make(map[Key]resolvedValue, len(registry.keys))
	for _, definition := range registry.keys {
		value, err := parseDefaultValue(definition)
		if err != nil {
			return nil, &Error{Code: ErrorRegistryInvalid, Key: definition.Key, Cause: err}
		}
		resolved[definition.Key] = resolvedValue{value: cloneValue(value), source: SourceDefault}
	}
	return resolved, nil
}

func applyExplicit(resolved map[Key]resolvedValue, explicit map[Key]any) {
	for key, value := range explicit {
		resolved[key] = resolvedValue{value: cloneValue(value), source: SourceFile}
	}
}

func applyEnvironmentMap(resolved map[Key]resolvedValue, registry registry, environment map[string]string) error {
	return applyEnvironmentMapForProfile(resolved, registry, environment, resolveProfileRuntime)
}

func applyEnvironmentMapForProfile(
	resolved map[Key]resolvedValue,
	registry registry,
	environment map[string]string,
	profile resolveProfile,
) error {
	normalized := make(map[Key]string, len(environment))
	for name, raw := range environment {
		key, canonical := registry.environmentTargets[name]
		if !canonical {
			key = registry.environmentAliases[name]
		}
		if profile == resolveProfileStateMigration && !isStateMigrationConfigKey(key) {
			continue
		}
		if _, duplicate := normalized[key]; duplicate {
			return &Error{Code: ErrorValueInvalid, Key: key, Cause: fmt.Errorf("config_environment_alias_ambiguous")}
		}
		normalized[key] = raw
	}
	for key, raw := range normalized {
		definition, _ := registry.definition(key)
		value, err := parseEnvironmentValue(definition, raw)
		if err != nil {
			return &Error{Code: ErrorValueInvalid, Key: definition.Key, Cause: err}
		}
		resolved[definition.Key] = resolvedValue{value: cloneValue(value), source: SourceEnv}
	}
	return nil
}

func validateCrossRegistryValues(
	registry registry,
	values map[Key]resolvedValue,
	sourcePath string,
	profile resolveProfile,
) error {
	if profile == resolveProfileStateMigration {
		return nil
	}
	fail := func(id string) error {
		return &Error{Code: ErrorCrossValidation, Cause: fmt.Errorf("%s", id)}
	}
	for _, validator := range registry.crossValidators {
		switch validator.ID {
		case "runtime_codex_timeout_before_scheduler_execution_timeout":
			runtimeTimeout, runtimeOK := values[KeyRuntimeCodexTimeout].value.(time.Duration)
			executionTimeout, executionOK := values[KeySchedulerExecutionTimeout].value.(time.Duration)
			if !runtimeOK || !executionOK || runtimeTimeout >= executionTimeout {
				return fail(validator.ID)
			}
		case "server_listen_loopback":
			listen, ok := values[KeyServerListen].value.(string)
			host, _, err := net.SplitHostPort(listen)
			ip := net.ParseIP(host)
			if !ok || err != nil || ip == nil || !ip.IsLoopback() {
				return fail(validator.ID)
			}
		case "server_mcp_path_literal":
			value, ok := values[KeyServerMCPPath].value.(string)
			if !ok || !literalMCPPath(value) {
				return fail(validator.ID)
			}
		case "runtime_paths_disjoint":
			if !runtimePathsDisjoint(values, sourcePath) {
				return fail(validator.ID)
			}
		case "runtime_codex_supervisor_start_timeout_bounded":
			start, startOK := values[KeyRuntimeCodexSupervisorStartTimeout].value.(time.Duration)
			runtimeTimeout, runtimeOK := values[KeyRuntimeCodexTimeout].value.(time.Duration)
			shutdownTimeout, shutdownOK := values[KeyServerShutdownTimeout].value.(time.Duration)
			maximum, maximumErr := time.ParseDuration(validator.MaximumDuration)
			if !startOK || !runtimeOK || !shutdownOK || maximumErr != nil ||
				start <= 0 || start > maximum ||
				start >= runtimeTimeout || start >= shutdownTimeout {
				return fail(validator.ID)
			}
		case "runtime_codex_account_profiles_complete":
			isolation, isolationOK := values[KeyRuntimeIsolation].value.(string)
			root, rootOK := values[KeyRuntimeCodexAccountHomeRoot].value.(string)
			profile, profileOK := values[KeyRuntimeCodexAccountProfile].value.(string)
			profiles, profilesOK := values[KeyRuntimeCodexAccountProfiles].value.([]string)
			maxConcurrent, concurrentOK := values[KeyRuntimeCodexMaxConcurrentExecutions].value.(int64)
			credentialRef, credentialOK := values[KeyRuntimeCodexCredentialRef].value.(CredentialRef)
			hasProfile := profile != ""
			hasProfiles := len(profiles) > 0
			if !isolationOK || !rootOK || !profileOK || !profilesOK || !concurrentOK || !credentialOK ||
				(root != "") != (hasProfile || hasProfiles) ||
				hasProfile && hasProfiles ||
				hasProfile && (maxConcurrent != 1 || !validCodexAccountProfile(profile)) ||
				hasProfiles && !validCodexAccountProfiles(profiles) ||
				(hasProfile || hasProfiles) && credentialRef != "" && isolation != "microvm" {
				return fail(validator.ID)
			}
		case "runtime_microvm_requirements":
			valid := validRuntimeMicroVMValues(values)
			if profile == resolveProfileCredentialProvisioning {
				valid = validRuntimeMicroVMCredentialProvisioningValues(values)
			} else if profile == resolveProfileContinuationCredentialProvisioning {
				valid = validRuntimeMicroVMContinuationCredentialProvisioningValues(values)
			}
			if !valid {
				return fail(validator.ID)
			}
		case "agent_firecracker_vsock_cid_lease_bounds":
			minimum, minimumOK := values[KeyAgentFirecrackerVsockCIDMinimumLeaseDuration].value.(time.Duration)
			maximum, maximumOK := values[KeyAgentFirecrackerVsockCIDMaximumLeaseDuration].value.(time.Duration)
			canonicalMaximum, maximumErr := time.ParseDuration(validator.MaximumDuration)
			if !minimumOK || !maximumOK || maximumErr != nil ||
				minimum <= 0 || maximum < minimum || maximum > canonicalMaximum {
				return fail(validator.ID)
			}
		case "identity_provider_requirements":
			provider, providerOK := values[KeyIdentityProvider].value.(string)
			issuer, issuerOK := values[KeyIdentityOIDCIssuer].value.(string)
			audience, audienceOK := values[KeyIdentityOIDCAudience].value.(string)
			clockSkew, skewOK := values[KeyIdentityOIDCClockSkew].value.(time.Duration)
			upstreamTimeout, timeoutOK := values[KeyIdentityOIDCUpstreamTimeout].value.(time.Duration)
			if !providerOK || !issuerOK || !audienceOK || !skewOK || !timeoutOK ||
				clockSkew <= 0 || clockSkew > 5*time.Minute ||
				upstreamTimeout <= 0 || upstreamTimeout > 30*time.Second {
				return fail(validator.ID)
			}
			if provider == "oidc" && (!canonicalOIDCIssuer(issuer) || audience == "") {
				return fail(validator.ID)
			}
		case "test_attestor_provider_requirements":
			if !validTestAttestorValues(values, validator) {
				return fail(validator.ID)
			}
		default:
			return fail(validator.ID)
		}
	}
	return nil
}

func isStateMigrationConfigKey(key Key) bool {
	switch key {
	case KeyStateSQLitePath, KeyStateSQLiteBusyTimeout, KeyStateSQLiteMaxOpenConnections:
		return true
	default:
		return false
	}
}

func validRuntimeMicroVMValues(values map[Key]resolvedValue) bool {
	return validRuntimeMicroVMValuesWithOptions(values, true, true)
}

func validRuntimeMicroVMCredentialProvisioningValues(values map[Key]resolvedValue) bool {
	return validRuntimeMicroVMValuesWithOptions(values, false, true)
}

func validRuntimeMicroVMContinuationCredentialProvisioningValues(values map[Key]resolvedValue) bool {
	return validRuntimeMicroVMValuesWithOptions(values, true, false)
}

func validRuntimeMicroVMValuesWithOptions(
	values map[Key]resolvedValue,
	requireProfileDescriptor bool,
	requireContinuationPublicKeyDigest bool,
) bool {
	provider, providerOK := values[KeyRuntimeProvider].value.(string)
	isolation, isolationOK := values[KeyRuntimeIsolation].value.(string)
	providerModel, modelOK := values[KeyRuntimeCodexModel].value.(string)
	providerCredentialRef, providerCredentialOK := values[KeyRuntimeCodexCredentialRef].value.(CredentialRef)
	placementRef, placementOK := values[KeyRuntimeMicroVMPlacementRef].value.(string)
	socketPath, socketOK := values[KeyRuntimeMicroVMSocketPath].value.(string)
	profilePath, profileOK := values[KeyRuntimeMicroVMProfileDescriptorPath].value.(string)
	profileDigest, digestOK := values[KeyRuntimeMicroVMExpectedProfileDescriptorSHA256].value.(string)
	keyID, keyIDOK := values[KeyRuntimeMicroVMLaunchGrantKeyID].value.(string)
	credentialRef, credentialOK := values[KeyRuntimeMicroVMLaunchGrantSigningCredentialRef].value.(CredentialRef)
	continuationCredentialRef, continuationCredentialOK := values[KeyRuntimeMicroVMExpiredLaunchContinuationAuthoritySigningCredentialRef].value.(CredentialRef)
	continuationKeyID, continuationKeyIDOK := values[KeyRuntimeMicroVMExpiredLaunchContinuationAuthorityKeyID].value.(string)
	continuationKeyEpoch, continuationKeyEpochOK := values[KeyRuntimeMicroVMExpiredLaunchContinuationAuthorityKeyEpoch].value.(int64)
	continuationTrustRevision, continuationTrustRevisionOK := values[KeyRuntimeMicroVMExpiredLaunchContinuationAuthorityTrustRevision].value.(int64)
	continuationPublicKeySHA256, continuationPublicKeyOK := values[KeyRuntimeMicroVMExpiredLaunchContinuationAuthorityPublicKeySHA256].value.(string)
	continuationValidity, continuationValidityOK := values[KeyRuntimeMicroVMExpiredLaunchContinuationAuthorityValidity].value.(time.Duration)
	brokerSocketPath, brokerSocketOK := values[KeyRuntimeMicroVMCredentialBrokerSocketPath].value.(string)
	brokerPeerUID, brokerPeerUIDOK := values[KeyRuntimeMicroVMCredentialBrokerPeerUID].value.(int64)
	brokerExchangeTimeout, brokerTimeoutOK := values[KeyRuntimeMicroVMCredentialBrokerExchangeTimeout].value.(time.Duration)
	brokerMaxConnections, brokerMaxConnectionsOK := values[KeyRuntimeMicroVMCredentialBrokerMaxConnections].value.(int64)
	egressPolicyRef, egressPolicyRefOK := values[KeyRuntimeMicroVMEgressPolicyRef].value.(string)
	egressPolicyPath, egressPolicyPathOK := values[KeyRuntimeMicroVMEgressPolicyPath].value.(string)
	egressPolicyDigest, egressPolicyDigestOK := values[KeyRuntimeMicroVMEgressPolicyExpectedSHA256].value.(string)
	egressPolicyMaxBytes, egressPolicyMaxBytesOK := values[KeyRuntimeMicroVMEgressPolicyMaxBytes].value.(int64)
	if !providerOK || !isolationOK || !modelOK || !providerCredentialOK || !placementOK || !socketOK ||
		!profileOK || !digestOK || !keyIDOK || !credentialOK || !continuationCredentialOK ||
		!continuationKeyIDOK || !continuationKeyEpochOK || !continuationTrustRevisionOK ||
		!continuationPublicKeyOK || !continuationValidityOK || !brokerSocketOK || !brokerPeerUIDOK ||
		!brokerTimeoutOK || !brokerMaxConnectionsOK || !egressPolicyRefOK || !egressPolicyPathOK ||
		!egressPolicyDigestOK || !egressPolicyMaxBytesOK {
		return false
	}
	egressPolicyConfigured := egressPolicyRef != "" || egressPolicyPath != "" || egressPolicyDigest != "" ||
		egressPolicyMaxBytes != 64<<10
	configured := placementRef != "" || socketPath != "" || profilePath != "" || profileDigest != "" ||
		keyID != "" || credentialRef != "" || brokerSocketPath != "" || brokerPeerUID != 0 ||
		continuationCredentialRef != "" || continuationKeyID != "" || continuationKeyEpoch != 0 ||
		continuationTrustRevision != 0 || continuationPublicKeySHA256 != "" || continuationValidity != 2*time.Minute ||
		brokerExchangeTimeout != 30*time.Second || brokerMaxConnections != 16 || egressPolicyConfigured
	if isolation == "process" {
		return !configured
	}
	validEgressPolicy := !egressPolicyConfigured ||
		(validRuntimeMicroVMEgressPolicyRef(egressPolicyRef) && canonicalAbsolutePath(egressPolicyPath) &&
			validBareSHA256(egressPolicyDigest) && egressPolicyMaxBytes > 0 && egressPolicyMaxBytes <= 64<<10)
	validProfileDescriptor := canonicalAbsolutePath(profilePath) && validBareSHA256(profileDigest)
	if !requireProfileDescriptor && profilePath == "" && profileDigest == "" {
		validProfileDescriptor = true
	}
	validContinuationPublicKeyDigest := validBareSHA256(continuationPublicKeySHA256)
	if !requireContinuationPublicKeyDigest && continuationPublicKeySHA256 == "" {
		validContinuationPublicKeyDigest = true
	}
	return isolation == "microvm" && provider == "codex" && validEgressPolicy &&
		validProviderModel(providerModel) &&
		providerCredentialRef != "" && providerCredentialRef != credentialRef &&
		validRuntimeMicroVMPlacementRef(placementRef) &&
		canonicalAbsolutePath(socketPath) &&
		validProfileDescriptor &&
		validLaunchGrantKeyID(keyID) &&
		credentialRef != "" && continuationCredentialRef != "" &&
		continuationCredentialRef != providerCredentialRef && continuationCredentialRef != credentialRef &&
		validExpiredLaunchContinuationKeyID(continuationKeyID) && continuationKeyEpoch > 0 &&
		continuationTrustRevision > 0 && validContinuationPublicKeyDigest &&
		continuationValidity > 0 && continuationValidity <= 5*time.Minute &&
		canonicalAbsolutePath(brokerSocketPath) && brokerSocketPath != socketPath &&
		brokerExchangeTimeout > 0 && brokerMaxConnections > 0
}

func validRuntimeMicroVMEgressPolicyRef(value string) bool {
	return value != "" && len(value) <= 512 && value == strings.TrimSpace(value) &&
		!strings.ContainsRune(value, '\x00') && utf8.ValidString(value)
}

func validProviderModel(value string) bool {
	if value == "" || len(value) > 128 || value != strings.TrimSpace(value) || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

func validRuntimeMicroVMPlacementRef(value string) bool {
	const prefix = "placement:"
	return strings.HasPrefix(value, prefix) && len(value) > len(prefix) &&
		strings.TrimSpace(value) == value && !strings.ContainsRune(value, '\x00')
}

func validLaunchGrantKeyID(value string) bool {
	const prefix = "clave-publica:"
	if !strings.HasPrefix(value, prefix) || len(value) > 160 {
		return false
	}
	suffix := strings.TrimPrefix(value, prefix)
	if suffix == "" {
		return false
	}
	for _, character := range []byte(suffix) {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' ||
			character == '-' || character == '_' {
			continue
		}
		return false
	}
	return true
}

func validExpiredLaunchContinuationKeyID(value string) bool {
	const prefix = "continuation-"
	if !strings.HasPrefix(value, prefix) || len(value) > 128 {
		return false
	}
	suffix := strings.TrimPrefix(value, prefix)
	if suffix == "" {
		return false
	}
	for _, character := range []byte(suffix) {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' ||
			character == '-' || character == '_' {
			continue
		}
		return false
	}
	return true
}

func validTestAttestorValues(
	values map[Key]resolvedValue,
	policy registryCrossValidatorDefinition,
) bool {
	provider, providerOK := values[KeyTestAttestorProvider].value.(string)
	command, commandOK := values[KeyTestAttestorBubblewrapCommand].value.(string)
	launcherSocket, launcherSocketOK := values[KeyTestAttestorMicroVMLauncherSocket].value.(string)
	expectedAssetDigest, assetDigestOK := values[KeyTestAttestorMicroVMExpectedAssetDigest].value.(string)
	guestMemoryMiB, guestMemoryOK := values[KeyTestAttestorMicroVMGuestMemoryMiB].value.(int64)
	toolchain, toolchainOK := values[KeyTestAttestorGoToolchainRoot].value.(string)
	maxOutput, outputOK := values[KeyRuntimeMaxOutputBytes].value.(int64)
	maxSubject, subjectOK := values[KeyTestAttestorMaxSubjectBytes].value.(int64)
	timeout, timeoutOK := values[KeyTestAttestorTimeout].value.(time.Duration)
	maxConcurrent, concurrentOK := values[KeyTestAttestorMaxConcurrentRuns].value.(int64)
	seedPath, seedOK := values[KeyRepositoryLocalSeedPath].value.(string)
	cgroupRoot, cgroupOK := values[KeyTestAttestorCgroupRoot].value.(string)
	memory, memoryOK := values[KeyTestAttestorMemoryMaxBytes].value.(int64)
	pids, pidsOK := values[KeyTestAttestorPIDsMax].value.(int64)
	quota, quotaOK := values[KeyTestAttestorCPUQuotaMicros].value.(int64)
	cleanup, cleanupOK := values[KeyServerShutdownTimeout].value.(time.Duration)
	attestLease, leaseOK := values[KeySchedulerAttestTestClaimLease].value.(time.Duration)
	executionTimeout, executionOK := values[KeySchedulerExecutionTimeout].value.(time.Duration)
	allTyped := providerOK && commandOK && launcherSocketOK && assetDigestOK && guestMemoryOK && toolchainOK &&
		outputOK && subjectOK && timeoutOK && concurrentOK && seedOK && cgroupOK && memoryOK &&
		pidsOK && quotaOK && cleanupOK && leaseOK && executionOK
	if !allTyped {
		return false
	}
	if provider == "disabled" {
		return true
	}
	common := seedPath != "" && maxOutput > 0 && maxSubject > 0 && maxConcurrent > 0 &&
		memory > maxOutput && pids > 0 && quota > 0 && timeout > 0 && cleanup > 0 &&
		timeout < attestLease && cleanup < attestLease-timeout && attestLease < executionTimeout
	if provider == "bubblewrap" {
		return canonicalAbsolutePath(command) && canonicalAbsolutePath(toolchain) &&
			canonicalAbsolutePath(cgroupRoot) && common
	}
	if provider == "microvm" {
		return canonicalAbsolutePath(launcherSocket) &&
			validBareSHA256(expectedAssetDigest) &&
			quota <= policy.MicroVMMaxCPUQuotaMicros &&
			validMicroVMCapacity(guestMemoryMiB, memory, maxSubject, maxOutput, policy) && common
	}
	return false
}

func validBareSHA256(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validMicroVMCapacity(
	guestMemoryMiB, memoryMaxBytes, maxSubjectBytes, maxOutputBytes int64,
	policy registryCrossValidatorDefinition,
) bool {
	if guestMemoryMiB < policy.MicroVMMinimumGuestMemoryMiB ||
		memoryMaxBytes <= 0 || maxSubjectBytes <= 0 || maxOutputBytes <= 0 {
		return false
	}
	guestBytes, ok := checkedMultiplyNonNegative(guestMemoryMiB, 1<<20)
	if !ok {
		return false
	}
	minimumCgroupBytes, ok := checkedAddNonNegative(guestBytes, policy.MicroVMCgroupHeadroomBytes)
	if !ok || memoryMaxBytes < minimumCgroupBytes {
		return false
	}
	snapshotWorkingBytes, ok := checkedMultiplyNonNegative(maxSubjectBytes, 2)
	if !ok {
		return false
	}
	snapshotAndOutputBytes, ok := checkedAddNonNegative(snapshotWorkingBytes, maxOutputBytes)
	if !ok {
		return false
	}
	minimumGuestBytes, ok := checkedAddNonNegative(snapshotAndOutputBytes, policy.MicroVMOperationalReserveBytes)
	return ok && guestBytes >= minimumGuestBytes
}

func checkedMultiplyNonNegative(left, right int64) (int64, bool) {
	if left < 0 || right < 0 || left != 0 && right > math.MaxInt64/left {
		return 0, false
	}
	return left * right, true
}

func checkedAddNonNegative(left, right int64) (int64, bool) {
	if left < 0 || right < 0 || left > math.MaxInt64-right {
		return 0, false
	}
	return left + right, true
}

func canonicalAbsolutePath(value string) bool {
	return value != "" && filepath.IsAbs(value) && filepath.Clean(value) == value
}

func canonicalOIDCIssuer(value string) bool {
	canonical, err := identity.CanonicalIssuer(value)
	return err == nil && canonical == value && strings.HasPrefix(canonical, "https://")
}

func validCodexAccountProfile(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for index, character := range value {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case index > 0 && (character == '_' || character == '-'):
		default:
			return false
		}
	}
	return true
}

func validCodexAccountProfiles(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !validCodexAccountProfile(value) {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func literalMCPPath(value string) bool {
	if !strings.HasPrefix(value, "/") || value == "/" || path.Clean(value) != value {
		return false
	}
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case strings.ContainsRune("/-._~", character):
		default:
			return false
		}
	}
	return true
}

func runtimePathsDisjoint(values map[Key]resolvedValue, sourcePath string) bool {
	canonical := func(key Key) (string, bool) {
		raw, ok := values[key].value.(string)
		if !ok || strings.TrimSpace(raw) == "" {
			return "", false
		}
		absolute, err := filepath.Abs(filepath.Clean(raw))
		return absolute, err == nil
	}
	statePath, stateOK := canonical(KeyStateSQLitePath)
	artifactRoot, artifactOK := canonical(KeyArtifactFilesystemRoot)
	credentialPath, credentialOK := canonical(KeyCredentialsLocalPath)
	workRoot, workOK := canonical(KeyRuntimeCodexWorkRoot)
	cacheRoot, cacheOK := canonical(KeyRuntimeCodexCacheRoot)
	accountHomeRoot, accountHomeOK := canonical(KeyRuntimeCodexAccountHomeRoot)
	egressPolicyPath, egressPolicyOK := canonical(KeyRuntimeMicroVMEgressPolicyPath)
	workspaceRoot, workspaceOK := canonical(KeyWorkspaceLocalRoot)
	effectivePath, effectiveOK := canonical(KeyConfigEffectivePath)
	tokenPath, tokenOK := canonical(KeyIdentityLocalTokenPath)
	manifestPath, manifestOK := canonical(KeyIdentityLocalPrincipalsManifestPath)
	manifestConfigured, _ := values[KeyIdentityLocalPrincipalsManifestPath].value.(string)
	egressPolicyConfigured, _ := values[KeyRuntimeMicroVMEgressPolicyPath].value.(string)
	if !stateOK || !artifactOK || !credentialOK || !workOK || !cacheOK ||
		!workspaceOK || !effectiveOK || !tokenOK ||
		(strings.TrimSpace(manifestConfigured) != "" && !manifestOK) ||
		(strings.TrimSpace(egressPolicyConfigured) != "" && !egressPolicyOK) {
		return false
	}
	stateDirectory, tokenDirectory := filepath.Dir(statePath), filepath.Dir(tokenPath)
	credentialRecoveryPath := credentialPath + ".next"
	pairs := [][2]string{
		{stateDirectory, artifactRoot}, {stateDirectory, workRoot}, {stateDirectory, cacheRoot},
		{artifactRoot, workRoot}, {artifactRoot, cacheRoot}, {workRoot, cacheRoot},
		{credentialPath, stateDirectory}, {credentialPath, artifactRoot}, {credentialPath, workRoot},
		{credentialPath, cacheRoot},
		{credentialPath, effectivePath}, {credentialPath, tokenPath},
		{effectivePath, statePath}, {effectivePath, artifactRoot}, {effectivePath, workRoot},
		{effectivePath, cacheRoot},
		{tokenDirectory, stateDirectory}, {tokenDirectory, artifactRoot}, {tokenDirectory, workRoot},
		{tokenDirectory, cacheRoot},
		{tokenDirectory, effectivePath},
		{workspaceRoot, stateDirectory}, {workspaceRoot, artifactRoot}, {workspaceRoot, credentialPath},
		{workspaceRoot, workRoot}, {workspaceRoot, cacheRoot}, {workspaceRoot, effectivePath},
		{workspaceRoot, tokenDirectory},
	}
	if accountHomeOK {
		pairs = append(pairs,
			[2]string{accountHomeRoot, stateDirectory},
			[2]string{accountHomeRoot, artifactRoot},
			[2]string{accountHomeRoot, credentialPath},
			[2]string{accountHomeRoot, workRoot},
			[2]string{accountHomeRoot, cacheRoot},
			[2]string{accountHomeRoot, workspaceRoot},
			[2]string{accountHomeRoot, effectivePath},
			[2]string{accountHomeRoot, tokenDirectory},
		)
	}
	if manifestOK {
		pairs = append(pairs,
			[2]string{manifestPath, stateDirectory},
			[2]string{manifestPath, artifactRoot},
			[2]string{manifestPath, credentialPath},
			[2]string{manifestPath, credentialRecoveryPath},
			[2]string{manifestPath, workRoot},
			[2]string{manifestPath, cacheRoot},
			[2]string{manifestPath, workspaceRoot},
			[2]string{manifestPath, effectivePath},
			[2]string{manifestPath, tokenPath},
		)
		if accountHomeOK {
			pairs = append(pairs, [2]string{manifestPath, accountHomeRoot})
		}
	}
	if egressPolicyOK {
		pairs = append(pairs,
			[2]string{egressPolicyPath, stateDirectory},
			[2]string{egressPolicyPath, artifactRoot},
			[2]string{egressPolicyPath, credentialPath},
			[2]string{egressPolicyPath, credentialRecoveryPath},
			[2]string{egressPolicyPath, workRoot},
			[2]string{egressPolicyPath, cacheRoot},
			[2]string{egressPolicyPath, workspaceRoot},
			[2]string{egressPolicyPath, effectivePath},
			[2]string{egressPolicyPath, tokenDirectory},
		)
		if accountHomeOK {
			pairs = append(pairs, [2]string{egressPolicyPath, accountHomeRoot})
		}
		if manifestOK {
			pairs = append(pairs, [2]string{egressPolicyPath, manifestPath})
		}
	}
	for _, pair := range pairs {
		if pathsOverlap(pair[0], pair[1]) {
			return false
		}
	}
	if strings.TrimSpace(sourcePath) != "" {
		configPath, err := filepath.Abs(filepath.Clean(sourcePath))
		if err != nil {
			return false
		}
		otherPaths := []string{
			statePath, artifactRoot, credentialPath, workRoot, cacheRoot,
			workspaceRoot, effectivePath, tokenDirectory,
		}
		if egressPolicyOK {
			otherPaths = append(otherPaths, egressPolicyPath)
		}
		if accountHomeOK {
			otherPaths = append(otherPaths, accountHomeRoot)
		}
		if manifestOK {
			otherPaths = append(otherPaths, manifestPath)
		}
		for _, other := range otherPaths {
			if pathsOverlap(configPath, other) {
				return false
			}
		}
	}
	return true
}

func pathsOverlap(left, right string) bool {
	relative, err := filepath.Rel(left, right)
	if err == nil && (relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))) {
		return true
	}
	relative, err = filepath.Rel(right, left)
	return err == nil && (relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))))
}

type snapshotHashDocument struct {
	RegistryRevision string              `json:"registry_revision"`
	RegistryHash     string              `json:"registry_hash"`
	Entries          []snapshotHashEntry `json:"entries"`
}

type snapshotHashEntry struct {
	Key    Key    `json:"key"`
	Source Source `json:"source"`
	Value  any    `json:"value"`
}

func buildSnapshot(registry registry, resolved map[Key]resolvedValue) (Snapshot, error) {
	snapshot := Snapshot{
		schemaVersion: registry.schemaVersion, registryRevision: registry.revision,
		registryHash: registry.semanticHash, entries: make([]snapshotEntry, 0, len(registry.keys)),
	}
	hashDocument := snapshotHashDocument{
		RegistryRevision: registry.revision, RegistryHash: registry.semanticHash,
		Entries: make([]snapshotHashEntry, 0, len(registry.keys)),
	}
	for _, definition := range registry.keys {
		value, found := resolved[definition.Key]
		if !found {
			return Snapshot{}, &Error{Code: ErrorRegistryInvalid, Key: definition.Key}
		}
		stored := cloneValue(value.value)
		hashValue := cloneValue(canonicalValue(stored))
		if definition.Sensitive {
			hashValue = redactedValue
		}
		snapshot.entries = append(snapshot.entries, snapshotEntry{
			key: definition.Key, value: stored,
			metadata: KeyMetadata{
				Source: value.source, Type: string(definition.Type), SemanticRef: definition.SemanticRef,
				Sensitive: definition.Sensitive, Scope: definition.Scope, RestartRequired: definition.RestartRequired,
				EnvAlias: definition.EnvAlias, ValidatorIDs: append([]string(nil), definition.ValidatorIDs...),
				AllowedValues: append([]string(nil), definition.AllowedValues...),
				Minimum:       cloneInt64Pointer(definition.Minimum), Maximum: cloneInt64Pointer(definition.Maximum),
			},
		})
		hashDocument.Entries = append(hashDocument.Entries, snapshotHashEntry{
			Key: definition.Key, Source: value.source, Value: cloneValue(hashValue),
		})
	}

	hashPayload, err := json.Marshal(hashDocument)
	if err != nil {
		return Snapshot{}, &Error{Code: ErrorRegistryInvalid, Cause: err}
	}
	digest := sha256.Sum256(hashPayload)
	snapshot.hash = "sha256:" + hex.EncodeToString(digest[:])
	return snapshot, nil
}
