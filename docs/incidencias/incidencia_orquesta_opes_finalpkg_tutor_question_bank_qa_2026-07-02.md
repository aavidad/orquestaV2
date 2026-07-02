# Incidencia: paquete final OPES cerraba sin tutor y QA de banco/tutor

Fecha: 2026-07-02.

## Hallazgo

El contrato ejecutable de `completed_syllabus_package` en el stack Codex
exigia manifest, HTML, RAG, audio, tests, visual, QA editorial estricta y
`topic_quality_contract_result_refs`, pero no bloqueaba el cierre si faltaban:

- evidencia final de `tutor`;
- `qa_passes.question_bank_publicable`;
- `qa_passes.tutor_assets_publicable`;
- informes `qa_report_refs.question_bank_publicable` y
  `qa_report_refs.tutor_assets_publicable`.

Esto dejaba una brecha entre `orquesta-opes-bridge`, que ya declaraba esos
required tests, y el cierre agregado de `orquesta-app-codex-stack`.

## Riesgo arquitectonico

No era un bug de un valor aislado. Era una desalineacion entre contratos:
el job final pedia evidencias publicables de tests/tutor, pero el cierre
aceptado podia validarse con una QA generica. Eso podia producir falso verde de
paquete final OPES aunque faltasen tutor o banco publicable.

## Cierre aplicado

- `codexStackOPESFinalPackageRequiredEvidenceCategoriesV0` ahora exige
  categoria `tutor`.
- `codexStackOPESFinalPackageQAPassesV0` conserva y valida
  `question_bank_publicable` y `tutor_assets_publicable`.
- `qa_report_refs` incorpora informes de `question_bank_publicable` y
  `tutor_assets_publicable`.
- El cierre Goal OPES proyecta esos fallos sobre
  `domain_receipt_refs.opes_final_package.qa_passes`.

## Evidencia

- `TestCodexStackOPESFinalPackageEvidenceIssueRefV0BloqueaSinTutorV0`
- `TestCodexStackOPESFinalPackageEvidenceIssueRefV0BloqueaQABancoYTutorV0`
- `TestCodexStackOPESFinalPackageEvidenceIssueRefV0AceptaTutorBancoYQAEstrictaV0`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-director ./modulos/orquesta-opes-topic-registry`

No se toca OPES productivo ni se leen cursos reales.
