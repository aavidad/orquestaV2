<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Initial template catalog by project type

> Developed with Orquesta by Alberto Avidad Fernandez.

## Goal

Define the minimum documentation set that Orquesta should prepare from the initial project-definition phase depending on the project type.

This catalog is aligned with the architecture policy by project type and avoids treating a core service, a quick operational tool, and an infrastructure controller as if they required the same documentation footprint.

## Catalog building blocks

- `README`: quick overview, purpose, scope, and documentation map
- `manual_usuario`: functional usage guide for non-technical or mixed audiences
- `manual_desarrollador`: architecture, local development, contracts, and quality
- `manual_sysadmin`: platform operations, configuration, and continuity
- `guia_instalacion`: initial installation or local / preproduction setup
- `guia_despliegue`: controlled deployment across environments and production rollout
- `faq`: recurring questions and quick answers
- `ayuda_contextual`: micro-help by screen, workflow, state, or action
- `runbook_operativo`: operational response for incidents and recurring tasks

## Selection rules

- if the project has a real user interface, include `manual_usuario` and `ayuda_contextual`
- if the project is deployed across environments or reaches production, include `guia_despliegue`
- if installation is not trivial or requires prerequisites, include `guia_instalacion`
- if there is ongoing operation or on-call responsibility, include `runbook_operativo`
- if development and operations are separated, include `manual_sysadmin`
- if third parties will extend or integrate the system, include `manual_desarrollador`

## Minimum set by project type

### 1. Core services and APIs

Typical context:

- main backend
- business APIs
- long-lived services with ongoing functional evolution

Minimum documentation:

- `README`
- `manual_desarrollador`
- `guia_instalacion`
- `guia_despliegue`
- `runbook_operativo`
- `faq`

Recommended documentation:

- `manual_sysadmin`
- `manual_usuario` if there is a dashboard or functional console
- `ayuda_contextual` if people directly operate an interface

### 2. Quick operational scripts and tools

Typical context:

- internal utilities
- assisted migrations
- support or maintenance tools
- narrow-scope automation

Minimum documentation:

- `README`
- `guia_instalacion`
- `faq`
- `runbook_operativo`

Recommended documentation:

- `manual_desarrollador` if the tool will evolve or accept contributions
- `manual_usuario` if non-technical profiles will use it
- `guia_despliegue` if it is distributed beyond the author's local environment

### 3. Infrastructure controllers

Typical context:

- provider integrations
- bridges to SDKs or third-party APIs
- deployment or provisioning automation

Minimum documentation:

- `README`
- `manual_desarrollador`
- `guia_despliegue`
- `manual_sysadmin`
- `runbook_operativo`
- `faq`

Recommended documentation:

- `guia_instalacion` if it can be tested or started locally
- `manual_usuario` and `ayuda_contextual` only if it exposes a real human-operated interface

## Suggested combinations

### Minimal small-project profile

- `README`
- `guia_instalacion`
- `faq`

### Standard professional profile

- `README`
- `manual_desarrollador`
- `guia_instalacion`
- `guia_despliegue`
- `faq`
- `runbook_operativo`

### Functional UI profile

- `README`
- `manual_usuario`
- `ayuda_contextual`
- `faq`
- the remaining technical pieces according to project type

## Quality rule

The existence of a template does not require artificial filler content. If a piece does not apply, state that explicitly in the project `README` instead of leaving an empty or misleading document behind.
