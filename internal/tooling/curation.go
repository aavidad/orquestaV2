package tooling

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"orquesta/internal/goal"
)

const (
	MaxCuratedCapabilities      = 64
	MaxCuratedScopes            = 32
	MaxCuratedSelectionsPerKind = 128
	MaxCuratedCatalogBytes      = 32 << 10

	ErrorCuratedCatalogInvalid   = "tooling.curated_catalog_invalid"
	ErrorCuratedContextInvalid   = "tooling.curated_context_invalid"
	ErrorCuratedContextDenied    = "tooling.curated_context_denied"
	ErrorCuratedSelectionInvalid = "tooling.curated_selection_invalid"
)

type CuratedToolSelection struct {
	ID         string `json:"id"`
	Version    string `json:"version"`
	SpecDigest string `json:"spec_digest"`
}

type CuratedSkillSelection struct {
	ID                 string `json:"id"`
	Version            string `json:"version"`
	RegistrationDigest string `json:"registration_digest"`
	ReleaseDigest      string `json:"release_digest"`
}

type CuratedPluginSelection struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	PluginDigest string `json:"plugin_digest"`
}

// CuratedCatalogSpec is one review-referenced allowlist. It contains no wildcard,
// handler, body, install request, activation state, or implicit latest rule.
type CuratedCatalogSpec struct {
	ID             string                   `json:"id"`
	Version        string                   `json:"version"`
	DescriptionKey string                   `json:"description_key"`
	ReviewRef      string                   `json:"review_ref"`
	ReviewDigest   string                   `json:"review_digest"`
	Capabilities   []string                 `json:"capabilities"`
	Scopes         []SkillScope             `json:"scopes"`
	Tools          []CuratedToolSelection   `json:"tools"`
	Skills         []CuratedSkillSelection  `json:"skills"`
	Plugins        []CuratedPluginSelection `json:"plugins"`
}

type CuratedRegistration struct {
	Spec   CuratedCatalogSpec `json:"spec"`
	Digest string             `json:"digest"`
}

type CuratedContext struct {
	capability goal.CapabilityRef
	scope      SkillScopeContext
	valid      bool
}

type CuratedCatalog struct {
	registration CuratedRegistration
	tools        *Registry
	skills       *SkillReleaseCatalog
	plugins      *PluginCatalog
	toolKeys     map[string]string
	skillKeys    map[string]CuratedSkillSelection
	pluginKeys   map[string]string
}

func NewCuratedContext(
	capability goal.CapabilityRef,
	scope SkillScopeContext,
) (CuratedContext, error) {
	if !scope.valid || !validCuratedScopeContext(scope) || !validCuratedCapability(capability.String()) {
		return CuratedContext{}, contractError(ErrorCuratedContextInvalid, "context")
	}
	return CuratedContext{capability: capability, scope: scope, valid: true}, nil
}

func NewCuratedCatalog(
	tools *Registry,
	skills *SkillReleaseCatalog,
	plugins *PluginCatalog,
	source CuratedCatalogSpec,
) (*CuratedCatalog, error) {
	if tools == nil || skills == nil || plugins == nil {
		return nil, contractError(ErrorCuratedCatalogInvalid, "registries")
	}
	spec, err := canonicalCuratedSpec(tools, skills, plugins, source)
	if err != nil {
		return nil, err
	}
	return &CuratedCatalog{
		registration: CuratedRegistration{Spec: spec, Digest: curatedSpecDigest(spec)},
		tools:        tools, skills: skills, plugins: plugins,
		toolKeys: curatedToolKeys(spec.Tools), skillKeys: curatedSkillKeys(spec.Skills),
		pluginKeys: curatedPluginKeys(spec.Plugins),
	}, nil
}

func (catalog *CuratedCatalog) Registration() CuratedRegistration {
	if catalog == nil {
		return CuratedRegistration{}
	}
	return cloneCuratedRegistration(catalog.registration)
}

func (catalog *CuratedCatalog) Digest() string {
	if catalog == nil {
		return ""
	}
	return catalog.registration.Digest
}

func (catalog *CuratedCatalog) ResolveTool(
	context CuratedContext,
	id string,
	version string,
) (Registration, bool, error) {
	if err := catalog.validateContext(context); err != nil {
		return Registration{}, false, err
	}
	expectedDigest, selected := catalog.toolKeys[registryKey(id, version)]
	if !selected {
		return Registration{}, false, nil
	}
	registration, found := catalog.tools.Lookup(id, version)
	if !found || registration.Digest != expectedDigest {
		return Registration{}, false, nil
	}
	return registration, true, nil
}

