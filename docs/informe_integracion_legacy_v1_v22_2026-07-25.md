# Informe total de integración legacy V1–V22

Fecha de corte: 2026-07-25

Repositorio receptor: `/home/alberto/Trabajo/orquestaV2`

Rama observada: `integracion/v23-intake-durable`

Commit observado al cerrar el censo: `f58992364b46`

Repositorio histórico principal, solo lectura: `/home/alberto/Trabajo/orquesta`

## 1. Conclusión ejecutiva

Orquesta V2 no ha perdido el núcleo valioso de V1–V22. Las veintidós
verticales reconstruidas están etiquetadas, acreditadas con contratos
ejecutables y son ancestros de la rama actual. El resultado actual es más
coherente que las generaciones legacy porque concentra el ciclo de vida en
`Goal`, la escritura de estado en aplicación, la planificación en un único
scheduler y los comandos en un único registro.

Sí quedan ideas valiosas de las generaciones anteriores que conviene adoptar,
pero casi ninguna justifica copiar paquetes completos. Las oportunidades de
mayor valor son:

1. Terminar de endurecer el atestador V17: detección real de “cero tests” y
   sustituir Bubblewrap por Firecracker para desbloquear snapshots grandes. La
   reparación estructural de Bubblewrap queda congelada hasta que Orquesta sea
   autoprogramable.
2. Convertir el antiguo inventario `AgentHomeV0` en un pool moderno de perfiles
   Codex aislados, con cuotas, enfriamiento y concurrencia por cuenta. El
   aislamiento de un perfil ya fue recuperado después de V22.
3. Recuperar como política neutral el routing de modelo/esfuerzo y la escalada
   por riesgo, sin introducir nombres de proveedor en el núcleo.
4. Adaptar el handoff completo de la antigua rotación de sesiones a
   `Goal`/`WorkItem`/mailbox, sin crear un segundo ciclo de vida.
5. Incorporar observación de CPU y shutdown cooperativo con checkpoint, sin
   convertir telemetría o palabras del contenido en rails de ejecución.
6. Aprovechar en V23 la UX del antiguo wizard —recomendación visible,
   explicación en lenguaje llano, formulario y chat sobre el mismo estado—,
   conservando `Application` como único escritor.
7. Recuperar adaptadores de proveedores, `DomainWork`, egress HTTP, OPES,
   documentos, notificaciones y view-models UI en las verticales futuras que ya
   les corresponden.

La regla de integración resultante es:

> Copiar literalmente solo fixtures, vectores negativos y pequeñas reglas
> deterministas que puedan volver a sellarse. Adaptar las ideas de producto y
> los contratos. No copiar stores, loops, autoridades, handlers ni paquetes
> completos legacy.

No se recomienda reabrir V1–V22 para introducir estas mejoras. Los hallazgos
deben entrar en V23 o verticales posteriores, salvo las dos regresiones
acotadas del atestador V17.

## 2. Qué significa V1–V22 en Orquesta V2

Los tags `v1` a `v22` no son veintidós copias completas e independientes de la
aplicación histórica. Son veintidós cortes verticales acreditados de la
reconstrucción. Cada tag sella un sujeto y un recibo de aceptación. Algunas
verticales se cerraron en el mismo commit:

- V1–V4 comparten el commit fuente `e68ad92280e0`.
- V11–V12 comparten el commit fuente `5e826de0e40c`.

Esto no invalida sus recibos: los sujetos, contratos y evidencias son distintos.
Sí evita interpretar el nombre del tag como una fotografía histórica única.

La reconstrucción clasifica 257 capacidades: 81 están acreditadas en V1–V22 y
176 permanecen declaradas para verticales posteriores. El informe separa:

- **Vertical acreditada:** contrato actual V1–V22.
- **Generación legacy:** repositorio, rama, árbol o backup anterior.
- **Reutilización literal:** bytes, fixture o helper pequeño incorporable con
  revisión y nueva evidencia.
- **Adaptación conceptual:** preservar semántica, no arquitectura ni código.
- **Ya integrado:** la solución actual cubre o supera la idea antigua.
- **No adoptar:** produciría una segunda autoridad, deuda o riesgo operativo.

## 3. Método y alcance

Se usaron inspecciones de solo lectura:

- `git for-each-ref`, `git rev-list`, `git show`, `git log` y comparación de
  árboles para el censo de refs y tags.
- `rg` y lecturas acotadas para contratos, evidencias, incidentes, código y
  documentación.
- `scripts/bootstrap_agent_tooling.sh --status` antes de la auditoría.
- Comparación con el roadmap, catálogo, ruta de reconstrucción, matriz de
  rescate, gate legacy V23, informe de generaciones y análisis forense previo.

No se lanzó ni indexó `codebase-memory-mcp`. El estado de bootstrap informó
cero procesos de ese servicio. No se abrieron tokens, credenciales, bases de
producción ni el contenido de backups binarios que no aportaban una evidencia
concreta.

El censo de refs del repositorio histórico fue exhaustivo para su metadata en
este corte: 78 ramas locales, 74 remotas, 33 tags, 152 commits distintos y 148
árboles distintos. La lectura semántica profunda se agrupó por árbol y familia;
no se volvió a leer como distinta cada rama que era alias del mismo árbol.

## 4. Inventario físico de generaciones

| Fuente | Commit/árbol observado | Papel | Decisión |
|---|---|---|---|
| `orquesta.bk(sin VM Berserk)` | `c653b9a3c2fb` / `803434c946e9` | UI, wizard y mailbox tempranos | Arqueología dirigida; adoptar UX, no estado |
| `orquesta-autonomia-clean` | `53ecf770c819` / `1123ba24ab8b` | Monolito de autonomía funcional | Extraer contratos; no copiar loop |
| `orquesta-autoprogramacion-20260512-120537` | `8924d5fd92b6` / `f1072674045f` | Recuperación/autoprogramación divergente | Adaptar DAG, write-set y tests |
| `orquesta-autoprogramacion-20260512-224047` | `77b6620a8dc8` / `2c080f43018f` | Estado durable de autoprogramación | Adaptar handoff/causalidad |
| `orquesta-goal-worktree-180f5631` | `180f56311c8f` / `2d38b9755d38` | Goal-first clásico y OPES | Fuente conceptual para adapters |
| `orquesta-wt-lease-generation-*` | `01cb27d778c6` / `4386…` | Reparación de lease de primer lanzamiento | Casos de carrera ya cubiertos en V12 |
| `orquesta` | `7576f60bd3b5` / `e0079…` | Generación clásica final | Biblioteca histórica de escenarios |
| `orquesta-rebuild` | `6f244a759414` / `98789e…` | Reconstrucción total anterior | Fuente directa de cierres V1–V22 |
| `orquestaV2` | `f58992364b46` al cierre | Receptor actual | Única línea de producto |
| `PlataformaMunicipal/orquestador` | `2314f1ad9806` | PTY, cuotas, systemd/watchdog externo | Referencia operativa, no núcleo |

