# V17: artefactos, snapshot Git y atestador independiente

Fecha de decisión: 2026-07-22. Base analizada:
`eb272b6645928d800619709c9afd272440b0dabf`.

Este documento fijó el contrato antes de programar y conserva debajo la
auditoría de cierre pre-P. V17 posee exclusivamente
`EVD-01`, `EVD-04`, `EVD-05` y `EVD-13`, bajo
`AC-V17-TEST-ATTESTOR`. Depende de V06, V09, V15 y V16. Autor/reviewers,
refinery y Consejo siguen en V18/V19.

## Cierre acreditado

El contrato quedó acreditado el 2026-07-22. La auditoría pre-P que sigue se
conserva como historia del razonamiento; sus apartados «todavía rojos» y
«P/S/E permanecen abiertas» ya no describen el estado vigente.

```text
P: d2f02073848c751130586fbc2f7eae5a9bfd822e
S: a97ea3bc3771c6d89ec055e8189bda1bc6f97ce6
E: c309c588b3badad51d5832b76c37863cd233c0e0
candidate: sha256:8c0463aab77cde82a570cafd1d481bb07d97c143eac72ea32e758e82efa4c6c5
output: sha256:49767386a260d3514158a3a66a2f00d62af36b13957ed376080999adca34471a
```

El fixture sellado enumera 202 subjects, aplica los límites finales
`6800/1900/4500/500/7500` y su argv pasó desde `detached_clean`. La evidencia
canónica vive en `product/evidence/v17_test_attestor.json`. Solo
`EVD-01/04/05/13` llevan sus tres refs; V18 es la siguiente dependencia.

## Evidencia del análisis

La nueva Orquesta dirigió un DAG real de tres análisis read-only mediante MCP:

- Goal: `goal:56f05a56bc822e896ac6762a357a4e73`, estado `succeeded`;
- dominio mínimo: `execution:5bf9a2e6aef1fc8d0b8ef6a05958be36`,
  `artifact:sha256:f4c50966f0d06b84c3b5b9eda15ce19ef73c3951554bd999b62fb98e7486ee4d`;
- seguridad filesystem/sandbox:
  `execution:dc3c08a70dc9f2b5c56c21eef144b57c`,
  `artifact:sha256:a9128e8fb3e02be69b70f550750834470521cc5ca613497f95ef3384030e2334`;
- composición/recovery:
  `execution:53f5c40b3d74e2fd65c22aa5b9afbc0d`,
  `artifact:sha256:af8c677683967c3fbb16013525fd06f8834b917326d6aeaa0f0e6f6b60ac840e`.

Los tres blobs se leyeron por la tool pública, conservaron scope, digest y
tamaño, y luego se volvieron a verificar localmente por SHA-256. Ningún worker
editó el repositorio.

## Diagnóstico estructural

La base correcta ya existe:

- un solo `Goal`, writer de aplicación, scheduler, outbox y SQLite;
- CAS filesystem por SHA-256;
- `WorkspaceBinding` y `ChangeSet` V16 con ejecución/intento, generaciones,
  spec hash, base/head/tree, diff y write-set;
- ledger V15 de intent, approval, attempt y receipt;
- recovery V09 y control Git local por CAS.

El hueco no exige otro subsistema. Exige completar la cadena existente.

La `AttestationRecord` actual con política `agent_output_present` prueba solo
que una observación terminal aportó contenido validado por CAS. No ejecuta
tests, no acredita un árbol Git ni es independiente del autor. Su nombre no
puede usarse para inferir V17. Un `AgentLaunchReceipt` prueba admisión del
provider; tampoco es evidencia de resultado.

El CAS actual también debe endurecerse: valida digest, tamaño, regularidad y
`SameFile`, pero aún debe rechazar de forma estable hardlinks, owner incorrecto,
modos inseguros de componentes preexistentes y drift de metadata durante la
lectura.

## Decisión de simplificación

