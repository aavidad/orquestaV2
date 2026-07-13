package main

import "testing"

import orquestaconfig "orquesta/modulos/orquesta-config"

func TestServerProjectConfigCatalogV0CoversRealSchemaWithExplicitPolicies(t *testing.T) {
	catalog, err := serverProjectConfigCatalogV0()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) < 50 {
		t.Fatalf("catalog unexpectedly small: %d", len(catalog))
	}
	seen := map[string]bool{}
	var telegramTokenFound bool
	var apiKeyFileFound bool
	for _, entry := range catalog {
		if seen[entry.Pointer] || entry.Presence == "" || entry.EditabilityReason == "" || entry.RestartBehavior == "" || entry.Editable || entry.SetterRef != "" {
			t.Fatalf("incomplete or phantom-setter catalog entry: %+v", entry)
		}
		seen[entry.Pointer] = true
		if entry.Pointer == "/telegram_operator/token" {
			telegramTokenFound = true
			if !entry.Sensitive || entry.Editable || entry.SetterRef != "" {
				t.Fatalf("telegram token policy=%+v", entry)
			}
		}
		if entry.Pointer == "/hermes_operator/api_key_file" {
			apiKeyFileFound = true
			if entry.Sensitive {
				t.Fatalf("api_key_file is a reference, not a secret: %+v", entry)
			}
		}
	}
	if !seen["/schema_version"] || !seen["/control_plane/token"] || !seen["/telegram_operator/token"] {
		t.Fatalf("real leaves missing: %+v", seen)
	}
	if !telegramTokenFound {
		t.Fatal("telegram token policy missing")
	}
	if !apiKeyFileFound {
		t.Fatal("api_key_file policy missing")
	}
}

func TestServerProjectConfigCatalogV0RejectsUnassignedLeafPolicyV0(t *testing.T) {
	for _, pointer := range []string{
		"/unassigned/leaf",
		"/unknown_root/leaf",
		"/server/new_leaf",
		"/autoprogramming/new_leaf",
	} {
		if _, ok := serverProjectConfigPolicyForLeafV0(orquestaconfig.JSONLeafV0{Pointer: pointer, Presence: orquestaconfig.PresenceOptionalV0}); ok {
			t.Fatalf("unassigned leaf received an implicit policy: %s", pointer)
		}
	}
}
