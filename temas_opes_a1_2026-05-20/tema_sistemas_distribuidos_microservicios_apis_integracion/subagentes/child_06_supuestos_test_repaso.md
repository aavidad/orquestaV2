# Supuestos prácticos, notas de test y repaso

Tema: Arquitecturas distribuidas, microservicios, APIs e integración de sistemas en la Administración pública.

## Supuesto guiado 1: tramitación distribuida de una ayuda pública

### Enunciado

Un ministerio quiere modernizar la gestión de una línea de ayudas. El procedimiento implica solicitud electrónica, consulta de identidad y representación, comprobación de estar al corriente de obligaciones, cálculo de baremo, notificación, subsanación, resolución y pago. Actualmente existe una aplicación monolítica conectada por intercambio de ficheros con varios organismos. La dirección quiere evolucionar a una arquitectura distribuida, con servicios reutilizables por comunidades autónomas y entidades locales, manteniendo trazabilidad, seguridad, interoperabilidad y continuidad del servicio.

### Análisis del problema

El primer paso no es dividir la aplicación por tablas ni por pantallas, sino por capacidades de negocio. En este caso pueden separarse, como mínimo, los dominios de expediente, solicitante, representación, elegibilidad, baremación, notificación, pagos, auditoría y archivo. Cada dominio debe tener una responsabilidad clara y una interfaz estable. La finalidad es reducir acoplamientos: que el módulo de baremación no dependa de cómo se almacena la notificación, ni el de pagos de la estructura interna del expediente.

La arquitectura puede combinar microservicios y componentes compartidos. No todo tiene que ser un microservicio. Los servicios con alta variabilidad, ciclo de vida propio o necesidad de escalado independiente son buenos candidatos. Los catálogos simples, reglas poco cambiantes o funciones transversales pueden resolverse con servicios comunes o librerías gobernadas. En Administración pública conviene evitar una multiplicación de servicios sin dueño, porque aumenta la carga de operación, seguridad, documentación y contratación.

### Diseño propuesto

Una solución razonable incluiría una API de expediente como fachada pública para las aplicaciones consumidoras, servicios internos por capacidad y un bus de eventos para hechos relevantes: solicitud presentada, documentación requerida, subsanación recibida, resolución dictada, pago ordenado. Las operaciones síncronas se reservarían para consultas inmediatas y validaciones necesarias en línea; las operaciones largas o dependientes de terceros se moverían a procesos asíncronos con reintentos y control de idempotencia.

Las APIs deberían publicar contratos versionados, esquemas de datos, códigos de error estables y reglas de autenticación. Para integraciones con otras Administraciones se debe preferir un modelo de contrato explícito, con identificación del organismo consumidor, finalidad del acceso, trazabilidad de cada consulta y minimización de datos. La interoperabilidad no se limita al formato técnico: exige significado común de los datos, correspondencia con procedimientos administrativos, conservación de evidencias y respeto a los principios de protección de datos.

El expediente actuaría como agregador funcional, pero no debería convertirse en un monolito encubierto. Por ejemplo, puede guardar el estado administrativo y referencias a evidencias, mientras que el servicio de notificaciones conserva sus propios justificantes, el de pagos su ciclo de ejecución y el de auditoría los eventos relevantes. Cuando se necesite una vista completa, se puede componer mediante lectura agregada, proyección o consulta federada, evitando transacciones distribuidas innecesarias.

### Seguridad, operación y gobierno

La solución debe incorporar autenticación robusta entre sistemas, autorización por rol, ámbito y finalidad, cifrado en tránsito, gestión de secretos, registro de auditoría y segregación de entornos. En APIs externas conviene usar un punto de entrada gobernado que aplique cuotas, validación de tokens, limitación de tráfico, registro y políticas de acceso. En servicios internos, la confianza implícita por estar en la red corporativa no es suficiente.

La observabilidad debe diseñarse desde el inicio. Cada solicitud debe tener un identificador de correlación que permita seguirla desde la sede electrónica hasta los servicios de comprobación, baremación y notificación. Las métricas deben cubrir latencia, errores, saturación, reintentos, colas pendientes y cumplimiento de acuerdos de nivel de servicio. Los logs deben ser útiles para soporte, pero no deben exponer datos personales innecesarios.

