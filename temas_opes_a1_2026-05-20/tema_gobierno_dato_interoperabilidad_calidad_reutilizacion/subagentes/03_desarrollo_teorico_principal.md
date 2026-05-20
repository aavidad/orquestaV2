# Gobierno del dato, interoperabilidad semántica, calidad del dato y reutilización de la información pública

## Bloque de desarrollo teórico principal

Este bloque desarrolla el núcleo conceptual del tema: qué significa gobernar datos en una Administración pública, cómo se conecta ese gobierno con la interoperabilidad semántica, qué papel desempeña la calidad del dato y por qué la reutilización de la información pública no puede entenderse como una simple publicación de ficheros. El enfoque está pensado para integrarse en un tema A1: combina marco jurídico, criterios técnicos, ejemplos administrativos, tablas comparativas, notas de examen y supuestos breves.

La idea directriz es sencilla: el dato público solo produce valor cuando puede ser entendido, protegido, mantenido, intercambiado y reutilizado sin perder su significado. Una Administración puede tener grandes volúmenes de datos y, sin embargo, ofrecer servicios pobres si esos datos no tienen responsables claros, metadatos suficientes, controles de calidad, vocabularios comunes, trazabilidad jurídica y condiciones de reutilización transparentes. El gobierno del dato convierte el dato en un activo institucional; la interoperabilidad semántica permite que distintas organizaciones entiendan el mismo concepto de forma compatible; la calidad evita que el dato sea formalmente accesible pero materialmente inútil; y la reutilización abre el dato, dentro de sus límites legales, para generar transparencia, innovación y mejores decisiones públicas.

## 1. Orientación de examen

En un examen A1, este tema suele exigir una respuesta transversal. No basta con enumerar normas sobre datos abiertos ni con describir formatos técnicos. La respuesta fuerte debe relacionar cuatro planos:

1. El plano organizativo: responsabilidades, políticas, roles, ciclo de vida, toma de decisiones y coordinación entre unidades.
2. El plano jurídico: procedimiento administrativo, régimen jurídico del sector público, protección de datos, transparencia, reutilización, interoperabilidad y seguridad.
3. El plano semántico y técnico: modelos de datos, metadatos, vocabularios, estándares, identificadores, catálogos, APIs y formatos abiertos.
4. El plano de valor público: servicios más simples, reducción de cargas, rendición de cuentas, reutilización económica y social, datos de alto valor, evidencia para políticas públicas y confianza ciudadana.

La trampa habitual es tratar el dato como un mero recurso informático. En el sector público, el dato es también evidencia administrativa, soporte de derechos, base de decisiones, activo reutilizable y objeto de garantías. Por eso, conceptos como calidad, interoperabilidad o reutilización no se responden con una lista de herramientas, sino con una explicación de cómo se gobierna el dato desde su creación hasta su conservación, intercambio, publicación o supresión.

Otra trampa frecuente es confundir transparencia con datos abiertos. La transparencia se orienta al acceso a información pública y al control democrático. Los datos abiertos y la reutilización se orientan a permitir que la información del sector público sea usada de nuevo por personas, empresas, investigadores u otras Administraciones, en condiciones jurídicas, técnicas y económicas previsibles. Ambos ámbitos se tocan, pero no son idénticos: puede existir información accesible para transparencia que no sea reutilizable automáticamente en cualquier condición, y puede existir información reutilizable publicada en formatos preparados para explotación automática.

## 2. Mapa inicial del tema

| Eje | Pregunta que resuelve | Conceptos clave | Resultado esperado |
| --- | --- | --- | --- |
| Gobierno del dato | ¿Quién decide, cuida y responde por los datos? | política de datos, roles, ciclo de vida, catálogo, linaje, riesgos | datos gestionados como activo público |
| Interoperabilidad semántica | ¿Entendemos todos lo mismo cuando intercambiamos datos? | vocabularios, modelos comunes, metadatos, ontologías, identificadores | intercambio con significado consistente |
| Calidad del dato | ¿El dato es apto para el uso previsto? | exactitud, completitud, actualidad, consistencia, unicidad, trazabilidad | decisiones y servicios fiables |
| Reutilización | ¿Puede usarse de nuevo la información pública? | datos abiertos, licencias, formatos, APIs, conjuntos de alto valor | valor social, económico y democrático |
| Garantías | ¿Qué límites protegen derechos e intereses públicos? | protección de datos, seguridad, confidencialidad, propiedad intelectual, secreto | apertura responsable y legal |

La secuencia lógica para estudiar el tema es la siguiente: primero se define el dato como activo público; después se explica cómo se gobierna; a continuación se conecta con la interoperabilidad semántica y técnica; luego se desarrolla la calidad como requisito de confianza; finalmente se examina la reutilización de la información pública, sus obligaciones y límites.

## 3. Definiciones esenciales

### 3.1. Dato, información y conocimiento

Un dato es una representación formalizada de hechos, conceptos o instrucciones, apta para ser comunicada, interpretada o procesada. Esta definición es útil porque evita reducir el dato a una celda de una hoja de cálculo. Un dato puede aparecer en un registro administrativo, en un documento electrónico, en una base de datos, en una medición de sensores, en un catálogo de contratos, en un padrón, en un expediente o en un servicio de consulta automatizada.

La información surge cuando los datos adquieren contexto. El número “2025” aislado es un dato; “ejercicio presupuestario 2025 de una subvención concedida” ya es información, porque se vincula a un objeto, una fecha, un procedimiento y una entidad responsable. El conocimiento aparece cuando esa información se interpreta para decidir, evaluar o actuar. Una Administración que mejora la calidad de sus datos no solo ordena ficheros: aumenta la capacidad de decidir con evidencia.

En examen conviene distinguir estos tres niveles porque ayudan a explicar por qué los metadatos son tan importantes. Un dato sin metadatos puede conservar la forma, pero perder significado. Si se publica un conjunto de datos sobre expedientes sin indicar fecha de actualización, unidad responsable, cobertura temporal, diccionario de campos y licencia, la reutilización será débil aunque el fichero esté disponible.

### 3.2. Gobierno del dato

El gobierno del dato es el conjunto de políticas, roles, procesos, estándares y controles mediante los cuales una organización asegura que sus datos se gestionan de forma coherente, segura, trazable, interoperable y orientada al valor. En el sector público, este gobierno tiene una dimensión adicional: debe garantizar legalidad, derechos de las personas interesadas, transparencia, conservación documental, rendición de cuentas y coordinación interadministrativa.

Gobernar datos no equivale a centralizar todos los datos en una única base. Tampoco significa que una unidad tecnológica decida por sí sola. El gobierno del dato es una función institucional distribuida: las unidades de negocio conocen el significado y el uso administrativo; las unidades jurídicas interpretan límites y garantías; las unidades tecnológicas materializan modelos, controles y servicios; los responsables de seguridad gestionan riesgos; y los responsables de transparencia, archivo o reutilización aseguran que la información se publica, conserva o transfiere adecuadamente.