También existen copias de fuente bajo `.orquesta-v22-e-*/source*`, workspaces,
runtimes, estados y evidencias E2E. Son pruebas o material de recuperación, no
nuevas generaciones canónicas. Deben conservarse mientras una incidencia los
cite y eliminarse solo mediante limpieza gobernada.

### 4.1 Familias de ramas legacy

| Familia de refs | Contenido útil | Estado frente a V2 |
|---|---|---|
| `archive/*`, `backup/*`, `salvage/*` | Fixtures, incidencias, recuperación de servidor | Usar solo ante regresión concreta |
| `autoprogramacion/*` | DAG, write-sets, tests, handoff, reentrada | Adaptar a Goal; no importar aggregate |
| `reconstruction/v07-*` | Configuración y doctor | Superado por V7 |
| `reconstruction/v18-*` a `v22-*` | Reviews, council, comandos, i18n, E2E | Integrado y acreditado |
| ramas V22 especializadas | E2E, budgets, RBAC, revocación, config/MCP, mailbox, cierre | Nueve árboles revisados; no faltan ficheros de producto |
| `H0-*`, `H4-*` | Canon/promoción y gates | Útil como trazabilidad, no runtime |
| `model-routing/*`, `fix/*` | Selección de modelo, cuotas, reparaciones | Adaptar en V25/V27 |
| `goal/*`, `opes/*` | Goal-first, OPES, uso de servidor | Adaptar en V28–V30 |
| auditorías/bugs remotos | Casos negativos y falsos verdes | Reutilizar tests/fixtures |

Los worktrees de reconstrucción V23–V37 contienen principalmente contratos
rojos o ramas de preparación. No deben tratarse como implementaciones perdidas.
Los worktrees del corte V07 están superados y los V22 especializados convergen
en el V22 acreditado. La comparación por árbol evita rescatar varias veces el
mismo código con nombres de rama distintos.

## 5. Estado acreditado por vertical

### V1 — Catálogo ejecutable

- **Fuente acreditada:** tag `v1`, commit `e68ad92280e0`.
- **Evidencia:** `product/evidence/v01_source_integration.json`, PASS.
- **Propósito:** inventario exacto de 257 capacidades, 13 familias, cuatro
  hashes de autoridad y 33 mapeos diferidos.
- **Problema histórico resuelto:** roadmap narrativo no verificable y recuentos
  que cambiaban según el documento leído.
- **Legacy útil:** scripts de inventario y taxonomías de las generaciones
  anteriores aportan fixtures comparativos.
- **Decisión:** ya integrado y superior. Solo copiar literalmente listas de IDs
  faltantes si coinciden con el catálogo sellado.
- **No adoptar:** recuentos calculados desde Markdown vivo o estados de UI.
- **Aceptación futura:** cualquier capacidad rescatada debe recibir ID,
  clasificación y vertical propietaria antes de código.

### V2 — Autoridad de reconstrucción

- **Fuente acreditada:** tag `v2`, commit `e68ad92280e0`.
- **Evidencia:** `product/evidence/v02_authority_rules.json`, PASS.
- **Capacidades:** `GOV-03`, `GOV-21`.
- **Propósito:** `Goal` es el único ciclo de vida, aplicación el único escritor
  y scheduler; el proveedor no gobierna el producto.
- **Problemas resueltos:** deriva del catálogo de escritores y el intento de una
  segunda mutación de `Goal` durante recuperación.
- **Legacy útil:** únicamente escenarios negativos de doble writer.
- **Decisión:** no hay código legacy que mejore esta frontera.
- **No adoptar:** control-plane antiguo, `legacy_director_loop`, stores
  paralelos, handlers que escriben ciclo de vida o estado del proveedor.
- **Aceptación futura:** toda adopción debe probar que no crea otro writer,
  scheduler, lifecycle, queue u outbox.

### V3 — Ledgers y trazabilidad

- **Fuente acreditada:** tag `v3`, commit `e68ad92280e0`.
- **Evidencia:** `product/evidence/v03_canonical_ledgers.json`, PASS.
- **Capacidad:** `GOV-16`.
- **Cobertura:** 829 fuentes Markdown, 334 disposiciones, 2.108 tareas, 920
  exclusiones, 170 bloques revisados y 334 IDs históricos de bugs normalizados.
- **Problemas resueltos:** recibos invalidados por el worktree vivo, inferencia
  falsa de cierre histórico y mezcla de ledgers vivos con sujetos sellados.
- **Legacy útil:** los inventarios de bugs y exclusiones son datos, no
  autoridad.
- **Decisión:** ya integrado. Los futuros rescates deben enlazar blobs/commits
  sellados, no rutas mutables.
- **No adoptar:** “cerrado porque aparece en un documento” ni receipts
  recalculados desde la rama actual.

### V4 — Intent, AppSpec y amendments

- **Fuente acreditada:** tag `v4`, commit `e68ad92280e0`.
- **Evidencia:** `product/evidence/v04_intent_appspec.json`, PASS.
- **Capacidad:** `GOV-02`.
- **Propósito:** bytes/hash exactos de Intent, confirmación, `AppSpec`
  inmutable y amendment causal que genera sucesor.
- **Problemas resueltos:** confirmación SQLite más estrecha que el revisor,
  validación incompleta de migración, deriva del reason, regex de cero tests,
  consumidor E2E omitido y parches sobre la capacidad equivocada.
- **Legacy útil:** ejemplos de conversaciones del wizard para mejorar la
  captura de intención en V23.
- **Decisión:** adaptar UX, no el state machine de sesión.
- **No adoptar:** edición in-place de `AppSpec` o mutación por una interfaz.
- **Aceptación futura:** mismo Intent para formulario/chat, confirmación exacta
  y cero efectos ante mismatch.

