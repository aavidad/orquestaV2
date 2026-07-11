package orquestaruntimecodexappserver

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type serverCodexAppServerProbePortV0 interface {
	ProbeV0(context.Context) error
}

type serverCodexAppServerWebSocketProtocolV0 struct {
	SocketPath        string
	Timeout           time.Duration
	DiagnosticLogPath string
}

const (
	codexAppServerWebSocketMaxFrameBytesV0           = 16 * 1024 * 1024
	codexAppServerThreadReadMaxResponseFrameBytesV0  = 256 * 1024
	codexAppServerThreadReadFrameTooLargeIssueCodeV0 = "codex_app_server_thread_read_response_too_large"
)

func (protocol serverCodexAppServerWebSocketProtocolV0) ProbeV0(ctx context.Context) error {
	var response serverCodexAppServerThreadLoadedListResponseV0
	return protocol.callV0(ctx, "thread/loaded/list", map[string]interface{}{}, &response)
}

func (protocol serverCodexAppServerWebSocketProtocolV0) StartThreadV0(
	ctx context.Context,
	params serverCodexAppServerThreadStartParamsV0,
) (serverCodexAppServerThreadV0, error) {
	var response serverCodexAppServerThreadStartResponseV0
	err := protocol.callV0(ctx, "thread/start", params.toJSONV0(), &response)
	return response.Thread, err
}

func (protocol serverCodexAppServerWebSocketProtocolV0) UpdateThreadSettingsV0(
	ctx context.Context,
	params serverCodexAppServerThreadSettingsUpdateParamsV0,
) error {
	var response struct{}
	return protocol.callV0(ctx, "thread/settings/update", params.toJSONV0(), &response)
}

func (protocol serverCodexAppServerWebSocketProtocolV0) SetGoalV0(
	ctx context.Context,
	params serverCodexAppServerThreadGoalSetParamsV0,
) (serverCodexAppServerThreadGoalV0, error) {
	var response serverCodexAppServerThreadGoalSetResponseV0
	err := protocol.callV0(ctx, "thread/goal/set", params.toJSONV0(), &response)
	return response.Goal, err
}

func (protocol serverCodexAppServerWebSocketProtocolV0) StartTurnV0(
	ctx context.Context,
	params serverCodexAppServerTurnStartParamsV0,
) (serverCodexAppServerTurnV0, error) {
	var response serverCodexAppServerTurnStartResponseV0
	err := protocol.callV0(ctx, "turn/start", params.toJSONV0(), &response)
	return response.Turn, err
}

func (protocol serverCodexAppServerWebSocketProtocolV0) GetGoalV0(
	ctx context.Context,
	threadID string,
) (*serverCodexAppServerThreadGoalV0, error) {
	var response serverCodexAppServerThreadGoalGetResponseV0
	err := protocol.callV0(ctx, "thread/goal/get", map[string]interface{}{"threadId": strings.TrimSpace(threadID)}, &response)
	if err != nil {
		return nil, err
	}
	return response.Goal, nil
}

func (protocol serverCodexAppServerWebSocketProtocolV0) ReadThreadV0(
	ctx context.Context,
	threadID string,
	includeTurns bool,
) (serverCodexAppServerThreadReadV0, error) {
	var response serverCodexAppServerThreadReadResponseV0
	err := protocol.callWithMaxResponseFrameBytesV0(ctx, "thread/read", map[string]interface{}{
		"threadId":     strings.TrimSpace(threadID),
		"includeTurns": includeTurns,
	}, &response, codexAppServerThreadReadMaxResponseFrameBytesV0, codexAppServerThreadReadFrameTooLargeIssueCodeV0)
	return response.Thread, err
}

func (protocol serverCodexAppServerWebSocketProtocolV0) callV0(
	ctx context.Context,
	method string,
	params interface{},
	out interface{},
) error {
	return protocol.callWithMaxResponseFrameBytesV0(ctx, method, params, out, 0, "")
}

