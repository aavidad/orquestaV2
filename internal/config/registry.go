package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type valueType string

const (
	valueTypeString        valueType = "string"
	valueTypeInteger       valueType = "integer"
	valueTypeDuration      valueType = "duration"
	valueTypePath          valueType = "path"
	valueTypeOptionalPath  valueType = "optional_path"
	valueTypeStringList    valueType = "string_list"
	valueTypeCredentialRef valueType = "credential_ref"
)

type registryFile struct {
	SchemaVersion   int                                `json:"schema_version"`
	Revision        string                             `json:"revision"`
	Precedence      []Source                           `json:"precedence"`
	DocumentLimits  registryDocumentLimits             `json:"document_limits"`
	Aliases         []registryAliasDefinition          `json:"aliases"`
	CrossValidators []registryCrossValidatorDefinition `json:"cross_validators"`
	Keys            []registryKeyDefinition            `json:"keys"`
}

type registryDocumentLimits struct {
	SourceMaxBytes     int64 `json:"source_max_bytes"`
	AuditEntryMaxBytes int64 `json:"audit_entry_max_bytes"`
}

type registryAliasDefinition struct {
	Kind                AliasKind `json:"kind"`
	Name                string    `json:"name"`
	Target              Key       `json:"target"`
	IntroducedRevision  string    `json:"introduced_revision"`
	RemoveAfterRevision string    `json:"remove_after_revision"`
}

type registryCrossValidatorDefinition struct {
	ID                             string `json:"id"`
	Keys                           []Key  `json:"keys"`
	MaximumDuration                string `json:"maximum_duration,omitempty"`
	MicroVMMinimumGuestMemoryMiB   int64  `json:"microvm_minimum_guest_memory_mib,omitempty"`
	MicroVMCgroupHeadroomBytes     int64  `json:"microvm_cgroup_headroom_bytes,omitempty"`
	MicroVMOperationalReserveBytes int64  `json:"microvm_operational_reserve_bytes,omitempty"`
	MicroVMMaxCPUQuotaMicros       int64  `json:"microvm_max_cpu_quota_micros,omitempty"`
}

type registryKeyDefinition struct {
	Key             Key             `json:"key"`
	GoName          string          `json:"go_name"`
	SemanticRef     string          `json:"semantic_ref"`
	Type            valueType       `json:"type"`
	Default         json.RawMessage `json:"default"`
	Sensitive       bool            `json:"sensitive"`
	Scope           string          `json:"scope"`
	RestartRequired bool            `json:"restart_required"`
	EnvAlias        string          `json:"env_alias"`
	ValidatorIDs    []string        `json:"validator_ids"`
	AllowedValues   []string        `json:"allowed_values,omitempty"`
	Minimum         *int64          `json:"minimum,omitempty"`
	Maximum         *int64          `json:"maximum,omitempty"`
}

type registry struct {
	schemaVersion      int
	revision           string
	semanticHash       string
	precedence         []Source
	documentLimits     registryDocumentLimits
	keys               []registryKeyDefinition
	aliases            []registryAliasDefinition
	crossValidators    []registryCrossValidatorDefinition
	byKey              map[Key]registryKeyDefinition
	tomlAliasTargets   map[Key]Key
	environmentTargets map[string]Key
	environmentAliases map[string]Key
	prefixes           map[string]struct{}
}

func loadRegistry() (registry, error) {
	result, err := parseRegistrySource([]byte(generatedRegistryJSON))
	if err != nil {
		return registry{}, err
	}
	if result.revision != generatedRegistryRevision || result.semanticHash != generatedRegistrySemanticSHA256 {
		return registry{}, &Error{Code: ErrorRegistryInvalid, Cause: fmt.Errorf("generated registry identity mismatch")}
	}
	return result, nil
}

func parseRegistrySource(content []byte) (registry, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var source registryFile
	if err := decoder.Decode(&source); err != nil {
		return registry{}, &Error{Code: ErrorRegistryInvalid, Cause: err}
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return registry{}, &Error{Code: ErrorRegistryInvalid, Cause: err}
	}
	return validateAndBuildRegistry(source)
}

