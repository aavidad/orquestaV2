# Material parcial: interoperabilidad semántica en la Administración pública

## Alcance del bloque

Este material desarrolla la interoperabilidad semántica dentro del gobierno del dato público. Su función es servir como bloque integrable en un tema A1 sobre gobierno del dato, interoperabilidad, calidad y reutilización de la información pública. El enfoque combina el marco jurídico-técnico español, la evolución europea de los metadatos de datos abiertos y las prácticas necesarias para que varias Administraciones puedan intercambiar, publicar, enlazar y reutilizar información con significado común.

La idea central es sencilla: no basta con que dos sistemas se conecten. Tampoco basta con que usen un formato técnico válido. La interoperabilidad semántica exige que los datos intercambiados signifiquen lo mismo para quien los emite, quien los recibe, quien los transforma y quien los reutiliza. En el sector público esto resulta decisivo porque una misma palabra puede tener efectos jurídicos, presupuestarios, estadísticos o administrativos distintos según el procedimiento, el órgano competente o la norma aplicable.

Fuentes oficiales de referencia consultadas para este bloque: texto consolidado del Esquema Nacional de Interoperabilidad; Norma Técnica de Interoperabilidad de Relación de modelos de datos; Norma Técnica de Interoperabilidad de Reutilización de recursos de información; materiales oficiales de datos.gob.es sobre DCAT-AP-ES; especificación europea DCAT-AP 3.0.1; marco europeo de datos abiertos, reutilización de la información pública y conjuntos de datos de alto valor.

## Mapa conceptual textual

Interoperabilidad pública
-> dimensión organizativa: acuerdos, competencias, responsabilidades y condiciones de acceso.
-> dimensión técnica: estándares, formatos, redes, servicios, interfaces y seguridad.
-> dimensión semántica: significado común de datos, metadatos, conceptos, códigos y relaciones.

Interoperabilidad semántica
-> modelos de datos de intercambio: estructura, entidades, atributos y reglas de validación.
-> definiciones: significado preciso de cada entidad, atributo y valor.
-> codificaciones: catálogos de valores autorizados para representar conceptos.
-> vocabularios controlados: listas, taxonomías, tesauros y esquemas de conceptos.
-> ontologías: clases, propiedades y relaciones formales entre conceptos.
-> identificadores persistentes: referencias estables para recursos, organismos, procedimientos, conjuntos de datos y conceptos.
-> equivalencias: correspondencias entre vocabularios, modelos o códigos distintos.
-> metadatos: descripción normalizada de catálogos, conjuntos de datos, distribuciones, servicios y condiciones de reutilización.

Marco español
-> ENI: reconoce la interoperabilidad semántica como dimensión necesaria y regula activos semánticos.
-> NTI de Relación de modelos de datos: fija condiciones para publicar modelos comunes y modelos de materias sujetas a intercambio.
-> Centro de Interoperabilidad Semántica: repositorio y punto de publicación de modelos, definiciones y codificaciones.
-> NTI de Reutilización de recursos de información: establece criterios de identificación, descripción, formatos, condiciones de uso y puesta a disposición de recursos reutilizables.

Marco europeo y datos abiertos
-> DCAT: vocabulario de catálogo de datos.
-> DCAT-AP: perfil europeo para describir catálogos, conjuntos de datos, distribuciones y servicios de datos.
-> DCAT-AP-ES: adaptación española prevista como modelo de referencia para catálogos públicos alineados con estándares europeos.
-> Conjuntos de datos de alto valor: refuerzan la necesidad de metadatos homogéneos, formatos reutilizables y servicios de datos interoperables.

Resultado esperado
-> menos ambigüedad.
-> mayor automatización.
-> más calidad del dato.
-> mejor reutilización.
-> trazabilidad y conservación del significado a lo largo del tiempo.

## Definiciones clave

