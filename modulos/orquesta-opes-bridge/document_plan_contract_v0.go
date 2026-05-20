package orquestaopesbridge

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func withOPESGlobalEditorialPolicyFieldV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	if fieldHasNameV0(fields, "opes_global_editorial_policy_2026_05_18") {
		return fields
	}
	out := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields)+1)
	out = append(out, orquestadomainwork.DomainWorkFieldV0{
		Name:   "opes_global_editorial_policy_2026_05_18",
		Values: opesGlobalEditorialPolicyV0(),
	})
	out = append(out, fields...)
	return out
}

func withOPESHTMLPublicationPolicyFieldV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	if fieldHasNameV0(fields, "opes_html_publication_policy_2026_05_19") {
		return fields
	}
	out := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields)+1)
	out = append(out, orquestadomainwork.DomainWorkFieldV0{
		Name:   "opes_html_publication_policy_2026_05_19",
		Values: opesHTMLPublicationPolicyV0(),
	})
	out = append(out, fields...)
	return out
}

func withOPESHTMLTopicTemplateFieldV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	if fieldHasNameV0(fields, "opes_html_topic_template_v1") {
		return fields
	}
	out := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields)+1)
	out = append(out, orquestadomainwork.DomainWorkFieldV0{
		Name:   "opes_html_topic_template_v1",
		Values: opesHTMLTopicTemplateV1(),
	})
	out = append(out, fields...)
	return out
}

func OPESGlobalEditorialPolicyV0() []string {
	return append([]string(nil), opesGlobalEditorialPolicyV0()...)
}

func OPESHTMLPublicationPolicyV0() []string {
	return append([]string(nil), opesHTMLPublicationPolicyV0()...)
}

func OPESHTMLTopicTemplateV1() []string {
	return append([]string(nil), opesHTMLTopicTemplateV1()...)
}

func appendDocumentPlanContractFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	jobType string,
) []orquestadomainwork.DomainWorkFieldV0 {
	if !isDocumentPlanJobTypeV0(jobType) {
		return fields
	}
	if !fieldHasNameV0(fields, "expected_schema") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:  "expected_schema",
			Value: orquestadomainwork.DomainDocumentPlanSchemaV0,
		})
	}
	if !fieldHasNameV0(fields, "required_plan_parts") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "required_plan_parts",
			Values: documentPlanRequiredPartsV0(),
		})
	}
	if !fieldHasNameV0(fields, "allowed_document_plan_work_kinds") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "allowed_document_plan_work_kinds",
			Values: documentPlanAllowedWorkKindsV0(),
		})
	}
	if !fieldHasNameV0(fields, "minimum_quality_gates") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "minimum_quality_gates",
			Values: documentPlanQualityGatesV0(),
		})
	}
	if !fieldHasNameV0(fields, "opes_editorial_workflow") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "opes_editorial_workflow",
			Values: documentPlanOPESEditorialWorkflowV0(),
		})
	}
	if !fieldHasNameV0(fields, "opes_level_derivation_policy") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "opes_level_derivation_policy",
			Values: documentPlanOPESLevelDerivationPolicyV0(),
		})
	}
	if !fieldHasNameV0(fields, "opes_assimilation_method") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "opes_assimilation_method",
			Values: documentPlanOPESAssimilationMethodV0(),
		})
	}
	if !fieldHasNameV0(fields, "opes_quality_requirements") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "opes_quality_requirements",
			Values: documentPlanOPESQualityRequirementsV0(),
		})
	}
	return fields
}

func isDocumentPlanJobTypeV0(jobType string) bool {
	switch strings.TrimSpace(jobType) {
	case orquestadomainwork.DomainWorkKindPlanDocumentV0,
		orquestadomainwork.DomainWorkKindPlanTopicV0,
		orquestadomainwork.DomainWorkKindPlanSyllabusV0:
		return true
	default:
		return false
	}
}

func documentPlanRequiredPartsV0() []string {
	return []string{
		"sections",
		"deliverables",
		"quality_criteria",
		"review_steps",
		"visuals_when_useful",
	}
}

func documentPlanAllowedWorkKindsV0() []string {
	return []string{
		"root: plan_documento|plan_tema|plan_temario",
		"sections: draft_content_block",
		"visuals: generate_visual_asset",
		"review_steps: review_legal|review_pedagogical|review_quality|validate_topic|assemble_topic",
	}
}

