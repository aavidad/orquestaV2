# Fuentes oficiales, normativa, doctrina y trazabilidad

Tema: Arquitecturas distribuidas, microservicios, APIs e integracion de sistemas en la Administracion publica.

Fecha de consulta de referencias principales: 2026-05-20.

## Uso editorial recomendado

Este dosier esta pensado como material de integracion para el tema A1. No sustituye el desarrollo teorico. Sirve para que el redactor principal pueda:

- anclar los conceptos tecnicos en obligaciones juridicas reales;
- separar lo que es mandato normativo de lo que es buena practica tecnica;
- evitar ejemplos privados o dependientes de un proveedor;
- citar fuentes oficiales de forma editorial, sin mostrar URL reales en el texto final;
- construir notas de test con trampas razonables sobre administracion electronica, interoperabilidad, seguridad y APIs.

En el Markdown y el HTML finales conviene citar como "Ley 39/2015", "Ley 40/2015", "Real Decreto 203/2021", "Esquema Nacional de Interoperabilidad", "Esquema Nacional de Seguridad", "Reglamento Interoperable Europe", "Reglamento eIDAS" o "Marco Europeo de Interoperabilidad". Las URL de consulta deben permanecer en el fichero de fuentes o en el archivo local de evidencias, no en el texto visible de estudio.

## Tesis normativa del tema

La Administracion publica no adopta arquitecturas distribuidas, microservicios, APIs o plataformas de integracion por moda tecnologica. Las adopta, cuando procede, porque necesita prestar servicios digitales interoperables, seguros, reutilizables, auditables y resistentes, cumpliendo los derechos de las personas interesadas y las obligaciones de cooperacion entre administraciones.

La idea central para el tema es esta:

> En el sector publico, una arquitectura distribuida correcta no es solo una coleccion de servicios tecnicos. Es una forma de organizar capacidades, datos, expedientes, evidencias, identidades, trazas, documentos, comunicaciones y responsabilidades entre organos y administraciones distintas.

Esta tesis permite unir cuatro planos:

| Plano | Pregunta de examen | Fuente principal | Riesgo si se explica mal |
| --- | --- | --- | --- |
| Juridico-administrativo | Que exige la administracion electronica? | Ley 39/2015, Ley 40/2015, RD 203/2021 | Reducir el tema a informatica generica |
| Interoperabilidad | Como cooperan sistemas y administraciones? | ENI, NTI, EIF, Reglamento Interoperable Europe | Confundir integracion con simple conexion tecnica |
| Seguridad y confianza | Como se protege el servicio y la evidencia? | ENS, eIDAS, RGPD, NIS2, CCN-STIC | Tratar APIs y microservicios sin identidad, trazabilidad ni gestion del riesgo |
| Arquitectura tecnica | Como se disenan servicios distribuidos gobernables? | NIST SP 800-204, OpenAPI, OWASP API, patrones de integracion | Presentar microservicios como receta universal |

## Fuentes normativas espanolas principales

### Ley 39/2015, de Procedimiento Administrativo Comun

Identificador editorial: Ley 39/2015.

Uso en el tema: derechos y obligaciones de relacion electronica, expediente, documentos, copias, notificaciones, actuacion automatizada y consulta de datos.

Puntos que conviene integrar:

- La relacion electronica no es un accesorio del procedimiento, sino una via normal de actuacion para determinados sujetos y una opcion/derecho para otros.
- El articulo 14 es clave para explicar obligados a relacionarse electronicamente, especialmente personas juridicas, entidades sin personalidad, profesionales colegiados en ciertos tramites, empleados publicos por razon de su condicion y representantes de obligados.
- El articulo 28 permite explicar el principio de no aportar documentos que ya obren en poder de la Administracion o que hayan sido elaborados por cualquier Administracion, con matices de consentimiento, oposicion y habilitacion legal. Es la base juridica que justifica plataformas de intermediacion, servicios de consulta y APIs entre administraciones.
- Los articulos sobre registros, archivo, documentos electronicos, copias y expediente justifican que la integracion no solo intercambie "datos sueltos", sino documentos, metadatos, sellos de tiempo, firmas, trazas y evidencias.
- La notificacion electronica y el acceso al expediente obligan a pensar en servicios digitales de extremo a extremo, no en aplicaciones aisladas.

Aplicacion didactica:

La Ley 39/2015 debe aparecer al explicar por que las APIs publicas no pueden limitarse a "dar datos". Deben respetar el procedimiento, la finalidad, la competencia del organo, la proteccion de datos, la trazabilidad y el derecho de la persona interesada a conocer el estado del expediente.

Pregunta de test posible:

| Enunciado | Respuesta esperada | Trampa |
| --- | --- | --- |
| Que precepto conecta directamente la integracion de datos entre administraciones con el derecho de la persona interesada a no aportar documentos ya disponibles? | Articulo 28 de la Ley 39/2015 | Responder solo "ENI", que regula interoperabilidad pero no agota el derecho procedimental |

### Ley 40/2015, de Regimen Juridico del Sector Publico

Identificador editorial: Ley 40/2015.

Uso en el tema: principios de actuacion, funcionamiento electronico, cooperacion, transmisiones de datos, ENI, reutilizacion de sistemas y relaciones interadministrativas.

Puntos que conviene integrar:

- Los principios generales de actuacion publica incluyen eficacia, eficiencia, servicio efectivo a la ciudadania, coordinacion, cooperacion y respeto a la distribucion competencial. Estos principios explican por que las arquitecturas distribuidas no eliminan responsabilidades juridicas.
- El articulo 3 ayuda a conectar arquitectura y administracion: los sistemas deben servir a la eficacia, la coordinacion, la simplicidad y la proximidad al ciudadano.
- El articulo 44 sobre medios electronicos en la actuacion administrativa permite introducir que la tecnologia es soporte de actos, comunicaciones y evidencias con efectos juridicos.
- El articulo 155 es central para transmisiones de datos entre administraciones. Debe usarse para explicar plataformas de intermediacion, consultas de datos, finalidad concreta y seguridad.
- El articulo 156 vincula el Esquema Nacional de Interoperabilidad con criterios y recomendaciones de seguridad, conservacion y normalizacion de la informacion, formatos y aplicaciones.
- El articulo 157 sirve para explicar reutilizacion de sistemas y aplicaciones de titularidad publica, transferencia tecnologica y eficiencia.

Aplicacion didactica:

La Ley 40/2015 permite corregir un error frecuente: pensar que la interoperabilidad es una cuestion meramente tecnica. En realidad, la interoperabilidad administrativa exige que cada administracion conserve sus competencias y responsabilidades, mientras coopera mediante servicios, datos y plataformas comunes.

Pregunta de test posible:

| Enunciado | Respuesta esperada | Trampa |
| --- | --- | --- |
| Que articulo de la Ley 40/2015 se usa habitualmente para fundamentar transmisiones de datos entre administraciones? | Articulo 155 | Confundirlo con el articulo 157, que se orienta a reutilizacion de sistemas y aplicaciones |

### Real Decreto 203/2021, Reglamento de actuacion y funcionamiento del sector publico por medios electronicos

