# Architectural Policy by Project Type

Orquesta does not impose a single rigid structure for every technical situation, but establishes strict guidelines based on the project type:

1. **Core Services and APIs:** Must strictly follow a hexagonal architecture (Separation into ports and adapters). This is a vital requirement for the growth of `Orquestador` itself and `PlataformaMunicipal`, to avoid coupling business logic to the database (e.g. SQLite).
2. **Scripts and Quick Operational Tools:** May use a flat or monolithic structure if their purpose is sporadic and they do not manage complex persistent state.
3. **Infrastructure Controllers:** Are allowed to couple directly to SDKs and libraries if their sole purpose is deployment or API bridging.

*(Note: Hexagonalization will act as an architectural "gate" prior to version upgrades in core modules to prevent future technical debt).*