### 3.3. Interoperabilidad

La interoperabilidad es la capacidad de organizaciones y sistemas para compartir datos, información y conocimiento, y para que lo compartido sea comprendido y usado correctamente. En el ámbito público español, el Esquema Nacional de Interoperabilidad distingue una visión multidimensional: organizativa, semántica y técnica. Esta división es muy examinable.

La interoperabilidad organizativa se ocupa de acuerdos, procedimientos, responsabilidades y alineamiento de procesos. La interoperabilidad semántica se ocupa del significado: vocabularios, modelos de datos, definiciones comunes y metadatos. La interoperabilidad técnica se ocupa de la conexión efectiva: formatos, protocolos, servicios, interfaces, identificadores y estándares.

La dimensión semántica es el centro de este tema. Dos sistemas pueden conectarse técnicamente y, aun así, no ser interoperables de verdad si usan conceptos incompatibles. Por ejemplo, si un sistema entiende “unidad familiar” según criterios tributarios y otro según criterios de servicios sociales, no basta con enviar un campo llamado `unidad_familiar`; hay que explicar definición, alcance, fecha de referencia, reglas de cálculo y fuente.

### 3.4. Calidad del dato

La calidad del dato es el grado en que un dato resulta apto para un uso determinado. Esta definición es preferible a una definición absoluta. Un dato puede ser suficiente para una estadística agregada, pero insuficiente para resolver un procedimiento individual; puede ser válido para una consulta histórica, pero no para una decisión en tiempo real; puede ser correcto técnicamente, pero no estar legitimado para una reutilización concreta.

Las dimensiones más frecuentes de calidad son exactitud, completitud, consistencia, actualidad, unicidad, validez, integridad, trazabilidad y disponibilidad. En el sector público se añaden criterios de legalidad, proporcionalidad, conservación, seguridad y transparencia sobre la procedencia del dato.

### 3.5. Reutilización de la información pública

La reutilización de la información del sector público consiste en el uso de documentos o recursos de información que obran en poder de organismos públicos por personas físicas o jurídicas, con fines comerciales o no comerciales, siempre que ese uso no constituya una actividad administrativa pública. Esta idea se apoya en la Ley 37/2007, en la normativa europea de datos abiertos y en las normas técnicas de interoperabilidad.

La reutilización no es una concesión discrecional sin reglas. El ordenamiento europeo y español impulsa que los datos públicos sean reutilizables, especialmente cuando están en formatos abiertos, con metadatos, condiciones claras y acceso automatizable. Pero existen límites: protección de datos personales, seguridad pública, secreto, derechos de propiedad intelectual de terceros, confidencialidad estadística, protección de intereses comerciales legítimos, información clasificada y otros supuestos previstos por la normativa aplicable.

## 4. Marco normativo y político de referencia

El marco normativo debe estudiarse como un sistema. No hay una única ley de “gobierno del dato” que lo explique todo. El tema se apoya en normas de procedimiento, régimen jurídico, interoperabilidad, reutilización, datos abiertos, protección de datos, seguridad y marco europeo de la economía del dato.

| Norma o instrumento | Papel en el tema | Idea que conviene recordar |
| --- | --- | --- |
| Ley 39/2015, del Procedimiento Administrativo Común | Relación electrónica, documentos, datos aportados por interesados y tramitación administrativa | el dato sirve al procedimiento y a los derechos de las personas interesadas |
| Ley 40/2015, de Régimen Jurídico del Sector Público | Funcionamiento electrónico, cooperación, ENI y ENS | la interoperabilidad es un deber estructural del sector público |
| Real Decreto 4/2010, Esquema Nacional de Interoperabilidad | Criterios de interoperabilidad organizativa, semántica y técnica | la interoperabilidad se incorpora desde el diseño y durante todo el ciclo de vida |
| Real Decreto 203/2021 | Desarrollo reglamentario del funcionamiento electrónico | refuerza la actuación electrónica y la cooperación mediante medios digitales |
| Ley 37/2007, sobre reutilización de la información del sector público | Régimen jurídico de la reutilización | la información pública debe poder reutilizarse con condiciones claras y límites legales |
| Real Decreto 1495/2011 | Desarrollo de la reutilización en el sector público estatal | concreta obligaciones y modalidades de reutilización en el ámbito estatal |
| NTI de reutilización de recursos de información | Pautas de selección, identificación, descripción, formato, condiciones de uso y puesta a disposición | convierte la reutilización en una práctica interoperable |
| Directiva (UE) 2019/1024 | Datos abiertos y reutilización de la información del sector público | impulsa formatos abiertos, datos dinámicos y conjuntos de alto valor |
| Reglamento de Ejecución (UE) 2023/138 | Conjuntos de datos de alto valor | identifica categorías prioritarias y requisitos de publicación |
| Reglamento (UE) 2022/868, Gobernanza de Datos | Reutilización de ciertas categorías protegidas, intermediación y altruismo de datos | integra la reutilización en la gobernanza europea del dato |
| Reglamento (UE) 2023/2854, Datos | Acceso y uso justo de datos en la economía del dato | amplía el marco europeo de acceso a datos, con aplicación progresiva |
| Reglamento (UE) 2024/903, Europa Interoperable | Interoperabilidad del sector público en la Unión | conecta interoperabilidad transfronteriza, evaluaciones y soluciones reutilizables |
| Reglamento (UE) 2016/679 y Ley Orgánica 3/2018 | Protección de datos personales | apertura y reutilización no eliminan las garantías sobre datos personales |
| Real Decreto 311/2022, Esquema Nacional de Seguridad | Seguridad de la información y servicios electrónicos | la confianza en los datos depende también de confidencialidad, integridad, disponibilidad y trazabilidad |

Para examen, la clave no es memorizar todas las fechas, sino saber ubicar cada pieza. Ley 40/2015 y ENI explican interoperabilidad; Ley 37/2007 y Directiva 2019/1024 explican reutilización; RGPD y Ley Orgánica 3/2018 explican límites de datos personales; ENS explica seguridad; y los reglamentos europeos recientes muestran la evolución hacia espacios de datos, interoperabilidad transfronteriza y economía del dato.

## 5. Gobierno del dato en la Administración pública

### 5.1. Por qué el dato público necesita gobierno

La Administración produce datos en el ejercicio de potestades públicas, en la prestación de servicios, en la gestión presupuestaria, en la contratación, en la inspección, en la atención ciudadana, en la gestión de subvenciones, en los registros, en la actividad estadística y en la evaluación de políticas públicas. Muchos de esos datos tienen efectos jurídicos. Un dato erróneo puede impedir una ayuda, duplicar una carga, provocar una notificación incorrecta o distorsionar una estadística.

Por eso, el dato público no puede tratarse como un subproducto técnico. Debe gobernarse desde el momento en que se diseña un procedimiento o un servicio. Si un formulario pide un dato sin definir su finalidad, formato y base jurídica, se está creando deuda administrativa. Si un sistema almacena municipios sin código normalizado, se dificultará el intercambio posterior. Si una unidad publica datos sin metadatos, la reutilización dependerá de interpretación manual. Si no hay responsable de calidad, los errores se normalizarán.

