package orquestacontext

import "strings"

const maxContextMaterializedEntryBytesV0 = 12000

func contextMaterializationIssueV0(
	code ContextMaterializationIssueCodeV0,
	field string,
	message string,
) ContextMaterializationIssueV0 {
	return ContextMaterializationIssueV0{Code: code, Field: field, Message: message}
}

func contextMaterializationReadableRefV0(
	bundle ContextBundleV0,
	entry ContextBundleEntryV0,
) (string, bool) {
	switch entry.Kind {
	case ContextEntryDocRefV0:
		return entry.SourceRef, true
	case ContextEntryReadRefV0:
		if strings.HasPrefix(entry.SourceRef, "modulos/") {
			return entry.SourceRef, true
		}
		return "modulos/" + bundle.TargetModule + "/" + entry.SourceRef, true
	default:
		return "", false
	}
}

func contextMaterializationEntryMaxBytesV0(entry ContextBundleEntryV0) int {
	if entry.MaxBytes <= 0 {
		return defaultContextBundleEntryBytesV0
	}
	if entry.MaxBytes > maxContextMaterializedEntryBytesV0 {
		return maxContextMaterializedEntryBytesV0
	}
	return entry.MaxBytes
}

func contextCommonRulesContentV0(sourceRef string) (string, bool) {
	if sourceRef != "orquesta-common-rules:v0" {
		return "", false
	}
	return strings.Join([]string{
		"Contexto pequeno por modulo.",
		"Hexagonal siempre: runtime, persistence, filesystem, LLM, cache, cola y deploy son conectores.",
		"i18n por defecto cuando haya UI, texto de app o documentacion generada.",
		"Funciones pequenas, write-set cerrado y pruebas declaradas.",
		"Si falta informacion de otro grupo, emitir CONSULTA AL DIRECTOR.",
	}, "\n"), true
}

func contextMaterializedRefOnlyV0(entry ContextBundleEntryV0) ContextMaterializedEntryV0 {
	return ContextMaterializedEntryV0{
		EntryRef:  entry.EntryRef,
		Layer:     entry.Layer,
		Kind:      entry.Kind,
		SourceRef: entry.SourceRef,
		Mode:      ContextMaterializationModeRefOnlyV0,
		Required:  entry.Required,
	}
}

func contextMaterializedContentV0(
	entry ContextBundleEntryV0,
	content ContextRefContentV0,
) ContextMaterializedEntryV0 {
	return ContextMaterializedEntryV0{
		EntryRef:  entry.EntryRef,
		Layer:     entry.Layer,
		Kind:      entry.Kind,
		SourceRef: entry.SourceRef,
		Mode:      ContextMaterializationModeContentV0,
		Content:   content.Content,
		Bytes:     content.Bytes,
		Truncated: content.Truncated,
		Required:  entry.Required,
	}
}

func contextMaterializedContentHasForbiddenDetailV0(content string) bool {
	lower := strings.ToLower(content)
	for _, fragment := range []string{
		"sk-", "bearer ", "access_token", "refresh_token", "secret=",
		"postgres://", "mysql://", "mongodb://", "/home/", "\\home\\",
		"c:\\users\\", "\\users\\",
	} {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}
