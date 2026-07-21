# V18: autor, reviews independientes y refinery causal

Fecha de decisión preflight: 2026-07-22 Europe/Madrid. Base analizada:
`eb272b6645928d800619709c9afd272440b0dabf`.

Estado: `awaiting_dependency`. Este documento, el fixture y el test de
aceptación son contrato rojo de preparación. No son producto, wiring, receipt
ni evidencia de V18. V17 todavía no está sellada; por tanto V18 no fija nombres
ni firmas de V17 y no puede pasar a implementación.

V18 posee exactamente `GOV-12`, `STG-13`, `STG-14`, `STG-16` y `EVD-06` bajo
`AC-V18-INDEPENDENT-REVIEWS`. Depende de V14, V16 y V17. Consejo, ballots,
disenso, veto y `skip_by_operator` pertenecen íntegramente a V19.

## Resultado contractual

Todo candidato de código promocionable queda ligado a un único sujeto
inmutable y a tres launches reales, distintos y durables:

```text
autor                  launch A -> ChangeSet + PASS de tests exacto
reviewer primario      launch P -> assessment estructurado del mismo sujeto
reviewer adversarial   launch D -> assessment estructurado del mismo sujeto

A != P != D
subject(A) == subject(P) == subject(D)
```

La promoción requiere `PASS` de tests, `approve` primario y `approve`
adversarial sobre ese sujeto exacto. Cambiar generación, árbol, diff, write-set,
tests, política, atestación o launch del autor crea otro sujeto e invalida las
aprobaciones anteriores. Aun con las dos aprobaciones, la integración continúa
siendo un comando/efecto explícito y autorizado; review nunca hace merge ni
cierra el Goal por sí sola.

Un `changes_requested` conserva toda la evidencia y habilita rework causal,
no descarte ni fallo por heurística. El sucesor declara `ReworkOf`, referencia
el `ChangeSet` padre y produce un sujeto nuevo que exige de nuevo autor,
reviewer primario y reviewer adversarial.

## 1. Inventario acotado de dependencias

El inventario fue read-only, sin `codebase-memory-mcp`, sobre V12, V14, V16 y
la superficie V17 visible. Solo V14 y V16 son dependencias selladas directas;
V12 explica la procedencia de `GOV-12`. V17 visible es provisional hasta su
receipt.

### 1.1 V12: Director con lease, no sistema de review

Superficie estable observada:

- `internal/application/director.go` ofrece `ClaimDirector`, `RenewDirector` y
  `ProposeDirectorPlan` con principal, lease token, fence y revisión esperada;
- `internal/application/director_state.go` conserva lease y decisión causal en
  el mismo repositorio;
- `AgentLaunchReceipt` ya liga Execution, Goal, WorkItem, generaciones,
  intento, `SpecHash` y `ReceiptRef`;
- `ExecutionRecord.LaunchReceiptRef` conserva el receipt durable del launch;
- V12 difiere expresamente `GOV-12` a
  `AC-V18-INDEPENDENT-REVIEWS`.

Conclusión: el Director puede proponer replan, pero no acredita una review. El
lease del Director tampoco identifica autor ni reviewer y nunca sustituirá un
launch receipt.

### 1.2 V14: controles y replan causal append-only

Superficie estable observada:

- `goal.ApplyReplan` crea sucesores append-only y aumenta
  `PlanGeneration`;
- cada sucesor conserva `ReworkOf` y no puede reusar ref, generación, estado o
  write-set incompatible;
- aplicación valida Source WorkItem, Source Execution, intento, causa y fence;
- las causas actuales son split pending, execution stopped y execution failed;
- pausa, stop, retry, cancel y replan usan el mismo writer, repositorio,
  scheduler y outbox.

Conclusión: V18 debe ampliar este mismo camino con una causa tipada
`review_changes_requested`; no puede crear refinery lifecycle, cola o store.
La ampliación solo se programará con lease `L-AGGREGATE` después del rebase.

