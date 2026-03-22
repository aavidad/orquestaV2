# Task and Project Language Selection Policy

Orquesta's technical philosophy strongly encourages the use of the programming language or tool that guarantees the highest productivity output and risk mitigation for any specific use case, strictly preventing monoculture dogmatism.

- **Go (Golang):** The systematic favorite for driving core services, scalable inter-municipality APIs, and agile command-line tools. It stands as the foundational stone of Orquesta's main codebase due to its native concurrency and simple deployment paradigms.
- **Rust:** To be reserved deliberately for isolated modules where extreme optimization is hypercritical, memory safety features are compulsorily required at systemic levels (ownership, lifetimes), or low-level hardware constraints demand it.
- **TypeScript & JavaScript:** The default stack chosen for everything related to frontend development, Web UIs (like the operational dashboard) or seamless node-environment integrations.
- **Python:** Language restricted selectively to data analysis exploitations, generic local pipelines and data-science/machine learning scenarios.