Identificador editorial: RD 203/2021.

Uso en el tema: desarrollo reglamentario de la actuacion electronica, sedes, puntos de acceso, registros, identificacion, firma, archivo, copias, interoperabilidad y funcionamiento digital.

Puntos que conviene integrar:

- Es la referencia reglamentaria para explicar que la administracion digital tiene reglas concretas de funcionamiento, no solo principios abstractos.
- Permite ordenar el discurso sobre sedes electronicas, portales, Punto de Acceso General, carpetas ciudadanas, registros electronicos, asistencia a personas interesadas, representacion, identificacion, firma, archivo y expediente.
- Refuerza que un servicio distribuido debe preservar la experiencia juridica completa: quien actua, ante que organo, con que fecha, con que documento, con que registro, con que notificacion y con que trazabilidad.
- Es util para diferenciar canal, sede, servicio, expediente, documento, registro y archivo.

Aplicacion didactica:

Al hablar de microservicios en la Administracion, conviene advertir que dividir internamente una aplicacion no puede fragmentar la responsabilidad publica frente al ciudadano. La persona interesada no debe sufrir la complejidad de decenas de servicios internos. El diseno debe ocultar la complejidad y preservar una actuacion administrativa comprensible.

Pregunta de test posible:

| Enunciado | Respuesta esperada | Trampa |
| --- | --- | --- |
| Que norma reglamentaria desarrolla el funcionamiento electronico del sector publico estatal tras las Leyes 39/2015 y 40/2015? | Real Decreto 203/2021 | Citar solo el ENS o el ENI, que regulan planos especificos |

### Real Decreto 4/2010, Esquema Nacional de Interoperabilidad

Identificador editorial: ENI / RD 4/2010.

Uso en el tema: interoperabilidad organizativa, semantica y tecnica; condiciones de intercambio; documentos; metadatos; firma; formatos; red de comunicaciones; reutilizacion; normas tecnicas.

Puntos que conviene integrar:

- El ENI es la fuente basica para explicar que interoperabilidad no significa solo "conectar sistemas". Incluye dimensiones organizativa, semantica y tecnica.
- La interoperabilidad organizativa atiende a procesos, responsabilidades y acuerdos entre administraciones.
- La interoperabilidad semantica busca que el significado de los datos intercambiados sea comun y estable.
- La interoperabilidad tecnica cubre formatos, protocolos, interfaces, infraestructuras, redes y estandares.
- El ENI debe conectarse con documentos electronicos, expedientes, politicas de firma, metadatos, formatos reutilizables y conservacion.
- Las Normas Tecnicas de Interoperabilidad concretan aspectos como documento electronico, expediente electronico, digitalizacion, copiado autentico, politica de firma, catalogo de estandares, relacion de modelos de datos, protocolos de intermediacion y reutilizacion de recursos de informacion.

Aplicacion didactica:

Las APIs son una pieza tecnica dentro de un marco mas amplio. Una API puede estar perfectamente documentada y aun asi ser mala para la Administracion si no respeta el significado de los datos, no conserva evidencias, no se alinea con el procedimiento o no permite auditoria.

Pregunta de test posible:

| Enunciado | Respuesta esperada | Trampa |
| --- | --- | --- |
| Cuales son las dimensiones clasicas de la interoperabilidad en el ENI? | Organizativa, semantica y tecnica | Responder "confidencialidad, integridad y disponibilidad", que son propiedades de seguridad |

### Normas Tecnicas de Interoperabilidad

Identificador editorial: NTI.

Uso en el tema: concrecion operativa del ENI.

NTI y guias especialmente utiles para este tema:

| NTI o familia | Uso en el tema | Vinculo con arquitectura distribuida |
| --- | --- | --- |
| Documento electronico | Explicar unidad documental, metadatos y firma | Evita que un microservicio trate documentos como simples ficheros |
| Expediente electronico | Ordenar relaciones entre documentos, indice y evidencias | Conecta workflow, integracion y archivo |
| Digitalizacion de documentos | Entrada de documentos analogicos al circuito digital | Afecta a servicios de registro y tramitacion |
| Copiado autentico | Copias con valor administrativo | Relevante para servicios compartidos de verificacion |
| Politica de firma y certificados | Firma, sellos y validacion | Importante en APIs que producen actos o documentos |
| Catalogo de estandares | Formatos y estandares aceptados | Reduce dependencia de proveedor |
| Protocolos de intermediacion de datos | Consultas y transmisiones entre administraciones | Base directa de APIs de verificacion e intercambio |
| Relacion de modelos de datos | Semantica comun | Impide que cada organo reinvente campos incompatibles |
| Reutilizacion de recursos de informacion | Publicacion y reaprovechamiento | Conecta con datos abiertos y plataformas comunes |

Aplicacion didactica:

Una buena tabla del tema puede comparar "API tecnica" y "servicio interoperable ENI". La API tecnica ofrece una operacion; el servicio interoperable aporta finalidad, modelo de datos, semantica, seguridad, trazabilidad, versionado, responsabilidad y encaje procedimental.

### Real Decreto 311/2022, Esquema Nacional de Seguridad

Identificador editorial: ENS / RD 311/2022.

Uso en el tema: seguridad de sistemas que soportan servicios publicos digitales, gestion del riesgo, principios basicos, requisitos minimos, medidas de seguridad, categorizacion y auditoria.

Puntos que conviene integrar:

- El ENS se aplica a sistemas que sustentan el ejercicio de derechos, el cumplimiento de deberes y la prestacion de servicios publicos digitales.
- Sus principios basicos permiten explicar seguridad integral, gestion del riesgo, prevencion, deteccion, respuesta, conservacion, reevaluacion periodica y funcion diferenciada.
- Sus requisitos minimos se deben traducir en arquitectura: identificacion, autenticacion, autorizacion, control de acceso, registro de actividad, proteccion de comunicaciones, continuidad, adquisicion segura, configuracion, mantenimiento, auditoria y proteccion de informacion.
- En microservicios, el ENS obliga a no confiar ciegamente en la red interna. Se necesitan identidad de servicio, cifrado, control de permisos, segregacion, trazas y gestion de secretos.
- La categorizacion del sistema y las medidas asociadas condicionan arquitectura, despliegue, controles de API y operaciones.

Aplicacion didactica:

El ENS ayuda a explicar que una API publica o interadministrativa no es segura por estar detras de una red privada. Debe tener autenticacion, autorizacion, control de finalidad, trazabilidad, limites de uso, proteccion de comunicaciones y capacidad de auditoria.

Pregunta de test posible:

| Enunciado | Respuesta esperada | Trampa |
| --- | --- | --- |
| Que esquema nacional se centra en la seguridad de los sistemas de informacion del sector publico? | ENS | Confundirlo con el ENI, que se centra en interoperabilidad |

### RGPD y Ley Organica 3/2018

Identificadores editoriales: RGPD y LOPDGDD.

Uso en el tema: tratamiento de datos personales en integraciones, consultas, APIs y servicios distribuidos.

Puntos que conviene integrar:

- Las APIs no eliminan las obligaciones de proteccion de datos. Cada consulta o transmision debe tener base juridica, finalidad, minimizacion, seguridad y trazabilidad.
- Deben distinguirse responsable, encargado, corresponsabilidad, destinatario, cesion, interoperabilidad interna y tratamiento por cuenta de otro.
- La minimizacion de datos es clave para APIs. Un servicio de consulta no debe devolver "todo lo que existe", sino lo necesario para una finalidad concreta.
- Los registros de actividad, la evaluacion de impacto y la privacidad desde el diseno pueden ser necesarios en servicios de gran escala o alto riesgo.
- La integracion debe evitar usos secundarios no previstos y consultas masivas sin causa.

Aplicacion didactica:

Puede plantearse un ejemplo: un ayuntamiento consulta datos de familia numerosa para resolver una bonificacion. El diseno correcto no replica una base de datos completa; expone una verificacion proporcional, registra la finalidad y conserva evidencia de la consulta.

### Real Decreto-ley 12/2018 y Real Decreto 43/2021 sobre seguridad de redes y sistemas

Identificadores editoriales: RDL 12/2018 y RD 43/2021.

Uso en el tema: continuidad, incidentes, servicios esenciales y marco nacional derivado de NIS.

Puntos que conviene integrar:

- Aportan contexto sobre seguridad de redes y sistemas de informacion antes de la plena aplicacion de NIS2.
- Son utiles para explicar incidentes, operadores, medidas de seguridad y notificacion.
- No sustituyen al ENS en el sector publico, pero ayudan a conectar ciberseguridad, continuidad y servicios esenciales.

Nota de actualidad:

La Directiva NIS2 fija un marco europeo actualizado. Para el texto final conviene redactar de forma prudente: "junto al marco nacional vigente y a la evolucion derivada de NIS2", salvo que el redactor confirme una transposicion espanola concreta en BOE en el momento de cierre.

## Fuentes europeas principales

### Reglamento (UE) 2024/903, Interoperable Europe Act

Identificador editorial: Reglamento Interoperable Europe.

Uso en el tema: interoperabilidad transfronteriza de servicios publicos digitales, gobernanza europea, soluciones reutilizables, evaluaciones de interoperabilidad y cooperacion.

Puntos que conviene integrar:

- Es una referencia europea reciente para explicar que la interoperabilidad de los servicios publicos digitales ya no es solo una recomendacion tecnica.
- Refuerza la necesidad de soluciones interoperables, reutilizables y coordinadas entre administraciones.
- Permite conectar el tema con servicios transfronterizos, evaluaciones de interoperabilidad y gobernanza comun.
- Es especialmente util para justificar que una API publica debe ser gobernada: catalogo, documentacion, versionado, condiciones de uso, seguridad y ciclo de vida.

Aplicacion didactica:

El Reglamento permite explicar que la interoperabilidad no termina dentro de una administracion nacional. Un servicio digital puede necesitar validar datos, documentos, identidades o evidencias en otro Estado miembro.

### Marco Europeo de Interoperabilidad, Comunicacion COM(2017) 134

Identificador editorial: EIF 2017.

Uso en el tema: principios de interoperabilidad, capas, gobernanza, servicios publicos europeos, apertura, reutilizacion, seguridad y proteccion de datos.

Puntos que conviene integrar:

- Ofrece una estructura pedagogica muy clara: gobernanza integrada, interoperabilidad legal, organizativa, semantica y tecnica.
- Ayuda a explicar que el ENI espanol tiene correspondencia con enfoques europeos, aunque no sean identicos.
- Sus principios permiten hablar de subsidiariedad y proporcionalidad, apertura, transparencia, reutilizacion, neutralidad tecnologica, portabilidad, orientacion al usuario, inclusion, seguridad y privacidad.

Aplicacion didactica:

Puede usarse para una tabla de comparacion entre capas:

| Capa EIF | Traduccion para APIs publicas |
| --- | --- |
| Legal | La API debe tener habilitacion y finalidad |
| Organizativa | Debe haber responsables, acuerdos, procesos y SLA |
| Semantica | Los datos deben significar lo mismo para emisor y receptor |
| Tecnica | Deben existir interfaces, protocolos, formatos y seguridad |

### Reglamento (UE) 2018/1724, Pasarela Digital Unica

Identificador editorial: Reglamento SDG.

Uso en el tema: acceso transfronterizo a informacion, procedimientos, asistencia y principio de "solo una vez" mediante intercambio de evidencias.

Puntos que conviene integrar:

- Es la fuente europea principal para explicar servicios digitales transfronterizos y el principio de no pedir de nuevo determinada informacion o evidencia si puede intercambiarse de forma controlada.
- Permite hablar del sistema tecnico de "once-only" como integracion entre administraciones, no como repositorio unico centralizado.
- Debe conectarse con identidad electronica, intercambio de evidencias, consentimiento o solicitud del usuario cuando proceda, trazabilidad y confianza.

Aplicacion didactica:

El Reglamento SDG es util para un supuesto practico: una persona solicita un tramite en un Estado miembro y debe aportar una evidencia que obra en otro. El enfoque correcto no es enviar correos con documentos, sino usar un sistema de intercambio de evidencias con identificacion, autorizacion, semantica y registro.

### Reglamento de Ejecucion (UE) 2022/1463, sistema tecnico de solo una vez

Identificador editorial: Reglamento OOTS.

Uso en el tema: detalles operativos del intercambio transfronterizo de evidencias bajo la Pasarela Digital Unica.

Puntos que conviene integrar:

- Conecta directamente con arquitectura de integracion: proveedores de evidencias, solicitantes, intercambio, intermediarios, directorios, identificacion y servicios comunes.
- Es buena fuente para explicar que una integracion publica compleja necesita metadatos, catalogos, validacion y trazabilidad, no solo endpoints.
- Permite explicar que las APIs son contratos entre organizaciones, no simples metodos remotos.

### Reglamento (UE) 910/2014 eIDAS y Reglamento (UE) 2024/1183

Identificadores editoriales: eIDAS y Marco Europeo de Identidad Digital.

Uso en el tema: identificacion electronica, servicios de confianza, firmas, sellos, certificados, sellos de tiempo, entrega electronica certificada, autenticacion de sitios web y cartera europea de identidad digital.

Puntos que conviene integrar:

- La identidad electronica es una dependencia esencial de los servicios distribuidos con efectos juridicos.
- La firma y el sello electronico no son adornos: aportan autenticidad, integridad, no repudio funcional y evidencia juridica cuando procede.
- La validacion de certificados, sellos de tiempo y servicios de confianza debe integrarse en la arquitectura.
- La evolucion hacia carteras de identidad digital aumenta la importancia de APIs de atributos, evidencias y credenciales verificables.

Aplicacion didactica:

Cuando un microservicio genera una resolucion, una copia autentica o una notificacion, debe quedar claro quien actua, con que competencia, en que momento y con que evidencia. eIDAS ayuda a explicar esa capa de confianza.

### Directiva (UE) 2022/2555, NIS2

