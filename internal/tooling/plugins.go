package tooling

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

const (
	PluginDescriptorFormatV1     = "plugin_descriptor_v1"
	MaxPluginsPerCatalog         = 128
	MaxPluginComponentsPerKind   = 128
	MaxPluginCapabilities        = 64
	MaxPluginScopes              = 32
	MaxPluginPermissions         = 64
	MaxPluginConfigContractBytes = 1 << 20
	MaxPluginDescriptorBytes     = 32 << 10
	ErrorPluginSpecInvalid       = "tooling.plugin_spec_invalid"
	ErrorPluginSpecDuplicate     = "tooling.plugin_spec_duplicate"
)

type PluginConnectorSpec struct {
	ID             string                `json:"id"`
	Version        string                `json:"version"`
	ContractRef    string                `json:"contract_ref"`
	ContractDigest string                `json:"contract_digest"`
	Permissions    []identity.Permission `json:"permissions"`
}

type PluginToolRequirement struct {
	ID         string `json:"id"`
	Version    string `json:"version"`
	SpecDigest string `json:"spec_digest"`
}

type PluginSkillRequirement struct {
	ID                 string `json:"id"`
	Version            string `json:"version"`
	RegistrationDigest string `json:"registration_digest"`
	ReleaseDigest      string `json:"release_digest,omitempty"`
}

type PluginConfigRequirement struct {
	ContractRef    string `json:"contract_ref"`
	ContractDigest string `json:"contract_digest"`
	SizeBytes      int64  `json:"size_bytes"`
}

type PluginSpec struct {
	Format         string                   `json:"format"`
	ID             string                   `json:"id"`
	Version        string                   `json:"version"`
	DescriptionKey string                   `json:"description_key"`
	Capabilities   []string                 `json:"capabilities"`
	Scopes         []SkillScope             `json:"scopes"`
	Config         *PluginConfigRequirement `json:"config,omitempty"`
	Connectors     []PluginConnectorSpec    `json:"connectors"`
	Tools          []PluginToolRequirement  `json:"tools"`
	Skills         []PluginSkillRequirement `json:"skills"`
	Permissions    []identity.Permission    `json:"permissions"`
}

type PluginRegistration struct {
	Spec   PluginSpec `json:"spec"`
	Digest string     `json:"digest"`
}

type PluginCatalog struct {
	entries []PluginRegistration
	byKey   map[string]int
	digest  string
}

type pluginSkillSource interface {
	Digest() string
	pluginSkill(id, version string) (SkillRegistration, string, bool, bool)
}

func (registry *SkillRegistry) pluginSkill(id, version string) (SkillRegistration, string, bool, bool) {
	registration, found := registry.Lookup(id, version)
	return registration, "", false, found
}

func (catalog *SkillReleaseCatalog) pluginSkill(id, version string) (SkillRegistration, string, bool, bool) {
	release, found := exactPluginSkillRelease(catalog, id, version)
	return release.Registration, release.ReleaseDigest, release.Revoked, found
}

func NewPluginCatalog(
	tools *Registry,
	skills pluginSkillSource,
	specs ...PluginSpec,
) (*PluginCatalog, error) {
	if tools == nil || skills == nil || skills.Digest() == "" || len(specs) > MaxPluginsPerCatalog {
		return nil, contractError(ErrorPluginSpecInvalid, "registries")
	}
	entries := make([]PluginRegistration, 0, len(specs))
	seen := make(map[string]struct{}, len(specs))
	for _, source := range specs {
		spec, err := canonicalPluginSpec(tools, skills, source)
		if err != nil {
			return nil, err
		}
		key := registryKey(spec.ID, spec.Version)
		if _, duplicate := seen[key]; duplicate {
			return nil, contractError(ErrorPluginSpecDuplicate, "identity")
		}
		seen[key] = struct{}{}
		entries = append(entries, PluginRegistration{Spec: spec, Digest: pluginSpecDigest(spec)})
	}
	sort.Slice(entries, func(left, right int) bool {
		return surfaceIdentityLess(
			entries[left].Spec.ID, entries[left].Spec.Version,
			entries[right].Spec.ID, entries[right].Spec.Version,
		)
	})
	byKey := make(map[string]int, len(entries))
	for index, entry := range entries {
		byKey[registryKey(entry.Spec.ID, entry.Spec.Version)] = index
	}
	return &PluginCatalog{
		entries: entries, byKey: byKey, digest: pluginCatalogDigest(entries),
	}, nil
}

