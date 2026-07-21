# Ruta canónica hasta Orquesta total

Fecha: 2026-07-21

Estado: autoridad de ejecución del rebuild. Esta ruta define el trabajo; no
afirma que el producto total esté terminado.

## 1. Mandato y fuentes congeladas

Objetivo: completar todas las capacidades aceptadas de Orquesta mediante un
solo núcleo, sin adaptar el runtime antiguo ni volver a crear generaciones,
stores, loops o catálogos paralelos.

Fuentes del estudio, verificadas por contenido:

| Fuente | SHA-256 |
|---|---|
| `catalogo_decision_capacidades_orquesta_2026-07-14.md` | `351c2258562424fc5c0dab9e0dcce0b623f9eabf42f2485b6ae45432694d4fd3` |
| `ruta_reconstruccion_estructural_orquesta_2026-07-14.md` | `483d947532d81841fa9caf58a176400552d2ea5482e8f93be375c07e4ae62b73` |
| `corte_auditoria_orquesta_100_2026-07-13.md` | `852568908c1754004f2e0acc04bf40e272e7b22a94192bedac04af6222423e99` |
| orden original `orden_auditoria_integral_diseno_y_bugs_2026-07-13.md` | `252a84fdf5c68e61def6ae7d1e44d1039c1ca9297385af271d3fda7aed145d13` |

Estas fuentes explican decisiones y experiencia. Después de integrarlas, el
estado operativo vive solo en los manifests/ledgers de `product/`, ligado a
commit, binario, configuración y receipts. Modificar Markdown no acredita una
capacidad.

El árbol antiguo `/home/alberto/Trabajo/orquesta` queda siempre en solo
lectura. Las superficies antiguas presentes en este worktree quedan en solo
lectura hasta V34; únicamente el write-set acreditado de cutover puede
retirarlas de la nueva release. Se consultan para extraer semántica,
caracterizar bugs y demostrar retirada; no se importan, cablean, ejecutan ni
amplían.

## 2. Qué significa “total al 100 %”

El catálogo exhaustivo contiene 257 IDs canónicos:

| Familia | IDs |
|---|---:|
| Gobierno `GOV` | 22 |
| Wizard/entrada `WIZ` | 25 |
| Etapas `STG` | 21 |
| Orquestación `ORC` | 29 |
| Evidencia `EVD` | 15 |
| Interfaces `UI` | 18 |
| Agentes `AGT` | 12 |
| Extensiones `EXT` | 23 |
| OPES `OPE` | 20 |
| Operación `OPS` | 30 |
| Tools/skills `TLS` | 14 |
| Contexto `CTX` | 12 |
| Apps generadas `APP` | 16 |
| **Total** | **257** |

Cada ID debe tener, como mínimo:

```text
decision
kind
release_target
cutover_required
owner_context
dependencies
acceptance_contracts
status: declared | implemented | wired | exercised | accredited
evidence_refs
supersedes o alias_of, cuando aplique
```

“Total” no obliga a implementar una opción que el operador haya rechazado. Sí
obliga a que cada ID esté clasificado y a que cada capacidad aceptada para el
producto total quede `accredited`. `deferred` significa trabajo futuro y no
cuenta como producto total; `rejected` exige decisión explícita y conserva
trazabilidad. Una dependencia externa no disponible se registra como bloqueo,
no como verde.

El manifest actual del corte mínimo usa IDs locales y 33 bloques diferidos. Su
evidencia es reutilizable, pero ningún ID local se suma automáticamente al
catálogo canónico. La Vertical 01 debe mapear cada contrato mínimo a IDs exactos
y volver a acreditar lo que corresponda sobre la release total.

## 3. Resultado arquitectónico único

```text
HTTP | MCP | CLI | Web
          |
Command/Query Registry
          |
Application Engine                 único escritor
          |
IntentManifest -> AppSpec -> Goal -> WorkItem DAG
                              |         |
                         decisiones   ejecuciones
                              |         |
                         evidencia <- receipts
          |
puertos: state, artifact, credentials, identity, workspace, VCS,
         agents, tools, context, domains, effects, notifications, telemetry
          |
adaptadores elegidos en bootstrap
```

