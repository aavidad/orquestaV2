# Microservicios y APIs en arquitecturas distribuidas de la Administración pública

## Encaje dentro del tema

En una Administración pública digital, las arquitecturas distribuidas responden a la necesidad de integrar organismos, procedimientos, sedes electrónicas, plataformas comunes y sistemas heredados con seguridad, continuidad, trazabilidad y evolución. Los microservicios organizan capacidades de negocio en componentes desplegables de forma independiente, y las APIs exponen contratos estables para aplicaciones internas, otros organismos, ciudadanía, empresas o procesos automatizados.

La idea clave es separar la lógica de dominio de los canales de acceso. Un servicio puede gestionar expedientes, notificaciones, pagos, identidad, documentos, firma, intercambio registral o consulta de datos, pero debe hacerlo mediante contratos claros, gobernados y observables. En este contexto, una API no es solo una interfaz técnica: es un compromiso operativo entre proveedor y consumidor. Define datos, operaciones, reglas de uso, errores, seguridad, niveles de servicio, versionado y ciclo de vida.

## Microservicios: límites, autonomía e integración

Un microservicio representa una capacidad funcional con responsabilidad bien delimitada. Su valor no está en ser pequeño, sino en tener un límite coherente: datos propios, reglas propias, despliegue propio y una API explícita. En el sector público, estos límites deberían alinearse con capacidades administrativas reconocibles, evitando partir un procedimiento en servicios puramente técnicos que generen acoplamiento innecesario.

Una arquitectura de microservicios bien diseñada favorece la evolución independiente. Por ejemplo, un componente de consulta de estados de expedientes puede evolucionar sin afectar al servicio de notificaciones, siempre que mantenga el contrato acordado. Esta independencia exige observabilidad, pruebas, control de versiones, gobierno de contratos y gestión de fallos; sin esos elementos, la distribución aumenta la complejidad y deteriora el servicio.

La integración entre microservicios puede ser síncrona, mediante APIs REST u otros estilos de llamada directa, o asíncrona, mediante eventos y mensajería. La llamada síncrona es adecuada cuando el consumidor necesita una respuesta inmediata: validar una identidad, recuperar un expediente o consultar un catálogo. La comunicación por eventos es más apropiada cuando interesa desacoplar procesos: expediente creado, documento firmado, pago confirmado, notificación practicada o plazo vencido. La arquitectura debe combinar ambos patrones según las necesidades de consistencia, latencia, auditoría y resiliencia.

## Contrato de API y enfoque contract-first

El contrato de una API describe de forma precisa qué ofrece el servicio y cómo debe consumirse. Incluye recursos, operaciones, parámetros, esquemas de datos, códigos de respuesta, errores, requisitos de autenticación, límites de uso y condiciones de compatibilidad. En administraciones con múltiples proveedores, organismos y equipos, el contrato es el principal instrumento para reducir ambigüedad.

El enfoque contract-first consiste en diseñar y revisar el contrato antes de implementar. Esto permite detectar incoherencias de dominio, problemas de seguridad, falta de datos obligatorios, errores de modelado o incompatibilidades con consumidores existentes. También permite generar documentación, clientes, validadores y pruebas de contrato. En entornos públicos, esta práctica es especialmente útil porque facilita la interoperabilidad entre equipos y evita que la implementación interna condicione indebidamente la interfaz pública.

Un buen contrato debe ser estable, expresivo y mínimo. Estable, porque los consumidores no pueden adaptarse continuamente a cambios arbitrarios. Expresivo, porque debe reflejar el lenguaje del dominio administrativo, no detalles internos de base de datos. Mínimo, porque exponer datos o acciones innecesarias aumenta el riesgo de seguridad, privacidad y mantenimiento.

## OpenAPI como especificación documental y ejecutable

OpenAPI es una especificación habitual para describir APIs HTTP de forma estructurada. Define endpoints, métodos, parámetros, cuerpos de petición y respuesta, esquemas, códigos de estado, mecanismos de seguridad y ejemplos. En una Administración pública, convierte la documentación de APIs en un artefacto gobernable y verificable.