func documentPlanQualityGatesV0() []string {
	return []string{
		"schema_version=domain_document_plan.v0",
		"artifact_type=document_plan",
		"work_kind raiz debe coincidir con el job_type recibido",
		"raiz DomainDocumentPlanV0: plan_ref, domain_ref, work_kind, document_kind, language_code, title, objective, estimated_pages_min, estimated_pages_max",
		"sections DomainDocumentPlanSectionV0: section_ref, order, title, objective, work_kind, target_words_min, target_words_max, required_elements, acceptance_criteria",
		"visuals DomainDocumentPlanVisualV0: visual_ref, visual_type, placement_ref, objective, work_kind",
		"review_steps DomainDocumentPlanReviewV0: review_ref, order, work_kind, objective",
		"work_kind de sections/visuals/review_steps debe usar allowed_document_plan_work_kinds",
		"deliverables DomainDocumentPlanDeliverableV0: deliverable_ref, artifact_type, title, required",
		"sections ejecutables con work_kind y acceptance_criteria",
		"deliverables obligatorios declarados",
		"sin redactar contenido final en la planificacion",
		"quality_criteria/constraints deben incorporar politica editorial OPES cuando domain_ref=opes",
	}
}

func documentPlanAcceptanceCriteriaV0() []string {
	return []string{
		"devolver DomainDocumentPlanV0 valido",
		"incluir sections y deliverables obligatorios",
		"incluir quality_criteria y review_steps aplicables",
		"no redactar el documento final dentro del plan",
		"para OPES, respetar flujo editorial: inventario, agrupacion, mapa de dependencias, temas maestros, derivacion por nivel, revision y HTML publicable",
		"si existe maestro A1/A2 o A1 equivalente, planificar primero ese maestro y despues derivar B/C1/C2/AP por resumen, reduccion editorial y adaptacion de nivel",
		"si no existe equivalente superior, marcar creacion_directa_nivel en criterios, constraints o secciones",
		"aplicar metodo OPES de asimilacion: recuperacion activa, repaso espaciado, ejemplos trabajados, carga cognitiva controlada, visuales utiles, elaboracion e intercalado",
	}
}

func documentPlanOPESEditorialWorkflowV0() []string {
	return []string{
		"inventario completo del temario",
		"agrupacion por comunes, transversales y especificos",
		"mapa de dependencias",
		"identificacion de temas maestros",
		"redaccion o validacion del maestro superior",
		"derivacion por nivel",
		"revision editorial",
		"HTML publicable como salida canonica",
	}
}

func documentPlanOPESLevelDerivationPolicyV0() []string {
	return []string{
		"A1/A2 o A1 maestro -> B/C1 -> C2/AP",
		"si existe equivalente superior, crear o validar primero el maestro superior",
		"B/C1/C2/AP se derivan despues por resumen, reduccion editorial y adaptacion de nivel",
		"los niveles inferiores pueden aportar contraste, ejemplos operativos o necesidades de examen, pero no fijan el canon superior",
		"no elevar un resumen B/C1/C2/AP para construir un maestro A1/A2 o A1",
		"si no hay equivalente superior, redactar en el nivel solicitado y marcar creacion_directa_nivel",
	}
}

func documentPlanOPESAssimilationMethodV0() []string {
	return []string{
		"recuperacion activa",
		"repaso espaciado",
		"ejemplos trabajados y fading",
		"carga cognitiva controlada",
		"doble codificacion con visuales utiles",
		"elaboracion y preguntas profundas",
		"intercalado con temas relacionados",
	}
}

func documentPlanOPESQualityRequirementsV0() []string {
	return []string{
		"mapa inicial y orientacion de examen",
		"definiciones claras y desarrollo por capitulos cortos",
		"autores, teorias, doctrina o modelos cuando proceda",
		"normativa, versiones y fuentes oficiales actualizadas cuando proceda",
		"procedimientos paso a paso, tablas comparativas y esquemas utiles",
		"ejemplos de oposicion, errores frecuentes y claves de examen",
		"preguntas de recuperacion por capitulo y preguntas finales",
		"plan de repaso espaciado",
		"conexiones con temas relacionados",
		"fuentes locales o descargadas sin depender de enlaces fragiles",
		"modo tutor completo: que significa, por que importa, con que se confunde y como se reconoce en examen",
		"tono adulto, claro y tecnico; facil pedagogicamente, no infantilizar",
		"notas de test separadas de la teoria; trampas y distractores solo en notas de test, enfoque o repaso",
		"primera lectura continua: tablas, esquemas, test y visuales apoyan, no sustituyen el texto",
		"repaso antes del examen con mapa de ideas, definiciones rapidas, diferencias, errores y checklist",
	}
}