| Concepto | Definición operativa | Por qué importa en examen |
|---|---|---|
| Interoperabilidad semántica | Capacidad de intercambiar información conservando el significado de los datos, sus relaciones y sus condiciones de uso. | Suele confundirse con interoperabilidad técnica. La técnica conecta sistemas; la semántica alinea significados. |
| Modelo de datos | Representación estructurada de entidades, atributos, relaciones, tipos de dato, restricciones y codificaciones aplicables a un intercambio o dominio. | En el ENI no es un documento decorativo: es un activo semántico que orienta intercambios administrativos. |
| Modelo común | Modelo de datos de aplicación preferente para intercambios públicos cuando tiene carácter común. | La palabra "preferente" es relevante: no equivale siempre a obligatorio. |
| Modelo de titular competente | Modelo publicado por un órgano titular de una competencia o de una infraestructura común. | En los intercambios de su materia puede tener aplicación obligatoria. |
| Vocabulario controlado | Conjunto gobernado de términos autorizados y, en su caso, relaciones entre ellos. | Evita que varias Administraciones llamen de forma distinta a la misma cosa o igual a cosas distintas. |
| Codificación | Asignación de códigos normalizados a valores de un dominio. | Permite validación automática, intercambio sin ambigüedad y agregación estadística. |
| Taxonomía | Organización jerárquica de conceptos. | Ayuda a clasificar información y a navegar por niveles de generalidad. |
| Tesauro | Vocabulario controlado con relaciones de equivalencia, jerarquía y asociación. | Es más rico que una lista plana, pero no necesariamente tan formal como una ontología. |
| Ontología | Modelo formal de clases, propiedades y relaciones de un dominio, interpretable por máquinas. | Es útil cuando se necesita razonamiento, integración compleja o datos enlazados. |
| RDF | Modelo de representación basado en tripletas sujeto-predicado-objeto. | Aparece en datos abiertos, DCAT, SKOS y catálogos semánticos. |
| SKOS | Modelo para representar sistemas de organización del conocimiento, como tesauros y taxonomías. | Es una pieza habitual para temas, códigos y vocabularios de clasificación. |
| DCAT | Vocabulario para describir catálogos de datos. | Sirve de base a DCAT-AP y a los perfiles nacionales. |
| DCAT-AP | Perfil europeo de DCAT para describir catálogos y conjuntos de datos del sector público. | La clave es la federación: catálogos distintos pueden agregarse y buscarse de forma homogénea. |
| DCAT-AP-ES | Perfil español alineado con DCAT-AP para describir recursos de información pública. | En España se vincula a la modernización de la NTI de reutilización. |
| Identificador persistente | Identificador estable, único y gobernado para un recurso o concepto. | Sin persistencia no hay enlace fiable ni reutilización madura. |
| Equivalencia semántica | Relación entre dos conceptos o códigos de distintos vocabularios que representan lo mismo o algo muy próximo. | En examen conviene distinguir equivalencia exacta, aproximada y jerárquica. |
| Distribución | Forma concreta en que se ofrece un conjunto de datos, con un formato, acceso y condiciones determinadas. | Un mismo conjunto puede tener varias distribuciones: CSV, JSON, API o RDF, por ejemplo. |

## 1. Sentido de la interoperabilidad semántica en el ENI

El Esquema Nacional de Interoperabilidad entiende la interoperabilidad como una cualidad integral, no como una actuación puntual al final de un proyecto. Debe estar presente desde la concepción del servicio y mantenerse durante todo su ciclo de vida: planificación, diseño, contratación, construcción, despliegue, explotación, publicación, conservación y acceso. En materia semántica esto significa que las decisiones sobre nombres de campos, códigos, metadatos, definiciones y vocabularios deben tomarse antes de desplegar el intercambio y no cuando ya existen decenas de integraciones incompatibles.

El ENI distingue tres dimensiones: organizativa, semántica y técnica. La dimensión semántica se ocupa de que la información conserve su significado al cruzar fronteras administrativas o tecnológicas. Un servicio de consulta de datos padronales, una plataforma de intercambio de expedientes, un portal de datos abiertos o un catálogo de procedimientos pueden funcionar técnicamente y, sin embargo, fracasar semánticamente si los conceptos que intercambian no están definidos de forma común.

Ejemplo: si una Administración codifica "estado del expediente" con valores "abierto", "en curso" y "resuelto", mientras otra usa "iniciado", "tramitación", "subsanación", "terminado" y "archivado", la conexión técnica no resuelve el problema. Hace falta una tabla de equivalencias, una definición común de cada estado y reglas sobre qué valores pueden convertirse sin pérdida. En caso contrario, un cuadro de mando agregará expedientes heterogéneos y ofrecerá conclusiones erróneas.

La interoperabilidad semántica no solo sirve para la relación entre Administraciones. También protege a la ciudadanía y a los reutilizadores. Cuando un catálogo público describe de forma homogénea sus datos, una persona puede encontrar información, comparar fuentes y reutilizarla con menor coste. Cuando los datos se exponen con metadatos normalizados, un sistema puede procesarlos automáticamente. Cuando los códigos son persistentes, las referencias no se rompen cada vez que cambia una web o una aplicación.

El ENI regula los activos semánticos en su artículo dedicado a modelos de datos. En síntesis, establece que deben mantenerse modelos de datos de intercambio comunes, que los órganos titulares de competencias deben publicar sus modelos cuando afecten a intercambios con ciudadanía u otras Administraciones, y que esos modelos deben acompañarse de definiciones y codificaciones asociadas. La consecuencia práctica es que una Administración no debería inventar de forma aislada un modelo para un dominio ya cubierto por un modelo común o por un modelo sectorial aplicable.

### Modo tutor

Qué significa: la interoperabilidad semántica es el acuerdo sobre el significado, no solo sobre el canal. Si se intercambia "fecha de alta", debe estar claro si es fecha de solicitud, fecha de registro, fecha de efectos, fecha contable o fecha de publicación.

Por qué importa: muchas integraciones fallan por ambigüedad, no por falta de API. En el sector público esa ambigüedad puede producir errores de tramitación, indicadores falsos o reutilizaciones jurídicamente problemáticas.

Con qué se confunde: se confunde con usar XML, JSON, CSV o RDF. Esos formatos son medios técnicos; la semántica está en la definición del dato, sus valores, sus relaciones y su contexto.

Cómo se reconoce en examen: las palabras señal son "modelo de datos", "definiciones", "codificaciones", "vocabularios", "significado", "activos semánticos", "Centro de Interoperabilidad Semántica", "DCAT", "SKOS" y "metadatos".

