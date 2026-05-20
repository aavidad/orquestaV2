# Arquitecturas distribuidas en la Administracion publica

## 1. Concepto y alcance

Una arquitectura distribuida es un modelo de organizacion de sistemas en el que las capacidades de negocio, los datos y los procesos se reparten entre varios componentes conectados por red. Estos componentes cooperan para prestar un servicio comun, pero pueden ejecutarse en maquinas, centros de datos, plataformas cloud o dominios administrativos distintos. En la Administracion publica, este enfoque aparece cuando un tramite combina identidad digital, registro electronico, carpeta ciudadana, pago, notificacion, archivo, firma, intermediacion de datos y sistemas sectoriales que pertenecen a organismos diferentes.

La distribucion no debe confundirse con una simple separacion fisica. Lo relevante es la division de responsabilidades, la coordinacion entre servicios y la forma de gestionar fallos parciales. En un sistema monolitico clasico, muchas funciones comparten proceso, base de datos y despliegue. En una arquitectura distribuida, cada parte puede evolucionar, escalar o fallar de forma distinta. Esto aporta flexibilidad, pero introduce complejidad: latencia, seguridad entre servicios, versionado de APIs, duplicidad de mensajes, consistencia eventual, trazabilidad, gobierno de datos y dependencia de redes que nunca son perfectas.

Los microservicios son una forma concreta de arquitectura distribuida. Consisten en servicios pequenos, autonomos y alineados con capacidades de negocio. Cada microservicio expone contratos claros, normalmente APIs sincronas o eventos asincronos, y gestiona su propio ciclo de vida. No todo sistema distribuido es de microservicios, ni toda Administracion necesita microservicios para cualquier caso. Un sistema puede estar distribuido mediante servicios compartidos, integraciones por mensajeria, buses de interoperabilidad, componentes de datos o plataformas de tramitacion. La decision debe responder a autonomia organizativa, volumen, criticidad, frecuencia de cambio y necesidad real de escalado independiente.

## 2. Acoplamiento y cohesion

La cohesion mide cuanto se relacionan entre si las responsabilidades internas de un componente. Un servicio con alta cohesion concentra funciones que pertenecen a una misma capacidad: por ejemplo, gestionar expedientes sancionadores, verificar identidad o emitir notificaciones. La baja cohesion aparece cuando un componente mezcla responsabilidades heterogeneas, como autenticacion, calculo de tasas, generacion documental y gestion de archivo. Esa mezcla dificulta entender, probar, desplegar y gobernar el sistema.

El acoplamiento mide el grado de dependencia entre componentes. En la Administracion publica, un acoplamiento excesivo se manifiesta cuando un organismo debe conocer tablas internas de otro, cuando una aplicacion depende de codigos de error no documentados, cuando una API cambia sin versionado, o cuando un tramite solo funciona si todos los sistemas participantes responden en tiempo real. El acoplamiento no puede eliminarse por completo, porque integrar sistemas implica dependencia. La arquitectura debe hacerlo explicito, controlado y soportado por contratos estables.

Un buen diseno busca alta cohesion y bajo acoplamiento. Para ello se delimitan contextos funcionales, se publican contratos de servicio, se evita compartir bases de datos entre dominios, se aplican politicas de versionado y se incorporan mecanismos de compatibilidad hacia atras. En el sector publico es especialmente importante separar el modelo interno del organismo del contrato de interoperabilidad. La API o el evento deben expresar una semantica administrativa estable, no la estructura accidental de una aplicacion legacy.

Tambien conviene distinguir acoplamiento temporal, funcional y tecnologico. El acoplamiento temporal obliga a que dos sistemas esten disponibles al mismo tiempo. Se reduce mediante colas, eventos, reintentos y procesos compensatorios. El acoplamiento funcional aparece cuando un cambio normativo en un dominio obliga a modificar muchos servicios. Se reduce con limites de dominio claros. El acoplamiento tecnologico surge cuando todos deben usar el mismo producto, libreria o base de datos. Se reduce con estandares abiertos, contratos documentados y adaptadores.

## 3. Consistencia, disponibilidad y decisiones de datos

