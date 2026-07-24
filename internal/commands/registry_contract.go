package commands

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strings"
)

func validateDefinition(definition Definition) error {
	if !strings.HasPrefix(definition.ID, "orquesta.") || definition.Version != "1" ||
		(definition.Kind != KindCommand && definition.Kind != KindQuery) ||
		(definition.Audience != AudiencePrincipal && definition.Audience != AudienceExecution) ||
		definition.ExecutionBound != (definition.Audience == AudienceExecution) ||
		definition.Permission == "" || definition.Handler == "" || ValidateSchemaContract(definition.InputSchema) != nil ||
		ValidateSchemaContract(definition.OutputSchema) != nil || definition.HTTP.Method != "POST" ||
		definition.HTTP.Path != "/api/v1/commands/"+definition.ID || definition.MCP.Tool != definition.ID ||
		definition.MCP.Annotations.ReadOnly != (definition.Kind == KindQuery) ||
		(definition.MCP.Annotations.ReadOnly && definition.MCP.Annotations.Destructive) ||
		!definition.MCP.Annotations.Idempotent ||
		definition.MCP.Annotations.OpenWorld ||
		len(definition.CLI.Path) == 0 || len(definition.ErrorCodes) != len(stableErrorCodes) {
		return errContract
	}
	wantReplay := ReplayApplicationReceipt
	if definition.Kind == KindQuery {
		wantReplay = ReplayReadReexecute
	}
	if definition.ReplayMode != wantReplay || expectedHandlerPermissions[definition.Handler] != definition.Permission {
		return errContract
	}
	for index, code := range definition.ErrorCodes {
		if code != stableErrorCodes[index] {
			return errContract
		}
	}
	return nil
}

func cloneDefinitions(source []Definition) []Definition {
	result := make([]Definition, len(source))
	for index, definition := range source {
		definition.InputSchema = cloneJSON(definition.InputSchema)
		definition.OutputSchema = cloneJSON(definition.OutputSchema)
		definition.ErrorCodes = append([]string(nil), definition.ErrorCodes...)
		definition.CLI.Path = append([]string(nil), definition.CLI.Path...)
		result[index] = definition
	}
	return result
}

func cloneJSON(value json.RawMessage) json.RawMessage { return append(json.RawMessage(nil), value...) }

func failureCode(value *Failure) string {
	if value == nil {
		return ""
	}
	return value.Code
}

func validOpaque(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}

func digestFields(fields ...string) string {
	digest := sha256.New()
	var size [8]byte
	for _, field := range fields {
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(field))
	}
	return hex.EncodeToString(digest.Sum(nil))
}