## 2. Modelos de datos: núcleo de la semántica administrativa

Un modelo de datos público no debe limitarse a una lista de campos. Debe explicar qué representa cada entidad, qué atributos la describen, qué tipos de dato se admiten, qué valores están permitidos, qué relaciones existen entre entidades y qué reglas de validación son aplicables. En una organización madura, el modelo también indica su versión, responsable, ámbito de aplicación, estado de vigencia y relación con modelos anteriores.

La Norma Técnica de Interoperabilidad de Relación de modelos de datos concreta el mandato del ENI. Su objetivo es definir condiciones para establecer y publicar modelos de datos que tengan carácter común en la Administración y modelos referidos a materias sujetas a intercambio de información con la ciudadanía u otras Administraciones. También incluye las definiciones y codificaciones asociadas, de cara a su publicación en el Centro de Interoperabilidad Semántica.

La distinción entre tipos de modelo es importante. Los modelos comunes son de aplicación preferente en los intercambios de información. Los modelos publicados por titulares de competencias en materias sujetas a intercambio, o vinculados a infraestructuras, servicios y herramientas comunes, pueden ser de aplicación obligatoria dentro de su ámbito. Esta diferencia permite responder preguntas de examen que juegan con los términos "preferente" y "obligatorio".

El ciclo de vida de un modelo de datos debería incluir al menos seis fases. Primera, identificación del dominio: qué procedimiento, servicio, conjunto de datos o intercambio se quiere modelar. Segunda, inventario de conceptos existentes: normas, formularios, bases de datos, modelos comunes y vocabularios ya publicados. Tercera, definición del modelo: entidades, atributos, relaciones y reglas. Cuarta, vinculación con codificaciones y vocabularios. Quinta, publicación en el repositorio semántico correspondiente. Sexta, mantenimiento: versionado, cambios, equivalencias y comunicación a las partes afectadas.

| Elemento del modelo | Pregunta que responde | Ejemplo en Administración pública | Riesgo si falta |
|---|---|---|---|
| Entidad | ¿Qué objeto se describe? | Procedimiento, órgano, oficina, expediente, subvención, contrato, conjunto de datos. | Se mezclan realidades distintas en la misma tabla o API. |
| Atributo | ¿Qué dato caracteriza a la entidad? | Fecha de resolución, código DIR3, estado, importe concedido, idioma. | Los campos se interpretan de forma distinta según el sistema. |
| Tipo de dato | ¿Cómo se representa técnicamente? | Fecha, número decimal, texto, booleano, lista codificada. | Fallos de validación, ordenación o agregación. |
| Definición | ¿Qué significa exactamente? | "Fecha de efectos" no es lo mismo que "fecha de firma". | Ambigüedad jurídica y estadística. |
| Codificación | ¿Qué valores son válidos? | Códigos de órgano, códigos de idioma, códigos territoriales. | Valores libres, duplicados y difíciles de integrar. |
| Relación | ¿Cómo se conecta con otras entidades? | Un órgano publica varios procedimientos; un conjunto tiene varias distribuciones. | Pérdida de contexto y navegación deficiente. |
| Regla de validación | ¿Qué condiciones debe cumplir? | El importe debe ser no negativo; una fecha de fin no puede preceder a una fecha de inicio. | Datos formalmente presentes pero incorrectos. |
| Versión | ¿Qué edición del modelo se aplica? | Modelo de subvenciones versión 2.0 frente a versión 1.3. | Integraciones rotas o resultados no comparables. |

### Ejemplo trabajado: modelo de datos de subvenciones

Supongamos un intercambio entre una comunidad autónoma y varios ayuntamientos para publicar subvenciones concedidas. Un modelo semántico mínimo debería distinguir convocatoria, beneficiario, concesión, órgano concedente, línea de ayuda, base reguladora, importe solicitado, importe concedido, fecha de concesión y finalidad. Además, debería definir qué significa "beneficiario" en el contexto de la publicación, cómo se representa una persona jurídica, qué datos personales quedan excluidos o agregados, y qué clasificación temática se usa.

Si cada ayuntamiento publica "ayuda", "beca", "subvención" o "aportación" sin definición común, el portal autonómico podrá mostrar listados, pero no podrá ofrecer análisis fiable. Si se usa una taxonomía común de materias y códigos de órgano normalizados, la información puede agregarse por política pública, entidad concedente, ámbito territorial y periodo.

## 3. Vocabularios controlados y codificaciones

Los vocabularios controlados reducen la variabilidad del lenguaje natural. En la Administración pública son esenciales porque los procedimientos se tramitan con efectos jurídicos y porque muchos datos se agregan entre niveles estatal, autonómico, provincial, insular, comarcal y local. Una misma categoría debe poder reconocerse aunque proceda de sistemas distintos.

Una lista de valores es el vocabulario más simple. Por ejemplo, un campo "idioma" puede admitir un conjunto de etiquetas normalizadas. Una taxonomía añade jerarquía: una materia general puede tener submaterias. Un tesauro incorpora relaciones más ricas, como términos equivalentes, más amplios, más específicos o relacionados. Una ontología formaliza clases, propiedades y restricciones, lo que permite expresar relaciones complejas y, en ciertos casos, inferir conocimiento.

