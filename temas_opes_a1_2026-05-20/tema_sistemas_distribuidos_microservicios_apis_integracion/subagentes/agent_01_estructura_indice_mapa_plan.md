# Entrega agente 01: estructura, indice, mapa conceptual y plan de secciones

Tema de trabajo: Arquitecturas distribuidas, microservicios, APIs e integracion de sistemas en la Administracion publica.

Este documento aporta material de estructura para ensamblar un tema A1 completo. La cabecera, las notas de integracion y las marcas de reparto no deben trasladarse literalmente al tema final. Las partes que si pueden reutilizarse son el indice editorial, el mapa conceptual, las definiciones, las tablas propuestas, los supuestos practicos, las notas de test separadas y el plan de visuales.

## 1. Enfoque general del tema

La idea central del tema debe ser que las arquitecturas distribuidas, los microservicios, las APIs y la integracion de sistemas no son una moda tecnica aislada. En el sector publico son una forma de construir servicios digitales interoperables, seguros, mantenibles y reutilizables, siempre subordinada al marco juridico, a la responsabilidad administrativa, a la proteccion de datos, a la seguridad, a la accesibilidad y a la continuidad del servicio.

El tema debe evitar dos errores de enfoque. El primero es convertirlo en una lista de tecnologias: contenedores, pasarelas, colas, mensajeria, mallas de servicio o formatos de intercambio. Esos elementos deben aparecer, pero como medios para resolver problemas publicos concretos. El segundo es presentar los microservicios como solucion universal. Para examen A1 conviene insistir en el criterio de adecuacion: una arquitectura distribuida puede aportar escalabilidad, autonomia de equipos y evolucion independiente, pero tambien introduce latencia, complejidad operativa, consistencia eventual, observabilidad exigente y mayor superficie de ataque.

La respuesta madura debe conectar tres planos:

| Plano | Pregunta que responde | Contenido clave |
| --- | --- | --- |
| Administrativo | Que servicio publico se presta y bajo que garantias | Procedimiento, derechos de la ciudadania, interoperabilidad, evidencia administrativa, archivo, accesibilidad, continuidad |
| Arquitectonico | Como se organiza el sistema para prestar ese servicio | Capas, dominios, componentes, contratos, APIs, eventos, integracion con sistemas legados |
| Operativo | Como se gobierna, protege y evoluciona en produccion | Seguridad, observabilidad, pruebas, despliegue, versionado, catalogo, trazabilidad, gestion de cambios |

El hilo conductor recomendado es "del derecho al servicio digital, del servicio digital al contrato tecnico y del contrato tecnico a la operacion gobernada". Asi se evita que el tema parezca un manual de herramientas y se refuerza la preparacion para preguntas de desarrollo, supuestos y test.

## 2. Orientacion de examen

La orientacion inicial del tema deberia ocupar unas 800-1.000 palabras. Debe explicar al opositor que este contenido suele evaluarse de tres maneras:

1. Como tema teorico de arquitectura: se preguntan conceptos, ventajas, riesgos, patrones y criterios de eleccion.
2. Como tema transversal de Administracion digital: se pide relacionar interoperabilidad, seguridad, identidad, procedimiento y reutilizacion.
3. Como supuesto practico: se describe una necesidad de integracion entre organismos, una modernizacion de un sistema legado o la apertura de servicios mediante APIs, y se espera una solucion razonada.

La orientacion debe advertir que el examinador valora especialmente:

- Distinguir arquitectura distribuida, SOA, microservicios, APIs e integracion, sin mezclarlos.
- Relacionar interoperabilidad organizativa, semantica y tecnica con decisiones reales de arquitectura.
- Justificar cuando conviene una API sincrona, cuando un evento asincrono y cuando una integracion por lote sigue siendo aceptable.
- Incorporar seguridad desde el diseno, no como anexo.
- Reconocer limites: consistencia eventual, transacciones distribuidas, duplicidad de datos, dependencia de red, gobierno de versiones y observabilidad.
- Aterrizar en el sector publico: sede electronica, carpeta ciudadana, registro, notificaciones, intermediacion de datos, archivo, identidad, firma, accesibilidad y proteccion de datos.

Una formulacion de apertura reutilizable seria:

"En la Administracion publica, una arquitectura distribuida no se justifica por distribuir componentes, sino por permitir que varios organos, aplicaciones y niveles administrativos colaboren de forma segura, trazable e interoperable para prestar servicios digitales. Los microservicios, las APIs y los mecanismos de integracion son instrumentos para materializar esa colaboracion, pero deben alinearse con el procedimiento administrativo, el Esquema Nacional de Interoperabilidad, el Esquema Nacional de Seguridad, la proteccion de datos y el ciclo de vida completo del servicio."

## 3. Mapa conceptual inicial

El mapa conceptual debe situar en el centro el servicio publico digital y desplegar seis ramas. La version visual puede basarse en el SVG entregado junto a este documento.

Rama 1: finalidad publica y marco juridico.

- Servicio publico digital.
- Derechos de las personas interesadas.
- Procedimiento administrativo electronico.
- Evidencia, expediente, registro, notificacion y archivo.
- Accesibilidad y no discriminacion tecnologica.

