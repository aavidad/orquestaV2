package codexwork

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func validPacket() WorkPacketV1 {
	return WorkPacketV1{
		Schema: WorkPacketSchemaV1, Prompt: "  conserva bytes\nexactos  ",
		Model: "gpt-5.6", Effort: "high", TokenBudget: 4096,
		MaxOutputBytes: 4096, TimeBudgetMS: 60_000,
		GoalRef: "opaque goal/α", WorkItemRef: "opaque-work",
		ExecutionRef: "opaque-execution", EffectAttemptRef: "opaque-attempt",
	}
}

func TestWorkPacketExactJSONFieldRegistryDoesNotDrift(t *testing.T) {
	typeOfPacket := reflect.TypeOf(WorkPacketV1{})
	if typeOfPacket.NumField() != len(workPacketJSONFields) {
		t.Fatalf("packet fields=%d registry=%d", typeOfPacket.NumField(), len(workPacketJSONFields))
	}
	for index := 0; index < typeOfPacket.NumField(); index++ {
		parts := strings.Split(typeOfPacket.Field(index).Tag.Get("json"), ",")
		name := parts[0]
		required, exists := workPacketJSONFields[name]
		if !exists {
			t.Fatalf("field %s (%q) missing from exact registry", typeOfPacket.Field(index).Name, name)
		}
		optional := len(parts) > 1 && parts[1] == "omitempty"
		if required == optional {
			t.Fatalf("field %q required=%v optional_tag=%v", name, required, optional)
		}
	}
}

func TestDecodeWorkPacketStrictAndExact(t *testing.T) {
	want := validPacket()
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeWorkPacketV1(append(raw, '\n', ' ', '\t'))
	if err != nil {
		t.Fatal(err)
	}
	if got != want || got.Prompt != "  conserva bytes\nexactos  " {
		t.Fatalf("packet alterado: %#v", got)
	}

	invalid := []struct {
		name string
		raw  string
		code Code
	}{
		{"unknown", strings.TrimSuffix(string(raw), "}") + `,"credential":"secret"}`, CodePacketMalformed},
		{"trailing", string(raw) + `{}`, CodePacketMalformed},
		{"duplicate", strings.Replace(string(raw), `"schema":`, `"schema":"duplicate","schema":`, 1), CodePacketMalformed},
		{"uppercase egress key", strings.TrimSuffix(string(raw), "}") + `,"CONTROLLED_EGRESS_PROXY":"http://127.0.0.1:18080"}`, CodePacketMalformed},
		{"uppercase required key", strings.Replace(string(raw), `"schema":`, `"SCHEMA":`, 1), CodePacketMalformed},
		{"schema", strings.Replace(string(raw), WorkPacketSchemaV1, "old", 1), CodePacketSchema},
		{"negative", strings.Replace(string(raw), `"token_budget":4096`, `"token_budget":-1`, 1), CodePacketMalformed},
		{"fraction", strings.Replace(string(raw), `"time_budget_ms":60000`, `"time_budget_ms":1.5`, 1), CodePacketMalformed},
		{"null", `null`, CodePacketMalformed},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeWorkPacketV1([]byte(test.raw)); ErrorCode(err) != test.code {
				t.Fatalf("code=%q err=%v", ErrorCode(err), err)
			}
		})
	}

	tooLarge := make([]byte, MaxPacketBytesV1+1)
	if _, err := DecodeWorkPacketV1(tooLarge); ErrorCode(err) != CodePacketTooLarge {
		t.Fatalf("oversize: %v", err)
	}
	if _, err := DecodeWorkPacketV1([]byte{0xff}); ErrorCode(err) != CodePacketMalformed {
		t.Fatalf("utf8: %v", err)
	}
}