Identificador editorial: Directiva NIS2.

Uso en el tema: ciberseguridad europea, entidades esenciales e importantes, gestion de riesgos, incidentes, cadena de suministro y gobernanza.

Puntos que conviene integrar:

- NIS2 es relevante para administraciones publicas y proveedores criticos de servicios digitales, con el alcance que determine la transposicion nacional.
- Refuerza gestion del riesgo, medidas tecnicas y organizativas, notificacion de incidentes y seguridad de la cadena de suministro.
- En microservicios y APIs obliga a pensar en dependencias: proveedor cloud, librerias, contenedores, gateways, observabilidad, CI/CD y servicios terceros.

Redaccion prudente:

Usar NIS2 como marco europeo. No afirmar una obligacion espanola concreta no verificada en el BOE si el texto se cierra en una fecha posterior sin nueva comprobacion.

### Directiva (UE) 2019/1024, datos abiertos y reutilizacion de la informacion del sector publico

Identificador editorial: Directiva Open Data.

Uso en el tema: APIs para reutilizacion, formatos abiertos, datos de alto valor y acceso a informacion publica reutilizable.

Puntos que conviene integrar:

- No toda API publica es de procedimiento administrativo. Algunas sirven para publicar informacion reutilizable.
- La reutilizacion exige condiciones claras, formatos adecuados, calidad, metadatos y disponibilidad.
- Debe distinguirse API de datos abiertos de API interadministrativa de verificacion. Tienen finalidades, usuarios, restricciones y controles diferentes.

### Reglamento (UE) 2022/868, Gobernanza de Datos

Identificador editorial: Data Governance Act.

Uso en el tema: reutilizacion de determinadas categorias de datos protegidos del sector publico, intermediacion de datos y gobernanza.

Puntos que conviene integrar:

- Es mas propio del tema de gobierno del dato, pero aqui ayuda a explicar que la integracion tecnica debe estar sometida a condiciones de gobernanza.
- Puede citarse brevemente cuando se explique que compartir datos no equivale a abrir bases sin control.

## Servicios y plataformas publicas espanolas como ejemplos de integracion

Estos ejemplos ayudan a hacer el tema reconocible para un opositor A1. Deben presentarse como patrones funcionales, no como inventario exhaustivo.

### Red SARA

Identificador editorial: Red SARA.

Uso en el tema: red de comunicaciones de las Administraciones Publicas espanolas.

Valor didactico:

- Sirve para explicar la capa de conectividad segura entre administraciones.
- No debe confundirse con una API, una base de datos comun o un sistema de tramitacion.
- Es infraestructura habilitadora: permite comunicacion, acceso a servicios comunes y conectividad interadministrativa.

Trampa de examen:

Red SARA no resuelve por si sola la interoperabilidad semantica ni la habilitacion juridica de una consulta. Aporta una red y servicios de comunicacion, pero la legalidad y el significado de los datos dependen de otros planos.

### Plataforma de Intermediacion de Datos

Identificador editorial: PID / Plataforma de Intermediacion.

Uso en el tema: consulta y verificacion de datos entre administraciones para evitar solicitar documentos a la ciudadania.

Valor didactico:

- Es el ejemplo mas directo para unir Ley 39/2015 articulo 28, Ley 40/2015 articulo 155, ENI y APIs.
- Permite explicar patron de intermediacion: organismo cedente, organismo cesionario, servicio de verificacion, finalidad, consentimiento/oposicion cuando proceda, trazabilidad y control.
- Evita que cada administracion construya integraciones punto a punto con cada cedente.

Trampa de examen:

No debe presentarse como una base de datos central universal. Su logica es intermediacion de consultas y transmisiones con finalidad administrativa.

### Sistema de Interconexion de Registros

Identificador editorial: SIR.

Uso en el tema: intercambio registral entre administraciones.

Valor didactico:

- Permite explicar integracion de oficinas, registros, asientos, documentos y destino administrativo.
- Es buen ejemplo de interoperabilidad organizativa y tecnica.
- Ayuda a diferenciar intercambio de documentos administrativos de consulta de datos estructurados.

Trampa de examen:

SIR no es una herramienta generica de mensajeria ni un bus universal de microservicios. Tiene finalidad registral y reglas de interoperabilidad asociadas.

### Directorio Comun de Unidades Organicas y Oficinas

Identificador editorial: DIR3.

Uso en el tema: identificacion comun de organos, unidades y oficinas.

Valor didactico:

- En integraciones publicas no basta con decir "ministerio", "ayuntamiento" u "organismo". Se necesitan identificadores estables y compartidos.
- DIR3 es clave para enrutar comunicaciones, registros e integraciones con referencia a unidades administrativas.
- Es ejemplo de interoperabilidad semantica y organizativa.

Trampa de examen:

DIR3 no es un sistema de autenticacion ni un registro de personas. Es un directorio de unidades, organos y oficinas.

### Carpeta Ciudadana, Punto de Acceso General y sedes electronicas

Identificadores editoriales: Carpeta Ciudadana, PAG, sedes electronicas.

Uso en el tema: experiencia integrada de servicios publicos digitales.

Valor didactico:

- Permiten explicar que la ciudadania no debe conocer la arquitectura interna.
- Una vista unificada puede depender de multiples sistemas distribuidos: identidad, expedientes, notificaciones, registros, certificados, consultas y servicios sectoriales.
- El diseno de APIs debe pensar en composicion de servicios sin romper competencias ni responsabilidad.

Trampa de examen:

Una interfaz unica no significa una base de datos unica ni una competencia unica. Puede ser una capa de agregacion sobre servicios de varias administraciones.

## Doctrina tecnica y estandares de apoyo

### NIST SP 800-204, seguridad en sistemas basados en microservicios

Identificador editorial: NIST SP 800-204.

Uso en el tema: explicar microservicios con rigor sin caer en publicidad tecnologica.

Puntos que conviene integrar:

- Microservicios implican descomposicion de una aplicacion en servicios pequenos, desplegables de forma independiente y comunicados por interfaces ligeras.
- La seguridad debe contemplar comunicacion servicio a servicio, autenticacion, autorizacion, descubrimiento, gateway, observabilidad y gestion de secretos.
- La segmentacion ayuda a limitar fallos, pero aumenta complejidad operativa.
- El patron no es siempre adecuado: exige madurez en automatizacion, monitorizacion, pruebas, despliegue y gobierno.

Aplicacion didactica:

El tema debe evitar decir que microservicios son "mejores" que monolitos. La respuesta A1 madura es: son utiles cuando hay dominios bien delimitados, equipos capaces, necesidad de evolucion independiente y operacion automatizada; pueden ser contraproducentes si solo fragmentan un sistema sin gobernanza.

### NIST SP 800-204A, service mesh

Identificador editorial: NIST SP 800-204A.

Uso en el tema: explicar malla de servicios como capa de comunicacion, seguridad y observabilidad entre microservicios.

Puntos que conviene integrar:

- Una malla de servicios puede externalizar funciones transversales: cifrado entre servicios, politicas, retries, timeouts, trazas y control de trafico.
- No sustituye la autorizacion de negocio ni la gobernanza administrativa.
- Puede ser excesiva para sistemas pequenos.

