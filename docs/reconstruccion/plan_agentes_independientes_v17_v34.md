# Plan operativo de agentes independientes V18-V37

El nombre del fichero se conserva porque ya forma parte de evidencia sellada
V17. El contenido vigente alcanza V37; no se crea una copia paralela.

Fecha de autoridad: 2026-07-22 Europe/Madrid.

Este documento convierte el DAG de `product/roadmap.json` en una ruta de
ejecución paralela. No cambia alcance, dependencias ni contratos de aceptación.
Su objetivo es que cada vertical tenga un propietario estable, un entorno
aislado y un write-set inequívoco, sin crear otra arquitectura ni otra fuente
de verdad.

Fuentes superiores:

1. `AGENTS.md`;
2. `product/roadmap.json`;
3. `docs/reconstruccion/ruta_total_100.md`;
4. `docs/reconstruccion/estado_y_handoff_rebuild.md`;
5. este plan operativo.

Si este documento contradice una fuente superior, el agente se detiene antes
de editar y registra la contradicción. No la resuelve inventando compatibilidad.

## 1. Corte de partida

- V01-V17 están acreditados por 17 receipts V3.
- El checkpoint acreditado es el HEAD posterior al cierre V17, con cadena P/S/E
  registrada en `product/evidence/v17_test_attestor.json`.
- V18 y V20 son el ready-set vigente y usan worktrees distintos.
- Ningún agente abre implementación V18-V37 hasta que todos sus `depends_on`
  tengan receipt válido y estén integrados en el HEAD que usará como base.
- Análisis y contrato rojo pueden prepararse antes, en una rama aislada, pero
  no cuentan como vertical iniciada ni autorizan asumir APIs pendientes.

## 2. Unidad de propiedad

Una vertical es la unidad mínima asignable. Un agente propietario recibe el
vertical completo y conserva la propiedad hasta que quede integrado y sellado:

```text
análisis -> contrato rojo -> implementación -> migración/recovery -> wiring
-> focales -> race/seguridad -> E2E -> contrarrevisión -> P/S/E -> handoff
```

No se asigna “solo el adapter”, “solo SQLite” o “solo tests” como trabajo final
a otro propietario. El propietario puede crear subagentes internos para
subproblemas disjuntos, pero responde por su integración y no entrega una suma
de piezas inconexas.

Un agente no cambia de vertical por haber terminado una capa. Solo termina
cuando se cumple una de estas condiciones:

- `sealed`: receipt V3 reproducible, integrado y suite del HEAD verde;
- `awaiting_dependency`: apareció una dependencia no declarada y el
  coordinador la ratificó; el agente conserva branch, worktree y handoff;
- `external_blocked`: falta autoridad o servicio externo imprescindible, con
  tres intentos gobernados y evidencia. Un test difícil o código incompleto no
  es bloqueo.

Estados permitidos:

```text
queued -> active -> integration_ready -> sealing -> sealed
                 \-> awaiting_dependency
                 \-> external_blocked
```

`done`, “prácticamente terminado” o un porcentaje no sustituyen `sealed`.

## 3. Orquesta gobierna el programa

La superficie operativa es un Goal de programa con un WorkItem por vertical:

- `GoalRef`: objetivo estable `finish_orquesta_v37`;
- `WorkItemRef`: `V18` ... `V37`;
- `Dependencies`: copia exacta de `product/roadmap.json`;
- `WriteSet`: manifest de la sección 5;
- `RequiredTests`: contrato focal, regresiones, race/seguridad y E2E;
- `HandoffRequired=true`: el Director no cierra el padre sin ACK o bloqueo
  causal del propietario;
- un launch acreditado por propietario y vertical; un relanzamiento por caída
  conserva `WorkItemRef`, generación, branch y worktree;
- el scheduler libera un WorkItem cuando sus receipts de dependencia validan,
  no cuando otro agente dice que terminó.

El Director maximiza ready-set. No crea una cola por numeración y tampoco
ejecuta trabajo bloqueado para mantener agentes ocupados. El coordinador humano
o integrador no escribe producto en lugar del propietario salvo reparación
puntual documentada; devuelve conflicto o gate rojo al mismo propietario.

## 4. Aislamiento Git y filesystem

Cada vertical nueva usa branch y worktree propios, fuera del worktree de
integración:

```text
branch:   reconstruccion/vNN-<slug>
worktree: /home/alberto/Trabajo/orquesta-rebuild-worktrees/vNN-<slug>
base:     OID exacto del HEAD integrado que contiene todos sus receipts previos
runtime:  <worktree>/.orquesta-runtime/vNN
cache:    /srv/orquesta-self/runtime/cache/vNN
```

Reglas:

1. El agente verifica branch, HEAD, árbol limpio y receipts de dependencia
   antes de escribir.
2. Nunca trabaja en `/home/alberto/Trabajo/orquesta`; ese repositorio sigue
   congelado.
3. Nunca comparte `TMPDIR`, runtime, DB SQLite, CAS, puerto, logs o caché con
   otra vertical.
4. No hace merge, rebase, cherry-pick ni cambia el branch de integración sin
   lease del coordinador.
5. No borra worktrees, evidence, ACK, outbox o runtime ajenos.
6. Al terminar limpia solo recursos declarados en su manifest y acredita cero
   procesos propios.

V18 y V20 partirán del mismo HEAD sellado V17, pero de worktrees diferentes.
Cuando el primero se integre, el segundo será rebasado por su mismo propietario
sobre el nuevo HEAD y repetirá sus gates. El integrador no adivina cómo resolver
un conflicto semántico.

## 5. Manifest de write-set obligatorio

Antes de editar, cada propietario crea una propuesta equivalente a:

```json
{
  "vertical": "V18",
  "base_oid": "<40-hex>",
  "dependency_receipts": ["product/evidence/v17_test_attestor.json"],
  "owned_globs": [
    "internal/review/**",
    "internal/adapters/reviewer/**",
    "acceptance/v18_*",
    "docs/reconstruccion/analisis_y_contrato_v18_*"
  ],
  "integration_files_requested": [],
  "locks_requested": [],
  "forbidden_globs": [
    "product/evidence/v0*",
    "/home/alberto/Trabajo/orquesta/**"
  ],
  "required_tests": [],
  "runtime_roots": [],
  "status": "proposed"
}
```

El coordinador normaliza globs y los compara contra agentes activos. Un solape
no se arregla con “tener cuidado”: se divide por archivo o se serializa mediante
un lease. El manifest aceptado es parte del ACK del launch.

### 5.1 Superficie privada del propietario

El agente puede editar sin coordinación adicional:

- paquetes nuevos de su feature;
- adapters nuevos de su feature;
- tests focales propios;
- `acceptance/vNN_*` y su fixture;
- su análisis/contrato y su handoff de trabajo;
- fakes locales al paquete, sin crear stores, schedulers o lifecycles nuevos.

Los nombres productivos describen responsabilidad (`review`, `council`,
`commands`), no generación (`v18_core.go`) ni proveedor accidental.

### 5.2 Superficies compartidas con lease

Estos recursos nunca se editan en paralelo:

| Lease | Superficie | Regla |
|---|---|---|
| `L-AGGREGATE` | agregados existentes de `internal/goal` y `internal/application` | Preferir archivo feature nuevo; modificar archivo existente solo durante integración. |
| `L-STATE` | contrato existente de `StateRepository`, snapshot y recovery | Una ampliación atómica por vez; mismo writer y misma revisión CAS. |
| `L-SQLITE-MIGRATION` | número de migración, schema y recovery SQLite | Reservar número después de rebase; nunca dos migraciones con el mismo ordinal. |
| `L-CONFIG` | `internal/config/registry.json` y generados | Registrar primero clave, tipo, default, sensibilidad, scope y restart; ningún env fuera. |
| `L-BOOTSTRAP` | composición, `internal/bootstrap` y `cmd` | Wiring solamente; cero regla de dominio. |
| `L-BINDINGS` | registro generado y bindings HTTP/MCP/CLI | Desde V20, toda superficie pública nace del registro único. |
| `L-DEPS` | `go.mod`, `go.sum`, `vendor`, herramientas instaladas | Una actualización auditada; justificar mantenimiento y licencia. |
| `L-TRACE` | roadmap, catálogos, bugs y trazabilidad maestros | Solo integrador; el agente entrega propuesta mecánica. |
| `L-SEAL` | fixture sellada, output y receipt V3 | Una vertical por operación P/S/E. |

Adquirir un lease no autoriza ampliar alcance. El agente entrega el lease tras
compilar y deja en el handoff OID, diff y tests ejecutados.

## 6. DAG y máximo paralelismo seguro

La liberación es por eventos; las cohortes muestran el máximo ready-set si cada
corte anterior ya está sellado. No son barreras artificiales.