### V5 — Goal, DAG, fases y ready set

- **Fuente acreditada:** tag `v5`, commit `fb38152787e0`.
- **Evidencia:** `product/evidence/v05_goal_dag_phases.json`, PASS.
- **Capacidades:** `GOV-04`, `STG-00`, `ORC-01`, `ORC-02`, `ORC-06`,
  `ORC-18`.
- **Propósito:** dependencias, cohortes máximas sin conflicto, fases inmutables,
  planes append-only y requisitos neutrales.
- **Problemas resueltos:** ownership/dependencias sobredeclarados, compiladores
  de plan divergentes, colisión de claves locales, generación nueva que
  invalidaba trabajo vivo y retry sobre generación equivocada.
- **Legacy útil:** `BuildAutoprogrammingAutonomyProgramV0` conserva una buena
  descomposición por nodos, dependencias, write-sets y tests.
- **Decisión:** adoptar esa descomposición como compilación a
  `Goal`/`WorkItem`; no copiar `AutonomyProgram`.
- **No adoptar:** un aggregate de autoprogramación con lifecycle propio.
- **Aceptación futura:** self-change es trabajo normal, usa el mismo ready set,
  conflictos, claims, tests y receipts.

### V6 — Estado atómico, eventos, outbox y claims

- **Fuente acreditada:** tag `v6`, commit `dc54f283919d`.
- **Evidencia:** `product/evidence/v06_atomic_state_outbox.json`, PASS.
- **Capacidades:** `GOV-05`, `GOV-06`, `ORC-12`, `ORC-13`, `ORC-17`,
  `EVD-02`, `OPS-09`, `OPS-10`, `OPS-12`.
- **Propósito:** snapshot/revisión/evento/outbox atómicos, intentos múltiples,
  leases con tiempo confiable, fences y reentrada.
- **Problemas resueltos:** outcome desconocido tras crash, claim stale,
  duplicados de outbox, reclaim incorrecto, consulta N+1 y divergencia de
  semántica entre memoria y SQLite.
- **Legacy útil:** fixtures de crash en cada frontera y casos de fence stale.
- **Decisión:** copiar fixtures negativos cuando cubran una frontera aún no
  probada; nunca copiar repositorios ni outboxes legacy.
- **No adoptar:** dual write, reintento automático de un efecto
  `unknown_applied` o un segundo `StateRepository`.

### V7 — Configuración canónica

- **Fuente acreditada:** tag `v7`, commit `6d88f0f2d53c`.
- **Evidencia:** `product/evidence/v07_config.json`, PASS.
- **Capacidades:** `OPS-01`, `OPS-02`, `OPS-04`, `OPS-05`, `OPS-06`,
  `OPS-26`–`OPS-30`.
- **Propósito:** un registry estricto, TOML humano, configuración efectiva
  redactada, CAS, recuperación, pending-restart, getters, schema y docs.
- **Problemas resueltos:** tests atados a una revisión, canonicalizadores OIDC
  duplicados, límites no canónicos, mailbox sin límite, bloat generado y raíz
  de workspace omitida por validación cruzada.
- **Legacy útil:** nombres anteriores solo como aliases temporales declarados.
- **Decisión:** ya integrado. No crear variables fuera del registry.
- **No adoptar:** `os.Getenv` disperso, prefijos amplios ni defaults en
  adaptadores.

### V8 — Credenciales

- **Fuente acreditada:** tag `v8`, commit `afb7857ed37c`.
- **Evidencia:** `product/evidence/v08_credentials.json`, PASS.
- **Capacidades:** `EVD-11`, `EVD-12`, `OPS-03`, `OPS-08`.
- **Propósito:** `CredentialStore` neutral, owner/scope/purpose/version,
  rotación/revocación, LeakGuard y allowlist exacta para hijos.
- **Problemas resueltos:** secretos cortos presentes dentro de `[REDACTED]`,
  seam Codex solo fake, campos durables omitidos en escaneo, errores que
  retenían secretos, firmas sobre el store completo, colisión `.next` y
  confusión UID real/efectivo.
- **Legacy útil:** fixtures de symlink, hardlink, modos, propietario,
  revocación y filtrado de errores.
- **Decisión:** fixtures sí; stores y copia de credenciales no.
- **No adoptar:** copiar `auth.json` por ejecución o persistir tokens/HOME en
  core.

### V9 — Recovery, backup y restore

- **Fuente acreditada:** tag `v9`, commit `70fb59e9fcde`.
- **Evidencia:** `product/evidence/v09_recovery_backup.json`, PASS.
- **Capacidades:** `EVD-15`, `OPS-14`.
- **Propósito:** puerto neutral, backup online SQLite, manifest CAS, restore
  privado/inactivo, integridad y no reejecución terminal.
- **Problemas resueltos:** contención de writers filtrada como conflicto de
  negocio, cadena de recovery manipulable, hechos de integración incompletos,
  falsos rojos por temporales inseguros y timeout de harness confundido con
  fallo real.
- **Legacy útil:** corpus de backups corruptos, manifests manipulados y
  recuperación tras crash.
- **Decisión:** reutilizar fixtures; no restaurar la base o los scripts legacy
  como producto.
- **Aceptación futura:** restauración sobre instancia nueva, validación de
  agregados y prohibición de reejecutar terminales.

### V10 — Identidad, proyectos, RBAC y auditoría

- **Fuente acreditada:** tag `v10`, commit `c74b6d766426`.
- **Evidencia:** `product/evidence/v10_identity_projects_rbac.json`, PASS.
- **Capacidades:** `GOV-19`, `GOV-20`, `GOV-22`.
- **Propósito:** principal y proyecto explícitos, jerarquía lógica, siete roles,
  default-deny, invisibilidad cross-project y auditoría durable.
- **Problemas resueltos:** proyectos sin owner, bypass de política por cliente
  inyectado, validación JWKS sobredeclarada y parsing `Bearer` duplicado.
- **Legacy útil:** fixtures de revocación, colaboración y aislamiento.
- **Decisión:** V10 no necesita modificación estructural. El “primer admin” de
  una instalación solo OIDC continúa asignado a V24.
- **No adoptar:** autorización en cada transporte o identidad inferida del
  directorio/HOME.

### V11 — OIDC y Active Directory instalables

