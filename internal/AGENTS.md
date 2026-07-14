# Nueva Orquesta: reglas de implementación

Ámbito: `internal/**` del corte mínimo funcional.

- No importar `orquesta/modulos/...` ni copiar paquetes legacy completos.
- `goal` posee estado y transiciones; ningún adaptador escribe lifecycle.
- `application` es el único escritor mediante puertos transaccionales.
- Interfaces se declaran donde las consume la aplicación.
- Adaptadores dependen de dominio/puertos; dominio nunca depende de adaptadores.
- Provider, DB, filesystem, MCP, env, paths y proceso viven fuera de dominio.
- `ActorRef` y `ProjectRef` viajan explícitos; sin globals de usuario/proyecto.
- Errores tienen código máquina estable; texto humano sale de i18n.
- No añadir generación `V0/V1/V2` a nombres internos.
- Nueva abstracción exige frontera real o duplicación nombrada.
- Cada cambio incluye test de aceptación del manifest que cubre.
