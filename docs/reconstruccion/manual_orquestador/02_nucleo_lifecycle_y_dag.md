# 02. Núcleo, ciclo de vida y grafo de trabajo

> **Responsabilidad:** definir cómo construir el núcleo autoritativo de una
> aplicación de orquestación y cómo convertir una intención confirmada en un
> grafo acíclico dirigido de trabajo (DAG) ejecutable sin crear ciclos de vida
> paralelos.
>
> **Alcance:** `Goal`, `WorkItem`, `PhaseInstance`, `Execution`, `Action`,
> planificación, conjunto listo, Director, escritor y planificador únicos,
> comandos, consultas, fallos, pruebas y orden de construcción.
>
> **No acredita:** este documento no acredita código, una versión, el producto
> completo, el entorno elástico de agentes ni Firecracker. El estado válido
> vive en la base versionada `HEAD:product/roadmap.json`, que se consulta con
> `git show HEAD:product/roadmap.json`, el
> [registro de capacidades](../../../product/capabilities.json) y las
> [evidencias](../../../product/evidence/).

## Al terminar este capítulo, el lector sabrá…

- modelar `Goal`, `WorkItem`, `PhaseInstance`, `Execution` y `Action` sin
  duplicar el ciclo de vida;
- derivar el conjunto listo a partir del DAG completo;
- separar las propuestas del Director de las decisiones del motor;
- diseñar pruebas que detecten escritores, planificadores o colas paralelos.