La especificación OpenAPI puede usarse para revisión funcional, documentación de consumidores, validación automática en pasarelas o pruebas, generación de clientes internos y comprobación de compatibilidad entre versiones. También puede integrarse en un catálogo de APIs con titularidad, estado del ciclo de vida, criticidad, datos tratados, nivel de protección, política de uso y contacto responsable.

No debe confundirse OpenAPI con una garantía completa de calidad. Documenta la forma de la API, pero no sustituye al análisis de seguridad, a las pruebas de negocio, a la gestión de identidades ni al seguimiento operativo. Debe formar parte de una práctica más amplia de gobierno de APIs.

## REST: recursos, métodos y semántica

REST es un estilo de arquitectura muy usado para APIs HTTP. En una API REST, el diseño gira alrededor de recursos identificables: expedientes, documentos, unidades administrativas, asientos registrales, notificaciones, terceros, solicitudes o trámites. Las operaciones se expresan mediante métodos HTTP con semántica conocida: consulta, creación, actualización, sustitución, modificación parcial o eliminación, cuando proceda.

La claridad semántica es importante. Una consulta no debería producir efectos laterales relevantes; una creación debería devolver la localización o identificador del recurso creado; una actualización debe documentar si reemplaza todo el recurso o solo algunos campos. También conviene usar códigos de estado de forma coherente: éxito, creación, ausencia de contenido, petición incorrecta, no autorizado, prohibido, no encontrado, conflicto, límite excedido o error interno, entre otros.

En el ámbito público, el diseño REST debe cuidar especialmente la exposición de datos personales y administrativos. No todo recurso interno debe ser visible como recurso externo. A menudo se necesitan vistas específicas, filtros por legitimación, minimización de datos y controles de finalidad. Además, se debe evitar que identificadores previsibles permitan enumerar expedientes o consultar información sin autorización suficiente.

## Versionado y compatibilidad

El versionado de APIs permite evolucionar sin romper consumidores. La regla práctica es distinguir cambios compatibles de cambios incompatibles. Añadir un campo opcional, incorporar un nuevo filtro o ampliar una enumeración puede ser compatible si los consumidores toleran extensiones. Cambiar el significado de un campo, eliminar datos, modificar tipos, renombrar propiedades o alterar códigos de error suele ser incompatible.

Existen varias estrategias de versionado: en la ruta, en cabeceras, por negociación de contenido o mediante versiones de esquema. Lo importante no es solo la técnica, sino la política: cuándo se crea una nueva versión, cuánto tiempo se mantiene la anterior, cómo se comunica la obsolescencia, qué métricas de consumo se usan y quién aprueba la retirada.

En APIs públicas, el versionado debe ser especialmente conservador. Una API publicada para empresas, ciudadanía u otros organismos puede tener consumidores no inventariados o con ciclos largos de adaptación. En APIs internas puede existir más coordinación, pero no debe asumirse libertad absoluta para romper contratos: las integraciones internas suelen sostener procesos críticos y cadenas de tramitación.

## Eventos e integración asíncrona

Las APIs síncronas no resuelven todos los escenarios. En sistemas administrativos complejos, muchos procesos son largos, compuestos y sujetos a estados: alta de una solicitud, subsanación, informe, firma, registro, notificación, pago, archivo o interoperabilidad con otra entidad. Para estos casos, los eventos permiten publicar hechos de negocio y que otros componentes reaccionen sin acoplamiento directo.

Un evento debe representar algo que ya ha ocurrido, no una orden encubierta. Por ejemplo, "expediente admitido", "documento firmado" o "notificación puesta a disposición" expresan hechos. El consumidor decide qué hacer con esa información según su responsabilidad. Este enfoque mejora la extensibilidad: nuevos consumidores pueden incorporarse sin modificar el servicio emisor, siempre que se respete el contrato del evento.

La integración por eventos exige diseñar identificadores, marcas temporales, versión del evento, tipo, origen, correlación, datos mínimos y reglas de idempotencia. También requiere gestionar reintentos, duplicados, entregas fuera de orden y ventanas de consistencia eventual. En Administración pública, además, debe conservarse evidencia suficiente para auditoría, trazabilidad del procedimiento y explicación posterior de decisiones automatizadas o semiautomatizadas.

## API gateway y gobierno del tráfico

