# V22 — Codex real completo por MCP

Estado: contrato rojo. Base integrada y acreditada: V21, commit
`665ef446a32a7f0512255640fa99b2e7edd9f29d`.

## Decisión

V22 no crea otro director ni otro núcleo. Activa el adaptador Codex sobre el
único ciclo Goal/DAG ya construido y demuestra su operación completa usando la
misma API MCP pública que usaría un operador o un agente externo.

El objetivo de V22 tampoco es aceptar una intención abierta ni modificar
Orquesta a sí misma. Eso pertenece a V23/V33. Aquí el plan, sus fases, worksets,
tests y criterios son explícitos.

## Huecos reales de la base V21

1. `ExecutionAuthorityResolver` ya protege los comandos execution-bound. El
   WIP V22 deriva ahora la autoridad *material-free* del tuple durable
   Goal/WorkItem/Execution (incluidas generación e intento): no crea una fila
   ni alias durable de sesión. El material secreto vive únicamente en el
   `CredentialStore` existente. Falta probarlo y acreditarlo en composición;
   `BUG-ORQ-20260723-366` no se considera cerrado por esta documentación.
2. El prompt del adaptador Codex se construye con literales ingleses en
   `internal/adapters/agent/codex/process.go`. V21 dejó `prompts` como superficie
   futura: V22 debe consumir el catálogo canónico antes de activar Codex.
3. El E2E Codex existente acredita un Goal y un artefacto. No prueba cuatro
   Goals concurrentes, parada selectiva, mailbox padre/hijo, workspace Git,
   tests/review, crash/restart, restauración de backup ni censo final de
   procesos.
4. El placeholder de `AC-V22-CODEX-E2E` puede dar verde sin ejecutar V22:
   excluye `./acceptance`, busca un nombre de test inexistente y declara una
   ruta de fixture no canónica. Se registra como
   `BUG-ORQ-20260723-390`; el roadmap se corregirá con el producto, no dentro
   del contrato rojo.
5. **P0 causal:** `AdmitMailbox` exige que el hijo ya esté `succeeded` y que su
   artefacto esté persistido. El WIP materializa `admit_mailbox` en la outbox
   existente y el worker llama al MCP público con la identidad exacta del hijo;
   no depende de que `codex exec` siga vivo ni admite suplencia humana. Sigue
   siendo `BUG-ORQ-20260723-391` hasta que los negativos, restart y E2E lo
   acrediten.

El informe Claude de 2026-07-23 es auditoría estática de un worktree mutante:
no es sello P/S/E ni PASS. Para V22 mantiene relevantes los riesgos 366, 390 y
391, la prohibición de verde por `Skip` y la necesidad de contrapeso de revisión
externa; los riesgos de otras verticales no amplían este contrato.

## Autoridad única

```text
cliente MCP
  -> bindings generados V20
  -> dispatcher único
  -> aplicación Goal/DAG/mailbox
  -> scheduler único
  -> puerto AgentLauncher/Observer
  -> adaptador Codex
```

Ninguna interfaz, adaptador o harness puede escribir estados de ciclo de vida
por fuera de aplicación. No se añade un loop Goal-first alternativo, una cola
especial Codex ni una segunda tabla de comandos.

## Escenario acreditable A/B/C/D

Los cuatro Goals se ejecutan de forma concurrente en el proyecto local
provisionado. El harness puede usar más proyectos solo si los provisiona antes
por una superficie pública existente; V22 no inventa `project.create`. Lo
obligatorio es el aislamiento de Goal, ejecución, write-set y efectos:

- **A — DAG y mailbox:** padre e hijo reales; admisión, claim, entrega,
  consumo y ACK pasan por MCP con identidad de servicio ligada exactamente a
  la ejecución.
- **B — parada selectiva:** se detiene mientras A/C/D siguen progresando. Su
  proceso queda terminado y su estado terminal es el cancelado controlado que
  defina el dominio; no contamina los demás Goals.
- **C — programación:** usa workspace Git aislado, produce cambio, ejecuta el
  test requerido, recibe review y solo entonces integra y cierra.
- **D — recuperación:** queda vivo durante backup y muerte no cooperativa del
  servidor; tras reinicio desde el estado durable continúa sin duplicar efecto
  ni agente y cierra.

El resultado final debe ser A/C/D completados, B detenido de forma explícita,
mailbox sin mensajes huérfanos, ninguna contradicción entre Goal, ejecución,
acción, test, review e integración, y ningún proceso propiedad del harness.

## Identidad de servicio de ejecución

- Se deriva para un tuple exacto de `project_ref`, Goal, WorkItem, ejecución,
  intento, reemplazo, generaciones y `spec_hash`; no es una nueva entidad ni
  fila durable. Tras restart se recalcula desde Goal/Execution durables.
