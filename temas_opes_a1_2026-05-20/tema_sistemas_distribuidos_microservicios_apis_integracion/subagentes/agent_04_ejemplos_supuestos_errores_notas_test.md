# Material parcial: ejemplos, supuestos prácticos, errores frecuentes y notas de test

## Criterio de integración editorial

Este material está pensado para insertarse en un tema A1 sobre arquitecturas distribuidas, microservicios, APIs e integración de sistemas en la Administración pública. No sustituye al desarrollo teórico principal: lo complementa con ejemplos, casos guiados, advertencias de examen y preguntas de recuperación. La recomendación editorial es colocarlo después de explicar los conceptos base de arquitectura distribuida, contrato de API, integración síncrona y asíncrona, interoperabilidad, seguridad y gobierno de servicios.

Conviene mantener separadas las notas de test de la teoría. En el cuerpo principal pueden aparecer los ejemplos narrativos y los supuestos prácticos. Las trampas, distractores y fórmulas de reconocimiento rápido deben ir en cajas diferenciadas de "nota de test" o en la sección final de enfoque de examen.

Las referencias normativas que sostienen los ejemplos deben citarse de forma editorial, sin convertir el tema en un repertorio de artículos. Para este bloque bastan referencias al marco de procedimiento y régimen jurídico electrónico, al Esquema Nacional de Interoperabilidad, al Esquema Nacional de Seguridad, al Reglamento de actuación y funcionamiento del sector público por medios electrónicos, al marco europeo de interoperabilidad y a la normativa europea sobre identificación electrónica, servicios de confianza y reutilización de la información pública cuando el caso lo requiera.

## Ejemplos trabajados

### Ejemplo 1. Del trámite aislado al servicio distribuido

Un ayuntamiento dispone de una aplicación antigua para gestionar licencias de obra menor. La aplicación tiene base de datos propia, genera documentos en una carpeta compartida y exige al ciudadano aportar certificados que ya obran en poder de otra Administración. Desde el punto de vista del usuario, el trámite parece digital porque se inicia mediante un formulario web. Desde el punto de vista arquitectónico, sin embargo, no es todavía un servicio público digital integrado: el formulario solo es una fachada sobre un procedimiento fragmentado.

La mejora no consiste únicamente en "hacer una API". El rediseño debe separar capacidades: presentación de solicitudes, identificación y firma, comprobación de requisitos, consulta de datos externos, cálculo de tasas, generación de expediente, notificación, archivo y seguimiento. Algunas capacidades pueden prestarse mediante servicios comunes o plataformas externas. Otras serán propias del ayuntamiento. El valor de la arquitectura distribuida aparece cuando cada capacidad tiene una responsabilidad clara, un contrato técnico estable y un mecanismo de trazabilidad que permite reconstruir qué se consultó, cuándo, con qué base jurídica y con qué resultado.

En este ejemplo, una API interna puede exponer el estado del expediente a la carpeta ciudadana municipal; un conector puede consultar datos de intermediación cuando proceda; un servicio de notificación puede desacoplar la emisión del acto de su puesta a disposición; y un módulo de archivo puede recibir documentos y metadatos normalizados. El trámite deja de ser una pantalla única conectada a una base de datos local y pasa a ser una composición de servicios.

La enseñanza para el examen es doble. Primero, una arquitectura distribuida no se justifica por moda tecnológica, sino por la necesidad de integrar competencias, sistemas y responsabilidades diferentes. Segundo, distribuir sin gobierno puede empeorar el problema: si cada área crea su propia API sin catálogo, versionado, seguridad ni semántica común, el resultado será más opaco que el sistema inicial.

### Ejemplo 2. Carpeta ciudadana como agregador y no como propietario de todos los datos

Una carpeta ciudadana permite que una persona consulte expedientes, notificaciones, citas, justificantes y datos de distintas unidades. Un error habitual es imaginarla como una gran base de datos central donde todas las Administraciones copian su información. Ese diseño generaría duplicidades, conflictos de actualización, riesgos de protección de datos y una responsabilidad difícil de delimitar.

El enfoque más razonable es tratar la carpeta como un agregador de servicios. Cada sistema responsable mantiene la fuente autorizada de sus datos. La carpeta consulta APIs o servicios de interoperabilidad para mostrar información al ciudadano, aplicando control de acceso, trazabilidad, límites de uso y reglas de presentación. Si un expediente pertenece a una consejería, el sistema de esa consejería conserva la responsabilidad funcional del expediente. La carpeta no "se convierte" en el expediente: ofrece una vista integrada.

El patrón técnico puede combinar llamadas síncronas para información inmediata con mecanismos asíncronos para avisos, cambios de estado o sincronización de índices. La parte crítica es no romper la frontera de responsabilidad. Una vista agregada no debe modificar datos maestros si no existe un contrato explícito para hacerlo.

Este ejemplo ayuda a distinguir integración de centralización. Integrar no siempre significa reunir todos los datos en un único repositorio. En muchos servicios públicos, integrar significa respetar la fuente competente, exponer contratos claros y permitir que el usuario vea una experiencia coherente sin que la Administración pierda control jurídico, semántico y técnico sobre cada dato.

### Ejemplo 3. Plataforma de intermediación de datos y principio de no aportación

En un procedimiento de ayuda social, la persona interesada no debería aportar documentos que ya estén en poder de las Administraciones públicas cuando la normativa permite su consulta. Para lograrlo, el sistema tramitador necesita consultar datos de identidad, residencia, discapacidad, desempleo, renta u otros extremos según el procedimiento y las habilitaciones aplicables.

Desde la perspectiva de integración, la plataforma de intermediación funciona como un mecanismo de intercambio controlado entre organismos emisores y organismos requirentes. El sistema que tramita la ayuda no debe conectarse informalmente a bases de datos ajenas. Debe usar servicios autorizados, con finalidad determinada, trazabilidad, control de consentimiento u oposición cuando proceda, y registro de la consulta realizada.

Un diseño técnicamente limpio puede incluir un servicio interno llamado "verificación de requisitos", que no conoce los detalles de cada organismo emisor. Ese servicio invoca conectores normalizados y devuelve al expediente un resultado operativo: requisito verificado, no verificado, no disponible o pendiente de subsanación. De este modo, la lógica del procedimiento no queda llena de llamadas punto a punto difíciles de mantener.

La idea clave para el opositor es que la interoperabilidad no es solo conexión técnica. Incluye dimensión jurídica, organizativa, semántica y técnica. Una API que permite consultar un dato sin habilitación o sin finalidad administrativa legítima no es una buena integración pública, aunque funcione desde el punto de vista informático.

### Ejemplo 4. Registro electrónico e intercambio registral

Una solicitud presentada ante una Administración puede tener que llegar a otra unidad competente. En un modelo antiguo, la transmisión podía depender de remisiones manuales, correos internos o cargas de documentos en sistemas heterogéneos. En un modelo integrado, el asiento registral y la documentación asociada viajan mediante mecanismos de intercambio que conservan metadatos, fecha, origen, destino y estado de la remisión.

