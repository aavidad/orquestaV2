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
  criterios y vuelta a programacion.
- `TestAppChangeDirectorDecisionSourceV0NoInventaMicrotareaSinWriteSet`: sin
  write-set no hay microtarea automatica.
- `TestAppChangeDirectorDecisionSourceV0NoInventaMicrotareaSinCriterios`: sin
  criterios no hay microtarea automatica.
- `TestArquitecturaAppChangeDirectorSourceNoImportaAdaptadoresConcretosV0`: la
  fuente no importa DB, runtime ni adaptadores concretos.

Evidencia 2026-05-11: la microtarea de programacion generada por la fuente
incluye `required_tests`; si el write-set toca codigo Go, incluye
`go test ./...`.

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