Rama 2: interoperabilidad.

- Interoperabilidad organizativa: acuerdos, responsabilidades, procesos y gobernanza entre administraciones.
- Interoperabilidad semantica: significado compartido de datos, vocabularios, metadatos, modelos comunes y calidad.
- Interoperabilidad tecnica: protocolos, formatos, interfaces, seguridad de comunicaciones, identificadores y estandares.
- Interoperabilidad europea: servicios publicos transfronterizos, evaluaciones, reutilizacion de soluciones y marco europeo.

Rama 3: arquitectura distribuida.

- Componentes desplegados en nodos distintos.
- Comunicacion por red.
- Fallos parciales.
- Escalabilidad horizontal.
- Latencia, tolerancia a fallos y consistencia.
- Patrones: capas, hexagonal, event-driven, API gateway, colas, sagas, CQRS cuando proceda.

Rama 4: microservicios.

- Servicios pequenos, autonomos y orientados a capacidades de negocio.
- Propiedad de datos por servicio.
- Contratos explicitos.
- Despliegue independiente.
- Observabilidad y automatizacion.
- Riesgos de fragmentacion, duplicidad y exceso de granularidad.

Rama 5: APIs e integracion.

- API como contrato de servicio.
- REST, SOAP en entornos legacy, mensajeria, eventos, intercambio batch y APIs internas.
- Gestion de ciclo de vida: diseno, publicacion, versionado, documentacion, monitorizacion y retirada.
- Gateway, catalogo, politicas de acceso, cuotas, auditoria y trazabilidad.
- Integracion con sistemas existentes y servicios comunes.

Rama 6: seguridad y operacion.

- Identidad, autenticacion, autorizacion y servicios de confianza.
- ENS, gestion de riesgos, registro de actividad y continuidad.
- Proteccion de datos desde el diseno.
- DevSecOps, pruebas, despliegue continuo, gestion de configuracion.
- Observabilidad: logs, metricas, trazas, indicadores de servicio y alertas.

## 4. Indice propuesto con presupuesto de palabras

El tema final deberia apuntar a unas 20.900-21.700 palabras. El reparto siguiente deja margen para que el agente integrador amplie ejemplos, tablas y supuestos sin superar el maximo A1.

| Num. | Seccion | Objetivo | Palabras objetivo |
| --- | --- | --- | ---: |
| 1 | Orientacion de examen y ruta de estudio | Situar el tema, explicar como se pregunta y fijar criterios de respuesta | 900 |
| 2 | Mapa inicial y conceptos clave | Presentar el esquema mental, vocabulario y relaciones entre conceptos | 1.100 |
| 3 | Marco publico de la arquitectura digital | Relacionar servicio publico, procedimiento, interoperabilidad, seguridad, accesibilidad y reutilizacion | 1.500 |
| 4 | Fundamentos de arquitecturas distribuidas | Explicar distribucion, red, fallos parciales, escalabilidad, consistencia y acoplamiento | 1.600 |
| 5 | Estilos arquitectonicos y evolucion desde sistemas legados | Comparar monolito modular, SOA, microservicios, eventos e integracion hibrida | 1.500 |
| 6 | Microservicios: criterios, diseno y limites | Definir microservicio, bounded context, datos, despliegue, observabilidad y riesgos | 1.800 |
| 7 | APIs como contratos de servicio | Desarrollar REST, recursos, contratos, versionado, documentacion, seguridad y gobierno | 1.700 |
| 8 | Integracion de sistemas publicos | Explicar sincronismo, asincronismo, mensajeria, eventos, batch, legados y servicios comunes | 1.600 |
| 9 | Interoperabilidad semantica y datos en la integracion | Conectar modelos de datos, metadatos, calidad, catalogos y reutilizacion | 1.200 |
| 10 | Seguridad, identidad y confianza | ENS, eIDAS, autorizacion, trazabilidad, cifrado, auditoria, proteccion de datos | 1.500 |
| 11 | Operacion, resiliencia y observabilidad | Continuidad, tolerancia a fallos, pruebas, monitorizacion, SRE/DevSecOps | 1.400 |
| 12 | Gobierno, contratacion y ciclo de vida | Catalogo de APIs, arquitectura corporativa, deuda tecnica, compra publica y SLA | 1.200 |
| 13 | Ejemplos y supuestos practicos guiados | Casos de integracion administrativa, modernizacion y API publica | 1.700 |
| 14 | Errores frecuentes y diferencias finas | Preparar trampas de examen y decisiones de criterio | 900 |
| 15 | Repaso final, enfoque de examen y muestra test | Cierre activo con definiciones rapidas, mapa de ideas y test progresivo | 1.500 |
| 16 | Plan de visuales y fuentes oficiales | Ubicar fuentes y visuales sin recargar el cuerpo teorico | 500 |
|  | Total orientativo |  | 21.000 |

## 5. Secuencia pedagogica recomendada

La secuencia debe ir de lo general a lo tecnico, y volver al contexto publico al cerrar cada bloque. Una estructura eficaz seria:

1. Abrir con el problema publico: prestar servicios digitales que cruzan organos, registros, sedes, plataformas y niveles administrativos.
2. Definir interoperabilidad antes de hablar de tecnologia. El opositor debe entender que una API no resuelve por si sola acuerdos organizativos ni significado de datos.
3. Explicar la arquitectura distribuida como decision de diseno: componentes separados que se comunican por red y aceptan fallos parciales.
4. Introducir microservicios como una forma concreta, no como sinonimo de cualquier sistema distribuido.
5. Presentar APIs e integracion como contratos y mecanismos de comunicacion.
6. Insertar seguridad, identidad, proteccion de datos y operacion como dimensiones transversales.
7. Cerrar con gobierno: catalogo, versionado, retirada, contratacion, medicion y mejora continua.

Debe cuidarse la carga cognitiva. En lugar de presentar veinte patrones seguidos, conviene agruparlos por problema:

| Problema | Decisiones de arquitectura | Riesgo de examen |
| --- | --- | --- |
| Necesito consultar un dato actualizado | API sincrona, cache con caducidad, autorizacion, trazabilidad | Olvidar que el organismo fuente mantiene responsabilidad sobre el dato |
| Necesito avisar de un cambio de estado | Evento, cola, suscripcion, idempotencia | Tratar el evento como si fuera una llamada sincrona garantizada |
| Necesito integrar un sistema legado | Fachada, adaptador, strangler pattern, lote temporal | Forzar microservicios sin evaluar coste y criticidad |
| Necesito abrir un servicio a terceros | API gateway, contrato, seguridad, cuotas, documentacion | Publicar datos o funcionalidades sin base juridica ni gobierno |
| Necesito continuidad de servicio | Redundancia, circuit breaker, retry controlado, degradacion | Confundir alta disponibilidad con ausencia total de fallos |

## 6. Definiciones que conviene fijar al principio

Arquitectura distribuida: forma de organizar un sistema en componentes que se ejecutan en nodos o procesos distintos y colaboran mediante comunicaciones de red. Su ventaja es permitir escalado, separacion de responsabilidades y resiliencia, pero exige disenar para latencia, fallos parciales, seguridad de comunicaciones, observabilidad y consistencia.

Sistema distribuido: conjunto de componentes autonomos que cooperan para ofrecer una funcionalidad percibida como un servicio coherente. En el sector publico puede abarcar aplicaciones de varios organos, plataformas comunes, servicios de verificacion, registros, pasarelas de pago, notificaciones, archivo y sistemas de terceros.

Microservicio: servicio pequeno y autonomo, orientado a una capacidad de negocio o dominio, con contrato explicito, despliegue independiente y responsabilidad clara sobre sus datos y reglas. No debe definirse por numero de lineas, lenguaje de programacion ni contenedor, sino por autonomia, cohesion y bajo acoplamiento.

API: interfaz programable que expone capacidades o datos mediante un contrato tecnico y funcional. Una API publica o interna debe tener semantica clara, reglas de autorizacion, versionado, documentacion, monitorizacion, condiciones de uso y ciclo de vida.

Integracion de sistemas: conjunto de tecnicas, patrones, acuerdos y controles que permiten que sistemas distintos intercambien datos, eventos o funcionalidades de forma fiable. Incluye integracion sincrona, asincrona, por lotes, mediante servicios comunes, mediante adaptadores o mediante plataformas de interoperabilidad.

Interoperabilidad organizativa: capacidad de entidades y procesos para colaborar con responsabilidades, acuerdos y procedimientos compatibles. En una integracion administrativa, no basta con que el mensaje llegue; debe estar claro quien solicita, quien responde, con que base juridica, con que nivel de servicio y con que evidencia.

Interoperabilidad semantica: capacidad de compartir datos con significado comun. Exige vocabularios, modelos de datos, identificadores, metadatos y reglas de calidad. Es la diferencia entre intercambiar un campo llamado "estado" y saber si significa estado civil, estado del expediente, estado de una solicitud o estado de una notificacion.

Interoperabilidad tecnica: capacidad de los sistemas para conectarse mediante protocolos, formatos, seguridad, identificadores y estandares compatibles. Incluye HTTP, TLS, JSON, XML, firma, certificados, mensajeria, catalogos de servicios y normas tecnicas.

Contrato de API: especificacion que indica recursos, operaciones, entradas, salidas, errores, codigos, seguridad, restricciones, versiones y comportamiento esperado. Debe tratarse como un compromiso estable entre proveedor y consumidor.

Acoplamiento: grado de dependencia entre componentes. Una arquitectura distribuida busca reducir el acoplamiento funcional y de despliegue, pero puede aumentar el acoplamiento temporal o de datos si se diseña mal.

Cohesion: grado en que las responsabilidades internas de un componente pertenecen al mismo dominio o capacidad. Un microservicio bien diseñado tiene alta cohesion; un servicio que mezcla expedientes, pagos, notificaciones y usuarios suele ser un monolito distribuido.

Consistencia eventual: modelo en el que diferentes replicas o servicios pueden no reflejar el mismo estado durante un periodo limitado, pero convergen si no hay nuevas actualizaciones y los procesos de sincronizacion funcionan. Debe explicarse con ejemplos administrativos para evitar que parezca una licencia para perder integridad.

