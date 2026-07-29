# Balance legacy y preflight antirregresión

Fecha de corte: 2026-07-29. Base de Orquesta V2:
`232ed4ef4485c596c55d708182244de207dee7bd`.

## Veredicto

Orquesta V2 tomó la decisión arquitectónica correcta: reconstrucción limpia,
un `Goal`, un lifecycle, un escritor de aplicación y un scheduler. V1–V22 están
acreditadas mediante receipts propios; no dependen en ejecución de repositorios
legacy.

El rescate documental anterior fue amplio, pero no suficiente como prevención:
la lección más importante de las primeras Orquestas —delegar microtareas
cerradas por función o invariante— quedó reducida en V2 a write-set, tests y
split/replan. El producto puede crear todavía un único `WorkItem` para un
objetivo amplio. Esto explica parte de los agentes que pierden contexto,
rediseñan demasiado o entregan candidatos difíciles de integrar.

No debe copiarse el núcleo viejo. Sí deben recuperarse contratos, negativos y
patrones concretos. El índice
`product/knowledge/legacy_preflight_patterns.jsonl` hace esa consulta local,
determinista y sin modelos.

## Alcance y fuentes

Familias localizadas:

| Familia | Corte o papel | Uso correcto |
|---|---|---|
| `orquesta` clásica | `7576f60bd3b5424a6c1c19e4f319cb634b12f231` | Incidencias maduras, Director, OPES, runtime y operación. Solo consulta. |
| `orquesta.bk(sin VM Berserk)` | generación temprana | UI, wizard y mailbox; no estado. |
| `orquesta-autonomia-clean` | `53ecf770c819cdf9db6a0461366389b13c03f1b0` | Microprogramación, capacidad, mailbox y contexto acotado. |
| `orquesta-autoprogramacion-*` | generaciones intermedias | DAG, microtareas, reentrada y fixtures; no loops. |
| `orquesta-goal-worktree-*` | goal-first clásico | Conocimiento de frontera, OPES y adaptadores. |
| `orquesta-wt-lease-generation-*` | carrera del primer lease | Fixture de V12. |
| `orquesta-rebuild` | `6f244a759414...` | Fuente directa de la reconstrucción V1–V22. |
| `orquestaV2-legacy-consulta` | copia navegable | Consulta física separada del producto. |
| `PlataformaMunicipal/orquestador` | referencia externa | PTY, cuotas, systemd y watchdog. |
| `.orquesta-*`, workspaces y runtimes | evidencia causal | No son generaciones ni fuente canónica. |
| bundle pre-separación | backup verificado 2026-07-26 | Abrir solo ante pérdida concreta. |

El censo anterior ya recorrió 78 ramas locales, 74 remotas, 33 tags, 152
commits y 148 árboles distintos. Se revisaron además el informe V1–V22, matriz
física, mapa de Directores, corte de auditoría clásica, análisis de fallos
estructurales y ledgers V03.

No se extrajo el tar comprimido: existe copia navegable y no había una pérdida
concreta que justificara duplicar espacio y superficie. Ningún backup fue
borrado.

## Balance V1–V22