### 1.3 V16: workspace, ChangeSet e integración explícita

Superficie estable observada:

- `WorkspaceBinding` liga ejecución, generaciones, `SpecHash`, write-set,
  repositorio, base y receipt de efecto;
- `ChangeSet` liga ejecución/intento, plan/AppSpec, base/parent/head/tree,
  diff, paths, write-set, `ParentChangeRef` y receipts;
- `ChangeSet.Digest()` es identidad canónica del cambio;
- `IntegrateChange` es un caso de uso explícito, autorizado e idempotente;
- admission y processing vuelven a validar Goal, WorkItem, Execution, target y
  efecto antes de llamar a `VersionControl`;
- conflicto o stale dejan el cambio pendiente y no mutan target;
- `parentChangeRef` exige al rework apuntar al `ChangeSet` de origen.

Conclusión: V18 amplía el gate de `IntegrateChange`; no añade otro merge gate,
otro VCS ni integración automática.

### 1.4 V17 visible, aún no contractual

La rama de integración muestra trabajo V17 sin commit/sello. Su análisis y
superficie visible proponen:

- tests requeridos estructurados;
- sujeto exacto de test ligado a workspace, ChangeSet, Git, diff, write-set,
  tests y política;
- `required_tests/passed` durable y separado de provenance del output;
- `commit_change -> attest_test`; PASS deja el cambio pendiente;
- admission y processing de integración revalidan el PASS exacto;
- mismo writer, state, outbox, effect ledger y SQLite.

Esto orienta V18 pero no autoriza importar tipos, campos, métodos, migración o
errores V17. Tras el receipt V17, V18 hará rebase, volverá a leer su API sellada
y sustituirá en este contrato cualquier término provisional por el nombre real.
Hasta entonces el fixture solo expresa el sujeto V18 neutral que V17 deberá
alimentar.

## 2. Diagnóstico: hueco real y piezas que no hacen falta

Hoy un author launch puede producir output, workspace, commit y solicitud de
integración. V17 añade tests independientes. Falta acreditar que dos launches
de revisión distintos analizaron exactamente ese mismo candidato y que el
resultado sigue vigente cuando se admite y ejecuta el merge.

No hacen falta:

- `ReviewStore`, `ReviewQueue`, `ReviewScheduler` o `ReviewLifecycle`;
- un provider o puerto `Reviewer` separado;
- un daemon/resident refinery;
- otra state machine para promoción;
- otro `Goal`, otro WorkItem DAG o una fase que escriba lifecycle;
- una copia de Git tree/diff en otro CAS;
- heurísticas sobre palabras como “approved”, “LGTM” o “looks good”;
- Consejo reducido o veto encubierto;
- integración automática tras dos approvals.

Los reviewers usan el puerto `Agent`, sus actions, claims, leases, budgets,
effect receipts, observación y CAS ya existentes. Review añade valor de dominio
estructurado y hechos causales al mismo `GoalRecord`/`StateRepository`.

## 3. Modelo mínimo neutral

### 3.1 `review.Subject`

`internal/review` es dominio puro. No conoce SQLite, filesystem, Git CLI,
provider, paths, prompts, MCP ni bootstrap. Su `Subject` incluye:

```text
GoalRef
WorkItemRef
AuthorExecutionRef
AuthorExecutionAttempt
PlanGeneration
WorkItemGeneration
AppSpecGeneration
SpecHash
AuthorLaunchReceiptRef
WorkspaceBindingDigest
ChangeSetRef
ChangeSetDigest
TreeOID
DiffDigest
WriteSetDigest
RequiredTestsDigest
TestAttestationRef
TestSubjectDigest
TestPolicyDigest
```

`NewSubject` valida refs, generaciones, digests y coherencia básica.
`Subject.Digest()` usa framing por longitud y orden fijo. Excluye clocks, paths,
target branch mutable, prompt, stdout/stderr, provider/model y diagnóstico.