func TestWorkPacketEgressExtensionRequiresTheMatchingGuestAsset(t *testing.T) {
	type legacyWorkPacketWithoutEgress struct {
		Schema           string `json:"schema"`
		Prompt           string `json:"prompt"`
		Model            string `json:"model"`
		Effort           string `json:"effort"`
		TokenBudget      uint64 `json:"token_budget"`
		MaxOutputBytes   uint64 `json:"max_output_bytes"`
		TimeBudgetMS     uint64 `json:"time_budget_ms"`
		GoalRef          string `json:"goal_ref"`
		WorkItemRef      string `json:"work_item_ref"`
		ExecutionRef     string `json:"execution_ref"`
		EffectAttemptRef string `json:"effect_attempt_ref"`
	}
	decodeLegacy := func(raw []byte) error {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		return decoder.Decode(&legacyWorkPacketWithoutEgress{})
	}

	legacyRaw, err := json.Marshal(validPacket())
	if err != nil {
		t.Fatal(err)
	}
	if legacyErr := decodeLegacy(legacyRaw); legacyErr != nil {
		t.Fatalf("old->new compatible packet=%s error=%v", legacyRaw, legacyErr)
	}
	withEgress := validPacket()
	withEgress.ControlledEgressProxy = ControlledEgressProxyURLV1
	newRaw, err := json.Marshal(withEgress)
	if err != nil {
		t.Fatal(err)
	}
	if err := decodeLegacy(newRaw); err == nil {
		t.Fatal("guest asset anterior aceptó la extensión de egreso")
	}
}

func TestWorkPacketRejectsInvalidFieldsWithoutInterpretingOpaqueRefs(t *testing.T) {
	mutations := map[string]func(*WorkPacketV1){
		"prompt":        func(packet *WorkPacketV1) { packet.Prompt = "" },
		"model trimmed": func(packet *WorkPacketV1) { packet.Model = " gpt " },
		"effort":        func(packet *WorkPacketV1) { packet.Effort = "maximum" },
		"tokens zero":   func(packet *WorkPacketV1) { packet.TokenBudget = 0 },
		"tokens high":   func(packet *WorkPacketV1) { packet.TokenBudget = MaxTokenBudgetV1 + 1 },
		"output zero":   func(packet *WorkPacketV1) { packet.MaxOutputBytes = 0 },
		"output high":   func(packet *WorkPacketV1) { packet.MaxOutputBytes = MaxOutputBytesV1 + 1 },
		"time zero":     func(packet *WorkPacketV1) { packet.TimeBudgetMS = 0 },
		"time high":     func(packet *WorkPacketV1) { packet.TimeBudgetMS = MaxTimeBudgetMSV1 + 1 },
		"goal empty":    func(packet *WorkPacketV1) { packet.GoalRef = "" },
		"work control":  func(packet *WorkPacketV1) { packet.WorkItemRef = "work\nitem" },
		"execution":     func(packet *WorkPacketV1) { packet.ExecutionRef = " execution" },
		"attempt":       func(packet *WorkPacketV1) { packet.EffectAttemptRef = "" },
		"egress proxy": func(packet *WorkPacketV1) {
			packet.ControlledEgressProxy = "http://127.0.0.1:18081"
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			packet := validPacket()
			mutate(&packet)
			if err := packet.Validate(); ErrorCode(err) != CodePacketField {
				t.Fatalf("code=%q err=%v", ErrorCode(err), err)
			}
		})
	}
}

func TestWorkPacketAcceptsOnlyTheSealedControlledEgressProxy(t *testing.T) {
	packet := validPacket()
	packet.ControlledEgressProxy = ControlledEgressProxyURLV1
	raw, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeWorkPacketV1(raw)
	if err != nil || decoded != packet {
		t.Fatalf("controlled egress round trip=%+v error=%v", decoded, err)
	}

	for _, mutated := range []string{
		"http://127.0.0.1:18081",
		"http://192.168.1.1:18080",
		"https://proxy.example.com",
		"http://127.0.0.1:18080/extra",
	} {
		candidate := packet
		candidate.ControlledEgressProxy = mutated
		if err := candidate.Validate(); ErrorCode(err) != CodePacketField {
			t.Fatalf("mutated proxy %q code=%q error=%v", mutated, ErrorCode(err), err)
		}
	}
}
