package acceptance_test

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"
)

const evidenceCandidateDigestAlgorithmV1 = "sha256:length-framed-path-and-content:v1"

type evidenceReceiptV1 struct {
	SchemaVersion            int      `json:"schema_version"`
	Contract                 string   `json:"contract"`
	Result                   string   `json:"result"`
	Command                  string   `json:"command"`
	ExecutedAt               string   `json:"executed_at"`
	FixtureSHA256            string   `json:"fixture_sha256"`
	CandidateDigestAlgorithm string   `json:"candidate_digest_algorithm"`
	CandidateSubjects        []string `json:"candidate_subjects"`
	CandidateSHA256          string   `json:"candidate_sha256"`
}

type evidenceReceiptExpectation struct {
	Contract          string
	Command           string
	FixturePath       string
	ReceiptPath       string
	CandidateSubjects []string
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
	handle, err := os.Open(filename)
	if err != nil {
		t.Fatalf("open strict JSON %s: %v", filename, err)
	}
	defer handle.Close()
	decoder := json.NewDecoder(handle)
	decoder.DisallowUnknownFields()
	var value T
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode strict JSON %s: %v", filename, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("decode trailing JSON %s: %v", filename, err)
	}
	return value
}

func evidenceAssertReceiptV1(t *testing.T, repositoryRoot string, expected evidenceReceiptExpectation) {
	t.Helper()
	if expected.Contract == "" || expected.Command == "" || expected.FixturePath == "" || expected.ReceiptPath == "" || len(expected.CandidateSubjects) == 0 {
		t.Fatal("receipt expectation is incomplete")
	}
	receipt := evidenceDecodeStrictJSON[evidenceReceiptV1](t, filepath.Join(repositoryRoot, filepath.FromSlash(expected.ReceiptPath)))
	if receipt.SchemaVersion != 1 || receipt.Contract != expected.Contract || receipt.Result != "PASS" || receipt.Command != expected.Command {
		t.Fatalf("invalid receipt identity/result: %+v", receipt)
	}
	if _, err := time.Parse(time.RFC3339, receipt.ExecutedAt); err != nil {
		t.Fatalf("invalid receipt executed_at %q: %v", receipt.ExecutedAt, err)
	}
	if receipt.CandidateDigestAlgorithm != evidenceCandidateDigestAlgorithmV1 {
		t.Fatalf("candidate digest algorithm = %q, want %q", receipt.CandidateDigestAlgorithm, evidenceCandidateDigestAlgorithmV1)
	}
	if !slices.Equal(receipt.CandidateSubjects, expected.CandidateSubjects) {
		t.Fatalf("receipt candidate subjects = %v, want %v", receipt.CandidateSubjects, expected.CandidateSubjects)
	}
	for _, subject := range expected.CandidateSubjects {
		if subject == expected.ReceiptPath {
			t.Fatalf("receipt %q is a self-referential candidate subject", expected.ReceiptPath)
		}
	}
	fixtureSHA := evidenceFileSHA256(t, filepath.Join(repositoryRoot, filepath.FromSlash(expected.FixturePath)))
	if receipt.FixtureSHA256 != fixtureSHA {
		t.Fatalf("receipt fixture_sha256 = %q, want %q", receipt.FixtureSHA256, fixtureSHA)
	}
	candidateSHA := evidenceCandidateDigest(t, repositoryRoot, expected.CandidateSubjects)
	if receipt.CandidateSHA256 != candidateSHA {
		t.Fatalf("receipt candidate_sha256 = %q, want %q", receipt.CandidateSHA256, candidateSHA)
	}
}

func evidenceFileSHA256(t *testing.T, filename string) string {
	t.Helper()
	content := evidenceReadRegularFile(t, filename)
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func evidenceCandidateDigest(t *testing.T, repositoryRoot string, subjects []string) string {
	t.Helper()
	if !sort.StringsAreSorted(subjects) {
		t.Fatalf("candidate subjects must be sorted: %v", subjects)
	}
	digest := sha256.New()
	seen := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		if subject == "" || filepath.ToSlash(filepath.Clean(subject)) != subject || filepath.IsAbs(subject) || strings.HasPrefix(subject, "../") || strings.Contains(subject, `\`) {
			t.Fatalf("invalid repository-relative subject %q", subject)
		}
		if _, duplicate := seen[subject]; duplicate {
			t.Fatalf("duplicate candidate subject %q", subject)
		}
		seen[subject] = struct{}{}
		content := evidenceReadRegularFile(t, filepath.Join(repositoryRoot, filepath.FromSlash(subject)))
		evidenceWriteFrame(t, digest, []byte(subject))
		evidenceWriteFrame(t, digest, content)
	}
	return "sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func evidenceReadRegularFile(t *testing.T, filename string) []byte {
	t.Helper()
	info, err := os.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("evidence subject %s must be a regular file: %v", filename, err)
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read evidence subject %s: %v", filename, err)
	}
	return content
}

func evidenceWriteFrame(t *testing.T, writer io.Writer, value []byte) {
	t.Helper()
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	if _, err := writer.Write(size[:]); err != nil {
		t.Fatalf("write evidence frame size: %v", err)
	}
	if _, err := writer.Write(value); err != nil {
		t.Fatalf("write evidence frame value: %v", err)
	}
}
