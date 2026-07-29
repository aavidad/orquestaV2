package localtoken

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

const (
	manifestDocumentType = "orquesta.local_token_principals"
	manifestSchema       = 1
)

type ManifestOptions struct {
	Path             string
	OwnerUID         int
	MaxDocumentBytes int64
	MaxEntries       int
	Reserved         []identity.Principal
}

type manifestDocument struct {
	SchemaVersion int             `json:"schema_version"`
	DocumentType  string          `json:"document_type"`
	Principals    []manifestEntry `json:"principals"`
}

type manifestEntry struct {
	PrincipalRef string `json:"principal_ref"`
	ActorRef     string `json:"actor_ref"`
	TokenPath    string `json:"token_path"`
}

type resolvedManifestEntry struct {
	principal identity.Principal
	tokenPath string
}

// OpenManifest loads only authentication bindings. It never grants a role or
// provisions project membership; RBAC remains an explicit application command.
func OpenManifest(options ManifestOptions) (*Authenticator, error) {
	path, err := normalizePath(options.Path)
	if err != nil || options.OwnerUID < 0 || options.MaxDocumentBytes <= 0 || options.MaxEntries <= 0 {
		return nil, &Error{Code: CodeManifestInvalid, Cause: err}
	}
	content, err := readManifest(path, uint32(options.OwnerUID), options.MaxDocumentBytes)
	if err != nil {
		return nil, err
	}
	defer clear(content)
	var document manifestDocument
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return nil, &Error{Code: CodeManifestInvalid}
	}
	if err := requireManifestEOF(decoder); err != nil ||
		document.SchemaVersion != manifestSchema ||
		document.DocumentType != manifestDocumentType ||
		len(document.Principals) == 0 || len(document.Principals) > options.MaxEntries {
		return nil, &Error{Code: CodeManifestInvalid}
	}

	directory := filepath.Dir(path)
	resolved := make([]resolvedManifestEntry, 0, len(document.Principals))
	principalRefs := make(map[string]struct{}, len(document.Principals))
	actorRefs := make(map[string]struct{}, len(document.Principals))
	tokenPaths := make(map[string]struct{}, len(document.Principals))
	for _, principal := range options.Reserved {
		if identity.ValidatePrincipal(principal) != nil || principal.Method != AuthenticationMethod {
			return nil, &Error{Code: CodeManifestInvalid}
		}
		principalRefs[principal.Ref.String()] = struct{}{}
		actorRefs[principal.ActorRef.String()] = struct{}{}
	}
	for _, entry := range document.Principals {
		principalRef, principalErr := identity.NewPrincipalRef(entry.PrincipalRef)
		actorRef, actorErr := goal.NewActorRef(entry.ActorRef)
		tokenPath, pathErr := manifestTokenPath(directory, entry.TokenPath)
		if principalErr != nil || actorErr != nil || pathErr != nil || tokenPath == path ||
			principalRef.String() != actorRef.String() {
			return nil, &Error{Code: CodeManifestInvalid}
		}
		if _, duplicate := principalRefs[principalRef.String()]; duplicate {
			return nil, &Error{Code: CodeManifestInvalid}
		}
		if _, duplicate := actorRefs[actorRef.String()]; duplicate {
			return nil, &Error{Code: CodeManifestInvalid}
		}
		if _, duplicate := tokenPaths[tokenPath]; duplicate {
			return nil, &Error{Code: CodeManifestInvalid}
		}
		principalRefs[principalRef.String()] = struct{}{}
		actorRefs[actorRef.String()] = struct{}{}
		tokenPaths[tokenPath] = struct{}{}

		principal, principalErr := identity.NewPrincipal(
			principalRef, actorRef, identity.PrincipalKindHuman, AuthenticationMethod,
		)
		if principalErr != nil {
			return nil, &Error{Code: CodeManifestInvalid}
		}
		resolved = append(resolved, resolvedManifestEntry{principal: principal, tokenPath: tokenPath})
	}

	// No token path is read until every entry has passed strict structural,
	// identity, uniqueness, traversal, capacity, and reserved-identity
	// validation. Secondary credentials are provisioned explicitly so startup
	// cannot leave a partially created credential set.
	providers := make([]*Authenticator, 0, len(resolved))
	for _, entry := range resolved {
		credential, credentialErr := openExisting(entry.tokenPath, options.OwnerUID)
		if credentialErr != nil {
			return nil, credentialErr
		}
		provider, bindErr := credential.ForPrincipal(entry.principal)
		if bindErr != nil {
			return nil, bindErr
		}
		providers = append(providers, provider)
	}
	return Combine(providers...)
}

func readManifest(path string, ownerUID uint32, maximum int64) ([]byte, error) {
	directory := filepath.Dir(path)
	if err := validateDirectoryChain(directory); err != nil {
		return nil, &Error{Code: CodeManifestInvalid}
	}
	directoryInfo, err := os.Lstat(directory)
	if err != nil || !privateDirectory(directoryInfo, ownerUID) {
		return nil, &Error{Code: CodeManifestPermissions}
	}
	before, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, &Error{Code: CodeManifestInvalid}
	}
	if err != nil || !privateRegular(before, ownerUID, maximum) {
		if err == nil && before.Mode().IsRegular() && before.Size() > maximum {
			return nil, &Error{Code: CodeManifestTooLarge}
		}
		if err == nil && before.Mode().IsRegular() && before.Mode().Perm() != 0o600 {
			return nil, &Error{Code: CodeManifestPermissions}
		}
		return nil, &Error{Code: CodeManifestInvalid}
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, &Error{Code: CodeManifestInvalid}
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !privateRegular(opened, ownerUID, maximum) || !os.SameFile(before, opened) {
		return nil, &Error{Code: CodeManifestInvalid}
	}
	content, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(content)) > maximum || int64(len(content)) != opened.Size() {
		clear(content)
		if int64(len(content)) > maximum {
			return nil, &Error{Code: CodeManifestTooLarge}
		}
		return nil, &Error{Code: CodeManifestInvalid}
	}
	final, err := file.Stat()
	if err != nil || !privateRegular(final, ownerUID, maximum) || !os.SameFile(opened, final) ||
		final.Size() != opened.Size() || !final.ModTime().Equal(opened.ModTime()) {
		clear(content)
		return nil, &Error{Code: CodeManifestInvalid}
	}
	after, err := os.Lstat(path)
	if err != nil || !privateRegular(after, ownerUID, maximum) || !os.SameFile(final, after) {
		clear(content)
		return nil, &Error{Code: CodeManifestInvalid}
	}
	return content, nil
}

func manifestTokenPath(directory, raw string) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw || strings.ContainsRune(raw, '\x00') ||
		filepath.IsAbs(raw) {
		return "", errors.New("localtoken.manifest_token_path_invalid")
	}
	cleaned := filepath.Clean(raw)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", errors.New("localtoken.manifest_token_path_invalid")
	}
	absolute, err := filepath.Abs(filepath.Join(directory, cleaned))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(directory, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("localtoken.manifest_token_path_invalid")
	}
	return absolute, nil
}

func requireManifestEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("localtoken.manifest_trailing_value")
	}
	return err
}
