# Continuación del inventario y reutilización legacy

Fecha: 2026-08-13.

Estado: read-model advisory activo. No cambia el roadmap, no acredita una
capability y no autoriza imports, copia de paquetes ni arranque del runtime V1.

## Entrada rápida para el siguiente agente

No recorras el legado completo. Antes de diseñar una función o tarea ejecuta:

```bash
scripts/preflight_reutilizacion_legacy.sh \
  --capability ID \
  --path RUTA_V2 \
  --operation OPERACION \
  --task 'CONDUCTA A IMPLEMENTAR' \
  --function 'NOMBRE O FIRMA, SI EXISTE'
```

La salida compacta separa:

- solución V2 acreditada o brecha declarada;
- resultado histórico declarado y sus límites;
- mecanismo y autoridad que decidía en V1;
- qué conservar y qué evitar;
- reuso de implementación V2, contrato, idea o negativo;
- enlaces exactos a funciones V1 y candidatos solo léxicos;
- pistas y fuentes aún pendientes de evaluación profunda.

Una coincidencia textual nunca es equivalencia. Solo
`--legacy-function-ref occurrence_ref` exige un enlace AST exacto ya revisado.
El preflight devuelve `reuse | reimplement | characterize | reject`; ninguna
de esas decisiones autoriza copiar un cuerpo legacy.

Para cualquier otra aplicación existe una capa general independiente del
roadmap y de las capabilities V2:

```bash
/home/alberto/Trabajo/orquestaV2/scripts/consultar_catalogo_funciones_legacy_v1.sh \
  --name NOMBRE --json
```

Puede ejecutarse desde cualquier `cwd`. Expone función, firma, procedencia,
mecanismo, aciertos, fallos, tests, licencia y condiciones, sin cuerpos ni una
decisión de la aplicación consumidora. Como la licencia/autoría por función no
está establecida, siempre conserva `code_reuse_authorized=false`.

## Estado reproducible, sin cifras congeladas

Obtén siempre el corte vigente de las cuatro capas:

```bash
scripts/consultar_fuentes_legacy.sh --summary
scripts/consultar_pistas_legacy.sh --summary
scripts/consultar_funciones_legacy.sh --summary
scripts/consultar_catalogo_funciones_legacy_v1.sh --summary
scripts/consultar_reutilizacion_legacy.sh --summary
```

Interpretación:

| Capa | Cobertura | Profundidad |
| --- | --- | --- |
| fuentes Markdown | clasificación de todas las fuentes visibles | accionable o solo contexto |
| pistas `TASKENTRY` | capability, razón, rango y hashes | no evalúa acierto/fallo |
| funciones Go | identidad, ruta, firma y contexto de módulo | no evalúa utilidad por sí sola |
| conductas profundas | problema, resultado, mecanismo, fallos y valor V2 | dictamen advisory contrarrevisado |

La amplitud profunda usa como denominador las capabilities distintas del
ledger y es la métrica principal para saber si un agente encontrará una ficha
al buscar un problema. La cobertura de `TASKENTRY` es una métrica secundaria
de detalle: no se infla convirtiendo duplicados del mismo backlog en conductas
repetidas. Ninguna es porcentaje del producto, de funciones ni de líneas. El
censo estructural es 100 % únicamente del snapshot productivo del índice Git.
Historia y raíces externas permanecen fuera mientras su compuerta física siga
cerrada.

El corte cerrado actual es reproducible: 840/840 fuentes, 2.108/2.108 pistas,
184/184 capabilities con ficha profunda y 15.543/15.543 declaraciones Go del
snapshot visible. Estas últimas viven en dos censos separados: 13.204 bajo
`modulos/**` y 2.339 bajo el `cmd/**` legacy, siempre excluyendo
`cmd/orquesta/**`. Las 642 pistas agrupadas en 235 conductas dan un 30,45 % de
detalle documental; no son el denominador de amplitud porque el resto incluye
duplicaciones del mismo backlog o mecanismo.

## Artefactos que gobiernan este read-model

- `product/traceability/task_entries.jsonl`: ledger de pistas revisadas.
- `product/traceability/fixtures/behavior_characterization_agent_batch_*.jsonl`:
  caracterizaciones profundas inmutables y sin disposición canónica.
- `product/knowledge/legacy_reuse_assessments_v1.jsonl`: dictamen advisory y
  relaciones función–conducta.
- `product/knowledge/legacy_reuse_assessments_v1.manifest.json`: digest,
  bytes y conteos fail-closed del assessment.
