package orquestapersistence

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
)

var forbiddenPersistenceConnectorFieldsV0 = map[string]bool{
	"backend":       true,
	"connector":     true,
	"connector_ref": true,
	"database":      true,
	"db":            true,
	"dialect":       true,
	"dialecto":      true,
	"driver":        true,
	"dsn":           true,
	"engine":        true,
	"motor":         true,
}

var forbiddenPersistenceRuleFieldsV0 = map[string]bool{
	"query":   true,
	"queries": true,
	"sql":     true,
	"table":   true,
	"tables":  true,
	"tabla":   true,
	"tablas":  true,
}

var forbiddenConcretePersistenceTermsV0 = []string{
	"postgresql",
	"postgres",
	"sqlite",
	"mysql",
	"mariadb",
	"cockroach",
	"oracle",
	"mssql",
	"sqlserver",
	"mongodb",
	"mongo",
	"redis",
	"dynamodb",
	"cassandra",
	"neo4j",
	"firebird",
	"duckdb",
	"snowflake",
	"bigquery",
	"redshift",
	"db2",
}

func addRequiredV0(issues *[]PersistenceRepositoryValidationIssueV0, field, value string) {
	if trimV0(value) == "" {
		*issues = append(*issues, issueV0(ErrPayloadInvalidoV0, field, "campo requerido"))
	}
}

func prefixIssuesV0(prefix string, issues []PersistenceRepositoryValidationIssueV0) []PersistenceRepositoryValidationIssueV0 {
	out := make([]PersistenceRepositoryValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		if issue.Field != "" {
			issue.Field = prefix + issue.Field
		}
		out = append(out, issue)
	}
	return out
}

func indexedPrefixV0(prefix string, index int) string {
	return prefix + "." + strconv.Itoa(index) + "."
}

func validPersistenceConnectorRefV0(value string) bool {
	ref := trimV0(value)
	return ref != "" && strings.Contains(ref, "connector") && strings.Contains(ref, "ref") &&
		!strings.Contains(ref, "/") && !strings.Contains(ref, ":") &&
		!containsForbiddenPersistenceTermV0(ref)
}

func stringSetV0(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[trimV0(value)] = true
	}
	return set
}

func issueV0(code, field, message string) PersistenceRepositoryValidationIssueV0 {
	return PersistenceRepositoryValidationIssueV0{
		Code:    code,
		Field:   field,
		Message: message,
	}
}

func trimV0(value string) string {
	return strings.TrimSpace(value)
}

func rejectForbiddenPersistenceContractFieldsV0(data []byte, prefix string) []PersistenceRepositoryValidationIssueV0 {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	return rejectForbiddenPersistenceContractValueV0(raw, trimTrailingDotV0(prefix))
}

func rejectForbiddenPersistenceContractValueV0(value any, field string) []PersistenceRepositoryValidationIssueV0 {
	var issues []PersistenceRepositoryValidationIssueV0
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			childField := joinFieldV0(field, key)
			if issue, forbidden := forbiddenPersistenceFieldIssueV0(key, childField); forbidden {
				issues = append(issues, issue)
			}
			issues = append(issues, rejectForbiddenPersistenceContractValueV0(child, childField)...)
		}
	case []any:
		for index, child := range typed {
			issues = append(issues, rejectForbiddenPersistenceContractValueV0(child, joinFieldV0(field, strconv.Itoa(index)))...)
		}
	case string:
		if containsForbiddenPersistenceTermV0(typed) {
			issues = append(issues, issueV0(
				ErrPersistenceConnectorNoSoportadoV0,
				field,
				"detalle concreto de persistencia no pertenece al contrato",
			))
		}
	}
	return issues
}

func forbiddenPersistenceFieldIssueV0(field, path string) (PersistenceRepositoryValidationIssueV0, bool) {
	normalized := strings.ToLower(trimV0(field))
	switch {
	case forbiddenPersistenceConnectorFieldsV0[normalized]:
		return issueV0(
			ErrPersistenceConnectorNoSoportadoV0,
			path,
			"detalle concreto de persistencia no pertenece al contrato",
		), true
	case forbiddenPersistenceRuleFieldsV0[normalized]:
		return issueV0(
			ErrReglaNegocioEnAdaptadorV0,
			path,
			"detalle operativo de consulta no pertenece al contrato",
		), true
	default:
		return PersistenceRepositoryValidationIssueV0{}, false
	}
}

func validateNoForbiddenPersistenceTermsV0(value any, prefix string) []PersistenceRepositoryValidationIssueV0 {
	return validateNoForbiddenPersistenceTermsValueV0(reflect.ValueOf(value), trimTrailingDotV0(prefix))
}

func validateNoForbiddenPersistenceTermsValueV0(value reflect.Value, field string) []PersistenceRepositoryValidationIssueV0 {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	var issues []PersistenceRepositoryValidationIssueV0
	switch value.Kind() {
	case reflect.String:
		if containsForbiddenPersistenceTermV0(value.String()) {
			issues = append(issues, issueV0(
				ErrPersistenceConnectorNoSoportadoV0,
				field,
				"detalle concreto de persistencia no pertenece al contrato",
			))
		}
	case reflect.Slice, reflect.Array:
		for index := 0; index < value.Len(); index++ {
			issues = append(issues, validateNoForbiddenPersistenceTermsValueV0(value.Index(index), joinFieldV0(field, strconv.Itoa(index)))...)
		}
	case reflect.Struct:
		valueType := value.Type()
		for index := 0; index < value.NumField(); index++ {
			structField := valueType.Field(index)
			if !structField.IsExported() {
				continue
			}
			jsonName := jsonFieldNameV0(structField)
			if jsonName == "" {
				continue
			}
			issues = append(issues, validateNoForbiddenPersistenceTermsValueV0(value.Field(index), joinFieldV0(field, jsonName))...)
		}
	}
	return issues
}

func containsForbiddenPersistenceTermV0(value string) bool {
	normalized := strings.ToLower(trimV0(value))
	for _, term := range forbiddenConcretePersistenceTermsV0 {
		if containsSeparatedTermV0(normalized, term) {
			return true
		}
	}
	return false
}

func containsSeparatedTermV0(value, term string) bool {
	index := strings.Index(value, term)
	for index >= 0 {
		before := index == 0 || !isPersistenceTermRuneV0(rune(value[index-1]))
		afterIndex := index + len(term)
		after := afterIndex == len(value) || !isPersistenceTermRuneV0(rune(value[afterIndex]))
		if before && after {
			return true
		}
		nextFrom := index + 1
		next := strings.Index(value[nextFrom:], term)
		if next < 0 {
			return false
		}
		index = nextFrom + next
	}
	return false
}

func isPersistenceTermRuneV0(value rune) bool {
	return (value >= 'a' && value <= 'z') || (value >= '0' && value <= '9')
}

func jsonFieldNameV0(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	if name != "" {
		return name
	}
	return field.Name
}

func joinFieldV0(prefix, field string) string {
	if prefix == "" {
		return field
	}
	if field == "" {
		return prefix
	}
	return prefix + "." + field
}

func trimTrailingDotV0(value string) string {
	return strings.TrimSuffix(value, ".")
}

func HasPersistenceRepositoryIssueV0(issues []PersistenceRepositoryValidationIssueV0, code, field string) bool {
	for _, issue := range issues {
		if issue.Code == code && (field == "" || issue.Field == field) {
			return true
		}
	}
	return false
}
