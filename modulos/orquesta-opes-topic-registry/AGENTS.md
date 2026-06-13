# AGENTS: orquesta-opes-topic-registry

Adaptador OPES para ejecutar la herramienta oficial del registro de trabajo por
temas.

## Reglas

- No editar JSON de OPES directamente.
- No importar runtime, stack Codex, servidor ni conectores HTTP.
- Construir comandos contra la herramienta oficial y ejecutar por runner
  inyectado.
- Las pruebas no deben tocar el registro real de OPES: usar runner falso.
- Todo resultado debe conservar refs compactas, sin filtrar rutas locales salvo
  que el operador haya configurado explicitamente el `tool_path`.
