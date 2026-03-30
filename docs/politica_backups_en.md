<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# SQLite Backend Backup Policy

> Doctrinal note: this policy covers the SQLite backend only.
> The official manual snapshot path is `orquesta respaldo bd`.
> Normal operational verification lives in `orquesta persistencia verificar`.
> If the active backend is not SQLite, an equivalent engine-specific policy must exist.

## Principle

When SQLite is the active backend, Orquesta's database remains the single source of truth for the entire agent ecosystem: tasks, proposals, sessions, rules, decisions, and audit logs. Its loss or corruption would have an irreversible impact on the development of all projects. Therefore, performing periodic backups is **mandatory**.

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

1. Stop the daemon and active agents in an orderly way.
2. Make a copy of the current SQLite file before restoring.
3. Restore the desired `.db.bak` over the active SQLite file.
4. Verify with `orquesta persistencia info` and `orquesta persistencia verificar`.
5. Bring the daemon back and resume operations.

## Traceability

Every backup operation (creation and restoration) must be recorded as a **note** in the Orchestrator indicating:
- Date and time.
- Type of operation (creation / restoration).
- Path of the file involved.
- Agent or process responsible.

## Technical Implementation

Reference implementation may rely on:
- `orquesta respaldo bd` for manual snapshots through the official path
- `scripts/backup_bd.sh` and `scripts/verificar_bd.sh` only as SQLite-specific rescue/automation

This policy defines the requirements, not the implementation.
