package codexwork

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type State string

const (
	StateReady              State = "ready"
	StateAwaitInitialize    State = "await_initialize"
	StateAwaitThread        State = "await_thread"
	StateAwaitTurn          State = "await_turn"
	StateRunning            State = "running"
	StateCompleted          State = "completed"
	StateFailed             State = "failed"
	sealedWorkspaceCWD            = "/trabajo"
	sealedSandboxMode             = "danger-full-access"
	sealedSandboxPolicyType       = "dangerFullAccess"
	sealedEnvironmentID           = "local"
)

type WorkResultV1 struct {
	Artifact string `json:"artifact"`
}

type Transition struct {
	Outbound [][]byte
	Result   *WorkResultV1
	State    State
}

type requestStage uint8

const (
	requestNone requestStage = iota
	requestInitialize
	requestThread
	requestTurn
)

type Machine struct {
	packet           WorkPacketV1
	state            State
	digest           string
	pendingID        string
	pendingStage     requestStage
	handledIDs       map[string]struct{}
	seenFrames       map[[32]byte]struct{}
	seenEvents       map[string]struct{}
	threadID         string
	turnID           string
	observedThreadID string
	observedTurnID   string
	finalAnswer      *string
	unknownPhase     *string
}

func NewMachine(packet WorkPacketV1) (*Machine, error) {
	if err := packet.Validate(); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(packet)
	if err != nil {
		return nil, protocolError(CodePacketMalformed)
	}
	if len(canonical) > MaxPacketBytesV1 {
		return nil, protocolError(CodePacketTooLarge)
	}
	digest := sha256.Sum256(canonical)
	return &Machine{
		packet:     packet,
		state:      StateReady,
		digest:     hex.EncodeToString(digest[:]),
		handledIDs: make(map[string]struct{}),
		seenFrames: make(map[[32]byte]struct{}),
		seenEvents: make(map[string]struct{}),
	}, nil
}

func (machine *Machine) State() State {
	if machine == nil {
		return StateFailed
	}
	return machine.state
}

func (machine *Machine) Start() ([]byte, error) {
	if machine == nil || machine.state != StateReady {
		return nil, protocolError(CodeSequence)
	}
	id := machine.derivedID("initialize")
	frame, err := marshalLine(initializeRequest{
		ID: id, Method: "initialize",
		Params: initializeParams{
			ClientInfo: initializeClientInfo{
				Name: "orquesta-codex-work", Version: "1", Title: "sealed-worker",
			},
			Capabilities: initializeCapabilities{ExperimentalAPI: true},
		},
	})
	if err != nil {
		return nil, err
	}
	machine.pendingID, machine.pendingStage = id, requestInitialize
	machine.state = StateAwaitInitialize
	return frame, nil
}

func (machine *Machine) AcceptJSONL(frame []byte) (Transition, error) {
	if machine == nil || machine.state == StateReady {
		return Transition{}, protocolError(CodeSequence)
	}
	message, digest, err := decodeFrame(frame)
	if err != nil {
		return Transition{}, err
	}
	trackFrame := message.method != "error"
	if _, duplicate := machine.seenFrames[digest]; trackFrame && duplicate {
		return Transition{}, protocolError(CodeDuplicateFrame)
	}
	var transition Transition
	if message.method != "" {
		if message.id != "" {
			return Transition{}, protocolError(CodeMethod)
		}
		transition, err = machine.acceptNotification(message.method, message.params)
	} else {
		transition, err = machine.acceptResponse(message)
	}
	if trackFrame && (err == nil || ErrorCode(err) == CodeRemote || ErrorCode(err) == CodeTurnFailed ||
		ErrorCode(err) == CodeTurnInterrupted) {
		machine.seenFrames[digest] = struct{}{}
	}
	if err != nil {
		return Transition{}, err
	}
	transition.State = machine.state
	return transition, nil
}