// ValidateRegistrySource is used by the generator so runtime and generated
// artifacts share one validation authority.
func ValidateRegistrySource(content []byte) error {
	_, err := parseRegistrySource(content)
	return err
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("trailing JSON value")
	}
	return err
}

func validateAndBuildRegistry(source registryFile) (registry, error) {
	fail := func(cause error) (registry, error) {
		return registry{}, &Error{Code: ErrorRegistryInvalid, Cause: cause}
	}
	if source.SchemaVersion < 1 {
		return fail(fmt.Errorf("schema_version must be positive"))
	}
	if strings.TrimSpace(source.Revision) == "" {
		return fail(fmt.Errorf("revision is required"))
	}
	activeRevision, err := parseRegistryRevision(source.Revision)
	if err != nil {
		return fail(fmt.Errorf("revision is not canonical"))
	}
	if len(source.Keys) == 0 {
		return fail(fmt.Errorf("registry has no keys"))
	}
	wantPrecedence := []Source{SourceDefault, SourceFile, SourceEnv}
	if len(source.Precedence) != len(wantPrecedence) {
		return fail(fmt.Errorf("precedence must be default,file,env"))
	}
	for index := range wantPrecedence {
		if source.Precedence[index] != wantPrecedence[index] {
			return fail(fmt.Errorf("precedence must be default,file,env"))
		}
	}
	if source.DocumentLimits.SourceMaxBytes <= 0 || source.DocumentLimits.AuditEntryMaxBytes <= 0 {
		return fail(fmt.Errorf("document limits must be positive"))
	}

	result := registry{
		schemaVersion:      source.SchemaVersion,
		revision:           source.Revision,
		precedence:         append([]Source(nil), source.Precedence...),
		documentLimits:     source.DocumentLimits,
		keys:               cloneRegistryKeyDefinitions(source.Keys),
		aliases:            append([]registryAliasDefinition(nil), source.Aliases...),
		crossValidators:    cloneCrossValidators(source.CrossValidators),
		byKey:              make(map[Key]registryKeyDefinition, len(source.Keys)),
		tomlAliasTargets:   make(map[Key]Key, len(source.Aliases)),
		environmentTargets: make(map[string]Key, len(source.Keys)),
		environmentAliases: make(map[string]Key, len(source.Aliases)),
		prefixes:           make(map[string]struct{}),
	}
	goNames := make(map[string]struct{}, len(source.Keys))
	semanticRefs := make(map[string]struct{}, len(source.Keys))
	for _, definition := range source.Keys {
		if err := validateRegistryDefinition(definition); err != nil {
			return fail(fmt.Errorf("%s: %w", definition.Key, err))
		}
		if _, exists := result.byKey[definition.Key]; exists {
			return fail(fmt.Errorf("duplicate key"))
		}
		if _, exists := goNames[definition.GoName]; exists {
			return fail(fmt.Errorf("duplicate go name"))
		}
		if _, exists := semanticRefs[definition.SemanticRef]; exists {
			return fail(fmt.Errorf("duplicate semantic_ref"))
		}
		if _, exists := result.environmentTargets[definition.EnvAlias]; exists {
			return fail(fmt.Errorf("duplicate env alias"))
		}
		result.byKey[definition.Key] = cloneRegistryKeyDefinition(definition)
		goNames[definition.GoName] = struct{}{}
		semanticRefs[definition.SemanticRef] = struct{}{}
		result.environmentTargets[definition.EnvAlias] = definition.Key

		parts := strings.Split(string(definition.Key), ".")
		for index := 1; index < len(parts); index++ {
			result.prefixes[strings.Join(parts[:index], ".")] = struct{}{}
		}
	}
	for _, alias := range source.Aliases {
		if _, targetExists := result.byKey[alias.Target]; !targetExists {
			return fail(fmt.Errorf("alias target is not canonical"))
		}
		introduced, introducedErr := parseRegistryRevision(alias.IntroducedRevision)
		removeAfter, removeErr := parseRegistryRevision(alias.RemoveAfterRevision)
		if introducedErr != nil || removeErr != nil || introduced.compare(removeAfter) >= 0 {
			return fail(fmt.Errorf("alias requires ordered introduction and removal revisions"))
		}
		if activeRevision.compare(introduced) < 0 || activeRevision.compare(removeAfter) >= 0 {
			return fail(fmt.Errorf("alias is not active for registry revision"))
		}
		switch alias.Kind {
		case AliasKindTOMLKey:
			name := Key(alias.Name)
			if !validCanonicalKey(alias.Name) || name == alias.Target {
				return fail(fmt.Errorf("invalid TOML alias"))
			}
			if _, collision := result.byKey[name]; collision {
				return fail(fmt.Errorf("TOML alias collides with canonical key"))
			}
			if _, duplicate := result.tomlAliasTargets[name]; duplicate {
				return fail(fmt.Errorf("duplicate TOML alias"))
			}
			result.tomlAliasTargets[name] = alias.Target
			parts := strings.Split(alias.Name, ".")
			for index := 1; index < len(parts); index++ {
				result.prefixes[strings.Join(parts[:index], ".")] = struct{}{}
			}
		case AliasKindEnvironment:
			if !validEnvironmentName(alias.Name) {
				return fail(fmt.Errorf("invalid environment alias"))
			}
			if _, collision := result.environmentTargets[alias.Name]; collision {
				return fail(fmt.Errorf("environment alias collides with canonical environment"))
			}
			if _, duplicate := result.environmentAliases[alias.Name]; duplicate {
				return fail(fmt.Errorf("duplicate environment alias"))
			}
			result.environmentAliases[alias.Name] = alias.Target
		default:
			return fail(fmt.Errorf("unsupported alias kind"))
		}
	}
	if err := validateCrossValidators(source.CrossValidators, result.byKey); err != nil {
		return fail(err)
	}
	semanticPayload, err := json.Marshal(source)
	if err != nil {
		return fail(err)
	}
	digest := sha256.Sum256(semanticPayload)
	result.semanticHash = "sha256:" + hex.EncodeToString(digest[:])
	return result, nil
}

