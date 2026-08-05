package codexwork

import (
	"encoding/json"
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