Un API gateway actúa como punto de entrada controlado a un conjunto de APIs. Puede encargarse de autenticación inicial, autorización delegada, terminación TLS, limitación de tasa, cuotas, validación básica, enrutamiento, registro de métricas, correlación de trazas y protección frente a patrones abusivos. En organizaciones grandes evita que cada servicio implemente de forma distinta controles transversales.

El gateway no debe convertirse en un núcleo de negocio opaco. Su función es aplicar políticas comunes y facilitar la operación, no concentrar reglas funcionales complejas. Si incorpora demasiada lógica, crea acoplamiento y dificulta la evolución independiente de los servicios.

Para APIs públicas, el gateway es también una pieza de gobierno: permite aplicar planes de consumo, claves de aplicación, límites diferenciados, observabilidad por consumidor, bloqueo de credenciales comprometidas y publicación controlada. Para APIs internas, puede ayudar a segmentar redes, aplicar confianza cero, exigir identidad de servicio y normalizar auditoría.

## BFF: backend for frontend

El patrón BFF, backend for frontend, crea una capa de API adaptada a un canal concreto: portal ciudadano, aplicación móvil, intranet de gestión, panel de atención presencial o integración con un tercero. Es útil cuando distintos canales necesitan composiciones de datos, formatos o ritmos de evolución diferentes.

Un BFF evita que el frontend conozca demasiados microservicios internos y reduce llamadas múltiples desde el cliente. También permite presentación, agregación y adaptación de respuestas sin contaminar los servicios de dominio. Por ejemplo, una sede electrónica puede mostrar en una pantalla datos de expedientes, documentos, pagos y notificaciones, mientras cada microservicio conserva su responsabilidad.

El riesgo del patrón es duplicar lógica de negocio o crear BFFs sin gobierno. Debe quedar claro que el BFF adapta la experiencia de canal, pero no decide reglas sustantivas del procedimiento. Las validaciones administrativas, autorizaciones de fondo y cambios de estado deben residir en servicios de dominio o componentes específicos de negocio.

## Autenticación y autorización

La autenticación verifica quién es el sujeto o sistema que accede. La autorización decide qué puede hacer. En APIs públicas e internas conviene separar ambos conceptos. Un consumidor puede estar correctamente autenticado y, aun así, no estar autorizado para consultar un expediente, invocar una operación o acceder a determinados atributos.

En escenarios de Administración pública se combinan identidades de ciudadanos, empleados públicos, aplicaciones, servicios y organismos. Pueden intervenir federación de identidad, certificados, OAuth 2.0, OpenID Connect, tokens firmados, mTLS, credenciales de aplicación y delegación de permisos. La elección depende del consumidor, riesgo, datos tratados y trazabilidad necesaria.

La autorización debe modelar finalidad, rol, ámbito organizativo, representación, expediente, unidad responsable y nivel de acceso. No basta con comprobar que el usuario pertenece a una aplicación. Es habitual necesitar controles por expediente, órgano, fase procedimental o habilitación normativa. Además, la API debe devolver solo lo necesario para la finalidad autorizada.

En comunicación entre servicios, la identidad de máquina también importa. Cada servicio debe poder acreditar quién llama, bajo qué contexto y con qué permisos. La confianza implícita por estar en una red interna es insuficiente en arquitecturas distribuidas modernas.

## Errores, problemas e idempotencia

El tratamiento de errores forma parte del contrato. Una API debe devolver errores previsibles, documentados y útiles para el consumidor, sin filtrar información sensible. Conviene diferenciar errores de validación, autenticación, autorización, inexistencia de recurso, conflicto de estado, límite de consumo, indisponibilidad temporal y fallos internos.

Un formato homogéneo de error mejora la integración. Puede incluir código interno estable, mensaje comprensible, detalle de campos inválidos, identificador de correlación, estado HTTP y orientación básica para el consumidor. El mensaje no debe exponer trazas internas, sentencias de base de datos ni datos personales innecesarios.

La idempotencia es relevante en operaciones de escritura, especialmente cuando hay reintentos por cortes de red. Si un consumidor envía dos veces una misma solicitud de alta o pago, la API debe evitar duplicidades o proporcionar una clave idempotente. En procedimientos administrativos, duplicar actuaciones puede tener efectos jurídicos o contables.