Campos explícitos `TreeOID`, `DiffDigest` y `RequiredTestsDigest` no son
duplicación autoritativa: permiten demostrar mecánicamente el criterio EVD-06;
`TestSubjectDigest` liga además el sujeto completo del atestador sellado.

`AuthorLaunchReceiptRef` identifica el launch A. En V18, `launch_ref` canónico
significa el `ReceiptRef` durable de `AgentLaunchReceipt`, respaldado por el
`ExecutionRecord` y el effect ledger. No nace otra identidad de launch.

### 3.2 Roles y verdicts

Roles cerrados:

```text
author
primary
adversarial
```

Verdicts cerrados para reviewers:

```text
approve
changes_requested
```

`author` prueba procedencia del candidato; no emite verdict de review. Primary
y adversarial deben ser launches y executions distintos del autor y entre sí.
No se exige provider/model distinto: la independencia contractual es identidad
de ejecución/launch y rol, no una promesa de calidad entre modelos que V18 no
puede medir.

### 3.3 `review.Assessment`

El assessment estructurado incluye:

```text
SubjectDigest
Role = primary | adversarial
Verdict = approve | changes_requested
ReviewerExecutionRef + attempt
LaunchReceiptRef
AssessmentArtifactRef + AssessmentDigest
RecordedAt
```

El artifact conserva hallazgos recuperables. El verdict no se extrae por
strings libres; el output contract exige schema canónico. Payload malformado se
conserva como artifact/provenance y produce rework/retry, nunca approval.

### 3.4 Hecho application-side

`application.ReviewRecord` añade scope y procedencia al assessment:

```text
Ref
GoalRef + WorkItemRef + ChangeSetRef
SubjectDigest
Role + Verdict
ReviewerExecutionRef + attempt
LaunchReceiptRef
PrincipalRef + AgentRef
AssessmentArtifactRef + AssessmentDigest
RecordedAt
```

`GoalRecord.Reviews` es parte del mismo snapshot transaccional. No existe
estado mutable `approved=true`: la evaluación se deriva de Subject, author
launch y exactamente un assessment primary y otro adversarial válidos.

`ExecutionRecord` distingue propósito `author`, `primary_review` o
`adversarial_review` y, para reviewers, conserva `ReviewSubjectDigest`. La
migración de executions históricas las clasifica como author solo cuando la
semántica sellada lo permita; jamás crea review facts retrospectivos.

## 4. Gate puro de review

`EvaluateGate` recibe sujeto, author execution/receipt y reviews durables.
Devuelve un resultado derivado, no lifecycle persistente:

```text
missing       falta launch/assessment válido de primary o adversarial
changes_requested  al menos un reviewer pide cambios
approved      ambos reviewers aprueban el mismo sujeto exacto
stale         cualquier scope/generación/digest/launch no coincide
```

Reglas:

1. Debe existir exactamente un launch author válido para la execution del
   `ChangeSet`.
2. Debe existir exactamente un launch primary y uno adversarial.
3. Los tres `ExecutionRef` y tres `LaunchReceiptRef` son pairwise distinct.
4. Ambos assessments declaran el mismo `SubjectDigest` canónico.
5. Cada assessment procede de su execution, intento, receipt, principal y
   artifact exactos.
6. Launch ACK, output del author, PASS de tests o assessment del otro rol no
   sustituyen un assessment faltante.
7. Duplicado para un rol es conflicto/tamper; no se elige “el último”.
8. `changes_requested` gana frente a approve, pero se esperan y conservan ambos
   assessments para completar la crítica independiente.
9. El `ReviewGateDigest` liga SubjectDigest, los tres launch refs, las dos
   decision refs/verdicts y sus artifact digests.

## 5. Flujo autoritativo positivo

Los nombres exactos de la transición V17 se congelarán tras rebase. La
semántica V18 es:

```text
author launch A
  -> output provenance
  -> ChangeSet inmutable
  -> PASS de tests V17 exacto
  -> misma transacción agenda launch primary P + launch adversarial D
       usando Action/outbox/scheduler/Agent existentes
  -> P y D observan y publican assessment artifacts estructurados
  -> cada assessment se persiste al consumir su action exacta
  -> EvaluateGate = approved
  -> candidato continúa pending; ningún merge automático
  -> principal autorizado llama IntegrateChange explícitamente
  -> admission liga target digest + ReviewGateDigest
  -> processing revalida PASS, sujeto, reviews, launch refs y target
  -> VersionControl.Integrate
  -> mismo writer persiste receipt y transición del WorkItem/Goal
```

Los launches de review no son WorkItems adicionales con un DAG privado. Son
ejecuciones especializadas sobre el mismo candidato, reclamadas por el
scheduler único. Así el WorkItem author puede seguir pendiente de promoción
sin crear el deadlock `author no termina hasta merge / reviewers no arrancan
hasta author terminal`.

## 6. Drift invalida antes del efecto

Admission y processing de `IntegrateChange` validan de nuevo:

- Goal, proyecto, WorkItem y author Execution/attempt;
- PlanGeneration, WorkItemGeneration, AppSpecGeneration y SpecHash;
- workspace binding, ChangeSet y su digest;
- TreeOID, DiffDigest y WriteSetDigest;
- required tests digest, test attestation ref, test subject y policy digest;
- author, primary y adversarial launch receipts;
- primary/adversarial assessment artifacts y verdicts;
- `ReviewGateDigest` incluido en el target digest del effect intent;
- target Git esperado por V16.

Cualquier drift devuelve código estable `review.subject_mismatch` o conflicto
causal antes de llamar al VCS. Un action ya admitido no puede aprovechar
reviews que cambiaron entre admission y claim. El action se cuarentena sin
merge; no repara silenciosamente ni selecciona facts alternativos.

## 7. Rework/refinery causal

Cuando el round termina con `changes_requested`:

1. se conservan ambos assessments y artifacts;
2. integración sigue bloqueada;
3. application deja el source en estado tipado “needs rework” usando el mismo
   Goal/Execution, no `failed` por texto;
4. el Director propone un replan explícito con lease/fence y causa
   `review_changes_requested`;
5. la propuesta referencia source WorkItem, source Execution/attempt, review
   subject/round y revisión esperada;
6. `ApplyReplan` crea sucesor append-only con `ReworkOf=source`;
7. V16 exige `ParentChangeRef=source ChangeSet` al nuevo commit;
8. nueva author execution produce nuevo ChangeSet y nuevo SubjectDigest;
9. ninguna review del sujeto anterior es reutilizable;
10. el nuevo candidato repite A/P/D y solo entonces admite integración
    explícita.

Refinery es este camino gobernado de rework, no un cuarto motor residente. Una
composición podrá asignar el mismo actor author u otro autorizado al sucesor,
pero debe haber author launch nuevo y receipt nuevo.

## 8. Persistencia, atomicidad y recovery

V18 reutiliza `StateRepository`. La ampliación SQLite se reserva después del
rebase para no colisionar con la migración final V17.

Persistencia mínima esperada:

- purpose y review subject digest de reviewer executions;
- `ReviewRecord` inmutable con índice único por
  `(Goal, WorkItem, SubjectDigest, Role)`;
- refs a execution/attempt, launch receipt, artifact y ChangeSet;
- eventos y consumption receipts en la misma transacción que consume la
  action;
- integration effect target ligado al `ReviewGateDigest`.

Orden seguro por assessment:

1. observación del agent y contenido estructurado;
2. publicación CAS del assessment artifact;
3. revalidación del blob;
4. transacción SQLite que consume action, persiste ReviewRecord/evento y, si
   corresponde, deja gate derived approved o needs-rework;
5. ningún paso agenda integración.

Recovery valida:

- refs/scope exactos y existencia de Goal, WorkItem, ChangeSet y executions;
- attempt/generaciones/spec exactos;
- launch receipt/effect/consumption causal por cada participante;
- roles primary/adversarial únicos y launches pairwise distinct;
- SubjectDigest recalculado y assessment artifact digest;
- ausencia de cross-project, cross-goal o stale generation;
- gate/target digest de integration action ya admitida;
- rework successor, `ReworkOf`, rejected round y `ParentChangeRef`.

Una fila faltante, duplicada o contradictoria hace inválido el recovery. No se
“reconstruye” approval desde output, event text o launch ACK.

## 9. Replay e idempotencia

Frontiers obligatorias:

```text
antes de agendar reviews
después de agendar, antes de claim
después del efecto launch y antes de persistir receipt
después de primary, antes de adversarial
después de ambos assessments
después de admission de integración, antes del VCS
después del VCS, antes del commit SQLite
después de needs-rework, antes del replan
después del replan, antes del nuevo author launch
```

Propiedades:

- refs/actions/idempotency keys de primary y adversarial son deterministas por
  SubjectDigest+role y no colisionan;
- receipt terminal no relanza ni cambia de rol;
- replay exacto devuelve el mismo hecho; misma key con otro sujeto/verdict es
  conflicto;
- fence perdedor no publica segundo ReviewRecord;
- restart tras primary conserva primary y solo ejecuta adversarial pendiente;
- restart tras ambos no integra sin comando explícito;
- retry de reviewer crea nuevo attempt/launch y solo el attempt vigente puede
  producir assessment;
- receipt de round anterior nunca migra al sucesor de rework.

## 10. Migración

La migración se diseña sobre el schema V17 sellado, no sobre su árbol sucio.
Reglas ya decididas:

- reservar ordinal SQLite después del rebase;
- no backfill de approvals;
- executions históricas no se convierten en primary/adversarial;
- un pending ChangeSet histórico permanece pending y no integrable hasta que
  una acción explícita y autorizada cree un round nuevo sobre PASS V17 exacto;
- backup/restore conserva ReviewRecords, actions, effects, artifacts,
  integration intents y causalidad de rework;
- dry recovery rechaza tamper antes de publicar schema/receipt de migración;
- rollback deja schema anterior íntegro;
- no dual read permanente ni tabla/store paralelo.

## 11. Arquitectura hexagonal

```text
internal/review                 dominio puro: Subject, Assessment, gate
internal/application            writer, scheduling, admission, replay, rework
internal/ports                  sin puerto nuevo V18; reutiliza Agent/Artifact/VCS
internal/adapters/state/sqlite  persistencia y recovery del mismo StateRepository
internal/bootstrap              wiring del mismo Orchestrator
```

No se crea `internal/adapters/reviewer` salvo que después del rebase aparezca
una frontera externa real con segundo consumidor y fake contractual. Parsear el
schema canónico de assessment es dominio/protocolo, no provider nuevo.

Provider-specific prompt, model routing y calidad real quedan en V22/V25. V18
acredita semántica neutral con fake agents y composición productiva local.

## 12. Seguridad y privacidad

- solo el exacto reviewer execution claim puede publicar su rol;
- author no puede autoatribuirse primary/adversarial por payload;
- principal/proyecto proceden de application/access y facts durables;
- assessment no acepta refs de otro Goal/proyecto/ChangeSet;
- output libre se conserva, pero solo schema estructurado puede decidir;
- evidencia pública no incluye prompt, workspace path, argv, env, HOME,
  provider-private IDs, tokens, raw stdout/stderr ni credenciales;
- artifact grande viaja por ref/digest, no se incrusta en eventos;
- `changes_requested` no es fallo por palabra ni rail heurístico;
- Consejo/veto no se infieren de severidad o texto de hallazgo;
- review approval no concede permiso `changes.integrate`.

## 13. Códigos máquina mínimos

