# Entrega parcial - desarrollo teorico principal

> Metadatos de trabajo para integracion editorial: subagente 03, rol desarrollo teorico principal. Esta cabecera no debe copiarse al temario final.

## Alcance

Este documento aporta material integrable para el tema "Arquitecturas distribuidas, microservicios, APIs e integracion de sistemas en la Administracion publica". No sustituye el `tema_a1.md` final y no declara el tema listo. El texto esta redactado para ser reutilizado por el agente integrador junto con el material de los demas subagentes.

Se incluyen:

- desarrollo teorico principal;
- definiciones operativas;
- tablas de comparacion;
- ejemplos publicos;
- modo tutor;
- notas de test separadas;
- supuestos practicos guiados;
- errores frecuentes;
- repaso final;
- plan de visuales;
- fuentes oficiales y tecnicas sin URL visible.

---

# Desarrollo teorico principal

## 1. Enfoque del tema

Las arquitecturas distribuidas, los microservicios, las APIs y la integracion de sistemas no son modas tecnicas aisladas. En una Administracion publica son la forma material de conseguir que procedimientos, registros, expedientes, datos, identidades, notificaciones, pagos, portales y servicios comunes funcionen como un sistema administrativo coherente, aunque esten desarrollados por organismos distintos, con tecnologias distintas y con responsabilidades juridicas distintas.

El punto central para un tema A1 es comprender que una arquitectura distribuida no se justifica porque "sea moderna", sino porque permite repartir responsabilidades, escalar servicios, reutilizar capacidades, conectar organizaciones y mantener sistemas complejos sin convertir cada cambio en una reforma total. Esa ventaja tiene un precio: aumenta la complejidad de la comunicacion, la observabilidad, la seguridad, la consistencia de datos, la gobernanza contractual y la operacion. En el sector publico ese precio se incrementa porque la tecnologia debe respetar el procedimiento administrativo, la interoperabilidad, la seguridad nacional de los sistemas de informacion, la proteccion de datos personales, la accesibilidad, la trazabilidad, la conservacion documental y la rendicion de cuentas.

Una respuesta de examen madura debe evitar dos extremos. El primero es presentar los microservicios como una receta universal. El segundo es reducir la integracion de sistemas a una lista de siglas tecnicas. Lo importante es relacionar cada decision arquitectonica con una necesidad administrativa: interoperar, reutilizar, evitar pedir datos ya disponibles, conservar evidencia, garantizar disponibilidad, proteger derechos y sostener servicios publicos digitales durante anos.

En este tema conviene usar una idea conductora: la Administracion digital necesita sistemas autonomos en su construccion, pero coordinados en su comportamiento. La autonomia permite que cada unidad evolucione su servicio; la coordinacion evita que el ciudadano, la empresa o el empleado publico perciban una suma incoherente de islas tecnologicas.

## 2. Conceptos basicos y definiciones

Una arquitectura distribuida es un modelo en el que las funciones de una aplicacion o de un ecosistema de aplicaciones se reparten entre varios componentes que se ejecutan en nodos, procesos, servicios o plataformas diferentes y que se comunican mediante red. Lo distribuido no se define solo por estar "en varios servidores"; se define por la existencia de componentes separados que cooperan a traves de contratos de comunicacion.

Un sistema distribuido aparece cuando varias partes independientes deben actuar como un todo. En una Administracion publica, un expediente puede requerir identidad, registro, consulta de datos, firma, notificacion, archivo, pasarela de pagos y sistemas sectoriales. Cada pieza puede estar gestionada por un organismo distinto, pero el procedimiento exige una experiencia y una trazabilidad completas.

Un monolito es una aplicacion desplegada como una unidad principal. Puede estar internamente bien modularizada, pero suele compartir proceso, base de datos y ciclo de despliegue. No es necesariamente malo. En procedimientos simples, equipos pequenos o dominios muy cohesionados, un monolito modular puede ser mas gobernable que una constelacion prematura de servicios.

Un microservicio es un servicio pequeno en comparacion con el sistema global, orientado a una capacidad de negocio o administrativa concreta, desplegable de forma independiente, con contrato de comunicacion explicito y con alta cohesion interna. La palabra "pequeno" no debe interpretarse de forma cuantitativa. Lo relevante es que tenga una responsabilidad clara, un modelo de datos propio y un ciclo de vida razonablemente autonomo.

Una API, o interfaz de programacion de aplicaciones, es un contrato que permite a un consumidor invocar capacidades de un proveedor de forma documentada. En el contexto de Administracion digital, una API no es solo un endpoint tecnico; es una frontera de responsabilidad. Define que datos se aceptan, que datos se devuelven, que errores se comunican, que autenticacion se exige, que nivel de servicio se espera y que version contractual esta vigente.

La integracion de sistemas es el conjunto de patrones, plataformas, contratos y procesos que permiten que sistemas distintos intercambien datos, soliciten servicios, coordinen procesos o publiquen eventos. Puede ser sincrona, cuando un sistema espera una respuesta inmediata, o asincrona, cuando un sistema emite un mensaje o evento y otro lo procesa despues.

La interoperabilidad es la capacidad de organizaciones y sistemas para trabajar juntos. En el marco publico no es una comodidad tecnica, sino un requisito de buena administracion. Comprende dimensiones juridicas, organizativas, semanticas y tecnicas. La interoperabilidad juridica asegura que las normas no impidan el intercambio; la organizativa alinea procesos y responsabilidades; la semantica da significado comun a los datos; la tecnica establece formatos, protocolos y requisitos de conexion.

La reutilizacion es la posibilidad de aprovechar sistemas, aplicaciones, componentes, modelos de datos o servicios ya existentes para evitar duplicidades y reducir coste publico. En el sector publico espanol se conecta con el principio de consultar soluciones disponibles antes de adquirir o desarrollar nuevas aplicaciones cuando puedan satisfacer la necesidad.

La resiliencia es la capacidad de un sistema para seguir prestando servicio, degradarse de forma controlada y recuperarse ante fallos. En arquitecturas distribuidas es esencial porque la red falla, un proveedor puede no responder, una cola puede acumular mensajes y una dependencia externa puede degradarse. Disenar para el fallo no es pesimismo: es profesionalidad operativa.

La observabilidad es la capacidad de comprender el estado interno de un sistema a partir de sus salidas externas: logs, metricas, trazas, eventos de auditoria y estados de salud. En microservicios, sin observabilidad, un error de ciudadano puede convertirse en una investigacion manual entre diez equipos.

## 3. Administracion publica: por que el contexto cambia la arquitectura

En una empresa privada, la arquitectura suele evaluarse por coste, velocidad de entrega, escalabilidad y experiencia de usuario. En la Administracion publica esos criterios siguen siendo importantes, pero se subordinan a un marco mas amplio. Un sistema publico debe ser legal, interoperable, seguro, accesible, auditable, conservable y neutral respecto a la tecnologia cuando asi lo exija el marco normativo.