### Resolución esperada

La respuesta de examen debe justificar una arquitectura híbrida, con APIs para contratos externos, microservicios para capacidades con autonomía real, eventos para desacoplar procesos largos y controles de seguridad e interoperabilidad. También debe advertir que partir un monolito sin rediseñar dominios solo traslada el problema a una red de servicios frágiles. La clave es equilibrar modularidad, gobierno y sostenibilidad operativa.

## Supuesto guiado 2: integración de servicios municipales con una plataforma autonómica

### Enunciado

Una comunidad autónoma ofrece una plataforma de interoperabilidad para que ayuntamientos pequeños consulten datos de padrón, tributos, registro, licencias urbanísticas y servicios sociales. Algunos municipios tienen aplicaciones modernas con APIs; otros solo exportan ficheros; otros dependen de proveedores distintos. Se pide diseñar una estrategia de integración que permita avanzar sin bloquear a los municipios menos maduros, manteniendo calidad de datos, seguridad y evolución futura.

### Análisis del problema

El reto principal es la heterogeneidad. No se puede exigir la misma madurez técnica a todos los organismos desde el primer día, pero tampoco conviene consolidar indefinidamente integraciones punto a punto. La solución debe ofrecer una arquitectura gradual: conectores para sistemas existentes, normalización de datos, catálogo de APIs, mecanismos de mensajería y reglas comunes de gobierno.

Desde el punto de vista organizativo, debe existir un modelo de responsabilidad. La plataforma autonómica no puede corregir por sí sola todos los datos municipales. Puede validar formatos, detectar incoherencias y registrar incidencias, pero cada organismo debe conservar responsabilidad sobre sus datos de origen. La arquitectura debe reflejar esa frontera.

### Diseño propuesto

La plataforma puede organizarse en tres capas. La primera es la capa de adaptación, con conectores para APIs, ficheros firmados, colas o servicios heredados. La segunda es la capa de normalización, donde se transforman modelos locales a un vocabulario común, se validan campos obligatorios y se generan eventos o registros normalizados. La tercera es la capa de exposición, formada por APIs y servicios de consulta para consumidores autorizados.

Para municipios con APIs maduras, se puede integrar mediante contratos síncronos con autenticación mutua y control de versiones. Para municipios con menor madurez, se pueden aceptar cargas periódicas de datos mediante ficheros estructurados, siempre que existan validaciones, acuses de recepción, trazabilidad y calendario de migración hacia interfaces más automáticas. Las cargas por fichero no deben verse como solución final, sino como mecanismo de transición.

La plataforma debe incluir un catálogo de servicios con descripciones funcionales, responsables, disponibilidad, condiciones de uso, datos tratados y mecanismos de soporte. También debe disponer de un modelo de versionado: una versión nueva de una API no debería romper consumidores existentes sin periodo de convivencia, pruebas y comunicación. En integraciones públicas, la estabilidad del contrato es tan importante como la tecnología elegida.

### Patrón de integración

Para consultas simples y de baja latencia, una API síncrona puede ser adecuada. Para sincronización de padrones, licencias o expedientes con cambios frecuentes, conviene usar eventos o cargas incrementales. Para procesos que requieren validación humana, subsanación o conciliación, es preferible una cola de trabajo con estados y evidencias, no una llamada bloqueante que deje al consumidor sin respuesta.

La calidad de datos debe tratarse como parte del sistema. Deben definirse reglas de obligatoriedad, formatos, catálogos comunes, deduplicación, historificación y tratamiento de errores. Un dato rechazado debe producir una incidencia comprensible, con causa, organismo responsable, fecha, versión del esquema y posible acción correctiva. Sin esta gestión, la interoperabilidad se degrada en un intercambio técnico sin fiabilidad administrativa.

### Resolución esperada