func (protocol serverCodexAppServerWebSocketProtocolV0) callWithMaxResponseFrameBytesV0(
	ctx context.Context,
	method string,
	params interface{},
	out interface{},
	maxResponseFrameBytes int,
	frameTooLargeIssueCode string,
) error {
	socketPath := strings.TrimSpace(protocol.SocketPath)
	if socketPath == "" {
		return codexAppServerCallErrorV0{Code: "codex_app_server_websocket_socket_missing", Err: errors.New("codex_app_server_websocket_socket_missing")}
	}
	timeout := protocol.Timeout
	if timeout <= 0 {
		timeout = time.Duration(defaultCodexGoalTimeoutMSV0) * time.Millisecond
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(callCtx, "unix", socketPath)
	if err != nil {
		return protocol.wrapWebSocketErrorV0(
			codexAppServerCallErrorV0{Code: codexAppServerIssueCodeFromCommandFailureV0(err.Error(), err), Err: err},
		)
	}
	defer conn.Close()
	if deadline, ok := callCtx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	reader := bufio.NewReader(conn)
	if err := codexAppServerWebSocketHandshakeV0(conn, reader); err != nil {
		return protocol.wrapWebSocketErrorV0(err)
	}
	if err := codexAppServerWebSocketWriteJSONV0(conn, codexAppServerWebSocketInitializeRequestV0()); err != nil {
		return protocol.wrapWebSocketErrorV0(err)
	}
	var initResponse map[string]interface{}
	if err := codexAppServerWebSocketReadResponseV0(reader, 1, &initResponse); err != nil {
		return protocol.wrapWebSocketErrorV0(err)
	}
	if err := codexAppServerWebSocketWriteJSONV0(conn, map[string]interface{}{
		"id":     2,
		"method": method,
		"params": params,
	}); err != nil {
		return protocol.wrapWebSocketErrorV0(err)
	}
	return protocol.wrapWebSocketErrorV0(codexAppServerWebSocketReadResponseWithMaxFrameBytesV0(
		reader,
		2,
		out,
		maxResponseFrameBytes,
		frameTooLargeIssueCode,
	))
}

func (protocol serverCodexAppServerWebSocketProtocolV0) wrapWebSocketErrorV0(err error) error {
	if err == nil {
		return nil
	}
	if code := codexAppServerIssueCodeFromLogFileV0(protocol.DiagnosticLogPath); code != "" {
		return codexAppServerCallErrorV0{Code: code, Err: err}
	}
	return err
}

func codexAppServerWebSocketInitializeRequestV0() map[string]interface{} {
	return map[string]interface{}{
		"id":     1,
		"method": "initialize",
		"params": map[string]interface{}{
			"clientInfo": map[string]interface{}{
				"name":    "orquesta-server",
				"title":   nil,
				"version": "0",
			},
			"capabilities": map[string]interface{}{
				"experimentalApi": true,
				// This app-server client capability is unrelated to the
				// independently observed required-test receipt. Enabling it
				// would still yield provider/runtime data, never closure authority.
				"requestAttestation":        false,
				"optOutNotificationMethods": []string{},
			},
		},
	}
}

func codexAppServerWebSocketHandshakeV0(conn net.Conn, reader *bufio.Reader) error {
	keyRaw := make([]byte, 16)
	if _, err := rand.Read(keyRaw); err != nil {
		return err
	}
	key := base64.StdEncoding.EncodeToString(keyRaw)
	request := "GET / HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	if _, err := io.WriteString(conn, request); err != nil {
		return err
	}
	response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodGet})
	if err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_websocket_handshake_failed", Err: err}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusSwitchingProtocols {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_websocket_handshake_rejected",
			Err:  fmt.Errorf("status=%d", response.StatusCode),
		}
	}
	wantAccept := codexAppServerWebSocketAcceptV0(key)
	if got := strings.TrimSpace(response.Header.Get("Sec-WebSocket-Accept")); got != wantAccept {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_websocket_handshake_rejected",
			Err:  fmt.Errorf("accept=%q", got),
		}
	}
	return nil
}

