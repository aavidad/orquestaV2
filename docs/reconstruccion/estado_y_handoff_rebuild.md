# Estado y handoff vivo del rebuild

Última actualización: 2026-07-14 11:16 Europe/Madrid.

Este documento permite continuar el rebuild sin reconstruir el contexto de la
sesión. Es estado operativo, no evidencia de aceptación. Los estados canónicos
de capacidades, verticales y contratos viven en `product/roadmap.json`; los
verdes viven en receipts fuera de su propio candidato.

## Entorno que debe preservarse

- árbol antiguo, solo lectura: `/home/alberto/Trabajo/orquesta`;
- rebuild: `/home/alberto/Trabajo/orquesta-rebuild`;
- rama: `reconstruccion/orquesta-total-20260714`;
- no integrar ni ampliar paridad Claude/Gemini/Ollama/local antes de V25;
- no usar la Orquesta antigua para coordinar el rebuild;
- hasta que V22 acredite autodirección, los subagentes Codex directos son una
  excepción bootstrap y el integrador revisa cada entrega;
- no usar `codebase-memory-mcp` para este frente;
- no ejecutar `go test ./...`: atraviesa superficies legacy y puede lanzar
  smokes reales. Usar paquetes nuevos explícitos.

Snapshot de control del árbol antiguo al último checkpoint: SHA-256 de
`git status --porcelain=v1` =
`75577492db531e71f8a47f7b7fec115e79996aed45e26ce66426b4c120d42a80`.
Debe permanecer igual; si cambia, no asumir que pertenece a este rebuild.

Commits base integrados:

```text
98dc0010da feat: reconstruir Orquesta minima sobre nucleo unico
aa50856699 docs: fijar ruta ejecutable de Orquesta total
82d5643ba9 test: congelar trazabilidad mecanica del legacy
795725b880 test: acreditar autoridad unica del rebuild
f6a053aca7 feat: incorporar DAG causal y estado atomico
c89dff6c8f test: renovar E2E Codex real del DAG
ba396a88ff test: desacoplar recibo V02 del roadmap vivo
a8bff60949 test: unificar verificacion de recibos de aceptacion
```

Checkpoint inmediato: V01, V02 y V03 tienen receipts V2 verificables y sus
baterías normal, `vet` y `-race` están verdes. Una contrarrevisión detectó y
cerró falsos verdes en receipts, evidencias Markdown, contrarrevisiones task,
acreditaciones parciales, comandos planificados y la excepción sin receipt de
V01. Los generadores temporales ya no existen. Falta revisar el índice final y
commitear V03; después empieza V04. Hasta ese commit, `HEAD` sigue en
`a8bff609492f312fe2d6bf8ccccde02b6e5c8426` y el worktree grande es esperado:
no resetearlo ni regenerar los ledgers.

Contrarrevisión final: tres revisores independientes devolvieron `APPROVE`
después de reabrir y cerrar sus bloqueos de circularidad V01, evidencia real
de `BUG-020` y write-set del handoff. No queda bloqueo conocido de V03.

## Progreso honesto

- V01 catálogo ejecutable: receipt V2 válido; se reemite al cambiar roadmap.
- V02 autoridad única del rebuild: receipt V2 válido.
- V03 trazabilidad y lecciones: receipt V2 válido; commit pendiente.
- V04–V34: pendientes. No contar código heredado o una prueba aislada como
  vertical cerrada.
- progreso vertical mecánico: 3 de 34 receipts válidos, 8,8 % de la ruta;
- progreso de capacidades: 3 de 257 en estado `accredited`, 1,17 %: `GOV-03`,
  `GOV-16` y `GOV-21`. No usar porcentajes subjetivos de “núcleo funcional”.

## V03: trabajo ya realizado

Se separó el antiguo test monolítico de trazabilidad en gates focales. El censo
Go del legacy, la autoridad única y los recibos V02 permanecen verdes.

La primera revisión de 696 asignaciones de tareas contenía un falso verde:
450 confirmaciones procedían de ramas automáticas y razones enlatadas. Se
registró `BUG-REBUILD-20260714-008`, se retiró la acreditación y se revisaron
de nuevo las 696 entradas contra bloque fuente y roadmap:

```text
547 confirmed
149 corrected
0 pending
```

