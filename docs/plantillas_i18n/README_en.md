<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Base i18n template for projects

## Goal

Define the minimum contract that Orquesta should prepare when a project starts with mandatory i18n.

It does not cover final rendering or framework-specific details. It covers structure, seed files, and minimum rules so new projects stop shipping without a multilingual foundation.

## Expected structure

```text
i18n/
  config.json
  README.md
  es/
    common.json
    navigation.json
    actions.json
    validation.json
    errors.json
  en/
    common.json
    navigation.json
    actions.json
    validation.json
    errors.json
```

## Contract rules

- `es` is the initial default language unless policy says otherwise
- the initial planned pack is `es`, `en`, `de`, `fr`, `it`, `zh`, `gl`, `eu`, `ca`, and `val`, unless policy restricts it further
- each language lives in its own subdirectory
- each functional domain lives in a separate file
- keys must be stable and namespaced
- if a key is missing in a secondary language, the initial fallback is the default language

## Base files

- `common.json`: cross-cutting app copy and generic states
- `navigation.json`: menu, breadcrumbs, and navigation labels
- `actions.json`: reusable buttons and actions
- `validation.json`: validation and form messages
- `errors.json`: network, permission, auth, and generic error copy

## Recommended minimum seed

Each domain should start with at least:

- a base title or block
- main states
- common actions
- generic errors
- frequent form validations

## Orquesta support

The base implementation from this task is materialized through:

```bash
orquesta lenguaje esqueleto <project-path>
```

That command creates the `i18n/` structure, `config.json`, `README.md`, and the initial seed files by language and domain.