func (catalog *PluginCatalog) Lookup(id, version string) (PluginRegistration, bool) {
	if catalog == nil {
		return PluginRegistration{}, false
	}
	index, found := catalog.byKey[registryKey(id, version)]
	if !found {
		return PluginRegistration{}, false
	}
	return clonePluginRegistration(catalog.entries[index]), true
}

func (catalog *PluginCatalog) List() []PluginRegistration {
	if catalog == nil {
		return nil
	}
	result := make([]PluginRegistration, len(catalog.entries))
	for index, entry := range catalog.entries {
		result[index] = clonePluginRegistration(entry)
	}
	return result
}

func (catalog *PluginCatalog) Digest() string {
	if catalog == nil {
		return ""
	}
	return catalog.digest
}

func canonicalPluginSpec(
	tools *Registry,
	skills pluginSkillSource,
	source PluginSpec,
) (PluginSpec, error) {
	if source.Format == "" {
		if _, compatible := skills.(*SkillReleaseCatalog); compatible {
			source.Format = PluginDescriptorFormatV1
		}
	}
	if source.Format != PluginDescriptorFormatV1 {
		return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "format")
	}
	if !validToolID(source.ID) {
		return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "id")
	}
	if _, err := parseVersion(source.Version); err != nil {
		return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "version")
	}
	if source.DescriptionKey != "plugin."+source.ID+".description" || !validToolID(source.DescriptionKey) {
		return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "description_key")
	}
	if len(source.Capabilities) > MaxPluginCapabilities || len(source.Scopes) > MaxPluginScopes ||
		len(source.Connectors) > MaxPluginComponentsPerKind || len(source.Tools) > MaxPluginComponentsPerKind ||
		len(source.Skills) > MaxPluginComponentsPerKind || len(source.Permissions) > MaxPluginPermissions {
		return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "limits")
	}
	if len(source.Connectors)+len(source.Tools)+len(source.Skills) == 0 {
		return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "components")
	}
	for _, capability := range source.Capabilities {
		if !validPluginCapability(capability) {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "capabilities")
		}
	}
	capabilities := append([]string(nil), source.Capabilities...)
	sort.Strings(capabilities)
	for index := 1; index < len(capabilities); index++ {
		if capabilities[index] == capabilities[index-1] {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "capabilities")
		}
	}
	scopes := append([]SkillScope(nil), source.Scopes...)
	if len(scopes) > 0 {
		var err error
		scopes, err = canonicalSkillScopes(scopes)
		if err != nil {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "scopes")
		}
	}
	config, err := canonicalPluginConfig(source.Config)
	if err != nil {
		return PluginSpec{}, err
	}
	permissionSet := make(map[identity.Permission]struct{})
	connectors := append([]PluginConnectorSpec(nil), source.Connectors...)
	for index := range connectors {
		connector := &connectors[index]
		if !validToolID(connector.ID) {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "connector_id")
		}
		if _, err := parseVersion(connector.Version); err != nil {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "connector_version")
		}
		if !validCanonicalPluginDigest(connector.ContractDigest) ||
			connector.ContractRef != "artifact:"+connector.ContractDigest {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "connector_contract")
		}
		permissions, err := canonicalPluginComponentPermissions(connector.Permissions)
		if err != nil {
			return PluginSpec{}, err
		}
		connector.Permissions = permissions
		for _, permission := range permissions {
			permissionSet[permission] = struct{}{}
		}
	}
	sort.Slice(connectors, func(left, right int) bool {
		return surfaceIdentityLess(
			connectors[left].ID, connectors[left].Version,
			connectors[right].ID, connectors[right].Version,
		)
	})
	if hasDuplicatePluginConnector(connectors) {
		return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "connectors")
	}

	toolRequirements := append([]PluginToolRequirement(nil), source.Tools...)
	toolKeys := make(map[string]string, len(toolRequirements))
	for _, requirement := range toolRequirements {
		if _, err := parseVersion(requirement.Version); !validToolID(requirement.ID) || err != nil ||
			!validCanonicalPluginDigest(requirement.SpecDigest) {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "tools")
		}
		toolKeys[registryKey(requirement.ID, requirement.Version)] = requirement.SpecDigest
	}
	sort.Slice(toolRequirements, func(left, right int) bool {
		return surfaceIdentityLess(
			toolRequirements[left].ID, toolRequirements[left].Version,
			toolRequirements[right].ID, toolRequirements[right].Version,
		)
	})
	for index, requirement := range toolRequirements {
		if index > 0 && registryKey(requirement.ID, requirement.Version) ==
			registryKey(toolRequirements[index-1].ID, toolRequirements[index-1].Version) {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "tools")
		}
		registration, found := tools.Lookup(requirement.ID, requirement.Version)
		if !found || registration.Digest != requirement.SpecDigest {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "tools")
		}
		for _, permission := range registration.Spec.Permissions {
			permissionSet[permission] = struct{}{}
		}
	}

	skillRequirements := append([]PluginSkillRequirement(nil), source.Skills...)
	for _, requirement := range skillRequirements {
		if _, err := parseVersion(requirement.Version); !validToolID(requirement.ID) || err != nil ||
			!validCanonicalPluginDigest(requirement.RegistrationDigest) ||
			requirement.ReleaseDigest != "" && !validCanonicalPluginDigest(requirement.ReleaseDigest) {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "skills")
		}
	}
	sort.Slice(skillRequirements, func(left, right int) bool {
		return surfaceIdentityLess(
			skillRequirements[left].ID, skillRequirements[left].Version,
			skillRequirements[right].ID, skillRequirements[right].Version,
		)
	})
	for index, requirement := range skillRequirements {
		if index > 0 && registryKey(requirement.ID, requirement.Version) ==
			registryKey(skillRequirements[index-1].ID, skillRequirements[index-1].Version) {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "skills")
		}
		registration, releaseDigest, revoked, found := skills.pluginSkill(requirement.ID, requirement.Version)
		if !found || revoked || registration.Digest != requirement.RegistrationDigest ||
			releaseDigest != requirement.ReleaseDigest {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "skills")
		}
		for _, requiredTool := range registration.Spec.RequiredTools {
			if digest, included := toolKeys[registryKey(requiredTool.ID, requiredTool.Version)]; !included || digest != requiredTool.SpecDigest {
				return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "skill_tools")
			}
		}
		for _, permission := range registration.Spec.Permissions {
			permissionSet[permission] = struct{}{}
		}
	}
	for _, permission := range source.Permissions {
		if identity.ValidatePermission(permission) != nil {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "permissions")
		}
	}
	permissions := append([]identity.Permission(nil), source.Permissions...)
	sort.Slice(permissions, func(left, right int) bool { return permissions[left] < permissions[right] })
	if len(permissions) != len(permissionSet) {
		return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "permissions")
	}
	for index, permission := range permissions {
		_, required := permissionSet[permission]
		if !required || index > 0 && permission == permissions[index-1] {
			return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "permissions")
		}
	}
	source.Capabilities = capabilities
	source.Scopes = scopes
	source.Config = config
	source.Connectors = connectors
	source.Tools = toolRequirements
	source.Skills = skillRequirements
	source.Permissions = permissions
	encoded, err := json.Marshal(source)
	if err != nil || len(encoded) > MaxPluginDescriptorBytes {
		return PluginSpec{}, contractError(ErrorPluginSpecInvalid, "descriptor_size")
	}
	return source, nil
}