Las codificaciones son la parte más operativa del vocabulario. Un código estable evita que los cambios de denominación rompan la integración. Por ejemplo, una unidad administrativa puede cambiar de nombre sin que deba cambiar su identificador. En ese caso, el histórico de denominaciones se gestiona como metadato, no como sustituto del código.

| Recurso semántico | Uso principal | Ejemplo administrativo | Buena práctica |
|---|---|---|---|
| Lista controlada | Validar valores cerrados. | Estado de una publicación: borrador, publicado, retirado. | Definir cada valor y evitar sinónimos no gobernados. |
| Taxonomía | Clasificar por categorías jerárquicas. | Sectores de datos abiertos o materias de procedimientos. | Mantener códigos estables aunque cambien etiquetas. |
| Tesauro | Relacionar términos equivalentes o asociados. | Materias documentales y términos de búsqueda. | Distinguir sinónimo, concepto relacionado y concepto más específico. |
| Ontología | Modelar relaciones formales complejas. | Contratación, territorio, servicios, organizaciones y eventos. | Usarla cuando aporte integración real, no por complejidad aparente. |
| Directorio común | Identificar órganos, unidades y oficinas. | Código único de una unidad tramitadora. | No sustituir el código por el nombre visible. |
| Codificación estadística | Agregar información de forma homogénea. | Clasificaciones territoriales, económicas o demográficas. | Usar la fuente competente y documentar versión. |
| Etiqueta lingüística | Identificar idioma del contenido. | Español, catalán, gallego, euskera o inglés. | Usar etiquetas normalizadas, no descripciones libres. |

### Criterios para elegir un vocabulario

El primer criterio es autoridad. Debe preferirse el vocabulario oficial o sectorial aplicable antes que crear uno local. El segundo criterio es estabilidad. Un vocabulario que cambia sin versión ni equivalencias genera deuda semántica. El tercer criterio es granularidad. Un catálogo demasiado grueso no permite análisis; uno excesivamente detallado dificulta la aplicación práctica. El cuarto criterio es mantenibilidad. Debe existir una unidad responsable, un procedimiento de alta, modificación y baja, y una forma de comunicar cambios.

El quinto criterio es interoperabilidad europea. En dominios como datos abiertos, contratación, estadística, información geográfica o conjuntos de alto valor, conviene alinear el vocabulario local con perfiles europeos. Esa alineación no significa traducir mecánicamente todos los campos, sino mapear conceptos, conservar identificadores y documentar diferencias.

### Errores habituales con vocabularios

Un error frecuente es usar campos de texto libre para valores que deberían estar codificados. Otro error es considerar que una etiqueta visible es suficiente. La etiqueta puede variar por idioma, reforma organizativa o criterio editorial; el código debe seguir siendo estable. También es habitual crear un vocabulario propio cuando ya existe un estándar aplicable. Esto aumenta el coste de integración y obliga a mantener equivalencias innecesarias.

## 4. DCAT, DCAT-AP y DCAT-AP-ES

DCAT es un vocabulario para describir catálogos de datos. Su utilidad está en que ofrece una estructura común para representar catálogos, conjuntos de datos, distribuciones, servicios de datos, responsables, licencias, temas, cobertura temporal, cobertura espacial y otros metadatos. DCAT no obliga a que todos los portales sean iguales, pero proporciona un lenguaje común para describirlos.

DCAT-AP es el perfil europeo de aplicación de DCAT. Un perfil de aplicación toma un vocabulario general y concreta su uso para un contexto. En este caso, el contexto son los portales europeos de datos abiertos y la federación de catálogos. DCAT-AP actúa como filtro y como guía: selecciona propiedades relevantes, establece restricciones, recomienda clases y facilita que los catálogos nacionales, regionales y locales puedan agregarse.

DCAT-AP-ES es la adaptación española alineada con el marco europeo. Los materiales oficiales de datos.gob.es indican que la actualización de la norma española de reutilización incorpora DCAT-AP-ES como modelo de referencia para describir conjuntos y servicios de datos, con el objetivo de mejorar la interoperabilidad entre catálogos nacionales y europeos. Para una oposición A1 conviene formularlo con precisión: la NTI de reutilización vigente de 2013 ya se apoya en DCAT y tecnologías de web semántica; la modernización hacia DCAT-AP-ES refuerza la alineación con los perfiles europeos más recientes y con los conjuntos de datos de alto valor.

| Nivel | Función | Entidades típicas | Idea de examen |
|---|---|---|---|
| DCAT | Vocabulario base para catálogos de datos. | Catálogo, conjunto de datos, distribución, servicio de datos. | Es el vocabulario de partida. |
| DCAT-AP | Perfil europeo de DCAT. | Catálogos federables, datasets, distribuciones, servicios, responsables, licencias. | Concreta DCAT para portales europeos. |
| DCAT-AP-ES | Perfil español alineado con DCAT-AP. | Catálogos y recursos públicos en el contexto nacional. | Adapta el perfil al marco español y a la NTI de reutilización. |
| NTI de reutilización | Norma española de puesta a disposición de recursos reutilizables. | Metadatos mínimos, formatos, condiciones de uso, identificación. | No es solo datos abiertos: regula pautas básicas de reutilización. |