func (machine *Machine) acceptResponse(message wireFrame) (Transition, error) {
	if message.id == "" {
		return Transition{}, protocolError(CodeFrameMalformed)
	}
	if _, duplicate := machine.handledIDs[message.id]; duplicate {
		return Transition{}, protocolError(CodeDuplicateFrame)
	}
	if message.id != machine.pendingID || machine.pendingStage == requestNone {
		return Transition{}, protocolError(CodeUnknownID)
	}
	stage := machine.pendingStage
	machine.handledIDs[message.id] = struct{}{}
	machine.pendingID, machine.pendingStage = "", requestNone
	if len(message.remoteError) != 0 {
		machine.state = StateFailed
		return Transition{}, protocolError(CodeRemote)
	}
	switch stage {
	case requestInitialize:
		return machine.acceptInitialize(message.result)
	case requestThread:
		return machine.acceptThread(message.result)
	case requestTurn:
		return machine.acceptTurn(message.result)
	default:
		return Transition{}, protocolError(CodeSequence)
	}
}

func (machine *Machine) acceptInitialize(raw json.RawMessage) (Transition, error) {
	if machine.state != StateAwaitInitialize {
		return Transition{}, protocolError(CodeSequence)
	}
	var response struct {
		UserAgent      *string `json:"userAgent"`
		CodexHome      *string `json:"codexHome"`
		PlatformFamily *string `json:"platformFamily"`
		PlatformOS     *string `json:"platformOs"`
	}
	if json.Unmarshal(raw, &response) != nil || response.UserAgent == nil || response.CodexHome == nil ||
		response.PlatformFamily == nil || response.PlatformOS == nil {
		machine.state = StateFailed
		return Transition{}, protocolError(CodeFrameMalformed)
	}
	initialized := []byte("{\"method\":\"initialized\"}\n")
	id := machine.derivedID("thread-start")
	thread, err := marshalLine(struct {
		ID     string `json:"id"`
		Method string `json:"method"`
		Params struct {
			Ephemeral             bool                    `json:"ephemeral"`
			Model                 string                  `json:"model"`
			ApprovalPolicy        string                  `json:"approvalPolicy"`
			CWD                   string                  `json:"cwd"`
			Sandbox               string                  `json:"sandbox"`
			Environments          []turnEnvironmentParams `json:"environments"`
			RuntimeWorkspaceRoots []string                `json:"runtimeWorkspaceRoots"`
		} `json:"params"`
	}{ID: id, Method: "thread/start", Params: struct {
		Ephemeral             bool                    `json:"ephemeral"`
		Model                 string                  `json:"model"`
		ApprovalPolicy        string                  `json:"approvalPolicy"`
		CWD                   string                  `json:"cwd"`
		Sandbox               string                  `json:"sandbox"`
		Environments          []turnEnvironmentParams `json:"environments"`
		RuntimeWorkspaceRoots []string                `json:"runtimeWorkspaceRoots"`
	}{Ephemeral: true, Model: machine.packet.Model, ApprovalPolicy: "never", CWD: sealedWorkspaceCWD,
		Sandbox: sealedSandboxMode, Environments: []turnEnvironmentParams{{EnvironmentID: sealedEnvironmentID, CWD: sealedWorkspaceCWD, RuntimeWorkspaceRoots: []string{sealedWorkspaceCWD}}},
		RuntimeWorkspaceRoots: []string{sealedWorkspaceCWD}}})
	if err != nil {
		return Transition{}, err
	}
	machine.pendingID, machine.pendingStage = id, requestThread
	machine.state = StateAwaitThread
	return Transition{Outbound: [][]byte{initialized, thread}}, nil
}