func codexAppServerWebSocketAcceptV0(key string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(key) + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func codexAppServerWebSocketWriteJSONV0(conn net.Conn, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	header := []byte{0x81}
	switch size := len(raw); {
	case size < 126:
		header = append(header, byte(0x80|size))
	case size <= 65535:
		header = append(header, 0x80|126, byte(size>>8), byte(size))
	default:
		header = append(header, 0x80|127)
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(size))
		header = append(header, length[:]...)
	}
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	header = append(header, mask...)
	masked := make([]byte, len(raw))
	for index, value := range raw {
		masked[index] = value ^ mask[index%4]
	}
	if _, err := conn.Write(append(header, masked...)); err != nil {
		return err
	}
	return nil
}

func codexAppServerWebSocketReadResponseV0(reader *bufio.Reader, responseID int, out interface{}) error {
	return codexAppServerWebSocketReadResponseWithMaxFrameBytesV0(reader, responseID, out, 0, "")
}

func codexAppServerWebSocketReadResponseWithMaxFrameBytesV0(
	reader *bufio.Reader,
	responseID int,
	out interface{},
	maxFrameBytes int,
	frameTooLargeIssueCode string,
) error {
	for {
		payload, opcode, err := codexAppServerWebSocketReadFrameWithMaxBytesV0(reader, maxFrameBytes, frameTooLargeIssueCode)
		if err != nil {
			return err
		}
		switch opcode {
		case 0x1, 0x2:
		case 0x8:
			return codexAppServerCallErrorV0{Code: "codex_app_server_websocket_closed", Err: errors.New("codex_app_server_websocket_closed")}
		default:
			continue
		}
		line := bytes.TrimSpace(payload)
		if len(line) == 0 {
			continue
		}
		var response serverCodexAppServerRPCResponseV0
		if err := json.Unmarshal(line, &response); err != nil {
			continue
		}
		if response.ID != responseID {
			continue
		}
		if response.Error != nil {
			return codexAppServerRPCErrorV0(response.Error.Code, response.Error.Message)
		}
		if out == nil {
			return nil
		}
		if len(response.Result) == 0 {
			return errors.New("codex_app_server_empty_result")
		}
		if err := json.Unmarshal(response.Result, out); err != nil {
			return err
		}
		sanitizeCodexAppServerRPCDecodedOutV0(out)
		return nil
	}
}

func codexAppServerWebSocketReadFrameV0(reader *bufio.Reader) ([]byte, byte, error) {
	return codexAppServerWebSocketReadFrameWithMaxBytesV0(reader, 0, "")
}

func codexAppServerWebSocketReadFrameWithMaxBytesV0(
	reader *bufio.Reader,
	maxFrameBytes int,
	frameTooLargeIssueCode string,
) ([]byte, byte, error) {
	header, err := reader.Peek(2)
	if err != nil {
		return nil, 0, err
	}
	_, _ = reader.Discard(2)
	opcode := header[0] & 0x0f
	masked := header[1]&0x80 != 0
	length := uint64(header[1] & 0x7f)
	switch length {
	case 126:
		extended, err := reader.Peek(2)
		if err != nil {
			return nil, 0, err
		}
		_, _ = reader.Discard(2)
		length = uint64(binary.BigEndian.Uint16(extended))
	case 127:
		extended, err := reader.Peek(8)
		if err != nil {
			return nil, 0, err
		}
		_, _ = reader.Discard(8)
		length = binary.BigEndian.Uint64(extended)
	}
	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(reader, mask); err != nil {
			return nil, 0, err
		}
	}
	limit := codexAppServerWebSocketMaxFrameBytesV0
	code := "codex_app_server_websocket_frame_too_large"
	if maxFrameBytes > 0 && maxFrameBytes < limit {
		limit = maxFrameBytes
		if trimmed := strings.TrimSpace(frameTooLargeIssueCode); trimmed != "" {
			code = trimmed
		}
	}
	if length > uint64(limit) {
		return nil, 0, codexAppServerCallErrorV0{Code: code, Err: fmt.Errorf("bytes=%d limit=%d", length, limit)}
	}
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, 0, err
	}
	if masked {
		for index := range payload {
			payload[index] ^= mask[index%4]
		}
	}
	return payload, opcode, nil
}