Aplicacion didactica:

Usar service mesh como ejemplo avanzado, no como requisito universal. En administraciones con sistemas criticos puede aportar trazabilidad y control, pero tambien introduce dependencia tecnologica y complejidad.

### NIST SP 800-190, seguridad en contenedores

Identificador editorial: NIST SP 800-190.

Uso en el tema: riesgos y controles de contenedores en despliegues modernos.

Puntos que conviene integrar:

- Riesgos de imagenes, registros, orquestadores, configuracion, secretos y aislamiento.
- Necesidad de escaneo, firma o verificacion de imagenes, minimo privilegio, actualizacion y control de runtime.
- Relacion directa con microservicios, CI/CD y despliegue cloud o hibrido.

Aplicacion didactica:

Cuando se mencione Kubernetes, contenedores o plataformas cloud, se debe hacer con foco en gobernanza y seguridad. No es necesario explicar comandos ni configuraciones de producto.

### OWASP API Security Top 10

Identificador editorial: OWASP API Security.

Uso en el tema: doctrina de riesgos frecuentes de APIs.

Puntos que conviene integrar:

- Autorizacion rota a nivel de objeto o funcion.
- Autenticacion debil.
- Exposicion excesiva de datos.
- Falta de limitacion de recursos.
- Consumo inseguro de APIs de terceros.
- Gestion deficiente de inventario y versionado.

Aplicacion didactica:

OWASP es util para notas de test y errores frecuentes, pero no debe presentarse como norma juridica. Conviene llamarlo "referencia tecnica ampliamente utilizada".

### OpenAPI Specification

Identificador editorial: OpenAPI.

Uso en el tema: contratos de APIs REST/HTTP.

Puntos que conviene integrar:

- Permite describir endpoints, operaciones, parametros, esquemas, respuestas y seguridad.
- Facilita generacion de documentacion, validacion, pruebas y gobernanza.
- No garantiza por si sola interoperabilidad administrativa ni seguridad.

Aplicacion didactica:

OpenAPI puede aparecer en la parte tecnica, junto a contratos, versionado, compatibilidad hacia atras y pruebas de contrato.

### W3C, JSON-LD, RDF, XML, URI y web semantica

Identificador editorial: W3C.

Uso en el tema: semantica, identificadores, datos enlazados y formatos.

Puntos que conviene integrar:

- La interoperabilidad semantica requiere vocabularios, codigos, identificadores y modelos compartidos.
- En integraciones administrativas puede haber XML, JSON, CSV, RDF o JSON-LD segun finalidad y contexto.
- No hay que absolutizar un formato. Lo relevante es el contrato, la semantica, el ciclo de vida y el cumplimiento normativo.

## Definiciones de autoridad para incorporar al tema

Estas definiciones estan redactadas como material propio. Pueden adaptarse al texto final.

### Arquitectura distribuida

Una arquitectura distribuida organiza una solucion en componentes que se ejecutan en nodos, procesos o sistemas distintos y que cooperan mediante comunicaciones de red. En la Administracion publica, esta distribucion suele coincidir con competencias, organos, plataformas comunes, sedes, registros, servicios sectoriales y sistemas heredados. Su ventaja es permitir evolucion, escalabilidad, resiliencia y cooperacion; su riesgo es introducir latencia, fallos parciales, complejidad de seguridad y problemas de consistencia.

### Microservicio

Un microservicio es una unidad de software pequena y autonoma, alineada con una capacidad de negocio o dominio, que se despliega y evoluciona de forma independiente y se comunica mediante interfaces bien definidas. No es simplemente "una clase remota" ni "un endpoint pequeno". En el sector publico, un microservicio solo es adecuado si mantiene trazabilidad, control de acceso, responsabilidad funcional y coherencia con el procedimiento.

### API

Una API es un contrato de acceso a una capacidad o informacion de un sistema. Define operaciones, datos de entrada, respuestas, errores, politicas de seguridad y condiciones de uso. En la Administracion, una API no debe verse como un canal tecnico neutro: puede materializar una consulta de datos, una transmision interadministrativa, una actuacion automatizada o una publicacion de informacion reutilizable.

### Integracion de sistemas

La integracion de sistemas es el conjunto de decisiones, contratos, mecanismos y controles que permiten que sistemas distintos cooperen de forma fiable. Incluye conectividad, autenticacion, autorizacion, transformacion de datos, orquestacion, mensajeria, monitorizacion, trazabilidad, gestion de errores y gobierno del ciclo de vida.

### Interoperabilidad

La interoperabilidad es la capacidad de organizaciones y sistemas para compartir informacion y conocimiento mediante procesos, datos y tecnologias compatibles. En el marco publico se expresa en planos organizativo, semantico y tecnico, y se orienta a que los servicios puedan prestarse de forma coordinada sin exigir a la ciudadania aportar de nuevo informacion ya disponible.

### Intermediacion de datos

La intermediacion de datos es un patron de intercambio en el que una plataforma o servicio facilita consultas o transmisiones entre organismos, con finalidad administrativa, controles de seguridad y registro. Es distinta de la replicacion masiva de bases de datos y de la publicacion abierta de datos.

### Servicio comun

Un servicio comun es una capacidad compartida por varias administraciones u organos para evitar duplicidades y favorecer eficiencia e interoperabilidad. Puede estar orientado a identidad, firma, notificaciones, registro, intermediacion, pagos, archivo o comunicaciones. Su uso exige respetar competencias, responsabilidades y acuerdos de prestacion.

### Gateway de API

Un gateway de API es una pieza de entrada que centraliza funciones transversales como autenticacion, autorizacion inicial, control de trafico, enrutamiento, versionado, cuotas, registro y observabilidad. No debe confundirse con la logica de negocio ni con la decision juridica de autorizar una consulta.

### Bus de servicios y mensajeria

Un bus de servicios o plataforma de mensajeria permite intercambio de mensajes, eventos o documentos entre sistemas. Es util cuando se necesita desacoplamiento, asincronia o integracion de sistemas heterogeneos. En el sector publico debe combinarse con trazabilidad, idempotencia, gestion de errores y evidencias.

### Evento

Un evento representa algo que ha ocurrido y que puede interesar a otros sistemas: registro presentado, expediente iniciado, documento incorporado, notificacion puesta a disposicion, pago confirmado o resolucion emitida. El evento no debe sustituir al acto administrativo ni al documento cuando estos sean juridicamente necesarios.

### Consistencia eventual

La consistencia eventual describe sistemas distribuidos en los que las replicas o vistas pueden no actualizarse simultaneamente, aunque convergen si no se producen nuevas modificaciones. Es aceptable para ciertos usos informativos, pero puede ser inadecuada para decisiones administrativas que requieren estado exacto, plazo o evidencia inmediata.

## Mapa de trazabilidad por bloques del tema

