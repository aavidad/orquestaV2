# Tablas comparativas integrables

Las tablas deben ir acompanadas de explicacion. Antes de cada tabla conviene
definir el criterio de comparacion; despues, anadir un ejemplo administrativo.

## T01. Enfoques arquitectonicos

| Enfoque | Idea central | Ventajas | Riesgos | Uso razonable en Administracion publica |
| --- | --- | --- | --- | --- |
| Monolito modular | Una aplicacion unica organizada en modulos internos | Simplicidad operativa, transacciones locales, menor complejidad inicial | Acoplamiento si los modulos no estan bien delimitados, despliegue conjunto | Aplicaciones departamentales estables con equipo pequeno y cambios previsibles |
| SOA | Servicios reutilizables con contratos, a menudo coordinados por bus o plataforma de integracion | Reutilizacion, integracion de sistemas heterogeneos, gobierno centralizado | Exceso de mediacion, dependencia de productos centrales, contratos pesados | Integracion corporativa entre organismos, sistemas legados y servicios comunes |
| Microservicios | Servicios pequenos, autonomos y desplegables de forma independiente | Escalado selectivo, autonomia de equipos, evolucion por dominios | Complejidad distribuida, observabilidad exigente, consistencia eventual | Plataformas con alta evolucion funcional, equipos maduros y necesidad de despliegue frecuente |
| Arquitectura orientada a eventos | Productores publican eventos y consumidores reaccionan de forma asincrona | Desacoplamiento, absorcion de picos, extension sin bloquear el tramite | Duplicados, orden, idempotencia, trazabilidad mas compleja | Notificaciones, sincronizacion entre registros, expedientes y analitica operacional |
| Serverless gestionado | Funciones o servicios gestionados que se ejecutan bajo demanda | Menor administracion de infraestructura, elasticidad, pago por uso | Dependencia de proveedor, limites de ejecucion, observabilidad especifica | Tareas puntuales, integraciones ligeras, procesos de baja duracion y eventos administrativos |

Ejemplo integrable:

> Un portal de cita previa estable puede funcionar correctamente como monolito
> modular. En cambio, una plataforma de tramitacion con pagos, notificaciones,
> identidad, carpeta ciudadana y servicios de consulta a terceros puede requerir
> separar capacidades y gobernar contratos de integracion.

## T02. Estilos de integracion

| Estilo | Cuando encaja | Fortalezas | Precauciones | Trampa de examen |
| --- | --- | --- | --- | --- |
| API REST/JSON | Consulta o modificacion sincrona de recursos, consumo web o movil | Sencillez, amplio soporte, cache y contratos documentables | Versionado, seguridad, paginacion, errores coherentes, idempotencia | Pensar que REST es solo "usar HTTP"; requiere modelar recursos y semantica |
| SOAP/XML | Integraciones formales con contrato WSDL, seguridad WS-* o legado institucional | Contrato estricto, herramientas maduras, compatibilidad con sistemas existentes | Verbosidad, complejidad, gobierno de esquemas, evolucion dificil | Darlo por obsoleto en todo caso; sigue siendo valido si el contexto lo exige |
| Mensajeria asincrona | Procesos no bloqueantes, colas, eventos, integracion entre sistemas desacoplados | Resiliencia, absorcion de picos, desacoplamiento temporal | Duplicados, orden, reintentos, idempotencia, trazabilidad | Usarla para respuestas inmediatas cuando el ciudadano espera confirmacion sincrona |
| Batch/ETL | Cargas periodicas, sincronizaciones masivas o consolidacion de datos | Eficiencia en volumen, control de ventanas, menor presion online | Latencia, reconciliacion, calidad de datos, trazabilidad | Confundirlo con interoperabilidad en tiempo real |
| Ficheros intercambiados | Sistemas antiguos, bajo volumen o procedimientos cerrados | Simplicidad inicial, bajo acoplamiento tecnologico | Fragilidad, control de versiones, errores manuales, seguridad del transporte | Aceptarlo sin controles de integridad, cifrado y auditoria |