V17 no copiará árboles Git completos a otro formato CAS. Git ya es el almacén
content-addressed del código y V16 ya sella `BaseOID`, `HeadOID`, `TreeOID` y
`DiffDigest`. Crear tarballs de base/tree/diff duplicaría bytes, parsers,
límites y recovery sin mejorar la autoridad.

El snapshot V17 es el sujeto canónico formado por:

```text
Goal + WorkItem + Execution/intento + generaciones + SpecHash
+ WorkspaceBinding digest + ChangeSet digest
+ RepositoryRef + object format + BaseOID + ParentOID + HeadOID + TreeOID
+ DiffDigest + WriteSetDigest
+ required-tests digest + sandbox-policy digest
```

El adapter Git verifica esos hechos contra el workspace opaco antes y después
del test. CAS conserva salida del agente, manifest canónico del sujeto y
reporte estructurado del atestador. SQLite conserva las relaciones causales.

## Modelo mínimo

### RequiredTestSpec

Cada WorkItem de código declara tests estructurados. No se admiten strings de
shell:

```text
RequiredTestSpec
  Ref
  ToolRef
  Arguments[]
  WorkingDirectory (repository-relative; "." permitido)
```

`ToolRef` selecciona una herramienta registrada por composición; nunca una
ruta física enviada por agente. Orden, argumentos y cwd entran en el digest
canónico. No se aceptan NUL, rutas absolutas, traversal ni duplicados. Un
WorkItem con `WriteSet` y sin test requerido falla cerrado antes de ejecutar.
Trabajo read-only sin `WriteSet` conserva el camino de artifact provenance y
no inventa tests.

### ArtifactRecord y AttestationRecord

No nace otro store. `ArtifactRecord` sigue apuntando al blob CAS y gana la
causalidad exacta de Execution/generaciones/spec. La misma carga puede aparecer
en varias ejecuciones: identidad CAS y ocurrencia causal son conceptos
distintos y SQLite no debe colapsarlos.

`AttestationRecord` queda tipada:

- `artifact_provenance`: contenido observado, CAS validado, ligado a la
  ejecución; no significa tests pasados;
- `required_tests`: verdict `passed|failed`, sujeto exacto, manifest/report
  CAS, attestor/policy refs, timestamps y digests.

Solo `required_tests/passed` con igualdad completa habilita el siguiente paso
técnico para un WorkItem de código. `failed` se conserva como evidencia y deja
el candidato pendiente/rework; nunca se convierte en ausencia de evidencia.

### Puertos

- `TestAttestor.Attest`: ejecuta tests contra sujeto exacto y devuelve resultado
  estructurado. No recibe `Goal`, SQLite, lifecycle, secretos globales ni
  autoridad de integración.
- `SnapshotStreamSource.OpenSnapshotStream`: resuelve el binding durable exacto
  `RepositoryRef + WorkspaceRef + WorkspaceBindingDigest`, revalida marcadores
  Git/OID/digests y entrega una única captura object-stream sellada; no devuelve
  paths ni materializa un segundo worktree host.
- `ArtifactStore`: permanece igual como frontera de bytes. Sus validadores se
  endurecen; no aparece `EvidenceStore` paralelo.

El fake contractual de `TestAttestor` prueba neutralidad. El adapter inicial es
local con bubblewrap; futuros contenedor/remoto entran por el mismo puerto.

## Flujo autoritativo

```text
observe completed
  -> output CAS + artifact_provenance
  -> commit_change
  -> attest_test (mismo outbox/claim/fence/ledger)
       -> OpenSnapshotStream una vez contra binding y marcadores durables
       -> validar OID/tree/diff/tests/policy y sellar bytes por descriptor
       -> TestAttestor sobre esa captura inmutable
       -> manifest/report CAS
       -> TestAttestation durable
  -> passed: candidato queda atestado y pendiente
  -> IntegrateChange autorizado: admite integrate_change existente
  -> failed: conservar candidato y abrir camino de rework
  -> integración exacta: SucceedWorkItem/Close según Goal
```

