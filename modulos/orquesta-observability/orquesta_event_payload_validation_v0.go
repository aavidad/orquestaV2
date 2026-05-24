package orquestaobservability

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func validateCompactValueV0(field string, value any, depth int, add func(string, string)) {
	if depth > maxPayloadDepthV0 {
		add(ErrEventoDemasiadoExtensoV0, field)
		return
	}
	if value == nil {
		return
	}

	switch typed := value.(type) {
	case string:
		if utf8.RuneCountInString(typed) > maxPayloadStringRunesV0 {
			add(ErrEventoDemasiadoExtensoV0, field)
		}
		return
	case bool:
		return
	case json.Number:
		if _, err := strconv.ParseFloat(string(typed), 64); err != nil {
			add(ErrOrquestaEventInvalidoV0, field)
		}
		return
	case float32:
		if math.IsNaN(float64(typed)) || math.IsInf(float64(typed), 0) {
			add(ErrOrquestaEventInvalidoV0, field)
		}
		return
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			add(ErrOrquestaEventInvalidoV0, field)
		}
		return
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return
	}

	valueRef := reflect.ValueOf(value)
	if !valueRef.IsValid() {
		return
	}
	for valueRef.Kind() == reflect.Pointer || valueRef.Kind() == reflect.Interface {
		if valueRef.IsNil() {
			return
		}
		valueRef = valueRef.Elem()
	}

	switch valueRef.Kind() {
	case reflect.Map:
		validateCompactMapV0(field, valueRef, depth+1, add)
	case reflect.Slice, reflect.Array:
		validateCompactArrayV0(field, valueRef, depth+1, add)
	case reflect.String:
		if utf8.RuneCountInString(valueRef.String()) > maxPayloadStringRunesV0 {
			add(ErrEventoDemasiadoExtensoV0, field)
		}
	case reflect.Bool:
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	case reflect.Float32, reflect.Float64:
		floatValue := valueRef.Convert(reflect.TypeOf(float64(0))).Float()
		if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
			add(ErrOrquestaEventInvalidoV0, field)
		}
	default:
		add(ErrOrquestaEventInvalidoV0, field)
	}
}

func validateCompactMapV0(field string, valueRef reflect.Value, depth int, add func(string, string)) {
	if valueRef.IsNil() {
		return
	}
	if valueRef.Type().Key().Kind() != reflect.String {
		add(ErrOrquestaEventInvalidoV0, field)
		return
	}
	if valueRef.Len() > maxPayloadPropertiesV0 {
		add(ErrEventoDemasiadoExtensoV0, field)
	}

	keys := make([]string, 0, valueRef.Len())
	for _, keyRef := range valueRef.MapKeys() {
		keys = append(keys, keyRef.String())
	}
	sort.Strings(keys)

	for _, key := range keys {
		keyField := fieldV0(field, key)
		validatePayloadKeyV0(key, keyField, add)
		keyRef := reflect.ValueOf(key).Convert(valueRef.Type().Key())
		validateCompactValueV0(keyField, valueRef.MapIndex(keyRef).Interface(), depth+1, add)
	}
}

func validateCompactArrayV0(field string, valueRef reflect.Value, depth int, add func(string, string)) {
	if valueRef.Kind() == reflect.Slice && valueRef.IsNil() {
		return
	}
	if valueRef.Len() > maxPayloadArrayItemsV0 {
		add(ErrEventoDemasiadoExtensoV0, field)
	}
	for index := 0; index < valueRef.Len(); index++ {
		validateCompactValueV0(fmt.Sprintf("%s[%d]", field, index), valueRef.Index(index).Interface(), depth+1, add)
	}
}

func validatePayloadKeyV0(key string, field string, add func(string, string)) {
	if !validSizedPatternV0(key, minPayloadPropertyRunesV0, maxPayloadPropertyRunesV0, payloadKeyPatternV0) {
		add(ErrOrquestaEventInvalidoV0, field)
		return
	}
	if code := forbiddenPayloadKeyCodeV0(key); code != "" {
		add(code, field)
	}
}

func forbiddenPayloadKeyCodeV0(key string) string {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return ""
	}
	lower := strings.ToLower(key)
	for _, forbidden := range forbiddenPayloadKeysV0 {
		if strings.Contains(lower, forbidden) {
			if forbidden == "transcript" {
				return ErrTranscriptNoPermitidoV0
			}
			if containsAnyV0(lower, secretPayloadKeyPartsV0) {
				return ErrSecretoDetectadoV0
			}
			return ErrOrquestaEventInvalidoV0
		}
	}
	return ""
}