### Metadatos esenciales en catálogos reutilizables

Un catálogo público interoperable necesita al menos un título, una descripción, un órgano publicador, fechas de publicación y actualización, idioma, cobertura temática, condiciones de uso y relación con los conjuntos de datos incluidos. Cada conjunto de datos necesita a su vez título, descripción, tema, identificador, fechas, responsable, condiciones, cobertura, frecuencia de actualización y distribuciones. Cada distribución añade formato, acceso, tamaño, disponibilidad, licencia y condiciones técnicas.

La clave no es memorizar una tabla cerrada, sino comprender la lógica: el catálogo agrupa; el conjunto de datos describe una colección lógica de información; la distribución indica una forma concreta de acceso o descarga; el servicio de datos permite acceso dinámico o consulta. Un mismo conjunto puede tener una distribución CSV para descarga masiva, una API para consulta y una representación RDF para integración semántica.

### Ejemplo trabajado: catálogo municipal de movilidad

Un ayuntamiento publica datos de aparcamientos, cortes de tráfico y estaciones de bicicleta pública. Sin DCAT o DCAT-AP, cada recurso podría aparecer con nombres, formatos y descripciones heterogéneos. Con un perfil común, el catálogo declara el órgano publicador, los temas, la cobertura territorial, la licencia y la actualización. Cada conjunto de datos tiene identificador persistente, descripción, periodicidad y distribuciones. Una plataforma autonómica o estatal puede recolectar esos metadatos y mostrar el recurso junto a otros municipios.

El valor semántico aparece cuando "estación de bicicleta", "aparcamiento disuasorio" o "incidencia de tráfico" se relacionan con vocabularios comunes y cuando los formatos y campos no dependen de convenciones locales no documentadas. Así, una empresa, universidad o unidad de planificación puede reutilizar datos de varios municipios sin construir un traductor diferente para cada portal.

## 5. Ontologías, RDF y datos enlazados

Las ontologías permiten describir un dominio mediante clases, propiedades y relaciones. En Administración pública pueden emplearse para representar organizaciones, procedimientos, servicios, documentos, expedientes, contratos, subvenciones, territorio o recursos de información. Su valor aumenta cuando los datos proceden de varias fuentes y se necesita integrar significado, no solo columnas.

RDF expresa información mediante tripletas. Una tripleta une un sujeto, un predicado y un objeto. Por ejemplo, un conjunto de datos puede tener como sujeto el recurso "catálogo de subvenciones", como predicado "tiene tema" y como objeto el concepto "economía y hacienda" de una taxonomía. Si todos los elementos están identificados de forma persistente, otros sistemas pueden enlazar, consultar y combinar esa información.

SKOS se usa para representar vocabularios controlados. Permite declarar conceptos, etiquetas preferentes, etiquetas alternativas, conceptos más amplios, conceptos más específicos y relaciones asociativas. Esto resulta muy útil en portales de datos abiertos, archivos administrativos y sistemas de búsqueda porque permite que un usuario encuentre información aunque use un término equivalente o una denominación anterior.

OWL añade capacidad expresiva para ontologías más formales. Puede ser apropiado cuando se necesitan restricciones y razonamiento más complejo. Sin embargo, en un proyecto público debe evitarse el exceso de sofisticación. No todo vocabulario necesita OWL. La regla práctica es seleccionar el nivel semántico que resuelve el problema: lista controlada para valores simples, SKOS para conceptos gobernados y ontología formal cuando hay relaciones complejas y una necesidad real de inferencia o integración avanzada.

| Necesidad | Solución proporcionada | Ejemplo |
|---|---|---|
| Evitar valores libres inconsistentes | Lista controlada | Estado de publicación de un recurso. |
| Clasificar recursos por materias | Taxonomía | Sectores de datos abiertos. |
| Gestionar sinónimos y relaciones | SKOS | Tesauro de materias administrativas. |
| Integrar relaciones complejas | Ontología | Relación entre órgano, competencia, procedimiento, servicio y recurso. |
| Publicar metadatos de catálogos | DCAT/DCAT-AP | Catálogo de datos abiertos federable. |
| Enlazar recursos persistentes | RDF e identificadores | Conectar datasets, órganos, territorios y normativa. |

### Modo tutor

Qué significa: una ontología no es una base de datos. Es una descripción formal del significado de un dominio. Puede usarse junto con bases de datos, APIs o catálogos, pero no los sustituye.

Por qué importa: en examen se tiende a identificar semántica con tecnología compleja. Lo correcto es explicar que ontologías, RDF y SKOS son herramientas posibles dentro de una política semántica más amplia.

Con qué se confunde: se confunde un vocabulario controlado con una ontología. Una lista de valores puede ser suficiente para un campo. Una ontología se reserva para relaciones más elaboradas.

Cómo se reconoce: si el caso plantea clases, propiedades, relaciones, equivalencias y datos enlazados, la respuesta apunta a ontologías/RDF/SKOS. Si solo plantea valores permitidos, basta hablar de codificación o lista controlada.

## 6. Identificadores persistentes