- **Fuente acreditada:** tag `v11`, commit `5e826de0e40c`.
- **Evidencia:** `product/evidence/v11_oidc_ad.json`, PASS.
- **IDs nuevos:** ninguno; es cierre transversal deliberado.
- **Propósito:** `IdentityProvider` neutral, token/JWKS exactos, subject estable,
  PKCE/state/nonce y smoke real Dex 2.45.1 + Samba AD 4.17.12 por LDAPS.
- **Problemas resueltos:** autoridad de identidad mezclada con LDAP y claims no
  verificados.
- **Legacy útil:** composición de smoke y certificados temporales.
- **Decisión:** conservar como test instalable; no importar cliente LDAP en
  dominio/core.
- **Pendientes externos:** bootstrap de primer admin en V24 y telemetría de
  `unknown kid` en V32.

### V12 — Director lease y fencing

- **Fuente acreditada:** tag `v12`, commit `5e826de0e40c`.
- **Evidencia:** `product/evidence/v12_director_lease.json`, PASS.
- **Capacidades:** `GOV-08`, `GOV-09`, `GOV-10`, `ORC-24`.
- **Propósito:** humanos y servicios usan el mismo protocolo; una lease por
  Goal, token/fence, takeover, expiry confiable y propuesta exacta.
- **Problemas resueltos:** token activo casi persistido en history/snapshot,
  segunda autoridad y replay cuadrático, orden de replay/revocación, receipts
  concurrentes ambiguos y takeover sin fence suficiente.
- **Legacy útil:** worktree específico de lease de primer lanzamiento como
  fixture de carrera.
- **Decisión:** escenarios ya integrados; mantener fixture si cubre una carrera
  no duplicada.
- **No adoptar:** sesión privada del Director ni token vivo incrustado en Goal.

### V13 — Mailbox causal

- **Fuente acreditada:** tag `v13`, commit `5daf174bde3e`.
- **Evidencia:** `product/evidence/v13_mailbox.json`, PASS.
- **Capacidades:** `ORC-04`, `ORC-05`, `ORC-14`.
- **Propósito:** separar admisión y entrega; parent/child/source/recipient
  exactos, claim/fence, ordinal y replay.
- **Problemas resueltos:** mailbox huérfano tras fallo de recipient,
  divergencia fake/SQLite, ordering desigual, tabla de fences duplicada,
  admisión de edge no contractual y bloqueo del DAG al inferir que todo
  `Parent` exigía handoff.
- **Legacy útil:** handoff completo de rotación de sesiones.
- **Decisión:** adaptar el contenido del handoff a mensajes causales y
  `HandoffRequired`; no copiar cola/sesión.
- **Aceptación futura:** objetivo, progreso, siguiente acción, pendientes,
  write-set, tests, riesgos, refs y evidencia, todos ligados al destinatario.

### V14 — Controles operativos

- **Fuente acreditada:** tag `v14`, commit `e3e7c28e669c`.
- **Evidencia:** `product/evidence/v14_controls.json`, PASS.
- **Capacidades:** `GOV-07`, `STG-15`, `ORC-03`, `ORC-16`.
- **Propósito:** pause/resume/cancel/stop/replan exactos, gate de dispatch,
  muerte física confirmada y supersession forzada atómica.
- **Problemas resueltos:** stop pendiente que hacía hot-loop de fsync y
  bloqueaba launches; recursión/adaptación sobredeclarada.
- **Legacy útil:** protocolo de shutdown autorizado con checkpoint, relectura
  de actividad, deadline, stop forzado y cero residuo.
- **Decisión:** adaptar esos escenarios en V32 sobre controles actuales.
- **No adoptar:** dar por parado un proceso por cambiar estado lógico o
  reintentar un outcome externo desconocido.

### V15 — Presupuestos, permisos y efectos

- **Fuente acreditada:** tag `v15`, commit `3eaa533d6fd1`.
- **Evidencia:** `product/evidence/v15_budgets_effects.json`, PASS.
- **Capacidades:** `GOV-15`, `STG-09`, `ORC-08`–`ORC-11`, `EVD-03`,
  `EVD-14`.
- **Propósito:** dimensiones/scope tipados, reservas, cuotas temporales,
  permisos y separación intent/approval/attempt/receipt.
- **Problemas resueltos:** status de efectos ambiguo, idempotencia no ligada al
  payload, liberación de workspace en replay, unknown-applied, N+1 de claims y
  eliminación accidental de una regresión histórica con nombre.
- **Legacy útil:** datos de cuota/cooldown de `AgentHomeV0` y políticas de
  escalada de modelo.
- **Decisión:** adaptar a presupuestos neutrales; no meter cuentas/proveedores
  dentro de Goal.
- **No adoptar:** tratar texto, ACK o “hecho” del agente como efecto aplicado.

### V16 — Workspace y Git

- **Fuente acreditada:** tag `v16`, commit `a4f602ab01c4`.
- **Evidencia:** `product/evidence/v16_workspace_git.json`, PASS.
- **Capacidades:** `STG-02`, `STG-10`, `EXT-10`.
- **Propósito:** workspace por ejecución, binding/base/write-set, changeset,
  CAS de integración, rework y replay.
- **Problemas resueltos:** marker del perdedor CAS, marker de release,
  temporales inseguros, identidad de diff solo por path, worktree 0775 y
  snapshot no ligado al workspace.
- **Legacy útil:** fixtures Git conflictivos, write-sets disjuntos y casos de
  recuperación de worktree.
- **Decisión:** copiar fixtures acotados; no adoptar rutas/worktrees legacy por
  parecer recientes.
- **No adoptar:** path como identidad de artefacto o integración fuera de CAS.

### V17 — Tests y atestador

- **Fuente acreditada:** tag `v17`, commit `a97ea3bc3771`.
- **Evidencia:** `product/evidence/v17_test_attestor.json`, PASS.
- **Capacidades:** `EVD-01`, `EVD-04`, `EVD-05`, `EVD-13`.
- **Propósito:** tests estructurados, artefactos CAS, atestación independiente,
  snapshot sellado y ejecución confinada.
- **Problemas resueltos:** snapshot/diff que no ligaban contenido/modo, offset
  compartido de memfd, AppArmor por path y receipts placeholder.
- **Legacy útil de alta prioridad:** el helper
  `localCommandGoTestWithoutExecutedTestsV0` detecta exit 0 con
  `[no tests to run]` y escaneo vacío.
