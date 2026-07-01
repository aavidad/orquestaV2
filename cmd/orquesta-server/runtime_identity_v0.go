package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverRuntimeIdentityFromExecutableV0() orquestaserver.ServerRuntimeIdentityV0 {
	executable, err := os.Executable()
	if err != nil {
		return orquestaserver.ServerRuntimeIdentityV0{
			EvidenceRefs: []string{"evidence-ref-server-runtime-identity-executable-unavailable"},
		}
	}
	abs, err := filepath.Abs(executable)
	if err != nil {
		abs = strings.TrimSpace(executable)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	binarySHA := serverRuntimeBinarySHA256V0(abs)
	if binarySHA == "" {
		binarySHA = serverRuntimeBinarySHA256V0("/proc/self/exe")
	}
	commitRef, modified := serverRuntimeBuildInfoCommitV0()
	return orquestaserver.NormalizeServerRuntimeIdentityV0(orquestaserver.ServerRuntimeIdentityV0{
		BinaryPath:    abs,
		BinaryPathRef: orquestaserver.ServerRuntimeBinaryPathRefV0,
		BinaryName:    filepath.Base(abs),
		BinarySHA256:  binarySHA,
		BuildRef:      serverRuntimeBuildRefV0(commitRef, binarySHA, modified),
		CommitRef:     commitRef,
		EvidenceRefs:  serverRuntimeIdentityEvidenceRefsV0(binarySHA, commitRef),
	})
}

func serverRuntimeBinarySHA256V0(path string) string {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func serverRuntimeBuildInfoCommitV0() (string, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	commit := ""
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			commit = strings.TrimSpace(setting.Value)
		case "vcs.modified":
			modified = strings.EqualFold(strings.TrimSpace(setting.Value), "true")
		}
	}
	return commit, modified
}

func serverRuntimeBuildRefV0(commitRef string, binarySHA string, modified bool) string {
	source := strings.TrimSpace(commitRef)
	if source == "" {
		source = strings.TrimSpace(binarySHA)
	}
	if source == "" {
		return ""
	}
	if len(source) > 12 {
		source = source[:12]
	}
	ref := "build-ref-orquesta-server-" + strings.ToLower(source)
	if modified {
		ref += "-modified"
	}
	return ref
}

func serverRuntimeIdentityEvidenceRefsV0(binarySHA string, commitRef string) []string {
	refs := []string{"evidence-ref-server-runtime-identity-v0"}
	if strings.TrimSpace(binarySHA) != "" {
		refs = append(refs, "evidence-ref-server-runtime-binary-sha256")
	}
	if strings.TrimSpace(commitRef) != "" {
		refs = append(refs, "evidence-ref-server-runtime-build-vcs")
	}
	return refs
}
