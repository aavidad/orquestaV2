# Documento de continuidad de Orquesta — 2026-07-30

## Avance operativo del 2026-08-04

Este apartado prevalece sobre los cortes anteriores.

- B09 queda `exercised` sobre el candidato reproducible `bb90346`: dos builds
  dieron huésped `44941753…`, initramfs `837a38ed…` y perfil `04954ea4…`
  idénticos. `df6469c` cerró el falso verde musl y `BUG-ORQ-20260804-615`
  conserva la incidencia.
- El ADR B09 regulariza el corte `7c9ef00..bb90346` como
  `P=+2683/-117,V=+1179/-69`, sin reutilizar B08 ni crear holgura. V38 queda en
  `P=9.283,V=10.712`.
- B04 queda `exercised_without_kvm`: Orquesta `e80b8873` y Agente MicroVM
  `da0bd609` están ligados por commits, árboles Git, contrato SHA-256 y binario
  reproducible `faa230b9…`. El gate cruzado pasó normal y con `-race`; el
  hermano pasó 221 pruebas activas, Clippy y su conector Go normal/`-race`/vet.
- B01 queda `exercised` en `d3134bab`, `P=0,V=50`: selección, aislamiento,
  cuota, colocación y evidencia B04 permanecen ortogonales; microVM incompleta
  falla antes de construir proceso y el límite Codex no gobierna el despacho.
- B10.0b2 queda cerrado en Agente MicroVM `3fc86eb`: el puerto durable fija
  intención exacta, replay conservador, receipt reconstruible sin fabricar
  cambios privados, CAS canónico y conflicto para toda repetición divergente.
  Pasaron 218 pruebas activas, Clippy estricto y build musl; B10, SQLite y el
  primer agente físico siguen abiertos.
- B10.0b3a queda cerrado en Agente MicroVM `ee8f15c` + `56f1426`: SQLite v15
  persiste claves, intención, sesión, receipts, eventos y resultado sin otra
  autoridad de revisión/efectos; migra v14 poblada y revierte completa ante
  fallo DDL. `B10.0b3b`, recovery y el recorrido real siguen abiertos.
- No se arrancaron Orquesta, Firecracker, Jailer ni KVM. El servidor técnico
  mínimo B04 terminó cooperativamente y retiró su socket; no quedan procesos
  propios.
- Orden causal vigente: B10, B11 y B12. La microVM física real permanece
  reservada a B12; no relanzar Goals V23 antiguos.

## Avance operativo del 2026-08-02

Este apartado prevalece sobre las menciones posteriores que todavía presentan
Q4 como pendiente o A05 como meramente conectada.

- B09 queda guardada en el repositorio hermano Agente MicroVM hasta el
  candidato de código `fde9654` y el corte documental `cae246f`, sin
  acreditación física. `34bc74c`/`53403d1` fijan la misma concesión de egreso
  firmada en Rust y Go; `3b612ec` persiste intentos y comprobantes idempotentes
  en SQLite v14; `c95d808` aplica lista exacta, DNS/IP pública fijada, límites y
  túnel host; `fde9654` conecta las sesiones de agente mediante CONNECT local y
  AF_VSOCK sin NIC, TAP, puente o NAT.
- El túnel B09 es opaco: Agente MicroVM no termina TLS ni representa URL,
  ruta, consulta, cabeceras, cuerpo o credenciales. El host recibe solo
  host/puerto, reserva el intento antes de red y rechaza resoluciones vacías,
  mixtas, locales, privadas, ULA, enlace local, metadatos, multicast y rangos
  especiales. El ejecutor sigue siendo neutral para Codex, Claude, Gemini u
  otro perfil.
- Pasaron 221 pruebas Rust activas, permanecen dos smokes físicos ignorados de
  forma declarada, Clippy estricto y el conector Go normal, con `-race` y
  `vet`. No se arrancaron Orquesta, Agente MicroVM, Firecracker, Jailer o KVM.
- B09 no se promueve a `accredited`: falta reconstruir y acreditar el
  initramfs/SBOM, ejecutar ioctl/vsock/tráfico real en B12 y registrar el
  comprobante en la causalidad de Orquesta mediante B09.4/B10. El delta físico
  bruto es `+3836/-181`; excede el presupuesto prospectivo y requiere medida
  `P/V` y corrección canónica del plan, sin reutilizar la compensación B08.
- Para reanudar desde otro equipo, leer primero
  `agente_microvm/docs/HANDOFF_CONTINUACION_2026-08-02.md`. No relanzar bloques
  ya implementados. El orden causal siguiente es reconstrucción/evidencia B09,
  corrección presupuestaria, B04, B10, B11 y B12. Orquesta solo se arranca en
  la prueba final pedida por el operador y se detiene con limpieza exacta.

- B08 quedó `exercised` en Agente MicroVM, candidato de código `2f65254`, sin
  `push`. `713235f`/`42ad492` implementan el intermediario `vsock` bloqueante,
  identidad bilateral exacta, separación causal por `RunRef`/CID, recuperación
  de sockets propios y parada sin sondeo. `d12d41f` añade el firmante Go
  Ed25519 neutral y portable para cualquier proveedor.