`product/traceability/task_semantic_review_provenance.json` conserva cuatro
pases disjuntos, sus digests, la partición exacta 696/696 y la invalidación de
los dos pases falsos. Cada `TaskEntry` independiente lleva
`semantic_review_pass_ref`; el pass entra en `semantic_review_ref` y en el
digest final. Los gates focales ya quedaron verdes.

También se repararon:

- `BUG-REBUILD-20260714-009`: los flags de emisión de bugs ya no evitan la
  validación del ledger;
- `BUG-REBUILD-20260714-010`: la autoridad de capacidades de bugs resuelve a
  `product/roadmap.json#capability_entries`;
- `BUG-REBUILD-20260714-013`: el JSON Schema se compila y valida cada JSON y
  cada fila JSONL, con negativos adversariales;
- la abreviatura `BUG-ORQ-20260710-208A-D` se expande como alias exacto a
  208A, 208B, 208C y 208D; no crea un bug agregado.

## V03: censo no circular y bugs históricos

La selección de fuentes era circular: `pending_sources.json` definía el mismo
universo que el test pretendía demostrar. Ya quedó sustituida por un censo del
filesystem de 835 Markdown:

```text
docs/**: 493
modulos/orquesta-*/docs/*.md: 323
skills/*/SKILL.md: 19
total: 835
```

El ledger inmutable contiene ahora solo las 829 fuentes históricas. Los seis
documentos vivos de `docs/reconstruccion/**` siguen censados mecánicamente como
autoridad actual, pero no se congelan dentro del ledger histórico; así este
handoff puede actualizarse sin invalidar V03 ni convertirse en evidencia de su
propio cierre.

El gate se endureció para detectar candidatos task `strict` y `broad`, no solo
`strict`. Reveló 61 Markdown legacy que antes pasaban sin role/review explícito.
Las 61 ya tienen revisión semántica durable y sus contrarrevisiones están
integradas. El censo filesystem no circular está verde con 164 fuentes task.

`BUG-REBUILD-20260714-011` y `012` están cerrados con gates filesystem no
circulares. El censo descubrió dos fuentes task adicionales, P01-P10 y S1-S10;
ambas ya entraron en `source_dispositions`, sin inventar cierre. Estado de la
autoridad de fuentes:

```text
primary task: 164
primary bug: 151
primary skill: 19
unique source dispositions: 334
bug role total en source_dispositions: 159
task+bug: 8
```

El ledger histórico de bugs también se regeneró desde todo el alcance legacy:

```text
bug sources: 159 (79 con ID literal, 80 con lesson de fuente)
rich rows: 239
occurrences: 1292
normalized IDs: 334 (197 row-covered, 137 narrative-only)
narrative review bindings: 137
```

El gate distingue tabla rica revisada de una tabla resumida: todo literal sigue
siendo occurrence, pero solo dos fuentes pueden aportar campos ricos. Así no se
inventan estado, área o hipótesis desde tablas `ID/residual/acción`.

`BUG-REBUILD-20260714-014` está cerrado. V02 y V03 usan receipt V2 con argv
estructurado, exit code, output hash, versión Go, candidato/source-tree y
timestamp. Receipt y salida capturada quedan fuera de su propio candidato.

La contrarrevisión final añadió y cerró cuatro lecciones estructurales:

- `BUG-REBUILD-20260714-017`: cada revisión Markdown queda ligada 1:1 a su
  fila, clasificación, razón y adjudicación; no basta hash+conteo;
- `BUG-REBUILD-20260714-018`: evidencia parcial no acredita una capacidad
  completa; solo `GOV-03`, `GOV-16` y `GOV-21` permanecen acreditadas;
- `BUG-REBUILD-20260714-019`: todo contrato `planned` lleva comando
  `planned:...` no ejecutable para impedir verdes sin tests;
- `BUG-REBUILD-20260714-020`: V01 ya no se acredita por excepción hardcoded;
  todo contrato ejecutable exige receipt concreto.

## V03: revisión task terminada

Write-set activo V03: `product/traceability/**`, `product/evidence/v01_*`,
`product/evidence/v02_*`, `product/evidence/v03_*`, `product/roadmap.json`,
`product_roadmap_test.go`, tests `traceability_*` de raíz, `acceptance/**`,
`scripts/check_rebuild_write_set.sh` y este handoff. No incluye `internal/**`,
`cmd/**`, `modulos/**` ni el árbol antiguo.

