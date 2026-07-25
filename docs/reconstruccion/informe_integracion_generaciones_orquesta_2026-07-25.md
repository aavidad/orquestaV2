# Informe de integración de generaciones Orquesta — 2026-07-25

Estado: mapa inicial cerrado; barrido de todas las familias legacy antes de
cada V y revisión global al final del roadmap.

## Veredicto ejecutivo

Se localizaron y auditaron cuatro familias históricas de código bajo
`/home/alberto/Trabajo`, además de la línea OrquestaV2:

1. `orquesta.bk(sin VM Berserk)`: generación temprana.
2. `orquesta-autonomia-clean` y los cortes
   `orquesta-autoprogramacion-*`: autonomía y autoprogramación.
3. `orquesta`: generación clásica final, con Director, Goal-first, conectores,
   operación real e historial de incidencias.
4. `orquesta-rebuild`: reconstrucción que produjo V01--V22 y preparó los
   contratos V23--V37.

La conclusión no es «copiar el legado». La línea clásica final y los cierres
V01--V22 son ancestros del target actual, de modo que gran parte del trabajo ya
está incorporado por descendencia. Sin embargo, algunas capacidades quedaron
solo en módulos congelados, adaptadores antiguos, scripts operativos o
documentación y nunca se conectaron al producto OrquestaV2. El aislamiento
persistente de `HOME` por cuenta era uno de esos huecos.

El rescate se regirá por cuatro condiciones acumulativas:

- aporta una mejora demostrable frente al target;
- encaja en la arquitectura y autoridad única de OrquestaV2;
- no reintroduce un problema histórico ya conocido;
- puede verificarse con contrato y pruebas actuales.

Nada se integra solo por existir en una versión anterior. No se harán
`cherry-pick` masivos, imports al árbol clásico, bridges permanentes ni
dual-writes. Las fuentes antiguas se conservan y se usan para extraer contratos,
casos adversariales, fixtures y decisiones de diseño.

## Puerta de entrada obligatoria para cada V

Antes de iniciar una nueva V se revisará el inventario completo de las cuatro
familias históricas, sus árboles independientes deduplicados y las ramas de
reconstrucción relacionadas. El barrido debe comprobar si existe trabajo útil
en cualquiera de ellas, no solo en la fuente que parezca más cercana.

Para mantener el avance, el primer paso usa identidades Git, índices, diffs,
símbolos y documentos de cierre; la lectura profunda se reserva para los
candidatos aplicables a la V. Cada hallazgo queda clasificado como
`ya_integrado`, `adaptar` o `descartar`, con evidencia y motivo. Superar esta
puerta no acredita la V: el código adaptado todavía debe pasar sus contratos,
pruebas y recibos actuales.

La revisión global al final del roadmap se mantiene como una segunda defensa:
volverá a comparar el producto completo con todas las generaciones para
detectar mejoras transversales que no pertenecían claramente a una sola V.

## Alcance y evidencia

Target auditado:

- repositorio: `orquestaV2-v23-home-aislado`;
- base confirmada: `2881bea08982`;
- estado acreditado: V01--V22;
- estado parcial no sellado: V23;
- estado planificado: V24--V37.

La deduplicación se hizo por `HEAD`, `HEAD^{tree}`, relación de ancestros, blobs
compartidos y presencia de producto real frente a fixtures o runtime. No se
leyeron credenciales ni se trataron logs, ACKs, estados o binarios como fuente.
La matriz física de rutas y copias está en
[`matriz_rescate_generaciones_orquesta_2026-07-25.md`](matriz_rescate_generaciones_orquesta_2026-07-25.md).

Identidades relevantes:

