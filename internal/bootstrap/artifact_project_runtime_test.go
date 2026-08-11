package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/adapters/artifact/filesystem"
	"orquesta/internal/config"
	"orquesta/internal/ports"
)

func TestBuildBindsFilesystemArtifactsToCanonicalProjectWithoutLegacyMigration(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatalf("make test root private: %v", err)
	}
	artifactRoot := filepath.Join(root, "shared-artifacts")
	content := []byte("same digest, different project metadata")
	if err := os.MkdirAll(artifactRoot, 0o700); err != nil {
		t.Fatalf("create shared artifact root: %v", err)
	}
	if err := os.Chmod(artifactRoot, 0o700); err != nil {
		t.Fatalf("make shared artifact root private: %v", err)
	}
	legacy, err := filesystem.Open(artifactRoot)
	if err != nil {
		t.Fatalf("open legacy artifact layout: %v", err)
	}
	legacyStored, err := legacy.Put(context.Background(), ports.PutArtifactRequest{
		MediaType: "text/plain", Content: content,
	})
	if err != nil {
		_ = legacy.Close()
		t.Fatalf("seed legacy artifact layout: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("close legacy artifact layout: %v", err)
	}

	alphaConfig := writeProjectArtifactRuntimeConfig(t, filepath.Join(root, "alpha"), artifactRoot, "project:artifact-alpha")
	alpha := buildProjectArtifactRuntime(t, alphaConfig)
	if _, err := alpha.artifacts.Get(context.Background(), legacyStored.Ref, legacyStored.Size); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorNotFound {
		t.Fatalf("project-aware runtime read legacy artifact: error=%v", err)
	}
	alphaStored, err := alpha.artifacts.Put(context.Background(), ports.PutArtifactRequest{
		MediaType: "text/plain", Content: content,
	})
	if err != nil {
		_ = alpha.Shutdown(context.Background())
		t.Fatalf("write alpha artifact: %v", err)
	}
	shutdownProjectArtifactRuntime(t, alpha)

	betaConfig := writeProjectArtifactRuntimeConfig(t, filepath.Join(root, "beta"), artifactRoot, "project:artifact-beta")
	beta := buildProjectArtifactRuntime(t, betaConfig)
	betaStored, err := beta.artifacts.Put(context.Background(), ports.PutArtifactRequest{
		MediaType: "application/json", Content: content,
	})
	if err != nil {
		_ = beta.Shutdown(context.Background())
		t.Fatalf("write beta artifact: %v", err)
	}
	if alphaStored.Ref != betaStored.Ref {
		t.Fatalf("content-addressed refs differ across projects: alpha=%s beta=%s", alphaStored.Ref, betaStored.Ref)
	}
	betaContent, err := beta.artifacts.Get(context.Background(), betaStored.Ref, betaStored.Size)
	if err != nil || betaContent.MediaType != "application/json" {
		_ = beta.Shutdown(context.Background())
		t.Fatalf("beta artifact=%+v error=%v", betaContent, err)
	}
	shutdownProjectArtifactRuntime(t, beta)

	alphaRestarted := buildProjectArtifactRuntime(t, alphaConfig)
	alphaContent, err := alphaRestarted.artifacts.Get(context.Background(), alphaStored.Ref, alphaStored.Size)
	if err != nil || alphaContent.MediaType != "text/plain" {
		_ = alphaRestarted.Shutdown(context.Background())
		t.Fatalf("alpha restart artifact=%+v error=%v", alphaContent, err)
	}
	shutdownProjectArtifactRuntime(t, alphaRestarted)

	legacy, err = filesystem.Open(artifactRoot)
	if err != nil {
		t.Fatalf("reopen legacy artifact layout: %v", err)
	}
	defer func() { _ = legacy.Close() }()
	legacyContent, err := legacy.Get(context.Background(), legacyStored.Ref, legacyStored.Size)
	if err != nil || string(legacyContent.Content) != string(content) {
		t.Fatalf("legacy artifact changed during project composition: content=%+v error=%v", legacyContent, err)
	}
}

func TestOpenBuildArtifactsUsesConfiguredProjectWithoutLocalIdentity(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	snapshot, err := config.Resolve(config.ResolveOptions{TOML: []byte(
		"[artifact.filesystem]\nroot = " + strconv.Quote(filepath.Join(root, "artifacts")) + "\n\n[project]\ndefault = \"project:oidc-configured\"\n",
	)})
	if err != nil {
		t.Fatalf("resolve configured project: %v", err)
	}
	store, err := openBuildArtifacts(snapshot)
	if err != nil {
		t.Fatalf("open artifacts without local identity composition: %v", err)
	}
	defer func() { _ = store.Close() }()
	stored, err := store.Put(context.Background(), ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("configured project")})
	if err != nil {
		t.Fatalf("write configured project artifact: %v", err)
	}
	if _, err := store.Get(context.Background(), stored.Ref, stored.Size); err != nil {
		t.Fatalf("read configured project artifact: %v", err)
	}
}

func writeProjectArtifactRuntimeConfig(t *testing.T, root, artifactRoot, projectRef string) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "root = "+strconv.Quote(filepath.Join(root, "artifacts")), "root = "+strconv.Quote(artifactRoot))
	replaceTestConfigValue(t, configPath, "[server]\n", "[project]\ndefault = "+strconv.Quote(projectRef)+"\n\n[server]\n")
	return configPath
}

func buildProjectArtifactRuntime(t *testing.T, configPath string) *Runtime {
	t.Helper()
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "artifact-project-runtime",
		AgentFactory: countingFactory(&atomic.Int64{}),
	})
	if err != nil {
		t.Fatalf("build project artifact runtime: %v", err)
	}
	return runtime
}

func shutdownProjectArtifactRuntime(t *testing.T, runtime *Runtime) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := runtime.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown project artifact runtime: %v", err)
	}
}