La consistencia define que grado de acuerdo existe entre las distintas copias o vistas de la informacion. En un tramite publico puede ser necesario que el justificante de registro, el estado del expediente y la notificacion al interesado reflejen una misma realidad. Sin embargo, en sistemas distribuidos no siempre es viable mantener consistencia fuerte en todos los puntos sin sacrificar disponibilidad, rendimiento o autonomia.

La consistencia fuerte garantiza que, tras una operacion confirmada, todos los lectores observan el mismo estado. Es adecuada para pagos, asientos registrales, actos administrativos firmes o cambios con consecuencias juridicas inmediatas. Requiere transacciones estrictas, bloqueo o coordinacion, y suele aumentar latencia y fragilidad ante fallos.

La consistencia eventual acepta que distintas partes del sistema converjan con cierto retraso. Es util para indices de busqueda, paneles de seguimiento, sincronizacion de copias, analitica, notificaciones informativas o replicacion entre sedes. No significa inconsistencia descontrolada: exige reglas claras de convergencia, identificadores unicos, marcas temporales, resolucion de conflictos y comunicacion al usuario cuando el estado mostrado puede estar pendiente de actualizacion.

En la Administracion publica, el diseno debe distinguir datos de verdad juridica, datos operativos y datos derivados. El registro de entrada, una resolucion firmada o una evidencia de consentimiento suelen requerir controles fuertes. En cambio, una bandeja de seguimiento puede tolerar retrasos si se informa adecuadamente y si el dato juridico permanece protegido. Esta separacion evita sobredimensionar todo el sistema y permite aplicar garantias donde realmente importan.

El uso de transacciones distribuidas debe evaluarse con cautela. Coordinar varias bases de datos u organismos en una unica transaccion global puede ser costoso y fragil. Con frecuencia es mejor modelar procesos de larga duracion mediante sagas: una secuencia de pasos locales, cada uno con confirmacion propia, eventos y acciones compensatorias. Por ejemplo, si un pago se completa pero la generacion documental falla, el sistema puede reintentar, marcar el expediente como pendiente tecnico o iniciar una compensacion administrativa, sin ocultar el estado real.

## 4. Disponibilidad, resiliencia e idempotencia

La disponibilidad es la capacidad de prestar servicio cuando se necesita. En servicios publicos digitales no solo importa el porcentaje anual de disponibilidad, sino el impacto en plazos legales, ventanas de presentacion, campanas masivas, colectivos vulnerables y continuidad institucional. Un sistema de cita previa, una sede electronica o un servicio de notificaciones pueden requerir estrategias distintas segun criticidad, horario, volumen y alternativas presenciales o asistidas.

La resiliencia es la capacidad de absorber fallos y recuperarse sin degradar de forma inaceptable el servicio. En arquitecturas distribuidas hay fallos parciales: un servicio responde lento, una cola se satura, un proveedor externo no esta disponible, un certificado caduca, una replica queda atrasada o una red interadministrativa tiene cortes. El diseno debe asumir esos fallos como normales. Patrones como timeouts, circuit breakers, bulkheads, reintentos con espera progresiva, colas de aparcamiento y degradacion controlada evitan que una incidencia local colapse todo el sistema.

La idempotencia es esencial en integraciones publicas. Una operacion idempotente produce el mismo resultado aunque se repita una o varias veces con la misma clave logica. Si una persona presenta una solicitud y la red falla antes de recibir confirmacion, el sistema debe poder procesar el reintento sin duplicar expedientes ni cargos. Para ello se usan claves de idempotencia, identificadores de solicitud, deduplicacion de mensajes, registros de operaciones y respuestas repetibles. La idempotencia no es solo una propiedad tecnica; protege derechos, evita cargas administrativas y facilita auditoria.

Los reintentos deben disenarse con prudencia. Reintentar de inmediato y sin limite puede empeorar una caida. Lo adecuado es aplicar tiempos de espera, limites, variacion aleatoria, clasificacion de errores y mecanismos de aparcamiento para revision posterior. Los errores funcionales, como datos invalidos o falta de legitimacion, no deben reintentarse como si fueran errores tecnicos. Los errores transitorios, como un timeout, pueden pasar por una politica de reintentos controlada.

