package appserver

import (
	"encoding/json"
)

const quotaReadMethod = "account/rateLimits/read"
const quotaUpdatedMethod = "account/rateLimits/updated"

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Title   string `json:"title,omitempty"`
}
type initializeResponse struct {
	UserAgent      *string `json:"userAgent"`
	CodexHome      *string `json:"codexHome"`
	PlatformFamily *string `json:"platformFamily"`
	PlatformOS     *string `json:"platformOs"`
}
type Session struct {
	pending map[string]string
	state   uint8
}

func NewQuotaSession() *Session {
	return &Session{pending: make(map[string]string)}
}

func (session *Session) Initialize(id any, info ClientInfo) ([]byte, error) {
	if session.state != 0 {
		return nil, ErrSequence
	}
	// La sesión de cuota no negocia capacidades experimentales ni de herramientas.
	params, err := json.Marshal(struct {
		ClientInfo   ClientInfo      `json:"clientInfo"`
		Capabilities json.RawMessage `json:"capabilities"`
	}{info, json.RawMessage("null")})
	if err != nil {
		return nil, ErrMalformed
	}
	frame, err := session.request(id, "initialize", params)
	if err == nil {
		session.state = 1
	}
	return frame, err
}

func (session *Session) Initialized() ([]byte, error) {
	if session.state != 2 {
		return nil, ErrSequence
	}
	session.state = 3
	return []byte("{\"method\":\"initialized\"}\n"), nil
}

func (session *Session) ReadQuota(id any) ([]byte, error) {
	if session.state != 3 {
		return nil, ErrSequence
	}
	return session.request(id, quotaReadMethod, json.RawMessage("null"))
}

func (session *Session) request(id any, method string, params json.RawMessage) ([]byte, error) {
	key, err := requestID(id)
	if err != nil {
		return nil, err
	}
	if _, exists := session.pending[key]; exists {
		return nil, ErrDuplicateID
	}
	frame, err := json.Marshal(struct {
		ID     any             `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}{id, method, params})
	if err != nil {
		return nil, ErrMalformed
	}
	session.pending[key] = method
	return append(frame, '\n'), nil
}

func (session *Session) Accept(message Message) (string, error) {
	if message.Kind == KindNotification {
		if session.state != 3 {
			return "", ErrSequence
		}
		if message.Method == quotaUpdatedMethod {
			return message.Method, nil
		}
		// Notifications unrelated to quota are advisory for this read-only
		// session. Ignore them without inspecting their payload; server requests
		// and uncorrelated responses remain rejected below.
		return "", nil
	}
	if message.Kind != KindSuccess && message.Kind != KindFailure {
		return "", ErrMethodNotAllowed
	}
	key, err := requestID(message.ID)
	if err != nil {
		return "", err
	}
	method, exists := session.pending[key]
	if !exists {
		return "", ErrUnknownID
	}
	delete(session.pending, key)
	if method == "initialize" {
		if session.state != 1 {
			return "", ErrSequence
		}
		if message.Kind == KindFailure {
			session.state = 4
			return method, ErrInitialize
		}
		var response initializeResponse
		if json.Unmarshal(message.Payload, &response) != nil ||
			response.UserAgent == nil || response.CodexHome == nil ||
			response.PlatformFamily == nil || response.PlatformOS == nil {
			session.state = 4
			return method, ErrInitialize
		}
		session.state = 2
		return method, nil
	}
	if session.state != 3 {
		return "", ErrSequence
	}
	if message.Kind == KindFailure {
		return method, ErrRemote
	}
	return method, nil
}
