package orquestastatefile

import (
	"encoding/base64"
	"sort"
	"strconv"
	"strings"

	orquestacore "orquesta/modulos/orquesta-core"
)

func pageFunctionContractRecordsV0(
	records []functionContractIndexRecordV0,
	limit int,
	offset int,
) orquestacore.ListFunctionContractsResultV0 {
	limit = boundedFunctionContractLimitV0(limit)
	if offset > len(records) {
		offset = len(records)
	}
	end := offset + limit
	if end > len(records) {
		end = len(records)
	}
	items := make([]orquestacore.FunctionContractSummaryV0, 0, end-offset)
	for _, record := range records[offset:end] {
		items = append(items, functionContractSummaryFromRecordV0(record))
	}
	result := orquestacore.ListFunctionContractsResultV0{
		Items:    items,
		Warnings: []string{"function_contract_payload_no_materializado"},
	}
	if end < len(records) {
		result.NextCursor = encodeFunctionContractCursorV0(end)
	}
	return result
}

func functionContractSummaryFromRecordV0(record functionContractIndexRecordV0) orquestacore.FunctionContractSummaryV0 {
	return orquestacore.FunctionContractSummaryV0{
		FunctionContractRef: record.Ref,
		Titulo:              record.Summary,
		SimboloObjetivo:     strings.Join(record.FunctionNames, ","),
		Estado:              orquestacore.FunctionContractEstadoEvidenciaInsuficienteV0,
		Version:             0,
	}
}

func filterFunctionContractRecordsV0(
	records []functionContractIndexRecordV0,
	filters orquestacore.FunctionContractFiltersV0,
) []functionContractIndexRecordV0 {
	state := strings.TrimSpace(filters.Estado)
	symbol := strings.TrimSpace(filters.SimboloObjetivo)
	out := records[:0]
	for _, record := range records {
		if state != "" && state != orquestacore.FunctionContractEstadoEvidenciaInsuficienteV0 {
			continue
		}
		if symbol != "" && !functionContractRecordHasSymbolV0(record, symbol) {
			continue
		}
		out = append(out, record)
	}
	return out
}

func unsupportedFunctionContractFilterV0(filters orquestacore.FunctionContractFiltersV0) string {
	if strings.TrimSpace(filters.Modulo) != "" {
		return "filtros.modulo"
	}
	if strings.TrimSpace(filters.ArchivoObjetivo) != "" {
		return "filtros.archivo_objetivo"
	}
	return ""
}

func functionContractRecordHasSymbolV0(record functionContractIndexRecordV0, symbol string) bool {
	for _, name := range record.FunctionNames {
		if strings.EqualFold(strings.TrimSpace(name), symbol) {
			return true
		}
	}
	return false
}

func sortedFunctionContractIndexRecordsV0(records map[string]functionContractIndexRecordV0) []functionContractIndexRecordV0 {
	out := make([]functionContractIndexRecordV0, 0, len(records))
	for _, record := range records {
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ref < out[j].Ref })
	return out
}

func boundedFunctionContractLimitV0(limit int) int {
	if limit <= 0 {
		return functionContractIndexDefaultLimitV0
	}
	if limit > functionContractIndexMaxLimitV0 {
		return functionContractIndexMaxLimitV0
	}
	return limit
}

func encodeFunctionContractCursorV0(offset int) string {
	raw := functionContractIndexCursorPrefixV0 + strconv.Itoa(offset)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeFunctionContractCursorV0(cursor string) (int, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	decoded := string(raw)
	if !strings.HasPrefix(decoded, functionContractIndexCursorPrefixV0) {
		return 0, strconv.ErrSyntax
	}
	value := strings.TrimPrefix(decoded, functionContractIndexCursorPrefixV0)
	offset, err := strconv.Atoi(value)
	if err != nil || offset < 0 {
		return 0, strconv.ErrSyntax
	}
	return offset, nil
}
