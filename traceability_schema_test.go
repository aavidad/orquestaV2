package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

func TestTraceabilityRebuildSchemaValidatesCanonicalLedgers(t *testing.T) {
	const schemaPath = "product/traceability/schema.json"
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	traceRequireJSONWithoutDuplicateKeys(t, schemaPath, schemaBytes)
	var schema jsonschema.Schema
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode traceability schema: %v", err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("compile traceability schema: %v", err)
	}

	entries, err := os.ReadDir("product/traceability")
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == filepath.Base(schemaPath) {
			continue
		}
		if extension := filepath.Ext(entry.Name()); extension == ".json" || extension == ".jsonl" {
			paths = append(paths, filepath.Join("product/traceability", entry.Name()))
		}
	}
	sort.Strings(paths)
	if len(paths) < 15 {
		t.Fatalf("canonical traceability documents=%d, want at least 15", len(paths))
	}
	validated := 0
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Ext(path) == ".json" {
			traceRequireJSONWithoutDuplicateKeys(t, path, content)
			instance := traceDecodeSchemaInstance(t, path, content)
			if err := resolved.Validate(instance); err != nil {
				t.Errorf("schema rejects %s: %v", path, err)
			}
			validated++
			continue
		}
		scanner := bufio.NewScanner(bytes.NewReader(content))
		scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
		for line := 1; scanner.Scan(); line++ {
			traceRequireJSONWithoutDuplicateKeys(t, fmt.Sprintf("%s:%d", path, line), scanner.Bytes())
			instance := traceDecodeSchemaInstance(t, fmt.Sprintf("%s:%d", path, line), scanner.Bytes())
			if err := resolved.Validate(instance); err != nil {
				t.Errorf("schema rejects %s:%d: %v", path, line, err)
			}
			validated++
		}
		if err := scanner.Err(); err != nil {
			t.Fatal(err)
		}
	}
	if validated < 2000 {
		t.Fatalf("schema validated records=%d, want at least 2000", validated)
	}

	validEntry := traceDecodeSchemaInstance(t, "task fixture", []byte(`{
		"schema_version":2,"entry_ref":"TASKENTRY-aaaaaaaaaaaaaaaaaaaaaaaa","candidate_ref":"TASKCAND-aaaaaaaaaaaaaaaaaaaaaaaa",
		"source_ref":"docs/x.md","source_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"source_history":"historical","source_line":1,"detection_kind":"heading_id",
		"marker_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"subject_first_line":1,"subject_last_line":1,"subject_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"original_state":"open","capability_id":"GOV-16","capability_decision":"accept",
		"disposition":"historical_superseded_by_capability","basis_code":"independent_semantic_review",
		"semantic_review_state":"reviewed","semantic_reason":"Specific independently reviewed semantic reason.",
		"semantic_review_pass_ref":"TASKREVIEWPASS-aaaaaaaaaaaaaaaaaaaaaaaa","semantic_review_ref":"TASKREVIEW-aaaaaaaaaaaaaaaaaaaaaaaa",
		"semantic_review_verdict":"confirmed","previous_capability_id":"GOV-16",
		"decision_ref":"product/roadmap.json#capability_entries/GOV-16","closure_evidence":"not_verified"
	}`)).(map[string]any)
	for _, mutation := range []struct {
		name string
		edit func(map[string]any)
	}{
		{name: "missing pass", edit: func(value map[string]any) { delete(value, "semantic_review_pass_ref") }},
		{name: "unknown field", edit: func(value map[string]any) { value["parallel_authority"] = true }},
		{name: "invalid closure", edit: func(value map[string]any) { value["closure_evidence"] = "legacy_closed" }},
	} {
		t.Run("reject_"+strings.ReplaceAll(mutation.name, " ", "_"), func(t *testing.T) {
			copyValue := make(map[string]any, len(validEntry))
			for key, value := range validEntry {
				copyValue[key] = value
			}
			mutation.edit(copyValue)
			if err := resolved.Validate(copyValue); err == nil {
				t.Fatalf("schema accepted adversarial mutation %q", mutation.name)
			}
		})
	}

	historicalContent, err := os.ReadFile("product/traceability/historical_bug_ids.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	historicalLine, _, _ := bytes.Cut(historicalContent, []byte{'\n'})
	historical := traceDecodeSchemaInstance(t, "historical closure fixture", historicalLine).(map[string]any)
	historical["closure_evidence"] = "verified"
	if err := resolved.Validate(historical); err == nil {
		t.Fatal("schema allowed capability coverage to claim historical incident closure")
	}

	rootsContent, err := os.ReadFile("product/traceability/legacy_source_roots_2026-07-30.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []struct {
		name string
		edit func(map[string]any)
	}{
		{name: "campo superior desconocido", edit: func(value map[string]any) {
			value["autoridad_paralela"] = true
		}},
		{name: "cierre sin censo", edit: func(value map[string]any) {
			value["closed"] = true
		}},
		{name: "digest bruto inventado", edit: func(value map[string]any) {
			batches := value["observation_batches"].([]any)
			batches[0].(map[string]any)["raw_census_sha256"] = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}},
		{name: "raíz sin estado físico", edit: func(value map[string]any) {
			roots := value["roots"].([]any)
			delete(roots[0].(map[string]any), "physical_census_status")
		}},
	} {
		t.Run("rechaza_raices_"+strings.ReplaceAll(mutation.name, " ", "_"), func(t *testing.T) {
			value := traceDecodeSchemaInstance(t, "raíces históricas", rootsContent).(map[string]any)
			mutation.edit(value)
			if err := resolved.Validate(value); err == nil {
				t.Fatalf("el esquema aceptó la mutación adversarial %q", mutation.name)
			}
		})
	}
}

func traceDecodeSchemaInstance(t *testing.T, source string, content []byte) any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(content))
	var instance any
	if err := decoder.Decode(&instance); err != nil {
		t.Fatalf("decode %s: %v", source, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatalf("trailing JSON in %s: %v", source, err)
	}
	return instance
}

func traceRequireJSONWithoutDuplicateKeys(t *testing.T, source string, content []byte) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := traceReadUniqueJSONValue(decoder, source); err != nil {
		t.Fatal(err)
	}
	if token, err := decoder.Token(); err != io.EOF {
		t.Fatalf("trailing JSON token in %s: token=%v err=%v", source, token, err)
	}
}

func traceReadUniqueJSONValue(decoder *json.Decoder, source string) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode token in %s: %w", source, err)
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("decode object key in %s: %w", source, err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("non-string object key in %s", source)
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON key %q in %s", key, source)
			}
			seen[key] = struct{}{}
			if err := traceReadUniqueJSONValue(decoder, source); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := traceReadUniqueJSONValue(decoder, source); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q in %s", delimiter, source)
	}
	closing, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode closing delimiter in %s: %w", source, err)
	}
	want := json.Delim('}')
	if delimiter == '[' {
		want = ']'
	}
	if closing != want {
		return fmt.Errorf("closing delimiter in %s is %q, want %q", source, closing, want)
	}
	return nil
}
