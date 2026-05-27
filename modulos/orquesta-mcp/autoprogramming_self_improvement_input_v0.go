package orquestamcp

import (
	"encoding/json"
	"strings"
)

func (input *MCPAutoprogrammingSelfImprovementToolInputV0) UnmarshalJSON(raw []byte) error {
	type alias MCPAutoprogrammingSelfImprovementToolInputV0
	var envelope struct {
		alias
		OperatorAdvice json.RawMessage `json:"operator_advice"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	*input = MCPAutoprogrammingSelfImprovementToolInputV0(envelope.alias)
	if len(envelope.OperatorAdvice) == 0 || strings.TrimSpace(string(envelope.OperatorAdvice)) == "null" {
		return nil
	}
	var advice mcpAutoprogrammingOperatorAdviceListV0
	if err := json.Unmarshal(envelope.OperatorAdvice, &advice); err != nil {
		return err
	}
	input.OperatorAdvice = []MCPAutoprogrammingOperatorAdviceV0(advice)
	return nil
}