func (machine *Machine) acceptThread(raw json.RawMessage) (Transition, error) {
	if machine.state != StateAwaitThread {
		return Transition{}, protocolError(CodeSequence)
	}
	var response struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
		ApprovalPolicy        json.RawMessage `json:"approvalPolicy"`
		CWD                   string          `json:"cwd"`
		Model                 string          `json:"model"`
		RuntimeWorkspaceRoots []string        `json:"runtimeWorkspaceRoots"`
		Sandbox               struct {
			Type string `json:"type"`
		} `json:"sandbox"`
	}
	var approvalPolicy string
	if json.Unmarshal(raw, &response) != nil || !validOpaqueRef(response.Thread.ID) ||
		json.Unmarshal(response.ApprovalPolicy, &approvalPolicy) != nil || approvalPolicy != "never" ||
		response.CWD != sealedWorkspaceCWD || response.Model != machine.packet.Model ||
		len(response.RuntimeWorkspaceRoots) != 1 || response.RuntimeWorkspaceRoots[0] != sealedWorkspaceCWD ||
		response.Sandbox.Type != sealedSandboxPolicyType {
		machine.state = StateFailed
		return Transition{}, protocolError(CodeFrameMalformed)
	}
	if machine.observedThreadID != "" && machine.observedThreadID != response.Thread.ID {
		machine.state = StateFailed
		return Transition{}, protocolError(CodeThreadMismatch)
	}
	machine.threadID = response.Thread.ID
	id := machine.derivedID("turn-start")
	clientMessageID := machine.derivedID("client-message")
	turn, err := marshalLine(turnStartRequest{
		ID: id, Method: "turn/start",
		Params: turnStartParams{
			ThreadID:              response.Thread.ID,
			Input:                 []textInput{{Type: "text", Text: machine.packet.Prompt}},
			ClientUserMessageID:   clientMessageID,
			Model:                 machine.packet.Model,
			Effort:                machine.packet.Effort,
			Environments:          []turnEnvironmentParams{{EnvironmentID: sealedEnvironmentID, CWD: sealedWorkspaceCWD, RuntimeWorkspaceRoots: []string{sealedWorkspaceCWD}}},
			RuntimeWorkspaceRoots: []string{sealedWorkspaceCWD},
			OutputSchema: artifactOutputSchema{
				Type: "object", AdditionalProperties: false,
				Properties: map[string]artifactSchemaProperty{"artifact": {Type: "string"}},
				Required:   []string{"artifact"},
			},
		},
	})
	if err != nil {
		return Transition{}, err
	}
	machine.pendingID, machine.pendingStage = id, requestTurn
	machine.state = StateAwaitTurn
	return Transition{Outbound: [][]byte{turn}}, nil
}

func (machine *Machine) acceptTurn(raw json.RawMessage) (Transition, error) {
	if machine.state != StateAwaitTurn {
		return Transition{}, protocolError(CodeSequence)
	}
	var response struct {
		Turn struct {
			ID     string             `json:"id"`
			Status string             `json:"status"`
			Items  *[]json.RawMessage `json:"items"`
		} `json:"turn"`
	}
	if json.Unmarshal(raw, &response) != nil || !validOpaqueRef(response.Turn.ID) ||
		response.Turn.Status != "inProgress" || response.Turn.Items == nil {
		machine.state = StateFailed
		return Transition{}, protocolError(CodeFrameMalformed)
	}
	if machine.observedTurnID != "" && machine.observedTurnID != response.Turn.ID {
		machine.state = StateFailed
		return Transition{}, protocolError(CodeTurnMismatch)
	}
	machine.turnID = response.Turn.ID
	machine.state = StateRunning
	return Transition{}, nil
}