| V | Commit de cierre | Valor vigente | Rescate útil / prohibición |
|---|---|---|---|
| V1 | `e68ad92280e0` | Catálogo ejecutable | Reusar hashes/censo; no inventario alterno. |
| V2 | `e68ad92280e0` | Goal y escritor únicos | No restaurar Plan/Run/Director como autoridad. |
| V3 | `e68ad92280e0` | Ledgers y sujetos sellados | Bugs sirven como lección, no como cierre nuevo. |
| V4 | `e68ad92280e0` | Intent/AppSpec/amendment causal | No drafts con lifecycle por canal. |
| V5 | `fb38152787e0` | Goal, DAG, fases y ready-set | Recuperar microgranularidad; no otra unidad ejecutiva. |
| V6 | `dc54f283919d` | Estado, eventos, outbox y claims atómicos | No JSON/outbox paralelo. |
| V7 | `6d88f0f2d53c` | Registro/TOML/effective config/CAS | No env/default repartido. |
| V8 | `afb7857ed37c` | CredentialStore y LeakGuard | Reusar negativos de links/permisos/fugas. |
| V9 | `70fb59e9fcde` | Recovery, backup y restore | Reusar corrupción y replay; no fallback-store. |
| V10 | `c74b6d766426` | Identidad, proyectos, RBAC y auditoría | No middleware alterno. |
| V11 | `5e826de0e40c` | OIDC/AD por puertos | Proveedores siempre opt-in. |
| V12 | `5e826de0e40c` | Director lease y fencing | No otro coordinador con lease. |
| V13 | `5daf174bde3e` | Mailbox causal y recipient exacto | Persistir no equivale a entregar. |
| V14 | `e3e7c28e669c` | Pause/resume/cancel/stop/replan | Stop exige muerte física. |
| V15 | `3eaa533d6fd1` | Presupuestos, permisos y efectos | Conservar intent→approval→attempt→receipt. |
| V16 | `a4f602ab01c4` | Workspace/Git binding y CAS | No confiar en ruta. |
| V17 | `a97ea3bc3771` | Tests y atestador independiente | No ACK/exit 0 como prueba. |
| V18 | `32ee17e40700` | Reviews independientes | Autor, reviewer y adversarial distintos. |
| V19 | `f7a574e36528` | Council durable | No rama roja ni skip narrativo. |
| V20 | `7f27685d992c` | Registro único de comandos | No policy duplicada en DTO/handler. |
| V21 | `216e61e80c1b` | i18n total | Un catálogo, códigos máquina estables. |
| V22 | `d1b551a136fe` | Codex E2E y autoservicio | Codex sigue fuera del core; HOME se cerró después. |

Conclusión: V1–V22 no deben reabrirse en bloque. Su conducta útil está
reimplementada y acreditada. El rescate pendiente es transversal y selectivo.

## Patrones amplios

1. Autoridades duplicadas producen estados contradictorios.
2. Estado durable y proceso vivo se confunden.
3. Frentes amplios trasladan arquitectura al agente y generan deriva.
4. Persistencia no atómica rompe replay, leases y outbox.
5. Write-set sin binding Git/workspace permite integrar otro árbol.
6. Tests amplios o mal ligados producen falsos verdes.
7. Config, HOME, OAuth y sesión dispersos cruzan identidades.
8. Capacidad sin fairness/progreso causa starvation o CPU inútil.
9. Confinamiento convertido en gate móvil engorda verticales funcionales.
10. Dominios externos dentro del núcleo rompen reutilización.
11. Rails textuales descartan trabajo recuperable.
12. Fuente, binario, config y remoto sin digest común producen drift.
13. Documentar sin consulta previa ni ratchet permite repetir errores.

El detalle estructurado está en el JSONL: síntomas, causa, solución reutilizable,
antipatrones, capacidades, paths, operaciones, evidencia, tests y hueco.

## Secuencias de recurrencia actuales

Estas regresiones no invalidan el cierre V1–V22, pero prueban que el preflight
era necesario:

| Secuencia | Solución histórica | Regresión/hueco posterior |
|---|---|---|
| autoridad | Goal/writer/scheduler únicos | BUG593 y BUG603 intentaron duplicar autoridad o policy |
| granularidad | microtareas dirigidas | tareas amplias y launches que pueden posponer progreso |
| cierre/evidencia | mailbox + atestador + review | BUG454, 594, 604–606 |
| estado/proceso | stop físico y recovery | BUG595 y BUG608 |
| concurrencia | claims, CAS y scheduler | BUG607: hambre de commit/attest/integrate/observe |
| HOME/auth | perfil físico aislado | BUG583: confundió perfil con daemon |
| Git | binding/base OID/ChangeSet | BUG598: metadata padre distinta del árbol físico |
| efectos | intent/attempt/receipt | BUG600 y visibilidad de efectos pendientes |
| Council/rework | sujetos y ballots sellados | BUG597 |
| config | snapshot canónico | BUG576: identidad ligada a ruta mutable |

Prioridad al reanudar: BUG607, BUG598, BUG595/608, BUG604, BUG603, BUG605/606,
BUG597 y BUG583. Antes de integrar reparaciones ya existentes, revisar sus
candidatos y tests; no rehacerlos.

## Regresión concreta: granularidad

Legacy definía `EspecificacionFuncion` con:

- archivo y símbolo objetivo;
- descripción exacta;
- precondiciones y postcondiciones;
- dependencias permitidas/prohibidas;
- tests obligatorios;
- write-set y formato de salida.

Emitía `MICROTAREA CERRADA`, recortaba contexto al símbolo y tenía negativos de
deriva. La idea es buena; su store, lifecycle, materializador y atestación
autodeclarada no deben copiarse.

V2 tiene `WorkItem`, DAG, write-set, tests requeridos y replan. Sin embargo:

- el plan mínimo puede crear un único item con el objetivo completo;
- `WorkItem` no expresa target symbol/slice, pre/postcondiciones ni
  dependencias de código permitidas/prohibidas;
- `ORC-03` acredita controles y split/replan, no granularidad semántica;
- templates `build_app` y `self_change` conservan unidades amplias.

Mejora futura: ampliar el contrato del `WorkItem` existente, no crear otra
entidad. El Director divide por función, contrato o invariante causal cuando sea
posible. Tarea mayor solo con justificación durable. “Pequeña” no significa
artificial: dominio, contexto y write-set mandan.

## Estado del conocimiento histórico

V03 conserva:

- 348 IDs históricos normalizados;
- 1.307 ocurrencias exactas;
- 339 disposiciones de fuente;
- 349 bugs del rebuild en el corte consultado.

Pero 200 lecciones históricas siguen en `pending_invariant_test` con referencias
`planned:`. El censo existe; parte aún no es ratchet. Deben cerrarse por riesgo
y cercanía al write-set, no copiando tests en masa.

El clásico también tenía un `CodeContextBrokerV0` con `search`, `symbol`,
`architecture`, `repo_map`, callers/imports, límites, caché y fallback `rg`.
Contrato y fixtures pueden inspirar V27. No se porta ahora: para este problema
un índice exacto de 13 filas es más barato, auditable y sin procesos residentes.

## Preflight sin tokens

```bash
scripts/consultar_lecciones_legacy.sh \
  --capability ORC-03 \
  --path internal/application \
  --operation plan
```

Más consultas:

```bash
scripts/consultar_lecciones_legacy.sh --operation shutdown
scripts/consultar_lecciones_legacy.sh --path internal/config --operation config
scripts/consultar_lecciones_legacy.sh --all --json
```

Propiedades:

- Bash + `jq`;
- cero LLM, embeddings, RAG, red o codebase-memory;
- coincidencias exactas por capability/operación y prefijos de ruta;
- no bloquea ni descarta trabajo;
- no duplica cierres: enlaza roadmap, evidence y traceability;
- salida compacta antes de diseñar función o módulo.

Sin coincidencia devuelve estado 1: “índice incompleto”, nunca “trabajo
prohibido”.

## Invariantes para no volver a tropezar

- Una autoridad por concepto.
- Progreso durable ejecutable antes de nuevos launches.
- Metadata causal y realidad física coinciden: tree/base/PID/HOME/config.
- Stop significa proceso desaparecido y shutdown deja cero recursos propios.
- `produced`, `blocked`, `unknown_applied`, `failed` y `canceled` no se mezclan.
- Recovery aplica los mismos invariantes que escritura normal.
- Tests declaran toolchain y capacidades del sandbox antes del launch.
- Un perfil Codex tiene HOME/auth exclusivos; una Orquesta usa muchos perfiles.
- Provider/modelo nunca entra en Goal/core.
- Limpieza solo por identidad exacta y después de shutdown.
- Rescate legacy: microtarea, test, commit pequeño en castellano; nunca merge
  masivo.

## Próximo orden

1. Convertir `LEGACY-001` en contrato y prueba focal de descomposición.
2. Adjuntar resultado del preflight a planificación como receipt advisory.
3. Priorizar `pending_invariant_test` por write-set de cada tarea.
4. Reanudar V23 con tareas cortas, disjuntas y revisables.
5. Mantener Firecracker/microVM como adaptador separado; no gate de V23.

## Estado al cerrar

No se modificó código funcional de V23. No se borró, reseteó ni integró ningún
candidato. El trabajo no versionado de Analizador y los worktrees de agentes
quedan preservados. Este corte vive en rama/worktree aislados de auditoría.