- **Decisión:** reimplementar esa semántica en el adaptador actual y copiar sus
  vectores de prueba. No copiar el módulo
  `orquesta-runtime-required-test`.
- **Regresiones actuales:** BUG-454, falso verde cuando Go ejecuta cero tests;
  BUG-462, expansión grande del árbol/NOFILE, ya reparada según inventario;
  BUG-463, límite de argumentos de Bubblewrap, congelado como deuda por
  decisión del propietario.
- **Aceptación:** exit 0 no basta; debe demostrarse al menos un test ejecutado.
  El snapshot grande debe viajar O(1) en argumentos y conservar bytes/modos.
  En el alcance inmediato esta aceptación corresponde al adaptador Firecracker
  de `TestAttestor`, no a una reparación de Bubblewrap.

### V18 — Reviews independientes

- **Fuente acreditada:** tag `v18`, commit `32ee17e40700`.
- **Evidencia:** `product/evidence/v18_independent_reviews.json`, PASS.
- **Capacidades:** `GOV-12`, `STG-13`, `STG-14`, `STG-16`, `EVD-06`.
- **Propósito:** autor, reviewer primario y adversarial distintos; mismo sujeto
  exacto, invalidación por drift y rework causal.
- **Problemas resueltos:** acceso de evidencia no autorizado, lease incorrecta,
  seam congelado sobre implementación, falso rojo por lifecycle histórico,
  causa de replan desigual y gate histórico recalculado tras deriva.
- **Legacy útil:** prompts/rúbricas compactas de Claude/Gemini como revisores,
  siempre sobre sujeto sellado.
- **Decisión:** adaptar rúbricas; no entregar contexto bruto ni usar el reviewer
  como writer.
- **No adoptar:** review por nombre de fichero mutable o “aprobado” sin hash.

### V19 — Council

- **Fuente acreditada:** tag `v19`, commit `f7a574e36528`.
- **Evidencia:** `product/evidence/v19_council.json`, PASS.
- **Capacidades:** `GOV-11`, `GOV-13`, `GOV-14`, `STG-06`, `STG-08`,
  `EVD-07`.
- **Propósito:** council automático/requerido/skip por operador, ballots
  launch-bound, veto, desacuerdo y recibos P/S/E.
- **Problemas resueltos:** ownership omitido, regex que elegía test inexistente
  pero aceptaba package PASS, skip que sintetizaba decisión, recovery/digests,
  timestamp de autorización durable, replan negativo y shapes canceladas.
- **Legacy útil:** matrices de desacuerdo y veto, no motores de decisión.
- **Decisión:** conservar fixtures por política; no crear otro council state.
- **No adoptar:** `skip` como aprobación implícita ni timestamp como clave
  idempotente.

### V20 — Registro único de comandos

- **Fuente acreditada:** tag `v20`, commit `7f27685d992c`.
- **Evidencia:** `product/evidence/v20_command_registry.json`, PASS.
- **Capacidades:** `GOV-17`, `UI-02`.
- **Propósito:** una definición genera HTTP/MCP/CLI/SDK con auth, schema, i18n y
  errores comunes.
- **Problema histórico resuelto:** una primera rama privada incompleta, basada
  en catálogo stale y sin SQLite/bootstrap/recovery/E2E, fue rechazada. La
  versión final recompuso selectivamente 25 comandos sobre V19.
- **Legacy útil:** descripciones y ejemplos de las seis herramientas MCP
  antiguas como aliases/documentación.
- **Decisión:** ya integrado y superior.
- **No adoptar:** toolsets escritos a mano, DTO de transporte como autoridad o
  lectura del registry desde disco durante runtime.

### V21 — Internacionalización total

- **Fuente acreditada:** tag `v21`, commit `216e61e80c1b`.
- **Evidencia:** `product/evidence/v21_i18n.json`, PASS.
- **Capacidad:** `UI-18`.
- **Propósito:** catálogo/manifest único, español por defecto, fallback inglés,
  BCP47, plurales, fechas, números, moneda, timezone y campos máquina
  invariantes.
- **Problemas resueltos:** documentación temprana describía material sin
  sellar; el tag final sí tiene evidencia PASS.
- **Legacy útil:** glosarios y microcopy del wizard, después de normalización.
- **Decisión:** adaptar textos, nunca traducir IDs/códigos/refs.
- **No adoptar:** traducir toda la documentación legacy o anunciar superficies
  futuras como implementadas.

### V22 — E2E Codex real y autoservicio

- **Fuente acreditada:** tag `v22`, commit `d1b551a136fe`.
- **Evidencia:** `product/evidence/v22_codex_e2e.json`, PASS.
- **Capacidades:** `STG-11`, `EVD-09`, `AGT-01`, `AGT-03`.
- **Propósito:** MCP conduce plan/DAG/mailbox/workspace/tests/reviews/cierre;
  cuatro Goals concurrentes, stop selectivo, restart, backup y shutdown, sin
  contradicciones terminales ni procesos residuales.
- **Problemas inicialmente rojos y cerrados:** autoridad de execution service
  no durable, prompt en inglés, smoke de un solo Goal, placeholder de cero
  tests y continuación P0 de mailbox.
- **Solución final:** identidad material-free ligada a ejecución, continuación
  por outbox, MCP público y E2E real A/B/C/D.
- **Hallazgo post-V22:** el aislamiento de HOME/cuenta no formaba parte de ese
  sujeto. BUG-455 mostró colisión de perfiles y se cerró después con
  `account_home`, configuración canónica y
  `scripts/orquesta_profile_server.sh`.
- **Decisión:** V22 está acreditado; no debe reabrirse. Las regresiones
  post-release van al inventario y a commits nuevos.
- **No adoptar:** copiar `auth.json`, lock transitorio o fallback al HOME del
  daemon.

## 6. Funcionalidades legacy reutilizables

