//go:build linux

package firecrackerguest

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testSnapshotEntry struct {
	mode, name, target string
	content            []byte
}

func TestSnapshotValidatesAndMaterializesExactModesAndSafeSymlink(t *testing.T) {
	subject := guestTestDigest("subject")
	snapshot := buildSnapshot(t, subject, []testSnapshotEntry{
		{mode: "100755", name: "cmd/tool", content: []byte("executable fixture\n")},
		{mode: "100644", name: "go.mod", content: []byte("module example\n\ngo 1.25\n")},
		{mode: "120000", name: "tool-link", target: "cmd/tool"},
	}, nil)
	identity, err := ValidateSnapshot(context.Background(), bytes.NewReader(snapshot), subject)
	if err != nil {
		t.Fatal(err)
	}
	if identity.SubjectDigest != subject || identity.ObjectFormat != "sha1" ||
		identity.Entries != 3 || identity.Directories != 1 {
		t.Fatalf("identity=%+v", identity)
	}
	root := filepath.Join(t.TempDir(), "subject")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cleanupPrivateTree(context.Background(), root, maximumCleanupRecords())
	}()
	materialized, err := MaterializeSnapshot(
		context.Background(), bytes.NewReader(snapshot), root, subject,
	)
	if err != nil {
		t.Fatal(err)
	}
	if materialized != identity {
		t.Fatalf("materialized=%+v want=%+v", materialized, identity)
	}
	assertFile(t, filepath.Join(root, "go.mod"), "module example\n\ngo 1.25\n", 0o444)
	assertFile(t, filepath.Join(root, "cmd", "tool"), "executable fixture\n", 0o555)
	if info, err := os.Stat(filepath.Join(root, "cmd")); err != nil || info.Mode().Perm() != 0o555 {
		t.Fatalf("directory mode=%v err=%v", info, err)
	}
	target, err := os.Readlink(filepath.Join(root, "tool-link"))
	if err != nil || target != "cmd/tool" {
		t.Fatalf("symlink=%q err=%v", target, err)
	}
}

func TestSnapshotRejectsIdentityOrderChecksumTruncationAndTrailing(t *testing.T) {
	subject := guestTestDigest("subject")
	valid := buildSnapshot(t, subject, []testSnapshotEntry{
		{mode: "100644", name: "a", content: []byte("a")},
		{mode: "100644", name: "b", content: []byte("b")},
	}, nil)
	tests := []struct {
		name  string
		value []byte
		code  string
	}{
		{"subject mismatch", valid, CodeSnapshotInvalid},
		{"checksum", mutateGuestBytes(valid, len(valid)-1), CodeSnapshotInvalid},
		{"truncated", valid[:len(valid)-1], CodeSnapshotInvalid},
		{"trailing", append(append([]byte(nil), valid...), 1), CodeSnapshotInvalid},
		{
			"unordered",
			buildSnapshot(t, subject, []testSnapshotEntry{
				{mode: "100644", name: "b", content: []byte("b")},
				{mode: "100644", name: "a", content: []byte("a")},
			}, nil),
			CodeSnapshotInvalid,
		},
		{
			"head invalid",
			buildSnapshot(t, subject, nil, func(fields *[4]string) { fields[2] = "xyz" }),
			CodeSnapshotInvalid,
		},
		{
			"tree invalid",
			buildSnapshot(t, subject, nil, func(fields *[4]string) { fields[3] = strings.Repeat("A", 40) }),
			CodeSnapshotInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expected := subject
			if test.name == "subject mismatch" {
				expected = guestTestDigest("other")
			}
			if _, err := ValidateSnapshot(
				context.Background(), bytes.NewReader(test.value), expected,
			); ErrorCode(err) != test.code {
				t.Fatalf("code=%q want=%q err=%v", ErrorCode(err), test.code, err)
			}
		})
	}
}