El gobierno del dato busca evitar esa fragmentación. Su objetivo no es acumular más información, sino producir datos confiables, proporcionales, interoperables y útiles. En la práctica, esto requiere inventariar datos, definir propietarios funcionales, establecer reglas de calidad, crear catálogos, normalizar metadatos, gestionar accesos, documentar transformaciones, medir incidencias y revisar periódicamente usos y riesgos.

### 5.2. Principios de un buen gobierno del dato

Un modelo sólido de gobierno del dato en el sector público debería apoyarse, al menos, en los siguientes principios:

| Principio | Explicación | Ejemplo administrativo |
| --- | --- | --- |
| Finalidad pública | Los datos se recogen y tratan para cumplir competencias, prestar servicios o habilitar derechos | datos de una solicitud de beca usados para tramitar esa beca y controles previstos |
| Minimización y proporcionalidad | Se piden y conservan los datos necesarios, no los máximos posibles | no solicitar documentación ya disponible si puede consultarse legítimamente |
| Responsabilidad | Cada conjunto relevante tiene una unidad responsable de significado, calidad y actualización | padrón, registro de ayudas, catálogo de contratos o inventario de bienes |
| Calidad desde el origen | La calidad se diseña en la captura y validación inicial, no solo al final | listas normalizadas, validaciones de formato y controles de duplicidad |
| Interoperabilidad por diseño | El dato nace preparado para ser compartido cuando proceda | códigos DIR3, identificadores persistentes, vocabularios y metadatos comunes |
| Seguridad y protección | El acceso se controla según riesgos, categorías de datos y finalidad | perfiles, auditoría, cifrado, segregación de entornos y trazabilidad |
| Transparencia y explicabilidad | Debe conocerse fuente, significado, actualización y condiciones de uso | ficha de metadatos en un catálogo de datos abiertos |
| Ciclo de vida | Se regula creación, uso, conservación, archivo, publicación y eliminación | calendarios de conservación y políticas de gestión documental |

Estos principios no deben citarse como una lista decorativa. Hay que explicar que se refuerzan entre sí. La interoperabilidad sin calidad produce intercambio de errores. La apertura sin seguridad puede vulnerar derechos. La calidad sin responsabilidad se deteriora con el tiempo. La reutilización sin metadatos reduce el valor del dato. El gobierno del dato es precisamente el mecanismo que coordina esos equilibrios.

### 5.3. Roles y responsabilidades

En organizaciones maduras se distinguen varios roles. Los nombres pueden variar, pero la lógica es común.

El responsable funcional del dato conoce el significado administrativo. Decide, por ejemplo, qué significa “expediente activo”, cuándo una ayuda está “concedida” y qué fuente es la autoridad para una determinada información. No siempre coincide con el responsable técnico del sistema.

La persona o unidad encargada de custodia técnica asegura disponibilidad, integridad, rendimiento, copias, interfaces y controles técnicos. Su papel es esencial, pero no sustituye el juicio funcional. Un sistema puede estar técnicamente sano y contener datos conceptualmente ambiguos.

El responsable de calidad define reglas, indicadores, umbrales y circuitos de corrección. En una Administración, las incidencias de calidad deben conectarse con los procedimientos. Si una dirección está incompleta, no basta con marcarla como error: hay que saber si impide una notificación, si puede subsanarse, si procede consultar una fuente oficial o si debe corregirse por la persona interesada.

La unidad jurídica y de protección de datos evalúa bases jurídicas, límites de tratamiento, publicidad, cesiones, reutilización, anonimización y derechos. Su intervención es especialmente importante cuando se combinan conjuntos de datos, se publican datos abiertos o se crean servicios de intercambio.

La unidad de archivo y gestión documental aporta reglas de conservación, autenticidad, integridad documental, metadatos de gestión y disposición final. Esta función evita que la transformación digital destruya el valor probatorio de los documentos y datos administrativos.

La unidad de seguridad aplica medidas acordes con el ENS y con el análisis de riesgos. La seguridad no se limita a impedir accesos indebidos. También protege disponibilidad, integridad, trazabilidad, autenticidad y conservación, elementos centrales para confiar en los datos.

### 5.4. Ciclo de vida del dato

El ciclo de vida ayuda a ordenar el tema. Un dato público atraviesa fases que deben gobernarse:

1. Diseño: se define finalidad, base jurídica, modelo de datos, calidad esperada, interoperabilidad, conservación y posibles usos futuros.
2. Captura o generación: el dato se recibe de la ciudadanía, de otra Administración, de una medición, de un documento o de un sistema interno.
3. Validación: se comprueba formato, obligatoriedad, rango, coherencia, duplicidad y fuente.
4. Almacenamiento y custodia: se conserva con controles de acceso, seguridad, integridad y metadatos.
5. Uso administrativo: se emplea para tramitar, resolver, inspeccionar, evaluar o prestar servicios.
6. Intercambio: se comparte con otros órganos o Administraciones cuando existe habilitación y finalidad.
7. Publicación o reutilización: se ofrece como información pública o dato abierto cuando procede.
8. Conservación, archivo o eliminación: se aplica la política documental y de conservación correspondiente.

La fase de diseño es la más importante y la más olvidada. Si un servicio público nace sin modelo de datos claro, después será difícil abrir datos, intercambiar información o medir calidad. Por ejemplo, un registro de autorizaciones que no normaliza actividad, territorio y fechas tendrá problemas para integrarse con portales de transparencia o sistemas de inspección.

## 6. Interoperabilidad semántica

### 6.1. Concepto y relevancia

La interoperabilidad semántica garantiza que el significado de los datos intercambiados se conserva entre sistemas, organizaciones y contextos. Su objetivo no es que todos usen necesariamente la misma aplicación, sino que puedan entender conceptos comunes de forma compatible. En Administraciones públicas, esta dimensión es crítica porque el mismo ciudadano, empresa, inmueble, expediente o prestación puede aparecer en múltiples procedimientos.

La interoperabilidad semántica se apoya en definiciones compartidas, metadatos, esquemas, códigos normalizados, vocabularios controlados, ontologías y modelos de intercambio. También se apoya en acuerdos organizativos: una definición común debe ser aprobada, mantenida y difundida. La semántica no se resuelve solo con tecnología.

Un ejemplo simple es la codificación territorial. Si una Administración usa nombres libres de municipios y otra usa códigos normalizados, aparecerán variantes, errores ortográficos y ambigüedades. Un ejemplo más complejo es el concepto de “unidad económica” o “persona beneficiaria”, que puede variar según normativa sectorial. La interoperabilidad semántica exige documentar esas diferencias y, si se intercambian datos, establecer correspondencias.

### 6.2. Dimensiones de la interoperabilidad