func (catalog *CuratedCatalog) ResolveSkill(
	context CuratedContext,
	id string,
	version string,
) (SkillRelease, bool, error) {
	if err := catalog.validateContext(context); err != nil {
		return SkillRelease{}, false, err
	}
	selection, selected := catalog.skillKeys[registryKey(id, version)]
	if !selected {
		return SkillRelease{}, false, nil
	}
	release, found, err := catalog.skills.ResolveReviewed(id, version, context.scope)
	if err != nil || !found {
		return SkillRelease{}, false, err
	}
	if release.Registration.Digest != selection.RegistrationDigest ||
		release.ReleaseDigest != selection.ReleaseDigest {
		return SkillRelease{}, false, nil
	}
	return release, true, nil
}

func (catalog *CuratedCatalog) ResolvePlugin(
	context CuratedContext,
	id string,
	version string,
) (PluginRegistration, bool, error) {
	if err := catalog.validateContext(context); err != nil {
		return PluginRegistration{}, false, err
	}
	expectedDigest, selected := catalog.pluginKeys[registryKey(id, version)]
	if !selected {
		return PluginRegistration{}, false, nil
	}
	registration, found := catalog.plugins.Lookup(id, version)
	if !found || registration.Digest != expectedDigest ||
		len(registration.Spec.Capabilities) > 0 && !containsCuratedCapability(registration.Spec.Capabilities, context.capability.String()) ||
		len(registration.Spec.Scopes) > 0 && !matchesCuratedScope(registration.Spec.Scopes, context.scope) {
		return PluginRegistration{}, false, nil
	}
	for _, requirement := range registration.Spec.Skills {
		release, visible, err := catalog.skills.ResolveReviewed(
			requirement.ID, requirement.Version, context.scope,
		)
		if err != nil {
			return PluginRegistration{}, false, err
		}
		if !visible || release.Registration.Digest != requirement.RegistrationDigest ||
			release.ReleaseDigest != requirement.ReleaseDigest {
			return PluginRegistration{}, false, nil
		}
	}
	return registration, true, nil
}

func (catalog *CuratedCatalog) validateContext(context CuratedContext) error {
	if catalog == nil || !context.valid || !context.scope.valid {
		return contractError(ErrorCuratedContextInvalid, "context")
	}
	if !containsCuratedCapability(catalog.registration.Spec.Capabilities, context.capability.String()) ||
		!matchesCuratedScope(catalog.registration.Spec.Scopes, context.scope) {
		return contractError(ErrorCuratedContextDenied, "context")
	}
	return nil
}