Invariantes finales:

- `Goal` es la única autoridad de lifecycle; WorkItems forman su DAG.
- `PhaseInstance` es causal e inmutable; progreso y timeline son proyecciones.
- un motor determinista valida y escribe; Director y agentes proponen o
  ejecutan mediante protocolo común.
- snapshot, evento auditivo y outbox cambian en una transacción.
- una sola fuente de estado está activa por despliegue.
- todos los elementos sustituibles, desde DB hasta Hermes, son adaptadores.
- un binario productivo `cmd/orquesta`; bootstrap no contiene política.
- un registro de configuración y uno de comandos.
- ningún bridge, import, fallback o dual write al runtime antiguo.
- monolito modular por defecto; procesos externos solo por frontera real.

## 4. Fusiones: varias filas, una implementación

Los IDs siguientes permanecen visibles en el ledger, pero comparten vertical,
contrato o evidencia. No justifican código duplicado:

| IDs relacionados | Implementación única |
|---|---|
| `WIZ-14`, `UI-09` | opción de cliente desktop; wrapper fino de web si se acepta |
| `ORC-25`, `OPS-18` | watchdog de CPU/progreso operativo, nunca rail de contenido |
| `UI-07`, `UI-15`, `UI-16`, `EXT-12`, `EXT-13` | hooks y puerto de notificación con sinks email/Telegram |
| `EXT-14`, `OPS-23`, `OPS-24` | familia de efectos deploy y targets por plugins |
| `GOV-11..14`, `STG-08`, `EVD-07` | bounded context de Consejo/Decision |
| `GOV-12`, `STG-14`, `EVD-06` | autor, review primaria y adversarial independientes |
| `OPS-01`, `OPS-26` | registro canónico de configuración |
| `OPS-06`, `OPS-28` | guard arquitectónico de env/keys/defaults |
| `EVD-15`, `OPS-14` | backup, restore y migraciones verificadas |
| `UI-04`, `UI-12` | misma web administrativa y sus vistas |
| `ORC-19..22`, `CTX-01..12`, `EXT-08` | broker de contexto, retrieval, memoria y handoff |
| `EXT-09`, `TLS-01`, `TLS-05` | registro y SDK canónicos de tools |
| `GOV-09`, `AGT-04` | protocolo de Director transferible; Hermes es un adaptador |
| `STG-00..20` | biblioteca de templates/instancias; nunca veinte state machines |
| `APP-01..16` | perfil de atestación de una app, no dieciséis subsistemas internos |

Un alias puede compartir `evidence_ref` solo si su aceptación está cubierta de
forma explícita. “Mismo vertical” no permite omitir un criterio distinto.

Decisiones de producto ya resueltas para evitar ramas abiertas:

- se aceptan multiusuario/RBAC (`GOV-20`), consola de rescate gobernada
  (`UI-17`) y PostgreSQL (`OPS-11`);
- se rechazan A2UI declarativo (`WIZ-12`), escritorio nativo
  (`WIZ-14`/`UI-09`), time-travel (`ORC-18`) y juegos OPES
  (`OPE-14`): PWA, generaciones causales, backup y plugins externos cubren la
  necesidad sin otro subsistema;
- `CTX-09..11` quedan condicionales al benchmark versionado de V27. Ese gate
  debe convertir cada una en `accept` o `reject`; una decisión condicional
  abierta impide el cierre total.

## 5. Las 34 verticales causales

El estado de cada vertical se obtiene del ledger, no de esta tabla. El orden es
causal; dentro de cada corte se ejecuta la ola máxima con write-sets disjuntos.