La Ley 39/2015 articula el procedimiento administrativo comun y situa la relacion electronica como parte ordinaria del funcionamiento administrativo. La Ley 40/2015 regula el regimen juridico del sector publico e incorpora la cooperacion entre Administraciones, la interoperabilidad y la reutilizacion como elementos relevantes para el funcionamiento electronico. El Real Decreto 203/2021 desarrolla la actuacion y funcionamiento del sector publico por medios electronicos. El Esquema Nacional de Interoperabilidad fija criterios para asegurar que informacion, formatos, aplicaciones y servicios puedan interoperar. El Esquema Nacional de Seguridad establece principios y requisitos para proteger la informacion tratada por medios electronicos.

La consecuencia practica es clara: una API publica no puede disenarse igual que una API interna de un producto comercial. Debe tener en cuenta identificacion, firma, consentimiento o habilitacion legal, minimizacion de datos, registro de accesos, conservacion de evidencias, trazabilidad del expediente, accesibilidad, gestion de incidencias y continuidad del servicio. Ademas, su evolucion debe evitar romper procedimientos que dependen de ella.

Tambien cambia el concepto de usuario. En una arquitectura publica hay ciudadanos, empresas, empleados publicos, organos administrativos, otras Administraciones, proveedores tecnologicos, sistemas de auditoria, organos de control y, en ocasiones, instituciones europeas. Cada actor tiene derechos, obligaciones y niveles de confianza distintos.

Un error habitual consiste en tratar la integracion administrativa como si fuera una conexion entre aplicaciones equivalentes. No lo es. Un intercambio de datos entre Administraciones puede implicar base juridica, finalidad concreta, minimizacion, consentimiento cuando proceda, trazabilidad, control de acceso y prueba de que el dato se uso dentro del procedimiento correcto. La tecnologia debe hacer posible todo eso, no ocultarlo.

## 4. Evolucion: de sistemas aislados a ecosistemas de servicios

La Administracion digital ha evolucionado desde aplicaciones departamentales aisladas hacia ecosistemas de servicios comunes, plataformas compartidas e integraciones entre organismos. En los sistemas aislados, cada unidad resolvia su necesidad con su propia aplicacion, su propio modelo de datos y sus propios canales. Ese modelo puede ser rapido al principio, pero genera duplicidad, falta de informacion compartida, integraciones punto a punto y costes crecientes de mantenimiento.

El siguiente paso fue la integracion mediante buses, plataformas comunes y servicios horizontales. Aparecen registros electronicos, servicios de firma, notificacion, intermediacion de datos, archivo, pasarelas de pago, directorios administrativos y redes de comunicaciones. El valor no esta solo en que existan herramientas, sino en que se convierten en capacidades reutilizables por muchos procedimientos.

Con el tiempo, la demanda de servicios digitales mas flexibles impulsa arquitecturas orientadas a APIs, eventos y componentes desplegables de forma independiente. En este punto entran los microservicios. No sustituyen necesariamente a todas las aplicaciones existentes, sino que pueden convivir con monolitos, sistemas legados, plataformas comunes y capas de integracion.

La evolucion correcta no es "monolito malo, microservicio bueno". La evolucion correcta es pasar de acoplamientos ocultos a contratos claros. Un monolito modular con dominios bien definidos puede ser una base excelente. Un conjunto de microservicios con datos compartidos, contratos improvisados y despliegues no coordinados puede ser peor que el monolito original.

| Modelo | Ventaja principal | Riesgo principal | Uso razonable en Administracion |
| --- | --- | --- | --- |
| Aplicacion departamental aislada | Rapidez local y control directo | Duplicidad e integraciones fragiles | Casos pequenos sin intercambio relevante |
| Monolito modular | Simplicidad operativa y coherencia transaccional | Ciclo de despliegue unico y crecimiento excesivo | Procedimientos cohesionados y equipos acotados |
| SOA o servicios corporativos | Reutilizacion de capacidades comunes | Gobernanza pesada si se burocratiza | Servicios horizontales y plataformas compartidas |
| Microservicios | Despliegue independiente y escalado selectivo | Complejidad distribuida y observabilidad exigente | Dominios con alta evolucion, carga diferenciada o equipos independientes |
| Arquitectura orientada a eventos | Desacoplamiento temporal y trazabilidad de hechos | Consistencia eventual y depuracion compleja | Expedientes, notificaciones, auditoria, sincronizacion y procesos largos |

## 5. Principios de diseno en arquitecturas distribuidas publicas

El primer principio es la separacion de responsabilidades. Cada componente debe tener una funcion comprensible. En un sistema de ayudas publicas, por ejemplo, no conviene mezclar gestion de solicitudes, validacion de identidad, calculo de baremo, consulta de datos tributarios, notificacion y archivo en una sola pieza indivisible si esas capacidades tienen ritmos de cambio y responsabilidades distintas.

El segundo principio es el contrato explicito. La comunicacion entre servicios debe describirse con suficiente precision: esquemas de datos, codigos de error, reglas de versionado, requisitos de seguridad, trazabilidad y condiciones de uso. En APIs REST, OpenAPI puede ayudar a documentar y validar contratos. En eventos, el contrato debe definir el nombre del evento, su significado administrativo, el esquema del payload, la politica de compatibilidad y las garantias de entrega.

El tercer principio es la autonomia con responsabilidad. Un servicio autonomo no significa un servicio sin gobierno. Significa que puede evolucionar dentro de sus limites, pero debe respetar los contratos publicados y las obligaciones de seguridad, interoperabilidad y trazabilidad. En Administracion, la autonomia tecnica nunca elimina la responsabilidad juridica del organo competente ni la responsabilidad funcional del servicio publico.

El cuarto principio es la minimizacion del acoplamiento. Dos servicios estan acoplados no solo cuando se llaman entre si, sino cuando comparten base de datos, conocen detalles internos del otro, dependen de despliegues simultaneos o interpretan campos no documentados. El acoplamiento oculto es una de las principales causas de fallos en arquitecturas distribuidas.

El quinto principio es la observabilidad desde el diseno. Cada peticion relevante debe poder correlacionarse. Cada evento administrativo debe poder rastrearse. Cada error debe aportar informacion suficiente para diagnosticar sin exponer datos personales innecesarios. La trazabilidad tecnica y la auditoria administrativa no son lo mismo, pero deben coordinarse.

El sexto principio es la seguridad por defecto. La seguridad no puede anadirse al final como una capa de cortafuegos. Debe aparecer en la identificacion de consumidores, autorizacion de operaciones, cifrado de comunicaciones, gestion de secretos, validacion de entrada, proteccion frente a abuso, segregacion de entornos, registro de accesos y respuesta ante incidentes.

El septimo principio es la evolucion compatible. Las APIs y eventos de Administracion suelen tener consumidores que no pueden adaptarse inmediatamente. El versionado, la deprecacion controlada y la compatibilidad hacia atras son parte de la obligacion de continuidad del servicio.

## 6. Microservicios: definicion tecnica y criterio de uso

Los microservicios organizan una aplicacion como un conjunto de servicios independientes, cada uno centrado en una capacidad concreta. Lo mas importante no es el tamano del servicio, sino el limite del dominio. Un microservicio bien definido responde a una pregunta funcional: identidad, notificaciones, expediente, catalogo de procedimientos, cita previa, pago, verificacion de datos, archivo o analitica operativa. Un microservicio mal definido responde a una division tecnica superficial: "servicio de controladores", "servicio de tablas", "servicio de utilidades".