| Familia | Commit / relación | Lectura |
| --- | --- | --- |
| `orquesta.bk(sin VM Berserk)` | `c653b9a3c2fb` | Ancestro temprano; bajo valor salvo arqueología puntual de UI/mailbox. |
| `orquesta-autonomia-clean` | `53ecf770c819` | Ancestro; conserva el monolito funcional que después se retiró. |
| `orquesta-autoprogramacion-...120537` | `8924d5fd92b6` | Rama divergente; contiene cierre/evidencias que deben compararse, no portarse. |
| `orquesta-autoprogramacion-...224047` | `77b6620a8dc8` | Ancestro; estado durable absorbido por la línea posterior. |
| `orquesta-goal-worktree-180f5631` | `180f56311c8f` | Ancestro; Goal-first y OPES fueron absorbidos en el clásico final. |
| `orquesta-wt-lease-generation-20260710` | `01cb27d778c6` | Ancestro; lease/generación ya absorbidos. |
| `orquesta` | `7576f60bd3b5` | Fuente clásica final y ancestro del target; comparte 5.289 de 5.294 blobs. |
| `orquesta-rebuild` | cierres V01--V22 ancestros | Fuente directa de los contratos acreditados de OrquestaV2. |

Los múltiples `orquestaV2-v22-*`, `orquestaV2-v23-*`, workspaces Goal, smokes y
runtimes no son generaciones completas adicionales. Son worktrees, ramas
parciales, snapshots o evidencias y se deduplican antes de inspeccionarlos.

## Estado de integración V01--V22

Los 22 contratos figuran como `executable`, tienen receipts en
`product/evidence` y sus cierres terminales están contenidos en el target. La
decisión general es no portar ramas antiguas completas.

| V | Capacidad | Estado y decisión |
| --- | --- | --- |
| V01 | Catálogo ejecutable | Integrada. Mantener catálogo y hashes como autoridad; no importar inventarios alternos. |
| V02 | Reglas de autoridad | Integrada. Toda reutilización debe respetar escritor y lifecycle únicos. |
| V03 | Ledgers y trazabilidad | Integrada. Reusar lecciones históricas, no receipts antiguos como prueba nueva. |
| V04 | Intención, AppSpec y amendments | Integrada. No restaurar Drafts o sesiones por canal como autoridad. |
| V05 | Goal, DAG, fases y ready-set | Integrada. Reusar negativos de write-set, recuperación y recursión. |
| V06 | Estado, eventos, outbox y claims atómicos | Integrada. No reintroducir stores JSON o outboxes paralelos. |
| V07 | Configuración canónica | Integrada y superior. Las ramas V07 parciales están superadas. |
| V08 | CredentialStore | Integrada y superior. Reusar pruebas de symlink, hardlink, permisos, revocación y fugas. |
| V09 | Recovery, backup y restore | Integrada. Reusar corrupción, descriptor exacto y CAS causal como negativos. |
| V10 | Proyectos, multiusuario y RBAC | Integrada. No portar middleware de autorización alterno. |
| V11 | OIDC y AD instalables | Integrada. Los proveedores siguen siendo adaptadores opt-in. |
| V12 | Director lease y fencing | Integrada. No crear otro coordinador o daemon con lease propio. |
| V13 | Mailbox causal | Integrada. Reusar ACK exacto y deduplicación; no crear otro buzón. |
| V14 | Pause, resume, cancel, stop y replan | Integrada. La muerte real del proceso sigue siendo requisito, no solo estado. |
| V15 | Presupuestos, permisos y efectos | Integrada. Conservar `intent -> approval -> attempt -> receipt` y `unknown_applied`. |
| V16 | Workspace y Git | Integrada. Preservar binding exacto y CAS; no adoptar worktrees por ruta. |
| V17 | Artefactos y atestador | Integrada, con BUG-454 abierto. Adaptar el detector Go legacy; no portar el módulo. |
| V18 | Reviews independientes | Integrada. Mantener separación autor/reviewer/adversarial y rework causal. |
| V19 | Council | Integrada. Reusar veto, disenso y decisión durable; descartar la rama roja anterior. |
| V20 | Registro único de comandos | Integrada. HTTP, MCP, CLI, SDK y web deben derivar del mismo descriptor. |
| V21 | i18n | Integrada. Extender catálogo único, no crear catálogos por provider. |
| V22 | Codex E2E | Integrada. El HOME por perfil era un hueco operativo posterior, no una V22 nueva. |

Las ramas especializadas V22 no ancestro no contienen ningún fichero ausente
del cierre final. Fueron sustituidas por revisiones posteriores. Solo se abrirán
para extraer una prueba puntual cuando exista un bug actual que la justifique.

## Estado y reutilización V23--V37