- Orquesta retiró en `59f84d98` y `0e1abc67` la autorización física duplicada,
  el desafío HMAC, `CredentialStore` y la política de red sin consumidor. La
  concesión final no exporta secretos ni tipos internos; el proxy continúa
  rechazado hasta B09.
- La medida hermana es `P=928,V=512`; la retirada exacta aporta
  `P=-1146,V=-1248` y deja el delta causal B08 en `P=-218,V=-736`, dentro del
  padre. El exceso del hijo intermediario y su compensación quedan explícitos
  en `adr_v38_b08_intermediario_compensado_2026-08-02.md`; no crean bolsa para
  B09.
- Pasaron formato, Clippy, 193 pruebas Rust activas —más dos smokes KVM
  omitidos—, nueve pruebas Go normales y con `-race`, `vet`, la suite interna
  completa de Orquesta y los focales de trazabilidad. El gate raíz conserva
  solo los dos receipts Codex caducados que se regenerarán en el smoke final.
  B08 no acredita `ORC-28` ni V38; B09 sigue antes de B04 y B12 conserva KVM.
  No se arrancó Orquesta, Agente MicroVM, Firecracker o Jailer.
- B07 quedó `exercised` en Agente MicroVM, candidato de código `1811361` y
  corte documentado local `e663bd3`, sin `push`. `c938451` reserva capacidad,
  ejecución y CID de forma atómica; `a0792b5` añade el ledger SQLite v12 de
  recuperación; `daca72b` retira la exclusión global de una sola VM; y
  `9b89f93`/`1811361` reanudan la misma operación cercada sin relanzar.
- El máximo físico configurado queda entre 1 y 256 y llega a capacidades y al
  repositorio desde una sola configuración. La misma referencia o CID se
  rechazan; referencias y CIDs distintos conviven. Replay, revisión, cerca,
  PID/inicio/`boot_id`/`cgroup`/socket/CID, reinicio, ausencia inequívoca y
  ambigüedad quedan verificados.
- B07 consume `P=490,V=418` de `P<=500,V<=450`. Pasaron formato, Clippy, 187
  pruebas activas —más dos smokes KVM ignorados—, Go normal/`-race`/`vet` y
  veinte repeticiones tanto de la última plaza concurrente como de la
  recuperación tras reinicio.
- El read-model local no encontró una lección para
  `ORC-28 + registrar-motor-b07`; queda documentado como hueco advisory y no
  se creó otra autoridad. B07 no acredita `ORC-28` ni V38: dos microVM físicas
  simultáneas y limpieza final pertenecen a B12. B08 y B09 siguen abiertas
  antes de B04. Orquesta, Agente MicroVM, Firecracker y Jailer no se arrancaron.
- B06 quedó `exercised` en Agente MicroVM, candidato local `8ce4c12`, sin
  `push`. `54b80fb` hace que el cliente Go acredite blobs, rutas, límites y
  raíz canónica antes del socket y vuelva a acreditar la respuesta; Rust y Go
  fijan el mismo vector. El recorrido durable ya enlaza API Unix, SQLite, CAS,
  motor/vsock y huésped sin compartir rutas o almacenes.
- Orquesta retiró en `b1824ecd` `agent_microvm_bundle` y su único consumidor
  `firecracker/launchplan`: eran 475 líneas de producto que solo proyectaban
  metadata `planned_not_applied` y no tenían consumidor productivo. B10
  compondrá el módulo Go público sin reimplementar HTTP.
- El delta B06 es `P=-282,V=-338` tras la retirada compensatoria, dentro de
  `P<=400,V<=350`. Formato, Clippy, 181 pruebas Rust —más dos smokes KVM
  ignorados—, Go normal/`-race`/`vet`, los focales de Orquesta, su suite
  `internal/cmd`, `vet` y la aceptación V38 quedaron verdes. El gate raíz solo
  conserva los dos receipts Codex caducados conocidos.
- B06 no arrancó Orquesta, Agente MicroVM, Firecracker o Jailer y no acredita
  V38. B05+B06 abren B07; B08 y B09 continúan abiertas. El E2E físico continuo
  y la persistencia final del conjunto de cambios pertenecen a B12/B10.

- B05 quedó `exercised` en `agente_microvm`, candidato local `1aec635`, sin
  `push`. `2295ee2`, `ce8ae2b` y `50d329a` fijan sesiones neutrales de agente,
  turnos idempotentes, reconexión por cursor, contrapresión y una única
  autoridad de procesos por microVM. El perfil decide Codex, Claude, Gemini u
  otro ejecutor; el núcleo huésped no conoce proveedores.
- `0970fee` embebe una SBOM SPDX 2.3. Dos construcciones independientes
  coincidieron en initramfs `7e858d…92c30` y perfil `4c1daa…26ce8`, ambos modo
  `0400`; los temporales se retiraron exactamente. Pasaron 181 pruebas Rust,
  dos smokes KVM quedaron ignorados, y las sesiones se repitieron 20 veces y
  la compatibilidad corta 100 veces.
- B05 consume `P=480,V=319` de `P<=490,V<=320`, más `A=70` líneas de datos
  SPDX sin reglas. No acredita B07, B10, `ORC-28` ni V38 y no arrancó Orquesta,
  Agente MicroVM, Firecracker o Jailer. La siguiente dependencia serial es B06;
  B05+B06 abren B07.
