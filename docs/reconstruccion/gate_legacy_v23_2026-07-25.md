# Gate legacy previo a V23 — 2026-07-25

Estado: **superado para abrir siguiente tramo; no acredita V23**.

## Actualización dirigida del 2026-07-26

La comparación del Wizard de `/home/alberto/Trabajo/orquesta` y
`/home/alberto/Trabajo/orquesta-rebuild` confirma que ambos contienen la misma
implementación útil; `orquesta-autonomia-clean` no aporta otra variante más
avanzada. No existe una solución V23 posterior escondida.

Se reutilizan como diseño y casos de prueba, no como lifecycle ni estado:

- el catálogo U1–U12, T1–T8 y las reglas cruzadas R1–R8 de
  `modulos/orquesta-web/nueva_app_wizard_gaps_v0.go`;
- sus packs combinables de agenda, ecommerce, mapas, inventario, documentos,
  proyectos, finanzas, CRM, reservas, salud, educación, comunidad, IoT, media
  y facturación;
- las etapas y microtareas de `orquesta-factory/backlog_v0.go` y
  `orquesta-app-planner/plan_large_v0.go`, compiladas al `Goal` vigente;
- slots repetibles, ayuda, glosario y corpus RAG como proyecciones del único
  intake;
- listas editoriales y diagramas de
  `modulos/orquesta-web/nueva_app_wizard_dossier_v0.go`.

No se copian `WebNuevaAppIntakeSessionV0`, estados web paralelos, defaults
técnicos silenciosos, normalización de recomendaciones inválidas, activación
por palabras como rail, refs de dossier no content-addressed, ni planners
legacy completos. Los tests históricos que se portan como especificación son:
opciones con ayuda, «qué es» sin avance de ronda, packs combinados sin
duplicados, bot sin opciones inventadas, slot filling sobre refs permitidas y
diagramas obligatorios.

## Evidencia de barrido

Se consultaron `informe_integracion_generaciones_orquesta_2026-07-25.md`,
`matriz_rescate_generaciones_orquesta_2026-07-25.md`, `product/roadmap.json`
y el contrato parcial `acceptance/v23_wizard_test.go`/fixture. El inventario
deduplicado cubre las cuatro familias, no solo la más próxima:

| Familia | Evidencia deduplicada | Hallazgo V23 | Decisión |
| --- | --- | --- | --- |
| `orquesta.bk(sin VM Berserk)` | `c653b9a3c2fb`; generación temprana | UI/mailbox temprano; no writer durable compatible | `descartar`: arqueología solo ante bug concreto. |
| `orquesta-autonomia-clean` + `orquesta-autoprogramacion-*` | `53ecf770c819`, `8924d5fd92b6`, `77b6620a8dc8` | autonomía, cierres y estado anteriores | `adaptar`: extraer invariantes/negativos de autoprogramación; no monolito, loop ni evidencias antiguas. |
| `orquesta` clásico final | `7576f60bd3b5`, 5.289/5.294 blobs compartidos | Director/goal-first, smokes e incidencias; wizard/control-plane no son producto V23 transferible | `ya_integrado` para V01--V22; `descartar` control-plane, sesiones y stores paralelos. |
| `orquesta-rebuild` + worktrees V23 | cierres V01--V22 ancestros; V23 fixture/test/contrato sin producto durable | `internal/intake` parcial: WIZ-03/04/15, snapshot chat/form y cambios atómicos en memoria | `adaptar`: contrato y negativos actuales; no copiar worktree rojo ni declarar capacidad terminada. |

Regla aplicada: la reutilización solo extrae contrato, fixture, caso adversarial
o invariante. Quedan prohibidos import del árbol clásico, bridge, dual-write,
fallback, lifecycle por sesión/canal y receipts heredados como acreditación.

## Base V23 y límite de este gate

`AC-V23-WIZARD` continúa `planned` en roadmap. Fixture declara
`partial_green_unsealed`, `seal_status=not_sealed`, sin receipt, y difiere
persistencia CAS/restart, idempotencia de aplicación, binding de comandos,
dossier, plan, confirmación, web y sello. Lo existente solo cubre WIZ-03,
WIZ-04 y WIZ-15: preguntas por huecos/contradicciones, recomendación visible y
un estado versionado compartido por chat/form.

Este gate prueba consulta y decisión de legado; **no prueba producto, wiring,
persistencia, aceptación completa ni acreditación V23**.

## Riesgos históricos retenidos

- Autoridad fragmentada: no introducir Draft/SessionRef, plan, outbox o store
  paralelo; `Goal` y escritor de aplicación siguen siendo autoridades únicas.
- Estado/canal divergente: chat y formulario mutan el mismo `state_ref` y la
  misma revisión esperada, nunca estados privados por origen.
- Reintentos/restart: no aceptar éxito por `exit 0` ni receipt histórico;
  prueba nombrada no puede caer en `[no tests to run]`.
- Control-plane legado: no restaurar Director loop, daemon, middleware o
  persistencia clásica para resolver V23.
- Copia de especificación roja: fixture/test V23 orientan contratos; no son
  evidencia de implementación ni autorizan adelantar dossier/plan/web.

## Contrato mínimo del siguiente tramo

Implementar exclusivamente escritor durable de `intake.Change` y binding al
registro único de comandos, con puerto local `IntakeRepository` y adaptador
SQLite. Contrato:

- `Create`, `Get` y `Apply` atómicos por `state_ref` y `expected_revision`.
- vínculo obligatorio a proyecto/principal; aislamiento también por origen;
  sin estado implícito de canal.
- fingerprint de petición: replay idéntico devuelve resultado persistido;
  misma referencia con contenido distinto produce conflicto.
- restart conserva snapshot, revisión e historial; batch inválido no deja filas
  parciales.
- registry rechaza payload sobrante y enruta al mismo escritor; no crea otra
  autoridad.

Pruebas mínimas: chat→form→restart SQLite; carrera CAS con único ganador;
replay/divergencia; aislamiento proyecto/principal/origen; validaciones sin
persistencia; binding/extra payload; y frontera explícita sin Goal/AppSpec,
plan, acciones, outbox, dossier ni web.

Write-set previsto: `internal/application/intake*.go`, puerto local,
`internal/commands/{registry.json,definitions_generated.go,application_handlers.go,handlers_intake.go}`,
`internal/adapters/state/sqlite/{intake*.go,migrations.go}` y pruebas focales
de aplicación/comandos/SQLite/aceptación. Cualquier ampliación exige nuevo gate
de legado y decisión registrada.
