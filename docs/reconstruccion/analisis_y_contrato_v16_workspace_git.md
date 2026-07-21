# V16: análisis y contrato de workspace y Git local

Fecha de decisión: 2026-07-18. Checkpoint de implementación: 2026-07-21.

Estado del checkpoint: análisis e implementación completos en el worktree;
sellado P/S/E y receipt V3 pendientes. Esta implementación no acredita V16 por
sí sola: V16 cuenta cerrado únicamente cuando
`TestAcceptanceV16WorkspaceGitReceipt` valida el receipt V3 `PASS` generado por
el argv exacto desde el candidato S en checkout `detached_clean`.

## 1. Decisión

V16 será deliberadamente pequeño. Acreditará solo:

- `STG-02`: descubrimiento e inventario del workspace;
- `STG-10`: preparación de entorno y worktree;
- `EXT-10`: Git local y worktrees.

`EXT-11` —GitHub, GitLab y Gitea— se conserva, pero pasa a V28. Allí se
implementará detrás de un único puerto neutral `Forge`, cuando ya existan
CredentialStore, registro de comandos, tools/plugins, permisos y un consumidor
real. V16 no hará push, pull request ni merge remoto.

Esta separación corrige un defecto del roadmap anterior: el contrato V16 solo
probaba Git local, pero pretendía acreditar tres APIs remotas. Hacerlo habría
producido otro falso verde o tres adapters prematuros sin consumidor.

## 2. Resultado funcional exacto

Al cerrar V16, un WorkItem con `WriteSet` no vacío podrá:

1. obtener un workspace privado ligado a su `ExecutionRef` y a un OID base
   exacto;
2. lanzar el agente dentro de ese workspace, no en el checkout canónico;
3. inventariar y rechazar cambios fuera del write-set;
4. producir un commit causal y un receipt inmutable;
5. intentar una integración local contra un target exacto;
6. conservar como pendiente un conflicto, una base stale o el perdedor de una
   carrera, sin modificar la rama destino;
7. consultar ese trabajo pendiente con scope RBAC por actor y proyecto, incluso
   después de reiniciar.

Un WorkItem sin `WriteSet` conserva el camino actual: no se inventa un workspace
ni se interpreta texto para decidir si el trabajo «parece código».

V16 aporta la primitiva de integración. V18 aporta la política que exige autor,
reviews y refinery antes de usarla. Un commit local no cierra por sí mismo un
Goal ni autoriza una integración.

## 3. Lo que no cambia

- `Goal` y `WorkItem` siguen siendo el único lifecycle.
- `application.Orchestrator` sigue siendo el único writer.
- `StateRepository` y SQLite siguen siendo la única autoridad durable.
- Se reutilizan el outbox, scheduler, claim, lease y fence de V06.
- Se reutiliza `EffectIntent -> EffectApproval -> EffectAttempt ->
  EffectReceipt` de V15.
- `ProjectRef`, `RepositoryRef`, principal, membresía y RBAC proceden de V10.
- `WriteSet`, `ReworkOf`, generación y `ExecutionRef` proceden de V05/V06.
- stop/cancel de V14 no borra el workspace ni oculta trabajo parcial.

No se crean `WorkspaceStore`, `GitDB`, scheduler Git, daemon, task model,
director, cola, lock service ni lifecycle paralelos.

```text
Goal/WorkItem ready
        |
        v
mismo outbox: prepare_workspace
        |
        v
WorkspaceManager ----> Git CLI local
        |
        v
WorkspaceBinding + receipt en StateRepository
        |
        v
mismo outbox: launch_agent(workspace_ref)
        |
        v
mismo outbox: commit_change
        |
        v
ChangeSet + receipt ----> integrate_change explícito
                              |
                              v
                    merge-tree + update-ref CAS
```

## 4. Hechos mínimos

No hace falta un nuevo agregado. Bastan cuatro hechos inmutables y proyecciones
derivadas.

### `WorkspaceBinding`

Une un `ExecutionWorkspaceRef` opaco con:

- principal, actor, proyecto y `RepositoryRef` lógico;
- Goal, WorkItem, Execution, intento y generaciones exactas;
- hash de AppSpec y digest del write-set;
- target ref opaca, OID base y formato de objetos Git;
- host/adapter ref opaca, intent, attempt, fence y receipt;
- fecha de preparación.

