# Specs

The document formats and standards for spec-driven development with AI coding agents.

Everything in this directory is installed into a project by `just install <dir>`, which copies it to `docs/specs/` in the target. The paths are the same in both places, so every link below resolves identically wherever it is read.

## Directory Structure

```
/specs/
├── architecture.yaml
├── README.md
└── features/
    └── <component>/
        └── <feature>/
            ├── spec.md
            ├── orchestration.md
            └── architecture.yaml

/projects/
└── <slug>.yaml
```

The top-level `/specs/architecture.yaml` covers the **current** modules of the application ([Architecture Format](architecture.md)). `/specs/README.md` is an index of the features below it.

Specs, orchestrations, and per-feature architecture are co-located under `/specs/features`:

- `spec.md` — behavioral requirements and scenarios ([Spec Format](spec.md))
- `orchestration.md` — idealized domain logic ([Orchestration Format](orchestration.md))
- `architecture.yaml` (optional) — **future** modules introduced by this feature ([Architecture Format](architecture.md))

Project files live at `/projects/<slug>.yaml` and define units of work for a coding agent to execute, drawing on the specs, orchestrations, and architecture above ([Project Format](project.md)).

See [Component](glossary.md#component) and [Feature](glossary.md#feature) in the glossary.

## Formats

| Document | Purpose |
|---|---|
| [Spec](spec.md) | Behavior contracts — structured requirements and scenarios |
| [Orchestration](orchestration.md) | Idealized domain logic as an implementation contract |
| [Architecture](architecture.md) | The deep modules of an application, in YAML |
| [Project](project.md) | Units of work for a coding agent |

## Standards

| Document | Purpose |
|---|---|
| [Writing Code](code.md) | Module placement, and what belongs in each category of module |
| [Testing](testing.md) | Test layers, mock rules, and module boundaries in tests |
| [Writing Requirements](requirements.md) | What makes a good unit of work |
| [Agent Prompts](prompts.md) | How to structure a single-task prompt |
| [Glossary](glossary.md) | Terms used throughout these documents |

## Prompts

| Document | Purpose |
|---|---|
| [Outline](outline.md) | Analyze an existing repo and generate a `/specs` tree for it |

## Order of Authorship

The four formats build on each other. For a new feature:

1. **Spec** — what the system must do, as observable behavior.
2. **Architecture** — which modules the feature needs, and what each is for.
3. **Orchestration** — the shape of the domain logic, with modules assigned.
4. **Project** — the work, split into one unit per iteration, sourced from the three above.

Each step is optional in principle, but a project drafted without an orchestration has to invent implementation shapes, and an orchestration drafted without an architecture has nowhere to put its helpers. Skipping a step moves the decision, it does not remove it.
