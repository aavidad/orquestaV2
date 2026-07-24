package acceptance_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestV19PSourceFixtureEnvelopeAndProductPaths(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v19Fixture](t, v19FixtureRepositoryPath(repositoryRoot))
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.TrustedBaseGitCommitOID != fixture.ProductDeltaBaseGitCommitOID ||
		fixture.TrustedBaseGitCommitOID != "6f244a7594141c74dc28e095e0e5e32a05102826" ||
		fixture.OutputPath != v19OutputPath || fixture.ReceiptPath != v19ReceiptPath ||
		fixture.Command == "" || len(fixture.ExecutionArgv) != 3 {
		t.Fatalf("V19 fixture is not V3-envelope ready: %+v", fixture)
	}
	if fixture.ExecutionArgv[0] != "sh" || fixture.ExecutionArgv[1] != "-c" ||
		fixture.ExecutionArgv[2] != v19ValidationShellBody() ||
		fixture.Command != "sh -c '"+fixture.ExecutionArgv[2]+"'" {
		t.Fatalf("V19 frozen command/argv drift: %q %#v", fixture.Command, fixture.ExecutionArgv)
	}
	assertV19ProductPresence(t, fixture)
}

func v19FixtureRepositoryPath(repositoryRoot string) string {
	return filepath.Join(repositoryRoot, "acceptance", v19FixturePath)
}

func v19AssertSealRepositoryBindings(t *testing.T, repositoryRoot string, fixture v19Fixture) {
	t.Helper()
	manifest := loadV19SealManifest(t, repositoryRoot, fixture)
	if err := validateV19SealRepositoryBindings(repositoryRoot, manifest, fixture); err != nil {
		t.Fatal(err)
	}
}

func validateV19SealRepositoryBindings(repositoryRoot string, manifest v19SealManifest, fixture v19Fixture) error {
	product := manifest.ProductSource
	if err := evidenceValidateSealedCommit(repositoryRoot, fixture.ProductDeltaBaseGitCommitOID, product.GitCommitOID); err != nil {
		return err
	}
	tree, err := evidenceGitTreeOID(repositoryRoot, product.GitCommitOID)
	if err != nil || tree != product.GitTreeOID {
		return fmt.Errorf("v19 seal tree=%q want=%q: %w", product.GitTreeOID, tree, err)
	}
	changedRaw, err := evidenceGit(repositoryRoot, "diff", "--name-only", fixture.ProductDeltaBaseGitCommitOID, product.GitCommitOID, "--")
	if err != nil {
		return err
	}
	changed := v19NonEmptyLines(changedRaw)
	sort.Strings(changed)
	if !reflect.DeepEqual(changed, fixture.CandidateSubjects) {
		return fmt.Errorf("v19 seal candidate paths differ from product delta")
	}
	candidateSHA, err := evidenceGitCandidateDigest(repositoryRoot, product.GitCommitOID, fixture.CandidateSubjects, fixture.ProductDeltaBaseGitCommitOID)
	if err != nil || candidateSHA != product.CandidateSubjectsSHA {
		return fmt.Errorf("v19 seal candidate digest=%q want=%q: %w", product.CandidateSubjectsSHA, candidateSHA, err)
	}
	if err := validateV19SealV18Binding(repositoryRoot, product.GitCommitOID, manifest); err != nil {
		return err
	}
	production, tests, large, err := v19SealMeasuredBudget(repositoryRoot, fixture, product.GitCommitOID)
	if err != nil {
		return err
	}
	if production != manifest.Budget.ProductionNet || tests != manifest.Budget.TestSupportNet ||
		!reflect.DeepEqual(large, manifest.Budget.FilesOver350) {
		return fmt.Errorf("v19 seal measured budget or large-file inventory differs")
	}
	binarySHA, binarySize, goVersion, err := v19BuildSealedBinary(repositoryRoot, product.GitCommitOID, manifest)
	if err != nil {
		return err
	}
	if binarySHA != manifest.Binary.SHA256 || binarySize != manifest.Binary.SizeBytes || goVersion != manifest.Binary.GoVersion {
		return fmt.Errorf("v19 seal binary identity differs")
	}
	return nil
}

