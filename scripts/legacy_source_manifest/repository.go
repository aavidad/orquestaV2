// Este fichero inspecciona revisiones y referencias de repositorios Git.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func inspectRepository(path string, bareCandidate bool, timeout time.Duration) sourceRecord {
	kind := "git_repository"
	if bareCandidate {
		kind = "git_bare_repository"
	}
	if code := repositoryAccessError(path, bareCandidate); code != "" {
		return sealSource(sourceRecord{
			Path: path, Kind: kind, Status: "error", ErrorCode: code,
			RequiresPhysicalInventory: true,
		})
	}

	objectFormat, err := gitText(timeout, path, "rev-parse", "--show-object-format")
	if err != nil {
		return sealSource(sourceRecord{
			Path: path, Kind: kind, Status: "excluded", Reason: "invalid_git_repository",
			RequiresPhysicalInventory: true,
		})
	}
	isBare, err := gitText(timeout, path, "rev-parse", "--is-bare-repository")
	if err != nil {
		return gitInspectionError(path, kind, err)
	}
	if isBare == "true" {
		kind = "git_bare_repository"
	} else if marker, markerErr := os.Lstat(filepath.Join(path, ".git")); markerErr == nil &&
		marker.Mode().IsRegular() {
		kind = "git_worktree"
	}

	var statusBefore []byte
	if kind != "git_bare_repository" {
		statusBefore, err = repositoryWorktreeStatus(timeout, path)
		if err != nil {
			return unassessedWorktree(path, kind, "git_worktree_status_failed")
		}
	}
	gitDirectory, err := gitText(timeout, path, "rev-parse", "--path-format=absolute", "--git-dir")
	if err != nil {
		return gitInspectionError(path, kind, err)
	}
	commonDirectory, err := gitText(timeout, path, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return gitInspectionError(path, kind, err)
	}
	refsBefore, err := repositoryReferences(timeout, path)
	if err != nil {
		return gitInspectionError(path, kind, err)
	}
	headReference, headObject := repositoryHead(timeout, path)
	commitCountText, err := gitText(timeout, path, "rev-list", "--all", "--count")
	if err != nil {
		return gitInspectionError(path, kind, err)
	}
	commitCount, err := strconv.Atoi(commitCountText)
	if err != nil || commitCount < 0 {
		return sealSource(sourceRecord{
			Path: path, Kind: kind, Status: "error", ErrorCode: "git_invalid_commit_count",
			RequiresPhysicalInventory: true,
		})
	}
	refsAfter, err := repositoryReferences(timeout, path)
	if err != nil {
		return gitInspectionError(path, kind, err)
	}
	finalHeadReference, finalHeadObject := repositoryHead(timeout, path)
	if !equalReferences(refsBefore, refsAfter) ||
		headReference != finalHeadReference || headObject != finalHeadObject {
		return sealSource(sourceRecord{
			Path: path, Kind: kind, Status: "error",
			ErrorCode:                 "git_references_changed_during_scan",
			RequiresPhysicalInventory: true,
		})
	}
	worktreeState := ""
	worktreeStatusSHA := ""
	worktreeChangeCount := 0
	// Ni las refs ni un estado Git limpio cubren hooks, configuración, reflogs,
	// ficheros ignorados y demás hechos físicos del repositorio.
	requiresPhysicalInventory := true
	if kind != "git_bare_repository" {
		statusAfter, statusErr := repositoryWorktreeStatus(timeout, path)
		if statusErr != nil {
			return unassessedWorktree(path, kind, "git_worktree_status_failed")
		}
		if !bytes.Equal(statusBefore, statusAfter) {
			return unassessedWorktree(path, kind, "git_worktree_changed_during_scan")
		}
		worktreeStatusSHA = digestBytes("orquesta.git-worktree-status.v1", statusBefore)
		worktreeChangeCount = countWorktreeChanges(statusBefore)
		worktreeState = "clean"
		if worktreeChangeCount > 0 {
			worktreeState = "dirty"
		}
	}
	return sealSource(sourceRecord{
		Path:                      path,
		Kind:                      kind,
		Status:                    "included",
		GitDirectory:              normalizeGitPath(path, gitDirectory),
		GitCommonDirectory:        normalizeGitPath(path, commonDirectory),
		ObjectFormat:              objectFormat,
		HeadReference:             headReference,
		HeadObject:                headObject,
		References:                refsBefore,
		ReferenceSHA256:           digestJSON("orquesta.git-references.v1", refsBefore),
		ReachableCommitCount:      &commitCount,
		WorktreeState:             worktreeState,
		WorktreeStatusSHA256:      worktreeStatusSHA,
		WorktreeChangeCount:       worktreeChangeCount,
		RequiresPhysicalInventory: requiresPhysicalInventory,
	})
}