| ID | Dependencias | Resultado y gate de salida |
|---|---|---|
| V01 Catálogo ejecutable | ninguna | Integrar los cuatro hashes; registrar los 257 IDs exactos, decisiones, targets, dependencias, aliases y aceptación. Gate: conteo/familias exactos, cero duplicados y cero decisiones de producto bloqueantes. |
| V02 Autoridad del rebuild | V01 | `AGENTS.md`, guards y docs nuevas gobiernan; superficies antiguas quedan read-only. Gate: ningún agente recibe órdenes operativas antiguas y cero import/adaptador nuevo al runtime anterior. |
| V03 Trazabilidad y lecciones | V01–V02 | Ledgers mecánicos tarea→capability, símbolo→destino, bug→invariante/test, skills/rulepacks→disposición. Gate progresivo por write-set y final `unmapped = 0`; toda retirada tiene caracterización. |
| V04 Intención, AppSpec y amendments | V01,V03 | `IntentManifest` inmutable, normalización `AppSpec`, confirmación, hash, revisiones y amendments causales. Gate: cambios nunca reescriben historia ni reaprovechan evidencia de otra generación. |
| V05 DAG y fases | V04 | WorkItem DAG, dependencias, write-sets, split/replan, recursión parent/child, templates, `PhaseInstance` inmutable y ready-set derivado. Gate: `unsatisfied_dependency_started=0`, paralelismo máximo sin conflicto, hijo contractual no desaparece y fase no gobierna lifecycle. |
| V06 Snapshot, eventos, outbox y scheduler atómicos | V04–V05 | Repositorio state-centric, CAS por revisión, idempotencia, outbox y scheduler neutral con claim, lease, fencing, capability matching, reclaim y exclusión por generación. Gate: máximo un claim activo por WorkItem/generación, crash-reclaim, stale fence rechazado, race/restart/replay sin doble efecto y cero cola privada. El lease worker no es el lease del Director. |
| V07 Config mutable y doctor | V06 | Registro único, TOML no secreto, effective config redactado, motor neutral `Manager`+`DocumentStore` con CAS/replace+fsync, `pending_restart`, generación de getters/schema/docs/descriptor UI sin web y doctor `reuse|replace|new`. Gate: cero env/key/default/alias fuera del ingress canónico. La mutación pública por API/MCP/web/CLI (`OPS-07`) no se acredita aquí: queda completa en V24 sobre RBAC, registro de comandos, i18n y Wizard. |
| V08 CredentialStore | V07 | Puerto y backend local privado para owner/scope/version/use/rotate/revoke; Goals conservan refs, no secretos. Gate: leak scan y negativos de spoof/revocación; hijo recibe allowlist exacta. |
| V09 Repositorio, backup y restore | V06–V08 | Suite contractual común de estado, backup online, restore verificado y migrador dry-run desde el inicio. Gate: snapshot restaurado conserva refs, revisiones, outbox, artifacts y terminalidad; shutdown limpio. |
| V10 Proyectos, multiusuario, RBAC y auditoría | V06,V08–V09 | Principal autenticado, proyectos/membresías y roles `platform_admin`, `project_owner`, `project_admin`, `contributor`, `reviewer`, `operator`, `viewer`; agentes como service principals. Gate: autorización application-side, aislamiento entre proyectos, colaboración dentro del mismo proyecto y audit receipt. |
| V11 OIDC/Active Directory | V08,V10 | `PrincipalSource`/`IdentityMapper` intercambiables; local token para este equipo, OIDC para Entra/AD FS y adaptador LDAP/LDAPS instalable cuando AD no ofrece OIDC. Gate OIDC: Authorization Code+PKCE, state, nonce, issuer/audience/subject/tiempos, rotación JWKS y revocación con IdP temporal. Gate LDAP/LDAPS: bind, TLS, mapeo estable, grupos, revocación y negativos contra fixture AD/Samba; Kerberos/SAML quedan fuera salvo nueva decisión explícita. |
| V12 Director con lease y fencing | V05–V06,V10 | Protocolo común para operador, Hermes o cualquier agente; claim/renew/takeover y propuestas con expected revision/fencing. Gate: director expirado no escribe, caída no pierde plan y no existe estado privado paralelo. |
| V13 Mailbox `child_delivery` | V06,V12 | `Parent` expresa linaje; no activa una barrera. `HandoffRequired` es separado, explícito, `false` por defecto e inválido como `true` sin `Parent`. Solo esa arista opt-in genera un envelope compacto `child_delivery`, ligado a proyecto, Goal/generación y source/recipient principal+WorkItem+Execution exactos. Lifecycle: `admitted → claimed → delivered → consumed → acknowledged|blocked`; el fallo previo del destinatario exacto lo deja `retired` sin readdress ni ACK sintético. Reutiliza el mismo `StateRepository`, transacción y outbox; `outbox.fence` es el único ordinal. Helper puro de resolución, Orchestrator como único writer y SQLite como misma autoridad; límite de envelope en configuración canónica. Gate: crash/restart permite reclaim sin pérdida; replay conserva fronteras históricas sin renovar lease ni redeliver; sucesor no suplanta destinatario; padre contractual no cierra sin ACK/bloqueo causal. V13 no expone el opt-in por MCP: el DAG público V05 con `Parent` cierra sin mailbox, requeue ni acción pendiente. Mensajes genéricos, sesiones y handoff de proveedor (`ORC-15`) quedan en V27; bindings públicos llegan por V20–V22 y no se simulan aquí. |
| V14 Pausa, resume, cancel, stop y replan | V05–V06,V12–V13 | Controles tipados por Goal/WorkItem/Execution y generación exacta; parada cooperativa y forzada según capability. Gate A/B/C/D: detener B preserva A/C/D antes y después de restart; terminales no se reejecutan. |
| V15 Presupuestos, permisos y efectos | V06,V10,V12,V14 | Tokens, dinero, tiempo, procesos, disco, fairness y riesgo; `EffectIntent`, approval, attempt y receipt separados. Gate: cuota temporal no se vuelve fallo terminal; efecto sin autoridad no se ejecuta; retry es idempotente. |
| V16 Workspace y Git local | V05–V06,V10,V14–V15 | `WorkspaceManager` y `VersionControl`; un worktree/ref opaco por ejecución, base exacta, inventario, commit e integración local por CAS. Gate: workspace persistido, Codex dentro del binding exacto, rework causal, conflicto/stale sin mutar destino y trabajo pendiente visible por usuario/proyecto. Reutiliza el mismo state/outbox/scheduler/ledger; no crea lifecycle ni store propios. `EXT-11` remoto queda en V28. |
| V17 Artefactos y atestador | V06,V09,V15–V16 | CAS inmutable FS, metadata causal, `TestAttestor`, snapshot/diff y ataques de filesystem/sandbox. Gate: tests independientes reproducibles; traversal/symlink/hardlink/owner/modo/leaks fallan; ACK no equivale a artefacto. |
| V18 Autor, reviews y refinery | V14,V16–V17 | Autor, reviewer primario y adversarial con tres launches distintos sobre misma generación/tree/diff/tests; rework e integración explícitos. Gate: retirar cualquier launch o cambiar el árbol bloquea promoción. |
| V19 Consejo | V12,V17–V18 | Políticas `auto|required|skip_by_operator`, propuestas, crítica, ballots, disenso, veto de seguridad y decisión. Gate: tres E2E aislados; skip lleva principal/motivo/fecha/spec hash; Consejo nunca sustituye reviews. |
| V20 Registro de comandos | V07,V10,V17 | Una definición de handlers/schemas/auth/i18n genera bindings HTTP, MCP y CLI y SDK pequeño. Gate: paridad semántica y códigos estables; ningún endpoint/provider mantiene lifecycle o DTO autoritativo propio. |
| V21 i18n total | V20 | Catálogo owner para web, Wizard, CLI, notificaciones, errores, prompts y docs públicas; español default/fallback y locales BCP-47. Gate: paridad, plurales, fechas, números, moneda, timezone y fallback probados; códigos máquina invariantes. |
| V22 Codex real completo | V04–V10,V12–V21 | Primer vertical de agente, sin depender de OIDC/AD: plan, DAG, mailbox, launch/observe/stop, workspace, tests, reviews, cierre, backup/restart y shutdown por bindings generados desde el registro. Gate `AC-V22-CODEX-E2E`: cuatro Goals A/B/C/D concurrentes, stop B preserva A/C/D, restart y cierre por MCP real, cero contradicción terminal y cero proceso propio. |
| V23 Wizard, dossier y fábrica | V04–V05,V20–V21 | Chat/formulario sobre mismo estado, preguntas por huecos, máximo configurable de rondas, recomendaciones/elección, preview, dossier y templates `research`, `build_app`, `change_app`, `domain_production`, `deploy`, `self_change`. Gate: confirmación causal crea plan; no aparece otro motor. |
| V24 Web administrativa | V10,V20–V23 | Web responsive/PWA para proyectos, Goals, fases, trabajo, agentes, ramas, artifacts, decisions, cuotas, reviews, effects y config; materializa `OPS-07` mediante los bindings API/MCP/web/CLI del registro único sobre el motor neutral V07, con expected revision, confirmación, audit receipt y `pending_restart`; streaming y rescate sin autoridad paralela. Gate: paridad multicanal, RBAC, WCAG 2.2 AA automatizado, matriz de teclado, estados comparados con queries canónicas e i18n en E2E browser. Desktop nativo queda rechazado; PWA es la única UI instalada. |
| V25 Hermes y proveedores restantes | V12–V14,V17–V22 | Hermes como Director estándar; Claude, Gemini, Ollama y runtime local mediante la familia neutral; catálogo de modelos, capabilities, cuota, uso, routing y fallback explícito. Gate: suite por adapter, smokes reales disponibles y aislamiento de ausencia/fallo; nunca paridad de calidad inventada. |
| V26 Tools, resources, skills y rulepacks | V08,V10,V15,V21 | Registro/SDK único, namespaces lazy, resources paginados, skills progresivas, scopes, trust, hashes, tests, install/upgrade/revoke/rollback y bootstrap/doctor. Gate: tool no autorizada no corre, resultado grande va a artifact y cero skill/rulepack observado sin disposición. |
| V27 Contexto, RAG, routing y evals | V17,V25–V26 | ContextBundle mínimo, refs, FTS5/BM25, memoria por proyecto, handoffs, caching/compaction y routing barato medido. Gate: dataset versionado mide calidad/coste/latencia; embeddings/reranker/vector DB solo tras mejora demostrada. |
| V28 Plugins y Forge remotos | V16–V17,V21,V26–V27 | Protocolo `DomainPlugin`; `change_app`, investigación, web/browser, documentos/PDF, datos, DB externas, presentaciones, OCR, imagen/audio, shell y computer-use como conectores gobernados. Un único puerto neutral `Forge` recibe adapters GitHub/GitLab/Gitea; push, pull request y merge remoto son efectos explícitos con credential ref, egress, permiso, CAS e idempotencia. Gate: refs opacas, permisos, contrato/E2E por adapter y cero acceso a estado/filesystem interno. |
| V29 Deploy y notificaciones | V08,V10,V15,V21,V28 | Hooks, webhooks, email, Telegram y efectos dry-run/local/contenedor/systemd; remotos como plugins ratificados. Gate: aprobación, scope, receipt, rollback e idempotencia; ningún sink decide lifecycle. |
| V30 OPES temporal completo | V18–V22,V25–V29 | Plugin externo para inventario/reutilización, investigación, temas, visuales, tests, supuestos, reviews, ensamblado, audio, tutor/RAG, HTML, manual y paquete. Gate `AC-V30-OPES`: fixture OPES temporal con DB/FS separados, seis launches acreditados cuando se exijan, lista exacta de artefactos/QA y producción imposible sin confirmación/scope. |
| V31 PostgreSQL, S3 y multihost | V06,V09–V10,V13–V17,V22 | Adapters Postgres y S3-compatible; worker/host affinity, claims con fencing, recuperación de host y workspaces clonables. Gate: misma suite que SQLite/FS, carga/concurrencia, aislamiento y colaboración multiusuario/multihost sin cambiar dominio. |
| V32 Operación completa | V07–V11,V20–V22,V29,V31 | Install, doctor, update, rollback, migraciones, retención, cleanup, idle wakeup/backoff, watchdog cooperativo, métricas/trazas, health/readiness y smokes/nightly. Gate: upgrade/rollback/restore reales, spoof rechazado, perfil idle y shutdown API con cero residuos. |
| V33 Apps externas Go y no-Go | V23–V29,V31–V32 | Dos apps completas creadas por superficies públicas y atestadas contra `APP-01..16`: hexagonal, i18n, accesibilidad, config/secretos, auth, pruebas, docs y deploy proporcional. Gate: misma tree/image y E2E real; ficheros generados sin wiring no cuentan. |
| V34 Migración, cutover y retirada | V01–V33 | Censo final, cierre de ingreso antiguo, drain/sello, snapshot/import único, configuración migrada sin doble lectura, E2E final y retirada de módulos/cmd/scripts superseded. Gate global de la sección 9; legacy queda evidencia read-only, no runtime. |