func validateV19SealV18Binding(repositoryRoot, productOID string, manifest v19SealManifest) error {
	dependency := manifest.DependencyV18
	entry, err := evidenceGitBlobAt(repositoryRoot, productOID, dependency.ReceiptPath)
	if err != nil {
		return err
	}
	if evidenceBytesSHA256(entry.Content) != dependency.ReceiptSHA256 {
		return fmt.Errorf("v19 seal V18 receipt digest differs")
	}
	receipt, err := evidenceDecodeStrictJSONBytes[evidenceReceiptV3](entry.Content)
	if err != nil {
		return err
	}
	if receipt.Contract != dependency.Contract || receipt.Result != dependency.Result ||
		receipt.SealedSource.GitCommitOID != dependency.SealedSourceGitCommitOID ||
		receipt.CandidateSHA256 != dependency.CandidateSHA256 {
		return fmt.Errorf("v19 seal V18 receipt content differs")
	}
	return nil
}

func v19SealMeasuredBudget(repositoryRoot string, fixture v19Fixture, productOID string) (v19SealLOC, v19SealLOC, []v19SealLargeFile, error) {
	numstat, err := evidenceGit(repositoryRoot, "diff", "--numstat", fixture.ProductDeltaBaseGitCommitOID, productOID, "--")
	if err != nil {
		return v19SealLOC{}, v19SealLOC{}, nil, err
	}
	var production, tests v19SealLOC
	for _, row := range v19NonEmptyLines(numstat) {
		fields := strings.SplitN(row, "\t", 3)
		if len(fields) != 3 || fields[0] == "-" || fields[1] == "-" {
			continue
		}
		added, addErr := strconv.Atoi(fields[0])
		deleted, deleteErr := strconv.Atoi(fields[1])
		layer := v19SealBudgetLayer(fields[2])
		if addErr != nil || deleteErr != nil || layer == "" {
			continue
		}
		target := &production
		if strings.HasSuffix(fields[2], "_test.go") {
			target = &tests
		}
		v19AddSealLOC(target, layer, added-deleted)
	}
	large := make([]v19SealLargeFile, 0, 53)
	for _, subject := range fixture.CandidateSubjects {
		entry, err := evidenceGitBlobAt(repositoryRoot, productOID, subject)
		if err != nil {
			return v19SealLOC{}, v19SealLOC{}, nil, err
		}
		lines := bytes.Count(entry.Content, []byte{'\n'})
		if len(entry.Content) > 0 && entry.Content[len(entry.Content)-1] != '\n' {
			lines++
		}
		if lines > 350 {
			large = append(large, v19SealLargeFile{Path: subject, LOC: lines, Kind: v19SealFileKind(subject)})
		}
	}
	return production, tests, large, nil
}

func v19SealBudgetLayer(path string) string {
	for prefix, layer := range map[string]string{
		"internal/council/": "domain", "internal/goal/": "domain", "internal/identity/": "domain",
		"internal/application/": "application", "internal/adapters/state/sqlite/": "sqlite", "internal/bootstrap/": "bootstrap",
		"internal/interfaces/mcp/": "transport",
	} {
		if strings.HasPrefix(path, prefix) {
			return layer
		}
	}
	return ""
}

func v19AddSealLOC(value *v19SealLOC, layer string, delta int) {
	switch layer {
	case "domain":
		value.Domain += delta
	case "application":
		value.Application += delta
	case "sqlite":
		value.SQLiteRecovery += delta
	case "bootstrap":
		value.Bootstrap += delta
	case "transport":
		value.TransportAdapters += delta
	}
	value.Total += delta
}

func v19SealFileKind(path string) string {
	if strings.HasPrefix(path, "docs/") || strings.HasPrefix(path, "product/") || strings.HasPrefix(path, "acceptance/fixtures/") {
		return "docs_data"
	}
	if strings.HasSuffix(path, "_test.go") {
		return "test"
	}
	return "production"
}