El detalle final se ajustará al patrón V17 sellado. Semántica mínima:

```text
review.subject_invalid
review.subject_mismatch
review.role_invalid
review.launch_missing
review.launch_not_distinct
review.assessment_invalid
review.assessment_duplicate
review.assessment_stale
review.required
review.changes_requested
review.rework_causality_invalid
review.gate_digest_mismatch
```

Textos humanos usarán catálogo cuando V21 lo cablee. V18 no añade strings
públicos localizados ni bindings HTTP/MCP/CLI.

## 14. Contrato de aceptación rojo

Archivos:

```text
acceptance/fixtures/v18_independent_reviews.json
acceptance/v18_independent_reviews_test.go
```

El fixture valida ahora, sin V17 API:

- ownership/dependencias/base/estado no sellado;
- sujeto exacto y digest canónico de ejemplo;
- tres participants con ExecutionRef y LaunchReceiptRef pairwise distinct;
- primary/adversarial sobre el mismo SubjectDigest;
- ocho mutaciones de generación/tree/diff/tests/policy/launch que cambian el
  digest;
- rework con rejected subject, `ReworkOf`, ParentChangeRef y sujeto nuevo;
- budget, recovery/replay, E2E y superficies diferidas.

El test compila sobre la base V16 y su subtest de seams V12/V14/V16 queda
verde. El rojo esperado procede solo de tipos, markers y behavior tests V18
ausentes. No importa ni exige tipos V17 no sellados.

Tests conductuales que la implementación debe materializar exactamente:

1. subject digest liga generaciones/tree/diff/tests;
2. A/P/D usan tres launches distintos;
3. roles no reutilizan launch;
4. assessment exige sujeto/verdict estructurado;
5. ambos approvals son necesarios;
6. retirar cualquier launch bloquea;
7. drift bloquea antes del efecto;
8. output/PASS/author no autoaprueban;
9. changes requested persiste y bloquea;
10. rework exige rejected round y parent change;
11. rework crea sujeto nuevo;
12. integración sigue explícita;
13. crash/replay no duplica;
14. recovery rechaza tamper/scope;
15. claims concurrentes dan un decision fact por rol;
16. evidencia no filtra datos privados;
17. arquitectura conserva writer/state/outbox/scheduler únicos;
18. E2E fake+Git+SQLite+CAS+attestor cierra el flujo.

Mutaciones obligatorias incluyen quitar cada launch, reusar launch, cambiar
generación/tree/diff/tests/policy, aceptar una sola review, autoaprobar desde el
author, integrar automáticamente, rework sin causa/parent y reutilizar review
stale.

## 15. E2E y límites de evidencia

E2E V18 usa:

```text
fake agents + Git local + SQLite + filesystem CAS + TestAttestor sellado V17
```

Escenario positivo:

1. author launch produce ChangeSet;
2. PASS exacto;
3. primary y adversarial launches distintos;
4. restart después de primary;
5. adversarial completa;
6. ninguna integración ocurre sola;
7. `IntegrateChange` explícito integra y cierra por el writer único;
8. restart verifica hechos y cero procesos propios.

Escenario rework:

1. primary `changes_requested`, adversarial `approve`;
2. integración imposible;
3. Director replan causal;
4. successor con ReworkOf/ParentChangeRef;
5. nuevo A/P/D y nuevo sujeto;
6. ambos approve;
7. integración explícita.

Escenarios negativos eliminan launch y cambian tree/diff/tests entre admission
y processing. El VCS fake cuenta llamadas y debe quedar en cero.

No se promete provider real en V18. Codex real completo y su workspace de
review pertenecen a `AC-V22-CODEX-E2E`. Un fake E2E acredita semántica y wiring
local, no calidad editorial ni independencia entre modelos.

## 16. Budget de simplicidad

Tope de producción V18 después de desbloqueo: 4.200 LOC netas.