Los identificadores persistentes son una condición práctica para la reutilización. Un identificador debe ser único, estable, comprensible dentro de un patrón gobernado y, cuando proceda, resoluble. La NTI de reutilización ya insistía en la identificación común y persistente de los recursos publicados. La lógica es clara: si un conjunto de datos, un concepto, un órgano o una distribución cambia de identificador cada vez que se rediseña un portal, todos los enlaces externos, referencias y procesos automáticos quedan dañados.

Un buen identificador no debe depender de detalles de implementación. No conviene incrustar tecnología, nombres de aplicaciones, extensiones de archivo o rutas temporales. Tampoco debe incluir información volátil, como el nombre político de un departamento que puede cambiar con una reforma administrativa. Lo estable debe ser el código o patrón de identificación; las denominaciones y adscripciones se gestionan como metadatos históricos.

| Recurso que identificar | Identificador recomendable | Observación |
|---|---|---|
| Órgano o unidad | Código oficial de directorio común. | El nombre puede cambiar; el código mantiene la referencia. |
| Procedimiento | Código del inventario administrativo aplicable. | Permite conectar sede, tramitación, estadísticas y catálogos. |
| Conjunto de datos | Identificador persistente de catálogo. | Debe sobrevivir a cambios de portal o rediseño visual. |
| Distribución | Identificador específico de la forma de acceso. | Una distribución CSV y una API no son el mismo recurso. |
| Concepto de vocabulario | Identificador del concepto, no solo etiqueta. | Las etiquetas pueden estar en varios idiomas o cambiar. |
| Versión de modelo | Identificador de versión. | Permite reproducir intercambios antiguos. |

### Ejemplo: cambio de denominación de una unidad

Una unidad administrativa llamada "Dirección General de Transformación Digital" pasa a denominarse "Dirección General de Administración Digital". Si los datos históricos usaban la denominación como clave, las consultas antiguas y nuevas no agregan correctamente. Si se emplea un código estable de órgano y se registra la denominación vigente en cada periodo, el análisis conserva continuidad y permite explicar el cambio.

## 7. Equivalencias y mapeos semánticos

En la práctica, las Administraciones ya tienen sistemas, formularios y vocabularios propios. La interoperabilidad semántica no siempre consiste en imponer un único vocabulario desde el primer día. Muchas veces consiste en construir equivalencias controladas entre modelos existentes y un modelo común.

Una equivalencia exacta significa que dos conceptos pueden tratarse como el mismo en el contexto definido. Una equivalencia aproximada indica que los conceptos son parecidos, pero no intercambiables sin cautela. Una relación jerárquica indica que un concepto es más general o más específico que otro. Una relación asociativa indica que existe conexión, pero no identidad.

| Tipo de relación | Significado | Ejemplo administrativo | Precaución |
|---|---|---|---|
| Equivalencia exacta | Dos valores representan el mismo concepto. | "Publicado" en dos catálogos con igual definición. | Verificar que las reglas de negocio coinciden. |
| Equivalencia aproximada | Los conceptos son próximos pero no idénticos. | "En tramitación" y "en curso". | No agregar sin nota metodológica. |
| Más específico | Un concepto detalla otro. | "Ayuda al alquiler joven" dentro de "vivienda". | La agregación hacia arriba suele ser posible; la desagregación no. |
| Más general | Un concepto cubre varios más concretos. | "Servicios sociales" frente a "dependencia" y "familia". | Puede perderse granularidad. |
| Relación asociada | Conceptos conectados sin jerarquía. | "Licencia urbanística" y "planeamiento". | No debe tratarse como equivalencia. |

### Ejemplo: estados de expediente

Sistema A: iniciado, subsanación, resolución, cerrado.

Sistema B: abierto, pendiente de documentación, resuelto, archivado.

Un mapeo apresurado podría decir que "cerrado" equivale a "archivado". Pero no siempre es cierto. Un expediente cerrado puede estar terminado por resolución favorable, desistimiento, caducidad o archivo. La equivalencia correcta exige revisar la definición jurídica y funcional de cada estado. Puede que "cerrado" sea un concepto más general que agrupa varios estados del otro sistema.

## 8. Codificaciones, formatos y valores normalizados

La interoperabilidad semántica necesita codificaciones consistentes y formatos procesables. Algunas reglas son muy prácticas:

- Las fechas deben expresarse con un formato normalizado y, cuando sea necesario, con zona horaria.
- Los importes deben distinguir valor, moneda, precisión y regla de redondeo.
- Los idiomas deben indicarse con etiquetas normalizadas.
- Los formatos de distribución deben identificarse de forma uniforme.
- Los territorios, órganos y procedimientos deben usar códigos de fuente competente.
- Las listas de valores deben publicarse con versión y fecha de vigencia.
- Las bajas o sustituciones no deben borrar el histórico.