La degradacion elegante permite mantener una parte del servicio aunque otra no este disponible. Por ejemplo, una sede puede registrar la solicitud y dejar la consulta de ciertos datos para un procesamiento posterior, siempre que la normativa lo permita y quede constancia. Tambien puede mostrar estados parciales, habilitar canales alternativos o diferir tareas no criticas. La clave es no prometer al ciudadano un resultado que el sistema aun no ha consolidado.

## 5. Escalabilidad y rendimiento

La escalabilidad es la capacidad de aumentar la carga atendida sin redisenar completamente el sistema. Puede ser vertical, ampliando recursos de una instancia, u horizontal, anadiendo instancias. Las arquitecturas distribuidas facilitan el escalado horizontal, pero solo si los servicios son suficientemente autonomos, sin estado local imprescindible y con almacenamiento preparado para concurrencia.

En la Administracion publica hay patrones de carga muy marcados: vencimientos de convocatorias, campanas tributarias, plazos de subvenciones, matriculaciones, elecciones, oposiciones o citas sanitarias. La arquitectura debe prever picos y no basarse solo en medias. Esto implica pruebas de carga realistas, colas para absorber avalanchas, limites de tasa, cache de datos no sensibles, separacion entre lectura y escritura, procesamiento diferido y priorizacion de operaciones criticas.

La escalabilidad no se resuelve multiplicando microservicios. Un exceso de servicios pequenos puede aumentar llamadas de red, complejidad operativa y coste de observabilidad. La frontera correcta es aquella que permite evolucion y operacion independiente sin fragmentar artificialmente un proceso. En sistemas publicos, donde los cambios normativos suelen afectar flujos completos, conviene alinear los servicios con capacidades administrativas estables y no con capas tecnicas genericas.

El rendimiento debe medirse extremo a extremo. No basta con que una API individual responda rapido si el tramite completo depende de diez llamadas secuenciales. Deben analizarse latencias acumuladas, puntos de bloqueo, consultas pesadas, serializacion de documentos, firma, validaciones externas y tiempos de cola. Las decisiones de cache deben respetar proteccion de datos, vigencia de certificados, revocaciones, autorizaciones y trazabilidad.

## 6. Observabilidad y gobierno operacional

La observabilidad permite entender el comportamiento interno del sistema a partir de senales externas. En arquitecturas distribuidas se apoya en registros estructurados, metricas, trazas distribuidas, eventos de auditoria y paneles de salud. Para la Administracion publica, la observabilidad debe cubrir tanto aspectos tecnicos como administrativos: expediente afectado, operacion realizada, organismo responsable, estado del tramite, evidencias generadas y tiempos relevantes.

Las trazas distribuidas son especialmente utiles cuando una peticion cruza varios servicios. Cada operacion debe portar un identificador de correlacion que permita reconstruir el camino sin exponer datos personales innecesarios. Los logs deben ser estructurados, con niveles coherentes y politicas de retencion. Las metricas deben incluir latencia, tasa de error, saturacion, volumen de colas, reintentos, duplicados detectados, circuitos abiertos y resultados funcionales.

La observabilidad no sustituye a la auditoria. La auditoria se centra en responsabilidad, evidencia y cumplimiento: quien hizo que, cuando, con que autorizacion y con que efecto. Los sistemas publicos requieren separar logs tecnicos, evidencias administrativas y registros de seguridad, aplicando minimizacion de datos, control de acceso y conservacion conforme al marco aplicable.

Tambien es necesario un gobierno operacional claro. Cada servicio debe tener responsable, nivel de servicio, procedimiento de incidencia, politica de versionado, dependencias declaradas y plan de continuidad. Cuando participan varios organismos, los acuerdos de interoperabilidad deben fijar semantica de datos, tiempos de respuesta, gestion de cambios, ventanas de mantenimiento y mecanismos de comunicacion ante incidencias.

## 7. Patrones utiles

El patron API Gateway centraliza entrada, autenticacion, autorizacion, limitacion de tasa, enrutamiento y observabilidad de APIs. Debe evitar convertirse en un monolito de logica de negocio. Su funcion es proteger y ordenar el acceso, no concentrar reglas administrativas que pertenecen a servicios de dominio.

El patron Backend for Frontend adapta APIs a canales concretos, como sede electronica, aplicacion movil, puesto de atencion presencial o integracion maquina a maquina. Permite optimizar experiencia y seguridad por canal sin forzar a los servicios internos a conocer detalles de presentacion.

