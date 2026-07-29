# Corte neutral: una microVM bajo la autoridad de red existente

Fecha: 2026-07-29.

Autoridad de apertura: `input:operator-authorization-2026-07-29`.

Estado: caracterización contractual; no modifica estado de producto.

## Autoridad única

`product/roadmap.json` conserva una sola decisión de implementación para este
frente: `agent_microvm_network`, con estado `planned_not_applied`. Sus
capabilities son `AGT-01`, `AGT-03`, `EVD-13` y `ORC-15`.

La fixture `acceptance/fixtures/agent_firecracker_single_vm.json` caracteriza
un ejemplo acotado a una microVM y un intento. No es otra decisión, capability,
vertical, acceptance contract, composición, ejecución ni evidencia de
acreditación. Tampoco atribuye una identidad o un binario de agente al guest.

## Conectividad cerrada

El transporte futuro continúa siendo `vsock_only`. La allowlist exacta mantiene
los nombres canónicos `orquesta_broker` y `controlled_egress_proxy`.

Quedan prohibidos:

- red IP del guest;
- TAP, bridge y NAT;
- tráfico inbound y east-west;
- acceso directo a Internet.

El CID y el puerto vsock no son identidad. Un servicio solo puede hacerse
alcanzable tras comprobar la política causal, la atestación de lanzamiento y
una prueba de credencial de un uso obtenida mediante `CredentialStore`. La
prueba ejecutable está en
`internal/adapters/agent/firecracker/networkauth/verifier_test.go`: autoriza un
único ganador, rechaza replay idéntico y conserva un solo uso del almacén,
también bajo carrera concurrente.

No hay fallback a una ruta IP, servicio adicional o credencial reutilizable.
Un fallo permanece fail-closed y no promueve el estado
`planned_not_applied`.

## Separación de TestAttestor

`TestAttestor` no consume esta caracterización. Su alcance canónico permanece
`unchanged_no_network_no_vsock`: sin red y sin vsock. El ratchet ejecutable
`TestValidatePhaseRejectsEveryRequiredPhysicalInvariant` rechaza de forma
independiente la presencia de cualquiera de esos mecanismos.

La prueba de `CredentialStore` pertenece a la autorización de red de agentes;
la prueba de ausencia de red/vsock pertenece a `TestAttestor`. Compartir
Firecracker como tecnología no fusiona contratos, receipts ni acreditaciones.

## Resultado de este corte

Se conserva del change-set recuperado la frontera segura de una microVM, la
ausencia de conectividad IP, el uso único de credencial y la separación del
atestador. Se retiran el nombre de vertical de los artefactos y cualquier
segunda autoridad de implementación.

No se crea capability, vertical, acceptance contract ni receipt. No hay wiring
productivo ni ejecución física acreditada. La siguiente dependencia causal
sigue siendo composición explícita y E2E físico sobre un candidato futuro antes
de cambiar `planned_not_applied`.

## Evidencia de validación y frontera del huésped

La primera atestación física sí arrancó una microVM, ejecutó el candidato y la
limpió, pero el combinado de aceptación terminó con exit `1`. La reproducción
fiel del entorno aislado identificó un único gate incompatible:
`TestAcceptanceV03CanonicalLedgers` invoca Git y necesita `.git`, mientras el
huésped recibe deliberadamente un snapshot sin metadata Git y un `PATH`
hermético con solo el toolchain Go.

V03 permanece como gate fuerte de host. La atestación física debe repetir solo
pruebas Go herméticas que no necesiten Git, red ni estado exterior: el contrato
neutral, la credencial de un uso y la separación de red/vsock del
`TestAttestor`. No se añadirá Git ni `.git` al huésped para fabricar un verde y
no se afirmará que V03 fue acreditado dentro de Firecracker.

Esta restricción pertenece al huésped mínimo de `TestAttestor`, no al futuro
agente. Una microVM de agente podrá contener su propio ejecutable Git y una
copia aislada del repositorio recibida por vsock. Nunca montará ni leerá el
workspace Git del host: devolverá el changeset o bundle por el broker para que
el host lo valide e integre.

## Integración sobre la rama vigente

El changeset se produjo desde una semilla once commits atrasada. Al detectarlo,
la semilla se adelantó de forma segura a la rama vigente y se detuvo
cooperativamente el sucesor antes de que publicase otro candidato obsoleto. La
parada dejó una incidencia separada: la ejecución terminó
`codex.execution_stopped`, pero el control continuó `requested`, su WorkItem
siguió `running` y apareció un retry `queued`; el replan quedó bloqueado por
esa combinación ambigua.

No se modificó SQLite para ocultar la inconsistencia. La integración manual
queda limitada al changeset neutral ya revisado, resolviendo su colisión
documental contra la rama actual. El lifecycle de parada, recovery y replan
debe corregirse con prueba durable antes de considerarlo cerrado.
