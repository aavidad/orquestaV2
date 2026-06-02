package orquestaopesbridge

func opesGlobalEditorialPolicyV0() []string {
	return []string{
		"guia_global_estilo_2026_05_18_aplica_a_todos_los_temarios_opes",
		"objetivo: tema profesional, estudiable, riguroso, trazable y preparado para examen; no volcado de informacion",
		"A1: 45-50 folios, 20.250-22.500 palabras, minimo de cierre 20.250 palabras salvo regla superior explicita",
		"no marcar ready_profesional, ready_candidate_html ni apto_para_subida_controlada si A1 no alcanza el minimo aplicable",
		"estructura minima: orientacion examen, mapa/ruta, conceptos clave, definiciones autoridad, bloques cortos, tablas, ejemplos, modo tutor, notas test, visuales, supuestos, errores, repaso, enfoque, muestra test y fuentes",
		"flujo temario completo local: investigacion de examenes relacionados, redaccion por temas, infografias por tema, banco de tests, revisiones, validacion, ensamblado, audios por apartado, tutor/bots y HTML local operativo antes de produccion",
		"investigacion externa: buscar por internet examenes, convocatorias, temarios y pruebas de administraciones relacionadas; priorizar boletines, sedes oficiales, tribunales, universidades/institutos publicos y fuentes sindicales verificables; registrar URL, fecha, administracion, anio, categoria y utilidad",
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
		"tutor/bots del temario obligatorios para local: responder dudas por tema/apartado, explicar fallos de test, recomendar repaso y limitarse a fuentes y contenido del paquete",
		"audios accesibles obligatorios: audio por tema y por apartado/seccion, manifest con section_ref, duracion, formato, checksum/ref y revision de numeros romanos antes de TTS",
		"HTML local obligatorio antes de produccion: usar logos USO y aspecto coherente con la web USO/TCAE promocion interna que aporte el adaptador OPES/USO; navegacion, modo estudio, audios, tutor, tests permitidos y assets locales deben funcionar sin backend productivo",
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
