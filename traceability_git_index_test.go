package orquesta_test

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const traceGitIndexMaxBlobBytes = int64(4 << 20)

type traceGitIndexEntry struct {
	Mode string
	OID  string
	Path string
}

type traceGitIndexSnapshot struct {
	Entries  []traceGitIndexEntry
	Contents map[string][]byte
}

func traceLoadGitIndexSnapshot(
	t *testing.T,
	repositoryRoot string,
	sourceRoot string,
	includeContent func(string) bool,
) traceGitIndexSnapshot {
	t.Helper()
	snapshot, err := traceReadGitIndexSnapshot(repositoryRoot, sourceRoot, includeContent)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func traceReadGitIndexSnapshot(
	repositoryRoot string,
	sourceRoot string,
	includeContent func(string) bool,
) (traceGitIndexSnapshot, error) {
	sourceRoot = filepath.ToSlash(filepath.Clean(sourceRoot))
	if sourceRoot == "." || sourceRoot == "" || filepath.IsAbs(sourceRoot) || strings.HasPrefix(sourceRoot, "../") {
		return traceGitIndexSnapshot{}, fmt.Errorf("invalid Git index source root %q", sourceRoot)
	}
	if err := traceRejectGitWorktreeOverrides(repositoryRoot, sourceRoot); err != nil {
		return traceGitIndexSnapshot{}, err
	}
	listing, err := traceGitOutput(repositoryRoot, "ls-files", "--stage", "-z", "--", sourceRoot)
	if err != nil {
		return traceGitIndexSnapshot{}, err
	}
	entries, err := traceParseGitIndexEntries(listing, sourceRoot)
	if err != nil {
		return traceGitIndexSnapshot{}, err
	}
	selected := make([]traceGitIndexEntry, 0)
	for _, entry := range entries {
		if includeContent(entry.Path) {
			selected = append(selected, entry)
		}
	}
	contents, err := traceReadGitIndexBlobs(repositoryRoot, selected)
	if err != nil {
		return traceGitIndexSnapshot{}, err
	}
	return traceGitIndexSnapshot{Entries: entries, Contents: contents}, nil
}

func traceRejectGitWorktreeOverrides(repositoryRoot, sourceRoot string) error {
	unstaged, err := traceGitOutput(repositoryRoot, "diff", "--name-only", "-z", "--", sourceRoot)
	if err != nil {
		return err
	}
	if paths := traceGitNULPaths(unstaged); len(paths) != 0 {
		return fmt.Errorf("legacy source has unstaged worktree overrides: %v", paths)
	}
	untracked, err := traceGitOutput(repositoryRoot, "ls-files", "--others", "--exclude-standard", "-z", "--", sourceRoot)
	if err != nil {
		return err
	}
	if paths := traceGitNULPaths(untracked); len(paths) != 0 {
		return fmt.Errorf("legacy source has untracked worktree overrides: %v", paths)
	}
	return nil
}

func traceGitNULPaths(raw []byte) []string {
	parts := bytes.Split(raw, []byte{0})
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) != 0 {
			paths = append(paths, string(part))
		}
	}
	sort.Strings(paths)
	return paths
}

func traceParseGitIndexEntries(raw []byte, sourceRoot string) ([]traceGitIndexEntry, error) {
	records := bytes.Split(raw, []byte{0})
	entries := make([]traceGitIndexEntry, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	prefix := strings.TrimSuffix(sourceRoot, "/") + "/"
	for _, record := range records {
		if len(record) == 0 {
			continue
		}
		parts := bytes.SplitN(record, []byte{'\t'}, 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid Git index record")
		}
		metadata := strings.Fields(string(parts[0]))
		path := filepath.ToSlash(string(parts[1]))
		if len(metadata) != 3 || metadata[2] != "0" {
			return nil, fmt.Errorf("ambiguous Git index stage for %q", path)
		}
		if metadata[0] != "100644" && metadata[0] != "100755" {
			return nil, fmt.Errorf("non-regular Git index mode %q for %q", metadata[0], path)
		}
		decodedOID, err := hex.DecodeString(metadata[1])
		if err != nil || (len(decodedOID) != 20 && len(decodedOID) != 32) {
			return nil, fmt.Errorf("invalid Git object id for %q", path)
		}
		if path == "" || filepath.IsAbs(path) || filepath.Clean(path) != path ||
			!strings.HasPrefix(path, prefix) {
			return nil, fmt.Errorf("unsafe Git index path %q", path)
		}
		if _, duplicate := seen[path]; duplicate {
			return nil, fmt.Errorf("duplicate Git index path %q", path)
		}
		seen[path] = struct{}{}
		entries = append(entries, traceGitIndexEntry{
			Mode: metadata[0],
			OID:  metadata[1],
			Path: path,
		})
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Path < entries[right].Path
	})
	return entries, nil
}

