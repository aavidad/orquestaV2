package acceptance_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	evidenceCandidateDigestAlgorithmV3 = "sha256:length-framed-git-blob-set:v1"
	evidenceSealedSourceIdentityV3     = "sealed_git_subject_set"
)

type evidenceReceiptV3 struct {
	SchemaVersion            int                        `json:"schema_version"`
	EvidenceKind             string                     `json:"evidence_kind"`
	Contract                 string                     `json:"contract"`
	Result                   string                     `json:"result"`
	ValidationCommand        string                     `json:"validation_command"`
	ExecutedAt               string                     `json:"executed_at"`
	Execution                evidenceReceiptExecutionV3 `json:"execution"`
	FixtureSHA256            string                     `json:"fixture_sha256"`
	CandidateDigestAlgorithm string                     `json:"candidate_digest_algorithm"`
	CandidateSHA256          string                     `json:"candidate_sha256"`
	SealedSource             evidenceSealedSourceV3     `json:"sealed_source"`
}

type evidenceReceiptExecutionV3 struct {
	Argv                        []string `json:"argv"`
	SourceGitCommitOID          string   `json:"source_git_commit_oid"`
	SourceWorktreeState         string   `json:"source_worktree_state"`
	SourceStatusPorcelainSHA256 string   `json:"source_status_porcelain_sha256"`
	ExitCode                    int      `json:"exit_code"`
	CombinedOutputPath          string   `json:"combined_output_path"`
	CombinedOutputSHA256        string   `json:"combined_output_sha256"`
	GoVersion                   string   `json:"go_version"`
}

type evidenceSealedSourceV3 struct {
	IdentityKind   string `json:"identity_kind"`
	GitCommitOID   string `json:"git_commit_oid"`
	GitTreeOID     string `json:"git_tree_oid"`
	FixtureBlobOID string `json:"fixture_blob_oid"`
	SHA256         string `json:"sha256"`
}

type evidenceReceiptV3Expectation struct {
	Contract                string
	FixturePath             string
	ReceiptPath             string
	ExecutedNotBefore       string
	TrustedBaseGitCommitOID string
}

