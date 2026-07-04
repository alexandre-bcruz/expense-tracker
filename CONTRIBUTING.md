# Contributing

## Commit Conventions

This project follows [Conventional Commits 1.0.0](https://www.conventionalcommits.org/).
Commit messages are written in **English**, in the **imperative mood**
("add", not "added" / "adds").

### Format

```
<type>(<scope>): <subject>

[optional body]

[optional footer(s)]
```

- **Header** is mandatory; **scope** is optional but encouraged.
- Keep the **subject** under ~72 characters, lowercase, no trailing period.
- Separate the body and footers with a blank line.
- Use the body to explain **what** and **why**, not **how**.

### Types

| Type       | Use for                                                        |
|------------|----------------------------------------------------------------|
| `feat`     | A new feature or capability                                    |
| `fix`      | A bug fix                                                      |
| `refactor` | Code change that neither fixes a bug nor adds a feature        |
| `perf`     | A change that improves performance                            |
| `test`     | Adding or correcting tests                                     |
| `docs`     | Documentation only (README, ADRs, comments)                   |
| `build`    | Build system, dependencies, `go.mod`, tooling                 |
| `ci`       | CI configuration and scripts                                  |
| `chore`    | Maintenance that doesn't touch src or tests                   |
| `style`    | Formatting only (no logic change)                             |
| `revert`   | Reverts a previous commit                                     |

### Scopes

Scopes reflect the Event Sourcing + CQRS architecture (see
[ADR-001](docs/adr/ADR-001.md)). Suggested scopes:

- `expense` — the expense aggregate / domain
- `event` — domain events and the event schema
- `store` — event store / persistence
- `command` — command side (write model, handlers)
- `query` — query side (read model)
- `projection` — projections that build read models from events
- `api` — HTTP / transport layer
- `adr` — architecture decision records
- `deps` — dependency bumps

Introduce new scopes as the codebase grows; keep them consistent.

### Breaking changes

Signal a breaking change with a `!` after the type/scope **and/or** a
`BREAKING CHANGE:` footer:

```
feat(event)!: rename ExpenseCreated to ExpenseRecorded

BREAKING CHANGE: stored events using the old type name must be migrated.
```

### Examples

```
feat(expense): add ExpenseCreated event
fix(projection): handle out-of-order events during replay
refactor(store): extract append-only writer into its own type
test(command): cover RecordExpense handler validation
docs(adr): add ADR-002 for event store choice
build(deps): bump go to 1.26
```

### Rules of thumb

- One logical change per commit; keep the history readable and reviewable.
- A commit should build and pass tests on its own.
- Reference issues in the footer when applicable: `Refs #12` / `Closes #12`.

### Enforcement (commit-msg hook)

A versioned `commit-msg` hook validates every commit message against the
rules above. Because Git hook paths are not cloned automatically, enable it
once after cloning:

```bash
git config core.hooksPath .githooks
```

The hook (`.githooks/commit-msg`) is a dependency-free POSIX shell script. It
rejects messages that don't match `type(scope)?: subject`, end with a period,
or exceed 72 characters. Merge, revert, and `fixup!`/`squash!` commits are
exempt.