Una buena respuesta debe proponer una integración progresiva, con adaptadores para sistemas existentes, normalización semántica, APIs gobernadas, eventos donde aporten valor y una estrategia clara de versionado y calidad. También debe señalar que la plataforma no sustituye la gobernanza del dato: necesita responsables, acuerdos, controles y mecanismos de evolución.

## Errores frecuentes

1. Confundir arquitectura distribuida con repartir cualquier aplicación en muchos servicios pequeños. La distribución aumenta latencia, fallos parciales, complejidad de pruebas y necesidades de observabilidad.

2. Diseñar microservicios alrededor de tablas de base de datos. Un microservicio debe responder a una capacidad de negocio y controlar su modelo, no ser una simple capa remota sobre una tabla compartida.

3. Suponer que una API es interoperabilidad completa. Una API puede resolver el acceso técnico, pero la interoperabilidad administrativa exige significado común, derechos de acceso, finalidad, trazabilidad, seguridad y calidad.

4. Ignorar las transacciones distribuidas. En muchos procesos públicos es mejor usar consistencia eventual, eventos, compensaciones y estados explícitos que intentar una transacción global frágil.

5. No versionar contratos. Cambiar campos, códigos o reglas sin compatibilidad rompe consumidores y genera dependencia informal entre organismos.

6. Reducir la seguridad a autenticación. También hacen falta autorización, auditoría, minimización de datos, control de finalidad, segregación, cifrado, rotación de secretos y respuesta ante incidentes.

7. No prever operación. Una arquitectura con servicios distribuidos sin métricas, trazas, alarmas, gestión de errores y procedimientos de soporte es difícil de mantener.

## Notas de test separadas

### Conceptos que suelen preguntarse

Microservicio: unidad desplegable, orientada a una capacidad de negocio, con autonomía relativa de datos, contrato explícito y ciclo de vida propio.

API: contrato de acceso a una funcionalidad o dato. Debe incluir operaciones, esquemas, errores, seguridad, versionado, límites de uso y documentación funcional.

Integración síncrona: el consumidor espera respuesta inmediata. Es útil para validaciones en línea, pero acopla disponibilidad y latencia.

Integración asíncrona: el productor y el consumidor no necesitan coincidir en el tiempo. Es adecuada para procesos largos, eventos, reintentos y desacoplamiento.

Idempotencia: propiedad por la que repetir una operación con la misma intención no produce efectos duplicados. Es esencial en pagos, registros, notificaciones y reintentos.

Observabilidad: capacidad de entender el comportamiento del sistema mediante logs, métricas y trazas, con correlación entre servicios.

### Contrastes típicos

Monolito modular no equivale a mala arquitectura. Puede ser adecuado si el dominio está bien separado y no hay necesidad real de despliegue independiente.

Microservicios no equivalen a independencia absoluta. Requieren gobierno común, estándares de seguridad, observabilidad, despliegue y documentación.

API pública no equivale a API abierta sin control. En Administración pública el acceso debe estar vinculado a competencia, finalidad, consentimiento o habilitación normativa cuando proceda.

Evento no equivale a orden completa. Un evento comunica un hecho ocurrido; si se necesita estado consolidado, pueden hacer falta proyecciones, consultas o procesos de reconciliación.

### Pistas de respuesta rápida

Si el caso menciona muchos organismos, pensar en interoperabilidad, contratos, catálogo, gobernanza y trazabilidad.

Si el caso menciona lentitud o procesos largos, valorar asincronía, colas, eventos, reintentos e idempotencia.

Si el caso menciona cambios frecuentes, destacar versionado, compatibilidad y pruebas de contrato.

Si el caso menciona datos personales, incluir minimización, finalidad, control de acceso, auditoría y conservación de evidencias.

## Preguntas de recuperación

1. ¿Qué problema resuelve una arquitectura de microservicios y qué costes introduce?

2. ¿Por qué no es recomendable que varios microservicios compartan directamente la misma base de datos?

3. ¿Cuándo elegirías una integración síncrona y cuándo una asíncrona en un procedimiento administrativo?

4. ¿Qué elementos debe tener una API para ser gobernable en un entorno público?

