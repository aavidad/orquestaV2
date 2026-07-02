# Incidencia: QA por tema OPES no estaba cableada al registro causal

Fecha: 2026-07-02

## Resumen

El contrato puro `OPESTopicQualityContractV0` detectaba extension insuficiente,
andamiaje interno, metacomentarios y visual didactico ausente, pero el productor
causal OPES no lo usaba antes de actualizar el registro de tema.

Esto permitia que una entrega con campos de cierre textual y estado `ready`
terminase en `topic_registry` sin convertir fallos editoriales en
`pending_refs` ni rework causal.

## Causa

El validador existia como pieza determinista y con tests propios, pero no estaba
integrado en `topicRegistryStatusForRecordV0` ni en `followupRefsForRecordV0`.
El registro solo miraba estado declarado, followups existentes y cierre de
paquete final.

## Cierre aplicado

- Cuando un artefacto OPES declara datos de QA textual por tema, se ejecuta
  `ValidateOPESTopicQualityContractV0`.
- Si el resultado es `needs_rework`, el registro publica
  `proposed_status=pendiente_rework_editorial`.
- Los fallos se materializan en `pending_refs` y generan followup causal de
  rework.
- Si la QA por tema pasa, no se bloquea el flujo existente.
- Si el artefacto no declara QA de tema, no se introduce un veto nuevo.

## Evidencia

Tests:

```bash
go test -count=1 ./modulos/orquesta-opes-director -run 'TestProduceOPESCausalJobsV0(BloqueaRegistroPorQATemaFallida|NoBloqueaRegistroConQATemaCompleta|PaqueteFinalCompleteConManifest)'
go test -count=1 ./modulos/orquesta-opes-director
```

## Estado

Cerrado en codigo local. Pendiente de commit/push y sincronizacion remota.