| Dimensión | Qué resuelve | Riesgo si falla | Instrumentos habituales |
| --- | --- | --- | --- |
| Organizativa | Alinea procesos, competencias y acuerdos | intercambio sin base procedimental o con responsabilidades confusas | convenios, protocolos, acuerdos de servicio, gobernanza |
| Semántica | Preserva significado | datos técnicamente recibidos pero mal interpretados | modelos de datos, vocabularios, metadatos, diccionarios |
| Técnica | Permite conexión y procesamiento | sistemas aislados o dependientes de formatos propietarios | APIs, servicios, formatos abiertos, estándares, identificadores |
| Jurídica | Asegura habilitación y límites | cesiones indebidas, vulneración de derechos o publicación improcedente | bases legales, análisis de protección de datos, licencias |

Aunque el ENI destaque tres dimensiones clásicas, en una respuesta A1 resulta útil mencionar la dimensión jurídica como garantía transversal. La Administración no debe intercambiar datos solo porque técnicamente pueda hacerlo. Debe existir competencia, finalidad, habilitación y cumplimiento de protección de datos, seguridad y procedimiento.

### 6.3. Metadatos

Los metadatos son datos que describen otros datos. En un catálogo de datos abiertos, los metadatos indican título, descripción, organismo responsable, fecha de actualización, cobertura temporal, cobertura geográfica, formato, licencia, frecuencia, tema, identificadores, vocabulario usado y distribuciones disponibles. En gestión documental, los metadatos describen documento, expediente, firma, estado, trazabilidad y conservación.

Los metadatos son esenciales por tres razones. Primero, permiten encontrar datos. Segundo, permiten entenderlos. Tercero, permiten reutilizarlos de forma automatizada. Sin metadatos, un conjunto de datos puede existir, pero quedar invisible o ser malinterpretado.

La calidad de metadatos debe medirse. Un catálogo con muchos conjuntos de datos, pero con descripciones vagas, fechas ausentes y formatos heterogéneos, ofrece una apariencia de apertura sin una reutilización sólida. La publicación moderna exige metadatos estructurados y alineados con perfiles comunes, como DCAT-AP y sus adaptaciones nacionales, cuando resulten aplicables.

### 6.4. Vocabularios controlados, taxonomías y ontologías

Un vocabulario controlado es una lista normalizada de términos permitidos. Sirve para evitar variantes innecesarias. Por ejemplo, una lista normalizada de sectores, tipos de contrato, estados de expediente o categorías territoriales.

Una taxonomía organiza términos en una estructura jerárquica. Permite navegar de categorías generales a específicas. Por ejemplo, un portal de datos puede clasificar conjuntos en economía, sector público, medio ambiente, transporte o educación.

Una ontología expresa conceptos, relaciones y reglas con mayor formalización. Es útil cuando se requiere interoperabilidad semántica avanzada, especialmente en entornos de datos enlazados. No todo proyecto administrativo necesita una ontología compleja, pero todo proyecto que intercambia datos debería tener definiciones claras y vocabularios controlados.

La regla práctica es proporcionalidad: no convertir cada formulario en un proyecto semántico complejo, pero tampoco permitir campos libres cuando existen códigos oficiales o vocabularios comunes. La madurez consiste en aplicar el nivel de formalización adecuado al impacto del dato.

### 6.5. Identificadores persistentes

Un identificador persistente permite referirse de forma estable a una entidad, recurso o conjunto de datos. En la Administración, los identificadores son esenciales para evitar duplicidades, enlazar información, auditar cambios y mantener referencias a largo plazo.

No todos los identificadores tienen la misma función. Un identificador interno de base de datos puede servir para una aplicación, pero no necesariamente para intercambio público. Un identificador de expediente debe seguir reglas de gestión administrativa. Un identificador de recurso reutilizable debe ser estable, documentado y, si procede, resoluble.

La persistencia es clave. Si un conjunto de datos cambia de ubicación o de nombre sin redirección ni historial, se rompe la reutilización. Quien construye un servicio basado en esos datos necesita continuidad. Por eso, las políticas de publicación deben prever versiones, historiales, cambios de esquema y retirada responsable de datos.

## 7. Modelos de datos y catálogos

### 7.1. Modelo conceptual, lógico y físico

Un modelo de datos conceptual define entidades y relaciones en lenguaje cercano al negocio. Por ejemplo: persona interesada, expediente, solicitud, órgano gestor, resolución, pago. Un modelo lógico traduce esas entidades a estructuras más precisas: atributos, claves, relaciones, restricciones. Un modelo físico implementa el diseño en una tecnología concreta: tablas, documentos, grafos, índices o servicios.

En examen, esta distinción permite explicar por qué la interoperabilidad semántica no debe depender de una base de datos concreta. Dos Administraciones pueden tener modelos físicos distintos y, aun así, intercambiar información si comparten un modelo conceptual y un esquema de intercambio. Lo importante es que el significado sea claro y las transformaciones estén documentadas.

Un error habitual es publicar el modelo físico como si fuera el modelo semántico. Exportar columnas de una tabla con nombres técnicos puede ser rápido, pero no siempre es comprensible ni estable. Un buen catálogo debe explicar qué representa cada campo, cuál es su dominio de valores, cómo se actualiza y qué limitaciones tiene.

### 7.2. Catálogo de datos

El catálogo de datos es una herramienta de gobierno. Permite inventariar conjuntos, responsables, definiciones, calidad, sensibilidad, accesos, linaje y condiciones de uso. Puede haber catálogos internos, orientados a gestión y control, y catálogos externos, orientados a reutilización pública.

Un catálogo interno responde preguntas como: qué datos existen, quién los custodia, cuál es la fuente maestra, qué calidad tienen, qué restricciones legales aplican y qué servicios los consumen. Un catálogo de datos abiertos responde preguntas como: qué conjuntos se publican, cómo se descargan, qué licencia tienen, con qué frecuencia se actualizan y cómo deben citarse.

| Tipo de catálogo | Usuarios principales | Contenido típico | Valor |
| --- | --- | --- | --- |
| Catálogo interno de datos | unidades administrativas, tecnología, seguridad, analítica | inventario, responsables, linaje, sensibilidad, calidad, accesos | gobierno y control institucional |
| Catálogo de datos abiertos | ciudadanía, empresas, academia, periodistas, otras Administraciones | conjuntos reutilizables, metadatos, formatos, licencias, APIs | reutilización y transparencia activa |
| Catálogo semántico | equipos de interoperabilidad, arquitectura de datos, proyectos transversales | vocabularios, modelos, definiciones, códigos, relaciones | significado común e integración |
| Catálogo documental | archivo, tramitación, gestión documental | series, expedientes, metadatos, conservación, transferencia | autenticidad, conservación y prueba |

La integración entre catálogos evita duplicidades. Si el catálogo abierto no se alimenta de un gobierno interno de datos, la publicación se convierte en una tarea manual y frágil. Si el catálogo interno ignora la reutilización, puede perder oportunidades de apertura y valor público.

## 8. Calidad del dato

