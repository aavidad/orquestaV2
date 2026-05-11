package orquestaruntimecodex

import (
	"encoding/json"
	"fmt"
	"os"
)

const CodexAgentAckSchemaVersionV0 = "codex_agent_ack.v0"

type CodexAgentAckV0 struct {
	SchemaVersion string         `json:"schema_version"`
	RequestID     string         `json:"request_id"`
	CorrelationID string         `json:"correlation_id"`
	AckRef        string         `json:"ack_ref"`
	TargetModule  string         `json:"target_module"`
	TaskRef       string         `json:"task_ref"`
	Status        string         `json:"status"`
	Files         EvidenceListV0 `json:"files,omitempty"`
	Tests         EvidenceListV0 `json:"tests,omitempty"`
	Notes         EvidenceListV0 `json:"notes,omitempty"`
}

type EvidenceListV0 []string

func (list *EvidenceListV0) UnmarshalJSON(data []byte) error {
	var stringsOnly []string
	if err := json.Unmarshal(data, &stringsOnly); err == nil {
		*list = stringsOnly
		return nil
	}
	var objects []map[string]any
	if err := json.Unmarshal(data, &objects); err != nil {
		return err
	}
	values := make([]string, 0, len(objects))
	for _, item := range objects {
		values = append(values, evidenceObjectSummaryV0(item))
	}
	*list = values
	return nil
}

func evidenceObjectSummaryV0(item map[string]any) string {
	for _, key := range []string{"path", "command", "name", "ref"} {
		if value, ok := item[key].(string); ok && value != "" {
			return value
		}
	}
	raw, err := json.Marshal(item)
	if err != nil {
		return fmt.Sprint(item)
	}
	return string(raw)
}

func ReadCodexAgentAckFileV0(path string) (CodexAgentAckV0, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CodexAgentAckV0{}, err
	}
	var ack CodexAgentAckV0
	if err := json.Unmarshal(data, &ack); err != nil {
		return CodexAgentAckV0{}, err
	}
	return ack, nil
}

func (ack CodexAgentAckV0) ValidFor(requestID string) bool {
	return ack.SchemaVersion == CodexAgentAckSchemaVersionV0 &&
		ack.RequestID == requestID &&
		ack.CorrelationID != "" &&
		ack.AckRef != "" &&
		ack.TargetModule != "" &&
		ack.TaskRef != "" &&
		ack.Status != ""
}
