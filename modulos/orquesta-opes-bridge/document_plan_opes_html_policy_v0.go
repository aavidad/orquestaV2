package orquestaopesbridge

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