func v19BuildSealedBinary(repositoryRoot, productOID string, manifest v19SealManifest) (string, int64, string, error) {
	root, err := os.MkdirTemp("", "orquesta-v19-seal-")
	if err != nil {
		return "", 0, "", err
	}
	defer os.RemoveAll(root)
	source, binary := filepath.Join(root, "source"), filepath.Join(root, "orquesta-v19")
	if output, err := exec.Command("git", "-C", repositoryRoot, "worktree", "add", "--detach", "--quiet", source, productOID).CombinedOutput(); err != nil {
		return "", 0, "", fmt.Errorf("add V19 sealed worktree: %w: %s", err, output)
	}
	defer exec.Command("git", "-C", repositoryRoot, "worktree", "remove", "--force", source).Run()
	for _, directory := range []string{"cache", "tmp"} {
		if err := os.Mkdir(filepath.Join(root, directory), 0o700); err != nil {
			return "", 0, "", err
		}
	}
	env := append(os.Environ(), "GOCACHE="+filepath.Join(root, "cache"), "GOTMPDIR="+filepath.Join(root, "tmp"))
	probe := []byte("package main\nimport(\"crypto/sha256\";\"encoding/hex\";\"fmt\";\"orquesta/internal/config\")\nfunc main(){s,e:=config.Resolve(config.ResolveOptions{Environment:map[string]string{}});if e!=nil{panic(e)};b,e:=s.EffectiveJSON();if e!=nil{panic(e)};d:=sha256.Sum256(b);fmt.Printf(\"%d %s %s %s sha256:%s\",s.SchemaVersion(),s.RegistryRevision(),s.RegistryHash(),s.Hash(),hex.EncodeToString(d[:]))}\n")
	if err := os.WriteFile(filepath.Join(source, "v19_config_probe.go"), probe, 0o600); err != nil {
		return "", 0, "", err
	}
	check := exec.Command("sh", "-c", "go test -mod=vendor -count=1 ./internal/config -run '^TestCanonicalRegistryAndEveryGeneratedArtifactStaySynchronized$' >/dev/null && go run -mod=vendor ./v19_config_probe.go")
	check.Dir, check.Env = source, env
	identity, err := check.Output()
	if fields := strings.Fields(string(identity)); err != nil || !v19ConfigIdentityMatches(fields, manifest) {
		return "", 0, "", fmt.Errorf("V19 sealed config at %s differs from historical manifest: %w", productOID, err)
	}
	build := exec.Command("go", "build", "-mod=vendor", "-trimpath", "-buildvcs=false", "-ldflags=-buildid=", "-o", binary, "./cmd/orquesta")
	build.Dir = source
	build.Env = append(env, "CGO_ENABLED=0")
	if output, err := build.CombinedOutput(); err != nil {
		return "", 0, "", fmt.Errorf("build V19 sealed binary: %w: %s", err, output)
	}
	info, err := os.Stat(binary)
	if err != nil {
		return "", 0, "", err
	}
	sha, err := evidenceRegularFileSHA256(binary)
	if err != nil {
		return "", 0, "", err
	}
	version := exec.Command("go", "version")
	version.Dir = source
	versionRaw, err := version.Output()
	return sha, info.Size(), strings.TrimSpace(string(versionRaw)), err
}

func v19ConfigIdentityMatches(fields []string, manifest v19SealManifest) bool {
	sealed := manifest.EffectiveConfig
	return len(fields) == 5 && fields[0] == strconv.Itoa(sealed.SchemaVersion) && fields[1] == sealed.RegistryRevision &&
		fields[2] == sealed.RegistrySHA256 && fields[3] == sealed.SnapshotSHA256 && fields[4] == sealed.EffectiveSHA256
}

func v19NonEmptyLines(content []byte) []string {
	if strings.TrimSpace(string(content)) == "" {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(content)), "\n")
}