La division por dominios se apoya en el concepto de contexto delimitado. Cada contexto tiene su propio vocabulario y sus propias reglas. El termino "interesado", por ejemplo, puede tener matices distintos en un expediente sancionador, en una subvencion, en un registro de representantes o en una notificacion. Separar contextos evita que un modelo de datos unico, supuestamente universal, acabe lleno de excepciones.

Un microservicio debe poseer sus datos o, al menos, controlar el acceso a los datos que forman parte de su responsabilidad. Compartir una base de datos entre varios microservicios suele destruir la independencia prometida. Si todos los servicios leen y escriben las mismas tablas, el despliegue independiente se convierte en una ficcion, porque cualquier cambio de esquema puede romper consumidores ocultos.

En la Administracion publica, la propiedad de los datos debe interpretarse con cautela. Un servicio tecnico puede custodiar o exponer datos, pero la titularidad, la competencia y la finalidad pertenecen al organo o Administracion correspondiente. Por eso conviene distinguir propiedad tecnica del dato, responsabilidad funcional y base juridica del tratamiento.

Los microservicios son adecuados cuando hay dominios diferenciados, equipos capaces de operar sus servicios, necesidades de escalado distintas, cambios frecuentes y una plataforma comun de observabilidad, seguridad y despliegue. No son adecuados cuando el equipo es pequeno, el dominio esta poco entendido, la operacion no esta madura o se busca resolver un problema organizativo mediante fragmentacion tecnica.

| Decision | Senal favorable a microservicios | Senal favorable a monolito modular |
| --- | --- | --- |
| Ritmo de cambio | Partes cambian con frecuencia distinta | Cambios afectan al conjunto |
| Equipo | Equipos autonomos con capacidad operativa | Un equipo pequeno y unico |
| Datos | Dominios con datos y reglas separables | Modelo de datos muy cohesionado |
| Escalado | Cargas muy diferentes por modulo | Carga homogenea |
| Resiliencia | Se necesita aislar fallos por capacidad | El coste de distribuir supera el beneficio |
| Gobierno | Contratos, observabilidad y CI/CD maduros | Falta plataforma operativa comun |

### Modo tutor: microservicio

Un microservicio significa una unidad de servicio con responsabilidad de dominio, contrato explicito y despliegue independiente. Importa porque permite evolucionar y escalar partes del sistema sin paralizar el conjunto. Se confunde con "servicio pequeno", "endpoint" o "contenedor". En examen se reconoce porque la pregunta suele mencionar autonomia de despliegue, acoplamiento bajo, datos propios, comunicacion por API o eventos y necesidad de observabilidad distribuida.

> Nota de test: si una opcion dice que los microservicios siempre reducen la complejidad, es sospechosa. Reducen cierto acoplamiento interno, pero aumentan complejidad de red, consistencia, seguridad, despliegue y observabilidad.

## 7. APIs: contrato, producto publico y frontera de responsabilidad

Las APIs son el mecanismo habitual para exponer capacidades entre sistemas. En Administracion pueden ser internas, interadministrativas o abiertas. Una API interna conecta componentes de una misma organizacion. Una API interadministrativa permite que otra Administracion consulte o invoque una capacidad dentro de un marco habilitante. Una API abierta puede permitir reutilizacion publica de datos o servicios bajo condiciones determinadas.

Una API bien gobernada tiene documentacion, version, autenticacion, autorizacion, semantica de errores, limites de uso, trazabilidad, responsable funcional, responsable tecnico, entorno de pruebas y procedimiento de cambios. Sin esos elementos, una API es solo una conexion fragil.

HTTP es el protocolo mas comun para APIs web. Sus metodos tienen semantica: GET se asocia a lectura, POST a creacion o procesamiento, PUT a sustitucion, PATCH a modificacion parcial y DELETE a eliminacion. En sistemas publicos no basta con usar el verbo correcto; hay que definir el significado administrativo de cada operacion. "Enviar solicitud", "subsanar expediente" o "consultar estado" son actos con consecuencias distintas.

REST es un estilo arquitectonico basado en recursos, identificadores, representaciones y operaciones uniformes. Una API REST madura no expone procedimientos internos, sino recursos del dominio. Por ejemplo, `expedientes`, `solicitudes`, `documentos`, `notificaciones` o `pagos`. Aun asi, muchas operaciones administrativas son procesos, no simples operaciones CRUD. En esos casos conviene modelar acciones de dominio con claridad: registrar una solicitud, iniciar una firma, emitir una notificacion o cerrar un tramite.

OpenAPI permite describir APIs HTTP de forma estructurada: rutas, metodos, parametros, cuerpos, respuestas, esquemas, seguridad y ejemplos. Su valor en Administracion es doble. Primero, mejora la coordinacion entre proveedor y consumidor. Segundo, permite validacion automatica, generacion de clientes, pruebas contractuales y documentacion consistente.

Las APIs tambien pueden seguir otros estilos. GraphQL permite al consumidor seleccionar campos, lo que puede ser util en frontales complejos, pero requiere control cuidadoso de autorizacion y rendimiento. gRPC usa contratos estrictos y comunicacion eficiente, adecuado para integraciones internas de alto rendimiento. Los servicios SOAP siguen existiendo en entornos publicos por historia, contratos formales y compatibilidad con plataformas existentes. La madurez A1 consiste en no ridiculizar tecnologias existentes, sino entender donde encajan y como gobernarlas.

| Estilo de API | Fortalezas | Riesgos | Encaje habitual |
| --- | --- | --- | --- |
| REST/HTTP | Simplicidad, estandarizacion web, cache, herramientas amplias | Disenos CRUD pobres, errores inconsistentes | APIs publicas e interadministrativas |
| SOAP | Contratos formales, WS-Security, legado corporativo | Verbosidad y menor ergonomia moderna | Integraciones historicas o reguladas |
| gRPC | Rendimiento, contratos estrictos, streaming | Menor facilidad para consumidores web simples | Microservicios internos |
| GraphQL | Flexibilidad de consulta para cliente | Autorizacion fina y control de coste complejos | Frontales ricos y agregacion de datos |
| Eventos | Desacoplamiento temporal, reaccion a hechos | Consistencia eventual y trazabilidad exigente | Procesos largos, auditoria, sincronizacion |

### Modo tutor: API como contrato

Una API no se reconoce solo por tener una URL. Se reconoce porque hay un compromiso estable entre proveedor y consumidor. Importa porque la integracion publica no puede depender de llamadas improvisadas. Se confunde con "servicio web" o con "pantalla reutilizada". En examen, las pistas son versionado, documentacion, seguridad, esquema de datos, gestion de errores y consumidor externo o interno.

> Nota de test: una API no elimina la necesidad de interoperabilidad semantica. Dos sistemas pueden comunicarse por HTTP y aun asi no entenderse si usan codigos, estados o conceptos administrativos incompatibles.

## 8. Integracion sincrona y asincrona