| Hallazgo | Fuente legacy | Estado en V2 | Tipo | Beneficio | Riesgo/dependencia | Prioridad |
|---|---|---|---|---|---|---|
| HOME persistente por perfil y lock de por vida | perfiles clásicos + `AgentHomeV0` | Integrado después de V22 | Concepto mejorado | Varias cuentas sin pisarse | permisos, symlink, hardlink, auth exacta | Hecho/P0 |
| Pool de 2–8 homes, cuota, cooldown y sesiones | `AgentHomeV0` | Falta | Conceptual | capacidad multi-cuenta y fairness | V15, V25, V32; no exponer paths | P1 |
| Routing por riesgo y complejidad | `ResolveModelRoutingV0` | Falta | Conceptual | coste/calidad coherentes | V25/V27; proveedor fuera de core | P1 |
| Handoff completo antes de rotar sesión | session rotation legacy | Parcial en mailbox | Conceptual + fixture | recuperación sin pérdida | V13/V25/V27 | P1 |
| Self-change con DAG/write-set/tests | autoprogramación legacy | Semántica parcial | Conceptual | Orquesta se mejora usando su propio contrato | V23/V33; sin aggregate paralelo | P1 |
| Detectar exit 0 con cero tests | required-test runner | Falta/regresión BUG-454 | Helper/fixture | elimina falso verde | atestador V17 | P0 |
| Snapshot grande O(1) en argv | incidencias V17 actuales | Firecracker activo; Bubblewrap congelado (BUG-463) | Implementación actual | árboles reales sin explosión de argumentos | `TestAttestor`/Firecracker | P0 acotado |
| Watchdog de CPU sin progreso | T259/generación clásica | Falta | Conceptual | evita procesos muertos consumiendo CPU | telemetría y shutdown V32 | P1 |
| Shutdown con checkpoint y cero residuo | server-shutdown legacy | Parcial | Escenarios/fixtures | cierre fiable | V14/V32; efecto externo exacto | P1 |
| Routing Claude/Gemini/Ollama | adapters clásicos | Falta | Conceptual | proveedores alternativos/review barato | V25; normalizador neutral | P2 |
| Wizard U1–U12/T1–T8/R1–R8 | primera generación/UI | Parcial | UX conceptual | intake comprensible y recomendaciones | V23; Application único writer | P1 |
| Egress HTTP con allowlist/redirect/retry | `domain-work-http.ClientV0` | Falta | Conceptual + tests | integraciones seguras | V28; SSRF, budgets, Retry-After | P2 |
| `DomainWorkJobRequestV0` y adapters | Goal-first clásico | Falta | Contrato conceptual | apps externas sin contaminar core | V28 | P2 |
| Conector OPES temporal/causal | `orquesta-opes-*` | Falta en V2 | Conceptual + smoke | reutilización real en dominio | V30; instancia temporal | P2 |
| PDF/OpenXML/presentaciones | tooling clásico | Falta | Tool/plugin | producción documental | V26/V28; confinamiento | P2 |
| Email/Telegram/notificaciones | hooks clásicos | Falta | Adapter conceptual | operación asíncrona | V29; efectos/secretos | P2 |
| View-models de telemetría/estado | web clásica | Falta | Formato conceptual | observabilidad útil | V24/V32; UI no writer | P2 |

### 6.1 Aislamiento HOME: qué se recuperó realmente

Los commits posteriores a V22 observados para este frente fueron:

- `8ca85ac4`: configuración canónica del perfil.
- `40d00301`: aislamiento del agente Codex.
- `297d9af5`: servidor de perfil.
- `74f280e0`: renovación de E2E real.
- `7e344c55`: documentación.

La solución actual usa una referencia opaca de perfil, asigna
`HOME=CODEX_HOME` persistente, mantiene `flock` durante toda la vida, valida
owner/modo/symlink/hardlink, exige el `auth.json` exacto, limita por ahora la
concurrencia del perfil a uno y liga la identidad al journal. No cae al HOME del
daemon.

Esto mejora la versión antigua. `AgentHomeV0` mezclaba inventario lógico,
account/entitlement, cuota y paths en una estructura próxima a runtime. Para el
pool futuro deben conservarse las refs opacas y separar:

- perfil/cuenta y credencial;
- capacidad, reserva y uso;
- asignación de ejecución;
- HOME físico, propiedad exclusiva del adapter;
- política de routing, propiedad de aplicación/configuración.

### 6.2 Selección de modelos

La política antigua distinguía severidad `normal`, `complex` y `critical`,
trabajo trivial, requisitos de evidencia, policy ref, model ref y effort. La
idea sigue siendo buena:

- trivial/exploración/documentación: modelo económico, esfuerzo bajo o medio;
- programación ordinaria: esfuerzo medio/alto;
- concurrencia, persistencia, seguridad, causalidad y recuperación: alto;
- `xhigh`: solo riesgo justificado o autorización causal.

Debe expresarse como requerimientos neutrales de trabajo y resolverse en el
adapter/proveedor. No deben aparecer nombres comerciales en `Goal`, DAG,
scheduler o core. Cuota y cooldown pertenecen a presupuestos/capacidad, no a
heurísticas por texto.

### 6.3 Autoprogramación

La pieza más aprovechable del legacy no es su loop, sino el compilador mental:

1. descomponer objetivo en nodos;
2. declarar dependencias;
3. declarar write-set;
4. declarar tests/artefactos;
5. lanzar cohortes sin conflicto;
6. revisar sobre sujeto sellado;
7. replanificar causalmente.

En V2 esto debe ser un `Goal` normal de tipo `self_change`, sin privilegios y
sin `AutonomyProgram`, `PlanState`, cola, store u outbox adicionales. La propia
Orquesta puede ser su aplicación objetivo, pero debe pasar los mismos permisos,
presupuestos, workspaces, tests, reviews, council y efectos.

## 7. Qué puede copiarse literalmente

La lista literal es deliberadamente corta:

1. Vectores de prueba de “exit 0, cero tests”, adaptados al runner actual.
2. Fixtures corruptos de recovery: manifest, checksum, orden, revisión y
   terminal replay.
3. Fixtures de credenciales: symlink, hardlink, owner, modo y error con secreto.
4. Fixtures de mailbox: duplicado, fence stale, cross-project, parent sin
   handoff y replay por ordinal.
5. Fixtures de lease/takeover y CAS Git.
6. Mensajes y glosario del wizard, después de pasar por i18n V21.

Incluso estos datos deben copiarse a un write-set actual, eliminar paths o
secretos locales, actualizar sus expectativas al contrato V2 y generar un
receipt nuevo. El hecho de que un fixture pasara en legacy no es evidencia
actual.

## 8. Qué no debe copiarse

- `cmd/db/internal`, `ensureLocalDB` o el control-plane clásico.
- `legacy_director_loop`, `OperationalDirectorPlanStateV0` o un plan vivo
  paralelo a Goal.