- `product/knowledge/legacy_go_function_snapshot_v1.jsonl`: índice AST sin
  cuerpos del snapshot Git.
- `product/knowledge/legacy_go_function_snapshot_v1.manifest.json`: identidad
  y conteos del censo estructural.
- `product/knowledge/legacy_cmd_go_function_snapshot_v1.jsonl`: índice AST del
  bootstrap/runtime `cmd/**` legacy, sin cuerpos y sin valoración semántica.
- `product/knowledge/legacy_cmd_go_function_snapshot_v1.manifest.json`:
  identidad, exclusiones y conteos del segundo censo estructural.
- `product/knowledge/legacy_test_observations_v1.jsonl`: tests históricos
  acotados observados; no son acreditación de V1 ni V2.
- `product/roadmap.json`, `product/capabilities.json` y `product/evidence/`:
  única autoridad sobre estado y cierre V2.

No crear otra base paralela. Las consultas anteriores son proyecciones locales
de estas fuentes y fallan de forma cerrada ante digest, shape o drift inválido.

## Regla de decisión

```text
V2 accredited + encaje exacto -> abrir owner/acceptance/evidence y reutilizar V2
V2 parcial                    -> comparar contrato y negativos; implementar solo brecha
V2 no implementado            -> conservar contrato/idea V1 y reimplementar en V2
fallo histórico               -> convertirlo en negativo, mutación o ratchet
código V1                     -> no copiar sin procedencia, licencia, fit y tests
```

`legacy` significa sin autoridad runtime, no sin valor. Goal, lifecycle, writer,
scheduler, configuración, permisos y receipts continúan bajo los contratos V2.

## Cómo ampliar sin fabricar fichas vacías

1. Ejecutar el preflight de la capability y guardar el hueco si no hay ficha.
2. Pedir pistas paginadas con `consultar_pistas_legacy.sh`.
3. Abrir únicamente `source_ref` y rangos seleccionados en la copia legacy de
   consulta; no materializar `modulos/**` en el worktree de producto.
4. Agrupar fuentes que describan una misma conducta. No contar duplicados del
   mismo backlog como evidencia independiente.
5. Documentar problema, entradas/salidas, estado, autoridad, efectos, retry,
   concurrencia/restart, aciertos, fallos, intentos e incertidumbres.
6. Clasificar el resultado de forma conservadora:
   `documented_green_unverified`, `partial_or_mixed`, `failed_or_negative` o
   `unknown`.
7. Seleccionar como máximo unas pocas funciones que expliquen el mecanismo;
   obtener sus metadatos con `legacy_function_inventory --lookup-index-path`.
8. Enlazar tests o fuentes que pertenezcan a la conducta. Nunca guardar cuerpos.
9. Contrarrevisar y publicar assessment + manifiesto de forma conjunta.
10. Ejecutar los gates y comprobar que el preflight devuelve la nueva ficha.

Prioriza cobertura de capabilities y mecanismos distintos. El objetivo no es
convertir miles de declaraciones triviales en prosa repetida: todas son
buscables estructuralmente y tienen contexto de módulo; la capa profunda se
reserva para decisiones que eviten rehacer o repetir un fallo.

## Gates

```bash
scripts/test_consultar_fuentes_legacy.sh
scripts/test_consultar_pistas_legacy.sh
scripts/test_consultar_funciones_legacy.sh
scripts/test_consultar_catalogo_funciones_legacy_v1.sh
scripts/test_consultar_reutilizacion_legacy.sh
scripts/test_preflight_reutilizacion_legacy.sh
go test -mod=vendor -count=1 ./scripts/legacy_function_inventory
go test -mod=vendor -count=1 . \
  -run '^TestLegacyFunctionSnapshotIndexMatchesGitIndex$'
git diff --check
```

Los tests rechazan duplicados, JSON no canónico, metadata manipulada, drift de
ruta/firma/AST/cuerpo, evidencia ajena, parseo parcial y read-model truncado.

## Estado de cierre del relevo

```text
hecho: censo estructural/documental completo y amplitud profunda 184/184 del snapshot visible
invariante: V1 conserva valor y V2 conserva autoridad exclusiva
autoridad final: roadmap/capabilities/evidence
legacy retirado: ninguno; este frente solo caracteriza
riesgo principal: confundir pista, enlace AST o test unitario con verde completo
siguiente dependencia: usar el preflight al programar V2; reabrir el inventario solo ante drift o una raíz externa autorizada
```