La integracion sincrona ocurre cuando un consumidor llama a un proveedor y espera respuesta antes de continuar. Es adecuada para consultas inmediatas, validaciones de baja latencia y operaciones que requieren confirmacion rapida. Por ejemplo, consultar si un documento esta firmado, obtener el estado de un expediente o validar un identificador en un momento concreto.

Su problema principal es el acoplamiento temporal. Si el proveedor no responde, el consumidor se bloquea o debe degradarse. En cadenas largas, una peticion puede depender de cinco servicios y fallar por cualquiera de ellos. Por eso son necesarios tiempos maximos, reintentos controlados, circuit breakers, colas de compensacion y mensajes de error claros.

La integracion asincrona ocurre cuando un sistema publica un mensaje, comando o evento, y otro lo procesa sin que el emisor espere una respuesta inmediata. Es adecuada para procesos largos, notificaciones, sincronizaciones, analitica, auditoria, integracion con legados y desacoplamiento entre organismos. En un expediente, la presentacion de una solicitud puede generar eventos para registrar entrada, iniciar validaciones, crear tarea de revision y preparar notificacion.

La integracion asincrona introduce consistencia eventual. Eso significa que distintos componentes pueden tardar en ver el mismo estado. No es un defecto si esta previsto y explicado. Es un problema si el usuario ve estados contradictorios o si el procedimiento exige una confirmacion inmediata que el sistema no puede garantizar.

| Aspecto | Sincrona | Asincrona |
| --- | --- | --- |
| Relacion temporal | El consumidor espera respuesta | El emisor continua tras publicar |
| Latencia percibida | Importante e inmediata | Puede diferirse |
| Fallo del proveedor | Impacta directamente | Se amortigua con cola y reintento |
| Consistencia | Mas inmediata | Eventual |
| Uso tipico | Consulta, validacion, operacion breve | Procesos largos, eventos, integracion |
| Riesgo | Cadenas fragiles de dependencias | Mensajes duplicados, orden y trazabilidad |

Un patron muy importante es el outbox transaccional. Cuando un servicio modifica su base de datos y debe publicar un evento, no conviene hacer ambas cosas de forma separada sin control. Si guarda el dato pero falla la publicacion, otros sistemas no se enteran. Si publica el evento pero falla la transaccion, otros sistemas creen que ocurrio algo que no existe. El outbox registra el evento en la misma transaccion local y un proceso posterior lo publica de forma fiable.

Otro patron esencial es la idempotencia. Una operacion idempotente puede repetirse sin producir efectos duplicados. En integraciones publicas, los reintentos son inevitables. Si una solicitud se envia dos veces por un corte de red, el sistema debe reconocer el identificador de operacion y evitar duplicar expedientes, pagos o notificaciones.

> Nota de test: "asincrono" no significa "sin control". Una cola sin trazabilidad, sin reintentos definidos, sin gestion de mensajes fallidos y sin idempotencia puede ser menos fiable que una llamada sincrona sencilla.

## 9. Patrones de integracion relevantes

El API Gateway centraliza la entrada a un conjunto de APIs. Puede resolver autenticacion, autorizacion inicial, limitacion de tasa, enrutamiento, transformacion ligera, registro y publicacion de documentacion. Su ventaja es concentrar politicas transversales. Su riesgo es convertirse en un punto unico de fallo o en una capa con demasiada logica de negocio.

El Backend for Frontend crea APIs especificas para un canal: portal ciudadano, aplicacion movil, intranet de empleados o integracion con otra Administracion. Evita que todos los consumidores dependan de una API excesivamente generica. En el sector publico puede mejorar accesibilidad y experiencia, pero debe evitar duplicar reglas de procedimiento en cada canal.

El bus de servicios o ESB permite orquestar, enrutar y transformar mensajes entre sistemas. Fue habitual en arquitecturas SOA. Puede ser util para integraciones corporativas complejas, pero si concentra demasiada logica se convierte en un monolito de integracion. El criterio moderno es que el bus facilite comunicacion, no que oculte dominios mal definidos.

El broker de eventos permite publicar y consumir eventos. Desacopla productores y consumidores, ayuda a absorber picos y permite que nuevos consumidores se incorporen sin modificar al productor. En Administracion puede utilizarse para trazabilidad de expedientes, sincronizacion de estados, notificaciones internas y analitica. Requiere gobierno de esquemas, control de datos personales y politicas de retencion.

El adaptador o capa anticorrupcion protege un dominio frente a modelos externos. Si un sistema legado representa estados de expediente de forma ambigua, el nuevo servicio no deberia contaminar su modelo interno con esas ambiguedades. Un adaptador traduce, valida y documenta equivalencias.

La saga coordina una transaccion distribuida mediante pasos locales y compensaciones. En una arquitectura distribuida no suele haber una transaccion unica que abarque todos los servicios. Si un alta de beneficiario implica crear expediente, reservar credito, solicitar verificacion y notificar, cada paso puede confirmarse localmente. Si un paso falla, se ejecutan compensaciones. En procedimientos administrativos hay que tener cuidado: no toda accion juridica es "compensable" tecnicamente.

El patron strangler permite sustituir progresivamente un sistema legado. En lugar de reescribir todo, se rodea el legado con APIs, se desvia funcionalidad nueva a componentes modernos y se retiran partes antiguas de forma gradual. Es especialmente relevante en Administraciones, donde los sistemas historicos suelen sostener servicios criticos.

| Patron | Para que sirve | Precaucion A1 |
| --- | --- | --- |
| API Gateway | Entrada comun, politicas transversales | No meter negocio excesivo |
| BFF | Adaptar API a canal concreto | No duplicar reglas juridicas |
| Broker de eventos | Desacoplar procesos y absorber picos | Gobernar esquemas y retencion |
| Outbox | Publicar eventos de forma fiable | Controlar duplicados e idempotencia |
| Saga | Coordinar procesos distribuidos | Validar si la compensacion es juridicamente posible |
| Anticorruption layer | Proteger dominio frente a legado | Mantener traducciones documentadas |
| Strangler | Modernizar sin big bang | Medir coexistencia y retirada real |

## 10. Datos, consistencia y transacciones

La gestion de datos es uno de los puntos mas delicados de una arquitectura distribuida. En un monolito, una transaccion de base de datos puede garantizar que varios cambios se confirmen juntos. En microservicios, cada servicio suele tener su propio almacenamiento. Eso mejora autonomia, pero dificulta mantener consistencia inmediata entre servicios.

La regla practica es no distribuir si se necesita una transaccion fuerte constante entre las partes. Si dos componentes siempre cambian juntos y no pueden aceptar divergencia temporal, quizas no sean dos microservicios. Separarlos puede crear una deuda permanente de sincronizacion.

La consistencia eventual es aceptable cuando el negocio puede tolerar que el estado tarde en propagarse. Por ejemplo, una solicitud puede aparecer como registrada en el sistema de entrada y unos segundos despues como pendiente de validacion en el sistema sectorial. Lo que no es aceptable es que el ciudadano no sepa que ha pasado, que se duplique el asiento o que se pierda la evidencia.