| Bloque del futuro tema | Fuentes principales | Ideas que deben aparecer | Ejemplo recomendado |
| --- | --- | --- | --- |
| Orientacion de examen | Ley 39, Ley 40, ENI, ENS, RD 203 | No es tema de tecnologia aislada; une procedimiento, interoperabilidad y seguridad | Pregunta que compare ENI y ENS |
| Contexto de administracion digital | Ley 39, Ley 40, RD 203 | Derechos, obligados, expediente, registro, sede, actuacion electronica | Tramite que consulta datos y notifica electronicamente |
| Arquitecturas distribuidas | NIST, ENS, ENI, EIF | Fallos parciales, resiliencia, acoplamiento, trazabilidad | Carpeta que compone datos de varios sistemas |
| Microservicios | NIST SP 800-204, ENS, CCN-STIC | Autonomia, dominio, despliegue, seguridad servicio a servicio | Servicio de pagos separado de expediente |
| APIs | OpenAPI, OWASP, ENI, Ley 40 art. 155, RGPD | Contrato, versionado, autenticacion, finalidad, minimizacion | API de verificacion de datos |
| Integracion sincronica | ENI, PID, SARA, ENS | Consulta inmediata, errores, timeouts, autorizacion | Validacion de identidad o certificado |
| Integracion asincrona | SIR, expedientes, eventos, mensajeria | Mensajes, colas, idempotencia, reintentos, acuse | Intercambio registral |
| Interoperabilidad semantica | ENI, NTI, EIF, DIR3 | Modelos de datos, codigos, significados comunes | Codigos de unidad organica |
| Seguridad | ENS, RGPD, eIDAS, NIS2 | Riesgo, identidad, firma, trazas, cadena de suministro | API con firma, token, registro y auditoria |
| Plataformas comunes | PAe, SARA, PID, SIR, DIR3 | Servicios comunes como habilitadores | Evitar pedir certificado de empadronamiento si procede consulta |
| Gobierno de APIs | ENI, ENS, EIF, OpenAPI, OWASP | Catalogo, ciclo de vida, versionado, SLA, retirada | Deprecacion de version de API |
| Supuestos practicos | Ley 39 art. 28, Ley 40 art. 155, RGPD, ENS | Finalidad, base juridica, minimizacion, evidencia | Bonificacion municipal con consulta a otra administracion |
| Errores frecuentes | Todas | No confundir red, API, bus, base de datos, ENI, ENS | Preguntas de descarte |
| Repaso final | Todas | 5-7 ideas fuerza | Mapa mental de capas |

## Diferencias clave para tablas del tema

### ENI frente a ENS

| Aspecto | ENI | ENS |
| --- | --- | --- |
| Pregunta principal | Como interoperan sistemas y administraciones? | Como se protegen los sistemas de informacion? |
| Plano dominante | Organizativo, semantico y tecnico | Seguridad, riesgo, medidas y auditoria |
| Ejemplos | Documento electronico, expediente, formatos, NTI, modelos de datos | Autenticacion, control de acceso, registro, continuidad, proteccion de comunicaciones |
| Error frecuente | Creer que solo trata protocolos | Creer que solo trata ciberseguridad tecnica sin gestion |

### API interadministrativa frente a API de datos abiertos

| Aspecto | API interadministrativa | API de datos abiertos |
| --- | --- | --- |
| Finalidad | Tramitar, verificar o transmitir datos para un procedimiento o servicio publico | Reutilizacion de informacion publica |
| Usuarios | Organos o sistemas autorizados | Reutilizadores, ciudadania, empresas, investigacion |
| Datos | Pueden incluir datos personales o protegidos si hay base juridica | Normalmente datos publicables, anonimizados o sin restricciones indebidas |
| Control | Fuerte: identidad, autorizacion, finalidad, trazabilidad | Condiciones de reutilizacion, disponibilidad, metadatos, limites razonables |
| Fuente normativa | Ley 39, Ley 40, RGPD, ENI, ENS | Directiva Open Data, normativa de reutilizacion, ENI |

### Integracion punto a punto frente a plataforma comun

| Aspecto | Punto a punto | Plataforma comun |
| --- | --- | --- |
| Ventaja | Rapida para casos pequenos | Escalable para multiples organismos |
| Riesgo | Multiplica acoplamientos y contratos | Requiere gobierno, catalogo y operacion estable |
| Uso adecuado | Integracion acotada y justificada | Servicios compartidos o consultas recurrentes |
| Ejemplo | Dos sistemas intercambian un mensaje sectorial | Plataforma de Intermediacion de Datos |

### REST/HTTP frente a mensajeria asincrona

| Aspecto | REST/HTTP sincronico | Mensajeria asincrona |
| --- | --- | --- |
| Modelo | Peticion-respuesta | Mensajes, eventos o colas |
| Mejor para | Consulta inmediata, validacion, operaciones simples | Procesos largos, desacoplamiento, resiliencia |
| Riesgo | Timeouts, dependencia en cadena, fragilidad si se encadenan servicios | Duplicados, orden, idempotencia, seguimiento de estado |
| Ejemplo publico | Consulta de dato de verificacion | Intercambio registral o tramitacion por eventos |

### Monolito modular frente a microservicios

| Aspecto | Monolito modular | Microservicios |
| --- | --- | --- |
| Despliegue | Una unidad principal | Multiples servicios independientes |
| Complejidad operativa | Menor | Mayor |
| Escalado | Conjunto completo o modulos internos | Por servicio o capacidad |
| Riesgo | Acoplamiento interno, ciclos de version | Red distribuida, observabilidad, seguridad servicio a servicio |
| Criterio publico | Valido si es mantenible y trazable | Valido si hay madurez operativa y dominios claros |

## Notas de test separadas

Estas notas deben quedar separadas de la teoria en el tema final.

1. Si se pregunta por el derecho a no aportar documentos ya disponibles, la respuesta debe mirar a la Ley 39/2015, especialmente articulo 28, no solo al ENI.
2. Si se pregunta por transmisiones de datos entre administraciones, la referencia clave es Ley 40/2015 articulo 155.
3. Si se pregunta por dimensiones de interoperabilidad, la respuesta esperada es organizativa, semantica y tecnica.
4. Si se pregunta por seguridad de sistemas del sector publico, el esquema es el ENS, no el ENI.
5. Si se pregunta por formatos, documentos, expedientes, metadatos y normas tecnicas, el marco es ENI y NTI.
6. Red SARA es red de comunicaciones e infraestructura, no base de datos unica.
7. La Plataforma de Intermediacion evita pedir documentos al ciudadano cuando procede, pero no autoriza consultas indiscriminadas.
8. Una API con OpenAPI no es automaticamente interoperable en sentido administrativo.
9. Un gateway de API no sustituye la autorizacion de negocio ni la base juridica de una consulta.
10. Microservicios no son obligatorios ni siempre mejores; exigen gobierno, observabilidad, seguridad y automatizacion.
11. La consistencia eventual puede ser aceptable en vistas informativas, pero peligrosa en actos, plazos o decisiones que requieren estado exacto.
12. eIDAS se relaciona con identidad electronica y servicios de confianza; no regula por si solo todo el procedimiento administrativo.
13. NIS2 es marco europeo de ciberseguridad; conviene comprobar transposicion nacional antes de afirmar detalles internos espanoles.
14. Datos abiertos no equivalen a datos personales disponibles por API.
15. Reutilizacion de aplicaciones publicas y reutilizacion de informacion publica son ideas relacionadas pero juridicamente distintas.

