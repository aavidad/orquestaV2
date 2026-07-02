# Incidencia: cierre OPES final sin resultados de calidad por tema

Fecha: 2026-07-02.

## Sintoma

Un `completed_syllabus_package` podia declarar `qa_passes` finales,
`qa_report_refs` y evidencias agregadas de HTML/RAG/audio/tests/visual/QA sin
transportar resultados durables del `OPESTopicQualityContractV0` por tema.

Eso dejaba una ruta de falso verde: el cierre agregado parecia publicable aunque
la calidad editorial por tema siguiera dependiendo de informes narrativos o de
estado implicito.

## Causa

El contrato final OPES ya exigia manifest, checksums, informes QA estrictos y
evidencias por categoria, pero no exigia refs estructuradas a los resultados de
calidad por tema. El validador de tema existia y el productor podia usarlo, pero
el cierre final no lo convertia en evidencia obligatoria de paquete.

## Mitigacion

El manifest `opes_final_package_evidence_manifest.v0` ahora requiere
`topic_quality_contract_result_refs` o `topic_quality_contract_results`.
Si faltan, el builder deja el artefacto como no terminal y el validador de
`GoalWorkClosure` bloquea el cierre con
`domain_work_opes_topic_quality_contract_missing`.

## Evidencia

- `TestDefaultDomainWorkArtifactSubmissionBuilderV0NoCompletaOPESFinalSinTopicQualityContractRefs`
- `TestGoalDomainReceiptClosureValidatorV0BloqueaOPESFinalSinTopicQualityContractRefsV0`
- `TestOperationalClosureSourceV0NoCierraOPESFinalSinResultadosTopicQualityPorTema`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`

## Alcance

Cierra el subproblema de cierre agregado OPES dentro de `BUG-ORQ-20260701-058`.
No cierra los residuales de lifecycle goal-first: heartbeat/checkpoint durable
por tema, reconciliacion terminal y criterios `done/settled` de goals OPES.