- `AutonomyProgram` como aggregate independiente.
- Stores, snapshots, queues, outboxes o schedulers legacy.
- Paquetes completos o cherry-picks amplios desde `orquesta`.
- Handlers web/MCP/CLI que muten estado directamente.
- DTOs de transporte convertidos en dominio.
- Proveedor, modelo, HOME, OAuth, tokens o paths físicos dentro del core.
- Lectura/escritura directa de filesystem o DB interna de una app externa.
- Copia de `auth.json` por ejecución.
- Locks adquiridos solo durante el arranque.
- Idempotencia basada en timestamp, path o mensaje de texto.
- Rails de contenido por palabras, formato recuperable, log, diff o
  “progreso” heurístico.
- ACK, texto del agente o exit 0 como prueba de efecto/test.
- Worktrees antiguos adoptados por fecha o nombre sin comparar árbol.
- Recibos legacy presentados como evidencia de la rama actual.
- Dual writes durante una “migración temporal”.

## 9. Backlog de adopción

### I-01 — Cerrar el falso verde de cero tests

- **Prioridad:** P0.
- **Origen:** required-test runner legacy y BUG-454.
- **Write-set recomendado:** adaptador/atestador V17 y tests focales.
- **Implementación:** parser estructurado del resultado del runner; reconocer
  package PASS sin casos ejecutados, `[no tests to run]`, filtro que no
  selecciona tests y salida vacía.
- **Aceptación:** fixtures negativos fallan; un test real pasa; receipt liga
  comando, filtro, subject y evidencia de caso ejecutado.

### I-02 — Sustituir Bubblewrap por Firecracker en `TestAttestor`

- **Prioridad:** P0.
- **Origen:** BUG-462/BUG-463 y lecciones de memfd V17.
- **Write-set recomendado:** adaptador Firecracker de `TestAttestor`,
  composición canónica y tests Linux focales.
- **Implementación:** transportar el snapshot sellado a una microVM efímera sin
  expandir una pareja de argumentos por entrada. No convertir este frente en
  ejecución general de agentes.
- **Aceptación:** árbol sintético mayor que los límites previos, modos/bytes
  exactos, sin red/root/env, sin proceso residual y uso acotado de CPU/memoria.
- **Deuda congelada:** recompilar Bubblewrap con 65.536 argumentos, resolver
  AppArmor y diseñar su transporte O(1). Solo se reabre después de que Orquesta
  sea autoprogramable.

### I-03 — Terminar V23 con intake durable y UX rescatada

- **Prioridad:** P1, después de I-01/I-02 si bloquean el gate.
- **Origen:** wizard legacy y contrato actual V23.
- **Write-set recomendado:** únicamente piezas declaradas por el vertical V23.
- **Implementación:** recommendation visible, ayuda llana, gaps U/T/R,
  formulario y chat sobre el mismo estado, defaults técnicos silenciosos pero
  inspeccionables.
- **Aceptación:** `Application` sigue siendo único writer; confirmación exacta;
  restart; i18n; transports generados desde V20.

### I-04 — Pool multi-HOME y routing de perfiles

- **Prioridad:** P1.
- **Origen:** `AgentHomeV0`, perfiles clásicos y BUG-455.
- **Dependencias:** V15, V25 y telemetría V32.
- **Write-set recomendado:** adapter Codex, puertos de capacidad,
  configuración canónica y tests; no core Goal.
- **Aceptación:** dos perfiles reales simultáneos no comparten identidad,
  credencial, HOME, lock ni journal; max sessions, reserva, cooldown, restart y
  revocación son durables.

### I-05 — Routing neutral de modelo/esfuerzo

- **Prioridad:** P1.
- **Origen:** `ResolveModelRoutingV0`.
- **Dependencias:** V15, V25/V27.
- **Write-set recomendado:** política de aplicación + adapters de proveedor.
- **Aceptación:** misma entrada produce decisión determinista y receipt;
  modelos no disponibles degradan o bloquean explícitamente; xhigh exige causa;
  el núcleo no importa proveedor.

### I-06 — Rotación con handoff causal

- **Prioridad:** P1.
- **Origen:** session rotation legacy.
- **Dependencias:** V13, V25/V27.
- **Write-set recomendado:** comandos/aplicación/mailbox, no nueva sesión de
  dominio.
- **Aceptación:** el sustituto no arranca hasta admitir el handoff completo; se
  conservan refs, write-set, tests y pendientes; replay no duplica ejecución.

### I-07 — Self-change como Goal normal

- **Prioridad:** P1.
- **Origen:** autoprogramación legacy.
- **Dependencias:** V23 para intake/plantilla y vertical self-change asignada
  por roadmap.
- **Write-set recomendado:** compilador/plantilla de aplicación.
- **Aceptación:** no hay aggregate/store/outbox nuevos; el cambio pasa
  workspace, tests, reviews, council, CAS, efectos y receipt normal.

### I-08 — Watchdog de CPU y shutdown cooperativo

- **Prioridad:** P1.
- **Origen:** T259, server-shutdown y orquestador municipal.
- **Dependencias:** V14 y V32.
- **Write-set recomendado:** composición de servidor, puertos de telemetría y
  shutdown.
- **Aceptación:** CPU sostenida + ausencia de progreso produce diagnóstico y
  checkpoint/stop cooperativo; nunca juzga contenido; proceso confirmado
  muerto; restart consistente; cero residuos.

### I-09 — Proveedores alternativos y revisión compacta

- **Prioridad:** P2.
- **Origen:** adapters Claude/Gemini/Ollama.
- **Dependencias:** V18, V25 y routing I-05.
- **Aceptación:** mismo puerto y normalizador; sujeto sellado; coste/cuota en
  receipt; proveedor nunca writer.

### I-10 — DomainWork, egress y OPES

- **Prioridad:** P2.
- **Origen:** Goal-first clásico y módulos `orquesta-domain-work*` /
  `orquesta-opes-*`.
- **Dependencias:** V28–V30.
- **Aceptación:** refs opacas, app temporal, allowlist, redirección revalidada,
  `Retry-After`, budgets, P/S/E y cero acceso a DB/filesystem interno.

### I-11 — Documentos, notificaciones y UI

- **Prioridad:** P2.
- **Origen:** tools/bridges/web clásicos.
- **Dependencias:** V24, V26, V28, V29.
- **Aceptación:** adapters instalables, confinamiento y límites; command
  registry único; UI solo read-model/comandos; secretos por CredentialStore.

