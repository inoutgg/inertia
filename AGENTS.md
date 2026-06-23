# AGENTS

## General

Hard rules that must never be violated:

- Never commit changes unless the user explicitly asks for it.
- Never revert, discard, overwrite, or clean user changes made outside the agent's cycle.
- Do not make source edits while committing unless the user explicitly asks for fixes too.
- Do not push.

## Testing

### Setup

- Always use `internal/inertiatest/` for configuring a new inertia test request in tests (e.g. `inertiatest.NewRequest`, `inertiatest.NewPropTestBuilder`).

### Modifying tests

- Never change any tests without explicitly asking and getting approval. Ask per each test change rather than as a batch.

### Test messages

- Always use the following format for test messages: "it should <tested-behaviour> when <tested-condition>".

### Complex test subjects

Complex (with multiple if-else branches/explicit guarded features/etc.) struct/function/etc. requiring multiple test behaviours must follow the rules:

- split the tests into testing groups via `Test<StructName>_<TestedFeature>`.
- the `Test<StructName>_<TestedFeature>` contains `t.Run(...)` per each tested behaviour.