func (machine *Machine) acceptNotification(method string, raw json.RawMessage) (Transition, error) {
	if machine.state == StateCompleted || machine.state == StateFailed {
		return Transition{}, protocolError(CodeSequence)
	}
	switch method {
	case "thread/started":
		if machine.state != StateAwaitThread && machine.state != StateAwaitTurn && machine.state != StateRunning {
			return Transition{}, protocolError(CodeSequence)
		}
		threadID, err := notificationThreadObjectID(raw)
		if err != nil {
			return Transition{}, err
		}
		if !validOpaqueRef(threadID) {
			return Transition{}, protocolError(CodeFrameMalformed)
		}
		if machine.threadID != "" && threadID != machine.threadID {
			return Transition{}, protocolError(CodeThreadMismatch)
		}
		if machine.observedThreadID != "" && threadID != machine.observedThreadID {
			return Transition{}, protocolError(CodeThreadMismatch)
		}
		transition, err := machine.acceptUniqueEvent("thread/started:" + threadID)
		if err != nil {
			return Transition{}, err
		}
		if machine.state == StateAwaitThread {
			machine.observedThreadID = threadID
		}
		return transition, nil
	case "turn/started":
		if machine.state != StateAwaitThread && machine.state != StateAwaitTurn && machine.state != StateRunning {
			return Transition{}, protocolError(CodeSequence)
		}
		threadID, turnID, err := notificationTurnObjectIDs(raw)
		if err != nil {
			return Transition{}, err
		}
		if !validOpaqueRef(threadID) || !validOpaqueRef(turnID) {
			return Transition{}, protocolError(CodeFrameMalformed)
		}
		if threadID != machine.threadID {
			return Transition{}, protocolError(CodeThreadMismatch)
		}
		if machine.turnID != "" && turnID != machine.turnID {
			return Transition{}, protocolError(CodeTurnMismatch)
		}
		if machine.observedTurnID != "" && turnID != machine.observedTurnID {
			return Transition{}, protocolError(CodeTurnMismatch)
		}
		transition, err := machine.acceptUniqueEvent("turn/started:" + turnID)
		if err != nil {
			return Transition{}, err
		}
		if machine.state == StateAwaitTurn {
			machine.observedTurnID = turnID
		}
		return transition, nil
	case "configWarning":
		if !machine.acceptsGlobalNotification() {
			return Transition{}, protocolError(CodeSequence)
		}
		if err := validateConfigWarning(raw); err != nil {
			return Transition{}, err
		}
		return Transition{}, nil
	case "remoteControl/status/changed":
		if !machine.acceptsGlobalNotification() {
			return Transition{}, protocolError(CodeSequence)
		}
		if err := validateRemoteControlStatus(raw); err != nil {
			return Transition{}, err
		}
		return Transition{}, nil
	case "mcpServer/startupStatus/updated":
		if !machine.acceptsGlobalNotification() {
			return Transition{}, protocolError(CodeSequence)
		}
		threadID, err := validateMCPServerStartupStatus(raw)
		if err != nil {
			return Transition{}, err
		}
		if threadID != nil {
			expectedThreadID := machine.threadID
			if expectedThreadID == "" {
				expectedThreadID = machine.observedThreadID
			}
			if expectedThreadID != "" && *threadID != expectedThreadID {
				return Transition{}, protocolError(CodeThreadMismatch)
			}
		}
		return Transition{}, nil
	case "account/rateLimits/updated":
		if !machine.acceptsGlobalNotification() {
			return Transition{}, protocolError(CodeSequence)
		}
		if err := validateAccountRateLimitsUpdated(raw); err != nil {
			return Transition{}, err
		}
		return Transition{}, nil
	case "thread/environment/connected", "thread/environment/disconnected":
		if machine.state != StateAwaitTurn && machine.state != StateRunning {
			return Transition{}, protocolError(CodeSequence)
		}
		expectedThreadID := machine.threadID
		if expectedThreadID == "" {
			expectedThreadID = machine.observedThreadID
		}
		if expectedThreadID == "" {
			return Transition{}, protocolError(CodeSequence)
		}
		if err := validateEnvironmentConnection(raw, expectedThreadID); err != nil {
			return Transition{}, err
		}
		if _, err := machine.acceptUniqueEvent(method + ":" + expectedThreadID); err != nil {
			return Transition{}, err
		}
		if method == "thread/environment/disconnected" {
			machine.state = StateFailed
			return Transition{}, protocolError(CodeRemote)
		}
		return Transition{}, nil
	case "thread/settings/updated":
		if machine.state != StateRunning || machine.threadID == "" {
			return Transition{}, protocolError(CodeSequence)
		}
		if err := validateThreadSettingsUpdated(raw, machine.threadID, machine.packet.Model, machine.packet.Effort); err != nil {
			return Transition{}, err
		}
		return machine.acceptUniqueEvent("thread/settings/updated:" + machine.threadID)
	case "item/completed":
		return machine.acceptCompletedItem(raw)
	case "turn/completed":
		return machine.acceptCompletedTurn(raw)
	case "item/started", "item/agentMessage/delta", "item/commandExecution/outputDelta",
		"item/fileChange/outputDelta", "item/plan/delta", "item/reasoning/summaryTextDelta",
		"item/reasoning/summaryPartAdded", "item/reasoning/textDelta", "turn/diff/updated",
		"turn/plan/updated":
		if machine.state != StateRunning {
			return Transition{}, protocolError(CodeSequence)
		}
		threadID, turnID, err := notificationCorrelation(raw)
		if err != nil || threadID == "" || turnID == "" {
			if err == nil {
				err = protocolError(CodeFrameMalformed)
			}
			return Transition{}, err
		}
		if err := machine.validateCorrelation(threadID, turnID); err != nil {
			return Transition{}, err
		}
		return Transition{}, nil
	case "thread/status/changed", "thread/tokenUsage/updated", "thread/compacted",
		"hook/started", "hook/completed", "item/autoApprovalReview/started",
		"item/autoApprovalReview/completed", "item/commandExecution/terminalInteraction",
		"item/fileChange/patchUpdated", "item/mcpToolCall/progress", "serverRequest/resolved",
		"model/rerouted", "model/verification", "model/safetyBuffering/updated",
		"turn/moderationMetadata", "warning", "guardianWarning", "deprecationNotice":
		if machine.state != StateRunning {
			return Transition{}, protocolError(CodeSequence)
		}
		threadID, turnID, err := notificationCorrelation(raw)
		if err != nil {
			return Transition{}, err
		}
		if err := machine.validateCorrelation(threadID, turnID); err != nil {
			return Transition{}, err
		}
		return Transition{}, nil
	case "error":
		if machine.state != StateRunning {
			return Transition{}, protocolError(CodeSequence)
		}
		willRetry, threadID, turnID, err := decodeErrorNotification(raw)
		if err != nil {
			return Transition{}, err
		}
		if err := machine.validateCorrelation(threadID, turnID); err != nil {
			return Transition{}, err
		}
		if willRetry {
			return Transition{}, nil
		}
		machine.state = StateFailed
		return Transition{}, protocolError(CodeRemote)
	default:
		return Transition{}, protocolError(CodeMethod)
	}
}