| Dato | Mala práctica | Buena práctica |
|---|---|---|
| Fecha | "3/4/26" sin contexto. | Fecha normalizada con significado definido: publicación, efecto, modificación o captura. |
| Idioma | "castellano", "español", "ESP" mezclados. | Etiqueta normalizada y repetible para contenido multilingüe. |
| Órgano | Nombre libre del departamento. | Código oficial y denominación como atributo. |
| Estado | Texto libre introducido por cada gestor. | Lista controlada con definición de cada estado. |
| Materia | Etiquetas espontáneas. | Taxonomía o tesauro común, con etiquetas alternativas si procede. |
| Formato | "Excel" o "fichero" sin precisión. | Tipo de medio o formato especificado de forma normalizada. |
| Licencia | Texto copiado sin referencia estable. | Condición de uso identificada y reutilizable. |

## 9. Supuesto práctico guiado

### Situación

Una diputación provincial quiere federar en un único catálogo los conjuntos de datos abiertos de veinte ayuntamientos. Cada ayuntamiento publica recursos sobre presupuesto, contratación, subvenciones, movilidad, urbanismo y agenda institucional. Algunos usan CSV, otros hojas de cálculo, otros APIs y otros páginas HTML. Las denominaciones de órganos, materias, estados y licencias no coinciden.

### Pistas del enunciado

El caso menciona federación de catálogos, reutilización, datos abiertos, metadatos heterogéneos, materias distintas y necesidad de búsqueda común. Esto apunta a interoperabilidad semántica, DCAT-AP, vocabularios controlados, codificaciones y gobierno de identificadores.

### Preguntas que deberían formularse

1. ¿Qué perfil de metadatos se usará para describir catálogos, conjuntos de datos, distribuciones y servicios?
2. ¿Qué identificador persistente tendrá cada catálogo, dataset y distribución?
3. ¿Qué vocabulario de materias se aplicará?
4. ¿Cómo se identificarán los órganos publicadores?
5. ¿Qué formatos se admitirán y cómo se describirán?
6. ¿Cómo se mapearán las materias locales ya existentes?
7. ¿Qué reglas se aplicarán a actualizaciones, bajas y versiones?
8. ¿Cómo se documentarán licencias, cobertura temporal, cobertura geográfica y frecuencia?

### Resolución paso a paso

Primero, se aprueba un perfil común de metadatos alineado con DCAT-AP y con la adaptación española aplicable. No se exige que todos los ayuntamientos cambien sus sistemas internos de inmediato, pero sí que publiquen o proporcionen metadatos con una estructura común.

Segundo, se define una política de identificadores persistentes. Cada catálogo municipal, conjunto de datos y distribución tendrá una referencia estable. Si cambia el portal o el formato visual, el identificador del recurso lógico no debe cambiar.

Tercero, se adopta una taxonomía de materias. Las etiquetas locales se mapean contra esa taxonomía. Cuando una equivalencia no sea exacta, se marca como aproximada y se evita usarla para indicadores críticos sin revisión.

Cuarto, se normaliza el órgano publicador mediante códigos oficiales. La denominación visible se conserva como metadato, pero no se usa como clave primaria de integración.

Quinto, se documentan las distribuciones. Un dataset de presupuesto puede tener una distribución CSV, otra en hoja de cálculo y una API. Cada distribución se describe con formato, acceso, fecha de actualización y condiciones.

Sexto, se crea un proceso de calidad semántica. Antes de federar un recurso, se valida que tenga título, descripción, publicador, materia, licencia, identificador, fecha de actualización y al menos una distribución útil. Además, se revisan códigos no reconocidos, materias demasiado genéricas y duplicidades.

Séptimo, se establece mantenimiento. Los ayuntamientos notifican altas, bajas y cambios de vocabulario. La diputación publica equivalencias y versiones para que los reutilizadores sepan cómo interpretar series históricas.

### Errores a evitar

No debe resolverse el problema creando solo una interfaz común. La interfaz permite acceder, pero no garantiza que los datos signifiquen lo mismo. Tampoco debe sustituirse la política semántica por una hoja de cálculo manual sin gobierno. Esa solución puede servir para una carga inicial, pero no para mantenimiento estable. Otro error sería borrar las categorías locales sin conservar equivalencias, porque se perdería trazabilidad histórica.

### Mini comprobación

Una respuesta correcta debe mencionar al menos: ENI, dimensión semántica, modelos de datos, vocabularios controlados, identificadores persistentes, DCAT-AP o DCAT-AP-ES, metadatos de catálogo/dataset/distribución, codificación de órganos y gestión de equivalencias.

## 10. Ejemplos breves en Administración pública

### Ejemplo 1: procedimiento administrativo

Un procedimiento tiene código, título, órgano responsable, nivel de administración, materia, forma de inicio, plazo de resolución, silencio administrativo, normativa aplicable y canales de tramitación. Si estos datos se representan con modelos y códigos comunes, pueden reutilizarse en sedes electrónicas, buscadores, asistentes de tramitación, cuadros de mando y portales de transparencia.

El punto semántico crítico es que "plazo de resolución" no es solo un número. Debe saberse unidad de medida, momento inicial de cómputo, excepciones y efectos. Si un sistema interpreta el plazo desde la presentación y otro desde la entrada en registro del órgano competente, el dato agregado será engañoso.

### Ejemplo 2: datos territoriales

Una política pública puede analizarse por municipio, provincia, comunidad autónoma o ámbito supramunicipal. Para interoperar se necesitan códigos territoriales oficiales, versiones y reglas sobre cambios de límites o denominaciones. Si un municipio se segrega, fusiona o cambia de nombre, la serie temporal debe preservar la interpretación de los datos anteriores.