El caso permite explicar por qué la interoperabilidad documental y de metadatos es tan importante como la API de transporte. Si el sistema receptor recibe un PDF sin metadatos fiables, tendrá que reinterpretar a mano datos que ya existían en origen. Si recibe un paquete con metadatos normalizados, identificación del órgano, asunto, interesado y trazabilidad, puede automatizar parte de la clasificación y reducir errores.

El patrón arquitectónico típico combina un sistema registral, un servicio de intercambio, validaciones de formato, sellado temporal cuando corresponda y un mecanismo de acuse. La integración no termina al "enviar": debe contemplar confirmación, rechazo, reintento, reconciliación y auditoría.

En examen, este ejemplo sirve para reconocer que los sistemas distribuidos públicos no pueden limitarse a disponibilidad y rendimiento. Deben preservar efectos jurídicos: fecha de presentación, integridad de documentos, competencia del órgano, constancia de recepción y conservación.

### Ejemplo 5. Notificación electrónica desacoplada del gestor de expedientes

Un gestor de expedientes dicta una resolución y debe notificarla a la persona interesada. Si el gestor incorpora internamente toda la lógica de notificación, acabará duplicando funciones: puesta a disposición, control de plazos, comparecencia, rechazo, avisos complementarios, acuses y consulta histórica. En una Administración con muchos procedimientos, esa duplicación produce inconsistencias.

Una arquitectura más sostenible separa el acto administrativo del servicio de notificación. El gestor de expedientes genera el acto, lo firma cuando proceda, lo asocia al expediente y solicita al servicio de notificaciones la puesta a disposición. El servicio de notificaciones gestiona los estados propios de la notificación y devuelve eventos o consultas de estado al gestor. El gestor no tiene que conocer cada detalle interno de la plataforma, pero sí debe poder acreditar la situación de la notificación en el expediente.

Este ejemplo es útil para explicar la diferencia entre llamada síncrona y proceso de larga duración. La solicitud de notificación puede ser una llamada inicial, pero el ciclo de vida de la notificación es asíncrono: se producen cambios de estado con el paso del tiempo. Un buen diseño usa identificadores de correlación, eventos de estado, consultas idempotentes y registros de auditoría.

La trampa de examen está en pensar que todo debe resolverse en una única transacción técnica. En sistemas administrativos, muchas operaciones tienen efectos temporales y jurídicos que no encajan en una transacción corta de base de datos. El diseño debe aceptar esa realidad y modelar estados intermedios.

### Ejemplo 6. API Gateway en una sede electrónica con varios trámites

Una sede electrónica expone trámites de urbanismo, tributos, contratación y recursos humanos. Cada ámbito usa sistemas internos distintos. Sin una capa de gobierno, cada trámite podría publicar sus endpoints, su autenticación, sus formatos de error y su versionado. El resultado sería una sede difícil de proteger, documentar y mantener.

Un API Gateway puede ayudar a centralizar funciones transversales: terminación TLS, autenticación, autorización inicial, limitación de tasa, enrutamiento, registro de accesos, validación básica, transformación ligera y publicación controlada. Sin embargo, no debe convertirse en el lugar donde se mete toda la lógica de negocio. La autorización fina, las reglas procedimentales y las validaciones materiales siguen perteneciendo al servicio responsable.

El ejemplo permite enseñar una regla práctica: el Gateway gobierna el borde; no sustituye al dominio. Si el Gateway decide si una solicitud de licencia cumple requisitos urbanísticos, la arquitectura queda mal distribuida. Si el Gateway exige autenticación, enruta al servicio correcto y registra la llamada, está cumpliendo una función razonable.

En examen, una respuesta madura explicará tanto los beneficios como los límites: seguridad uniforme, observabilidad y control de exposición, pero riesgo de cuello de botella, dependencia operativa y exceso de lógica si se diseña mal.

### Ejemplo 7. Microservicio de pagos y tasas

Varios procedimientos administrativos necesitan liquidar tasas: expedición de certificados, licencias, pruebas selectivas o autorizaciones. Una solución ingenua consistiría en que cada aplicación calcule y gestione sus pagos. A corto plazo parece rápido; a medio plazo crea diferencias de criterios, problemas contables y mantenimiento duplicado.

Un servicio de pagos puede encapsular la generación de autoliquidaciones, la conexión con pasarelas, la conciliación, el estado del pago y la emisión de justificantes. Los sistemas tramitadores no deberían almacenar todos los detalles bancarios ni replicar la lógica de conciliación. Deben invocar el servicio de pagos con datos mínimos, recibir un identificador de operación y consultar o recibir el estado final.

La parte distribuida aparece en los fallos. Puede ocurrir que el banco confirme el pago, pero el gestor de expedientes no reciba la respuesta por una caída temporal. Por eso el diseño debe incluir idempotencia, reintentos, consulta de estado y conciliación posterior. El expediente no debería duplicar el cobro si el ciudadano pulsa dos veces, ni perder la acreditación si la respuesta llega tarde.

La nota de examen es clara: en integración de pagos, "respuesta correcta en pantalla" no equivale a consistencia del procedimiento. Lo importante es el ciclo completo: operación identificable, estado verificable, no duplicidad, conciliación y evidencia documental.

### Ejemplo 8. Eventos para cambios de estado de expedientes

Un sistema de becas necesita informar a otros sistemas cuando una solicitud pasa de "presentada" a "en revisión", "requerida", "concedida" o "denegada". Una opción es que cada sistema interesado consulte periódicamente la base de datos del gestor. Esa solución es frágil: acopla a los consumidores al modelo interno, aumenta carga y dificulta cambiar la aplicación.

Una alternativa es publicar eventos de dominio. El gestor emite un evento cuando cambia el estado relevante, con un identificador de expediente, fecha, estado nuevo y metadatos mínimos. Los consumidores autorizados procesan el evento para actualizar cuadros de mando, enviar avisos o preparar estadísticas. El evento no debe contener más datos personales de los necesarios.

El diseño exige disciplina. Un evento no es un volcado de la tabla interna. Debe tener contrato, versión, semántica estable y política de retención. Además, debe asumirse que el procesamiento puede ser eventual: un consumidor puede tardar unos segundos o minutos en reflejar el cambio.

La idea clave es que la arquitectura orientada a eventos mejora desacoplamiento, pero introduce complejidad: orden, duplicados, reintentos, monitorización y consistencia eventual. No debe presentarse como solución universal.

### Ejemplo 9. Integración con un sistema legado de mainframe o cliente-servidor

Muchas Administraciones conservan sistemas críticos que no pueden reemplazarse de golpe. Un sistema tributario antiguo puede contener reglas estables, datos históricos y procesos de recaudación consolidados. El objetivo de modernización no siempre será sustituirlo inmediatamente, sino encapsularlo de forma segura.

Un patrón habitual es crear una capa anticorrupción o adaptador. Esta capa traduce entre el modelo moderno de servicios y el modelo del sistema legado. Puede convertir formatos, gestionar sesiones, aplicar validaciones, limitar operaciones expuestas y evitar que los nuevos servicios dependan de detalles internos del legado.