### 8.1. Calidad como aptitud para el uso

La calidad del dato debe definirse según el uso. Un dato de localización aproximada puede ser suficiente para planificar recursos sanitarios, pero no para practicar una notificación. Una fecha de actualización mensual puede ser adecuada para un indicador estadístico, pero no para una consulta de disponibilidad de plazas en tiempo real. La calidad siempre se mide contra una finalidad.

Esto no significa relativismo. Hay dimensiones objetivas que pueden controlarse. La exactitud mide si el dato representa correctamente la realidad o la fuente autorizada. La completitud mide si faltan valores necesarios. La consistencia mide si el dato no contradice otros datos. La actualidad mide si está actualizado para el uso previsto. La unicidad mide si no hay duplicados indebidos. La validez mide si cumple formato y dominio. La integridad mide si se mantienen relaciones y restricciones. La trazabilidad mide si se conoce origen y transformación.

### 8.2. Dimensiones de calidad

| Dimensión | Pregunta de control | Ejemplo de incidencia | Medida de mejora |
| --- | --- | --- | --- |
| Exactitud | ¿El dato refleja la fuente correcta? | importe de subvención mal transcrito | validación contra resolución o sistema contable |
| Completitud | ¿Faltan datos necesarios? | expediente sin fecha de resolución | campos obligatorios y control de cierre |
| Consistencia | ¿Contradice otros datos? | expediente marcado como cerrado sin resolución | reglas de coherencia entre estados |
| Actualidad | ¿Está al día para el uso previsto? | catálogo de centros con teléfonos obsoletos | periodicidad de revisión y responsable |
| Unicidad | ¿Hay duplicados indebidos? | dos registros de la misma entidad beneficiaria | reglas de deduplicación y fuente maestra |
| Validez | ¿Cumple formato y dominio? | código postal con letras | validaciones de formato y listas oficiales |
| Integridad | ¿Se preservan relaciones? | pago asociado a expediente inexistente | claves, restricciones y conciliaciones |
| Trazabilidad | ¿Se conoce origen y transformación? | indicador sin método de cálculo documentado | linaje, versión y ficha metodológica |

La calidad debe gestionarse con indicadores y circuitos. Medir errores sin corregirlos crea frustración. Corregirlos sin registrar causa impide aprender. La práctica adecuada combina prevención, detección, corrección y mejora continua.

### 8.3. Calidad en datos abiertos

En datos abiertos, la calidad tiene una dimensión externa. La persona reutilizadora no conoce los procesos internos de la Administración. Necesita metadatos, documentación, estructura estable, formatos reutilizables, fechas de actualización, ejemplos, licencia clara y canales de contacto o incidencias.

Un conjunto de datos abierto de baja calidad puede generar más costes que beneficios. Si cambia el esquema sin aviso, rompe aplicaciones. Si no indica la fecha de actualización, impide evaluar vigencia. Si contiene valores inconsistentes, exige limpieza previa. Si se publica como PDF escaneado, puede ser legalmente accesible pero técnicamente poco reutilizable.

La calidad de los datos abiertos debe analizarse en niveles:

| Nivel | Pregunta | Ejemplo |
| --- | --- | --- |
| Acceso | ¿El dato puede obtenerse sin barreras innecesarias? | descarga directa o API documentada |
| Formato | ¿El formato permite procesamiento automático? | CSV, JSON, XML, RDF u otros formatos adecuados |
| Metadatos | ¿Se entiende el contenido? | descripción, responsable, frecuencia y licencia |
| Estabilidad | ¿Se mantienen identificadores y esquemas? | versiones y aviso de cambios |
| Semántica | ¿Los campos tienen significado claro? | diccionario de datos y vocabularios |
| Legalidad | ¿El uso está permitido y limitado correctamente? | licencia y exclusiones documentadas |

### 8.4. Linaje y trazabilidad

El linaje describe de dónde viene un dato, qué transformaciones ha sufrido y dónde se utiliza. En Administración es especialmente importante cuando se crean indicadores, sistemas de analítica, interoperabilidad entre organismos o publicaciones de datos abiertos.

Sin linaje, no se puede responder con solvencia a preguntas básicas: ¿qué fuente alimenta este indicador?, ¿cuándo se actualizó?, ¿se excluyeron expedientes anulados?, ¿se anonimizaron datos personales?, ¿qué versión se publicó?, ¿qué sistema consumió el dato?, ¿quién corrigió un error?

El linaje también es una herramienta de responsabilidad. Si una decisión automatizada o asistida por datos produce un resultado incorrecto, la organización debe poder reconstruir qué datos se usaron y con qué reglas. Aunque este tema no sea un tema de inteligencia artificial, el gobierno del dato es condición previa para cualquier uso responsable de analítica avanzada.

## 9. Reutilización de la información pública

### 9.1. Sentido de la reutilización

La reutilización de la información pública persigue que los datos elaborados o custodiados por el sector público puedan generar valor más allá de su uso administrativo inicial. Ese valor puede ser democrático, económico, social, científico o interno. La ciudadanía puede fiscalizar políticas públicas; las empresas pueden crear servicios; la academia puede investigar; los medios pueden analizar; otras Administraciones pueden mejorar servicios; y la propia organización puede detectar ineficiencias.

La reutilización se conecta con el principio de “datos abiertos por diseño y por defecto”, impulsado por el marco europeo, aunque siempre dentro de los límites legales. El enfoque moderno no consiste en publicar ocasionalmente ficheros, sino en diseñar datos reutilizables desde el origen: formatos abiertos, metadatos, identificadores persistentes, APIs cuando proceda, documentación y gobernanza de calidad.

### 9.2. Documentos, recursos de información y distribuciones

En reutilización conviene distinguir el recurso de información y sus distribuciones. El recurso es el conjunto o documento reutilizable; la distribución es una forma concreta de acceder a él en un formato determinado. Un mismo conjunto puede ofrecerse como CSV, JSON y API. Esta distinción es clave en catálogos basados en DCAT: el conjunto conserva identidad y metadatos comunes, mientras cada distribución describe formato, tamaño, acceso y características técnicas.

Ejemplo: un catálogo de centros educativos es el conjunto de datos. Sus distribuciones pueden ser un fichero CSV descargable, un fichero JSON para consumo automático y un servicio de consulta. El título, descripción, organismo responsable y licencia pertenecen al conjunto; el formato y punto de acceso pertenecen a cada distribución.

### 9.3. Condiciones de reutilización

Las condiciones de reutilización deben ser claras, proporcionadas y no discriminatorias. Pueden exigir citar la fuente, mencionar la fecha de última actualización, no desnaturalizar el sentido de la información o indicar si se han realizado transformaciones. Lo importante es que la persona reutilizadora sepa qué puede hacer y qué obligaciones asume.

