# Pruebas

Comando focal:

```bash
go test -count=1 ./modulos/orquesta-app-change-director-source
```

Evidencia esperada: la fuente emite decisiones validas para cambios concretos
y no inventa trabajo cuando falta write-set o criterios.

Casos focales:

- `TestAppChangeDirectorDecisionSourceV0GeneraCadenaCompleta`: cambio aceptado
  pasa a respuesta, decision, contrato, microtarea con write-set permitido,
  criterios, `work_profile_kind=implementation` y vuelta a programacion.
- `TestAppChangeDirectorDecisionSourceV0ProyectaMetadataRefsComoContextRefs`:
  `metadata_refs` compactas del cambio llegan a `create_microtask.task.context_refs`
  y validan como `WorkflowTaskV0`.
- `TestAppChangeDirectorDecisionSourceV0ProyectaTrabajoExterno`: un
  `external_work` publica contrato `ApplyExternalDomainWorkV0` y microtarea
  documental con `work_profile_kind=domain_work` y paquete de dominio
  suficiente.
- `TestAppChangeDirectorDecisionSourceV0ProyectaPlanDocumental`: `plan_tema`
  crea tarea de planificacion documental, exige `document_plan` y no permite
  redactar el documento final dentro del plan.
- `TestAppChangeDirectorDecisionSourceV0ProyectaTrabajoExternoSinWriteSetLocal`:
  un `external_work` `draft_content_block` de OPES crea microtarea usando
  scopes externos derivados, sin exigir rutas locales en `allowed_write_set`, y
  declara temario, esquema, objetivo, posicion, vecinos, fuentes, criterios y
  longitud si llegan por `input_fields`, ademas de no sobreatomizar temarios
  largos.
- `TestAppChangeDirectorDecisionSourceV0ResumenPermiteGranularidadPequena`:
  `summarize_chapter` admite tarea pequena porque es artefacto derivado y exige
  trazabilidad y conservacion de matices criticos.
- `TestAppChangeDirectorDecisionSourceV0ProyectaVisualAssetOPES`:
  `generate_visual_asset` crea unidad de trabajo externa con criterios de SVG
  seguro, accesibilidad, caption, alt text, ausencia de placeholders y entrega
  `visual_asset`.
- `TestAppChangeDirectorDecisionSourceV0SaneaCriteriosOperativosSinBloquearAutoPlan`:
  reproduce reglas operativas de autoprogramacion con `token economy`, Codex,
  modelo y provider; la fuente conserva esos terminos como vocabulario opaco y
  crea microtarea en vez de dejar la pregunta pendiente.
- `TestAppChangeDirectorDecisionSourceV0BloqueaDetalleSensibleEnAutoPlan`:
  reproduce un criterio con `api_key=` y valida que no se cree microtarea
  automatica.
- `TestAppChangeDirectorDecisionSourceV0CompactaCriteriosExternosAlLimiteDelDirector`:
  reproduce un job OPES con criterios generados y de usuario que antes excedian
  el limite compacto del DTO del director.
- `TestAppChangeDirectorDecisionSourceV0NoInventaMicrotareaSinWriteSetNiExternalWork`:
  sin write-set ni `external_work` no hay microtarea automatica.
- `TestAppChangeDirectorDecisionSourceV0NoInventaMicrotareaSinCriterios`: sin
  criterios no hay microtarea automatica.
- `TestArquitecturaAppChangeDirectorSourceNoImportaAdaptadoresConcretosV0`: la
  fuente no importa DB, runtime ni adaptadores concretos.

Evidencia 2026-05-11: la microtarea de programacion generada por la fuente
incluye `required_tests`; si el write-set toca codigo Go, incluye
`go test ./...`.

Evidencia adicional: si `AppChangeRequestV0.required_tests` trae pruebas
explicitas, la fuente las conserva al principio de la microtarea y solo infiere
`go test ./...` si no llega ya un comando Go.

Evidencia adicional: un `AppChangeIntentEventV0` recibido por
`orquesta-app-change` queda en el store y la fuente lo convierte en decisiones
de replan si el run contiene la pregunta del director.

Validacion integrada desde stack:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run TestDrainRunV0AppChangeConProgramacionPendienteArrancaCambio -v
```

Evidencia esperada: con programacion pendiente, un cambio concreto se convierte
en pregunta respondida, contrato publicado, microtarea creada y descriptor de
agente nuevo sin esperar a que termine el agente anterior.
