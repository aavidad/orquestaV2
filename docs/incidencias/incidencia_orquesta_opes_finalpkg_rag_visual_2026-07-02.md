# Incidencia: falso verde OPES finalpkg RAG/visual

Fecha: 2026-07-02

Bug: `BUG-ORQ-20260702-120` residual.

## Resumen

El contrato de cierre `completed_syllabus_package` ya exigia RAG canonico y
evidencia visual, pero el script ejecutable del bridge solo validaba paquetes de
tema A1. Un paquete final podia quedar aparentemente listo si tenia HTML/RAG por
presencia superficial, aunque:

- el RAG reconstruido no declarase metadata por chunk como `course_id` y
  `source_variant` o equivalentes;
- un rebuild HTML hubiese eliminado figuras ya declaradas en
  `visuals_manifest.json` o `visual_reuse_manifest.json`.

## Causa

La regla vivia en contratos y required tests, pero no en el validador mecanico
`modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py`. El
script no tenia modo de paquete final y, por tanto, no podia comprobar
`manifest_cierre.json`, `rag/corpus/*` ni la coherencia entre manifest visual y
HTML reconstruido.

## Cierre aplicado

- El script acepta ahora `--final-package-dir`.
- El modo de paquete final exige `manifest_cierre.json` con schema
  `opes_final_package_evidence_manifest.v0`.
- Valida RAG canonico en `rag/corpus/chunks.jsonl`,
  `rag/corpus/summary.json` y `rag/manifest.json` apuntando a esas rutas.
- Bloquea `rag/chunks.jsonl` y `rag/summary.json` sueltos como cierre.
- Exige metadata de curso y variante en el RAG: `course_id`/equivalente en
  manifest, summary y chunks, y `source_variant`/equivalente en summary/chunks.
- Si existe manifest visual, valida `pending_count=0`, que los assets de
  `html_final`/`html_ampliado` existan y que las paginas `tema_*.html` sigan
  referenciandolos tras rebuild.

No se ha tocado OPES productivo ni se han drenado colas OPES. Este cambio es una
reparacion local acotada del adaptador/validador Orquesta.

## Evidencia

Tests:

```bash
go test -count=1 ./modulos/orquesta-opes-bridge -run 'TestOPESValidateTopicPackageV1FinalPackage'
go test -count=1 ./modulos/orquesta-opes-bridge
```

## Estado

Cierre parcial del residual RAG/visual de `BUG-ORQ-20260702-120`.

Siguen fuera de este cierre: estado vivo `stopped/crashed/unreachable`, QA
editorial de tests/tutor y direccion TTS reanudable con heartbeat/timeout de
proveedor.
