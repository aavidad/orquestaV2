package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const codexGoalWorkspaceIndexMaxBytesV1 = 64 << 10

func openCodexGoalWorkspaceIndexDirV1(path string, create bool) (*os.File, error) {
	dir, err := openPrivateIntentManifestDirV0(path, create)
	if err != nil {
		return nil, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	return dir, nil
}

func openCodexGoalWorkspaceIndexLockV1(dir *os.File, name string) (*os.File, error) {
	if dir == nil || strings.TrimSpace(name) == "" || strings.ContainsAny(name, `/\\`) {
		return nil, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	fd, err := syscall.Openat(int(dir.Fd()), name, syscall.O_RDWR|syscall.O_CREAT|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0o600)
	if err != nil {
		return nil, err
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil ||
		stat.Mode&syscall.S_IFMT != syscall.S_IFREG ||
		stat.Mode&0o777 != 0o600 ||
		stat.Uid != uint32(os.Geteuid()) ||
		stat.Nlink != 1 {
		_ = syscall.Close(fd)
		return nil, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	file := os.NewFile(uintptr(fd), name)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	return file, nil
}

func lockCodexGoalWorkspaceIndexContextV1(ctx context.Context, file *os.File) error {
	if ctx == nil || file == nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func readCodexGoalWorkspaceIndexV1(dir *os.File, name string) ([]byte, error) {
	if dir == nil || strings.TrimSpace(name) == "" || strings.ContainsAny(name, `/\\`) {
		return nil, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	fd, err := syscall.Openat(int(dir.Fd()), name, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil ||
		stat.Mode&syscall.S_IFMT != syscall.S_IFREG ||
		stat.Mode&0o777 != 0o600 ||
		stat.Uid != uint32(os.Geteuid()) ||
		stat.Nlink != 1 ||
		stat.Size < 0 || stat.Size > codexGoalWorkspaceIndexMaxBytesV1 {
		_ = syscall.Close(fd)
		return nil, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	file := os.NewFile(uintptr(fd), name)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, codexGoalWorkspaceIndexMaxBytesV1+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(raw) > codexGoalWorkspaceIndexMaxBytesV1 {
		return nil, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	return raw, nil
}

func writeCodexGoalWorkspaceIndexV1(dir *os.File, name string, data []byte) error {
	if dir == nil || strings.TrimSpace(name) == "" || strings.ContainsAny(name, `/\\`) || len(data) == 0 || len(data) > codexGoalWorkspaceIndexMaxBytesV1 {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	suffix, err := intentManifestTempSuffixV0()
	if err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	tmpName := "." + name + ".tmp-" + suffix
	fd, err := syscall.Openat(int(dir.Fd()), tmpName, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0o600)
	if err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	tmpPresent := true
	defer func() {
		if tmpPresent {
			_ = syscall.Unlinkat(int(dir.Fd()), tmpName)
		}
	}()
	file := os.NewFile(uintptr(fd), tmpName)
	if file == nil {
		_ = syscall.Close(fd)
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	written, writeErr := file.Write(data)
	if writeErr == nil && written != len(data) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	if err := syscall.Renameat(int(dir.Fd()), tmpName, int(dir.Fd()), name); err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	tmpPresent = false
	if err := dir.Sync(); err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	return nil
}

func decodeCodexGoalWorkspaceIndexV1(raw []byte) (codexGoalWorkspaceAdapterIndexV0, error) {
	if len(raw) == 0 || len(raw) > codexGoalWorkspaceIndexMaxBytesV1 || !codexGoalWorkspaceJSONKeysUniqueV1(raw) {
		return codexGoalWorkspaceAdapterIndexV0{}, errCodexGoalWorkspaceAdapterConflictV0
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var index codexGoalWorkspaceAdapterIndexV0
	if err := decoder.Decode(&index); err != nil {
		return codexGoalWorkspaceAdapterIndexV0{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return codexGoalWorkspaceAdapterIndexV0{}, errCodexGoalWorkspaceAdapterConflictV0
	}
	return index, nil
}

func validateCodexGoalWorkspaceIndexV1(index codexGoalWorkspaceAdapterIndexV0, goalRef string) bool {
	goalRef = strings.TrimSpace(goalRef)
	identity := index.Identity
	if goalRef == "" || index.GoalRef != goalRef || identity.GoalRef != goalRef ||
		identity.RunRef == "" || identity.ProjectRef == "" || identity.WorktreeRef == "" ||
		identity.RunRef != strings.TrimSpace(identity.RunRef) ||
		identity.ProjectRef != strings.TrimSpace(identity.ProjectRef) ||
		identity.WorktreeRef != strings.TrimSpace(identity.WorktreeRef) {
		return false
	}
	switch index.SchemaVersion {
	case codexGoalWorkspaceAdapterIndexSchemaV0:
		return identity.IntentManifestRef == "" && identity.IntentManifestSHA256 == "" &&
			identity.SourceGoalRef == "" && identity.WriteSetSHA256 == "" && identity.ReworkPolicySHA256 == "" &&
			identity.WorkspaceRef == "" && identity.ProviderRef == "" && identity.RuntimeGenerationRef == "" && identity.ExternalGoalRef == ""
	case codexGoalWorkspaceAdapterIndexSchemaV1:
		if identity.WorkspaceRef != "" || identity.ProviderRef != "" || identity.RuntimeGenerationRef != "" || identity.ExternalGoalRef != "" {
			return false
		}
		return validateCodexGoalWorkspaceIndexIdentityV1(identity)
	case codexGoalWorkspaceAdapterIndexSchemaV2:
		if !validateCodexGoalWorkspaceIndexIdentityV1(identity) ||
			identity.WorkspaceRef != orquestagoal.GoalWorkspaceRefForGoalV0(goalRef) ||
			identity.ProviderRef != strings.TrimSpace(identity.ProviderRef) ||
			identity.RuntimeGenerationRef != strings.TrimSpace(identity.RuntimeGenerationRef) ||
			identity.ExternalGoalRef != strings.TrimSpace(identity.ExternalGoalRef) {
			return false
		}
		if identity.ProviderRef == "" && identity.RuntimeGenerationRef == "" && identity.ExternalGoalRef == "" {
			return true
		}
		if identity.ProviderRef == "" || identity.RuntimeGenerationRef == "" || identity.ExternalGoalRef == "" {
			return false
		}
		if len(orquestagoal.ValidateGoalWorkspaceAuthorityV0(
			goalRef,
			orquestagoal.GoalWorkspaceAuthoritySchemaV0,
			identity.WorkspaceRef,
			identity.ProviderRef,
			identity.RuntimeGenerationRef,
			false,
		)) != 0 {
			return false
		}
		if len(orquestagoal.ValidateGoalObservationRequestV0(orquestagoal.GoalObservationRequestV0{
			GoalRef: goalRef, ExternalGoalRef: identity.ExternalGoalRef,
			IntentManifestRef: identity.IntentManifestRef, IntentManifestSHA256: identity.IntentManifestSHA256,
			WorkspaceAuthoritySchemaVersion: orquestagoal.GoalWorkspaceAuthoritySchemaV0,
			WorkspaceRef:                    identity.WorkspaceRef, ProviderRef: identity.ProviderRef,
			RuntimeGenerationRef: identity.RuntimeGenerationRef,
		})) != 0 {
			return false
		}
		return true
	default:
		return false
	}
}

func validateCodexGoalWorkspaceIndexIdentityV1(identity codexGoalWorkspaceRequestIdentityV0) bool {
	if !codexGoalWorkspaceLowerSHA256V1(identity.WriteSetSHA256) ||
		!codexGoalWorkspaceLowerSHA256V1(identity.ReworkPolicySHA256) {
		return false
	}
	manifestPresent := identity.IntentManifestRef != "" || identity.IntentManifestSHA256 != ""
	if manifestPresent && (identity.IntentManifestRef == "" ||
		!codexGoalWorkspaceLowerSHA256V1(identity.IntentManifestSHA256) ||
		strings.ContainsAny(identity.IntentManifestRef, `/\\`) ||
		strings.Contains(identity.IntentManifestRef, "..")) {
		return false
	}
	return identity.SourceGoalRef == strings.TrimSpace(identity.SourceGoalRef)
}

func codexGoalWorkspaceLowerSHA256V1(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func codexGoalWorkspaceJSONKeysUniqueV1(raw []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if !codexGoalWorkspaceJSONValueKeysUniqueV1(decoder) {
		return false
	}
	_, err := decoder.Token()
	return errors.Is(err, io.EOF)
}

func codexGoalWorkspaceJSONValueKeysUniqueV1(decoder *json.Decoder) bool {
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	delim, compound := token.(json.Delim)
	if !compound {
		return true
	}
	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			key, ok := keyToken.(string)
			if err != nil || !ok {
				return false
			}
			if _, duplicate := seen[key]; duplicate {
				return false
			}
			seen[key] = struct{}{}
			if !codexGoalWorkspaceJSONValueKeysUniqueV1(decoder) {
				return false
			}
		}
		end, err := decoder.Token()
		return err == nil && end == json.Delim('}')
	case '[':
		for decoder.More() {
			if !codexGoalWorkspaceJSONValueKeysUniqueV1(decoder) {
				return false
			}
		}
		end, err := decoder.Token()
		return err == nil && end == json.Delim(']')
	default:
		return false
	}
}
