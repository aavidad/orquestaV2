# Handoff de parada de Orquesta V2

Fecha: 2026-07-26.

Rama: `integracion/v23-intake-durable`.

La revisión autoritativa es el commit que contiene este documento. El árbol se
dejó limpio y sincronizado con `origin`.

## Corrección autoritativa de 2026-07-29

La autorización `input:operator-authorization-2026-07-29` conserva una sola
autoridad: `agent_microvm_network`, `planned_not_applied`, bajo `AGT-01`,
`AGT-03`, `EVD-13` y `ORC-15`. La fixture neutral de una microVM no crea otra
decisión, vertical, capability, acceptance contract ni receipt.

El contrato mantiene `vsock_only`, los servicios exactos `orquesta_broker` y
`controlled_egress_proxy`, y prohíbe red IP del guest, TAP, bridge, NAT,
inbound, east-west e Internet directo. La autorización exige prueba de
`CredentialStore` de un uso. `TestAttestor` conserva su alcance independiente,
sin red ni vsock.

## 1. Estado que no debe reinterpretarse

| Superficie | Estado |
| --- | --- |
| Roadmap nuevo | 37 verticales nuevas, V1-V37 |
| Contratos ejecutables/acreditados | V1-V22, 22 contratos |
| Contratos planificados | V23-V37, 15 contratos |
| V23 | `partial_green_unsealed`; Wizard/intake/dossier en curso |
| Runtime legacy | fuera de la vista activa y prohibido como dependencia |
| Firecracker | código opt-in conservado; no es gate de V23 |
| V38/V39 | candidatos documentados, todavía no verticales canónicas |
| Fixture de una microVM | caracterización neutral de `agent_microvm_network`; no cambia estado |

V1-V37 no son generaciones del antiguo Director. Son verticales causales del
producto reconstruido. “Sin dependencia del legacy” significa que el producto
nuevo no importa, arranca, adapta ni comparte estado con `modulos/**` o
`cmd/orquesta-server/**`. No significa que las verticales carezcan de
dependencias entre sí o de adaptadores externos.

## 2. Árbol activo y legado

Producto:

`/home/alberto/Trabajo/orquestaV2`

Consulta legacy completa:

`/home/alberto/Trabajo/orquestaV2-legacy-consulta`

Bundle independiente verificado:

`/home/alberto/Trabajo/orquestaV2-backups/2026-07-26-pre-separacion-legacy/orquestaV2-completo.bundle`

SHA-256 del bundle:

`d819211492e7e95d837fe54806e40037b328b6c18e3cac2cc30489fedf25ba97`

Rama y tag remotos de rescate:

`backup/pre-separacion-legacy-20260726`

El worktree activo usa `sparse-checkout`. No se borró contenido. Quedaron fuera
de la vista `modulos/**`, comandos antiguos, salidas OPES históricas,
documentos raíz obsoletos, 26 scripts legacy y seis tests raíz del runtime
anterior. El manifiesto reversible exacto está en
`separacion_legacy_consulta_2026-07-26.md`.

No desactivar sparse-checkout ni materializar `modulos/**` para corregir un
import. La copia de consulta es solo lectura.

## 3. Firecracker: qué existe realmente

### TestAttestor microVM

Código presente en la arquitectura nueva:

- cliente: `internal/adapters/attestor/firecrackerclient`;
- guest: `internal/adapters/attestor/firecrackerguest`;
- launcher: `internal/adapters/attestor/firecrackerlauncher`;
- protocolo neutral:
  `internal/testattestorprotocol/{launcher,rawdrive}`;
- supervisor y publicación E2E:
  `internal/e2e/firecrackerattestor*`;
- binarios auxiliares:
  `cmd/orquesta/{firecracker-launcher,firecracker-attestor-e2e,test-guest}`;
- builders, verificador, instalador y smoke bajo `scripts/*firecracker*`.

Configuración canónica:

```text
test_attestor.provider = disabled | bubblewrap | microvm
default = disabled
```

Bootstrap construye el adaptador Firecracker solo con `microvm` explícito.
`disabled` no exige KVM, socket, launcher, guest ni digest de assets. No existe
provider `auto` ni fallback silencioso.

El código compila y tiene cobertura offline/focal, pero la activación física no
queda acreditada por este handoff. No hay receipt que permita convertirla en
precondición de V23 o activar un servicio de producción.

### MicroVM para agentes

Preparación presente:

- contratos:
  `internal/ports/agent_microvm_{network,launch_auth,vsock_cid}.go`;
- adaptadores parciales:
  `internal/adapters/agent/firecracker/{networkplan,networkauth,vsocklease}`;
- decisión de implementación:
  `product/roadmap.json -> agent_microvm_network`;
- estado canónico: `planned_not_applied`.

Esto no constituye todavía un runtime completo de agentes Firecracker. No hay
wiring productivo que lance todos los agentes en microVM, ni se acredita una
ola de 16, RAM/tmpfs, checkpoints o red de producción.

La corrección de 2026-07-29 añade una fixture neutral sobre esta preparación:
una sola microVM sin red IP, acotada por los dos servicios vsock canónicos. No
es otra decisión ni prueba que authorizer, allocator o proxy estén cableados.

## 4. Ubicación de las ideas posteriores

| Candidato | Contenido | Estado |
| --- | --- | --- |
| V38 | Activar y acreditar Firecracker como adapter de `TestAttestor`, sin red | documentado, no canónico |
| V39 | Runtime de agentes: hasta 16 microVM, RAM/tmpfs, red aislada, broker y proxy controlado | documentado, no canónico |

V38/V39 están deliberadamente fuera del roadmap canónico V1-V37. Formalizarlas
exige una decisión posterior que actualice de forma atómica:

- `product/roadmap.json`;
- conteo y dependencias de verticales;
- acceptance contracts;
- fixtures y receipts;
- pruebas que hoy exigen exactamente 37 verticales.

No añadir V38/V39 silenciosamente ni reutilizar evidencia de V17/V23. El corte
de autoridad completo es
`corte_alcance_v23_firecracker_diferido_2026-07-26.md`.

La fixture recuperada vive en
`acceptance/fixtures/agent_firecracker_single_vm.json`. Su nombre neutral evita
formalizar un número de vertical y ratchea únicamente la autoridad existente;
no reutiliza ningún receipt.

Bubblewrap con `MAX_ARGS=65536` permanece como deuda congelada. No compilar,
instalar ni reabrirla antes de acreditar Orquesta autoprogramable.

## 5. Estado V23 y frentes separados

V23 continúa únicamente por `AC-V23-WIZARD`:

- capacidades WIZ restantes;
- generación editorial;
- flujos restantes del Wizard;
- gates exactos;
- receipt de la misma revisión.

No necesita Firecracker.

Dos ramas SQLite quedaron publicadas como candidatas independientes y no se
integraron durante esta parada:

- `origin/agente/smoke-sqlite-reproducible-v23`;
- `origin/agente/promocion-transaccional-profile`.

El perfil vivo `Codex12` no fue migrado, reconfigurado ni detenido por este
corte. Su promoción, si se retoma, requiere backup, parada cooperativa,
verificación transaccional y rollback propios; no Firecracker.

## 6. Verificación ejecutada

Verde:

```bash
git status --short --branch
go test -mod=vendor -run '^$' ./...
```

El segundo comando compiló todos los paquetes visibles, incluidos raíz y
`acceptance`, sin ejecutar tests.

También pasaron los focales:

```text
TestTestAttestorRegistryIsMinimalAndDisabledByDefault
TestDisabledProviderIgnoresIncompleteMicroVMConfiguration
TestDisabledTestAttestorKeepsNonCodeCompositionOperational
TestAcceptanceV23WizardIntakeContract
```

No se declara la suite completa verde. Un lote anterior se detuvo tras un falso
rojo de permisos del TMPDIR y un test SQLite al 100 % de un núcleo que ya no
podía cambiar el resultado del lote. El proceso y su hijo quedaron cerrados y
el temporal creado por la sesión quedó limpio.

## 7. Punto exacto de reanudación

1. Leer `AGENTS.md`, `LEEME_AGENTE_ORQUESTAV2.md` y este handoff.
2. Confirmar árbol limpio y que `modulos/**` no está visible.
3. Obtener el siguiente write-set de `AC-V23-WIZARD`, no de documentos legacy.
4. Usar Orquesta V22 para dirigir V23 cuando el flujo acreditado lo permita;
   Codex directo queda como bootstrap/desbloqueo acotado.
5. No abrir Firecracker, Bubblewrap, V38 o V39 mientras V23 siga abierto.
6. Mantener commits pequeños en castellano y push frecuente.
7. Antes de parar, revisar y cerrar solo procesos propios de pruebas.