PASS no integra, no programa integración y no cierra por sí mismo. V16 ya dejó
la admisión de integración como caso de uso explícito y autorizado; V17 añade
ahí el gate de una atestación PASS exacta. Solo después se crea y consume
`integrate_change`. V18 ampliará ese mismo gate con reviews obligatorios sin
cambiar el puerto ni el receipt del atestador.

`attest_test` reutiliza `ActionRecord`, scheduler, claim lease, fence, budgets y
effect ledger. Prohibido crear daemon, cola, lease, DB, state machine o writer
de tests.

## Sandbox local acreditable

El adapter inicial usa bubblewrap como conector reemplazable, no la bandera de
sandbox de un provider. Política mínima:

- identidad del proceso host EUID/GID no root; si no puede demostrarse,
  `test_attestor.bubblewrap_identity_unsafe`;
- binario bubblewrap y toolchain son entradas root-owned, no escribibles y
  capturadas en descriptores/memfd sellados; la propiedad de esas entradas no
  se confunde con la identidad no-root que dirige el proceso;
- `--unshare-all`, red nueva sin interfaces externas, capabilities eliminadas,
  sesión y PID namespace propios;
- árbol sujeto read-only y toolchain autorizado read-only mediante file
  descriptors; no paths arbitrarios del Goal;
- tmpfs privado para cache/temporales;
- environment borrado y reconstruido con allowlist fija;
- HOME vacío del sandbox; no HOME real, `/srv`, `/run`, producción, OPES,
  Docker socket, metadata endpoint ni mounts host no autorizados;
- argv directo, sin shell;
- timeout y output acotados;
- reporte público con códigos/digests, no paths, argv, environment, URLs,
  credenciales ni stdout/stderr crudos.

V17 registra una herramienta Go local para su E2E. El registro general de
tools/toolchains llega en V26; no se simula aquí.

## Reproducibilidad

`SubjectDigest` es determinista y excluye reloj, paths físicos y diagnóstico.
Dos requests con los mismos campos canónicos producen el mismo digest. Cambiar
un solo binding, OID, diff, test, argumento, cwd, tool o policy lo cambia y un
replay con la misma idempotency key falla por conflicto.

El receipt de ejecución puede variar en duración, pero siempre declara el mismo
`SubjectDigest`, verdict, códigos de exit y digests de salida. Timestamps no se
usan como identidad.

## Seguridad CAS

La raíz, shards, temporales y blobs se validan fail-closed:

- sin traversal ni symlinks/archivos especiales;
- owner EUID exacto;
- raíz/shards `0700`, temp/blob `0600`;
- blob/temp con `nlink == 1`;
- descriptor abierto coincide con `Lstat`, y tipo/owner/mode/nlink/tamaño se
  comparan antes y después de leer;
- contenido y ref SHA-256 vuelven a validarse;
- objetos preexistentes inseguros no se reparan silenciosamente.

Errores públicos usan códigos estables y no filtran paths.

## Atomicidad y recovery

CAS y SQLite no comparten transacción. Orden seguro:

1. publicar blobs CAS inmutables;
2. revalidarlos;
3. persistir en una transacción SQLite atestación, artifacts causales, evento,
   consumo de action y siguiente outbox.

Un blob huérfano es recuperable; una referencia SQLite rota es inválida.
Crash/replay antes del proceso, tras proceso, tras CAS y tras commit SQLite
debe converger. Un receipt ya persistido no relanza tests. Fence/revisión
perdedores no duplican facts ni acciones.

## Contrato de aceptación

`AC-V17-TEST-ATTESTOR` debe dejar rojo al menos por estas mutaciones:

1. aceptar launch ACK como PASS;
2. saltar `attest_test` e integrar desde `commit_change`;
3. cambiar tree/diff/test/policy después de sellar;
4. usar workspace distinto del binding;
5. devolver PASS desde el mismo AgentObserver;
6. aceptar hardlink, symlink, FIFO, owner o modo inseguros;
7. heredar HOME/env/red/Docker socket/mount host;
8. persistir PASS parcial tras crash;
9. relanzar tras restart con receipt terminal;
10. cerrar con test failed, ausente o report inválido.

