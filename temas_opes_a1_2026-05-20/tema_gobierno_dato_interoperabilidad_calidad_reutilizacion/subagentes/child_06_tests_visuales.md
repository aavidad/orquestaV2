# Entrega parcial child_06: visuales, supuestos, test progresivo y checklist editorial

## 1. Alcance de esta entrega

Esta entrega aporta material de apoyo para el tema A1 **Gobierno del dato, interoperabilidad semantica, calidad del dato y reutilizacion de la informacion publica**. Esta pensada para integrarse en un tema largo, no para sustituir el desarrollo teorico principal. El material se organiza en cuatro bloques:

1. Propuestas de visuales didacticos deterministas.
2. Supuestos practicos guiados con solucion separable.
3. Muestra de test progresivo, sin banco completo.
4. Checklist editorial A1 para validar calidad, trazabilidad y preparacion de examen.

Los visuales sugeridos deben construirse como SVG, HTML/CSS o tablas responsivas locales. No necesitan fotografia. El banco completo de preguntas debe permanecer fuera del tema, en ficheros i18n separados; aqui solo se define estructura, muestras y criterios.

## 2. Principios didacticos para este tema

El tema combina derecho administrativo, tecnologia de datos y gobierno organizativo. El riesgo principal para el opositor A1 es estudiar conceptos sueltos sin entender su relacion operativa. Por eso conviene que los visuales y las preguntas no se limiten a nombrar normas o siglas. Deben obligar a distinguir niveles: dato, metadato, modelo semantico, servicio interoperable, reutilizacion, calidad, proteccion de datos, seguridad y responsabilidad publica.

La secuencia pedagogica recomendada es:

1. Primero, fijar el mapa: gobierno del dato como sistema de decision, no como repositorio.
2. Despues, separar interoperabilidad tecnica, semantica, organizativa y juridica.
3. A continuacion, introducir calidad del dato como propiedad medible y gestionable.
4. Luego, explicar reutilizacion de informacion publica como regimen juridico y operativo.
5. Finalmente, entrenar casos donde chocan apertura, privacidad, seguridad, confidencialidad, propiedad intelectual, licencias, coste marginal, trazabilidad y valor publico.

El material de test debe evaluar comprension aplicada. Las preguntas literales son utiles solo en nivel base. En nivel A1, el examen suele premiar la capacidad de reconocer conflictos: no todo dato publico es dato abierto, no toda interoperabilidad es tecnica, no todo metadato garantiza calidad, no toda reutilizacion equivale a transparencia y no toda cesion de datos queda justificada por eficiencia administrativa.

## 3. Propuestas de visuales didacticos

### Visual 1. Mapa de capas del gobierno del dato publico

**Objetivo didactico:** mostrar que el gobierno del dato no es una herramienta unica, sino una arquitectura de responsabilidades, reglas, procesos y controles.

**Formato recomendado:** SVG local responsivo con cuatro franjas horizontales y conectores verticales. Debe admitir desplazamiento horizontal en movil.

**Estructura visual:**

| Capa | Contenido | Mensaje clave |
|---|---|---|
| Direccion y responsabilidad | estrategia, comite de datos, roles, prioridades, cartera de datos | El dato se gobierna con decisiones explicitas y responsables identificables. |
| Reglas y marco juridico | proteccion de datos, transparencia, reutilizacion, interoperabilidad, seguridad, archivo | La apertura y el intercambio se subordinan al marco juridico aplicable. |
| Gestion operativa | inventario, metadatos, ciclo de vida, calidad, linaje, catalogo, validaciones | La calidad se produce mediante procesos repetibles, no por declaracion. |
| Uso y valor publico | servicios digitales, analitica, reutilizacion, evaluacion, rendicion de cuentas | El dato se justifica por mejores decisiones, servicios y control democratico. |

**Texto alternativo recomendado:** "Mapa de cuatro capas que conecta decision, reglas, gestion operativa y uso del dato publico".

**Notas de integracion:** ubicarlo tras la orientacion de examen o el mapa inicial. No debe contener siglas sin desarrollar. Puede incluir etiquetas cortas: "responsables", "normas", "procesos", "valor".

### Visual 2. Diferencia entre dato, informacion, documento, metadato y conjunto de datos

**Objetivo didactico:** evitar confusiones basicas que suelen contaminar preguntas de reutilizacion e interoperabilidad.

**Formato recomendado:** tabla enriquecida o tarjetas SVG con flechas de composicion.

**Estructura visual:**

| Concepto | Funcion | Ejemplo administrativo | Error frecuente |
|---|---|---|---|
| Dato | unidad registrable | fecha de presentacion de una solicitud | Creer que por si solo explica el expediente. |
| Informacion | dato interpretado en contexto | estado de tramitacion de una ayuda | Confundirla con el soporte documental. |
| Documento | unidad con contenido y contexto probatorio | resolucion administrativa firmada | Pensar que solo existe en papel o PDF. |
| Metadato | dato que describe otro dato o documento | organo emisor, fecha, formato, licencia | Tratarlo como adorno tecnico. |
| Conjunto de datos | coleccion estructurada reutilizable | listado anonimizado de contratos menores | Publicarlo sin calidad, licencia o diccionario. |