func validateRegistryDefinition(definition registryKeyDefinition) error {
	if !validCanonicalKey(string(definition.Key)) {
		return fmt.Errorf("invalid canonical key")
	}
	if strings.TrimSpace(definition.GoName) == "" || strings.TrimSpace(definition.Scope) == "" {
		return fmt.Errorf("go_name and scope are required")
	}
	if definition.SemanticRef != "orquesta.config."+string(definition.Key) {
		return fmt.Errorf("semantic_ref must be stable and canonical")
	}
	if !validEnvironmentName(definition.EnvAlias) {
		return fmt.Errorf("env_alias is invalid")
	}
	switch definition.Type {
	case valueTypeString, valueTypeInteger, valueTypeDuration, valueTypePath, valueTypeOptionalPath,
		valueTypeStringList, valueTypeCredentialRef:
	default:
		return fmt.Errorf("unsupported value type")
	}
	if definition.Type != valueTypeInteger && (definition.Minimum != nil || definition.Maximum != nil) {
		return fmt.Errorf("bounds require integer type")
	}
	if definition.Minimum != nil && definition.Maximum != nil && *definition.Minimum > *definition.Maximum {
		return fmt.Errorf("minimum exceeds maximum")
	}
	if definition.Sensitive != (definition.Type == valueTypeCredentialRef) {
		return fmt.Errorf("only credential references may be sensitive")
	}
	if err := validateValidatorIDs(definition); err != nil {
		return err
	}
	seenAllowed := make(map[string]struct{}, len(definition.AllowedValues))
	for _, allowed := range definition.AllowedValues {
		if allowed == "" {
			return fmt.Errorf("allowed value is empty")
		}
		if _, duplicate := seenAllowed[allowed]; duplicate {
			return fmt.Errorf("duplicate allowed value")
		}
		seenAllowed[allowed] = struct{}{}
	}
	value, err := parseDefaultValue(definition)
	if err != nil {
		return err
	}
	return validateAllowedValue(definition, value)
}

func requiredValidatorIDs(definition registryKeyDefinition) []string {
	result := make([]string, 0, 2)
	switch definition.Type {
	case valueTypeInteger:
		result = append(result, "integer_bounds")
	case valueTypeDuration:
		result = append(result, "positive_duration")
	case valueTypePath:
		result = append(result, "non_empty_path")
	case valueTypeOptionalPath:
		result = append(result, "optional_path")
	case valueTypeStringList:
		result = append(result, "unique_non_empty_string_list")
	case valueTypeCredentialRef:
		result = append(result, "credential_ref")
	}
	if len(definition.AllowedValues) > 0 {
		result = append(result, "allowed_values")
	}
	return result
}

