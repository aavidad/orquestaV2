package tooling

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"orquesta/internal/identity"
)

const (
	SkillFormatMarkdownV1        = "skill_markdown_v1"
	MaxSkillInstructionsBytes    = 256 << 10
	ErrorSkillSpecInvalid        = "tooling.skill_spec_invalid"
	ErrorSkillSpecDuplicate      = "tooling.skill_spec_duplicate"
	ErrorSkillNotFound           = "tooling.skill_not_found"
	ErrorSkillContentInvalid     = "tooling.skill_content_invalid"
	ErrorSkillLoadRequestInvalid = "tooling.skill_load_request_invalid"
)

type SkillToolRequirement struct {
	ID         string `json:"id"`
	Version    string `json:"version"`
	SpecDigest string `json:"spec_digest"`
}

// SkillToolCatalog is the narrow consumer-owned boundary required to bind a
// skill to already reviewed tools. Tool registry implementations adapt to this
// view; the skill registry does not own or import their concrete descriptors.
type SkillToolCatalog interface {
	LookupSkillTool(id, version string) (SkillToolRegistration, bool)
}

type SkillToolRegistration struct {
	ID          string
	Version     string
	SpecDigest  string
	Permissions []identity.Permission
}

type skillContractFailure struct{ code, field string }

func (failure *skillContractFailure) Error() string { return failure.code + ":" + failure.field }
func SkillErrorCode(err error) string {
	var failure *skillContractFailure
	if errors.As(err, &failure) {
		return failure.code
	}
	return ""
}

func skillContractError(code, field string) error {
	return &skillContractFailure{code: code, field: field}
}

// SkillSpec is metadata only. Instructions remain in CAS and are loaded by
// exact ref/digest only after a later application policy admits the skill.
type SkillSpec struct {
	ID                 string                 `json:"id"`
	Version            string                 `json:"version"`
	Format             string                 `json:"format"`
	DescriptionKey     string                 `json:"description_key"`
	InstructionsRef    string                 `json:"instructions_ref"`
	InstructionsDigest string                 `json:"instructions_digest"`
	Scopes             []SkillScope           `json:"scopes"`
	Resources          []SkillResourceSpec    `json:"resources"`
	RequiredTools      []SkillToolRequirement `json:"required_tools"`
	Permissions        []identity.Permission  `json:"permissions"`
}

// SkillCandidate is construction input only. Registry state never retains the
// instructions bytes supplied here.
type SkillCandidate struct {
	Spec             SkillSpec              `json:"spec"`
	Instructions     []byte                 `json:"-"`
	ResourceContents []SkillResourceContent `json:"-"`
}

type SkillRegistration struct {
	Spec   SkillSpec `json:"spec"`
	Digest string    `json:"digest"`
}

type SkillLoadRequest struct {
	ID                 string `json:"id"`
	Version            string `json:"version"`
	RegistrationDigest string `json:"registration_digest"`
	InstructionsRef    string `json:"instructions_ref"`
	InstructionsDigest string `json:"instructions_digest"`
	MaxBytes           int64  `json:"max_bytes"`
}

type SkillRegistry struct {
	entries []SkillRegistration
	byKey   map[string]int
	digest  string
}

// NewSkillRegistry validates every candidate and its SKILL.md bytes before it
// publishes one immutable metadata catalog. It never returns partial state.
func NewSkillRegistry(tools SkillToolCatalog, candidates ...SkillCandidate) (*SkillRegistry, error) {
	if tools == nil {
		return nil, skillContractError(ErrorSkillSpecInvalid, "tools")
	}
	entries := make([]SkillRegistration, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		spec, err := canonicalSkillSpec(
			tools, candidate.Spec, candidate.Instructions, candidate.ResourceContents,
		)
		if err != nil {
			return nil, err
		}
		key := skillRegistryKey(spec.ID, spec.Version)
		if _, duplicate := seen[key]; duplicate {
			return nil, skillContractError(ErrorSkillSpecDuplicate, "identity")
		}
		seen[key] = struct{}{}
		entries = append(entries, SkillRegistration{Spec: spec, Digest: skillSpecDigest(spec)})
	}
	sort.Slice(entries, func(left, right int) bool {
		return skillIdentityLess(
			entries[left].Spec.ID, entries[left].Spec.Version,
			entries[right].Spec.ID, entries[right].Spec.Version,
		)
	})
	byKey := make(map[string]int, len(entries))
	for index, entry := range entries {
		byKey[skillRegistryKey(entry.Spec.ID, entry.Spec.Version)] = index
	}
	return &SkillRegistry{
		entries: entries, byKey: byKey, digest: skillRegistryDigest(entries),
	}, nil
}

