package orquestapersistence

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func DecodeGuardarProyectoBorradorMaterialV0(data []byte) (GuardarProyectoBorradorMaterialV0, []PersistenceRepositoryValidationIssueV0) {
	var material GuardarProyectoBorradorMaterialV0
	if issues := rejectForbiddenMaterialFieldsV0(data); len(issues) > 0 {
		return material, issues
	}
	if issues := decodeStrictV0(data, &material); len(issues) > 0 {
		return material, issues
	}
	return material, ValidateGuardarProyectoBorradorMaterialV0(material)
}

func DecodeGuardarProyectoBorradorRequestV0(data []byte) (GuardarProyectoBorradorRequestV0, []PersistenceRepositoryValidationIssueV0) {
	var request GuardarProyectoBorradorRequestV0
	if issues := rejectForbiddenRequestFieldsV0(data, ""); len(issues) > 0 {
		return request, issues
	}
	if issues := decodeStrictV0(data, &request); len(issues) > 0 {
		return request, issues
	}
	return request, ValidateGuardarProyectoBorradorRequestV0(request)
}

func rejectForbiddenMaterialFieldsV0(data []byte) []PersistenceRepositoryValidationIssueV0 {
	root, ok := rawObjectV0(data)
	if !ok {
		return nil
	}
	var issues []PersistenceRepositoryValidationIssueV0
	if _, exists := root["query"]; exists {
		issues = append(issues, issueV0(ErrReglaNegocioEnAdaptadorV0, "query", "query no pertenece al contrato material"))
	}
	if raw, exists := root["adapter_metadata"]; exists {
		issues = append(issues, rejectForbiddenPersistenceContractFieldsV0(raw, "adapter_metadata.")...)
	}
	if raw, exists := root["request"]; exists {
		issues = append(issues, rejectForbiddenRequestFieldsV0(raw, "request.")...)
	}
	return issues
}

func rejectForbiddenRequestFieldsV0(data []byte, prefix string) []PersistenceRepositoryValidationIssueV0 {
	request, ok := rawObjectV0(data)
	if !ok {
		return nil
	}
	var issues []PersistenceRepositoryValidationIssueV0
	for field, raw := range request {
		if field == "payload" {
			continue
		}
		childField := prefix + field
		if issue, forbidden := forbiddenPersistenceFieldIssueV0(field, childField); forbidden {
			issues = append(issues, issue)
		}
		issues = append(issues, rejectForbiddenPersistenceContractFieldsV0(raw, childField)...)
	}
	return issues
}

func decodeStrictV0(data []byte, value any) []PersistenceRepositoryValidationIssueV0 {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return []PersistenceRepositoryValidationIssueV0{issueV0(ErrPersistenceJSONInvalidoV0, "", err.Error())}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return []PersistenceRepositoryValidationIssueV0{issueV0(ErrPersistenceJSONInvalidoV0, "", "json contiene valores extra")}
	}
	return nil
}

func rawObjectV0(data []byte) (map[string]json.RawMessage, bool) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return nil, false
	}
	return object, true
}
