package orquestaconfig

import (
	"errors"
	"testing"
)

func TestCanonicalJSONV0RevisionAndStrictBoundedDecode(t *testing.T) {
	first, err := EncodeCanonicalJSONV0(map[string]any{"z": 1, "a": "value"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := EncodeCanonicalJSONV0(map[string]any{"a": "value", "z": 1})
	if err != nil {
		t.Fatal(err)
	}
	if string(first.Bytes) != string(second.Bytes) || first.Revision != RevisionSHA256V0(first.Bytes) {
		t.Fatalf("canonical=%+v second=%+v", first, second)
	}
	var target struct {
		Known string `json:"known"`
	}
	if err := DecodeStrictBoundedJSONV0([]byte(`{"known":"too long"}`), &target, 2); !errors.Is(err, ErrDocumentTooLargeV0) {
		t.Fatalf("err=%v", err)
	}
}

func TestDecodeStrictBoundedJSONV0KeepsTargetOnError(t *testing.T) {
	target := struct {
		Known string `json:"known"`
	}{Known: "stable"}
	if err := DecodeStrictBoundedJSONV0([]byte(`{"known":"candidate","unknown":true}`), &target, 100); err == nil {
		t.Fatal("unknown field accepted")
	}
	if target.Known != "stable" {
		t.Fatalf("target mutated after rejected decode: %+v", target)
	}
}
