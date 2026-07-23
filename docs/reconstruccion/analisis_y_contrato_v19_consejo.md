# V19 — Consejo: análisis y P/S/E cerrados

Fecha: 2026-07-23. Base integrada: V18 acreditada en `6f244a7594`.

Estado: V19 acreditada. P=`cccfb4a64bd2cfb788a2d604ccd8d008e946e8a3` y
S=`f7a574e36528871ee07a5529a88be1f0a88511d7`. El manifest S liga árbol,
binario, configuración, V18, tres sujetos Council y presupuesto. El argv exacto
ejecutado desde S `detached_clean` cerró suite, race, V18 real, los tres E2E
Council y vet. Receipt V3 `PASS` y revisión externa `GO` promocionan únicamente
AC-V19 y sus seis capabilities.

## Decisión estructural

V19 no crea otro orquestador. Reutiliza un Goal, un lifecycle, un writer de
aplicación, un scheduler, un `StateRepository`, CAS, eventos y outbox. El
Consejo añade hechos causales y tres propósitos de ejecución; no añade store,
cola, daemon, scheduler, lifecycle, writer, puerto outbound ni `ActionKind`.

El gate V18 sigue siendo obligatorio. Autor, reviewer primario y reviewer
adversarial son tres launches distintos: autor aporta producción/provenance;
primario y adversarial aprueban el mismo sujeto exacto con attestation `PASS`.
Consejo nunca sustituye esas reviews, integra, cierra ni replanea. Solo habilita
o bloquea un comando explícito de integración.

## Sujeto exacto

`CouncilSubject` puro calcula digest SHA-256 con separación de dominio y campos
length-framed sobre:

1. `ProjectRef`;
2. `ReviewSubjectDigest` V18;
3. `ReviewGateDigest` V18;
4. política Council inmutable.

`GoalRef`, `WorkItemRef`, `ChangeSetRef`, `SpecHash` y generaciones de plan,
item y AppSpec se persisten como enlaces verificables. No se copia ni reinventa
el sujeto V18: cada apertura, fact, decisión, skip, admisión y procesamiento de
integración reconstruye el gate V18 y comprueba ambos digests.

La política nace únicamente en `WorkItemSpec.CouncilPolicy`, se valida como
`council.Policy`, se incorpora al `WorkItem` y a `GoalSnapshot` y queda durable
antes del launch autor. Es obligatoria para todo item con write-set; un item
read-only puede carecer de ella porque no produce change ni sujeto Council. Un
rework declara otra vez política en el nuevo item. Nunca se deriva de texto,
rol, criticidad, configuración mutable ni estado posterior a las reviews.

## Políticas excluyentes

- `auto`: aplicación abre una ronda exactamente una vez tras gate V18 válido y
  agenda `proposer`, `critic` y `arbiter`.
- `required`: integración queda bloqueada. Director con lease/fence vivo abre
  explícitamente la ronda; después usa los mismos tres roles y reglas.
- `skip_by_operator`: humano con permiso exacto `council.skip`, rol
  `platform_admin`, `project_owner` u `operator`, registra principal, motivo no
  vacío, UTC, `SpecHash`, idempotency key y subject digest. Solo existe antes de
  cualquier ronda/fact/launch Council. Agenda cero agentes. No es ballot ni
  decisión sintética. V18 sigue siendo obligatorio.

La política se congela para el sujeto. No existe caída automática de
`required` a `auto`, de Consejo fallido a skip ni de skip a ronda.

## Deliberación y decisión

La ronda exige tres launches Council nuevos, distintos entre sí y distintos de
los launches V18:

- `proposer` publica contribución de propuesta y ballot;
- `critic` publica contribución crítica y ballot;
- `arbiter` publica ballot arbitral.

Cada rol emite un único envelope estricto
`orquesta.council.contribution.v1` con sujeto, rol, body, ballot y evidencia
tipada. Proposer deriva facts proposal+ballot; critic, critique+ballot; arbiter,
ballot. Todos referencian el mismo artifact CAS de esa observación. No se
fragmenta una observación en efectos externos ni artifacts inventados.