func traceReadGitIndexBlobs(repositoryRoot string, entries []traceGitIndexEntry) (map[string][]byte, error) {
	if len(entries) == 0 {
		return map[string][]byte{}, nil
	}
	var requests bytes.Buffer
	for _, entry := range entries {
		_, _ = fmt.Fprintln(&requests, entry.OID)
	}
	command := exec.Command("git", "-C", repositoryRoot, "cat-file", "--batch")
	command.Stdin = &requests
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git cat-file --batch: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	reader := bufio.NewReader(bytes.NewReader(output))
	contents := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		header, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read Git blob header for %q: %w", entry.Path, err)
		}
		fields := strings.Fields(header)
		if len(fields) != 3 || fields[0] != entry.OID || fields[1] != "blob" {
			return nil, fmt.Errorf("unexpected Git blob header for %q: %q", entry.Path, strings.TrimSpace(header))
		}
		size, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil || size < 0 || size > traceGitIndexMaxBlobBytes {
			return nil, fmt.Errorf("invalid Git blob size %q for %q", fields[2], entry.Path)
		}
		content := make([]byte, int(size))
		if _, err := io.ReadFull(reader, content); err != nil {
			return nil, fmt.Errorf("read Git blob %q: %w", entry.Path, err)
		}
		separator, err := reader.ReadByte()
		if err != nil || separator != '\n' {
			return nil, fmt.Errorf("invalid Git blob separator for %q", entry.Path)
		}
		contents[entry.Path] = content
	}
	if trailing, err := reader.ReadByte(); err != io.EOF {
		return nil, fmt.Errorf("unexpected trailing Git batch output byte %q: %v", trailing, err)
	}
	return contents, nil
}

func traceGitOutput(repositoryRoot string, arguments ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", repositoryRoot}, arguments...)...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(arguments, " "), err, strings.TrimSpace(stderr.String()))
	}
	return output, nil
}

func traceLoadLegacySourceSnapshot(t *testing.T, repositoryRoot string) traceGitIndexSnapshot {
	t.Helper()
	snapshot := traceGitIndexSnapshot{Contents: make(map[string][]byte)}
	for _, source := range []struct {
		root    string
		include func(string) bool
	}{
		{
			root: "modulos",
			include: func(path string) bool {
				return strings.HasSuffix(path, ".md") || strings.HasSuffix(path, "_test.go")
			},
		},
		{
			root:    "cmd",
			include: func(path string) bool { return strings.HasSuffix(path, "_test.go") },
		},
		{
			root:    "deploy",
			include: func(path string) bool { return strings.HasSuffix(path, "_test.go") },
		},
		{
			root:    "scripts",
			include: func(path string) bool { return strings.HasSuffix(path, ".sh") },
		},
	} {
		part := traceLoadGitIndexSnapshot(t, repositoryRoot, source.root, source.include)
		snapshot.Entries = append(snapshot.Entries, part.Entries...)
		for path, content := range part.Contents {
			if _, duplicate := snapshot.Contents[path]; duplicate {
				t.Fatalf("duplicate legacy source %q across Git index roots", path)
			}
			snapshot.Contents[path] = content
		}
	}
	sort.Slice(snapshot.Entries, func(left, right int) bool {
		return snapshot.Entries[left].Path < snapshot.Entries[right].Path
	})
	return snapshot
}