| Cohorte | Verticales que pueden ejecutarse en paralelo | Condición mínima |
|---|---|---|
| C0 | V17 | V01-V16 selladas. |
| C1 | V18, V20 | V17 sellada. |
| C2 | V19, V21, V26 | V18 libera V19; V20 libera V21 y V26. |
| C3 | V22, V23 | V22 espera V19+V21; V23 espera V21. |
| C4 | V24, V25, V31 | V24 espera V23; V25 y V31 esperan V22. |
| C5 | V27 | V25+V26 selladas. |
| C6 | V28 | V27 sellada. |
| C7 | V29 | V28 sellada. |
| C8 | V30, V32 | Ambas esperan V29; V32 espera además V31. |
| C9 | V33 | V24-V29 y V31-V32 selladas. |
| C10 | V34 | V30 y V33, además de todas sus dependencias, selladas. |
| C11 | V35 | V34 sellada; usa solo la superficie pública cortada. |
| C12 | V36 | V35 sellada. |
| C13 | V37 | V36 sellada. |

Cadena crítica:

```text
V17 -> V18 -> V19 -> V22 -> V25 -> V27 -> V28 -> V29 -> V30 -> V34 -> V35 -> V36 -> V37
   \-> V20 -> V21 -> V22
           \-> V26 -> V27
                  V23 -> V24 ----------------------> V33 -> V34
                         V22 -> V31 -> V32 --------/
                                      V29 -> V32
```

V17 ya está sellada. V18 y V20 pueden implementarse en paralelo porque solo
comparten esa dependencia y sus write-sets privados están aislados. Los leases
de estado/SQLite/bootstrap se serializan cuando ambas necesiten integración.

## 7. Fichas de propiedad V17-V37

La columna “raíces privadas” es el espacio preferente. Cualquier cambio fuera
de él se declara en `integration_files_requested` y exige el lease aplicable.

### V17 — Artefactos y atestador

- Dependencias: V06, V09, V15, V16.
- Capabilities: `EVD-01`, `EVD-04`, `EVD-05`, `EVD-13`.
- Raíces: `internal/adapters/artifact/filesystem`,
  `internal/adapters/attestor`, contratos neutrales de artifact/attestation y
  pruebas V17.
- Gate: sujeto exacto tree/diff/tests reproducible; traversal, symlink,
  hardlink, owner, modos, sandbox y leaks fallan; ACK nunca es PASS.
- Exclusión: autor/reviews V18 y Consejo V19.
- Estado: sellada; su worktree/runtime temporal quedó retirado. No se reabre
  salvo regresión reproducible.

### V18 — Autor, reviewers y refinery

- Dependencias: V14, V16, V17.
- Capabilities: `GOV-12`, `STG-13`, `STG-14`, `STG-16`, `EVD-06`.
- Raíces: `internal/review`, `internal/adapters/reviewer`, acceptance V18.
- Gate: autor, reviewer primario y adversarial son tres launches distintos;
  misma generación/tree/diff/tests; cualquier cambio invalida aprobación;
  rework e integración son explícitos.
- Exclusión: ballots, veto y política del Consejo pertenecen a V19.

### V19 — Consejo

- Dependencias: V12, V17, V18.
- Capabilities: `GOV-11`, `GOV-13`, `GOV-14`, `STG-06`, `STG-08`, `EVD-07`.
- Raíces: `internal/council`, adapters/policies del Consejo, acceptance V19.
- Gate: E2E separados `auto|required|skip_by_operator`; identidad del ballot
  deriva de launch acreditado; skip conserva principal, motivo, tiempo y spec
  hash; veto de seguridad y disenso quedan durables.
- Exclusión: nunca sustituye reviews V18 ni se vuelve otro Director.

### V20 — Registro único de comandos

- Dependencias: V07, V10, V17.
- Capabilities: `GOV-17`, `UI-02`.
- Raíces: `internal/commands`, generador/descriptor y adapters de binding
  dedicados; acceptance V20.
- Gate: una definición produce HTTP, MCP y CLI con mismo schema, autorización,
  claves i18n y códigos máquina; ninguna interfaz escribe lifecycle.
- Exclusión: traducciones completas son V21; UI administrativa es V24.

### V21 — i18n total