Contrarrevisiones ya terminadas en `/tmp`:

- `/tmp/v03_candidate_sources_review_s0.jsonl` — 41 fuentes;
- `/tmp/v03_candidate_sources_review_s1.jsonl` — 38 fuentes;
- `/tmp/v03_candidate_sources_review_s2.jsonl` — 37 fuentes;
- `/tmp/v03_tasklike_source_review.jsonl` — 21 nombres task-like;
- `/tmp/v03_acceptance_adversarial.md` — informe adversarial completo.

Las cuatro revisiones de selección ya tienen copia durable exacta en:

```text
product/traceability/fixtures/markdown_source_review_s0.jsonl
product/traceability/fixtures/markdown_source_review_s1.jsonl
product/traceability/fixtures/markdown_source_review_s2.jsonl
product/traceability/fixtures/markdown_tasklike_source_review.jsonl
```

Sus SHA-256 coinciden con los `review_ref` del censo. Las decisiones task S0 y
S1 también están preservadas en
`product/traceability/fixtures/task_candidate_review_s{0,1}.jsonl`; no dependen
ya de que sobreviva `/tmp`.

El follow-up P01-P10/S1-S10 quedó revisado 20/20 y vive en
`product/traceability/fixtures/task_candidate_review_followup.jsonl` con digest
`sha256:cbfbd0a408440a1e0e68b60d93317e7aff33c299f139098c51f96181786397c9`.
S3 vive ya en `product/traceability/fixtures/task_candidate_review_s3.jsonl`,
digest
`sha256:465815e3de09e031c1f35fdd62ef2e9b2a375744b288aa50f8af7665aaf6f586`.
S2 vive en `product/traceability/fixtures/task_candidate_review_s2.jsonl`,
digest
`sha256:67ff0a64d04bb67251e1713595df0ba95e0c51a4385d63b509ff9b9185b97de8`.

La contrarrevisión final dejó 47 fuentes nuevas con trabajo propio o mixto. El
documento `promocion_core_workflow.md` de concurrencia quedó como
`task_delegated`; `roadmap_cierre_nucleo.md` sí conserva un hueco propio en
líneas 256-257. El censo añadió además P01-P10 y S1-S10.

El scanner produjo 1.265 candidatos automáticos nuevos:

```text
primera ola: 1245 (474 strict, 771 broad)
follow-up P01-P10/S1-S10: 20 strict
shards principales: 312 + 311 + 311 + 311
```

Estado final de los cuatro shards principales:

```text
S0: 312/312, 203 entries, 109 exclusiones, terminado
S1: 311/311, 65 entries, 246 exclusiones, terminado
S2: 311/311, 74 entries, 237 exclusiones, terminado
S3: 311/311, 213 entries, 98 exclusiones, terminado
follow-up P01-P10/S1-S10: 20/20 entries, decisión durable terminada
```

Ningún fallback automático acreditó pendientes. Los bloques narrativos sin
marcador scanner se materializaron y revisaron explícitamente; `TaskEntries`
sigue siendo la única autoridad de mapping.

Los 1.265 candidatos automáticos ya están revisados 1.265/1.265. El frente
distinto revelado al endurecer el censo broad también tiene cobertura semántica
61/61, particionada 21+20+20 en estas fixtures:

```text
markdown_broad_source_review_s0.jsonl  21  sha256:fabe108cac513a312226d6335f189312c0ba8d2e644b1bf0cb4e88e545eb4844
markdown_broad_source_review_s1.jsonl  20  sha256:c9ff30ed5f8cad4bbcf26e31154bf96a497663a3710649da37c3bd3e0dcc605d
markdown_broad_source_review_s2.jsonl  20  sha256:4e5218535b2093aa8f9dca648aaab60b87f7ede08108c3fb5f12f49fbc053cc4
```

S0 y S2 son revisiones independientes. S1, los suplementos y la discrepancia
SDK ya tienen contrarrevisión independiente 28/28 en
`markdown_source_counterreview.jsonl`, digest
`sha256:2cbe467f301987f2e891331dd90991416a7242ddf9c50a22efeff42f31b1a2db`:
20 confirmaciones y 8 correcciones. SDK queda `task_delegated` porque existe
owner exacto; no se duplica. Se corrigen capacidades/rangos de control activo,
autonomía, autoprogramación, cierre del núcleo, hot store y OPES; catálogo ES
queda `supporting_contract` y el runbook de control queda `manual_mixed`.

