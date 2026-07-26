# Trazabilidad V03

Este directorio no importa ni ejecuta el legado. Congela un censo mecánico y
reglas explícitas de disposición para poder reimplementar conducta útil sin
copiar el núcleo anterior.

`legacy_go.json` clasifica los 106 directorios de `modulos/` por rutas exactas.
Cada paquete, fichero y símbolo productivo hereda únicamente la regla de su
módulo; los nombres y palabras del código no deciden su destino. El test AST
falla si aparece un módulo sin regla o cambia el hash/conteo del censo.

Un `supersede` significa que las capacidades aceptadas expresan la conducta que
debe caracterizarse y rehacerse. No acredita equivalencia, no autoriza copiar
código y todavía no permite retirar el módulo. Un `retire` también exige una
caracterización que demuestre que no se pierde conducta útil.

`pending_sources.json` conserva rangos estructurales de tareas, bugs, skills y
rulepacks. `source_dispositions.jsonl` materializa una fila explícita por cada
fuente descubierta; no aplica reglas por palabras. Cada fila guarda ref, hash,
decisión, capacidades del roadmap, motivo controlado y estado de evidencia.
`disposition_reasons.json` define esos motivos y el protocolo reproducible de
hash. Disponer una fuente no cierra `AC-V03-CANONICAL-LEDGERS`: la disposición
por entrada y sus pruebas de caracterización/lección siguen siendo gates
separados.

`task_extraction_policy.json` define gramática exacta V2 para 164 Markdown:
checkboxes, campos/encabezados/listas/tablas con ID, tres formas amplias y el
candidato estricto `reviewed_block` para un rango narrativo revisado que no
tenga marcador. Las únicas autoridades persistentes de asignación son
`task_entries.jsonl`, `task_candidate_exclusions.jsonl` y las filas `kind=task`
de `source_dispositions.jsonl`; los fixtures y ledgers de revisión aportan
procedencia, no un mapa task→capability paralelo.

El censo contiene 1.919 candidatos estrictos y 1.109 amplios. De los amplios,
189 son mandato incluido y 920 exclusiones revisadas; total: 2.108
`TaskEntry`. Cada entrada liga fuente/línea, marker hash, rango y hash del
bloque semántico, estado original, capability/decisión del roadmap y
`closure_evidence` sin inferir cierre.

Las 696 asignaciones débiles heredadas fueron contrarrevisadas de forma
independiente e integradas como receipt por entrada: 547 confirmadas y 149
corregidas, cero pendientes. La ampliación posterior revisó 1.571 candidatos
en 11 pases disjuntos: 706 entries y 865 exclusiones. Cada decisión queda
ligada al input, fixture final, reviewer opaco, pass, hashes y digest en
`task_candidate_review_provenance.json`.

La procedencia incluye además una contrarrevisión independiente de 55 filas
para `broad_s2_s3`: 46 candidatos de `broad_s2` y nueve de `broad_s3`. Conserva
36 transiciones `entry→exclude`, 19 `exclude→exclude`, el fixture exacto y el
hash de la procedencia anterior que queda supersedida; no altera el universo
de 1.571 candidatos ni crea una autoridad paralela.

Una auditoría posterior detectó sobreinventario en la primera ola: 177
candidatos amplios eran resumen, evidencia o duplicado de una unidad canónica.
Se corrigieron a exclusión después de dos contrarrevisiones independientes. Los
164 candidatos estrictos examinados en ese mismo corte siguen siendo entries:
la ubicación fuera de un bloque narrativo residual no permite borrar una tarea
histórica explícita.

`reviewed_task_blocks.jsonl` normaliza 170 rangos accionables exactos con hash:
102 quedan cubiertos por entradas del scanner y 68 usan una entrada estricta
`reviewed_block`, revisada semánticamente. El bloque solo prueba cobertura; la
capability continúa viviendo exclusivamente en `task_entries.jsonl`.

El gate rescanea las 164 fuentes con una implementación independiente, exige
cobertura exacta candidato→entrada/exclusión y recalcula refs, hashes, unión de
capabilities y receipts. Distingue 33 fuentes sin candidato de 34 sin
`TaskEntry`: la adicional contiene 37 encabezados de incidencias Grupo B,
dispuestos exactamente y delegados a la capa de bugs. Una fuente sin candidato
se revisa como documento completo; nunca se fabrica una tarea para cubrirla.

La revisión semántica no caracteriza todavía toda conducta retenida ni acredita
equivalencia. Esas pruebas siguen siendo el gate pendiente antes de retirar
código o usar una tarea histórica como evidencia de producto.

`historical_bug_extraction_policy.json` congela el universo literal `BUG-ORQ`
de 160 fuentes con rol bug: 152 primarias y ocho fuentes task con rol adicional.
`historical_bug_rows.jsonl` conserva 239
filas Markdown ricas (235 del inventario vivo y cuatro de la incidencia de
invariantes causales), con estado original, área, hipótesis, acción,
capabilities y lesson candidato sin acreditar. El detector normaliza sufijos
compactos como `...-211/212/213/214` y materializa 1.295 ocurrencias exactas —ref,
línea, columna y hash— en `historical_bug_occurrences.jsonl`.

Los 337 IDs normalizados quedan en `historical_bug_ids.jsonl`: 197 heredan la
unión exacta de sus filas ricas y 140 `narrative_only` usan una revisión
semántica explícita de `historical_bug_id_reviews.jsonl`, ligada a sus
ocurrencias por `historical_bug_id_review_bindings.jsonl`. Las cuatro filas
ricas redactadas desde documentos narrativos conservan procedencia semántica en
`historical_bug_row_enrichments.jsonl`. Todos conservan
`closure_evidence=not_verified`; un test histórico citado es solo candidato de
lección, no cierre del rebuild. De las 160 fuentes, 80 no declaran ningún ID
`BUG-ORQ`; el gate las mantiene dentro del universo mediante su lesson de
fuente, con digest propio, para que no desaparezcan por carecer de token.

La extracción mecánica puede inspeccionarse sin sustituir ledgers con
`ORQUESTA_TRACEABILITY_EMIT_HISTORICAL_BUG_OCCURRENCES=1`; la proyección final
de IDs, después de revisar las 140 asignaciones narrativas, con
`ORQUESTA_TRACEABILITY_EMIT_HISTORICAL_BUG_IDS=1`. Ambos flags se aplican al
test focal `TestTraceabilityRebuildHistoricalBugIDs`.

`rebuild_bugs.jsonl` es un ledger vivo de bugs observados durante la
reconstrucción. Sus primeras lecciones forman un baseline mínimo, no un tope:
pueden añadirse filas ordenadas sin editar un conteo global. Un bug solo
figura `closed` cuando existe su test de lección y el comando exacto quedó
verde; los demás conservan `planned:` y estado `open`.

Gate focal:

```bash
go test -mod=vendor -count=1 . -run '^TestTraceabilityRebuild'
```
