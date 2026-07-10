package orquestadataingestionfile

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	ingestion "orquesta/modulos/orquesta-data-ingestion"
)

type columnStateV0 struct {
	name     string
	kind     string
	nullable bool
}

func (adapter *AdapterV0) profileFileV0(ctx context.Context, path string, kind ingestion.DataSourceKindV0) ([]ingestion.DataColumnV0, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("data_file_source_unavailable: %w", err)
	}
	defer file.Close()
	reader := io.LimitReader(file, adapter.maxBytes+1)
	switch kind {
	case ingestion.DataSourceKindCSVV0:
		return adapter.profileCSVV0(ctx, reader)
	case ingestion.DataSourceKindJSONV0:
		return adapter.profileJSONV0(ctx, reader)
	default:
		return nil, 0, fmt.Errorf("data_file_source_kind_unsupported")
	}
}

func (adapter *AdapterV0) profileCSVV0(ctx context.Context, input io.Reader) ([]ingestion.DataColumnV0, int64, error) {
	reader := csv.NewReader(input)
	header, err := reader.Read()
	if err == io.EOF {
		return nil, 0, fmt.Errorf("data_file_csv_header_required")
	}
	if err != nil {
		return nil, 0, fmt.Errorf("data_file_csv_invalid: %w", err)
	}
	states := make([]columnStateV0, len(header))
	seen := make(map[string]struct{}, len(header))
	for index, name := range header {
		name = strings.TrimSpace(name)
		if name == "" || containsV0(seen, name) {
			return nil, 0, fmt.Errorf("data_file_csv_header_invalid")
		}
		seen[name] = struct{}{}
		states[index] = columnStateV0{name: name}
	}
	var rows int64
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("data_file_csv_invalid: %w", err)
		}
		if len(record) != len(states) {
			return nil, 0, fmt.Errorf("data_file_csv_row_width_invalid")
		}
		rows++
		if rows > adapter.maxRows {
			return nil, 0, fmt.Errorf("data_file_row_limit_exceeded")
		}
		for index, value := range record {
			observeTextV0(&states[index], value)
		}
	}
	return columnsV0(states), rows, nil
}

func (adapter *AdapterV0) profileJSONV0(ctx context.Context, input io.Reader) ([]ingestion.DataColumnV0, int64, error) {
	decoder := json.NewDecoder(input)
	token, err := decoder.Token()
	if err != nil || token != json.Delim('[') {
		return nil, 0, fmt.Errorf("data_file_json_array_required")
	}
	states := []columnStateV0{}
	indices := map[string]int{}
	var rows int64
	for decoder.More() {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		var record map[string]json.RawMessage
		if err := decoder.Decode(&record); err != nil {
			return nil, 0, fmt.Errorf("data_file_json_invalid: %w", err)
		}
		if record == nil {
			return nil, 0, fmt.Errorf("data_file_json_object_rows_required")
		}
		rows++
		if rows > adapter.maxRows {
			return nil, 0, fmt.Errorf("data_file_row_limit_exceeded")
		}
		for name := range indices {
			if _, exists := record[name]; !exists {
				states[indices[name]].nullable = true
			}
		}
		names := make([]string, 0, len(record))
		for name := range record {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			raw := record[name]
			index, exists := indices[name]
			if !exists {
				index = len(states)
				indices[name] = index
				states = append(states, columnStateV0{name: name, nullable: rows > 1})
			}
			observeJSONV0(&states[index], raw)
		}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, 0, fmt.Errorf("data_file_json_invalid: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, 0, fmt.Errorf("data_file_json_trailing_data")
		}
		return nil, 0, fmt.Errorf("data_file_json_invalid: %w", err)
	}
	if len(states) == 0 {
		return nil, 0, fmt.Errorf("data_file_json_columns_required")
	}
	return columnsV0(states), rows, nil
}

func observeTextV0(state *columnStateV0, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		state.nullable = true
		return
	}
	mergeKindV0(state, classifyTextV0(value))
}

func observeJSONV0(state *columnStateV0, raw json.RawMessage) {
	value := strings.TrimSpace(string(raw))
	if value == "null" {
		state.nullable = true
		return
	}
	if value == "" {
		state.nullable = true
		return
	}
	switch value[0] {
	case '"':
		mergeKindV0(state, "string")
	case '{':
		mergeKindV0(state, "object")
	case '[':
		mergeKindV0(state, "array")
	case 't', 'f':
		mergeKindV0(state, "boolean")
	default:
		if strings.ContainsAny(value, ".eE") {
			mergeKindV0(state, "number")
		} else {
			mergeKindV0(state, "integer")
		}
	}
}

func classifyTextV0(value string) string {
	if value == "true" || value == "false" {
		return "boolean"
	}
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return "integer"
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return "number"
	}
	return "string"
}

func mergeKindV0(state *columnStateV0, observed string) {
	if state.kind == "" || state.kind == observed {
		state.kind = observed
		return
	}
	if (state.kind == "integer" && observed == "number") || (state.kind == "number" && observed == "integer") {
		state.kind = "number"
		return
	}
	state.kind = "string"
}

func columnsV0(states []columnStateV0) []ingestion.DataColumnV0 {
	columns := make([]ingestion.DataColumnV0, len(states))
	for index, state := range states {
		kind := state.kind
		if kind == "" {
			kind = "string"
		}
		columns[index] = ingestion.DataColumnV0{ColumnRef: fmt.Sprintf("column:%d", index+1), Name: state.name, DataType: kind, Nullable: state.nullable}
	}
	return columns
}

func containsV0(values map[string]struct{}, value string) bool {
	_, exists := values[value]
	return exists
}