- B02 quedó `exercised` en `agente_microvm`, candidato local `3c827d9`, sin
  `push`. Los commits `9ad41b6`, `b8fc520` y `178d519` reservan la ejecución y
  el CID en una transacción SQLite privada, ligan el CID exacto al motor y
  reconcilian reservas expiradas con cuarentena, fencing y reutilización
  controlada.
- Ocho reservas concurrentes no colisionan; la prueba se repitió diez veces.
  Formato, Clippy, 178 pruebas Rust —más dos smokes KVM ignorados— y el cliente
  Go normal, con detector de carreras y `vet`, quedaron verdes. B02 no arrancó
  Orquesta, Agente MicroVM, Firecracker ni Jailer.
- Orquesta retiró en `127c0a45` el puerto CID, allocator y renderer físicos sin
  consumidores. El delta hermano `P=372,V=314` queda compensado conjuntamente
  en `P=-1534,V=-1909`; no existe ya una segunda autoridad CID. B02 no acredita
  B04, `ORC-28` ni V38. Su siguiente dependencia B05 ya quedó ejercitada.
- B03 quedó `exercised` en el repositorio independiente
  `/home/alberto/Trabajo/agente_microvm`, candidato local `f3ae2b4`, sin
  `push`. La aplicación no importa ni comparte estado, rutas, configuración o
  tipos de Orquesta; el protocolo sigue siendo `agentmicrovm.local.v1` por
  HTTP/1.1 sobre socket Unix privado y sin escucha TCP.
- Los commits `27ca627`, `8cdd2a9`, `cfb6302` y `e84fd2a` fijan guardas de
  independencia, vocabulario neutral v1 y arranque desde una única
  configuración TOML absoluta y estricta. B05 ya implementa eventos dentro del
  huésped, pero capacidades no anuncia `eventos_ejecucion` ni
  `recuperar_ejecucion` antes de la traducción pública B10.
- El delta B03 real frente a `bdba503` es `P=34,V=195`, dentro de
  `P<=250,V<=300`; el baseline de 29.221 líneas Rust/Go continúa medido aparte.
  Formato, Clippy, 172 pruebas Rust —más dos smokes KVM ignorados—, Go normal y
  con detector de carreras, `vet` y las pruebas focales quedaron verdes.
- B03 no acredita B04, `ORC-28` ni V38. Sus dependencias B02 y B05 ya quedaron
  ejercitadas; B06 es el siguiente bloque serial. No se arrancó Orquesta,
  Firecracker, Jailer ni Agente MicroVM.
- A08 quedó `exercised` en `e6352ec2`. La compuerta neutral usa SQLite y el
  almacén de artefactos reales con un agente sustituto contractual; conserva
  cohortes lógicas `1/16/70/500`, capacidad física parcial `1/5/10/20`,
  saturación y reservas tras reapertura, y refresco de observaciones sin KVM,
  Firecracker ni `agentmicrovm`.
- A08 consume `P=0,V=348` de `P=0,V=600`. Tres rondas normales y la pasada
  focal con detector de carreras quedaron verdes; esta última tardó 329,839 s
  en aceptación y 171,653 s en SQLite. También pasaron los contratos exactos
  de orden/ACK del buzón, parada, recuperación, preservación y despacho.
- A08 no crea todavía receipt en `product/evidence`, no cambia el contrato
  V38 de `planned` ni promueve `ORC-28`: el comprobante publicable exige los
  mismos resúmenes criptográficos de A+B+C y pertenece a C04/C05. La siguiente
  dependencia causal es B03 en el repositorio hermano ya existente
  `/home/alberto/Trabajo/agente_microvm`.
- A07 quedó `exercised` en `fc5d920c` sobre las bases `7ae98026` y
  `e4a6a98e`. El despachador ya no lee el presupuesto global como límite de
  concurrencia ni usa `ExcludeLaunch`: solo ejecuta lanzamientos que el claim
  entrega después de la reserva física durable de A04. Stop y observe siguen
  progresando, no hay giro activo y la parada drena las gorutinas propias.
- A07 consume `P=84,V=689` de `P=165,V=695`; el corte final retiró 15 líneas
  netas de producto y 6 de prueba. Las cohortes `1/5/10/16/20`, cinco rondas
  focales con detector de carreras, bootstrap completo, aplicación y `vet`
  quedaron verdes. La siguiente dependencia causal es A08, compuerta A
  neutral sin KVM.
- A06.1–A06.3 quedó `exercised` en `4e5a7ae3`, `af0639de` y `2831d3c2`.
  El contrato neutral, las migraciones 025/026 y el mismo `StateRepository`
  conservan el comprobante causal exacto; el requisito de preservación se fija
  al aceptar el lanzamiento y todas las escrituras terminales fallan cerradas
  mientras falte ese comprobante.
- A06 consumió `P=350,V=247` de `P=350,V=350`. Sus focales normales y con
  detector de carreras, aplicación, puertos, `vet` y la suite SQLite completa
  quedaron verdes; esta última tardó 170,968 s. La carrera intermitente de la
  prueba V17 quedó aislada y cerrada como `BUG-REBUILD-20260802-001`: el test
  mantiene vivo el primer efecto y prueba la cerca antes de liberar la acción
  posterior.
