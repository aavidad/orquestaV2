<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Database Backup Policy

## Principle

Orquesta's database is the single source of truth for the entire agent ecosystem: tasks, proposals, sessions, rules, decisions, and audit logs. Its loss or corruption would have an irreversible impact on the development of all projects. Therefore, performing periodic backups is **mandatory**.

## Frequency and Retention

| Type          | Frequency    | Minimum Retention |
|---------------|-------------|------------------|
| Daily         | Every 24 h  | 7 days           |
| Weekly        | Every Monday | 4 weeks         |
| Monthly       | 1st of each month | 3 months   |

## Storage Path

Backups must be stored **outside the orchestrator's working directory** so that an accidental deletion of the repository does not also remove the backup:

```
~/Trabajo/backups/orquestador/
  └── YYYY-MM-DD_HH-MM-SS_orquesta.db.bak
```

## Restoration Procedure

1. Stop all active agents (`orquesta sesion fin <agent>`).
2. Make a copy of the current DB before restoring (in case restoration worsens the state).
3. Copy the desired `.db.bak` file over `orquesta.db`.
4. Verify integrity with `orquesta status`.
5. Restart agent sessions.

## Traceability

Every backup operation (creation and restoration) must be recorded as a **note** in the Orchestrator indicating:
- Date and time.
- Type of operation (creation / restoration).
- Path of the file involved.
- Agent or process responsible.

## Technical Implementation

The implementation of the script or scheduled task that materialises these backups is the responsibility of the programmer agent assigned to **Task #217**. This policy defines the requirements, not the implementation.