type evidenceFixtureEnvelopeV3 struct {
	SchemaVersion                int      `json:"schema_version"`
	ReceiptSchemaVersion         int      `json:"receipt_schema_version"`
	ContractID                   string   `json:"contract_id"`
	TrustedBaseGitCommitOID      string   `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID string   `json:"product_delta_base_git_commit_oid"`
	Command                      string   `json:"command"`
	ExecutionArgv                []string `json:"execution_argv"`
	OutputPath                   string   `json:"output_path"`
	ReceiptPath                  string   `json:"receipt_path"`
	CandidateSubjects            []string `json:"candidate_subjects"`
}

type evidenceGitBlobEntry struct {
	Mode    string
	OID     string
	Content []byte
}

func evidenceRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve acceptance support path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolve repository root %s: %v", root, err)
	}
	return root
}

func evidenceDecodeStrictJSON[T any](t *testing.T, filename string) T {
	t.Helper()
	value, err := evidenceDecodeStrictJSONFile[T](filename)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func evidenceDecodeStrictJSONFile[T any](filename string) (T, error) {
	var value T
	info, err := os.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() {
		return value, fmt.Errorf("strict JSON %s must be a regular file: %v", filename, err)
	}
	handle, err := os.Open(filename)
	if err != nil {
		return value, fmt.Errorf("open strict JSON %s: %w", filename, err)
	}
	defer handle.Close()
	content, err := io.ReadAll(handle)
	if err != nil {
		return value, fmt.Errorf("read strict JSON %s: %w", filename, err)
	}
	return evidenceDecodeStrictJSONBytes[T](content)
}

func evidenceDecodeStrictJSONBytes[T any](content []byte) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return value, fmt.Errorf("trailing JSON: %v", err)
	}
	return value, nil
}

func evidenceAssertReceiptV3(t *testing.T, repositoryRoot string, expected evidenceReceiptV3Expectation) {
	t.Helper()
	if err := evidenceValidateReceiptV3(repositoryRoot, expected); err != nil {
		t.Fatal(err)
	}
}

func evidenceValidateReceiptV3(repositoryRoot string, expected evidenceReceiptV3Expectation) error {
	if expected.Contract == "" || expected.FixturePath == "" || expected.ReceiptPath == "" ||
		expected.ExecutedNotBefore == "" || expected.TrustedBaseGitCommitOID == "" {
		return fmt.Errorf("V3 receipt expectation is incomplete")
	}
	for _, path := range []string{expected.FixturePath, expected.ReceiptPath} {
		if err := evidenceValidateRepositoryPath(path); err != nil {
			return err
		}
	}
	receipt, err := evidenceDecodeStrictJSONFile[evidenceReceiptV3](filepath.Join(repositoryRoot, filepath.FromSlash(expected.ReceiptPath)))
	if err != nil {
		return err
	}
	if receipt.SchemaVersion != 3 || receipt.EvidenceKind != "reproducibility_descriptor" ||
		receipt.Contract != expected.Contract || receipt.Result != "PASS" {
		return fmt.Errorf("invalid V3 receipt identity/result: %+v", receipt)
	}
	if receipt.Execution.ExitCode != 0 {
		return fmt.Errorf("receipt execution exit_code = %d, want 0", receipt.Execution.ExitCode)
	}
	if receipt.Execution.SourceWorktreeState != "detached_clean" ||
		receipt.Execution.SourceStatusPorcelainSHA256 != evidenceBytesSHA256(nil) {
		return fmt.Errorf("execution source was not attested as a clean detached worktree: state=%q status=%q", receipt.Execution.SourceWorktreeState, receipt.Execution.SourceStatusPorcelainSHA256)
	}
	if receipt.Execution.GoVersion == "" || strings.ContainsAny(receipt.Execution.GoVersion, "\r\n") {
		return fmt.Errorf("invalid receipt Go version %q", receipt.Execution.GoVersion)
	}
	if err := evidenceValidateExecutionTime(receipt.ExecutedAt, expected.ExecutedNotBefore); err != nil {
		return err
	}
	if receipt.CandidateDigestAlgorithm != evidenceCandidateDigestAlgorithmV3 {
		return fmt.Errorf("candidate digest algorithm = %q, want %q", receipt.CandidateDigestAlgorithm, evidenceCandidateDigestAlgorithmV3)
	}
	if receipt.SealedSource.IdentityKind != evidenceSealedSourceIdentityV3 {
		return fmt.Errorf("sealed source identity = %q, want %q", receipt.SealedSource.IdentityKind, evidenceSealedSourceIdentityV3)
	}
	sealedCommit := receipt.SealedSource.GitCommitOID
	if receipt.Execution.SourceGitCommitOID != sealedCommit {
		return fmt.Errorf("execution source commit %q differs from sealed commit %q", receipt.Execution.SourceGitCommitOID, sealedCommit)
	}
	if err := evidenceValidateSealedCommit(repositoryRoot, expected.TrustedBaseGitCommitOID, sealedCommit); err != nil {
		return err
	}
	commitTime, err := evidenceGitCommitTime(repositoryRoot, sealedCommit)
	if err != nil {
		return err
	}
	executedAt, err := time.Parse(time.RFC3339, receipt.ExecutedAt)
	if err != nil {
		return err
	}
	if executedAt.Before(commitTime) {
		return fmt.Errorf("receipt executed_at %q predates sealed commit time %q", receipt.ExecutedAt, commitTime.Format(time.RFC3339))
	}
	treeOID, err := evidenceGitTreeOID(repositoryRoot, sealedCommit)
	if err != nil {
		return err
	}
	if receipt.SealedSource.GitTreeOID != treeOID {
		return fmt.Errorf("sealed git_tree_oid = %q, want %q", receipt.SealedSource.GitTreeOID, treeOID)
	}
	fixtureEntry, err := evidenceGitBlobAt(repositoryRoot, sealedCommit, expected.FixturePath)
	if err != nil {
		return err
	}
	if receipt.SealedSource.FixtureBlobOID != fixtureEntry.OID {
		return fmt.Errorf("sealed fixture_blob_oid = %q, want %q", receipt.SealedSource.FixtureBlobOID, fixtureEntry.OID)
	}
	fixtureSHA := evidenceBytesSHA256(fixtureEntry.Content)
	if receipt.FixtureSHA256 != fixtureSHA {
		return fmt.Errorf("receipt fixture_sha256 = %q, want sealed blob %q", receipt.FixtureSHA256, fixtureSHA)
	}
	fixture, err := evidenceDecodeFixtureEnvelope(fixtureEntry.Content)
	if err != nil {
		return fmt.Errorf("decode sealed fixture %s: %w", expected.FixturePath, err)
	}
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != expected.Contract || fixture.TrustedBaseGitCommitOID != expected.TrustedBaseGitCommitOID ||
		fixture.ReceiptPath != expected.ReceiptPath || fixture.Command == "" || len(fixture.ExecutionArgv) == 0 {
		return fmt.Errorf("sealed fixture identity differs from receipt expectation: %+v", fixture)
	}
	if err := evidenceValidateRepositoryPath(fixture.OutputPath); err != nil {
		return err
	}
	if receipt.ValidationCommand != fixture.Command || !stringSlicesEqual(receipt.Execution.Argv, fixture.ExecutionArgv) {
		return fmt.Errorf("receipt command/argv differ from sealed fixture: command=%q/%q argv=%v/%v", receipt.ValidationCommand, fixture.Command, receipt.Execution.Argv, fixture.ExecutionArgv)
	}
	if receipt.Execution.CombinedOutputPath != fixture.OutputPath {
		return fmt.Errorf("receipt combined output path = %q, want sealed fixture %q", receipt.Execution.CombinedOutputPath, fixture.OutputPath)
	}
	if err := evidenceValidateCandidateSubjects(fixture.CandidateSubjects, expected.ReceiptPath, fixture.OutputPath); err != nil {
		return err
	}
	candidateSHA, err := evidenceGitCandidateDigest(
		repositoryRoot, sealedCommit, fixture.CandidateSubjects, fixture.ProductDeltaBaseGitCommitOID,
	)
	if err != nil {
		return err
	}
	if receipt.CandidateSHA256 != candidateSHA || receipt.SealedSource.SHA256 != candidateSHA {
		return fmt.Errorf("candidate identity mismatch: receipt=%q sealed=%q want=%q", receipt.CandidateSHA256, receipt.SealedSource.SHA256, candidateSHA)
	}
	outputSHA, err := evidenceRegularFileSHA256(filepath.Join(repositoryRoot, filepath.FromSlash(fixture.OutputPath)))
	if err != nil {
		return err
	}
	if receipt.Execution.CombinedOutputSHA256 != outputSHA {
		return fmt.Errorf("receipt combined output sha256 = %q, want %q", receipt.Execution.CombinedOutputSHA256, outputSHA)
	}
	return nil
}

func evidenceValidateExecutionTime(executedAtRaw, notBeforeRaw string) error {
	executedAt, err := time.Parse(time.RFC3339, executedAtRaw)
	if err != nil {
		return fmt.Errorf("invalid receipt executed_at %q: %w", executedAtRaw, err)
	}
	if executedAt.After(time.Now().Add(5 * time.Second)) {
		return fmt.Errorf("receipt executed_at %q is in the future", executedAtRaw)
	}
	notBefore, err := time.Parse(time.RFC3339, notBeforeRaw)
	if err != nil {
		return fmt.Errorf("invalid expected not-before %q: %w", notBeforeRaw, err)
	}
	if executedAt.Before(notBefore) {
		return fmt.Errorf("receipt executed_at %q predates release bound %q", executedAtRaw, notBeforeRaw)
	}
	return nil
}

func evidenceDecodeFixtureEnvelope(content []byte) (evidenceFixtureEnvelopeV3, error) {
	var fixture evidenceFixtureEnvelopeV3
	decoder := json.NewDecoder(bytes.NewReader(content))
	if err := decoder.Decode(&fixture); err != nil {
		return fixture, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fixture, fmt.Errorf("trailing JSON: %v", err)
	}
	return fixture, nil
}

func evidenceValidateCandidateSubjects(subjects []string, receiptPath, outputPath string) error {
	if err := evidenceValidateCandidateSubjectSet(subjects); err != nil {
		return err
	}
	for _, subject := range subjects {
		if subject == receiptPath || subject == outputPath {
			return fmt.Errorf("receipt/output %q/%q must stay outside candidate subjects", receiptPath, outputPath)
		}
	}
	return nil
}

// evidenceValidateCandidateSubjectAtDelta accepts a readable subject or an
// exact deletion in the sealed delta. Deletions are product changes too and
// must stay in the candidate set even though no file exists at HEAD.
func evidenceValidateCandidateSubjectAtDelta(
	repositoryRoot, baseCommit, sealedCommit, subject string,
) error {
	if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(subject))); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("candidate subject %q is not readable: %w", subject, err)
	}
	output, err := evidenceGit(
		repositoryRoot, "diff", "--no-renames", "--name-status",
		baseCommit, sealedCommit, "--", subject,
	)
	if err != nil {
		return fmt.Errorf("resolve absent candidate subject %q: %w", subject, err)
	}
	if strings.TrimSpace(string(output)) != "D\t"+subject {
		return fmt.Errorf("absent candidate subject %q is not an exact sealed deletion", subject)
	}
	return nil
}

func evidenceValidateCandidateSubjectSet(subjects []string) error {
	if len(subjects) == 0 || !sort.StringsAreSorted(subjects) {
		return fmt.Errorf("candidate subjects must be non-empty and sorted: %v", subjects)
	}
	seen := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		if err := evidenceValidateRepositoryPath(subject); err != nil {
			return err
		}
		if _, duplicate := seen[subject]; duplicate {
			return fmt.Errorf("duplicate candidate subject %q", subject)
		}
		seen[subject] = struct{}{}
	}
	return nil
}

func evidenceValidateRepositoryPath(path string) error {
	if path == "" || filepath.IsAbs(path) || filepath.ToSlash(filepath.Clean(path)) != path ||
		path == ".." || strings.HasPrefix(path, "../") || strings.ContainsAny(path, "\\\x00\r\n\t") {
		return fmt.Errorf("invalid repository-relative evidence path %q", path)
	}
	return nil
}

func evidenceValidateSealedCommit(repositoryRoot, trustedBase, sealedCommit string) error {
	base, err := evidenceGitCanonicalCommit(repositoryRoot, trustedBase)
	if err != nil {
		return fmt.Errorf("trusted base: %w", err)
	}
	sealed, err := evidenceGitCanonicalCommit(repositoryRoot, sealedCommit)
	if err != nil {
		return fmt.Errorf("sealed commit: %w", err)
	}
	if err := evidenceGitAncestor(repositoryRoot, base, sealed); err != nil {
		return fmt.Errorf("sealed commit %s does not descend from trusted base %s: %w", sealed, base, err)
	}
	headRaw, err := evidenceGit(repositoryRoot, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return fmt.Errorf("resolve current HEAD: %w", err)
	}
	head := strings.TrimSpace(string(headRaw))
	if err := evidenceValidateGitOID(head); err != nil {
		return fmt.Errorf("current HEAD: %w", err)
	}
	if err := evidenceGitAncestor(repositoryRoot, sealed, head); err != nil {
		return fmt.Errorf("sealed commit %s is not an ancestor of current HEAD %s: %w", sealed, head, err)
	}
	return nil
}

func evidenceGitCanonicalCommit(repositoryRoot, oid string) (string, error) {
	if err := evidenceValidateGitOID(oid); err != nil {
		return "", err
	}
	resolvedRaw, err := evidenceGit(repositoryRoot, "rev-parse", "--verify", oid+"^{commit}")
	if err != nil {
		return "", err
	}
	resolved := strings.TrimSpace(string(resolvedRaw))
	if resolved != oid {
		return "", fmt.Errorf("commit OID %q resolves as %q", oid, resolved)
	}
	return resolved, nil
}

func evidenceValidateGitOID(oid string) error {
	if len(oid) != 40 || strings.ToLower(oid) != oid {
		return fmt.Errorf("Git OID %q must be a full lowercase SHA-1", oid)
	}
	if _, err := hex.DecodeString(oid); err != nil {
		return fmt.Errorf("invalid Git OID %q: %w", oid, err)
	}
	return nil
}

func evidenceGitAncestor(repositoryRoot, ancestor, descendant string) error {
	_, err := evidenceGit(repositoryRoot, "merge-base", "--is-ancestor", ancestor, descendant)
	return err
}

func evidenceGitTreeOID(repositoryRoot, commitOID string) (string, error) {
	raw, err := evidenceGit(repositoryRoot, "rev-parse", "--verify", commitOID+"^{tree}")
	if err != nil {
		return "", fmt.Errorf("resolve tree for commit %s: %w", commitOID, err)
	}
	oid := strings.TrimSpace(string(raw))
	if err := evidenceValidateGitOID(oid); err != nil {
		return "", fmt.Errorf("tree for commit %s: %w", commitOID, err)
	}
	return oid, nil
}

func evidenceGitCommitTime(repositoryRoot, commitOID string) (time.Time, error) {
	raw, err := evidenceGit(repositoryRoot, "show", "-s", "--format=%cI", commitOID)
	if err != nil {
		return time.Time{}, fmt.Errorf("resolve committer time for %s: %w", commitOID, err)
	}
	value := strings.TrimSpace(string(raw))
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse committer time %q for %s: %w", value, commitOID, err)
	}
	return parsed, nil
}

func evidenceGitCandidateDigest(
	repositoryRoot, commitOID string,
	subjects []string,
	deletionBase ...string,
) (string, error) {
	if err := evidenceValidateCandidateSubjectSet(subjects); err != nil {
		return "", err
	}
	if len(deletionBase) > 1 {
		return "", fmt.Errorf("candidate digest accepts at most one deletion base")
	}
	baseCommit := ""
	if len(deletionBase) == 1 {
		baseCommit = deletionBase[0]
	}
	digest := sha256.New()
	for _, subject := range subjects {
		entry, err := evidenceGitBlobAt(repositoryRoot, commitOID, subject)
		if err != nil {
			if baseCommit == "" {
				return "", err
			}
			entry, err = evidenceGitDeletedBlobAtDelta(repositoryRoot, baseCommit, commitOID, subject)
			if err != nil {
				return "", err
			}
			entry.Mode = "deleted:" + entry.Mode
		}
		for _, value := range [][]byte{[]byte(subject), []byte(entry.Mode), entry.Content} {
			if err := evidenceWriteFrameRaw(digest, value); err != nil {
				return "", err
			}
		}
	}
	return "sha256:" + hex.EncodeToString(digest.Sum(nil)), nil
}

func evidenceGitDeletedBlobAtDelta(
	repositoryRoot, baseCommit, sealedCommit, subject string,
) (evidenceGitBlobEntry, error) {
	output, err := evidenceGit(
		repositoryRoot, "diff", "--no-renames", "--name-status",
		baseCommit, sealedCommit, "--", subject,
	)
	if err != nil {
		return evidenceGitBlobEntry{}, fmt.Errorf("resolve deleted candidate %s: %w", subject, err)
	}
	if strings.TrimSpace(string(output)) != "D\t"+subject {
		return evidenceGitBlobEntry{}, fmt.Errorf("candidate %s is absent but not an exact sealed deletion", subject)
	}
	entry, err := evidenceGitBlobAt(repositoryRoot, baseCommit, subject)
	if err != nil {
		return evidenceGitBlobEntry{}, fmt.Errorf("read deleted candidate %s at base: %w", subject, err)
	}
	return entry, nil
}

func evidenceGitBlobAt(repositoryRoot, commitOID, subject string) (evidenceGitBlobEntry, error) {
	if err := evidenceValidateGitOID(commitOID); err != nil {
		return evidenceGitBlobEntry{}, err
	}
	if err := evidenceValidateRepositoryPath(subject); err != nil {
		return evidenceGitBlobEntry{}, err
	}
	raw, err := evidenceGit(repositoryRoot, "ls-tree", "-z", commitOID, "--", subject)
	if err != nil {
		return evidenceGitBlobEntry{}, fmt.Errorf("inspect sealed subject %s: %w", subject, err)
	}
	if len(raw) == 0 || raw[len(raw)-1] != 0 || bytes.Count(raw, []byte{0}) != 1 {
		return evidenceGitBlobEntry{}, fmt.Errorf("sealed subject %s does not resolve to one tree entry", subject)
	}
	metadata, resolvedPath, ok := bytes.Cut(raw[:len(raw)-1], []byte{'\t'})
	if !ok || string(resolvedPath) != subject {
		return evidenceGitBlobEntry{}, fmt.Errorf("sealed subject %s resolved as %q", subject, resolvedPath)
	}
	fields := strings.Fields(string(metadata))
	if len(fields) != 3 || fields[1] != "blob" || (fields[0] != "100644" && fields[0] != "100755") {
		return evidenceGitBlobEntry{}, fmt.Errorf("sealed subject %s must be a regular blob, got %q", subject, metadata)
	}
	if err := evidenceValidateGitOID(fields[2]); err != nil {
		return evidenceGitBlobEntry{}, fmt.Errorf("sealed subject %s blob: %w", subject, err)
	}
	content, err := evidenceGit(repositoryRoot, "cat-file", "blob", fields[2])
	if err != nil {
		return evidenceGitBlobEntry{}, fmt.Errorf("read sealed subject %s blob: %w", subject, err)
	}
	return evidenceGitBlobEntry{Mode: fields[0], OID: fields[2], Content: content}, nil
}

func evidenceGit(repositoryRoot string, args ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", repositoryRoot}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func evidenceRegularFileSHA256(filename string) (string, error) {
	info, err := os.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("evidence subject %s must be a regular file: %v", filename, err)
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("read evidence subject %s: %w", filename, err)
	}
	return evidenceBytesSHA256(content), nil
}

func evidenceBytesSHA256(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func evidenceWriteFrameRaw(writer io.Writer, value []byte) error {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	if _, err := writer.Write(size[:]); err != nil {
		return fmt.Errorf("write evidence frame size: %w", err)
	}
	if _, err := writer.Write(value); err != nil {
		return fmt.Errorf("write evidence frame value: %w", err)
	}
	return nil
}

func evidenceWriteFrame(t *testing.T, writer io.Writer, value []byte) {
	t.Helper()
	if err := evidenceWriteFrameRaw(writer, value); err != nil {
		t.Fatal(err)
	}
}

func evidenceFileSHA256(t *testing.T, filename string) string {
	t.Helper()
	digest, err := evidenceRegularFileSHA256(filename)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