No debe confundirse condición de reutilización con barrera arbitraria. La tendencia europea favorece la reutilización amplia y la reducción de restricciones técnicas, jurídicas y económicas. Cuando existan tasas, exclusividades o limitaciones, deben justificarse conforme a la normativa aplicable. En los conjuntos de alto valor, el marco europeo refuerza disponibilidad, gratuidad, formatos legibles por máquina y acceso mediante API o descarga masiva, según categoría y condiciones concretas.

### 9.4. Límites de la reutilización

La reutilización no anula otros bienes jurídicos. Antes de publicar, la Administración debe analizar si el conjunto incluye datos personales, información protegida, secreto comercial, seguridad pública, propiedad intelectual de terceros, confidencialidad estadística, información ambiental con restricciones específicas u otras limitaciones.

El caso más frecuente es la presencia de datos personales. La regla práctica es que no basta con quitar nombres si el resto de variables permite reidentificar. La anonimización exige evaluar riesgo de reidentificación, contexto, combinaciones posibles y evolución tecnológica. Cuando se usan datos agregados, debe comprobarse que los grupos no son tan pequeños que permitan identificar personas. La seudonimización reduce riesgos, pero no convierte automáticamente los datos en anónimos.

Por tanto, el gobierno del dato debe integrar protección de datos desde el diseño de la publicación. La pregunta no es solo “¿podemos publicar?”, sino “¿qué versión, agregación, anonimización, periodicidad, formato y condiciones permiten maximizar valor sin vulnerar derechos?”.

### 9.5. Datos de alto valor

La Directiva de datos abiertos y su desarrollo europeo introducen la categoría de conjuntos de datos de alto valor. Son conjuntos cuyo potencial de beneficio social, económico, ambiental o de innovación justifica requisitos reforzados de disponibilidad. Las categorías europeas incluyen ámbitos como geoespacial, observación de la Tierra y medio ambiente, meteorología, estadística, sociedades y propiedad de sociedades, y movilidad, de acuerdo con la lista aplicable.

Para examen, lo importante es entender la razón: no todos los datos tienen la misma prioridad de apertura. Un conjunto de alto valor puede generar servicios transfronterizos, investigación, innovación empresarial o mejora de políticas públicas. Por eso se promueve su disponibilidad en formatos legibles por máquina, mediante APIs y descarga masiva cuando corresponda.

## 10. Datos abiertos, APIs y formatos

### 10.1. Formatos abiertos y legibles por máquina

Un formato abierto es aquel cuya especificación está disponible y puede ser implementada sin restricciones injustificadas. Un formato legible por máquina permite procesamiento automático. La combinación de ambos favorece reutilización, competencia, independencia tecnológica y conservación.

No todos los formatos sirven igual. Un PDF puede ser adecuado para lectura humana de una resolución, pero no para analizar miles de registros. Una imagen escaneada dificulta la reutilización. Un CSV puede ser suficiente para tablas simples; JSON puede ser útil para APIs; XML puede ser adecuado para intercambio estructurado; RDF y datos enlazados pueden aportar semántica avanzada.

La elección debe responder al uso. Publicar todo en el formato más complejo no es madurez. Madurez es ofrecer formatos adecuados, documentados y estables.

| Necesidad | Formato o enfoque habitual | Advertencia |
| --- | --- | --- |
| Tabla sencilla descargable | CSV | documentar codificación, separador, cabeceras y tipos |
| Intercambio estructurado | XML o JSON | publicar esquema y versiones |
| Consulta por aplicaciones | API | documentar autenticación si existe, límites, errores y cambios |
| Datos semánticos enlazables | RDF, vocabularios, URIs | requiere gobernanza semántica |
| Documento para lectura | PDF accesible | no sustituye al dato reutilizable si se requiere explotación automática |

### 10.2. APIs públicas

Las APIs permiten acceso automatizado a datos o servicios. En reutilización, una API puede ser más útil que una descarga estática cuando los datos cambian con frecuencia o cuando se requieren consultas parciales. Sin embargo, una API mal documentada puede ser una barrera. Debe indicar parámetros, formatos de respuesta, códigos de error, límites de uso, paginación, versiones, ejemplos y condiciones.

Las APIs no sustituyen siempre a la descarga masiva. Para análisis completos, la descarga puede ser más eficiente. Para datos dinámicos, la API puede ser imprescindible. El buen diseño combina ambos enfoques cuando procede.

Una Administración debe evitar que la API se convierta en un punto opaco. Si solo se permite consulta caso a caso y no se ofrece descarga cuando la reutilización lo requiere, se limita el potencial del dato. Si se ofrece descarga sin actualización ni documentación, se limita la confianza. La decisión debe atender al tipo de dato, frecuencia, volumen, demanda y obligaciones normativas.

## 11. Protección de datos, seguridad y apertura responsable

### 11.1. Protección de datos personales

El RGPD y la Ley Orgánica 3/2018 son límites estructurales de cualquier política de datos. La Administración puede tener obligación o habilitación para tratar datos personales, pero eso no implica que pueda publicarlos o reutilizarlos libremente. Hay que respetar principios como licitud, lealtad, transparencia, limitación de finalidad, minimización, exactitud, limitación del plazo de conservación, integridad, confidencialidad y responsabilidad proactiva.

La reutilización de información pública debe distinguir datos anónimos, datos personales, datos seudonimizados y datos agregados. Los datos verdaderamente anónimos quedan fuera del régimen de protección de datos personales, pero lograr anonimización robusta exige análisis. Los datos seudonimizados siguen siendo datos personales si puede reidentificarse a la persona con información adicional. Los datos agregados reducen riesgos, pero pueden reidentificar si las celdas son pequeñas o si se combinan con otros conjuntos.

En examen, una respuesta madura diría: la apertura se presume deseable, pero debe aplicarse con evaluación jurídica y técnica. No se debe publicar menos por miedo genérico, ni más por entusiasmo tecnológico. Debe publicarse lo correcto, en el nivel de detalle adecuado y con garantías.

### 11.2. Seguridad de la información

El ENS conecta directamente con el gobierno del dato. La seguridad protege confidencialidad, integridad, disponibilidad, autenticidad, trazabilidad y conservación. Si un dato se altera sin control, deja de ser confiable. Si un servicio de datos no está disponible, se perjudican servicios públicos y reutilizadores. Si no hay trazabilidad, no puede investigarse una incidencia.

La seguridad también se aplica a datos abiertos. Que un dato sea público no significa que el sistema que lo publica carezca de riesgos. Deben protegerse infraestructura, integridad de ficheros, disponibilidad, control de versiones, prevención de manipulaciones y trazabilidad de publicación. Un portal de datos abiertos que difunde ficheros alterados puede producir daños importantes.

### 11.3. Anonimización y agregación

La anonimización consiste en transformar datos personales de modo que las personas no sean identificables razonablemente. Técnicas habituales son supresión de identificadores, generalización, agregación, perturbación, recodificación, muestreo o enmascaramiento. La técnica adecuada depende del riesgo y del uso.