- Q4/A04.2–A04.4 quedó integrada en `a6c326d7`. El mismo
  `BEGIN IMMEDIATE` del claim materializa o repite la observación física,
  relee la cuota exacta, resta held una vez por `source+pool`, crea una única
  reserva y su binding 1:1, y devuelve la colocación opaca fijada.
- Receipt aceptado, prueba estructural de no aplicación, ambigüedad y
  terminalización transicionan la reserva con CAS y fencing. Un lease expirado
  puede adquirir una cerca posterior, pero replay conserva la reserva y la
  colocación originales; el pool Codex ya no reselecciona perfiles.
- El corte consumió exactamente `P=489,V=515` de `P≤489,V≤537`. SQLite,
  Codex, bootstrap, `internal/**`, `cmd/orquesta`, `vet`, compilación global,
  aceptación V38 planificada y focales `-race` quedaron verdes. La suite raíz
  solo conserva los dos receipts Codex físicos del 25 de julio, obsoletos por
  diseño hasta emitir el candidato final.
- `BUG-ORQ-20260801-613` queda cerrado en código. La latencia aislada y no
  reproducida de un E2E bootstrap se conserva como
  `BUG-ORQ-20260802-614`, sin ampliar timeouts ni ocultar un posible bloqueo.
- Orquesta, Firecracker, jailer y Agente MicroVM permanecen detenidos; no se
  relanzaron los tres Goals V23 y no se hizo `push`.
- A04/A05/A06/A07/A08 quedan `exercised`, no `accredited`. La siguiente
  dependencia causal es B03; no se arrancó Orquesta ni se relanzaron los
  Goals V23.

## Avance operativo del 2026-08-01

Este apartado prevalece sobre las menciones posteriores que todavía presentan
A05 o la aplicación hermana como inexistentes.

- La aplicación hermana existe como **Agente MicroVM** en
  `/home/alberto/Trabajo/agente_microvm`; su repositorio, crate y CLI conservan
  ese nombre y su protocolo máquina sigue siendo `agentmicrovm.local.v1`.
  Firecracker 1.16.1 y Jailer quedaron ejercitados físicamente y el candidato
  fue publicado en su repositorio independiente. Paperclip no forma parte del
  conector.
- A05.3b quedó implementada localmente en `f2cb965b`: controlador persistente
  por perfil, entorno exacto ya validado, reconexión gobernada por la política
  canónica y recogida exacta de procesos. No acredita por sí sola A05.
- A05.2b/A05.4 quedaron `wired` en `e7f9e60e` y `993e3d03`: el observador
  publica capacidad bruta renovable, Codex cataloga una colocación opaca de
  capacidad uno por perfil durable y `application` obtiene la cuota vigente,
  ordena/deduplica entregas físicas sin revisión y las presenta en
  `ClaimRequest.CapacityCandidates`. Ausencia, error, timeout, cuota agotada u
  obsoleta cierran solo nuevos lanzamientos; stop/observe siguen reclamables.
  Un catálogo vacío no activa aún la compuerta y conserva la compatibilidad
  V23 anterior al cutover. A05 no está acreditada hasta que Q4 sea el consumidor
  transaccional real.
- El corte conjunto consumió `P=180,V=120` de A05.2b y `P=44,V=107` del
  contrato preparatorio ya presupuestado de A04.2; no amplió el total V38. La
  guarda V38 exige productor, consumidor y composición, y prohíbe que la fuente
  o el catálogo adquieran DB, `StateRepository` o bucle residente.
- Pasaron completos `internal/application`, Codex y `internal/bootstrap`; las
  pruebas focales `-race` de reclamo y fuente y la guarda V38 también pasaron.
  Las suites globales y `vet` se repetirán sobre el candidato que incluya Q4.
- La suite raíz solo falla porque los receipts reales Codex V17/V22 están
  ligados al árbol anterior. Deben renovarse con el candidato final, no
  reescribirse como evidencia de este corte parcial.
- Orquesta continúa detenida, los tres Goals V23 se preservan sin relanzar y no
  se hizo `push` desde este repositorio. La siguiente dependencia causal es
  A04.2/Q4: materializar/repetir la observación y reservar/fijar exactamente una
  colocación dentro del mismo `BEGIN IMMEDIATE` del claim.

## Parada operativa del 2026-07-31

Este corte prevalece sobre el corte del 30 de julio y sobre las menciones
posteriores a procesos activos.

- El operador pidió cerrar porque va a apagar el equipo. No se arrancó la
  Orquesta real, no se recuperaron Goals y no se hizo `push`.
- Se detuvo, con autorización expresa, la única unidad Firecracker activa:
  `orquesta-firecracker-attestor-57ede79dee780873a0012f3518eeea668de2135a78a3ca4478d7c3c555a7bb09.service`.
  Quedó `inactive` y `disabled`; el censo final no encontró `firecracker`,
  `jailer`, `microvm` ni launchers activos.
- `agentmicrovm` sigue sin existir. Se mantiene el orden causal: terminar la
  compuerta A antes de crear el repositorio hermano en B03.
