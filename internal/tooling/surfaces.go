package tooling

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"mime"
	"net/url"
	"sort"
	"strings"

	"orquesta/internal/identity"
)

const (
	ErrorSurfaceSpecInvalid     = "tooling.surface_spec_invalid"
	ErrorSurfaceSpecDuplicate   = "tooling.surface_spec_duplicate"
	ErrorPromptInputInvalid     = "tooling.prompt_input_invalid"
	ErrorResourceRequestInvalid = "tooling.resource_request_invalid"
	ErrorResourcePageInvalid    = "tooling.resource_page_invalid"
)

type ResourceSubscriptionPolicy string

const (
	ResourceSubscriptionNone            ResourceSubscriptionPolicy = "none"
	ResourceSubscriptionSnapshotChanged ResourceSubscriptionPolicy = "snapshot_changed"
)

type ResourcePagePolicy struct {
	DefaultItems uint32 `json:"default_items"`
	MaxItems     uint32 `json:"max_items"`
	MaxBytes     int64  `json:"max_bytes"`
}

// ResourceSpec describes addressable read-only context. Its item schema and
// page limits cannot be used as an invocation schema, receipt, cost, or handler.
type ResourceSpec struct {
	ID           string                     `json:"id"`
	Version      string                     `json:"version"`
	URI          string                     `json:"uri"`
	MediaType    string                     `json:"media_type"`
	Permissions  []identity.Permission      `json:"permissions"`
	MaxBytes     int64                      `json:"max_bytes"`
	ItemSchema   json.RawMessage            `json:"item_schema"`
	Page         ResourcePagePolicy         `json:"page"`
	Subscription ResourceSubscriptionPolicy `json:"subscription"`
}

// PromptSpec describes governed text and its arguments. Rendering belongs to
// a presentation adapter and cannot invoke a tool or read a resource.
type PromptSpec struct {
	ID              string          `json:"id"`
	Version         string          `json:"version"`
	TemplateKey     string          `json:"template_key"`
	ArgumentsSchema json.RawMessage `json:"arguments_schema"`
}

type ResourceRegistration struct {
	Spec   ResourceSpec `json:"spec"`
	Digest string       `json:"digest"`
}

type PromptRegistration struct {
	Spec   PromptSpec `json:"spec"`
	Digest string     `json:"digest"`
}

// SurfaceCatalog preserves three type-level boundaries. Tool lookups delegate
// to the exact TLS-01 registry instead of copying tool definitions.
type SurfaceCatalog struct {
	tools       *Registry
	resources   []ResourceRegistration
	resourceKey map[string]int
	prompts     []PromptRegistration
	promptKey   map[string]int
	digest      string
}

func NewSurfaceCatalog(
	tools *Registry,
	resources []ResourceSpec,
	prompts []PromptSpec,
) (*SurfaceCatalog, error) {
	if tools == nil {
		return nil, contractError(ErrorSurfaceSpecInvalid, "tools")
	}
	resourceEntries, resourceKey, err := canonicalResources(resources)
	if err != nil {
		return nil, err
	}
	promptEntries, promptKey, err := canonicalPrompts(prompts)
	if err != nil {
		return nil, err
	}
	catalog := &SurfaceCatalog{
		tools: tools, resources: resourceEntries, resourceKey: resourceKey,
		prompts: promptEntries, promptKey: promptKey,
	}
	catalog.digest = surfaceCatalogDigest(tools.Digest(), resourceEntries, promptEntries)
	return catalog, nil
}

func (catalog *SurfaceCatalog) LookupTool(id, version string) (Registration, bool) {
	if catalog == nil || catalog.tools == nil {
		return Registration{}, false
	}
	return catalog.tools.Lookup(id, version)
}

func (catalog *SurfaceCatalog) ListTools() []Registration {
	if catalog == nil || catalog.tools == nil {
		return nil
	}
	return catalog.tools.List()
}

func (catalog *SurfaceCatalog) LookupResource(id, version string) (ResourceRegistration, bool) {
	if catalog == nil {
		return ResourceRegistration{}, false
	}
	index, found := catalog.resourceKey[registryKey(id, version)]
	if !found {
		return ResourceRegistration{}, false
	}
	return cloneResourceRegistration(catalog.resources[index]), true
}

func (catalog *SurfaceCatalog) ListResources() []ResourceRegistration {
	if catalog == nil {
		return nil
	}
	result := make([]ResourceRegistration, len(catalog.resources))
	for index, entry := range catalog.resources {
		result[index] = cloneResourceRegistration(entry)
	}
	return result
}