### I-12 — Censo legacy automatizado y no sensible

- **Prioridad:** P2.
- **Origen:** coste de esta auditoría.
- **Write-set recomendado:** tooling de reconstrucción y manifest sellado.
- **Aceptación:** lista repos/ref/commit/tree/tamaño/fecha sin tokens ni
  contenido sensible; agrupa árboles idénticos; no arranca indexadores; permite
  comparar un nuevo gate sin releer 152 aliases.

## 10. Matriz final V1–V22

| V | Fuente | Estado | Mejor rescate | Decisión | Prioridad |
|---:|---|---|---|---|---|
| 1 | `e68ad92280e0` | PASS | fixtures de catálogo | ya integrado | — |
| 2 | `e68ad92280e0` | PASS | negativos de doble writer | tests, no código | permanente |
| 3 | `e68ad92280e0` | PASS | ledgers/bugs históricos | datos sellados | P2 |
| 4 | `e68ad92280e0` | PASS | UX de intención | adaptar en V23 | P1 |
| 5 | `fb38152787e0` | PASS | DAG/write-set/tests de autonomía | adaptar a Goal | P1 |
| 6 | `dc54f283919d` | PASS | fixtures crash/fence | copiar fixtures | P1 |
| 7 | `6d88f0f2d53c` | PASS | aliases conocidos | registry actual | — |
| 8 | `afb7857ed37c` | PASS | fixtures FS/secretos | copiar fixtures | P1 |
| 9 | `70fb59e9fcde` | PASS | corpus restore corrupto | copiar fixtures | P1 |
| 10 | `c74b6d766426` | PASS | RBAC/revocación | ya integrado | — |
| 11 | `5e826de0e40c` | PASS | smoke Dex/AD | conservar test | P2 |
| 12 | `5e826de0e40c` | PASS | carrera de first lease | fixture focal | P1 |
| 13 | `5daf174bde3e` | PASS | handoff de rotación | adaptar mailbox | P1 |
| 14 | `e3e7c28e669c` | PASS | shutdown/checkpoint | adaptar V32 | P1 |
| 15 | `3eaa533d6fd1` | PASS | cuotas/cooldown/routing | adaptar V25 | P1 |
| 16 | `a4f602ab01c4` | PASS | fixtures Git/workspace | copiar fixtures | P1 |
| 17 | `a97ea3bc3771` | PASS con regresiones posteriores | cero-tests y snapshot O(1) | Firecracker ahora; Bubblewrap después | P0 acotado |
| 18 | `32ee17e40700` | PASS | rúbricas multi-proveedor | adaptar | P2 |
| 19 | `f7a574e36528` | PASS | veto/desacuerdo | fixtures | P2 |
| 20 | `7f27685d992c` | PASS | aliases/docs MCP | ya superior | — |
| 21 | `216e61e80c1b` | PASS | glosario/microcopy | adaptar con i18n | P2 |
| 22 | `d1b551a136fe` | PASS | E2E + perfil HOME post-V22 | no reabrir | hecho |

## 11. Confianza y huecos

| Área | Confianza | Motivo/límite |
|---|---|---|
| Tags, commits y recibos V1–V22 | Alta | inspección directa de tags, roadmap y evidence |
| Contratos/objetivos por vertical | Alta | fuentes actuales selladas |
| Problemas históricos V1–V22 | Alta | inventario de bugs + documentos de cierre |
| Generaciones físicas principales | Alta | commit/árbol y AGENTS inspeccionados |
| Censo de ramas/refs | Alta en metadata | 152 commits/148 árboles; lectura agrupada |
| Semántica de cada alias remoto | Media | no se duplicó lectura de árboles idénticos |
| Workspaces/runtimes temporales | Media | tratados como evidencia, no fuente |
| Backups binarios/tar antiguos | Baja/no inspeccionada | sin necesidad concreta; posible contenido sensible |
| V17 en el worktree vivo | Media | había trabajo concurrente de reparación al corte |
| Funcionalidad posterior a V22 | Media-alta | HOME verificado en código/docs; no se ejecutó E2E en esta auditoría |

Los backups binarios solo deben abrirse ante una pérdida concreta que no esté
representada en Git. Antes de ello hay que calcular hash, montar una ruta
aislada, evitar credenciales y registrar exactamente qué se buscó.

## 12. Criterio de entrada para cualquier rescate

Una idea legacy solo entra si supera todas estas preguntas:

1. ¿Añade una capacidad ausente o cierra una regresión demostrada?
2. ¿Tiene vertical propietaria en el roadmap?
3. ¿Respeta Goal/application/scheduler/state/command registry únicos?
4. ¿Puede entrar por un puerto o adapter sin contaminar core?
5. ¿Declara write-set, dependencias, presupuesto y efectos?
6. ¿Tiene tests negativos y sujeto sellado?
7. ¿Produce receipts P/S/E actuales?
8. ¿Supera a la implementación presente, en vez de solo ser más antigua o más
   grande?
9. ¿Evita secretos, paths locales, DB compartida y heurísticas de contenido?
10. ¿Puede integrarse en un commit pequeño, en castellano y revertible?

Si alguna respuesta es no, se conserva como referencia histórica o se
documenta como idea; no se copia.

## 13. Veredicto

V1–V22 no necesitan una nueva reconstrucción ni un merge masivo desde legacy.
La mayor parte de sus soluciones esenciales ya está en Orquesta V2 y acreditada.
El valor restante está concentrado en unas pocas ideas operativas y fixtures.

El orden recomendado, actualizado por la decisión del propietario, es:

1. cerrar BUG-454 y sustituir Bubblewrap por Firecracker solo en
   `TestAttestor`;
2. recuperar el candidato y terminar V23 sin mezclar cambios legacy;
3. conseguir que Orquesta se autoprograme mediante `Goal`;
4. solo entonces reabrir BUG-463/Bubblewrap y evaluar la ejecución general de
   agentes en Firecracker, incluidos workspaces en RAM y checkpoints;
5. construir pool multi-HOME, routing neutral y watchdog/shutdown cooperativo
   en sus verticales;
6. recuperar providers, DomainWork, OPES, documentos, notificaciones y UI solo
   cuando sus verticales estén abiertas.

Así se conserva el trabajo anterior sin reintroducir las causas por las que fue
necesaria Orquesta V2.