La agregación presenta datos en grupos. Es útil para estadísticas, mapas e indicadores. Sin embargo, debe evitar grupos demasiado pequeños, combinaciones raras o series temporales que permitan inferencias. Por ejemplo, publicar ayudas por municipio, sexo, edad y condición específica puede parecer agregado, pero en municipios pequeños puede identificar a personas.

La buena práctica es documentar el criterio de anonimización o agregación, revisar riesgos antes de publicar y controlar cambios. La publicación repetida de datos con pequeñas variaciones puede permitir reconstrucciones.

## 12. Interoperabilidad y reutilización en servicios públicos

### 12.1. Una sola vez y reducción de cargas

Uno de los valores de la interoperabilidad es evitar que la ciudadanía aporte repetidamente documentos o datos que ya obran en poder de la Administración. Este principio, a menudo descrito como “solo una vez”, requiere algo más que voluntad política. Requiere fuentes fiables, servicios de consulta, bases jurídicas, trazabilidad, consentimiento cuando proceda, seguridad, calidad y acuerdos entre Administraciones.

Ejemplo: para tramitar una ayuda puede ser necesario verificar identidad, residencia, situación tributaria o discapacidad. Si cada órgano pide certificados en papel, aumenta la carga ciudadana y se multiplican errores. Si existen servicios interoperables, la Administración puede comprobar datos de forma más eficiente, siempre respetando finalidad, competencia y garantías.

### 12.2. Interoperabilidad semántica en expedientes

Los expedientes administrativos contienen documentos, datos, estados, firmas, plazos y actuaciones. Si una Administración intercambia expedientes o asientos registrales, necesita modelos de datos compatibles. No basta con remitir documentos; hay que conservar metadatos, contexto y estructura.

La gestión documental y la interoperabilidad se encuentran aquí. Un documento electrónico debe ser identificable, íntegro, auténtico y conservable. Sus metadatos permiten saber qué es, a qué expediente pertenece, quién lo firmó, cuándo se incorporó y qué estado tiene. Sin esa capa de metadatos, el intercambio documental pierde valor probatorio.

### 12.3. Reutilización interna

La reutilización no es solo externa. Las propias Administraciones pueden reutilizar datos para mejorar servicios, evaluar políticas, detectar fraude, planificar recursos o simplificar procedimientos. Esta reutilización interna también exige gobierno. No todo dato recogido para una finalidad puede usarse sin más para otra; debe analizarse base jurídica, compatibilidad, minimización y garantías.

Un ejemplo positivo sería usar datos agregados de tiempos de tramitación para detectar cuellos de botella. Un ejemplo que requiere cautela sería combinar datos sociales, fiscales y sanitarios para priorizar intervenciones: puede tener valor público, pero exige base jurídica, evaluación de riesgos, controles de acceso y explicabilidad.

## 13. Ejemplos administrativos desarrollados

### 13.1. Catálogo de subvenciones

Una Administración publica un catálogo de subvenciones concedidas. Para que sea reutilizable, no basta con subir un PDF anual. Debe identificar cada convocatoria, órgano concedente, persona o entidad beneficiaria cuando proceda, importe, fecha, finalidad, programa presupuestario, territorio y base reguladora, con las cautelas legales aplicables.

El gobierno del dato define quién es responsable del conjunto, qué sistema es la fuente, cómo se corrigen errores, cuándo se actualiza y qué campos se publican. La interoperabilidad semántica define códigos de órganos, territorio, actividad y tipos de beneficiarios. La calidad exige importes correctos, fechas completas, ausencia de duplicados y coherencia con contabilidad. La reutilización exige formato abierto, metadatos, licencia y documentación.

Si falta cualquiera de esos elementos, el valor baja. Un conjunto con importes correctos pero sin códigos normalizados dificulta análisis territorial. Un conjunto con metadatos pobres exige interpretación manual. Un conjunto sin actualización clara pierde confianza. Un conjunto con datos personales excesivos puede vulnerar derechos.

### 13.2. Registro de centros de servicios públicos

Un registro de centros administrativos, sanitarios, educativos o sociales suele ser muy reutilizable. Permite mapas, buscadores, planificación, accesibilidad y coordinación. Sus campos básicos pueden incluir identificador del centro, nombre, tipo, dirección, municipio, coordenadas, horario, contacto, servicios prestados y fecha de actualización.

La calidad geográfica es especialmente importante. Una dirección textual puede ser comprensible para una persona, pero no para un sistema de rutas. Las coordenadas deben tener precisión y sistema de referencia. Los municipios deben codificarse. Los horarios deben seguir estructura. Los servicios deben clasificarse con vocabulario.

La interoperabilidad semántica permite que un centro no sea clasificado de forma distinta en cada portal. La reutilización permite que terceros creen aplicaciones de búsqueda. La protección de datos apenas será un problema si se trata de centros públicos, pero la seguridad e integridad sí importan: un teléfono o dirección incorrecta puede impedir el acceso al servicio.

### 13.3. Datos de contratación pública

La contratación pública es un ámbito de alto interés por transparencia, control del gasto y competencia. Los datos pueden incluir licitaciones, adjudicatarios, importes, procedimientos, criterios, lotes, fechas y ejecución. La reutilización permite análisis de concurrencia, concentración de adjudicaciones, plazos, modificaciones y eficiencia.

La dificultad está en la normalización. Los nombres de empresas pueden variar, los códigos CPV deben usarse correctamente, los importes deben distinguir presupuesto base, valor estimado y adjudicación, y los estados del contrato deben estar claros. La calidad y semántica son decisivas para evitar conclusiones erróneas.

Este ejemplo es muy útil en examen porque muestra que publicar mucho no equivale a publicar bien. Un portal puede tener miles de contratos y, sin embargo, ser difícil de analizar si los campos son inconsistentes.

## 14. Supuesto práctico breve

Una comunidad autónoma quiere publicar datos abiertos sobre listas de espera de un servicio público y, al mismo tiempo, compartir datos con ayuntamientos para coordinar atención presencial.

Primera decisión: separar los usos. La publicación abierta busca transparencia y reutilización; el intercambio con ayuntamientos busca gestión de servicios. Cada uso tendrá base jurídica, nivel de detalle, controles y destinatarios distintos.

Segunda decisión: definir modelo de datos. Para publicación abierta pueden bastar datos agregados por territorio, tipo de servicio, tramo temporal y estado. Para intercambio operativo pueden necesitarse registros individuales, pero solo si existe habilitación y finalidad concreta.

Tercera decisión: proteger datos personales. La publicación abierta debe evitar identificación directa o indirecta. Si hay territorios pequeños o categorías sensibles, se agregan o suprimen detalles. El intercambio operativo requiere control de accesos, trazabilidad, minimización y seguridad.

Cuarta decisión: asegurar calidad. Se define fuente maestra, fecha de corte, reglas de cálculo, tratamiento de duplicados, estados y periodicidad. Se publica ficha metodológica para que la ciudadanía entienda el indicador.

Quinta decisión: facilitar reutilización. Se ofrecen datos en formato abierto, con metadatos, licencia, histórico y aviso de cambios metodológicos. Si la actualización es frecuente, se valora API.

