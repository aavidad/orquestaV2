package orquestacoreworkflow

import "strings"

func commandEffectRecordValidV0(record commandEffectRecordV0) bool {
	return strings.TrimSpace(record.EventType) != "" &&
		strings.TrimSpace(record.SubjectRef) != "" &&
		strings.TrimSpace(record.IdempotencyKey) != "" &&
		strings.TrimSpace(record.EventID) != "" &&
		strings.TrimSpace(record.PayloadHash) != ""
}

func commandEffectRecordKeyV0(record commandEffectRecordV0) string {
	return commandEffectKeyV0(record.EventType, record.SubjectRef)
}

func commandEffectKeyV0(eventType string, subjectRef string) string {
	eventType = strings.TrimSpace(eventType)
	subjectRef = strings.TrimSpace(subjectRef)
	if eventType == "" || subjectRef == "" {
		return ""
	}
	return eventType + "\n" + subjectRef
}