Este ejemplo ayuda a explicar que microservicios no significa "reescribir todo". Una transición responsable identifica dominios, dependencias y riesgos. Algunas capacidades podrán extraerse gradualmente; otras se mantendrán encapsuladas hasta que exista presupuesto, ventana de migración y garantía de continuidad.

En examen, conviene rechazar respuestas maximalistas. Una Administración no puede interrumpir nóminas, prestaciones o recaudación porque una arquitectura nueva sea más elegante. La continuidad del servicio, la trazabilidad y la seguridad pesan tanto como la modernización.

### Ejemplo 10. Catálogo de APIs internas y externas

Una comunidad autónoma impulsa un catálogo de APIs para que sus consejerías reutilicen servicios comunes: consulta de expedientes, validación de documentos, agenda de citas, notificaciones, pagos, firma, archivo y datos geográficos. Si el catálogo solo contiene nombres de endpoints, será insuficiente. Un catálogo útil describe finalidad, órgano responsable, versión, condiciones de uso, datos tratados, nivel de seguridad, disponibilidad, formatos, ejemplos de llamada, política de cambios y contacto funcional.

La diferencia entre publicar y gobernar es esencial. Publicar una API es hacerla accesible; gobernarla es asegurar que se usa de manera controlada, comprensible y sostenible. La gobernanza incluye revisión de contratos, control de versiones, retirada ordenada, métricas de consumo, gestión de incidencias y evaluación de impacto cuando se tratan datos personales o servicios críticos.

Este ejemplo puede conectarse con reutilización de soluciones. Si varias unidades necesitan la misma función, una API común reduce duplicidad. Pero la reutilización requiere que el servicio común esté bien financiado, documentado y operado. En caso contrario, cada unidad acabará creando su propia variante.

La enseñanza de examen es que el catálogo no es un inventario decorativo. Es una herramienta de interoperabilidad organizativa y técnica.

### Ejemplo 11. Contrato de API para consulta de estado de expediente

Una API de consulta de estado de expediente parece sencilla: recibe un identificador y devuelve un estado. Sin embargo, el diseño público exige más precisión. El contrato debe definir quién puede consultar, qué identificadores se aceptan, qué estados son oficiales, qué significan, qué datos se devuelven, qué errores se producen y cómo se registra la consulta.

Un mal contrato devuelve estados internos como `REV_TEC_03` o mensajes ambiguos como "pendiente de mesa". Un buen contrato expone estados comprensibles y estables: presentado, en tramitación, pendiente de subsanación, resuelto, archivado. Internamente puede existir más detalle, pero no todo detalle interno debe ser contrato externo.

El contrato debe incluir errores diferenciados. No es lo mismo expediente inexistente, expediente existente pero no accesible por la persona autenticada, servicio temporalmente no disponible o identificador mal formado. Agrupar todos los casos bajo un "error 500" impide al consumidor actuar correctamente y deteriora la experiencia ciudadana.

Este ejemplo ilustra que una API es también lenguaje administrativo. Debe representar el procedimiento de forma fiel, segura y comprensible.

### Ejemplo 12. Integración transfronteriza y servicios europeos

Un procedimiento universitario o profesional puede requerir comprobaciones transfronterizas: identidad electrónica reconocida, titulaciones, autorizaciones o intercambio de documentación entre Estados miembros. En estos casos, la integración no depende solo de decisiones nacionales. Debe considerar marcos europeos de identificación, confianza, interoperabilidad y prestación de servicios públicos digitales transfronterizos.

La arquitectura debe prever diferencias de idioma, formatos, niveles de garantía, semántica de atributos y responsabilidades entre organismos. No basta con traducir una pantalla. Una misma palabra, como "residencia" o "habilitación", puede tener efectos jurídicos distintos según el país o el procedimiento.

El ejemplo sirve para reforzar las cuatro capas de interoperabilidad: legal, organizativa, semántica y técnica. La interoperabilidad técnica puede resolver el transporte; la semántica evita que se intercambien datos mal entendidos; la organizativa aclara quién responde; y la legal determina si el intercambio es válido y con qué efectos.

En examen, cuando aparezcan servicios transfronterizos, hay que elevar el nivel de respuesta. No basta hablar de REST, JSON o certificados. Hay que mencionar reconocimiento, confianza, finalidad, equivalencia semántica y gobierno.

## Tabla de decisión rápida: patrón de integración

| Necesidad del servicio | Patrón razonable | Ventaja principal | Riesgo si se aplica mal | Señal de examen |
|---|---|---|---|---|
| Consultar un dato puntual y devolver respuesta inmediata | API síncrona | Simplicidad y respuesta directa | Acoplamiento fuerte y cascadas de fallo | "Necesito saber ahora si cumple un requisito" |
| Comunicar cambios de estado a varios sistemas | Evento de dominio | Desacoplamiento de consumidores | Duplicados, orden y consistencia eventual | "Varios sistemas deben enterarse cuando cambie algo" |
| Encapsular un sistema antiguo | Adaptador o capa anticorrupción | Protege al nuevo dominio del legado | Convertir el adaptador en un segundo sistema complejo | "No se puede sustituir el sistema crítico" |
| Exponer servicios a muchas unidades | Catálogo y API Gateway | Gobierno, seguridad y descubrimiento | Centralizar lógica de negocio en la puerta de entrada | "Muchas aplicaciones consumen servicios comunes" |
| Procesar operaciones largas | Flujo asíncrono con estados | Trazabilidad de procesos duraderos | Fingir una transacción instantánea | "La operación tarda, tiene acuses y plazos" |
| Intercambiar documentos con efectos jurídicos | Servicio interoperable con metadatos | Conserva contexto, fecha e integridad | Enviar solo ficheros sin semántica | "Importan asiento, órgano, fecha y recepción" |

## Supuestos prácticos guiados

### Supuesto 1. Rediseño de una ayuda con consulta de datos externos

**Situación.** Una consejería gestiona una ayuda que exige residencia, nivel de renta, situación laboral y ausencia de deudas. El formulario actual pide al ciudadano varios certificados. La dirección quiere reducir cargas administrativas y acelerar la resolución.

**Pistas relevantes.** Hay datos que pueden consultarse mediante mecanismos de interoperabilidad si existe habilitación. El expediente debe conservar evidencia de la consulta. No todos los datos tienen la misma fuente ni la misma disponibilidad. El ciudadano debe conocer el tratamiento de sus datos y el procedimiento debe respetar finalidad y proporcionalidad.

**Preguntas.**

1. ¿Qué capacidades separarías en la arquitectura?
2. ¿Qué patrón de integración usarías para verificar requisitos?
3. ¿Qué información debe quedar trazada en el expediente?
4. ¿Qué errores de diseño evitarías?

**Resolución paso a paso.** Primero se identifica el dominio del procedimiento: presentación, subsanación, instrucción, propuesta, resolución y notificación. Después se crea una capacidad específica de verificación de requisitos que actúa como intermediaria entre el gestor del expediente y los servicios de datos autorizados. Esa capacidad no debe devolver datos innecesarios; debe devolver resultados interpretables para el procedimiento, como "residencia verificada" o "renta no disponible".