Idempotencia: propiedad de una operacion que produce el mismo efecto aunque se ejecute mas de una vez con la misma intencion. Es esencial en reintentos, eventos y mensajeria, porque en sistemas distribuidos un mensaje puede llegar tarde, duplicado o despues de una duda sobre el resultado.

Observabilidad: capacidad de comprender el comportamiento interno de un sistema a partir de señales externas, como logs, metricas y trazas. En microservicios no es un lujo, porque una incidencia puede atravesar muchos componentes y organizaciones.

## 7. Desarrollo teorico por bloques

### 7.1 Marco publico de la arquitectura digital

Objetivo: que el lector entienda que la arquitectura sirve a derechos, procedimientos y garantias.

Contenido que debe incluir:

- La Administracion publica como red de servicios, no como aplicacion unica.
- Tramitacion electronica, expediente, registro, notificacion, archivo y evidencia.
- Principio de interoperabilidad y cooperacion entre administraciones.
- No discriminacion por eleccion tecnologica y accesibilidad.
- Reutilizacion de informacion publica y datos abiertos cuando proceda.
- Seguridad como condicion de confianza, continuidad y cumplimiento.
- Diferencia entre interoperabilidad juridica, organizativa, semantica y tecnica.

Ejemplo breve recomendable: un ciudadano solicita una ayuda y la Administracion comprueba identidad, empadronamiento, renta, discapacidad y cuenta bancaria mediante servicios de varios organismos. El valor publico no esta en una API aislada, sino en que el procedimiento sea legal, comprensible, trazable, seguro y no obligue a aportar documentos que ya obran en poder de la Administracion cuando la normativa lo permite.

### 7.2 Fundamentos de arquitecturas distribuidas

Objetivo: explicar las propiedades tecnicas basicas antes de saltar a microservicios.

Ideas a desarrollar:

- Distribucion fisica y logica.
- Comunicacion por red como fuente de latencia y fallo.
- Fallos parciales: un componente puede estar disponible mientras otro falla.
- Escalabilidad horizontal y vertical.
- Replicacion, balanceo y particionamiento.
- Consistencia, disponibilidad y tolerancia a particiones como tensiones de diseno.
- Acoplamiento temporal, funcional y de datos.
- Transacciones distribuidas y su dificultad.
- Necesidad de idempotencia, reintentos, timeouts y circuit breakers.

Modo tutor:

- Que significa: no todos los componentes comparten memoria, proceso ni ciclo de vida.
- Por que importa: permite evolucion y escalado, pero obliga a disenar para incertidumbre.
- Con que se confunde: con "tener varios servidores" o "usar la nube".
- Como se reconoce en examen: aparecen palabras como latencia, fallos parciales, consistencia, red, replicas, asincronia o escalabilidad.

### 7.3 Estilos arquitectonicos y evolucion

Objetivo: colocar microservicios dentro de una familia de opciones.

Comparaciones necesarias:

| Estilo | Rasgo principal | Ventaja | Riesgo | Encaje publico |
| --- | --- | --- | --- | --- |
| Monolito modular | Una aplicacion desplegada como unidad, con modulos internos | Simplicidad operativa | Evolucion lenta si no hay modularidad real | Valido para sistemas estables y equipos pequenos |
| SOA | Servicios compartidos, a menudo con bus corporativo | Reutilizacion e integracion | Gobernanza pesada y dependencia central | Util en plataformas comunes y legados |
| Microservicios | Servicios autonomos por dominio | Despliegue y escalado independientes | Complejidad operativa | Valido si hay madurez DevSecOps y dominios claros |
| Event-driven | Comunicacion por eventos | Desacoplamiento temporal | Trazabilidad y consistencia mas complejas | Adecuado para cambios de estado, avisos y sincronizacion |
| Integracion por lotes | Intercambio periodico | Robustez y bajo coste | Datos no inmediatos | Aceptable para procesos no interactivos |

Debe explicarse el patron strangler fig: envolver un sistema legado con una fachada o API, extraer capacidades de forma gradual y reducir riesgo sin sustituir todo de golpe. Es muy util para supuestos de modernizacion administrativa.

### 7.4 Microservicios

Objetivo: ofrecer una definicion A1, criterios de diseño y limites.

Ejes:

- Dominio y bounded context.
- Base de datos por servicio como principio, no dogma absoluto.
- Comunicacion sincrona y asincrona.
- Despliegue independiente.
- Automatizacion de pruebas y despliegues.
- Observabilidad desde el inicio.
- Gobierno de contratos.
- Seguridad servicio a servicio.

Advertencias:

- No partir por capas tecnicas. Un microservicio "usuarios", otro "controladores" y otro "base de datos" no es una arquitectura de microservicios, sino una separacion artificial.
- No crear servicios minusculos sin autonomia real.
- No compartir una base de datos como mecanismo principal de integracion.
- No ignorar la consistencia de expedientes y actos administrativos.
- No introducir microservicios sin capacidad operativa para monitorizar, desplegar y proteger.

Ejemplo: en una plataforma de ayudas publicas, podria haber capacidades separadas para convocatoria, solicitud, evaluacion, pago, notificacion y consulta ciudadana. Sin embargo, la separacion debe respetar el procedimiento y los datos maestros. Si todos los servicios dependen de una tabla central no versionada, la autonomia es aparente.

