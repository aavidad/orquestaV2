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
