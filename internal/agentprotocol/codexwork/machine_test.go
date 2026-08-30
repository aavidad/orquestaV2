package codexwork

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func mustMachine(t *testing.T) *Machine {
	t.Helper()
	machine, err := NewMachine(validPacket())
	if err != nil {
		t.Fatal(err)
	}
	return machine
}

func frameID(t *testing.T, frame []byte) string {
	t.Helper()
	var envelope struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(frame, &envelope); err != nil || envelope.ID == "" {
		t.Fatalf("frame sin id: %s %v", frame, err)
	}
	return envelope.ID
}

func advanceRunning(t *testing.T, machine *Machine) (string, string) {
	t.Helper()
	initialize, err := machine.Start()
	if err != nil {
		t.Fatal(err)
	}
	initResponse := `{"id":` + quote(frameID(t, initialize)) + `,"result":{"userAgent":"codex","codexHome":"/sealed","platformFamily":"unix","platformOs":"linux"}}`
	transition, err := machine.AcceptJSONL([]byte(initResponse))
	if err != nil || len(transition.Outbound) != 2 {
		t.Fatalf("initialize: %#v %v", transition, err)
	}
	threadRequest := transition.Outbound[1]
	transition, err = machine.AcceptJSONL([]byte(validThreadResponse(frameID(t, threadRequest), "thread-1", validPacket().Model)))
	if err != nil || len(transition.Outbound) != 1 {
		t.Fatalf("thread: %#v %v", transition, err)
	}
	turnRequest := transition.Outbound[0]
	if _, err := machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, turnRequest)) + `,"result":{"turn":{"id":"turn-1","status":"inProgress","items":[]}}}`)); err != nil {
		t.Fatal(err)
	}
	return "thread-1", "turn-1"
}

func quote(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func validThreadResponse(requestID, threadID, model string) string {
	return `{"id":` + quote(requestID) + `,"result":{"approvalPolicy":"never","cwd":"/trabajo","model":` + quote(model) +
		`,"runtimeWorkspaceRoots":["/trabajo"],"sandbox":{"type":"dangerFullAccess"},"thread":{"id":` + quote(threadID) + `}}}`
}

func TestMachineEmitsCurrentJSONLAndPreservesPrompt(t *testing.T) {
	machine := mustMachine(t)
	initialize, err := machine.Start()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(initialize, []byte("jsonrpc")) || !bytes.HasSuffix(initialize, []byte("\n")) {
		t.Fatalf("initialize dialecto incorrecto: %s", initialize)
	}
	var init struct {
		Method string `json:"method"`
		Params struct {
			Capabilities initializeCapabilities `json:"capabilities"`
		} `json:"params"`
	}
	if json.Unmarshal(initialize, &init) != nil || init.Method != "initialize" || !init.Params.Capabilities.ExperimentalAPI {
		t.Fatalf("initialize=%s", initialize)
	}

	transition, err := machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, initialize)) + `,"result":{"userAgent":"codex","codexHome":"/not-exported","platformFamily":"unix","platformOs":"linux"}}`))
	if err != nil || string(transition.Outbound[0]) != "{\"method\":\"initialized\"}\n" {
		t.Fatalf("initialized: %#v %v", transition, err)
	}
	threadRequest := transition.Outbound[1]
	if bytes.Contains(threadRequest, []byte("jsonrpc")) || !bytes.Contains(threadRequest, []byte(`"cwd":"/trabajo"`)) ||
		bytes.Contains(threadRequest, []byte("Home")) || bytes.Contains(threadRequest, []byte("credential")) {
		t.Fatalf("thread/start filtró detalle prohibido: %s", threadRequest)
	}

	transition, err = machine.AcceptJSONL([]byte(validThreadResponse(frameID(t, threadRequest), "thread-exact", validPacket().Model)))
	if err != nil {
		t.Fatal(err)
	}
	turnRequest := transition.Outbound[0]
	var turn struct {
		Method string `json:"method"`
		Params struct {
			ThreadID            string      `json:"threadId"`
			Input               []textInput `json:"input"`
			ClientUserMessageID string      `json:"clientUserMessageId"`
			Model               string      `json:"model"`
			Effort              string      `json:"effort"`
			OutputSchema        struct {
				Type                 string                            `json:"type"`
				AdditionalProperties bool                              `json:"additionalProperties"`
				Properties           map[string]artifactSchemaProperty `json:"properties"`
				Required             []string                          `json:"required"`
			} `json:"outputSchema"`
		} `json:"params"`
	}
	if err := json.Unmarshal(turnRequest, &turn); err != nil {
		t.Fatal(err)
	}
	if turn.Method != "turn/start" || turn.Params.ThreadID != "thread-exact" || len(turn.Params.Input) != 1 ||
		turn.Params.Input[0].Text != validPacket().Prompt || turn.Params.Model != validPacket().Model ||
		turn.Params.Effort != validPacket().Effort || turn.Params.ClientUserMessageID == "" ||
		turn.Params.OutputSchema.Type != "object" || turn.Params.OutputSchema.AdditionalProperties ||
		turn.Params.OutputSchema.Properties["artifact"].Type != "string" ||
		len(turn.Params.OutputSchema.Required) != 1 || turn.Params.OutputSchema.Required[0] != "artifact" {
		t.Fatalf("turn/start inválido: %#v", turn)
	}
	if bytes.Contains(turnRequest, []byte("token_budget")) || bytes.Contains(turnRequest, []byte("time_budget")) {
		t.Fatalf("campos no soportados enviados al proveedor: %s", turnRequest)
	}
}

