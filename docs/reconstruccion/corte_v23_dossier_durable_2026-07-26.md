# V23: corte durable del dossier previo al lanzamiento

Fecha: 2026-07-26.

Estado: **partial_green_unsealed**.

Este corte conserva una propuesta de dossier completa, content-addressed y
ligada al `IntakeRecord` y `PlanSpec` exactos. No genera el contenido editorial
del dossier, no expone todavía comandos públicos, no confirma ni congela el
intake y no crea AppSpec, Goal, plan aplicado ni outbox.

## Autoridad de aplicación

- `BuildIntakeDossier` recibe el record durable y el plan concreto; recalcula
  ambos digests y proyecta todas las decisiones actuales.
- `IntakeDossierService` usa `IntakeStore` solo para leer la fuente y un puerto
  `IntakeDossierStore` separado para la propuesta inmutable.
- El replay exacto se resuelve antes de releer el intake, por lo que continúa
  disponible aunque el intake avance después.
- Una escritura nueva exige revisión, receipt y digest actuales dentro de la
  transacción del adaptador.
- La autorización usa `goals.create` y un request ref específico de generación.
- `GetIntakeDossier` devuelve el receipt canónico de la primera creación. Otros
  receipts del mismo contenido solo se recuperan por replay de su request.
- Application recalcula el fingerprint desde dossier y autorización; un
  adaptador que reescriba de forma coherente fingerprint y receipt es rechazado.

## Persistencia y recuperación

La migración `018_intake_dossiers.sql` añade:

- `intake_dossiers`, una fila inmutable por contenido;
- `intake_dossier_generation_receipts`, una fila inmutable por request;
- FKs diferidas para insertar atómicamente el dossier y su primer receipt;
- guardas de autorización, fuente intake actual, bindings e inmutabilidad.

Dos requests distintos pueden acreditar el mismo dossier sin duplicar
contenido. No se usa `INSERT OR IGNORE`: replays, conflictos y errores de
integridad conservan resultados distintos.

Recovery restaura el snapshot mediante application, recalcula ref y digests,
reconstruye el dossier desde el receipt histórico del intake, valida el
fingerprint y comprueba la autorización histórica. Schema 17 sigue siendo
legible; schema 18 es el vigente. Backup/restore conserva el digest lógico.

## Evidencia

- application focal con `-race`: verde;
- SQLite dossier/recovery focal normal: verde;
- SQLite dossier/recovery focal con `-race`: verde en `160.045s`;
- SQLite amplio con schema 18: verde;
- arquitectura raíz, `go vet` y `git diff --check`: verdes;
- revisión premium final: 0 P0 / 0 P1.

Commits:

- `109a43a8`: builder ligado a intake y plan reales;
- `f4a06080`: servicio, replay, receipt y snapshot;
- `e868676e`: migración 018, adaptador y recovery.

## Pendiente causal

1. publicar prepare/get por comandos y wiring de composición;
2. generar el contenido del dossier mediante agentes sin confundirlo con el
   builder determinista;
3. confirmar el `dossier_ref` exacto y congelar nuevas mutaciones;
4. crear AppSpec, Goal, plan y outbox en la misma transacción de confirmación;
5. cerrar catálogo Wizard, web, packs, sello y promoción del roadmap.

La confirmación no puede implementarse como “freeze y después Submit”: esa
secuencia dejaría una ventana de crash con intake confirmado pero sin Goal.