El vocabulario compartido se define en el
[glosario técnico común](01_mision_limites_y_vocabulario.md#glosario-técnico-común).

## Índice del capítulo

- [1. Autoridades y forma de leer este capítulo](#1-autoridades-y-forma-de-leer-este-capítulo)
- [2. Resultado mínimo correcto](#2-resultado-mínimo-correcto)
- [3. Modelo único](#3-modelo-único)
- [4. Un escritor, un planificador y una fuente de estado](#4-un-escritor-un-planificador-y-una-fuente-de-estado)
- [5. Construcción y validación del DAG](#5-construcción-y-validación-del-dag)
- [6. Director inteligente y motor autoritativo](#6-director-inteligente-y-motor-autoritativo)
- [7. Comandos y consultas](#7-comandos-y-consultas)
- [8. Tratamiento de fallos](#8-tratamiento-de-fallos)
- [9. Secuencia de construcción desde cero](#9-secuencia-de-construcción-desde-cero)
- [10. Pruebas obligatorias](#10-pruebas-obligatorias)
- [11. Antipatrones que invalidan el diseño](#11-antipatrones-que-invalidan-el-diseño)
- [12. Lista de salida del capítulo](#12-lista-de-salida-del-capítulo)

## 1. Autoridades y forma de leer este capítulo

La arquitectura obligatoria se toma de
[AGENTS.md](../../../AGENTS.md), y el orden causal de
[ruta_total_100.md](../ruta_total_100.md). El código enlazado muestra la
composición actual, pero no sustituye esos contratos.

Cada afirmación de este capítulo pertenece a una de estas clases:

- **Invariante:** regla que cualquier implementación debe conservar.
- **Producto acreditado:** conducta cuyo ID figura como `accredited` y tiene
  evidencia de una revisión sellada. La evidencia acredita aquella revisión,
  no cualquier cambio posterior del árbol.
- **Diseño objetivo:** forma prescrita para ampliar el producto sin abrir otro
  núcleo.
- **Deuda:** conducta aún no acreditada; no puede anunciarse como disponible.

Los contratos V05, V06, V12, V14 y V15 respaldan partes nucleares ya
acreditadas. La gestión física elástica de muchos agentes y una micromáquina
virtual (`microVM`) por agente siguen siendo deuda separada. No se deducen de
que el grafo pueda representar mucha demanda.

## 2. Resultado mínimo correcto

Un orquestador correcto contiene una sola cadena de autoridad:

```text
intención confirmada
        |
      AppSpec
        |
       Goal                         única autoridad de ciclo de vida
        |
  WorkItems en DAG                  trabajo causal
        |
  Executions y Actions              intentos operativos
        |
artefactos, atestaciones y recibos     evidencia
```

`AppSpec` es la especificación confirmada de la aplicación. Los demás
identificadores se definen en la sección siguiente.

El Director puede razonar y proponer. El motor de aplicación autentica,
autoriza, valida revisiones, causalidad, presupuesto, riesgo y evidencia. Solo
entonces pide al repositorio una escritura atómica.

Las interfaces HTTP, MCP, línea de comandos (`CLI`) y web invocan los mismos
casos de uso. Los adaptadores de agentes, bases de datos o máquinas virtuales
no deciden estados del `Goal`.

## 3. Modelo único

| Tipo | Responsabilidad | Es autoridad de cierre |
|---|---|---|
| `Goal` | identidad, alcance, generación y ciclo de vida global | Sí |
| `WorkItem` | unidad causal del DAG y frontera de trabajo | Solo dentro del `Goal` |
| `PhaseInstance` | metadatos inmutables de una instancia de fase | No |
| `Execution` | intento concreto de ejecutar un `WorkItem` | No |
| `Action` | orden durable para efectuar el siguiente paso | No |
| evento | auditoría de una transición aceptada | No |
| recibo | prueba inmutable de consumo o efecto | No por sí solo |
| proyección | vista para operador o interfaz | No |

La referencia actual está en
[`internal/goal`](../../../internal/goal/),
[`internal/application/state.go`](../../../internal/application/state.go) y
[`internal/application/orchestrator.go`](../../../internal/application/orchestrator.go).

### 3.1 `Goal`

`Goal` es el agregado inmutable que contiene el `AppSpec`, actor, proyecto,
revisión, generación de plan, fases, unidades de trabajo y hechos de entrega
que afectan al cierre.

Estados, con el identificador persistido entre paréntesis:

```text
pendiente (pending) -> en curso (running) -> logrado (succeeded)
                                           \-> fallido (failed)
pendiente/en curso --------------------------> cancelado (canceled)
```

Reglas:

- nace pendiente (`pending`) con revisión 1;
- solo arranca con un plan válido y al menos un `WorkItem`;
- toda transición exige la revisión esperada;
- una propuesta de plan incrementa la generación, pero no cierra ni reabre;
- el plan crece de forma monótona: no reescribe el prefijo ya aceptado;
- el resultado de cierre se deriva de los resultados lógicos de todas las
  unidades, no del texto de un agente;
- una cancelación solicitada se completa mediante la transición gobernada
  correspondiente;
- un `Goal` terminal no se reabre; un cambio posterior crea un sucesor causal
  con un `AppSpec` nuevo.

El dominio actual concentra estas reglas en
[`internal/goal/goal.go`](../../../internal/goal/goal.go) y
[`internal/goal/control.go`](../../../internal/goal/control.go).

### 3.2 `WorkItem`

`WorkItem` describe una unidad pequeña y verificable. Conserva como mínimo:

- referencia opaca y vínculo exacto a actor, proyecto y `Goal`;
- objetivo;
- fase y rol;
- dependencias;
- padre causal opcional;
- política explícita de entrega al padre;
- conjunto de escritura;
- herramientas, habilidades y capacidades requeridas;
- pruebas requeridas y contrato de salida;
- demanda de presupuesto, criticidad y esfuerzo;
- revisión, estado e intento enlazado.

Estados, con el identificador persistido entre paréntesis:

```text
pendiente (pending) -> en curso (running) -> logrado (succeeded)
                                           \-> fallido (failed)
                                           \-> interrumpido (interrupted)
                                               -> nueva planificación o reintento causal
pendiente ------------------------------------> omitido (skipped)
pendiente/en curso ----------------------------> cancelado (canceled)
interrumpido ----------------------------------> sustituido (superseded)
                                                -> sucesor de retrabajo
```

`interrupted` no es un éxito ni un cierre terminal ordinario. Conserva el
trabajo útil y deja al Director proponer una reparación causal. `superseded`
solo es válido si existe un sucesor de retrabajo enlazado.

Una unidad exitosa solo puede completar cuando aporta los artefactos y
atestaciones exigidos y, si tiene hijos contractuales, sus entregas están
resueltas. Fallar una dependencia hace que los descendientes incompatibles se
omitan causalmente; no se ejecutan fingiendo independencia.

La implementación de referencia está en
[`internal/goal/work_item.go`](../../../internal/goal/work_item.go) y
[`internal/goal/replan.go`](../../../internal/goal/replan.go).

### 3.3 `PhaseInstance`

Una fase es metadato causal inmutable:

- referencia de instancia;
- clave;
- plantilla;
- referencias de entrada;
- referencias de criterios.

No tiene estado, cola, bucle ni base de datos. Su progreso se calcula a partir
de sus `WorkItems` y evidencias. Si una “fase” necesita pausar o cerrar el
`Goal`, esa regla pertenece al agregado o al motor, no a la fase.

### 3.4 `Execution`

Una ejecución es un intento. Identifica:

- `Goal`, `WorkItem`, generación de plan y generación de `AppSpec`;
- número de intento y ejecución sustituida;
- propósito: trabajo, autoría, revisión o participación en Consejo;
- proveedor, modelo, agente y referencia externa;
- sesión y espacio de trabajo;
- reservas de presupuesto e intención de efecto;
- plazos, observaciones, fallo y recibos.

Estados operativos actuales, con los identificadores reales:

```text
en cola (queued) -> despachando (dispatching) -> en curso (running)
                                               -> esperando registro del cambio (awaiting_commit)
                                               -> esperando atestación (awaiting_attestation)
                                               -> esperando integración (awaiting_integration)
                                               -> logrado | fallido | cancelado | detenido
```

Una ejecución no posee el resultado del `Goal`. Un reintento sustituye la
referencia de intento bajo revisión esperada; no crea otro ciclo de vida.
Revisión primaria, revisión adversarial y Consejo son ejecuciones con
propósitos distintos sobre el mismo sujeto causal.

### 3.5 `Action`

Una acción es una orden durable de la bandeja transaccional. Entre sus clases
actuales están preparar un espacio de trabajo, lanzar, observar o detener un
agente, confirmar pruebas, integrar un cambio, entregar correo interno o
revocar una sesión.

La acción conserva:

- sujeto exacto: `Goal`, `WorkItem`, `Execution` y, cuando aplica, cambio;
- generación de plan y revisión de unidad;
- instante de disponibilidad;
- intención y aprobación de efecto cuando hay efecto externo.

Una acción reclamada no demuestra que el efecto ocurrió. Solo el recibo
inmutable correspondiente puede demostrar consumo o efecto.

## 4. Un escritor, un planificador y una fuente de estado

### 4.1 Escritor único

`internal/application.Orchestrator` es la única autoridad que construye
transiciones. El repositorio:

- comprueba cercas, revisiones y restricciones dentro de su transacción;
- persiste el estado propuesto;
- nunca inventa una transición de negocio.

El dominio valida sus invariantes sin conocer SQL, procesos, proveedores,
rutas ni credenciales. Los transportes traducen entrada y salida, pero no
mutan el agregado directamente.

### 4.2 Planificador único

Existe un protocolo de planificación:

1. leer el `Goal` autoritativo;
2. derivar trabajo ejecutable desde el DAG;
3. crear una `Execution` por unidad aún no programada;
4. crear su primera `Action`;
5. persistir `Goal`, ejecuciones, acciones y eventos en la misma operación;
6. permitir que trabajadores compatibles reclamen acciones con arrendamiento
   y cerca.

No debe haber colas privadas por proveedor, fase, Director o tipo de agente.
Una optimización de paginación o una ventana de consulta tampoco puede
convertirse en un límite de demanda del núcleo.

### 4.3 Fuente activa única

Un despliegue elige un solo `StateRepository`. Réplicas de lectura,
proyecciones, eventos y copias de seguridad no son escritores alternativos.
Cambiar SQLite por PostgreSQL cambia el adaptador y la composición, no el
modelo ni los casos de uso.

## 5. Construcción y validación del DAG

### 5.1 Aristas diferentes

No mezclar estas relaciones:

- **dependencia:** B no puede empezar hasta que A tenga resultado lógico
  satisfactorio;
- **padre/hijo:** expresa linaje y recursión;
- **entrega requerida:** añade una barrera explícita del hijo al padre;
- **conflicto de escritura:** impide simultaneidad aunque no haya dependencia;
- **sucesión de retrabajo:** conserva la causa de una unidad interrumpida o
  sustituida.

`Parent` no activa por sí solo una entrega. `HandoffRequired=true` es una
decisión separada y no es válido sin padre.

### 5.2 Validaciones al admitir un plan

Rechazar antes de persistir:

- referencias vacías, duplicadas o fuera del `Goal`;
- fase inexistente o duplicada;
- dependencia o padre desconocido;
- ciclos por dependencias, linaje o sucesión;
- dos unidades activas con conjuntos de escritura solapados;
- unidad nueva que no nazca pendiente (`pending`) y limpia;
- cambio del prefijo de un plan ya aceptado;
- generación distinta de `actual + 1`;
- sucesor de retrabajo ambiguo o sin fuente válida;
- demanda que no cabe en el sobre máximo declarado;
- entrega requerida sin padre;
- herramienta, capacidad, prueba o contrato de salida inválidos.

### 5.3 Derivación del conjunto listo

La derivación debe ser pura y determinista:

```text
candidatas =
  Goal en curso (running)
  y Goal no pausado ni en cancelación
  y WorkItem pendiente (pending)
  y WorkItem no pausado ni en cancelación
  y todas sus dependencias satisfechas
  y sin conflicto con unidades ya en curso

conjunto listo =
  cohorte maximal determinista, en orden de plan,
  sin conflictos entre candidatas seleccionadas
```

“Maximal” significa que no se puede añadir otra candidata sin conflicto al
resultado construido; no promete el máximo cardinal global. El orden debe ser
estable para que reinicio y repetición produzcan la misma decisión.

La implementación acreditada se puede estudiar en
[`Goal.RunnableWorkItems` y `Goal.ReadyWorkItems`](../../../internal/goal/goal.go)
y en
[`scheduleReady`](../../../internal/application/planning.go).

### 5.4 Demanda lógica frente a capacidad física

Primero se calcula la demanda lógica completa. Después se admite capacidad
física según presupuesto, proveedor y sistema operativo.

Si caben 5 agentes de una cohorte de 70:

- no se recorta el DAG a 5;
- no se inventan 65 fallos;
- no se crea otra generación;
- las ejecuciones pendientes conservan su identidad;
- la capacidad liberada permite continuar.

El límite explícito de hijos por padre protege una explosión recursiva; no es
un techo global de agentes. Acreditar cohortes grandes, arranque físico
paralelo, conservación y cierre individual pertenece al entorno elástico
pendiente, no al contrato V05 por sí solo.

## 6. Director inteligente y motor autoritativo

El Director puede ser un operador, Codex, Hermes u otro agente. Todos usan el
mismo protocolo:

1. reclamar el arrendamiento de Director para un `Goal`;
2. leer la proyección autoritativa;
3. proponer un plan o reparación con revisión y generación esperadas;
4. incluir token y cerca del arrendamiento;
5. dejar que el motor valide identidad, permiso, causa, presupuesto,
   conflictos y evidencia;
6. persistir una decisión compacta o devolver un error estable.

El arrendamiento se puede renovar o transferir. Un Director expirado no
escribe. La toma de control no modifica plan, acciones ni `Goal` por sí sola.
El token no se conserva en el hecho de auditoría; la cerca sí enlaza la
decisión aceptada.

La frontera está descrita en
[`internal/application/director.go`](../../../internal/application/director.go)
y
[`internal/application/director_state.go`](../../../internal/application/director_state.go).

El Director no puede mantener:

- un plan privado más nuevo que el repositorio;
- una cola privada;
- un contador de generación alternativo;
- una base de datos de cierres;
- una decisión de Consejo o revisión que suplante al motor.

## 7. Comandos y consultas

### 7.1 Comando

Un comando solicita un cambio. Debe transportar:

- referencia de petición única;
- principal autenticado y proyecto explícito;
- permiso y sujeto;
- revisión y generación esperadas cuando muta estado existente;
- token y cerca cuando actúa un Director o trabajador;
- datos suficientes para una huella canónica;
- clave de idempotencia del efecto, si aplica.

Orden recomendado:

1. validar forma;
2. ligar principal y proyecto;
3. buscar una repetición exacta antes de generar IDs o efectos;
4. autorizar en aplicación;
5. leer el estado actual;
6. construir la transición con dominio puro;
7. persistir atómicamente;
8. validar la respuesta persistida.

Misma referencia y misma huella devuelven el resultado anterior. Misma
referencia y contenido distinto producen conflicto.

### 7.2 Consulta

Una consulta:

- autoriza lectura;
- restringe por proyecto;
- devuelve proyecciones del estado autoritativo;
- no genera IDs, eventos, arrendamientos ni efectos;
- no repara silenciosamente datos.

Una proyección puede estar atrasada; el comando siempre se cerca contra el
estado autoritativo. Ejemplos actuales:
[`internal/application/queries.go`](../../../internal/application/queries.go).

## 8. Tratamiento de fallos

| Fallo | Conducta correcta |
|---|---|
| revisión antigua | conflicto; releer y volver a decidir |
| dependencia fallida | omitir descendiente afectado con causa |
| agente no disponible | reprogramar sin cerrar el `Goal` |
| intento agotado | interrumpir y permitir reparación causal |
| trabajador caído | recuperar acción tras vencer arrendamiento |
| Director caído | tomar control con cerca superior |
| salida sin artefacto | no marcar éxito |
| prueba fallida | conservar candidato y abrir retrabajo |
| entrega contractual pendiente | impedir cierre del padre |
| escritura conflictiva | no incluir ambas unidades en la cohorte |
| efecto de resultado incierto | reconciliar o poner en cuarentena; no repetir a ciegas |
| cancelación | separar solicitud, parada de intentos y finalización |

Errores de texto o formato recuperables deben volver a revisión o
normalización. Solo causalidad imposible, fuga de secretos, identidad falsa o
efecto no autorizado justifican descartar trabajo útil.

## 9. Secuencia de construcción desde cero

Cada paso termina con contrato ejecutable antes de ampliar el siguiente:

1. **Referencias opacas y reloj.** Implementar actores, proyectos, intención,
   revisiones y tiempos canónicos.
2. **`AppSpec` inmutable.** Confirmación inicial y sucesor causal sin
   reescritura.
3. **`Goal` mínimo.** Estados, revisión esperada y cierre derivado.
4. **`WorkItem`.** Estados, evidencias, dependencias y conjuntos de escritura.
5. **Plan y fases.** Metadatos inmutables, validación de ciclos y crecimiento
   monótono.
6. **Conjunto listo.** Función pura, determinista y conflictiva por
   conjuntos de escritura.
7. **Puerto de estado.** Operaciones gruesas que persistan decisiones
   completas; no un método por tabla.
8. **Ejecuciones y acciones.** Intentos separados del ciclo de vida y bandeja
   durable.
9. **Motor de aplicación.** Único escritor que compone dominio, permisos y
   persistencia.
10. **Planificador.** Crear intentos para toda la cohorte lista sin cola
    privada.
11. **Director.** Arrendamiento, cerca, toma de control y propuestas.
12. **Controles.** Pausa, reanudación, cancelación, parada exacta y retrabajo.
13. **Presupuestos y efectos.** Reservas, aprobaciones y recibos.
14. **Adaptadores reales.** SQLite y agente falso primero; proveedor real solo
    tras pasar la batería contractual común.
15. **Superficies públicas.** Un registro de comandos alimenta todos los
    transportes.
16. **Escala física.** Solo ahora implementar aprovisionamiento elástico,
    aislamiento, conservación y medición.

## 10. Pruebas obligatorias

### 10.1 Dominio y propiedades

- ninguna transición acepta una revisión distinta;
- un plan cíclico o no monótono falla;
- una dependencia insatisfecha nunca arranca;
- conjuntos de escritura solapados nunca están activos a la vez;
- el conjunto listo es estable ante repetición y restauración;
- una fase no puede modificar estado;
- cerrar con una unidad no resuelta o entrega pendiente falla;
- un `Goal` terminal no se reabre;
- una unidad `superseded` tiene sucesor causal.

### 10.2 Aplicación y concurrencia

- dos comandos con la misma revisión: como máximo uno cambia estado;
- repetición exacta devuelve el mismo resultado;
- repetición divergente falla sin efectos;
- dos trabajadores no reclaman la misma acción;
- una cerca antigua no consume ni escribe;
- un Director expirado no aplica un plan;
- una toma de control no altera el DAG;
- un fallo de proveedor no se transforma en cierre terminal indebido;
- pausa o cancelación de B no modifica A, C o D;
- ejecutar con `-race` no descubre doble escritor ni estado compartido.

### 10.3 Extremo a extremo

- creación, DAG, ejecución, artefacto, atestación y cierre por una superficie
  pública;
- reinicio entre cada transición relevante;
- dos revisores independientes trabajan sobre el mismo sujeto sellado;
- parada de una ejecución exacta conserva las demás;
- cero procesos propios tras el apagado;
- una cohorte física solo se acredita con proveedores reales y mediciones.

Contratos de referencia:

- [V05 DAG y fases](../../../acceptance/v05_goal_dag_phases_test.go);
- [V06 estado y bandeja atómicos](../../../acceptance/v06_atomic_state_outbox_test.go);
- [V12 Director](../../../acceptance/v12_director_lease_test.go);
- [V14 controles](../../../acceptance/v14_controls_test.go);
- [V15 presupuestos y efectos](../../../acceptance/v15_budgets_effects_test.go).

## 11. Antipatrones que invalidan el diseño

- otro agregado llamado tarea, ejecución, fase o sesión que pueda cerrar;
- bucle de Director con base de datos o plan privado;
- cola por proveedor o por fase;
- reconstruir el estado desde eventos cuando la instantánea es la autoridad;
- dejar que un adaptador decida reintentos o cierre;
- usar un ACK del proveedor como evidencia de resultado;
- tratar `Parent` como dependencia o entrega implícita;
- ocultar un techo global detrás de un tamaño de lote;
- marcar como fallidas ejecuciones que solo esperan capacidad;
- compartir una ejecución entre dos unidades;
- reusar una revisión o atestación de otro árbol, generación o diferencia;
- crear microservicios internos para cada fase o rol.

## 12. Lista de salida del capítulo

Antes de continuar con persistencia y recuperación debe poder demostrarse:

```text
un solo escritor del ciclo de vida
una sola autoridad planificadora
Goal es la única autoridad de cierre
PhaseInstance no tiene estado
todo WorkItem pertenece a un único Goal
todo Execution es un intento de un WorkItem
toda Action está ligada a generación y sujeto exactos
el conjunto listo deriva solo del DAG y los conjuntos de escritura
se rechazan la revisión y la cerca obsoletas
el Director propone y el motor valida
las consultas no escriben
```

Pasar esta lista acredita únicamente los contratos que se hayan ejecutado
sobre un candidato sellado. La documentación no es evidencia.