func TestMachineIDsAreDeterministicAndPacketBound(t *testing.T) {
	first, second := mustMachine(t), mustMachine(t)
	a, _ := first.Start()
	b, _ := second.Start()
	if frameID(t, a) != frameID(t, b) {
		t.Fatalf("IDs no deterministas: %s %s", a, b)
	}
	changed := validPacket()
	changed.ExecutionRef = "another-execution"
	third, err := NewMachine(changed)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := third.Start()
	if frameID(t, a) == frameID(t, c) {
		t.Fatal("ID no ligado al packet")
	}
}

func TestNewMachineRejectsOversizedCanonicalPacket(t *testing.T) {
	packet := validPacket()
	packet.Prompt = strings.Repeat("x", MaxPacketBytesV1)
	if _, err := NewMachine(packet); ErrorCode(err) != CodePacketTooLarge {
		t.Fatalf("canonical oversize code=%q err=%v", ErrorCode(err), err)
	}
}

func TestWorkResultJSONContractIsExact(t *testing.T) {
	raw, err := json.Marshal(WorkResultV1{Artifact: " bytes exactos "})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"artifact":" bytes exactos "}` {
		t.Fatalf("resultado JSON inesperado: %s", raw)
	}
}

func TestMachineRejectsThreadResponseWithoutSealedExecutionContract(t *testing.T) {
	for _, mutation := range []struct{ old, new string }{
		{`"approvalPolicy":"never"`, `"approvalPolicy":"on-request"`},
		{`"cwd":"/trabajo"`, `"cwd":"/otro"`},
		{`"runtimeWorkspaceRoots":["/trabajo"]`, `"runtimeWorkspaceRoots":[]`},
		{`"type":"dangerFullAccess"`, `"type":"workspaceWrite"`},
	} {
		machine := mustMachine(t)
		initialize, _ := machine.Start()
		transition, err := machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, initialize)) + `,"result":{"userAgent":"codex","codexHome":"/sealed","platformFamily":"unix","platformOs":"linux"}}`))
		if err != nil {
			t.Fatal(err)
		}
		request := transition.Outbound[1]
		response := strings.Replace(validThreadResponse(frameID(t, request), "thread-1", validPacket().Model), mutation.old, mutation.new, 1)
		if _, err := machine.AcceptJSONL([]byte(response)); ErrorCode(err) != CodeFrameMalformed {
			t.Fatalf("contrato relajado aceptado: %s: %v", response, err)
		}
	}
}

func TestMachineAcceptsSealedEnvironmentSettingsAndFailsClosedOnDisconnect(t *testing.T) {
	machine := mustMachine(t)
	threadID, _ := advanceRunning(t, machine)
	if _, err := machine.AcceptJSONL([]byte(`{"method":"thread/environment/connected","params":{"environmentId":"local","threadId":` + quote(threadID) + `}}`)); err != nil {
		t.Fatal(err)
	}
	settings := `{"method":"thread/settings/updated","params":{"threadId":` + quote(threadID) + `,"threadSettings":{"approvalPolicy":"never","cwd":"/trabajo","effort":` + quote(validPacket().Effort) + `,"model":` + quote(validPacket().Model) + `,"sandboxPolicy":{"type":"dangerFullAccess"}}}}`
	if _, err := machine.AcceptJSONL([]byte(settings)); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.AcceptJSONL([]byte(`{"method":"thread/environment/disconnected","params":{"environmentId":"local","threadId":` + quote(threadID) + `}}`)); ErrorCode(err) != CodeRemote || machine.State() != StateFailed {
		t.Fatalf("disconnect=%v state=%s", err, machine.State())
	}
}

func TestMachineBuffersStartedNotificationsBeforeResponses(t *testing.T) {
	machine := mustMachine(t)
	initialize, _ := machine.Start()
	transition, err := machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, initialize)) + `,"result":{"userAgent":"codex","codexHome":"/sealed","platformFamily":"unix","platformOs":"linux"}}`))
	if err != nil {
		t.Fatal(err)
	}
	threadRequest := transition.Outbound[1]
	threadStarted := `{"method":"thread/started","params":{"thread":{"id":"thread-buffered"}}}`
	if _, err := machine.AcceptJSONL([]byte(threadStarted)); err != nil {
		t.Fatalf("thread/started temprano: %v", err)
	}
	if _, err := machine.AcceptJSONL([]byte(` {"method":"thread/started","params":{"thread":{"id":"thread-buffered"}}} `)); ErrorCode(err) != CodeDuplicateFrame {
		t.Fatalf("thread/started duplicado: %v", err)
	}
	if _, err := machine.AcceptJSONL([]byte(`{"method":"thread/started","params":{"thread":{"id":"thread-other"}}}`)); ErrorCode(err) != CodeThreadMismatch {
		t.Fatalf("thread/started discordante: %v", err)
	}
	transition, err = machine.AcceptJSONL([]byte(validThreadResponse(frameID(t, threadRequest), "thread-buffered", validPacket().Model)))
	if err != nil {
		t.Fatalf("respuesta thread reconciliada: %v", err)
	}
	turnRequest := transition.Outbound[0]
	turnStarted := `{"method":"turn/started","params":{"threadId":"thread-buffered","turn":{"id":"turn-buffered","status":"inProgress","items":[]}}}`
	if _, err := machine.AcceptJSONL([]byte(turnStarted)); err != nil {
		t.Fatalf("turn/started temprano: %v", err)
	}
	if _, err := machine.AcceptJSONL([]byte(` {"method":"turn/started","params":{"threadId":"thread-buffered","turn":{"id":"turn-buffered"}}} `)); ErrorCode(err) != CodeDuplicateFrame {
		t.Fatalf("turn/started duplicado: %v", err)
	}
	if _, err := machine.AcceptJSONL([]byte(`{"method":"turn/started","params":{"threadId":"thread-buffered","turn":{"id":"turn-other"}}}`)); ErrorCode(err) != CodeTurnMismatch {
		t.Fatalf("turn/started discordante: %v", err)
	}
	if _, err := machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, turnRequest)) + `,"result":{"turn":{"id":"turn-buffered","status":"inProgress","items":[]}}}`)); err != nil {
		t.Fatalf("respuesta turn reconciliada: %v", err)
	}
}

func TestMachineRejectsStartedResponseMismatchAndNonRunningTurnStatus(t *testing.T) {
	machine := mustMachine(t)
	initialize, _ := machine.Start()
	transition, _ := machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, initialize)) + `,"result":{"userAgent":"codex","codexHome":"/sealed","platformFamily":"unix","platformOs":"linux"}}`))
	threadRequest := transition.Outbound[1]
	_, _ = machine.AcceptJSONL([]byte(`{"method":"thread/started","params":{"thread":{"id":"thread-observed"}}}`))
	if _, err := machine.AcceptJSONL([]byte(validThreadResponse(frameID(t, threadRequest), "thread-response", validPacket().Model))); ErrorCode(err) != CodeThreadMismatch {
		t.Fatalf("thread response mismatch: %v", err)
	}

	machine = mustMachine(t)
	initialize, _ = machine.Start()
	transition, _ = machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, initialize)) + `,"result":{"userAgent":"codex","codexHome":"/sealed","platformFamily":"unix","platformOs":"linux"}}`))
	threadRequest = transition.Outbound[1]
	transition, _ = machine.AcceptJSONL([]byte(validThreadResponse(frameID(t, threadRequest), "thread-1", validPacket().Model)))
	turnRequest := transition.Outbound[0]
	if _, err := machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, turnRequest)) + `,"result":{"turn":{"id":"turn-1","status":"completed","items":[]}}}`)); ErrorCode(err) != CodeFrameMalformed {
		t.Fatalf("turn response status no inProgress: %v", err)
	}

	machine = mustMachine(t)
	initialize, _ = machine.Start()
	transition, _ = machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, initialize)) + `,"result":{"userAgent":"codex","codexHome":"/sealed","platformFamily":"unix","platformOs":"linux"}}`))
	threadRequest = transition.Outbound[1]
	transition, _ = machine.AcceptJSONL([]byte(validThreadResponse(frameID(t, threadRequest), "thread-1", validPacket().Model)))
	turnRequest = transition.Outbound[0]
	_, _ = machine.AcceptJSONL([]byte(`{"method":"turn/started","params":{"threadId":"thread-1","turn":{"id":"turn-observed","status":"inProgress","items":[]}}}`))
	if _, err := machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, turnRequest)) + `,"result":{"turn":{"id":"turn-response","status":"inProgress","items":[]}}}`)); ErrorCode(err) != CodeTurnMismatch {
		t.Fatalf("turn response mismatch: %v", err)
	}
}

func TestMachineAcceptsCodex0146GlobalAndMCPNotificationsAcrossActiveStates(t *testing.T) {
	machine := mustMachine(t)
	initialize, _ := machine.Start()
	activeNotifications := func(stage, suffix, status, mcpStatus, threadID string) {
		t.Helper()
		frames := []string{
			`{"method":"configWarning","params":{"summary":"warning-` + suffix + `","details":null,"path":null,"range":{"start":{"line":0,"column":1},"end":{"line":2,"column":3}}},"emittedAtMs":1785924308223}`,
			`{"method":"remoteControl/status/changed","params":{"environmentId":null,"installationId":"installation-` + suffix + `","serverName":"server","status":"` + status + `"},"emittedAtMs":1785924308224}`,
			`{"method":"mcpServer/startupStatus/updated","params":{"name":"server","status":"` + mcpStatus + `","error":null,"failureReason":null,"threadId":` + threadID + `},"emittedAtMs":1785924308225}`,
		}
		for _, frame := range frames {
			if _, err := machine.AcceptJSONL([]byte(frame)); err != nil {
				t.Fatalf("%s rechazó %s: %v", stage, frame, err)
			}
		}
	}
	activeNotifications("await_initialize", "init", "disabled", "starting", "null")
	transition, _ := machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, initialize)) + `,"result":{"userAgent":"codex","codexHome":"/sealed","platformFamily":"unix","platformOs":"linux"}}`))
	threadRequest := transition.Outbound[1]
	activeNotifications("await_thread", "thread", "connecting", "ready", "null")
	transition, _ = machine.AcceptJSONL([]byte(validThreadResponse(frameID(t, threadRequest), "thread-1", validPacket().Model)))
	turnRequest := transition.Outbound[0]
	activeNotifications("await_turn", "turn", "connected", "failed", `"thread-1"`)
	_, _ = machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, turnRequest)) + `,"result":{"turn":{"id":"turn-1","status":"inProgress","items":[]}}}`))
	activeNotifications("running", "run", "errored", "cancelled", `"thread-1"`)
	if _, err := machine.AcceptJSONL([]byte(`{"method":"mcpServer/startupStatus/updated","params":{"name":"server","status":"ready","threadId":"thread-other"}}`)); ErrorCode(err) != CodeThreadMismatch {
		t.Fatalf("MCP threadId discordante: %v", err)
	}
}

func TestMachineAcceptsEmittedAtMSOnlyOnNotifications(t *testing.T) {
	machine := mustMachine(t)
	initialize, _ := machine.Start()
	initializeID := quote(frameID(t, initialize))

	for _, frame := range []string{
		`{"id":` + initializeID + `,"result":{"userAgent":"codex","codexHome":"/sealed","platformFamily":"unix","platformOs":"linux"},"emittedAtMs":1}`,
		`{"id":"server-request","method":"approval","params":{},"emittedAtMs":1}`,
		`{"method":"configWarning","params":{"summary":"x"},"emittedAtMs":-1}`,
		`{"method":"configWarning","params":{"summary":"x"},"emittedAtMs":1.0}`,
		`{"method":"configWarning","params":{"summary":"x"},"emittedAtMs":1e3}`,
		`{"method":"configWarning","params":{"summary":"x"},"emittedAtMs":1.5}`,
		`{"method":"configWarning","params":{"summary":"x"},"emittedAtMs":"1"}`,
		`{"method":"configWarning","params":{"summary":"x"},"emittedAtMs":null}`,
		`{"method":"configWarning","params":{"summary":"x"},"emittedAtMs":18446744073709551616}`,
		`{"method":"configWarning","params":{"summary":"x"},"emittedAtMs":1,"extra":true}`,
	} {
		if _, err := machine.AcceptJSONL([]byte(frame)); ErrorCode(err) != CodeFrameMalformed {
			t.Fatalf("emittedAtMs inválido aceptado: %s err=%v", frame, err)
		}
	}
	for _, frame := range []string{
		`{"method":"configWarning","params":{"summary":"zero"},"emittedAtMs":0}`,
		`{"method":"configWarning","params":{"summary":"max"},"emittedAtMs":18446744073709551615}`,
	} {
		if _, err := machine.AcceptJSONL([]byte(frame)); err != nil {
			t.Fatalf("emittedAtMs uint64 rechazado: %s err=%v", frame, err)
		}
	}

	response := `{"id":` + initializeID + `,"result":{"userAgent":"codex","codexHome":"/sealed","platformFamily":"unix","platformOs":"linux"}}`
	if _, err := machine.AcceptJSONL([]byte(response)); err != nil {
		t.Fatalf("respuesta válida consumida por rechazo previo: %v", err)
	}
}

func TestMachineHandlesCodex0146RetryableAndTerminalErrors(t *testing.T) {
	machine := mustMachine(t)
	threadID, turnID := advanceRunning(t, machine)
	retry := `{"method":"error","params":{"error":{"message":"Reconnecting... 2/5","codexErrorInfo":{"responseStreamDisconnected":{"httpStatusCode":401}},"additionalDetails":"secret remote detail"},"willRetry":true,"threadId":"` + threadID + `","turnId":"` + turnID + `"},"emittedAtMs":1785924362451}`
	transition, err := machine.AcceptJSONL([]byte(retry))
	if err != nil || transition.State != StateRunning || transition.Result != nil || len(transition.Outbound) != 0 || machine.State() != StateRunning {
		t.Fatalf("retry alteró ejecución: transition=%#v err=%v state=%s", transition, err, machine.State())
	}
	transition, err = machine.AcceptJSONL([]byte(retry))
	if err != nil || transition.State != StateRunning || machine.State() != StateRunning {
		t.Fatalf("retry idéntico tratado como replay: transition=%#v err=%v", transition, err)
	}

	terminal := `{"method":"error","params":{"error":{"message":"remote secret","codexErrorInfo":"unauthorized","additionalDetails":null},"willRetry":false,"threadId":"` + threadID + `","turnId":"` + turnID + `"},"emittedAtMs":1785924362452}`
	_, err = machine.AcceptJSONL([]byte(terminal))
	if ErrorCode(err) != CodeRemote || strings.Contains(err.Error(), "secret") || machine.State() != StateFailed {
		t.Fatalf("error terminal=%v state=%s", err, machine.State())
	}
}

func TestMachineRejectsMalformedOrMismatchedCodex0146Errors(t *testing.T) {
	for _, test := range []struct {
		name   string
		params string
		code   Code
	}{
		{"missing error", `{"willRetry":true,"threadId":"thread-1","turnId":"turn-1"}`, CodeFrameMalformed},
		{"null error", `{"error":null,"willRetry":true,"threadId":"thread-1","turnId":"turn-1"}`, CodeFrameMalformed},
		{"error without message", `{"error":{"additionalDetails":null},"willRetry":true,"threadId":"thread-1","turnId":"turn-1"}`, CodeFrameMalformed},
		{"error unknown field", `{"error":{"message":"x","unknown":true},"willRetry":true,"threadId":"thread-1","turnId":"turn-1"}`, CodeFrameMalformed},
		{"missing retry", `{"error":{"message":"x"},"threadId":"thread-1","turnId":"turn-1"}`, CodeFrameMalformed},
		{"retry wrong type", `{"error":{"message":"x"},"willRetry":"true","threadId":"thread-1","turnId":"turn-1"}`, CodeFrameMalformed},
		{"empty thread", `{"error":{"message":"x"},"willRetry":true,"threadId":"","turnId":"turn-1"}`, CodeFrameMalformed},
		{"extra param", `{"error":{"message":"x"},"willRetry":true,"threadId":"thread-1","turnId":"turn-1","extra":true}`, CodeFrameMalformed},
		{"thread mismatch", `{"error":{"message":"x"},"willRetry":true,"threadId":"other","turnId":"turn-1"}`, CodeThreadMismatch},
		{"turn mismatch", `{"error":{"message":"x"},"willRetry":true,"threadId":"thread-1","turnId":"other"}`, CodeTurnMismatch},
	} {
		t.Run(test.name, func(t *testing.T) {
			machine := mustMachine(t)
			advanceRunning(t, machine)
			frame := `{"method":"error","params":` + test.params + `,"emittedAtMs":1}`
			if _, err := machine.AcceptJSONL([]byte(frame)); ErrorCode(err) != test.code {
				t.Fatalf("code=%q want=%q err=%v", ErrorCode(err), test.code, err)
			}
			if machine.State() != StateRunning {
				t.Fatalf("rechazo mutó state=%s", machine.State())
			}
		})
	}

	machine := mustMachine(t)
	_, _ = machine.Start()
	frame := `{"method":"error","params":{"error":{"message":"x"},"willRetry":false,"threadId":"thread-1","turnId":"turn-1"},"emittedAtMs":1}`
	if _, err := machine.AcceptJSONL([]byte(frame)); ErrorCode(err) != CodeSequence || machine.State() != StateAwaitInitialize {
		t.Fatalf("error fuera de turno: err=%v state=%s", err, machine.State())
	}
}

func TestMachineRejectsMalformedCodex0146GlobalAndMCPNotifications(t *testing.T) {
	machine := mustMachine(t)
	_, _ = machine.Start()
	frames := []string{
		`{"method":"configWarning","params":{"details":"missing summary"}}`,
		`{"method":"configWarning","params":{"summary":"x","unknown":true}}`,
		`{"method":"configWarning","params":{"summary":"x","range":{"start":{"line":0},"end":{"line":0,"column":0}}}}`,
		`{"method":"remoteControl/status/changed","params":{"installationId":"id","serverName":"server","status":"unknown"}}`,
		`{"method":"remoteControl/status/changed","params":{"serverName":"server","status":"connected"}}`,
		`{"method":"mcpServer/startupStatus/updated","params":{"name":"server","status":"unknown"}}`,
		`{"method":"mcpServer/startupStatus/updated","params":{"name":"server","status":"failed","failureReason":"other"}}`,
		`{"method":"mcpServer/startupStatus/updated","params":{"name":"server","status":"ready","threadId":"bad\nref"}}`,
	}
	for _, frame := range frames {
		if _, err := machine.AcceptJSONL([]byte(frame)); ErrorCode(err) != CodeFrameMalformed {
			t.Fatalf("frame inválido aceptado: %s err=%v", frame, err)
		}
	}
}

func TestMachineUsesOnlyLastFinalAnswerAndNeverDelta(t *testing.T) {
	machine := mustMachine(t)
	threadID, turnID := advanceRunning(t, machine)
	delta := `{"method":"item/agentMessage/delta","params":{"threadId":"` + threadID + `","turnId":"` + turnID + `","itemId":"delta","delta":"{\"artifact\":\"DELTA\"}"}}`
	if transition, err := machine.AcceptJSONL([]byte(delta)); err != nil || transition.Result != nil {
		t.Fatalf("delta produjo resultado: %#v %v", transition, err)
	}
	items := []string{
		`{"id":"comment","type":"agentMessage","phase":"commentary","text":"{\"artifact\":\"COMMENT\"}"}`,
		`{"id":"compat","type":"agentMessage","text":"{\"artifact\":\"COMPAT\"}"}`,
		`{"id":"final-1","type":"agentMessage","phase":"final_answer","text":"{\"artifact\":\"FIRST\"}"}`,
		`{"id":"final-2","type":"agentMessage","phase":"final_answer","text":"{\"artifact\":\"  exact\\nbytes  \"}"}`,
		`{"id":"compat-late","type":"agentMessage","text":"{\"artifact\":\"LATE-COMPAT\"}"}`,
	}
	for _, item := range items {
		frame := `{"method":"item/completed","params":{"threadId":"` + threadID + `","turnId":"` + turnID + `","completedAtMs":1,"item":` + item + `}}`
		if transition, err := machine.AcceptJSONL([]byte(frame)); err != nil || transition.Result != nil {
			t.Fatalf("item: %#v %v", transition, err)
		}
	}
	completed := `{"method":"turn/completed","params":{"threadId":"` + threadID + `","turn":{"id":"` + turnID + `","status":"completed","items":[],"itemsView":"summary"}}}`
	transition, err := machine.AcceptJSONL([]byte(completed))
	if err != nil || transition.State != StateCompleted || transition.Result == nil ||
		transition.Result.Artifact != "  exact\nbytes  " {
		t.Fatalf("resultado=%#v err=%v", transition, err)
	}
	if _, err := machine.AcceptJSONL([]byte(completed)); ErrorCode(err) != CodeDuplicateFrame {
		t.Fatalf("terminal duplicado: %v", err)
	}
}

func TestMachineCompatibilityPhaseAbsentAndTerminalItems(t *testing.T) {
	machine := mustMachine(t)
	threadID, turnID := advanceRunning(t, machine)
	completed := `{"method":"turn/completed","params":{"threadId":"` + threadID + `","turn":{"id":"` + turnID + `","status":"completed","items":[` +
		`{"id":"a","type":"agentMessage","phase":"commentary","text":"{\"artifact\":\"NO\"}"},` +
		`{"id":"b","type":"agentMessage","phase":null,"text":"{\"artifact\":\"YES\"}"}` +
		`]}}}`
	transition, err := machine.AcceptJSONL([]byte(completed))
	if err != nil || transition.Result == nil || transition.Result.Artifact != "YES" {
		t.Fatalf("compat=%#v err=%v", transition, err)
	}
}

func TestMachineTurnCompletedItemsViewControlsReconstruction(t *testing.T) {
	for _, itemsView := range []string{"summary", "notLoaded"} {
		t.Run(itemsView+" preserves completed items", func(t *testing.T) {
			machine := mustMachine(t)
			threadID, turnID := advanceRunning(t, machine)
			item := `{"method":"item/completed","params":{"threadId":"` + threadID + `","turnId":"` + turnID + `","item":{"id":"final","type":"agentMessage","phase":"final_answer","text":"{\"artifact\":\"accumulated\"}"}}}`
			if _, err := machine.AcceptJSONL([]byte(item)); err != nil {
				t.Fatal(err)
			}
			completed := `{"method":"turn/completed","params":{"threadId":"` + threadID + `","turn":{"id":"` + turnID + `","status":"completed","items":[],"itemsView":"` + itemsView + `"}}}`
			transition, err := machine.AcceptJSONL([]byte(completed))
			if err != nil || transition.Result == nil || transition.Result.Artifact != "accumulated" {
				t.Fatalf("itemsView=%s transition=%#v err=%v", itemsView, transition, err)
			}
		})
	}

	for _, itemsView := range []string{"", "full"} {
		name := itemsView
		if name == "" {
			name = "default"
		}
		t.Run(name+" rebuilds even when empty", func(t *testing.T) {
			machine := mustMachine(t)
			threadID, turnID := advanceRunning(t, machine)
			item := `{"method":"item/completed","params":{"threadId":"` + threadID + `","turnId":"` + turnID + `","item":{"id":"final","type":"agentMessage","phase":"final_answer","text":"{\"artifact\":\"must-not-survive\"}"}}}`
			_, _ = machine.AcceptJSONL([]byte(item))
			viewField := ""
			if itemsView != "" {
				viewField = `,"itemsView":"` + itemsView + `"`
			}
			completed := `{"method":"turn/completed","params":{"threadId":"` + threadID + `","turn":{"id":"` + turnID + `","status":"completed","items":[]` + viewField + `}}}`
			if _, err := machine.AcceptJSONL([]byte(completed)); ErrorCode(err) != CodeOutputMissing {
				t.Fatalf("itemsView=%s no reconstruyó vacío: %v", name, err)
			}
		})
	}

	t.Run("full rebuilds terminal payload", func(t *testing.T) {
		machine := mustMachine(t)
		threadID, turnID := advanceRunning(t, machine)
		completed := `{"method":"turn/completed","params":{"threadId":"` + threadID + `","turn":{"id":"` + turnID + `","status":"completed","items":[{"id":"final","type":"agentMessage","phase":"final_answer","text":"{\"artifact\":\"terminal\"}"}],"itemsView":"full"}}}`
		transition, err := machine.AcceptJSONL([]byte(completed))
		if err != nil || transition.Result == nil || transition.Result.Artifact != "terminal" {
			t.Fatalf("full transition=%#v err=%v", transition, err)
		}
	})
}

func TestMachineTurnCompletedRequiresItemsAndValidItemsView(t *testing.T) {
	for _, test := range []struct {
		name       string
		turnFields string
	}{
		{"missing items", `"status":"completed"`},
		{"null items", `"status":"completed","items":null`},
		{"null items view", `"status":"completed","items":[],"itemsView":null`},
		{"invalid items view", `"status":"completed","items":[],"itemsView":"partial"`},
		{"invalid terminal item", `"status":"completed","items":[{"id":"x"}],"itemsView":"summary"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			machine := mustMachine(t)
			threadID, turnID := advanceRunning(t, machine)
			completed := `{"method":"turn/completed","params":{"threadId":"` + threadID + `","turn":{"id":"` + turnID + `",` + test.turnFields + `}}}`
			if _, err := machine.AcceptJSONL([]byte(completed)); ErrorCode(err) != CodeFrameMalformed {
				t.Fatalf("code=%q err=%v", ErrorCode(err), err)
			}
		})
	}
}