Las adjudicaciones ya están integradas mecánicamente en el censo y en
`source_dispositions`: 829 fuentes legacy censadas, 334 disposiciones únicas y
164 fuentes task. Los gates filesystem
`TestTraceabilityRebuildLegacyMarkdownSourceRoles` y
`TestTraceabilityRebuildSourceDispositions` están verdes con conteos finales
164/151/19 y 334 disposiciones únicas.

La segunda ola automática de las 15 fuentes broad accionables produjo 238
candidatos, todos broad y ninguno strict, particionados 60+59+60+59. Los inputs
exactos viven en `task_candidate_input_broad_s{0,1,2,3}.jsonl`. Los cuatro
shards ya tienen revisión independiente; broad_s2 y el shard inicialmente
revisado por el integrador recibieron además contrarrevisión completa.

La segunda ola quedó revisada 238/238. Resultado final:

```text
broad_s0: 60 = 17 entry + 43 exclude
broad_s1: 59 = 24 entry + 35 exclude
broad_s2: 60 = 14 entry + 46 exclude, contrarrevisado
broad_s3: 59 =  8 entry + 51 exclude, contrarrevisado
total:    238 = 63 entry + 175 exclude
```

`BUG-REBUILD-20260714-015` registra una desviación real: broad_s2 declaró
validar la política, pero emitió un código inexistente y convirtió 36 hallazgos
fuera de bloques accionables en tareas. La contrarrevisión corrigió 36
`entry→exclude`, los diez códigos y amplió nueve razones de broad_s3; el cambio
exacto vive en `task_candidate_counterreview_broad_s2_s3.jsonl`.
`BUG-REBUILD-20260714-015` está cerrado: los gates permanentes de decisiones,
contrarrevisión 177/177 y bloques revisados están verdes tras la integración.

Los bloques narrativos están normalizados en 170 rangos exactos con hash:
102 quedan cubiertos por entries del scanner y 68 no tenían entry. Estos 68 se
materializaron como candidatos estrictos `reviewed_block`, shards 34+34 en
`task_candidate_input_reviewed_blocks_s{0,1}.jsonl`; ambos pases están
revisados 68/68 como entry e integrados. Ningún bloque asigna por sí mismo una
capability.

La primera partición provisional de 1.265 era 752 `entry` + 513 `exclude`. Una
auditoría de candidatos fuera de bloques marcó 345 casos: 164 strict se
conservaron por contrato y 177 broad se corrigieron de entry a exclusión tras
dos contrarrevisiones independientes; cuatro broad restantes eran trabajo
legítimo. La partición final de esa ola es 575 entry + 690 exclude.

La integración final de los once pases queda así:

```text
inputs revisados: 1571
entries nuevas: 706
exclusiones nuevas: 865
TaskEntry total: 2108
exclusiones total: 920
strict total: 1919
broad total: 1109 (189 entry + 920 exclude)
```

La aceptación V03 quedó emitida sobre 43 sujetos exactos, incluidos el guard
de write-set y su prueba conductual:

```text
fixture:          acceptance/fixtures/v03_canonical_ledgers.json
receipt:          product/evidence/v03_canonical_ledgers.json
output:           product/evidence/v03_canonical_ledgers.output.txt
fixture sha256:   634cb498b2f972fea57812034870b4ed9e42b01c2c78ea5d2f9d3b4e5807f841
candidate sha256: a73f772f0e0bac1ccc50609152648b626d715dfb220fcda232f5023ea1140c23
output sha256:    14fd3c9ee0f05dd564c03eb032cd69d7f951994711040a2a78bab93909aee8dd
executed_at:      2026-07-14T11:14:21+02:00
```

V02 fue reemitida después de documentar V03 en `acceptance/README.md`:

```text
fixture sha256:   96e39bfc0a0a039e9ac37e6bd39108f863e68e23aa1275803f3e9f5fc37984ea
candidate sha256: 31c8461d6de4b80d78c27073595ffe80c085faab4381e110982c4029639ad9b0
output sha256:    bf68ecd029e8f45c0d5da7139d815491432ccd994a37e99fc92f12e600b2f7b4
executed_at:      2026-07-14T11:14:18+02:00
```