- La proyección contiene refs y principal de servicio, nunca bearer ni otro
  material. El único almacén de secreto es el `CredentialStore` ya compuesto;
  no nace store, tabla ni alias secreto paralelo.
- Los ocho comandos execution-bound fallan cerrados si falta binding, cambia
  proyecto/ejecución/principal o la credencial está revocada.
- Tras restart el binding válido sigue resolviendo.
- Al terminar o reemplazar una ejecución se revoca la identidad anterior. El
  sucesor recibe otra identidad y no puede reutilizar la anterior.
- La identidad no se revoca antes de completar sus obligaciones causales
  post-artefacto. Una continuación acotada o delivery worker puede admitir el
  handoff después de persistir el artefacto, pero actúa con el mismo service
  principal y binding de la ejecución hija.
- El secreto solo entra al proceso mediante el canal de credenciales existente:
  no aparece en argv, entorno heredado, prompt, Goal, journal, receipt, log ni
  artefacto.
- El principal de ejecución no puede usar comandos humanos: el dispatcher solo
  deja comandos execution-bound con su ejecución autenticada. La excepción
  `artifacts.read` no concede lectura genérica: solo permite un artefacto de un
  envelope mailbox ya admitido cuyo destinatario es exactamente ese principal y
  ejecución.
- Un principal humano no puede efectuar `mailbox.admit` en nombre del hijo; el
  audit debe identificar la ejecución hija y su service principal real.

La continuación post-artefacto reclama la acción `admit_mailbox` de la outbox
existente y llama `orquesta.mailbox.admit` por MCP. No decide lifecycle, no
reabre Codex, no produce otro artefacto ni crea scheduler/cola paralelos; debe
ser idempotente ante crash/restart.

## Prompt Codex e i18n

El adaptador recibe un renderer/puerto tipado construido por bootstrap. Las
claves del prompt y sus placeholders viven en el manifiesto y los catálogos
V21. Español es el fallback; cambiar locale modifica solo texto humano, nunca
refs, write-set, capacidades, esquema de salida o autoridad. No se introduce
un segundo lector de JSON ni literales públicos ad hoc en el adaptador.

## Frontera del E2E

El harness puede crear temporales, compilar/arrancar el binario, matar el
proceso, copiar/restaurar backup y hacer el censo de PIDs. Todas las operaciones
de negocio —crear/leer Goal, dirigir, mailbox, cambios, artefactos y cierre—
usan un cliente MCP oficial contra el servidor real.

La prueba real requiere opt-in explícito (`v22_real_e2e`) y Codex disponible.
No se permite `Skip`: sin precondición declarada, credencial o runtime, el gate
falla. El comando de acreditación exige eventos JSON `run` y `pass` de cada
test real para impedir el verde por selección vacía.

## Fuera de V22

- wizard/dossier/fábrica (V23);
- web (V24);
- Claude, Gemini, Ollama, catálogo de proveedores y Hermes (V25);
- tools/skills, RAG, plugins y deploy (V26–V29);
- Postgres/S3/multihost y operaciones completas (V31–V32);
- generación de apps y automejora de Orquesta (V33);
- cutover final (V34).

Backup (`EVD-15`) y shutdown (`OPS-16`) se ejercitan como regresión, pero V22
no se apropia de esas capabilities.

## Presupuesto de simplicidad

- un scheduler y un escritor de lifecycle;
- un registro de comandos generado;
- un adaptador de proveedor activo: Codex;
- cero stores top-level nuevos y cero lectores de entorno fuera de config;
- objetivo original de 2.800 líneas de producto y 3.800 de tests/harness;
- techo operativo V22 de 3.000/4.200 autorizado por el operador el
  2026-07-23: el exceso queda como deuda explícita, sin recortar cobertura ni
  legibilidad antes del E2E real;
- todo binding durable entra por puerto y reutiliza la transacción/repositorio
  SQLite existente;
- archivos de producto nuevos o modificados por debajo de 500 líneas salvo
  justificación explícita.

## P / S / E

- **P:** implementación, tests y trazabilidad quedan congelados en un commit sin
  receipt, output ni acreditación.
- **S:** un segundo commit enlaza el OID P, el blob exacto del fixture y el
  conjunto ordenado de sujetos; aún no implica PASS.
- **E:** desde un checkout detached-clean de S se ejecuta literalmente el argv
  del fixture. Receipt V3 y output nacen fuera del candidato y solo después se
  promociona el roadmap.

No se integrará V22 por existencia de código o por un smoke parcial: solo por
el E2E real completo y su evidencia P/S/E.
