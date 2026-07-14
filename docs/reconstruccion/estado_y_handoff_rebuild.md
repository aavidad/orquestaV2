# Estado y handoff vivo del rebuild

Última actualización: 2026-07-14 17:19 Europe/Madrid.

Este documento permite continuar el rebuild sin reconstruir el contexto de la
sesión. Es estado operativo, no evidencia de aceptación. Los estados canónicos
de capacidades, verticales y contratos viven en `product/roadmap.json`; los
verdes viven en receipts fuera de su propio candidato.

## Checkpoint vigente: V06 cerrado

V06 está cerrado funcionalmente y con evidencia reproducible. V07 no está
abierto. El próximo agente debe partir del `HEAD` que contiene este documento y
no reanalizar, reimplementar ni resellar V01–V06 salvo regresión reproducible.

V06 es el límite del núcleo transaccional mínimo, no el final de Orquesta. Cierra
exactamente nueve capacidades: `EVD-02`, `GOV-05`, `GOV-06`, `OPS-09`, `OPS-10`,
`OPS-12`, `ORC-12`, `ORC-13` y `ORC-17`. `EVD-01` pertenece a V17, `GOV-17` a
V20 y `OPS-13` a V31; siguen declaradas, no acreditadas. El total queda en
18/257 capacidades, 7,00 %, y 6/34 verticales, 17,6 %.

Resultado funcional del núcleo V01–V06:

- existe un único lifecycle autoritativo `Goal -> WorkItem`; fases, scheduler,
  ejecuciones y proyecciones no pueden cerrar otro estado paralelo;
- un `WorkItem` conserva DAG, lineage, write-set y requisitos neutrales y puede
  tener una cadena 1:N de `ExecutionRecord` reemplazables y monotónicos;
- intento de ejecución de proveedor e intento de entrega outbox son conceptos
  distintos; capacidad temporal o polling no gastan un intento de ejecución;
- CAS, lease, reloj del repositorio, fence monotónico, outbox y receipt inmutable
  impiden doble efecto, rewire de identidad y consumo con fence obsoleto;
- provider/model/agent/external, generaciones y spec hash quedan causalmente
  ligados al launch receipt y sobreviven restart;
- el scheduler es neutral y selecciona trabajo durable por requisitos opacos;
  no contiene una cola privada ni routing Codex en el núcleo;
- SQLite WAL es el adaptador local por defecto a través de un único puerto de
  estado. Schema 5 preserva V4 con doble sentinel solo para observación legacy;
- fake, Codex, SQLite, MCP y bootstrap implementan el mismo contrato. Codex V4
  conserva compatibilidad local V3 sin relanzar terminales ni contaminar core;
- el E2E real crea y cierra un Goal por servidor/API MCP, ejecuta Codex real y
  lee el artefacto con marcador exacto.

Decisiones que no deben revertirse al continuar:

- `scheduler.max_action_attempts`, `ExecutionRecord.MaxAttempts` y
  `Dependencies.MaxActionAttempts` se retiraron porque mezclaban dos políticas
  y ya no gobernaban nada. La entrega sigue reintentable y no terminal por un
  contador inventado; límites globales de tiempo, coste y efectos pertenecen a
  V15;
- `DefaultModelRef` es un selector lógico estable del adaptador, no una falsa
  atestación del nombre físico que el proveedor resuelva internamente;
- no crear un segundo lifecycle, scheduler específico de proveedor, DB global
  concreta ni acceso directo a entorno fuera del registro canónico.

Cadena autoritativa V06:

```text
base confiable V05: 238ebc59025d3dd2bdd9593febd878650af6a660
contrato rojo B:     00b760c5989300c205eda9d3a12e0a86d1f096c2
producto P:          57ad1986255af99f60bca1a9c59d6da1772adf77
sellado C:           dc54f283919da15195cf026bad76677f1cf418b2
tree C:              5a4d9e137497cb28cc23d84fa2741ab9424ccabd
evidencia E:         d912719a2883738bad6a48770a75edc03406bb14
candidate SHA:       sha256:845ad6413a9d32794567bc7fc3c202ce01d1ab993812461ce5da250d9ca7da0c
fixture SHA:         sha256:b5da42272d6df06ebb3d3a1dd65d700a48a88c22b6cc6741af9d1a3807fb1678
output SHA:          sha256:448ac06eacfef2c5ab2288cdd47050f05b1b6eb6003d8ec1768a3a5b1966ea29
```

