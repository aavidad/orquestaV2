package acceptance_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSealedGitReceiptIgnoresLiveCandidateDriftAndRejectsTampering(t *testing.T) {
	repositoryRoot := t.TempDir()
	evidenceTestGit(t, repositoryRoot, "init", "-q")
	evidenceTestGit(t, repositoryRoot, "config", "user.name", "Orquesta Evidence Test")
	evidenceTestGit(t, repositoryRoot, "config", "user.email", "evidence-test@orquesta.invalid")

	fixturePath := "fixture.json"
	receiptPath := "receipt.json"
	outputPath := "result.txt"
	candidatePath := "subject.txt"
	fixture := evidenceFixtureEnvelopeV3{
		SchemaVersion: 1, ContractID: "AC-SEALED-TEST",
		Command: "test-command", ExecutionArgv: []string{"test-command"}, OutputPath: outputPath,
		ReceiptPath: receiptPath, CandidateSubjects: []string{fixturePath, candidatePath},
	}
	evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, fixturePath), fixture)
	evidenceTestWriteFile(t, filepath.Join(repositoryRoot, candidatePath), []byte("sealed\n"), 0o644)
	evidenceTestGit(t, repositoryRoot, "add", fixturePath, candidatePath)
	evidenceTestGit(t, repositoryRoot, "commit", "-q", "-m", "seal candidate")
	sealedCommit := strings.TrimSpace(evidenceTestGit(t, repositoryRoot, "rev-parse", "HEAD"))
	trustedBase := sealedCommit
	legacyFixtureCommit := sealedCommit
	fixture.ReceiptSchemaVersion = 3
	fixture.TrustedBaseGitCommitOID = trustedBase
	// Trust anchor is intentionally added in a predecessor-independent second
	// commit; production fixtures use an already-known historical base.
	evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, fixturePath), fixture)
	evidenceTestGit(t, repositoryRoot, "add", fixturePath)
	evidenceTestGit(t, repositoryRoot, "commit", "-q", "-m", "bind trusted base")
	sealedCommit = strings.TrimSpace(evidenceTestGit(t, repositoryRoot, "rev-parse", "HEAD"))
	treeOID, err := evidenceGitTreeOID(repositoryRoot, sealedCommit)
	if err != nil {
		t.Fatal(err)
	}
	fixtureEntry, err := evidenceGitBlobAt(repositoryRoot, sealedCommit, fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	candidateSHA, err := evidenceGitCandidateDigest(repositoryRoot, sealedCommit, fixture.CandidateSubjects)
	if err != nil {
		t.Fatal(err)
	}
	evidenceTestWriteFile(t, filepath.Join(repositoryRoot, outputPath), []byte("PASS\n"), 0o644)
	outputSHA, err := evidenceRegularFileSHA256(filepath.Join(repositoryRoot, outputPath))
	if err != nil {
		t.Fatal(err)
	}
	receipt := evidenceReceiptV3{
		SchemaVersion: 3, EvidenceKind: "reproducibility_descriptor", Contract: fixture.ContractID,
		Result: "PASS", ValidationCommand: fixture.Command, ExecutedAt: time.Now().UTC().Format(time.RFC3339),
		Execution: evidenceReceiptExecutionV3{
			Argv: []string{"test-command"}, SourceGitCommitOID: sealedCommit,
			SourceWorktreeState: "detached_clean", SourceStatusPorcelainSHA256: evidenceBytesSHA256(nil), ExitCode: 0,
			CombinedOutputPath: outputPath, CombinedOutputSHA256: outputSHA, GoVersion: "go-test",
		},
		FixtureSHA256:            evidenceBytesSHA256(fixtureEntry.Content),
		CandidateDigestAlgorithm: evidenceCandidateDigestAlgorithmV3, CandidateSHA256: candidateSHA,
		SealedSource: evidenceSealedSourceV3{
			IdentityKind: evidenceSealedSourceIdentityV3, GitCommitOID: sealedCommit,
			GitTreeOID: treeOID, FixtureBlobOID: fixtureEntry.OID, SHA256: candidateSHA,
		},
	}
	evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, receiptPath), receipt)
	expected := evidenceReceiptV3Expectation{
		Contract: fixture.ContractID, FixturePath: fixturePath, ReceiptPath: receiptPath,
		ExecutedNotBefore: "2026-01-01T00:00:00Z", TrustedBaseGitCommitOID: trustedBase,
	}
	if err := evidenceValidateReceiptV3(repositoryRoot, expected); err != nil {
		t.Fatalf("fresh sealed receipt rejected: %v", err)
	}

	t.Run("fixture without V3 marker cannot be wrapped", func(t *testing.T) {
		legacyTree, err := evidenceGitTreeOID(repositoryRoot, legacyFixtureCommit)
		if err != nil {
			t.Fatal(err)
		}
		legacyFixture, err := evidenceGitBlobAt(repositoryRoot, legacyFixtureCommit, fixturePath)
		if err != nil {
			t.Fatal(err)
		}
		legacyCandidate, err := evidenceGitCandidateDigest(repositoryRoot, legacyFixtureCommit, []string{fixturePath, candidatePath})
		if err != nil {
			t.Fatal(err)
		}
		legacyReceipt := receipt
		legacyReceipt.Execution.SourceGitCommitOID = legacyFixtureCommit
		legacyReceipt.FixtureSHA256 = evidenceBytesSHA256(legacyFixture.Content)
		legacyReceipt.CandidateSHA256 = legacyCandidate
		legacyReceipt.SealedSource = evidenceSealedSourceV3{
			IdentityKind: evidenceSealedSourceIdentityV3, GitCommitOID: legacyFixtureCommit,
			GitTreeOID: legacyTree, FixtureBlobOID: legacyFixture.OID, SHA256: legacyCandidate,
		}
		evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, receiptPath), legacyReceipt)
		legacyExpected := expected
		legacyExpected.TrustedBaseGitCommitOID = legacyFixtureCommit
		err = evidenceValidateReceiptV3(repositoryRoot, legacyExpected)
		if err == nil || !strings.Contains(err.Error(), "sealed fixture identity") {
			t.Fatalf("legacy fixture without V3 marker failed open or for wrong reason: %v", err)
		}
		evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, receiptPath), receipt)
	})

	// Neither changed bytes nor a changed candidate list in the live worktree may
	// rewrite the historical identity sealed by the commit.
	evidenceTestWriteFile(t, filepath.Join(repositoryRoot, candidatePath), []byte("live drift\n"), 0o644)
	fixture.CandidateSubjects = []string{fixturePath, "later.txt", candidatePath}
	evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, fixturePath), fixture)
	evidenceTestWriteFile(t, filepath.Join(repositoryRoot, "later.txt"), []byte("live only\n"), 0o644)
	if err := evidenceValidateReceiptV3(repositoryRoot, expected); err != nil {
		t.Fatalf("live candidate drift invalidated sealed receipt: %v", err)
	}

	t.Run("digest tampering fails", func(t *testing.T) {
		tampered := receipt
		tampered.CandidateSHA256 = "sha256:" + strings.Repeat("0", 64)
		evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, receiptPath), tampered)
		if err := evidenceValidateReceiptV3(repositoryRoot, expected); err == nil {
			t.Fatal("tampered candidate digest passed")
		}
		evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, receiptPath), receipt)
	})

	t.Run("sealed metadata and execution tampering fail", func(t *testing.T) {
		mutations := []struct {
			name   string
			mutate func(*evidenceReceiptV3)
		}{
			{"tree", func(value *evidenceReceiptV3) { value.SealedSource.GitTreeOID = strings.Repeat("0", 40) }},
			{"fixture blob", func(value *evidenceReceiptV3) { value.SealedSource.FixtureBlobOID = strings.Repeat("0", 40) }},
			{"fixture sha", func(value *evidenceReceiptV3) { value.FixtureSHA256 = "sha256:" + strings.Repeat("0", 64) }},
			{"sealed sha", func(value *evidenceReceiptV3) { value.SealedSource.SHA256 = "sha256:" + strings.Repeat("0", 64) }},
			{"execution commit", func(value *evidenceReceiptV3) { value.Execution.SourceGitCommitOID = trustedBase }},
			{"worktree state", func(value *evidenceReceiptV3) { value.Execution.SourceWorktreeState = "dirty" }},
			{"worktree status", func(value *evidenceReceiptV3) {
				value.Execution.SourceStatusPorcelainSHA256 = "sha256:" + strings.Repeat("0", 64)
			}},
			{"output sha", func(value *evidenceReceiptV3) {
				value.Execution.CombinedOutputSHA256 = "sha256:" + strings.Repeat("0", 64)
			}},
		}
		for _, mutation := range mutations {
			t.Run(mutation.name, func(t *testing.T) {
				tampered := receipt
				mutation.mutate(&tampered)
				evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, receiptPath), tampered)
				if err := evidenceValidateReceiptV3(repositoryRoot, expected); err == nil {
					t.Fatal("tampered receipt passed")
				}
				evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, receiptPath), receipt)
			})
		}
	})

	t.Run("execution after release bound but before commit fails commit-time check", func(t *testing.T) {
		tampered := receipt
		tampered.ExecutedAt = "2026-06-01T00:00:00Z"
		evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, receiptPath), tampered)
		err := evidenceValidateReceiptV3(repositoryRoot, expected)
		if err == nil || !strings.Contains(err.Error(), "predates sealed commit time") {
			t.Fatalf("commit-time guard failed open or for wrong reason: %v", err)
		}
		evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, receiptPath), receipt)
	})

	t.Run("candidate shape rejects exclusions duplicates order and traversal", func(t *testing.T) {
		invalid := [][]string{
			{receiptPath},
			{outputPath},
			{candidatePath, candidatePath},
			{candidatePath, fixturePath},
			{"../escape.txt"},
		}
		for _, subjects := range invalid {
			if err := evidenceValidateCandidateSubjects(subjects, receiptPath, outputPath); err == nil {
				t.Fatalf("invalid candidate subjects passed: %v", subjects)
			}
		}
		if _, err := evidenceGitBlobAt(repositoryRoot, sealedCommit, "."); err == nil {
			t.Fatal("tree candidate passed as regular blob")
		}
	})

	t.Run("receipt and output symlinks fail", func(t *testing.T) {
		for _, relative := range []string{receiptPath, outputPath} {
			original := filepath.Join(repositoryRoot, relative)
			regular := original + ".regular"
			if err := os.Rename(original, regular); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Base(regular), original); err != nil {
				t.Fatal(err)
			}
			if err := evidenceValidateReceiptV3(repositoryRoot, expected); err == nil {
				t.Fatalf("symlink %s passed", relative)
			}
			if err := os.Remove(original); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(regular, original); err != nil {
				t.Fatal(err)
			}
		}
	})

	t.Run("noncanonical or missing commit fails", func(t *testing.T) {
		for _, oid := range []string{sealedCommit[:12], strings.Repeat("f", 40)} {
			tampered := receipt
			tampered.SealedSource.GitCommitOID = oid
			tampered.Execution.SourceGitCommitOID = oid
			evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, receiptPath), tampered)
			if err := evidenceValidateReceiptV3(repositoryRoot, expected); err == nil {
				t.Fatalf("invalid sealed commit %q passed", oid)
			}
		}
		evidenceTestWriteJSON(t, filepath.Join(repositoryRoot, receiptPath), receipt)
	})

	t.Run("sibling commit fails ancestry", func(t *testing.T) {
		sibling := evidenceTestCommitTree(t, repositoryRoot, treeOID, trustedBase)
		if err := evidenceValidateSealedCommit(repositoryRoot, trustedBase, sibling); err == nil {
			t.Fatal("sibling sealed commit passed as ancestor of HEAD")
		}
	})

	t.Run("worktree-only path and symlink fail", func(t *testing.T) {
		if _, err := evidenceGitCandidateDigest(repositoryRoot, sealedCommit, []string{candidatePath, "later.txt"}); err == nil {
			t.Fatal("path absent from sealed commit passed")
		}
		if err := os.Symlink(candidatePath, filepath.Join(repositoryRoot, "link.txt")); err != nil {
			t.Fatal(err)
		}
		evidenceTestGit(t, repositoryRoot, "add", candidatePath, fixturePath, "later.txt", "link.txt")
		evidenceTestGit(t, repositoryRoot, "commit", "-q", "-m", "later live state")
		laterCommit := strings.TrimSpace(evidenceTestGit(t, repositoryRoot, "rev-parse", "HEAD"))
		if _, err := evidenceGitCandidateDigest(repositoryRoot, laterCommit, []string{"link.txt"}); err == nil {
			t.Fatal("symlink candidate passed")
		}
		if err := os.Chmod(filepath.Join(repositoryRoot, candidatePath), 0o755); err != nil {
			t.Fatal(err)
		}
		evidenceTestGit(t, repositoryRoot, "add", candidatePath)
		evidenceTestGit(t, repositoryRoot, "commit", "-q", "-m", "change candidate mode")
		executableCommit := strings.TrimSpace(evidenceTestGit(t, repositoryRoot, "rev-parse", "HEAD"))
		beforeMode, err := evidenceGitCandidateDigest(repositoryRoot, laterCommit, []string{candidatePath})
		if err != nil {
			t.Fatal(err)
		}
		afterMode, err := evidenceGitCandidateDigest(repositoryRoot, executableCommit, []string{candidatePath})
		if err != nil || beforeMode == afterMode {
			t.Fatalf("candidate mode change not bound: before=%s after=%s err=%v", beforeMode, afterMode, err)
		}
		tooNewBase := expected
		tooNewBase.TrustedBaseGitCommitOID = executableCommit
		if err := evidenceValidateReceiptV3(repositoryRoot, tooNewBase); err == nil {
			t.Fatal("sealed commit older than trusted base passed")
		}
	})

}

