package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestasecurefile "orquesta/modulos/orquesta-secure-file"
)

const maxAutoprogrammingPrepareRunIdempotencyClaimBytesV0 = orquestaautoprogramming.AutoprogrammingIntentManifestMaxBytesV0

func (store serverAutoprogrammingIntentManifestStoreV0) CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(
	ctx context.Context,
	claim orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0,
) (orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, err
		}
	}
	claim = orquestaautoprogramming.NormalizeAutoprogrammingPrepareRunIdempotencyClaimV0(claim)
	if len(orquestaautoprogramming.ValidateAutoprogrammingPrepareRunIdempotencyClaimV0(claim)) != 0 {
		return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimInvalidV0
	}
	raw, err := json.Marshal(claim)
	if err != nil || len(raw) > maxAutoprogrammingPrepareRunIdempotencyClaimBytesV0 {
		return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimInvalidV0
	}
	root := filepath.Join(store.RootDir, "prepare-run-idempotency-claims")
	dir, err := openPrivateIntentManifestDirV0(root, true)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimUnavailableV0
	}
	defer dir.Close()
	name := autoprogrammingPrepareRunIdempotencyClaimNameV0(claim.IdempotencyKey)
	if existing, found, err := loadAutoprogrammingPrepareRunIdempotencyClaimAtV0(ctx, dir, name, claim.IdempotencyKey); err != nil {
		return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, err
	} else if found {
		if !orquestaautoprogramming.EqualAutoprogrammingPrepareRunIdempotencyClaimV0(existing, claim) {
			return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimConflictV0
		}
		return existing, nil
	}
	storedRaw, _, err := orquestasecurefile.CreateFileIfAbsentAtV0(dir, name, raw, orquestasecurefile.FileOptionsV0{
		MaxBytes:  maxAutoprogrammingPrepareRunIdempotencyClaimBytesV0,
		ExactMode: 0o400,
	})
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimUnavailableV0
	}
	stored, err := decodeAutoprogrammingPrepareRunIdempotencyClaimV0(storedRaw, name, claim.IdempotencyKey)
	if err != nil || !orquestaautoprogramming.EqualAutoprogrammingPrepareRunIdempotencyClaimV0(stored, claim) {
		return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimConflictV0
	}
	return stored, nil
}

func autoprogrammingPrepareRunIdempotencyClaimNameV0(idempotencyKey string) string {
	digest := sha256.Sum256([]byte(idempotencyKey))
	return hex.EncodeToString(digest[:]) + ".json"
}

func loadAutoprogrammingPrepareRunIdempotencyClaimAtV0(
	ctx context.Context,
	dir *os.File,
	name string,
	idempotencyKey string,
) (orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0, bool, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, false, err
		}
	}
	raw, err := readAutoprogrammingPrepareRunIdempotencyClaimAtV0(dir, name)
	if errors.Is(err, os.ErrNotExist) {
		return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, false, nil
	}
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, false, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimUnavailableV0
	}
	claim, err := decodeAutoprogrammingPrepareRunIdempotencyClaimV0(raw, name, idempotencyKey)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, false, err
	}
	return claim, true, nil
}

func decodeAutoprogrammingPrepareRunIdempotencyClaimV0(raw []byte, name, idempotencyKey string) (orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var claim orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0
	if err := decoder.Decode(&claim); err != nil {
		return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimConflictV0
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimConflictV0
	}
	claim = orquestaautoprogramming.NormalizeAutoprogrammingPrepareRunIdempotencyClaimV0(claim)
	canonicalRaw, marshalErr := json.Marshal(claim)
	if marshalErr != nil || !bytes.Equal(raw, canonicalRaw) || claim.IdempotencyKey != idempotencyKey || name != autoprogrammingPrepareRunIdempotencyClaimNameV0(claim.IdempotencyKey) || len(orquestaautoprogramming.ValidateAutoprogrammingPrepareRunIdempotencyClaimV0(claim)) != 0 {
		return orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0{}, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimConflictV0
	}
	return claim, nil
}

func readAutoprogrammingPrepareRunIdempotencyClaimAtV0(dir *os.File, name string) ([]byte, error) {
	return orquestasecurefile.ReadFileAtV0(dir, name, orquestasecurefile.FileOptionsV0{
		MaxBytes:  maxAutoprogrammingPrepareRunIdempotencyClaimBytesV0,
		ExactMode: 0o400,
	})
}