## Supuestos practicos con fuentes

### Supuesto 1: bonificacion municipal con consulta de datos

Situacion:

Un ayuntamiento tramita una bonificacion en una tasa. Para resolverla necesita comprobar una condicion familiar o economica que ya consta en otra administracion.

Fuentes:

- Ley 39/2015: derecho a no aportar documentos ya disponibles y reglas de procedimiento.
- Ley 40/2015: transmision de datos entre administraciones.
- ENI y NTI de intermediacion: interoperabilidad y modelos de intercambio.
- RGPD/LOPDGDD: base juridica, minimizacion y finalidad.
- ENS: autenticacion, control de acceso y registro de actividad.

Resolucion tecnica esperada:

- No replicar toda la base de datos externa.
- Usar un servicio de consulta o verificacion proporcionado por plataforma autorizada.
- Registrar finalidad, organo solicitante, usuario o sistema, fecha, resultado y evidencia.
- Devolver solo el dato necesario para resolver.
- Gestionar errores, indisponibilidad y alternativa procedimental.

Errores:

- Hacer consultas masivas por comodidad.
- Guardar mas datos de los necesarios.
- Creer que la red privada ya basta como autorizacion.
- No documentar version del servicio ni respuesta.

### Supuesto 2: expediente electronico compuesto por servicios

Situacion:

Una consejeria moderniza un procedimiento con registro electronico, validacion de identidad, consulta de datos, generacion de resolucion, notificacion y archivo.

Fuentes:

- Ley 39/2015 y RD 203/2021: procedimiento, registro, expediente, notificacion y archivo.
- ENI/NTI: documento, expediente, metadatos, firma, conservacion.
- ENS: seguridad y continuidad.
- eIDAS: identidad, firma, sello y servicios de confianza.

Resolucion tecnica esperada:

- Separar capacidades internas sin romper la unidad del expediente.
- Mantener indice, documentos, metadatos, firmas y trazas.
- Controlar identidad de persona y de organo actuante.
- Notificar con evidencia y gestionar plazos.
- Archivar de forma interoperable y conservable.

Errores:

- Tratar el expediente como una carpeta de ficheros.
- Perder metadatos al pasar entre servicios.
- Usar eventos tecnicos como sustituto de actos administrativos.
- No contemplar indisponibilidad o reintentos.

### Supuesto 3: API publica de datos abiertos

Situacion:

Una administracion publica un conjunto de datos estadisticos mediante API para reutilizacion.

Fuentes:

- Directiva Open Data y normativa espanola de reutilizacion.
- ENI: formatos, estandares, reutilizacion de recursos de informacion.
- ENS: disponibilidad y proteccion de la infraestructura.
- RGPD: anonimato, agregacion y ausencia de datos personales identificables si procede.

Resolucion tecnica esperada:

- Publicar metadatos, condiciones de reutilizacion y versionado.
- Ofrecer formatos abiertos y documentacion.
- Controlar disponibilidad, limites razonables y cambios.
- Distinguir datos agregados de datos personales.

Errores:

- Mezclar datos abiertos con consulta administrativa autenticada.
- Publicar identificadores indirectamente reidentificables sin analisis.
- No versionar cambios de estructura.

### Supuesto 4: sustitucion de integraciones punto a punto por plataforma

Situacion:

Un ministerio detecta decenas de integraciones directas con comunidades autonomas para consultar datos similares.

Fuentes:

- Ley 40/2015: cooperacion, transmision de datos y reutilizacion.
- ENI/NTI: interoperabilidad organizativa, semantica y tecnica.
- ENS: seguridad comun.
- EIF: gobernanza y reutilizacion.

Resolucion tecnica esperada:

- Inventariar servicios, finalidades y consumidores.
- Definir modelo comun de datos y catalogo de servicios.
- Establecer acuerdos, politicas de acceso, trazabilidad y SLA.
- Migrar gradualmente evitando ruptura de procedimientos.

Errores:

- Crear un bus sin gobierno de datos.
- Mantener semanticas distintas bajo el mismo nombre de campo.
- No planificar retirada de versiones antiguas.

## Errores frecuentes que conviene explotar en el tema

1. Confundir interoperabilidad con integracion tecnica. La integracion conecta; la interoperabilidad asegura cooperacion con significado, responsabilidad y reglas.
2. Confundir ENI y ENS. El primero ordena interoperabilidad; el segundo seguridad.
3. Presentar microservicios como solucion universal. Son una opcion arquitectonica que requiere madurez y no elimina obligaciones juridicas.
4. Pensar que una API equivale a autorizacion. La API es el canal; la autorizacion depende de competencia, base juridica, finalidad y permisos.
5. Identificar Red SARA con base de datos unica. Es infraestructura de comunicaciones.
6. Considerar la Plataforma de Intermediacion como almacen central de todos los datos. Es un mecanismo de intermediacion y consulta controlada.
7. Creer que publicar datos abiertos permite publicar datos personales. La reutilizacion tiene limites y exige analisis.
8. Olvidar versionado y compatibilidad. En administraciones, romper una API puede romper procedimientos de terceros.
9. No distinguir identidad de usuario e identidad de sistema. Ambas pueden ser necesarias.
10. Omitir trazabilidad. Sin trazas no hay control, auditoria ni defensa ante incidentes o recursos.
11. Confundir evento tecnico con acto administrativo. Un evento puede informar de un acto, pero no lo sustituye si el ordenamiento exige documento o resolucion.
12. Sobrecargar el tema con marcas comerciales. El enfoque A1 debe ser conceptual, normativo y transferible.

## Plan de visuales recomendado

### Visual 1: capas de interoperabilidad publica

Formato: SVG local.

Contenido:

- Capa juridica: Ley 39, Ley 40, RD 203, RGPD.
- Capa organizativa: organos, competencias, acuerdos, procesos.
- Capa semantica: modelos de datos, codigos, DIR3, metadatos.
- Capa tecnica: APIs, mensajeria, redes, formatos.
- Capa de seguridad transversal: ENS, identidad, firma, trazas.

Uso:

Colocarlo al inicio tras el mapa del tema. Debe ser horizontal o responsive con scroll.

### Visual 2: patron de intermediacion de datos

Formato: SVG local.

Contenido:

- Organismo solicitante.
- Plataforma de intermediacion.
- Organismo cedente.
- Controles: finalidad, autorizacion, trazabilidad, minimizacion.
- Resultado: verificacion o dato necesario.

Uso:

Apoyar Ley 39 articulo 28 y Ley 40 articulo 155.

### Visual 3: API gateway y microservicios en administracion

Formato: SVG local.

Contenido:

- Usuario/sistema consumidor.
- Sede o aplicacion.
- Gateway.
- Servicios: identidad, expediente, pagos, notificaciones, archivo.
- Observabilidad y seguridad transversal.