func TestMachineRejectsMalformedDuplicateOutOfOrderAndMismatchedFrames(t *testing.T) {
	machine := mustMachine(t)
	initialize, _ := machine.Start()
	badFrames := []string{
		`{"jsonrpc":"2.0","id":"x","result":{}}`,
		`{"id":"x","result":{}} {}`,
		`{"id":"x","id":"y","result":{}}`,
		`{"method":"turn/completed","params":{}}`,
		`{"id":"server","method":"approval","params":{}}`,
	}
	want := []Code{CodeFrameMalformed, CodeFrameMalformed, CodeFrameMalformed, CodeSequence, CodeMethod}
	for index, frame := range badFrames {
		if _, err := machine.AcceptJSONL([]byte(frame)); ErrorCode(err) != want[index] {
			t.Fatalf("frame %d code=%q err=%v", index, ErrorCode(err), err)
		}
	}
	if _, err := machine.AcceptJSONL([]byte(`{"id":"unknown","result":{}}`)); ErrorCode(err) != CodeUnknownID {
		t.Fatalf("unknown=%v", err)
	}
	response := []byte(`{"id":` + quote(frameID(t, initialize)) + `,"result":{"userAgent":"x","codexHome":"/x","platformFamily":"unix","platformOs":"linux"}}`)
	if _, err := machine.AcceptJSONL(response); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.AcceptJSONL(response); ErrorCode(err) != CodeDuplicateFrame {
		t.Fatalf("duplicate=%v", err)
	}

	running := mustMachine(t)
	threadID, turnID := advanceRunning(t, running)
	mismatch := `{"method":"item/completed","params":{"threadId":"other","turnId":"` + turnID + `","item":{"id":"x","type":"agentMessage","text":"{}"}}}`
	if _, err := running.AcceptJSONL([]byte(mismatch)); ErrorCode(err) != CodeThreadMismatch {
		t.Fatalf("thread mismatch=%v", err)
	}
	mismatch = `{"method":"item/completed","params":{"threadId":"` + threadID + `","turnId":"other","item":{"id":"x","type":"agentMessage","text":"{}"}}}`
	if _, err := running.AcceptJSONL([]byte(mismatch)); ErrorCode(err) != CodeTurnMismatch {
		t.Fatalf("turn mismatch=%v", err)
	}
	item := `{"method":"item/completed","params":{"threadId":"` + threadID + `","turnId":"` + turnID + `","item":{"id":"same","type":"commandExecution"}}}`
	if _, err := running.AcceptJSONL([]byte(item)); err != nil {
		t.Fatal(err)
	}
	semanticallySame := ` { "method":"item/completed", "params":{"threadId":"` + threadID + `","turnId":"` + turnID + `","item":{"type":"commandExecution","id":"same"}}}`
	if _, err := running.AcceptJSONL([]byte(semanticallySame)); ErrorCode(err) != CodeDuplicateFrame {
		t.Fatalf("semantic duplicate=%v", err)
	}

	tooLarge := bytes.Repeat([]byte{'x'}, MaxPacketBytesV1+1)
	if _, err := running.AcceptJSONL(tooLarge); ErrorCode(err) != CodeFrameTooLarge {
		t.Fatalf("oversize=%v", err)
	}
}

