package appserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strconv"
)

type ErrorCode string

func (code ErrorCode) Error() string { return string(code) }

const (
	ErrFrameTooLarge    ErrorCode = "appserver.frame_too_large"
	ErrMalformed        ErrorCode = "appserver.malformed"
	ErrInvalidID        ErrorCode = "appserver.invalid_id"
	ErrDuplicateID      ErrorCode = "appserver.duplicate_id"
	ErrUnknownID        ErrorCode = "appserver.unknown_id"
	ErrMethodNotAllowed ErrorCode = "appserver.method_not_allowed"
	ErrSequence         ErrorCode = "appserver.sequence"
	ErrInitialize       ErrorCode = "appserver.initialize"
	ErrRemote           ErrorCode = "appserver.remote"
)

type Kind uint8

const (
	KindRequest Kind = iota
	KindNotification
	KindSuccess
	KindFailure
)

type Message struct {
	Kind    Kind
	ID      any
	Method  string
	Payload json.RawMessage
}
type wireMessage struct {
	ID      json.RawMessage `json:"id"`
	Method  *string         `json:"method"`
	Params  json.RawMessage `json:"params"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
	JSONRPC json.RawMessage `json:"jsonrpc"`
}
type remoteError struct {
	Code    *int64  `json:"code"`
	Message *string `json:"message"`
}
type Decoder struct {
	reader   *bufio.Reader
	max      int
	terminal error
}

func NewDecoder(reader io.Reader, max int) (*Decoder, error) {
	if reader == nil || max < 1 {
		return nil, ErrMalformed
	}
	return &Decoder{reader: bufio.NewReader(reader), max: max}, nil
}

func (decoder *Decoder) Next() (Message, error) {
	if decoder.terminal != nil {
		return Message{}, decoder.terminal
	}
	var frame []byte
	for {
		part, err := decoder.reader.ReadSlice('\n')
		ended := len(part) > 0 && part[len(part)-1] == '\n'
		if ended {
			part = part[:len(part)-1]
		}
		if len(frame)+len(part) > decoder.max {
			decoder.terminal = ErrFrameTooLarge
			return Message{}, decoder.terminal
		}
		frame = append(frame, part...)
		if ended || err != bufio.ErrBufferFull {
			if err == io.EOF && len(frame) == 0 {
				return Message{}, io.EOF
			}
			if err != nil && err != io.EOF {
				return Message{}, ErrMalformed
			}
			return decode(frame)
		}
	}
}

func decode(frame []byte) (Message, error) {
	var wire wireMessage
	if len(bytes.TrimSpace(frame)) == 0 || json.Unmarshal(frame, &wire) != nil || len(wire.JSONRPC) != 0 {
		return Message{}, ErrMalformed
	}
	id, idErr := decodedID(wire.ID)
	if len(wire.ID) != 0 && idErr != nil {
		return Message{}, idErr
	}
	method, result, failure := wire.Method != nil, len(wire.Result) != 0, len(wire.Error) != 0
	switch {
	case method && len(wire.ID) != 0 && !result && !failure:
		return Message{Kind: KindRequest, ID: id, Method: *wire.Method, Payload: wire.Params}, nil
	case method && len(wire.ID) == 0 && !result && !failure:
		return Message{Kind: KindNotification, Method: *wire.Method, Payload: wire.Params}, nil
	case !method && len(wire.ID) != 0 && result != failure:
		if failure {
			var detail remoteError
			if json.Unmarshal(wire.Error, &detail) != nil || detail.Code == nil || detail.Message == nil {
				return Message{}, ErrMalformed
			}
			return Message{Kind: KindFailure, ID: id}, nil
		}
		return Message{Kind: KindSuccess, ID: id, Payload: wire.Result}, nil
	default:
		return Message{}, ErrMalformed
	}
}

func requestID(value any) (string, error) {
	switch id := value.(type) {
	case string:
		raw, _ := json.Marshal(id)
		return string(raw), nil
	case int64:
		return strconv.FormatInt(id, 10), nil
	}
	return "", ErrInvalidID
}

func decodedID(raw json.RawMessage) (any, error) {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text, nil
	}
	var number int64
	if json.Unmarshal(raw, &number) == nil {
		return number, nil
	}
	return nil, ErrInvalidID
}