## Paginación, filtrado y rendimiento

Las APIs que devuelven colecciones deben paginar. Devolver todos los expedientes, documentos o registros en una sola respuesta no es escalable ni seguro. La paginación puede basarse en desplazamiento y tamaño de página, o en cursores. El modelo por desplazamiento es sencillo, pero puede ser inestable si los datos cambian durante la consulta. El modelo por cursor suele ser más robusto para listados grandes o flujos de sincronización.

El contrato debe indicar tamaño máximo de página, ordenación por defecto, campos filtrables, criterios de búsqueda, comportamiento ante filtros inválidos y metadatos de navegación. También debe evitar filtros excesivamente flexibles que permitan consultas costosas o exfiltración por enumeración. En datos sensibles, la paginación debe combinarse con autorización por fila o por recurso.

Para rendimiento, es útil distinguir endpoints de lectura operacional, búsquedas complejas, exportaciones y analítica. No todas las necesidades deben resolverse con la misma API. Una exportación masiva puede requerir proceso asíncrono, generación de artefacto, control de permisos y auditoría específica.

## Trazabilidad, auditoría y observabilidad

En sistemas distribuidos, una petición puede atravesar gateway, BFF, varios microservicios, colas, bases de datos y adaptadores externos. Sin trazabilidad, investigar incidencias o acreditar actuaciones se vuelve muy difícil. Cada llamada debe transportar identificadores de correlación y contexto suficiente para reconstruir el recorrido técnico y funcional.

La trazabilidad técnica incluye logs estructurados, métricas, trazas distribuidas, identificadores de petición, latencias, errores y dependencias. La auditoría funcional registra quién hizo qué, sobre qué recurso, con qué finalidad, desde qué aplicación, cuándo y con qué resultado. Ambas son complementarias: la primera sirve para operación y diagnóstico; la segunda para cumplimiento, control interno y evidencia administrativa.

Debe cuidarse la protección de datos en logs y trazas. Registrar demasiado puede crear riesgos: datos personales en mensajes, tokens, documentos, identificadores sensibles o información de expedientes. La observabilidad debe diseñarse con minimización, retención limitada, control de acceso y capacidad de investigación proporcionada.

## Ciclo de vida de APIs públicas e internas

Una API debe gestionarse desde su diseño hasta su retirada. El ciclo de vida comienza con la identificación de necesidad, titular funcional, consumidores previstos, datos tratados, clasificación de seguridad, modelo de autorización y nivel de servicio. Después se diseña el contrato, se revisa, se implementa, se prueba, se publica en catálogo y se opera con métricas.

Durante la explotación se deben medir uso, errores, latencias, consumidores activos, llamadas rechazadas, versiones utilizadas y cumplimiento de límites. Estos datos permiten decidir mejoras, detectar integraciones rotas, dimensionar infraestructura y planificar retiradas. En APIs públicas deben anunciarse versiones nuevas, obsolescencias, ventanas de migración y condiciones de soporte.

La retirada de una API no debe consistir en apagar un endpoint sin más. Debe incluir inventario de consumidores, análisis de impacto, alternativa disponible, periodo de convivencia, avisos, monitorización de consumo residual y fecha de desactivación. Para APIs internas, este proceso puede ser más ágil, pero sigue siendo necesario para evitar interrupciones de trámites o cadenas de integración.

La gobernanza distingue APIs públicas, privadas internas, APIs de integración entre organismos y APIs de plataforma. Cada categoría puede tener controles distintos de publicación, seguridad, documentación, pruebas y soporte. En todos los casos, la API debe tener propietario, contrato vigente, versión soportada, política de cambios y evidencias de operación.

## Ideas fuerza para el desarrollo del tema

Los microservicios y las APIs permiten modularidad e interoperabilidad, pero solo aportan valor si se acompañan de contratos estables, seguridad rigurosa, trazabilidad y gobierno del ciclo de vida. REST y OpenAPI ofrecen una base práctica para APIs síncronas; los eventos complementan ese modelo cuando se requiere desacoplamiento y procesos de larga duración. En la Administración pública, una API pública o interna debe ser comprensible, versionada, segura, paginada, observable y auditable, sin exponer complejidad interna ni comprometer datos personales o expedientes.