Ejemplo integrable:

> La consulta del estado de un expediente desde una sede electronica suele
> necesitar respuesta sincrona. La comunicacion de que el expediente ha cambiado
> de fase puede publicarse como evento para que otros sistemas actualicen sus
> vistas sin bloquear la operacion principal.

## T03. Gobierno del ciclo de vida de APIs

| Fase | Pregunta clave | Practica esperada | Evidencia de madurez |
| --- | --- | --- | --- |
| Descubrimiento | Que necesidad publica o integracion justifica la API | Identificar consumidores, datos, finalidad, base juridica y nivel de servicio | Ficha de API con responsable funcional y tecnico |
| Diseno | Que contrato estable se ofrece | Especificacion OpenAPI, errores normalizados, paginacion, filtros, versionado | Revision de contrato antes del desarrollo |
| Seguridad | Quien puede consumir y bajo que condiciones | Autenticacion, autorizacion, minimizacion, cifrado y auditoria | Politicas aplicadas en gateway y registros de acceso |
| Pruebas | Como se evita romper consumidores | Tests de contrato, pruebas de carga, validacion de esquemas | Pipeline con validaciones automaticas |
| Publicacion | Como se facilita el consumo correcto | Portal de APIs, documentacion, ejemplos y condiciones de uso | Catalogo actualizado y trazabilidad de versiones |
| Operacion | Como se mide y se corrige | Metricas, logs, trazas, alertas y SLO | Cuadros de mando y gestion de incidencias |
| Evolucion y retirada | Como se cambia sin dano | Deprecacion, coexistencia de versiones y comunicacion previa | Calendario de retirada y consumidores inventariados |

Nota de tutor integrable:

> Una API publica o interadministrativa no es solo un endpoint tecnico. Es un
> compromiso de servicio: contrato, seguridad, disponibilidad, soporte, versionado
> y responsabilidad.

## T04. Patrones de resiliencia

| Patron | Problema que resuelve | Como se reconoce | Riesgo si se aplica mal |
| --- | --- | --- | --- |
| Timeout | Evita esperas indefinidas ante servicios lentos | Se fija un tiempo maximo de espera por llamada | Cortar operaciones que necesitaban respuesta mas lenta pero valida |
| Retry con backoff | Reintenta fallos transitorios sin saturar el sistema | Reintentos separados por esperas crecientes | Multiplicar carga sobre un servicio ya degradado |
| Circuit breaker | Corta temporalmente llamadas a un servicio que falla | Tras cierto umbral, abre el circuito y devuelve fallo controlado | Bloquear demasiado tiempo si no hay prueba de recuperacion |
| Bulkhead | Aisla recursos para que un fallo no se extienda | Pools, colas o limites separados por tipo de carga | Infrautilizacion si se dimensiona sin datos |
| Idempotencia | Permite repetir una operacion sin duplicar efectos | Claves de idempotencia, deduplicacion y control de estado | Creer que toda operacion HTTP es idempotente por defecto |
| Cola de errores | Conserva mensajes que no pudieron procesarse | Mensajes fallidos pasan a una cola revisable | Acumular errores sin proceso de correccion |

Nota de test integrable:

> Si el enunciado insiste en "fallo transitorio", "servicio externo inestable" o
> "picos de carga", busca patrones de resiliencia. Si insiste en "dato duplicado"
> tras reintento, piensa en idempotencia.

## T05. Capas de interoperabilidad

| Capa | Pregunta que responde | Ejemplo en integracion de sistemas | Error frecuente |
| --- | --- | --- | --- |
| Juridica | Existe habilitacion para compartir o consultar el dato | Base juridica, competencia, finalidad y proteccion de datos | Reducir interoperabilidad a una conexion tecnica |
| Organizativa | Que organismo hace que y con que responsabilidad | Convenios, acuerdos de servicio, responsables y procedimientos | No definir propietario del dato ni soporte |
| Semantica | Significa lo mismo el dato para todos | Modelos comunes, codigos, vocabularios y metadatos | Intercambiar campos con nombres parecidos pero significado distinto |
| Tecnica | Como se conectan los sistemas | APIs, servicios web, certificados, redes, formatos y protocolos | Elegir tecnologia sin resolver gobierno ni significado |