El receipt V3 se emitió ejecutando el argv contractual desde `C` en checkout
detached, limpio y con status vacío. Los 50 sujetos coinciden exactamente con
el delta `B..P`; receipt y salida quedan fuera del candidato. Los receipts
V01–V06, normal, `-race`, `vet`, roadmap causal, inventario de bugs,
`diff-check` y contrato V06 quedaron verdes.

Los bugs `BUG-REBUILD-20260714-043` a `058` permanecen cerrados en el ledger
con causa, invariante y test de lección. La última contrarrevisión repitió los
cuatro bloqueos SQLite originales y confirmó 0 P0/P1: Kind causal exacto,
identidad provider write-once, cronología de replacement y rollback post-staging.
La contrarrevisión completa fuera de SQLite también devolvió 0 P0/P1.

El E2E Codex real vigente está ligado a source digest
`sha256:debe6822990df9ab8de14c5c0a1a390d92389584ec34e5d43c5149ed1e8dfca3`
y marcador `ORQUESTA_CODEX_E2E_OK_1cfb920052a4bbe2494717f7699e7f4b`.
Cualquier cambio posterior en código activo de proveedor invalida el receipt y
obliga a repetir el E2E.

Uso honesto actual: Orquesta puede recibir por MCP un DAG declarado, persistirlo
atómicamente, paralelizar ready items sin conflicto, ejecutar Codex real,
reemplazar intentos fallidos dentro de política y cerrar con evidencia. Aún no
transforma por sí sola una petición abierta en investigación, plan, subagentes,
reviews, replan e integración: eso culmina en V22. V07–V15 construyen las capas
operativas alrededor del núcleo; V16 inicia workspace/Git para programación
externa y V22 acredita programación externa gobernada y autoservicio Codex.
Hasta V16, el uso desde un agente director y la integración manual se rigen por
`docs/reconstruccion/runbook_agente_director_v06.md`; `write_set`, skills y tools
del worker son metadata, no acceso al repositorio ni permisos efectivos.

## Checkpoint histórico de V01–V05

La incidencia longitudinal de receipts quedó cerrada como
`BUG-REBUILD-20260714-033`. V2 recalculaba fixture y candidato desde el worktree
vivo y podía invalidar verticales anteriores al cambiar roadmap, tests o ledger.
V3 liga ruta, modo y contenido de cada sujeto a blobs de un commit Git sellado;
exige `base -> sealed -> HEAD`, tree/fixture/candidate hashes, commit de ejecución
idéntico, timestamp posterior, checkout `detached_clean`, status vacío y
receipt/salida regulares. El adversarial cubre drift vivo, downgrade, commit
hermano, tree/blob/SHA, argv, modos, paths, symlinks y manipulación de salida.

Cadena de emisión cerrada:

```text
C candidato: e68ad92280e0b60a78bcb191909734fdc5d224e3
tree C:      e6f30fa3d3b254414aeb001476ef36e1cc64e2df
E evidencia: 70dbab89e3ad36d1e3b5d2226e1eac4f0b00ede3
```

Protocolo y argv V01–V04 se ejecutaron en worktree temporal detached, limpio y
exactamente en `C`; el postflight conservó el mismo HEAD y status vacío. Los
cuatro receipt tests, `./acceptance` completo, root focal, race focal, vet,
diff-check y write-set quedaron verdes. Dos contrarrevisores independientes
devolvieron `ACCEPT`. El delta `C..E` contiene solo siete artefactos de evidencia;
la salida V01 no aparece porque la nueva ejecución produjo bytes idénticos al
blob ya versionado.