La integridad administrativa exige identificar los hechos relevantes. No basta con almacenar el estado final; conviene registrar eventos significativos: solicitud presentada, documento anexado, firma validada, datos consultados, subsanacion requerida, notificacion puesta a disposicion, comparecencia realizada, resolucion dictada. Estos eventos ayudan a trazabilidad, auditoria y reconstruccion del expediente.

La privacidad obliga a limitar datos. Un evento no debe publicar mas datos personales de los necesarios. A veces es mejor publicar una referencia opaca y permitir que consumidores autorizados consulten los detalles. Esto reduce exposicion y facilita aplicar controles de acceso.

La semantica de datos tambien es critica. Dos sistemas pueden usar el campo `estado` con valores similares y significados diferentes. En un sistema, "cerrado" puede significar finalizado administrativamente; en otro, puede significar archivado tecnicamente. La interoperabilidad semantica exige diccionarios, modelos comunes, codigos controlados y documentacion de equivalencias.

### Modo tutor: consistencia eventual

Consistencia eventual significa que, tras una actualizacion, los componentes distribuidos pueden tardar un tiempo en converger hacia un estado coherente. Importa porque muchas arquitecturas distribuidas la usan para ganar disponibilidad y desacoplamiento. Se confunde con inconsistencia descontrolada. En examen se reconoce por colas, eventos, replicacion, reintentos, estados intermedios y necesidad de informar al usuario.

> Nota de test: si el procedimiento exige una decision unica e inmediata, la consistencia eventual puede no ser adecuada. Si el proceso admite fases y confirmaciones, puede ser perfectamente valida.

## 11. Seguridad: ENS, identidad, autorizacion y API security

La seguridad de una arquitectura distribuida publica debe partir de una idea sencilla: cada nueva comunicacion entre servicios es una nueva superficie de ataque y un nuevo punto de control. Distribuir no reduce por si solo el riesgo. Puede aislar fallos, pero tambien multiplica credenciales, endpoints, reglas de red, secretos, logs y dependencias.

El Esquema Nacional de Seguridad exige una vision organizada de la seguridad: principios, requisitos, medidas, categorizacion, analisis de riesgos y mejora continua. Para un tema tecnico, lo importante es conectar ENS con decisiones concretas: segmentacion, control de acceso, cifrado, auditoria, continuidad, gestion de incidentes, proteccion de datos y seguridad en la cadena de suministro.

La identificacion responde a quien es el sujeto o sistema. La autenticacion verifica esa identidad. La autorizacion decide que puede hacer. En APIs publicas o interadministrativas, estas tres capas deben separarse. Saber que un consumidor es una Administracion no basta para permitirle consultar cualquier dato. Debe existir competencia, finalidad, procedimiento y permiso concreto.

OAuth 2.0 es un marco de autorizacion ampliamente usado para delegar acceso mediante tokens. OpenID Connect, construido sobre OAuth 2.0, incorpora autenticacion de identidad. JWT es un formato de token con afirmaciones o claims, pero no es una solucion completa por si solo. Un JWT mal usado puede exponer datos, aceptar firmas incorrectas, no revocar sesiones o conceder permisos excesivos.

En integraciones maquina a maquina pueden utilizarse certificados, mTLS, tokens de cliente, redes privadas, listas de permisos y politicas de gateway. En entornos publicos, la identidad electronica y los servicios de confianza se conectan con el marco eIDAS y con sus evoluciones europeas, especialmente para reconocimiento transfronterizo, firma, sello y servicios de confianza.

OWASP API Security Top 10 recuerda riesgos especificos de APIs: autorizacion rota a nivel de objeto, autenticacion deficiente, exposicion excesiva, consumo inseguro de APIs, falta de limites de recursos, configuraciones inseguras y otros errores recurrentes. En Administracion, estos riesgos pueden traducirse en acceso indebido a expedientes, consulta masiva de datos, manipulacion de identificadores, abuso de servicios o fuga de informacion personal.

| Capa | Riesgo tipico | Control esperado |
| --- | --- | --- |
| Identidad | Consumidor no verificado | Autenticacion fuerte, certificados, federacion |
| Autorizacion | Acceso a expediente ajeno | Politicas por sujeto, rol, finalidad y objeto |
| Transporte | Intercepcion o manipulacion | TLS actualizado, mTLS cuando proceda |
| Entrada | Inyeccion o payload malicioso | Validacion de esquemas y limites |
| Token | Reutilizacion o privilegio excesivo | Caducidad, audiencia, scopes, revocacion |
| Operacion | Abuso o denegacion de servicio | Rate limiting, cuotas, circuit breakers |
| Auditoria | Imposibilidad de investigar | Logs correlados, trazas y evidencias |

La seguridad tambien debe contemplar dependencias internas. Un microservicio no debe confiar automaticamente en otro solo porque esta en la red interna. El enfoque de confianza cero recomienda verificar identidad, autorizacion y contexto en cada acceso relevante. No significa desconfiar de todo de forma irracional, sino eliminar la confianza implicita por ubicacion de red.

> Nota de test: "usar HTTPS" es necesario, pero no suficiente. Una API puede estar cifrada y aun asi permitir que un usuario consulte expedientes de otro si falla la autorizacion a nivel de objeto.

## 12. Interoperabilidad juridica, organizativa, semantica y tecnica

La interoperabilidad juridica exige que el intercambio tenga cobertura normativa y respete competencias, procedimientos, proteccion de datos y conservacion de evidencias. En una consulta de datos entre Administraciones, no basta con que el endpoint funcione. Debe estar claro por que se consulta, para que procedimiento, con que base juridica, con que limites y durante cuanto tiempo se conserva la evidencia.

La interoperabilidad organizativa exige que las organizaciones alineen procesos. Si una Administracion publica un servicio de consulta pero no define soporte, horarios, responsables, niveles de servicio, gestion de incidencias y cambios, la integracion sera fragil. La tecnologia no compensa la falta de acuerdo organizativo.

La interoperabilidad semantica exige entender los datos igual. Incluye vocabularios, codigos, diccionarios, modelos de datos, metadatos y reglas de transformacion. En Administracion es decisiva porque los terminos juridicos y administrativos no son intercambiables. "Titular", "representante", "interesado", "beneficiario", "solicitante" y "tercero" pueden solaparse, pero no significan siempre lo mismo.

La interoperabilidad tecnica exige protocolos, formatos, conectividad, seguridad y especificaciones. Es la dimension mas visible para perfiles TIC, pero no la unica. Una API JSON sobre HTTPS puede ser tecnicamente interoperable y juridicamente inutil si no existe habilitacion para el intercambio.

El Reglamento sobre la Europa Interoperable refuerza la idea de interoperabilidad desde el diseno y de evaluaciones de interoperabilidad para servicios publicos digitales transeuropeos. Para un examen A1, conviene relacionarlo con el enfoque nacional: no se trata de crear sistemas aislados que despues se conectan, sino de incorporar la interoperabilidad en la fase de politica publica, arquitectura, contratacion y diseno de servicio.

