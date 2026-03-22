# Política de Selección de Lenguaje por Tarea y Proyecto

La filosofía técnica de Orquesta fomenta fuertemente el uso de la herramienta o lenguaje de programación que proporcione el máximo de productividad y mitigación de riesgos para cada caso de uso concreto, sin imponer dogmas de monocultura.

- **Go (Golang):** El preferido sistemáticamente para desarrollar los servicios core, APIs escalables inter-municipio y herramientas ágiles de consola. Es la piedra fundacional del código base general de Orquesta gracias a su manejo nativo de la concurrencia y simplicidad de despliegue.
- **Rust:** Se reservará para módulos aislados en donde la optimización sea muy crítica, la seguridad estricta de la memoria sea obligatoria a nivel sistémico (lifetimes, ownership), o requerimientos del hardware lo demanden.
- **TypeScript & JavaScript:** Pila predeterminada en todo lo tocante a frontend, Web UI (dashboard operativo de Orquesta) e integración fluida con entornos Node.js.
- **Python:** Lenguaje condicionado a los pipelines de explotación de datos analíticos intensivos, automatización genérica local o machine learning.