func repositoryWorktreeStatus(timeout time.Duration, path string) ([]byte, error) {
	return runGit(
		timeout,
		path,
		"status",
		"--porcelain=v2",
		"-z",
		"--untracked-files=all",
		"--ignore-submodules=none",
	)
}

func countWorktreeChanges(status []byte) int {
	count := 0
	for _, entry := range bytes.Split(status, []byte{0}) {
		if len(entry) < 2 {
			continue
		}
		switch entry[0] {
		case '1', '2', 'u', '?':
			if entry[1] == ' ' {
				count++
			}
		}
	}
	return count
}

func unassessedWorktree(path, kind, code string) sourceRecord {
	return sealSource(sourceRecord{
		Path:                      path,
		Kind:                      kind,
		Status:                    "error",
		ErrorCode:                 code,
		WorktreeState:             "dirty_state_unassessed",
		RequiresPhysicalInventory: true,
	})
}

func repositoryAccessError(path string, bare bool) string {
	marker := filepath.Join(path, ".git")
	if bare {
		marker = filepath.Join(path, "HEAD")
	}
	info, err := os.Lstat(marker)
	if err != nil {
		return filesystemErrorCode(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "git_marker_symbolic_link"
	}
	if info.Mode().Perm()&0o444 == 0 {
		return "permission_denied"
	}
	if info.IsDir() && info.Mode().Perm()&0o111 == 0 {
		return "permission_denied"
	}
	file, err := os.Open(marker)
	if err != nil {
		return filesystemErrorCode(err)
	}
	if info.IsDir() {
		_, readErr := file.Readdirnames(1)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			_ = file.Close()
			return filesystemErrorCode(readErr)
		}
	}
	if err := file.Close(); err != nil {
		return filesystemErrorCode(err)
	}
	return ""
}

func repositoryReferences(timeout time.Duration, path string) ([]reference, error) {
	output, err := runGit(
		timeout,
		path,
		"for-each-ref",
		"--sort=refname",
		"--format=%(refname)%09%(objectname)%09%(objecttype)%09%(*objectname)%09%(symref)",
	)
	if err != nil {
		return nil, err
	}
	var refs []reference
	for _, line := range strings.Split(strings.TrimSuffix(string(output), "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 5 || fields[0] == "" || fields[1] == "" {
			return nil, errors.New("referencia Git inválida")
		}
		refs = append(refs, reference{
			Name: fields[0], Object: fields[1], ObjectType: fields[2],
			PeeledObject: fields[3], Symbolic: fields[4],
		})
	}
	sortReferences(refs)
	return refs, nil
}

func repositoryHead(timeout time.Duration, path string) (string, string) {
	headReference, err := gitText(timeout, path, "symbolic-ref", "-q", "HEAD")
	if err != nil {
		headReference = ""
	}
	headObject, err := gitText(timeout, path, "rev-parse", "--verify", "HEAD")
	if err != nil {
		headObject = ""
	}
	return headReference, headObject
}

func sortReferences(refs []reference) {
	sort.Slice(refs, func(left, right int) bool {
		if refs[left].Name != refs[right].Name {
			return refs[left].Name < refs[right].Name
		}
		return refs[left].Object < refs[right].Object
	})
}

func equalReferences(left, right []reference) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func referenceObjectFormat(refs []reference) string {
	format := ""
	for _, ref := range refs {
		current := ""
		switch len(ref.Object) {
		case 40:
			current = "sha1"
		case 64:
			current = "sha256"
		default:
			return "unknown"
		}
		if format != "" && format != current {
			return "mixed"
		}
		format = current
	}
	if format == "" {
		return "unknown"
	}
	return format
}