**Mensaje clave:** el examen puede mezclar estos terminos. La respuesta correcta suele depender del nivel de abstraccion.

### Visual 3. Interoperabilidad en cuatro dimensiones

**Objetivo didactico:** distinguir la interoperabilidad semantica de otras capas.

**Formato recomendado:** diagrama de cuatro columnas con una misma transaccion atravesando las capas.

**Estructura visual:**

| Dimension | Pregunta que resuelve | Ejemplo |
|---|---|---|
| Juridica | ¿Puede compartirse este dato y con que base? | habilitacion normativa, finalidad, limites, conservacion |
| Organizativa | ¿Quien hace que y en que proceso? | convenio, protocolo, procedimiento comun, responsable funcional |
| Semantica | ¿Significa lo mismo para todos? | vocabulario comun, modelo de datos, codigos normalizados |
| Tecnica | ¿Como se transmite de forma segura y usable? | API, formato, identificador, autenticacion, registro de intercambio |

**Enfoque tutor:** resaltar que la interoperabilidad semantica no consiste solo en usar XML, JSON o una API. Consiste en que los conceptos compartidos tengan significado comun, reglas de interpretacion y equivalencias controladas.

### Visual 4. Ciclo de vida de un dato publico

**Objetivo didactico:** relacionar calidad, proteccion, reutilizacion y archivo.

**Formato recomendado:** flujo circular con siete etapas.

**Etapas sugeridas:**

1. Captura o generacion.
2. Validacion inicial.
3. Registro y metadatado.
4. Uso interno y tramitacion.
5. Intercambio entre administraciones.
6. Publicacion o reutilizacion cuando proceda.
7. Conservacion, archivo, depuracion o eliminacion.

**Controles transversales:** finalidad, minimizacion, seguridad, calidad, trazabilidad, accesibilidad, licencia, conservacion.

**Pregunta que debe responder el visual:** en que momentos hay que decidir si el dato puede abrirse, si debe anonimizarse, si necesita conservarse y si esta en condiciones de ser reutilizado.

### Visual 5. Matriz calidad del dato: dimensiones, controles y evidencias

**Objetivo didactico:** convertir la calidad en algo medible.

**Formato recomendado:** tabla markdown en el tema y version SVG compacta en HTML.

| Dimension | Que pregunta | Control ejemplo | Evidencia |
|---|---|---|---|
| Exactitud | ¿Refleja correctamente la realidad administrativa? | contraste con fuente autentica | regla de validacion y muestra revisada |
| Completitud | ¿Faltan campos necesarios? | campos obligatorios y umbrales | porcentaje de nulos por atributo |
| Consistencia | ¿Hay contradicciones internas? | reglas entre campos | incidencias de incoherencia |
| Actualidad | ¿Esta vigente para el uso previsto? | fecha de actualizacion y caducidad | sello temporal o version |
| Unicidad | ¿Evita duplicados indebidos? | identificadores y deduplicacion | tasa de duplicados |
| Validez | ¿Respeta formato, dominio y reglas? | catalogos cerrados y formatos | rechazos por validacion |
| Trazabilidad | ¿Puede saberse origen y transformaciones? | linaje y registro de cambios | historial de versionado |

**Nota de examen:** si una pregunta habla de "dato correcto pero antiguo", el problema no es exactitud en sentido estricto, sino actualidad o vigencia para el uso.

### Visual 6. Arbol de decision: publicar, compartir, restringir o no reutilizar

**Objetivo didactico:** entrenar decisiones con limites juridicos y tecnicos.

**Formato recomendado:** arbol de decision SVG con respuestas "si/no" y nodos finales.

**Nodos minimos:**

1. ¿El dato o documento esta dentro del ambito de una funcion publica?
2. ¿Existe limite legal, secreto, seguridad, confidencialidad, propiedad intelectual de terceros o proteccion de datos?
3. ¿Puede aplicarse anonimizacion, agregacion o disociacion con riesgo aceptable?
4. ¿Tiene calidad, metadatos y formato reutilizable suficientes?
5. ¿Hay licencia, condiciones y punto de acceso claros?
6. Resultado: publicar como dato abierto, compartir de forma controlada, publicar informacion agregada, denegar reutilizacion, aplazar por calidad insuficiente o someter a revision.

**Mensaje clave:** la decision no es binaria. Hay grados: acceso interno, intercambio interadministrativo, publicacion general, reutilizacion con condiciones o no publicacion.

### Visual 7. Diagrama de roles en gobierno del dato

**Objetivo didactico:** separar responsabilidades estrategicas, funcionales, tecnicas y juridicas.

**Formato recomendado:** matriz RACI simplificada.

| Actividad | Direccion | Responsable funcional | Unidad de datos | TIC | Juridico/DPD | Archivo/gestion documental |
|---|---|---|---|---|---|---|
| Priorizar dominios de datos | A | R | C | C | C | C |
| Definir significado y reglas | C | A/R | R | C | C | C |
| Implantar controles tecnicos | C | C | C | A/R | C | C |
| Evaluar publicacion | A | R | R | C | R | C |
| Gestionar conservacion | C | R | C | C | C | A/R |

**Convencion:** A = aprueba o asume responsabilidad final; R = ejecuta o responde funcionalmente; C = consulta experta.

