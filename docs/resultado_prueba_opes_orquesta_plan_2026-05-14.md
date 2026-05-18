# Resultado prueba OPES-Orquesta plan_tema

Fecha: 2026-05-14.

## Resultado

La prueba real de `plan_tema` paso el tramo critico:

- OPES creo un job externo.
- Orquesta lo dreno por bridge REST opt-in.
- Orquesta arranco Codex real `gpt-5.5` con razonamiento `xhigh`.
- El agente entrego un plan documental.
- Orquesta envio un `document_plan` valido a OPES.
- OPES completo el job inicial y creo trabajos derivados de secciones, visuales
  y revisiones.

Run conservado:
`/home/alberto/Trabajo/OPES/opes-uso/runs/orquesta-plan-real-20260514-235548`

## Lecciones

- El contrato de salida para `document_plan` debe pedir el DTO canonico, no solo
  "un JSON con secciones".
- Aun con buen contenido, los agentes pueden devolver alias razonables. El
  adaptador debe canonicalizar alias seguros y despues validar estrictamente.
- Si Codex se interrumpe tras escribir artefacto pero antes/durante el ACK,
  Orquesta debe recuperar solo si hay artefacto en write-set y la entrega supera
  el quality gate.
- La idempotencia de envio a OPES debe depender del job externo y tipo de
  artefacto, no del ACK del agente.

## Cambios aplicados

- Canonicalizacion de payload `DomainDocumentPlanV0`.
- Quality gate mantiene validacion estricta de `DomainDocumentPlanV0`.
- Recovery de entrega sin ACK para stderr con `turn interrupted`/`tokens used`.
- Idempotencia estable por `domain_ref + job_ref + artifact_type`.
- Contrato OPES-Orquesta documenta campos canonicos esperados.

## Validacion

- `go test -count=1 ./modulos/orquesta-domain-work`
- `go test -count=1 ./modulos/orquesta-opes-bridge`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestDefaultDomainWorkArtifactSubmissionBuilderV0IdempotenciaEstablePorJobYArtefacto|TestDefaultDomainWorkArtifactSubmissionBuilderV0CanonicalizaDocumentPlanAliasesOPES|TestCodexStackV0OPESPlanTemaDeliveryEnviaDocumentPlan|TestCodexStackV0OPESExternalWorkRecuperaEntregaSinACKConArtefactoValido'`
- `go test -count=1 ./...`