La organización V10 ya tiene `identity.WorkspaceRef`. El workspace físico de
ejecución usará `ExecutionWorkspaceRef`; no se reutilizará el mismo nombre para
dos conceptos.

Una ejecución tiene como máximo un binding. Un retry del mismo efecto devuelve
el mismo binding. Una ejecución sustituta recibe otro workspace. La ruta física
nunca entra en dominio, snapshot, API, receipt ni Goal.

### `ChangeSet`

Une un commit local con el binding y conserva:

- base, parent, head y tree OID exactos;
- digest del diff, paths cambiados y write-set;
- `parent_change_ref` cuando procede de rework explícito;
- identidad causal del agente y de la ejecución;
- intent, attempt, fence, idempotency key y receipt.

V16 permite un ChangeSet autoritativo por ejecución. Si hace falta rework se
crea una nueva ejecución y se señala el ChangeSet padre; no se reescribe el
anterior.

### `MergeObservation`

Registra el cálculo previo sin efecto:

- source OID y target OID observado;
- clean, conflicted o stale;
- tree candidato o evidencia compacta del conflicto;
- cero mutación del target.

### `IntegrationReceipt`

Solo existe después de un efecto aplicado o de un resultado estructuralmente
no aplicado. Conserva target before/after, source, tree, marker ref opaca,
effect receipt y estado `integrated`, `conflicted` o `stale`.

`pending work` no es otro estado autoritativo: es la consulta de ChangeSets sin
IntegrationReceipt aplicado, más conflictos/stale no resueltos, filtrada por
RBAC, proyecto y actor.

## 5. Puertos mínimos

### `WorkspaceManager`

- `Prepare`: idempotente; materializa un worktree exacto y devuelve solo refs,
  OIDs, digests y versión del adapter.
- `Inspect`: lectura acotada y parseable; no muta Git.
- `Release`: explícito y seguro; no fuerza la retirada de un workspace sucio o
  aún referenciado.

### `VersionControl`

- `Commit`: recibe binding, base, write-set, identidad causal e idempotency key;
  solo actualiza la ref aislada de esa ejecución.
- `PreviewIntegration`: calcula clean/conflict sin modificar worktree, índice o
  target.
- `Integrate`: exige source exacto y expected target OID; devuelve receipt
  estructurado.

El core nunca recibe path, URL, argv, variable de entorno, nombre de proveedor,
token ni salida humana de Git. Los adapters no deciden lifecycle ni cierre.

`Forge` no se añade en V16 «por si acaso». V28 definirá el puerto al introducir
el primer adapter real y ejecutará la misma suite contractual contra GitHub,
GitLab y Gitea.

## 6. Un solo ejecutor y un solo ledger

Se amplían de forma focal los tipos V15 con tres operaciones:

- `prepare_workspace`;
- `commit_change`;
- `integrate_change`.

Cada una entra por el mismo outbox y sigue la cadena:

```text
intent persistido
  -> approval exacta
  -> claim/lease/fence
  -> attempt persistido
  -> llamada al adapter
  -> receipt persistido
  -> ACK/consumo del mismo action
```

Preparar y commitear heredan la autoridad ya conservada para el WorkItem.
Integrar exige una decisión explícita y el permiso `changes.integrate`; un
contributor no lo obtiene por crear el Goal. V18 decidirá cuándo las reviews
permiten emitir esa aprobación.

No se mantiene una transacción SQLite abierta mientras Git trabaja. El fence
protege el intento y el OID esperado protege la ref. No hace falta un mutex
global ni un segundo lease por repositorio.

## 7. Estrategia del adapter Git local

Se usará el ejecutable Git instalado, mediante argv exacto y
`exec.CommandContext`. No se añadirá `go-git`, libgit2 ni un motor de merge.

Decisiones concretas:

- repositorio autorizado por `RepositoryRef` mediante un resolver inyectado;
  un Goal nunca aporta un path o remote URL;
- root privado de Orquesta y subdirectorio derivado de un ID opaco, no de texto
  de usuario;
