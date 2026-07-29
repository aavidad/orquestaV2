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

## Orden posterior

Las ideas nuevas se colocan al final y no se insertan dentro de V23:

| Candidato | Alcance | Precondición |
| --- | --- | --- |
| V38 | Activación y acreditación real del adaptador Firecracker de `TestAttestor`, sin red | V23 cerrado y Orquesta autoprogramable |
| V39 | Evaluación del runtime de agentes en microVM, hasta 16 instancias, red aislada, broker/proxy controlado y RAM/tmpfs | V38 acreditado y contrato propio aprobado |

V38 y V39 son candidatos posteriores al catálogo V1-V37 vigente. Esta decisión
no amplía todavía el catálogo de 257 capacidades, no crea receipts y no
autoriza implementación. Cuando se abran deberán incorporarse mediante
contratos, dependencias, presupuesto, negativos, restart/recovery y evidencia
propios.

## Corrección autoritativa de 2026-07-29

La autorización `input:operator-authorization-2026-07-29` ratifica
`agent_microvm_network` como única decisión de implementación. Su estado sigue
`planned_not_applied`, con transporte `vsock_only`, servicios exactos
`orquesta_broker` y `controlled_egress_proxy`, y prohibición de red IP del
guest, TAP, bridge, NAT, inbound, east-west e Internet directo.

La fixture neutral `acceptance/fixtures/agent_firecracker_single_vm.json`
caracteriza el subconjunto de una microVM y conserva la prueba de
`CredentialStore` de un uso. No crea otra decisión, vertical, capability,
acceptance contract, receipt ni afirmación de ejecución física.

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
- la fixture de una microVM no puede promover estado ni reutilizar receipts;
- Bubblewrap tampoco se reabre hasta que Orquesta sea autoprogramable.

## Qué sigue realmente en V23

El siguiente trabajo se obtiene de `AC-V23-WIZARD` y de su fixture vigente:
cerrar las capacidades WIZ restantes, completar generación editorial y flujos
del Wizard, ejecutar sus gates exactos y emitir un receipt V23 de la misma
revisión. Ninguna de esas acciones necesita Firecracker.