La respuesta de examen debe mostrar este razonamiento: finalidad, base jurídica, modelo, calidad, interoperabilidad, protección, publicación y seguimiento.

## 15. Notas de test separadas

> Nota de test 1. Si una pregunta contrapone interoperabilidad técnica y semántica, la técnica conecta sistemas y la semántica preserva significado. Una API no garantiza por sí sola que los datos se interpreten correctamente.

> Nota de test 2. La calidad del dato no es solo exactitud. También incluye completitud, consistencia, actualidad, unicidad, validez, integridad y trazabilidad.

> Nota de test 3. Datos abiertos y transparencia no son sinónimos. La transparencia se orienta al acceso y control democrático; la reutilización exige condiciones técnicas y jurídicas para usar de nuevo la información.

> Nota de test 4. La seudonimización no equivale a anonimización. Si puede reidentificarse a la persona con información adicional, sigue existiendo dato personal.

> Nota de test 5. El ENI no es una herramienta informática concreta. Es un marco de criterios y recomendaciones para decisiones tecnológicas, conservación, normalización e interoperabilidad.

> Nota de test 6. Un formato legible por máquina no siempre es abierto, y un formato abierto no siempre está bien documentado. Para reutilización real hacen falta formato, metadatos, estabilidad y condiciones de uso.

## 16. Errores frecuentes

| Error | Por qué es incorrecto | Corrección |
| --- | --- | --- |
| Identificar gobierno del dato con informática | reduce responsabilidades jurídicas y funcionales a una tarea técnica | explicar roles, políticas, calidad, seguridad y ciclo de vida |
| Confundir apertura con ausencia de límites | ignora protección de datos, seguridad y confidencialidad | hablar de apertura responsable y análisis previo |
| Pensar que publicar PDF es dato abierto suficiente | puede ser accesible para lectura, pero no reutilizable automáticamente | priorizar formatos estructurados y legibles por máquina |
| Tratar metadatos como detalle secundario | sin metadatos no se entiende ni se encuentra el dato | vincular metadatos con calidad y reutilización |
| Decir que interoperabilidad es solo conexión entre sistemas | omite significado y acuerdos organizativos | distinguir dimensiones organizativa, semántica y técnica |
| Creer que más datos siempre es mejor | puede aumentar riesgos, ruido y costes | aplicar finalidad, minimización, calidad y valor público |

## 17. Modo tutor: ideas que conviene dominar

### 17.1. Qué significa gobernar datos

Significa decidir de forma institucional cómo se crean, definen, protegen, comparten, corrigen, conservan y publican los datos. Importa porque una Administración moderna depende de datos para tramitar, resolver, informar y evaluar. Se confunde con gestión de bases de datos, pero es más amplio: incluye normas, roles, calidad, semántica, seguridad y reutilización. En examen se reconoce porque la pregunta habla de responsabilidades, ciclo de vida, catálogos, calidad o estrategia de datos.

### 17.2. Qué significa interoperabilidad semántica

Significa que el dato conserva su significado cuando se intercambia. Importa porque las Administraciones usan conceptos jurídicos y administrativos complejos. Se confunde con interoperabilidad técnica, pero esta última solo permite conexión y procesamiento. En examen se reconoce por palabras como vocabulario, modelo de datos, metadatos, definiciones comunes, códigos normalizados o diccionario de datos.

### 17.3. Qué significa calidad del dato

Significa aptitud para el uso previsto. Importa porque los datos públicos pueden afectar derechos, servicios, estadísticas y decisiones. Se confunde con ausencia de errores visibles, pero incluye muchas dimensiones. En examen se reconoce por menciones a exactitud, completitud, actualización, consistencia, duplicidades, trazabilidad o linaje.

### 17.4. Qué significa reutilización

Significa uso de información del sector público para fines distintos de la actividad administrativa pública que originó o custodia esa información. Importa porque genera transparencia, innovación, investigación y servicios. Se confunde con publicidad general o consulta humana. En examen se reconoce por datos abiertos, formatos, licencias, condiciones de uso, catálogos, APIs y conjuntos de alto valor.

## 18. Repaso final del bloque

El dato público es un activo institucional. Su valor depende de que pueda ser entendido, protegido, mantenido, intercambiado y reutilizado. El gobierno del dato crea el marco de responsabilidades y decisiones. La interoperabilidad semántica asegura significado común. La calidad permite confiar en el dato y usarlo para decisiones. La reutilización convierte información pública en valor social, económico y democrático, siempre con límites legales.

Para una respuesta de examen, conviene cerrar con una frase integradora: la Administración orientada al dato no es la que acumula más información, sino la que gobierna mejor sus datos para prestar servicios, proteger derechos, cooperar con otras Administraciones y abrir información pública de forma útil, segura y reutilizable.

## 19. Plan de visuales sugeridos

| Visual | Objetivo didáctico | Descripción |
| --- | --- | --- |
| Mapa de ciclo de vida del dato público | Mostrar continuidad entre diseño, captura, uso, intercambio y reutilización | Diagrama horizontal con fases y controles transversales de calidad, seguridad y metadatos |
| Triángulo de interoperabilidad | Diferenciar organizativa, semántica y técnica | Tres vértices con ejemplos administrativos |
| Matriz de calidad | Memorizar dimensiones | Tabla visual con dimensión, pregunta y ejemplo |
| Flujo de publicación de datos abiertos | Conectar gobierno interno y catálogo externo | Fuente maestra, validación, anonimización, metadatos, licencia, publicación y revisión |
| Semáforo de apertura responsable | Recordar límites | Verde: datos abiertos; ámbar: agregación/anonimización; rojo: protección o restricción legal |

## 20. Fuentes oficiales y de referencia para integrar

Fuentes normativas españolas: Ley 39/2015, Ley 40/2015, Real Decreto 4/2010 sobre Esquema Nacional de Interoperabilidad, Real Decreto 203/2021, Ley 37/2007 sobre reutilización de la información del sector público, Real Decreto 1495/2011, Resolución de 19 de febrero de 2013 por la que se aprueba la Norma Técnica de Interoperabilidad de Reutilización de recursos de la información, Resolución de 3 de octubre de 2012 sobre Catálogo de estándares, Ley Orgánica 3/2018 y Real Decreto 311/2022 sobre Esquema Nacional de Seguridad.

Fuentes de la Unión Europea: Reglamento (UE) 2016/679, Directiva (UE) 2019/1024, Reglamento de Ejecución (UE) 2023/138, Reglamento (UE) 2022/868, Reglamento (UE) 2023/2854 y Reglamento (UE) 2024/903.

Fuentes técnicas y doctrinales de apoyo: portal datos.gob.es, materiales de Iniciativa Aporta sobre datos abiertos y calidad de datos, guías sobre DCAT-AP y DCAT-AP-ES, documentación del marco europeo de interoperabilidad y recursos de interoperabilidad semántica de la Comisión Europea.
