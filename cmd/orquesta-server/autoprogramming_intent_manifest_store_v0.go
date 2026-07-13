package main

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestasecurefile "orquesta/modulos/orquesta-secure-file"
)

var errAutoprogrammingIntentManifestConflictV0 = errors.New("autoprogramming_intent_manifest_conflict")
var errAutoprogrammingIntentManifestUnavailableV0 = errors.New("autoprogramming_intent_manifest_unavailable")

const maxAutoprogrammingIntentManifestBytesV0 = orquestaautoprogramming.AutoprogrammingIntentManifestMaxBytesV0

// serverAutoprogrammingIntentManifestStoreV0 is append-only by request_ref.
// The store contains raw request bytes, never compacted projections.
type serverAutoprogrammingIntentManifestStoreV0 struct{ RootDir string }

func (store serverAutoprogrammingIntentManifestStoreV0) CreateAutoprogrammingIntentManifestIfAbsentV0(ctx context.Context, manifest orquestaautoprogramming.AutoprogrammingIntentManifestV0) (orquestaautoprogramming.AutoprogrammingIntentManifestV0, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, err
		}
	}
	manifest = orquestaautoprogramming.NormalizeAutoprogrammingIntentManifestV0(manifest)
	if len(manifest.RequestJSON) > maxAutoprogrammingIntentManifestBytesV0 {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, errAutoprogrammingIntentManifestUnavailableV0
	}
	if issues := orquestaautoprogramming.ValidateAutoprogrammingIntentManifestV0(manifest); len(issues) != 0 {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, errAutoprogrammingIntentManifestUnavailableV0
	}
	path, root, err := store.pathV0(manifest.RequestRef)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, err
	}
	dir, err := openPrivateIntentManifestDirV0(root, true)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, errAutoprogrammingIntentManifestUnavailableV0
	}
	defer dir.Close()
	name := filepath.Base(path)
	if existing, found, err := store.loadAtDirV0(ctx, dir, name, manifest.RequestRef); err != nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, err
	} else if found {
		if existing.RequestSHA256 != manifest.RequestSHA256 || !bytesEqualIntentManifestV0(existing.RequestJSON, manifest.RequestJSON) {
			return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, errAutoprogrammingIntentManifestConflictV0
		}
		return existing, nil
	}
	raw, _, err := orquestasecurefile.CreateFileIfAbsentAtV0(dir, name, manifest.RequestJSON, orquestasecurefile.FileOptionsV0{
		MaxBytes:  maxAutoprogrammingIntentManifestBytesV0,
		ExactMode: 0o400,
	})
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, errAutoprogrammingIntentManifestUnavailableV0
	}
	stored, issues := orquestaautoprogramming.AutoprogrammingIntentManifestFromRequestJSONV0(raw)
	if len(issues) != 0 || stored.RequestRef != manifest.RequestRef || stored.RequestSHA256 != manifest.RequestSHA256 || !bytesEqualIntentManifestV0(stored.RequestJSON, manifest.RequestJSON) {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, errAutoprogrammingIntentManifestConflictV0
	}
	return stored, nil
}

func (store serverAutoprogrammingIntentManifestStoreV0) LoadAutoprogrammingIntentManifestV0(ctx context.Context, requestRef string) (orquestaautoprogramming.AutoprogrammingIntentManifestV0, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, err
		}
	}
	path, root, err := store.pathV0(requestRef)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, err
	}
	dir, err := openPrivateIntentManifestDirV0(root, false)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, errAutoprogrammingIntentManifestUnavailableV0
	}
	defer dir.Close()
	manifest, found, err := store.loadAtDirV0(ctx, dir, filepath.Base(path), requestRef)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, errAutoprogrammingIntentManifestUnavailableV0
	}
	if !found {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, orquestaautoprogramming.ErrAutoprogrammingIntentManifestNotFoundV0
	}
	return manifest, nil
}

func (store serverAutoprogrammingIntentManifestStoreV0) loadAtDirV0(ctx context.Context, dir *os.File, name, requestRef string) (orquestaautoprogramming.AutoprogrammingIntentManifestV0, bool, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, false, err
		}
	}
	raw, err := readIntentManifestAtV0(dir, name)
	if errors.Is(err, os.ErrNotExist) {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, false, nil
	}
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, false, errAutoprogrammingIntentManifestUnavailableV0
	}
	manifest, issues := orquestaautoprogramming.AutoprogrammingIntentManifestFromRequestJSONV0(raw)
	if len(issues) != 0 || manifest.RequestRef != strings.TrimSpace(requestRef) {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, false, errAutoprogrammingIntentManifestConflictV0
	}
	return manifest, true, nil
}

func (store serverAutoprogrammingIntentManifestStoreV0) pathV0(requestRef string) (string, string, error) {
	requestRef = strings.TrimSpace(requestRef)
	root := strings.TrimSpace(store.RootDir)
	if root == "" || requestRef == "" || strings.ContainsAny(requestRef, `/\\`) || strings.Contains(requestRef, "..") {
		return "", "", errAutoprogrammingIntentManifestUnavailableV0
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", errAutoprogrammingIntentManifestUnavailableV0
	}
	path := filepath.Join(filepath.Clean(absRoot), requestRef+".json")
	rel, err := filepath.Rel(absRoot, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", "", errAutoprogrammingIntentManifestUnavailableV0
	}
	return path, absRoot, nil
}

func intentManifestTempSuffixV0() (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	const hex = "0123456789abcdef"
	out := make([]byte, len(raw)*2)
	for i := range raw {
		out[i*2], out[i*2+1] = hex[raw[i]>>4], hex[raw[i]&0x0f]
	}
	return string(out), nil
}

func openPrivateIntentManifestDirV0(path string, create bool) (*os.File, error) {
	abs, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil || strings.TrimSpace(path) == "" {
		return nil, errAutoprogrammingIntentManifestUnavailableV0
	}
	dir, err := orquestasecurefile.OpenDirectoryV0(filepath.Clean(abs), orquestasecurefile.DirectoryOptionsV0{
		Create:     create,
		CreateMode: 0o700,
		FinalMode:  0o700,
	})
	if err != nil {
		return nil, errAutoprogrammingIntentManifestUnavailableV0
	}
	return dir, nil
}

func readIntentManifestAtV0(dir *os.File, name string) ([]byte, error) {
	raw, err := orquestasecurefile.ReadFileAtV0(dir, name, orquestasecurefile.FileOptionsV0{
		MaxBytes:  maxAutoprogrammingIntentManifestBytesV0,
		ExactMode: 0o400,
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, errAutoprogrammingIntentManifestUnavailableV0
	}
	return raw, err
}

func bytesEqualIntentManifestV0(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	var delta byte
	for i := range left {
		delta |= left[i] ^ right[i]
	}
	return delta == 0
}