func canonicalPluginComponentPermissions(
	source []identity.Permission,
) ([]identity.Permission, error) {
	if len(source) == 0 || len(source) > MaxPluginPermissions {
		return nil, contractError(ErrorPluginSpecInvalid, "connector_permissions")
	}
	for _, permission := range source {
		if identity.ValidatePermission(permission) != nil {
			return nil, contractError(ErrorPluginSpecInvalid, "connector_permissions")
		}
	}
	permissions := append([]identity.Permission(nil), source...)
	sort.Slice(permissions, func(left, right int) bool { return permissions[left] < permissions[right] })
	for index, permission := range permissions {
		if index > 0 && permission == permissions[index-1] {
			return nil, contractError(ErrorPluginSpecInvalid, "connector_permissions")
		}
	}
	return permissions, nil
}

func canonicalPluginConfig(source *PluginConfigRequirement) (*PluginConfigRequirement, error) {
	if source == nil {
		return nil, nil
	}
	if !validCanonicalPluginDigest(source.ContractDigest) ||
		source.ContractRef != "artifact:"+source.ContractDigest || source.SizeBytes <= 0 ||
		source.SizeBytes > MaxPluginConfigContractBytes {
		return nil, contractError(ErrorPluginSpecInvalid, "config")
	}
	copyOfSource := *source
	return &copyOfSource, nil
}