Gates mínimos:

- unitarios de canonicalización y transiciones negativas;
- contrato común de `TestAttestor` con fake;
- integración CAS y bubblewrap real;
- SQLite atomicidad, recovery y tampering;
- Git snapshot exacto;
- E2E Git+SQLite+CAS+bubblewrap por composición productiva;
- `-race` focal;
- leak scan;
- guard de una autoridad;
- receipt V3 P/S/E fuera de su candidato.

## Presupuesto y write-sets

Objetivo de producto: máximo 5.200 LOC netas de producción V17. Core/puertos
máximo 2.000; adapters máximo 2.700; migración máximo 500. Tests no justifican
duplicar producción. Un fichero supera unas 350 líneas solo si tiene una
responsabilidad cohesionada y revisión explícita.

Ola A, disjunta:

1. CAS: `internal/ports/artifact.go` y
   `internal/adapters/artifact/filesystem/**`;
2. dominio/puertos: `internal/goal/**`, `internal/ports/test_attestor.go`,
   contratos focales;
3. aceptación/fixture/roadmap V17.

Ola B tras congelar tipos:

1. aplicación/flujo `attest_test`;
2. SQLite/migración/recovery;
3. Git verification + bubblewrap adapter.

Bootstrap/config/E2E y sellado se integran en serie porque comparten wiring y
evidencia.

## Diferido explícito

- autor, reviewers, refinery y aprobación de promoción: V18;
- Consejo: V19;
- bindings públicos completos: V20;
- i18n total: V21;
- Codex de extremo a extremo: V22;
- toolchain registry general: V26;
- remoto/S3/PostgreSQL/multihost: V28/V31;
- SBOM/firma/release final y mutation testing total: V34.

V17 no añade providers, Forge, UI, microservicios ni compatibilidad legacy.

## Auditoría de cierre pre-P: 2026-07-22

La contrarrevisión final leyó `RequiredTests`, `TestSubject`, `TestAttestor`,
CAS, object-stream Git, aplicación, SQLite, bubblewrap/cgroup, configuración y
bootstrap. El resultado es un candidato PRE-P coherente, no acreditación:

- `RequiredTestSpec` es inmutable, estructurado y durable. Un writer nuevo sin
  tests falla cerrado; read-only conserva provenance y legacy sin tests no
  recibe un PASS sintético.
- `TestSubject`, manifest y digest incluyen `WorkItemGeneration`, generaciones
  plan/AppSpec, intento y `SpecHash`.
- `commit_change` programa `attest_test`, nunca integración. Un PASS deja la
  ejecución pendiente de `IntegrateChange`; un FAIL conserva manifest, report,
  outcomes y atestación durable.
- Output del agente, ACK y `AgentLaunchReceipt` no pueden fabricar
  `required_tests/passed`.
- Git liga el stream al binding durable exacto, verifica los marcadores de
  prepare/commit, recalcula objetos y hashes, y entrega una sola captura sellada
  de OID/tree/diff/tests/policy/write-set. No existe materialización de worktree
  de tests ni doble recorrido diagnóstico `VerifySnapshot` en el hot path.
- CAS valida owner, ancestors, tipos, modos `0700/0600`, `nlink`, identidad de
  inode y drift.
- Bubblewrap separa dos identidades: el proceso director debe ser no-root y los
  binarios/toolchain confiables deben ser root-owned, no escribibles y
  ejecutados desde memfd/descriptores sellados. Usa argv directo,
  `--unshare-all`, capabilities eliminadas, environment vacío y HOME/tmp
  privados.
- Cada uno de los seis efectos físicos exige un `EffectAttempt` durable nuevo.
  Receipt terminal bloquea replay; solo un zero-release exacto con
  `CausalAttemptRef` autoriza retry, aunque `StartedAt == SettledAt`.
