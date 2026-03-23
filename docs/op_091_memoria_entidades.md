# OP-091: Memoria de Entidades (Shared Knowledge)

## Contexto
Los agentes a menudo pierden contexto sobre entidades críticas del negocio (ej: IDs de facturas, nombres de servicios, dependencias entre módulos) cuando cambian de sesión o runtime. Se requiere una base de conocimiento persistente y compartida que sirva como "Memoria a Largo Plazo".

## Especificación Técnica

### 1. Esquema de Datos (SQL)
```sql
CREATE TABLE entidades_memoria (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre TEXT NOT NULL UNIQUE,
    tipo TEXT NOT NULL CHECK(tipo IN ('negocio', 'api', 'db', 'infra', 'regla')),
    valor TEXT NOT NULL, -- Almacenamiento en JSON
    metadata TEXT DEFAULT '{}',
    ultima_verificacion DATETIME DEFAULT CURRENT_TIMESTAMP,
    verificado_por TEXT,
    proyecto_id INTEGER REFERENCES proyectos(id)
);
```

### 2. Concepto "Beads" (Perlas de Memoria)
Cada cambio significativo en una entidad genera una "perla" de conocimiento (`bead`) que el supervisor adjunta al `mailbox` local del agente antes de su ejecución.
- **Detección:** Los agentes escanean automáticamente la tabla al entrar en un módulo.
- **Persistencia:** Cualquier alucinación corregida se guarda como una nueva "perla" verificada.

### 3. Recuperación de Contexto
El `runtime_launcher` inyectará en el prompt del sistema:
```text
[MEMORIA_PROYECTO_ACTUAL]
- Entidad: Core_API, Tipo: api, Valor: v2 con soporte gRPC.
- Entidad: DB_Cluster, Tipo: infra, Valor: Solo lectura activado por defecto.
```

## Votación
- **Antigravity:** ACUERDO (Vital para la coherencia en proyectos complejos).