### Ejemplo 3: contratación pública

En contratación, conceptos como órgano de contratación, licitador, adjudicatario, lote, procedimiento, valor estimado, presupuesto base, importe de adjudicación y fecha de formalización deben definirse con precisión. Si se mezclan valor estimado e importe de adjudicación, un indicador de ahorro o concurrencia será incorrecto.

### Ejemplo 4: datos de alto valor

Los conjuntos de datos de alto valor requieren atención especial porque se espera que puedan reutilizarse con impacto económico, social o ambiental. La semántica ayuda a que su publicación no sea un mero volcado. Deben estar descritos con metadatos claros, formatos reutilizables, condiciones transparentes y, cuando proceda, servicios de acceso documentados.

## 11. Notas de test separadas

> Nota de test: si una pregunta contrapone "interoperabilidad técnica" e "interoperabilidad semántica", la respuesta correcta suele reservar la técnica para formatos, protocolos, servicios e infraestructuras, y la semántica para significado, modelos de datos, vocabularios y codificaciones.

> Nota de test: el ENI contempla la interoperabilidad como cualidad integral y multidimensional. No se limita a conectar aplicaciones; exige tener en cuenta organización, semántica, técnica y dimensión temporal.

> Nota de test: el Centro de Interoperabilidad Semántica se vincula a la publicación de modelos de datos, definiciones y codificaciones. No debe confundirse con un portal genérico de transparencia ni con un registro de expedientes.

> Nota de test: los modelos comunes tienen aplicación preferente, mientras que los modelos publicados por titulares competentes en materias sujetas a intercambio pueden tener aplicación obligatoria en su ámbito. La pregunta puede jugar con estos adjetivos.

> Nota de test: DCAT es vocabulario base; DCAT-AP es perfil europeo; DCAT-AP-ES es adaptación española. Si la pregunta habla de federación de catálogos europeos, la opción DCAT-AP suele ser la más precisa.

> Nota de test: una distribución no es lo mismo que un conjunto de datos. El conjunto es la unidad lógica; la distribución es una forma concreta de acceso o descarga.

> Nota de test: una equivalencia aproximada no permite tratar dos códigos como idénticos sin cautela. En indicadores oficiales, esta distinción es crítica.

> Nota de test: un identificador persistente no debe cambiar por una modificación estética del portal, un cambio de tecnología o una reorganización de rutas. Lo que cambia se documenta como metadato o versión.

> Nota de test: las notas sobre distractores deben mantenerse separadas del desarrollo teórico. En un examen tipo test, las trampas habituales son "confundir formato con significado", "confundir etiqueta con código" y "confundir catálogo con distribución".

## 12. Preguntas de recuperación activa

1. Explica en dos frases la diferencia entre interoperabilidad técnica y semántica.
2. ¿Por qué un código estable de órgano es mejor que usar la denominación visible como clave?
3. ¿Qué papel tienen los modelos de datos en el ENI?
4. ¿Cuándo basta una lista controlada y cuándo puede justificarse una ontología?
5. ¿Qué diferencia hay entre DCAT, DCAT-AP y DCAT-AP-ES?
6. ¿Por qué una distribución no equivale a un conjunto de datos?
7. ¿Qué problema resuelven las equivalencias semánticas?
8. ¿Qué riesgos aparecen si se agregan estados de expediente sin definición común?
9. ¿Qué metadatos mínimos esperarías en un dataset reutilizable?
10. ¿Por qué el versionado es parte de la calidad semántica?

## 13. Plan de visuales sugerido

1. Diagrama de tres capas de interoperabilidad: organizativa, semántica y técnica, con la semántica conectando modelos, vocabularios, códigos e identificadores.
2. Esquema de catálogo DCAT: catálogo -> dataset -> distribución -> servicio de datos, con metadatos alrededor.
3. Mapa de recursos semánticos: lista controlada, taxonomía, tesauro, ontología, mostrando aumento progresivo de expresividad.
4. Flujo de federación de catálogos: ayuntamientos -> perfil común de metadatos -> catálogo provincial -> reutilizadores.
5. Tabla visual de errores: etiqueta sin código, código sin definición, modelo sin versión, equivalencia sin tipo.

## 14. Encaje para el tema principal

Este bloque puede ubicarse después de explicar el gobierno del dato y antes de tratar calidad del dato y reutilización. La razón pedagógica es que la calidad no puede evaluarse sin semántica: un dato puede estar completo y en formato correcto, pero ser conceptualmente ambiguo. También conviene introducir DCAT-AP y la NTI de reutilización antes del apartado de reutilización de la información pública, porque los metadatos son el puente entre gobernar datos internamente y hacerlos reutilizables externamente.

La tesis de cierre recomendable es que la interoperabilidad semántica convierte los datos públicos en activos compartibles, verificables y reutilizables. Sin modelos, vocabularios, códigos, identificadores y equivalencias, la Administración puede publicar mucho y coordinar poco. Con gobierno semántico, la publicación y el intercambio pasan de ser volcado de información a infraestructura de servicio público.