func validateValidatorIDs(definition registryKeyDefinition) error {
	required := requiredValidatorIDs(definition)
	if len(definition.ValidatorIDs) < len(required) {
		return fmt.Errorf("validator_ids omit required type constraint")
	}
	seen := make(map[string]struct{}, len(definition.ValidatorIDs))
	for index, validator := range definition.ValidatorIDs {
		if _, duplicate := seen[validator]; duplicate {
			return fmt.Errorf("validator_ids contain duplicate")
		}
		seen[validator] = struct{}{}
		if index < len(required) {
			if validator != required[index] {
				return fmt.Errorf("validator_ids do not match type and constraints")
			}
			continue
		}
		switch validator {
		case "trimmed_non_empty_string", "trimmed_optional_string", "opaque_ref", "environment_name":
			if definition.Type != valueTypeString {
				return fmt.Errorf("string validator requires string type")
			}
		case "optional_path":
			if definition.Type != valueTypeOptionalPath {
				return fmt.Errorf("optional_path validator requires optional_path type")
			}
		case "environment_name_list":
			if definition.Type != valueTypeStringList {
				return fmt.Errorf("environment_name_list requires string_list type")
			}
		default:
			return fmt.Errorf("unsupported validator_id")
		}
	}
	if _, nonEmpty := seen["trimmed_non_empty_string"]; nonEmpty {
		if _, optional := seen["trimmed_optional_string"]; optional {
			return fmt.Errorf("conflicting string validators")
		}
	}
	return nil
}

type registryRevision struct {
	date     string
	sequence int
}

func parseRegistryRevision(value string) (registryRevision, error) {
	separator := strings.LastIndexByte(value, '.')
	if separator <= 0 || separator == len(value)-1 {
		return registryRevision{}, fmt.Errorf("invalid revision")
	}
	date := value[:separator]
	if parsed, err := time.Parse("2006-01-02", date); err != nil || parsed.Format("2006-01-02") != date {
		return registryRevision{}, fmt.Errorf("invalid revision date")
	}
	sequenceText := value[separator+1:]
	sequence, err := strconv.Atoi(sequenceText)
	if err != nil || sequence < 0 || strconv.Itoa(sequence) != sequenceText {
		return registryRevision{}, fmt.Errorf("invalid revision sequence")
	}
	return registryRevision{date: date, sequence: sequence}, nil
}

func (revision registryRevision) compare(other registryRevision) int {
	if revision.date < other.date {
		return -1
	}
	if revision.date > other.date {
		return 1
	}
	if revision.sequence < other.sequence {
		return -1
	}
	if revision.sequence > other.sequence {
		return 1
	}
	return 0
}

