package tooling

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ResourcePageRequest is derived from one canonical resource registration.
// Cursor is opaque; callers cannot select a URI or spec digest independently.
type ResourcePageRequest struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	URI         string `json:"uri"`
	SpecDigest  string `json:"spec_digest"`
	Cursor      string `json:"cursor,omitempty"`
	SnapshotRef string `json:"snapshot_ref,omitempty"`
	Limit       uint32 `json:"limit"`
}

// ResourcePage repeats the exact request identity and binds all items to one
// snapshot. A subsequent request carries NextCursor without interpreting it.
type ResourcePage struct {
	ID            string            `json:"id"`
	Version       string            `json:"version"`
	URI           string            `json:"uri"`
	SpecDigest    string            `json:"spec_digest"`
	RequestCursor string            `json:"request_cursor,omitempty"`
	SnapshotRef   string            `json:"snapshot_ref"`
	Items         []json.RawMessage `json:"items"`
	NextCursor    string            `json:"next_cursor,omitempty"`
	Complete      bool              `json:"complete"`
}

// NewResourcePageRequest resolves one exact resource and normalizes a zero
// limit to its declared default. It never selects a latest version.
func (catalog *SurfaceCatalog) NewResourcePageRequest(
	id, version, cursor, snapshotRef string,
	limit uint32,
) (ResourcePageRequest, error) {
	registration, found := catalog.LookupResource(id, version)
	if !found {
		return ResourcePageRequest{}, contractError(ErrorSpecNotFound, "resource_identity")
	}
	if !validOptionalResourceToken(cursor) || !validOptionalResourceToken(snapshotRef) ||
		(cursor == "") != (snapshotRef == "") {
		return ResourcePageRequest{}, contractError(ErrorResourceRequestInvalid, "cursor_snapshot")
	}
	if limit == 0 {
		limit = registration.Spec.Page.DefaultItems
	}
	if limit > registration.Spec.Page.MaxItems {
		return ResourcePageRequest{}, contractError(ErrorResourceRequestInvalid, "limit")
	}
	return ResourcePageRequest{
		ID: registration.Spec.ID, Version: registration.Spec.Version,
		URI: registration.Spec.URI, SpecDigest: registration.Digest,
		Cursor: cursor, SnapshotRef: snapshotRef, Limit: limit,
	}, nil
}

// ValidateResourcePage rejects identity, cursor, snapshot, schema, item-count,
// and byte-budget divergence before returning detached canonical items.
func (catalog *SurfaceCatalog) ValidateResourcePage(
	request ResourcePageRequest,
	page ResourcePage,
) (ResourcePage, error) {
	registration, found := catalog.LookupResource(request.ID, request.Version)
	if !found {
		return ResourcePage{}, contractError(ErrorSpecNotFound, "resource_identity")
	}
	canonicalRequest, err := catalog.NewResourcePageRequest(
		request.ID, request.Version, request.Cursor, request.SnapshotRef, request.Limit,
	)
	if err != nil || !validSurfaceDigest(request.SpecDigest) || canonicalRequest != request {
		return ResourcePage{}, contractError(ErrorResourceRequestInvalid, "request")
	}
	if page.ID != request.ID || page.Version != request.Version || page.URI != request.URI ||
		page.SpecDigest != request.SpecDigest || page.RequestCursor != request.Cursor ||
		!validRequiredResourceToken(page.SnapshotRef) ||
		request.SnapshotRef != "" && page.SnapshotRef != request.SnapshotRef {
		return ResourcePage{}, contractError(ErrorResourcePageInvalid, "causality")
	}
	if uint32(len(page.Items)) > request.Limit || uint32(len(page.Items)) > registration.Spec.Page.MaxItems {
		return ResourcePage{}, contractError(ErrorResourcePageInvalid, "item_count")
	}
	if page.Complete && page.NextCursor != "" || !page.Complete &&
		(len(page.Items) == 0 || !validRequiredResourceToken(page.NextCursor) || page.NextCursor == request.Cursor) {
		return ResourcePage{}, contractError(ErrorResourcePageInvalid, "next_cursor")
	}
	canonicalItems := make([]json.RawMessage, len(page.Items))
	for index, item := range page.Items {
		canonical, validateErr := validatePayloadAgainstSchema(registration.Spec.ItemSchema, item)
		if validateErr != nil {
			return ResourcePage{}, contractError(ErrorResourcePageInvalid, "item_schema")
		}
		canonicalItems[index] = canonical
	}
	encoded, err := json.Marshal(canonicalItems)
	if err != nil || int64(len(encoded)) > registration.Spec.Page.MaxBytes ||
		int64(len(encoded)) > registration.Spec.MaxBytes {
		return ResourcePage{}, contractError(ErrorResourcePageInvalid, "page_bytes")
	}
	page.Items = canonicalItems
	return cloneResourcePage(page), nil
}

func validOptionalResourceToken(value string) bool {
	return value == "" || validRequiredResourceToken(value)
}

func validRequiredResourceToken(value string) bool {
	if value == "" || len(value) > 1024 || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func validSurfaceDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func cloneResourcePage(source ResourcePage) ResourcePage {
	items := source.Items
	source.Items = make([]json.RawMessage, len(items))
	for index, item := range items {
		source.Items[index] = append(json.RawMessage(nil), item...)
	}
	return source
}