func TestSnapshotRejectsTraversalUnsafeSymlinksOIDMismatchAndPreexistingEscape(t *testing.T) {
	subject := guestTestDigest("subject")
	tests := []struct {
		name  string
		entry testSnapshotEntry
	}{
		{"parent", testSnapshotEntry{mode: "100644", name: "../escape", content: []byte("x")}},
		{"absolute", testSnapshotEntry{mode: "100644", name: "/escape", content: []byte("x")}},
		{"git", testSnapshotEntry{mode: "100644", name: ".GiT/config", content: []byte("x")}},
		{"symlink parent", testSnapshotEntry{mode: "120000", name: "link", target: "../escape"}},
		{"symlink absolute", testSnapshotEntry{mode: "120000", name: "link", target: "/escape"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := buildSnapshot(t, subject, []testSnapshotEntry{test.entry}, nil)
			if _, err := ValidateSnapshot(
				context.Background(), bytes.NewReader(value), subject,
			); ErrorCode(err) != CodeSnapshotInvalid {
				t.Fatalf("code=%q err=%v", ErrorCode(err), err)
			}
		})
	}
	oidMismatch := buildSnapshot(t, subject, []testSnapshotEntry{
		{mode: "100644", name: "safe", content: []byte("content")},
	}, nil)
	entryOIDOffset := snapshotEntryOIDOffset(t, oidMismatch)
	oidMismatch[entryOIDOffset] ^= 1
	resealSnapshot(oidMismatch)
	if _, err := ValidateSnapshot(
		context.Background(), bytes.NewReader(oidMismatch), subject,
	); ErrorCode(err) != CodeSnapshotInvalid {
		t.Fatalf("oid code=%q err=%v", ErrorCode(err), err)
	}

	parent := t.TempDir()
	root, outside := filepath.Join(parent, "root"), filepath.Join(parent, "outside")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "dir")); err != nil {
		t.Fatal(err)
	}
	value := buildSnapshot(t, subject, []testSnapshotEntry{
		{mode: "100644", name: "dir/file", content: []byte("forbidden")},
	}, nil)
	if _, err := MaterializeSnapshot(
		context.Background(), bytes.NewReader(value), root, subject,
	); ErrorCode(err) != CodeMaterializeFailed {
		t.Fatalf("escape code=%q err=%v", ErrorCode(err), err)
	}
	if _, err := os.Stat(filepath.Join(outside, "file")); !os.IsNotExist(err) {
		t.Fatalf("outside materialized err=%v", err)
	}
}

func TestSnapshotRejectsOversizedFieldAndDeclaredContent(t *testing.T) {
	subject := guestTestDigest("subject")
	var oversized bytes.Buffer
	oversized.Write(snapshotStreamMagic)
	_ = binary.Write(&oversized, binary.BigEndian, uint32(snapshotFieldMax+1))
	if _, err := ValidateSnapshot(
		context.Background(), bytes.NewReader(oversized.Bytes()), subject,
	); ErrorCode(err) != CodeSnapshotLimit {
		t.Fatalf("field code=%q err=%v", ErrorCode(err), err)
	}
	value := buildSnapshot(t, subject, []testSnapshotEntry{
		{mode: "100644", name: "file", content: []byte("x")},
	}, nil)
	sizeOffset := snapshotEntrySizeOffset(t, value)
	binary.BigEndian.PutUint64(value[sizeOffset:sizeOffset+8], firecrackerMaxSnapshotForTest())
	resealSnapshot(value)
	if _, err := ValidateSnapshot(
		context.Background(), bytes.NewReader(value), subject,
	); ErrorCode(err) != CodeSnapshotInvalid {
		t.Fatalf("size code=%q err=%v", ErrorCode(err), err)
	}
}

func TestSnapshotCountsUniqueDirectoriesAndCombinedRecordLimit(t *testing.T) {
	subject := guestTestDigest("subject")
	value := buildSnapshot(t, subject, []testSnapshotEntry{
		{mode: "100644", name: "a/b/file", content: []byte("1")},
		{mode: "100644", name: "a/c/file", content: []byte("2")},
		{mode: "100644", name: "d/file", content: []byte("3")},
	}, nil)
	identity, err := ValidateSnapshot(context.Background(), bytes.NewReader(value), subject)
	if err != nil || identity.Entries != 3 || identity.Directories != 4 {
		t.Fatalf("identity=%+v err=%v", identity, err)
	}

	atLimit := SnapshotIdentity{Entries: maxSnapshotEntries - 3, Directories: 1}
	if err := countSnapshotPath(&atLimit, []string{"a"}, []string{"b"}); err != nil {
		t.Fatalf("exact combined limit rejected: %v", err)
	}
	overLimit := SnapshotIdentity{Entries: maxSnapshotEntries - 2, Directories: 1}
	if err := countSnapshotPath(&overLimit, []string{"a"}, []string{"b"}); ErrorCode(err) != CodeSnapshotLimit {
		t.Fatalf("combined limit code=%q err=%v", ErrorCode(err), err)
	}
}