func validCanonicalKey(key string) bool {
	if key == "" || strings.HasPrefix(key, ".") || strings.HasSuffix(key, ".") || strings.Contains(key, "..") {
		return false
	}
	for _, character := range key {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '_' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func validEnvironmentName(name string) bool {
	if name == "" || name[0] < 'A' || name[0] > 'Z' {
		return false
	}
	for _, character := range name {
		if (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '_' {
			continue
		}
		return false
	}
	return true
}

func validateCrossValidators(definitions []registryCrossValidatorDefinition, keys map[Key]registryKeyDefinition) error {
	expected := map[string][]Key{
		"runtime_codex_timeout_before_scheduler_execution_timeout": {
			KeyRuntimeCodexTimeout, KeySchedulerExecutionTimeout,
		},
		"runtime_codex_supervisor_start_timeout_bounded": {
			KeyRuntimeCodexSupervisorStartTimeout, KeyRuntimeCodexTimeout, KeyServerShutdownTimeout,
		},
		"runtime_codex_account_profiles_complete": {
			KeyRuntimeCodexAccountHomeRoot, KeyRuntimeCodexAccountProfile, KeyRuntimeCodexMaxConcurrentExecutions,
			KeyRuntimeCodexCredentialRef,
		},
		"server_listen_loopback":  {KeyServerListen},
		"server_mcp_path_literal": {KeyServerMCPPath},
		"runtime_paths_disjoint": {
			KeyStateSQLitePath, KeyArtifactFilesystemRoot, KeyCredentialsLocalPath, KeyRuntimeCodexWorkRoot,
			KeyRuntimeCodexAccountHomeRoot, KeyWorkspaceLocalRoot,
			KeyConfigEffectivePath, KeyIdentityLocalTokenPath,
		},
		"identity_provider_requirements": {
			KeyIdentityProvider, KeyIdentityOIDCIssuer, KeyIdentityOIDCAudience, KeyIdentityOIDCClockSkew,
			KeyIdentityOIDCUpstreamTimeout,
		},
		"test_attestor_provider_requirements": {
			KeyTestAttestorProvider, KeyTestAttestorTimeout, KeyTestAttestorBubblewrapCommand,
			KeyTestAttestorMicroVMLauncherSocket, KeyTestAttestorMicroVMGuestMemoryMiB,
			KeyTestAttestorGoToolchainRoot,
			KeyRuntimeMaxOutputBytes, KeyTestAttestorMaxSubjectBytes,
			KeyTestAttestorMaxConcurrentRuns, KeyRepositoryLocalSeedPath,
			KeyTestAttestorCgroupRoot, KeyTestAttestorMemoryMaxBytes,
			KeyTestAttestorPIDsMax, KeyTestAttestorCPUQuotaMicros, KeyServerShutdownTimeout,
			KeySchedulerAttestTestClaimLease, KeySchedulerExecutionTimeout,
		},
	}
	if len(definitions) != len(expected) {
		return fmt.Errorf("cross validators are incomplete")
	}
	seen := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		want, supported := expected[definition.ID]
		if !supported {
			return fmt.Errorf("unsupported cross validator")
		}
		if _, duplicate := seen[definition.ID]; duplicate {
			return fmt.Errorf("duplicate cross validator")
		}
		seen[definition.ID] = struct{}{}
		seenKeys := make(map[Key]struct{}, len(definition.Keys))
		for _, key := range definition.Keys {
			if _, found := keys[key]; !found {
				return fmt.Errorf("cross validator references unknown key")
			}
			if _, duplicate := seenKeys[key]; duplicate {
				return fmt.Errorf("cross validator repeats key")
			}
			seenKeys[key] = struct{}{}
		}
		if len(definition.Keys) != len(want) {
			return fmt.Errorf("cross validator key set mismatch")
		}
		for index := range want {
			if definition.Keys[index] != want[index] {
				return fmt.Errorf("cross validator key order mismatch")
			}
		}
		requiresMaximumDuration := definition.ID == "runtime_codex_supervisor_start_timeout_bounded"
		if requiresMaximumDuration != (definition.MaximumDuration != "") {
			return fmt.Errorf("cross validator duration bound mismatch")
		}
		if definition.MaximumDuration != "" {
			maximum, err := time.ParseDuration(definition.MaximumDuration)
			if err != nil || maximum <= 0 {
				return fmt.Errorf("invalid cross validator duration bound")
			}
		}
		requiresMicroVMPolicy := definition.ID == "test_attestor_provider_requirements"
		hasMicroVMPolicy := definition.MicroVMMinimumGuestMemoryMiB != 0 ||
			definition.MicroVMCgroupHeadroomBytes != 0 ||
			definition.MicroVMOperationalReserveBytes != 0 ||
			definition.MicroVMMaxCPUQuotaMicros != 0
		if requiresMicroVMPolicy != hasMicroVMPolicy {
			return fmt.Errorf("cross validator microVM policy mismatch")
		}
		if !requiresMicroVMPolicy {
			continue
		}
		if definition.MicroVMMinimumGuestMemoryMiB <= 0 ||
			definition.MicroVMCgroupHeadroomBytes <= 0 ||
			definition.MicroVMOperationalReserveBytes <= 0 ||
			definition.MicroVMMaxCPUQuotaMicros <= 0 {
			return fmt.Errorf("invalid cross validator microVM policy")
		}
		guestDefinition := keys[KeyTestAttestorMicroVMGuestMemoryMiB]
		if guestDefinition.Minimum == nil ||
			*guestDefinition.Minimum != definition.MicroVMMinimumGuestMemoryMiB {
			return fmt.Errorf("microVM minimum guest policy differs from key bound")
		}
		quotaDefinition := keys[KeyTestAttestorCPUQuotaMicros]
		if quotaDefinition.Minimum == nil || quotaDefinition.Maximum == nil ||
			definition.MicroVMMaxCPUQuotaMicros < *quotaDefinition.Minimum ||
			definition.MicroVMMaxCPUQuotaMicros > *quotaDefinition.Maximum {
			return fmt.Errorf("microVM CPU quota policy exceeds key bounds")
		}
		guestBytes, ok := checkedMultiplyNonNegative(definition.MicroVMMinimumGuestMemoryMiB, 1<<20)
		if !ok {
			return fmt.Errorf("microVM minimum guest policy overflows")
		}
		if _, ok := checkedAddNonNegative(guestBytes, definition.MicroVMCgroupHeadroomBytes); !ok {
			return fmt.Errorf("microVM cgroup policy overflows")
		}
		memoryDefinition := keys[KeyTestAttestorMemoryMaxBytes]
		subjectDefinition := keys[KeyTestAttestorMaxSubjectBytes]
		outputDefinition := keys[KeyRuntimeMaxOutputBytes]
		guestDefault, guestErr := parseDefaultValue(guestDefinition)
		memoryDefault, memoryErr := parseDefaultValue(memoryDefinition)
		subjectDefault, subjectErr := parseDefaultValue(subjectDefinition)
		outputDefault, outputErr := parseDefaultValue(outputDefinition)
		quotaDefault, quotaErr := parseDefaultValue(quotaDefinition)
		guestValue, guestOK := guestDefault.(int64)
		memoryValue, memoryOK := memoryDefault.(int64)
		subjectValue, subjectOK := subjectDefault.(int64)
		outputValue, outputOK := outputDefault.(int64)
		quotaValue, quotaOK := quotaDefault.(int64)
		if guestErr != nil || memoryErr != nil || subjectErr != nil || outputErr != nil || quotaErr != nil ||
			!guestOK || !memoryOK || !subjectOK || !outputOK || !quotaOK ||
			quotaValue > definition.MicroVMMaxCPUQuotaMicros ||
			!validMicroVMCapacity(guestValue, memoryValue, subjectValue, outputValue, definition) {
			return fmt.Errorf("microVM defaults violate canonical policy")
		}
	}
	return nil
}

func parseDefaultValue(definition registryKeyDefinition) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(definition.Default))
	decoder.UseNumber()
	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	return parseFileValue(definition, raw)
}