**Nota de integracion:** explicar que los nombres concretos de puestos pueden variar por administracion. Lo examinable es la logica de responsabilidad.

### Visual 8. Flujo de reutilizacion de informacion publica

**Objetivo didactico:** conectar inventario, catalogo, licencia, formato, metadatos, calidad y uso por terceros.

**Formato recomendado:** flujo lineal con retroalimentacion.

**Secuencia:**

1. Identificacion de documentos o conjuntos de datos reutilizables.
2. Analisis de limites y derechos.
3. Preparacion: calidad, formato abierto, metadatos, anonimizacion si procede.
4. Publicacion en catalogo o puesta a disposicion.
5. Condiciones de reutilizacion claras.
6. Consumo por terceros.
7. Retroalimentacion: errores detectados, demanda, mejoras, nuevas versiones.

**Punto critico:** incluir una señal visual donde se separen transparencia, acceso y reutilizacion. Estan relacionados, pero no son identicos.

### Visual 9. Semaforo de riesgos para compartir datos

**Objetivo didactico:** facilitar supuestos practicos.

**Formato recomendado:** tabla con color accesible, no depender solo del color.

| Riesgo | Bajo | Medio | Alto |
|---|---|---|---|
| Identificacion personal | datos anonimos robustos | seudonimizados o agregados pequenos | identificadores directos o combinables |
| Sensibilidad | datos no personales ordinarios | datos con impacto economico o reputacional | categorias especiales, sanciones, vulnerabilidad |
| Finalidad | claramente compatible | requiere ponderacion | finalidad nueva no justificada |
| Seguridad | canal y registro adecuados | controles parciales | sin trazabilidad ni control de acceso |
| Calidad | validada y documentada | incidencias conocidas | errores no medidos |

**Uso recomendado:** antes de la solucion de supuestos, como herramienta de analisis.

### Visual 10. Taxonomia de errores de examen

**Objetivo didactico:** preparar repasos finales.

**Formato recomendado:** esquema radial o lista agrupada.

**Categorias:**

1. Confundir apertura con reutilizacion.
2. Confundir interoperabilidad tecnica con interoperabilidad semantica.
3. Olvidar proteccion de datos y confidencialidad.
4. Pensar que la calidad es solo correccion formal.
5. Publicar sin metadatos, licencia o formato util.
6. Tratar el dato como propiedad de una unidad administrativa aislada.
7. No distinguir dato original, transformado, agregado y anonimizado.
8. Ignorar archivo, conservacion y ciclo de vida.

## 4. Supuestos practicos guiados

Los supuestos deben aparecer despues del desarrollo teorico. Conviene que cada uno tenga situacion, pistas, preguntas, solucion paso a paso, errores frecuentes y mini comprobacion. En HTML, la solucion deberia poder ocultarse.

### Supuesto 1. Publicacion de datos de subvenciones municipales

**Situacion:** un ayuntamiento quiere publicar un conjunto de datos sobre subvenciones concedidas durante los ultimos cinco ejercicios. El area de transparencia propone publicar una hoja de calculo con beneficiarios, importes, concepto, fecha de concesion, estado de justificacion y observaciones internas. Algunos beneficiarios son personas fisicas, asociaciones pequenas y empresas. La unidad tecnica pregunta si basta con retirar el DNI.

**Pistas para el alumno:**

- Hay que distinguir transparencia, reutilizacion y proteccion de datos.
- La eliminacion de identificadores directos no siempre anonimiza.
- La calidad incluye definicion de campos, periodicidad, formatos y versionado.
- Las observaciones internas pueden contener informacion no normalizada o excesiva.
- La publicacion debe incorporar licencia o condiciones de reutilizacion.

**Preguntas:**

1. ¿Que decisiones previas debe adoptar la administracion antes de publicar?
2. ¿Que campos requieren especial revision?
3. ¿Que controles de calidad y metadatos son exigibles para que el conjunto sea reutilizable?
4. ¿Como se diferenciaria una publicacion para transparencia de una publicacion orientada a reutilizacion?

**Solucion orientativa:**

Primero debe identificarse la base juridica de la publicacion y su finalidad. Si la publicacion deriva de obligaciones de transparencia, no se convierte automaticamente en reutilizacion sin condiciones. La reutilizacion exige que los documentos o datos se pongan a disposicion con reglas claras, formatos adecuados y respeto de los limites aplicables.

Segundo, debe analizarse el riesgo de identificacion. Retirar el DNI no basta si el nombre, el municipio, la cuantia, la finalidad de la ayuda o las observaciones permiten identificar a una persona y revelar informacion sensible o especialmente protegida. En beneficiarios personas fisicas puede ser necesario ponderar, agregar, suprimir campos, limitar observaciones o aplicar tecnicas de anonimizacion. En personas juridicas el riesgo de proteccion de datos es menor, pero pueden existir otros limites: confidencialidad, secreto comercial o informacion no necesaria.

Tercero, deben normalizarse los campos. "Concepto", "linea de ayuda", "estado de justificacion" y "tipo de beneficiario" requieren diccionario de datos. Si cada unidad escribe textos libres, la reutilizacion sera pobre y la comparacion entre ejercicios sera dificil. La calidad exige completitud, consistencia temporal, identificadores estables de convocatoria, versionado y fecha de actualizacion.