func TestSnapshotHonorsCancelledContext(t *testing.T) {
	subject := guestTestDigest("subject")
	value := buildSnapshot(t, subject, []testSnapshotEntry{
		{mode: "100644", name: "file", content: []byte("content")},
	}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ValidateSnapshot(ctx, bytes.NewReader(value), subject); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel not preserved: %v", err)
	}
}

func buildSnapshot(
	t *testing.T,
	subject string,
	entries []testSnapshotEntry,
	mutateFields func(*[4]string),
) []byte {
	t.Helper()
	fields := [4]string{"sha1", subject, strings.Repeat("1", 40), strings.Repeat("2", 40)}
	if mutateFields != nil {
		mutateFields(&fields)
	}
	var output bytes.Buffer
	digest := sha256.New()
	canonical := io.MultiWriter(&output, digest)
	_, _ = canonical.Write(snapshotStreamMagic)
	writeGuestFields(t, canonical, fields[:]...)
	for _, entry := range entries {
		content := entry.content
		if entry.mode == "120000" {
			content = []byte(entry.target)
		}
		blob := sha1.New()
		_, _ = io.WriteString(blob, "blob ")
		_, _ = io.WriteString(blob, stringDecimal(len(content)))
		_, _ = blob.Write([]byte{0})
		_, _ = blob.Write(content)
		oid := hex.EncodeToString(blob.Sum(nil))
		_, _ = canonical.Write([]byte{snapshotEntryTag})
		writeGuestFields(t, canonical, entry.mode, entry.name, oid)
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(content)))
		_, _ = canonical.Write(size[:])
		_, _ = canonical.Write(content)
	}
	_ = output.WriteByte(snapshotEndTag)
	_, _ = output.Write(digest.Sum(nil))
	return output.Bytes()
}

func writeGuestFields(t *testing.T, writer io.Writer, values ...string) {
	t.Helper()
	for _, value := range values {
		var size [4]byte
		binary.BigEndian.PutUint32(size[:], uint32(len(value)))
		if _, err := writer.Write(size[:]); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(writer, value); err != nil {
			t.Fatal(err)
		}
	}
}

func resealSnapshot(value []byte) {
	checksumOffset := len(value) - sha256.Size
	digest := sha256.Sum256(value[:checksumOffset-1])
	copy(value[checksumOffset:], digest[:])
}

func snapshotEntryOIDOffset(t *testing.T, value []byte) int {
	t.Helper()
	offset := len(snapshotStreamMagic)
	for range 4 {
		length := int(binary.BigEndian.Uint32(value[offset : offset+4]))
		offset += 4 + length
	}
	offset++
	for range 2 {
		length := int(binary.BigEndian.Uint32(value[offset : offset+4]))
		offset += 4 + length
	}
	length := int(binary.BigEndian.Uint32(value[offset : offset+4]))
	if length != sha1.Size*2 {
		t.Fatalf("oid length=%d", length)
	}
	return offset + 4
}

func snapshotEntrySizeOffset(t *testing.T, value []byte) int {
	t.Helper()
	offset := snapshotEntryOIDOffset(t, value)
	return offset + sha1.Size*2
}

func mutateGuestBytes(value []byte, offset int) []byte {
	result := append([]byte(nil), value...)
	result[offset] ^= 1
	return result
}

func assertFile(t *testing.T, name, content string, mode os.FileMode) {
	t.Helper()
	value, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if string(value) != content || info.Mode().Perm() != mode {
		t.Fatalf("%s content=%q mode=%o", name, value, info.Mode().Perm())
	}
}

func guestTestDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func stringDecimal(value int) string {
	if value == 0 {
		return "0"
	}
	var encoded [32]byte
	index := len(encoded)
	for value > 0 {
		index--
		encoded[index] = byte('0' + value%10)
		value /= 10
	}
	return string(encoded[index:])
}

func firecrackerMaxSnapshotForTest() uint64 { return 16<<30 + 1 }