func opesGlobalEditorialPolicyV0() []string {
	return []string{
		"guia_global_estilo_2026_05_18_aplica_a_todos_los_temarios_opes",
		"objetivo: tema profesional, estudiable, riguroso, trazable y preparado para examen; no volcado de informacion",
		"A1: 45-50 folios, 20.250-22.500 palabras, minimo de cierre 20.250 palabras salvo regla superior explicita",
		"no marcar ready_profesional, ready_candidate_html ni apto_para_subida_controlada si A1 no alcanza el minimo aplicable",
		"estructura minima: orientacion examen, mapa/ruta, conceptos clave, definiciones autoridad, bloques cortos, tablas, ejemplos, modo tutor, notas test, visuales, supuestos, errores, repaso, enfoque, muestra test y fuentes",
		"modo tutor completo: que significa, por que importa, con que se confunde y como se reconoce en examen",
		"tono adulto, claro y tecnico; facil pedagogicamente, no infantilizar ni rellenar",
		"si existe tema o artefacto previo, estudiar primero que falla y priorizar modificacion localizada; rehacer completo solo con justificacion cuando sea mas sencillo o seguro que corregir",
		"notas de test separadas de la teoria; distractores y trampas solo en notas de test, enfoque o repaso",
		"primera lectura continua; tablas, esquemas, test y visuales son apoyo y no sustituyen el desarrollo teorico",
		"comprension lectora: una idea por parrafo, tecnicismo explicado, definicion antes del desarrollo complejo y ejemplos despues de conceptos abstractos",
		"metodo asimilacion: recuperacion activa, repaso espaciado, ejemplos trabajados, carga cognitiva controlada, doble codificacion, elaboracion e intercalado",
		"test progresivo en niveles base, aplicacion y examen real solo como banco privado para afiliados; no publicar test de prueba dentro del tema abierto",
		"banco_preguntas_i18n_es.json obligatorio para afiliados: minimo 50 preguntas por tema, 4 respuestas por pregunta, una correcta identificable y distractores plausibles que exijan conocer bien la materia; debe quedar en el mismo paquete/directorio de su temario; prohibido banco comun, banco global o reutilizacion indiferenciada entre temas; prohibidos distractores tontos, absurdos o claramente descartables",
		"tutor diagnostico de test: concepto fallado, por que la opcion elegida es incorrecta y donde repasar",
		"supuestos practicos guiados cuando proceda: situacion, pistas, preguntas, resolucion paso a paso, errores, criterio y mini comprobacion",
		"repaso antes del examen: mapa de 5-7 ideas, definiciones rapidas, diferencias, errores, checklist y preguntas de recuperacion",
		"visuales utiles no decorativos; preferir SVG, HTML o CSS determinista para diagramas con rotulos; fotos solo oficiales, licenciadas o generadas cuando aporten aprendizaje",
		"movil: esquemas responsivos con scroll horizontal; no recortar contenido ni hacerlo ilegible",
		"fuentes: normativa oficial espanola y europea primero; BOE, BOJA, BOP y UE; doctrina, jurisprudencia y guias; internacionales solo como apoyo",
		"texto visible no muestra rutas, agentes, prompts, arquitectura, ids tecnicos ni decisiones internas",
		"contenido visible debe ser i18n/localizable cuando vaya a produccion",
		"prohibido: listas densas sin explicar, parrafos con demasiadas comas, tono infantil, visuales decorativos, bancos completos dentro del tema, distractores absurdos y placeholders",
	}
}