V01 recibió evidencia propia y ya no depende de una excepción en el test de
causalidad:

```text
fixture:          acceptance/fixtures/v01_source_integration.json
receipt:          product/evidence/v01_source_integration.json
output:           product/evidence/v01_source_integration.output.txt
fixture sha256:   8c8a7d6b1e7cfebabe7c380b8bdfe325620ff94e4bd1714b0851b524faa37e0f
candidate sha256: 5fef175ab2ab210c6597966fc491140376ca88f9f9b51a3b2cf807fc155a6187
output sha256:    5bdbd566452d83272c4d9b9f11d919587cd94dba514ea1d68c68df2ed8d0ef7c
executed_at:      2026-07-14T11:14:17+02:00
```

El comando V01 también se ejecutó con su receipt retirado temporalmente y
quedó verde: el gate estructural exige la ruta declarada, no un placeholder;
el test posterior exige el fichero real y valida todos sus hashes.

`AC-V03-CANONICAL-LEDGERS` está `executable`; solo `GOV-16` queda acreditada
por V03. `ORC-23` vuelve correctamente a `declared` y pertenece a V32
`operations_telemetry`. Ningún estado histórico se interpreta como cierre del
rebuild.

`task_candidate_review_provenance.json` liga los once inputs y decisiones con
SHA, pass, reviewer y digests. `task_entries.jsonl`, exclusiones y uniones de
fuente ya están regenerados. Los gates de candidato, provenance, bloques,
schema, task entries, source dispositions y censo Markdown están verdes.

La corrección 177/177 vive en
`task_candidate_counterreview_original_outside_blocks.jsonl`, digest
`sha256:246ff54c0daca14c1e22760a30b54e4b704b45f5dc00a18c30c7471c73e70966`.
La contrarrevisión se divide 75+102 y no convierte ningún strict. Las decisiones
finales S0-S3 tienen digests
`sha256:e42886c59d9e43764732d281b4da2acf6774e3818ad3815eaeff55fc739505c4`,
`sha256:204ed22c12ee82c076fd90d2f64be8128469a92b84f9a51b0b4118f7633c0dd0`,
`sha256:67ff0a64d04bb67251e1713595df0ba95e0c51a4385d63b509ff9b9185b97de8`
y `sha256:465815e3de09e031c1f35fdd62ef2e9b2a375744b288aa50f8af7665aaf6f586`.

También se preservaron los inputs exactos en
`product/traceability/fixtures/task_candidate_input_{s0,s1,s2,s3,followup}.jsonl`
(312+311+311+311+20). La integración/provenance ya no depende de regenerar
shards desde `/tmp` ni de conservar el orden implícito de una sesión.

Diseño acordado para bloques narrativos, evitando doble autoridad: normalizar
cada rango revisado en `reviewed_task_blocks.jsonl`. Si una o varias
`TaskEntries` del scanner cubren el rango, el bloque solo referencia esos
`candidate_ref`; si ninguna entrada lo cubre, se crea un único candidato
`reviewed_block` exacto y se revisa semánticamente. El bloque nunca asigna una
capability por sí mismo: `task_entries.jsonl` sigue siendo la única autoridad
task→capability. El gate debe probar que todo bloque termina cubierto por entry,
nunca solo por una exclusión.

Nueve IDs históricos nuevos quedaron integrados y revisados:

```text
208K 208M 208N 208T 208U 208AB 208V 208W 208X
```

Orden inmediato de integración:

1. contrarrevisar el diff final y los tres receipts sin editar candidatos;
2. repetir `git diff --check` y guard de write-set si la revisión cambia algo;
3. commitear V03 sin incluir generadores temporales;
4. registrar aquí el SHA del commit en un checkpoint documental;
5. comenzar V04 creando primero su test de aceptación rojo: `IntentManifest`
   exacto, `AppSpec` normalizado, amendments
   causales y nueva generación de Goal.

No queda ningún `v03_*_tmp.go`. No regenerar ledgers ni reabrir revisiones salvo
regresión reproducible.

Los inputs y decisiones de los cinco pases de 1.265 candidatos ya tienen copia
durable en `product/traceability/fixtures/task_candidate_{input,review}_*.jsonl`.
No dependen de `/tmp`. Schema y gates ya validan provenance, decisiones
duplicadas/ausentes/fuera de scope y códigos de exclusión contra la política
canónica.