Nota de evidencia V13: un E2E Codex real que recorra composición, MCP, SQLite
y CAS pero no active `HandoffRequired` sirve como gate de no regresión. No
sustituye `AC-V13-MAILBOX` ni acredita mailbox o bindings públicos.

Corte verificable 2026-07-21: V01–V15 están acreditados por receipts V3
válidos. V16 está implementado, pero el código y los verdes focales no lo
acreditan: debe existir un receipt V3 `PASS` de `AC-V16-WORKSPACE-GIT`, generado
desde S en checkout `detached_clean` y validado por su test estricto. Los OID y
digests autoritativos vivirán únicamente en
`product/evidence/v16_workspace_git.json`.

```text
antes del receipt V16: 56/257 = 21,79 %; 15/34 = 44,12 %; 15/15 receipts
con receipt V16 válido: 59/257 = 22,96 %; 16/34 = 47,06 %; 16/16 receipts
```

El receipt V15 acredita únicamente `GOV-15`, `STG-09`, `ORC-08`, `ORC-09`,
`ORC-10`, `ORC-11`, `EVD-03` y `EVD-14`. Presupuestos jerárquicos, fairness,
riesgo/esfuerzo y ledger `intent -> approval -> attempt -> receipt` amplían el
mismo writer, scheduler, outbox y repositorio; no crean otro motor. Launch y
stop quedan gobernados también en composición Codex. HTTP/MCP/CLI públicos
siguen esperando V20 y la paridad i18n total espera V21.

