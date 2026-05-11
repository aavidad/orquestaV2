# Analisis forense del nucleo nuevo

Fecha: 2026-05-04
Estado: `canonico local`

Este documento fija como afrontar el nucleo nuevo de Orquesta. La conclusion no es copiar v1 ni seguir inflando v2. La conclusion es extraer los patrones que demostraron valor y reconstruir el centro como workflow durable pequeno, testeable y sustituible.

## Veredicto

El nucleo nuevo debe ser una maquina de workflow durable propia en Go para v0.

No debe ser:

- una sesion larga de Codex;
- un director monolitico con prompts gigantes;
- una coleccion de rutas CLI/API que mutan DB;
- un control plane de decenas de miles de lineas;
- un scheduler que conoce proveedores, HOME, OAuth, tmux, SQLite o Docker.

La parte que debe quedar dentro del nucleo es la decision de dominio reproducible:

- en que fase esta una orquestacion;
- que comando se acepta;
- que eventos se producen;
- que efectos externos se piden por outbox;
- que invariantes bloquean el avance;
- cuando hay que preguntar al director.

## Por que v1 y v2 no bastan como base directa

V1 tenia ideas buenas, pero se volvio fragil porque muchas quedaron mezcladas con infraestructura:

- DB usada como fuente de verdad y como motor de reglas;
- runtime, mailbox, handles, cuotas y control plane dentro de zonas demasiado acopladas;
- CLI/API/web con accesos directos o rutas con negocio;
- ficheros gigantes dificiles de auditar;
- bucles de reparacion que arreglaban sintomas sin aislar invariantes;
- mucho contexto historico cargado en sesiones de agentes.

V2 corrigio la direccion:

- mini-proyectos;
- contratos;
- perifericos hexagonales;
- contexto local por modulo;
- AppSpec, factory, capacity, runtime, persistence, observability, governance y deploy separados.

Pero `modulos/orquesta-core/` no debe crecer como scheduler central. Hoy es util como corte de compatibilidad y contratos iniciales, no como nucleo durable completo.

## Evidencia reutilizable

### Reutilizar como patron

De v1:

- fases del ciclo de vida;
- propuestas y votaciones para decisiones irreversibles;
- microprogramacion por funcion con write-set y tests;
- runtime orders, mailbox, handles y checkpoints como primitivas durables;
- supervisor residente, reviewer reservado y workers acotados;
- escalation ladder: worker actual, repair-helper, relevo, premium;
- multi-HOME y perfiles de agente separados de identidad logica;
- cuotas por ventana, bloqueo preventivo y handoff por presupuesto;
- state delta, artifacts y replay parcial inspirados en ADK;
- dispatch, inbox, ACK, worktree y resume como disciplina operativa inspirada en oh-my-codex.

De v2:

- `SolicitarNuevaApp v0` y `AppSpecV0` como entrada de producto;
- `RegistrarProyectoDesdeAppSpec v0` como corte puro provisional;
- `FunctionContractV0` como unidad de trabajo pequena;
- `CapacityDecisionV0` como frontera para modelo, reasoning, cuota y escalado;
- `RuntimeLaunchRequestV0` como frontera para arrancar agentes;
- `PersistenceRepositoryV0` como frontera de estado durable;
- `OrquestaEventV0` como frontera de observabilidad;
- `GovernanceCatalogV0` como fuente de reglas efectivas;
- `DeploymentPlanV0` como plan de entorno por conector.

### Reutilizar como codigo solo si pasa contrato

- funciones puras de `orquesta-core` que generen IDs estables o mapeen DTOs sin adaptadores;
- DTOs ya promovidos a contratos globales;
- fixtures de contratos;
- tests de caracterizacion del flujo probado.

Nada se copia por estar en v1 o v2. Cada extraccion debe ser pequena y tener prueba.

## Cuarentena

No se copian al nucleo nuevo:

- control plane gigante;
- `db/` como motor de negocio;
- SQL o schema como contrato de dominio;
- rutas CLI/API/MCP con negocio embebido;
- scripts manuales como flujo principal;
- nombres reales de agentes o cuentas;
- transcripts o audit logs masivos;
- valores historicos de configuracion sin decision nueva;
- algoritmos de polling sin evento, lease, timeout e idempotencia.

## Arquitectura propuesta

El nucleo queda dividido en piezas puras:

```text
State:
  OrchestrationRunV0

Commands:
  StartRun
  OpenPhase
  RequestBrainstorm
  RequestVote
  AcceptDecision
  CreateMicrotask
  PublishFunctionContract
  RequestCapacity
  RequestAgent
  RegisterDelivery
  RequestReview
  AcceptReview
  CloseTask
  ClosePhase
  BlockRun
  AskDirector

Events:
  RunStarted
  PhaseOpened
  BrainstormRequested
  VoteRequested
  ArchitectureDecisionAccepted
  MicrotaskCreated
  FunctionContractPublished
  CapacityRequested
  CapacityDecided
  AgentRequested
  AgentStarted
  AgentFailed
  AgentStopRequested
  AgentWorkAssessed
  DeliveryRegistered
  ReviewRequested
  ReviewAccepted
  TaskClosed
  FinalValidationRegistered
  RunClosed
  PhaseClosed
  DirectorQuestionRaised
  DirectorQuestionAnswered
  RunBlocked

Outbox:
  PersistRunEvents
  PublishOrquestaEvent
  RequestCapacityDecision
  LaunchRuntimeAgent
  RequestDeployPlan
  SendDirectorQuestion
```

El estado se reconstruye por replay. Si falla persistence, runtime o capacity, el nucleo no inventa progreso: conserva eventos y outbox para reintento. Desde NCW-039, `CapacityDecided` actua como barrera durable entre la solicitud de capacidad y el lanzamiento logico del agente, sin filtrar proveedor, modelo concreto ni cuotas reales.

## Patron contrastado

La ventaja de workflow durable frente a un agente director monolitico no es teorica:

- motores como Temporal o Durable Task separan historia durable, actividad externa y reintentos;
- controladores declarativos como Kubernetes reconcilian estado deseado y estado observado sin depender de una sesion viva;
- pipelines CI/CD como GitHub Actions, GitLab CI o Argo Workflows modelan pasos, eventos, logs y reintentos con trazabilidad;
- event sourcing y outbox transaccional se usan para reconstruir estado y reintentar efectos sin duplicar acciones.

Orquesta no adopta todavia uno de esos motores porque el primer objetivo es fijar contratos y reducir riesgo. Si mas adelante hacen falta timers, colas, concurrencia distribuida o retries avanzados, el motor externo se conectara como adaptador. El dominio no debe quedar encerrado en esa herramienta.

## Fases v0

Las fases candidatas son:

- `descubrimiento`;
- `brainstorming_arquitectura`;
- `votacion_y_decision`;
- `planificacion_microtareas`;
- `programacion`;
- `documentacion`;
- `integracion`;
- `revision`;
- `validacion_final`;
- `cierre`.

Cada fase debe declarar:

- criterios de entrada;
- criterios de salida;
- evidencia minima;
- politica de capacidad recomendada;
- eventos que puede emitir;
- condiciones de bloqueo.

## Politica de capacidad

El nucleo no elige proveedor ni modelo. El nucleo pide una decision a `orquesta-capacity`.

Reglas de dominio que si puede expresar:

- brainstorming, arquitectura, seguridad y decisiones irreversibles requieren capacidad alta o maxima recomendada;
- implementacion pequena y aislada puede usar capacidad media o local;
- mala calidad, fallo de review, conflicto entre agentes o cambio irreversible dispara escalado;
- cuota, HOME, OAuth, vLLM, Ollama o Codex son detalles de capacity/runtime.

## Votaciones

El brainstorming no se cierra por una opinion unica. Para arquitectura y cambios de contrato:

- se abre propuesta;
- se piden votos de agentes no autores cuando aplique;
- el resultado se convierte en evento;
- las opiniones tardias pueden conservarse como trazabilidad, pero no reabren consenso cerrado sin nueva decision.

El objetivo no es debate infinito. El voto existe para evitar decisiones invisibles e irreproducibles.

## Contexto por equipos

Este modulo debe permitir que varios grupos trabajen sin contexto gigante:

- cada microtarea tiene write-set pequeno;
- cada agente lee solo docs locales;
- si necesita otro modulo, eleva `CONSULTA AL DIRECTOR`;
- las reglas comunes se repiten de forma compacta en cada modulo;
- el estado durable sustituye a depender de una sesion larga.

## Criterios para empezar a programar

Antes de conectar adaptadores reales hay que cerrar:

- `OrchestrationRunV0`;
- fases v0;
- comandos/eventos internos;
- outbox;
- replay;
- idempotencia;
- errores publicos;
- primer puente desde AppSpec/ProyectoPlanBorrador.

La primera implementacion permitida es pura y local. No lanza agentes reales.
