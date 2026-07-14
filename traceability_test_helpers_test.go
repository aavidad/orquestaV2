package orquesta_test

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"sort"
	"strings"
	"testing"
)

type traceRoadmap struct {
	CapabilityEntries []struct {
		ID       string `json:"id"`
		Decision string `json:"decision"`
	} `json:"capability_entries"`
}

func traceReadSourceLines(t *testing.T, path string) []string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}

func traceMarshalJSONLines[T any](t *testing.T, entries []T) []string {
	t.Helper()
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		content, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, string(content))
	}
	return lines
}

func traceStringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func traceSHA256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func traceRequireGoTestRef(t *testing.T, ref string, wantPresent bool) {
	t.Helper()
	parts := strings.Split(ref, "#")
	if len(parts) != 2 || !strings.HasSuffix(parts[0], "_test.go") || !strings.HasPrefix(parts[1], "Test") {
		t.Fatalf("invalid Go lesson test ref %q", ref)
	}
	content, err := os.ReadFile(parts[0])
	if err != nil {
		t.Fatalf("lesson test ref %q: %v", ref, err)
	}
	present := bytes.Contains(content, []byte("func "+parts[1]+"("))
	if present != wantPresent {
		if wantPresent {
			t.Fatalf("closed lesson test %q is absent", ref)
		}
		t.Fatalf("planned lesson test %q now exists; run it and update bug status/evidence", ref)
	}
}

func traceAcceptedCapabilities(t *testing.T) map[string]struct{} {
	t.Helper()
	content, err := os.ReadFile("product/roadmap.json")
	if err != nil {
		t.Fatal(err)
	}
	var roadmap traceRoadmap
	if err := json.Unmarshal(content, &roadmap); err != nil {
		t.Fatal(err)
	}
	accepted := make(map[string]struct{})
	for _, entry := range roadmap.CapabilityEntries {
		if entry.Decision == "accept" {
			accepted[entry.ID] = struct{}{}
		}
	}
	return accepted
}

func traceScanStrictJSONL(
	t *testing.T,
	reader io.Reader,
	path string,
	decode func(decoder *json.Decoder, lineNumber int),
) {
	t.Helper()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			t.Fatalf("%s:%d empty line", path, lineNumber)
		}
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		decode(decoder, lineNumber)
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			t.Fatalf("decode %s:%d trailing content: %v", path, lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}

func traceStringsDigest(lines []string) string {
	hash := sha256.New()
	for _, line := range lines {
		_, _ = io.WriteString(hash, line)
		_, _ = io.WriteString(hash, "\n")
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func traceSortedContains(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}

func traceDecodeStrict(t *testing.T, path string, target any) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("decode %s trailing content: %v", path, err)
	}
}