Ejemplo integrable:

> Dos sistemas pueden conectarse tecnicamente y, aun asi, no ser interoperables
> si un campo como "domicilio" tiene reglas distintas, si no existe finalidad
> administrativa o si nadie asume la calidad del dato cedido.

## T06. Seguridad e identidad en APIs e integracion

| Control | Funcion | Ejemplo | Relacion con el examen |
| --- | --- | --- | --- |
| Autenticacion | Verificar quien es el consumidor o usuario | Certificado, federacion de identidad, token firmado | No equivale a autorizacion |
| Autorizacion | Decidir que puede hacer | Scope, rol, permiso por procedimiento o finalidad | Debe aplicar minimo privilegio |
| Cifrado en transito | Proteger el canal | TLS y configuracion robusta | No resuelve por si solo trazabilidad ni finalidad |
| Auditoria | Registrar accesos y decisiones relevantes | Logs firmes, sellado temporal, correlacion de trazas | Es clave en consultas interadministrativas |
| Limitacion de tasa | Evitar abuso o saturacion | Rate limit por consumidor o aplicacion | Puede afectar disponibilidad si se define sin criticidad |
| Gestion de secretos | Proteger claves y credenciales | Rotacion, vault, no incluir secretos en codigo | Fallo habitual en integraciones improvisadas |

## T07. Diferencias de examen rapidas

| Par que se confunde | Diferencia esencial | Pista para reconocerlo |
| --- | --- | --- |
| API y microservicio | La API es contrato de acceso; el microservicio es una unidad de despliegue y responsabilidad | Puede haber API sin microservicios y microservicio con varias APIs |
| Gateway y bus de integracion | El gateway gobierna entrada a APIs; el bus integra sistemas y media mensajes | Si hay consumidores externos y control de acceso, piensa en gateway |
| Sincrono y asincrono | Sincrono espera respuesta inmediata; asincrono desacopla productor y consumidor | Si no debe bloquear el tramite, piensa en asincronia |
| Escalabilidad y disponibilidad | Escalabilidad es crecer ante carga; disponibilidad es seguir prestando servicio | Mas instancias no garantizan continuidad si hay dependencia unica |
| Interoperabilidad y reutilizacion | Interoperabilidad permite trabajar entre sistemas; reutilizacion aprovecha datos, software o servicios existentes | La reutilizacion necesita calidad, licencia, metadatos y gobierno |
| Observabilidad y monitorizacion | Monitorizacion mira indicadores conocidos; observabilidad permite investigar estados no previstos | Trazas distribuidas son pista de observabilidad |

## T08. Mini supuesto practico visualizable

Situacion:

> Una consejeria quiere permitir que los ayuntamientos consulten el estado de
> expedientes, reciban aviso de cambios relevantes y no pidan al ciudadano
> certificados que ya obran en otra administracion.

Pistas:

| Necesidad | Solucion candidata | Justificacion |
| --- | --- | --- |
| Consulta inmediata del estado | API sincrona protegida por gateway | El consumidor espera respuesta en el momento |
| Aviso de cambio de fase | Evento administrativo | Varios consumidores pueden reaccionar sin bloquear el expediente |
| Consulta de certificado externo | Intermediacion de datos | Evita pedir documentos ya disponibles con control juridico y auditoria |
| Control de accesos | Autenticacion, autorizacion y auditoria | No basta con publicar endpoint |
| Gestion de fallos | Timeouts, reintentos, idempotencia y cola de errores | La red y los sistemas externos fallan |

Resolucion breve:

> La arquitectura combinaria API sincrona para consulta, mensajeria asincrona
> para cambios de estado y plataforma de intermediacion para datos de terceros.
> El diseno debe incluir contratos versionados, registro de consumidores,
> trazabilidad, seguridad y criterios de disponibilidad.