En segundo lugar se definen contratos para cada consulta. Cada contrato debe indicar finalidad, organismo emisor, datos mínimos, identificador de solicitud, fecha, resultado, posibles errores y política de reintento. La tramitación no debe bloquearse indefinidamente si un servicio externo no responde. Debe existir estado "pendiente de verificación" y un mecanismo de revisión manual o reintento controlado.

En tercer lugar se registra evidencia. El expediente debe poder acreditar qué consulta se hizo, cuándo, por qué procedimiento, con qué resultado y qué decisión se adoptó. Esta evidencia no equivale a copiar masivamente datos externos al expediente. Solo se incorporará lo necesario para justificar la decisión administrativa.

**Errores frecuentes.** Conectar directamente el gestor a cada base de datos externa, almacenar datos completos sin necesidad, no distinguir indisponibilidad técnica de requisito incumplido, no versionar contratos, no registrar trazabilidad y convertir la consulta de datos en una lógica dispersa por pantallas.

**Criterio de corrección A1.** Una respuesta excelente no se limita a decir "usar APIs". Debe mencionar interoperabilidad legal, organizativa, semántica y técnica; minimización de datos; trazabilidad; tratamiento de errores; y separación entre dato consultado y decisión procedimental.

**Mini comprobación.** Si el servicio de renta no responde, ¿debe denegarse automáticamente la ayuda? No. La indisponibilidad técnica no equivale a incumplimiento material. Debe modelarse un estado de pendiente, reintento o comprobación alternativa según las reglas del procedimiento.

### Supuesto 2. Caída parcial en una arquitectura de microservicios

**Situación.** Una sede electrónica usa servicios separados para identificación, expediente, pagos, notificaciones y archivo. Durante una convocatoria con alto volumen, el servicio de pagos queda intermitente. Las solicitudes pueden presentarse, pero algunas tasas no se confirman en tiempo real.

**Pistas relevantes.** En una arquitectura distribuida hay fallos parciales. No todos los componentes caen a la vez. El diseño debe evitar duplicidades, pérdida de operaciones y mensajes engañosos al ciudadano.

**Preguntas.**

1. ¿Qué diferencia hay entre fallo total y fallo parcial?
2. ¿Cómo debe comportarse el sistema de presentación?
3. ¿Qué mecanismos técnicos ayudan a recuperar consistencia?
4. ¿Qué debe comunicarse al usuario?

**Resolución paso a paso.** El fallo parcial significa que el sistema no está completamente indisponible, pero una capacidad necesaria funciona de forma degradada. La respuesta no debe ser necesariamente apagar toda la sede. Puede permitirse guardar una solicitud en borrador, presentar si la normativa lo permite con pago pendiente, o informar de indisponibilidad del pago y permitir reintento. La decisión depende del procedimiento y de la normativa aplicable.

Técnicamente, cada operación de pago debe tener un identificador idempotente. Si el ciudadano reintenta, el sistema debe reconocer si ya existe una operación previa y consultar su estado antes de crear otra. Deben existir reintentos controlados, conciliación con la pasarela, colas para procesar confirmaciones tardías y monitorización específica del servicio afectado.

El expediente debe distinguir "solicitud presentada", "pago iniciado", "pago confirmado", "pago fallido" y "pago pendiente de conciliación" si esos estados son relevantes. Un único campo "correcto/error" no basta.

**Errores frecuentes.** Cobrar dos veces por doble clic, perder la solicitud si falla el pago, mostrar "trámite completado" sin confirmación suficiente, ocultar la indisponibilidad, no guardar evidencias de reintento y diseñar todo como una transacción técnica única.

**Criterio de corrección A1.** La respuesta debe hablar de resiliencia, idempotencia, estados, conciliación, observabilidad y comunicación honesta al ciudadano. También debe reconocer que la solución técnica está condicionada por la regulación del trámite.

**Mini comprobación.** ¿Una cola de mensajes resuelve por sí sola el problema? No. Ayuda a desacoplar y recuperar operaciones, pero hacen falta contrato, idempotencia, seguimiento de estado y criterio jurídico sobre efectos de la presentación y del pago.

### Supuesto 3. Publicación de una API de consulta de expedientes para otras Administraciones

**Situación.** Un organismo estatal quiere permitir que comunidades autónomas consulten el estado de determinados expedientes cuando sea necesario para resolver sus propios procedimientos. Se propone publicar una API REST.

**Pistas relevantes.** La API tendrá consumidores institucionales. Habrá datos personales. El estado del expediente tiene significado administrativo. Debe haber control de acceso, finalidad, trazabilidad y contrato estable.

**Preguntas.**

1. ¿Qué debe contener el contrato de la API?
2. ¿Qué controles deben aplicarse antes de devolver información?
3. ¿Qué errores deben diferenciarse?
4. ¿Cómo se gestionaría el versionado?

**Resolución paso a paso.** El contrato debe definir recursos, operaciones, identificadores, estados oficiales, datos devueltos, errores, requisitos de autenticación, autorización, límites de uso, trazabilidad y condiciones de servicio. Debe evitar exponer nombres internos de tablas, códigos técnicos o estados no interpretables por los consumidores.

Antes de devolver información, el servicio debe comprobar la identidad del organismo consumidor, la autorización para ese tipo de consulta, la finalidad declarada y, si procede, la relación con un procedimiento concreto. La trazabilidad debe permitir auditar qué organismo consultó qué expediente, cuándo y con qué resultado.

Los errores deben diferenciar al menos solicitud mal formada, autenticación fallida, falta de autorización, expediente no encontrado, expediente no accesible para ese consumidor, conflicto de versión y servicio temporalmente no disponible. Esa diferenciación permite respuestas correctas de los consumidores y evita interpretaciones erróneas.

El versionado debe ser explícito. Cambios compatibles pueden añadirse sin romper consumidores; cambios incompatibles exigen nueva versión, periodo de convivencia y plan de retirada. La documentación debe estar en un catálogo accesible para los consumidores autorizados.

**Errores frecuentes.** Identificar API REST con interoperabilidad completa, devolver todos los datos del expediente "por si acaso", usar códigos internos sin glosario, no registrar consultas, cambiar el contrato sin aviso y confundir autenticación del consumidor con autorización material para cada consulta.

**Criterio de corrección A1.** La respuesta debe unir diseño técnico y garantías públicas. Una API administrativa no es solo un endpoint: es un contrato operativo con efectos sobre derechos, obligaciones, datos y decisiones.

**Mini comprobación.** Si una comunidad autónoma está autenticada, ¿puede consultar cualquier expediente? No. Autenticación demuestra quién llama; autorización determina qué puede consultar y para qué finalidad.

### Supuesto 4. Sustitución gradual de una aplicación monolítica de contratación

**Situación.** Una diputación tiene una aplicación monolítica para contratación pública. Gestiona expedientes, documentos, fiscalización, mesas, adjudicación y archivo. La aplicación es crítica y no puede pararse durante la modernización.