func (catalog *SurfaceCatalog) LookupPrompt(id, version string) (PromptRegistration, bool) {
	if catalog == nil {
		return PromptRegistration{}, false
	}
	index, found := catalog.promptKey[registryKey(id, version)]
	if !found {
		return PromptRegistration{}, false
	}
	return clonePromptRegistration(catalog.prompts[index]), true
}

func (catalog *SurfaceCatalog) ListPrompts() []PromptRegistration {
	if catalog == nil {
		return nil
	}
	result := make([]PromptRegistration, len(catalog.prompts))
	for index, entry := range catalog.prompts {
		result[index] = clonePromptRegistration(entry)
	}
	return result
}

func (catalog *SurfaceCatalog) ValidatePromptInput(
	id, version string,
	payload json.RawMessage,
) (json.RawMessage, error) {
	registration, found := catalog.LookupPrompt(id, version)
	if !found {
		return nil, contractError(ErrorSpecNotFound, "identity")
	}
	canonical, err := validatePayloadAgainstSchema(registration.Spec.ArgumentsSchema, payload)
	if err != nil {
		return nil, contractError(ErrorPromptInputInvalid, "payload")
	}
	return canonical, nil
}

func (catalog *SurfaceCatalog) Digest() string {
	if catalog == nil {
		return ""
	}
	return catalog.digest
}

func canonicalResources(
	specs []ResourceSpec,
) ([]ResourceRegistration, map[string]int, error) {
	entries := make([]ResourceRegistration, 0, len(specs))
	seen := make(map[string]struct{}, len(specs))
	for _, source := range specs {
		spec, err := canonicalResourceSpec(source)
		if err != nil {
			return nil, nil, err
		}
		key := registryKey(spec.ID, spec.Version)
		if _, duplicate := seen[key]; duplicate {
			return nil, nil, contractError(ErrorSurfaceSpecDuplicate, "resource_identity")
		}
		seen[key] = struct{}{}
		entries = append(entries, ResourceRegistration{Spec: spec, Digest: surfaceSpecDigest("resource", spec)})
	}
	sort.Slice(entries, func(left, right int) bool {
		return surfaceIdentityLess(
			entries[left].Spec.ID, entries[left].Spec.Version,
			entries[right].Spec.ID, entries[right].Spec.Version,
		)
	})
	return entries, surfaceResourceIndex(entries), nil
}

func canonicalPrompts(specs []PromptSpec) ([]PromptRegistration, map[string]int, error) {
	entries := make([]PromptRegistration, 0, len(specs))
	seen := make(map[string]struct{}, len(specs))
	for _, source := range specs {
		spec, err := canonicalPromptSpec(source)
		if err != nil {
			return nil, nil, err
		}
		key := registryKey(spec.ID, spec.Version)
		if _, duplicate := seen[key]; duplicate {
			return nil, nil, contractError(ErrorSurfaceSpecDuplicate, "prompt_identity")
		}
		seen[key] = struct{}{}
		entries = append(entries, PromptRegistration{Spec: spec, Digest: surfaceSpecDigest("prompt", spec)})
	}
	sort.Slice(entries, func(left, right int) bool {
		return surfaceIdentityLess(
			entries[left].Spec.ID, entries[left].Spec.Version,
			entries[right].Spec.ID, entries[right].Spec.Version,
		)
	})
	return entries, surfacePromptIndex(entries), nil
}

func canonicalResourceSpec(source ResourceSpec) (ResourceSpec, error) {
	if !validToolID(source.ID) {
		return ResourceSpec{}, contractError(ErrorSurfaceSpecInvalid, "resource_id")
	}
	if _, err := parseVersion(source.Version); err != nil {
		return ResourceSpec{}, contractError(ErrorSurfaceSpecInvalid, "resource_version")
	}
	parsed, err := url.Parse(source.URI)
	if err != nil || source.URI == "" || strings.TrimSpace(source.URI) != source.URI ||
		!parsed.IsAbs() || parsed.Scheme != strings.ToLower(parsed.Scheme) || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host == "" || parsed.Path == "" {
		return ResourceSpec{}, contractError(ErrorSurfaceSpecInvalid, "resource_uri")
	}
	mediaType, parameters, err := mime.ParseMediaType(source.MediaType)
	if err != nil || len(parameters) != 0 || mediaType != source.MediaType || mediaType != strings.ToLower(mediaType) {
		return ResourceSpec{}, contractError(ErrorSurfaceSpecInvalid, "resource_media_type")
	}
	permissions, err := canonicalSurfacePermissions(source.Permissions)
	if err != nil {
		return ResourceSpec{}, err
	}
	if source.MaxBytes <= 0 {
		return ResourceSpec{}, contractError(ErrorSurfaceSpecInvalid, "resource_max_bytes")
	}
	itemSchema, err := canonicalObjectSchema(source.ItemSchema)
	if err != nil {
		return ResourceSpec{}, contractError(ErrorSurfaceSpecInvalid, "resource_item_schema")
	}
	if source.Page.DefaultItems == 0 || source.Page.MaxItems < source.Page.DefaultItems ||
		source.Page.MaxBytes <= 0 || source.Page.MaxBytes > source.MaxBytes {
		return ResourceSpec{}, contractError(ErrorSurfaceSpecInvalid, "resource_page")
	}
	if source.Subscription != ResourceSubscriptionNone &&
		source.Subscription != ResourceSubscriptionSnapshotChanged {
		return ResourceSpec{}, contractError(ErrorSurfaceSpecInvalid, "resource_subscription")
	}
	source.Permissions = permissions
	source.ItemSchema = itemSchema
	return source, nil
}