| Dimension | Pregunta clave | Ejemplo |
| --- | --- | --- |
| Juridica | Puede intercambiarse este dato para esta finalidad? | Consulta de renta para una ayuda con base legal |
| Organizativa | Quien responde, opera y cambia el servicio? | Acuerdo de nivel de servicio entre organismos |
| Semantica | Significa lo mismo el dato en ambos sistemas? | Codigo comun de municipio, procedimiento o estado |
| Tecnica | Como se conectan los sistemas? | API, certificado, formato, red, version |
| Gobernanza | Como evoluciona sin romper consumidores? | Versionado, deprecacion, pruebas y comunicacion |

### Modo tutor: interoperabilidad

Interoperabilidad significa cooperacion efectiva entre organizaciones y sistemas. Importa porque la Administracion no puede pedir al ciudadano que haga manualmente de mensajero entre organos si los datos ya obran en poder publico y el marco aplicable permite consultarlos. Se confunde con conectividad. En examen se reconoce por referencias a ENI, dimensiones juridica, organizativa, semantica y tecnica, formatos, modelos de datos y servicios comunes.

> Nota de test: conectar dos sistemas no garantiza interoperabilidad. La conectividad es tecnica; la interoperabilidad incluye significado, reglas, organizacion y marco juridico.

## 13. Plataformas comunes e integracion administrativa

La Administracion digital se apoya en servicios comunes. Aunque el detalle puede variar por Administracion y por periodo, el enfoque es estable: evitar que cada organismo cree su propia solucion para identidad, firma, registro, notificacion, archivo, intermediacion de datos o directorios. Las plataformas comunes reducen duplicidades, homogeneizan garantias y facilitan interoperabilidad.

La intermediacion de datos es un ejemplo claro. Su finalidad es que una Administracion pueda consultar datos que ya obran en poder de otra, dentro de las condiciones aplicables, evitando exigir al interesado documentos que la Administracion puede obtener. Desde el punto de vista de arquitectura, esto requiere servicios de consulta, identificacion de cedente y cesionario, finalidad, trazabilidad, seguridad, catalogo de servicios, control de acceso y evidencias.

La red de comunicaciones administrativa, los servicios de firma, los directorios comunes, los sistemas de registro y las plataformas de notificacion pueden verse como capacidades transversales. En una arquitectura de microservicios, no deberian replicarse dentro de cada dominio. Lo correcto es integrarse con ellas mediante contratos estables.

La reutilizacion de aplicaciones y soluciones tambien es relevante. Antes de desarrollar una aplicacion nueva, debe valorarse si existe una solucion reutilizable que cubra total o parcialmente la necesidad, siempre que cumpla requisitos funcionales, tecnologicos, de seguridad e interoperabilidad. En examen, esto permite enlazar arquitectura con eficiencia publica y racionalizacion del gasto.

El problema practico es que las plataformas comunes tambien pueden crear dependencias fuertes. Si un servicio critico cae, muchos procedimientos se ven afectados. Por eso la arquitectura debe prever degradacion, colas, reintentos, mensajes claros al usuario, ventanas de mantenimiento, monitorizacion y planes de continuidad.

## 14. Operacion: despliegue, observabilidad y continuidad

Los microservicios desplazan parte de la complejidad desde el codigo hacia la operacion. Desplegar veinte servicios no es simplemente ejecutar veinte aplicaciones. Hay que gestionar configuracion, secretos, redes, certificados, versiones, dependencias, migraciones, logs, metricas, trazas, alarmas y recuperacion.

La integracion continua y el despliegue continuo ayudan a reducir riesgo si se aplican con controles. En Administracion no significan desplegar sin gobierno, sino automatizar compilacion, pruebas, analisis de seguridad, empaquetado, promocion entre entornos, aprobaciones cuando proceda y rollback. La automatizacion aporta repetibilidad y evidencia.

La observabilidad debe incluir logs estructurados, metricas de negocio y tecnicas, trazas distribuidas y eventos de auditoria. Las metricas tecnicas indican latencia, errores, CPU, memoria o saturacion de colas. Las metricas de servicio publico indican solicitudes registradas, notificaciones practicadas, consultas fallidas, expedientes pendientes o tiempos de tramitacion. Ambas miradas son necesarias.

Los SLO, u objetivos de nivel de servicio, traducen expectativas en medidas. Por ejemplo, porcentaje de respuestas bajo cierto tiempo, disponibilidad mensual, tiempo maximo de procesamiento de una cola o tasa de errores aceptable. En servicios publicos digitales, los SLO deben alinearse con criticidad del procedimiento, impacto ciudadano, obligaciones normativas y recursos disponibles.

La continuidad exige pensar en copias de seguridad, recuperacion ante desastre, alta disponibilidad, redundancia, pruebas de restauracion y procedimientos manuales o alternativos. Un sistema distribuido no es automaticamente mas disponible. Puede serlo si se disena para tolerar fallos parciales.

| Ambito operativo | Pregunta de control | Evidencia esperable |
| --- | --- | --- |
| Despliegue | Puede revertirse un cambio defectuoso? | Versiones, rollback, pruebas |
| Configuracion | Estan separados secretos y parametros? | Gestor de secretos, trazabilidad |
| Observabilidad | Puede seguirse una peticion completa? | Correlation id, trazas, logs |
| Continuidad | Que ocurre si falla un servicio comun? | Degradacion, cola, plan alternativo |
| Capacidad | Se conocen picos y limites? | Pruebas de carga, cuotas |
| Incidentes | Quien actua y como se comunica? | Runbooks, guardias, postmortems |

> Nota de test: en microservicios, "cada equipo despliega cuando quiere" solo es correcto si existen contratos, pruebas, compatibilidad y observabilidad. Sin esos elementos, el despliegue independiente aumenta el riesgo.

## 15. Gobierno de APIs y ciclo de vida

El gobierno de APIs es el conjunto de reglas que permite que las APIs sean fiables, reutilizables y sostenibles. Debe responder a preguntas concretas: quien puede publicar una API, que estandares de diseno se aplican, como se versiona, como se aprueba un cambio incompatible, como se autentica a consumidores, como se mide uso, como se retira una version y como se resuelven incidencias.

El ciclo de vida empieza antes de programar. Primero se identifica la capacidad administrativa. Despues se define el consumidor, la finalidad, los datos, las operaciones, la base juridica, la seguridad, la semantica y el nivel de servicio. Solo entonces tiene sentido implementar. Publicar endpoints sin ese analisis genera deuda contractual.

El versionado debe ser comprensible. Los cambios compatibles anaden campos opcionales, nuevos endpoints o nuevos valores documentados sin romper consumidores. Los cambios incompatibles eliminan campos, cambian significados, modifican reglas obligatorias o alteran codigos de error. En servicios publicos, un cambio incompatible debe planificarse con periodo de convivencia y comunicacion.

La deprecacion no es abandono silencioso. Debe indicar version afectada, motivo, alternativa, fechas, soporte y canal de contacto. En Administracion, consumidores externos pueden necesitar tiempo por contratacion, pruebas, ciclos presupuestarios o dependencia de terceros.