func validPluginCapability(value string) bool {
	if len(value) > 200 || !validRequiredResourceToken(value) || strings.ContainsAny(value, "*?[]") {
		return false
	}
	_, err := goal.NewCapabilityRef(value)
	return err == nil
}

func validCanonicalPluginDigest(value string) bool {
	return validSurfaceDigest(value) && value == strings.ToLower(value)
}

func hasDuplicatePluginConnector(connectors []PluginConnectorSpec) bool {
	for index := 1; index < len(connectors); index++ {
		if registryKey(connectors[index].ID, connectors[index].Version) ==
			registryKey(connectors[index-1].ID, connectors[index-1].Version) {
			return true
		}
	}
	return false
}

func exactPluginSkillRelease(catalog *SkillReleaseCatalog, id, version string) (SkillRelease, bool) {
	if catalog == nil {
		return SkillRelease{}, false
	}
	index, found := catalog.byKey[registryKey(id, version)]
	if !found {
		return SkillRelease{}, false
	}
	return cloneSkillRelease(catalog.entries[index]), true
}

func clonePluginRegistration(source PluginRegistration) PluginRegistration {
	source.Spec.Capabilities = append([]string(nil), source.Spec.Capabilities...)
	source.Spec.Scopes = append([]SkillScope(nil), source.Spec.Scopes...)
	if source.Spec.Config != nil {
		config := *source.Spec.Config
		source.Spec.Config = &config
	}
	source.Spec.Connectors = append([]PluginConnectorSpec(nil), source.Spec.Connectors...)
	for index := range source.Spec.Connectors {
		source.Spec.Connectors[index].Permissions = append(
			[]identity.Permission(nil), source.Spec.Connectors[index].Permissions...,
		)
	}
	source.Spec.Tools = append([]PluginToolRequirement(nil), source.Spec.Tools...)
	source.Spec.Skills = append([]PluginSkillRequirement(nil), source.Spec.Skills...)
	source.Spec.Permissions = append([]identity.Permission(nil), source.Spec.Permissions...)
	return source
}

func pluginSpecDigest(spec PluginSpec) string {
	encoded, err := json.Marshal(struct {
		Contract string     `json:"contract"`
		Spec     PluginSpec `json:"spec"`
	}{"orquesta.tooling.plugin-descriptor.v1", spec})
	if err != nil {
		panic("canonical plugin spec cannot fail JSON encoding: " + err.Error())
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func pluginCatalogDigest(entries []PluginRegistration) string {
	encoded, err := json.Marshal(struct {
		Contract string               `json:"contract"`
		Plugins  []PluginRegistration `json:"plugins"`
	}{"orquesta.tooling.plugins.v1", entries})
	if err != nil {
		panic("canonical plugin catalog cannot fail JSON encoding: " + err.Error())
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}