Ballots exactos: `accept`, `reject`, `abstain`, `security_veto`. ACK, texto,
severidad, alias o log no son evidencia. Todo fact requiere receipt de launch,
artifact estricto, sujeto, rol, intento e idempotencia exactos.

Regla determinista:

- cualquier `security_veto` tipado con artifact y evidence ref produce
  `blocked_security` después de recibir 3/3 ballots;
- sin veto, se esperan 3/3 ballots;
- al menos dos `accept` producen `accepted`;
- al menos dos `reject` producen `rejected`;
- todo otro conjunto completo produce `no_consensus`.

No existe decisión temprana: ni mayoría provisional ni veto retiran roles o
acciones pendientes. Así crash/replay usa una sola regla de cierre. Disenso
derivado exacto: `accepted` marca reject/abstain; `rejected` marca
accept/abstain; `no_consensus` marca los tres ballots; `blocked_security` marca
todo ballot no-veto. `rejected`, `no_consensus` y `blocked_security` preservan
trabajo y exigen replan causal del Director. Propuesta y sucesor del replan
incluyen `CouncilDecisionRef`, `CouncilDecisionDigest` y
`CouncilSubjectDigest` fuente.

## Aplicación, persistencia y recovery

`GoalRecord` incorpora rondas y facts Council. `StateRepository` recibe
mutaciones atómicas de apertura, contribución y skip; persiste snapshot, evento
y outbox en la misma transacción. Lanzamiento/observación usan
`ActionLaunchAgent` y `ActionObserveAgent` con propósitos Council. No hay
`CouncilStore`.

La migración SQLite `014_council.sql` sucede a V18/013. Debe fallar cerrado si
encuentra candidato V18 vivo pendiente/claim/unknown sin política Council
durable. No inventa backfill ni promociona retrospectivamente; registros V18 ya
completados se preservan. Recovery valida ronda→change→gate V18,
fact→launch→artifact, decisión/skip/veto e integración pendiente. Restart
recompone acciones persistidas, nunca duplica launch, fact o decisión.

Replay con identidad y payload iguales devuelve el mismo hecho. Cambiar payload,
sujeto, proyecto, generación, rol, intento o launch falla. Integración guarda
una resolución tipada y mutuamente excluyente: `auto|required` enlazan
`CouncilDecisionRef` + `CouncilDecisionDigest`; `skip_by_operator` enlaza
`CouncilSkipRef` + `CouncilSkipDigest`. Ambos llevan `CouncilSubjectDigest`.
Action, intent, admisión, target digest y processing permiten exactamente una
de esas dos formas y la revalidan junto a V18. Un skip nunca se serializa como
decisión sintética.

## E2E obligatorios

Tres runtimes aislados, DB/FS/agentes propios:

1. `auto`: gate V18 abre una ronda; tres launches; 2 accept + 1 reject; disenso;
   integración explícita; restart/replay sin duplicados; sustitución rechazada.
2. `required`: no abre ni integra sin Director válido; fence stale falla; veto
   tipado espera 3/3, queda durable y bloquea; segundo sujeto 1/1/1 demuestra
   `no_consensus` y tres disensos; ACK/texto no se transforma en evidencia.
3. `skip_by_operator`: RBAC humano exacto, payload completo, cero launches,
   replay igual idempotente, payload/cruce/post-ronda rechazado; V18 revalidado.

Tests unitarios/mutación cubren digest, quorum, veto, disenso, spoof, cruce,
replay e invariantes. SQLite/race/restart cubren upgrade 013→014, FK/triggers,
ballot/skip concurrentes y crashes antes/después de launch, fact, decisión e
integración.

## Simplicidad y P/S/E