La mensajeria asincrona desacopla tiempos y absorbe picos. Es util para notificaciones, generacion documental, sincronizacion de estados, analitica, indexacion y comunicacion entre organismos. Requiere diseno de idempotencia, orden, deduplicacion, reintentos y tratamiento de mensajes no procesables.

El patron Outbox publica eventos de forma fiable junto con cambios locales de datos. En vez de actualizar una base de datos y luego enviar un mensaje de forma separada, registra el evento en la misma transaccion local y lo publica posteriormente. Reduce perdidas y duplicidades, aunque no elimina la necesidad de consumidores idempotentes.

La saga modela procesos administrativos largos con pasos independientes y compensaciones. Encaja en tramitaciones con pagos, verificaciones, subsanaciones, informes, resoluciones y notificaciones. Puede ser coreografiada, cuando los servicios reaccionan a eventos, o coordinada, cuando un componente ordena los pasos. En entornos publicos suele ser conveniente una coordinacion explicita cuando hay consecuencias juridicas, plazos o necesidad de explicabilidad.

El patron Strangler permite modernizar sistemas legacy sustituyendo capacidades de forma incremental. Se enrutan nuevas funciones a servicios modernos mientras el sistema antiguo sigue operando hasta que se reduce su alcance. Es especialmente practico cuando no se puede detener una aplicacion critica ni migrar todos los expedientes de una vez.

## 8. Anti-patrones frecuentes en Administracion publica

Un primer anti-patron es distribuir sin autonomia real. Si todos los servicios comparten la misma base de datos, despliegue, modelo interno y equipo de aprobacion, la arquitectura tiene el coste de la distribucion sin sus beneficios. Cambiar una tabla puede romper muchos consumidores y cualquier despliegue exige coordinacion global.

Otro anti-patron es convertir el bus de integracion en un centro de logica de negocio. La plataforma de integracion debe transformar, enrutar y asegurar comunicaciones, pero no deberia acumular reglas administrativas opacas. Cuando lo hace, las decisiones quedan fuera de los dominios responsables y se dificulta auditar por que se adopto un resultado.

Tambien es peligroso usar APIs como exposicion directa de sistemas legacy. Publicar endpoints que replican pantallas, tablas o procedimientos internos genera contratos fragiles. La API debe representar capacidades estables y comprensibles para consumidores externos, con validaciones, versionado y semantica documentada.

La sincronia excesiva es otro problema habitual. Un tramite que necesita respuesta inmediata de demasiados sistemas externos queda expuesto a cualquier caida parcial. En muchos casos conviene registrar la solicitud, emitir acuse y procesar verificaciones de forma diferida, siempre que se respeten garantias juridicas y expectativas del ciudadano.

La falta de versionado rompe ecosistemas. En la Administracion publica, una API puede ser consumida por varios organismos, proveedores o aplicaciones de terceros. Cambios incompatibles sin periodo de convivencia generan interrupciones y costes. Deben existir versiones, avisos, pruebas de contrato y politicas de retirada.

Por ultimo, la observabilidad tardia suele aparecer cuando solo se piensa en despliegue inicial. Sin trazas, metricas y auditoria desde el diseno, las incidencias entre organismos se convierten en intercambios manuales de capturas, fechas aproximadas y sospechas. La trazabilidad debe nacer con la arquitectura, no anadirse cuando ya hay crisis.

## 9. Criterios de aplicacion

Para decidir una arquitectura distribuida en la Administracion publica conviene partir de preguntas concretas: que capacidad administrativa se quiere aislar, que datos son juridicamente maestros, que disponibilidad exige el servicio, que fallos se pueden tolerar, que operaciones deben ser idempotentes, que contratos consumiran otros organismos, como se observara el proceso y quien respondera ante una incidencia.

La buena arquitectura no es la mas novedosa, sino la que hace explicitas las responsabilidades y los compromisos. Un sistema distribuido bien disenado permite evolucionar por dominios, resistir fallos parciales, escalar cargas publicas, integrar organismos y preservar evidencia. Mal disenado, multiplica dependencias invisibles y hace que cada incidencia sea mas dificil de diagnosticar. En servicios publicos digitales, la arquitectura debe proteger continuidad, seguridad juridica, calidad de servicio y confianza ciudadana.
