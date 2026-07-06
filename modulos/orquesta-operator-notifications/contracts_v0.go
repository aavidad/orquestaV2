package orquestaoperatornotifications

import (
	"context"
	"strings"
	"sync"
)

const (
	SchemaVersionV0 = "operator_notifications.v0"

	EventKindTaskV0 = "task"
	EventKindGoalV0 = "goal"
	EventKindRunV0  = "run"

	StatusCompleteV0 = "complete"
	StatusBlockedV0  = "blocked"
	StatusInvalidV0  = "invalid"
	StatusStoppedV0  = "stopped"
	StatusCanceledV0 = "canceled"
)

type TerminalEventV0 struct {
	SchemaVersion string   `json:"schema_version,omitempty"`
	EventRef      string   `json:"event_ref,omitempty"`
	Kind          string   `json:"kind"`
	SubjectRef    string   `json:"subject_ref"`
	RunRef        string   `json:"run_ref,omitempty"`
	GoalRef       string   `json:"goal_ref,omitempty"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

type MessageV0 struct {
	SchemaVersion string          `json:"schema_version"`
	DedupeKey     string          `json:"dedupe_key"`
	TargetRef     string          `json:"target_ref"`
	Text          string          `json:"text"`
	Event         TerminalEventV0 `json:"event"`
	EvidenceRefs  []string        `json:"evidence_refs,omitempty"`
}

type SendPortV0 interface {
	SendOperatorNotificationV0(context.Context, MessageV0) (string, error)
}

type DedupeStoreV0 interface {
	SeenOrRecordV0(string) bool
}

type ServiceV0 struct {
	TargetRef string
	Sender    SendPortV0
	Dedupe    DedupeStoreV0
}

func NotifyTerminalEventV0(ctx context.Context, service ServiceV0, event TerminalEventV0) (MessageV0, bool, error) {
	event = NormalizeTerminalEventV0(event)
	if service.Sender == nil || !TerminalStatusV0(event.Status) || event.SubjectRef == "" {
		return MessageV0{}, false, nil
	}
	key := DedupeKeyV0(event)
	if service.Dedupe != nil && service.Dedupe.SeenOrRecordV0(key) {
		return MessageV0{}, false, nil
	}
	message := MessageV0{
		SchemaVersion: SchemaVersionV0,
		DedupeKey:     key,
		TargetRef:     strings.TrimSpace(service.TargetRef),
		Text:          CompactTelegramTextV0(event),
		Event:         event,
		EvidenceRefs:  compactStringsV0(append(event.EvidenceRefs, "evidence-ref-operator-terminal-notification")),
	}
	receiptRef, err := service.Sender.SendOperatorNotificationV0(ctx, message)
	if err != nil {
		return message, false, err
	}
	if strings.TrimSpace(receiptRef) != "" {
		message.EvidenceRefs = compactStringsV0(append(message.EvidenceRefs, receiptRef))
	}
	return message, true, nil
}

func NormalizeTerminalEventV0(event TerminalEventV0) TerminalEventV0 {
	event.SchemaVersion = firstNonEmptyV0(event.SchemaVersion, SchemaVersionV0)
	event.Kind = strings.ToLower(strings.TrimSpace(event.Kind))
	event.SubjectRef = strings.TrimSpace(event.SubjectRef)
	event.RunRef = strings.TrimSpace(event.RunRef)
	event.GoalRef = strings.TrimSpace(event.GoalRef)
	event.Status = strings.ToLower(strings.TrimSpace(event.Status))
	event.Summary = compactTextV0(event.Summary, 180)
	event.EvidenceRefs = compactStringsV0(event.EvidenceRefs)
	return event
}

func TerminalStatusV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case StatusCompleteV0, StatusBlockedV0, StatusInvalidV0, StatusStoppedV0, StatusCanceledV0:
		return true
	default:
		return false
	}
}

func DedupeKeyV0(event TerminalEventV0) string {
	event = NormalizeTerminalEventV0(event)
	if event.EventRef != "" {
		return event.EventRef
	}
	return event.Kind + ":" + event.SubjectRef + ":" + event.Status
}

func CompactTelegramTextV0(event TerminalEventV0) string {
	event = NormalizeTerminalEventV0(event)
	parts := []string{"Orquesta", event.Kind, event.SubjectRef, event.Status}
	if event.Summary != "" {
		parts = append(parts, event.Summary)
	}
	return compactTextV0(strings.Join(parts, " | "), 360)
}

type MemoryDedupeStoreV0 struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewMemoryDedupeStoreV0() *MemoryDedupeStoreV0 {
	return &MemoryDedupeStoreV0{seen: map[string]struct{}{}}
}

func (store *MemoryDedupeStoreV0) SeenOrRecordV0(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.seen == nil {
		store.seen = map[string]struct{}{}
	}
	if _, ok := store.seen[key]; ok {
		return true
	}
	store.seen[key] = struct{}{}
	return false
}

func firstNonEmptyV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func compactStringsV0(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func compactTextV0(value string, maxRunes int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if maxRunes <= 0 || len([]rune(value)) <= maxRunes {
		return value
	}
	return strings.TrimSpace(string([]rune(value)[:maxRunes])) + "..."
}