Cuarto, deben separarse observaciones internas de datos publicables. Un campo de observaciones puede incluir valoraciones, incidencias o referencias personales no pertinentes. Lo correcto es convertirlo en campos normalizados cuando aporte informacion reutilizable o excluirlo si no supera el analisis juridico y de minimizacion.

Quinto, la publicacion reutilizable debe ofrecer formato abierto o al menos procesable, metadatos, licencia o condiciones de reutilizacion, periodicidad, contacto responsable y advertencia sobre la version. Publicar solo un PDF o una hoja sin diccionario no satisface una buena practica de reutilizacion, aunque pueda cumplir parcialmente una obligacion informativa.

**Errores frecuentes:**

- Creer que eliminar el DNI anonimiza siempre.
- Publicar observaciones libres sin revision.
- Pensar que una tabla subida a un portal ya es dato abierto de calidad.
- No documentar la fecha de actualizacion ni la licencia.
- Confundir beneficiario persona juridica con ausencia total de limites.

**Mini comprobacion:**

Si el alumno puede explicar por que "dato publico" no significa "dato publicable sin condiciones", ha entendido el nucleo del supuesto.

### Supuesto 2. Intercambio de datos entre dos consejerias para simplificar un procedimiento

**Situacion:** una consejeria que gestiona becas pide a otra consejeria datos de discapacidad y familia numerosa para evitar que los ciudadanos aporten certificados. La unidad promotora quiere implantar una API directa. Cada consejeria usa codigos distintos para grados, fechas de vigencia y estados administrativos.

**Pistas para el alumno:**

- La simplificacion administrativa no elimina la necesidad de base juridica, finalidad y controles.
- La API resuelve transporte, no significado.
- Los codigos divergentes son un problema semantico.
- Debe existir trazabilidad del intercambio.
- La calidad del dato fuente condiciona la decision automatizada o asistida.

**Preguntas:**

1. ¿Que dimensiones de interoperabilidad aparecen?
2. ¿Por que el problema no se resuelve solo con una API?
3. ¿Que elementos semanticos habria que acordar?
4. ¿Que evidencias deberian conservarse?

**Solucion orientativa:**

El supuesto contiene interoperabilidad juridica, organizativa, semantica y tecnica. La juridica exige comprobar habilitacion, finalidad, minimizacion y regimen de consulta. La organizativa exige definir quien solicita, quien responde, en que procedimiento, con que plazos y bajo que responsabilidad. La semantica exige acordar significados: que se entiende por grado reconocido, fecha de efectos, vigencia, suspension, revision pendiente, familia numerosa general o especial, y como se traducen codigos entre sistemas. La tecnica incluye API, autenticacion, cifrado, disponibilidad, monitorizacion y registro.

La API no resuelve el problema de fondo porque dos sistemas pueden intercambiar mensajes correctos tecnicamente y entender cosas distintas. Si un sistema codifica "vigente" como resolucion no caducada y otro como certificado no suspendido, la decision puede ser erronea aunque la llamada tecnica funcione.

Los elementos semanticos minimos son un modelo de datos compartido, diccionario de atributos, catalogos de valores, reglas de equivalencia, versionado del esquema, tratamiento de nulos e incidencias, reglas de vigencia y criterios sobre datos historicos. Tambien debe definirse que ocurre si el dato fuente esta en revision o presenta conflicto.

Las evidencias incluyen registro de consulta, finalidad, identificacion del procedimiento, resultado devuelto, version del servicio, fecha y hora, reglas aplicadas, incidencias y, cuando proceda, informacion para auditoria de proteccion de datos y seguridad. No se trata de acumular datos innecesarios, sino de poder justificar la decision y detectar errores.

**Errores frecuentes:**

- Llamar "interoperabilidad semantica" al mero uso de JSON.
- No definir codigos comunes.
- Ignorar la vigencia temporal del dato.
- Automatizar una decision con datos no validados.
- No prever registros de auditoria.

**Mini comprobacion:**

Si el alumno puede distinguir "conectar sistemas" de "compartir significado", domina la idea central.

### Supuesto 3. Catalogo regional de datos abiertos con baja calidad

**Situacion:** una administracion autonomica mantiene un catalogo de datos abiertos. Tiene muchos conjuntos publicados, pero los reutilizadores se quejan de enlaces rotos, duplicados, falta de metadatos, licencias inconsistentes y formatos no procesables. La direccion presume de numero de datasets.

**Pistas para el alumno:**

- La cantidad de conjuntos no mide por si sola la madurez.
- La reutilizacion exige usabilidad real.
- Calidad del catalogo y calidad del dato son dimensiones relacionadas.
- La gobernanza necesita responsables por dominio.
- La retroalimentacion de reutilizadores es una fuente de mejora.

**Preguntas:**

1. ¿Que indicadores propondria para evaluar el catalogo?
2. ¿Que medidas de gobierno del dato corregirian el problema?
3. ¿Como priorizaria la mejora si los recursos son limitados?
4. ¿Que no deberia hacerse?

**Solucion orientativa:**

Los indicadores deben medir disponibilidad, actualizacion, completitud de metadatos, coherencia de licencias, formatos reutilizables, existencia de diccionarios, tasa de enlaces validos, demanda o uso, incidencias, tiempo de correccion y nivel de automatizacion de actualizaciones. Tambien debe medirse la calidad de conjuntos de alto valor, no solo el volumen global.

