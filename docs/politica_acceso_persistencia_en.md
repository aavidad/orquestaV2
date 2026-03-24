# Persistence Access Policy (AP-077)

## Fundamental Principle
Orquesta is the only entry and exit portal for system data. The internal integrity, traceability, and safety of the ecosystem depend on all interaction with the persistence occurring through the contracts defined in the application (CLI and API).

## Operational Rules

1. **Write and Mutation:** All operations for creating, modifying, or deleting data (tasks, proposals, agents, sessions, etc.) must be carried out exclusively through Orquesta commands. Direct write access to the database (e.g., using `sqlite3`) is strictly prohibited.
2. **Read and Inspection:** Direct read or inspection access to the persistence is tolerated only on an **exceptional** basis while the application does not functionally cover all necessary observability, diagnosis, administration, and export.
3. **Closing Gaps:** In response to any operational need that requires a direct query (SQL), the agent must immediately open a technical task to implement said function in the Orquesta CLI/API.
4. **Operational Blocking:** Once Orquesta's coverage is total in the aforementioned areas, the technical blocking of operational external access to persistence will proceed.

## Backend Independence
This policy is formulated against the application layer and not against the specific database engine. This allows the system to migrate from SQLite to other engines (such as MySQL/MariaDB) in the future without breaking agent operations or compromising data governance.

## Transition Status

As of `2026-03-24`, the project direction is already stable:

- the CLI and API cover most frequent operational reads and mutations
- the target remains `zero direct access` as the normal path
- exceptional diagnostic access is still tolerated only while real technical gaps remain

The blockers that still prevent turning on full technical blocking are:

- consolidating a single official daemon for continuous operation
- removing `runtime_connector` and `agente_console` from the main operational path
- closing the web layer as a full thin client
- completing end-to-end smoke tests with a real live runtime

The operational inventory of those gaps and their recommended closure order lives in [Inventory of pending work for autonomous orchestration](inventario_pendientes_orquestacion_autonoma_2026-03-24.md).