- `memory.events`, `pids.events` y `cgroup.events` se leen con límite y se
  parsean completos: tokens, duplicados, campos requeridos y overflow fallan
  cerrado. Un fallo de observación nunca se convierte en `exceeded=false` ni
  PASS, y cleanup sigue retirando la sesión.
- SQLite ya posee gates de PASS y FAIL atómicos, restart, fence, replay,
  receipt terminal, intento de efecto exacto, migración legacy sin PASS y
  recovery adversarial.
- bootstrap conecta el puerto neutral, resolver Git y policy exacta sin crear
  otro writer, store, cola, scheduler o lifecycle.

### Gates todavía rojos

P no puede crearse todavía:

1. No existe commit inmutable P. El OID continúa vacío; la última batería
   completa, race, vet y reconciliación deben terminar sobre el árbol congelado.
2. El E2E productivo ya pasó en el harness delegado systemd 259 con
   `Delegate=yes`, subgrupo `runner`, accounting de memoria/tareas, parent
   cgroup EUID/EGID 1000, modo `0700` y controllers `cpu memory pids`. Debe
   repetirse desde S para producir evidencia independiente del candidato.
3. S y E no existen. No hay output capturado ni
   `product/evidence/v17_test_attestor.json`; ninguna capability puede pasar a
   `accredited`.

Los lessons focales `296`, `305`, `309`, `311`, `312` y `316` están cerrados
por sus pruebas exactas: identidad no-root/inputs sellados, attempt nuevo para
los seis efectos, causalidad sin reloj, paridad SQLite, binding Git exacto y
cgroup fail-closed. Las filas que dependen del E2E real, budget de P o protocolo
P/S/E permanecen abiertas. Cerrar un bug focal no acredita V17.

### Fixture, suites y presupuesto

`acceptance/fixtures/v17_test_attestor.json` enumera 199 rutas del dirty delta
PRE-P exacto desde `eb272b6645928d800619709c9afd272440b0dabf`, incluida la
prueba cgroup fail-closed y excluidos runtime, output y receipt. Acceptance
compara ese inventario mientras P está vacío, localiza los behavior tests por
AST y comprueba que cada nombre de la suite race está seleccionado una vez en
Linux; así un `-run` stale no produce un verde vacío. La decisión y su deuda
quedan en `adr_v17_presupuesto_seguridad_2026-07-22.md`. El budget de V17 es:

```text
producto neto <= 6600 LOC
core/ports    <= 2000 LOC
adapters      <= 4300 LOC
migración     <=  500 LOC
tests         <= 7200 LOC
función nueva/modificada <= 80 líneas
fichero Go productivo nuevo <= 350 líneas
paquetes productivos nuevos <= 2
```

La lista de subjects excluye output y receipt. El OID de producto permanece
vacío y `seal_status=pre_p_unsealed_no_evidence`; así
`TestV17CandidateSubjectsCoverCommittedDelta` falla explícitamente con
`V17_GATE_P_PENDING`, no por schema, ruta inventada ni harness.

### Protocolo causal de cierre

```text
P: ejecutar la batería final focal/race/E2E/vet y crear el commit inmutable del
   delta con OID todavía vacío en el fixture.
S: escribir en el fixture el OID P y
   seal_status=p_product_delta_sealed_pending_evidence; comprobar delta,
   subjects y budgets; sellar solo contrato/roadmap, sin PASS.
E: desde checkout detached_clean de S ejecutar exactamente execution_argv,
   capturar output y emitir receipt V3 fuera del sujeto; después ligar roadmap
   y solo EVD-01/04/05/13 a ese receipt.
```

Un fallo en S o E conserva output y evidencias diagnósticas, pero no integra,
no cierra V17 y no cambia capabilities a `accredited`. El receipt V3 planeado
es `product/evidence/v17_test_attestor.json`; sigue ausente deliberadamente.
