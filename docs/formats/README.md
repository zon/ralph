# File Formats

Documentation of file formats used in Ralph development.

## Directory Structure

```
/specs/
├── architecture.yaml
└── features/
    └── <component>/
        └── <feature>/
            ├── spec.md
            ├── orchestration.md
            └── architecture.yaml

/projects/
└── <slug>.yaml
```

The top-level `/specs/architecture.yaml` covers the **current** modules of the application ([Architecture Format](./architecture.md)).

Specs, orchestrations, and per-feature architecture are co-located under `/specs/features`:

- `spec.md` — behavioral requirements and scenarios ([Spec Format](./specs.md))
- `orchestration.md` — idealized domain logic ([Orchestration Format](./orchestration.md))
- `architecture.yaml` (optional) — **future** modules introduced by this feature ([Architecture Format](./architecture.md))

Project files live at `/projects/<slug>.yaml` and define units of work for the ralph agent to execute, drawing on the specs, orchestrations, and architecture above ([Project Format](./project.md)).

See [Component](../glossary.md#component) and [Feature](../glossary.md#feature) in the glossary.

## Formats

### [Specs](./specs.md)

The spec format for describing system behavior using structured requirements and scenarios.

### [Orchestrations](./orchestration.md)

The orchestration format for documenting idealized domain logic as implementation contracts.

### [Architecture](./architecture.md)

The architecture format for outlining deep modules in YAML.

### [Projects](./project.md)

The project format for defining units of work for the ralph agent.