func canonicalCuratedSpec(
	tools *Registry,
	skills *SkillReleaseCatalog,
	plugins *PluginCatalog,
	source CuratedCatalogSpec,
) (CuratedCatalogSpec, error) {
	if len(source.Capabilities) > MaxCuratedCapabilities || len(source.Scopes) > MaxCuratedScopes ||
		len(source.Tools) > MaxCuratedSelectionsPerKind || len(source.Skills) > MaxCuratedSelectionsPerKind ||
		len(source.Plugins) > MaxCuratedSelectionsPerKind {
		return CuratedCatalogSpec{}, contractError(ErrorCuratedCatalogInvalid, "limits")
	}
	if !validToolID(source.ID) {
		return CuratedCatalogSpec{}, contractError(ErrorCuratedCatalogInvalid, "id")
	}
	if _, err := parseVersion(source.Version); err != nil {
		return CuratedCatalogSpec{}, contractError(ErrorCuratedCatalogInvalid, "version")
	}
	if source.DescriptionKey != "curation."+source.ID+".description" ||
		!validToolID(source.DescriptionKey) || !validCuratedReviewRef(source.ReviewRef) ||
		!validCuratedDigest(source.ReviewDigest) {
		return CuratedCatalogSpec{}, contractError(ErrorCuratedCatalogInvalid, "review")
	}
	capabilities := append([]string(nil), source.Capabilities...)
	sort.Strings(capabilities)
	if len(capabilities) == 0 {
		return CuratedCatalogSpec{}, contractError(ErrorCuratedCatalogInvalid, "capabilities")
	}
	for index, capability := range capabilities {
		if !validCuratedCapability(capability) || index > 0 && capability == capabilities[index-1] {
			return CuratedCatalogSpec{}, contractError(ErrorCuratedCatalogInvalid, "capabilities")
		}
	}
	scopes, err := canonicalSkillScopes(source.Scopes)
	if err != nil {
		return CuratedCatalogSpec{}, contractError(ErrorCuratedCatalogInvalid, "scopes")
	}
	for _, scope := range scopes {
		if !validCuratedScope(scope) {
			return CuratedCatalogSpec{}, contractError(ErrorCuratedCatalogInvalid, "scopes")
		}
	}
	if len(source.Tools)+len(source.Skills)+len(source.Plugins) == 0 {
		return CuratedCatalogSpec{}, contractError(ErrorCuratedCatalogInvalid, "selections")
	}
	toolSelections, err := canonicalCuratedTools(tools, source.Tools)
	if err != nil {
		return CuratedCatalogSpec{}, err
	}
	skillSelections, err := canonicalCuratedSkills(skills, source.Skills)
	if err != nil {
		return CuratedCatalogSpec{}, err
	}
	pluginSelections, err := canonicalCuratedPlugins(tools, skills, plugins, source.Plugins)
	if err != nil {
		return CuratedCatalogSpec{}, err
	}
	source.Capabilities = capabilities
	source.Scopes = scopes
	source.Tools = toolSelections
	source.Skills = skillSelections
	source.Plugins = pluginSelections
	if encoded, encodeErr := json.Marshal(source); encodeErr != nil || len(encoded) > MaxCuratedCatalogBytes {
		return CuratedCatalogSpec{}, contractError(ErrorCuratedCatalogInvalid, "size")
	}
	return source, nil
}

func canonicalCuratedTools(
	registry *Registry,
	source []CuratedToolSelection,
) ([]CuratedToolSelection, error) {
	selections := append([]CuratedToolSelection(nil), source...)
	sort.Slice(selections, func(left, right int) bool {
		return surfaceIdentityLess(selections[left].ID, selections[left].Version, selections[right].ID, selections[right].Version)
	})
	for index, selection := range selections {
		if index > 0 && registryKey(selection.ID, selection.Version) == registryKey(selections[index-1].ID, selections[index-1].Version) {
			return nil, contractError(ErrorCuratedSelectionInvalid, "tools")
		}
		registration, found := registry.Lookup(selection.ID, selection.Version)
		if !found || registration.Digest != selection.SpecDigest {
			return nil, contractError(ErrorCuratedSelectionInvalid, "tools")
		}
	}
	return selections, nil
}

func canonicalCuratedSkills(
	catalog *SkillReleaseCatalog,
	source []CuratedSkillSelection,
) ([]CuratedSkillSelection, error) {
	selections := append([]CuratedSkillSelection(nil), source...)
	sort.Slice(selections, func(left, right int) bool {
		return surfaceIdentityLess(selections[left].ID, selections[left].Version, selections[right].ID, selections[right].Version)
	})
	for index, selection := range selections {
		if index > 0 && registryKey(selection.ID, selection.Version) == registryKey(selections[index-1].ID, selections[index-1].Version) {
			return nil, contractError(ErrorCuratedSelectionInvalid, "skills")
		}
		release, found := exactPluginSkillRelease(catalog, selection.ID, selection.Version)
		if !found || release.Revoked || release.Registration.Digest != selection.RegistrationDigest ||
			release.ReleaseDigest != selection.ReleaseDigest {
			return nil, contractError(ErrorCuratedSelectionInvalid, "skills")
		}
	}
	return selections, nil
}