La catalogacion de APIs es una buena practica. Un portal interno o externo de APIs debe indicar descripcion, responsable, estado, version, entorno de pruebas, condiciones de uso, documentacion, esquema, seguridad y soporte. Esto evita que las integraciones dependan de conocimiento informal.

### Modo tutor: gobierno de APIs

Gobierno de APIs significa reglas para disenar, publicar, securizar, versionar, medir y retirar APIs. Importa porque una Administracion puede tener decenas o cientos de integraciones con impacto juridico. Se confunde con una herramienta de gateway. En examen se reconoce por ciclo de vida, catalogo, versionado, seguridad, SLA/SLO, responsable y deprecacion.

> Nota de test: una herramienta no sustituye el gobierno. Un gateway puede aplicar politicas, pero no decide por si mismo la semantica, la base juridica ni el ciclo de vida de una API.

## 16. Contratacion publica y arquitectura

La arquitectura tambien se decide en pliegos, contratos y modelos de servicio. Si un contrato exige una solucion cerrada, sin APIs documentadas, sin exportacion de datos, sin pruebas de interoperabilidad y sin transferencia de conocimiento, crea dependencia de proveedor. Si exige estandares abiertos, documentacion, portabilidad, seguridad, observabilidad y entrega de artefactos, facilita sostenibilidad.

Los pliegos deben evitar describir soluciones tecnicas de forma tan rigida que bloqueen la evolucion, pero deben fijar requisitos no funcionales claros: cumplimiento de ENS, interoperabilidad, accesibilidad, proteccion de datos, rendimiento, continuidad, documentacion, pruebas, calidad del codigo cuando proceda, trazabilidad, operacion y reversibilidad.

La reversibilidad es especialmente importante. Una Administracion debe poder cambiar de proveedor o evolucionar la solucion sin perder datos, documentacion, conocimiento operativo ni capacidad de mantenimiento. Las APIs y los modelos de datos son parte de esa reversibilidad.

En microservicios, los contratos deben aclarar responsabilidades. Quien opera cada servicio? Quien responde ante incidentes? Quien actualiza dependencias vulnerables? Quien mantiene esquemas de eventos? Quien gestiona certificados? Sin respuestas, la arquitectura distribuida puede convertirse en fragmentacion contractual.

## 17. Ejemplos aplicados a Administracion publica

### Ejemplo 1: expediente de subvenciones

Un ciudadano presenta una solicitud de subvencion. El portal ciudadano llama al servicio de solicitudes, que registra la entrada y genera un identificador. El servicio de expedientes crea el expediente administrativo. El servicio de intermediacion consulta datos necesarios si existe habilitacion. El servicio de documentos conserva anexos. El servicio de notificaciones comunica requerimientos. El servicio de pagos interviene al final si se concede la ayuda.

Una arquitectura monolitica podria resolverlo todo en una aplicacion unica. Una arquitectura distribuida separa capacidades. La clave es no perder la unidad del procedimiento. El ciudadano debe ver un estado coherente, el empleado publico debe tener trazabilidad y el expediente debe conservar evidencias.

### Ejemplo 2: consulta interadministrativa de datos

Un ayuntamiento necesita verificar un dato que obra en poder de otra Administracion. No deberia pedir al ciudadano que aporte un certificado si el marco aplicable permite consulta. La integracion requiere una API o servicio de intermediacion, identificacion del organismo solicitante, finalidad, autorizacion, trazabilidad y respuesta normalizada. Tecnologicamente puede ser una llamada sincrona; juridicamente es un acto de intercambio de datos con condiciones.

### Ejemplo 3: modernizacion de un registro antiguo

Un registro historico funciona con una aplicacion monolitica dificil de modificar. Reescribirlo entero es arriesgado. Se puede aplicar el patron strangler: publicar APIs alrededor del sistema, crear nuevos modulos para funcionalidades nuevas, sincronizar eventos y retirar progresivamente partes antiguas. Durante la convivencia, la capa anticorrupcion traduce modelos y evita que el nuevo dominio copie defectos del legado.

### Ejemplo 4: portal con picos de demanda

Un proceso selectivo abre plazo y recibe miles de solicitudes en pocas horas. La arquitectura debe absorber picos. Puede separar frontal, gestion de solicitudes, documentos, pagos y notificaciones. Puede usar colas para procesar tareas no inmediatas. Debe informar al usuario de forma clara, generar justificante y evitar duplicados mediante idempotencia.

## 18. Supuesto practico guiado

### Situacion

Una comunidad autonoma quiere crear una plataforma comun para tramitar ayudas sectoriales. Existen aplicaciones antiguas por consejeria, cada una con su base de datos. Se quiere permitir presentacion electronica, consulta de datos a otras Administraciones, firma, notificacion, seguimiento del expediente y explotacion estadistica. El presupuesto exige evolucion progresiva, no sustitucion completa en un ano.

### Pistas relevantes

Hay dominios diferenciados: identidad, solicitudes, expedientes, documentos, baremos, notificaciones, consultas de datos y estadisticas. Hay sistemas legados. Hay necesidad de interoperabilidad. Hay datos personales. Hay procesos largos. Hay picos por convocatorias. Hay riesgo de duplicidad y de dependencia de proveedor.

### Preguntas

1. Que arquitectura general propondrias?
2. Que integraciones deben ser sincronas y cuales asincronas?
3. Como protegerias datos personales y trazabilidad?
4. Como evitarias duplicar solicitudes o perder eventos?
5. Como modernizarias los sistemas legados?

### Resolucion orientativa

La solucion razonable es una arquitectura hibrida. No se debe imponer microservicios a todo, sino crear una plataforma modular con servicios comunes bien definidos y APIs gobernadas. Las aplicaciones antiguas se conectan mediante adaptadores y capas anticorrupcion. Las nuevas capacidades se desarrollan como servicios separados cuando exista dominio claro y capacidad operativa.

La presentacion de solicitud y la emision de justificante requieren confirmacion inmediata. Deben ser sincronas en lo esencial. La generacion de tareas internas, validaciones posteriores, avisos, analitica y sincronizaciones pueden ser asincronas. Las consultas de datos pueden ser sincronas si el procedimiento necesita respuesta inmediata, pero deben tener timeout, reintento y alternativa cuando el servicio externo no este disponible.

La proteccion de datos exige minimizacion, autorizacion por finalidad, registro de accesos, cifrado, segregacion de permisos y control de eventos. No todos los consumidores deben recibir todos los datos. Los eventos deben contener referencias o datos minimos cuando sea posible.

La idempotencia se resuelve con identificadores de solicitud, claves de operacion y control de duplicados. La publicacion fiable de eventos puede usar outbox transaccional. Los mensajes fallidos deben ir a una cola de errores con procedimiento de reproceso.

Los legados se modernizan con patron strangler. Primero se identifican funcionalidades criticas, se publican APIs estables, se crean adaptadores, se migra funcionalidad nueva a servicios modernos y se retiran partes antiguas con evidencias. La arquitectura debe incluir plan de reversibilidad, pruebas contractuales y observabilidad.

### Errores en la resolucion