- Se retiró el lector histórico de cuota por fichero y se implementó el
  controlador persistente `codex app-server` por perfil, el escritor neutral
  de observaciones, la ordenación/deduplicación y su composición antes del
  planificador.
- El E2E físico del entorno Go fallaba porque heredaba un
  `scheduler.claim_lease=1s`, menor que la composición con cgroups. La fixture
  usa ahora `10s`, pasa en 3,88 s y la incidencia
  `BUG-REBUILD-20260731-001` conserva causa e invariante.
- A05.3b/A05.4 sigue parcial y no acreditada. Antes de cerrarla hay que medir y
  resolver su envolvente conjunta `P≤237,V≤146`: el código actual la supera.
  También debe decidirse de forma canónica la demora de reconexión actualmente
  fijada a un segundo y ejecutar las suites completas, carreras y vet.
- La suite completa `./internal/... ./cmd/orquesta` y `go vet` están verdes.
  La suite raíz solo mantiene rojos
  `TestRealCodexReceiptMatchesCurrentProductSource` y
  `TestV17RealCodexReceiptMatchesCurrentProductSource`: el receipt físico del
  25 de julio referencia un árbol anterior. No se renovó porque exige el E2E
  Codex real y el operador ordenó no arrancar Orquesta antes de completar la
  aplicación hermana.

Commits locales nuevos:

```text
25b4eb4a retira el lector de cuota por fichero
56254b17 controla la cuota Codex por perfil
1a1b5f92 pruebas: calibra el lease del entorno Codex
531d5044 aplicación: persiste y ordena la cuota observada
a93ab6b7 arranque: compone los controladores de cuota
395c7a6e documentación: fija la parada segura de V38
09f00f63 documentación: registra Firecracker deshabilitado
a3955213 trazabilidad: conserva el lote cinco de conductas
cc3ad964 trazabilidad: conserva el lote seis de conductas
2c6acbac documentación: añade el manual del orquestador
```

Pruebas verdes del último corte:

```text
go test -mod=vendor -count=1 ./internal/application \
  -run '^(TestRegistrarObservacionCuotaPersisteEvidenciaYRepiteExactamente|TestOrdenarCandidatosColocacionDeduplicaYRechazaConflictos|TestAgentQuotaGateIsSeparateAndFailsClosed)$'
go test -mod=vendor -count=1 ./internal/adapters/agent/codex \
  -run '^(TestControladorCuotaLeeReconectaRotaYCierraExactamente|TestControladorCuotaRechazaConfiguracionInvalida)$'
go test -mod=vendor -count=1 ./internal/bootstrap \
  -run '^(TestAbrirControladoresCuotaRecogeTodosAnteFalloInicial|TestCodexProductionProcessReceivesPinnedGoEnvironment)$'
go test -mod=vendor -count=1 ./internal/... ./cmd/orquesta
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta
git diff --check
```

La siguiente acción causal es A04.2/Q4; A05 está conectada pero no acreditada.
No arrancar Orquesta ni Firecracker antes de consultar de nuevo el estado vivo.

### Reanudación desde otro equipo

La rama de continuidad es `integracion/v23-intake-durable` en
`git@github.com:aavidad/orquestaV2.git`. En un equipo nuevo:

```bash
git clone git@github.com:aavidad/orquestaV2.git
cd orquestaV2
git switch integracion/v23-intake-durable
git pull --ff-only
sed -n '1,220p' AGENTS.md
sed -n '1,220p' docs/reconstruccion/LEEME_AGENTE_ORQUESTAV2.md
sed -n '1,180p' docs/reconstruccion/handoff_continuacion_agente_2026-07-30.md
git status --short --branch
git log -12 --oneline --decorate
```

Después se consulta el estado vivo antes de actuar. Orquesta y Firecracker
quedaron detenidos y Firecracker sin autoarranque. No se recuperan ni relanzan
tareas existentes por intuición. El manual complementario, que no sustituye
al roadmap ni a este handoff, comienza en
`docs/reconstruccion/manual_orquestador/README.md`.

## Corte operativo más reciente

Este apartado prevalece sobre cualquier estado histórico posterior del mismo
documento.

- El operador ha ordenado detener la sesión al cerrar las tareas abiertas.
- No se ha usado Orquesta porque no dispone de cuota. La coordinación directa
  de Codex se mantiene como excepción temporal de arranque.
- No se ha hecho `push`.
- La mención histórica a una aplicación inexistente quedó sustituida: Agente
  MicroVM existe en `https://github.com/aavidad/agente_microvm`, con checkout
  operativo actual en `/home/alberto/Trabajo/agente_microvm`.
- Orquesta solo lo consumirá mediante `agentmicrovm.local.v1` sobre socket Unix.
  No compartirán base de datos, sistema de archivos, rutas, secretos ni
  importaciones.
- El presupuesto V38 regularizado es `P=9.283,V=10.712`; B09 sustituyó su
  estimación por el corte reproducible exacto mediante el ADR del 2026-08-04.
  A02 queda
  `170/238`, A03 `334/537`, A05 `682/508` y A07 `165/695`.
- La compuerta completa de agentes autónomos, microVM y Firecracker sigue
  incompleta y no acreditada.

Commits locales cerrados durante este corte:

```text
10f014ff fija PostgreSQL como adaptador productivo
69be0174 reordena el presupuesto causal de V38
4122dcf1 acota el protocolo de cuota de Codex
8578b503 añade la migración de cuota y colocación
4efd182d acredita las invariantes de cuota y colocación
b09a4695 traduce la cuota oficial de Codex
7b32df2e valida la recuperación de cuota y colocación
6165b65e persiste la cuota vigente por colocación
8108ec27 separa la aplicación de gestión de microVM
```

Pruebas repetidas al cerrar:

```text
go test -mod=vendor -count=1 ./internal/application
go test -mod=vendor -count=1 ./internal/adapters/state/sqlite \
  -run '^TestAgentQuotaAppendCurrentIsolationAndRestart$'
GOFLAGS=-mod=vendor go vet ./internal/application ./internal/adapters/state/sqlite
go test -mod=vendor -count=1 . \
  -run '^TestProductRoadmapIsExhaustiveAndCausal$'
go test -mod=vendor -count=1 ./acceptance \
  -run '^TestV38AgentRuntimeElasticPlan'
```

La suite SQLite completa también quedó verde antes del último cambio de textos:
`go test -mod=vendor -count=1 ./internal/adapters/state/sqlite`, 169,495 s.

## Objetivo invariable

Terminar Orquesta al 100 %, sin volver a convertir V23 en un bloque
indivisible. La prioridad vinculante corregida por el operador es:

1. abrir V38 `agent_runtime_elastic` sobre sus prerrequisitos ya acreditados,
   sin depender del cierre de V23 y con `ORC-28` como su única capacidad;
2. pasar la compuerta A del núcleo elástico neutral, sin exigir KVM ni
   Firecracker y sin acreditar todavía V38;
3. pasar la compuerta B mediante la aplicación hermana `agentmicrovm` y su
   conector local de activación explícita, sin sustitución automática, con una
   microVM por agente y sin acreditar todavía V38;
4. pasar la compuerta C mediante una ola física real sobre el mismo candidato
   de A y B; solo A+B+C acreditan V38;
5. demostrar por separado cohortes lógicas de 1, 16, 70 y 500 y escalones
   físicos de 1, 5, 10, 16 y 20, sin afirmar 70 o 500 agentes físicos cuando
   no existan recursos medidos;
6. usar la propia Orquesta elástica para completar V23 y los demás frentes;
7. ejecutar gates globales, revisar y cerrar el producto.

`ORC-15` permanece en V27 y `OPS-16`/`OPS-17` permanecen en V32. La
continuidad de mensajes, la parada exacta y la conservación del entorno son
conductas estrechas exigidas por V38, no capacidades que V38 reabra o acredite.
`EVD-13` y `TestAttestor` siguen siendo prerrequisitos ya acreditados; la
gestión de agentes Firecracker de la compuerta B no los sustituye. El entorno
general de ejecución no se difiere a V39: V39 no existe.

No se debe declarar terminado un corte por porcentaje, documentación o pruebas
locales: hacen falta change-set, atestación, revisiones, gobernanza aplicable,
integración y evidencia durable.

## Primera acción del siguiente agente

Antes de actuar:

```bash
cd /home/alberto/Trabajo/orquestaV2
git status --short --branch
git log -10 --oneline --decorate
```

No relanzar tareas ni usar Orquesta mientras siga sin cuota. Preservar los
archivos ajenos sin seguimiento.

La siguiente dependencia causal es A06:

1. caracterizar el contrato neutral de preservación y su relación exacta con
   una ejecución, cerca, base de workspace, change-set/bundle, inventario,
   configuración y rootfs;
2. persistir el comprobante durable antes de desmontar y fallar cerrado ante
   ausencia, digest alterado, proyecto ajeno o cerca obsoleta;
3. impedir que V38 introduzca una operación de borrado/GC o un nuevo estado del
   `Goal`;
4. cubrir repetición, reinicio y material todavía direccionable; después abrir
   A07 y la compuerta neutral A08.

Q4 consumió `P=489,V=515` de su bolsa `P≤489,V≤537` sin aumentar V38. La
migración progresiva es 024 y 022/023 siguen inmutables. Los commits
`a6c326d7`, `79799a4e` y `ad3c549f` conservan implementación y trazabilidad.

Antes de cada edición se debe consultar `ORC-28` mediante
`scripts/consultar_lecciones_legacy.sh`. La consulta de las tareas cerradas en
este corte no encontró patrones; se conservó como hueco consultivo.

No recrear ni renombrar Agente MicroVM. B03, después de la compuerta A,
caracteriza y completa el proyecto hermano ya existente; nunca lo copia bajo
un subdirectorio de Orquesta.

V23 queda preservada como frente posterior e independiente. No se relanza ni se
mezcla con el write-set de V38. Solo cuando se retome V23 se consulta su estado
vivo antes de editar:

```bash
STATE=/home/alberto/Trabajo/.orquesta-runtime-v2-v23/Codex12/state/orquesta-v23-microtasks-r3.sqlite
sqlite3 -header -column "$STATE" "
SELECT g.ref AS goal_ref,g.state,g.revision,w.state AS work_state,
       e.ref AS execution_ref,e.state AS execution_state,e.purpose,e.failure_code
FROM goals g
LEFT JOIN work_items w ON w.goal_ref=g.ref
LEFT JOIN executions e ON e.goal_ref=g.ref
WHERE g.ref IN (
  'goal:949f11c9635a1dc1b88d75a9103392d1',
  'goal:5a18a5071ae6bc9b0e13dc5c35f6eeda',
  'goal:d5daacfd3dba14624f50bdf89e0c89e1'
)
ORDER BY g.ref,e.created_at;"

sqlite3 -header -column "$STATE" "
SELECT o.goal_ref,o.kind,o.effect_intent_ref,o.last_error_code,i.digest
FROM outbox o
LEFT JOIN effect_intents i ON i.ref=o.effect_intent_ref
WHERE o.completed_at IS NULL AND o.retired_at IS NULL
ORDER BY o.available_at;"
```

No relanzar un frente hasta comprobar si su agente, change-set o artefacto ya
existe.

## Repositorios y runtime

- Producto: `/home/alberto/Trabajo/orquestaV2`.
- Rama: `integracion/v23-intake-durable`.
- El último commit de producto antes de actualizar este documento es
  `8108ec27`. Comprobar el `HEAD` real al reanudar.
- No hacer `push` sin una orden nueva del propietario.
- Semilla Git de Orquesta:
  `/home/alberto/Trabajo/.orquesta-runtime-v2-v23/Codex12/repositories/orquestaV2-v23-microtasks`.
- Estado SQLite:
  `/home/alberto/Trabajo/.orquesta-runtime-v2-v23/Codex12/state/orquesta-v23-microtasks-r3.sqlite`.
- Servicio de usuario: `orquesta-v23-pool.service`.
- API local: `http://127.0.0.1:18225`.
- Binario desplegado:
  `/home/alberto/Trabajo/.orquesta-runtime-v2-v23/tools/orquesta-804f519e`.
- La credencial se referencia mediante `--credential-file`; nunca se copia en
  documentación, prompts, logs ni comandos versionados.

## Trabajo vivo preservado

### ORC-03d: contrato nuclear de microtareas

- Goal: `goal:949f11c9635a1dc1b88d75a9103392d1`.
- Ejecución:
  `execution:1b96a3ad6ca2ead7230894a56cc3b164`.
- Estado al cerrar: autor activo.
- Write-set:
  - `docs/reconstruccion/contrato_minitareas_nucleo_2026-07-29.md`
  - `docs/reconstruccion/worksets/orc03_minitask_core_v1.json`
  - `orc03_minitask_core_plan_test.go`
- Prueba focal:

```bash
go test -mod=vendor -count=1 -v . \
  -run '^TestORC03MinitaskCorePlan$'
```

Debe corregir los hallazgos ya preservados en:

- `artifact:sha256:0669e611da025fc7de9ceb01231bce769a2f4556cbc0e35602779a76db5b4ca7`
- `artifact:sha256:36e38c0370f1bdedf22f1eb6e2fe734716e256202901ff032d4aaed18c2563b5`

No programar producción desde este Goal: solo contrato, manifiesto y test.

### BUG611b: límites de payload de revisiones

- Incidencia:
  `BUG-ORQ-20260730-611`.
- Goal: `goal:5a18a5071ae6bc9b0e13dc5c35f6eeda`.
- Ejecución:
  `execution:95f3efc68a5f2494ab06ca80bd5f3e0c`.
- Estado al cerrar: autor en `awaiting_integration`, atestación Firecracker
  `passed` con `exit_code=0`; revisiones principal y adversarial en cola.
- Candidato base preservado:
  `change-set:60b53618de802cfe892cadc6c46f1e39`.
- Change-set corregido:
  `change-set:b3fe43c194156e13d524ded6cdd9e426`, head
  `6d3990266f47b9f0fdf96c56b11d7453c0173d41`.
- Única corrección esperada: conservar el cambio y renombrar el test a
  `TestReviewerLaunchRequestPublishesReviewPayloadLimits`.
- Prueba focal:

```bash
go test -mod=vendor -count=1 -v ./internal/application \
  -run '^TestReviewerLaunchRequestPublishesReviewPayloadLimits$'
```

Está prohibido repetir `./internal/...` o `./...` durante esta autoría.

### V23-02: snapshot exacto del Wizard

- Goal revisado:
  `goal:d5daacfd3dba14624f50bdf89e0c89e1`.
- Change-set preservado:
  `change-set:ba9a164f39f2d321310e50c94667a895`.
- Head candidato: `17b59f7e57a59733e6c263938332c3a5cec25e77`.
- Revisión principal: `approve`.
- Revisión adversarial: `changes_requested`, artefacto
  `artifact:sha256:19341db540967c964e83bc47f8b6f19147f3d1251d171aa01a468d8dfb0bd4ed`.

El defecto es real: el candidato construye el binding después de persistir el
receipt y no lo transporta en `WizardGapsInputRecord` ni en las reservas. Tras
reinicio se recalcula; no existe persistencia causal atómica.

No hacer otro parche monolítico. Continuar el DAG ya diseñado en
`docs/reconstruccion/worksets/v23_agent_microtasks_v1.json`:

1. **V23-02 aplicación**, máximo seis ficheros: crear binding antes de reservar;
   transportarlo en record y reservas; validar ref, digest y bytes; distinguir
   replay durable de legacy.