Las medidas de gobierno incluyen designar responsables funcionales por dominio, aprobar criterios minimos de publicacion, implantar validaciones automaticas, normalizar licencias, exigir metadatos obligatorios, crear un proceso de versionado, revisar conjuntos obsoletos y establecer canal de incidencias. La unidad de datos debe coordinar, pero las areas propietarias conservan responsabilidad sobre significado y calidad.

La priorizacion deberia atender valor publico, demanda, impacto economico o social, obligaciones legales, riesgo de error y facilidad de correccion. Puede ser razonable retirar temporalmente o marcar como obsoletos conjuntos sin mantenimiento, siempre con transparencia y plan de sustitucion.

No deberia premiarse a las unidades solo por publicar mas conjuntos. Tampoco conviene convertir todos los datos a formatos visualmente atractivos pero no procesables. La mejora real requiere menos escaparate y mas calidad operacional.

**Errores frecuentes:**

- Confundir catalogo con gobierno del dato.
- Medir exito por numero bruto de datasets.
- Publicar sin responsable de mantenimiento.
- Cambiar formatos sin documentar versiones.
- Ignorar a los reutilizadores.

**Mini comprobacion:**

Una respuesta madura propone indicadores, responsables y proceso de mejora, no solo "actualizar los datos".

### Supuesto 4. Espacio de datos publico-privado para movilidad

**Situacion:** una ciudad quiere crear un espacio de datos de movilidad con operadores de transporte, aparcamientos, bicicletas compartidas y sensores urbanos. Se pretende combinar datos publicos y privados para mejorar planificacion, informacion al ciudadano y estudios ambientales. Algunos operadores temen revelar informacion comercial.

**Pistas para el alumno:**

- Un espacio de datos no es un portal de descarga indiscriminada.
- Deben definirse roles, condiciones de acceso, finalidad y reglas de uso.
- Puede haber datos personales, datos no personales y datos comercialmente sensibles.
- La interoperabilidad semantica es esencial para integrar fuentes heterogeneas.
- La reutilizacion publica puede coexistir con accesos graduados.

**Preguntas:**

1. ¿Que modelo de gobernanza seria necesario?
2. ¿Que datos podrian abrirse y cuales requeririan acceso controlado?
3. ¿Que papel tiene la calidad del dato?
4. ¿Como se evitaria que la interoperabilidad sea solo tecnica?

**Solucion orientativa:**

El modelo de gobernanza debe establecer participantes, finalidades, reglas de acceso, categorias de datos, obligaciones de calidad, responsabilidades, controles de seguridad, solucion de incidencias, auditoria y mecanismos de revision. Tambien debe distinguir usos internos de planificacion, informacion publica agregada, investigacion y reutilizacion por terceros.

Podrian abrirse datos agregados de disponibilidad, horarios, rutas, puntos de recarga o indicadores ambientales si no comprometen privacidad ni confidencialidad. Requeririan acceso controlado los datos de trazas detalladas, patrones que permitan identificar personas, informacion comercial sensible o datos con valor competitivo directo. En algunos casos bastara agregacion espacial o temporal; en otros sera necesario denegar o limitar.

La calidad es critica porque decisiones de movilidad pueden afectar inversiones, regulacion, informacion en tiempo real y evaluacion ambiental. Los datos deben tener criterios de frecuencia, precision espacial, puntualidad, consistencia, linaje y versionado. Un dato de sensor sin calibracion o con desfase temporal puede inducir politicas erroneas.

Para evitar una interoperabilidad solo tecnica, hay que definir vocabularios comunes, unidades de medida, identificadores de paradas y zonas, modelos de evento, codigos de incidencia, granularidad temporal y reglas de equivalencia entre operadores. El intercambio por API sera insuficiente si cada participante entiende de forma distinta "viaje", "ocupacion", "retraso" o "disponibilidad".

**Errores frecuentes:**

- Creer que espacio de datos significa publicarlo todo.
- Olvidar la confidencialidad comercial.
- No distinguir datos personales y no personales combinables.
- No definir granularidad temporal o espacial.
- Asociar calidad solo a ausencia de errores de formato.

**Mini comprobacion:**

La respuesta correcta reconoce accesos graduados y gobernanza comun, no una apertura indiscriminada.

## 5. Muestra de test progresivo

Estas preguntas son muestra integrada en el tema. El banco completo debe ir en estructura i18n externa. Las notas de test deben presentarse separadas de la teoria y con explicacion diagnostica: concepto fallado, por que la opcion elegida no procede y donde repasar.

### Nivel 1. Base conceptual

**Pregunta 1.** En el contexto del gobierno del dato publico, un metadato es:

A. Un dato que describe otro dato, documento o conjunto de datos.
B. Un dato reservado que nunca puede publicarse.
C. Un sinonimo de dato estadistico.
D. Un formato tecnico de intercambio entre administraciones.

**Respuesta correcta:** A.

**Nota de test:** el metadato no es necesariamente secreto ni estadistico. Su funcion es describir, contextualizar y facilitar gestion, busqueda, conservacion o reutilizacion.

**Pregunta 2.** La interoperabilidad semantica se refiere principalmente a:

A. Que todos los sistemas usen la misma marca de base de datos.
B. Que los datos intercambiados tengan significado comun y reglas de interpretacion compartidas.
C. Que la transmision se haga mediante una red segura.
D. Que los documentos se publiquen en formato PDF.

**Respuesta correcta:** B.

**Nota de test:** la seguridad de red pertenece al plano tecnico y de seguridad. El formato puede ayudar, pero no garantiza significado comun.

**Pregunta 3.** La reutilizacion de informacion del sector publico implica:

A. La posibilidad de usar documentos o datos publicos bajo condiciones juridicas y tecnicas determinadas.
B. La obligacion de publicar cualquier dato en poder de una administracion.
C. La eliminacion de los limites de proteccion de datos.
D. La sustitucion del procedimiento de acceso a la informacion publica.

**Respuesta correcta:** A.

**Nota de test:** reutilizacion no equivale a apertura absoluta. Opera con limites, condiciones, formatos, licencias y regimen juridico propio.

### Nivel 2. Aplicacion

**Pregunta 4.** Una administracion publica un conjunto de datos de contratos en una hoja sin diccionario de campos, con codigos internos no explicados y sin fecha de actualizacion. El principal problema desde la perspectiva de reutilizacion es:

A. Que los contratos nunca pueden publicarse.
B. Que falta calidad y metadatos suficientes para un uso fiable.
C. Que solo puede publicarse en PDF firmado.
D. Que la reutilizacion exige siempre autorizacion individual previa de cada empresa.

**Respuesta correcta:** B.

**Nota de test:** la pregunta no niega que existan limites o datos a revisar, pero el defecto descrito es de calidad, contexto y metadatado.

**Pregunta 5.** Dos administraciones intercambian datos mediante API, pero una codifica el estado "vigente" de una autorizacion de forma distinta a la otra. ¿Que dimension falla de manera mas directa?

A. Interoperabilidad semantica.
B. Publicidad activa.
C. Accesibilidad web.
D. Firma electronica.

**Respuesta correcta:** A.

**Nota de test:** la API puede funcionar tecnicamente y aun asi transmitir un significado ambiguo o incorrecto.

**Pregunta 6.** Un catalogo de datos abiertos contiene muchos conjuntos, pero la mitad estan desactualizados y carecen de licencia. La conclusion mas adecuada es:

A. El catalogo es maduro porque el volumen de conjuntos es alto.
B. El catalogo presenta un problema de gobernanza y calidad.
C. La ausencia de licencia mejora la libertad de reutilizacion.
D. La solucion es eliminar todos los conjuntos con baja demanda.

**Respuesta correcta:** B.

**Nota de test:** el numero de datasets es un indicador insuficiente. La licencia, mantenimiento, metadatos y actualizacion son parte de la madurez real.

### Nivel 3. Examen real

**Pregunta 7.** Una administracion quiere publicar datos de beneficiarios de ayudas sociales con nombre, cuantia, municipio y finalidad. Retira el documento identificativo y sostiene que ya no hay datos personales. ¿Cual es la respuesta mas correcta?

A. Es correcto, porque sin documento identificativo desaparece cualquier riesgo.
B. Es incorrecto, porque la identificacion puede producirse por combinacion de atributos y debe analizarse finalidad, base juridica, minimizacion y riesgo.
C. Es correcto siempre que el fichero se publique en formato abierto.
D. Es incorrecto solo si el beneficiario es una empresa.

**Respuesta correcta:** B.

**Nota de test:** la anonimizacion no se reduce a eliminar identificadores directos. El contexto, la granularidad y la combinabilidad importan.

**Pregunta 8.** En un proyecto de intercambio de datos para no exigir certificados al ciudadano, ¿cual es la secuencia mas completa?

A. Crear API, abrir puerto, probar rendimiento y publicar endpoint.
B. Definir base juridica, finalidad, responsables, modelo semantico, controles de calidad, seguridad, trazabilidad y despues implementar el canal tecnico.
C. Pedir al ciudadano que siga aportando documentos para evitar riesgos.
D. Publicar todos los datos en el portal de transparencia y consumirlos desde alli.

**Respuesta correcta:** B.

**Nota de test:** la simplificacion no elimina garantias. La tecnica debe apoyarse en decisiones juridicas, organizativas y semanticas.

**Pregunta 9.** Un conjunto de datos esta completo y en formato abierto, pero mezcla codigos de municipios antiguos y nuevos sin tabla de equivalencias. ¿Que dimension de calidad queda mas afectada?

A. Consistencia y trazabilidad semantica.
B. Tamano del fichero.
C. Color corporativo del portal.
D. Existencia de firma manuscrita.

**Respuesta correcta:** A.

**Nota de test:** el dato puede estar completo y ser procesable, pero si los codigos no son interpretables de forma coherente, la reutilizacion sera defectuosa.

**Pregunta 10.** Una unidad afirma que, como los datos son "no personales", pueden compartirse sin ninguna condicion. La objecion mas adecuada es:

A. Solo los datos personales tienen limites.
B. Los datos no personales pueden estar sujetos a confidencialidad, secreto comercial, seguridad, propiedad intelectual, calidad, finalidad o condiciones de acceso.
C. Los datos no personales no existen en la administracion publica.
D. Todo dato no personal debe destruirse al finalizar el procedimiento.