func canonicalPromptSpec(source PromptSpec) (PromptSpec, error) {
	if !validToolID(source.ID) {
		return PromptSpec{}, contractError(ErrorSurfaceSpecInvalid, "prompt_id")
	}
	if _, err := parseVersion(source.Version); err != nil {
		return PromptSpec{}, contractError(ErrorSurfaceSpecInvalid, "prompt_version")
	}
	if !strings.HasPrefix(source.TemplateKey, "prompt.") || !validToolID(source.TemplateKey) {
		return PromptSpec{}, contractError(ErrorSurfaceSpecInvalid, "prompt_template_key")
	}
	arguments, err := canonicalObjectSchema(source.ArgumentsSchema)
	if err != nil {
		return PromptSpec{}, contractError(ErrorSurfaceSpecInvalid, "prompt_arguments_schema")
	}
	source.ArgumentsSchema = arguments
	return source, nil
}

func canonicalSurfacePermissions(source []identity.Permission) ([]identity.Permission, error) {
	permissions := append([]identity.Permission(nil), source...)
	if len(permissions) == 0 {
		return nil, contractError(ErrorSurfaceSpecInvalid, "resource_permissions")
	}
	sort.Slice(permissions, func(left, right int) bool { return permissions[left] < permissions[right] })
	for index, permission := range permissions {
		if identity.ValidatePermission(permission) != nil || index > 0 && permission == permissions[index-1] {
			return nil, contractError(ErrorSurfaceSpecInvalid, "resource_permissions")
		}
	}
	return permissions, nil
}

func surfaceIdentityLess(leftID, leftVersion, rightID, rightVersion string) bool {
	if leftID != rightID {
		return leftID < rightID
	}
	left, _ := parseVersion(leftVersion)
	right, _ := parseVersion(rightVersion)
	return left < right
}

func surfaceResourceIndex(entries []ResourceRegistration) map[string]int {
	result := make(map[string]int, len(entries))
	for index, entry := range entries {
		result[registryKey(entry.Spec.ID, entry.Spec.Version)] = index
	}
	return result
}

func surfacePromptIndex(entries []PromptRegistration) map[string]int {
	result := make(map[string]int, len(entries))
	for index, entry := range entries {
		result[registryKey(entry.Spec.ID, entry.Spec.Version)] = index
	}
	return result
}

func cloneResourceRegistration(source ResourceRegistration) ResourceRegistration {
	source.Spec.Permissions = append([]identity.Permission(nil), source.Spec.Permissions...)
	source.Spec.ItemSchema = append(json.RawMessage(nil), source.Spec.ItemSchema...)
	return source
}

func clonePromptRegistration(source PromptRegistration) PromptRegistration {
	source.Spec.ArgumentsSchema = append(json.RawMessage(nil), source.Spec.ArgumentsSchema...)
	return source
}

func surfaceSpecDigest(kind string, spec any) string {
	encoded, err := json.Marshal(struct {
		Kind string `json:"kind"`
		Spec any    `json:"spec"`
	}{Kind: kind, Spec: spec})
	if err != nil {
		panic("canonical surface spec cannot fail JSON encoding: " + err.Error())
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func surfaceCatalogDigest(
	toolDigest string,
	resources []ResourceRegistration,
	prompts []PromptRegistration,
) string {
	encoded, err := json.Marshal(struct {
		Contract  string                 `json:"contract"`
		Tools     string                 `json:"tools"`
		Resources []ResourceRegistration `json:"resources"`
		Prompts   []PromptRegistration   `json:"prompts"`
	}{"orquesta.tooling.surfaces.v1", toolDigest, resources, prompts})
	if err != nil {
		panic("canonical surface catalog cannot fail JSON encoding: " + err.Error())
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}
