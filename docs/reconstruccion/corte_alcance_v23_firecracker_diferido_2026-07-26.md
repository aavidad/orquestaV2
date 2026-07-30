# Corte de alcance: V23 no depende de Firecracker

Fecha: 2026-07-26.

Estado: decisión vinculante de alcance para terminar V23 sin volver a inflarlo.

## Decisión

V23 conserva su propósito original: Wizard, intake compartido, dossier,
confirmación causal, generación editorial y flujos restantes declarados por
`AC-V23-WIZARD`.

Firecracker no es dependencia, gate, requisito de promoción ni criterio de
sello de V23. Tampoco lo son una topología `1 + 16`, KVM, `jailer`, una imagen
guest, un launcher microVM o la red futura de agentes.

## Evidencia del producto actual

- `product/roadmap.json` declara para V23 únicamente `intent_appspec`,
  `goal_dag_phases`, `command_registry` e `i18n`;
- `AC-V23-WIZARD` no contiene una aserción Firecracker;
- `test_attestor.provider` admite `disabled`, `bubblewrap` y `microvm`, con
  `disabled` como default canónico;
- bootstrap solo construye el adaptador microVM cuando la composición elige
  `microvm`;
- la configuración `disabled` no exige socket, digest de assets ni guest.

Por tanto, conservar código Firecracker compilable no lo convierte en
autoridad ni precondición de V23.

## Corrección prioritaria de 2026-07-30

V23 continúa sin depender de Firecracker, pero ya no precede al runtime de
agentes. El operador ha incorporado una única V38 canónica:

| Vertical | Alcance | Precondición |
| --- | --- | --- |
| V38 `agent_runtime_elastic` | Demanda completa sin techo oculto, arranque paralelo, una microVM por agente, parada individual, sellado, inventario, conservación y recuperación | `config`, `credentials`, `recovery_backup`, `controls`, `budgets_effects`, `workspace_git`, `test_attestor` y `codex_e2e`, ya acreditadas |

V38 amplía el roadmap a 38 verticales sin añadir capability IDs: el catálogo
permanece en 257. Su contrato está planificado; no crea receipts ni evidencia
anticipada. El antiguo candidato V38 de activación aislada de `TestAttestor`
queda sustituido sin número y V39 no se crea.

El gate se divide sin solapamientos:

| Subgate | Contrato | Puede acreditar V38 |
| --- | --- | --- |
| A | Núcleo elástico neutral, sin exigir KVM ni Firecracker. | No. |
| B | Firecracker para agentes, activación expresa sin sustitución automática y una microVM por agente. | No por sí solo. |
| C | Ola física real del mismo candidato que superó A y B. | Sí, únicamente después de A y B. |

Por tanto, solo A, B y C superados por el mismo candidato acreditan V38.
Ningún verde aislado, fixture neutral, prueba unitaria o atestación de
`TestAttestor` sustituye esa composición.

## Corrección autoritativa de 2026-07-29

La autorización `input:operator-authorization-2026-07-29` ratifica
`agent_microvm_network` como única decisión de implementación. Su estado sigue
`planned_not_applied`, con transporte `vsock_only`, servicios exactos
`orquesta_broker` y `controlled_egress_proxy`, y prohibición de red IP del
guest, TAP, bridge, NAT, inbound, east-west e Internet directo.

La fixture neutral `acceptance/fixtures/agent_firecracker_single_vm.json`
caracteriza el subconjunto de una microVM y conserva la prueba de
`CredentialStore` de un uso. No crea otra decisión, vertical, capability,
acceptance contract, receipt, evidencia ni afirmación de ejecución física.
Tampoco completa el subgate B: no hay cableado productivo ni lanzamiento físico.

Este ratchet no convierte Firecracker en gate de V23, no altera el default
`test_attestor.provider=disabled` y no modifica `TestAttestor`, que continúa
sin red ni vsock.

## Reglas para no reabrir el error

- una incidencia Firecracker no bloquea trabajo del Wizard;
- un smoke SQLite no debe encadenar Firecracker como “siguiente gate de V23”;
- una prueba V23 no debe requerir KVM, launcher, socket, guest ni provider
  `microvm`;
- el default `disabled` debe seguir permitiendo build, arranque y cierre sin
  infraestructura Firecracker;
- no se extrae ni reescribe ahora el adaptador existente: queda opt-in y fuera
  del camino de V23;
- la fixture de una microVM no puede promover `planned_not_applied`, acreditar
  V38 ni crear o reutilizar receipts o evidencias;
- Bubblewrap tampoco se reabre hasta que Orquesta sea autoprogramable.

## Qué sigue realmente en V23

El siguiente trabajo se obtiene de `AC-V23-WIZARD` y de su fixture vigente:
cerrar las capacidades WIZ restantes, completar generación editorial y flujos
del Wizard, ejecutar sus gates exactos y emitir un receipt V23 de la misma
revisión. Ninguna de esas acciones necesita Firecracker.
