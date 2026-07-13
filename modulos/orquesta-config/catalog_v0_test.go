package orquestaconfig

import "testing"

func TestBuildCatalogV0HandlesEmbeddedFieldsAndCollisions(t *testing.T) {
	type embedded struct {
		Name   *string `json:"name,omitempty"`
		Shared string  `json:"shared"`
	}
	type schema struct {
		embedded
		Shared  string `json:"shared"`
		Enabled bool   `json:"enabled"`
	}
	leaves, err := ReflectJSONLeavesV0(schema{})
	if err != nil {
		t.Fatal(err)
	}
	policies := make([]LeafPolicyV0, 0, len(leaves))
	for _, leaf := range leaves {
		policies = append(policies, LeafPolicyV0{Pointer: leaf.Pointer, Presence: leaf.Presence, Editable: true, EditabilityReason: "test", SetterRef: "test", RestartBehavior: RestartBehaviorNoneV0})
	}
	catalog, err := BuildCatalogV0(schema{}, policies)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) != 3 {
		t.Fatalf("catalog=%+v", catalog)
	}
	for _, entry := range catalog {
		if entry.Pointer == "/name" && entry.Presence != PresenceOptionalV0 {
			t.Fatalf("pointer presence=%+v", entry)
		}
	}
}

func TestBuildCatalogV0RejectsPolicyPresenceThatDiffersFromSchema(t *testing.T) {
	type schema struct {
		Optional *string `json:"optional,omitempty"`
	}
	_, err := BuildCatalogV0(schema{}, []LeafPolicyV0{{
		Pointer: "/optional", Presence: PresenceRequiredV0, Editable: false,
		EditabilityReason: "test", RestartBehavior: RestartBehaviorNoneV0,
	}})
	if err == nil {
		t.Fatal("catalog accepted policy presence that differs from the JSON schema")
	}
}

func TestReflectJSONLeavesV0TracksOmitEmptyAndOptionalParentsLikeEncodingJSON(t *testing.T) {
	type child struct {
		Value string `json:"value"`
	}
	type left struct {
		Collision string `json:"collision"`
	}
	type right struct {
		Collision string `json:"collision"`
	}
	type schema struct {
		Config   *child `json:"config,omitempty"`
		Omit     string `json:"omit,omitempty"`
		Required string `json:"required"`
		left
		right
	}
	leaves, err := ReflectJSONLeavesV0(schema{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]PresenceV0{}
	for _, leaf := range leaves {
		got[leaf.Pointer] = leaf.Presence
	}
	if got["/config/value"] != PresenceOptionalV0 || got["/omit"] != PresenceOptionalV0 || got["/required"] != PresenceRequiredV0 {
		t.Fatalf("presence=%+v", got)
	}
	if _, collision := got["/collision"]; collision {
		t.Fatalf("ambiguous encoding/json field leaked into catalog: %+v", got)
	}
}