V23 tiene implementación parcial real en el target. Los worktrees históricos
V24--V37, salvo V30 que no se localizó, contienen especificación, fixture y
test rojo deliberado, pero no producto en `internal`, `modulos` o `cmd`. Son
contratos útiles; no son capacidades terminadas.

| V | Estado real | Qué reutilizar | Qué evitar |
| --- | --- | --- | --- |
| V23 Wizard | Parcial: dominio `internal/intake`, pruebas focales y WIZ-03/04/15; sin sello. | Preguntas por gaps, snapshot común chat/form, recomendación única, decisión causal y atomicidad. | Copiar el worktree rojo; crear lifecycle por sesión o canal. |
| V24 Web administrativa | Solo contrato. | Descriptores V20, queries, i18n, paridad de superficies, RBAC, a11y y PWA como aceptación. | Que el frontend sea store, scheduler o autoridad RBAC. |
| V25 Providers | Solo contrato. | Familia neutral launcher/observer/controller y catálogo común de errores/capacidades. | Marcas, SDK, retries o secretos dentro del core. |
| V26 Tools/skills/SDK | Solo contrato. | Registry único, trust, hash, scope, install/upgrade/revoke/rollback y artifact refs. | Un registro o autorización independiente por provider/plugin. |
| V27 Context/RAG/evals | Solo contrato. | `ContextBundle`, provenance, freshness, ACL/cache, evaluación y promoción medida. | Índice como autoridad, fallback entre tenants o corpus completo en contexto. |
| V28 Plugins de dominio | Solo contrato. | `DomainPlugin`, refs opacas y ledger de efectos; MCP/HTTP y hosts Git como adaptadores. | DB/filesystem/lifecycle propio dentro del plugin o aceptar HTTP 2xx como evidencia final. |
| V29 Deploy/notificaciones | Solo contrato. | Preview, aprobación, intento, receipt, idempotencia y reconciliación tras timeout. | Que el adapter escriba lifecycle o que un retry duplique efectos. |
| V30 OPES | Planificada; sin worktree V30 localizado. | Puertos DomainWork/OPES, instancia temporal, scope, validadores y receipts del clásico. | Compartir DB/filesystem, meter dominio OPES en core o usar rails textuales globales. |
| V31 PostgreSQL/S3/multihost | Solo contrato. | Suite de paridad, fencing, claim, afinidad, recovery y cutover. | Dual-write/fallback como segunda autoridad o rutas host durables. |
| V32 Operación/telemetría | Solo contrato. | Telemetría observe-only, watchdog cooperativo, health/readiness y shutdown idempotente. | Guardian/daemon que decida lifecycle o mate procesos directamente. |
| V33 Apps externas | Solo contrato. | Dos consumidores reales, Go y no-Go, ingress público y evidencia del mismo tree/config. | Contar scaffold como producto o generar la app dentro del núcleo. |
| V34 Cutover/acreditación | Solo contrato. | Candidate inmutable, migración única, gates, rollback y retirada verificable. | Acreditar por fixture, mantener doble config o dejar legado activo. |
| V35 Juego/build externo | Solo contrato. | Composición externa y digests de fuente, build, ROM y toolchain. | Incorporar el producto juego al núcleo. |
| V36 QA reproducible | Solo contrato. | Emulador aislado, input inmutable, frames/audio/métricas y reviews independientes. | QA in-process o evidencia reutilizada después de drift. |
| V37 Promoción gobernada | Solo contrato. | Manifiesto sellado y ledger apply/replay/rollback/demotion. | Target concreto en core, receipt sintético o retry que republica. |

## Capacidades clásicas aún aprovechables

### Prioridad inmediata

| Capacidad | Fuente útil | Situación target | Integración correcta |
| --- | --- | --- | --- |
| HOME persistente por cuenta | perfiles clásicos, `AgentHomeV0`, launcher de olas | BUG-455; implementación local pendiente de sello | Adaptar identidad opaca y exclusión vitalicia a config/adapter V2. No copiar credenciales. |
| Semántica Go de tests vacíos | `orquesta-runtime-required-test/local_command_executor_v0.go` | BUG-454: exit 0 con `[no tests to run]` puede dar falso verde | Extraer detector y negativos al atestador actual. |
| Autoprogramación gobernada | `orquesta-autoprogramming`, `BuildAutoprogrammingAutonomyProgramV0` | GOV-18 declarado | Rehacer como dominio `self_change` sobre Goal/WorkItem, sin loop ni privilegios ocultos. |
| Shutdown real y cero residuos | `ShutdownServerV0` y tests de backend/checkpoint/liveness | Parcial; OPS-16 declarado | Portar escenarios de proceso real y confirmación, no el agregado Run. |
| CPU watchdog | T259 y auditorías clásicas | ORC-25/OPS-17/18 declarados | Telemetría y parada cooperativa por puertos; nunca rail de contenido. |