func (r registry) definition(key Key) (registryKeyDefinition, bool) {
	definition, found := r.byKey[key]
	return cloneRegistryKeyDefinition(definition), found
}

func (r registry) canonicalKey(key Key) (Key, bool) {
	if _, found := r.byKey[key]; found {
		return key, true
	}
	target, found := r.tomlAliasTargets[key]
	return target, found
}

func (r registry) hasPrefix(prefix string) bool {
	_, found := r.prefixes[prefix]
	return found
}

func cloneRegistryKeyDefinitions(source []registryKeyDefinition) []registryKeyDefinition {
	result := make([]registryKeyDefinition, len(source))
	for index, definition := range source {
		result[index] = cloneRegistryKeyDefinition(definition)
	}
	return result
}

func cloneRegistryKeyDefinition(definition registryKeyDefinition) registryKeyDefinition {
	definition.Default = append(json.RawMessage(nil), definition.Default...)
	definition.ValidatorIDs = append([]string(nil), definition.ValidatorIDs...)
	definition.AllowedValues = append([]string(nil), definition.AllowedValues...)
	definition.Minimum = cloneInt64Pointer(definition.Minimum)
	definition.Maximum = cloneInt64Pointer(definition.Maximum)
	return definition
}

func cloneCrossValidators(source []registryCrossValidatorDefinition) []registryCrossValidatorDefinition {
	result := make([]registryCrossValidatorDefinition, len(source))
	for index, definition := range source {
		result[index] = registryCrossValidatorDefinition{
			ID: definition.ID, Keys: append([]Key(nil), definition.Keys...),
			MaximumDuration:                definition.MaximumDuration,
			MicroVMMinimumGuestMemoryMiB:   definition.MicroVMMinimumGuestMemoryMiB,
			MicroVMCgroupHeadroomBytes:     definition.MicroVMCgroupHeadroomBytes,
			MicroVMOperationalReserveBytes: definition.MicroVMOperationalReserveBytes,
			MicroVMMaxCPUQuotaMicros:       definition.MicroVMMaxCPUQuotaMicros,
		}
	}
	return result
}