El inventario P2 honesto es 3.441 LOC netas: dominio 387, aplicación 1.214,
SQLite/recovery 1.835, bootstrap de producción 0 y adaptador MCP 5. El soporte
de test se mide aparte (3.731: bootstrap 668, MCP 43 y resto 3.020), para no
presentarlo como producción. No es una acreditación de simplicidad: hay 53
ficheros candidatos por encima de 350 líneas y V19 no puede afirmar
verde mientras subsistan.

Hasta V22 rige un ratchet verificable, no un falso techo verde: producto <=3650,
dominio <=400, aplicación <=1300, SQLite/recovery <=1850, fichero <=1500 y
bootstrap de producción <=400; MCP/adaptador de transporte <=50. El límite por
fichero aplica a producción; tests y datos grandes quedan inventariados como
deuda, no ocultos. S debe sellar esas mediciones, los 53
ficheros >350 y el digest del candidato. La deuda V22 es separar
`claim/read/validate/write` SQLite y los adaptadores de aplicación mayores de
350, recuperando presupuesto compacto por módulo. Sin duplicar tipos V18,
scheduler ni autenticación.

- `P` (`implemented_unsealed`): producto y tests ya integrados, pero toda
  capability V19 sigue `declared`, sin `evidence_refs` y sin receipt. La
  aceptación deja de usar el rojo de producto ausente, pero solo verifica el
  contrato P; no acredita nada.
- `S` (`sealed_unexecuted`, `V19_SEALED_UNEXECUTED`): sella árbol, binario, configuración efectiva,
  gate/receipt V18 y sujetos Council en
  `product/evidence/v19_council_seal.json`. Ese manifiesto no es receipt ni
  cambia roadmap/capabilities. El output y receipt E siguen ausentes.
- `E` (derivado del fixture S sellado): desde `detached_clean` ejecuta los E2E aislados y emite
  fuera del candidato `product/evidence/v19_council.output.txt` y receipt V3
  `product/evidence/v19_council.json`. Solo después de receipt válido y review
  externa `GO` se actualiza `AC-V19-COUNCIL` y las seis capabilities.

Antes de P, estado único `planned`: `V19_PRODUCT_PENDING`, sin ninguno de los
tres paths anteriores. No se permite saltar estados ni crear un placeholder de
receipt/output. El fixture declara exactamente P o S; E no reescribe el blob S
que se acredita, sino que se deriva solo de receipt V3 + output + roadmap
`executable`/seis capabilities `accredited`.

Comando E canónico (debe conservar eventos JSON `run` y `pass` de cada nombre,
no solo `PASS` del package):

```bash
e2e_events=$(CGO_ENABLED=0 go test -mod=vendor -tags=v18_real_e2e,v19_real_e2e -json -count=1 -timeout=240s ./internal/bootstrap -run '^(TestRealGitSQLiteFilesystemCASBubblewrapIndependentReviewsEndToEnd|TestV19CouncilAutoSQLiteFilesystemRestartE2E|TestV19CouncilRequiredStaleOpenAndVetoWaitsThreeE2E|TestV19CouncilSkipHumanReplayAndIntegrationE2E)$' 2>&1); e2e_status=$?; printf '%s\n' "$e2e_events"; [ "$e2e_status" -eq 0 ] && for test_name in TestRealGitSQLiteFilesystemCASBubblewrapIndependentReviewsEndToEnd TestV19CouncilAutoSQLiteFilesystemRestartE2E TestV19CouncilRequiredStaleOpenAndVetoWaitsThreeE2E TestV19CouncilSkipHumanReplayAndIntegrationE2E; do printf '%s\n' "$e2e_events" | grep -F '"Action":"run"' | grep -F '"Package":"orquesta/internal/bootstrap"' | grep -F "\"Test\":\"$test_name\"" >/dev/null && printf '%s\n' "$e2e_events" | grep -F '"Action":"pass"' | grep -F '"Package":"orquesta/internal/bootstrap"' | grep -F "\"Test\":\"$test_name\"" >/dev/null || exit 1; done
```

Orden: contrato rojo endurecido → contrarrevisión → dominio → aplicación →
SQLite/recovery → bootstrap/E2E → P/S/E → integración limpia.