func (machine *Machine) acceptCompletedItem(raw json.RawMessage) (Transition, error) {
	if machine.state != StateRunning {
		return Transition{}, protocolError(CodeSequence)
	}
	var params struct {
		ThreadID string          `json:"threadId"`
		TurnID   string          `json:"turnId"`
		Item     json.RawMessage `json:"item"`
	}
	if json.Unmarshal(raw, &params) != nil || len(params.Item) == 0 {
		return Transition{}, protocolError(CodeFrameMalformed)
	}
	if !validOpaqueRef(params.ThreadID) || !validOpaqueRef(params.TurnID) {
		return Transition{}, protocolError(CodeFrameMalformed)
	}
	if err := machine.validateCorrelation(params.ThreadID, params.TurnID); err != nil {
		return Transition{}, err
	}
	item, err := decodeThreadItem(params.Item)
	if err != nil {
		return Transition{}, err
	}
	if item.id == "" {
		return Transition{}, protocolError(CodeFrameMalformed)
	}
	eventKey := "item/completed:" + params.TurnID + ":" + item.id
	if _, duplicate := machine.seenEvents[eventKey]; duplicate {
		return Transition{}, protocolError(CodeDuplicateFrame)
	}
	machine.seenEvents[eventKey] = struct{}{}
	machine.rememberMessage(item)
	return Transition{}, nil
}