### Después del cierre V23

| Capacidad | Fuente útil | Integración propuesta |
| --- | --- | --- |
| Pool multi-home y cuotas | `orquesta-capacity.AgentHomeV0`, `CapacityDecisionV0` | Router neutral por identidad opaca, cuota fresca, cooldown y handoff; sin paths en dominio. |
| Routing y escalado de modelos | `ResolveModelRoutingV0`, `ModelEscalationPolicyV0` | Política neutral por capacidad, coste, confianza y presupuesto; providers en adapters. |
| Claude/Gemini/Ollama | backends clásicos `ClaudeGoalProcessBackendV0`, `GeminiGoalProcessBackendV0`, `OllamaModelManagerV0` | Implementar adapters nuevos tras V25 y compartir normalizador neutral. |
| Rotación y handoff | `EvaluateRuntimeSessionRotationV0` | Reusar presupuesto y heartbeat; la sesión no controla lifecycle. |
| Telemetría/status/web | `orquesta-observability` y viewmodels de `orquesta-web` | Reusar criterios de honestidad y vistas; no handlers gigantes ni estado propio. |
| DomainWork neutral | `DomainWorkJobRequestV0` y contract tests | Rehacer como plugin/puerto V28; no importar módulos congelados. |
| HTTP egress | `domain-work-http.ClientV0` | Reusar allowlist, revalidación de redirects, `Retry-After` y budgets. |
| OPES REST y QA causal | `opes-connector.RESTClientV0`, jobs causales y validadores clásicos | Adaptador V30 contra instancia temporal, con aprobación y refs opacas. |
| Ingesta/OpenXML | `ExtractDocumentV0`, `IngestDataV0`, adapter OpenXML | Convertir en tools/plugins V26/V28 con fixtures de seguridad. |

### Conocimiento que sí se conserva, pero cuyo código no se porta

- Plan por olas, criterios y write-sets del Director Operativo: traducir a
  `Goal`/`WorkItem`; no restaurar `PlanState` paralelo.
- Outbox y CAS de ficheros: conservar fixtures de replay/idempotencia; V06 es
  autoridad superior.
- State/run stores JSON: conservar corrupciones y crash frontiers como pruebas;
  no reintroducir múltiples stores.
- Cierre/evidencias de la rama divergente de autoprogramación: migrar escenarios
  adversariales; V17--V19 son la implementación vigente.
- Paneles, wizard y web clásicos: rescatar UX y viewmodels; no su DB, handlers,
  sessions ni lifecycle.

## Caso HOME: por qué existía y aun así faltaba

El legado resolvía partes diferentes del problema en lugares distintos:

- el selector clásico de perfiles usaba un directorio persistente con
  `HOME=CODEX_HOME`, lo que conservaba login y refresh;
- `orquesta-capacity.AgentHomeV0` modelaba homes/cuentas con refs opacas, cuotas,
  cooldown y sesiones;
- el launcher de olas creaba homes aislados y proyectaba tooling/credenciales
  hacia cada agente;
- wrappers de runtime ya transportaban campos HOME y contabilización de uso.

La composición V22 no conectó esas piezas al adapter Codex actual y continuó
heredando el HOME global. Fue un fallo de integración y trazabilidad, no falta
de conocimiento histórico.

La solución antigua no se puede copiar literalmente porque:

- no mantenía un lock durante toda la ejecución;
- permitía que dos procesos reutilizaran el mismo perfil;
- no ligaba cuenta, journal y receipt;
- copiaba `auth.json` en algunos caminos, rompiendo la persistencia del refresh;
- no validaba completamente propietario, hardlinks, symlinks y permisos;
- no estaba en el registro canónico de configuración.

