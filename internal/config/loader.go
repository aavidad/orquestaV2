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
	registry, err := loadRegistry()
	if err != nil {
		return Snapshot{}, err
	}
	values, err := defaultResolvedValues(registry)
	if err != nil {
		return Snapshot{}, err
	}
	if options.TOML != nil {
		explicit, err := parseExplicitWithRegistry(options.TOML, registry)
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
	if err := applyEnvironmentMap(values, registry, options.Environment); err != nil {
		return Snapshot{}, err
	}
	if err := validateCrossRegistryValues(registry, values, options.SourcePath); err != nil {
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
	normalized := make(map[Key]string, len(environment))
	for name, raw := range environment {
		key, canonical := registry.environmentTargets[name]
		if !canonical {
			key = registry.environmentAliases[name]
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

func validateCrossRegistryValues(registry registry, values map[Key]resolvedValue, sourcePath string) error {
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
			root, rootOK := values[KeyRuntimeCodexAccountHomeRoot].value.(string)
			profile, profileOK := values[KeyRuntimeCodexAccountProfile].value.(string)
			maxConcurrent, concurrentOK := values[KeyRuntimeCodexMaxConcurrentExecutions].value.(int64)
			credentialRef, credentialOK := values[KeyRuntimeCodexCredentialRef].value.(CredentialRef)
			if !rootOK || !profileOK || !concurrentOK || !credentialOK ||
				(root == "") != (profile == "") ||
				profile != "" && (maxConcurrent != 1 || credentialRef != "" || !validCodexAccountProfile(profile)) {
				return fail(validator.ID)
			}
		case "agent_firecracker_vsock_cid_lease_bounds":
			minimum, minimumOK := values[KeyAgentFirecrackerVsockCIDMinimumLeaseDuration].value.(time.Duration)
			maximum, maximumOK := values[KeyAgentFirecrackerVsockCIDMaximumLeaseDuration].value.(time.Duration)
			if !minimumOK || !maximumOK || minimum <= 0 || maximum < minimum {
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
	accountHomeRoot, accountHomeOK := canonical(KeyRuntimeCodexAccountHomeRoot)
	workspaceRoot, workspaceOK := canonical(KeyWorkspaceLocalRoot)
	effectivePath, effectiveOK := canonical(KeyConfigEffectivePath)
	tokenPath, tokenOK := canonical(KeyIdentityLocalTokenPath)
	if !stateOK || !artifactOK || !credentialOK || !workOK || !workspaceOK || !effectiveOK || !tokenOK {
		return false
	}
	stateDirectory, tokenDirectory := filepath.Dir(statePath), filepath.Dir(tokenPath)
	pairs := [][2]string{
		{stateDirectory, artifactRoot}, {stateDirectory, workRoot}, {artifactRoot, workRoot},
		{credentialPath, stateDirectory}, {credentialPath, artifactRoot}, {credentialPath, workRoot},
		{credentialPath, effectivePath}, {credentialPath, tokenPath},
		{effectivePath, statePath}, {effectivePath, artifactRoot}, {effectivePath, workRoot},
		{tokenDirectory, stateDirectory}, {tokenDirectory, artifactRoot}, {tokenDirectory, workRoot},
		{tokenDirectory, effectivePath},
		{workspaceRoot, stateDirectory}, {workspaceRoot, artifactRoot}, {workspaceRoot, credentialPath},
		{workspaceRoot, workRoot}, {workspaceRoot, effectivePath}, {workspaceRoot, tokenDirectory},
	}
	if accountHomeOK {
		pairs = append(pairs,
			[2]string{accountHomeRoot, stateDirectory},
			[2]string{accountHomeRoot, artifactRoot},
			[2]string{accountHomeRoot, credentialPath},
			[2]string{accountHomeRoot, workRoot},
			[2]string{accountHomeRoot, workspaceRoot},
			[2]string{accountHomeRoot, effectivePath},
			[2]string{accountHomeRoot, tokenDirectory},
		)
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
		otherPaths := []string{statePath, artifactRoot, credentialPath, workRoot, workspaceRoot, effectivePath, tokenDirectory}
		if accountHomeOK {
			otherPaths = append(otherPaths, accountHomeRoot)
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