**Pistas relevantes.** Hay continuidad del servicio, dependencia de datos históricos y procesos jurídicamente sensibles. Una migración big bang tiene riesgo alto. No todos los módulos tienen la misma volatilidad.

**Preguntas.**

1. ¿Cómo plantearías una modernización gradual?
2. ¿Qué servicios extraerías primero?
3. ¿Qué papel tendría una capa anticorrupción?
4. ¿Qué indicadores usarías para decidir avances?

**Resolución paso a paso.** Primero se cartografían capacidades y dependencias. No se empieza por dividir tablas, sino por entender dominios: gestión documental, firma, notificaciones, consulta pública, integración contable, agenda de órganos colegiados y archivo. Después se identifican capacidades transversales que pueden separarse con menor riesgo, como notificaciones, firma o consulta pública, siempre que existan contratos claros.

La capa anticorrupción encapsula el monolito. Los nuevos servicios no deberían depender de sus tablas ni de sus códigos internos. La capa traduce conceptos y protege al nuevo modelo de cambios o rarezas del sistema antiguo. A medida que una capacidad se estabiliza fuera del monolito, puede reducirse el acoplamiento.

La migración debe incluir pruebas de regresión, conciliación de datos, plan de reversión y monitorización. En procedimientos públicos, el error no es solo técnico: puede afectar plazos, publicidad, concurrencia o validez de actuaciones.

**Errores frecuentes.** Reescribir todo sin mapa de dependencias, crear microservicios que comparten la misma base de datos, extraer servicios sin dueño funcional, ignorar expedientes vivos, olvidar integraciones externas y medir el éxito por número de repositorios en vez de por reducción de riesgo y mejora del servicio.

**Criterio de corrección A1.** Una respuesta excelente defiende gradualidad, continuidad, contratos, encapsulamiento del legado, pruebas y gobernanza. No cae en el discurso de que el monolito es siempre malo y los microservicios siempre buenos.

**Mini comprobación.** ¿Separar el código en varios despliegues convierte el sistema en microservicios? No necesariamente. Si comparten base de datos, despliegue coordinado y modelo interno, puede ser un monolito distribuido.

### Supuesto 5. Integración de datos de sensores en una ciudad inteligente

**Situación.** Un ayuntamiento usa sensores de aparcamiento, calidad del aire y alumbrado. Quiere integrar esos datos con servicios de movilidad, portal de datos abiertos y cuadro de mando interno.

**Pistas relevantes.** Hay datos en tiempo real o casi real, diferentes frecuencias, calidad variable, seguridad de dispositivos, publicación abierta y posibles decisiones operativas.

**Preguntas.**

1. ¿Qué arquitectura de integración sería razonable?
2. ¿Qué datos deberían publicarse como abiertos y cuáles no?
3. ¿Cómo controlarías calidad y trazabilidad?
4. ¿Qué riesgos de seguridad aparecen?

**Resolución paso a paso.** Los sensores no deberían alimentar directamente todas las aplicaciones consumidoras. Conviene una plataforma de ingesta que valide, normalice, marque tiempo, controle calidad y publique flujos o APIs. Los sistemas internos pueden consumir eventos o series temporales. El portal de datos abiertos debe recibir conjuntos depurados, documentados y con licencia adecuada, no necesariamente el flujo bruto completo.

La calidad requiere metadatos: ubicación, tipo de sensor, fecha de calibración, unidad de medida, precisión, estado del dispositivo y reglas para valores anómalos. Sin esos metadatos, el dato puede ser técnicamente accesible pero administrativamente poco fiable.

La seguridad incluye autenticación de dispositivos, integridad de mensajes, segmentación de red, actualizaciones, gestión de claves y respuesta ante dispositivos comprometidos. En un servicio distribuido, un sensor manipulado puede afectar cuadros de mando o decisiones operativas.

**Errores frecuentes.** Publicar datos sin metadatos, confundir tiempo real con dato útil, mezclar datos personales o identificables sin evaluación, no distinguir APIs internas de datos abiertos, no controlar la cadena de captura y usar el mismo canal para operación crítica y reutilización pública sin filtros.

**Criterio de corrección A1.** La respuesta debe integrar arquitectura de eventos, gobierno del dato, seguridad y reutilización. Aunque el tema principal sea microservicios e integración, este supuesto permite conectar con calidad del dato e interoperabilidad semántica.

**Mini comprobación.** ¿Abrir una API pública equivale a cumplir una política de datos abiertos? No. Hacen falta licencia, documentación, formatos reutilizables, metadatos, calidad y condiciones claras de reutilización.

### Supuesto 6. Alta de una nueva aplicación departamental en un ecosistema común

**Situación.** Un departamento quiere crear una aplicación para gestionar inspecciones. Debe integrarse con identificación corporativa, gestor documental, notificaciones, firma, archivo, cuadro de mando y servicios de datos de otras Administraciones.

**Pistas relevantes.** El reto no es solo desarrollar una aplicación, sino incorporarla a un ecosistema de servicios comunes. Cada integración tiene contrato, requisitos de seguridad y responsabilidades.

**Preguntas.**

1. ¿Qué checklist técnico y funcional exigirías antes de producción?
2. ¿Cómo evitarías integraciones punto a punto desordenadas?
3. ¿Qué documentación debe quedar en el catálogo?
4. ¿Qué pruebas son imprescindibles?

**Resolución paso a paso.** Antes de producción debe existir identificación de responsables funcionales y técnicos, clasificación de la información, análisis de integración, contrato de APIs consumidas y expuestas, mapa de datos, matriz de permisos, trazabilidad, plan de continuidad, monitorización y procedimiento de soporte.

Para evitar integraciones punto a punto, la aplicación debe consumir servicios comunes mediante contratos publicados y, cuando exponga capacidades reutilizables, registrarlas en el catálogo. Las conexiones excepcionales deben justificarse, documentarse y someterse a revisión.

La documentación del catálogo debe incluir finalidad, operaciones, datos, versiones, seguridad, disponibilidad esperada, ejemplos, errores, contacto, dependencia de otros servicios y política de cambios. Las pruebas deben cubrir autenticación, autorización, contratos, rendimiento, indisponibilidad de dependencias, auditoría, accesibilidad cuando haya interfaz y recuperación.

**Errores frecuentes.** Dar de alta la aplicación solo desde el punto de vista de infraestructura, no revisar datos personales, depender de credenciales compartidas, no prever entornos de prueba, no documentar errores, no registrar llamadas y no tener plan de retirada de versiones antiguas.

**Criterio de corrección A1.** La respuesta debe sonar a gobierno de plataforma, no a instalación aislada. El opositor debe demostrar que comprende la Administración como ecosistema interoperable.

**Mini comprobación.** ¿Una aplicación que consume muchos servicios comunes es automáticamente más madura? No. Puede ser más frágil si no gestiona dependencias, límites, fallos y contratos.

## Errores frecuentes que conviene destacar