func canonicalCuratedPlugins(
	tools *Registry,
	skills *SkillReleaseCatalog,
	catalog *PluginCatalog,
	source []CuratedPluginSelection,
) ([]CuratedPluginSelection, error) {
	selections := append([]CuratedPluginSelection(nil), source...)
	sort.Slice(selections, func(left, right int) bool {
		return surfaceIdentityLess(selections[left].ID, selections[left].Version, selections[right].ID, selections[right].Version)
	})
	for index, selection := range selections {
		if index > 0 && registryKey(selection.ID, selection.Version) == registryKey(selections[index-1].ID, selections[index-1].Version) {
			return nil, contractError(ErrorCuratedSelectionInvalid, "plugins")
		}
		registration, found := catalog.Lookup(selection.ID, selection.Version)
		if !found || registration.Digest != selection.PluginDigest {
			return nil, contractError(ErrorCuratedSelectionInvalid, "plugins")
		}
		for _, requirement := range registration.Spec.Tools {
			tool, exists := tools.Lookup(requirement.ID, requirement.Version)
			if !exists || tool.Digest != requirement.SpecDigest {
				return nil, contractError(ErrorCuratedSelectionInvalid, "plugins")
			}
		}
		for _, requirement := range registration.Spec.Skills {
			release, exists := exactPluginSkillRelease(skills, requirement.ID, requirement.Version)
			if !exists || release.Revoked || release.Registration.Digest != requirement.RegistrationDigest ||
				release.ReleaseDigest != requirement.ReleaseDigest {
				return nil, contractError(ErrorCuratedSelectionInvalid, "plugins")
			}
		}
	}
	return selections, nil
}

func validCuratedCapability(value string) bool {
	if !validSkillScopeValue(value) || !validRequiredResourceToken(value) || strings.ContainsAny(value, "*?[]") {
		return false
	}
	_, err := goal.NewCapabilityRef(value)
	return err == nil
}

func validCuratedReviewRef(value string) bool {
	return validSkillEvidenceRef(value) && validRequiredResourceToken(value) &&
		!strings.ContainsAny(value, "*?[]")
}

func validCuratedDigest(value string) bool {
	return validSurfaceDigest(value) && value == strings.ToLower(value)
}

func validCuratedScope(scope SkillScope) bool {
	for _, value := range []string{scope.ProductID, scope.ProjectRef, string(scope.Role), scope.GoalRef} {
		if value != "" && (!validRequiredResourceToken(value) || strings.ContainsAny(value, "*?[]")) {
			return false
		}
	}
	return true
}

func validCuratedScopeContext(context SkillScopeContext) bool {
	return validCuratedScope(SkillScope{
		ProductID: context.productID, ProjectRef: context.projectRef.String(),
		Role: context.role, GoalRef: context.goalRef.String(),
	})
}

func containsCuratedCapability(capabilities []string, value string) bool {
	index := sort.SearchStrings(capabilities, value)
	return index < len(capabilities) && capabilities[index] == value
}

func matchesCuratedScope(scopes []SkillScope, context SkillScopeContext) bool {
	for _, scope := range scopes {
		if scope.matches(context) {
			return true
		}
	}
	return false
}

func curatedToolKeys(selections []CuratedToolSelection) map[string]string {
	result := make(map[string]string, len(selections))
	for _, selection := range selections {
		result[registryKey(selection.ID, selection.Version)] = selection.SpecDigest
	}
	return result
}

func curatedSkillKeys(selections []CuratedSkillSelection) map[string]CuratedSkillSelection {
	result := make(map[string]CuratedSkillSelection, len(selections))
	for _, selection := range selections {
		result[registryKey(selection.ID, selection.Version)] = selection
	}
	return result
}

func curatedPluginKeys(selections []CuratedPluginSelection) map[string]string {
	result := make(map[string]string, len(selections))
	for _, selection := range selections {
		result[registryKey(selection.ID, selection.Version)] = selection.PluginDigest
	}
	return result
}

func cloneCuratedRegistration(source CuratedRegistration) CuratedRegistration {
	source.Spec.Capabilities = append([]string(nil), source.Spec.Capabilities...)
	source.Spec.Scopes = append([]SkillScope(nil), source.Spec.Scopes...)
	source.Spec.Tools = append([]CuratedToolSelection(nil), source.Spec.Tools...)
	source.Spec.Skills = append([]CuratedSkillSelection(nil), source.Spec.Skills...)
	source.Spec.Plugins = append([]CuratedPluginSelection(nil), source.Spec.Plugins...)
	return source
}

func curatedSpecDigest(spec CuratedCatalogSpec) string {
	encoded, err := json.Marshal(struct {
		Contract string             `json:"contract"`
		Spec     CuratedCatalogSpec `json:"spec"`
	}{"orquesta.tooling.curated-catalog.v1", spec})
	if err != nil {
		panic("canonical curated catalog cannot fail JSON encoding: " + err.Error())
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}