func (machine *Machine) acceptCompletedTurn(raw json.RawMessage) (Transition, error) {
	if machine.state != StateRunning {
		return Transition{}, protocolError(CodeSequence)
	}
	var params struct {
		ThreadID string `json:"threadId"`
		Turn     struct {
			ID        string             `json:"id"`
			Status    string             `json:"status"`
			Items     *[]json.RawMessage `json:"items"`
			ItemsView json.RawMessage    `json:"itemsView"`
		} `json:"turn"`
	}
	if json.Unmarshal(raw, &params) != nil || params.Turn.Items == nil {
		return Transition{}, protocolError(CodeFrameMalformed)
	}
	if !validOpaqueRef(params.ThreadID) || !validOpaqueRef(params.Turn.ID) {
		return Transition{}, protocolError(CodeFrameMalformed)
	}
	if err := machine.validateCorrelation(params.ThreadID, params.Turn.ID); err != nil {
		return Transition{}, err
	}
	if _, duplicate := machine.seenEvents["turn/completed:"+params.Turn.ID]; duplicate {
		return Transition{}, protocolError(CodeDuplicateFrame)
	}
	itemsView := "full"
	if len(params.Turn.ItemsView) != 0 {
		var decodedItemsView *string
		if json.Unmarshal(params.Turn.ItemsView, &decodedItemsView) != nil || decodedItemsView == nil {
			return Transition{}, protocolError(CodeFrameMalformed)
		}
		itemsView = *decodedItemsView
	}
	if itemsView != "full" && itemsView != "summary" && itemsView != "notLoaded" {
		return Transition{}, protocolError(CodeFrameMalformed)
	}
	decodedItems := make([]threadItem, 0, len(*params.Turn.Items))
	for _, rawItem := range *params.Turn.Items {
		item, err := decodeThreadItem(rawItem)
		if err != nil {
			return Transition{}, err
		}
		decodedItems = append(decodedItems, item)
	}
	switch params.Turn.Status {
	case "failed":
		machine.seenEvents["turn/completed:"+params.Turn.ID] = struct{}{}
		machine.state = StateFailed
		return Transition{}, protocolError(CodeTurnFailed)
	case "interrupted":
		machine.seenEvents["turn/completed:"+params.Turn.ID] = struct{}{}
		machine.state = StateFailed
		return Transition{}, protocolError(CodeTurnInterrupted)
	case "completed":
	default:
		return Transition{}, protocolError(CodeFrameMalformed)
	}
	finalAnswer, unknownPhase := machine.finalAnswer, machine.unknownPhase
	if itemsView == "full" {
		finalAnswer, unknownPhase = nil, nil
		for _, item := range decodedItems {
			rememberCandidate(item, &finalAnswer, &unknownPhase)
		}
	}
	text := finalAnswer
	if text == nil {
		text = unknownPhase
	}
	if text == nil {
		machine.seenEvents["turn/completed:"+params.Turn.ID] = struct{}{}
		machine.state = StateFailed
		return Transition{}, protocolError(CodeOutputMissing)
	}
	result, err := decodeWorkResult(*text, machine.packet.MaxOutputBytes)
	if err != nil {
		machine.seenEvents["turn/completed:"+params.Turn.ID] = struct{}{}
		machine.state = StateFailed
		return Transition{}, err
	}
	machine.seenEvents["turn/completed:"+params.Turn.ID] = struct{}{}
	machine.state = StateCompleted
	return Transition{Result: &result}, nil
}

func (machine *Machine) acceptsGlobalNotification() bool {
	switch machine.state {
	case StateAwaitInitialize, StateAwaitThread, StateAwaitTurn, StateRunning:
		return true
	default:
		return false
	}
}

func (machine *Machine) rememberMessage(item threadItem) {
	rememberCandidate(item, &machine.finalAnswer, &machine.unknownPhase)
}

func rememberCandidate(item threadItem, finalAnswer, unknownPhase **string) {
	if item.kind != "agentMessage" {
		return
	}
	text := item.text
	if item.phase == nil {
		*unknownPhase = &text
		return
	}
	if *item.phase == "final_answer" {
		*finalAnswer = &text
	}
}

func (machine *Machine) acceptUniqueEvent(key string) (Transition, error) {
	if _, duplicate := machine.seenEvents[key]; duplicate {
		return Transition{}, protocolError(CodeDuplicateFrame)
	}
	machine.seenEvents[key] = struct{}{}
	return Transition{}, nil
}

func (machine *Machine) validateCorrelation(threadID, turnID string) error {
	if threadID != "" && threadID != machine.threadID {
		return protocolError(CodeThreadMismatch)
	}
	if turnID != "" && turnID != machine.turnID {
		return protocolError(CodeTurnMismatch)
	}
	return nil
}

func (machine *Machine) derivedID(stage string) string {
	digest := sha256.Sum256([]byte("orquesta.codex-work.v1\x00" + stage + "\x00" + machine.digest))
	return "codexwork:" + stage + ":" + hex.EncodeToString(digest[:16])
}
