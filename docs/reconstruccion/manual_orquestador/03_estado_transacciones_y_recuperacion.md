# 03. Estado, transacciones y recuperación

> **Responsabilidad:** definir cómo persistir y recuperar un orquestador sin
> perder causalidad, duplicar efectos ni convertir eventos, copias o procesos
> en autoridades alternativas.
>
> **Alcance:** estado centrado en instantáneas, revisión esperada, idempotencia,
> transacciones, bandeja de salida, eventos, arrendamientos, cercas, reinicio,
> repetición, migraciones, copia, verificación, restauración y pruebas de
> corrupción.
>
> **No acredita:** este documento no acredita el estado vivo, una copia
> concreta, PostgreSQL, despliegue en varios anfitriones, el entorno elástico,
> Firecracker ni el producto total. Solo los registros y recibos de un
> candidato sellado pueden hacerlo.

## Al terminar este capítulo, el lector sabrá…

- persistir una transición, su evento y su salida de forma atómica;
- distinguir instantánea autoritativa, auditoría, proyección y recibo;
- recuperar arrendamientos, acciones y efectos sin duplicarlos;
- diseñar copias, restauraciones y migraciones verificables;
- especificar una composición multianfitrión con PostgreSQL, S3 compatible y
  recuperación de agentes.

El vocabulario compartido se define en el
[glosario técnico común](01_mision_limites_y_vocabulario.md#glosario-técnico-común).

## Índice del capítulo

- [1. Autoridades y estado conocido](#1-autoridades-y-estado-conocido)
- [2. Principio central: el estado manda](#2-principio-central-el-estado-manda)
- [3. Fronteras de consistencia](#3-fronteras-de-consistencia)
- [4. Regla de atomicidad](#4-regla-de-atomicidad)
- [5. Revisión esperada e idempotencia](#5-revisión-esperada-e-idempotencia)
- [6. Bandeja transaccional](#6-bandeja-transaccional)
- [7. Arrendamientos y cercas](#7-arrendamientos-y-cercas)
- [8. Máquina segura alrededor de un efecto](#8-máquina-segura-alrededor-de-un-efecto)
- [9. Eventos y recibos](#9-eventos-y-recibos)
- [10. Reinicio y repetición](#10-reinicio-y-repetición)
- [11. Copia de seguridad verificable](#11-copia-de-seguridad-verificable)
- [12. Verificación de copia](#12-verificación-de-copia)
- [13. Restauración segura](#13-restauración-segura)
- [14. Migraciones](#14-migraciones)
- [15. Matriz de fallos](#15-matriz-de-fallos)
- [16. Pruebas obligatorias](#16-pruebas-obligatorias)
- [17. Secuencia de construcción desde cero](#17-secuencia-de-construcción-desde-cero)
- [18. Contrato ejecutable objetivo para V31 multianfitrión](#18-contrato-ejecutable-objetivo-para-v31-multianfitrión)
- [19. Antipatrones que invalidan la recuperación](#19-antipatrones-que-invalidan-la-recuperación)
- [20. Procedimiento operativo abreviado](#20-procedimiento-operativo-abreviado)
- [21. Lista de salida del capítulo](#21-lista-de-salida-del-capítulo)

## 1. Autoridades y estado conocido

Las reglas de arquitectura proceden de
[AGENTS.md](../../../AGENTS.md) y
[ruta_total_100.md](../ruta_total_100.md). Los contratos V06 y V09 aparecen
acreditados en la base versionada `HEAD:product/roadmap.json`, consultada con
`git show HEAD:product/roadmap.json`, con descriptores
sellados en:

- [evidencia V06](../../../product/evidence/v06_atomic_state_outbox.json);
- [evidencia V09](../../../product/evidence/v09_recovery_backup.json);
- [evidencia V12](../../../product/evidence/v12_director_lease.json).

Esas evidencias describen revisiones históricas exactas. Este capítulo explica
cómo conservar sus invariantes y cómo reconstruir la solución desde cero; no
afirma que cualquier modificación posterior esté acreditada.

El adaptador SQLite actual es una referencia concreta. PostgreSQL, S3, el
despliegue en varios anfitriones y la recuperación física de agentes son
diseño o deuda hasta que sus contratos propios pasen.

## 2. Principio central: el estado manda

La instantánea del `Goal`, junto con sus hechos causales persistidos, es la
autoridad de reconstrucción. El registro de eventos audita lo sucedido; no
compite como segundo origen.

```text
Estado autoritativo
  instantánea del Goal
  WorkItems
  Executions
  decisiones y controles
  intenciones/aprobaciones/intentos/recibos
  artefactos y atestaciones como metadatos causales
  bandeja transaccional (outbox) y recibos de consumo

Auditoría y vistas
  eventos
  timelines
  dashboards
  índices de lectura
```

Después de un reinicio se restaura la instantánea y se validan sus vínculos. No
se reproducen eventos para “adivinar” un estado que pueda discrepar de ella.

Una fuente transaccional está activa por despliegue. Una réplica, una copia de
seguridad, un fichero exportado o una proyección no pueden aceptar comandos.

## 3. Fronteras de consistencia

### 3.1 Agregado

El `Goal` y sus `WorkItems` forman la frontera de invariantes de dominio. Una
transición recibe la revisión esperada y produce una instantánea nueva e
inmutable.

Revisiones diferentes no son intercambiables:

- revisión de `Goal`: cerca todo cambio del agregado;
- revisión de `WorkItem`: cerca la unidad exacta;
- generación de plan: cerca el prefijo causal aceptado;
- generación de `AppSpec`: cerca la intención confirmada;
- número de intento: distingue ejecuciones sucesivas;
- cerca de acción: distingue reclamaciones de trabajadores;
- cerca de Director: distingue titulares del mando.

Nunca usar una sola de ellas como sustituto de las demás.

### 3.2 Operación de repositorio

El puerto de estado expone operaciones de aplicación completas. Por ejemplo,
“registrar lanzamiento preparado” o “aplicar plan del Director”, no “actualizar
columna estado”.

El motor:

1. lee;
2. valida con dominio;
3. construye el estado completo de la operación;
4. llama a una única operación del puerto.

El adaptador, dentro de una transacción:

1. relee las cercas necesarias;
2. usa su reloj confiable para arrendamientos;
3. comprueba identidad, generación y estado esperado;
4. escribe instantánea, hechos, eventos y bandeja;
5. consume o mantiene la reclamación;
6. confirma una vez.

El contrato actual se encuentra en
[`StateRepository`](../../../internal/application/state.go). Que la interfaz
sea amplia no autoriza a dividirla en almacenes con autoridades distintas.

## 4. Regla de atomicidad

Una transición que crea trabajo debe persistir, como una sola unidad:

```text
instantánea nueva
+ ejecuciones creadas o modificadas
+ acciones de la bandeja transaccional
+ evento de auditoría
+ decisiones/controles/autoridades que causan el cambio
+ reservas o liquidaciones de presupuesto aplicables
```

No se permite:

```text
confirmar la transacción del Goal
caída del proceso
INSERT en la tabla de la bandeja transaccional
```

ni:

```text
efecto externo
caída del proceso
marcar como despachando
```

El primer caso pierde trabajo; el segundo puede duplicar un efecto. La
transición local previa al efecto debe ser durable. El resultado externo se
registra después mediante un intento y un recibo idempotentes.

En SQLite las escrituras usan una transacción y un escritor dedicado; véase
[`internal/adapters/state/sqlite/transaction.go`](../../../internal/adapters/state/sqlite/transaction.go).
Esa elección física no debe filtrarse al dominio.

## 5. Revisión esperada e idempotencia

### 5.1 Comparación y sustitución

Cada comando que parte de estado existente declara sus revisiones esperadas.
El repositorio las vuelve a comprobar justo antes de confirmar.

Resultado:

- si coinciden, puede confirmar;
- si no coinciden, devuelve conflicto;
- nunca fusiona de forma implícita;
- el llamador relee y decide de nuevo.

La comparación y sustitución evita pérdida de actualización, pero no sustituye
la idempotencia del comando ni la cerca de una reclamación.

### 5.2 Petición repetible

Una petición durable conserva:

- `request_ref`;
- huella canónica de todos los campos con significado;
- principal, proyecto y sujeto;
- resultado compacto persistido.

Casos:

| Entrada posterior | Resultado |
|---|---|
| misma referencia, misma huella | devolver el resultado ya persistido |
| misma referencia, huella distinta | conflicto sin efecto |
| referencia nueva, revisión antigua | conflicto de revisión |
| referencia nueva, revisión actual | nueva decisión posible |

La búsqueda de repetición debe ocurrir antes de generar IDs, pedir otra
autorización o ejecutar un efecto. Los campos generados internamente que no
forman parte de la intención no deben hacer divergente una repetición exacta.

### 5.3 Idempotencia de efecto

Todo efecto externo separa cuatro hechos:

```text
intención -> aprobación -> intento -> recibo
```

- intención (`intent`): qué se pretende, contra qué sujeto y con qué clave;
- aprobación (`approval`): quién lo autorizó, alcance y caducidad;
- intento (`attempt`): invocación concreta bajo una cerca;
- recibo: resultado observado del sistema externo.

Un texto del agente, un código HTTP de admisión o un proceso iniciado no
sustituyen al recibo.

## 6. Bandeja transaccional

### 6.1 Creación

La acción nace en la misma transacción que el estado que la vuelve necesaria.
Incluye sujeto, generaciones, disponibilidad e intención de efecto.

### 6.2 Reclamación

Un trabajador solicita la siguiente acción compatible con:

- referencia de trabajador;
- token de reclamación nuevo;
- duración de arrendamiento;
- capacidades reales;
- política de presupuesto.

El repositorio:

1. usa tiempo de su propia transacción;
2. selecciona una acción disponible y compatible;
3. valida aprobación y capacidad;
4. reserva presupuesto cuando corresponde;
5. incrementa el intento de entrega y la cerca;
6. fija el vencimiento;
7. confirma la reclamación.

La selección por ventanas o páginas es una optimización interna. Debe poder
seguir recorriendo candidatos y no puede convertirse en un tope global oculto.
La referencia SQLite está en
[`internal/adapters/state/sqlite/claim.go`](../../../internal/adapters/state/sqlite/claim.go).

### 6.3 Consumo, reprogramación y cuarentena

- **consumo:** termina la entrega y crea un recibo inmutable;
- **reprogramación:** libera o vence la acción y fija una disponibilidad
  posterior; no crea un recibo de consumo falso;
- **cuarentena:** conserva una contradicción o resultado externo incierto para
  intervención;
- **retirada:** elimina la acción de la cola activa por una causa terminal
  persistida, no borra su historia.

Una repetición con la misma cerca puede recuperar el recibo existente. Una
cerca anterior nunca puede consumir la acción reclamada de nuevo.

## 7. Arrendamientos y cercas

### 7.1 Arrendamiento de trabajador

Protege el consumo de una acción. Contiene token, trabajador, vencimiento y
cerca. Al vencer, otro trabajador puede reclamar la misma acción con una cerca
superior. Cualquier escritura tardía del anterior se rechaza.

El reloj del solicitante sirve como dato causal, no como prueba de vigencia.
La vigencia se decide con el reloj de la transacción.

### 7.2 Arrendamiento de Director

Protege el derecho a proponer cambios a un `Goal`. Es distinto del
arrendamiento de ejecución:

- reclamarlo o renovarlo no reclama acciones;
- tomar control no modifica el plan;
- una propuesta debe presentar token y cerca vivos;
- un Director sustituido puede leer hechos históricos, pero no escribir;
- la decisión durable conserva la cerca y omite el token secreto.

La implementación de referencia está en
[`internal/application/director_state.go`](../../../internal/application/director_state.go)
y
[`internal/adapters/state/sqlite/director.go`](../../../internal/adapters/state/sqlite/director.go).

### 7.3 Otras concesiones

CID de microVM, sesión de agente, espacio de trabajo y afinidad de anfitrión pueden
necesitar concesiones propias. No deben reutilizar el arrendamiento del
Director ni crear autoridad de ciclo de vida. Cada concesión exige identidad,
recurso, titular, vencimiento, cerca y recuperación.

## 8. Máquina segura alrededor de un efecto

El lanzamiento de un agente ilustra el patrón general:

```text
Action en cola
  -> reclamación con arrendamiento y cerca
  -> transición local durable a despachando
  -> llamada al proveedor con clave idempotente
  -> recibo externo
  -> transición durable a en curso
  -> nueva Action de observación
```

Si falla antes de la llamada, se puede reprogramar. Si falla después de la
llamada pero antes de guardar el recibo, el resultado es desconocido:

1. consultar al proveedor por la identidad idempotente;
2. reconciliar si existe un hecho inequívoco;
3. poner en cuarentena si no puede saberse;
4. nunca lanzar de nuevo a ciegas.

Para detener, publicar, desplegar o integrar se aplica el mismo esquema. Una
operación reversible no justifica omitir recibos; una irreversible exige aún
más disciplina.

## 9. Eventos y recibos

### 9.1 Eventos

Un evento:

- tiene referencia estable;
- enlaza `Goal`, y cuando aplica `WorkItem` y `Execution`;
- declara tipo e instante;
- nace en la misma transacción que el cambio auditado.

Sirve para auditoría, notificación y proyecciones. No decide el estado actual.
Si falta un evento obligatorio, la transacción completa debe fallar; no se
“repara” emitiéndolo después sin enlace causal.

### 9.2 Recibos

Los recibos demuestran hechos concretos:

- autorización;
- consumo de acción;
- efecto externo;
- artefacto;
- atestación;
- integración;
- copia o restauración.

Un recibo debe ligar sujeto, intento, cerca, generación, instante y resultado
aplicables. No se reutiliza entre generaciones, árboles, binarios,
configuraciones o proveedores distintos.

El recibo de la versión se emite fuera del árbol sellado y referencia sus
resúmenes criptográficos;
no se obliga a un fichero a contener la huella criptográfica del árbol que lo
contiene.

## 10. Reinicio y repetición

### 10.1 Arranque

Al arrancar:

1. cargar configuración efectiva redactada;
2. abrir exactamente un repositorio;
3. aplicar o verificar migraciones autorizadas;
4. rehidratar instantáneas y validar invariantes;
5. comprobar coherencia de eventos, bandeja, recibos y cercas;
6. abrir adaptadores y planificador;
7. publicar disponibilidad solo al final.

No se reconstruye el estado mirando procesos vivos. Un proceso sin
`ExecutionRef`, sesión, cerca y recibo compatibles es un hecho externo por
reconciliar.

### 10.2 Recuperación de acciones

Después de una caída:

- acción no reclamada: vuelve a ser candidata;
- reclamación no vencida: se respeta;
- reclamación vencida: puede obtener una cerca superior;
- acción consumida: su recibo impide otra entrega;
- ejecución terminal: nunca se vuelve a ejecutar;
- ejecución despachando (`dispatching`): exige reconciliación del posible
  efecto;
- espera por capacidad: debe conservar la ejecución sin gastar un intento
  (diseño del entorno elástico, aún no acreditado);
- Director vencido: permite toma de control sin perder plan.

El contrato V06 prueba reinicio, repetición y rechazo de cercas antiguas. La
recuperación elástica de entornos de agentes requiere además su contrato
físico; no queda acreditada por restaurar SQLite.

### 10.3 Repetición

“Repetir” puede significar:

- repetir un comando idempotente;
- volver a entregar una acción tras vencer;
- reconstruir una proyección desde hechos;
- restaurar una copia;
- crear un sucesor causal.

No son equivalentes. En particular, restaurar no autoriza repetir efectos ni
crear otra generación.

## 11. Copia de seguridad verificable

### 11.1 Frontera neutral

El puerto
[`StateRecovery`](../../../internal/application/recovery.go) expone tres
operaciones:

- crear copia;
- verificar copia;
- restaurar en un destino nuevo.

Las referencias son opacas y direccionadas por contenido. Las rutas físicas
pertenecen al adaptador.

### 11.2 Copia SQLite de referencia

El adaptador actual:

1. valida que raíces de copia, restauración y base activa no se solapen;
2. exige directorios privados y bloqueados por identidad;
3. crea un área temporal exclusiva;
4. usa la API de copia en línea de SQLite;
5. sincroniza el contenido de la copia;
6. calcula la huella criptográfica física;
7. abre la copia y valida esquema, integridad y semántica;
8. escribe un manifiesto JSON canónico;
9. sincroniza área y directorio;
10. publica sin sustituir un destino existente;
11. devuelve un recibo direccionado por contenido.

La implementación está en
[`recovery_backup.go`](../../../internal/adapters/state/sqlite/recovery_backup.go).

Una copia concurrente contiene el estado completo anterior o posterior a una
transacción; nunca una mezcla parcial.

### 11.3 Qué incluye y qué no

La copia V09 conserva el estado causal de SQLite: Goals, unidades, ejecuciones,
eventos, bandeja, reclamaciones y recibos aplicables.

Su manifiesto declara que no incluye:

- secretos del `CredentialStore`;
- objetos binarios del almacén de artefactos.

Por tanto una recuperación completa del despliegue necesita planes separados
y coordinados para:

- base transaccional;
- artefactos por contenido;
- credenciales mediante su adaptador de almacenamiento y política;
- configuración explícita no sensible;
- binario o imagen sellados.

Nunca incluir secretos “para que la copia sea completa”.

## 12. Verificación de copia

Antes de restaurar:

1. validar sintaxis y canonicalidad del manifiesto;
2. rechazar campos desconocidos o datos posteriores;
3. abrir el contenido de la copia como fichero regular privado;
4. rechazar enlaces simbólicos, enlaces duros, modos u propietarios inseguros;
5. verificar tamaño y huella criptográfica física;
6. comprobar versión y cadena de migraciones;
7. ejecutar integridad y claves foráneas;
8. comparar inventario de esquema;
9. rehidratar y validar cada agregado;
10. validar vínculos evento–bandeja–recibo–generación;
11. calcular la huella criptográfica lógica;
12. volver a comprobar identidad y huella criptográfica para detectar cambios durante la
    lectura.

La referencia está en
[`recovery_verify.go`](../../../internal/adapters/state/sqlite/recovery_verify.go)
y
[`recovery_validation.go`](../../../internal/adapters/state/sqlite/recovery_validation.go).

Verificar que SQLite abre no es suficiente. Una base íntegra físicamente puede
contener una cerca, generación o recibo causalmente imposible.

## 13. Restauración segura

La restauración debe ser fuera de línea respecto del destino:

1. inspeccionar y retener la identidad del contenido verificado;
2. exigir una referencia de destino nueva y acotada;
3. rechazar destino o área temporal preexistentes;
4. copiar a un fichero parcial exclusivo;
5. sincronizar;
6. recalcular la huella criptográfica física;
7. validar de nuevo esquema y huella criptográfica lógica;
8. cerrar el fichero;
9. publicar sin reemplazo;
10. sincronizar el directorio;
11. emitir recibo.

Si cualquier paso falla, no se publica un destino parcial. La referencia está
en
[`recovery_restore.go`](../../../internal/adapters/state/sqlite/recovery_restore.go).

Restaurar no cambia automáticamente la fuente activa. La promoción requiere
una operación gobernada:

1. verificar copia y recibo;
2. detener admisión de comandos;
3. drenar o cercar escritores;
4. cerrar limpiamente la fuente actual;
5. abrir el destino restaurado en aislamiento;
6. ejecutar comprobaciones de lectura y recuperación;
7. cambiar la composición de forma explícita;
8. arrancar una sola fuente;
9. conservar reversión y auditoría.

Nunca hacer doble escritura durante el cambio.

## 14. Migraciones

Una migración:

- tiene versión y contenido inmutables;
- avanza, no reescribe una migración ya publicada;
- se ejecuta dentro del mecanismo del adaptador;
- conserva referencias, revisiones y terminalidad;
- tiene prueba desde cada versión soportada;
- admite inspección o simulación previa cuando la operación lo requiera;
- falla antes de servir tráfico si el esquema no coincide.

La validación de recuperación debe conocer la frontera de cada versión para no
exigir a una copia antigua columnas que entonces no existían. Después de
migrar, el lector actual rehidrata el modelo y comprueba sus invariantes.

No usar migraciones para insertar hechos de negocio inventados. Si un dato
histórico no puede acreditarse, se conserva como legado explícito o bloqueo.

## 15. Matriz de fallos

| Punto de fallo | Estado recuperable esperado |
|---|---|
| antes de confirmar la transacción | ningún cambio visible |
| tras instantánea y antes de bandeja | imposible por transacción |
| tras reclamación y antes de efecto | nueva reclamación al vencer |
| tras efecto y antes de recibo | reconciliación o cuarentena |
| tras recibo y antes de siguiente acción | misma transacción conserva ambos |
| durante copia en línea | copia completa anterior o posterior |
| tras sincronizar copia y antes de publicar | temporal retirado al recuperar |
| tras publicar y antes del recibo al llamador | verificar por referencia de contenido |
| durante restauración parcial | destino final ausente |
| tras publicar destino y perder la respuesta | destino preservado; verificar y reconciliar antes de repetir |
| Director caído | toma de control con cerca superior |
| trabajador tardío | escritura rechazada por cerca |
| reloj del cliente manipulado | no altera vigencia del arrendamiento |

Cada punto debe probarse con inyección de fallo, no solo describirse.

## 16. Pruebas obligatorias

### 16.1 Contrato común de repositorio

- creación y lectura sin pérdida;
- comparación y sustitución concurrente;
- repetición exacta y divergente;
- atomicidad de instantánea, evento y bandeja;
- máximo una reclamación activa por acción;
- token de reclamación no reutilizable;
- cerca antigua rechazada;
- recuperación tras vencimiento;
- una acción consumida no vuelve a entregarse;
- terminales no reejecutados;
- aislamiento por proyecto;
- cierre y reapertura limpios.

### 16.2 Pruebas negativas

- revisión, generación, intento o sujeto alterados;
- acción de bandeja que no coincide con su ejecución;
- recibo ligado a otra acción o cerca;
- evento sin sujeto válido;
- plan terminal con acción productiva pendiente;
- reloj externo usado como autoridad;
- manifiesto no canónico, duplicado, desconocido o con datos posteriores;
- copia truncada o modificada;
- esquema con tabla, índice o disparador divergente;
- claves foráneas rotas;
- enlace simbólico o duro;
- propietario o modo inseguros;
- sustitución de raíz entre comprobación y uso;
- destino de restauración preexistente;
- cancelación en cada punto de fallo.

### 16.3 Concurrencia y reinicio

- `-race` sobre dominio, aplicación y adaptador;
- dos escritores contra la misma revisión;
- dos trabajadores reclamando;
- renovación contra toma de control;
- copia simultánea con una transacción;
- reinicio en cada estado de ejecución;
- restaurar y continuar trabajo pendiente;
- procesar trabajo pendiente no modifica un `Goal` terminal;
- apagar deja cero procesos, bloqueos y temporales propios.

Contratos de referencia:

- [aceptación V06](../../../acceptance/v06_atomic_state_outbox_test.go);
- [batería SQLite V06](../../../internal/adapters/state/sqlite/v06_state_test.go);
- [aceptación V09](../../../acceptance/v09_recovery_backup_test.go);
- [batería SQLite de recuperación](../../../internal/adapters/state/sqlite/recovery_v09_test.go);
- [pruebas de Director](../../../internal/application/director_test.go).

## 17. Secuencia de construcción desde cero

1. **Instantánea de dominio.** Definir versión, serialización neutral y
   rehidratación estricta.
2. **Puerto grueso de estado.** Modelar operaciones atómicas de aplicación.
3. **Adaptador en memoria.** Usarlo solo para el contrato y fallos lógicos; no
   acredita durabilidad.
4. **SQLite mínimo.** Migraciones, escritor dedicado, transacciones y lecturas.
5. **Revisión esperada.** Añadir conflictos deterministas.
6. **Idempotencia de comando.** Referencia, huella y respuesta durable.
7. **Eventos atómicos.** Auditoría en la misma transacción.
8. **Bandeja transaccional.** Acciones, reclamación, arrendamiento, cerca y
   recibo.
9. **Efectos gobernados.** Intención, aprobación, intento, recibo y
   reconciliación.
10. **Reinicio.** Rehidratar, recuperar vencidos y proteger terminales.
11. **Copia.** API en línea, publicación atómica y manifiesto por contenido.
12. **Verificación semántica.** Esquema, agregado, bandeja y recibos.
13. **Restauración nueva.** Nunca sobreescribir el activo.
14. **Migraciones compatibles.** Probar cada versión soportada.
15. **Adaptador alternativo.** Ejecutar la misma batería de pruebas antes de
    componerlo.
16. **Operación real.** Instalación, promoción, reversión y apagado sobre los
    mismos resúmenes criptográficos sellados.

No comenzar por replicación, varios anfitriones o máquinas virtuales. Primero
debe
existir una única verdad local correctamente cercada y recuperable.

## 18. Contrato ejecutable objetivo para V31 multianfitrión

Marcador contractual: `v31_multianfitrion_con_fencing`.

V31 es una vertical planificada, no una capacidad acreditada. En la hoja de
ruta, `OPS-11` y `OPS-13` siguen declaradas y
`AC-V31-POSTGRES-S3-MULTIHOST` sigue planificado. Esta sección convierte su
resultado esperado en operaciones y pruebas concretas; no afirma que los
adaptadores existan.

### 18.1 Un estado lógico compartido

Un despliegue con varios anfitriones conserva una sola fuente lógica de estado:

```text
nodos Orquesta A, B, C
          │
          ├── mismo contrato StateRepository
          ▼
clúster PostgreSQL, una autoridad lógica

adaptadores de artefactos
          ├── sistema de ficheros para composición local
          └── almacén compatible con S3 para composición compartida
```

Un clúster PostgreSQL puede replicar físicamente sus datos, pero presenta una
sola autoridad transaccional. Quedan prohibidas dos bases activas, la escritura
doble SQLite/PostgreSQL y la conciliación posterior como método ordinario.

### 18.2 Identidad y arrendamiento de nodo

Cada proceso de Orquesta registra una identidad de nodo distinta de la
identidad del anfitrión:

```text
referencia de nodo (`NodeRef`)
referencia de anfitrión (`HostRef`)
referencia de arranque (`BootRef`)
instante de inicio (`StartedAt`)
capacidades (`Capabilities`)
token de arrendamiento (`LeaseToken`)
cerca de arrendamiento (`LeaseFence`)
vencimiento del arrendamiento (`LeaseExpiresAt`)
```

La estructura es conceptual. El contrato definitivo debe versionarla.
`NodeRef` identifica al participante lógico, `HostRef` al equipo físico o
virtual y `BootRef` a un arranque concreto. Clonar un disco no puede clonar una
identidad de nodo vigente.

El arrendamiento de nodo:

- se crea y renueva usando el tiempo transaccional de PostgreSQL;
- recibe una cerca monotónica;
- no sustituye al arrendamiento del Director ni al de un `WorkItem`;
- deja de autorizar reclamaciones al caducar;
- no mata procesos o agentes por el mero hecho de caducar;
- se invalida al restaurar una copia en otro despliegue.

### 18.3 Transacción PostgreSQL

El adaptador PostgreSQL debe pasar la misma batería de pruebas de repositorio
que SQLite y
conservar en una transacción:

```text
comprobar revisión esperada
comprobar proyecto y generación
comprobar arrendamiento y cerca
escribir instantánea nueva
añadir evento de auditoría
añadir acción a la bandeja de salida
confirmar una sola vez
```

La selección de una acción utiliza bloqueo transaccional o comparación y
sustitución equivalente. Dos nodos que reclaman el mismo trabajo obtienen un
único ganador. Un bloqueo de fila no sustituye a la cerca: la cerca protege
también frente a un propietario antiguo que reaparece después de una
partición.

Migraciones, conjunto de conexiones, dirección de conexión y salud pertenecen
al adaptador y a la composición. El dominio no contiene SQL ni decisiones
específicas de PostgreSQL.

### 18.4 Reclamaciones y cercas distribuidas

Una reclamación distribuida sigue esta secuencia:

```text
reclamar(trabajo, nodo, arrendamiento)
  comenzar transacción
  usar hora de PostgreSQL
  verificar que el trabajo está listo
  verificar que no existe reclamación vigente
  incrementar la cerca
  guardar referencia de nodo (`NodeRef`), referencia de anfitrión (`HostRef`),
          token de arrendamiento (`LeaseToken`), cerca (`LeaseFence`) y caducidad
  confirmar
  devolver la reclamación durable
```

Toda escritura posterior presenta `LeaseToken` y `LeaseFence`. PostgreSQL
rechaza:

- una cerca menor que la persistida;
- un token distinto;
- un nodo o anfitrión distinto;
- una generación obsoleta;
- un arrendamiento caducado;
- un proyecto diferente.

La repetición exacta devuelve la misma reclamación. La misma clave con otro
sujeto es conflicto.

### 18.5 Particiones, red y relojes

El reloj local sirve para métricas, nunca para conceder autoridad. Las
caducidades se comparan con la hora de PostgreSQL. Las pruebas introducen
desviaciones de reloj hacia delante y hacia atrás sin producir dos
propietarios.

Si un nodo pierde comunicación con PostgreSQL:

1. deja de reclamar y renovar;
2. no confirma transiciones en memoria;
3. conserva diagnósticos y resultados locales como no admitidos;
4. permite que sus arrendamientos caduquen;
5. al recuperar red, relee cercas antes de escribir.

Si conserva comunicación con agentes pero no con estado, puede observarlos
para diagnóstico, pero no atribuirles progreso durable. Si conserva
PostgreSQL pero pierde el almacén de objetos, persiste la intención de
transferencia y reintenta; nunca publica una referencia de artefacto sin
recibo verificable.

### 18.6 Artefactos compatibles con S3

El adaptador compartido conserva el contrato del sistema de ficheros:

- dirección por contenido;
- escritura inmutable o condicional;
- tamaño, tipo y huella criptográfica;
- aislamiento de proyecto;
- verificación después de escribir;
- lectura por referencia opaca;
- rechazo de contenido manipulado;
- recibo de almacenamiento.

PostgreSQL y S3 no comparten una transacción. La aplicación usa una máquina de
efecto explícita:

```text
transacción 1: intención de almacenar + acción de salida
efecto: escritura condicional en S3 con clave idempotente
transacción 2: recibo verificado + referencia de artefacto admitida
```

Un tiempo límite después de escribir se reconcilia consultando la clave y la
huella esperadas. No se repite con otro nombre ni se inventa el recibo.

### 18.7 Adopción y recuperación de agentes por anfitrión

Un agente puede sobrevivir a la caída del nodo que lo lanzó. El estado durable
de su ejecución conserva:

- `ExecutionRef` e intento;
- `NodeRef` y `HostRef` de lanzamiento;
- referencia externa del proveedor;
- recibo de lanzamiento;
- espacio de trabajo y artefactos por referencia;
- último estado observado;
- arrendamiento y cerca.

El reconciliador de recuperación:

```text
obtener reclamación nueva con una cerca mayor
resolver el anfitrión y la referencia externa exactos
observar el agente sin atribuir éxito
si identidad y sujeto coinciden:
    adoptar la observación bajo la cerca nueva
si el agente terminó:
    recoger y verificar resultado
si sigue vivo pero no puede adoptarse:
    solicitar parada exacta autorizada y sellar
si no puede demostrarse su identidad:
    poner en cuarentena; no arrancar un duplicado
```

Si se pierde el anfitrión completo, otro nodo solo relanza después de que la
cerca antigua quede invalidada y la política determine que no existe un efecto
físico recuperable. El nuevo espacio de trabajo se reconstruye desde Git,
almacén de contenido y referencias durables; nunca desde una ruta local
inferida.

### 18.8 Conmutación por fallo

La conmutación de PostgreSQL mantiene la misma identidad lógica del
repositorio. Los clientes:

1. pierden la conexión;
2. descartan transacciones incompletas;
3. resuelven el nuevo primario mediante la composición;
4. reintentan con la misma identidad idempotente;
5. releen revisión, arrendamiento y cerca;
6. aceptan el recibo anterior o compiten de nuevo.

No se promueve una copia de seguridad mientras la fuente anterior siga
admitiendo escrituras. La restauración crea un destino inactivo, invalida
arrendamientos y exige una promoción explícita con un solo escritor.

### 18.9 Pruebas de aceptación

La vertical necesita, como mínimo:

- la misma batería contractual completa sobre SQLite y PostgreSQL;
- la misma batería de artefactos sobre sistema de ficheros y S3 compatible;
- dos nodos reclamando simultáneamente un único `WorkItem`;
- propietario antiguo escribiendo después de una cerca nueva;
- partición entre nodo y PostgreSQL;
- partición entre nodo, agente y almacén de objetos;
- desviación de relojes locales;
- caída del nodo antes y después de lanzar un agente;
- adopción de un agente vivo en el mismo anfitrión;
- pérdida total del anfitrión y relanzamiento gobernado;
- conmutación del primario durante cada frontera transaccional;
- tiempo límite de S3 antes y después de almacenar;
- aislamiento negativo entre proyectos y propietarios;
- carga, contrapresión y carreras con varios nodos;
- copia, restauración y promoción sin doble escritor;
- migración SQLite a PostgreSQL sin escritura doble;
- reinicio repetido con cero ejecución o efecto duplicados.

El extremo a extremo debe ejecutar procesos o contenedores realmente separados,
PostgreSQL y un servicio S3 compatible temporales, inyectar las caídas y
demostrar recibos del mismo candidato. Dobles en memoria acreditan contratos
locales, no el comportamiento multianfitrión.

## 19. Antipatrones que invalidan la recuperación

- usar el registro de eventos como segunda base autoritativa;
- separar instantánea y bandeja en confirmaciones distintas;
- aceptar la hora del trabajador para validar un arrendamiento;
- reintentar un efecto de resultado desconocido sin reconciliar;
- borrar acciones, intentos o entornos antes de revisar sus pruebas;
- copiar una base activa con una copia de fichero no coordinada;
- considerar válida una copia porque su huella criptográfica física coincide;
- restaurar sobre un destino existente;
- compartir raíces de base activa, copias y restauraciones;
- incluir secretos u objetos binarios sin declarar su frontera;
- mantener dos fuentes activas durante una migración;
- ejecutar migraciones de negocio que inventen recibos;
- declarar recuperación completa tras restaurar solo SQLite;
- atribuir a V09 la recuperación de agentes o microVM no acreditada.

## 20. Procedimiento operativo abreviado

### Antes de una operación

```text
identificar versión, revisión Git, árbol Git, binario y configuración
comprobar fuente activa única
comprobar espacio y raíces privadas
inventariar escritores, acciones, arrendamientos y efectos inciertos
crear la referencia de operación y la identidad del operador
```

### Copia

```text
CreateBackup (crear copia)
VerifyBackup (verificar copia)
guardar el recibo fuera del sujeto
copiar o verificar por separado los artefactos y las credenciales
registrar resúmenes criptográficos y política de retención
```

### Restauración

```text
VerifyBackup (verificar copia)
RestoreBackup (restaurar copia) a un destino nuevo
abrir destino sin tráfico
validar estado lógico y trabajo pendiente
probar cercas obsoletas y terminales
promover con un solo escritor
conservar origen para reversión
```

### Cierre

```text
sin temporales propios
sin bloqueos propios
sin procesos propios
sin doble escritor
recibo durable y enlazado
riesgos y siguientes dependencias documentados
```

## 21. Lista de salida del capítulo

```text
un solo repositorio de estado activo por despliegue
una sola autoridad de instantáneas
instantánea, evento y bandeja se confirman atómicamente
toda mutación usa la revisión esperada
toda escritura reclamada usa una cerca vigente
el arrendamiento del Director es distinto del de una acción
la repetición exacta es idempotente
la repetición divergente se rechaza
los efectos externos inciertos se reconcilian o ponen en cuarentena
el trabajo terminal no se reejecuta tras reiniciar
la copia se direcciona por contenido y se verifica semánticamente
la restauración nunca sobrescribe un destino existente
secretos y objetos binarios de artefactos tienen planes de recuperación separados
```

Solo una ejecución de los contratos aplicables sobre un candidato inmutable
puede convertir estas condiciones en evidencia. Este capítulo sigue siendo una
guía, no un recibo.
