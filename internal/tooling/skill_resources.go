package tooling

import (
	"encoding/json"
	"mime"
	"sort"
	"strings"
)

const (
	MaxSkillResourceBytes            = 4 << 20
	MaxSkillResourcesPerSkill        = 1024
	DefaultSkillResourcePageItems    = 32
	MaxSkillResourcePageItems        = 128
	ErrorSkillResourceInvalid        = "tooling.skill_resource_invalid"
	ErrorSkillResourceNotFound       = "tooling.skill_resource_not_found"
	ErrorSkillResourcePageInvalid    = "tooling.skill_resource_page_invalid"
	ErrorSkillResourceLoadInvalid    = "tooling.skill_resource_load_invalid"
	ErrorSkillResourceContentInvalid = "tooling.skill_resource_content_invalid"
)

type SkillResourceSpec struct {
	ID            string `json:"id"`
	MediaType     string `json:"media_type"`
	ContentRef    string `json:"content_ref"`
	ContentDigest string `json:"content_digest"`
	SizeBytes     int64  `json:"size_bytes"`
}

// SkillResourceContent is construction input only. Registry state never
// retains progressively loaded resource bytes.
type SkillResourceContent struct {
	ID      string `json:"id"`
	Content []byte `json:"-"`
}

type SkillResourceLoadRequest struct {
	SkillID            string `json:"skill_id"`
	SkillVersion       string `json:"skill_version"`
	RegistrationDigest string `json:"registration_digest"`
	ScopesDigest       string `json:"scopes_digest"`
	ResourceID         string `json:"resource_id"`
	MediaType          string `json:"media_type"`
	ContentRef         string `json:"content_ref"`
	ContentDigest      string `json:"content_digest"`
	SizeBytes          int64  `json:"size_bytes"`
	MaxBytes           int64  `json:"max_bytes"`
}

// SkillResourcePageRequest binds metadata pagination to one immutable skill
// registration and its exact declarative scope set. Scope visibility and
// authorization remain separate application concerns.
type SkillResourcePageRequest struct {
	SkillID            string `json:"skill_id"`
	SkillVersion       string `json:"skill_version"`
	RegistrationDigest string `json:"registration_digest"`
	ScopesDigest       string `json:"scopes_digest"`
	Cursor             string `json:"cursor,omitempty"`
	Limit              uint32 `json:"limit"`
}

type SkillResourcePage struct {
	Request    SkillResourcePageRequest `json:"request"`
	Items      []SkillResourceSpec      `json:"items"`
	NextCursor string                   `json:"next_cursor,omitempty"`
	Complete   bool                     `json:"complete"`
}

func (registry *SkillRegistry) ListResources(id, version string) ([]SkillResourceSpec, bool) {
	registration, found := registry.Lookup(id, version)
	if !found {
		return nil, false
	}
	return append([]SkillResourceSpec(nil), registration.Spec.Resources...), true
}

func (registry *SkillRegistry) NewResourcePageRequest(
	id string,
	version string,
	cursor string,
	limit uint32,
) (SkillResourcePageRequest, error) {
	registration, found := registry.Lookup(id, version)
	if !found {
		return SkillResourcePageRequest{}, skillContractError(ErrorSkillNotFound, "identity")
	}
	if limit == 0 {
		limit = DefaultSkillResourcePageItems
	}
	if limit > MaxSkillResourcePageItems || resourcePageStart(registration, cursor) < 0 {
		return SkillResourcePageRequest{}, skillContractError(ErrorSkillResourcePageInvalid, "request")
	}
	return SkillResourcePageRequest{
		SkillID: registration.Spec.ID, SkillVersion: registration.Spec.Version,
		RegistrationDigest: registration.Digest, ScopesDigest: skillScopesDigest(registration.Spec.Scopes),
		Cursor: cursor, Limit: limit,
	}, nil
}

// ListResourcePage returns detached metadata only. Its cursor is derived from
// the exact registration and cannot be replayed after descriptor freshness
// changes or against another skill/version.
func (registry *SkillRegistry) ListResourcePage(request SkillResourcePageRequest) (SkillResourcePage, error) {
	canonical, err := registry.NewResourcePageRequest(
		request.SkillID, request.SkillVersion, request.Cursor, request.Limit,
	)
	if err != nil || canonical != request {
		return SkillResourcePage{}, skillContractError(ErrorSkillResourcePageInvalid, "request")
	}
	registration, _ := registry.Lookup(request.SkillID, request.SkillVersion)
	start := resourcePageStart(registration, request.Cursor)
	end := start + int(request.Limit)
	if end > len(registration.Spec.Resources) {
		end = len(registration.Spec.Resources)
	}
	items := append([]SkillResourceSpec(nil), registration.Spec.Resources[start:end]...)
	page := SkillResourcePage{Request: request, Items: items, Complete: end == len(registration.Spec.Resources)}
	if !page.Complete {
		page.NextCursor = skillResourceCursor(registration, end)
	}
	return page, nil
}