func TestSealedGitCandidateDigestBindsExactDeletions(t *testing.T) {
	repositoryRoot := t.TempDir()
	evidenceTestGit(t, repositoryRoot, "init", "-q")
	evidenceTestGit(t, repositoryRoot, "config", "user.name", "Orquesta Evidence Test")
	evidenceTestGit(t, repositoryRoot, "config", "user.email", "evidence-test@orquesta.invalid")
	deletedPath := "deleted.txt"
	evidenceTestWriteFile(t, filepath.Join(repositoryRoot, deletedPath), []byte("remove cleanly\n"), 0o644)
	evidenceTestGit(t, repositoryRoot, "add", deletedPath)
	evidenceTestGit(t, repositoryRoot, "commit", "-q", "-m", "add deletion subject")
	deletionBase := strings.TrimSpace(evidenceTestGit(t, repositoryRoot, "rev-parse", "HEAD"))
	if err := os.Remove(filepath.Join(repositoryRoot, deletedPath)); err != nil {
		t.Fatal(err)
	}
	evidenceTestGit(t, repositoryRoot, "add", "-u", "--", deletedPath)
	evidenceTestGit(t, repositoryRoot, "commit", "-q", "-m", "delete subject")
	deletionCommit := strings.TrimSpace(evidenceTestGit(t, repositoryRoot, "rev-parse", "HEAD"))
	if _, err := evidenceGitCandidateDigest(
		repositoryRoot, deletionCommit, []string{deletedPath}, deletionBase,
	); err != nil {
		t.Fatalf("exact deletion rejected: %v", err)
	}
	if _, err := evidenceGitCandidateDigest(repositoryRoot, deletionCommit, []string{deletedPath}); err == nil {
		t.Fatal("deletion without its base passed")
	}
	if _, err := evidenceGitCandidateDigest(
		repositoryRoot, deletionCommit, []string{"never-existed.txt"}, deletionBase,
	); err == nil {
		t.Fatal("absent non-deletion passed")
	}
}

func evidenceTestGit(t *testing.T, repositoryRoot string, args ...string) string {
	t.Helper()
	output, err := evidenceGit(repositoryRoot, args...)
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}

func evidenceTestCommitTree(t *testing.T, repositoryRoot, treeOID, parentOID string) string {
	t.Helper()
	command := exec.Command("git", "-C", repositoryRoot, "commit-tree", treeOID, "-p", parentOID)
	command.Stdin = strings.NewReader("sibling evidence commit\n")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git commit-tree: %v: %s", err, output)
	}
	oid := strings.TrimSpace(string(output))
	if err := evidenceValidateGitOID(oid); err != nil {
		t.Fatal(err)
	}
	return oid
}

func evidenceTestWriteJSON(t *testing.T, filename string, value any) {
	t.Helper()
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	evidenceTestWriteFile(t, filename, content, 0o644)
}

func evidenceTestWriteFile(t *testing.T, filename string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(filename, content, mode); err != nil {
		t.Fatal(err)
	}
}