1. **Confundir digitalización de pantalla con integración real.** Un formulario web no implica que el procedimiento esté integrado. Puede seguir exigiendo documentos innecesarios, duplicando datos y enviando información manualmente entre unidades.

2. **Identificar microservicio con endpoint pequeño.** Un microservicio no es cualquier operación REST. Debe tener responsabilidad de negocio clara, autonomía razonable, contrato estable, datos bajo control coherente y capacidad de despliegue y evolución sin coordinación excesiva.

3. **Crear un monolito distribuido.** Separar una aplicación en varios procesos que comparten base de datos, modelo interno y despliegues coordinados suele aumentar la complejidad sin lograr autonomía. Es una trampa clásica en preguntas de examen.

4. **Usar integración punto a punto sin gobierno.** Si cada sistema se conecta con cada otro sistema mediante acuerdos ad hoc, el mapa resultante se vuelve difícil de auditar, securizar y cambiar. La integración necesita catálogo, patrones comunes y responsabilidades.

5. **Pensar que el API Gateway resuelve toda la seguridad.** El Gateway ayuda en el borde, pero no sustituye autorización de negocio, clasificación de información, auditoría, seguridad interna, gestión de secretos ni controles del ENS.

6. **Ignorar la semántica de los datos.** Dos sistemas pueden intercambiar JSON válido y aun así no entenderse. Si "estado", "residente", "unidad familiar" o "fecha de efecto" significan cosas distintas, la integración falla aunque el transporte funcione.

7. **No diseñar para fallos parciales.** En sistemas distribuidos, una dependencia puede caer mientras el resto sigue funcionando. Deben existir timeouts, circuit breakers cuando proceda, reintentos controlados, colas, estados pendientes y mensajes comprensibles.

8. **No aplicar idempotencia.** Operaciones como pagos, registro, alta de solicitud o generación de justificante pueden repetirse por reintentos. Sin idempotencia se producen duplicidades o incoherencias.

9. **No versionar contratos.** Cambiar una API sin plan de compatibilidad rompe consumidores. En el sector público, además, puede afectar servicios prestados por otras unidades o Administraciones.

10. **Copiar datos externos sin necesidad.** Consultar datos para verificar un requisito no autoriza a almacenar todo el conjunto de datos. Debe aplicarse minimización, finalidad y conservación adecuada.

11. **No distinguir autenticación de autorización.** Saber quién llama no basta. Hay que decidir qué puede hacer, sobre qué expediente, para qué finalidad y bajo qué condiciones.

12. **Diseñar eventos como volcados de base de datos.** Un evento de dominio debe expresar algo significativo para consumidores autorizados. Publicar cambios técnicos de tablas acopla sistemas y filtra detalles internos.

13. **Olvidar la trazabilidad administrativa.** La arquitectura debe permitir reconstruir actuaciones, consultas, envíos, acuses, errores y decisiones. La observabilidad técnica no sustituye la evidencia administrativa.

14. **Elegir microservicios sin capacidad operativa.** Microservicios exigen automatización, monitorización, pruebas, despliegue, gestión de configuración y cultura de operación. Sin esas capacidades, el coste puede superar el beneficio.

15. **No prever retirada de versiones.** Mantener indefinidamente versiones antiguas de APIs encarece seguridad y mantenimiento. La retirada debe planificarse, comunicarse y acompañarse de migración.

16. **Convertir un bus de integración en vertedero de lógica.** El bus, broker o plataforma de integración no debe acumular reglas de negocio opacas que nadie gobierna. Debe facilitar comunicación, transformación controlada y trazabilidad.

17. **Tratar la nube como excepción normativa.** Usar servicios cloud no elimina obligaciones de seguridad, interoperabilidad, protección de datos, continuidad o contratación. La clasificación y las medidas siguen siendo necesarias.

18. **No separar entorno de pruebas y producción.** Integrar con servicios reales sin entorno controlado provoca riesgos de datos, trazas erróneas y operaciones no deseadas. Las pruebas deben usar datos y servicios adecuados.

19. **No cuidar mensajes de error.** Un error técnico críptico en sede electrónica puede impedir al ciudadano ejercer derechos o cumplir plazos. Los errores deben ser seguros, comprensibles y accionables.

20. **Medir éxito solo por rendimiento.** En Administración importan también validez jurídica, accesibilidad, continuidad, seguridad, auditabilidad, interoperabilidad y reducción de cargas.

## Notas de test separadas

### Nivel base

- Si una pregunta contrapone "API" e "interoperabilidad", la respuesta correcta suele recordar que la API es un medio técnico y la interoperabilidad es más amplia: legal, organizativa, semántica y técnica.
- Si aparece "microservicios" como sinónimo de "aplicación dividida en módulos", hay que buscar autonomía, responsabilidad de negocio y contrato. Sin esas notas, la opción puede ser incompleta.
- Si una opción afirma que el API Gateway contiene toda la lógica de negocio, normalmente es falsa. El Gateway gobierna entrada, seguridad transversal y enrutamiento, pero el dominio permanece en los servicios.
- Si se habla de consulta de datos entre Administraciones, debe aparecer finalidad, habilitación, minimización y trazabilidad. Una respuesta puramente técnica queda corta.
- Si una operación puede repetirse por reintentos, la palabra clave es idempotencia. Es típica en pagos, altas, registros y generación de justificantes.

### Nivel aplicación

- En un proceso de notificación, el cambio de estado no suele resolverse con una transacción síncrona única. Hay puesta a disposición, acceso, rechazo, plazos y acuses. La arquitectura debe modelar estados.
- En eventos, una opción que promete consistencia inmediata para todos los consumidores suele ser sospechosa. Los eventos favorecen consistencia eventual y desacoplamiento, no sincronía absoluta.
- En sistemas legados, la capa anticorrupción protege el nuevo modelo. No es un parche de mala calidad: puede ser una técnica prudente de transición.
- En integración documental, el documento sin metadatos pierde parte de su valor administrativo. Fecha, órgano, expediente, interesado, firma, formato y estado importan.
- En catálogo de APIs, la documentación no es un anexo menor. Forma parte del gobierno del servicio y permite reutilización, seguridad y evolución.

### Nivel examen real

- Cuando una pregunta ofrezca "centralizar todos los datos en una base única" como solución a la integración, hay que desconfiar. Puede haber casos de repositorio común, pero en Administración suele ser esencial respetar fuente autorizada, competencia y finalidad.
- Si se plantea una caída parcial, la mejor respuesta no es siempre "interrumpir todo" ni "seguir como si nada". Hay que degradar de forma controlada, informar y conservar evidencias.
- Si se pregunta por seguridad en microservicios, no basta con cifrar comunicaciones. Hay que hablar de identidad de servicio, autorización, secretos, registro, segmentación, vulnerabilidades, dependencias y medidas del ENS según categoría.
- Si se pregunta por reutilización, no basta con publicar código o endpoints. Deben existir condiciones de uso, soporte, versionado, documentación, licencia cuando proceda y responsabilidad.
- Si se pregunta por interoperabilidad europea, conviene mencionar reconocimiento transfronterizo, servicios de confianza, marcos europeos y equivalencia semántica, además de protocolos.