El análisis y la implementación V16 están completos en el worktree según
`docs/reconstruccion/analisis_y_contrato_v16_workspace_git.md`. El siguiente y
único paso es sellar P/S/E, ejecutar su argv exacto y validar el receipt. V16
solo acredita `STG-02`, `STG-10` y `EXT-10`; Forge remoto no se abre hasta V28.

## 6. Olas y transición a auto-orquestación

Dependencias anteriores forman el DAG; una numeración no obliga a cola lineal.
Ejemplos de paralelismo seguro:

- V07 y contratos iniciales de V10 pueden avanzar tras V06 con write-sets
  separados; V08 consume la salida canónica de V07.
- Mientras V16 no tenga receipt estricto, solo se ejecuta su sellado P/S/E;
  V17 no se adelanta sobre su autoridad de workspace/Git ni se integra trabajo
  preparatorio de otra vertical.
- V17, adapters iniciales de V21 y catálogos i18n pueden desarrollarse en ramas
  separadas, pero solo integran con sus dependencias acreditadas.
- tras congelar contrato en V20, los adapters de V25 se portan en paralelo.
- plugins de V28 se paralelizan por proceso/namespace; OPES espera únicamente
  las capabilities que declara su E2E.

Hasta V22, coordinación Codex directa es excepción bootstrap: nueva Orquesta
aún no puede dirigir su propia ola completa y la antigua permanece detenida.
V22 incluye el gate de autoservicio:

```text
plan DAG + write-sets
launch hijos con refs
mailbox/ACK/entrega
wait causal
tests y dos reviews
integración/rework
restart/stop/shutdown
```

Cuando ese gate pase, nueva Orquesta dirige V23–V34 por defecto. Codex directo
queda para observación, integración o desbloqueo acotado documentado. Si el
producto se auto-modifica, usa la misma plantilla `self_change`, permisos y
receipts; no obtiene privilegios ocultos.

## 7. Evidencia exigida por tipo de capacidad

| Promesa | Evidencia mínima |
|---|---|
| regla de dominio | unitario + transición negativa/propiedad |
| puerto sustituible | suite de contrato común + fake de control |
| adaptador DB/artefactos | integración real + concurrencia/ataques aplicables |
| lifecycle/replay | restart por transición + idempotencia + race |
| interfaz pública | E2E mediante binding publicado + auth/error/i18n |
| provider | suite neutral + smoke real aislado cuando se promete disponibilidad |
| efecto externo | intent/aprobación/attempt/receipt + fallo/timeout/rollback |
| seguridad | negativos, leak scan y aislamiento; no solo happy path |
| colaboración | dos usuarios/dos proyectos negativos + mismo proyecto concurrente |
| release | tests, mutaciones críticas, E2E, restore y shutdown sobre los mismos digests inmutables de candidato |