- Dependencia: V20.
- Capability: `UI-18`.
- Raíces: `internal/i18n`, catálogos, formatters y tests de paridad.
- Gate: español default/fallback, locales BCP-47, plurales, fecha, número,
  moneda y timezone; códigos máquina invariantes.
- Exclusión: ningún handler mantiene texto público privado fuera del catálogo.

### V22 — Codex real completo

- Dependencias: V04-V10 y V12-V21, salvo V11 no requerida por el E2E local.
- Capabilities: `STG-11`, `EVD-09`, `AGT-01`, `AGT-03`.
- Raíces: adapter Codex, composición E2E y fixtures/runtime aislados V22.
- Gate: MCP público recorre plan, DAG, mailbox, launch/observe/stop, workspace,
  tests, reviews y cierre; Goals A/B/C/D concurrentes sobreviven stop selectivo,
  backup/restart y shutdown; cero contradicción terminal o proceso residual.
- Exclusión: Codex es adapter, no núcleo; otros proveedores esperan V25.

### V23 — Wizard, dossier y fábrica

- Dependencias: V04, V05, V20, V21.
- Capabilities: `WIZ-01..25`, `STG-01`, `STG-03`, `STG-07`, `UI-05`.
- Raíces: `internal/wizard`, dossier, intake y templates declarativos.
- Gate: chat y formulario mutan un único intake versionado; preguntas y
  amendments causales; solo confirmación explícita crea plan.
- Exclusión: no crea segundo planificador, scheduler ni estado de Goal.

### V24 — Web administrativa

- Dependencias: V10, V20-V23.
- Capabilities: `UI-01`, `UI-03`, `UI-04`, `UI-06`, `UI-08..14`, `UI-17`,
  `OPS-07`.
- Raíces: `web`, adapter HTTP generado, assets y browser E2E.
- Gate: PWA responsive, WCAG 2.2 AA, teclado, RBAC/aislamiento, i18n y estado
  idéntico a queries canónicas; config usa expected revision, confirmación,
  audit receipt y `pending_restart`.
- Exclusión: web no decide lifecycle ni mantiene DTO/estado autoritativo.

### V25 — Hermes y proveedores restantes

- Dependencias: V12-V14, V17-V22.
- Capabilities: `ORC-07`, `ORC-26..29`, `AGT-02`, `AGT-04..12`.
- Raíces: adapters Hermes, Claude, Gemini, Ollama/local, catálogo neutral de
  modelos y suites contractuales de provider.
- Gate: contrato neutral por adapter, smoke real aislado cuando se anuncie
  disponibilidad y fallo/cuota de uno sin corromper otro Goal.
- Exclusión: no copiar loops, estado, tools ni lifecycle por proveedor.

### V26 — Tools, resources, skills y rulepacks

- Dependencias: V08, V10, V15, V20-V21.
- Capabilities: `STG-04`, `STG-12`, `EXT-09`, `TLS-01..14`.
- Raíces: `internal/tooling`, SDK público, registries de tools/resources/skills
  y rulepacks; adapters de instalación aislados.
- Gate: namespace lazy, paginación, scope/trust/hash, install/upgrade/revoke/
  rollback; herramienta no autorizada no corre y resultado grande va a CAS.
- Exclusión: una skill no obtiene autoridad de aplicación por instalarse.

### V27 — Contexto, RAG, routing y evals

- Dependencias: V17, V25, V26.
- Capabilities: `STG-05`, `ORC-15`, `ORC-19..22`, `CTX-01..12`.
- Raíces: `internal/context`, baseline FTS5/BM25, datasets y harness de eval.
- Gate: dataset versionado publica calidad/coste/latencia/recall; contexto
  mínimo por refs; embeddings, reranker o vector DB solo se aceptan si superan
  benchmark y coste de mantenimiento.
- Exclusión: no cargar corpus completo ni crear memoria global entre proyectos.

### V28 — Plugins genéricos y Forge remoto

- Dependencias: V16, V17, V21, V26, V27.
- Capabilities: `EXT-00..08`, `EXT-11`, `EXT-15..22`.
- Raíces: `internal/plugins`, `internal/forge`, adapters GitHub/GitLab/Gitea y
  conectores de investigación, web, documentos, datos y media.
- Gate: refs opacas, permisos, contrato/E2E por adapter, cero lectura de estado
  o filesystem interno; push/PR/merge son efectos con credential, egress,
  target, CAS, idempotencia y receipt exactos.
- Exclusión: un plugin no escribe lifecycle ni comparte DB con Orquesta.