La adaptación actual conserva el comportamiento valioso —un HOME persistente
por cuenta— y añade binding opaco, lock vitalicio, validación cerrada, config
canónica y pruebas de colisión. El contrato multi-home de `orquesta-capacity`
no se sustituye: será la base del router/pool posterior.

## Problemas históricos que se convierten en reglas de integración

1. **Autoridad fragmentada:** no repartir lifecycle entre statefile, proceso,
   artefacto, adapter y proyección. Goal y su escritor siguen siendo autoridad.
2. **Stop narrativo:** `stopped` no prueba que el backend murió. Actuar sobre
   identidad exacta y verificar el proceso.
3. **Tests contaminados:** no ejecutar pruebas contra runtimes o servidores
   vivos. Usar harness privado y drain gobernado.
4. **Write-set parcial:** debe gobernar scheduler, rutas durables, efectos y Git,
   no solo el prompt del agente.
5. **Drift:** receipts deben ligar tree, binario y configuración efectiva.
6. **Falsos verdes:** presencia de código, mock, fixture o exit 0 no acredita
   capacidad ni tests útiles.
7. **Heurísticas como veto:** diff, tokens, logs o contexto son telemetría; no
   cambian lifecycle. El commit clásico `7576f60` ya corrigió este error.
8. **Config dispersa:** ninguna env, default o alias nuevo fuera del registro
   canónico.
9. **Evento por timestamp:** idempotencia por clave causal, no por instante.
10. **Fallo global:** una tarea no debe congelar otros Goals/WorkItems.
11. **Compactación destructiva:** exige gate de tamaño y evidencia.
12. **Required tests incompletos:** se derivan también de dependencias.
13. **Rutas durables incorrectas:** no versionar artefactos de ejecución ni
    escribir estado bajo fuentes.
14. **OPES dentro del núcleo:** OPES sigue siendo consumidor por puertos y refs.
15. **Código existente igual a producto:** solo acceptance ejecutable y receipt
    causal permiten declarar una capacidad terminada.

## Plan de integración recomendado

### Ola 0 — cerrar el hueco operativo actual

1. Sellar HOME persistente por perfil con pruebas focales, integración, race y
   script operativo.
2. Cerrar BUG-454 adaptando la semántica Go encontrada en el legado.
3. Repetir la suite con harness privado correcto y registrar fallos históricos
   independientes sin atribuirlos al cambio HOME.

### Ola 1 — terminar V23

1. Repositorio y aplicación durable/CAS para `internal/intake`.
2. Dossier confirmado y amendments causales.
3. Integración con registro V20, i18n V21 y plan/Goal.
4. E2E real, receipt y acreditación; ninguna sesión web como autoridad.

### Ola 2 — paralelismo después de V23

- V24, V25, V26 y V31 pueden abrirse en paralelo con write-sets disjuntos.
- V27 depende de V25+V26.
- V28 depende de V26+V27.
- V29 depende de V28.
- V32 depende de V29+V31.

### Ola 3 — consumidores y cutover

- V30 OPES usa instancia temporal y puertos ya sellados.
- V33 espera V23--V29 y V31--V32.
- V34 espera V01--V33 y evidencia completa.
- V35 -> V36 -> V37 son consumidores externos, nunca módulos de producto del
  núcleo.

## Regla operativa permanente

Toda capacidad o bug nuevo debe registrar, antes de implementar:

1. fuentes históricas consultadas;
2. solución previa más cercana;
3. incidentes y negativos asociados;
4. decisión `ya_integrado`, `adaptar` o `descartar`;
5. contrato/pruebas que demostrarán la mejora.

Si no se encuentra una mejora aplicable, no se modifica el target. Si se
encuentra, se implementa sobre la arquitectura actual y se acredita de nuevo;
nunca se reutiliza el receipt histórico como prueba del código nuevo.

## Límites de esta auditoría

- No se extrajeron backups comprimidos porque no hubo una pregunta concreta que
  justificara duplicar datos o arriesgar secretos.
- Los binarios y runtimes se conservaron como evidencia, pero no permiten
  atribuir equivalencia de fuente.
- V24--V37 siguen planificadas aunque tengan tests o fixtures históricos.
- El trabajo HOME y V23 local no cuenta como acreditado hasta commit, suite
  válida, push y receipt correspondiente.