### 7.5 APIs como contratos de servicio

Objetivo: que el lector sepa explicar API design de forma funcional, juridica y operativa.

Elementos:

- API como frontera entre proveedor y consumidor.
- Diferencia entre API interna, API de colaboracion interadministrativa y API publica.
- REST y recursos: identificadores, verbos, representaciones, codigos de estado.
- SOAP y servicios web en contextos con contratos formales, WS-Security o legados.
- OpenAPI u otras especificaciones como documentacion ejecutable.
- Versionado: URI, cabeceras, compatibilidad hacia atras y calendario de retirada.
- Errores: codigos claros, correlacion, mensajes no sensibles.
- Seguridad: autenticacion, autorizacion, certificados, tokens, firma, cifrado, cuotas.
- Gobierno: catalogo, propietario, nivel de servicio, pruebas de contrato, monitorizacion.

Tabla recomendada:

| Decision | Buena practica | Error frecuente |
| --- | --- | --- |
| Nombrar recursos | Usar nombres de dominio comprensibles | Exponer nombres de tablas internas |
| Versionar | Mantener compatibilidad y anunciar cambios | Romper consumidores sin transicion |
| Errores | Distinguir validacion, autorizacion, no encontrado y fallo interno | Devolver siempre error generico |
| Seguridad | Autorizar por contexto y finalidad | Confiar solo en estar dentro de la red |
| Documentacion | Mantener contrato actualizado y ejemplos | Publicar un PDF desfasado |

### 7.6 Integracion de sistemas publicos

Objetivo: cubrir los mecanismos reales de integracion en Administracion.

Bloques:

- Integracion sincrona: consulta en tiempo real cuando el usuario espera respuesta o el procedimiento necesita validacion inmediata.
- Integracion asincrona: eventos, colas y mensajeria para desacoplar tiempos y absorber picos.
- Integracion por lotes: cargas periodicas para analitica, migraciones, conciliaciones o procesos no interactivos.
- Integracion con legados: adaptadores, fachadas, transformaciones, bus, ETL temporal y gobernanza de datos.
- Plataformas comunes: identidad, firma, notificaciones, registro, intercambio de datos, archivo y carpeta ciudadana.
- Trazabilidad: identificador de correlacion, registro de acceso, finalidad y evidencia.

El tema debe explicar que no existe un mecanismo unico. La pregunta correcta es: que necesita el procedimiento, que calidad del dato se exige, que latencia es aceptable, que base juridica habilita el intercambio, quien responde ante errores y como se audita.

### 7.7 Interoperabilidad semantica y datos

Objetivo: conectar con el segundo gran tema de la ola sin invadirlo, porque aqui se trata la semantica al servicio de integracion.

Contenido:

- Modelos comunes de datos.
- Metadatos y vocabularios controlados.
- Identificadores persistentes.
- Calidad: completitud, validez, actualidad, unicidad y trazabilidad.
- Transformaciones y mapeos entre dominios.
- Evitar semantica implicita en nombres tecnicos.
- Reutilizacion de informacion publica y datos abiertos cuando el regimen juridico lo permita.

Ejemplo: dos organismos pueden intercambiar un JSON correcto tecnicamente y, aun asi, no ser interoperables si "fecha de alta" significa fecha de solicitud en uno y fecha de resolucion en otro. La semantica debe documentarse, gobernarse y probarse.

### 7.8 Seguridad, identidad y confianza

Objetivo: integrar ENS, eIDAS, identidad y proteccion de datos sin convertirlo en un tema de seguridad independiente.

Puntos:

- Gestion de riesgos y medidas de seguridad proporcionales.
- Identidad de usuarios, empleados publicos, aplicaciones y servicios.
- Autenticacion y autorizacion contextual.
- Principio de minimo privilegio.
- Cifrado en transito y, cuando proceda, en reposo.
- Trazabilidad de accesos y operaciones.
- Firma, sello, certificados y servicios de confianza.
- Proteccion de datos desde el diseno y por defecto.
- Seguridad de APIs: gateway, rate limiting, validacion de entrada, proteccion ante abuso, gestion de secretos.
- Seguridad servicio a servicio: mTLS, politicas, rotacion de certificados, identidad de workload.

En examen es importante no decir que la seguridad se resuelve "poniendo OAuth" o "usando certificados". Esas son piezas. La respuesta completa incluye base juridica, finalidad, autorizacion, registro de actividad, minimizacion, conservacion, gestion de riesgos y respuesta ante incidentes.

### 7.9 Operacion, resiliencia y observabilidad

Objetivo: mostrar madurez A1. La arquitectura no termina al desplegar.

Ideas:

- DevSecOps como integracion de desarrollo, seguridad y operaciones.
- Integracion continua, pruebas automatizadas, pruebas de contrato y analisis de seguridad.
- Despliegues progresivos, rollback, feature flags cuando proceda.
- Monitorizacion de disponibilidad, latencia, errores y saturacion.
- Trazas distribuidas para reconstruir una transaccion.
- Logs con correlacion y sin exponer datos personales innecesarios.
- Gestion de capacidad y picos de demanda.
- Continuidad, respaldo y recuperacion.
- Caida controlada y degradacion elegante.

