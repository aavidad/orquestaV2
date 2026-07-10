package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	serverCanonicalGitRemoteURLV0         = "git@github.com:aavidad/orquestador.git"
	serverCanonicalGitRefV0               = "trabajo/plataforma-agentes"
	serverWorktreeIdentityMissingV0       = "worktree_missing"
	serverWorktreeIdentityInvalidV0       = "worktree_invalid"
	serverWorktreeIdentitySymlinkV0       = "worktree_symlink_not_allowed"
	serverWorktreeIdentityRetiredV0       = "worktree_retired"
	serverWorktreeIdentityStaleV0         = "worktree_stale"
	serverWorktreeIdentityNotAlignedV0    = "worktree_not_aligned"
	serverWorktreeIdentityRemoteV0        = "worktree_remote_not_canonical"
	serverWorktreeIdentityRefV0           = "worktree_ref_not_canonical"
	serverWorktreeIdentityBinaryV0        = "runtime_binary_identity_invalid"
	serverWorktreeIdentityBuildCommitV0   = "runtime_build_commit_mismatch"
	serverWorktreeIdentityBuildModifiedV0 = "runtime_build_not_reproducible"
)

type serverWorktreeIdentityIssueV0 struct{ Code string }

type serverIdentityPreflightV0 struct {
	ProjectWorkDir  string
	WorktreeDir     string
	RuntimeIdentity orquestaserver.ServerRuntimeIdentityV0
	Issue           *serverWorktreeIdentityIssueV0
}

func serverIdentityPreflightFromEnvV0(projectConfigPath string) (serverIdentityPreflightV0, error) {
	projectDir, err := projectDirFromEnvOrProjectConfigPathV0(projectConfigPath)
	if err != nil {
		return serverIdentityPreflightV0{}, err
	}
	identity := serverRuntimeIdentityFromExecutableV0()
	return serverIdentityPreflightForWorkdirsV0(
		context.Background(),
		projectDir,
		serverWorktreeDirFromEnvV0(projectDir),
		identity,
	), nil
}

// serverIdentityPreflightForWorkdirsV0 keeps the external target project
// separate from the Orquesta source worktree that proves server identity.
func serverIdentityPreflightForWorkdirsV0(
	ctx context.Context,
	projectWorkDir string,
	worktreeDir string,
	identity orquestaserver.ServerRuntimeIdentityV0,
) serverIdentityPreflightV0 {
	return serverIdentityPreflightV0{
		ProjectWorkDir:  projectWorkDir,
		WorktreeDir:     worktreeDir,
		RuntimeIdentity: identity,
		Issue: validateServerWorktreeIdentityWithRuntimeV0(
			ctx,
			worktreeDir,
			identity,
		),
	}
}

func serverWorktreeDirFromEnvV0(fallback string) string {
	if configured := strings.TrimSpace(os.Getenv(envServerWorktreeV0)); configured != "" {
		return configured
	}
	// Compatibility only: older compositions used the project workdir for both
	// roles. New compositions must set ORQUESTA_SERVER_WORKTREE explicitly.
	return fallback
}

func validateServerWorktreeIdentityV0(ctx context.Context, workdir string) *serverWorktreeIdentityIssueV0 {
	return validateServerWorktreeIdentityWithRuntimeV0(ctx, workdir, serverRuntimeIdentityFromExecutableV0())
}

func validateServerWorktreeIdentityWithRuntimeV0(
	ctx context.Context,
	workdir string,
	identity orquestaserver.ServerRuntimeIdentityV0,
) *serverWorktreeIdentityIssueV0 {
	canonicalWorkdir, issue := canonicalServerWorkdirV0(workdir)
	if issue != nil {
		return issue
	}
	for _, marker := range []string{".orquesta-retired", "RETIRED", ".orquesta-stale", "STALE"} {
		if _, err := os.Stat(filepath.Join(canonicalWorkdir, marker)); err == nil {
			code := serverWorktreeIdentityRetiredV0
			if strings.Contains(strings.ToLower(marker), "stale") {
				code = serverWorktreeIdentityStaleV0
			}
			return &serverWorktreeIdentityIssueV0{Code: code}
		}
	}
	if !serverWorktreeGitOKV0(ctx, canonicalWorkdir, "rev-parse", "--is-inside-work-tree") ||
		!serverWorktreeGitOKV0(ctx, canonicalWorkdir, "rev-parse", "--verify", "HEAD") {
		return &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityInvalidV0}
	}
	root := serverWorktreeGitOutputV0(ctx, canonicalWorkdir, "rev-parse", "--show-toplevel")
	canonicalRoot, rootIssue := canonicalServerWorkdirV0(root)
	if rootIssue != nil || canonicalRoot != canonicalWorkdir {
		return &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityInvalidV0}
	}

	branch := serverWorktreeGitOutputV0(ctx, canonicalWorkdir, "symbolic-ref", "--quiet", "--short", "HEAD")
	upstream := serverWorktreeGitOutputV0(ctx, canonicalWorkdir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	remote, upstreamBranch, ok := strings.Cut(upstream, "/")
	if branch != serverCanonicalGitRefV0 || !ok || remote == "" ||
		upstreamBranch != serverCanonicalGitRefV0 || branch != upstreamBranch {
		return &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityRefV0}
	}
	remoteURL := serverWorktreeGitOutputV0(ctx, canonicalWorkdir, "remote", "get-url", remote)
	if remoteURL != serverCanonicalGitRemoteURLV0 {
		return &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityRemoteV0}
	}
	head := serverWorktreeGitOutputV0(ctx, canonicalWorkdir, "rev-parse", "HEAD")
	upstreamSHA := serverWorktreeGitOutputV0(ctx, canonicalWorkdir, "rev-parse", upstream)
	if head == "" || upstreamSHA == "" {
		return &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityRefV0}
	}
	if head != upstreamSHA {
		if serverWorktreeGitOKV0(ctx, canonicalWorkdir, "merge-base", "--is-ancestor", "HEAD", upstream) {
			return &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityStaleV0}
		}
		return &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityNotAlignedV0}
	}
	return validateServerRuntimeIdentityAgainstHeadV0(identity, head)
}