```text
internal/review dominio puro        <= 700
application/writer/flows            <= 1.800
SQLite/migración/recovery            <= 1.000
adapters/bootstrap/composición       <= 700
fichero individual orientativo       <= 350 líneas
```

Ratchets:

```text
nuevos lifecycle writers = 0
nuevos schedulers         = 0
nuevos stores             = 0
nuevos daemons            = 0
nuevos outbound ports     = 0
E2E focal                 <= 180 s
```

Superar un límite exige ADR, causa medible y retirada compensatoria. Tests no
justifican duplicar producto.

## 17. Write-set y leases tras V17

Preflight actual posee solo:

```text
docs/reconstruccion/analisis_y_contrato_v18_*.md
docs/reconstruccion/worksets/v18*.json
acceptance/v18_*.go
acceptance/fixtures/v18_*.json
```

Tras receipt V17 y rebase se solicitarán:

- `L-AGGREGATE` para purpose/rework/gate en Goal/application existentes;
- `L-STATE` para ampliar atómicamente `GoalRecord`/`StateRepository`;
- `L-SQLITE-MIGRATION` para reservar ordinal y recovery;
- `L-BOOTSTRAP` solo al cablear composición/E2E;
- `L-TRACE` al integrador para cambiar roadmap/ledgers;
- `L-SEAL` al ejecutar P/S/E.

Raíz privada preferente: `internal/review/**`. Cualquier edición compartida se
declarará por archivo después de conocer el delta V17 real.

## 18. Riesgos P0/P1

### P0

- integrar con dos approvals bound a un sujeto distinto;
- usar el mismo launch para dos roles;
- aceptar ACK/output/PASS como review;
- llamar al VCS antes de revalidar review gate;
- auto-integrar tras approvals;
- recovery que inventa o reasigna review facts;
- rework que reutiliza approvals antiguas;
- segundo writer/store/scheduler/lifecycle.

### P1

- duplicado de assessment por crash/fence race;
- migration que marca executions legacy como reviewers;
- retry que permite al attempt anterior decidir;
- `ReviewGateDigest` ausente del effect target;
- prompt/path/provider-private data en evidence;
- fichero/controller sobredimensionado;
- tests que prueban nombres pero no E2E/tamper/restart;
- V18 que asume API V17 del árbol sucio.

## 19. Secuencia exacta de desbloqueo y continuidad

Cuando V17 quede sellada e integrada:

1. conservar este mismo branch/worktree/propietario;
2. verificar receipt V17 desde el nuevo HEAD;
3. rebase sobre el OID integrado con lease del coordinador;
4. releer delta/API V17 sellados y actualizar base/manifest;
5. ejecutar contrato V18: debe seguir rojo solo por ausencia V18;
6. reservar leases compartidos y ordinal SQLite;
7. implementar `internal/review` puro;
8. integrar writer/actions/executions/assessment/rework;
9. persistencia/recovery/replay;
10. wiring fake y E2E;
11. focales, race, seguridad, leak scan y suite proporcional;
12. contrarrevisión independiente;
13. entregar propuesta mecánica de roadmap/ledgers al integrador;
14. P/S/E desde checkout detached clean;
15. solo receipt V3 válido cambia estado a `sealed`.

Hasta ese trigger el estado correcto es `awaiting_dependency`, nunca
`integration_ready`, `sealing` ni `sealed`. El propietario V18 no se reasigna
ni abandona la vertical.

## 20. Diferido explícito

- Consejo, ballots, disenso, veto, `auto|required|skip_by_operator`: V19;
- bindings públicos y registro único: V20;
- i18n total: V21;
- Codex real completo: V22;
- Hermes/Claude/Gemini/Ollama/local: V25;
- tools/context/Forge: V26–V28;
- OPES y reviews editoriales de dominio: V30;
- Postgres/S3/multihost: V31;
- firma/release/cutover global: V34.

V18 no añade provider, Forge, UI, OPES, configuración, env, secreto,
microservicio ni compatibilidad legacy.