**Respuesta correcta:** B.

**Nota de test:** proteccion de datos no agota los limites juridicos ni organizativos del intercambio y la reutilizacion.

## 6. Estructura propuesta para banco i18n externo

El banco completo no debe insertarse en el tema. Debe mantenerse en ficheros externos localizables y versionables. Propuesta de estructura logica para `banco_preguntas_i18n/es/`:

| Fichero sugerido | Contenido | Uso |
|---|---|---|
| `nivel_base.json` | definiciones, conceptos y distinciones iniciales | primera recuperacion activa |
| `nivel_aplicacion.json` | casos breves con decision razonada | estudio intermedio |
| `nivel_examen.json` | preguntas largas con distractores verosimiles | simulacion A1 |
| `supuestos_practicos.json` | escenarios, pistas, preguntas y soluciones | practica guiada |
| `diagnosticos.json` | explicaciones por error y lugar de repaso | tutor de fallos |
| `metadata.json` | version, tema, competencias, etiquetas y fuentes editoriales | trazabilidad |

Campos minimos recomendados por pregunta:

| Campo | Funcion |
|---|---|
| `id` | identificador estable de la pregunta |
| `nivel` | base, aplicacion o examen |
| `bloque` | gobierno, semantica, calidad, reutilizacion, seguridad, proteccion de datos |
| `competencia` | definicion, distincion, aplicacion, diagnostico, priorizacion |
| `enunciado` | pregunta localizable |
| `opciones` | opciones localizables |
| `respuesta_correcta` | clave de opcion |
| `explicacion` | razon de la respuesta correcta |
| `diagnostico_distractores` | por que falla cada distractor |
| `repasar_en` | seccion del tema para volver a estudiar |
| `fuente_editorial` | fuente normativa o doctrinal sin URL visible |
| `estado` | borrador, revisada, validada |

Ejemplo de objeto conceptual, no banco final:

```json
{
  "id": "gobdato-sem-001",
  "nivel": "base",
  "bloque": "interoperabilidad_semantica",
  "competencia": "distincion_conceptual",
  "enunciado": "La interoperabilidad semantica exige principalmente que...",
  "opciones": {
    "a": "los sistemas compartan significado y reglas de interpretacion",
    "b": "todas las conexiones usen el mismo proveedor",
    "c": "los documentos se impriman antes de archivarse",
    "d": "la informacion se publique sin metadatos"
  },
  "respuesta_correcta": "a",
  "explicacion": "La clave es compartir significado, no solo canal tecnico.",
  "diagnostico_distractores": {
    "b": "Confunde interoperabilidad con proveedor tecnologico.",
    "c": "No guarda relacion con intercambio semantico.",
    "d": "La ausencia de metadatos perjudica la comprension y reutilizacion."
  },
  "repasar_en": "Interoperabilidad semantica y dimensiones de interoperabilidad",
  "fuente_editorial": "Esquema Nacional de Interoperabilidad y Reglamento sobre la Europa Interoperable",
  "estado": "muestra"
}
```

## 7. Checklist editorial A1 especifico del tema

### 7.1. Cobertura minima de contenido

| Item | Criterio de aceptacion |
|---|---|
| Gobierno del dato | Explica estrategia, roles, responsabilidades, ciclo de vida, catalogo, metadatos, calidad y valor publico. |
| Interoperabilidad semantica | La distingue de interoperabilidad juridica, organizativa y tecnica, con ejemplos administrativos. |
| Calidad del dato | Presenta dimensiones medibles, controles, evidencias e impacto en decisiones publicas. |
| Reutilizacion | Explica regimen, condiciones, limites, licencias, formatos, catalogos y diferencia con transparencia. |
| Proteccion y seguridad | Integra minimizacion, finalidad, riesgo de reidentificacion, confidencialidad y trazabilidad sin convertir el tema en monografico de proteccion de datos. |
| Marco europeo y estatal | Situa ENI, reutilizacion de informacion publica, datos abiertos, gobernanza europea de datos, espacios de datos e interoperabilidad europea. |
| Administracion publica | Usa ejemplos de procedimientos, certificados, subvenciones, contratos, movilidad, padron, salud publica o estadistica con cautelas. |

### 7.2. Calidad pedagogica

| Item | Verificacion |
|---|---|
| Definicion antes de complejidad | Cada bloque introduce definiciones antes de casos o excepciones. |
| Una idea por parrafo | Evita parrafos excesivamente densos o encadenados por muchas subordinadas. |
| Ejemplo despues de concepto | Cada nocion abstracta relevante tiene ejemplo administrativo. |
| Modo tutor | Incluye "que significa", "por que importa", "con que se confunde" y "como reconocerlo en examen". |
| Notas de test separadas | Las trampas y distractores no invaden el desarrollo teorico. |
| Recuperacion activa | Incluye preguntas breves, supuestos y repaso final. |
| Doble codificacion | Visuales y tablas apoyan, pero no sustituyen, la explicacion. |

### 7.3. Calidad tecnica y juridica

