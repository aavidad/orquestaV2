# 04. Gestión elástica de agentes

> **Responsabilidad:** definir cómo construir y operar la gestión elástica de
> agentes de un orquestador.
>
> **Alcance:** demanda completa, capacidad, reservas, arranque paralelo,
> sesiones, supervisión, parada, conservación y recuperación.
>
> **No acredita:** este capítulo es diseño, guía de implementación y criterio
> de aceptación; no acredita por sí mismo ninguna capacidad del producto.

## Al terminar este capítulo, el lector sabrá…

- separar demanda durable, capacidad observada y simultaneidad física;
- diseñar reservas seguras y un reconciliador que no duplique agentes;
- arrancar, supervisar, relevar y detener un agente exacto;
- conservar los entornos terminados y recuperar la gestión tras un reinicio;
- convertir el diseño en pruebas por cohortes y tareas pequeñas.

## Índice del capítulo

- [Vocabulario](#0-vocabulario-mínimo), [resultado](#1-resultado-que-debe-conseguir-el-orquestador) y [estado actual](#2-estado-actual-honesto).
- [Modelo de demanda](#3-modelo-demanda-completa-admisión-limitada), [contratos](#4-contratos-mínimos) y [reconciliador](#5-reconciliador-de-agentes).
- [Cohortes](#6-escalado-por-cohortes), [alta y relevo](#7-alta-reutilización-y-relevo) y [parada y conservación](#8-parada-exacta-y-conservación).
- [Recuperación](#9-reinicio-y-recuperación-sin-duplicados), [telemetría](#10-telemetría-necesaria) y [acreditación](#11-pruebas-y-acreditación).
- [Diagnóstico](#12-diagnóstico-sin-alterar-el-estado), [tareas pequeñas](#13-división-en-tareas-pequeñas), [lecciones](#14-lecciones-que-se-conservan-del-legado) y [fuentes](#15-fuentes-vigentes-de-esta-especificación).

## 0. Vocabulario mínimo

- `Goal`: objetivo durable; es la única autoridad de identidad, generación y
  ciclo de vida.
- `WorkItem`: unidad pequeña de trabajo dentro del grafo de dependencias del
  objetivo; no es otro objetivo ni otra máquina de estados.
- DAG: grafo dirigido sin ciclos que expresa dependencias entre trabajos.
- ciclo de vida (`lifecycle`): secuencia autoritativa de estados de una entidad.
- arrendamiento (`lease`): concesión temporal del uso exclusivo de un recurso
  hasta una fecha.
- cerca (`fence`; la técnica aparece como `fencing`): ordinal que invalida a
  propietarios antiguos; una orden con una cerca menor que la vigente se
  rechaza.
- buzón causal (`mailbox`): intercambio durable y ordenado de mensajes ligado
  a identidades, generación y ejecución exactas.
- recibo (`receipt`): registro estructurado que demuestra un intento, resultado
  o efecto sin sustituir el estado autoritativo.
- huella criptográfica (`digest` o `hash`): resumen verificable de un sujeto
  inmutable.
- conjunto listo: todos los trabajos cuyas dependencias ya se satisfacen.
- reconciliador: servicio que compara lo que debería existir con lo que existe
  realmente y emite las acciones mínimas para hacerlos converger.

## 1. Resultado que debe conseguir el orquestador

Un orquestador no puede depender de una lista fija de procesos ni esperar que
un operador le proporcione manualmente «tres agentes». Debe:

1. derivar toda la demanda ejecutable del DAG del `Goal`;
2. conservar esa demanda aunque no haya capacidad inmediata;
3. observar la capacidad real y su vigencia;
4. reservar capacidad sin sobreasignarla;
5. arrancar, dirigir, supervisar y detener cada agente mediante contratos
   sustituibles;
6. recuperar el control tras reinicios sin duplicar procesos ni efectos;
7. conservar el entorno y sus datos al terminar, hasta que una revisión
   posterior autorice expresamente su retirada.

El número de agentes no es una constante de la arquitectura. Si existen
setenta trabajos independientes y recursos suficientes, los setenta deben
poder progresar. Si existen quinientos y solo caben veinte, los quinientos
siguen siendo demanda durable: veinte trabajan y los demás esperan capacidad
sin perder identidad, orden causal ni número de intento.

Esta gestión pertenece a aplicación. El núcleo de dominio no conoce procesos,
cuentas, terminales, Firecracker, proveedores ni sistemas operativos. Cada
adaptador informa y ejecuta; la aplicación decide.

## 2. Estado actual honesto

La documentación usa tres marcas:

- **ACREDITADO**: existe composición productiva y evidencia de la misma
  revisión.
- **PARCIAL**: existe una pieza útil, pero no satisface el contrato completo.
- **PENDIENTE**: no existe todavía la composición exigida o no hay evidencia
  suficiente.

| Área | Estado | Situación comprobada |
|---|---|---|
| `Goal`, DAG y conjunto completo de trabajos listos | ACREDITADO | `ReadyWorkItems` deriva trabajos ejecutables y `scheduleReady` crea una ejecución y una acción por trabajo no programado. |
| Un único escritor de ciclo de vida | ACREDITADO | La aplicación persiste estado, evento y salida transaccional con revisión esperada. |
| Reclamación por arrendamiento y cerca | ACREDITADO | Las acciones se reclaman con arrendamiento, cerca y número de entrega. |
| Efecto de arranque con intención, aprobación, intento y recibo | ACREDITADO | El lanzamiento usa el gobierno común de efectos y contempla resultado de aplicación desconocido. |
| Reespera ante falta temporal y cierta de capacidad | ACREDITADO | Un fallo temporal y definitivamente no aplicado vuelve a cola sin convertir el trabajo en fallo terminal. |
| Puertos neutrales de arranque, observación y control | ACREDITADO | Existen contratos independientes del proveedor, con parada cooperativa o forzada identificada exactamente. |
| Sesión de ejecución y buzón causal básico | ACREDITADO | Existe la entrega causal hija (`child_delivery`); no equivale aún al protocolo completo de mensajes de `ORC-15` en V27. V38 exige continuidad de mensajes como conducta acotada, pero no posee ni acredita `ORC-15`. |
| Adaptador Codex durable por perfil | PARCIAL | Supervisa procesos, diario por ejecución, grupos de control, recuperación de rutas y parada; su conjunto de perfiles se configura de forma estática. |
| Demanda deseada frente a agentes reales | PENDIENTE | No existe un reconciliador durable que compare ambos conjuntos y corrija la diferencia. |
| Disponibilidad, cuota y límites de proveedor (`AGT-12`) | PENDIENTE | Pertenece a V25; el uso Codex se informa actualmente como desconocido y no hay fuente fiable de cuota, fichas ni gasto. |
| Agrupadores, licencias y plazas como capacidad externa (`ORC-28`) | PENDIENTE | V38 es su vertical propietaria canónica. El contrato continúa planificado y no existe todavía la composición acreditada de reserva y consumo externo. |
| Reserva transaccional de plaza | PENDIENTE | No hay reserva de capacidad con arrendamiento y cerca en la capa de aplicación. |
| Arranque realmente paralelo | PENDIENTE | El planificador recorre todo el conjunto listo, pero un único bucle llama en serie a `ProcessNext`; el máximo por ciclo limita la ráfaga, no crea trabajadores paralelos. |
| Altas y bajas dinámicas de agentes | PENDIENTE | El agrupador actual de perfiles (`Pool`) elige entre perfiles configurados; no añade ni retira agentes en caliente. |
| Supervisión estructurada y vigilante | PENDIENTE | Faltan contrato unificado de salud/progreso, política de aviso y escalado de parada. |
| Conservación pendiente de revisión (`preserved_pending_review`) | PENDIENTE | No hay inventario durable de entornos conservados ni flujo de sellado previo al desmontaje. `OPS-17` sigue en V32; V38 requiere esta conducta sin poseerla ni acreditarla. |
| Demanda lógica 1/16/70/500 | PENDIENTE | El contrato planificado V38 exige exactamente esos cuatro tamaños de demanda durable. No son una afirmación de simultaneidad física ni acreditación. |
| Pasos físicos 1/5/10/16/20 | PENDIENTE | El mismo contrato exige exactamente esos cinco escalones de agentes simultáneos sobre recursos medidos. No incluyen 70 ni 500 y todavía no tienen evidencia acreditadora. |

Las capacidades todavía no acreditadas que gobiernan este capítulo son
principalmente `ORC-15`, `ORC-28`, `OPS-16` y `OPS-17`. V38 posee únicamente
`ORC-28`; `ORC-15` pertenece a V27 y `OPS-16`/`OPS-17` a V32. Mensajes,
parada exacta y conservación son requisitos acotados de V38, pero no trasladan
ni vuelven a acreditar esas tres capacidades. Son prerrequisitos útiles ya
acreditados `GOV-21`, `ORC-10`,
`EVD-13`, `AGT-01` y `AGT-03`. `AGT-12` es una dependencia distinta de
proveedor en V25; no se considera incluida por mencionar capacidad externa.
El estado adoptado debe consultarse en
`product/roadmap.json`, `product/capabilities.json` y `product/evidence/`; esta
tabla no los sustituye.

### 2.1 Estado canónico de V38

La revisión actual de `product/roadmap.json` contiene 257 capacidades, 38
verticales y 38 contratos. V1–V22 están acreditadas; V23–V38 permanecen
planificadas. V38 `agent_runtime_elastic` es canónica y prioritaria, pero
**planificada no significa implementada, conectada, ejercitada ni acreditada**.
Su fixture y sus pruebas protegen el plan; no son evidencia de producto.

V38 posee solo `ORC-28`. Enumera `ORC-15`, `OPS-16` y `OPS-17` como
requisitos no propios, sin reasignarlos ni acreditarlos.

V23 y V38 son independientes en ambos sentidos: V23 no depende de
Firecracker, KVM ni microVM para completar el Wizard; V38 y Firecracker tampoco
dependen de cerrar V23. Un fallo o retraso en una de las dos verticales no
autoriza a bloquear la otra.

### 2.2 Compuertas inseparables de V38

El contrato planificado se ejecuta en tres compuertas causales:

| Compuerta | Alcance exacto | Efecto de estado |
|---|---|---|
| A | Núcleo elástico neutral; no requiere KVM ni Firecracker. | No puede acreditar V38. |
| B | Adaptador Firecracker de agentes con activación explícita, sin sustitución automática y con una microVM por agente. | No puede acreditar V38 por sí sola. |
| C | Ola física real 1/5/10/16/20 sobre el mismo candidato que superó A y B. | Solo permite acreditar V38 después de A y B. |

Solo A+B+C superadas por el mismo candidato permiten acreditar V38. Un verde
de A no demuestra Firecracker; uno de B no demuestra la ola física; uno de C
no sustituye el núcleo neutral ni la composición previa.

### 2.3 Cuota, capacidad externa y aislamiento físico

No deben mezclarse dos contratos:

- `AGT-12`, en V25, descubre disponibilidad, cuota y límites de un proveedor;
- `ORC-28`, propiedad canónica de V38, representa agrupadores, licencias y
  plazas como capacidad externa reservable.

Una cuota observada puede alimentar una fuente de capacidad, pero no sustituye
la reserva externa. Una plaza reservada tampoco demuestra cuánto presupuesto o
cuota conserva el proveedor. Si el contrato V38 necesita
`AGT-12`, esa dependencia se añade primero al catálogo y se adopta antes de
programar; no se infiere desde este capítulo.

V38 exige continuidad de mensajes, parada exacta y conservación, pero las
mantiene explícitamente como conductas no propias de `ORC-15`, `OPS-16` y
`OPS-17`. Esas capacidades continúan respectivamente en V27, V32 y V32.

La arquitectura sigue permitiendo adaptadores alternativos detrás de los
puertos neutrales. Sin embargo, las compuertas B y C de V38 exigen una microVM
Firecracker por agente y activación expresa sin sustitución automática. Una
prueba con procesos locales puede satisfacer A, pero no sustituye B ni la
prueba física de C.

## 3. Modelo: demanda completa, admisión limitada

### 3.1 La demanda nace del DAG

La demanda deseada se obtiene de todos los `WorkItem` que cumplen a la vez:

- pertenecen a la generación vigente del `Goal`;
- sus dependencias están satisfechas;
- no están ya concluidos ni cancelados;
- no tienen una ejecución viva equivalente;
- su actor, proyecto, permisos y presupuesto permiten solicitar ejecución;
- no existe conflicto de escritura que obligue a serializarlos.

No se debe truncar este conjunto por el número actual de agentes. Confundir
«trabajos listos» con «trabajos que caben ahora» pierde demanda y produce el
atasco que el orquestador debe evitar.

La separación correcta es:

```text
DAG -> conjunto listo completo -> ejecuciones durables
                              -> admisión según capacidad
                              -> reservas
                              -> arranques
```

Una ejecución que espera plaza conserva su `ExecutionRef`, generación y
número de intento. La falta temporal de capacidad no consume un nuevo intento
de trabajo porque todavía no se ha ejecutado el agente.

### 3.2 Estado deseado y estado real

El reconciliador compara dos proyecciones subordinadas al `Goal`; no crea un
segundo ciclo de vida:

- **Deseado**: ejecuciones que deben estar esperando, aprovisionándose,
  ejecutándose, deteniéndose o conservándose.
- **Real**: reservas, procesos, microVM, sesiones, identidades externas,
  latidos, recibos y entornos que el adaptador puede demostrar.

Una proyección orientativa del vínculo de ejecución puede usar estos estados.
Los identificadores técnicos se muestran después de su significado:

| Significado | Identificador propuesto |
|---|---|
| esperando capacidad | `waiting_capacity` |
| reservado | `reserved` |
| lanzamiento solicitado | `launch_requested` |
| en ejecución | `running` |
| parada solicitada | `stop_requested` |
| conservando | `preserving` |
| conservado pendiente de revisión | `preserved_pending_review` |
| liberado | `released` |

Son estados del vínculo operativo, no sustitutos del estado del `Goal`, del
`WorkItem` ni de la ejecución. Las transiciones importantes se persisten con
revisión esperada, idempotencia, evento y salida en la misma transacción.

### 3.3 Regla de convergencia

Cada pasada debe ser idempotente:

1. leer una instantánea consistente del estado durable;
2. observar capacidad y agentes reales con evidencia fechada;
3. detectar vínculos faltantes, sobrantes, caducados o inciertos;
4. proponer una lista determinista de acciones;
5. reclamar cada acción por arrendamiento y cerca;
6. ejecutar efectos por el gobierno común;
7. persistir recibos y volver a calcular.

No se mantiene una cola privada dentro del proveedor. Tras una caída, el mismo
estado durable permite reconstruir la siguiente decisión.

## 4. Contratos mínimos

Los nombres concretos pueden ajustarse al código, pero las responsabilidades
no deben mezclarse.

### 4.1 Fuente de capacidad viva

Debe devolver una observación estructurada, nunca inferida de texto de consola:

```text
CapacityObservation
  proveedor y perfil
  instante de observación y caducidad
  calidad: disponible | agotada | desconocida | caducada | inaccesible
  plazas totales, ocupadas, reservadas y disponibles
  límites de fichas, dinero, tiempo, procesos y recursos, si son medibles
  fuente y recibo verificable
```

`CapacityObservation` significa «observación de capacidad» y se conserva aquí
solo como posible identificador técnico del contrato. Sus campos describen la
fuente observada, su vigencia, la calidad del dato, las plazas y los límites
medibles, junto con el recibo que permite verificarla.

`desconocida` no significa cero ni ilimitado. La política de aplicación decide si
espera, reduce la oleada o admite una operación de bajo riesgo. El adaptador
solo observa.

La disponibilidad derivada no se convierte en otra verdad persistente. Se
conservan hechos —observación, reservas y ocupaciones— y se recalcula.

### 4.2 Reserva de capacidad

Una reserva debe contener:

- proveedor y perfil exactos;
- `GoalRef`, generación, `WorkItemRef`, `ExecutionRef` e intento;
- unidades reservadas;
- identificador de arrendamiento (`lease_id`), fecha límite y cerca (`fence`);
- revisión de la observación que justificó la reserva;
- clave de idempotencia;
- estado de consumo o liberación.

La operación de reserva debe ser atómica respecto de las demás reservas del
mismo límite. Dos planificadores no pueden consumir la última plaza. Una cerca
antigua nunca puede lanzar, renovar ni liberar una reserva nueva.

### 4.3 Vínculo con el agente

El vínculo durable reúne las identidades causales y operativas:

- referencias de `Goal`, trabajo, ejecución, generación e intento;
- proveedor, modelo, perfil y especificación de lanzamiento;
- reserva consumida;
- `agent_ref` interno e identidad externa;
- proceso, grupo de control, microVM o sesión, según el adaptador;
- arrendamiento y cerca de control;
- estado de salud y última evidencia;
- referencia del entorno conservado.

No debe deducirse la identidad por nombre de ventana, orden de proceso o texto
que el agente haya escrito.

### 4.4 Sesión y buzón

El canal de órdenes debe funcionar durante toda la ejecución, no solo al
arranque. Como mínimo soporta:

- orden con identidad causal, secuencia y clave de idempotencia;
- confirmación de admisión distinta de confirmación de resultado;
- progreso estructurado;
- solicitud de contexto o herramienta;
- entrega de artefacto por referencia y resumen;
- aviso cooperativo de parada;
- respuesta final y recibo de cierre;
- reanudación desde el último mensaje confirmado.

La sesión puede materializarse con procesos, protocolo Codex, `vsock` u otro
transporte. `tmux` puede ser una herramienta de operación o diagnóstico, pero
no autoridad de identidad, cuota, progreso ni ciclo de vida.

### 4.5 Salud y progreso

La observación separa:

- **salud del entorno**: proceso o microVM presente, grupo de control,
  conectividad del canal, identidad y cerca vigentes;
- **salud del agente**: protocolo responde, latido válido;
- **progreso**: mensaje, herramienta, cambio de artefacto o punto de control;
- **resultado**: éxito o fallo acreditado.

El silencio por sí solo no demuestra bloqueo. Tampoco una frase del agente
demuestra cuota agotada o finalización. El vigilante puede avisar o pedir un
punto de control después de un plazo, pero solo escala a parada forzada por
hechos definidos: vencimiento confirmado, identidad imposible, proceso
irrecuperable, infracción de seguridad o incumplimiento de la parada
cooperativa.

## 5. Reconciliador de agentes

### 5.1 Un único dueño de la decisión

El reconciliador vive en `internal/application`. Puede despertarse por:

- cambio del DAG;
- llegada de un recibo;
- caducidad de una reserva o arrendamiento;
- observación de capacidad;
- latido o pérdida de salud;
- orden de parada;
- temporizador durable;
- recuperación tras reinicio.

El proveedor no decide qué trabajo cerrar, reemplazar o reabrir. Un Director
puede proponer prioridad o descomposición; la aplicación valida y persiste.

### 5.2 Cálculo de admisión

Para cada ejecución en espera de capacidad (`waiting_capacity`):

1. comprobar que sigue siendo necesaria y que su generación está vigente;
2. aplicar autorización, presupuesto, riesgo y conflicto de escritura;
3. obtener observaciones no caducadas de los adaptadores compatibles;
4. descontar reservas vivas y margen operativo explícito;
5. ordenar de forma determinista por prioridad y antigüedad;
6. crear la reserva con comparación de revisión;
7. crear la intención de lanzamiento;
8. reclamar y ejecutar el efecto fuera de la transacción;
9. persistir intento y recibo;
10. consumir o liberar la reserva según el resultado.

Si el proveedor responde «capacidad temporalmente ausente» y demuestra que no
aplicó el lanzamiento, se libera o renueva la reserva y la misma ejecución
vuelve a esperar. Si no puede saber si lo aplicó, se entra en cuarentena de
resultado desconocido: primero se reconcilia por identidad e idempotencia;
nunca se relanza a ciegas.

Pseudocódigo de una pasada:

```text
conciliar(instante):
    estado = repositorio.leer_revision_consistente()
    demanda = derivar_todo_el_conjunto_listo(estado)
    asegurar_ejecuciones_durables(demanda)

    observaciones = capacidad.observar_todos(instante)
    reales = agentes.observar_identidades_conocidas()

    para cada vinculo en ordenar_deterministamente(estado.vinculos):
        si vinculo.deberia_existir y no reales.contiene(vinculo):
            resolver_ausencia_o_resultado_desconocido(vinculo)
        si no vinculo.deberia_existir y reales.contiene(vinculo):
            solicitar_parada_exacta_y_conservacion(vinculo)

    candidatas = ejecuciones_en_espera(estado)
    para cada ejecucion en ordenar_por_prioridad_y_antiguedad(candidatas):
        plaza = reservar_atomicamente(ejecucion, observaciones)
        si plaza.existe:
            publicar_intencion_de_lanzamiento(ejecucion, plaza)

    reclamar_y_despachar_acciones_hasta_capacidad_visible()
```

La función puede repetirse después de cualquier caída. Su corrección no
depende de variables de memoria conservadas entre pasadas.

### 5.3 Equidad, contrapresión y estabilidad

El reconciliador mantiene una sola decisión de planificación y añade política
durable sobre el mismo conjunto listo. No crea un segundo planificador, una
cola privada por proveedor ni contadores decisorios en memoria.

| Código | Invariante | Estado durable necesario |
|---|---|---|
| `equidad_contrapresion_histeresis` | La presión no pierde demanda, la equidad evita inanición y la escala no oscila por una sola muestra. | Turno o déficit por proyecto y proveedor, antigüedad, siguiente intento, ventana de estabilización, contadores de ráfaga y última decisión. |

Reglas de equidad:

- repartir entre proyectos y proveedores con pesos y presupuestos explícitos;
- envejecer trabajo válido para impedir inanición por prioridades nuevas;
- conservar la posición lógica después de reiniciar;
- excluir temporalmente solo por permiso, riesgo, conflicto o capacidad
  demostrados;
- registrar por qué una ejecución quedó detrás de otra.

La colocación separa requisitos duros y preferencias:

- arquitectura, memoria, KVM, activos, red y datos locales son restricciones;
- afinidad de anfitrión, coste y localidad son preferencias;
- el vínculo elegido conserva anfitrión, revisión y cerca;
- en varios anfitriones, la fuente transaccional asigna una cerca creciente
  que invalida escrituras del anfitrión anterior;
- perder un anfitrión devuelve la misma ejecución a conciliación, no crea otra
  identidad.

La contrapresión usa una espera exponencial durable con dispersión aleatoria.
Se persisten número de espera, causa y `siguiente_intento_en`; el reinicio no
los reinicia a cero. La dispersión se elige una vez y se guarda para que dos
anfitriones no vuelvan a golpear al proveedor al mismo tiempo.

Para evitar oscilaciones y tormentas:

- umbrales distintos para aumentar y reducir capacidad;
- ventanas de estabilización y observaciones sucesivas;
- cadencia mínima de conciliación;
- límite de altas y bajas por proveedor, proyecto y anfitrión;
- presupuesto de reintentos y máximo de efectos simultáneos;
- apertura temporal del circuito ante fallos repetidos;
- renovación anticipada, pero acotada, de arrendamientos.

Una nueva observación urgente puede despertar al reconciliador, pero no saltar
los límites ni crear otro bucle de autoridad.

### 5.4 Paralelismo real

Recorrer rápido el conjunto listo no basta. Debe existir un único despachador
global de aplicación. Ese despachador reclama acciones en el orden canónico de
la bandeja; no mantiene una cola privada ni un selector dedicado solo a
lanzamientos. Dentro de ese orden, una parada reclamable tiene prioridad sobre
cualquier lanzamiento nuevo.

La concurrencia se abre después de reclamar:

- si el límite de lanzamientos es cero o todos sus permisos están ocupados, el
  despachador usa el selector global sin lanzamiento y sigue reclamando
  observaciones, paradas y las demás acciones no concurrentes;
- si existe permiso potencial, reclama la siguiente acción global sin cambiar
  el orden canónico y respetando parada antes de lanzamiento;
- solo cuando la acción ya reclamada es un lanzamiento obtiene un permiso y
  crea una tarea concurrente;
- al terminar ese lanzamiento, libera el permiso aunque el efecto falle;
- la selección global sin lanzamiento conserva equidad durable entre sus
  acciones, de modo que la observación progresa aunque todos los permisos de
  lanzamiento estén ocupados;
- parada, observación, preparación del espacio de trabajo, operaciones Git y
  atestación se ejecutan serialmente en el despachador global; no quedan detrás
  de una cola privada de lanzamientos;
- la contrapresión conserva acciones durables y no las descarta.

No hay un techo global oculto de tres, seis o setenta. Sí puede haber límites
reales y visibles por proveedor, perfil, proyecto, anfitrión, memoria, CPU,
descriptores, almacenamiento o presupuesto. Quinientas ejecuciones durables no
implican quinientos trabajadores ociosos: solo existen tareas concurrentes para
lanzamientos ya reclamados y con permiso.

Pseudocódigo del despacho concurrente:

```text
despachar_siguiente():
    limite = politica.lanzamientos_simultaneos()
    si limite == 0 o permisos_lanzamiento.ocupados(limite):
        selector = selector_global_sin_lanzamiento
    en_otro_caso:
        selector = selector_global_cualquier_accion

    accion = repositorio.reclamar_siguiente_en_orden_canonico(
        selector,
        prioridad = parada_antes_de_nuevo_lanzamiento,
        equidad_no_lanzamiento = observacion_debe_progresar,
        nuevo_arrendamiento(),
        siguiente_cerca()
    )
    si accion.no_existe:
        terminar_pasada
    si accion.es_lanzamiento:
        permiso = permisos_lanzamiento.adquirir()
        ejecutar_lanzamiento_en_tarea(accion, permiso)
        terminar_pasada
    ejecutar_en_el_despachador(accion)

ejecutar_lanzamiento_en_tarea(accion, permiso):
    al_terminar liberar(permiso)
    si accion.cerca_ya_no_es_vigente:
        terminar_sin_efecto
    recibo = proveedor.ejecutar_idempotente(accion)
    repositorio.persistir_recibo_si_cerca_vigente(recibo)
```

## 6. Escalado por cohortes

Las cohortes validan más que el número de procesos. Cada una debe probar:

- generación completa de demanda;
- reservas sin sobreasignación;
- arranque concurrente;
- mensajería bidireccional;
- trabajo útil independiente;
- recogida de resultados;
- parada exacta;
- conservación individual;
- reinicio del orquestador durante al menos una fase;
- ausencia de duplicados y procesos propios huérfanos.

Orden operativo recomendado:

Demanda lógica exacta del contrato planificado V38:

| Ejecuciones durables | Objetivo lógico |
|---:|---|
| 1 | Recorrido causal mínimo de una ejecución durable. |
| 16 | Demanda completa sin recorte oculto y admisión según capacidad. |
| 70 | Demanda grande conservada aunque la capacidad simultánea sea menor. |
| 500 | Todas las ejecuciones existen, esperan y progresan por oleadas sin pérdida. |

Pasos físicos exactos del mismo contrato:

| Agentes simultáneos | Objetivo físico |
|---:|---|
| 1 | Identidad, arranque, orden, resultado, parada y conservación completos. |
| 5 | Primera carrera real entre reservas, lanzamientos y cierres. |
| 10 | Contrapresión, renovación de arrendamientos y reparto entre perfiles. |
| 16 | Presión combinada sobre aislamiento, reservas y recuperación. |
| 20 | Supervisión y recuperación con fallos parciales inducidos. |

Las dos series prueban sujetos distintos. Que 1 y 16 aparezcan en
ambas no permite fusionarlas: una comprueba generación y conservación de
demanda; la otra, simultaneidad física medida. Ni 70 ni 500 son pasos físicos
de V38. Que las cifras sean canónicas no las vuelve acreditadas: faltan la
implementación, la composición y la evidencia del mismo candidato.

Antes de cada tamaño lógico o paso físico se registra un presupuesto:

- CPU, memoria, procesos, descriptores, disco e imágenes;
- plazas y cuota de cada proveedor;
- fichas y dinero máximos;
- tiempo máximo de arranque y de trabajo;
- reserva para el propio orquestador;
- criterio de suspensión y de reanudación.

Para 500 trabajos con veinte plazas, la aceptación correcta es que haya
quinientas ejecuciones durables, nunca más de veinte reservas activas, y que
todas progresen por oleadas sin duplicarse. No es correcto crear solo veinte y
olvidar las otras cuatrocientas ochenta.

## 7. Alta, reutilización y relevo

Hay que distinguir tres recursos:

1. el perfil o cuenta de proveedor;
2. el proceso o microVM que aloja una ejecución;
3. la sesión causal del agente.

Un perfil puede reutilizarse cuando recupera capacidad. Un entorno de ejecución
no debe reutilizarse para otro trabajo salvo que el adaptador acredite un
reinicio completo: limpieza de credenciales, archivos, procesos, red, contexto
y vínculo causal, más una nueva atestación. La opción segura por defecto es un
entorno por ejecución y conservación posterior.

El relevo por cuota o contexto sigue esta secuencia:

1. detener nuevas órdenes al agente saliente;
2. pedir punto de control estructurado;
3. sellar referencias de cambios, artefactos, mensajes confirmados y asuntos
   pendientes;
4. iniciar otra ejecución o sesión con nueva reserva;
5. entregar solo el contexto mínimo por referencias verificadas;
6. confirmar recepción e identidad;
7. revocar la capacidad de escribir del saliente mediante cerca;
8. detener y conservar su entorno.

Nunca se decide un relevo buscando palabras como «límite» o «cuota» en la
salida. La causa debe venir de una observación estructurada del proveedor o de
un límite interno medido.

## 8. Parada exacta y conservación

### 8.1 Identidad de parada

Una solicitud de parada identifica exactamente:

- `GoalRef`, generación y `WorkItemRef`;
- `ExecutionRef` e intento;
- especificación de lanzamiento y proveedor;
- `agent_ref` e identidad externa;
- arrendamiento y cerca de control;
- modo y motivo autorizados;
- plazo cooperativo;
- clave de idempotencia.

No se permiten barridos amplios por nombre, usuario o patrón de proceso.

### 8.2 Secuencia cooperativa y forzada

La secuencia obligatoria por agente es:

1. impedir nuevas asignaciones y marcar parada solicitada (`stop_requested`);
2. enviar solicitud cooperativa por el canal autenticado;
3. esperar confirmación y punto de control hasta el plazo;
4. verificar que el proceso o microVM exacto terminó;
5. si no terminó, autorizar explícitamente el escalado;
6. aplicar señal o apagado forzado solo al recurso identificado;
7. verificar descendientes y grupo de control;
8. cerrar credenciales y servicios;
9. inventariar y sellar el entorno;
10. persistir el recibo y el estado conservado pendiente de revisión
    (`preserved_pending_review`);
11. desmontar o desasociar recursos activos sin borrar datos.

La parada forzada no es una segunda orden independiente sin relación: debe
referenciar la cooperativa vencida y su evidencia.

### 8.3 Cero borrado automático

El estado conservado pendiente de revisión (`preserved_pending_review`)
significa:

- ningún proceso conserva capacidad de ejecución;
- ninguna credencial continúa válida;
- el contenido está inmutable o sellado;
- existen inventario, tamaños, propietarios y huellas;
- se conocen rutas o referencias de almacenamiento;
- el estado puede inspeccionarse y atribuirse a una ejecución;
- no se ha borrado ningún dato.

La eliminación es otro efecto, posterior, autorizado, idempotente y auditable.
Debe indicar qué revisión permitió retirar qué entorno. Ni el vigilante, ni el
reconciliador, ni el adaptador pueden convertir «ya no se usa» en permiso de
borrado.

## 9. Reinicio y recuperación sin duplicados

Al arrancar, el orquestador no da por muertos todos los agentes ni relanza todo.
Ejecuta una conciliación:

1. carga acciones, reservas, vínculos y recibos durables;
2. invalida reclamaciones caducadas mediante una cerca superior;
3. pregunta a cada adaptador por identidades externas conocidas;
4. coteja diarios, procesos, grupos de control, microVM y sesiones;
5. clasifica cada elemento como presente y coincidente, ausente,
   contradictorio o incierto;
6. adopta solo recursos cuya identidad causal y cerca coincidan;
7. completa recibos que el adaptador puede demostrar;
8. reintenta únicamente efectos definitivamente no aplicados;
9. pone en cuarentena los resultados desconocidos;
10. detiene y conserva recursos propios huérfanos identificables.

Casos que deben probarse:

- caída antes de persistir la intención;
- caída después de la intención y antes del intento;
- caída durante el lanzamiento;
- agente arrancado y recibo aún no persistido;
- reserva caducada mientras el proceso sigue vivo;
- respuesta final duplicada;
- parada cooperativa aplicada y recibo perdido;
- entorno sellado antes de persistir el estado conservado pendiente de revisión
  (`preserved_pending_review`).

En todos ellos una clave de idempotencia estable y una identidad externa
recuperable evitan el segundo agente accidental.

## 10. Telemetría necesaria

Las métricas y trazas son proyecciones, no autoridad. Deben permitir responder:

- demanda lista total y antigüedad;
- ejecuciones esperando capacidad;
- observaciones por calidad y edad;
- reservas vivas, caducadas, consumidas y disputadas;
- latencia de reserva, arranque, primera respuesta y cierre;
- agentes reales por proveedor, proyecto y estado;
- renovaciones rechazadas por cerca;
- avisos, puntos de control y paradas forzadas;
- resultados de aplicación desconocidos;
- entornos conservados pendientes de revisión (`preserved_pending_review`),
  tamaño y antigüedad;
- procesos o microVM propios huérfanos;
- gasto y cuota medidos frente a presupuesto;
- presión de CPU, memoria, disco, procesos y descriptores.

Cada dato público visible usa claves de internacionalización. Las etiquetas no
incluyen secretos, texto de plantillas de instrucciones ni identificadores de
cardinalidad ilimitada.

## 11. Pruebas y acreditación

### 11.1 Contratos y negativos

Como mínimo:

- dos reservas concurrentes no consumen una única plaza;
- una cerca antigua no renueva, lanza, detiene ni libera;
- capacidad desconocida no se interpreta como ilimitada;
- falta temporal no consume intento ni pierde ejecución;
- resultado desconocido no provoca relanzamiento;
- identidad incompleta no permite parada;
- silencio sin evidencia no permite matar;
- un agente no puede confirmar el resultado de otro;
- un proyecto no observa ni controla agentes de otro;
- un proyecto con carga continua no impide para siempre el progreso de otro
  con derecho a capacidad;
- un reinicio conserva turno, antigüedad y siguiente intento;
- observaciones alternantes alrededor de un umbral no provocan altas y bajas
  continuas;
- el límite de cadencia impide una tormenta de reintentos contra el proveedor;
- un anfitrión que perdió la cerca no puede escribir ni detener el agente
  reasignado;
- detener uno entre cientos no afecta a sus vecinos;
- sellar no borra;
- retirar sin aprobación se rechaza;
- un reinicio repetido mantiene una sola ejecución externa.

### 11.2 Mutaciones

Las pruebas deben matar variantes defectuosas que:

- trunquen el conjunto listo al tamaño de la capacidad;
- resten reservas después de lanzar;
- reinicien a cero la espera exponencial o la posición de equidad;
- eliminen la ventana de estabilización o el límite de cadencia;
- acepten una escritura de un anfitrión con cerca antigua;
- omitan generación, intento o cerca;
- traten capacidad desconocida como disponible;
- acepten texto libre como agotamiento de cuota;
- reutilicen una sesión sin reinicio acreditado;
- marquen conservado antes del inventario;
- borren en el camino de parada;
- consideren un acuse de admisión como resultado.

### 11.3 Extremo a extremo

Cada prueba de tamaño lógico o paso físico guarda:

- huella del código, binario o imagen y configuración efectiva;
- instantánea inicial de demanda y presupuesto;
- recibos de reservas, arranques, mensajes, resultados y paradas;
- inventario de entornos conservados;
- prueba de reinicio y reconciliación;
- comprobación final de cero procesos propios no reclamados;
- revisión independiente sobre los mismos artefactos.

Solo entonces la capacidad puede pasar de declarada a implementada, conectada,
ejercitada y finalmente acreditada.

## 12. Diagnóstico sin alterar el estado

El diagnóstico comienza por lecturas y no por reinicios. El orden recomendado
es:

1. identificar `Goal`, generación, `WorkItem`, ejecución e intento;
2. leer el estado durable y su revisión;
3. leer la acción, reclamación, arrendamiento y cerca vigentes;
4. revisar intención, aprobación, intento y recibo del efecto;
5. consultar la última observación de capacidad y su caducidad;
6. consultar al adaptador por la identidad externa exacta;
7. cotejar sesión, diario, proceso o microVM y grupo de control;
8. reconstruir la decisión que tomaría el reconciliador;
9. modificar estado solo mediante el comando de recuperación previsto.

| Síntoma | Hipótesis que comprobar primero | Evidencia decisiva |
|---|---|---|
| Hay trabajos listos pero no arrancan | No se crearon ejecuciones, el despachador no reclama, los permisos están ocupados o falta capacidad | Conjunto listo, ejecuciones durables, acciones reclamables, permisos y observación fechada |
| Siempre trabajan tres agentes | Lista estática de perfiles o límite configurado | Configuración efectiva, perfiles observados y reservas |
| Se crean agentes duplicados | Clave de idempotencia cambiante o resultado desconocido relanzado | Intenciones, identidades externas y recibos por ejecución |
| Un agente parece colgado | Canal roto, proveedor detenido o trabajo largo sin progreso | Salud estructurada, último mensaje confirmado, proceso y consumo |
| La parada afecta a otro agente | Identidad incompleta o cerca omitida | Solicitud exacta, vínculo y recibo del controlador |
| Quedan procesos después del cierre | Inventario del adaptador incompleto o grupo de control no verificado | Registro propio, árbol de procesos y grupo exacto |
| Desaparece un entorno terminado | La limpieza mezcló desmontaje con borrado | Recibo de conservación y cualquier orden separada de retirada |

No se arregla una discrepancia editando directamente la base de datos,
matando procesos por patrón o borrando diarios. Esas acciones destruyen la
evidencia necesaria para saber qué ocurrió.

## 13. División en tareas pequeñas

El orden causal recomendado evita construir otra aplicación paralela:

1. caracterizar el contrato de capacidad y sus calidades;
2. añadir el puerto y un adaptador falso contractual;
3. modelar reserva, arrendamiento, cerca e idempotencia;
4. implementar reserva atómica en la fuente de estado vigente;
5. representar la espera de capacidad (`waiting_capacity`) sin duplicar el
   ciclo de vida;
6. implementar el reconciliador determinista con el adaptador falso;
7. hacer concurrentes los lanzamientos ya reclamados sin duplicar el
   despachador global;
8. adaptar Codex para observación estructurada y reserva real;
9. añadir alta y baja dinámica de perfiles o ejecutores;
10. completar sesión y buzón bidireccional recuperable;
11. añadir salud, progreso, aviso y punto de control;
12. implementar parada cooperativa y escalado exacto;
13. implementar inventario, sellado y el estado conservado pendiente de
    revisión (`preserved_pending_review`);
14. implementar recuperación y adopción tras reinicio;
15. ejecutar la demanda lógica 1, 16, 70 y 500;
16. ejecutar los pasos físicos 1, 5, 10, 16 y 20 como compuerta C del mismo
    candidato que superó A y B;
17. acreditar y retirar las decisiones del legado ya sustituidas.

Cada tarea declara capacidad, invariante, autoridad, puertos, adaptadores,
conjunto de escritura, dependencias, prueba contractual, negativo, mutación,
prueba integral y presupuesto. No se abre la siguiente vertical si la anterior
no deja una frontera ejecutable y verificable.

## 14. Lecciones que se conservan del legado

La consulta histórica aporta conducta, no código ni autoridad:

- la capacidad se deriva de fuentes vivas; no se mantiene a mano;
- proveedor observa y aplicación decide;
- agotamiento de cuota requiere evidencia estructurada, no palabras en salida;
- antes de un relevo se obtiene un punto de control;
- el agente saliente pierde escritura mediante una cerca;
- una sesión de terminal puede ayudar a operar, pero no representa la verdad;
- el registro de procesos necesita identidad durable y recuperación;
- aviso, sondeo, relevo y parada son operaciones distintas;
- la autogestión sigue sometida a aprobación, presupuesto y alcance.

Las consultas obligatorias de lecciones para `ORC-28`, `AGT-12`, `ORC-15`,
`OPS-16` y `OPS-17` no encontraron coincidencias automáticas para las rutas de
este manual. Ese hueco no invalida las fuentes históricas revisadas ni crea un
bloqueo; debe quedar registrado al abrir las tareas de implementación.

Fuentes históricas consultadas, siempre en solo lectura:

Todas las rutas de la lista siguiente se resolvieron bajo la copia de consulta
`/home/alberto/Trabajo/orquestaV2-legacy-consulta`; no pertenecen al producto
nuevo ni se importan desde él.

- `docs/diseno_pools_capacidad.md`
- `docs/diseno_minimo_pools_y_presupuestos.md`
- `docs/diseno_control_activo_agentes.md`
- `docs/op_087_autogestion_supervisada_agentes.md`
- `docs/op_089_relevo_agentes_por_token.md`
- `docs/op_090_sondeo_y_nudge_de_agentes.md`
- `docs/runbook_control_plane_agentes.md`
- `docs/tarea_codex_quota_auto_replan_2026-06-13.md`
- `modulos/orquesta-runtime-codex-delivery/progress_failure_context_v0.go`
- `modulos/orquesta-app-codex-stack/agent_usage_runtime_source_v0.go`
- `modulos/orquesta-app-codex-stack/drain_live_agent_reconciliation_v0.go`
- `modulos/orquesta-director/agent_progress_supervisor_v0.go`
- `modulos/orquesta-state-file/agent_process_registry_v0.go`

## 15. Fuentes vigentes de esta especificación

- `AGENTS.md`
- `product/roadmap.json`
- `product/capabilities.json`
- `docs/reconstruccion/ruta_total_100.md`
- `docs/reconstruccion/handoff_continuacion_agente_2026-07-30.md`
- `internal/application/`
- `internal/ports/agent.go`
- `internal/ports/agent_control.go`
- `internal/adapters/agent/codex/`
- `internal/bootstrap/scheduler.go`
- `config/registry.json`

Propuesta local aún no adoptada:

- `acceptance/fixtures/v38_agent_runtime_elastic_plan.json`

Antes de implementar una tarea se vuelven a consultar la hoja de ruta, el estado
vivo, las lecciones del legado por capacidad/ruta/operación y el `AGENTS.md` local
del módulo afectado. Este capítulo orienta la construcción; la acreditación la
deciden contratos ejecutados y recibos inmutables.
