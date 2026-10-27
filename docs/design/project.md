# Project Design

A project is a YAML or JSON file that lists work. Ralph needs one thing from the file: an array of items, selected with a jq query. Each item is one unit of work and one iteration. Every other field is passed to the agent as context.

## Items

An item is one element of the resolved array. A string, a mapping, and a nested structure are all valid items. Nothing is required of its shape.

An item has two derived values:

* **Key** - the scalar `slug`, `id`, or `name` field of a mapping item, checked in that order. It labels the item in prompts and logs. It is not an identifier.
* **Text** - canonical YAML for the item's value. A plain string item is shown to the picker as its own value, so quoting cannot change which item it names.

Ralph identifies items by the [completion hash](completion.md) of their text. Two items with the same text share a hash. An item keeps its identity across runs while its text is unchanged.

## Item Query

The item query is a jq expression that resolves the file to its items. It defaults to `.`, which fits a file whose top level is already an array.

Resolution collects the query's outputs:

* One output that is an array makes its elements the items.
* Any other case makes each output an item.

Ralph drops empty outputs before it assigns indices. An output is empty when it is null, `false`, `0`, a blank string, `{}`, or `[]`. An index is therefore a position in the remaining list. When nothing remains, the command that needs items fails with `item query yielded no items: <query>`.

The query resolves once per run and stays fixed. `ralph run`, `ralph complete`, `ralph incomplete`, and `ralph validate` resolve it the same way, so they agree on items and indices.

## Metadata

When the top level is a mapping, Ralph reads two optional fields:

* `slug` sets the project slug, and from it the branch name. It falls back to the file's base name.
* `title` sets the pull request title when the AI description has no title. It falls back to the slug.

A file whose top level is an array has neither field, so both come from the file name.

## Progress

A run reads the project file and resolves its items once, then leaves it unchanged. The only write Ralph makes to a project file is the cleanup deletion after every item is complete. See [Completion Design](completion.md).

## Validation

`ralph validate` checks that the file parses, that the query evaluates, and that at least one non-empty item remains. See [Validate Design](validate.md).