5. ¿Cómo se garantiza la trazabilidad de una solicitud que atraviesa varios sistemas?

6. ¿Qué relación existe entre interoperabilidad técnica, semántica y organizativa?

7. ¿Por qué la idempotencia es importante en reintentos y procesos de pago o notificación?

8. ¿Qué riesgos aparecen al partir un monolito sin definir dominios y responsabilidades?

## Muestra de preguntas tipo test con diagnóstico

### Pregunta 1

En una arquitectura de microservicios, ¿cuál es el criterio más adecuado para definir los límites de un servicio?

A. La tabla principal que utiliza.

B. La capacidad de negocio que presta y su responsabilidad funcional.

C. El lenguaje de programación usado por el equipo.

D. El número de pantallas de la aplicación.

Respuesta correcta: B.

Diagnóstico: si se elige A, se está pensando en una división técnica que suele crear acoplamiento por datos. C y D son criterios accesorios. El límite debe venir del dominio y de la responsabilidad que el servicio puede asumir.

### Pregunta 2

¿Qué ventaja aporta una integración asíncrona mediante eventos en un procedimiento administrativo?

A. Elimina la necesidad de seguridad.

B. Garantiza siempre consistencia inmediata entre todos los sistemas.

C. Desacopla productor y consumidor y facilita reintentos ante fallos temporales.

D. Sustituye la necesidad de contratos.

Respuesta correcta: C.

Diagnóstico: los eventos no eliminan seguridad ni contratos. Tampoco garantizan consistencia inmediata; normalmente aceptan consistencia eventual. Su valor está en reducir dependencia temporal y mejorar resiliencia.

### Pregunta 3

Una API administrativa gobernada debe incluir, como mínimo:

A. Operaciones, datos, errores, seguridad, versionado y condiciones de uso.

B. Solo una descripción informal para los desarrolladores.

C. Acceso sin identificación para facilitar reutilización.

D. Cambios inmediatos sin compatibilidad para acelerar la evolución.

Respuesta correcta: A.

Diagnóstico: la gobernanza exige contrato estable y controlado. B es insuficiente, C puede vulnerar principios de seguridad y D rompe consumidores.

### Pregunta 4

¿Qué afirmación describe mejor la interoperabilidad en la Administración pública?

A. Basta con que dos sistemas usen el mismo protocolo.

B. Solo depende del proveedor tecnológico.

C. Requiere acuerdos técnicos, semánticos, organizativos y jurídicos.

D. Se consigue reemplazando todos los sistemas por una única aplicación.

Respuesta correcta: C.

Diagnóstico: el protocolo es solo una parte. La interoperabilidad incluye significado de datos, competencias, finalidad, responsabilidades y evidencias. Unificar todo en una aplicación no suele ser viable ni deseable.

### Pregunta 5

¿Por qué es importante la idempotencia en APIs que ordenan pagos, registros o notificaciones?

A. Porque permite repetir una petición sin duplicar efectos.

B. Porque hace innecesaria la auditoría.

C. Porque sustituye el control de acceso.

D. Porque impide cualquier fallo de red.

Respuesta correcta: A.

Diagnóstico: la idempotencia no evita todos los fallos, pero permite gestionar reintentos con menor riesgo. Es una propiedad clave cuando el consumidor no sabe si una operación llegó a completarse.

## Mapa final de ideas

1. Distribuir un sistema solo aporta valor si los límites siguen capacidades de negocio y responsabilidades claras.

2. Las APIs son contratos gobernados, no simples puntos técnicos de acceso.

3. En procedimientos administrativos, la trazabilidad, la finalidad del acceso y la conservación de evidencias son requisitos centrales.

4. La asincronía, los eventos y las colas ayudan en procesos largos, fallos temporales y desacoplamiento entre organismos.

5. La interoperabilidad combina dimensiones técnicas, semánticas, organizativas y jurídicas.

6. La seguridad debe cubrir autenticación, autorización, auditoría, minimización de datos y operación segura.

7. La arquitectura distribuida exige observabilidad, versionado, pruebas de contrato y gobierno sostenido.