func TestMachineTerminalErrorsAreTypedAndRedacted(t *testing.T) {
	for _, status := range []struct {
		value string
		code  Code
	}{{"failed", CodeTurnFailed}, {"interrupted", CodeTurnInterrupted}} {
		t.Run(status.value, func(t *testing.T) {
			machine := mustMachine(t)
			threadID, turnID := advanceRunning(t, machine)
			frame := `{"method":"turn/completed","params":{"threadId":"` + threadID + `","turn":{"id":"` + turnID + `","status":"` + status.value + `","error":{"message":"secret"},"items":[]}}}`
			_, err := machine.AcceptJSONL([]byte(frame))
			if ErrorCode(err) != status.code || strings.Contains(err.Error(), "secret") || machine.State() != StateFailed {
				t.Fatalf("err=%v state=%s", err, machine.State())
			}
		})
	}

	machine := mustMachine(t)
	initialize, _ := machine.Start()
	_, err := machine.AcceptJSONL([]byte(`{"id":` + quote(frameID(t, initialize)) + `,"error":{"code":-1,"message":"secret-token"}}`))
	if ErrorCode(err) != CodeRemote || strings.Contains(err.Error(), "secret") || machine.State() != StateFailed {
		t.Fatalf("remote=%v", err)
	}
}