Un test offline puede acreditar semántica offline, no disponibilidad real. Un
smoke real de un provider no acredita la calidad de otro. OIDC/AD puede pasar
con IdP estándar de test en este equipo; la conexión al tenant del despliegue
se acredita durante instalación. Una limitación del entorno queda explícita.

Primero se sella un candidato inmutable, se ejecutan sus gates y se emite la
atestación fuera del sujeto —CAS/servicio de release—. Si el repositorio
versiona después un índice de evidencia, ese índice apunta a
`subject_source_tree_digest`; nunca afirma contener su propio digest.

Cada receipt de release incluye como mínimo:

```text
subject source-tree digest + commit/tree Git sellados autoritativos
binary/image digest
binary digest
config registry revision + effective config hash redactado
capability IDs y acceptance refs
adapter/provider/model versions
actor/project/goal/generation refs
test/E2E result refs
timestamp y host/runtime class
```

## 8. Colaboración progresiva sin reescribir núcleo

La progresión multiusuario se divide para mantener primero un producto útil:

1. V10: usuarios, proyectos, membresías, RBAC y auditoría sobre SQLite en un
   servidor único.
2. V16: cada trabajo de código usa workspace/branch/worktree causal; la web
   muestra trabajo no integrado, autor, estado de review, conflicto y merge.
3. V24: varios usuarios operan el mismo proyecto desde la web sin acceder a
   tablas ni paths internos.
4. V31: Postgres, objetos compartidos, afinidad de host y recuperación permiten
   varios hosts/workers.

Git resuelve historial, ramas, diff y merge; no resuelve por sí solo permisos,
leases, double execution, artifact ownership ni actualización de Goal. Es un
adaptador VCS detrás de esos contratos.

Autenticación se elige al instalar: `local_token` en este equipo y `oidc` para
Active Directory/Entra/AD FS donde exista. Cambiar proveedor de identidad no
migra el Goal ni modifica handlers; cambia composición y mapeo de principals.

## 9. Gate global de Orquesta total

Orquesta solo se declara total cuando una única revisión/release acredita:

```text
canonical_capability_ids == 257
unclassified_capabilities == 0
blocking_product_decisions == 0
accepted_capabilities_without_owner_dependency_or_acceptance == 0
accepted_total_capabilities_not_accredited == 0
cutover_required_capabilities_not_accredited == 0
P0_open == 0
P1_open == 0

mechanical_symbol_census == 100%
meaningful_symbols_without_disposition == 0
historical_tasks_without_capability_or_supersession == 0
historical_bugs_without_lesson_test == 0
modules_without_primary_disposition == 0
skills_without_disposition_trust_or_activation_test == 0
rulepacks_without_scope_hash_or_disposition == 0

lifecycle_writers == 1
scheduler_authorities == 1
active_state_repositories_per_deployment == 1
command_registries == 1
config_registries == 1
productive_go_roots == 1
runtime_legacy_imports_or_bridges == 0
legacy_authoritative_writers == 0
legacy_active_leases_or_unreconciled_receipts == 0

sqlite_postgres_contracts == green
filesystem_s3_contracts == green
multiuser_project_isolation_and_collaboration == green
local_and_oidc_identity_contracts == green
codex_claude_gemini_ollama_local_adapter_contracts == green
accepted_real_provider_e2es == green
hermes_takeover_contract == green
http_mcp_cli_web_parity == green
full_i18n_accessibility_matrix == green
opes_temporary_e2e == green
generated_go_and_non_go_APP_01_16 == green
install_upgrade_rollback_restore_shutdown == green
critical_negative_and_mutation_matrix == green
```

Si una capacidad opcional queda `rejected`, su decisión no convierte en verde
una implementación inexistente; simplemente la saca del producto aceptado. Si
queda `post_release` o `deferred`, Orquesta puede cerrar una release menor, pero
no esta meta total.

## 10. Control de progreso y desvío

El informe periódico usa hechos:

```text
capacidades por estado y release target
vertical activa y dependencias acreditadas
P0/P1 y bloqueos externos
tests/E2E/receipts nuevos
legacy retirado
LOC netas, writers, loops, stores, comandos y paquetes
siguiente dependencia causal
```

Puede calcularse una fracción mecánica `accredited / accepted`, siempre con
denominador y release target visibles. No se usa una estimación de esfuerzo ni
se presenta una release parcial como “100 %”.

Antes de comenzar cada tarea se declara capability, invariante, writer,
puertos, adapters, write-set, dependencia, retirada, contratos, negativos/E2E
y presupuesto. Al cerrar se registran cambios, evidencia, retirada/bloqueo,
LOC netas, riesgos y siguiente dependencia. El formato completo vive en
`AGENTS.md`.

## 11. Ratchets de simplicidad

- objetivo orientativo: núcleo de dominio pequeño y producto base muy por
  debajo del runtime anterior; LOC nunca justifican quitar garantías;
- medir además writers, loops, stores, roots, comandos, dependencias y
  complejidad;
- todo paquete/interfaz/servicio nuevo necesita frontera real y consumidor;
- superar presupuesto ratificado exige ADR, alternativa evaluada y pieza
  anterior retirada;
- generated y plugins no son vertedero para maquillar tamaño;
- cada vertical retira su autoridad sustituida; excepciones llevan owner,
  causa, caducidad y gate;
- no instalar tool, vector DB, microservicio o provider sin caso, contrato,
  operación y retirada mantenibles;
- normalizar lo recuperable; rails heurísticos no deciden trabajo;
- documentación explica y enlaza; nunca guarda estados manuales paralelos.

## 12. Resultado final esperado

Una Orquesta instalable localmente con SQLite/filesystem o en servidor con
PostgreSQL/S3, multiusuario y colaborativa; identidad local u OIDC/Active
Directory elegida en instalación; Director intercambiable, agentes iguales por
protocolo y roles asignables; Codex, Hermes, Claude, Gemini, Ollama y local como
adaptadores; Consejo y reviews acreditados; API/MCP/CLI/web coherentes; Wizard,
Git, tools, skills, contexto, plugins, deploy, OPES y operación completa.

Todo ello conserva un único Goal, un único writer y una única composición. Si
el gate de la sección 9 no pasa, el estado correcto es “producto parcial con
estas capacidades acreditadas”, nunca “Orquesta total”.