## Preguntas de recuperación y diagnóstico

### Pregunta 1

Una Administración publica una API para consultar el estado de expedientes. ¿Cuál es la afirmación más correcta?

A. La API garantiza por sí sola la interoperabilidad completa entre Administraciones.
B. La API debe tener contrato, control de acceso, semántica de estados, trazabilidad y política de versionado.
C. La API debe devolver todos los datos disponibles para evitar futuras modificaciones.
D. La API no necesita documentación si usa JSON y HTTPS.

**Respuesta correcta: B.** La API es un instrumento técnico, pero necesita gobierno. La opción A exagera; la C vulnera minimización y acopla consumidores; la D confunde formato con contrato.

### Pregunta 2

En una arquitectura de microservicios, dos servicios comparten la misma base de datos y se despliegan siempre juntos. ¿Qué problema aparece?

A. No hay ningún problema si ambos usan contenedores.
B. Puede existir un monolito distribuido con acoplamiento fuerte.
C. El sistema es necesariamente más seguro.
D. La interoperabilidad semántica queda garantizada.

**Respuesta correcta: B.** La separación física no garantiza autonomía. Compartir datos y despliegue suele indicar acoplamiento.

### Pregunta 3

Un servicio de pagos recibe dos veces la misma solicitud por reintento del navegador. ¿Qué propiedad ayuda a evitar cobros duplicados?

A. Elasticidad.
B. Idempotencia.
C. Interoperabilidad organizativa.
D. Cacheado agresivo.

**Respuesta correcta: B.** La idempotencia permite que repetir una operación con el mismo identificador no produzca efectos duplicados.

### Pregunta 4

¿Qué afirmación describe mejor una capa anticorrupción frente a un sistema legado?

A. Permite que los nuevos servicios dependan directamente de las tablas antiguas.
B. Traduce modelos y protege al nuevo dominio de detalles internos del legado.
C. Elimina la necesidad de pruebas de integración.
D. Sustituye las obligaciones de seguridad.

**Respuesta correcta: B.** La capa anticorrupción reduce acoplamiento conceptual y técnico. No elimina pruebas ni controles.

### Pregunta 5

En un servicio orientado a eventos, ¿qué riesgo debe controlarse especialmente?

A. Que todos los consumidores reciban siempre datos en una única transacción inmediata.
B. Duplicados, orden de eventos, reintentos y consistencia eventual.
C. Que no pueda existir trazabilidad.
D. Que sea imposible versionar mensajes.

**Respuesta correcta: B.** Los eventos aportan desacoplamiento, pero exigen gestión de entrega, duplicados, orden y versiones.

### Pregunta 6

Una carpeta ciudadana muestra expedientes de varias Administraciones. ¿Cuál es el enfoque más sólido?

A. Copiar todos los expedientes a una base única de la carpeta.
B. Tratarla como agregador con consultas a fuentes responsables y controles de acceso.
C. Evitar cualquier integración y mostrar solo enlaces.
D. Permitir modificación directa de cualquier dato desde la vista agregada.

**Respuesta correcta: B.** La carpeta puede integrar experiencia sin desplazar la responsabilidad de cada fuente.

### Pregunta 7

En una consulta de datos entre Administraciones, ¿qué elemento no puede faltar?

A. Finalidad y trazabilidad de la consulta.
B. Copia completa de todos los datos del interesado.
C. Ausencia de controles para facilitar rapidez.
D. Uso obligatorio de una misma base de datos por todos los organismos.

**Respuesta correcta: A.** La consulta debe estar justificada, ser proporcional y quedar registrada.

### Pregunta 8

¿Cuál es una función típica de un API Gateway?

A. Resolver las reglas sustantivas de cada procedimiento.
B. Centralizar todas las bases de datos.
C. Enrutar, aplicar controles transversales y registrar acceso en el borde.
D. Sustituir al archivo electrónico.

**Respuesta correcta: C.** El Gateway ayuda en exposición y gobierno técnico del acceso. No debe absorber el dominio.

### Pregunta 9

Un sistema receptor obtiene un PDF sin metadatos desde otro organismo. ¿Qué problema principal aparece?

A. El documento pesa siempre demasiado.
B. Se pierde contexto administrativo necesario para automatizar y acreditar el intercambio.
C. JSON habría resuelto cualquier problema jurídico.
D. No puede almacenarse en ningún caso.

**Respuesta correcta: B.** Los metadatos aportan contexto, trazabilidad, clasificación y efecto administrativo.

### Pregunta 10

¿Qué respuesta es más adecuada ante una indisponibilidad temporal de un servicio externo de verificación?

A. Denegar automáticamente el expediente.
B. Marcar estado pendiente, reintentar o prever comprobación alternativa según el procedimiento.
C. Borrar la solicitud.
D. Inventar un resultado por defecto para no retrasar la tramitación.

**Respuesta correcta: B.** La indisponibilidad técnica no equivale a incumplimiento material.

### Pregunta 11

Una API cambia el significado de un campo sin avisar a consumidores. ¿Qué principio se vulnera principalmente?

A. Gobierno y versionado del contrato.
B. Compresión de datos.
C. Diseño visual.
D. Balanceo de carga.

**Respuesta correcta: A.** Cambiar semántica sin gestión rompe interoperabilidad y confianza.

### Pregunta 12

En seguridad de microservicios públicos, ¿cuál es la afirmación más completa?

A. Basta con usar HTTPS.
B. Deben combinarse identidad, autorización, segmentación, secretos, registro, gestión de vulnerabilidades y medidas conforme a la clasificación del sistema.
C. La seguridad corresponde solo al proveedor cloud.
D. Los servicios internos no necesitan controles.

**Respuesta correcta: B.** HTTPS es necesario pero insuficiente. La seguridad es multicapa.

## Tabla de errores y corrección pedagógica

| Error del opositor | Por qué es insuficiente | Respuesta mejorada |
|---|---|---|
| "Microservicios es dividir una aplicación en muchas APIs" | Se centra en tamaño y transporte, no en dominio ni autonomía | "Microservicios separa capacidades de negocio con contratos, datos y evolución relativamente autónomos" |
| "Interoperabilidad es que los sistemas se conecten" | Olvida dimensiones legal, organizativa y semántica | "La conexión técnica solo es una capa de la interoperabilidad" |
| "El Gateway protege todo" | Reduce seguridad a una frontera | "El Gateway ayuda, pero cada servicio conserva controles y trazabilidad" |
| "Si falla un servicio, falla todo" | No contempla degradación controlada | "El diseño debe prever fallos parciales y estados intermedios" |
| "Los eventos sustituyen todas las consultas" | Convierte un patrón en dogma | "Eventos y consultas se combinan según necesidad de consistencia, latencia y trazabilidad" |
| "Copiamos datos para ir más rápido" | Puede vulnerar minimización y generar inconsistencias | "Se consulta o replica solo lo necesario, con finalidad, calidad y conservación definidas" |