func (registry *SkillRegistry) Lookup(id, version string) (SkillRegistration, bool) {
	if registry == nil {
		return SkillRegistration{}, false
	}
	index, found := registry.byKey[skillRegistryKey(id, version)]
	if !found {
		return SkillRegistration{}, false
	}
	return cloneSkillRegistration(registry.entries[index]), true
}

func (registry *SkillRegistry) List() []SkillRegistration {
	if registry == nil {
		return nil
	}
	result := make([]SkillRegistration, len(registry.entries))
	for index, entry := range registry.entries {
		result[index] = cloneSkillRegistration(entry)
	}
	return result
}

func (registry *SkillRegistry) Digest() string {
	if registry == nil {
		return ""
	}
	return registry.digest
}

// NewLoadRequest reveals only the immutable CAS subject needed for progressive
// loading. It contains no local path, body, credential, or activation state.
func (registry *SkillRegistry) NewLoadRequest(id, version string) (SkillLoadRequest, error) {
	registration, found := registry.Lookup(id, version)
	if !found {
		return SkillLoadRequest{}, skillContractError(ErrorSkillNotFound, "identity")
	}
	return SkillLoadRequest{
		ID: registration.Spec.ID, Version: registration.Spec.Version,
		RegistrationDigest: registration.Digest,
		InstructionsRef:    registration.Spec.InstructionsRef,
		InstructionsDigest: registration.Spec.InstructionsDigest,
		MaxBytes:           MaxSkillInstructionsBytes,
	}, nil
}

// ValidateLoadedInstructions proves that progressively loaded bytes are the
// exact reviewed SKILL.md subject registered earlier and returns a detached copy.
func (registry *SkillRegistry) ValidateLoadedInstructions(
	request SkillLoadRequest,
	instructions []byte,
) ([]byte, error) {
	canonical, err := registry.NewLoadRequest(request.ID, request.Version)
	if err != nil || canonical != request {
		return nil, skillContractError(ErrorSkillLoadRequestInvalid, "request")
	}
	if !validSkillMarkdown(request.ID, instructions) ||
		skillContentDigest(instructions) != request.InstructionsDigest {
		return nil, skillContractError(ErrorSkillContentInvalid, "instructions")
	}
	return append([]byte(nil), instructions...), nil
}

func canonicalSkillSpec(
	tools SkillToolCatalog,
	source SkillSpec,
	instructions []byte,
	resourceContents []SkillResourceContent,
) (SkillSpec, error) {
	if !validSkillID(source.ID) {
		return SkillSpec{}, skillContractError(ErrorSkillSpecInvalid, "id")
	}
	if _, err := parseSkillVersion(source.Version); err != nil {
		return SkillSpec{}, skillContractError(ErrorSkillSpecInvalid, "version")
	}
	if source.Format != SkillFormatMarkdownV1 {
		return SkillSpec{}, skillContractError(ErrorSkillSpecInvalid, "format")
	}
	if source.DescriptionKey != "skill."+source.ID+".description" || !validSkillID(source.DescriptionKey) {
		return SkillSpec{}, skillContractError(ErrorSkillSpecInvalid, "description_key")
	}
	if !validSkillDigest(source.InstructionsDigest) ||
		source.InstructionsRef != "artifact:"+source.InstructionsDigest {
		return SkillSpec{}, skillContractError(ErrorSkillSpecInvalid, "instructions_ref")
	}
	if !validSkillMarkdown(source.ID, instructions) ||
		skillContentDigest(instructions) != source.InstructionsDigest {
		return SkillSpec{}, skillContractError(ErrorSkillContentInvalid, "instructions")
	}
	scopes, err := canonicalSkillScopes(source.Scopes)
	if err != nil {
		return SkillSpec{}, err
	}
	resources, err := canonicalSkillResources(source.Resources, resourceContents)
	if err != nil {
		return SkillSpec{}, err
	}
	requirements := append([]SkillToolRequirement(nil), source.RequiredTools...)
	sort.Slice(requirements, func(left, right int) bool {
		return skillIdentityLess(
			requirements[left].ID, requirements[left].Version,
			requirements[right].ID, requirements[right].Version,
		)
	})
	permissionSet := make(map[identity.Permission]struct{})
	for index, requirement := range requirements {
		if index > 0 && skillRegistryKey(requirement.ID, requirement.Version) ==
			skillRegistryKey(requirements[index-1].ID, requirements[index-1].Version) {
			return SkillSpec{}, skillContractError(ErrorSkillSpecInvalid, "required_tools")
		}
		registration, found := tools.LookupSkillTool(requirement.ID, requirement.Version)
		if !found || registration.ID != requirement.ID || registration.Version != requirement.Version ||
			registration.SpecDigest != requirement.SpecDigest || !validSkillDigest(registration.SpecDigest) {
			return SkillSpec{}, skillContractError(ErrorSkillSpecInvalid, "required_tools")
		}
		for _, permission := range registration.Permissions {
			permissionSet[permission] = struct{}{}
		}
	}
	permissions := append([]identity.Permission(nil), source.Permissions...)
	sort.Slice(permissions, func(left, right int) bool { return permissions[left] < permissions[right] })
	if len(permissions) != len(permissionSet) {
		return SkillSpec{}, skillContractError(ErrorSkillSpecInvalid, "permissions")
	}
	for index, permission := range permissions {
		_, required := permissionSet[permission]
		if identity.ValidatePermission(permission) != nil || !required ||
			index > 0 && permission == permissions[index-1] {
			return SkillSpec{}, skillContractError(ErrorSkillSpecInvalid, "permissions")
		}
	}
	source.RequiredTools = requirements
	source.Permissions = permissions
	source.Scopes = scopes
	source.Resources = resources
	return source, nil
}