func canonicalServerWorkdirV0(workdir string) (string, *serverWorktreeIdentityIssueV0) {
	workdir = strings.TrimSpace(workdir)
	if workdir == "" {
		return "", &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityMissingV0}
	}
	abs, err := filepath.Abs(workdir)
	if err != nil {
		return "", &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityInvalidV0}
	}
	abs = filepath.Clean(abs)
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityMissingV0}
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityInvalidV0}
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil || filepath.Clean(resolved) != abs {
		return "", &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentitySymlinkV0}
	}
	return abs, nil
}

func validateServerRuntimeIdentityAgainstHeadV0(
	identity orquestaserver.ServerRuntimeIdentityV0,
	head string,
) *serverWorktreeIdentityIssueV0 {
	identity = orquestaserver.NormalizeServerRuntimeIdentityV0(identity)
	binaryPath := strings.TrimSpace(identity.BinaryPath)
	resolvedBinary, err := filepath.EvalSymlinks(binaryPath)
	if err != nil || binaryPath == "" || resolvedBinary != binaryPath ||
		!serverRuntimeBinarySHA256MatchesV0(binaryPath, identity.BinarySHA256) {
		return &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityBinaryV0}
	}
	if strings.TrimSpace(identity.CommitRef) != strings.TrimSpace(head) {
		return &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityBuildCommitV0}
	}
	expectedBuildRef := serverRuntimeBuildRefV0(identity.CommitRef, identity.BinarySHA256, false)
	if strings.HasSuffix(identity.BuildRef, "-modified") {
		return &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityBuildModifiedV0}
	}
	if identity.BuildRef == "" || identity.BuildRef != expectedBuildRef {
		return &serverWorktreeIdentityIssueV0{Code: serverWorktreeIdentityBinaryV0}
	}
	return nil
}

func serverRuntimeIdentityReadyForExactMatchV0(identity orquestaserver.ServerRuntimeIdentityV0) bool {
	identity = orquestaserver.NormalizeServerRuntimeIdentityV0(identity)
	return identity.SchemaVersion == orquestaserver.ServerRuntimeIdentitySchemaVersionV0 &&
		identity.BinaryPath != "" &&
		identity.BinaryPathRef == orquestaserver.ServerRuntimeBinaryPathRefV0 &&
		identity.BinaryName != "" &&
		validServerRuntimeBinarySHA256V0(identity.BinarySHA256) &&
		identity.BuildRef != "" &&
		identity.CommitRef != ""
}

func validServerRuntimeBinarySHA256V0(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func serverRuntimeBinarySHA256MatchesV0(path string, expected string) bool {
	expected = strings.TrimSpace(expected)
	if !validServerRuntimeBinarySHA256V0(expected) {
		return false
	}
	return strings.EqualFold(serverRuntimeBinarySHA256V0(path), expected)
}

func serverRuntimeIdentityWithWorktreeEvidenceV0(identity orquestaserver.ServerRuntimeIdentityV0, _ string) orquestaserver.ServerRuntimeIdentityV0 {
	identity.EvidenceRefs = append(identity.EvidenceRefs,
		serverIdentityEvidenceValidV0,
		"evidence-ref-server-worktree-root-canonical",
		"evidence-ref-server-worktree-head-build-exact",
		"evidence-ref-server-worktree-upstream-exact",
		"evidence-ref-server-worktree-remote-canonical",
	)
	return orquestaserver.NormalizeServerRuntimeIdentityV0(identity)
}

func serverWorktreeGitOKV0(ctx context.Context, repo string, args ...string) bool {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	return cmd.Run() == nil
}

func serverWorktreeGitOutputV0(ctx context.Context, repo string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