func traceReadGitIndexOverlayFile(t *testing.T, path string, legacy traceGitIndexSnapshot) []byte {
	t.Helper()
	clean := filepath.ToSlash(filepath.Clean(path))
	if path == "" || filepath.IsAbs(path) || clean != path || strings.HasPrefix(clean, "../") {
		t.Fatalf("non-canonical source path %q", path)
	}
	if content, exists := legacy.Contents[path]; exists {
		return content
	}
	if strings.HasPrefix(path, "modulos/") ||
		strings.HasPrefix(path, "cmd/") ||
		strings.HasPrefix(path, "deploy/") ||
		strings.HasPrefix(path, "scripts/") {
		if strings.HasSuffix(path, ".md") ||
			strings.HasSuffix(path, "_test.go") ||
			strings.HasSuffix(path, ".sh") {
			t.Fatalf("legacy source %q is absent from Git index snapshot", path)
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func traceReadFrozenGitIndexOverlayFile(
	t *testing.T,
	path string,
	wantSHA256 string,
	allowAppendedContent bool,
	legacy traceGitIndexSnapshot,
) []byte {
	t.Helper()
	content := traceReadGitIndexOverlayFile(t, path, legacy)
	if traceBytesSHA256(content) == wantSHA256 {
		return content
	}
	if !allowAppendedContent {
		t.Fatalf("legacy source %q digest=%s, want %s", path, traceBytesSHA256(content), wantSHA256)
	}

	hasher := sha256.New()
	start := 0
	for start < len(content) {
		offset := bytes.IndexByte(content[start:], '\n')
		end := len(content)
		if offset >= 0 {
			end = start + offset + 1
		}
		_, _ = hasher.Write(content[start:end])
		if "sha256:"+hex.EncodeToString(hasher.Sum(nil)) == wantSHA256 {
			return content[:end]
		}
		start = end
	}
	t.Fatalf("append-only legacy source %q has no prefix with digest %s", path, wantSHA256)
	return nil
}

func traceBytesSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func traceBytesLines(t *testing.T, content []byte) []string {
	t.Helper()
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}

func TestTraceGitIndexSnapshotReadsSkipWorktreeFiles(t *testing.T) {
	repositoryRoot := t.TempDir()
	if _, err := traceGitOutput(repositoryRoot, "init", "--quiet"); err != nil {
		t.Fatal(err)
	}
	path := "modulos/orquesta-legacy/source.go"
	absolutePath := filepath.Join(repositoryRoot, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("package legacy\n")
	if err := os.WriteFile(absolutePath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := traceGitOutput(repositoryRoot, "add", "--", path); err != nil {
		t.Fatal(err)
	}
	if _, err := traceGitOutput(repositoryRoot, "update-index", "--skip-worktree", "--", path); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(repositoryRoot, "modulos")); err != nil {
		t.Fatal(err)
	}
	snapshot, err := traceReadGitIndexSnapshot(repositoryRoot, "modulos", func(candidate string) bool {
		return candidate == path
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Entries) != 1 || snapshot.Entries[0].Path != path {
		t.Fatalf("index entries=%#v, want exact hidden path", snapshot.Entries)
	}
	if got := snapshot.Contents[path]; !bytes.Equal(got, content) {
		t.Fatalf("hidden content=%q, want %q", got, content)
	}

	untrackedPath := filepath.Join(repositoryRoot, "modulos", "orquesta-legacy", "untracked.go")
	if err := os.MkdirAll(filepath.Dir(untrackedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(untrackedPath, []byte("package untracked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := traceReadGitIndexSnapshot(repositoryRoot, "modulos", func(string) bool { return true }); err == nil ||
		!strings.Contains(err.Error(), "untracked worktree overrides") {
		t.Fatalf("untracked legacy source error=%v", err)
	}
	if err := os.Remove(untrackedPath); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(absolutePath, []byte("package changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := traceGitOutput(repositoryRoot, "update-index", "--no-skip-worktree", "--", path); err != nil {
		t.Fatal(err)
	}
	if _, err := traceReadGitIndexSnapshot(repositoryRoot, "modulos", func(string) bool { return true }); err == nil ||
		!strings.Contains(err.Error(), "unstaged worktree overrides") {
		t.Fatalf("unstaged legacy source error=%v", err)
	}
}

func TestTraceGitIndexEntriesRejectAmbiguousOrUnsafeRecords(t *testing.T) {
	const oid = "0123456789012345678901234567890123456789"
	for name, record := range map[string]string{
		"conflict_stage": "100644 " + oid + " 2\tmodulos/orquesta-legacy/source.go\x00",
		"symlink_mode":   "120000 " + oid + " 0\tmodulos/orquesta-legacy/source.go\x00",
		"outside_root":   "100644 " + oid + " 0\tother/source.go\x00",
		"parent_escape":  "100644 " + oid + " 0\tmodulos/../other/source.go\x00",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := traceParseGitIndexEntries([]byte(record), "modulos"); err == nil {
				t.Fatal("unsafe Git index record was accepted")
			}
		})
	}
}