func cloneValue(value any) any {
	if list, ok := value.([]string); ok {
		return append([]string(nil), list...)
	}
	return value
}

// Definitions returns detached canonical definitions in registry order.
func Definitions() []KeyDefinition {
	loaded, err := loadRegistry()
	if err != nil {
		return nil
	}
	result := make([]KeyDefinition, 0, len(loaded.keys))
	for _, definition := range loaded.keys {
		value, _ := parseDefaultValue(definition)
		result = append(result, publicDefinition(definition, value))
	}
	return result
}

// Definition returns one detached canonical definition.
func Definition(key Key) (KeyDefinition, bool) {
	loaded, err := loadRegistry()
	if err != nil {
		return KeyDefinition{}, false
	}
	definition, found := loaded.definition(key)
	if !found {
		return KeyDefinition{}, false
	}
	value, err := parseDefaultValue(definition)
	if err != nil {
		return KeyDefinition{}, false
	}
	return publicDefinition(definition, value), true
}

func publicDefinition(definition registryKeyDefinition, defaultValue any) KeyDefinition {
	return KeyDefinition{
		Key: definition.Key, GoName: definition.GoName, SemanticRef: definition.SemanticRef,
		Type: string(definition.Type), Default: cloneValue(canonicalValue(defaultValue)),
		Sensitive: definition.Sensitive, Scope: definition.Scope, RestartRequired: definition.RestartRequired,
		EnvAlias: definition.EnvAlias, ValidatorIDs: append([]string(nil), definition.ValidatorIDs...),
		AllowedValues: append([]string(nil), definition.AllowedValues...), Minimum: cloneInt64Pointer(definition.Minimum),
		Maximum: cloneInt64Pointer(definition.Maximum),
	}
}

// Aliases returns detached, time-bounded migration aliases.
func Aliases() []AliasDefinition {
	loaded, err := loadRegistry()
	if err != nil {
		return nil
	}
	result := make([]AliasDefinition, len(loaded.aliases))
	for index, alias := range loaded.aliases {
		result[index] = AliasDefinition(alias)
	}
	return result
}

// CrossValidators returns detached multi-key invariants in registry order.
func CrossValidators() []CrossValidatorDefinition {
	loaded, err := loadRegistry()
	if err != nil {
		return nil
	}
	result := make([]CrossValidatorDefinition, len(loaded.crossValidators))
	for index, definition := range loaded.crossValidators {
		result[index] = CrossValidatorDefinition{
			ID: definition.ID, Keys: append([]Key(nil), definition.Keys...),
			MaximumDuration:                definition.MaximumDuration,
			MicroVMMinimumGuestMemoryMiB:   definition.MicroVMMinimumGuestMemoryMiB,
			MicroVMCgroupHeadroomBytes:     definition.MicroVMCgroupHeadroomBytes,
			MicroVMOperationalReserveBytes: definition.MicroVMOperationalReserveBytes,
			MicroVMMaxCPUQuotaMicros:       definition.MicroVMMaxCPUQuotaMicros,
		}
	}
	return result
}

// SourceMaxBytes is the canonical maximum accepted TOML document size.
func SourceMaxBytes() int64 {
	loaded, err := loadRegistry()
	if err != nil {
		return 0
	}
	return loaded.documentLimits.SourceMaxBytes
}

// AuditEntryMaxBytes is the canonical maximum serialized config audit entry.
func AuditEntryMaxBytes() int64 {
	loaded, err := loadRegistry()
	if err != nil {
		return 0
	}
	return loaded.documentLimits.AuditEntryMaxBytes
}