## Supuestos cortos para intercalar en el tema

### Supuesto corto A. Doble presentación de solicitud

Una persona pulsa dos veces el botón de presentar por lentitud de la sede. El sistema recibe dos peticiones iguales. La respuesta correcta no es confiar en que el usuario no repita, sino diseñar una clave idempotente de presentación. Si el sistema detecta la misma operación, debe devolver el mismo justificante o informar de que ya existe una presentación equivalente, evitando dos expedientes duplicados salvo que la normativa permita solicitudes múltiples.

**Nota de test.** Idempotencia no significa que todas las operaciones sean de solo lectura. Significa que repetir la misma orden identificada no multiplica sus efectos.

### Supuesto corto B. Estado interno frente a estado público

Un expediente tiene internamente estados como "REV-AREA2", "FIRMA-PTE-JS" o "PROP-VAL-04". Mostrar esos códigos al ciudadano o a otra Administración puede generar confusión. La API pública debe mapearlos a estados comprensibles y jurídicamente relevantes, como "en tramitación", "pendiente de firma" o "resuelto", según proceda.

**Nota de test.** No todo estado interno debe exponerse. El contrato externo debe ser estable y semánticamente claro.

### Supuesto corto C. Evento con demasiados datos

Un evento de cambio de estado incluye nombre completo, domicilio, renta y documentos adjuntos, aunque los consumidores solo necesitan saber que el expediente ha sido resuelto. El diseño incumple minimización y aumenta riesgo. Un evento debe contener lo necesario para su finalidad; los detalles se consultarán mediante APIs autorizadas cuando proceda.

**Nota de test.** Evento no equivale a volcado documental ni a copia de expediente.

### Supuesto corto D. Integración con sistema estadístico

Un cuadro de mando necesita indicadores agregados de tiempos de tramitación. No necesita datos nominales de interesados. La integración adecuada puede basarse en eventos o extracciones agregadas, con anonimización o minimización según el caso. Copiar expedientes completos para calcular medias sería desproporcionado.

**Nota de test.** La finalidad analítica no justifica automáticamente acceso a datos personales completos.

### Supuesto corto E. Retirada de versión antigua

Una API de notificaciones tiene versión 1 y versión 2. Tres consumidores siguen usando la versión antigua. Retirarla sin aviso puede romper procedimientos. Mantenerla eternamente aumenta costes y riesgos. La solución madura es anunciar calendario, ofrecer documentación de migración, medir consumos, acompañar pruebas y retirar cuando se cumplan condiciones.

**Nota de test.** Versionar no es solo poner `/v2` en la URL. Incluye gobernanza de ciclo de vida.

## Guion para caja "modo tutor"

**Qué significa.** Una arquitectura distribuida reparte capacidades entre componentes que colaboran mediante contratos. En la Administración, esos contratos no son solo técnicos: deben respetar competencia, procedimiento, seguridad, protección de datos, conservación e interoperabilidad.

**Por qué importa.** Los servicios públicos digitales rara vez viven en una sola aplicación. Un trámite puede necesitar identidad, firma, registro, pago, consulta de datos, notificación, archivo y publicación. Sin integración gobernada, el ciudadano ve pantallas digitales pero la Administración sigue funcionando como compartimentos aislados.

**Con qué se confunde.** Se confunde con comprar una plataforma, publicar APIs sin gobierno, dividir un monolito por capas técnicas o centralizar todos los datos. También se confunde con automatizar sin revisar base jurídica y responsabilidad funcional.

**Cómo se reconoce en examen.** Aparecen expresiones como expediente, sede, notificación, consulta de datos, servicios comunes, reutilización, interoperabilidad, seguridad, trazabilidad, sistemas heredados, alta disponibilidad, eventos, colas, Gateway, catálogo o versionado. La respuesta debe combinar técnica con garantías públicas.

## Recomendaciones para visuales locales

1. **Diagrama de integración de trámite.** Ciudadano, sede, identificación, gestor de expedientes, intermediación de datos, pagos, notificaciones y archivo. Debe mostrar flujos diferenciados y no parecer una base de datos central única.

2. **Esquema API Gateway.** Consumidores internos y externos en el borde; Gateway con autenticación, enrutamiento, límites y observabilidad; servicios de dominio detrás. Incluir nota visual de que la lógica de negocio no vive en el Gateway.

3. **Secuencia de pago idempotente.** Solicitud con identificador de operación, pasarela, confirmación tardía, consulta de estado y conciliación. Útil para explicar reintentos.

4. **Eventos de expediente.** Gestor emite evento; consumidores autorizados actualizan avisos, cuadro de mando y carpeta. Debe incluir duplicados/reintento como idea visual, sin saturar.

5. **Capas de interoperabilidad.** Legal, organizativa, semántica y técnica, con un ejemplo breve en cada capa. Este visual es clave para evitar respuestas puramente tecnológicas.

6. **Migración de legado.** Monolito existente, capa anticorrupción, servicios extraídos progresivamente y consumidores. Debe transmitir gradualidad, no ruptura brusca.

## Fuentes oficiales y técnicas a citar editorialmente

- Ley 39/2015, del Procedimiento Administrativo Común de las Administraciones Públicas, especialmente por administración electrónica, derechos de las personas y no aportación de documentos cuando proceda.
- Ley 40/2015, de Régimen Jurídico del Sector Público, por funcionamiento electrónico, cooperación, reutilización de sistemas y Esquema Nacional de Interoperabilidad.
- Real Decreto 203/2021, Reglamento de actuación y funcionamiento del sector público por medios electrónicos, por sedes, identificación, firma, actuación administrativa automatizada, expediente, archivo y relaciones electrónicas.
- Real Decreto 4/2010, Esquema Nacional de Interoperabilidad, por principios y criterios de interoperabilidad, conservación, normalización y reutilización.
- Real Decreto 311/2022, Esquema Nacional de Seguridad, por principios, requisitos y medidas para proteger información y servicios públicos digitales.
- Reglamento europeo de Interoperabilidad de la Europa Interoperable, por cooperación y servicios públicos digitales transfronterizos.
- Reglamento eIDAS y su modificación sobre identidad digital europea, por identificación electrónica, servicios de confianza y reconocimiento transfronterizo.
- Directiva europea de datos abiertos y reutilización de la información del sector público, cuando se conecte integración con APIs públicas y reutilización.
- Guías del Portal de Administración Electrónica y del Centro Criptológico Nacional cuando se necesiten ejemplos operativos de servicios comunes, seguridad, interoperabilidad y administración digital.

## Cierre de integración

Este bloque puede aportar entre 4.500 y 6.000 palabras al tema final si se integra completo. Para mantener calidad A1, se recomienda no volcarlo seguido como anexo. La mejor colocación es distribuir los ejemplos tras los epígrafes teóricos correspondientes, reservar los supuestos guiados para una sección propia y llevar las notas de test a cajas ocultables o diferenciadas. El banco de preguntas completo debe quedar fuera del temario principal; aquí solo se ofrece una muestra diagnóstica para lectura y repaso.
