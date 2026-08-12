# Agent Prompts

Agent prompts drive a single, focused task. A well-written prompt gives the agent a clear role, the context it needs, and unambiguous instructions for what to produce.

## Structure

A well-structured prompt contains these sections in order:

1. **Role** — one sentence assigning the agent a persona (e.g. "You are a software architect.")
2. **Task** — a brief statement of what the agent must accomplish
3. **Context** — relevant information the agent needs to reason about the task
4. **Definitions** — inline definitions for any domain-specific concepts
5. **Instructions** — a numbered list of concrete steps, ordered by execution sequence
6. **Output** — an explicit description of what to produce and where

## Principles

- **One task per prompt.** Each prompt drives one agent action. Combining tasks causes the agent to lose focus.
- **Give the agent only the context it needs.** Irrelevant context dilutes attention. Omit optional sections when they have nothing to contribute.
- **Use numbered steps for instructions.** Ordered steps give the agent a clear execution path and reduce ambiguous behavior.
- **Define domain terms inline.** If the prompt relies on a concept with a precise meaning, define it in the prompt. Don't assume the agent infers it from context.
- **Specify output explicitly.** Tell the agent exactly what to produce — format, location, and any schema or example it should follow.
- **State blocking behavior.** If the agent can get stuck, tell it what to do rather than leaving it to guess.

## Prompts vs Skills

A prompt is invoked once with its context inlined. A skill is installed into a repository and invoked by name, so it must locate its own context — which is why the skills in this repository reference documents by path and let the agent read them, rather than restating their contents. When a task is repeated across projects, write it as a skill; when it is a one-off with specific inputs, write it as a prompt. [Outline](outline.md) is an example of the latter.