- branch/ref única por ejecución y OID base explícito, nunca `HEAD` implícito;
- creación con refs auxiliares de Orquesta y `git worktree add --lock`;
- inventario con formatos machine-readable terminados en NUL, en particular
  `status --porcelain=v2 -z` y `worktree list --porcelain -z`;
- cálculo de integración con `git merge-tree --write-tree`, que no toca índice
  ni worktree;
- creación determinista del commit candidato y actualización del target con
  `git update-ref` usando expected old OID;
- target y `refs/orquesta/effects/<intent-digest>` se actualizan en la misma
  transacción de refs Git. Ese marker permite reconciliar un crash posterior al
  efecto y anterior al receipt SQLite;
- nunca se ejecuta `git merge` sobre el checkout principal;
- nunca se hace push implícito;
- un worktree sucio no se elimina con `--force`.

Git conserva objetos y refs físicos. Orquesta conserva autoridad, causalidad,
permisos y receipts. Ninguno sustituye al otro.

Referencias primarias consultadas:

- [git-worktree](https://git-scm.com/docs/git-worktree): linked worktrees,
  locking y formato porcelain;
- [git-merge-tree](https://git-scm.com/docs/git-merge-tree.html): merge real sin
  modificar worktree ni índice;
- [git-update-ref](https://git-scm.com/docs/git-update-ref.html): compare-and-swap
  y transacciones de refs;
- [git-status](https://git-scm.com/docs/git-status): formato porcelain estable y
  terminación NUL.

## 8. Crash, replay e idempotencia

El adapter debe reconciliar estas fronteras:

1. intent persistido, sin refs/worktree;
2. branch/marker creado, sin worktree;
3. worktree creado, sin receipt SQLite;
4. objeto commit creado, sin actualización de ref;
5. branch o target y marker actualizados, sin receipt SQLite;
6. release parcial.

Un replay con igual idempotency key y payload devuelve el mismo resultado. La
misma key con payload distinto falla. Un marker, branch o binding inesperado se
cuarentena y conserva para revisión; nunca se corrige con force/reset/borrado.

Los commits creados por Orquesta fijan parent, tree, autor lógico y tiempos
desde el intent persistido para que el retry no fabrique otro OID.

## 9. Concurrencia y multiusuario

- Dos ejecuciones, incluso del mismo usuario y proyecto, usan worktrees y refs
  diferentes y pueden avanzar en paralelo.
- El aislamiento de lectura/autorización procede de V10; listar pendientes de
  otro proyecto produce cero filas, no una lista filtrada después.
- Dos integraciones contra el mismo target pueden calcular en paralelo. Solo
  una gana el `update-ref` con expected old OID; la otra queda `stale` y conserva
  su ChangeSet.
- Un conflicto produce evidencia y no muta la rama destino.
- Los write-sets siguen evitando conflictos previsibles en el DAG. Git CAS es
  la defensa final entre Goals, hosts o cambios externos; Git no sustituye el
  CAS del Goal ni los fences del outbox.
- V31 cambia SQLite/FS por PostgreSQL/S3 y añade afinidad/recuperación multihost;
  no cambia estos contratos.

## 10. Seguridad proporcional

V16 asegura el control Git local; V17/V22 aportarán atestación y sandbox contra
un proceso hostil completo.

Obligatorio en V16:

- roots y metadata de control privados `0700`, owner exacto y ancestros no
  escribibles por grupo ni terceros;
- ninguna ruta derivada de actor, objetivo, branch solicitada o nombre de
  fichero sin normalización estructural;
- `Lstat`/`open` con `NOFOLLOW` y apertura no bloqueante: traversal, ancestro
  symlink, hardlink, FIFO, device, owner ajeno y modificación de `.git` fallan
  antes de mutar refs;
- el `git-dir` y `common-dir` resueltos pertenecen al repositorio autorizado,
  no contienen symlinks, conservan owner/modo/ancestros privados y no pueden
  sustituirse por metadata de otro checkout;
- un symlink Git legítimo solo puede tratarse como leaf sin seguir su destino;
- write-set repository-relative, sin globs dentro del dominio y con pathspecs
  terminados en NUL;
- entorno hijo construido desde allowlist; se eliminan `GIT_DIR`,
  `GIT_WORK_TREE`, `GIT_INDEX_FILE`, object directories y alternates;
- prompts, pager, credential helper, hooks, fsmonitor, external diff, filtros
  ejecutables y firma externa quedan desactivados o hacen fallar el preflight;
- ningún secreto, path absoluto, remote URL, argv o entorno entra en receipts o
  diagnósticos públicos;
- el adapter vuelve a inventariar antes de commit/update-ref y liga el receipt
  al tree final exacto.

Límite honesto: otro proceso malicioso con el mismo UID todavía puede competir
en el filesystem. V16 detecta y falla cerrado en las fronteras observables, pero
no promete confinamiento. La ejecución non-root/sandbox y la atestación contra
el tree exacto pertenecen a V17/V22.

## 11. Configuración

No se crea una variable por operación ni por repositorio. Si el wiring real
necesita una raíz, habrá una sola clave `workspace.local.root`, añadida primero
al registro canónico, a `effective_config` y al validador de roots disjuntos.

Los repositorios se identifican por `RepositoryRef` y un binding de conector
administrado; no por variables de entorno dinámicas, prefijos, paths dentro del
Goal ni aliases. Los tests inyectan roots y resolvers temporales sin añadir
configuración de producción ficticia.

## 12. Lecciones históricas que pasan a invariantes

| Incidencia heredada | Invariante V16 |
|---|---|
| goals disjuntos compartieron worktree | binding único por Execution y negativos de contaminación cruzada |
| Codex arrancó en checkout canónico | launch exige el binding exacto y el adapter resuelve su path |
| agente intentó poseer índice/commit | agente solo escribe ficheros; VCS lo gobierna Orquesta |
| integración inferida por `clean/promoted` | solo `IntegrationReceipt` con target before/after acredita integración |
| push sin destino explícito | V16 no hace push; V28 exigirá target/permiso/credential ref |
| detached HEAD y branch stale | base y target son OIDs/ref exactos, nunca heurística de HEAD |
| servidor sobrevivió a worktree borrado | binding y proceso se verifican antes de release; V32 automatiza retención |
| limpieza borraba rutas referenciadas | dirty/referenced nunca se borra por patrón ni con force |
| workspace `0775` y ancestros inseguros | root privado, owner/mode/ancestros validados |
| cache o entorno heredado read-only | entorno hijo canónico y roots aislados; caches generales siguen en V26/V32 |
| preflight mutó `go.sum` | inventario/preflight es read-only; tests pertenecen al atestador V17 |

No se porta `modulos/orquesta-runtime-worktree`. Su código histórico sirve para
caracterización, pero mezcla `os.Environ`, HEAD implícito, parsing frágil,
locks locales, staging/commit y push. V16 se reimplementa contra estos contratos.

## 13. Contrato de aceptación

El primer test se creó exactamente como `TestAcceptanceV16WorkspaceGit`; el
comando no usa el patrón genérico `^TestAcceptance$`, que puede quedar verde
sin ejecutar pruebas. Este apartado conserva el contrato que gobernó la
implementación; no sustituye el receipt final.

La aceptación cubrirá como mínimo:

1. prepare idempotente y un workspace por Execution;
2. reemplazo de Execution sin reutilización;
3. launch dentro del binding exacto;
4. commit ligado a base, diff, tree, write-set y ejecución;
5. cambio fuera de write-set con cero mutación Git;
6. rework solo desde `parent_change_ref` explícito;
7. conflicto y expected target stale con target intacto;
8. dos integraciones concurrentes: un CAS ganador, un pendiente preservado;
9. crash/replay antes y después de cada frontera externa;
10. pending work aislado por actor/proyecto/RBAC después de restart;
11. negativos traversal, symlink ancestor, hardlink, special file, owner, mode,
    `.git`, hooks, filtros, pager y credential helper;
12. leak scan de paths, URL, argv, entorno y secretos;
13. guard de arquitectura: cero store, queue, scheduler, DB o Goal lifecycle
    nuevo;
14. E2E con Git temporal real, SQLite real, dos worktrees, commits, integración,
    conflicto/stale, restart y consulta pendiente;
15. `-race` focal para prepare/commit/integrate concurrentes.
16. misma idempotency key con payload semántico distinto falla; `AttemptRef` y
    fence pueden avanzar únicamente como envelope de reintento del mismo intent;
17. prepare fija la base antes del claim, commit no adopta un hijo arbitrario,
    el perdedor CAS revalida el marker exacto y release reconcilia el crash
    posterior a la eliminación física;
18. los tests de CAS alcanzan de forma determinista la frontera `update-ref`, y
    los controles `.git` cubren special files y `common-dir` ajeno/inseguro.

El fixture declarará Git mínimo, formato de objetos, base/target OID, dos
actores, dos proyectos, write-sets, cambios clean/conflict/out-of-scope y puntos
de crash. No dependerá del checkout del propio rebuild.

## 14. Orden de implementación ejecutada

1. Commit documental de esta decisión y movimiento de `EXT-11`.
2. Test de aceptación exacto rojo + fixture; verificar que no hay falso verde.
3. Tipos/contratos neutrales y permisos focales.
4. Extensión del mismo StateRepository/outbox/effect ledger y migración SQLite.
5. Fake contractual de workspace/VCS.
6. Adapter Git CLI local y suite de seguridad/replay.
7. Wiring de composición y `workspace_ref` en launch Codex.
8. E2E Git+SQLite, restart, concurrencia y race.
9. Contrarrevisión independiente de los bloqueos iniciales.
10. Receipt V3 reproducible desde candidato detached y limpio.

El receipt ya valida y V16 cuenta: 59 de 257 capacidades y 16 de 34
verticales, con 16/16 receipts. `EXT-11` sigue pendiente y no se contará hasta
V28.

## 15. Implementación realizada y cierre acreditado

El checkpoint 2026-07-22 acredita el alcance local completo sin abrir V17:

- contratos neutrales y validación estructural en `internal/ports`;
- modelos, casos de uso y procesamiento por el único `application.Orchestrator`;
- persistencia, migración 011 y recovery adversarial en el mismo
  `StateRepository` SQLite;
- adapter Git CLI local en `internal/adapters/workspace/gitlocal`, sin forge ni
  motor Git embebido;
- wiring opt-in de bootstrap y resolución opaca del workspace para Codex;
- claves canónicas `workspace.local.root`, `repository.local.seed_path` y
  `repository.local.target_ref`, con validación de roots disjuntos;
- aceptación estructural, E2E Git+SQLite real, crash/replay, RBAC, seguridad,
  concurrencia y `-race` focal.

La contrarrevisión hizo visible que el presupuesto rojo inicial infravaloraba
las pruebas adversariales y la reconciliación durable: cabía el camino feliz,
pero no `common-dir` hostil, CAS forzado en la frontera real, release después
de borrado ni replay por objeto determinista. No se ocultaron esos tests para
mantener una cifra. El ratchet final queda fijo en neto base→S: core 2050,
adaptadores 3650, migración 650, tests 5600 y producción total 6250. La
compensación estructural fue retirar el mutex global, separar responsabilidades
y dejar todos los ficheros V16 nuevos bajo 400 líneas y funciones bajo 80; una
ampliación posterior exige otra vertical y su propio presupuesto.

Los gates focales ejecutados sobre S incluyen, entre otros,
`TestRealGitSQLiteWorkspaceLifecycleEndToEnd`,
`TestWorkspaceEffectsReplayEveryCrashFrontierExactlyOnce`,
`TestSQLiteWorkspaceGitRestartRaceAndReplay` y
`TestRecoveryV16RejectsWorkspaceCausalTampering`, además de los negativos de
replay, CAS, release y `common-dir` inventariados en los bugs 247–254. Los bugs
255–257 conservan además las lecciones del sellado y su entorno de ejecución.

Cadena final: P=`b48162b0433dd32b6369ee324358e5f87af325ad`,
S=`a4f602ab01c4e77f0d876c79c2c8e86b44b68fa4` y
E=`38e1ffb82d6f71610ac0a745d693d52fd43dac22`. El receipt V3 liga 122 sujetos,
argv literal, checkout `detached_clean`, 22 carreras, vet y salida capturada.
V17 permanece sin iniciar hasta una próxima sesión autorizada.