| Item | Verificacion |
|---|---|
| No absolutiza apertura | Reconoce limites legales, tecnicos, eticos y de calidad. |
| No reduce interoperabilidad a API | Trata significado, organizacion, marco juridico y tecnica. |
| No reduce calidad a formato | Incluye exactitud, completitud, consistencia, actualidad, unicidad, validez y trazabilidad. |
| No confunde anonimizacion con supresion simple | Explica riesgo de reidentificacion y contexto. |
| No mezcla transparencia y reutilizacion | Distingue acceso, publicidad activa, reutilizacion y datos abiertos. |
| No usa fuentes privadas como base principal | Prioriza normativa y guias oficiales. |
| No muestra URLs reales | Las fuentes se citan editorialmente y se archivan aparte. |

### 7.4. HTML y visuales

| Item | Verificacion |
|---|---|
| Primera lectura activa | El HTML permite leer el tema sin interrupciones de test o tablas excesivas. |
| Visuales ocultables | Tablas, esquemas y notas pueden mostrarse u ocultarse. |
| Responsivo | Diagramas con `overflow-x` y textos legibles en movil. |
| Sin assets inexistentes | Todo SVG, CSS o JSON referenciado existe localmente. |
| Notas de test diferenciadas | Fondo azul, barra lateral y estilo separado de teoria. |
| Supuestos con solucion ocultable | El alumno puede intentar resolver antes de ver respuesta. |
| Banco externo | El HTML no incrusta el banco completo. |

### 7.5. Control de examen

| Item | Verificacion |
|---|---|
| Preguntas base | Comprueban definiciones y distinciones. |
| Preguntas de aplicacion | Obligan a decidir en escenarios breves. |
| Preguntas tipo A1 | Incluyen distractores verosimiles, no absurdos. |
| Diagnostico de errores | Cada distractor explica el concepto fallado. |
| Repaso final | Incluye mapa de 5-7 ideas, definiciones rapidas, diferencias y errores frecuentes. |
| Supuestos practicos | Incorporan pistas, resolucion paso a paso y criterio de correccion. |

## 8. Fuentes oficiales a considerar en la integracion editorial

La integracion final deberia contrastar el desarrollo y las notas de test con fuentes oficiales, citadas sin URL visible en el texto final. Relacion orientativa:

| Fuente editorial | Uso principal en el tema |
|---|---|
| Esquema Nacional de Interoperabilidad | Principios, dimensiones y condiciones de interoperabilidad en el sector publico. |
| Normas tecnicas de interoperabilidad | Documento electronico, expediente, catalogos, politica de gestion documental, reutilizacion y modelos de datos cuando proceda. |
| Reglamento de actuacion y funcionamiento del sector publico por medios electronicos | Funcionamiento electronico, intercambio, servicios y referencias al marco de interoperabilidad. |
| Ley de reutilizacion de la informacion del sector publico | Regimen basico de reutilizacion, ambito, condiciones y limites. |
| Directiva europea de datos abiertos y reutilizacion de la informacion del sector publico | Marco europeo de apertura y datos de alto valor. |
| Reglamento de Gobernanza de Datos | Intermediacion, altruismo de datos, reutilizacion de categorias protegidas y gobernanza europea. |
| Reglamento de Datos | Acceso y uso justo de datos, disponibilidad para organismos publicos en supuestos de necesidad excepcional e interoperabilidad. |
| Reglamento sobre la Europa Interoperable | Cooperacion e interoperabilidad transfronteriza del sector publico europeo. |
| Reglamento general de proteccion de datos y normativa organica espanola | Finalidad, minimizacion, base juridica, derechos, riesgos y garantias. |
| Esquema Nacional de Seguridad | Seguridad, trazabilidad, control de acceso y proteccion de servicios e informacion. |
| Portal nacional de datos abiertos y guias oficiales de datos | Catalogacion, formatos, metadatos, licencias y buenas practicas de publicacion. |

## 9. Plan de integracion sugerido

1. Insertar el Visual 1 al inicio del tema como mapa general.
2. Usar el Visual 2 en el bloque de definiciones.
3. Usar el Visual 3 en el bloque de interoperabilidad semantica.
4. Usar el Visual 5 en calidad del dato y recuperarlo en supuestos.
5. Usar el Visual 6 antes de reutilizacion o como cierre aplicado.
6. Distribuir los supuestos al final de bloques teoricos, no todos juntos si rompen la primera lectura.
7. Mantener la muestra de test como seccion separada, con notas diferenciadas.
8. Crear el banco completo fuera del tema con campos de diagnostico por distractor.
9. Revisar que el HTML final no muestre URLs reales ni referencias internas de trabajo.

## 10. Resumen para integracion

El bloque mas importante para este tema es la distincion entre **compartir datos**, **hacer interoperables los sistemas**, **dar significado comun**, **garantizar calidad** y **permitir reutilizacion bajo condiciones**. Los visuales deben reforzar esas separaciones. Los supuestos practicos deben evitar soluciones simples y entrenar ponderacion juridica, semantica y operativa. El test progresivo debe empezar por definiciones, avanzar a casos breves y terminar en preguntas A1 con distractores plausibles.

El cierre editorial debe verificar que el tema no transmite una idea ingenua de apertura total ni una vision puramente tecnica del dato. El enfoque correcto es institucional: datos gobernados, interoperables, de calidad, seguros y reutilizables cuando el marco juridico y la finalidad lo permiten.