2. **V23-03/04 SQLite**, migración forward-only y store: persistir receipt y
   snapshot en la misma transacción; no cambiar el hash del receipt ni inventar
   evidencia legacy.
3. **V23-05 recovery**: XOR estricto entre fila legacy sin snapshot y fila nueva
   con snapshot completo; tamper, restart y migración desde schema 21.
4. **V23-06 público**: `evaluation_replay_exact=true` solo al restaurar bytes
   durables exactos.

Después siguen V23-07, V23-09, V23-10, revisiones V23-11/12 y sello V23-13.
No afirmar que V23-03..06 ya están cerradas: el árbol actual solo contiene la
migración `021_wizard_gaps_inputs.sql`.

## Disciplina para cada microtarea

1. Leer `AGENTS.md`, instrucciones locales y fuentes vigentes.
2. Ejecutar preflight legacy aplicable.
3. Declarar un write-set disjunto y pequeño.
4. Usar Orquesta para crear y dirigir el Goal.
5. Ejecutar únicamente pruebas focales durante autoría.
6. Inspeccionar diff y comprobar `git diff --check`.
7. Aprobar commit y atestación Firecracker solo con evidencia real.
8. Exigir revisión principal y adversarial.
9. Resolver Council cuando la política lo requiera.
10. Integrar explícitamente y trasladar el commit al repo producto.
11. Commit pequeño, en castellano y con una sola responsabilidad.
12. Registrar cualquier fallo o falso verde en
    `docs/inventario_bugs_orquesta_2026-06-30.md`.

No usar `codebase-memory-mcp` por defecto, no lanzar indexadores, no esperar
todos los agentes vivos del run y no matar procesos por patrón. Para higiene:

```bash
ps -eo pid,ppid,stat,pcpu,pmem,etimes,comm,args --sort=-pcpu |
  rg 'orquesta|firecracker|jailer|codex|go test'
```

Un proceso solo se termina con owner/ejecución comprobados.

## Aprobación de efectos

Usar variables locales y un `request-ref` nuevo; nunca pegar el token:

```bash
BIN=/home/alberto/Trabajo/.orquesta-runtime-v2-v23/tools/orquesta-804f519e
TOKEN_FILE=/ruta/privada/al/token

"$BIN" command \
  --url http://127.0.0.1:18225 \
  --credential-file "$TOKEN_FILE" \
  --max-credential-bytes 4096 \
  --max-response-bytes 1048576 \
  --timeout 20s \
  --request-ref request:unico \
  --project-ref project:default \
  --payload '{
    "goal_ref":"goal:...",
    "intent_ref":"effect-intent:...",
    "expected_intent_digest":"...",
    "decision":"approved",
    "reason":"Evidencia inspeccionada y alcance acotado."
  }' \
  -- effects decide
```

El `intent_ref` y digest deben obtenerse del outbox vivo, no de este documento.

## Firecracker y V38 sin esperar al cierre de V23

El TestAttestor Firecracker está instalado y sirve para atestar pruebas, pero
quedó detenido por orden del operador el 31 de julio. Todavía no es runtime de
agentes.

V23 no depende de Firecracker, KVM ni microVM. A la inversa, V38 y su adaptador
Firecracker tampoco esperan a que V23 termine: avanzan mediante dependencias y
conjuntos de escritura separados.

Las tres compuertas canónicas son:

1. **A, núcleo neutral**: observación, reserva y liberación de capacidad,
   despacho global, prioridad de parada, progreso de observación, reinicio y
   recuperación sin KVM ni Firecracker. Esta compuerta no acredita V38.
2. **B, aplicación hermana y Firecracker**: `agentmicrovm` se activa de forma
   explícita por su protocolo local, crea una microVM por agente y usa un
   sistema de archivos raíz inmutable con Codex. Conserva concesión CID
   durable, intermediario y proxy controlados por `vsock`, transporte aislado,
   credenciales efímeras, parada y sellado. Esta compuerta tampoco acredita
   V38.
3. **C, ola física**: ejecutar sobre el mismo candidato de A y B los escalones
   1, 5, 10, 16 y 20. Solo después de A+B+C puede acreditarse V38.

Las cohortes lógicas 1, 16, 70 y 500 prueban cálculo completo de demanda, no
prometen esas cantidades físicas. Si la capacidad no basta, la misma
`Execution` espera sin consumir intento.

No usar NAT, TAP, puente, interfaz de red del huésped, Internet directo ni Git
del anfitrión. El huésped recibe un paquete sellado, trabaja en un sistema de
archivos aislado y devuelve un conjunto de cambios al intermediario. El proxy
del anfitrión por `vsock` aplica una lista permitida y bloquea redes locales,
metadatos, SSRF y comunicación lateral entre agentes. No reutilizar el huésped
del `TestAttestor` como sistema de archivos raíz de Codex.

## Regla de cierre

Preservar todo trabajo útil, incluso candidatos rechazados. No borrar legacy;
consultarlo mediante el índice vigente, sin mezclarlo en el árbol productivo.
Cuando una revisión encuentre un defecto real, crear una hija pequeña: no
ensanchar el WorkItem ni declarar que el control es un bloqueo circular.