Ejemplo: si falla el servicio de notificaciones, el tramite no deberia perder la solicitud. Debe quedar registro, reintento controlado, aviso operativo y una forma de reconciliar el estado.

### 7.10 Gobierno y ciclo de vida

Objetivo: cerrar el tema con responsabilidad directiva.

Contenido:

- Arquitectura corporativa.
- Catalogo de APIs y servicios.
- Propietario funcional y tecnico.
- Politicas de versionado y retirada.
- Contratos de nivel de servicio.
- Gestion de deuda tecnica.
- Compra publica: evitar dependencia de proveedor, exigir portabilidad, documentacion, pruebas, seguridad y transferencia de conocimiento.
- Evaluacion de interoperabilidad y reutilizacion de soluciones.
- Indicadores: disponibilidad, latencia, uso, errores, adopcion, coste y satisfaccion.

Mensaje de cierre: una Administracion no gobierna solo codigo. Gobierna servicios, datos, contratos, riesgos y responsabilidades.

## 8. Tablas que deberia contener el tema final

### Tabla A: conceptos que se confunden

| Concepto | No es | Clave de examen |
| --- | --- | --- |
| Arquitectura distribuida | Simplemente tener servidores separados | Componentes que colaboran por red y pueden fallar parcialmente |
| Microservicios | Cualquier API pequeña | Servicios autonomos por dominio con despliegue y datos gobernados |
| API | Una URL tecnica | Contrato funcional, tecnico, seguro y versionado |
| Interoperabilidad | Conexion tecnica | Colaboracion organizativa, semantica y tecnica |
| Integracion | Copiar datos entre sistemas | Hacer que procesos y sistemas cooperen con garantia |
| Observabilidad | Guardar logs | Poder reconstruir comportamiento y diagnosticar fallos |

### Tabla B: cuando elegir cada mecanismo

| Necesidad | Mecanismo preferente | Motivo | Precaucion |
| --- | --- | --- | --- |
| Validacion inmediata en un tramite | API sincrona | Respuesta en el flujo de usuario | Timeouts, autorizacion y fallback |
| Propagar cambio de estado | Evento asincrono | Desacopla productores y consumidores | Idempotencia y trazabilidad |
| Conciliacion periodica | Lote | Simple y robusto | Control de calidad y ventana temporal |
| Exponer servicio a varios consumidores | API gestionada | Contrato comun y catalogo | Versionado y cuotas |
| Modernizar legado | Fachada/adaptador | Reduce riesgo y protege consumidores | No perpetuar deuda sin plan |

### Tabla C: riesgos y controles

| Riesgo | Efecto | Control |
| --- | --- | --- |
| Acoplamiento excesivo entre servicios | Cambios en cascada | Contratos estables, versionado, pruebas de contrato |
| Fallo parcial | Interrupciones dificiles de diagnosticar | Timeouts, circuit breakers, observabilidad |
| Duplicidad de datos | Inconsistencias | Propiedad clara, eventos, reconciliacion |
| API sin gobierno | Abuso, cambios incompatibles, inseguridad | Catalogo, gateway, politicas, auditoria |
| Semantica ambigua | Decisiones administrativas incorrectas | Modelos comunes y validacion semantica |
| Secretos mal gestionados | Compromiso de sistemas | Rotacion, vault, minimo privilegio |

## 9. Supuestos practicos propuestos

### Supuesto 1: ayuda publica con consulta interadministrativa

Situacion: una comunidad autonoma tramita ayudas y necesita verificar identidad, residencia, renta y discapacidad sin exigir documentos que puedan consultarse por via administrativa.

Pistas:

- Hay varios organismos fuente.
- No todos los datos tienen la misma frecuencia de actualizacion.
- Debe quedar evidencia de la consulta.
- La persona interesada debe poder entender el tramite.
- Deben aplicarse seguridad, finalidad y minimizacion.

Preguntas:

1. Que arquitectura de integracion propondrias.
2. Que datos consultarias en tiempo real y cuales podrian precargarse.
3. Que controles de seguridad y trazabilidad incorporarias.
4. Como tratarias indisponibilidades.

Resolucion esperada:

- APIs o servicios interoperables para consultas criticas.
- Eventos o lotes para sincronizaciones no interactivas.
- Identificadores de correlacion y registro de finalidad.
- Autorizacion por procedimiento y rol.
- Fallback administrativo cuando un servicio fuente no responda.
- No copiar mas datos de los necesarios.

### Supuesto 2: modernizacion de un sistema legado de expedientes

Situacion: un organismo tiene una aplicacion monolitica de expedientes que no puede sustituirse de golpe. Se quiere abrir consulta ciudadana, integrar notificaciones y mejorar mantenimiento.

Pistas:

- El legado sigue siendo sistema de registro.
- Hay riesgo de interrupcion del servicio.
- Existen consumidores internos y externos.
- Se desea evolucion gradual.

Resolucion esperada:

- Fachada/API delante del legado.
- Extraccion gradual por capacidades.
- Strangler pattern.
- Pruebas de contrato para consumidores.
- Observabilidad desde la fachada.
- Plan de retirada de dependencias directas a la base de datos.