### V29 — Deploy y notificaciones

- Dependencias: V08, V10, V15, V20-V21, V28.
- Capabilities: `STG-18`, `STG-19`, `UI-07`, `UI-15`, `UI-16`, `EXT-12..14`,
  `OPS-22..25`.
- Raíces: `internal/deploy`, adapters de target y notificación, fixtures V29.
- Gate: preview, aprobación, ejecución, receipt, rollback e idempotencia;
  destino fallido observable/reintentable y sink incapaz de cerrar trabajo.
- Exclusión: producción remota no ocurre sin efecto autorizado.

### V30 — OPES temporal completo

- Dependencias: V18-V22 y V25-V29.
- Capabilities: `OPE-01..20`.
- Raíces: plugin/composición OPES externa y fixture temporal V30.
- Gate: DB y FS separados de producción; inventario/reutilización antes de
  rehacer; seis subroles son launches reales cuando se exijan; lista exacta de
  artifacts/QA; trabajo parcial se conserva; publicar exige confirmación exacta.
- Exclusión: no tocar OPES productivo ni introducir reglas OPES en núcleo.

### V31 — PostgreSQL, S3 y multihost

- Dependencias: V06, V09-V10, V13-V17, V22.
- Capabilities: `OPS-11`, `OPS-13`.
- Raíces: adapters PostgreSQL, S3-compatible y host/workspace multihost.
- Gate: misma suite contractual SQLite/PostgreSQL y FS/S3; fencing, affinity,
  recuperación, aislamiento y colaboración concurrente reales.
- Exclusión: no cambiar dominio para satisfacer una DB o storage concreto.

### V32 — Operación completa

- Dependencias: V07-V11, V20-V22, V29, V31.
- Capabilities: `STG-20`, `ORC-23`, `ORC-25`, `OPS-15..21`.
- Raíces: `internal/operations`, packaging, telemetría y adapters de servicio.
- Gate: install/doctor/update/rollback/migrate/restore/retention/cleanup,
  idle-backoff, watchdog cooperativo, health/readiness y shutdown real; spoof
  rechazado y cero residuos.
- Exclusión: telemetría observa; no decide contenido ni lifecycle.

### V33 — Apps externas Go y no-Go

- Dependencias: V23-V29 y V31-V32.
- Capabilities: `GOV-01`, `GOV-18`, `APP-01..16`.
- Raíces: fixtures/apps generadas V33, contract tests y evidence de sus árboles
  o imágenes; ajustes a fábrica solo tras reproducir el fallo.
- Gate: una app Go y una no-Go pasan hexagonal, i18n, accesibilidad, config,
  secretos, auth, tests, docs, deploy y E2E públicos sobre el mismo artefacto.
- Exclusión: scaffolding sin wiring/E2E no cuenta como aplicación.

### V34 — Migración, cutover y retirada

- Dependencias: V01-V33.
- Capabilities: `STG-17`, `EVD-08`, `EVD-10` y acreditación final de las 257.
- Raíces: deliberadamente transversal y exclusiva; ningún otro agente de
  producto permanece activo.
- Gate: censo final, ingreso legacy cerrado, drain/sello, import único, config
  sin doble lectura, P0/P1 cero, 257/257 mappings probados, release/restore/
  rollback/shutdown verdes y cero writer/bridge/lease legacy.
- Exclusión: no borrar por apariencia; cada retirada necesita referencias,
  caracterización y evidencia de reemplazo.

### V35 — Videojuegos: composición y build externo

- Dependencia: V34.
- Capabilities: ninguna nueva; compone `EXT-00`, `EXT-01`, `EXT-09`,
  `EXT-20`, `EXT-21` y las garantías ya acreditadas de effects/apps.
- Raíces: fixture/plugin externo V35 y contratos de consumo público; cero
  imports del proyecto de juegos en el núcleo.
- Gate: un Goal construye una ROM reproducible con manifest, diagnósticos y
  receipts ligados a source/plugin/toolchain/config exactos.
- Exclusión: Orquesta no posee spec, engine, emulador, DB ni filesystem del
  proyecto `/home/alberto/Trabajo/juegos`.

### V36 — Videojuegos: QA reproducible

- Dependencia: V35.
- Capabilities: ninguna nueva; reutiliza artefactos, attestor, reviews, tools,
  media, contexto y plugins.
