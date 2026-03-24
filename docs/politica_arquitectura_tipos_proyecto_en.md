# Architectural Policy by Project Type

Orquesta does not impose a single rigid structure for every technical situation, but establishes strict guidelines based on the project type:

1. **Core Services and APIs:** Must strictly follow a hexagonal architecture (Separation into ports and adapters). This is a vital requirement for the growth of `Orquestador` itself and `PlataformaMunicipal`, to avoid coupling business logic to the database (e.g. SQLite).
2. **Scripts and Quick Operational Tools:** May use a flat or monolithic structure if their purpose is sporadic and they do not manage complex persistent state.
3. **Infrastructure Controllers:** Are allowed to couple directly to SDKs and libraries if their sole purpose is deployment or API bridging.

*(Note: Hexagonalization will act as an architectural "gate" prior to version upgrades in core modules to prevent future technical debt).*

## Architecture selection matrix

| Project type | Default architecture | Mandatory | Allowed | Not allowed |
| --- | --- | --- | --- | --- |
| Core service / business API | Strict hexagonal | Ports and adapters, separated use cases, mediated persistence, contract tests | Internal modules by domain, application services | Coupling business logic to SQLite, handlers with business logic, infrastructure shortcuts |
| Web or desktop client on top of Orquesta | Thin client | Consume exposed API/services only, local state limited to UI, i18n from the start when applicable | Local read cache, presentation adapters | Direct DB access, second control plane, duplicating business rules |
| Shared library / reusable package | Responsibility-based modular design | Small stable API, tests, usage docs, layered reuse | Standalone functions, `structs`, small packages | Growing into a generic framework without need, unnecessary heavy dependencies |
| Worker / automation / integration | Lightweight hexagonal or modular | Separate trigger, use case and external adapter when complexity or state exists | Flatter design if scope is truly narrow | Mixing business policy with external SDK without a clear boundary when the module is expected to grow |
| One-off script / operational utility | Flat structure | Clear entrypoint, validation, minimal logs, no complex persistence | Single file or small package | Artificial over-engineering or fake layers |
| Infrastructure / deployment controller | Direct adapter | Scope limited to deployment, provisioning or technical bridging | Couple directly to SDKs or CLIs if no domain logic exists | Becoming a container for domain logic |

## Decision rules

1. If the project contains business logic, persistence, or a stable API, the default choice is hexagonal.
2. If the project only presents data or consumes another service API, it must remain a thin client.
3. If the real scope fits in a function, a `struct`, or a small package, it must not be inflated into a large architecture.
4. If the project touches infrastructure but starts to absorb business policy, it must stop being treated as a simple controller and move to a design with explicit boundaries.
5. The architectural choice must be documented in the project's initial report and be justifiable by app type, cost, and complexity.

## Hexagonalization gate

Hexagonalization is mandatory before further growth in:

- core services
- business APIs
- persistent modules with expected evolution
- Orquesta pieces that belong to the control plane

It should not be imposed as ritual in:

- ephemeral scripts
- one-off migration utilities
- short-lived proofs of concept