El ciclo de digest de este handoff está resuelto: los seis documentos vivos de
`docs/reconstruccion/**` se censan desde filesystem como autoridad rebuild, pero
no se congelan dentro del ledger histórico inmutable. Un rojo de source roles
debe corresponder a una fuente legacy no revisada o a drift real, no al acto de
actualizar este relevo.

## Comandos seguros

```bash
cd /home/alberto/Trabajo/orquesta-rebuild
go test -mod=vendor -count=1 . -run '^TestTraceabilityRebuild'
go test -mod=vendor -count=1 . -run '^TestProductRoadmap.*$'
go test -mod=vendor -count=1 ./acceptance -run '^TestAcceptanceV0[123]'
go test -mod=vendor -count=1 . ./internal/... ./cmd/orquesta ./acceptance
GOFLAGS=-mod=vendor go vet . ./internal/... ./cmd/orquesta ./acceptance
go test -race -mod=vendor -count=1 . ./acceptance -run '^(TestTraceabilityRebuild.*|TestProductRoadmap.*|TestAcceptanceV0[123].*)$'
git diff --check
scripts/check_rebuild_write_set.sh
```

Último resultado antes de este checkpoint: todos los comandos anteriores
`PASS`; guard `rebuild_write_set_ok` con 2.431 rutas desde su base congelada.
`BUG-REBUILD-20260714-016` conserva el fallo stale del allowlist y su prueba
conductual de aceptación/rechazo; `BUG-REBUILD-20260714-017` a `020` conservan
los falsos verdes encontrados por contrarrevisión. El árbol antiguo conserva exactamente
`75577492db531e71f8a47f7b7fec115e79996aed45e26ce66426b4c120d42a80`.

Antes de cualquier commit:

```bash
git -C /home/alberto/Trabajo/orquesta status --short
git -C /home/alberto/Trabajo/orquesta-rebuild status --short
```

El primer comando debe seguir mostrando únicamente los cambios previos del
operador. No limpiar, resetear ni incorporar ese árbol.

## V04 ya decidida

V04 implementa solo `GOV-02` sobre el núcleo único:

- `IntentManifest`: entrada exacta e inmutable;
- `AppSpec`: normalización inmutable, generación, parent ref/hash, motivo,
  confirmación explícita y hashes;
- amendment: nuevo Intent + AppSpec generación N+1 + Goal sucesor; nunca
  reescribe historia ni reutiliza evidencia;
- Goal porta AppSpec; no se duplica Intent/spec hash en estados hijos;
- launch/receipt/observation del proveedor ecoan `spec_hash` y cualquier
  mismatch se rechaza;
- MCP create/amend exige `confirm:true` y principal/tiempo del servidor;
- SQLite migra con mapping único y triggers de inmutabilidad;
- V04 no abre UI, multiusuario, Hermes ni proveedores aplazados.

## Regla de actualización

Actualizar este handoff al cerrar cada bloque material o antes de terminar una
sesión. No copiar aquí recibos completos ni convertirlo en una segunda fuente
de estado: resumir hecho, test, bloqueo y siguiente acción, y enlazar siempre
al ledger/contrato canónico correspondiente.

## Protocolo de relevo rápido

Un agente nuevo debe, en este orden:

1. leer `/home/alberto/Trabajo/orquesta-rebuild/AGENTS.md` y este handoff; el
   `AGENTS.md` antiguo solo aporta restricciones de lectura del legacy;
2. trabajar solo en `/home/alberto/Trabajo/orquesta-rebuild` y confirmar rama;
3. ejecutar `git status --short` en ambos árboles sin limpiar ninguno;
4. comprobar agentes vivos y resultados parciales antes de relanzar trabajo;
5. verificar el último receipt acreditado y no inferir cierre desde código;
6. continuar desde la primera acción pendiente de este documento;
7. actualizar este relevo tras cada bloque material y antes de abandonar sesión.

Formato mínimo de checkpoint: `hecho`, `evidencia/pruebas`, `estado rojo`,
`bloqueo/riesgo`, `siguiente acción exacta`. Una decisión arquitectónica nueva
debe entrar primero en el contrato o roadmap canónico correspondiente; este
documento solo registra su consecuencia operativa.
