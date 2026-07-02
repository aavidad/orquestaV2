# Incidencia: QA de tema OPES ignoraba evidencias visuales del payload

Fecha: 2026-07-02.

ID inventario: `BUG-ORQ-20260702-118`.

## Sintoma

El productor causal OPES podia marcar un tema como `pendiente_rework_editorial`
por `opes_visual_didactic_function_required` aunque el artefacto trajese
`require_didactic_visual=true` junto con un payload `visuals` valido:
`visual_ref`, `raster=true`, `anchor_ref`, `didactic_function` y evidencias.

## Causa

`topicRegistryQualityRequestForRecordV0` activaba el gate
`RequireDidacticVisual`, pero no hidrataba `Visuals` desde `PayloadFields`.
El contrato puro `OPESTopicQualityContractV0` funcionaba, pero el adaptador
perdia la evidencia visual antes de llamar al validador.

## Cierre

`orquesta-opes-director` normaliza evidencias visuales de tema desde campos JSON
`visuals`, `topic_visuals`, `didactic_visuals`, `visual_evidence`,
`visual_evidences` y `visual_assets`, y tambien desde aliases escalares
compactos como `visual_ref`, `visual_anchor_ref`, `visual_didactic_function` y
`visual_raster`.

El gate sigue exigiendo visual raster, ancla y funcion didactica. El cambio solo
evita el falso rework cuando OPES ya declara esos datos de forma estructurada.

## Evidencia

Test focal:

```bash
go test -count=1 ./modulos/orquesta-opes-director -run 'TestProduceOPESCausalJobsV0NoBloqueaRegistroConVisualDidacticoDeclaradoV0'
```

Validacion del lote OPES:

```bash
go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-director ./modulos/orquesta-opes-topic-registry
```

## Nota operativa

No se lanzo un OPES temporal ni un drain real. La actuacion queda documentada
como reparacion local acotada del conector OPES-Orquesta, sin efectos sobre OPES
productivo.
