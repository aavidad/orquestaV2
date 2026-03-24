# Base documentation templates

## Goal

This directory contains reusable templates for the initial documentation set of projects created or structured by Orquesta.

The goal is not to force every document in every case, but to provide a coherent minimum catalog based on project type and operational needs.

## Recommended starting point

Before copying templates, review:

- [Initial catalog by project type](catalogo_tipos_proyecto_en.md)

That catalog explains which pieces are mandatory, which are recommended, and when a template can be skipped without weakening the operational baseline.

## Available templates

- [Project README](README_en.md)
- [User manual](manual_usuario_en.md)
- [Developer manual](manual_desarrollador_en.md)
- [Operations / sysadmin manual](manual_sysadmin_en.md)
- [Installation guide](guia_instalacion_en.md)
- [Deployment guide](guia_despliegue_en.md)
- [FAQ](faq_en.md)
- [Contextual help](ayuda_contextual_en.md)
- [Operational runbook](runbook_operativo_en.md)

## Usage rules

- Spanish is the default baseline
- keep a bilingual structure when the project already targets multiple languages
- avoid duplication across documents: each piece should have a clear purpose
- cross-link related manuals so users, developers, and operators share context

## Suggested minimum copy set

Depending on the project type, a common baseline is:

- core services and APIs: README, developer manual, installation guide, deployment guide, runbook, FAQ
- operational tools: README, installation guide, FAQ, and runbook
- infrastructure controllers: README, developer manual, deployment guide, sysadmin manual, and runbook

If the project has a real user-facing interface for non-technical staff, also add:

- user manual
- contextual help