func TestMachineRejectsMissingOrInvalidOutput(t *testing.T) {
	tests := []struct {
		name string
		item string
		code Code
	}{
		{"missing", ``, CodeOutputMissing},
		{"unknown field", `{"id":"a","type":"agentMessage","phase":"final_answer","text":"{\"artifact\":\"x\",\"extra\":1}"}`, CodeOutputInvalid},
		{"trailing", `{"id":"a","type":"agentMessage","phase":"final_answer","text":"{\"artifact\":\"x\"}{}"}`, CodeOutputInvalid},
		{"wrong type", `{"id":"a","type":"agentMessage","phase":"final_answer","text":"{\"artifact\":1}"}`, CodeOutputInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			machine := mustMachine(t)
			threadID, turnID := advanceRunning(t, machine)
			items := "[]"
			if test.item != "" {
				items = "[" + test.item + "]"
			}
			frame := `{"method":"turn/completed","params":{"threadId":"` + threadID + `","turn":{"id":"` + turnID + `","status":"completed","items":` + items + `}}}`
			if _, err := machine.AcceptJSONL([]byte(frame)); ErrorCode(err) != test.code {
				t.Fatalf("code=%q err=%v", ErrorCode(err), err)
			}
		})
	}

	packet := validPacket()
	packet.MaxOutputBytes = 2
	machine, _ := NewMachine(packet)
	threadID, turnID := advanceRunning(t, machine)
	frame := `{"method":"turn/completed","params":{"threadId":"` + threadID + `","turn":{"id":"` + turnID + `","status":"completed","items":[{"id":"a","type":"agentMessage","phase":"final_answer","text":"{\"artifact\":\"abc\"}"}]}}}`
	if _, err := machine.AcceptJSONL([]byte(frame)); ErrorCode(err) != CodeOutputInvalid {
		t.Fatalf("output bound=%v", err)
	}
}

func TestProtocolPackageHasNoRuntimeOrProviderAdapterImports(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	directory := filepath.Dir(file)
	for _, name := range []string{"doc.go", "packet.go", "codec.go", "machine.go"} {
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		lower := strings.ToLower(string(data))
		for _, forbidden := range []string{
			"internal/adapters/agent/codex", "os/exec", `"os"`, `"net"`, "firecracker", "credential", "codex_home",
		} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s contiene %q", name, forbidden)
			}
		}
	}
}