### Supuesto 3: API publica de datos de contratos

Situacion: se quiere exponer informacion de contratacion publica para reutilizacion, con datos actualizados y uso por terceros.

Pistas:

- Datos abiertos y reutilizacion.
- Necesidad de documentacion y estabilidad.
- Riesgo de datos personales o informacion no publicable.
- Consumo intensivo por terceros.

Resolucion esperada:

- API documentada y catalogada.
- Versionado y condiciones de uso.
- Control de cuotas y monitorizacion.
- Separacion entre datos publicables y datos internos.
- Formatos abiertos y metadatos.
- Canal de cambios y retirada.

### Supuesto 4: integracion europea transfronteriza

Situacion: un servicio publico debe interoperar con otro Estado miembro para verificar una condicion administrativa.

Pistas:

- Diferencias semanticas entre paises.
- Identidad digital y servicios de confianza.
- Necesidad de trazabilidad y base juridica.
- Evaluacion de interoperabilidad.

Resolucion esperada:

- Marco europeo de interoperabilidad.
- Contratos semanticos y tecnicos.
- Identidad y confianza conforme a normativa europea.
- Registro de intercambio y minimizacion de datos.
- Mecanismo de gestion de errores y soporte entre autoridades.

## 10. Notas de test separadas de la teoria

Estas notas deben colocarse en cajas diferenciadas, no mezcladas con el desarrollo principal.

Nota de test 1: si la pregunta dice "interoperabilidad", no contestar solo con protocolos. La respuesta completa debe incluir organizacion, semantica y tecnica.

Nota de test 2: si aparece "microservicios", revisar si hay autonomia real. Si todos comparten base de datos y despliegue, probablemente no son microservicios maduros.

Nota de test 3: una API no es segura por ser interna. En entornos distribuidos debe haber identidad, autorizacion, cifrado y auditoria tambien entre servicios.

Nota de test 4: asincrono no significa sin control. Los eventos necesitan orden cuando sea relevante, idempotencia, reintentos, colas de error y observabilidad.

Nota de test 5: el sector publico no puede sacrificar garantias procedimentales por velocidad tecnica. La arquitectura debe conservar evidencia, trazabilidad, accesibilidad, seguridad y base juridica.

Muestra de preguntas:

| Nivel | Pregunta | Respuesta esperada |
| --- | --- | --- |
| Base | Que diferencia hay entre API e integracion | La API es un contrato de exposicion; la integracion es el conjunto de mecanismos y acuerdos para que sistemas cooperen |
| Aplicacion | Cuando elegirias un evento frente a una llamada REST | Cuando interesa desacoplar tiempos y propagar cambios sin bloquear al productor |
| Examen real | Un sistema de ayudas falla cuando no responde un servicio de renta. Que mejoras propones | Timeouts, fallback, trazabilidad, reintentos controlados, cola, degradacion y procedimiento alternativo |

## 11. Errores frecuentes que debe corregir el tema

1. Presentar microservicios como obligatorio para toda Administracion.
2. Confundir arquitectura distribuida con nube.
3. Confundir API con punto final HTTP sin contrato.
4. Olvidar interoperabilidad semantica.
5. Ignorar sistemas legados y migracion gradual.
6. Diseñar servicios por tablas de base de datos en lugar de dominios.
7. Usar reintentos sin idempotencia.
8. No prever trazas distribuidas.
9. Tratar la seguridad como un filtro al final.
10. Abrir datos o servicios sin base juridica, minimizacion ni control de acceso.
11. No versionar APIs.
12. No prever retirada de versiones antiguas.
13. Confundir alta disponibilidad con no fallar nunca.
14. Meter el banco completo de preguntas dentro del tema principal.
15. Saturar el tema con siglas sin explicar.

## 12. Repaso final sugerido

El repaso debe reducir el tema a siete ideas:

1. La arquitectura digital publica empieza por el servicio y sus garantias.
2. La interoperabilidad tiene dimensiones organizativa, semantica y tecnica.
3. Un sistema distribuido gana modularidad y escalabilidad, pero introduce fallos parciales.
4. Los microservicios solo son adecuados con dominios claros y madurez operativa.
5. Las APIs son contratos gobernados, no simples URLs.
6. La integracion combina sincronismo, asincronismo, lotes, eventos y adaptadores segun el caso.
7. Seguridad, observabilidad y ciclo de vida son condiciones de produccion, no complementos.

Definiciones rapidas para memorizar:

| Termino | Formula corta |
| --- | --- |
| API | Contrato programable para consumir datos o capacidades |
| Microservicio | Servicio autonomo por dominio, con contrato y ciclo de vida propio |
| Interoperabilidad | Capacidad de organizaciones y sistemas para cooperar con significado y reglas compartidas |
| Evento | Notificacion de algo relevante que ha ocurrido en un dominio |
| Idempotencia | Repetir una operacion no cambia el resultado mas alla del primer efecto valido |
| Observabilidad | Capacidad de diagnosticar el sistema a partir de logs, metricas y trazas |

Preguntas de recuperacion activa:

- Que problema publico resuelve una arquitectura distribuida bien gobernada.
- Que diferencia hay entre SOA y microservicios.
- Por que una API necesita propietario funcional.
- Que ocurre si dos servicios comparten base de datos sin contrato.
- Como se audita una consulta interadministrativa.
- Que riesgos aparecen al usar eventos.
- Que controles exige una API expuesta a terceros.

## 13. Plan de visuales

Visual 1: mapa conceptual del tema.

- Tipo: SVG local.
- Funcion: primera orientacion.
- Ubicacion recomendada: despues de la orientacion de examen.
- Elementos: servicio publico digital en el centro y seis ramas.
- Archivo sugerido: `assets/mapa_sistemas_distribuidos_apis_integracion.svg`.

Visual 2: flujo de una solicitud administrativa distribuida.

- Tipo: diagrama SVG o HTML/CSS.
- Funcion: explicar comunicacion entre sede, identidad, API gateway, servicios de tramite, registro, notificaciones y archivo.
- Ubicacion: bloque de integracion.
- Debe tener scroll horizontal en movil.

Visual 3: comparacion monolito modular, SOA y microservicios.

- Tipo: tabla visual o esquema de tres columnas.
- Funcion: fijar diferencias sin simplificar en exceso.
- Ubicacion: estilos arquitectonicos.

Visual 4: patron strangler.

- Tipo: diagrama de evolucion en tres fases.
- Funcion: mostrar modernizacion gradual de legado.
- Ubicacion: evolucion desde sistemas legados.

Visual 5: observabilidad distribuida.

- Tipo: diagrama de traza con correlacion.
- Funcion: mostrar como una peticion cruza componentes y genera logs, metricas y trazas.
- Ubicacion: operacion.

Visual 6: decision API/evento/lote.

- Tipo: arbol de decision sencillo.
- Funcion: resolver supuestos practicos.
- Ubicacion: integracion y repaso.

Reglas visuales:

- No usar visuales decorativos.
- Rotulos breves y legibles.
- Contenedor responsivo con overflow horizontal.
- En HTML, permitir ocultar visuales en primera lectura.
- Mantener textos localizables cuando el tema se publique.

## 14. Fuentes oficiales y tecnicas prioritarias

La bibliografia del tema final debe priorizar fuentes oficiales y no mostrar URLs reales en el cuerpo del tema. Para el archivo de fuentes, conviene clasificar:

### Marco espanol

- Ley 39/2015, de Procedimiento Administrativo Comun de las Administraciones Publicas.
- Ley 40/2015, de Regimen Juridico del Sector Publico.
- Real Decreto 203/2021, Reglamento de actuacion y funcionamiento del sector publico por medios electronicos.
- Real Decreto 4/2010, Esquema Nacional de Interoperabilidad.
- Real Decreto 311/2022, Esquema Nacional de Seguridad.
- Real Decreto 1112/2018, accesibilidad de sitios web y aplicaciones moviles del sector publico.
- Ley 37/2007, sobre reutilizacion de la informacion del sector publico.
- Normas Tecnicas de Interoperabilidad y guias publicadas por el Portal de Administracion Electronica.
- Guias CCN-STIC cuando se trate seguridad, bastionado, auditoria o servicios en la nube.

### Marco europeo

- Reglamento (UE) 2024/903, Europa Interoperable.
- Marco Europeo de Interoperabilidad de la Comision Europea.
- Reglamento (UE) 910/2014, eIDAS, y modificacion por Reglamento (UE) 2024/1183 sobre identidad digital europea.
- Directiva (UE) 2019/1024, datos abiertos y reutilizacion de informacion del sector publico.
- Reglamento (UE) 2016/679, proteccion de datos, para privacidad desde el diseno y minimizacion.
- Reglamento (UE) 2018/1724, pasarela digital unica, si se trata prestacion transfronteriza.

### Referencias tecnicas de apoyo

- NIST SP 800-204, seguridad en sistemas basados en microservicios.
- NIST SP 800-204A, uso de service mesh en aplicaciones de microservicios.
- NIST SP 800-207, arquitectura Zero Trust, solo como apoyo conceptual.
- OWASP API Security Top 10, como apoyo practico no normativo.
- OpenAPI Specification, como referencia de contratos de APIs.

## 15. Recomendaciones para el ensamblado del tema

1. Mantener primera lectura continua: el desarrollo teorico debe poder leerse sin tablas ni cajas de test.
2. Colocar tablas despues de explicar el concepto, nunca como sustituto.
3. Usar modo tutor en definiciones dificiles: que significa, por que importa, con que se confunde y como se reconoce en examen.
4. Reservar las trampas para notas de test, errores frecuentes y enfoque de examen.
5. No prometer que microservicios reducen automaticamente costes; explicar tradeoffs.
6. Evitar siglas sin expansion inicial: ENI, ENS, API, REST, SOAP, SSO, SLA, SLO, CI/CD.
7. Cerrar cada bloque tecnico con una frase de Administracion publica.
8. Incluir ejemplos concretos pero no dependientes de una plataforma comercial.
9. No exponer URLs reales en el tema final; archivarlas en fuentes.
10. Si el tema final no alcanza 20.250 palabras, crear informe de bloqueo y no declararlo listo.