func opesHTMLPublicationPolicyV0() []string {
	return []string{
		"html_publicable_2026_05_19_aplica_a_temarios_opes_cuando_se_genere_html",
		"patron web tipo Tema 11 cerrado por opes_html_topic_template_v1: hero, barra de modo de estudio, contenido principal y barra lateral plegable; no inventar otra estructura visual",
		"primera lectura activa por defecto; debe poder ocultar tablas, test, visuales y notas para lectura continua",
		"modo tutor con explicaciones adicionales separadas del texto base",
		"notas de test ocultables y con formato unico: fondo azul, barra izquierda y texto diferenciado de la teoria",
		"supuestos practicos con solucion ocultable cuando proceda",
		"repaso antes del enfoque de examen y test progresivo separado del temario",
		"banco de preguntas i18n externo por tema y reservado a la parte de afiliados; queda junto a su temario como artefacto del paquete; no existe banco comun de tests; el HTML del tema no incluye test de prueba, preguntas sueltas publicas ni banco final completo",
		"banco externo para afiliados: minimo 50 preguntas, 4 opciones, respuesta correcta y feedback; los distractores deben ser errores verosimiles de opositor, no bromas, obviedades ni respuestas imposibles",
		"markdown_final, markdown_importable y html_final deben mantenerse sincronizados cuando existan los tres formatos",
		"html final debe parsear correctamente y no referenciar assets inexistentes",
		"no mostrar URLs reales visibles en Markdown/HTML final; archivar fuentes externas localmente y citar de forma editorial",
		"URLs ficticias solo permitidas si son ejemplos tecnicos claramente no operativos",
		"assets visuales locales por tema: mapa SVG propio y reutilizacion de esquemas A1/canones cuando existan",
		"contenedores de esquemas con overflow/scroll horizontal y slider para evitar recortes en movil",
		"visuales pequenos: usar SVG determinista, HTML/CSS local o flujo ligero/estable; sd35large queda reservado para fotos grandes tipo hero",
		"fotos generadas o locales deben pasar revision humana antes de produccion si pueden no aportar valor didactico",
		"contenido privado de staging debe permanecer en rutas privadas/autenticadas; no copiar a static, www ni rutas publicas sin permisos",
		"servidor remoto puede revisar texto, HTML/Markdown e integracion, pero no debe usarse para generar imagenes",
		"ficheros manejables: evitar HTML o Markdown gigantes imposibles de revisar; partir bancos y assets en ficheros independientes",
		"cuando un paquete de nivel se adapte a otro cuerpo o grupo, crear derivacion separada y no reutilizar sin capa especifica del dominio",
	}
}

func opesHTMLTopicTemplateV1() []string {
	return []string{
		"template_ref=opes_html_topic_template_v1",
		"renderer obligatorio: modulos/orquesta-opes-bridge/scripts/opes_render_topic_html_v1.py usando modulos/orquesta-opes-bridge/templates/opes_html_topic_template_v1.html",
		"validador obligatorio: modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py sobre cada paquete final antes de marcarlo como listo",
		"doctype obligatorio: <!doctype html><html lang=\"es\"> con meta charset utf-8 y viewport responsive",
		"body por defecto: class=\"first-reading-on\" y data-topic-key=\"temaNNN\"; primera lectura debe estar activa al abrir",
		"estructura fija: header.hero, div.mode-bar, div.layout, nav.side-nav, main.content, section.topic-section",
		"hero: eyebrow con tema/categoria, h1 con titulo oficial, subtitle breve y hero media local opcional; sin imagen remota",
		"mode-bar sticky: controles checkbox con ids firstReadingToggle, tutorToggle, testNotesToggle y visualsToggle",
		"side-nav plegable: indice generado desde h2/h3 del contenido; boton navToggle con aria-expanded",
		"main.content: cada bloque doctrinal vive en section.topic-section con id estable y h2 unico",
		"bloques permitidos: p, ul/ol, blockquote doctrinal, div.table-wrap>table, figure.visual-card, aside.tutor, aside.test-note, details.case, div.question",
		"primera lectura: ocultar .test-note, .visual-section, .question, details.case, figuras y tablas auxiliares; mantener texto continuo legible",
		"modo tutor: aside.tutor oculto por defecto salvo primera lectura o toggle tutor-on; no mezclar estrategia de test en tutor",
		"notas de test: aside.test-note oculto por defecto, fondo azul claro, barra izquierda azul y titulo uniforme Nota de test",
		"visuales: solo assets locales relativos; envolver diagramas grandes en .diagram-scroll y, si hace falta, .diagram-slider",
		"supuestos: usar details.case con summary y solucion ocultable; no bloquear primera lectura",
		"banco final: no incrustar banco completo ni test de prueba en HTML publico; el consumo del JSON pertenece a la parte privada/de afiliados",
		"banco final JSON afiliados: `banco_preguntas_i18n_es.json` en el directorio del tema, minimo 50 items, cada item con enunciado, 4 opciones, respuesta correcta, explicacion y feedback de distractores cuando proceda",
		"css: un unico style local en head; sin frameworks, fuentes remotas, scripts remotos ni dependencias CDN",
		"javascript: solo toggles de clases first-reading-on, tutor-on, test-notes-on, visuals-off y nav-collapsed",
		"accesibilidad: aria-label en controles, alt descriptivo en imagenes, contraste suficiente y foco visible",
		"movil: layout una columna, side-nav no sticky, tablas/diagramas con overflow horizontal; sin texto solapado",
		"validacion obligatoria: 20.250-22.500 palabras A1, HTML parseable, refs src/href locales existentes, banco JSON con 50 preguntas y 4 opciones, SVG parseable, sin duplicados largos, sin file/http externos visibles y sin rutas internas",
	}
}