No hay código V05 abierto. Sus tres inventarios de solo lectura están resumidos
en la sección de reanudación; no repetir el análisis ni reescribir el DAG base.

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
cc571a089f test: acreditar trazabilidad canónica y receipts V01-V03
99eb627ce3 docs: registrar cierre V03 y relevo V04
defdbb7a74 docs: congelar análisis y relevo de V04
d313ae5183 test: abrir contrato rojo de intención y AppSpec
cd674ab964 feat: materializar nucleo V04 AppSpec
20e53d39f8 feat: persistir cadena AppSpec en SQLite
4e204c44df docs: fijar handoff tras cierre SQLite V04
56f1b30351 fix: vincular motivo inicial V04 al contrato
1a07ec4f3f docs: fijar relevo durante MCP V04
dc0533caf6 feat: exponer cadena AppSpec por MCP
e4c5f23074 test: separar cobertura MCP V04 por responsabilidad
a301a3bbac fix: integrar cierre ejecutable V04
33c0072b89 test: congelar candidato integral V04
9a6d255a08 test: acreditar cadena integral V04
3d6f1164ee docs: registrar bloqueo longitudinal de receipts
e68ad92280 test: sellar receipts contra blobs Git
70dbab89e3 test: emitir receipts V3 sellados
1a9ce60df7 docs: fijar relevo V3 y análisis V05
a7d5774086 feat: cerrar nucleo DAG y fases V05
81287490b4 test: sellar candidato integral V05
a1d4b77907 test: acreditar DAG y fases V05
fb38152787 fix: cerrar contrarrevision causal V05
51af2ed280 test: resellar evidencia causal V05
238ebc5902 docs: registrar cierre y uso honesto V05
00b760c598 test: abrir contrato atomico V06
57ad198625 feat: implementar estado atomico V06
dc54f28391 test: sellar delta V06
d912719a28 test: acreditar evidencia reproducible V06
```

Checkpoint histórico: `70dbab89e3` preservó el cierre funcional de V04 y migró
V01–V04 a receipts V3 inmutables. V04 acredita solo `GOV-02`; `GOV-01`
continúa declarado y sin evidencia. No queda cambio de producto ni evidencia
pendiente de commit antes de V05; este handoff se integra por separado.

Dos contrarrevisores finales independientes devolvieron `ACCEPT` sin editar:

- sellado: delta congelado `d313ae5183 -> a301a3bbac`, 60 rutas de delta y
  cuatro extras contractuales, 64 sujetos exactos, cero faltantes;
- comportamiento: cadena `IntentManifest -> AppSpec -> Goal`, fake, Codex,
  SQLite, MCP, i18n, bootstrap y E2E Codex real; normal, race sin filtro, vet,
  receipts, diff-check y write-set verdes;
- bugs `BUG-REBUILD-20260714-022` a `033` cerrados con lección ejecutable;
- ningún proceso del rebuild quedó vivo. El runtime temporal
  `.orquesta-runtime/v04-real-codex` se eliminó después de preservar el receipt;
- existe un servidor legacy ajeno en la sesión tmux
  `orq-live-bug255-replay13`. Pertenece al árbol antiguo: no detener ni limpiar
  desde este rebuild.

Digests de candidato acreditados:

```text
V01  sha256:1f1bd1813fc24e0b985f716bdc2da6dbb37b11fdf9ff477ccd67a1d3cfca2867
V02  sha256:de30c14f63ecd32b35f7ee04e3a40af944c3b0966a984c5d877bcde948cb3567
V03  sha256:c8d5ea1dafd28c04a69d0568ecdd283daa114a8dc57cdaead077d2b22e15de85
V04  sha256:803f6182f2dda6d7b45f6289ce6d49a26202968c54baf69cc30016a9a9c7aabb
V05  sha256:952230df433d87f9f9285a7d6a5bc68cd3d4875f5d3ec5dd60adfbe3020709f7
V06  sha256:845ad6413a9d32794567bc7fc3c202ce01d1ab993812461ce5da250d9ca7da0c
```

Receipt V04: fixture
`sha256:def3544f2dc84744dffe54592565c373f00c54ade326ca61e0bd1c6319173e41`,
output
`sha256:9ed9b3d5b36d7c3f2128a1bbd0ba92eb6ae2ac4bb8e834e6e4dc64877e5605bc`,
ejecutado `2026-07-14T12:01:11.167Z`. Commit, tree y blobs de `C` son identidad
autoritaria; no se recalculan desde el worktree actual.

Verificación rápida sin atravesar superficies legacy:

```bash
go test -mod=vendor -count=1 ./acceptance -run '^TestAcceptanceV0[1-6].*Receipt$'
git diff --check
scripts/check_rebuild_write_set.sh
```

## Progreso honesto

- V01 catálogo ejecutable: cerrado; receipt V3 válido.
- V02 autoridad única del rebuild: receipt V3 válido.
- V03 trazabilidad y lecciones: cerrado; receipt V3 válido.
- V04 intención, AppSpec y amendments: cerrado; receipt V3 válido; `GOV-02`
  acreditado con evidencia exacta.
- V05 DAG y fases neutrales: cerrado; receipt V3 válido; acredita solo
  `GOV-04`, `ORC-01`, `ORC-02`, `ORC-06` y `STG-00`.
- V06 estado, outbox y scheduler atómicos: cerrado; receipt V3 válido; acredita
  exactamente `EVD-02`, `GOV-05`, `GOV-06`, `OPS-09`, `OPS-10`, `OPS-12`,
  `ORC-12`, `ORC-13` y `ORC-17`.
- V07–V34: pendientes. No contar código heredado, groundwork o una prueba
  aislada como vertical posterior cerrada.
- progreso vertical cerrado: 6 de 34, 17,6 % de la ruta; receipts válidos: 6
  de 6 contratos ejecutables;
- progreso de capacidades: 18 de 257 en estado `accredited`, 7,00 %:
  `EVD-02`, `GOV-02`, `GOV-03`, `GOV-04`, `GOV-05`, `GOV-06`, `GOV-16`,
  `GOV-21`, `OPS-09`, `OPS-10`, `OPS-12`, `ORC-01`, `ORC-02`, `ORC-06`,
  `ORC-12`, `ORC-13`, `ORC-17` y `STG-00`.

## Siguiente acción exacta

Parar: V06 está cerrado. No abrir V07 ni comenzar programación externa en esta
sesión sin nueva orden del operador. Cuando se reanude, abrir solo el contrato
rojo V07 `config`: configuración mutable, `effective_config` y doctor sobre el
registro canónico existente. No reabrir el núcleo V06 ni introducir entorno,
credenciales, proveedor, HTTP o persistencia concreta dentro de Goal/workflow.

Los subagentes directos siguen siendo bootstrap hasta V22: para otro proyecto,
la nueva Orquesta solo puede coordinarlo hoy si el consumidor entrega un DAG
declarado. Una petición abierta necesita todavía dirección externa.

## Historial de ejecución (no sustituye el checkpoint vigente)

Las palabras “pendiente”, “siguiente” o “en curso” dentro del historial
describen checkpoints pasados. No son órdenes de reanudación. La única acción
vigente es parar tras V06; V07 requiere nueva orden del operador.

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
candidate sha256: c4446c90244153635adaa69a212f6274c4b3b2dadba6080a82269c8e6ac2a9dd
output sha256:    83c7c69f702f81a38f74a6f9feae09ad364927f85c813b5eb605df6178113780
executed_at:      2026-07-14T13:09:29+02:00
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
candidate sha256: 2bfabf2357b14d9ad6ff09edfeb54f9e2a9847699bf551a1d522aba901bfb1e4
output sha256:    30f3bf6dc3248ee651b3877eb1ead1a4f62983d263b1080ec43e4c17de881df4
executed_at:      2026-07-14T13:08:56+02:00
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

1. crear `acceptance/fixtures/v04_intent_appspec.json` y
   `acceptance/v04_intent_appspec_test.go` con fallos conductuales concretos;
2. cambiar `AC-V04-INTENT-APPSPEC` de `planned` a `executable` solo cuando el
   test y fixture existan; no acreditar capacidades ni emitir receipt todavía;
3. implementar de dentro afuera `IntentManifest` y `AppSpec` inmutables,
   generaciones/amendments causales y propagación de `spec_hash`;
4. añadir persistencia/migración SQLite y superficies MCP como adaptadores,
   manteniendo `internal/goal` y aplicación libres de HTTP, SQL y provider;
5. cerrar V04 con receipt propio, contrarrevisión y actualización de este
   handoff.

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

### Checkpoint de análisis V04 antes de programar

Tres revisores independientes inspeccionaron dominio/aplicación, SQLite/MCP y
aceptación/roadmap. Ninguno editó el árbol. Consenso integrado:

- reutilizar `IntentManifest` actual; no crear otro manifest ni otro lifecycle;
- añadir `AppSpec` inmutable con `ref`, generación, Intent ref/hash, parent
  ref/hash, objetivo normalizado, motivo, principal/fecha de confirmación y
  hash canónico;
- Goal porta AppSpec y deriva desde él Intent/spec hash; `GoalRecord`,
  WorkItem, Execution y Artifact no mantienen copias autoritativas;
- primera versión de amendment acepta solo Goal fuente terminal. Un Goal
  pendiente o activo se rechaza hasta que V14 aporte cancelación/supersesión;
- amendment crea nuevo Intent, AppSpec N+1 y Goal sucesor vacío. El padre y
  toda su evidencia quedan intactos;
- `spec_hash` solo cruza la frontera provider en launch, receipt y observation;
  un mismatch se rechaza antes de crear artefacto, atestación o cierre;
- `confirm:false` no escribe nada. MCP toma principal y tiempo del servidor y
  no acepta que el cliente los suplante;
- SQLite debe persistir la cadena, sobrevivir restart y bloquear update/delete
  directo de Intent/AppSpec. El backfill calcula hashes con código Go; no
  inventa hashes en SQL;
- V04 incluye fake provider, SQLite y MCP offline porque sin esos adaptadores no
  puede acreditarse extremo a extremo. Codex real, UI, multiusuario, V05,
  registry V20 y proveedores restantes siguen fuera.

La discrepancia entre revisores quedó resuelta: una propuesta quería aplazar
SQLite/MCP, pero eso solo permitiría marcar dominio `implemented`, no GOV-02
`accredited`. Se conserva el alcance extremo a extremo ya fijado por el
handoff y el roadmap.

Contrato rojo acordado:

```text
acceptance/fixtures/v04_intent_appspec.json
acceptance/v04_intent_appspec_test.go
```

Debe cubrir Intent exacto/hash, confirmación sin escritura, create/replay,
conflicto idempotente, amendment causal, padre intacto, rechazo de fuente no
terminal/CAS/scope, mismatch `spec_hash`, restart SQLite, triggers de
inmutabilidad y MCP sin spoof de principal/tiempo.

Comando focal seguro decidido:

```bash
go test -mod=vendor -count=1 ./acceptance ./internal/goal ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/state/sqlite ./internal/interfaces/mcp -run '^(TestAcceptanceV04IntentAppSpec|TestV04.*)$'
```

Estado exacto de relevo: V04 sigue sin empezar en código y GOV-02 permanece
`declared`. Primera acción pendiente: crear fixture/test compilables y rojos;
después cambiar `AC-V04-INTENT-APPSPEC` a `executable`, declarar su receipt
canónico y mantener solo tres GOV acreditados. Orden de implementación:
dominio AppSpec -> aplicación/amendment/fencing -> ports/fake -> SQLite -> MCP ->
focal/race -> sellado/receipts -> contrarrevisión. Al cerrar V04 se emite V04 y
se reemiten V01 y V03 porque roadmap y su gate cambian; V02 solo se reemite si
cambia su candidato.

Excepción operativa: se usaron tres subagentes Codex directos para análisis
solo lectura porque la autodirección del rebuild no queda acreditada hasta V22.
Esto cumple el bootstrap documentado; no se interpreta como diseño final ni
como sustituto de Orquesta.

### Checkpoint rojo V04

Ya existen el fixture y el gate estructural/contractual:

```text
acceptance/fixtures/v04_intent_appspec.json
acceptance/v04_intent_appspec_test.go
```

`AC-V04-INTENT-APPSPEC` está ahora `executable`, declara receipt canónico pero
GOV-02 sigue `declared`. El fixture liga inputs exactos, ocho invariantes y el
write-set candidato. La parte reutilizable de Intent está verde: conserva bytes
exactos, hash determinista y sensibilidad a cualquier cambio de entrada.

Rojo reproducible ejecutado:

```bash
go test -mod=vendor -count=1 ./acceptance ./internal/goal ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/state/sqlite ./internal/interfaces/mcp -run '^(TestAcceptanceV04IntentAppSpec|TestV04.*)$'
```

Resultado: falla solo `TestAcceptanceV04IntentAppSpec`; los otros seis paquetes
quedan verdes sin tests V04 todavía. Ausencias agrupadas: AppSpec/autoridad
única de Goal; confirmación y amendment de aplicación; `SpecHash` en tres
mensajes provider; migración 003; tool MCP amend y campos confirm/fencing. No
hay fallo de compilación, JSON ni fixture.

`BUG-REBUILD-20260714-021` queda registrado y cerrado: el comando planned V04
omitía `./acceptance` y usaba un nombre genérico inexistente. El nuevo gate
`TestProductRoadmapExecutableCommandsRunDeclaredTestPackage` exige que todo
contrato ejecutable ejecute el paquete propietario de su `test_ref`. Estos
gates están verdes:

```bash
go test -mod=vendor -count=1 . -run '^(TestProductRoadmap.*|TestTraceabilityRebuildBugLessons)$'
```

Revisiones operativas: dominio/aplicación y SQLite entregaron análisis
solo-lectura; la tarea delegada del fixture se interrumpió sin cambios porque
intentaba abarcar un harness demasiado grande. El integrador creó el contrato
rojo mínimo y documentó la excepción; no se perdió ni mezcló ningún write-set.

Siguiente write-set autorizado:

```text
internal/goal/{app_spec.go,app_spec_test.go,goal.go,goal_test.go,refs.go,restore.go,restore_test.go,snapshot.go}
```

Objetivo: AppSpec inicial/amend, hashes/generaciones, Goal sucesor vacío y
snapshot tamper-proof. Tras verde focal de dominio, actualizar este handoff
antes de abrir aplicación.

### V04 vivo posterior al contrato rojo

Checkpoint seguro `56f1b30351` (`fix: vincular motivo inicial V04 al contrato`)
contiene dominio, aplicación, ports, fake, Codex y SQLite, con bugs 023-028 y
contrarrevisiones cerrados:

- dominio: AppSpec inmutable, hash framed, N+1 causal, Goal sucesor terminal,
  snapshot schema 2 y negativos de tamper/self-parent. Verde normal y `-race`
  en `./internal/goal`, revisado por el integrador;
- provider: `SpecHash` obligatorio y canónico en launch/receipt/observation;
  fake y Codex lo ecoan desde request durable. Codex sube su schema local de 1
  a 2 y falla cerrado ante registros antiguos sin hash. Verde normal y `-race`
  en ports/fake/Codex;
- aplicación: Submit confirmado, GoalRecord sin Intent duplicado, amendment
  atómico por nuevo método de repositorio y fencing antes de persistir
  evidencia. Verde normal/race/vet, rechazo pre-IDs de fuente no terminal,
  unicidad de sucesor y campos AppSpec de `GoalSummary`;
- SQLite: schema V3 `Intent -> AppSpec -> Goal`, backfill canónico, rollback,
  inmutabilidad, amendment atómico y lectura/restart. Bugs 026/027 cerrados;
  normal/race/vet y contrarrevisión verdes. No se han abierto MCP ni bootstrap.

Contrarrevisión independiente rechazó dos falsos verdes de aplicación. Ambos
quedaron corregidos y registrados como `BUG-REBUILD-20260714-023` y
`BUG-REBUILD-20260714-024`:

- mismatch de `spec_hash` cerraba Goal como `failed`; ahora todo hash requerido,
  inválido o distinto pone en cuarentena la acción, deja Goal no terminal y no
  persiste artefacto, atestación ni cierre;
- `created=true` del repositorio aceptaba un Goal semánticamente parecido pero
  con ref/hash/tiempo/snapshot sustituido; create/amend comparan ahora snapshot
  y executions contra el candidato exacto y exigen ausencia de evidencia
  inesperada. Replay `created=false` conserva validación semántica.

Los tres tests de cierre exigidos están verdes. También están verdes normal,
`-race` y `go vet` sobre dominio, aplicación, ports, fake y Codex. Segunda
contrarrevisión confirmó ambos arreglos y detectó
`BUG-REBUILD-20260714-025`: validación de receipt clasificaba hash vacío o no
canónico como mismatch por comprobar igualdad demasiado pronto. Ya está cerrado:
orden `required -> invalid -> mismatch`, cobertura de cuarentena para las tres
ramas y focal/race/vet verdes. No quedan bloqueos de esa contrarrevisión.

Revisión de integración SQLite detectó y cerró
`BUG-REBUILD-20260714-026`: un trigger exigía `confirmed_by == Intent.actor`,
contrato más estrecho que dominio y futuro rol revisor. El trigger conserva
guarda temporal/integridad pero acepta revisor autenticado distinto; test de
round-trip SQLite verde.

Contrarrevisión SQLite completa rechazó el bloque por
`BUG-REBUILD-20260714-027`: backfill V2 validaba Intent pero no restauraba el
Goal completo antes de emitir receipt 3. Un Goal SQL-válido y dominio-inválido
podía dejar `Open` verde y fallar en `GetGoal`. Corrección en curso con el mismo
agente: rojo adversarial y reconstrucción canónica de cada agregado dentro de
la transacción, antes de receipt/user_version; fallo debe restaurar V2 íntegro.
Rojo reproducido por integrador: `Open` aceptó Goal `running` con `closed_at`.
Ya está cerrado: test adversarial verde; validación post-swap/pre-receipt usa
lector canónico + `goal.RestoreGoal` y comprueba bindings de executions,
artifacts y attestations. Rollback conserva V2, sin receipt 3 ni tablas staging.
Mismo contrarrevisor dio ACCEPT; SQLite normal/race/vet verdes. Próximo frente:
MCP create/amend y proyecciones AppSpec.

Antes de abrir MCP, integrador detectó `BUG-REBUILD-20260714-028`: fixture
declaraba motivo inicial `operator.initial_confirmation`, aplicación usaba
`initial_confirmation` y aceptación no consumía el campo. Test rojo ya liga
fixture a constante de aplicación. Ya cerrado: valor canónico namespaced y
focal aceptación/aplicación verde. MCP avanza en write-set disjunto.

Incidencia de disciplina: el agente de dominio tocó temporalmente
`internal/goal/intent_test.go` para ampliar una tabla de refs sin pedir el
write-set. El integrador lo detectó y el agente revirtió su hunk con
`apply_patch`; el fichero está limpio. No hubo mezcla ni pérdida. Antes de
sellar V04 esta desviación debe cerrarse como
`BUG-REBUILD-20260714-022`. El gate
`TestV04CandidateSubjectsCoverCommittedDelta` compara todo cambio desde
`d313ae5183` con el candidato V04; queda rojo de forma intencional mientras el
delta esté abierto y debe cerrarse antes del receipt.

MCP/i18n quedó integrado en `dc0533caf6`; el split mecánico de tests en
`e4c5f23074`; bootstrap/guía/E2E en `a301a3bbac`; candidato congelado en
`33c0072b89`; receipts y acreditación en `9a6d255a08`. La superficie final son
seis tools exactas, incluida `orquesta.goals.amend`.

`BUG-REBUILD-20260714-029` cerró el falso verde de paquetes con
`[no tests to run]`; el contrato final ejecuta aceptación focal y después todos
los paquetes propietarios sin filtro. `BUG-REBUILD-20260714-030` congela el
delta en `a301a3bbac`, de modo que V05 no invalida V04. `BUG-031` migró todos
los callsites/bootstrap a confirmación y `spec_hash`; `BUG-032` impide acreditar
la capacidad vecina por un parche sin contexto. Todos tienen tests de lección.

El E2E Codex real por API MCP cerró con marcador
`ORQUESTA_CODEX_E2E_OK_333775d0eab63976d648f0f96bb8184d`; su descriptor
durable vive en `product/evidence/real_codex_mcp_e2e.json`. El runtime temporal
fue eliminado tras el sellado. No reabrir V04 salvo regresión reproducible.

Digest de control del árbol antiguo comprobado al cierre V04:
`75577492db531e71f8a47f7b7fec115e79996aed45e26ce66426b4c120d42a80`.

## Regla de actualización

Actualizar este handoff al cerrar cada bloque material, antes de cambiar de
write-set y antes de terminar una sesión. Si una sesión puede cerrarse durante
un bloque largo, registrar también el último rojo/verde reproducible y la
próxima acción exacta. No copiar aquí recibos completos ni convertirlo en una
segunda fuente de estado: resumir hecho, test, bloqueo y siguiente acción, y
enlazar siempre al ledger/contrato canónico correspondiente.

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