Uso:

Explicar que la division interna no debe trasladarse al ciudadano.

### Visual 4: sincrono frente a asincrono

Formato: tabla visual o SVG local.

Contenido:

- Consulta inmediata: peticion-respuesta.
- Proceso largo: mensaje-evento-cola.
- Riesgos: timeout frente a duplicado/reintento.

Uso:

Relacionar APIs, mensajeria, SIR y procesos administrativos.

## Fuentes para fichero fuentes.md

La tabla siguiente puede trasladarse a `fuentes.md`. En el texto visible final se recomienda ocultar enlaces reales y mantener una cita editorial limpia.

| ID | Fuente | Tipo | Uso principal | Localizador de consulta |
| --- | --- | --- | --- | --- |
| ES-L39 | Ley 39/2015, de Procedimiento Administrativo Comun de las Administraciones Publicas | Ley espanola | Derechos, relacion electronica, documentos, expediente, articulo 28 | BOE-A-2015-10565 |
| ES-L40 | Ley 40/2015, de Regimen Juridico del Sector Publico | Ley espanola | Cooperacion, transmisiones de datos, ENI, reutilizacion, articulos 155-157 | BOE-A-2015-10566 |
| ES-RD203 | Real Decreto 203/2021 | Reglamento espanol | Funcionamiento electronico, sedes, registros, expediente, archivo | BOE-A-2021-5032 |
| ES-ENI | Real Decreto 4/2010, Esquema Nacional de Interoperabilidad | Reglamento espanol | Interoperabilidad organizativa, semantica y tecnica; NTI | BOE-A-2010-1331 |
| ES-ENS | Real Decreto 311/2022, Esquema Nacional de Seguridad | Reglamento espanol | Seguridad de sistemas publicos, riesgo, medidas, auditoria | BOE-A-2022-7191 |
| ES-LOPDGDD | Ley Organica 3/2018 | Ley organica espanola | Proteccion de datos personales y derechos digitales | BOE-A-2018-16673 |
| ES-NIS | Real Decreto-ley 12/2018 y Real Decreto 43/2021 | Normativa espanola | Seguridad de redes y sistemas, incidentes | BOE-A-2018-12257; BOE-A-2021-1192 |
| ES-PAE-ENI | Portal de Administracion Electronica, ENI y NTI | Fuente oficial administrativa | Guias, NTI, catalogo de estandares | PAe / administracionelectronica.gob.es |
| ES-PAE-PID | Plataforma de Intermediacion de Datos | Servicio comun | Consultas y verificacion de datos | PAe / administracionelectronica.gob.es |
| ES-PAE-SIR | Sistema de Interconexion de Registros | Servicio comun | Intercambio registral | PAe / administracionelectronica.gob.es |
| ES-PAE-SARA | Red SARA | Servicio comun | Comunicaciones interadministrativas | PAe / administracionelectronica.gob.es |
| ES-PAE-DIR3 | Directorio Comun DIR3 | Servicio comun | Identificacion de unidades, organos y oficinas | PAe / CTT |
| EU-IEA | Reglamento (UE) 2024/903, Interoperable Europe | Reglamento europeo | Interoperabilidad transfronteriza y gobernanza | CELEX 32024R0903 |
| EU-EIF | Marco Europeo de Interoperabilidad, COM(2017) 134 | Comunicacion de la Comision | Principios y capas de interoperabilidad | COM(2017) 134 final |
| EU-SDG | Reglamento (UE) 2018/1724, Pasarela Digital Unica | Reglamento europeo | Procedimientos transfronterizos y once-only | CELEX 32018R1724 |
| EU-OOTS | Reglamento de Ejecucion (UE) 2022/1463 | Reglamento de ejecucion | Sistema tecnico de solo una vez | CELEX 32022R1463 |
| EU-EIDAS | Reglamento (UE) 910/2014 y Reglamento (UE) 2024/1183 | Reglamentos europeos | Identidad electronica y servicios de confianza | CELEX 32014R0910; CELEX 32024R1183 |
| EU-NIS2 | Directiva (UE) 2022/2555 | Directiva europea | Ciberseguridad y gestion del riesgo | CELEX 32022L2555 |
| EU-OPEN-DATA | Directiva (UE) 2019/1024 | Directiva europea | Datos abiertos y reutilizacion | CELEX 32019L1024 |
| EU-DGA | Reglamento (UE) 2022/868 | Reglamento europeo | Gobernanza de datos | CELEX 32022R0868 |
| NIST-204 | NIST SP 800-204 | Guia tecnica oficial | Seguridad en microservicios | NIST Computer Security Resource Center |
| NIST-204A | NIST SP 800-204A | Guia tecnica oficial | Service mesh y microservicios | NIST CSRC |
| NIST-190 | NIST SP 800-190 | Guia tecnica oficial | Seguridad de contenedores | NIST CSRC |
| OWASP-API | OWASP API Security Top 10 | Referencia tecnica | Riesgos frecuentes de APIs | OWASP Foundation |
| OAS-OPENAPI | OpenAPI Specification | Especificacion tecnica | Contratos de APIs | OpenAPI Initiative |
| W3C-SEM | W3C RDF, JSON-LD, URI, web architecture | Estandares web | Semantica e identificadores | W3C |

## Recomendaciones de redaccion final

1. Abrir el tema con la idea de que la arquitectura tecnica esta subordinada al servicio publico, al procedimiento y a los derechos de la ciudadania.
2. Presentar definiciones antes de entrar en patrones complejos.
3. Explicar primero arquitectura distribuida, despues microservicios y despues APIs. Asi se evita que el lector confunda parte y todo.
4. Intercalar normativa en el desarrollo, no concentrarla en un bloque aislado.
5. Usar tablas para diferenciar conceptos que se confunden en test: ENI/ENS, API/datos abiertos, SARA/PID/SIR, monolito/microservicios.
6. Situar las notas de test fuera de la teoria principal, con formato diferenciado.
7. No incluir un banco completo de preguntas dentro del tema. Solo muestra progresiva.
8. Evitar marcas y productos salvo como ejemplos genericos no dependientes.
9. Al hablar de cloud, contenedores o Kubernetes, mantener el foco en arquitectura, seguridad y gobierno, no en administracion de sistemas.
10. Cerrar con un repaso de capas: procedimiento, organizacion, semantica, tecnica, seguridad y operacion.

## Huecos que debe validar el redactor principal

1. Confirmar si, en la fecha de cierre del tema, Espana ha publicado en BOE una norma nueva de transposicion o desarrollo de NIS2 que deba sustituir la redaccion prudente.
2. Confirmar si el ambito del temario exige citar normativa autonomica concreta. Para este dosier se ha priorizado normativa estatal y europea.
3. Revisar si el paquete final del tema usa una lista cerrada de fuentes OPES o una plantilla de `fuentes.md` ya existente.
4. Si se generan visuales, verificar que todos los SVG son locales, responsivos y no decorativos.
5. Si se genera HTML, comprobar que las notas de test quedan ocultables y que el banco completo queda fuera del HTML.
