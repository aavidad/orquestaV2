package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
)

func buildOPESRegistryFinalPkgRequestV0(
	config opesRegistryFinalPkgConfigV0,
	template json.RawMessage,
	topicID string,
) (orquestaexternalworkrun.StartExternalWorkRunRequestV0, error) {
	replaced, err := replaceOPESRegistryFinalPkgTopicIDRawV0(template, config.TemplateTopicID, topicID)
	if err != nil {
		return orquestaexternalworkrun.StartExternalWorkRunRequestV0{}, err
	}
	var appChange orquestaappchange.AppChangeRequestV0
	if err := json.Unmarshal(replaced, &appChange); err != nil {
		return orquestaexternalworkrun.StartExternalWorkRunRequestV0{}, fmt.Errorf("template_decode_error")
	}
	runRef := opesRegistryFinalPkgRunRefV0(config, topicID)
	appChange.RunRef = runRef
	appChange.UserIntent = "Finalizar paquete local verificable del tema " + topicID + " de Informatica A1 reutilizando material existente, checkpoint y fuente; no crear checkpoint nuevo."
	appChange.AcceptanceCriteria = compactOPESRegistryFinalPkgStringsV0(append([]string{
		"paquete_final contiene manifest_cierre.json, index.html, tests.json, visuales_plan.md, rag/manifest.json, tutor/tutor_prompt.md, qa_final.md, html_final/tema_*.html y html_ampliado/tema_*.html",
		"manifest_cierre.json declara evidencias de html, rag, audio, tests, visual y qa",
		"rag/manifest.json referencia rag/corpus/chunks.jsonl y rag/corpus/summary.json canonicos",
		"audio/manifests contiene un JSON por cada html_final/tema_*.html y html_ampliado/tema_*.html con material_path a la pagina tematica; index.html, portadas y listados no cuentan",
		"tests.json es JSON valido y contiene preguntas publicables con enunciado, opciones y respuesta",
		"REGISTRO_TRABAJO_TEMAS_OPES.json usa courses[course_id].topics como diccionario por topic_id; usa claim/update/release de la herramienta y no lo trates como lista",
		"al liberar el registro, preferir status paquete_final_local_verificable si el paquete cumple; si usa alias equivalente, explicar en summary/pending",
		"no subir a produccion; paquete local verificable",
	}, appChange.AcceptanceCriteria...))
	return orquestaexternalworkrun.StartExternalWorkRunRequestV0{
		SchemaVersion:    orquestaexternalworkrun.StartExternalWorkRunRequestSchemaV0,
		RequestID:        "request-ref-opes-a1-t" + topicID + "-finalpkg-autonomous-registry",
		CorrelationID:    "corr-opes-a1-t" + topicID + "-finalpkg-autonomous-registry",
		RunRef:           runRef,
		ProjectRef:       "opes-a1-informatica",
		AppSpecRef:       "app-spec-external-work-opes-a1-informatica",
		QueueRef:         config.QueueRef,
		PriorityScore:    88,
		OccurredAt:       time.Now().UTC().Format(time.RFC3339),
		RequestedBy:      "orquesta-opes-registry-finalpkg",
		AppChangeRequest: appChange,
	}, nil
}

func replaceOPESRegistryFinalPkgTopicIDRawV0(raw json.RawMessage, oldTopicID string, newTopicID string) ([]byte, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("template_decode_error")
	}
	value = replaceOPESRegistryFinalPkgTopicIDValueV0(value, strings.TrimSpace(oldTopicID), strings.TrimSpace(newTopicID))
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("template_encode_error")
	}
	return data, nil
}

func replaceOPESRegistryFinalPkgTopicIDValueV0(value any, oldTopicID string, newTopicID string) any {
	switch typed := value.(type) {
	case string:
		return strings.ReplaceAll(typed, oldTopicID, newTopicID)
	case []any:
		for index := range typed {
			typed[index] = replaceOPESRegistryFinalPkgTopicIDValueV0(typed[index], oldTopicID, newTopicID)
		}
		return typed
	case map[string]any:
		for key, item := range typed {
			typed[key] = replaceOPESRegistryFinalPkgTopicIDValueV0(item, oldTopicID, newTopicID)
		}
		return typed
	default:
		return value
	}
}