- Raíces: fixture/verificador externo V36 y evidence de emulación.
- Gate: el build sellado corre con input guionizado y produce capturas, audio,
  telemetría y reviews; cualquier drift invalida la QA.
- Exclusión: MAME/libretro/GnGeo siguen siendo adaptadores externos, nunca
  lógica del core.

### V37 — Videojuegos: promoción gobernada

- Dependencia: V36.
- Capabilities: ninguna nueva; reutiliza deploy/effects, RBAC, receipts,
  rollback y operación.
- Raíces: fixture/target temporal V37 y manifiesto de release externo.
- Gate: sin aprobación exacta no publica; apply/replay/rollback/demote son
  idempotentes y auditables sobre el mismo candidato.
- Exclusión: fabricación física, firma propietaria o targets no disponibles
  quedan plugins/efectos futuros, no deuda oculta del núcleo.

## 8. Contrato de entrega de cada agente

El handoff mínimo contiene:

```text
vertical y capability IDs
base OID y HEAD de branch
dependencias y receipts verificados
write-set real frente al manifest
commits ordenados
decisiones y simplificaciones
config keys nuevas o “ninguna”
migraciones y recovery
tests focales, race, E2E y comandos exactos
riesgos/bloqueos restantes
procesos, puertos, runtimes y temporales propios
estado: integration_ready | sealing | sealed
```

Prohibido entregar:

- código sin compilar o tests “para que otro los arregle”;
- una API asumida de dependencia no sellada;
- aliases/env/defaults fuera del registro canónico;
- duplicación temporal de store, scheduler, lifecycle, config o DTO “para
  avanzar”;
- cambios fuera de write-set no declarados;
- greens focales usados como prueba de suite o E2E total;
- documentación que afirma más de lo que acredita el receipt.

## 9. Integración y sellado

Para cada vertical:

1. El propietario actualiza su branch sobre el HEAD integrado vigente.
2. El verificador de write-set compara base→HEAD y rechaza archivos no
   declarados.
3. Se ejecutan `git diff --check`, focales, tests de dependencia, race/
   seguridad, E2E y suite proporcional desde árbol limpio.
4. Contrarrevisión comprueba arquitectura, seguridad, recuperación, config,
   simplicidad y ausencia de autoridad duplicada. Un hallazgo vuelve al mismo
   propietario; no abre una generación de opiniones infinita.
5. Se crea producto `P` sin autoacreditarse.
6. Se fijan sujetos/digests y se crea sello `S`.
7. El comando contractual corre desde checkout `detached_clean` de `S`; output
   y receipt V3 se incorporan en `E`.
8. Se integra el handoff, se verifica receipt desde HEAD y se liberan sus
   dependientes en Orquesta.
9. Solo entonces se elimina worktree/runtime privado tras comprobar procesos,
   ACK, outbox, evidence y retención.

El receipt no puede estar dentro de su propio candidato. Un `PASS` escrito a
mano, una prueba ejecutada sobre worktree sucio o un hash que no liga todos los
sujetos vale `unproven`.

## 10. Política de continuidad

- Si una sesión termina, el siguiente agente lee solo `AGENTS.md`, estado vivo,
  este plan, su contrato VNN, manifest y handoff; no necesita historial del chat.
- Si un propietario cae, Orquesta relanza la misma vertical sobre mismo
  worktree/branch y entrega refs compactas. No crea un segundo propietario
  concurrente.
- Si aparece bug, se añade al ledger con área, evidencia, hipótesis estructural,
  test y cierre; no se oculta para mantener porcentaje.
- Si una vertical crece sin control, se simplifica dentro de su contrato antes
  de seguir. No se divide creando otro núcleo, generación o capa legacy.
- El porcentaje canónico solo cambia por receipts válidos: capacidades/257,
  verticales/37 y receipts/verticales cerradas.

## 11. Próxima asignación exacta

1. Mantener V18 y V20 en worktrees separados sobre el V17 sellado.
2. Integrar cada una solo tras aceptación completa, contrarrevisión y P/S/E.
3. Al cerrar V18, lanzar V19. Al cerrar V20, lanzar V21 y V26.
4. Continuar por eventos hasta V34; desde V22, la nueva Orquesta dirige el
   ready-set por defecto.
5. Ejecutar V35, V36 y V37 como composición externa tras el cutover V34.
6. Conservar como máximo un propietario por vertical y un titular por lease
   compartido.

Este orden permite paralelismo real sin convertir la reconstrucción en una
mezcla de parches concurrentes.