Un error seria proponer "microservicios para todo" sin hablar de datos, operacion ni observabilidad. Otro seria mantener integraciones punto a punto entre todas las consejerias. Otro seria centralizar toda la logica en un bus corporativo hasta crear un nuevo monolito. Tambien seria incorrecto ignorar base juridica, ENS, proteccion de datos o trazabilidad.

### Mini comprobacion

La respuesta es buena si diferencia dominios, justifica sincrono/asincrono, contempla seguridad, explica modernizacion gradual y vincula la arquitectura con obligaciones publicas.

## 19. Errores frecuentes de examen

1. Confundir microservicios con cualquier servicio web. Un endpoint no es necesariamente un microservicio.
2. Afirmar que los microservicios siempre son mejores que los monolitos. Depende del dominio, equipo, operacion y requisitos.
3. Reducir interoperabilidad a protocolo tecnico. La interoperabilidad incluye dimensiones juridica, organizativa, semantica y tecnica.
4. Olvidar el ENS al hablar de APIs publicas. Seguridad, riesgos y medidas son parte de la arquitectura.
5. Suponer que una API REST esta bien disenada solo por usar JSON y HTTP.
6. No distinguir autenticacion de autorizacion. Saber quien es alguien no implica que pueda acceder a cualquier objeto.
7. Olvidar idempotencia en reintentos. En integraciones reales, los mensajes pueden duplicarse.
8. Tratar eventos como mensajes sin significado administrativo. Un evento debe representar un hecho claro.
9. Compartir base de datos entre microservicios y seguir llamandolos independientes.
10. No prever versionado y deprecacion de APIs.
11. Ignorar datos personales en logs, trazas y eventos.
12. Presentar el gateway como solucion universal. Es una pieza, no toda la arquitectura.

## 20. Repaso final

Las siete ideas que conviene retener son estas:

1. Una arquitectura distribuida reparte componentes que cooperan por red; aporta autonomia y escalabilidad, pero aumenta complejidad.
2. Los microservicios solo tienen sentido con limites de dominio, datos controlados, despliegue independiente, contratos y observabilidad.
3. Las APIs son contratos de responsabilidad, no simples URLs.
4. La integracion puede ser sincrona o asincrona; la eleccion depende de latencia, consistencia, resiliencia y naturaleza del proceso.
5. En Administracion publica, interoperabilidad significa dimensiones juridica, organizativa, semantica y tecnica.
6. Seguridad y proteccion de datos deben estar en el diseno: ENS, autorizacion, trazabilidad, minimizacion y control de accesos.
7. La modernizacion debe ser gradual, gobernada y verificable; no basta con reescribir sistemas ni con rodearlos de tecnologia nueva.

Definiciones rapidas:

- API: contrato documentado para consumir una capacidad de un sistema.
- Microservicio: servicio autonomo orientado a una capacidad de dominio, desplegable de forma independiente.
- Interoperabilidad: capacidad de organizaciones y sistemas para cooperar con significado y reglas comunes.
- Idempotencia: propiedad que permite repetir una operacion sin duplicar efectos.
- Consistencia eventual: convergencia diferida de estados en un sistema distribuido.
- Outbox: patron para registrar y publicar eventos de forma fiable desde una transaccion local.
- Gateway: punto de entrada comun para APIs y politicas transversales.
- Saga: coordinacion de pasos locales con compensaciones en procesos distribuidos.

## 21. Plan de visuales propuesto

Visual 1: mapa de capas de una arquitectura publica distribuida. Debe mostrar canales, API gateway, servicios de dominio, servicios comunes, sistemas legados, broker de eventos, datos y observabilidad. Debe ser SVG determinista con scroll horizontal en movil.

Visual 2: secuencia de presentacion de solicitud. Debe distinguir llamada sincrona para registro y justificante, y eventos asincronos para validacion, notificacion y analitica.

Visual 3: matriz de interoperabilidad. Cuatro columnas: juridica, organizativa, semantica y tecnica. Cada columna con pregunta clave, artefacto y riesgo.

Visual 4: comparativa monolito modular, SOA y microservicios. Debe evitar caricaturas y mostrar criterios de decision.

Visual 5: flujo de modernizacion strangler. Legado, adaptador, nueva API, nuevo servicio y retirada gradual.

Visual 6: seguridad de API en capas. Consumidor, identidad, gateway, autorizacion, servicio, datos, logs y auditoria.

## 22. Fuentes editoriales consultadas

Estas fuentes se citan sin URL visible para que el integrador las traslade al fichero de fuentes del tema con el formato editorial que corresponda:

- Ley 39/2015, de 1 de octubre, del Procedimiento Administrativo Comun de las Administraciones Publicas. BOE, texto consolidado consultado el 20 de mayo de 2026.
- Ley 40/2015, de 1 de octubre, de Regimen Juridico del Sector Publico. BOE, texto consolidado consultado el 20 de mayo de 2026.
- Real Decreto 203/2021, de 30 de marzo, por el que se aprueba el Reglamento de actuacion y funcionamiento del sector publico por medios electronicos. BOE, consultado el 20 de mayo de 2026.
- Real Decreto 4/2010, de 8 de enero, por el que se regula el Esquema Nacional de Interoperabilidad. BOE, texto consolidado consultado el 20 de mayo de 2026.
- Real Decreto 311/2022, de 3 de mayo, por el que se regula el Esquema Nacional de Seguridad. BOE, texto consolidado consultado el 20 de mayo de 2026.
- Reglamento (UE) 2024/903 del Parlamento Europeo y del Consejo, Reglamento sobre la Europa Interoperable. DOUE/BOE, consultado el 20 de mayo de 2026.
- Reglamento (UE) 2016/679, Reglamento General de Proteccion de Datos. DOUE/BOE, consultado el 20 de mayo de 2026.
- Reglamento (UE) 910/2014 y Reglamento (UE) 2024/1183 sobre identificacion electronica, servicios de confianza y marco europeo de identidad digital. DOUE/BOE, consultados el 20 de mayo de 2026.
- Esquema Nacional de Interoperabilidad y documentacion de apoyo del Portal de Administracion Electronica, especialmente intermediacion de datos y catalogo de servicios.
- IETF RFC 9110, HTTP Semantics.
- IETF RFC 6749, OAuth 2.0 Authorization Framework.
- IETF RFC 7519, JSON Web Token.
- IETF RFC 8446, TLS 1.3.
- OpenAPI Specification 3.1.1, OpenAPI Initiative.
- OWASP API Security Top 10 2023, OWASP Foundation.
- NIST Special Publication 800-204, Security Strategies for Microservices-based Application Systems.

## 23. Encaje recomendado en el tema final

Este material puede alimentar los bloques de definiciones, desarrollo teorico, interoperabilidad, APIs, microservicios, seguridad, operacion, ejemplos, supuestos y repaso. Para el `tema_a1.md` final conviene ampliarlo con:

- marco normativo desarrollado por otro bloque especializado;
- fuentes oficiales en `fuentes.md` con criterio editorial;
- banco de preguntas i18n externo;
- visuales locales en SVG;
- HTML sincronizado con modo primera lectura y notas ocultables;
- recuento final entre 20.250 y 22.500 palabras.