func (registry *SkillRegistry) NewResourceLoadRequest(
	id string,
	version string,
	resourceID string,
) (SkillResourceLoadRequest, error) {
	registration, found := registry.Lookup(id, version)
	if !found {
		return SkillResourceLoadRequest{}, skillContractError(ErrorSkillNotFound, "identity")
	}
	index := sort.Search(len(registration.Spec.Resources), func(index int) bool {
		return registration.Spec.Resources[index].ID >= resourceID
	})
	if index == len(registration.Spec.Resources) || registration.Spec.Resources[index].ID != resourceID {
		return SkillResourceLoadRequest{}, skillContractError(ErrorSkillResourceNotFound, "resource_id")
	}
	resource := registration.Spec.Resources[index]
	return SkillResourceLoadRequest{
		SkillID: registration.Spec.ID, SkillVersion: registration.Spec.Version,
		RegistrationDigest: registration.Digest, ScopesDigest: skillScopesDigest(registration.Spec.Scopes),
		ResourceID: resource.ID, MediaType: resource.MediaType,
		ContentRef: resource.ContentRef, ContentDigest: resource.ContentDigest,
		SizeBytes: resource.SizeBytes, MaxBytes: MaxSkillResourceBytes,
	}, nil
}

func (registry *SkillRegistry) ValidateLoadedResource(
	request SkillResourceLoadRequest,
	content []byte,
) ([]byte, error) {
	canonical, err := registry.NewResourceLoadRequest(request.SkillID, request.SkillVersion, request.ResourceID)
	if err != nil || canonical != request {
		return nil, skillContractError(ErrorSkillResourceLoadInvalid, "request")
	}
	if int64(len(content)) != request.SizeBytes || int64(len(content)) > request.MaxBytes ||
		skillContentDigest(content) != request.ContentDigest {
		return nil, skillContractError(ErrorSkillResourceContentInvalid, "content")
	}
	return append([]byte(nil), content...), nil
}

func canonicalSkillResources(
	source []SkillResourceSpec,
	contents []SkillResourceContent,
) ([]SkillResourceSpec, error) {
	if len(source) > MaxSkillResourcesPerSkill || len(contents) > MaxSkillResourcesPerSkill {
		return nil, skillContractError(ErrorSkillResourceInvalid, "resource_count")
	}
	resources := append([]SkillResourceSpec(nil), source...)
	sort.Slice(resources, func(left, right int) bool { return resources[left].ID < resources[right].ID })
	byID := make(map[string][]byte, len(contents))
	for _, content := range contents {
		if !validSkillID(content.ID) {
			return nil, skillContractError(ErrorSkillResourceInvalid, "content_id")
		}
		if _, duplicate := byID[content.ID]; duplicate {
			return nil, skillContractError(ErrorSkillResourceInvalid, "content_duplicate")
		}
		byID[content.ID] = content.Content
	}
	if len(resources) != len(byID) {
		return nil, skillContractError(ErrorSkillResourceInvalid, "content_set")
	}
	for index, resource := range resources {
		if index > 0 && resource.ID == resources[index-1].ID {
			return nil, skillContractError(ErrorSkillResourceInvalid, "resource_duplicate")
		}
		mediaType, parameters, err := mime.ParseMediaType(resource.MediaType)
		content, found := byID[resource.ID]
		if !validSkillID(resource.ID) || err != nil || len(parameters) != 0 ||
			mediaType != resource.MediaType || mediaType != strings.ToLower(mediaType) ||
			!validSkillDigest(resource.ContentDigest) ||
			resource.ContentRef != "artifact:"+resource.ContentDigest ||
			resource.SizeBytes <= 0 || resource.SizeBytes > MaxSkillResourceBytes || !found ||
			int64(len(content)) != resource.SizeBytes || skillContentDigest(content) != resource.ContentDigest {
			return nil, skillContractError(ErrorSkillResourceInvalid, "resource")
		}
	}
	return resources, nil
}

func resourcePageStart(registration SkillRegistration, cursor string) int {
	if cursor == "" {
		return 0
	}
	for index := 1; index < len(registration.Spec.Resources); index++ {
		if cursor == skillResourceCursor(registration, index) {
			return index
		}
	}
	return -1
}

func skillResourceCursor(registration SkillRegistration, end int) string {
	resourceID := registration.Spec.Resources[end-1].ID
	subject := "orquesta.skill-resource-page.v1\x00" + registration.Spec.ID + "\x00" +
		registration.Spec.Version + "\x00" + registration.Digest + "\x00" + resourceID
	return "skill-resource-cursor:" + strings.TrimPrefix(skillContentDigest([]byte(subject)), "sha256:")
}

func skillScopesDigest(scopes []SkillScope) string {
	encoded, err := json.Marshal(scopes)
	if err != nil {
		panic("canonical skill scopes cannot fail JSON encoding: " + err.Error())
	}
	return skillContentDigest(encoded)
}