func validSkillMarkdown(id string, instructions []byte) bool {
	if len(instructions) == 0 || len(instructions) > MaxSkillInstructionsBytes ||
		!utf8.Valid(instructions) || strings.ContainsRune(string(instructions), '\r') {
		return false
	}
	for _, character := range string(instructions) {
		if unicode.IsControl(character) && character != '\n' && character != '\t' {
			return false
		}
	}
	lines := strings.Split(string(instructions), "\n")
	if len(lines) < 7 || lines[0] != "---" {
		return false
	}
	closing := -1
	name, description := "", ""
	for index := 1; index < len(lines); index++ {
		line := lines[index]
		if line == "---" {
			closing = index
			break
		}
		switch {
		case strings.HasPrefix(line, "name: ") && name == "":
			name = strings.TrimPrefix(line, "name: ")
		case strings.HasPrefix(line, "description: ") && description == "":
			description = strings.TrimPrefix(line, "description: ")
		default:
			return false
		}
	}
	if closing != 3 || name != id || strings.TrimSpace(description) == "" ||
		closing+2 >= len(lines) || lines[closing+1] != "" {
		return false
	}
	for _, line := range lines[closing+2:] {
		if strings.HasPrefix(line, "# ") && strings.TrimSpace(strings.TrimPrefix(line, "# ")) != "" {
			return true
		}
		if strings.TrimSpace(line) != "" {
			return false
		}
	}
	return false
}

func skillContentDigest(instructions []byte) string {
	digest := sha256.Sum256(instructions)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func skillRegistryKey(id, version string) string { return id + "\x00" + version }

func skillIdentityLess(leftID, leftVersion, rightID, rightVersion string) bool {
	if leftID != rightID {
		return leftID < rightID
	}
	left, _ := parseSkillVersion(leftVersion)
	right, _ := parseSkillVersion(rightVersion)
	return left < right
}

func parseSkillVersion(version string) (uint64, error) {
	if version == "" || version == "0" || len(version) > 1 && version[0] == '0' {
		return 0, errors.New("skill_version_invalid")
	}
	value, err := strconv.ParseUint(version, 10, 64)
	if err != nil || value == 0 {
		return 0, errors.New("skill_version_invalid")
	}
	return value, nil
}

func validSkillID(id string) bool {
	if len(id) < 3 || len(id) > 128 || id[0] == '.' || id[len(id)-1] == '.' {
		return false
	}
	segments := strings.Split(id, ".")
	if len(segments) < 2 {
		return false
	}
	for _, segment := range segments {
		if segment == "" {
			return false
		}
		for index, character := range segment {
			if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' && index > 0 || character == '-' && index > 0 && index < len(segment)-1 {
				continue
			}
			return false
		}
	}
	return true
}

func validSkillDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func cloneSkillRegistration(source SkillRegistration) SkillRegistration {
	source.Spec.Scopes = append([]SkillScope(nil), source.Spec.Scopes...)
	source.Spec.Resources = append([]SkillResourceSpec(nil), source.Spec.Resources...)
	source.Spec.RequiredTools = append([]SkillToolRequirement(nil), source.Spec.RequiredTools...)
	source.Spec.Permissions = append([]identity.Permission(nil), source.Spec.Permissions...)
	return source
}

func skillSpecDigest(spec SkillSpec) string {
	encoded, err := json.Marshal(spec)
	if err != nil {
		panic("canonical skill spec cannot fail JSON encoding: " + err.Error())
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func skillRegistryDigest(entries []SkillRegistration) string {
	encoded, err := json.Marshal(struct {
		Contract string              `json:"contract"`
		Skills   []SkillRegistration `json:"skills"`
	}{"orquesta.tooling.skills.v1", entries})
	if err != nil {
		panic("canonical skill registry cannot fail JSON encoding: " + err.Error())
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}
