package orquestaopesbridge

func documentPlanOPESEditorialWorkflowV0() []string {
	return []string{
		"inventario completo del temario",
		"investigacion por internet de examenes, convocatorias y temarios de administraciones relacionadas con evidencia verificable",
		"agrupacion por comunes, transversales y especificos",
		"mapa de dependencias",
		"identificacion de temas maestros",
		"redaccion o validacion del maestro superior",
		"derivacion por nivel",
		"redaccion de todos los temas y apartados",
		"infografias utiles y no excesivas: integrarlas en puntos importantes, dificiles, comparativos o procedimentales donde aporten aprendizaje; evitar relleno visual",
		"banco de tests por tema",
		"revision editorial",
		"validacion legal, pedagogica, calidad, ortografia y consistencia",
		"audio accesible por tema y por apartado",
		"tutor y bots del temario",
		"HTML local operativo con logos USO y aspecto USO/TCAE promocion interna como salida canonica antes de produccion",
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
		"banco de tests por tema con 4 opciones A-D, una correcta exacta, distractores plausibles, explicacion tutor, HTML revisable, metadata, informe y validaciones limpias",
		"si hay importacion local de tests, backup Postgres previo, SQL con borrado limitado al banco nuevo y verificacion de conteos",
		"plan de repaso espaciado",
		"conexiones con temas relacionados",
		"infografias por tema o apartado solo cuando aporten aprendizaje, con texto alternativo, placement_ref y objetivo didactico",
		"audios por apartado con manifest trazable y revision de numeros romanos",
		"tutor/bots con alcance por tema, fuentes permitidas y diagnostico de errores",
		"HTML local revisable antes de produccion con marca USO/TCAE promocion interna",
		"fuentes locales o descargadas sin depender de enlaces fragiles",
		"modo tutor completo: que significa, por que importa, con que se confunde y como se reconoce en examen",
		"tono adulto, claro y tecnico; facil pedagogicamente, no infantilizar",
		"notas de test separadas de la teoria; trampas y distractores solo en notas de test, enfoque o repaso",
		"primera lectura continua: tablas, esquemas, test y visuales apoyan, no sustituyen el texto",
		"repaso antes del examen con mapa de ideas, definiciones rapidas, diferencias, errores y checklist",
	}
}
